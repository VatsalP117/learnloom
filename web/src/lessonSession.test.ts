import { describe, expect, it } from "vitest";
import {
  canRevealRetrieval,
  initialRetrievalState,
  resumeLessonProgress,
} from "./lessonSession";

describe("durable lesson session", () => {
  it("resumes from the most advanced bounded progress", () => {
    expect(resumeLessonProgress(62, 41)).toBe(62);
    expect(resumeLessonProgress(12, 74)).toBe(74);
    expect(resumeLessonProgress(120, 20)).toBe(100);
  });

  it("restores drafts without revealing and completed attempts with reveal", () => {
    const state = initialRetrievalState([
      {
        issueId: "issue-1",
        promptKey: "retrieval-1",
        response: "A saved draft answer",
        skipped: false,
        updatedAt: "2026-08-12T00:00:00Z",
      },
      {
        issueId: "issue-1",
        promptKey: "retrieval-2",
        response: "A revealed answer",
        skipped: false,
        revealedAt: "2026-08-12T00:01:00Z",
        updatedAt: "2026-08-12T00:01:00Z",
      },
    ]);

    expect(state["retrieval-1"].revealed).toBe(false);
    expect(state["retrieval-2"].revealed).toBe(true);
  });

  it("rejects empty or token non-attempts before reveal", () => {
    expect(canRevealRetrieval("  ")).toBe(false);
    expect(canRevealRetrieval("no")).toBe(false);
    expect(canRevealRetrieval("A mechanism and its limit.")).toBe(true);
  });
});
