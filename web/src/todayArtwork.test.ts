import { describe, expect, it } from "vitest";
import {
  artworkForStream,
  artworkIndexForStream,
  todayArtworks,
  todayFallbackArtwork,
} from "./todayArtwork";

describe("Today artwork assignment", () => {
  it("assigns the same artwork to a stream across calls", () => {
    const first = artworkForStream("ai-evaluation");
    const second = artworkForStream("ai-evaluation");

    expect(second).toBe(first);
    expect(second.id).toBe(first.id);
  });

  it("distributes known streams across the artwork set", () => {
    const streamIds = [
      "ai-evaluation",
      "intelligence",
      "agents",
      "distributed-systems",
      "ml-ops",
      "databases",
      "networks",
      "frontend-design",
      "backend-engineering",
      "llm-inference",
    ];
    const assigned = new Set(streamIds.map((id) => artworkForStream(id).id));

    expect(assigned.size).toBeGreaterThan(1);
  });

  it("maps a stream to a deterministic bucket inside the artwork array", () => {
    const index = artworkIndexForStream("ai-evaluation");

    expect(index).toBeGreaterThanOrEqual(0);
    expect(index).toBeLessThan(todayArtworks.length);
    expect(artworkIndexForStream("ai-evaluation")).toBe(index);
  });

  it("falls back to the neutral artwork when no stream ID exists", () => {
    expect(artworkForStream(undefined)).toBe(todayFallbackArtwork);
    expect(artworkForStream(null)).toBe(todayFallbackArtwork);
    expect(artworkForStream("")).toBe(todayFallbackArtwork);
  });

  it("exposes card-sized and hero-sized responsive sources", () => {
    for (const artwork of todayArtworks) {
      expect(artwork.cardSrcSet).toContain("680w");
      expect(artwork.heroSrcSet).toContain("1440w");
      expect(artwork.card).not.toBe(artwork.hero);
    }
  });
});
