import assert from "node:assert/strict";
import test from "node:test";
import {
  OpenAICompatibleProvider,
  checkProvider,
  createProvider,
} from "../src/provider.mjs";
import { validateConfig } from "../src/config.mjs";

const providerConfig = {
  kind: "openai-compatible",
  baseUrl: "https://models.example/v1",
  apiKeyEnv: "MODEL_KEY",
  model: "learn-model",
  maxTokens: 2048,
  timeoutSeconds: 30,
  retries: 2,
};

test("OpenAICompatibleProvider sends a Chat Completions request without shelling out", async () => {
  const calls = [];
  const provider = new OpenAICompatibleProvider(providerConfig, {
    env: { MODEL_KEY: "private-key" },
    fetchImpl: async (url, init) => {
      calls.push({ url, init });
      return jsonResponse({
        choices: [{ message: { content: "## Lesson\n\nUseful." } }],
      });
    },
  });
  const output = await provider.complete({
    stage: "teacher",
    instruction: "Teach clearly.",
    input: "Reference text",
  });
  assert.equal(output, "## Lesson\n\nUseful.");
  assert.equal(calls[0].url, "https://models.example/v1/chat/completions");
  assert.equal(calls[0].init.headers.authorization, "Bearer private-key");
  const body = JSON.parse(calls[0].init.body);
  assert.equal(body.model, "learn-model");
  assert.match(body.messages[1].content, /STAGE: teacher/);
});

test("OpenAICompatibleProvider requires its configured environment variable", async () => {
  const provider = new OpenAICompatibleProvider(providerConfig, { env: {} });
  await assert.rejects(
    provider.complete({ stage: "teacher", instruction: "Teach", input: "Input" }),
    /MODEL_KEY/,
  );
});

test("OpenAICompatibleProvider retries 429 without exposing credentials", async () => {
  let attempts = 0;
  const provider = new OpenAICompatibleProvider(providerConfig, {
    env: { MODEL_KEY: "never-print-this" },
    sleep: async () => {},
    fetchImpl: async () => {
      attempts += 1;
      if (attempts < 3) {
        return jsonResponse({ error: { message: "slow down" } }, 429);
      }
      return jsonResponse({ choices: [{ message: { content: "Done" } }] });
    },
  });
  assert.equal(
    await provider.complete({ stage: "teacher", instruction: "Teach", input: "Input" }),
    "Done",
  );
  assert.equal(attempts, 3);
});

test("checkProvider discovers a configured HTTP model", async () => {
  const config = validateConfig({
    interests: ["systems"],
    sources: [{ name: "Feed", url: "https://example.com/feed" }],
    provider: {
      kind: "openai-compatible",
      baseUrl: "https://models.example/v1",
      apiKeyEnv: "MODEL_KEY",
      model: "learn-model",
    },
  });
  const checks = await checkProvider(config, {
    env: { MODEL_KEY: "private-key" },
    fetchImpl: async () => jsonResponse({ data: [{ id: "learn-model" }] }),
  });
  assert.equal(checks.every((check) => check.ok), true);
});

test("createProvider selects the HTTP adapter", () => {
  const config = { provider: providerConfig };
  assert.ok(createProvider(config, { env: { MODEL_KEY: "key" } }) instanceof OpenAICompatibleProvider);
});

function jsonResponse(payload, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() {
      return payload;
    },
    async text() {
      return JSON.stringify(payload);
    },
  };
}
