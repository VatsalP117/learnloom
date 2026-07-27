import { describe, expect, it } from "vitest";
import { selectTodayLessons } from "./TodayPage";

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
});
