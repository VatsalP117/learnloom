import { describe, expect, it } from "vitest";
import starterReviewManifest from "../../docs/release-evidence/starter-path-review-v3.json";
import {
  STREAM_TEMPLATE_CATALOG_VERSION,
  streamTemplates,
} from "./streamTemplates";

describe("stream template catalog", () => {
  const launchTemplateIds = [
    "ai-evaluation-governance",
    "production-rag-quality",
    "reliable-ai-agents",
    "llm-application-security",
    "context-engineering",
    "production-ai-observability",
  ];

  it("uses stable unique IDs and bounded source sets", () => {
    expect(STREAM_TEMPLATE_CATALOG_VERSION).toBeGreaterThan(0);
    expect(streamTemplates).toHaveLength(6);
    expect(new Set(streamTemplates.map((template) => template.id)).size)
      .toBe(streamTemplates.length);
    for (const template of streamTemplates) {
      expect(template.version).toBeGreaterThan(0);
      expect(template.outcome.length).toBeGreaterThan(20);
      expect(template.sources.length).toBeGreaterThan(0);
      expect(template.sources.length).toBeLessThanOrEqual(6);
      expect(template.sources.every((source) => source.url.startsWith("https://")))
        .toBe(true);
      expect(["founder_review_pending", "human_reviewed"])
        .toContain(template.editorialStatus);
    }
  });

  it("keeps every starter path inside the selected AI/software launch wedge", () => {
    expect(streamTemplates.map((template) => template.id)).toEqual(launchTemplateIds);
    expect(streamTemplates.every((template) =>
      /AI|LLM|model|retrieval|agent/i.test(`${template.topic} ${template.outcome}`),
    )).toBe(true);
  });

  it("keeps the fail-closed human review packet aligned with catalog identity and sources", () => {
    expect(starterReviewManifest.catalogVersion).toBe(STREAM_TEMPLATE_CATALOG_VERSION);
    expect(starterReviewManifest.reviews.map((review) => review.templateId))
      .toEqual(streamTemplates.map((template) => template.id));
    for (const template of streamTemplates) {
      const review = starterReviewManifest.reviews.find((candidate) =>
        candidate.templateId === template.id,
      );
      expect(review).toMatchObject({
        templateVersion: template.version,
        name: template.name,
        sourceUrls: template.sources.map((source) => source.url),
      });
    }
  });
});
