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

test("buildDossier preserves every required section when source input exceeds the stage limit", async () => {
  const config = validateConfig({
    interests: ["systems"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
    limits: {
      maxItems: 20,
      maxItemCharacters: 10000,
      maxIntermediateCharacters: 2000,
    },
  });
  const inputs = new Map();
  const provider = {
    async complete({ stage, input }) {
      inputs.set(stage, input);
      return stage === "researcher"
        ? `RESEARCH-MARKER ${"r".repeat(3000)}`
        : stage === "skeptic"
          ? `CRITIQUE-MARKER ${"c".repeat(3000)}`
          : stage === "teacher"
            ? `LESSON-MARKER ${"l".repeat(3000)}`
            : "1. What is the key mechanism?";
    },
  };
  const items = [
    {
      source: "Large feed",
      title: "Large item",
      url: "https://example.com/large",
      summary: "s".repeat(10000),
      publishedAt: null,
    },
  ];
  await buildDossier({ config, items, history: [], provider });

  assert.match(inputs.get("skeptic"), /RESEARCH-MARKER/);
  assert.match(inputs.get("teacher"), /RESEARCH-MARKER/);
  assert.match(inputs.get("teacher"), /CRITIQUE-MARKER/);
  assert.match(inputs.get("examiner"), /LESSON-MARKER/);
  for (const input of [...inputs.values()].slice(1)) {
    assert.ok(input.length <= config.limits.maxIntermediateCharacters);
  }
});
