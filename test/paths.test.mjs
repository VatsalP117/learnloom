import assert from "node:assert/strict";
import path from "node:path";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { resolveAppPaths } from "../src/paths.mjs";

test("resolveAppPaths uses the config directory independently of cwd", () => {
  const config = validateConfig(
    {
      interests: ["systems"],
      sources: [{ name: "Feed", url: "https://example.com/feed" }],
      storage: { dataDirectory: "private-data", outputDirectory: "dossiers" },
    },
    "/opt/learnloom/profiles/main.json",
  );
  const paths = resolveAppPaths(config, { env: {} });
  assert.equal(paths.dataDirectory, path.normalize("/opt/learnloom/profiles/private-data"));
  assert.equal(paths.outputDirectory, path.normalize("/opt/learnloom/profiles/dossiers"));
});

test("resolveAppPaths honors LEARNLOOM_HOME", () => {
  const config = validateConfig(
    {
      interests: ["systems"],
      sources: [{ name: "Feed", url: "https://example.com/feed" }],
    },
    "/opt/learnloom/config.json",
  );
  const paths = resolveAppPaths(config, { env: { LEARNLOOM_HOME: "/var/lib/learnloom" } });
  assert.equal(paths.historyPath, path.normalize("/var/lib/learnloom/data/history.json"));
});
