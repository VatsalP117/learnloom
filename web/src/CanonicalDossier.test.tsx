import { describe, expect, it } from "vitest";
import { canonicalSources } from "./CanonicalDossier";

describe("canonical public Dossier", () => {
  it("keeps a visible, unique, secure source portfolio", () => {
    expect(canonicalSources).toHaveLength(5);
    expect(new Set(canonicalSources.map((source) => source.id)).size).toBe(5);
    expect(canonicalSources.every((source) => source.url.startsWith("https://"))).toBe(true);
    expect(canonicalSources.every((source) => source.use.length > 30)).toBe(true);
  });
});
