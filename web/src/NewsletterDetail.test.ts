import { describe, expect, it } from "vitest";
import {
  capabilityEvidence,
  lessonPresentation,
  latestGeneratedIssue,
  nextSelectedWeekdays,
  publicationAudience,
  recoveryAction,
  rhythmScheduleLabel,
  sourceRoleLabel,
  streamEmptyPresentation,
} from "./NewsletterDetail";

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

describe("learning rhythm controls", () => {
  it("keeps at least one selected weekday and supports one-day synthesis", () => {
    expect(nextSelectedWeekdays([1], 1)).toEqual([1]);
    expect(nextSelectedWeekdays([1, 3], 1)).toEqual([3]);
    expect(nextSelectedWeekdays([1, 3], 5)).toEqual([1, 3, 5]);
    expect(nextSelectedWeekdays([1, 3], 5, true)).toEqual([5]);
  });

  it("explains effective cadence instead of hiding automatic slowdown", () => {
    expect(rhythmScheduleLabel({
      id: "stream-1",
      name: "AI evaluation",
      topic: "AI evaluation",
      scheduleTime: "08:30",
      rhythmMode: "daily",
      effectiveRhythmMode: "weekly_synthesis",
      rhythmThrottledAt: "2026-08-12T00:00:00Z",
    })).toBe("Slowed to weekly · 08:30");
  });
});

describe("capability evidence", () => {
  it("uses completed and reviewed concept evidence instead of lesson-count progress", () => {
    expect(capabilityEvidence({
      completedCount: 1,
      reviewAttemptCount: 0,
      confidenceScore: 0,
    })).toContain("retrieval not attempted yet");
    expect(capabilityEvidence({
      completedCount: 2,
      reviewAttemptCount: 2,
      confidenceScore: 85,
    })).toBe("2 retrieval attempts · solid recall");
  });
});

describe("publication audience", () => {
  const site = { username: "alan", displayName: "Alan", visibility: "public", searchIndexing: false };
  const stream = { id: "stream-1", name: "Systems", topic: "systems", siteVisible: true };

  it("makes every content state and effective audience explicit", () => {
    expect(publicationAudience({ publicationState: "private" }, site, stream)).toContain("only you");
    expect(publicationAudience({ publicationState: "draft" }, site, stream)).toContain("until you publish");
    expect(publicationAudience({ publicationState: "published" }, { ...site, visibility: "private" }, stream))
      .toContain("site is private");
    expect(publicationAudience({ publicationState: "published" }, site, { ...stream, siteVisible: false }))
      .toContain("stream is private");
    expect(publicationAudience({ publicationState: "published" }, site, stream))
      .toBe("Public by link · anyone can read it; search discovery is off");
  });
});

describe("stream preparation and recovery presentation", () => {
  it("sets one honest preparation range while the first lesson is queued", () => {
    expect(streamEmptyPresentation({ status: "queued" })).toEqual({
      title: "Your first lesson is being prepared.",
      description: expect.stringContaining("5–15 minutes"),
    });
  });

  it("does not describe a deferred attempt as an unrequested first lesson", () => {
    expect(streamEmptyPresentation({
      status: "deferred",
      error: "There isn’t enough worthwhile evidence yet.",
    })).toEqual({
      title: "Waiting for stronger evidence.",
      description: "There isn’t enough worthwhile evidence yet.",
    });
  });

  it("makes the source approval pause explicit before lesson generation", () => {
    expect(streamEmptyPresentation({ status: "awaiting_approval" })).toEqual({
      title: "Your sources are ready to review.",
      description: expect.stringContaining("approve it to begin lesson generation"),
    });
  });

  it("keeps an established stream anchored to its latest usable lesson", () => {
    expect(latestGeneratedIssue([
      { id: "failed-new", status: "failed" },
      { id: "lesson-previous", status: "generated" },
    ])?.id).toBe("lesson-previous");
  });

  it("maps safe failure categories to an executable recovery action", () => {
    expect(recoveryAction({
      status: "failed",
      failureCategory: "content_quality",
      failureRetryable: true,
    }, "discovered")).toMatchObject({ kind: "retry" });

    expect(recoveryAction({
      status: "failed",
      failureCategory: "user_actionable",
      failureRetryable: false,
    }, "provided")).toMatchObject({ kind: "broaden_sources" });

    expect(recoveryAction({
      status: "failed",
      failureCategory: "provider",
      failureRetryable: false,
    }, "hybrid")).toMatchObject({ kind: "contact_support" });
  });

  it("explains source roles without presenting a numeric trust score", () => {
    expect(sourceRoleLabel("official_primary")).toBe("Official or primary");
    expect(sourceRoleLabel("counterweight")).toBe("Counterweight");
    expect(sourceRoleLabel(undefined, "provided")).toBe("Chosen by you");
  });
});
