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

    const markup = renderToStaticMarkup(<App />);

    expect(markup).toContain(expected);
    expect(markup).not.toContain("Preparing this space");
  });
});
