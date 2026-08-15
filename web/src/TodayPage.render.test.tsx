import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { artworkForStream } from "./todayArtwork";

const { useWorkspaceMock, defaultWorkspace, heroNewsletter, heroLesson } = vi.hoisted(() => {
  const createdAt = (daysAgo: number) => {
    const value = new Date();
    value.setDate(value.getDate() - daysAgo);
    value.setHours(8, 0, 0, 0);
    return value.toISOString();
  };

  const heroNewsletter = {
    id: "hero",
    name: "Hero Stream",
    topic: "Hero topic",
    active: true,
    lessonMinutes: 12,
  };
  const otherNewsletter = {
    id: "other",
    name: "Other Stream",
    topic: "Other topic description",
    active: true,
    lessonMinutes: 10,
  };
  const pausedNewsletter = {
    id: "paused",
    name: "Paused Stream",
    topic: "Paused topic",
    active: false,
  };
  const heroLesson = {
    id: "hero-1",
    title: "Hero lesson title",
    status: "generated",
    newsletterId: "hero",
    newsletter: heroNewsletter,
    createdAt: createdAt(0),
  };
  const otherLesson = {
    id: "other-1",
    title: "Other lesson title",
    status: "generated",
    newsletterId: "other",
    newsletter: otherNewsletter,
    createdAt: createdAt(1),
  };
  const reviews = [
    {
      id: "review-1",
      issueId: "some-issue",
      objective: "Recall the mechanism.",
      prompt: "Name the mechanism.",
      answerRubric: "Accurate mechanism.",
      correctiveExplanation: "Reopen the lesson.",
      stage: 0,
      dueAt: createdAt(0),
    },
    {
      id: "review-2",
      issueId: "some-issue",
      objective: "Recall the evidence.",
      prompt: "What evidence supports it?",
      answerRubric: "Accurate evidence.",
      correctiveExplanation: "Reopen the lesson.",
      stage: 0,
      dueAt: createdAt(0),
    },
    {
      id: "review-3",
      issueId: "some-issue",
      objective: "Recall the limits.",
      prompt: "What is an important limit?",
      answerRubric: "Accurate limit.",
      correctiveExplanation: "Reopen the lesson.",
      stage: 0,
      dueAt: createdAt(0),
    },
  ];

  const defaultWorkspace = {
    snapshot: { todayFocus: undefined, retention: undefined },
    newsletters: [heroNewsletter, otherNewsletter, pausedNewsletter],
    lessons: [heroLesson, otherLesson],
    reviews,
    error: "",
    loading: false,
    loadingMore: false,
    hasMore: false,
    loadMore: () => Promise.resolve(),
    reload: () => Promise.resolve(),
  };

  return { useWorkspaceMock: vi.fn(() => defaultWorkspace), defaultWorkspace, heroNewsletter, heroLesson };
});

vi.mock("./useWorkspace", () => ({
  useWorkspace: useWorkspaceMock,
}));

import TodayPage from "./TodayPage";

function stubWindow(progress: Record<string, { progress: number; completed: boolean }>) {
  const values = new Map<string, string>();
  values.set(
    "learnloom.learning-state.v1",
    JSON.stringify({ lessons: progress }),
  );
  vi.stubGlobal("window", {
    localStorage: {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
    },
    dispatchEvent: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  });
}

describe("redesigned Today page render", () => {
  beforeEach(() => {
    useWorkspaceMock.mockReturnValue(defaultWorkspace);
    stubWindow({ "hero-1": { progress: 35, completed: false } });
  });

  it("keeps the lesson hero dominant with deterministic stream artwork", () => {
    const markup = renderToStaticMarkup(<TodayPage />);
    const heroArt = artworkForStream("hero");

    expect(markup).toMatch(/Good (morning|afternoon|evening)/);
    expect(markup).toContain("Hero lesson title");
    expect(markup).toContain("Continue learning");
    expect(markup).toContain("35%");
    expect(markup).toContain("Resume lesson");
    expect(markup).toContain(`src="${heroArt.hero}"`);
    expect(markup).toContain("1440w");
  });

  it("shows the real due review count, never a hardcoded number", () => {
    const markup = renderToStaticMarkup(<TodayPage />);

    expect(markup).toContain("3 prompts due");
    expect(markup).not.toContain("24 prompts");
  });

  it("renders Another thread with the stream topic and its stream link", () => {
    const markup = renderToStaticMarkup(<TodayPage />);

    expect(markup).toContain("Another thread");
    expect(markup).toContain("Other topic description");
    expect(markup).toContain('href="/newsletters/other"');
  });

  it("shows other active streams in the strip with real progress, excluding the hero", () => {
    const markup = renderToStaticMarkup(<TodayPage />);

    expect(markup).toContain("Other Stream");
    expect(markup).toContain("Not started");
    expect(markup).toContain("10 min left");
    expect(markup).not.toContain("Paused Stream");
    expect(markup).toContain('href="/streams"');
    expect(markup).toContain("View all streams");
  });

  it("preserves the gentle re-entry hero and its controls", () => {
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      snapshot: {
        ...defaultWorkspace.snapshot,
        retention: {
          inactive: true,
          returnedAfterSevenDays: true,
          daysAway: 9,
          reentryNewsletterId: "other",
        },
      },
      reviews: [],
    });
    stubWindow({});

    const markup = renderToStaticMarkup(<TodayPage />);

    expect(markup).toContain("Begin with one useful step.");
    expect(markup).toContain("Slow to weekly");
    expect(markup).toContain("Pause stream");
    expect(markup).toContain("Clear older backlog");
  });

  it("renders the caught-up empty state when everything is complete", () => {
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      snapshot: { ...defaultWorkspace.snapshot },
      newsletters: [heroNewsletter],
      lessons: [heroLesson],
      reviews: [],
    });
    stubWindow({ "hero-1": { progress: 100, completed: true } });

    const markup = renderToStaticMarkup(<TodayPage />);

    expect(markup).toContain("You are caught up.");
    expect(markup).toContain("Open your library");
    expect(markup).not.toContain("Resume lesson");
  });

  it("renders the first-stream empty state without any streams", () => {
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      newsletters: [],
      lessons: [],
      reviews: [],
    });

    const markup = renderToStaticMarkup(<TodayPage />);

    expect(markup).toContain("Turn a question into a learning practice.");
    expect(markup).toContain("Create your first stream");
  });
});
