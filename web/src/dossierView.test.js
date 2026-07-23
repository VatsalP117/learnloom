import { describe, expect, it } from "vitest";
import { normalizeDossier } from "./dossierView.js";

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
  });
});
