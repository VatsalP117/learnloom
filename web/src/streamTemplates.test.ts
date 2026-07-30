import { describe, expect, it } from "vitest";
import {
  STREAM_TEMPLATE_CATALOG_VERSION,
  streamTemplates,
} from "./streamTemplates";

describe("stream template catalog", () => {
  it("uses stable unique IDs and bounded source sets", () => {
    expect(STREAM_TEMPLATE_CATALOG_VERSION).toBeGreaterThan(0);
    expect(new Set(streamTemplates.map((template) => template.id)).size)
      .toBe(streamTemplates.length);
    for (const template of streamTemplates) {
      expect(template.version).toBeGreaterThan(0);
      expect(template.outcome.length).toBeGreaterThan(20);
      expect(template.sources.length).toBeGreaterThan(0);
      expect(template.sources.length).toBeLessThanOrEqual(6);
      expect(template.sources.every((source) => source.url.startsWith("https://")))
        .toBe(true);
    }
  });
});
