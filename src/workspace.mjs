import { randomUUID } from "node:crypto";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { DatabaseSync } from "node:sqlite";

const ISSUE_STATES = new Set(["queued", "generating", "generated", "failed"]);
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
    this.initialize();
  }

  initialize() {
    this.database.exec(`
      PRAGMA foreign_keys = ON;
      PRAGMA busy_timeout = 5000;
      PRAGMA journal_mode = WAL;
    `);
    const currentVersion = Number(
      this.database.prepare("PRAGMA user_version").get().user_version,
    );
    if (currentVersion > 1) {
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
    if (currentVersion === 0) {
      this.database.exec("PRAGMA user_version = 1");
    }
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

  createNewsletter(input) {
    const now = this.now();
    const newsletter = normalizeNewsletter(input, now);
    this.database
      .prepare(`
        INSERT INTO newsletters (
          id, name, topic, learner_level, learner_goal, lesson_minutes,
          sources_json, schedule_hour, schedule_minute, time_zone, active,
          next_run_at, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
      );
    return this.getNewsletter(newsletter.id);
  }

  listNewsletters() {
    return this.database
      .prepare(`
        SELECT
          n.*,
          COUNT(i.id) AS issue_count,
          SUM(CASE WHEN i.status = 'generated' THEN 1 ELSE 0 END) AS generated_count,
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
        GROUP BY n.id
        ORDER BY n.created_at ASC
      `)
      .all()
      .map(mapNewsletter);
  }

  getNewsletter(id) {
    const row = this.database
      .prepare(`
        SELECT
          n.*,
          COUNT(i.id) AS issue_count,
          SUM(CASE WHEN i.status = 'generated' THEN 1 ELSE 0 END) AS generated_count
        FROM newsletters AS n
        LEFT JOIN issues AS i ON i.newsletter_id = n.id
        WHERE n.id = ?
        GROUP BY n.id
      `)
      .get(id);
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

  enqueueManualIssue(newsletterId) {
    this.requireNewsletter(newsletterId);
    const now = this.now().toISOString();
    const issueId = `issue-${randomUUID()}`;
    this.database
      .prepare(`
        INSERT INTO issues (
          id, newsletter_id, trigger, scheduled_local_date, status, created_at
        ) VALUES (?, ?, 'manual', NULL, 'queued', ?)
      `)
      .run(issueId, newsletterId, now);
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
          id, newsletter_id, trigger, scheduled_local_date, status, created_at
        ) VALUES (?, ?, 'scheduled', ?, 'queued', ?)
      `);
      const advance = this.database.prepare(`
        UPDATE newsletters
        SET next_run_at = ?, updated_at = ?
        WHERE id = ?
      `);

      for (const newsletter of newsletters) {
        const localDate = localDateFor(now, newsletter.timeZone);
        const issueId = `scheduled-${newsletter.id}-${localDate}`;
        const result = insertIssue.run(
          issueId,
          newsletter.id,
          localDate,
          now.toISOString(),
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
          SELECT * FROM issues
          WHERE status = 'queued'
          ORDER BY created_at ASC, id ASC
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
    const update = this.database
      .prepare(`
        UPDATE issues
        SET status = 'generated', dossier_title = ?, generation_id = ?,
            artifact_path = ?, dossier_path = ?, completed_at = ?, error = NULL
        WHERE id = ? AND status = 'generating'
      `)
      .run(
        title,
        generationId,
        artifactPath,
        dossierPath,
        now.toISOString(),
        id,
      );
    if (Number(update.changes) !== 1) {
      throw new Error(`Issue ${id} is not generating.`);
    }
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
    this.requireNewsletter(newsletterId);
    const limit = boundedInteger(options.limit ?? 50, 1, 100, "Issue limit");
    const offset = boundedInteger(options.offset ?? 0, 0, 1_000_000, "Issue offset");
    return this.database
      .prepare(`
        SELECT * FROM issues
        WHERE newsletter_id = ?
        ORDER BY created_at DESC, id DESC
        LIMIT ? OFFSET ?
      `)
      .all(newsletterId, limit, offset)
      .map(mapIssue);
  }

  getIssue(id) {
    const row = this.database.prepare("SELECT * FROM issues WHERE id = ?").get(id);
    return row ? mapIssue(row) : null;
  }

  close() {
    this.database.close();
  }

  requireNewsletter(id) {
    const newsletter = this.getNewsletter(id);
    if (!newsletter) throw new Error(`Newsletter ${id} was not found.`);
    return newsletter;
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
  };
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
