import { describe, expect, it } from "vitest";
import {
  classifyStreams,
  latestIncompleteLesson,
  rankActiveStreams,
  selectFeaturedStream,
  streamProgress,
  streamStatus,
} from "./StreamsPage";
import type { Newsletter } from "./types";

const stream: Newsletter = { id: "stream-1", name: "Systems", topic: "systems" };
const lesson = (id: string, newsletterId = stream.id, createdAt = 0) => ({
  id,
  newsletterId,
  status: "generated",
  createdAt: new Date(Date.now() - createdAt).toISOString(),
});
const state = (map: Record<string, { progress: number; completed: boolean }>) =>
  (issueId: string) =>
    map[issueId] ?? { progress: 0, completed: false };

describe("stream status classification", () => {
  it("is active with incomplete generated lessons", () => {
    expect(streamStatus({ ...stream, active: true }, [lesson("l-1")], state({
      "l-1": { progress: 30, completed: false },
    }))).toBe("active");
  });

  it("is paused when the stream is inactive", () => {
    expect(streamStatus({ ...stream, active: false }, [lesson("l-1")], state({
      "l-1": { progress: 30, completed: false },
    }))).toBe("paused");
  });

  it("is completed only with at least one generated lesson, all completed", () => {
    expect(streamStatus({ ...stream, active: true }, [lesson("l-1")], state({
      "l-1": { progress: 100, completed: true },
    }))).toBe("completed");
    expect(streamStatus({ ...stream, active: false }, [lesson("l-1")], state({
      "l-1": { progress: 100, completed: true },
    }))).toBe("completed");
  });

  it("never classifies a stream without generated lessons as completed", () => {
    expect(streamStatus({ ...stream, active: false }, [], state({}))).toBe("paused");
    expect(streamStatus({ ...stream, active: true }, [], state({}))).toBe("active");
  });

  it("ignores lessons that are not generated", () => {
    const draft = { ...lesson("l-1"), status: "draft" };
    expect(streamStatus({ ...stream, active: true }, [draft], state({}))).toBe("active");
  });

  it("classifies each stream into exactly one bucket", () => {
    const newsletters = [
      { ...stream, id: "a", name: "A", active: true },
      { ...stream, id: "b", name: "B", active: false },
      { ...stream, id: "c", name: "C", active: true },
    ];
    const lessons = [
      lesson("a-1", "a"),
      lesson("b-1", "b"),
      lesson("c-1", "c"),
    ];
    const classified = classifyStreams(
      newsletters,
      lessons,
      state({ "c-1": { progress: 100, completed: true } }),
    );

    expect(classified.active.map((item) => item.id)).toEqual(["a"]);
    expect(classified.paused.map((item) => item.id)).toEqual(["b"]);
    expect(classified.completed.map((item) => item.id)).toEqual(["c"]);
    expect(
      [...classified.active, ...classified.paused, ...classified.completed].length,
    ).toBe(newsletters.length);
  });
});

describe("stream progress summary", () => {
  it("averages generated lessons, treating completed as 100", () => {
    const progress = streamProgress(
      { ...stream, active: true },
      [lesson("l-1"), lesson("l-2")],
      state({
        "l-1": { progress: 35, completed: false },
        "l-2": { progress: 0, completed: true },
      }),
    );

    expect(progress.lessonCount).toBe(2);
    expect(progress.completedCount).toBe(1);
    expect(progress.progress).toBe(68); // (35 + 100) / 2
  });

  it("clamps progress to 0-100", () => {
    expect(streamProgress(stream, [lesson("l-1")], state({
      "l-1": { progress: -20, completed: false },
    })).progress).toBe(0);
    expect(streamProgress(stream, [lesson("l-1")], state({
      "l-1": { progress: 150, completed: false },
    })).progress).toBe(100);
  });

  it("reports zero for a stream without generated lessons", () => {
    expect(streamProgress(stream, [], state({}))).toEqual({
      lessonCount: 0,
      completedCount: 0,
      progress: 0,
    });
  });
});

describe("featured stream selection", () => {
  const newsletters = [
    { ...stream, id: "alpha", name: "Alpha", active: true },
    { ...stream, id: "beta", name: "Beta", active: true },
    { ...stream, id: "gamma", name: "Gamma", active: false },
  ];
  const lessons = [lesson("alpha-1", "alpha", 1000), lesson("beta-1", "beta", 1000)];

  it("prefers the todayFocus stream when it is still active", () => {
    const featured = selectFeaturedStream(
      newsletters,
      lessons,
      "beta",
      state({}),
    );

    expect(featured?.id).toBe("beta");
  });

  it("ignores a todayFocus stream that is paused or completed", () => {
    const featured = selectFeaturedStream(
      newsletters,
      lessons,
      "gamma",
      state({ "alpha-1": { progress: 40, completed: false } }),
    );

    expect(featured?.id).toBe("alpha"); // only active stream with progress
  });

  it("falls back to a stable active stream preferring incomplete lessons", () => {
    const featured = selectFeaturedStream(
      newsletters,
      lessons,
      undefined,
      state({ "alpha-1": { progress: 40, completed: false } }),
    );

    expect(featured?.id).toBe("alpha");
  });

  it("prefers incomplete lessons over untouched ones, then name for stability", () => {
    const noState = state({});
    const byName = [
      { ...stream, id: "alpha", name: "Aaa", active: true },
      { ...stream, id: "beta", name: "Zzz", active: true },
    ];
    const lessonsFor = [lesson("alpha-1", "alpha", 500)];

    // Only alpha has an unfinished generated lesson.
    expect(selectFeaturedStream(byName, lessonsFor, undefined, noState)?.id)
      .toBe("alpha");
    // With no lessons at all, fall back to name order for stability.
    expect(selectFeaturedStream(byName, [], undefined, noState)?.id)
      .toBe("alpha");
    expect(selectFeaturedStream(
      [
        { ...stream, id: "x", name: "Bbb", active: true },
        { ...stream, id: "y", name: "Aaa", active: true },
      ],
      [],
      undefined,
      noState,
    )?.id).toBe("y");
  });

  it("returns undefined when there are no active streams", () => {
    const allPaused = newsletters.map((item) => ({ ...item, active: false }));
    expect(selectFeaturedStream(allPaused, lessons, undefined, state({}))).toBeUndefined();
  });

  it("ranks active cards below the hero, excluding the featured stream", () => {
    const ranked = rankActiveStreams(
      newsletters,
      lessons,
      "alpha",
      state({ "alpha-1": { progress: 40, completed: false } }),
    );

    expect(ranked.map((item) => item.id)).toEqual(["beta"]);
  });

  it("does not rank an active stream whose generated lessons are complete", () => {
    const ranked = rankActiveStreams(
      newsletters,
      lessons,
      undefined,
      state({ "alpha-1": { progress: 100, completed: true } }),
    );

    expect(ranked.map((item) => item.id)).toEqual(["beta"]);
  });
});

describe("latest incomplete lesson", () => {
  it("returns the most recent incomplete generated lesson", () => {
    const lessons = [
      lesson("old", stream.id, 1000),
      lesson("new", stream.id, 100),
      lesson("done", stream.id, 50),
    ];
    const result = latestIncompleteLesson(
      stream,
      lessons,
      state({ done: { progress: 100, completed: true } }),
    );

    expect(result?.id).toBe("new");
  });

  it("is undefined when every generated lesson is completed", () => {
    expect(latestIncompleteLesson(
      stream,
      [lesson("l-1")],
      state({ "l-1": { progress: 100, completed: true } }),
    )).toBeUndefined();
  });
});
