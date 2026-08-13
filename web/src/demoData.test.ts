import { describe, expect, it } from "vitest";
import { demoResponse } from "./demoData";

describe("demo onboarding draft", () => {
  it("survives navigation-style reads until explicitly discarded", () => {
    demoResponse("/api/onboarding/draft", { method: "DELETE" });
    demoResponse("/api/onboarding/draft", {
      method: "PUT",
      body: {
        draftId: "8f42a08f-e6ab-4fab-92fd-f16a3e6cf33a",
        step: 2,
        payload: {
          topic: "AI evaluation",
          learnerGoal: "design reliable tests",
          sourceMode: "discovered",
        },
      },
    });
    expect(demoResponse("/api/onboarding/draft", { method: "GET" })).toMatchObject({
      draft: {
        id: "8f42a08f-e6ab-4fab-92fd-f16a3e6cf33a",
        step: 2,
        payload: {
          topic: "AI evaluation",
          learnerGoal: "design reliable tests",
        },
      },
    });
    demoResponse("/api/onboarding/draft", { method: "DELETE" });
    expect(demoResponse("/api/onboarding/draft", { method: "GET" })).toEqual({ draft: null });
  });
});

describe("demo lesson publishing trust", () => {
  it("provides a stable moderation snapshot and supports owner changes", () => {
    const path = "/api/issues/ai-evaluation-issue-1/moderation";
    expect(demoResponse(path)).toMatchObject({
      state: "clear",
      reason: "",
      corrections: [],
      reports: [],
    });
    expect(demoResponse(path, {
      method: "POST",
      body: { state: "held", reason: "Checking a disputed claim." },
    })).toMatchObject({
      state: "held",
      reason: "Checking a disputed claim.",
    });
    expect(demoResponse("/api/issues/ai-evaluation-issue-1/corrections", {
      method: "POST",
      body: { body: "The source version was updated." },
    })).toMatchObject({ body: "The source version was updated." });
    expect(demoResponse(path)).toMatchObject({
      state: "held",
      corrections: [{ body: "The source version was updated." }],
    });
  });
});
