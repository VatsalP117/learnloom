import { describe, expect, it } from "vitest";
import {
  resolveTodaySelection,
  selectTodayFocus,
  selectTodayLessons,
} from "./TodayPage";

const stream = { id: "stream-1", active: true };

describe("Today lesson selection", () => {
  it("does not put a completed lesson back in the primary card", () => {
    const lessons = [{
      id: "lesson-1",
      status: "generated",
      newsletter: stream,
    }];
    const selected = selectTodayLessons(
      lessons,
      () => ({ progress: 100, completed: true }),
    );

    expect(selected.primary).toBeUndefined();
    expect(selected.secondary).toBeUndefined();
  });

  it("prefers an in-progress unread lesson", () => {
    const lessons = [
      { id: "lesson-new", status: "generated", newsletter: stream },
      { id: "lesson-started", status: "generated", newsletter: stream },
    ];
    const selected = selectTodayLessons(
      lessons,
      (id) => ({ progress: id === "lesson-started" ? 35 : 0, completed: false }),
    );

    expect(selected.primary?.id).toBe("lesson-started");
    expect(selected.secondary?.id).toBe("lesson-new");
  });

  it("prioritizes due retrieval before an untouched lesson", () => {
    const selected = selectTodayFocus(
      [{ id: "lesson-new", status: "generated", newsletter: stream }],
      [{ id: "review-1", issueId: "lesson-complete" }],
      () => ({ progress: 0, completed: false }),
    );

    expect(selected.focus).toBe("review");
    expect(selected.primary?.id).toBe("lesson-new");
  });

  it("preserves reading momentum ahead of due retrieval", () => {
    const selected = selectTodayFocus(
      [{ id: "lesson-started", status: "generated", newsletter: stream }],
      [{ id: "review-1", issueId: "lesson-complete" }],
      () => ({ progress: 35, completed: false }),
    );

    expect(selected.focus).toBe("lesson");
  });

  it("collapses backlog pressure into one re-entry action", () => {
    const selected = selectTodayFocus(
      [{ id: "lesson-new", status: "generated", newsletter: stream }],
      [{ id: "review-1", issueId: "lesson-complete" }],
      () => ({ progress: 0, completed: false }),
      { inactive: true, actionLabel: "Recall one idea", actionUrl: "/review" },
    );

    expect(selected.focus).toBe("reentry");
  });

  it("uses the stored server selection and learner-facing reason", () => {
    const lessons = [
      { id: "newest", title: "Newest", status: "generated", newsletter: stream },
      { id: "neglected", title: "Neglected", status: "generated", newsletter: stream },
    ];
    const selected = resolveTodaySelection(
      {
        kind: "lesson",
        subjectId: "neglected",
        reason: "This path has waited longer than your other active work.",
        actionLabel: "Begin lesson",
        actionUrl: "/issues/neglected",
      },
      lessons,
      [],
      () => ({ progress: 0, completed: false }),
    );

    expect(selected.primary?.id).toBe("neglected");
    expect(selected.reason).toContain("waited longer");
    expect(selected.actionUrl).toBe("/issues/neglected");
  });

  it("can hydrate a selected lesson outside the first workspace page", () => {
    const selected = resolveTodaySelection(
      {
        kind: "lesson",
        subjectId: "older-selected",
        newsletterId: "stream-2",
        title: "A neglected but worthwhile lesson",
        newsletterName: "Systems",
        lessonMinutes: 12,
        reason: "Return to this path.",
        actionLabel: "Begin lesson",
        actionUrl: "/issues/older-selected",
      },
      [],
    );

    expect(selected.primary?.id).toBe("older-selected");
    expect(selected.primary?.newsletter.name).toBe("Systems");
  });
});
