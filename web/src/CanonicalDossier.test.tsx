import { describe, expect, it } from "vitest";
import { canonicalDossierOrder, canonicalSources } from "./CanonicalDossier";

describe("canonical public Dossier", () => {
  it("keeps a visible, unique, secure source portfolio", () => {
    expect(canonicalSources).toHaveLength(5);
    expect(new Set(canonicalSources.map((source) => source.id)).size).toBe(5);
    expect(canonicalSources.every((source) => source.url.startsWith("https://"))).toBe(true);
    expect(canonicalSources.every((source) => source.use.length > 30)).toBe(true);
  });

  it("orders the complete learning arc exactly", () => {
    expect(canonicalDossierOrder).toEqual([
      "learning-objective",
      "two-minute-recall",
      "why-this-matters",
      "mental-model",
      "how-it-works",
      "worked-example",
      "common-misconception",
      "practical-experiment",
      "takeaway",
      "retrieval-practice",
      "application-challenge",
      "answer-key",
      "ai-exploration",
      "sources",
    ]);
    // The arc is order-sensitive: retrieval practice must precede the answer
    // key, and the opt-in AI Exploration must sit between the lesson and the
    // source index, never the other way around.
    const order = canonicalDossierOrder.map((id) => id);
    expect(order.indexOf("retrieval-practice")).toBeLessThan(order.indexOf("answer-key"));
    expect(order.indexOf("ai-exploration")).toBeGreaterThan(order.indexOf("answer-key"));
    expect(order.indexOf("ai-exploration")).toBeLessThan(order.indexOf("sources"));
  });
});