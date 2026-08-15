import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { useLibraryMock, defaultLibrary } = vi.hoisted(() => {
  const createdAt = (daysAgo: number) => {
    const value = new Date();
    value.setDate(value.getDate() - daysAgo);
    value.setHours(8, 0, 0, 0);
    return value.toISOString();
  };
  const lesson = (
    id: string,
    title: string,
    stream: string,
    minutes: number,
    daysAgo: number,
  ) => ({
    id,
    title,
    createdAt: createdAt(daysAgo),
    newsletter: { name: stream, lessonMinutes: minutes },
  });

  const defaultLibrary = {
    lessons: [
      lesson("lesson-1", "First lesson title", "Hero Stream", 12, 0),
      lesson("lesson-2", "Second lesson title", "Other Stream", 10, 2),
    ],
    error: "",
    loading: false,
    loadingMore: false,
    hasMore: true,
    loadMore: () => Promise.resolve(),
    reload: () => Promise.resolve(),
  };

  return { useLibraryMock: vi.fn(() => defaultLibrary), defaultLibrary };
});

vi.mock("./useLibrary", () => ({
  useLibrary: useLibraryMock,
}));

import LibraryPage from "./LibraryPage";

describe("redesigned Library page render", () => {
  beforeEach(() => {
    useLibraryMock.mockReturnValue(defaultLibrary);
  });

  it("renders the redesigned shell, header, search, and four filters", () => {
    const markup = renderToStaticMarkup(<LibraryPage />);

    expect(markup).toContain('class="atelier-app atelier-today"');
    expect(markup).toContain("Your lasting archive");
    expect(markup).toContain("<h1>Library</h1>");
    expect(markup).toContain(
      "Find a lesson by title, concept, source, retrieval question, or stream.",
    );
    expect(markup).toContain('role="group"');
    expect(markup).toContain('aria-label="Filter lessons"');
    expect(markup).toContain("All lessons");
    expect(markup).toContain("Unread");
    expect(markup).toContain("In progress");
    expect(markup).toContain("Completed");
    expect(markup).toContain('type="search"');
    expect(markup).toContain('placeholder="Search concepts, sources, or questions"');
    expect(markup).toContain(
      "Search lessons, concepts, sources, and retrieval prompts",
    );
  });

  it("renders text-first archive cards with stream chip, title, and footer", () => {
    const markup = renderToStaticMarkup(<LibraryPage />);

    expect(markup).toContain('href="/issues/lesson-1"');
    expect(markup).toContain("First lesson title");
    expect(markup).toContain("Hero Stream");
    expect(markup).toContain("12 min");
    expect(markup).toContain("Generated");
    expect(markup).toContain('href="/issues/lesson-2"');
    expect(markup).toContain("Second lesson title");
    expect(markup).toContain("Other Stream");
    expect(markup).toContain("10 min");
  });

  it("keeps the pagination control and its exact labels", () => {
    const markup = renderToStaticMarkup(<LibraryPage />);

    expect(markup).toContain("Load older lessons");
    expect(markup).not.toContain("Loading older lessons");
  });

  it("renders the no-results empty state", () => {
    useLibraryMock.mockReturnValue({
      ...defaultLibrary,
      lessons: [],
      hasMore: false,
    });

    const markup = renderToStaticMarkup(<LibraryPage />);

    expect(markup).toContain("No lessons found.");
    expect(markup).toContain("Try another term or choose a different reading state.");
    expect(markup).not.toContain("Load older lessons");
  });

  it("preserves the loading state", () => {
    useLibraryMock.mockReturnValue({
      ...defaultLibrary,
      lessons: [],
      loading: true,
      hasMore: false,
    });

    const markup = renderToStaticMarkup(<LibraryPage />);

    expect(markup).toContain("Opening your library…");
    expect(markup).not.toContain("No lessons found.");
  });

  it("preserves the error state with a retry action", () => {
    useLibraryMock.mockReturnValue({
      ...defaultLibrary,
      lessons: [],
      error: "The library is unreachable.",
      hasMore: false,
    });

    const markup = renderToStaticMarkup(<LibraryPage />);

    expect(markup).toContain("We couldn’t load this part of your learning home.");
    expect(markup).toContain("The library is unreachable.");
    expect(markup).toContain("Try again");
  });
});
