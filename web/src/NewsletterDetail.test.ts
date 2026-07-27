import { describe, expect, it } from "vitest";
import { lessonPresentation } from "./NewsletterDetail";

describe("stream lesson presentation", () => {
  it("shows a completed latest lesson as available for review", () => {
    expect(lessonPresentation({ progress: 100, completed: true })).toEqual({
      status: "Completed",
      cta: "Review lesson",
      historyCta: "Review",
      description: "You completed this lesson. Revisit it whenever you want to refresh the idea.",
    });
  });

  it("keeps partially read lessons resumable", () => {
    expect(lessonPresentation({ progress: 42, completed: false })).toMatchObject({
      status: "42% read",
      cta: "Continue lesson",
      historyCta: "Continue",
    });
  });
});
