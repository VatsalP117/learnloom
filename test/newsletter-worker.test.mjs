import assert from "node:assert/strict";
import { mkdtemp } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import {
  newsletterRuntimeConfig,
  processNextIssue,
  runWorkerCycle,
} from "../src/newsletter-worker.mjs";
import { DemoProvider } from "../src/provider.mjs";
import { SQLiteWorkspace } from "../src/workspace.mjs";

test("processNextIssue suppresses deliveries and records generated metadata", async () => {
  const fixture = await createFixture();
  const newsletter = fixture.workspace.createNewsletter(newsletterInput("RabbitMQ"));
  const queued = fixture.workspace.enqueueManualIssue(newsletter.id);
  let observed;
  const result = await processNextIssue({
    workspace: fixture.workspace,
    baseConfig: fixture.baseConfig,
    demo: true,
    now: fixture.now,
    home: fixture.home,
    async runDailyDossierFn(options) {
      observed = options;
      return {
        dossier: { title: "RabbitMQ flow control" },
        record: {
          generationId: "generation-1",
          artifactPath: "/tmp/rabbit.md",
          dossierPath: "/tmp/rabbit.json",
        },
      };
    },
  });
  assert.equal(result.id, queued.id);
  assert.equal(result.status, "generated");
  assert.deepEqual(observed.deliveries, []);
  assert.equal(observed.runId, queued.id);
  assert.equal(observed.config.profileId, newsletter.id);
  fixture.workspace.close();
});

test("processNextIssue safely records generation failure", async () => {
  const fixture = await createFixture();
  const newsletter = fixture.workspace.createNewsletter(newsletterInput("RabbitMQ"));
  fixture.workspace.enqueueManualIssue(newsletter.id);
  const result = await processNextIssue({
    workspace: fixture.workspace,
    baseConfig: fixture.baseConfig,
    now: fixture.now,
    async runDailyDossierFn() {
      throw new Error("provider\nunavailable");
    },
  });
  assert.equal(result.status, "failed");
  assert.equal(result.error, "provider unavailable");
  fixture.workspace.close();
});

test("processNextIssue records a fresh completion timestamp", async () => {
  const fixture = await createFixture();
  const newsletter = fixture.workspace.createNewsletter(newsletterInput("RabbitMQ"));
  fixture.workspace.enqueueManualIssue(newsletter.id);
  const timestamps = [
    new Date("2026-07-18T03:00:00.000Z"),
    new Date("2026-07-18T03:07:00.000Z"),
  ];
  const result = await processNextIssue({
    workspace: fixture.workspace,
    baseConfig: fixture.baseConfig,
    clock: () => timestamps.shift(),
    async runDailyDossierFn() {
      return {
        dossier: { title: "Seven-minute generation" },
        record: {
          generationId: "generation-1",
          artifactPath: "/tmp/issue.md",
          dossierPath: "/tmp/issue.json",
        },
      };
    },
  });
  assert.equal(result.startedAt, "2026-07-18T03:00:00.000Z");
  assert.equal(result.completedAt, "2026-07-18T03:07:00.000Z");
  fixture.workspace.close();
});

test("runWorkerCycle generates isolated Dossiers for two Newsletters", async () => {
  const fixture = await createFixture();
  const rabbit = fixture.workspace.createNewsletter(newsletterInput("RabbitMQ"));
  const postgres = fixture.workspace.createNewsletter(
    newsletterInput("PostgreSQL", { topic: "PostgreSQL query planning" }),
  );
  fixture.workspace.enqueueManualIssue(rabbit.id);
  fixture.workspace.enqueueManualIssue(postgres.id);

  const result = await runWorkerCycle({
    workspace: fixture.workspace,
    baseConfig: fixture.baseConfig,
    demo: true,
    now: fixture.now,
    home: fixture.home,
    provider: new DemoProvider(),
  });
  assert.equal(result.processed.length, 2);
  assert.ok(result.processed.every((issue) => issue.status === "generated"));
  assert.notEqual(
    result.processed[0].artifactPath,
    result.processed[1].artifactPath,
  );
  const rabbitIssues = fixture.workspace.listIssues(rabbit.id);
  const postgresIssues = fixture.workspace.listIssues(postgres.id);
  assert.match(rabbitIssues[0].artifactPath, new RegExp(rabbit.id));
  assert.match(postgresIssues[0].artifactPath, new RegExp(postgres.id));
  assert.equal(rabbitIssues.length, 1);
  assert.equal(postgresIssues.length, 1);
  fixture.workspace.close();
});

test("newsletterRuntimeConfig maps Newsletter identity without deliveries", async () => {
  const fixture = await createFixture();
  const newsletter = fixture.workspace.createNewsletter(newsletterInput("RabbitMQ"));
  const config = newsletterRuntimeConfig(fixture.baseConfig, newsletter);
  assert.equal(config.profileId, newsletter.id);
  assert.deepEqual(config.interests, ["RabbitMQ messaging queues"]);
  assert.deepEqual(config.deliveries, []);
  fixture.workspace.close();
});

async function createFixture() {
  const home = await mkdtemp(path.join(os.tmpdir(), "learnloom-worker-"));
  const now = new Date("2026-07-18T03:00:00.000Z");
  const baseConfig = validateConfig(
    {
      profileId: "default",
      timeZone: "Asia/Kolkata",
      interests: ["default"],
      sources: [{ name: "Demo", url: "https://example.com/feed" }],
      provider: { kind: "demo" },
      deliveries: [
        {
          id: "must-not-send",
          kind: "resend",
          enabled: true,
          from: "daily@example.com",
          to: "reader@example.com",
        },
      ],
      storage: { dataDirectory: "data", outputDirectory: "output" },
    },
    path.join(home, "config.json"),
  );
  return {
    home,
    now,
    baseConfig,
    workspace: new SQLiteWorkspace(":memory:", { now: () => now }),
  };
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
      },
    ],
    scheduleTime: "10:00",
    timeZone: "Asia/Kolkata",
    ...overrides,
  };
}
