import { describe, expect, it } from "vitest";
import { performancePage } from "./performance";

describe("performancePage", () => {
  it("removes resource identifiers from metric dimensions", () => {
    expect(performancePage("/issues/issue-123")).toBe("/issues/:id");
    expect(performancePage("/newsletters/stream-123")).toBe("/newsletters/:id");
    expect(performancePage("/library")).toBe("/library");
  });
});
