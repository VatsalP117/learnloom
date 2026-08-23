import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { LessonFeedbackPanel, ReaderReturnLink, readerAudienceLabel } from "./IssueDetail";

describe("LessonFeedbackPanel", () => {
  it("hydrates durable learner signals and keeps them optional", () => {
    const markup = renderToStaticMarkup(
      <LessonFeedbackPanel
        issueId="issue-1"
        initialFeedback={{
          difficulty: "right",
          relevance: "very_relevant",
          recallConfidence: "medium",
        }}
      />,
    );

    expect(markup).toContain("Shape what comes next");
    expect(markup).toContain('aria-pressed="true">About right');
    expect(markup).toContain('aria-pressed="true">Very relevant');
    expect(markup).toContain('aria-pressed="true">Partial');
    expect(markup).toContain("These signals are private");
  });
});

describe("reader return control", () => {
  it("renders a real anchor for the contextual origin", () => {
    const markup = renderToStaticMarkup(
      <ReaderReturnLink href="/streams?tab=all" label="Streams" />,
    );

    expect(markup).toContain('href="/streams?tab=all"');
    expect(markup).toContain("Back to Streams");
  });

  it("renders the parent-stream fallback anchor", () => {
    const markup = renderToStaticMarkup(
      <ReaderReturnLink href="/newsletters/quantum-42" label="Quantum Field Theory" />,
    );

    expect(markup).toContain('href="/newsletters/quantum-42"');
    expect(markup).toContain("Back to Quantum Field Theory");
  });
});

describe("reader audience label", () => {
  const stream = { siteVisible: true };
  const site = { visibility: "public", searchIndexing: false };

  it("reports content state and effective publishing gates", () => {
    expect(readerAudienceLabel({ publicationState: "draft" }, stream, site)).toBe("Audience: draft, only you");
    expect(readerAudienceLabel({ publicationState: "published" }, stream, { ...site, visibility: "private" }))
      .toContain("site private");
    expect(readerAudienceLabel({ publicationState: "published" }, stream, site)).toBe("Audience: public by link");
  });
});
