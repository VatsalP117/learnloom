import assert from "node:assert/strict";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { DEMO_ITEMS } from "../src/demo-data.mjs";
import { buildDossier } from "../src/pipeline.mjs";
import { DemoProvider } from "../src/provider.mjs";

test("buildDossier performs the quality pipeline and returns a validated Dossier v2", async () => {
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
  assert.deepEqual(stages, [
    "curator",
    "blueprint",
    "researcher",
    "skeptic",
    "teacher",
    "examiner",
    "editor",
  ]);
  assert.match(result.markdown, /Learning Dossier — 2026-07-18/);
  assert.match(result.markdown, /Source Index/);
  assert.match(result.markdown, /\[S1\]/);
  assert.equal(result.dossier.profileId, "default");
  assert.equal(result.dossier.version, 2);
  assert.equal(result.dossier.date, "2026-07-18");
  assert.equal(result.dossier.exploration, null);
  assert.equal(result.dossier.quality.checks.sourceGrounding, true);
  assert.equal(result.dossier.blueprint.learningObjective.length > 0, true);
  assert.equal(result.historyEntry.recallQuestions.length, 3);
  assert.equal(result.historyEntry.learningObjective.length > 0, true);
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
      if (stage === "curator") {
        return JSON.stringify({
          theme: "Large mechanism",
          rationale: "One source is available.",
          selectedSourceIds: ["S1"],
        });
      }
      if (stage === "blueprint") {
        return JSON.stringify(blueprintFixture());
      }
      if (stage === "researcher") {
        return `RESEARCH-MARKER [S1] ${"r".repeat(3000)}`;
      }
      if (stage === "skeptic") {
        return `CRITIQUE-MARKER [S1] ${"c".repeat(3000)}`;
      }
      if (stage === "teacher") return lessonFixture();
      if (stage === "examiner") return practiceFixture();
      if (stage === "editor") {
        return JSON.stringify({
          lesson: lessonFixture(),
          critique: "## Evidence limits\n\nBounded evidence [S1].",
          practice: practiceFixture(),
          exploration: null,
          qualityNotes: [],
        });
      }
      throw new Error(`Unexpected stage ${stage}`);
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

  assert.match(inputs.get("blueprint"), /Large item/);
  assert.match(inputs.get("skeptic"), /RESEARCH-MARKER/);
  assert.match(inputs.get("teacher"), /RESEARCH-MARKER/);
  assert.match(inputs.get("teacher"), /CRITIQUE-MARKER/);
  assert.match(inputs.get("examiner"), /Learning objective/);
  assert.match(inputs.get("editor"), /Application challenge/);
  for (const input of [...inputs.values()].slice(1)) {
    assert.ok(input.length <= config.limits.maxIntermediateCharacters);
  }
});

test("buildDossier excludes prior lessons when historyEntries is zero", async () => {
  const config = validateConfig({
    interests: ["systems"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
    limits: { historyEntries: 0 },
  });
  let researcherInput = "";
  const demo = new DemoProvider();
  const provider = {
    async complete(input) {
      if (input.stage === "researcher") researcherInput = input.input;
      return demo.complete(input);
    },
  };
  await buildDossier({
    config,
    items: DEMO_ITEMS,
    history: [{ date: "2026-07-17", lessonSummary: "SHOULD-NOT-APPEAR" }],
    provider,
  });
  assert.doesNotMatch(researcherInput, /SHOULD-NOT-APPEAR/);
});

test("buildDossier generates a separate uncited AI Exploration only when enabled", async () => {
  const config = validateConfig({
    interests: ["learning"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
    content: { aiExplorationEnabled: true },
  });
  const stages = [];
  const result = await buildDossier({
    config,
    items: DEMO_ITEMS,
    history: [],
    provider: new DemoProvider(),
    onStage: (stage) => stages.push(stage),
  });
  assert.ok(stages.includes("exploration"));
  assert.match(result.dossier.exploration, /Synthetic mental model/);
  assert.doesNotMatch(result.dossier.exploration, /\[S\d+\]/);
  assert.equal(result.dossier.quality.checks.explorationBoundary, true);
});

test("buildDossier repairs one malformed structured model response", async () => {
  const config = validateConfig({
    interests: ["learning"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
  });
  const demo = new DemoProvider();
  let curatorCalls = 0;
  const provider = {
    async complete(options) {
      if (options.stage === "curator") {
        curatorCalls += 1;
        if (curatorCalls === 1) return "not json";
        assert.match(options.input, /Contract repair/);
        assert.match(options.input, /invalid JSON/);
      }
      return demo.complete(options);
    },
  };
  const result = await buildDossier({
    config,
    items: DEMO_ITEMS,
    history: [],
    provider,
  });
  assert.equal(curatorCalls, 2);
  assert.equal(result.dossier.version, 2);
});

test("editor repair receives deterministic quality-gate failures", async () => {
  const config = validateConfig({
    interests: ["learning"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
  });
  const demo = new DemoProvider();
  let editorCalls = 0;
  const provider = {
    async complete(options) {
      if (options.stage === "editor") {
        editorCalls += 1;
        if (editorCalls === 1) {
          return JSON.stringify({
            lesson: "## Learning objective\n\nToo short [S1] [S2].",
            critique: "Evidence limits [S1].",
            practice: practiceFixture(),
            exploration: null,
            qualityNotes: [],
          });
        }
        assert.match(options.input, /Contract repair/);
        assert.match(options.input, /missing required sections/);
      }
      return demo.complete(options);
    },
  };
  const result = await buildDossier({
    config,
    items: DEMO_ITEMS,
    history: [],
    provider,
  });
  assert.equal(editorCalls, 2);
  assert.equal(result.dossier.quality.checks.requiredLessonSections, true);
});

test("curated source budgeting preserves every selected Source Item", async () => {
  const config = validateConfig({
    interests: ["systems"],
    sources: [{ name: "Demo", url: "https://example.com/feed" }],
    provider: { kind: "demo" },
    limits: { maxIntermediateCharacters: 12_000 },
  });
  const items = Array.from({ length: 5 }, (_, index) => ({
    source: "Large source",
    title: `Source ${index + 1}`,
    url: `https://example.com/${index + 1}`,
    summary: `${`SOURCE-${index + 1} `.repeat(900)}`,
    publishedAt: null,
  }));
  const demo = new DemoProvider();
  let researchInput = "";
  const provider = {
    async complete(options) {
      if (options.stage === "curator") {
        return JSON.stringify({
          theme: "Five-source mechanism",
          rationale: "Each source contributes evidence.",
          selectedSourceIds: ["S1", "S2", "S3", "S4", "S5"],
        });
      }
      if (options.stage === "researcher") researchInput = options.input;
      return demo.complete(options);
    },
  };
  await buildDossier({
    config,
    items,
    history: [],
    provider,
    enrichItemsFn: async (selected) =>
      selected.map((item) => ({
        ...item,
        contentSource: "article",
        canonicalUrl: item.url,
      })),
  });
  for (let index = 1; index <= 5; index += 1) {
    assert.match(researchInput, new RegExp(`\\[S${index}\\] Source ${index}`));
  }
  assert.ok(researchInput.length <= config.limits.maxIntermediateCharacters);
});

function blueprintFixture() {
  return {
    learningObjective: "Explain the mechanism.",
    prerequisites: ["Basic systems"],
    centralMechanism: "One mechanism.",
    workedExample: "One worked example.",
    misconception: "One misconception.",
    practicalExperiment: "One practical experiment.",
    continuityBridge: "A bridge to prior learning.",
  };
}

function lessonFixture() {
  return `## Learning objective
Explain the large mechanism [S1].
## Two-minute recall
Recall a prerequisite.
## Why this matters
It matters [S1].
## Mental model
Use a flow.
## How it works
The mechanism propagates.
## Worked example
Apply one input.
## Common misconception
Size is not quality.
## Practical experiment
Change one variable.
## Takeaway
Mechanisms matter.`;
}

function practiceFixture() {
  return `## Retrieval practice
1. What is the mechanism?
2. Why does it matter?
3. How would you test it?
## Application challenge
Apply it to a new system.
<details>
<summary>Answer key</summary>
1. It propagates a constraint.
2. It changes behavior.
3. Vary one input.
</details>`;
}
