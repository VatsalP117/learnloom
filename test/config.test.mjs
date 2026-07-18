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

