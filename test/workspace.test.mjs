import assert from "node:assert/strict";
import { mkdtemp } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { DatabaseSync } from "node:sqlite";
import test from "node:test";
import { SQLiteWorkspace, nextDailyOccurrence } from "../src/workspace.mjs";

test("SQLiteWorkspace initializes idempotently with safety pragmas", () => {
  const workspace = new SQLiteWorkspace(":memory:");
  workspace.initialize();
  const diagnostics = workspace.diagnostics();
  assert.equal(diagnostics.userVersion, 2);
  assert.equal(diagnostics.foreignKeys, true);
  assert.equal(diagnostics.busyTimeout, 5_000);
  assert.match(diagnostics.journalMode, /^(memory|wal)$/);
  workspace.close();
});

test("SQLiteWorkspace rejects a newer schema version", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-schema-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const workspace = new SQLiteWorkspace(databasePath);
  workspace.database.exec("PRAGMA user_version = 3");
  workspace.close();
  assert.throws(
    () => new SQLiteWorkspace(databasePath),
    /newer than this Learnloom release/,
  );
});

test("SQLiteWorkspace migrates schema v1 without losing data", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-migrate-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const legacy = new DatabaseSync(databasePath);
  legacy.exec(`
    PRAGMA foreign_keys = ON;
    CREATE TABLE newsletters (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      topic TEXT NOT NULL,
      learner_level TEXT NOT NULL,
      learner_goal TEXT NOT NULL,
      lesson_minutes INTEGER NOT NULL,
      sources_json TEXT NOT NULL,
      schedule_hour INTEGER NOT NULL,
      schedule_minute INTEGER NOT NULL,
      time_zone TEXT NOT NULL,
      active INTEGER NOT NULL,
      next_run_at TEXT NOT NULL,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    ) STRICT;
    CREATE TABLE issues (
      id TEXT PRIMARY KEY,
      newsletter_id TEXT NOT NULL REFERENCES newsletters(id) ON DELETE CASCADE,
      trigger TEXT NOT NULL,
      scheduled_local_date TEXT,
      status TEXT NOT NULL,
      dossier_title TEXT,
      generation_id TEXT,
      artifact_path TEXT,
      dossier_path TEXT,
      error TEXT,
      created_at TEXT NOT NULL,
      started_at TEXT,
      completed_at TEXT
    ) STRICT;
    INSERT INTO newsletters VALUES (
      'legacy', 'Legacy', 'Queues', 'experienced', 'learn', 15,
      '[{"name":"Feed","url":"https://example.com/feed","limit":10}]',
      10, 0, 'Asia/Kolkata', 1, '2026-07-19T04:30:00.000Z',
      '2026-07-18T00:00:00.000Z', '2026-07-18T00:00:00.000Z'
    );
    INSERT INTO issues (
      id, newsletter_id, trigger, status, created_at
    ) VALUES ('legacy-issue', 'legacy', 'manual', 'queued',
      '2026-07-18T00:00:00.000Z');
    PRAGMA user_version = 1;
  `);
  legacy.close();

  const workspace = new SQLiteWorkspace(databasePath);
  assert.equal(workspace.diagnostics().userVersion, 2);
  assert.equal(workspace.getNewsletter("legacy").emailEnabled, false);
  assert.deepEqual(workspace.getNewsletter("legacy").emailRecipients, []);
  assert.equal(workspace.getIssue("legacy-issue").status, "queued");
  workspace.close();
});

test("SQLiteWorkspace creates isolated Newsletters and dashboard counts", () => {
  const workspace = fixtureWorkspace();
  const first = workspace.createNewsletter(newsletterInput("RabbitMQ at depth"));
  const second = workspace.createNewsletter(
    newsletterInput("PostgreSQL internals", {
      topic: "PostgreSQL query planning",
      scheduleTime: "11:30",
    }),
  );
  workspace.enqueueManualIssue(first.id);
  workspace.enqueueManualIssue(first.id);
  workspace.enqueueManualIssue(second.id);

  const newsletters = workspace.listNewsletters();
  const firstSummary = newsletters.find((item) => item.id === first.id);
  const secondSummary = newsletters.find((item) => item.id === second.id);
  assert.equal(newsletters.length, 2);
  assert.equal(firstSummary.issueCount, 2);
  assert.equal(secondSummary.issueCount, 1);
  assert.notEqual(first.id, second.id);
  workspace.close();
});

test("SQLiteWorkspace dispatches one scheduled Issue and respects pause state", () => {
  let now = new Date("2026-07-18T03:59:00.000Z");
  const workspace = new SQLiteWorkspace(":memory:", { now: () => now });
  const dueSoon = workspace.createNewsletter(
    newsletterInput("RabbitMQ", {
      scheduleTime: "09:30",
      timeZone: "Asia/Kolkata",
    }),
  );
  const paused = workspace.createNewsletter(
    newsletterInput("Kafka", {
      scheduleTime: "09:30",
      timeZone: "Asia/Kolkata",
    }),
  );
  workspace.setNewsletterActive(paused.id, false);

  now = new Date("2026-07-18T04:00:00.000Z");
  const firstDispatch = workspace.dispatchDue(now);
  const secondDispatch = workspace.dispatchDue(now);
  assert.equal(firstDispatch.length, 1);
  assert.equal(firstDispatch[0].newsletterId, dueSoon.id);
  assert.equal(secondDispatch.length, 0);
  assert.equal(workspace.listIssues(paused.id).length, 0);
  workspace.close();
});

test("overdue dispatch preserves the missed schedule date without queueing today early", () => {
  let now = new Date("2026-07-17T03:00:00.000Z");
  const workspace = new SQLiteWorkspace(":memory:", { now: () => now });
  const newsletter = workspace.createNewsletter(
    newsletterInput("RabbitMQ", {
      scheduleTime: "10:00",
      timeZone: "Asia/Kolkata",
    }),
  );

  now = new Date("2026-07-18T03:00:00.000Z");
  const overdue = workspace.dispatchDue(now);
  assert.equal(overdue.length, 1);
  assert.equal(overdue[0].scheduledLocalDate, "2026-07-17");
  assert.equal(workspace.dispatchDue(now).length, 0);

  now = new Date("2026-07-18T04:30:00.000Z");
  const today = workspace.dispatchDue(now);
  assert.equal(today.length, 1);
  assert.equal(today[0].newsletterId, newsletter.id);
  assert.equal(today[0].scheduledLocalDate, "2026-07-18");
  workspace.close();
});

test("SQLiteWorkspace atomically claims a queued Issue once", () => {
  const workspace = fixtureWorkspace();
  const newsletter = workspace.createNewsletter(newsletterInput("RabbitMQ"));
  const queued = workspace.enqueueManualIssue(newsletter.id);
  const claimed = workspace.claimNextIssue();
  const secondClaim = workspace.claimNextIssue();
  assert.equal(claimed.id, queued.id);
  assert.equal(claimed.status, "generating");
  assert.equal(claimed.newsletter.id, newsletter.id);
  assert.equal(secondClaim, null);
  workspace.close();
});

test("two SQLite connections cannot claim the same queued Issue", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-sqlite-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const first = new SQLiteWorkspace(databasePath);
  const second = new SQLiteWorkspace(databasePath);
  const newsletter = first.createNewsletter(newsletterInput("RabbitMQ"));
  const queued = first.enqueueManualIssue(newsletter.id);
  assert.equal(first.claimNextIssue().id, queued.id);
  assert.equal(second.claimNextIssue(), null);
  first.close();
  second.close();
});

test("two SQLite connections serialize Issues for one Newsletter", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-serial-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const first = new SQLiteWorkspace(databasePath);
  const second = new SQLiteWorkspace(databasePath);
  const newsletter = first.createNewsletter(newsletterInput("RabbitMQ"));
  first.enqueueManualIssue(newsletter.id);
  first.enqueueManualIssue(newsletter.id);

  const firstClaim = first.claimNextIssue();
  assert.equal(second.claimNextIssue(), null);
  first.completeIssue(firstClaim.id, {
    title: "First",
    generationId: "generation-1",
    artifactPath: "/tmp/first.md",
    dossierPath: "/tmp/first.json",
  });
  const secondClaim = second.claimNextIssue();
  assert.equal(secondClaim.newsletterId, newsletter.id);
  assert.notEqual(secondClaim.id, firstClaim.id);
  first.close();
  second.close();
});

test("two SQLite connections may claim different Newsletters", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-parallel-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const first = new SQLiteWorkspace(databasePath);
  const second = new SQLiteWorkspace(databasePath);
  const rabbit = first.createNewsletter(newsletterInput("RabbitMQ"));
  const postgres = first.createNewsletter(
    newsletterInput("PostgreSQL", { topic: "PostgreSQL planning" }),
  );
  first.enqueueManualIssue(rabbit.id);
  first.enqueueManualIssue(postgres.id);
  const firstClaim = first.claimNextIssue();
  const secondClaim = second.claimNextIssue();
  assert.notEqual(firstClaim.newsletterId, secondClaim.newsletterId);
  first.close();
  second.close();
});

test("SQLiteWorkspace completes and fails Issue lifecycle transitions", () => {
  const workspace = fixtureWorkspace();
  const newsletter = workspace.createNewsletter(newsletterInput("RabbitMQ"));
  const generatedClaim = claimManual(workspace, newsletter.id);
  const generated = workspace.completeIssue(generatedClaim.id, {
    title: "Queues and backpressure",
    generationId: "generation-1",
    artifactPath: "/tmp/issue.md",
    dossierPath: "/tmp/issue.json",
  });
  assert.equal(generated.status, "generated");
  assert.equal(generated.title, "Queues and backpressure");

  const failedClaim = claimManual(workspace, newsletter.id);
  const failed = workspace.failIssue(
    failedClaim.id,
    new Error("provider\nfailed with a long but safe error"),
  );
  assert.equal(failed.status, "failed");
  assert.equal(failed.error, "provider failed with a long but safe error");
  workspace.close();
});

test("Newsletter email settings are normalized and validated", () => {
  const workspace = fixtureWorkspace();
  const newsletter = workspace.createNewsletter(
    newsletterInput("RabbitMQ", {
      emailEnabled: true,
      emailRecipients: ["READER@example.com", "reader@example.com"],
    }),
  );
  assert.equal(newsletter.emailEnabled, true);
  assert.deepEqual(newsletter.emailRecipients, ["reader@example.com"]);

  const disabled = workspace.setNewsletterEmail(newsletter.id, {
    enabled: false,
    recipients: ["next@example.com"],
  });
  assert.equal(disabled.emailEnabled, false);
  assert.deepEqual(disabled.emailRecipients, ["next@example.com"]);
  assert.throws(
    () =>
      workspace.setNewsletterEmail(newsletter.id, {
        enabled: true,
        recipients: [],
      }),
    /at least one recipient/,
  );
  assert.throws(
    () =>
      workspace.setNewsletterEmail(newsletter.id, {
        enabled: true,
        recipients: ["not-an-email"],
      }),
    /Invalid Newsletter email recipient/,
  );
  workspace.close();
});

test("Issue completion atomically queues and completes an email Delivery Receipt", () => {
  const workspace = fixtureWorkspace();
  const newsletter = workspace.createNewsletter(
    newsletterInput("RabbitMQ", {
      emailEnabled: true,
      emailRecipients: ["reader@example.com"],
    }),
  );
  const claimed = claimManual(workspace, newsletter.id);
  const generated = workspace.completeIssue(claimed.id, {
    title: "Queues and backpressure",
    generationId: "generation-1",
    artifactPath: "/tmp/issue.md",
    dossierPath: "/tmp/issue.json",
  });
  assert.equal(generated.status, "generated");
  assert.equal(generated.delivery.status, "pending");

  const delivery = workspace.claimNextDelivery();
  assert.equal(delivery.issue.id, claimed.id);
  assert.equal(delivery.newsletter.emailRecipients[0], "reader@example.com");
  assert.equal(delivery.status, "delivering");
  assert.equal(delivery.attemptCount, 1);

  const completed = workspace.completeDelivery(claimed.id, "email-provider-1");
  assert.equal(completed.status, "delivered");
  assert.equal(completed.externalId, "email-provider-1");
  assert.equal(workspace.getNewsletter(newsletter.id).sentCount, 1);
  workspace.close();
});

test("two connections claim a Delivery Receipt once and failed delivery retries", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-delivery-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const first = new SQLiteWorkspace(databasePath);
  const second = new SQLiteWorkspace(databasePath);
  const newsletter = first.createNewsletter(
    newsletterInput("RabbitMQ", {
      emailEnabled: true,
      emailRecipients: ["reader@example.com"],
    }),
  );
  const claimedIssue = claimManual(first, newsletter.id);
  first.completeIssue(claimedIssue.id, {
    title: "Queues",
    generationId: "generation-1",
    artifactPath: "/tmp/issue.md",
    dossierPath: "/tmp/issue.json",
  });

  assert.equal(first.claimNextDelivery().issueId, claimedIssue.id);
  assert.equal(second.claimNextDelivery(), null);
  const failed = first.failDelivery(
    claimedIssue.id,
    new Error("provider\nunavailable"),
  );
  assert.equal(failed.status, "failed");
  assert.equal(failed.error, "provider unavailable");
  assert.equal(first.claimNextDelivery(), null);

  assert.equal(first.retryDelivery(claimedIssue.id).status, "pending");
  assert.equal(second.claimNextDelivery().attemptCount, 2);
  first.close();
  second.close();
});

test("nextDailyOccurrence handles DST gaps and repeated times", () => {
  const skippedGap = nextDailyOccurrence(
    new Date("2026-03-08T06:55:00.000Z"),
    "America/New_York",
    2,
    30,
  );
  assert.equal(skippedGap.toISOString(), "2026-03-09T06:30:00.000Z");

  const firstRepeatedTime = nextDailyOccurrence(
    new Date("2026-11-01T04:59:00.000Z"),
    "America/New_York",
    1,
    30,
  );
  assert.equal(firstRepeatedTime.toISOString(), "2026-11-01T05:30:00.000Z");
});

test("SQLiteWorkspace validates Newsletter input", () => {
  const workspace = fixtureWorkspace();
  assert.throws(
    () =>
      workspace.createNewsletter(
        newsletterInput("Bad source", {
          sources: [{ name: "File", url: "file:///etc/passwd" }],
        }),
      ),
    /HTTP or HTTPS/,
  );
  assert.throws(
    () =>
      workspace.createNewsletter(
        newsletterInput("Bad time", { scheduleTime: "25:99" }),
      ),
    /HH:MM/,
  );
  workspace.close();
});

function fixtureWorkspace() {
  const now = new Date("2026-07-18T03:00:00.000Z");
  return new SQLiteWorkspace(":memory:", { now: () => now });
}

function newsletterInput(name, overrides = {}) {
  return {
    name,
    topic: "RabbitMQ messaging queues",
    learnerLevel: "technically experienced",
    learnerGoal: "understand queue architecture and operations",
    lessonMinutes: 15,
    sources: [
      {
        name: "RabbitMQ blog",
        url: "https://www.rabbitmq.com/blog/feed.xml",
        limit: 10,
      },
    ],
    scheduleTime: "10:00",
    timeZone: "Asia/Kolkata",
    ...overrides,
  };
}

function claimManual(workspace, newsletterId) {
  workspace.enqueueManualIssue(newsletterId);
  return workspace.claimNextIssue();
}
