import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./useWorkspace", () => ({
  useWorkspace: () => ({
    newsletters: [],
    lessons: [],
    reviews: [],
    loading: false,
    loadingMore: false,
    error: "",
    hasMore: false,
    loadMore: vi.fn(),
    reload: vi.fn(),
  }),
}));

import App from "./App";
import AppGraph from "./AppGraph";

describe("primary app routes", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "window", {
      configurable: true,
      value: {
        location: {
          origin: "https://app.learnloom.blog",
          pathname: "/",
          search: "",
          hash: "",
        },
      },
    });
  });

  it.each([
    ["/", "Your learning practice"],
    ["/streams", "Learning streams"],
    ["/library", "Your lasting archive"],
    ["/review", "Spaced retrieval"],
    ["/publishing", "Share deliberately"],
  ])("renders %s without the route-level loader", (pathname, expected) => {
    window.location.pathname = pathname;

    const markup = renderToStaticMarkup(<AppGraph />);

    expect(markup).toContain(expected);
    expect(markup).not.toContain("Preparing this space");
  });

  it("routes /settings through the lazy Settings chunk fallback", () => {
    window.location.pathname = "/settings";

    const markup = renderToStaticMarkup(<AppGraph />);

    // Server rendering cannot resolve the lazy chunk, so the Suspense
    // fallback stands in; a real browser resolves it and shows the page.
    expect(markup).toContain("Preparing this space");
    expect(markup).not.toContain("Prompts & recaps");
  });

  it("renders the App wrapper fallback while the app graph chunk loads", () => {
    const markup = renderToStaticMarkup(<App />);

    // The wrapper carries only the lazy import boundary, so server rendering
    // shows the Suspense fallback instead of any authenticated page.
    expect(markup).toContain("Preparing your workspace");
    expect(markup).not.toContain("Your learning practice");
  });
});
