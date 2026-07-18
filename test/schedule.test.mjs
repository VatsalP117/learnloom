import assert from "node:assert/strict";
import test from "node:test";
import { buildLaunchAgent } from "../src/schedule.mjs";

test("buildLaunchAgent creates a 9am plist and escapes paths", () => {
  const plist = buildLaunchAgent({
    nodePath: "/node",
    cliPath: "/project/a&b/bin/learn.mjs",
    configPath: "/project/config.json",
    workingDirectory: "/project/a&b",
    logDirectory: "/project/logs",
    hour: 9,
    minute: 0,
    environmentPath: "/bin:/usr/bin",
  });
  assert.match(plist, /<integer>9<\/integer>/);
  assert.match(plist, /a&amp;b/);
  assert.match(plist, /ai\.learning-engine\.morning/);
});

