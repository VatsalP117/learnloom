import { describe, expect, it } from "vitest";
import { normalizeDossier } from "./dossierView";

describe("normalizeDossier", () => {
  it("projects a stored Dossier into the lesson reader shape", () => {
    const result = normalizeDossier({
      curation: { rationale: "Why the mechanism matters." },
      blueprint: { learningObjective: "Explain the mechanism." },
      lesson: "## Central mechanism\n\nA clear explanation.\n\n## Worked example\n\nA useful example.",
      critique: "## Skeptical review\n\nA meaningful limitation.",
      practice: "## Retrieval practice\n\n1. What drives the mechanism?\n2. When does it fail?\n\n## Application challenge\n\nApply it to a current project.",
    }, { lessonMinutes: 12 });

    expect(result.readTime).toBe(12);
    expect(result.sections).toHaveLength(3);
    expect(result.retrieval).toEqual([
      "What drives the mechanism?",
      "When does it fail?",
    ]);
    expect(result.application).toBe("Apply it to a current project.");
    expect(result.limitations).toEqual([]);
  });

  it("preserves structured orientation, evidence markers, and answer rubrics", () => {
    const result = normalizeDossier({
      lesson: "## Mental model\n\nRetrieval changes later access [S1] [S2].",
      critique: "## Skeptical review\n\nFeedback quality changes the result [S2].",
      practice: "## Retrieval practice\n\n1. Why does retrieval help?",
      learning: {
        selectionRationale: "This advances the learner’s current goal.",
        continuityBridge: "It builds on recognition versus recall.",
        concepts: [{ label: "Retrieval strength" }],
        retrieval: [{
          id: "retrieval-1",
          prompt: "Why does retrieval help?",
          answerRubric: "It strengthens later access.",
        }],
        claims: [{
          id: "claim-1",
          text: "Retrieval improves later access.",
          sourceIds: ["S1"],
        }],
        limitations: [{
          id: "limitation-1",
          text: "Feedback quality changes the result.",
          sourceIds: ["S2"],
        }],
        lessonType: "foundation",
        evidenceStatus: "source_bounded",
        suggestedNextConcepts: ["Spacing effects"],
        application: "Compare recall with rereading.",
      },
    }, { lessonMinutes: 10 });

    expect(result.whyNow).toContain("current goal");
    expect(result.continuityBridge).toContain("recognition");
    expect(result.concepts).toEqual(["Retrieval strength"]);
    expect(result.retrievalItems[0].answerRubric).toContain("later access");
    expect(result.claims[0]).toEqual({
      id: "claim-1",
      text: "Retrieval improves later access.",
      sourceIds: ["S1"],
    });
    expect(result.limitations[0].sourceIds).toEqual(["S2"]);
    expect(result.lessonType).toBe("foundation");
    expect(result.evidenceStatus).toBe("source_bounded");
    expect(result.nextConcepts).toEqual(["Spacing effects"]);
    expect(result.sections[1].heading).toBe("Limits and verification");
    expect(result.sections[0].paragraphs[0]).toEqual({
      text: "Retrieval changes later access.",
      sourceIds: ["S1", "S2"],
    });
  });
});
