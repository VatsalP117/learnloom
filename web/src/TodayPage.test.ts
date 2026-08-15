import { describe, expect, it } from "vitest";
import {
  rankTodayStreams,
  resolveTodaySelection,
  selectAnotherThread,
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

describe("Today stream ranking", () => {
  const at = (hoursAgo: number) =>
    new Date(Date.now() - hoursAgo * 3600 * 1000).toISOString();

  it("ranks other active streams by their most recent unfinished lesson", () => {
    const streams = [
      { id: "hero-stream", name: "Hero", topic: "Hero", active: true },
      { id: "older", name: "Older work", topic: "Older", active: true },
      { id: "newer", name: "Newer work", topic: "Newer", active: true },
    ];
    const lessons = [
      { id: "older-1", newsletterId: "older", status: "generated", createdAt: at(48) },
      { id: "newer-1", newsletterId: "newer", status: "generated", createdAt: at(2) },
    ];

    const ranked = rankTodayStreams(
      streams,
      lessons,
      "hero-stream",
      () => ({ progress: 0, completed: false }),
    );

    expect(ranked.map((entry) => entry.newsletter.id)).toEqual(["newer", "older"]);
    expect(ranked[0].lesson.id).toBe("newer-1");
  });

  it("excludes the hero stream and inactive streams", () => {
    const streams = [
      { id: "hero", name: "Hero", topic: "Hero", active: true },
      { id: "paused", name: "Paused", topic: "Paused", active: false },
      { id: "other", name: "Other", topic: "Other", active: true },
    ];
    const lessons = [
      { id: "hero-1", newsletterId: "hero", status: "generated", createdAt: at(1) },
      { id: "paused-1", newsletterId: "paused", status: "generated", createdAt: at(1) },
      { id: "other-1", newsletterId: "other", status: "generated", createdAt: at(3) },
    ];

    const ranked = rankTodayStreams(
      streams,
      lessons,
      "hero",
      () => ({ progress: 0, completed: false }),
    );

    expect(ranked.map((entry) => entry.newsletter.id)).toEqual(["other"]);
  });

  it("omits streams whose only lessons are completed", () => {
    const streams = [{ id: "done", name: "Done", topic: "Done", active: true }];
    const lessons = [{ id: "done-1", newsletterId: "done", status: "generated", createdAt: at(1) }];

    const ranked = rankTodayStreams(
      streams,
      lessons,
      null,
      () => ({ progress: 100, completed: true }),
    );

    expect(ranked).toEqual([]);
  });

  it("reports real progress and remaining minutes when available", () => {
    const streams = [{ id: "stream", name: "Stream", topic: "Stream", active: true, lessonMinutes: 12 }];
    const lessons = [{ id: "lesson-1", newsletterId: "stream", status: "generated", createdAt: at(1) }];

    const [entry] = rankTodayStreams(
      streams,
      lessons,
      null,
      () => ({ progress: 25, completed: false }),
    );

    expect(entry.progress).toBe(25);
    expect(entry.remainingMinutes).toBe(9);
  });

  it("caps the strip at three streams", () => {
    const streams = ["a", "b", "c", "d"].map((id) => ({
      id,
      name: id.toUpperCase(),
      topic: id,
      active: true,
    }));
    const lessons = streams.map((stream, index) => ({
      id: `${stream.id}-1`,
      newsletterId: stream.id,
      status: "generated",
      createdAt: at(index + 1),
    }));

    const ranked = rankTodayStreams(
      streams,
      lessons,
      null,
      () => ({ progress: 0, completed: false }),
    );

    expect(ranked).toHaveLength(3);
  });

  it("another thread prefers the top-ranked stream and falls back by name", () => {
    const streams = [
      { id: "hero", name: "Hero", topic: "Hero", active: true },
      { id: "brand-new", name: "Brand new", topic: "Brand new", active: true },
      { id: "active", name: "Active", topic: "Active", active: true },
    ];
    const lessons = [{ id: "active-1", newsletterId: "active", status: "generated", createdAt: at(2) }];

    expect(selectAnotherThread(streams, lessons, "hero", () => ({ progress: 0, completed: false }))?.id)
      .toBe("active");
    expect(selectAnotherThread(streams, [], "hero", () => ({ progress: 0, completed: false }))?.id)
      .toBe("active");
    expect(selectAnotherThread(streams, [], "active", () => ({ progress: 0, completed: false }))?.id)
      .toBe("brand-new");
  });
});
