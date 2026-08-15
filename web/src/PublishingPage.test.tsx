import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

vi.mock("./useWorkspace", () => ({
  useWorkspace: () => ({
    newsletters: [],
    loading: false,
  }),
}));

import PublishingPage from "./PublishingPage";
import type { Site } from "./types";

function renderPublishing(site: Site) {
  return renderToStaticMarkup(
    <PublishingPage site={site} onSiteUpdate={vi.fn()} />,
  );
}

describe("PublishingPage search discovery", () => {
  it("keeps search indexing separate from public visibility", () => {
    const markup = renderPublishing({
      username: "maya",
      displayName: "Maya’s Learning Garden",
      visibility: "public",
      searchIndexing: false,
    });

    expect(markup).toContain("Search discovery");
    expect(markup).toContain("Allow indexing");
    expect(markup).toContain("Public links work");
  });

  it("does not allow indexing while the site is private", () => {
    const markup = renderPublishing({
      username: "maya",
      displayName: "Maya’s Learning Garden",
      visibility: "private",
      searchIndexing: false,
    });

    expect(markup).toContain('<button type="button" disabled="">Allow indexing</button>');
  });

  it("shows when the owner has opted into search indexing", () => {
    const markup = renderPublishing({
      username: "maya",
      displayName: "Maya’s Learning Garden",
      visibility: "public",
      searchIndexing: true,
    });

    expect(markup).toContain("Disable indexing");
    expect(markup).toContain("Eligible published pages may appear");
  });
});

describe("PublishingPage redesigned shell", () => {
  it("opts into the redesigned dashboard shell and keeps the privacy ladder", () => {
    const markup = renderPublishing({
      username: "maya",
      displayName: "Maya’s Learning Garden",
      visibility: "private",
      searchIndexing: false,
    });

    expect(markup).toContain('class="atelier-app atelier-today"');
    expect(markup).toContain("publishing-page");
    expect(markup).toContain("Public identity");
    expect(markup).toContain("Visibility ladder");
    expect(markup).toContain("Site private");
    expect(markup).toContain("Search discovery");
    expect(markup).toContain("Publish site");
    expect(markup).toContain("Allow indexing");
  });

  it("keeps the claim flow and its validation for unclaimed sites", () => {
    const markup = renderToStaticMarkup(
      <PublishingPage site={null} onSiteUpdate={vi.fn()} />,
    );

    expect(markup).toContain("Claim your learning home");
    expect(markup).toContain("site starts private");
    expect(markup).toContain('minLength="3"');
    expect(markup).toContain('maxLength="30"');
    expect(markup).toContain('pattern="[a-zA-Z][a-zA-Z0-9-]{1,28}[a-zA-Z0-9]"');
    expect(markup).toContain("Create private site");
  });
});
