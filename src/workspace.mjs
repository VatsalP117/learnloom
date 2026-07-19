import { randomUUID } from "node:crypto";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { DatabaseSync } from "node:sqlite";
import { isPotentialSiteUsername } from "./host-routing.mjs";

const ISSUE_STATES = new Set(["queued", "generating", "generated", "failed"]);
const DELIVERY_STATES = new Set([
  "pending",
  "delivering",
  "delivered",
  "failed",
  "cancelled",
  "unknown",
]);
const MAX_SCHEDULE_SCAN_MINUTES = 8 * 24 * 60;

export class SQLiteWorkspace {
  constructor(databasePath, options = {}) {
    this.now = options.now ?? (() => new Date());
    if (databasePath !== ":memory:") {
      mkdirSync(path.dirname(path.resolve(databasePath)), {
        recursive: true,
        mode: 0o700,
      });
    }
    this.database = new DatabaseSync(databasePath);
    try {
      this.initialize();
    } catch (error) {
      this.database.close();
      throw error;
    }
  }

  initialize() {
    this.database.exec("PRAGMA busy_timeout = 5000");
    this.database.exec("PRAGMA foreign_keys = ON");
    execWithBusyRetry(
      this.database,
      "PRAGMA journal_mode = WAL",
      5_000,
    );
    this.transaction(() => {
      const currentVersion = Number(
        this.database.prepare("PRAGMA user_version").get().user_version,
      );
      if (currentVersion > 5) {
        throw new Error(
          `Workspace schema version ${currentVersion} is newer than this Learnloom release supports.`,
        );
      }
      this.database.exec(`
        CREATE TABLE IF NOT EXISTS newsletters (
          id TEXT PRIMARY KEY,
          name TEXT NOT NULL,
          topic TEXT NOT NULL,
          learner_level TEXT NOT NULL,
          learner_goal TEXT NOT NULL,
          lesson_minutes INTEGER NOT NULL CHECK (lesson_minutes BETWEEN 5 AND 90),
          sources_json TEXT NOT NULL,
          schedule_hour INTEGER NOT NULL CHECK (schedule_hour BETWEEN 0 AND 23),
          schedule_minute INTEGER NOT NULL CHECK (schedule_minute BETWEEN 0 AND 59),
          time_zone TEXT NOT NULL,
          active INTEGER NOT NULL CHECK (active IN (0, 1)),
          next_run_at TEXT NOT NULL,
          created_at TEXT NOT NULL,
          updated_at TEXT NOT NULL
        ) STRICT;

        CREATE TABLE IF NOT EXISTS issues (
          id TEXT PRIMARY KEY,
          newsletter_id TEXT NOT NULL REFERENCES newsletters(id) ON DELETE CASCADE,
          trigger TEXT NOT NULL CHECK (trigger IN ('scheduled', 'manual')),
          scheduled_local_date TEXT,
          status TEXT NOT NULL CHECK (
            status IN ('queued', 'generating', 'generated', 'failed')
          ),
          dossier_title TEXT,
          generation_id TEXT,
          artifact_path TEXT,
          dossier_path TEXT,
          error TEXT,
          created_at TEXT NOT NULL,
          started_at TEXT,
          completed_at TEXT
        ) STRICT;

        CREATE UNIQUE INDEX IF NOT EXISTS issues_one_scheduled_per_day
          ON issues(newsletter_id, scheduled_local_date)
          WHERE trigger = 'scheduled';
        CREATE INDEX IF NOT EXISTS issues_queue
          ON issues(status, created_at);
        CREATE INDEX IF NOT EXISTS issues_newsletter_history
          ON issues(newsletter_id, created_at DESC);
      `);
      if (currentVersion < 2) {
        this.database.exec(`
          ALTER TABLE newsletters
            ADD COLUMN email_enabled INTEGER NOT NULL DEFAULT 0
            CHECK (email_enabled IN (0, 1));
          ALTER TABLE newsletters
            ADD COLUMN email_recipients_json TEXT NOT NULL DEFAULT '[]';

          CREATE TABLE issue_deliveries (
            issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
            channel TEXT NOT NULL CHECK (channel = 'email'),
            status TEXT NOT NULL CHECK (
              status IN (
                'pending', 'delivering', 'delivered', 'failed', 'cancelled',
                'unknown'
              )
            ),
            attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
            external_id TEXT,
            error TEXT,
            created_at TEXT NOT NULL,
            started_at TEXT,
            completed_at TEXT,
            updated_at TEXT NOT NULL,
            PRIMARY KEY (issue_id, channel)
          ) STRICT;

          CREATE INDEX issue_deliveries_queue
            ON issue_deliveries(status, created_at);
          PRAGMA user_version = 2;
        `);
      }
      if (currentVersion < 3) {
        this.database.exec(`
          ALTER TABLE newsletters
            ADD COLUMN ai_exploration_enabled INTEGER NOT NULL DEFAULT 0
            CHECK (ai_exploration_enabled IN (0, 1));
          PRAGMA user_version = 3;
        `);
      }
      if (currentVersion < 4) {
        this.database.exec(`
          CREATE TABLE accounts (
            id TEXT PRIMARY KEY,
            clerk_user_id TEXT NOT NULL UNIQUE,
            status TEXT NOT NULL
              CHECK (status IN ('active', 'suspended', 'deleted')),
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            deleted_at TEXT
          ) STRICT;

          CREATE TABLE sites (
            id TEXT PRIMARY KEY,
            owner_account_id TEXT NOT NULL UNIQUE
              REFERENCES accounts(id) ON DELETE CASCADE,
            username TEXT NOT NULL COLLATE NOCASE UNIQUE,
            display_name TEXT NOT NULL,
            description TEXT NOT NULL DEFAULT '',
            visibility TEXT NOT NULL DEFAULT 'private'
              CHECK (visibility IN ('private', 'public')),
            claimed_at TEXT NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            CHECK (username = lower(username))
          ) STRICT;

          ALTER TABLE newsletters
            ADD COLUMN owner_account_id TEXT
            REFERENCES accounts(id) ON DELETE CASCADE;

          CREATE INDEX newsletters_owner
            ON newsletters(owner_account_id, created_at);
          PRAGMA user_version = 4;
        `);
      }
      if (currentVersion < 5) {
        this.database.exec(`
          ALTER TABLE newsletters
            ADD COLUMN public_slug TEXT;
          ALTER TABLE newsletters
            ADD COLUMN site_visible INTEGER NOT NULL DEFAULT 1
            CHECK (site_visible IN (0, 1));
          ALTER TABLE issues
            ADD COLUMN public_id TEXT;
          ALTER TABLE issues
            ADD COLUMN public_slug TEXT;
          ALTER TABLE issues
            ADD COLUMN publication_state TEXT NOT NULL DEFAULT 'published'
            CHECK (publication_state IN ('published', 'hidden'));
        `);

        const newsletterSlugs = new Map();
        const newsletters = this.database
          .prepare("SELECT id, owner_account_id, name FROM newsletters ORDER BY created_at, id")
          .all();
        const updateNewsletter = this.database.prepare(
          "UPDATE newsletters SET public_slug = ? WHERE id = ?",
        );
        for (const newsletter of newsletters) {
          const scope = newsletter.owner_account_id ?? `legacy:${newsletter.id}`;
          const used = newsletterSlugs.get(scope) ?? new Set();
          const slug = uniqueSlug(publicSlug(newsletter.name, "topic"), used);
          used.add(slug);
          newsletterSlugs.set(scope, used);
          updateNewsletter.run(slug, newsletter.id);
        }

        const issues = this.database
          .prepare("SELECT id, dossier_title FROM issues ORDER BY created_at, id")
          .all();
        const updateIssue = this.database.prepare(`
          UPDATE issues
          SET public_id = ?, public_slug = ?
          WHERE id = ?
        `);
        for (const issue of issues) {
          updateIssue.run(
            `dossier-${randomUUID()}`,
            issue.dossier_title
              ? publicSlug(issue.dossier_title, "dossier")
              : null,
            issue.id,
          );
        }

        this.database.exec(`
          CREATE UNIQUE INDEX newsletters_owner_public_slug
            ON newsletters(owner_account_id, public_slug COLLATE NOCASE);
          CREATE UNIQUE INDEX issues_public_id
            ON issues(public_id)
            WHERE public_id IS NOT NULL;
          CREATE INDEX issues_public_archive
            ON issues(newsletter_id, publication_state, completed_at DESC);
          PRAGMA user_version = 5;
        `);
      }
    });
  }

  diagnostics() {
    return {
      userVersion: Number(this.database.prepare("PRAGMA user_version").get().user_version),
      foreignKeys:
        Number(this.database.prepare("PRAGMA foreign_keys").get().foreign_keys) === 1,
      journalMode: String(
        this.database.prepare("PRAGMA journal_mode").get().journal_mode,
      ),
      busyTimeout: Number(
        this.database.prepare("PRAGMA busy_timeout").get().timeout,
      ),
    };
  }

  forAccount(accountId) {
    const account = this.requireActiveAccount(accountId);
    const scope = { accountId: account.id };
    const requireNewsletter = (id) => this.requireNewsletter(id, scope);
    const requireIssue = (id) => {
      const issue = this.getIssue(id, scope);
      if (!issue) throw new Error(`Issue ${id} was not found.`);
      return issue;
    };
    return Object.freeze({
      listNewsletters: () => this.listNewsletters(scope),
      getNewsletter: (id) => this.getNewsletter(id, scope),
      createNewsletter: (input) => this.createNewsletter(input, scope),
      setNewsletterActive: (id, active) => {
        requireNewsletter(id);
        return this.setNewsletterActive(id, active);
      },
      setNewsletterEmail: (id, input) => {
        requireNewsletter(id);
        return this.setNewsletterEmail(id, input);
      },
      setNewsletterContent: (id, input) => {
        requireNewsletter(id);
        return this.setNewsletterContent(id, input);
      },
      setNewsletterSiteVisible: (id, visible) => {
        requireNewsletter(id);
        return this.setNewsletterSiteVisible(id, visible);
      },
      enqueueManualIssue: (id) => {
        requireNewsletter(id);
        return this.enqueueManualIssue(id);
      },
      listIssues: (newsletterId, options) =>
        this.listIssues(newsletterId, { ...options, ...scope }),
      getIssue: (id) => this.getIssue(id, scope),
      setIssuePublication: (id, state) => {
        requireIssue(id);
        return this.setIssuePublication(id, state);
      },
      retryDelivery: (issueId) => {
        requireIssue(issueId);
        return this.retryDelivery(issueId);
      },
    });
  }

  ensureAccount(clerkUserId) {
    const normalizedClerkUserId = requireText(
      clerkUserId,
      "Clerk user ID",
      200,
    );
    const now = this.now().toISOString();
    this.database
      .prepare(`
        INSERT OR IGNORE INTO accounts (
          id, clerk_user_id, status, created_at, updated_at
        ) VALUES (?, ?, 'active', ?, ?)
      `)
      .run(`account-${randomUUID()}`, normalizedClerkUserId, now, now);
    return this.getAccountByClerkUserId(normalizedClerkUserId);
  }

  getAccountByClerkUserId(clerkUserId) {
    const row = this.database
      .prepare("SELECT * FROM accounts WHERE clerk_user_id = ?")
      .get(clerkUserId);
    return row ? mapAccount(row) : null;
  }

  getSiteForAccount(accountId) {
    const row = this.database
      .prepare("SELECT * FROM sites WHERE owner_account_id = ?")
      .get(accountId);
    return row ? mapSite(row) : null;
  }

  isSiteUsernameAvailable(value) {
    const username = String(value ?? "").trim().toLowerCase();
    if (!isPotentialSiteUsername(username)) return false;
    return !this.database
      .prepare("SELECT 1 FROM sites WHERE username = ? COLLATE NOCASE")
      .get(username);
  }

  claimSite(accountId, input) {
    const account = this.requireActiveAccount(accountId);
    const username = String(input?.username ?? "").trim().toLowerCase();
    if (!isPotentialSiteUsername(username)) {
      throw new Error("Username is invalid or reserved.");
    }
    const displayName = requireText(
      input?.displayName ?? username,
      "Site display name",
      100,
    );
    return this.transaction(() => {
      const existing = this.getSiteForAccount(account.id);
      if (existing) {
        if (existing.username === username) return existing;
        throw new Error("This account has already claimed a username.");
      }
      if (
        this.database
          .prepare("SELECT 1 FROM sites WHERE username = ? COLLATE NOCASE")
          .get(username)
      ) {
        throw new Error("Username is already claimed.");
      }
      const now = this.now().toISOString();
      this.database
        .prepare(`
          INSERT INTO sites (
            id, owner_account_id, username, display_name, description,
            visibility, claimed_at, created_at, updated_at
          ) VALUES (?, ?, ?, ?, '', 'private', ?, ?, ?)
        `)
        .run(
          `site-${randomUUID()}`,
          account.id,
          username,
          displayName,
          now,
          now,
          now,
        );
      return this.getSiteForAccount(account.id);
    });
  }

  updateSiteSettings(accountId, input) {
    const account = this.requireActiveAccount(accountId);
    const site = this.getSiteForAccount(account.id);
    if (!site) throw new Error("Claim a username before updating your site.");
    if (!input || typeof input !== "object" || Array.isArray(input)) {
      throw new Error("Site settings must be an object.");
    }
    const visibility = input.visibility ?? site.visibility;
    if (!["private", "public"].includes(visibility)) {
      throw new Error("Site visibility must be private or public.");
    }
    const displayName =
      input.displayName === undefined
        ? site.displayName
        : requireText(input.displayName, "Site display name", 100);
    const description =
      input.description === undefined
        ? site.description
        : optionalText(input.description, "Site description", 500);
    this.database
      .prepare(`
        UPDATE sites
        SET display_name = ?, description = ?, visibility = ?, updated_at = ?
        WHERE owner_account_id = ?
      `)
      .run(
        displayName,
        description,
        visibility,
        this.now().toISOString(),
        account.id,
      );
    return this.getSiteForAccount(account.id);
  }

  getPublicSite(username) {
    const normalized = String(username ?? "").trim().toLowerCase();
    if (!isPotentialSiteUsername(normalized)) return null;
    const row = this.database
      .prepare(`
        SELECT s.*
        FROM sites AS s
        JOIN accounts AS a ON a.id = s.owner_account_id
        WHERE s.username = ? COLLATE NOCASE
          AND s.visibility = 'public'
          AND a.status = 'active'
      `)
      .get(normalized);
    return row ? mapSite(row) : null;
  }

  listPublicNewsletters(username) {
    const site = this.getPublicSite(username);
    if (!site) return [];
    return this.database
      .prepare(`
        SELECT n.*,
          COUNT(i.id) AS issue_count,
          COUNT(i.id) AS generated_count,
          MAX(i.completed_at) AS latest_completed_at
        FROM newsletters AS n
        JOIN issues AS i ON i.newsletter_id = n.id
        WHERE n.owner_account_id = ?
          AND n.site_visible = 1
          AND i.status = 'generated'
          AND i.publication_state = 'published'
          AND i.artifact_path IS NOT NULL
          AND i.dossier_path IS NOT NULL
        GROUP BY n.id
        ORDER BY latest_completed_at DESC, n.created_at ASC
      `)
      .all(site.ownerAccountId)
      .map(mapNewsletter);
  }

  listPublicIssues(username, options = {}) {
    const site = this.getPublicSite(username);
    if (!site) return [];
    const limit = boundedInteger(options.limit ?? 50, 1, 100, "Issue limit");
    const newsletterSlug = options.newsletterSlug ?? null;
    return this.database
      .prepare(`
        SELECT i.*, n.name AS newsletter_name,
          n.public_slug AS newsletter_public_slug
        FROM issues AS i
        JOIN newsletters AS n ON n.id = i.newsletter_id
        WHERE n.owner_account_id = ?
          AND n.site_visible = 1
          AND (? IS NULL OR n.public_slug = ? COLLATE NOCASE)
          AND i.status = 'generated'
          AND i.publication_state = 'published'
          AND i.public_id IS NOT NULL
          AND i.public_slug IS NOT NULL
          AND i.artifact_path IS NOT NULL
          AND i.dossier_path IS NOT NULL
        ORDER BY i.completed_at DESC, i.created_at DESC
        LIMIT ?
      `)
      .all(site.ownerAccountId, newsletterSlug, newsletterSlug, limit)
      .map(mapPublicIssue);
  }

  getPublicIssue(username, publicId) {
    const site = this.getPublicSite(username);
    if (!site) return null;
    const row = this.database
      .prepare(`
        SELECT i.*, n.name AS newsletter_name,
          n.public_slug AS newsletter_public_slug
        FROM issues AS i
        JOIN newsletters AS n ON n.id = i.newsletter_id
        WHERE n.owner_account_id = ?
          AND n.site_visible = 1
          AND i.public_id = ?
          AND i.status = 'generated'
          AND i.publication_state = 'published'
          AND i.public_slug IS NOT NULL
          AND i.artifact_path IS NOT NULL
          AND i.dossier_path IS NOT NULL
      `)
      .get(site.ownerAccountId, publicId);
    return row ? mapPublicIssue(row) : null;
  }

  createNewsletter(input, options = {}) {
    const ownerAccountId = options.accountId ?? null;
    if (ownerAccountId) this.requireActiveAccount(ownerAccountId);
    const now = this.now();
    const newsletter = normalizeNewsletter(input, now);
    const publicNewsletterSlug = this.allocateNewsletterSlug(
      ownerAccountId,
      newsletter.name,
    );
    this.database
      .prepare(`
        INSERT INTO newsletters (
          id, name, topic, learner_level, learner_goal, lesson_minutes,
          sources_json, schedule_hour, schedule_minute, time_zone, active,
          next_run_at, created_at, updated_at, email_enabled,
          email_recipients_json, ai_exploration_enabled, owner_account_id,
          public_slug, site_visible
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
      `)
      .run(
        newsletter.id,
        newsletter.name,
        newsletter.topic,
        newsletter.learnerLevel,
        newsletter.learnerGoal,
        newsletter.lessonMinutes,
        JSON.stringify(newsletter.sources),
        newsletter.scheduleHour,
        newsletter.scheduleMinute,
        newsletter.timeZone,
        newsletter.active ? 1 : 0,
        newsletter.nextRunAt,
        newsletter.createdAt,
        newsletter.updatedAt,
        newsletter.emailEnabled ? 1 : 0,
        JSON.stringify(newsletter.emailRecipients),
        newsletter.aiExplorationEnabled ? 1 : 0,
        ownerAccountId,
        publicNewsletterSlug,
      );
    return this.getNewsletter(newsletter.id, options);
  }

  listNewsletters(options = {}) {
    const accountId = options.accountId ?? null;
    return this.database
      .prepare(`
        SELECT
          n.*,
          COUNT(i.id) AS issue_count,
          SUM(CASE WHEN i.status = 'generated' THEN 1 ELSE 0 END) AS generated_count,
          (
            SELECT COUNT(*)
            FROM issue_deliveries AS sent
            JOIN issues AS sent_issue ON sent_issue.id = sent.issue_id
            WHERE sent_issue.newsletter_id = n.id
              AND sent.channel = 'email'
              AND sent.status = 'delivered'
          ) AS sent_count,
          (
            SELECT latest.status
            FROM issues AS latest
            WHERE latest.newsletter_id = n.id
            ORDER BY latest.created_at DESC
            LIMIT 1
          ) AS latest_status,
          (
            SELECT latest.completed_at
            FROM issues AS latest
            WHERE latest.newsletter_id = n.id
            ORDER BY latest.created_at DESC
            LIMIT 1
          ) AS latest_completed_at
        FROM newsletters AS n
        LEFT JOIN issues AS i ON i.newsletter_id = n.id
        WHERE (? IS NULL OR n.owner_account_id = ?)
        GROUP BY n.id
        ORDER BY n.created_at ASC
      `)
      .all(accountId, accountId)
      .map(mapNewsletter);
  }

  getNewsletter(id, options = {}) {
    const accountId = options.accountId ?? null;
    const row = this.database
      .prepare(`
        SELECT
          n.*,
          COUNT(i.id) AS issue_count,
          SUM(CASE WHEN i.status = 'generated' THEN 1 ELSE 0 END) AS generated_count,
          (
            SELECT COUNT(*)
            FROM issue_deliveries AS sent
            JOIN issues AS sent_issue ON sent_issue.id = sent.issue_id
            WHERE sent_issue.newsletter_id = n.id
              AND sent.channel = 'email'
              AND sent.status = 'delivered'
          ) AS sent_count
        FROM newsletters AS n
        LEFT JOIN issues AS i ON i.newsletter_id = n.id
        WHERE n.id = ? AND (? IS NULL OR n.owner_account_id = ?)
        GROUP BY n.id
      `)
      .get(id, accountId, accountId);
    return row ? mapNewsletter(row) : null;
  }

  setNewsletterActive(id, active) {
    if (typeof active !== "boolean") {
      throw new Error("Newsletter active state must be a boolean.");
    }
    const newsletter = this.requireNewsletter(id);
    const now = this.now();
    const nextRunAt = active
      ? nextDailyOccurrence(
          now,
          newsletter.timeZone,
          newsletter.scheduleHour,
          newsletter.scheduleMinute,
        ).toISOString()
      : newsletter.nextRunAt;
    this.database
      .prepare(`
        UPDATE newsletters
        SET active = ?, next_run_at = ?, updated_at = ?
        WHERE id = ?
      `)
      .run(active ? 1 : 0, nextRunAt, now.toISOString(), id);
    return this.getNewsletter(id);
  }

  setNewsletterEmail(id, input) {
    this.requireNewsletter(id);
    const settings = normalizeEmailSettings(input);
    const now = this.now().toISOString();
    this.transaction(() => {
      this.database
        .prepare(`
          UPDATE newsletters
          SET email_enabled = ?, email_recipients_json = ?, updated_at = ?
          WHERE id = ?
        `)
        .run(
          settings.enabled ? 1 : 0,
          JSON.stringify(settings.recipients),
          now,
          id,
        );
      if (!settings.enabled) {
        this.database
          .prepare(`
            UPDATE issue_deliveries
            SET status = 'cancelled', updated_at = ?
            WHERE issue_id IN (
              SELECT id FROM issues WHERE newsletter_id = ?
            ) AND channel = 'email' AND status IN ('pending', 'failed')
          `)
          .run(now, id);
      }
    });
    return this.getNewsletter(id);
  }

  setNewsletterContent(id, input) {
    this.requireNewsletter(id);
    if (!input || typeof input !== "object" || Array.isArray(input)) {
      throw new Error("Newsletter content settings must be an object.");
    }
    if (typeof input.aiExplorationEnabled !== "boolean") {
      throw new Error(
        "Newsletter AI Exploration enabled state must be a boolean.",
      );
    }
    this.database
      .prepare(`
        UPDATE newsletters
        SET ai_exploration_enabled = ?, updated_at = ?
        WHERE id = ?
      `)
      .run(
        input.aiExplorationEnabled ? 1 : 0,
        this.now().toISOString(),
        id,
      );
    return this.getNewsletter(id);
  }

  setNewsletterSiteVisible(id, visible) {
    if (typeof visible !== "boolean") {
      throw new Error("Newsletter site visibility must be a boolean.");
    }
    this.requireNewsletter(id);
    this.database
      .prepare(`
        UPDATE newsletters
        SET site_visible = ?, updated_at = ?
        WHERE id = ?
      `)
      .run(visible ? 1 : 0, this.now().toISOString(), id);
    return this.getNewsletter(id);
  }

  enqueueManualIssue(newsletterId) {
    this.requireNewsletter(newsletterId);
    const now = this.now().toISOString();
    const issueId = `issue-${randomUUID()}`;
    this.database
      .prepare(`
        INSERT INTO issues (
          id, newsletter_id, trigger, scheduled_local_date, status, created_at,
          public_id
        ) VALUES (?, ?, 'manual', NULL, 'queued', ?, ?)
      `)
      .run(issueId, newsletterId, now, `dossier-${randomUUID()}`);
    return this.getIssue(issueId);
  }

  dispatchDue(now = this.now()) {
    const due = [];
    this.transaction(() => {
      const newsletters = this.database
        .prepare(`
          SELECT * FROM newsletters
          WHERE active = 1 AND next_run_at <= ?
          ORDER BY next_run_at ASC
        `)
        .all(now.toISOString())
        .map(mapNewsletter);
      const insertIssue = this.database.prepare(`
        INSERT OR IGNORE INTO issues (
          id, newsletter_id, trigger, scheduled_local_date, status, created_at,
          public_id
        ) VALUES (?, ?, 'scheduled', ?, 'queued', ?, ?)
      `);
      const advance = this.database.prepare(`
        UPDATE newsletters
        SET next_run_at = ?, updated_at = ?
        WHERE id = ?
      `);

      for (const newsletter of newsletters) {
        const scheduledFor = new Date(newsletter.nextRunAt);
        const localDate = localDateFor(scheduledFor, newsletter.timeZone);
        const issueId = `scheduled-${newsletter.id}-${localDate}`;
        const result = insertIssue.run(
          issueId,
          newsletter.id,
          localDate,
          now.toISOString(),
          `dossier-${randomUUID()}`,
        );
        if (Number(result.changes) === 1) {
          due.push(this.getIssue(issueId));
        }
        const nextRunAt = nextDailyOccurrence(
          now,
          newsletter.timeZone,
          newsletter.scheduleHour,
          newsletter.scheduleMinute,
        );
        advance.run(nextRunAt.toISOString(), now.toISOString(), newsletter.id);
      }
    });
    return due;
  }

  claimNextIssue(now = this.now()) {
    let claimed = null;
    this.transaction(() => {
      const row = this.database
        .prepare(`
          SELECT queued.* FROM issues AS queued
          WHERE queued.status = 'queued'
            AND NOT EXISTS (
              SELECT 1 FROM issues AS active
              WHERE active.newsletter_id = queued.newsletter_id
                AND active.status = 'generating'
            )
          ORDER BY queued.created_at ASC, queued.id ASC
          LIMIT 1
        `)
        .get();
      if (!row) return;
      const result = this.database
        .prepare(`
          UPDATE issues
          SET status = 'generating', started_at = ?, error = NULL
          WHERE id = ? AND status = 'queued'
        `)
        .run(now.toISOString(), row.id);
      if (Number(result.changes) === 1) {
        claimed = this.getIssue(row.id);
      }
    });
    if (!claimed) return null;
    return {
      ...claimed,
      newsletter: this.requireNewsletter(claimed.newsletterId),
    };
  }

  completeIssue(id, result, now = this.now()) {
    const title = requireText(result.title, "Issue title", 300);
    const generationId = requireText(
      result.generationId,
      "Issue generation ID",
      200,
    );
    const artifactPath = requireText(
      result.artifactPath,
      "Issue artifact path",
      4_000,
    );
    const dossierPath = requireText(
      result.dossierPath,
      "Issue Dossier path",
      4_000,
    );
    const completedAt = now.toISOString();
    this.transaction(() => {
      const update = this.database
        .prepare(`
          UPDATE issues
          SET status = 'generated', dossier_title = ?, generation_id = ?,
              artifact_path = ?, dossier_path = ?, completed_at = ?, error = NULL,
              public_slug = ?
          WHERE id = ? AND status = 'generating'
        `)
        .run(
          title,
          generationId,
          artifactPath,
          dossierPath,
          completedAt,
          publicSlug(title, "dossier"),
          id,
        );
      if (Number(update.changes) !== 1) {
        throw new Error(`Issue ${id} is not generating.`);
      }
      this.database
        .prepare(`
          INSERT OR IGNORE INTO issue_deliveries (
            issue_id, channel, status, created_at, updated_at
          )
          SELECT i.id, 'email', 'pending', ?, ?
          FROM issues AS i
          JOIN newsletters AS n ON n.id = i.newsletter_id
          WHERE i.id = ? AND n.email_enabled = 1
        `)
        .run(completedAt, completedAt, id);
    });
    return this.getIssue(id);
  }

  failIssue(id, error, now = this.now()) {
    const message = String(error?.message ?? error ?? "Unknown generation error")
      .replace(/\s+/g, " ")
      .trim()
      .slice(0, 500);
    const update = this.database
      .prepare(`
        UPDATE issues
        SET status = 'failed', error = ?, completed_at = ?
        WHERE id = ? AND status = 'generating'
      `)
      .run(message, now.toISOString(), id);
    if (Number(update.changes) !== 1) {
      throw new Error(`Issue ${id} is not generating.`);
    }
    return this.getIssue(id);
  }

  listIssues(newsletterId, options = {}) {
    this.requireNewsletter(newsletterId, options);
    const limit = boundedInteger(options.limit ?? 50, 1, 100, "Issue limit");
    const offset = boundedInteger(options.offset ?? 0, 0, 1_000_000, "Issue offset");
    return this.database
      .prepare(`
        SELECT i.*, d.status AS delivery_status,
          d.attempt_count AS delivery_attempt_count,
          d.external_id AS delivery_external_id,
          d.error AS delivery_error,
          d.started_at AS delivery_started_at,
          d.completed_at AS delivery_completed_at
        FROM issues AS i
        LEFT JOIN issue_deliveries AS d
          ON d.issue_id = i.id AND d.channel = 'email'
        WHERE i.newsletter_id = ?
        ORDER BY i.created_at DESC, i.id DESC
        LIMIT ? OFFSET ?
      `)
      .all(newsletterId, limit, offset)
      .map(mapIssue);
  }

  getIssue(id, options = {}) {
    const accountId = options.accountId ?? null;
    const row = this.database
      .prepare(`
        SELECT i.*, d.status AS delivery_status,
          d.attempt_count AS delivery_attempt_count,
          d.external_id AS delivery_external_id,
          d.error AS delivery_error,
          d.started_at AS delivery_started_at,
          d.completed_at AS delivery_completed_at
        FROM issues AS i
        JOIN newsletters AS n ON n.id = i.newsletter_id
        LEFT JOIN issue_deliveries AS d
          ON d.issue_id = i.id AND d.channel = 'email'
        WHERE i.id = ? AND (? IS NULL OR n.owner_account_id = ?)
      `)
      .get(id, accountId, accountId);
    return row ? mapIssue(row) : null;
  }

  setIssuePublication(id, state) {
    if (!["published", "hidden"].includes(state)) {
      throw new Error("Issue publication state must be published or hidden.");
    }
    const issue = this.getIssue(id);
    if (!issue) throw new Error(`Issue ${id} was not found.`);
    this.database
      .prepare("UPDATE issues SET publication_state = ? WHERE id = ?")
      .run(state, id);
    return this.getIssue(id);
  }

  claimNextDelivery(now = this.now()) {
    let claimed = null;
    this.transaction(() => {
      const row = this.database
        .prepare(`
          SELECT d.* FROM issue_deliveries AS d
          JOIN issues AS i ON i.id = d.issue_id
          JOIN newsletters AS n ON n.id = i.newsletter_id
          WHERE d.status = 'pending' AND n.email_enabled = 1
          ORDER BY d.created_at ASC, d.issue_id ASC
          LIMIT 1
        `)
        .get();
      if (!row) return;
      const result = this.database
        .prepare(`
          UPDATE issue_deliveries
          SET status = 'delivering', attempt_count = attempt_count + 1,
              started_at = ?, completed_at = NULL, error = NULL, updated_at = ?
          WHERE issue_id = ? AND channel = 'email' AND status = 'pending'
            AND EXISTS (
              SELECT 1
              FROM issues AS i
              JOIN newsletters AS n ON n.id = i.newsletter_id
              WHERE i.id = issue_deliveries.issue_id
                AND n.email_enabled = 1
            )
        `)
        .run(now.toISOString(), now.toISOString(), row.issue_id);
      if (Number(result.changes) === 1) {
        const issue = this.getIssue(row.issue_id);
        claimed = {
          ...issue.delivery,
          issue,
          newsletter: this.requireNewsletter(issue.newsletterId),
        };
      }
    });
    return claimed;
  }

  completeDelivery(issueId, externalId, now = this.now()) {
    const providerId = requireText(externalId, "Delivery external ID", 500);
    const result = this.database
      .prepare(`
        UPDATE issue_deliveries
        SET status = 'delivered', external_id = ?, error = NULL,
            completed_at = ?, updated_at = ?
        WHERE issue_id = ? AND channel = 'email' AND status = 'delivering'
      `)
      .run(providerId, now.toISOString(), now.toISOString(), issueId);
    if (Number(result.changes) !== 1) {
      throw new Error(`Email delivery for Issue ${issueId} is not delivering.`);
    }
    return this.getIssue(issueId).delivery;
  }

  failDelivery(issueId, error, now = this.now()) {
    const message = safeError(error, "Unknown delivery error");
    const result = this.database
      .prepare(`
        UPDATE issue_deliveries
        SET status = 'failed', error = ?, completed_at = ?, updated_at = ?
        WHERE issue_id = ? AND channel = 'email' AND status = 'delivering'
      `)
      .run(message, now.toISOString(), now.toISOString(), issueId);
    if (Number(result.changes) !== 1) {
      throw new Error(`Email delivery for Issue ${issueId} is not delivering.`);
    }
    return this.getIssue(issueId).delivery;
  }

  markDeliveryUnknown(issueId, error, now = this.now()) {
    const message = safeError(error, "Unknown provider delivery outcome");
    const result = this.database
      .prepare(`
        UPDATE issue_deliveries
        SET status = 'unknown', error = ?, completed_at = ?, updated_at = ?
        WHERE issue_id = ? AND channel = 'email' AND status = 'delivering'
      `)
      .run(message, now.toISOString(), now.toISOString(), issueId);
    if (Number(result.changes) !== 1) {
      throw new Error(`Email delivery for Issue ${issueId} is not delivering.`);
    }
    return this.getIssue(issueId).delivery;
  }

  retryDelivery(issueId, now = this.now()) {
    const result = this.database
      .prepare(`
        UPDATE issue_deliveries
        SET status = 'pending', external_id = NULL, error = NULL,
            started_at = NULL, completed_at = NULL, updated_at = ?
        WHERE issue_id = ? AND channel = 'email' AND status = 'failed'
          AND EXISTS (
            SELECT 1
            FROM issues AS i
            JOIN newsletters AS n ON n.id = i.newsletter_id
            WHERE i.id = issue_deliveries.issue_id
              AND n.email_enabled = 1
          )
      `)
      .run(now.toISOString(), issueId);
    if (Number(result.changes) !== 1) {
      throw new Error(`Email delivery for Issue ${issueId} is not retryable.`);
    }
    return this.getIssue(issueId).delivery;
  }

  close() {
    this.database.close();
  }

  requireNewsletter(id, options = {}) {
    const newsletter = this.getNewsletter(id, options);
    if (!newsletter) throw new Error(`Newsletter ${id} was not found.`);
    return newsletter;
  }

  requireActiveAccount(id) {
    const account = this.database
      .prepare("SELECT * FROM accounts WHERE id = ? AND status = 'active'")
      .get(id);
    if (!account) throw new Error(`Account ${id} was not found or is inactive.`);
    return mapAccount(account);
  }

  allocateNewsletterSlug(ownerAccountId, name) {
    const base = publicSlug(name, "topic");
    let candidate = base;
    let suffix = 2;
    const exists = this.database.prepare(`
      SELECT 1 FROM newsletters
      WHERE owner_account_id IS ?
        AND public_slug = ? COLLATE NOCASE
    `);
    while (exists.get(ownerAccountId, candidate)) {
      candidate = `${base}-${suffix}`;
      suffix += 1;
    }
    return candidate;
  }

  transaction(callback) {
    this.database.exec("BEGIN IMMEDIATE");
    try {
      const result = callback();
      this.database.exec("COMMIT");
      return result;
    } catch (error) {
      this.database.exec("ROLLBACK");
      throw error;
    }
  }
}

export function nextDailyOccurrence(after, timeZone, hour, minute) {
  validateTimeZone(timeZone);
  const scheduleHour = boundedInteger(hour, 0, 23, "Schedule hour");
  const scheduleMinute = boundedInteger(minute, 0, 59, "Schedule minute");
  let candidateMs = Math.floor(after.valueOf() / 60_000) * 60_000 + 60_000;
  for (let index = 0; index < MAX_SCHEDULE_SCAN_MINUTES; index += 1) {
    const candidate = new Date(candidateMs);
    const parts = localParts(candidate, timeZone);
    if (parts.hour === scheduleHour && parts.minute === scheduleMinute) {
      return candidate;
    }
    candidateMs += 60_000;
  }
  throw new Error(
    `Could not find the next ${pad(scheduleHour)}:${pad(scheduleMinute)} occurrence in ${timeZone}.`,
  );
}

function normalizeNewsletter(input, now) {
  if (!input || typeof input !== "object" || Array.isArray(input)) {
    throw new Error("Newsletter input must be an object.");
  }
  const name = requireText(input.name, "Newsletter name", 100);
  const topic = requireText(input.topic, "Newsletter topic", 500);
  const timeZone = requireText(input.timeZone, "Newsletter timezone", 100);
  validateTimeZone(timeZone);
  const { hour, minute } = parseSchedule(input.scheduleTime);
  if (!Array.isArray(input.sources) || input.sources.length === 0) {
    throw new Error("Newsletter requires at least one Source Item feed.");
  }
  const sources = input.sources.map((source, index) =>
    normalizeSource(source, index),
  );
  const id = input.id
    ? requireSlug(input.id, "Newsletter ID")
    : `newsletter-${randomUUID()}`;
  const active = input.active ?? true;
  if (typeof active !== "boolean") {
    throw new Error("Newsletter active state must be a boolean.");
  }
  const createdAt = now.toISOString();
  const email = normalizeEmailSettings({
    enabled: input.emailEnabled ?? false,
    recipients: input.emailRecipients ?? [],
  });
  const aiExplorationEnabled = input.aiExplorationEnabled ?? false;
  if (typeof aiExplorationEnabled !== "boolean") {
    throw new Error(
      "Newsletter AI Exploration enabled state must be a boolean.",
    );
  }
  return {
    id,
    name,
    topic,
    learnerLevel: requireText(
      input.learnerLevel ?? "curious generalist",
      "Learner level",
      200,
    ),
    learnerGoal: requireText(
      input.learnerGoal ?? `build durable understanding of ${topic}`,
      "Learner goal",
      500,
    ),
    lessonMinutes: boundedInteger(
      input.lessonMinutes ?? 15,
      5,
      90,
      "Lesson minutes",
    ),
    sources,
    scheduleHour: hour,
    scheduleMinute: minute,
    timeZone,
    active,
    emailEnabled: email.enabled,
    emailRecipients: email.recipients,
    aiExplorationEnabled,
    nextRunAt: nextDailyOccurrence(now, timeZone, hour, minute).toISOString(),
    createdAt,
    updatedAt: createdAt,
  };
}

function normalizeSource(source, index) {
  if (!source || typeof source !== "object" || Array.isArray(source)) {
    throw new Error(`Newsletter source ${index + 1} must be an object.`);
  }
  const url = requireText(source.url, `Newsletter source ${index + 1} URL`, 2_000);
  let parsed;
  try {
    parsed = new URL(url);
  } catch {
    throw new Error(`Newsletter source ${index + 1} URL must be valid.`);
  }
  if (!["http:", "https:"].includes(parsed.protocol)) {
    throw new Error(`Newsletter source ${index + 1} must use HTTP or HTTPS.`);
  }
  return {
    name: requireText(
      source.name ?? parsed.hostname,
      `Newsletter source ${index + 1} name`,
      200,
    ),
    url: parsed.toString(),
    limit: boundedInteger(
      source.limit ?? 10,
      1,
      50,
      `Newsletter source ${index + 1} limit`,
    ),
  };
}

function mapNewsletter(row) {
  return {
    id: row.id,
    name: row.name,
    topic: row.topic,
    learnerLevel: row.learner_level,
    learnerGoal: row.learner_goal,
    lessonMinutes: Number(row.lesson_minutes),
    sources: JSON.parse(row.sources_json),
    scheduleHour: Number(row.schedule_hour),
    scheduleMinute: Number(row.schedule_minute),
    scheduleTime: `${pad(row.schedule_hour)}:${pad(row.schedule_minute)}`,
    timeZone: row.time_zone,
    active: Boolean(row.active),
    nextRunAt: row.next_run_at,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    issueCount: Number(row.issue_count ?? 0),
    generatedCount: Number(row.generated_count ?? 0),
    latestStatus: row.latest_status ?? null,
    latestCompletedAt: row.latest_completed_at ?? null,
    emailEnabled: Boolean(row.email_enabled),
    emailRecipients: JSON.parse(row.email_recipients_json ?? "[]"),
    sentCount: Number(row.sent_count ?? 0),
    aiExplorationEnabled: Boolean(row.ai_exploration_enabled),
    ownerAccountId: row.owner_account_id ?? null,
    publicSlug: row.public_slug ?? null,
    siteVisible: Boolean(row.site_visible),
  };
}

function mapAccount(row) {
  return {
    id: row.id,
    clerkUserId: row.clerk_user_id,
    status: row.status,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  };
}

function mapSite(row) {
  return {
    id: row.id,
    ownerAccountId: row.owner_account_id,
    username: row.username,
    displayName: row.display_name,
    description: row.description,
    visibility: row.visibility,
    claimedAt: row.claimed_at,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
  };
}

function mapIssue(row) {
  if (!ISSUE_STATES.has(row.status)) {
    throw new Error(`Issue ${row.id} has invalid status ${row.status}.`);
  }
  return {
    id: row.id,
    newsletterId: row.newsletter_id,
    trigger: row.trigger,
    scheduledLocalDate: row.scheduled_local_date,
    status: row.status,
    title: row.dossier_title,
    generationId: row.generation_id,
    artifactPath: row.artifact_path,
    dossierPath: row.dossier_path,
    error: row.error,
    createdAt: row.created_at,
    startedAt: row.started_at,
    completedAt: row.completed_at,
    publicId: row.public_id ?? null,
    publicSlug: row.public_slug ?? null,
    publicationState: row.publication_state ?? "published",
    delivery: row.delivery_status
      ? {
          issueId: row.id,
          channel: "email",
          status: validateDeliveryStatus(row.delivery_status, row.id),
          attemptCount: Number(row.delivery_attempt_count),
          externalId: row.delivery_external_id,
          error: row.delivery_error,
          startedAt: row.delivery_started_at,
          completedAt: row.delivery_completed_at,
        }
      : null,
  };
}

function mapPublicIssue(row) {
  return {
    ...mapIssue(row),
    newsletterName: row.newsletter_name,
    newsletterPublicSlug: row.newsletter_public_slug,
  };
}

function normalizeEmailSettings(input) {
  if (!input || typeof input !== "object" || Array.isArray(input)) {
    throw new Error("Newsletter email settings must be an object.");
  }
  const enabled = input.enabled ?? false;
  if (typeof enabled !== "boolean") {
    throw new Error("Newsletter email enabled state must be a boolean.");
  }
  if (!Array.isArray(input.recipients)) {
    throw new Error("Newsletter email recipients must be an array.");
  }
  if (input.recipients.length > 20) {
    throw new Error("Newsletter email supports at most 20 recipients.");
  }
  const recipients = [
    ...new Set(
      input.recipients.map((value) => {
        const recipient = requireText(value, "Newsletter email recipient", 320)
          .toLowerCase();
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(recipient)) {
          throw new Error(`Invalid Newsletter email recipient: ${recipient}`);
        }
        return recipient;
      }),
    ),
  ];
  if (enabled && recipients.length === 0) {
    throw new Error(
      "Newsletter email requires at least one recipient when enabled.",
    );
  }
  return { enabled, recipients };
}

function validateDeliveryStatus(status, issueId) {
  if (!DELIVERY_STATES.has(status)) {
    throw new Error(`Issue ${issueId} has invalid delivery status ${status}.`);
  }
  return status;
}

function safeError(error, fallback) {
  return String(error?.message ?? error ?? fallback)
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 500);
}

function parseSchedule(value) {
  const normalized = requireText(value, "Newsletter schedule", 5);
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(normalized);
  if (!match) {
    throw new Error("Newsletter schedule must use 24-hour HH:MM format.");
  }
  return { hour: Number(match[1]), minute: Number(match[2]) };
}

function localDateFor(date, timeZone) {
  const parts = localParts(date, timeZone);
  return `${parts.year}-${pad(parts.month)}-${pad(parts.day)}`;
}

function localParts(date, timeZone) {
  const values = {};
  for (const part of new Intl.DateTimeFormat("en-CA", {
    timeZone,
    hourCycle: "h23",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).formatToParts(date)) {
    if (part.type !== "literal") values[part.type] = Number(part.value);
  }
  return values;
}

function validateTimeZone(value) {
  try {
    new Intl.DateTimeFormat("en", { timeZone: value }).format();
  } catch {
    throw new Error(`Invalid Newsletter timezone: ${value}`);
  }
}

function requireText(value, field, maximum) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} must be a non-empty string.`);
  }
  const normalized = value.trim();
  if (normalized.length > maximum) {
    throw new Error(`${field} must be at most ${maximum} characters.`);
  }
  return normalized;
}

function optionalText(value, field, maximum) {
  if (typeof value !== "string") {
    throw new Error(`${field} must be a string.`);
  }
  const normalized = value.trim();
  if (normalized.length > maximum) {
    throw new Error(`${field} must be at most ${maximum} characters.`);
  }
  return normalized;
}

function publicSlug(value, fallback) {
  const normalized = String(value ?? "")
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80)
    .replace(/-+$/g, "");
  return normalized || fallback;
}

function uniqueSlug(base, used) {
  if (!used.has(base)) return base;
  let suffix = 2;
  while (used.has(`${base}-${suffix}`)) suffix += 1;
  return `${base}-${suffix}`;
}

function requireSlug(value, field) {
  const normalized = requireText(value, field, 64);
  if (!/^[a-z0-9][a-z0-9_-]*$/.test(normalized)) {
    throw new Error(`${field} must be a lowercase slug.`);
  }
  return normalized;
}

function boundedInteger(value, minimum, maximum, field) {
  if (!Number.isInteger(value) || value < minimum || value > maximum) {
    throw new Error(`${field} must be an integer from ${minimum} to ${maximum}.`);
  }
  return value;
}

function pad(value) {
  return String(value).padStart(2, "0");
}

function execWithBusyRetry(database, statement, timeoutMilliseconds) {
  const deadline = Date.now() + timeoutMilliseconds;
  while (true) {
    try {
      database.exec(statement);
      return;
    } catch (error) {
      if (error?.errcode !== 5 || Date.now() >= deadline) throw error;
      Atomics.wait(
        new Int32Array(new SharedArrayBuffer(4)),
        0,
        0,
        Math.min(25, Math.max(1, deadline - Date.now())),
      );
    }
  }
}
