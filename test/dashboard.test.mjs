import assert from "node:assert/strict";
import { mkdtemp, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { createDashboardServer } from "../src/dashboard.mjs";
import { SQLiteWorkspace } from "../src/workspace.mjs";

test("dashboard renders Newsletter overview with escaped content", async (context) => {
  const fixture = await dashboardFixture(context);
  fixture.workspace.createNewsletter({
    ...newsletterInput(),
    name: "<script>RabbitMQ</script>",
  });
  const response = await fetch(`${fixture.origin}/`);
  const html = await response.text();
  assert.equal(response.status, 200);
  assert.match(html, /Your Newsletters/);
  assert.match(html, /&lt;script&gt;RabbitMQ&lt;\/script&gt;/);
  assert.doesNotMatch(html, /<script>RabbitMQ/);
  assert.match(response.headers.get("content-security-policy"), /frame-ancestors 'none'/);
});

test("dashboard creates a Newsletter and queues Run Now through CSRF forms", async (context) => {
  const fixture = await dashboardFixture(context);
  const create = await fetch(`${fixture.origin}/newsletters`, {
    method: "POST",
    redirect: "manual",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      _csrf: fixture.csrfToken,
      name: "RabbitMQ Deep Dive",
      topic: "RabbitMQ backpressure and delivery guarantees",
      scheduleTime: "10:00",
      timeZone: "Asia/Kolkata",
      sourceUrls: "https://www.rabbitmq.com/blog/feed.xml",
    }),
  });
  assert.equal(create.status, 303);
  const location = create.headers.get("location");
  assert.match(location, /^\/newsletters\/newsletter-/);

  const newsletterId = location.split("/").at(-1);
  const run = await fetch(`${fixture.origin}/newsletters/${newsletterId}/run`, {
    method: "POST",
    redirect: "manual",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ _csrf: fixture.csrfToken }),
  });
  assert.equal(run.status, 303);
  assert.match(run.headers.get("location"), /\?queued=issue-/);
  assert.equal(fixture.workspace.listIssues(newsletterId).length, 1);

  const detail = await fetch(`${fixture.origin}${run.headers.get("location")}`);
  const html = await detail.text();
  assert.match(html, /Issue queued/);
  assert.match(html, /RabbitMQ Deep Dive/);
});

test("dashboard rejects missing CSRF and unsupported methods", async (context) => {
  const fixture = await dashboardFixture(context);
  const rejected = await fetch(`${fixture.origin}/newsletters`, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      name: "No token",
      topic: "Should fail",
      scheduleTime: "10:00",
      timeZone: "UTC",
    }),
  });
  assert.equal(rejected.status, 403);

  const unsupported = await fetch(`${fixture.origin}/`, { method: "DELETE" });
  assert.equal(unsupported.status, 405);
  assert.equal(unsupported.headers.get("allow"), "GET");
});

test("dashboard returns useful validation errors", async (context) => {
  const fixture = await dashboardFixture(context);
  const response = await fetch(`${fixture.origin}/newsletters`, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      _csrf: fixture.csrfToken,
      name: "Broken schedule",
      topic: "RabbitMQ",
      scheduleTime: "29:00",
      timeZone: "UTC",
      sourceUrls: "https://example.com/feed",
    }),
  });
  assert.equal(response.status, 400);
  assert.match(await response.text(), /24-hour HH:MM/);
});

test("dashboard shows a safe generated Dossier preview by Issue ID", async (context) => {
  const fixture = await dashboardFixture(context);
  const newsletter = fixture.workspace.createNewsletter(newsletterInput());
  fixture.workspace.enqueueManualIssue(newsletter.id);
  const issue = fixture.workspace.claimNextIssue();
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-preview-"));
  const dossierPath = path.join(directory, "dossier.json");
  const artifactPath = path.join(directory, "dossier.md");
  const dossier = {
    version: 1,
    profileId: newsletter.id,
    date: "2026-07-18",
    title: "Safe <Queues>",
    generatedAt: "2026-07-18T03:00:00.000Z",
    model: "demo",
    lesson: "## Lesson\n\n<script>alert('x')</script>",
    critique: "## Critique\n\nCheck assumptions.",
    practice: "## Practice\n\n1. Explain backpressure.",
    sources: [
      {
        title: "RabbitMQ",
        source: "RabbitMQ",
        url: "https://www.rabbitmq.com/",
        summary: "",
        publishedAt: null,
      },
    ],
  };
  await writeFile(dossierPath, JSON.stringify(dossier));
  await writeFile(artifactPath, "# Safe queues");
  fixture.workspace.completeIssue(issue.id, {
    title: dossier.title,
    generationId: "generation-1",
    artifactPath,
    dossierPath,
  });

  const response = await fetch(`${fixture.origin}/issues/${issue.id}`);
  const html = await response.text();
  assert.equal(response.status, 200);
  assert.match(html, /Safe &lt;Queues&gt;/);
  assert.match(html, /&lt;script&gt;/);
  assert.doesNotMatch(html, /<script>alert/);
  assert.doesNotMatch(html, /<!doctype html>[\s\S]*<!doctype html>/i);
});

test("dashboard returns 404 for unknown identifiers", async (context) => {
  const fixture = await dashboardFixture(context);
  const response = await fetch(`${fixture.origin}/issues/not-here`);
  assert.equal(response.status, 404);
});

async function dashboardFixture(context) {
  const now = new Date("2026-07-18T03:00:00.000Z");
  const workspace = new SQLiteWorkspace(":memory:", { now: () => now });
  const baseConfig = validateConfig({
    timeZone: "Asia/Kolkata",
    interests: ["default"],
    learner: {
      level: "technically experienced",
      goal: "build durable understanding",
      lessonMinutes: 15,
    },
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
    deliveries: [],
  });
  const { server, csrfToken } = createDashboardServer({
    workspace,
    baseConfig,
    csrfToken: "fixed-test-token",
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  context.after(
    () =>
      new Promise((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      ),
  );
  context.after(() => workspace.close());
  const address = server.address();
  return {
    workspace,
    csrfToken,
    origin: `http://127.0.0.1:${address.port}`,
  };
}

function newsletterInput() {
  return {
    name: "RabbitMQ Deep Dive",
    topic: "RabbitMQ messaging queues",
    learnerLevel: "technically experienced",
    learnerGoal: "understand queue architecture",
    lessonMinutes: 15,
    sources: [
      {
        name: "RabbitMQ",
        url: "https://www.rabbitmq.com/blog/feed.xml",
      },
    ],
    scheduleTime: "10:00",
    timeZone: "Asia/Kolkata",
  };
}
