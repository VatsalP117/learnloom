import assert from "node:assert/strict";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { DEMO_ITEMS } from "../src/demo-data.mjs";
import { buildDossier } from "../src/pipeline.mjs";
import { DemoProvider } from "../src/provider.mjs";

test("buildDossier performs four stages and returns a sourced document", async () => {
  const config = validateConfig({
    timeZone: "Asia/Kolkata",
    interests: ["learning"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
  });
  const stages = [];
  const result = await buildDossier({
    config,
    items: DEMO_ITEMS,
    history: [],
    provider: new DemoProvider(),
    now: new Date("2026-07-18T06:00:00.000Z"),
    onStage: (stage) => stages.push(stage),
  });
  assert.deepEqual(stages, ["researcher", "skeptic", "teacher", "examiner"]);
  assert.match(result.markdown, /Learning Dossier — 2026-07-18/);
  assert.match(result.markdown, /Source Index/);
  assert.match(result.markdown, /\[S1\]/);
  assert.equal(result.historyEntry.recallQuestions.length, 3);
});
