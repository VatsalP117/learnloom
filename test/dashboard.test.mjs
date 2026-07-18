import assert from "node:assert/strict";
import { mkdtemp, writeFile } from "node:fs/promises";
import { request as httpRequest } from "node:http";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { createDashboardServer } from "../src/dashboard.mjs";
import { SQLiteWorkspace } from "../src/workspace.mjs";

test("dashboard serves the React app and Newsletter API safely", async (context) => {
  const fixture = await dashboardFixture(context);
  fixture.workspace.createNewsletter({
    ...newsletterInput(),
    name: "<script>RabbitMQ</script>",
  });
  const response = await fetch(`${fixture.origin}/`);
  const html = await response.text();
  assert.equal(response.status, 200);
  assert.match(html, /Learnloom · Knowledge Dossiers/);
  assert.match(html, /<div id="root"><\/div>/);
  assert.doesNotMatch(html, /<script>RabbitMQ/);
  assert.match(response.headers.get("content-security-policy"), /script-src 'self'|default-src 'self'/);
  assert.match(response.headers.get("content-security-policy"), /frame-ancestors 'none'/);

  const apiResponse = await fetch(`${fixture.origin}/api/newsletters`);
  const snapshot = await apiResponse.json();
  assert.equal(apiResponse.status, 200);
  assert.equal(apiResponse.headers.get("content-type"), "application/json; charset=utf-8");
  assert.equal(snapshot.summary.newsletters, 1);
  assert.equal(snapshot.summary.active, 1);
  assert.equal(snapshot.newsletters[0].name, "<script>RabbitMQ</script>");
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
      aiExplorationEnabled: "on",
    }),
  });
  assert.equal(create.status, 303);
  const location = create.headers.get("location");
  assert.match(location, /^\/newsletters\/newsletter-/);

  const newsletterId = location.split("/").at(-1);
  assert.equal(
    fixture.workspace.getNewsletter(newsletterId).aiExplorationEnabled,
    true,
  );
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

test("dashboard rejects forged Host headers", async (context) => {
  const fixture = await dashboardFixture(context);
  const response = await rawRequest(
    `${fixture.origin}/newsletters/new`,
    "attacker.example",
  );
  assert.equal(response.statusCode, 421);
  assert.doesNotMatch(response.body, /name="_csrf"/);
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

test("dashboard saves per-Newsletter email settings", async (context) => {
  const fixture = await dashboardFixture(context);
  const newsletter = fixture.workspace.createNewsletter(newsletterInput());
  const response = await fetch(
    `${fixture.origin}/newsletters/${newsletter.id}/delivery`,
    {
      method: "POST",
      redirect: "manual",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        _csrf: fixture.csrfToken,
        emailEnabled: "on",
        emailRecipients: "READER@example.com\nteam@example.com",
      }),
    },
  );
  assert.equal(response.status, 303);
  assert.match(response.headers.get("location"), /\?delivery=saved$/);
  const saved = fixture.workspace.getNewsletter(newsletter.id);
  assert.equal(saved.emailEnabled, true);
  assert.deepEqual(saved.emailRecipients, [
    "reader@example.com",
    "team@example.com",
  ]);

  const detail = await fetch(
    `${fixture.origin}${response.headers.get("location")}`,
  );
  const html = await detail.text();
  assert.match(html, /Email delivery settings saved/);
  assert.match(html, /reader@example\.com/);
  assert.match(html, /No enabled Resend sender is configured/);
});

test("dashboard saves CSRF-protected AI Exploration preference", async (context) => {
  const fixture = await dashboardFixture(context);
  const newsletter = fixture.workspace.createNewsletter(newsletterInput());
  const rejected = await fetch(
    `${fixture.origin}/newsletters/${newsletter.id}/content`,
    {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ aiExplorationEnabled: "on" }),
    },
  );
  assert.equal(rejected.status, 403);

  const saved = await fetch(
    `${fixture.origin}/newsletters/${newsletter.id}/content`,
    {
      method: "POST",
      redirect: "manual",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        _csrf: fixture.csrfToken,
        aiExplorationEnabled: "on",
      }),
    },
  );
  assert.equal(saved.status, 303);
  assert.match(saved.headers.get("location"), /\?content=saved$/);
  assert.equal(
    fixture.workspace.getNewsletter(newsletter.id).aiExplorationEnabled,
    true,
  );

  const detail = await fetch(
    `${fixture.origin}${saved.headers.get("location")}`,
  );
  const html = await detail.text();
  assert.match(html, /Content settings saved/);
  assert.match(html, /Add AI Exploration to future Issues/);
  assert.match(html, /name="aiExplorationEnabled" checked/);
});

test("dashboard shows failed delivery and queues CSRF-protected retry", async (context) => {
  const fixture = await dashboardFixture(context);
  const newsletter = fixture.workspace.createNewsletter({
    ...newsletterInput(),
    emailEnabled: true,
    emailRecipients: ["reader@example.com"],
  });
  const issue = fixture.workspace.enqueueManualIssue(newsletter.id);
  fixture.workspace.claimNextIssue();
  fixture.workspace.completeIssue(issue.id, {
    title: "Delivery test",
    generationId: "generation-1",
    artifactPath: "/tmp/delivery.md",
    dossierPath: "/tmp/delivery.json",
  });
  fixture.workspace.claimNextDelivery();
  fixture.workspace.failDelivery(
    issue.id,
    new Error("provider rejected <sender>"),
  );

  const detail = await fetch(
    `${fixture.origin}/newsletters/${newsletter.id}`,
  );
  const html = await detail.text();
  assert.match(html, /Retry email/);
  assert.match(html, /provider rejected &lt;sender&gt;/);
  assert.doesNotMatch(html, /provider rejected <sender>/);

  const rejected = await fetch(
    `${fixture.origin}/issues/${issue.id}/retry-delivery`,
    {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams(),
    },
  );
  assert.equal(rejected.status, 403);

  const retried = await fetch(
    `${fixture.origin}/issues/${issue.id}/retry-delivery`,
    {
      method: "POST",
      redirect: "manual",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ _csrf: fixture.csrfToken }),
    },
  );
  assert.equal(retried.status, 303);
  assert.match(retried.headers.get("location"), /\?delivery=retried$/);
  assert.equal(fixture.workspace.getIssue(issue.id).delivery.status, "pending");
  assert.equal(fixture.workspace.getIssue(issue.id).status, "generated");
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

function rawRequest(url, host) {
  const target = new URL(url);
  return new Promise((resolve, reject) => {
    const request = httpRequest(
      {
        hostname: target.hostname,
        port: target.port,
        path: target.pathname,
        headers: { host },
      },
      (response) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () =>
          resolve({
            statusCode: response.statusCode,
            body: Buffer.concat(chunks).toString("utf8"),
          }),
        );
      },
    );
    request.on("error", reject);
    request.end();
  });
}
