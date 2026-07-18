import assert from "node:assert/strict";
import { mkdtemp } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { SQLiteWorkspace, nextDailyOccurrence } from "../src/workspace.mjs";

test("SQLiteWorkspace initializes idempotently with safety pragmas", () => {
  const workspace = new SQLiteWorkspace(":memory:");
  workspace.initialize();
  const diagnostics = workspace.diagnostics();
  assert.equal(diagnostics.userVersion, 1);
  assert.equal(diagnostics.foreignKeys, true);
  assert.equal(diagnostics.busyTimeout, 5_000);
  assert.match(diagnostics.journalMode, /^(memory|wal)$/);
  workspace.close();
});

test("SQLiteWorkspace rejects a newer schema version", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-schema-"));
  const databasePath = path.join(directory, "workspace.sqlite");
  const workspace = new SQLiteWorkspace(databasePath);
  workspace.database.exec("PRAGMA user_version = 2");
  workspace.close();
  assert.throws(
    () => new SQLiteWorkspace(databasePath),
    /newer than this Learnloom release/,
  );
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
