import { describe, expect, it } from "vitest";
import { libraryQueryPath } from "./useLibrary";

describe("Library query", () => {
  it("encodes search, filter, and pagination in one server query", () => {
    expect(libraryQueryPath("  systems & cities  ", "in-progress", "next/page"))
      .toBe(
        "/api/library?limit=24&filter=in-progress&q=systems+%26+cities&cursor=next%2Fpage",
      );
  });

  it("omits empty optional parameters", () => {
    expect(libraryQueryPath("   ", "all"))
      .toBe("/api/library?limit=24&filter=all");
  });
});
