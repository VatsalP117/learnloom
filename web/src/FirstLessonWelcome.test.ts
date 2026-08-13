import { describe, expect, it } from "vitest";
import { orientationPlan } from "./FirstLessonWelcome";

describe("first lesson orientation", () => {
  it("turns the learner intent into an honest pre-evidence learning arc", () => {
    const plan = orientationPlan({
      id: "stream-1",
      name: "AI systems",
      topic: "how AI systems learn and fail",
      learnerLevel: "advanced",
      learnerGoal: "evaluate product claims",
    });
    expect(plan.title).toContain("how AI systems learn and fail");
    expect(plan.objective).toContain("Stress-test");
    expect(plan.concepts).toHaveLength(4);
    expect(plan.concepts[3]).toContain("evaluate product claims");
  });
});
