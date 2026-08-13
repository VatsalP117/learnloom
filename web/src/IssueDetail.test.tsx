import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { LessonFeedbackPanel, readerAudienceLabel } from "./IssueDetail";

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
