import assert from "node:assert/strict";
import { mkdtemp, writeFile } from "node:fs/promises";
import { request as httpRequest } from "node:http";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { createDashboardServer } from "../src/dashboard.mjs";
import { resolveDeploymentConfig } from "../src/host-routing.mjs";
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
  assert.match(html, /Learnloom · Knowledge Dossiers/);
  const detailApi = await fetch(
    `${fixture.origin}/api/newsletters/${newsletterId}`,
  );
  const detailSnapshot = await detailApi.json();
  assert.equal(detailSnapshot.newsletter.name, "RabbitMQ Deep Dive");
  assert.equal(detailSnapshot.issues.length, 1);
  assert.equal(detailSnapshot.issues[0].status, "queued");
  assert.equal(detailSnapshot.csrfToken, fixture.csrfToken);
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
    `${fixture.origin}/api/newsletters/${newsletter.id}`,
  );
  const snapshot = await detail.json();
  assert.equal(snapshot.newsletter.emailEnabled, true);
  assert.deepEqual(snapshot.newsletter.emailRecipients, [
    "reader@example.com",
    "team@example.com",
  ]);
  assert.equal(snapshot.resendConfigured, false);
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
    `${fixture.origin}/api/newsletters/${newsletter.id}`,
  );
  const snapshot = await detail.json();
  assert.equal(snapshot.newsletter.aiExplorationEnabled, true);
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
    `${fixture.origin}/api/newsletters/${newsletter.id}`,
  );
  const snapshot = await detail.json();
  assert.equal(snapshot.issues[0].delivery.status, "failed");
  assert.equal(
    snapshot.issues[0].delivery.error,
    "provider rejected <sender>",
  );

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

test("hosted routing fails closed before authentication is enabled", async (context) => {
  const deployment = resolveDeploymentConfig({
    env: {
      LEARNLOOM_DEPLOYMENT_MODE: "hosted",
      LEARNLOOM_ROOT_DOMAIN: "learnloom.blog",
      LEARNLOOM_APP_ORIGIN: "https://app.learnloom.blog",
    },
  });
  const fixture = await dashboardFixture(context, { deployment });
  fixture.workspace.createNewsletter({
    ...newsletterInput(),
    name: "Tenant secret",
  });

  const health = await rawRequest(
    `${fixture.origin}/healthz`,
    "app.learnloom.blog",
  );
  assert.equal(health.statusCode, 200);

  const apex = await rawRequest(`${fixture.origin}/`, "learnloom.blog");
  assert.equal(apex.statusCode, 200);
  assert.match(apex.body, /A learning home that grows with you/);

  const app = await rawRequest(
    `${fixture.origin}/api/newsletters`,
    "app.learnloom.blog",
  );
  assert.equal(app.statusCode, 503);
  assert.doesNotMatch(app.body, /Tenant secret/);

  const site = await rawRequest(
    `${fixture.origin}/`,
    "vatsal.learnloom.blog",
  );
  assert.equal(site.statusCode, 404);
  assert.doesNotMatch(site.body, /Tenant secret/);

  const reserved = await rawRequest(
    `${fixture.origin}/`,
    "clerk.learnloom.blog",
  );
  assert.equal(reserved.statusCode, 421);

  const www = await rawRequest(
    `${fixture.origin}/welcome?from=www`,
    "www.learnloom.blog",
  );
  assert.equal(www.statusCode, 308);
  assert.equal(
    www.headers.location,
    "https://learnloom.blog/welcome?from=www",
  );
});

test("hosted authentication provisions accounts and isolates tenant APIs", async (context) => {
  const deployment = resolveDeploymentConfig({
    env: {
      LEARNLOOM_DEPLOYMENT_MODE: "hosted",
      LEARNLOOM_ROOT_DOMAIN: "learnloom.blog",
      LEARNLOOM_APP_ORIGIN: "https://app.learnloom.blog",
    },
  });
  const authenticator = {
    async authenticate(request) {
      const userId = request.headers["x-test-clerk-user"];
      return userId
        ? {
            status: "authenticated",
            clerkUserId: userId,
            sessionId: `session-${userId}`,
            headers: new Headers(),
          }
        : { status: "unauthenticated", headers: new Headers() };
    },
  };
  const fixture = await dashboardFixture(context, {
    deployment,
    authenticator,
  });

  const shell = await rawRequest(
    `${fixture.origin}/`,
    "app.learnloom.blog",
  );
  assert.equal(shell.statusCode, 200);
  assert.match(
    shell.headers["content-security-policy"],
    /https:\/\/clerk\.learnloom\.blog/,
  );

  const unauthenticated = await rawRequest(
    `${fixture.origin}/api/me`,
    "app.learnloom.blog",
  );
  assert.equal(unauthenticated.statusCode, 401);

  const aliceMe = await rawRequest(
    `${fixture.origin}/api/me`,
    "app.learnloom.blog",
    { headers: { "x-test-clerk-user": "user_alice" } },
  );
  assert.equal(aliceMe.statusCode, 200);
  const aliceProfile = JSON.parse(aliceMe.body);
  assert.equal(aliceProfile.site, null);
  assert.notEqual(aliceProfile.csrfToken, fixture.csrfToken);

  const availability = await rawRequest(
    `${fixture.origin}/api/usernames/alice`,
    "app.learnloom.blog",
    { headers: { "x-test-clerk-user": "user_alice" } },
  );
  assert.equal(JSON.parse(availability.body).available, true);

  const forgedClaim = await rawRequest(
    `${fixture.origin}/api/me/site/claim`,
    "app.learnloom.blog",
    {
      method: "POST",
      headers: {
        "x-test-clerk-user": "user_alice",
        "content-type": "application/x-www-form-urlencoded",
        origin: "https://attacker.example",
      },
      body: new URLSearchParams({
        _csrf: aliceProfile.csrfToken,
        username: "stolen",
        displayName: "Stolen",
      }).toString(),
    },
  );
  assert.equal(forgedClaim.statusCode, 403);

  const claim = await rawRequest(
    `${fixture.origin}/api/me/site/claim`,
    "app.learnloom.blog",
    {
      method: "POST",
      headers: {
        "x-test-clerk-user": "user_alice",
        "content-type": "application/x-www-form-urlencoded",
        origin: "https://app.learnloom.blog",
      },
      body: new URLSearchParams({
        _csrf: aliceProfile.csrfToken,
        username: "alice",
        displayName: "Alice",
      }).toString(),
    },
  );
  assert.equal(claim.statusCode, 201);
  assert.equal(JSON.parse(claim.body).site.username, "alice");
  const unavailable = await rawRequest(
    `${fixture.origin}/api/usernames/alice`,
    "app.learnloom.blog",
    { headers: { "x-test-clerk-user": "user_bob" } },
  );
  assert.equal(JSON.parse(unavailable.body).available, false);
  const bobMe = await rawRequest(
    `${fixture.origin}/api/me`,
    "app.learnloom.blog",
    { headers: { "x-test-clerk-user": "user_bob" } },
  );
  assert.notEqual(
    JSON.parse(bobMe.body).csrfToken,
    aliceProfile.csrfToken,
  );

  const alice = fixture.workspace.getAccountByClerkUserId("user_alice");
  const bob = fixture.workspace.ensureAccount("user_bob");
  const aliceNewsletter = fixture.workspace
    .forAccount(alice.id)
    .createNewsletter(newsletterInput());
  const bobNewsletter = fixture.workspace
    .forAccount(bob.id)
    .createNewsletter({ ...newsletterInput(), name: "Bob secret" });

  const aliceList = await rawRequest(
    `${fixture.origin}/api/newsletters`,
    "app.learnloom.blog",
    { headers: { "x-test-clerk-user": "user_alice" } },
  );
  const aliceSnapshot = JSON.parse(aliceList.body);
  assert.equal(aliceList.statusCode, 200);
  assert.deepEqual(
    aliceSnapshot.newsletters.map((item) => item.id),
    [aliceNewsletter.id],
  );
  assert.doesNotMatch(aliceList.body, /Bob secret/);

  const crossTenant = await rawRequest(
    `${fixture.origin}/api/newsletters/${bobNewsletter.id}`,
    "app.learnloom.blog",
    { headers: { "x-test-clerk-user": "user_alice" } },
  );
  assert.equal(crossTenant.statusCode, 404);
  const crossTenantMutation = await rawRequest(
    `${fixture.origin}/api/newsletters/${bobNewsletter.id}/site`,
    "app.learnloom.blog",
    {
      method: "POST",
      headers: {
        "x-test-clerk-user": "user_alice",
        "content-type": "application/x-www-form-urlencoded",
        origin: "https://app.learnloom.blog",
      },
      body: new URLSearchParams({
        _csrf: aliceProfile.csrfToken,
        visible: "false",
      }).toString(),
    },
  );
  assert.equal(crossTenantMutation.statusCode, 404);
  assert.equal(
    fixture.workspace.getNewsletter(bobNewsletter.id).siteVisible,
    true,
  );

  fixture.workspace.forAccount(alice.id).enqueueManualIssue(aliceNewsletter.id);
  const claimedIssue = fixture.workspace.claimNextIssue();
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-public-"));
  const dossierPath = path.join(directory, "dossier.json");
  const artifactPath = path.join(directory, "dossier.md");
  await writeFile(
    dossierPath,
    JSON.stringify({
      version: 1,
      profileId: aliceNewsletter.id,
      date: "2026-07-18",
      title: "Public <Queues>",
      generatedAt: "2026-07-18T03:00:00.000Z",
      model: "demo",
      lesson: "## Lesson\n\n<script>alert('no')</script>",
      critique: "## Critique\n\nCheck it.",
      practice: "## Practice\n\nExplain it.",
      sources: [],
    }),
  );
  await writeFile(artifactPath, "# Public queues");
  const generated = fixture.workspace.completeIssue(claimedIssue.id, {
    title: "Public <Queues>",
    generationId: "public-generation",
    artifactPath,
    dossierPath,
  });

  const privateSite = await rawRequest(
    `${fixture.origin}/`,
    "alice.learnloom.blog",
  );
  assert.equal(privateSite.statusCode, 404);
  const publish = await rawRequest(
    `${fixture.origin}/api/me/site/settings`,
    "app.learnloom.blog",
    {
      method: "POST",
      headers: {
        "x-test-clerk-user": "user_alice",
        "content-type": "application/x-www-form-urlencoded",
        origin: "https://app.learnloom.blog",
      },
      body: new URLSearchParams({
        _csrf: aliceProfile.csrfToken,
        visibility: "public",
      }).toString(),
    },
  );
  assert.equal(publish.statusCode, 200);
  assert.equal(JSON.parse(publish.body).site.url, "https://alice.learnloom.blog");

  const publicHome = await rawRequest(
    `${fixture.origin}/`,
    "alice.learnloom.blog",
  );
  assert.equal(publicHome.statusCode, 200);
  assert.match(publicHome.body, /Public &lt;Queues&gt;/);
  assert.doesNotMatch(publicHome.body, /Bob secret/);
  assert.match(publicHome.headers["cache-control"], /public/);
  assert.match(publicHome.headers["content-security-policy"], /form-action 'none'/);

  const topic = await rawRequest(
    `${fixture.origin}/topics/rabbitmq-deep-dive`,
    "alice.learnloom.blog",
  );
  assert.equal(topic.statusCode, 200);
  assert.match(topic.body, /RabbitMQ messaging queues/);

  const canonicalPath = `/d/${generated.publicId}/${generated.publicSlug}`;
  const canonicalRedirect = await rawRequest(
    `${fixture.origin}/d/${generated.publicId}`,
    "alice.learnloom.blog",
  );
  assert.equal(canonicalRedirect.statusCode, 308);
  assert.equal(
    canonicalRedirect.headers.location,
    `https://alice.learnloom.blog${canonicalPath}`,
  );
  const publicDossier = await rawRequest(
    `${fixture.origin}${canonicalPath}`,
    "alice.learnloom.blog",
  );
  assert.equal(publicDossier.statusCode, 200);
  assert.match(publicDossier.body, /rel="canonical"/);
  assert.match(publicDossier.body, /&lt;script&gt;/);
  assert.doesNotMatch(publicDossier.body, /<script>alert/);
});

async function dashboardFixture(context, options = {}) {
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
    deployment: options.deployment,
    authenticator: options.authenticator,
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

function rawRequest(url, host, options = {}) {
  const target = new URL(url);
  return new Promise((resolve, reject) => {
    const request = httpRequest(
      {
        hostname: target.hostname,
        port: target.port,
        path: `${target.pathname}${target.search}`,
        method: options.method ?? "GET",
        headers: { host, ...options.headers },
      },
      (response) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () =>
          resolve({
            statusCode: response.statusCode,
            headers: response.headers,
            body: Buffer.concat(chunks).toString("utf8"),
          }),
        );
      },
    );
    request.on("error", reject);
    if (options.body) request.write(options.body);
    request.end();
  });
}
