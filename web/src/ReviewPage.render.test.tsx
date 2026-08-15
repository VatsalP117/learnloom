import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { useWorkspaceMock, defaultWorkspace } = vi.hoisted(() => {
  const createdAt = (daysAgo: number) => {
    const value = new Date();
    value.setDate(value.getDate() - daysAgo);
    value.setHours(8, 0, 0, 0);
    return value.toISOString();
  };

  const newsletter = {
    id: "hero",
    name: "Hero Stream",
    active: true,
    lessonMinutes: 12,
  };
  const lesson = {
    id: "issue-1",
    title: "Hero lesson title",
    status: "generated",
    newsletterId: "hero",
    newsletter,
    createdAt: createdAt(0),
  };
  const reviews = [
    {
      id: "review-1",
      issueId: "issue-1",
      objective: "Recall the mechanism.",
      prompt: "Name the mechanism.",
      answerRubric: "Accurate mechanism.",
      correctiveExplanation: "Reopen the lesson.",
      stage: 0,
      dueAt: createdAt(0),
    },
    {
      id: "review-2",
      issueId: "issue-1",
      objective: "Recall the evidence.",
      prompt: "What evidence supports it?",
      answerRubric: "Accurate evidence.",
      correctiveExplanation: "Reopen the lesson.",
      stage: 0,
      dueAt: createdAt(0),
    },
  ];

  const defaultWorkspace = {
    snapshot: undefined,
    newsletters: [newsletter],
    lessons: [lesson],
    reviews,
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
  invalidateWorkspaceCache: vi.fn(),
}));

import ReviewPage from "./ReviewPage";

describe("redesigned Review page render", () => {
  beforeEach(() => {
    useWorkspaceMock.mockReturnValue(defaultWorkspace);
  });

  it("renders the redesigned shell and heading scaffold", () => {
    const markup = renderToStaticMarkup(<ReviewPage />);

    expect(markup).toContain('class="atelier-app atelier-today"');
    expect(markup).toContain("Strengthen the thread");
    expect(markup).toContain("<h1>Spaced retrieval</h1>");
    expect(markup).toContain(
      "Recall an idea before looking back. Honest effort matters more than a perfect answer.",
    );
  });

  it("keeps the active-recall card as the focus with a clear reveal action", () => {
    const markup = renderToStaticMarkup(<ReviewPage />);

    expect(markup).toContain("Active recall");
    expect(markup).toContain("2 prompts due");
    expect(markup).toContain("Hero Stream");
    expect(markup).toContain("Name the mechanism.");
    expect(markup).toContain(
      "Explain it aloud or in your own notes. Then reveal the lesson context and rate your recall.",
    );
    expect(markup).toContain("Reveal lesson context");
    // Context, rubric, and assessment stay behind the reveal by default.
    expect(markup).not.toContain("Learning objective");
    expect(markup).not.toContain("Useful answer");
  });

  it("renders the quiet side summary with the retained activity bars", () => {
    const markup = renderToStaticMarkup(<ReviewPage />);

    expect(markup).toContain("Learning rhythm");
    expect(markup).toContain('aria-label="Recent review activity"');
    expect(markup).toContain("Return when a lesson feels slightly difficult to recall.");
    expect(markup).toContain("This session");
    expect(markup).toContain("2 due now");
    expect(markup).toContain("Each answer schedules its own next review.");
    expect((markup.match(/<i style="height:/g) ?? []).length).toBe(7);
  });

  it("keeps the clear-queue state prominent with the library return link", () => {
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      reviews: [],
    });

    const markup = renderToStaticMarkup(<ReviewPage />);

    expect(markup).toContain("Your review queue is clear.");
    expect(markup).toContain("New prompts will appear as you complete more lessons.");
    expect(markup).toContain("Return to your library");
    expect(markup).toContain('href="/library"');
    expect(markup).not.toContain("Reveal lesson context");
  });

  it("preserves the loading and error states", () => {
    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      loading: true,
      reviews: [],
    });
    const loadingMarkup = renderToStaticMarkup(<ReviewPage />);
    expect(loadingMarkup).toContain("Preparing your review queue…");

    useWorkspaceMock.mockReturnValue({
      ...defaultWorkspace,
      loading: false,
      error: "The review queue is unreachable.",
      reviews: [],
    });
    const errorMarkup = renderToStaticMarkup(<ReviewPage />);
    expect(errorMarkup).toContain("We couldn’t load this part of your learning home.");
    expect(errorMarkup).toContain("The review queue is unreachable.");
    expect(errorMarkup).toContain("Try again");
  });
});
