import { describe, expect, it } from "vitest";
import { streamCapabilitySummary, streamRhythmSummary } from "./StreamsPage";

describe("stream summaries", () => {
  const stream = { id: "stream-1", name: "Systems", topic: "systems" };

  it("describes capability evidence rather than lesson-count progress", () => {
    expect(streamCapabilitySummary({ ...stream, capabilityCount: 3, recalledCapabilityCount: 1 }))
      .toBe("1 recalled · 3 established");
    expect(streamCapabilitySummary(stream)).toBe("First milestone ahead");
  });

  it("shows effective adaptive rhythm", () => {
    expect(streamRhythmSummary({
      ...stream,
      scheduleTime: "09:00",
      rhythmMode: "daily",
      effectiveRhythmMode: "weekly_synthesis",
      rhythmThrottledAt: "2026-08-12T00:00:00Z",
    })).toBe("Slowed to weekly · 09:00");
  });
});
