import assert from "node:assert/strict";
import test from "node:test";
import { ResendDelivery } from "../src/delivery.mjs";
import { renderDossierEmail } from "../src/render.mjs";

const dossier = {
  version: 1,
  profileId: "default",
  date: "2026-07-18",
  title: "Safe <Learning>",
  generatedAt: "2026-07-18T00:00:00.000Z",
  model: "demo",
  lesson: "## Lesson\n\n<script>alert('x')</script>\n\n**Important** idea.",
  critique: "## Critique\n\nBe skeptical.",
  practice: "## Practice\n\n1. What did you learn?",
  sources: [
    {
      title: "Source <One>",
      source: "Example",
      url: "https://example.com/read?q=one&safe=yes",
      summary: "",
      publishedAt: null,
    },
    {
      title: "Unsafe URL",
      source: "Example",
      url: "javascript:alert(1)",
      summary: "",
      publishedAt: null,
    },
  ],
};

test("renderDossierEmail escapes model content and rejects unsafe source links", () => {
  const rendered = renderDossierEmail(dossier, "# Text");
  assert.doesNotMatch(rendered.html, /<script>/);
  assert.match(rendered.html, /&lt;script&gt;/);
  assert.match(rendered.html, /<strong>Important<\/strong>/);
  assert.doesNotMatch(rendered.html, /href="javascript:/);
  assert.match(rendered.html, /https:\/\/example\.com\/read\?q=one&amp;safe=yes/);
});

test("ResendDelivery sends deterministic idempotent email", async () => {
  const calls = [];
  const adapter = new ResendDelivery(
    {
      id: "morning-email",
      apiKeyEnv: "RESEND_API_KEY",
      from: "Learnloom <daily@example.com>",
      to: ["reader@example.com"],
      subjectPrefix: "Learnloom",
    },
    {
      env: { RESEND_API_KEY: "resend-secret" },
      fetchImpl: async (url, init) => {
        calls.push({ url, init });
        return jsonResponse({ id: "email-123" });
      },
    },
  );
  const receipt = await adapter.deliver({
    runId: "default-2026-07-18",
    dossier,
    markdown: "# Text",
  });
  assert.deepEqual(receipt, { externalId: "email-123" });
  assert.equal(calls[0].url, "https://api.resend.com/emails");
  assert.equal(
    calls[0].init.headers["idempotency-key"],
    "learnloom/default-2026-07-18/morning-email",
  );
  assert.equal(calls[0].init.headers.authorization, "Bearer resend-secret");
  const body = JSON.parse(calls[0].init.body);
  assert.deepEqual(body.to, ["reader@example.com"]);
  assert.doesNotMatch(body.html, /<script>/);
});

test("ResendDelivery reports safe provider errors", async () => {
  const adapter = new ResendDelivery(
    {
      id: "email",
      apiKeyEnv: "RESEND_API_KEY",
      from: "daily@example.com",
      to: ["reader@example.com"],
      subjectPrefix: "Learnloom",
    },
    {
      env: { RESEND_API_KEY: "do-not-print" },
      fetchImpl: async () =>
        jsonResponse(
          { message: "domain not verified; token do-not-print rejected" },
          403,
        ),
    },
  );
  await assert.rejects(
    adapter.deliver({ runId: "run", dossier, markdown: "# Text" }),
    (error) =>
      /HTTP 403/.test(error.message) &&
      /domain not verified/.test(error.message) &&
      !/do-not-print/.test(error.message),
  );
});

function jsonResponse(payload, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() {
      return payload;
    },
  };
}
