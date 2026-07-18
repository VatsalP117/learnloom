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
          "Backpressure begins when",
          "Backpressure begins [S9] when",
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
  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: REQUIRED_TINY_LESSON,
        critique: "A nominal critique.",
        practice: `## Retrieval practice
1. What is repeated?
2. What is repeated?
3. What is repeated?
## Application challenge
x
<details><summary>Answer key</summary></details>
[S1] [S2]`,
        exploration: null,
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /substantive content/,
  );
  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: validLesson(),
        critique: "A bounded critique [S2].",
        practice: `## Retrieval practice
Use recall before opening the answer key.

## Application challenge
Diagnose a slow consumer in a three-service pipeline and identify where the feedback signal should be applied.

<details>
<summary>Answer key</summary>
1. What operating condition causes backpressure to begin?
2. How do returned credits limit an upstream producer?
3. Why can additional buffering temporarily hide sustained overload?
</details>`,
        exploration: null,
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /at least three retrieval questions/,
  );
  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: `${validLesson()}\n## Learning objective\nA duplicate objective adds enough content to look superficially valid.`,
        critique: "A bounded critique [S2].",
        practice: validPractice(),
        exploration: null,
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /exactly once and in the required order/,
  );
  assert.throws(
    () =>
      evaluateDossierContent({
        lesson: validLesson(),
        critique: "A bounded critique [S2].",
        practice: validPractice()
          .replace("1. What operating", "7. What operating")
          .replace("2. How do returned", "8. How do returned")
          .replace("3. Why can additional", "9. Why can additional"),
        exploration: null,
        sources: [{ sourceId: "S1" }, { sourceId: "S2" }],
        blueprint: validBlueprint(),
      }),
    /numbered sequentially from 1/,
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
Explain how flow-control signals prevent a fast producer from overwhelming a slower consumer [S1].
## Two-minute recall
Recall how queue acknowledgements reveal which messages a consumer has safely processed.
## Why this matters
Uncontrolled overload can spread across service boundaries and turn one slow consumer into a system-wide failure [S2].
## Mental model
Think of production and consumption as two rates joined by a feedback signal that regulates pressure.
## How it works
Credits constrain a producer by making further sends conditional on capacity returned by the consumer.
## Worked example
A consumer slows during a database incident, exhausts its credits, and automatically pauses upstream publishing.
## Common misconception
A larger buffer delays visible failure, but it does not create processing capacity or remove sustained overload.
## Practical experiment
Change the simulated consumer service rate and observe when queue depth and producer latency begin to rise.
## Takeaway
Backpressure is a feedback mechanism that protects finite downstream capacity by regulating upstream demand.`;
}

function validPractice() {
  return `## Retrieval practice
1. What operating condition causes backpressure to begin?
2. How do returned credits limit an upstream producer?
3. Why can additional buffering temporarily hide sustained overload?

## Application challenge
Diagnose a slow consumer in a three-service pipeline and identify where the feedback signal should be applied.

<details>
<summary>Answer key</summary>

1. Backpressure begins when incoming demand persistently exceeds the consumer's available service rate.
2. A producer waits because it cannot send another unit until the consumer returns usable credit.
3. Extra buffering delays the visible symptom while sustained demand continues to exceed real processing capacity.
</details>`;
}

const REQUIRED_TINY_LESSON = `## Learning objective
x
## Two-minute recall
x
## Why this matters
x
## Mental model
x
## How it works
x
## Worked example
x
## Common misconception
x
## Practical experiment
x
## Takeaway
x`;
