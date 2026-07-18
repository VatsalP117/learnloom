import assert from "node:assert/strict";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";

const valid = {
  interests: ["distributed systems"],
  sources: [{ name: "Example", url: "https://example.com/feed" }],
};

test("validateConfig applies safe defaults", () => {
  const config = validateConfig(valid);
  assert.equal(config.provider.kind, "commandcode");
  assert.equal(config.provider.model, "deepseek-v4-pro");
  assert.equal(config.sources[0].limit, 10);
});

test("validateConfig rejects non-web feed URLs", () => {
  assert.throws(
    () =>
      validateConfig({
        ...valid,
        sources: [{ name: "Local", url: "file:///etc/passwd" }],
      }),
    /HTTP or HTTPS/,
  );
});

test("validateConfig rejects empty interests", () => {
  assert.throws(() => validateConfig({ ...valid, interests: [] }), /at least one interest/);
});

test("validateConfig accepts an OpenAI-compatible provider and disabled Resend delivery", () => {
  const config = validateConfig({
    ...valid,
    profileId: "vatsal-learning",
    provider: {
      kind: "openai-compatible",
      baseUrl: "https://api.deepseek.com/",
      apiKeyEnv: "DEEPSEEK_API_KEY",
      model: "deepseek-v4-pro",
    },
    deliveries: [
      {
        kind: "resend",
        enabled: false,
        from: "Learnloom <daily@example.com>",
        to: "reader@example.com",
      },
    ],
  });
  assert.equal(config.provider.baseUrl, "https://api.deepseek.com");
  assert.equal(config.provider.apiKeyEnv, "DEEPSEEK_API_KEY");
  assert.deepEqual(config.deliveries[0].to, ["reader@example.com"]);
});

test("validateConfig rejects environment values in place of environment names", () => {
  assert.throws(
    () =>
      validateConfig({
        ...valid,
        provider: {
          kind: "openai-compatible",
          apiKeyEnv: "secret-value",
        },
      }),
    /environment variable name/,
  );
});

test("validateConfig rejects duplicate delivery identifiers", () => {
  assert.throws(
    () =>
      validateConfig({
        ...valid,
        deliveries: [
          {
            id: "email",
            kind: "resend",
            from: "daily@example.com",
            to: "one@example.com",
          },
          {
            id: "email",
            kind: "resend",
            from: "daily@example.com",
            to: "two@example.com",
          },
        ],
      }),
    /duplicate id/,
  );
});

test("validateConfig refuses API credentials over non-loopback HTTP", () => {
  assert.throws(
    () =>
      validateConfig({
        ...valid,
        provider: {
          kind: "openai-compatible",
          baseUrl: "http://api.example.com",
        },
      }),
    /must use HTTPS/,
  );
  const local = validateConfig({
    ...valid,
    provider: {
      kind: "openai-compatible",
      baseUrl: "http://127.0.0.1:11434/v1",
      allowInsecureHttp: true,
    },
  });
  assert.equal(local.provider.baseUrl, "http://127.0.0.1:11434/v1");
});
