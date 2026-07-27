import { beforeEach, describe, expect, it, vi } from "vitest";
import { lessonState, syncLessonProgress } from "./learningState";

describe("server-backed lesson progress", () => {
  beforeEach(() => {
    const values = new Map<string, string>();
    vi.stubGlobal("window", {
      localStorage: {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => values.set(key, value),
      },
      dispatchEvent: vi.fn(),
    });
  });

  it("hydrates durable completion into the browser state", () => {
    syncLessonProgress([{
      issueId: "lesson-1",
      progress: 100,
      completedAt: "2026-07-27T12:00:00Z",
    }]);

    expect(lessonState("lesson-1")).toMatchObject({
      progress: 100,
      completed: true,
      completedAt: "2026-07-27T12:00:00Z",
    });
  });
});
