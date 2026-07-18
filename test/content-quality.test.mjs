import assert from "node:assert/strict";
import test from "node:test";
import {
  evaluateDossierContent,
  parseStructuredStage,
  validateBlueprint,
  validateCuration,
  validateEditorial,
} from "../src/content-quality.mjs";

test("parseStructuredStage accepts plain or fenced JSON and rejects prose", () => {
  assert.deepEqual(parseStructuredStage('{"value":1}', "curator"), {
    value: 1,
  });
  assert.deepEqual(
    parseStructuredStage("```json\n{\"value\":2}\n```", "curator"),
    { value: 2 },
  );
  assert.throws(
    () => parseStructuredStage("Here is the result", "curator"),
    /invalid JSON/,
  );
});

test("validateCuration enforces known unique source identifiers", () => {
  assert.deepEqual(
    validateCuration(
      {
        theme: "Backpressure",
        rationale: "The sources explain one mechanism.",
        selectedSourceIds: ["S1", "S2", "S3", "S3"],
      },
      4,
    ).selectedSourceIds,
    ["S1", "S2", "S3"],
  );
  assert.throws(
    () =>
      validateCuration(
        {
          theme: "Bad",
          rationale: "Unknown source",
          selectedSourceIds: ["S1", "S2", "S9"],
        },
        4,
      ),
    /unknown Source Item S9/,
  );
});

test("validateBlueprint and editorial contracts require bounded separate fields", () => {
  const blueprint = validateBlueprint(validBlueprint());
  assert.equal(blueprint.learningObjective, "Explain flow control");
  assert.throws(
    () => validateBlueprint({ ...validBlueprint(), prerequisites: [] }),
    /non-empty array/,
  );
  assert.throws(
    () =>
      validateEditorial(
        {
          lesson: "Lesson",
          critique: "Critique",
          practice: "Practice",
          exploration: "Synthetic",
        },
        { explorationEnabled: false },
      ),
    /when it was disabled/,
  );
});

test("evaluateDossierContent validates structure, citations, practice, and boundary", () => {
  const result = evaluateDossierContent({
    lesson: validLesson(),
    critique: "## Evidence limits\n\nThe claim is bounded [S2].",
    practice: validPractice(),
    exploration: "A hypothetical queue could behave like a crowded theatre.",
    sources: [
      { sourceId: "S1", contentSource: "article" },
      { sourceId: "S2", contentSource: "feed-summary" },
    ],
    blueprint: validBlueprint(),
    historyCount: 2,
  });
  assert.equal(result.checks.explorationBoundary, true);
  assert.equal(result.metrics.enrichedSources, 1);
  assert.equal(result.metrics.retrievalQuestions, 3);
  assert.ok(result.score >= 80);

  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: validLesson().replace("[S2]", "[S9]"),
        critique: "Limits [S2].",
        practice: validPractice(),
        exploration: null,
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /unknown Source Items: S9/,
  );
  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: validLesson(),
        critique: "Limits [S2].",
        practice: validPractice().replace(
          "Demand exceeds service.",
          "Demand exceeds service [S9].",
        ),
        exploration: null,
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /unknown Source Items: S9/,
  );
  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: validLesson(),
        critique: "Limits [S2].",
        practice: validPractice(),
        exploration: "A cited synthetic idea [S1].",
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /must not use source citation markers/,
  );
});

function validBlueprint() {
  return {
    learningObjective: "Explain flow control",
    prerequisites: ["Queues"],
    centralMechanism: "Credit limits propagate backpressure.",
    workedExample: "A slow consumer fills a queue.",
    misconception: "More buffering always helps.",
    practicalExperiment: "Vary the consumer rate.",
    continuityBridge: "Builds on the previous queue lesson.",
  };
}

function validLesson() {
  return `## Learning objective
Explain flow control [S1].
## Two-minute recall
Recall queue acknowledgement.
## Why this matters
Overload can spread [S2].
## Mental model
Think in rates.
## How it works
Credits constrain producers.
## Worked example
A consumer slows.
## Common misconception
Buffers are not capacity.
## Practical experiment
Change the service rate.
## Takeaway
Backpressure is feedback.`;
}

function validPractice() {
  return `## Retrieval practice
1. What causes backpressure?
2. How do credits limit a producer?
3. Why can buffering hide overload?

## Application challenge
Diagnose a slow consumer.

<details>
<summary>Answer key</summary>

1. Demand exceeds service.
2. A producer waits for credit.
3. It delays the visible failure.
</details>`;
}
