import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { artworkForStream } from "./todayArtwork";

const { useWorkspaceMock, defaultWorkspace } = vi.hoisted(() => {
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
    learnerLevel: "intermediate",
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
  const doneNewsletter = {
    id: "done",
    name: "Done Stream",
    topic: "Done topic",
    active: true,
  };
  const lessons = [
    {
      id: "hero-1",
      title: "Hero lesson",
      status: "generated",
      newsletterId: "hero",
      createdAt: createdAt(0),
    },
    {
      id: "hero-2",
      title: "Older hero lesson",
      status: "generated",
      newsletterId: "hero",
      createdAt: createdAt(2),
    },
    {
      id: "other-1",
      title: "Other lesson",
      status: "generated",
      newsletterId: "other",
      createdAt: createdAt(1),
    },
    {
      id: "done-1",
      title: "Done lesson",
      status: "generated",
      newsletterId: "done",
      createdAt: createdAt(3),
    },
  ];

  const defaultWorkspace = {
    snapshot: {
      todayFocus: {
        kind: "lesson",
        subjectId: "hero-1",
        newsletterId: "hero",
        reasonCode: "test",
        reason: "",
        actionLabel: "",
        actionUrl: "",
        score: 0,
        components: {},
        selectedAt: createdAt(0),
      },
      retention: undefined,
    },
    newsletters: [heroNewsletter, otherNewsletter, pausedNewsletter, doneNewsletter],
    lessons,
    reviews: [],
    error: "",
    loading: false,
    loadingMore: false,
    hasMore: false,
    loadMore: () => Promise.resolve(),
    reload: () => Promise.resolve(),
  };

  return { useWorkspaceMock: vi.fn(() => defaultWorkspace), defaultWorkspace };
});

vi.mock("./useWorkspace", () => ({
  useWorkspace: useWorkspaceMock,
}));

import StreamsPage from "./StreamsPage";

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

describe("redesigned Streams page render", () => {
  beforeEach(() => {
    useWorkspaceMock.mockReturnValue(defaultWorkspace);
    stubWindow({
      "hero-1": { progress: 35, completed: false },
      "hero-2": { progress: 100, completed: true },
      "other-1": { progress: 0, completed: false },
      "done-1": { progress: 100, completed: true },
    });
  });

  it("renders the redesigned header without the search/filter toolbar", () => {
    const markup = renderToStaticMarkup(<StreamsPage />);

    expect(markup).toContain("Your learning journeys");
    expect(markup).toContain("<h1>Streams</h1>");
    expect(markup).toContain("Learning streams");
    expect(markup).toContain("New learning stream");
    expect(markup).toContain('href="/newsletters/new"');
    expect(markup).not.toContain("Search learning streams");
    expect(markup).not.toContain("Subjects you are following");
  });

  it("features the todayFocus stream with progress, lesson count, and minutes", () => {
    const markup = renderToStaticMarkup(<StreamsPage />);
    const heroArt = artworkForStream("hero");

    expect(markup).toContain("Hero Stream");
    expect(markup).toContain("Hero topic");
    expect(markup).toContain("2 lessons · 12 min");
    expect(markup).toContain("68%"); // (35 + 100) / 2
    expect(markup).toContain("Continue stream");
    expect(markup).toContain('href="/issues/hero-1"');
    expect(markup).toContain(`src="${heroArt.hero}"`);
    expect(markup).toContain("1440w");
  });

  it("links the hero to the stream detail when no lesson is unfinished", () => {
    const freshNewsletter = {
      id: "new",
      name: "New Stream",
      topic: "New topic",
      active: true,
      lessonMinutes: 8,
    };
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      snapshot: {
        ...defaultWorkspace.snapshot,
        todayFocus: {
          ...defaultWorkspace.snapshot.todayFocus,
          newsletterId: "new",
        },
      },
      newsletters: [...defaultWorkspace.newsletters, freshNewsletter],
    });

    const markup = renderToStaticMarkup(<StreamsPage />);

    expect(markup).toContain("New Stream");
    expect(markup).toContain("Open stream");
    expect(markup).toContain('href="/newsletters/new"');
  });

  it("renders active stream cards below the hero, excluding the featured stream", () => {
    const markup = renderToStaticMarkup(<StreamsPage />);
    const otherArt = artworkForStream("other");

    expect(markup).toContain("Active streams");
    expect(markup).toContain("Other Stream");
    expect(markup).toContain("Other topic description");
    expect(markup).toContain("1 lesson");
    expect(markup).toContain("10 min");
    expect(markup).toContain("0%");
    expect(markup).toContain('href="/newsletters/other"');
    expect(markup).not.toContain('href="/newsletters/hero"');
    expect(markup).toContain(`src="${otherArt.card}"`);
    expect(markup).toContain("680w");
  });

  it("lists paused and completed streams as neutral rows without artwork", () => {
    const markup = renderToStaticMarkup(<StreamsPage />);

    expect(markup).toContain("Paused Stream");
    expect(markup).toContain("Done Stream");
    expect(markup).not.toContain("Paused topic");
    expect(markup).not.toContain("Done topic");
    expect(markup).toContain("Completed");
    expect(markup).toContain('href="/newsletters/paused"');
    expect(markup).toContain('href="/newsletters/done"');
    // Shell favicon + hero artwork + the one active card artwork — the
    // paused/completed rows carry no decorative images.
    expect((markup.match(/<img/g) ?? []).length).toBe(3);
  });

  it("falls back to a stable active stream when todayFocus is not active", () => {
    stubWindow({
      "hero-1": { progress: 100, completed: true },
      "hero-2": { progress: 100, completed: true },
      "other-1": { progress: 0, completed: false },
      "done-1": { progress: 100, completed: true },
    });
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      snapshot: {
        ...defaultWorkspace.snapshot,
        todayFocus: { ...defaultWorkspace.snapshot.todayFocus, newsletterId: "paused" },
      },
    });

    const markup = renderToStaticMarkup(<StreamsPage />);
    const heroArt = artworkForStream("other");

    expect(markup).toContain("Other Stream");
    expect(markup).toContain(`src="${heroArt.hero}"`);
    expect(markup).not.toContain('href="/newsletters/other"'); // no other active card left
    expect(markup).not.toContain("Active streams"); // section hidden without a second active stream
  });

  it("renders the first-stream empty state without any streams", () => {
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      newsletters: [],
      lessons: [],
    });

    const markup = renderToStaticMarkup(<StreamsPage />);

    expect(markup).toContain("Turn a question into a learning practice.");
    expect(markup).toContain("Create your first stream");
  });

  it("preserves the loading and error states", () => {
    useWorkspaceMock.mockReturnValue({ ...defaultWorkspace, loading: true });
    const loadingMarkup = renderToStaticMarkup(<StreamsPage />);
    expect(loadingMarkup).toContain("Preparing your learning home");

    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      loading: false,
      error: "The workspace is unreachable.",
    });
    const errorMarkup = renderToStaticMarkup(<StreamsPage />);
    expect(errorMarkup).toContain("We couldn’t load this part of your learning home.");
    expect(errorMarkup).toContain("The workspace is unreachable.");
    expect(errorMarkup).toContain("Try again");
  });
});
