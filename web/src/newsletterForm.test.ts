import { describe, expect, it } from "vitest";
import {
  buildNewsletterPayload,
  canSubmitNewsletter,
  draftToFormValues,
} from "./newsletterForm";
import type { OnboardingDraftPayload } from "./types";

const defaults = {
  name: "",
  topic: "LLM inference",
  learnerLevel: "intermediate",
  learnerGoal: "",
  lessonMinutes: 20,
  scheduleTime: "08:00",
  timeZone: "Asia/Kolkata",
  active: true,
  emailEnabled: false,
  aiExplorationEnabled: false,
  siteVisible: false,
  sourceReviewMode: "auto" as const,
  sources: [],
  templateId: undefined,
  templateVersion: undefined,
};

describe("Newsletter source mode payloads", () => {
  it("supports topic-only discovered creation and omits empty source rows", () => {
    const values = {
      ...defaults,
      sourceMode: "discovered",
      sourceReviewMode: "auto" as const,
      onboardingDraftId: "a5aa94e1-f83b-4d24-bf40-76048a3fc1f0",
      onboardingDraftRevision: 4,
      sources: [{ name: "", url: "", limit: 8 }],
    };
    expect(canSubmitNewsletter(values)).toBe(true);
    expect(buildNewsletterPayload(values)).toMatchObject({
      topic: "LLM inference",
      sourceMode: "discovered",
      onboardingDraftId: "a5aa94e1-f83b-4d24-bf40-76048a3fc1f0",
      onboardingDraftRevision: 4,
      sources: [],
      siteVisible: false,
    });
  });

  it.each(["provided", "hybrid"])("requires a source in %s mode", (sourceMode) => {
    expect(canSubmitNewsletter({ ...defaults, sourceMode })).toBe(false);
    const values = {
      ...defaults,
      sourceMode,
      sources: [
        { name: "", url: "https://www.example.com/guide", limit: "8" },
        { name: "", url: "  ", limit: "8" },
      ],
    };
    expect(canSubmitNewsletter(values)).toBe(true);
    expect(buildNewsletterPayload(values).sources).toEqual([{
      name: "example.com",
      url: "https://www.example.com/guide",
      limit: 8,
    }]);
  });

  it("preserves versioned template attribution", () => {
    const payload = buildNewsletterPayload({
      ...defaults,
      sourceMode: "provided",
      templateId: "ai-systems-evidence",
      templateVersion: 2,
      sources: [{ name: "Evidence", url: "https://example.com/feed", limit: 8 }],
    });
    expect(payload).toMatchObject({
      templateId: "ai-systems-evidence",
      templateVersion: 2,
    });
  });

  it("can pause after source discovery for learner approval", () => {
    const payload = buildNewsletterPayload({
      ...defaults,
      sourceMode: "discovered",
      sourceReviewMode: "review",
    });
    expect(payload.sourceReviewMode).toBe("review");
  });

  it("never forwards display-only attribution on final creation", () => {
    const values = {
      ...defaults,
      sourceMode: "hybrid",
      sourceReviewMode: "review" as const,
      sources: [{ name: "Evidence", url: "https://example.com/feed", limit: 8 }],
      onboardingDraftId: "a5aa94e1-f83b-4d24-bf40-76048a3fc1f0",
      onboardingDraftRevision: 1,
      attribution: {
        dossierPublicId: "dossier-30000000-0000-0000-0000-000000000000",
        title: "A public Dossier",
        canonicalUrl: "https://maya.learnloom.blog/d/dossier-30000000-0000-0000-0000-000000000000/a",
        ownerName: "Maya",
      },
    };
    const payload = buildNewsletterPayload(values);
    expect(payload).not.toHaveProperty("attribution");
    expect(payload).not.toHaveProperty("dossierPublicId");
    expect(payload).not.toHaveProperty("canonicalUrl");
    expect(payload).not.toHaveProperty("ownerName");
  });
});

describe("draftToFormValues", () => {
  const seeded: OnboardingDraftPayload = {
    name: "Systems",
    topic: "How systems fail",
    learnerLevel: "intermediate",
    learnerGoal: "Explain failure modes",
    lessonMinutes: 12,
    scheduleTime: "08:00",
    sourceMode: "hybrid",
    sourceReviewMode: "review",
    sources: [
      { name: "Evidence one", url: "https://example.org/one", limit: 8 },
      { name: "Evidence two", url: "https://example.org/two" },
    ],
    attribution: {
      dossierPublicId: "dossier-30000000-0000-0000-0000-000000000000",
      title: "A public Dossier",
      canonicalUrl: "https://maya.learnloom.blog/d/dossier-30000000-0000-0000-0000-000000000000/a",
      ownerName: "Maya",
    },
  };

  it("maps every seeded field onto editable form values", () => {
    const values = draftToFormValues(seeded, true);
    expect(values).toMatchObject({
      name: "Systems",
      topic: "How systems fail",
      learnerLevel: "intermediate",
      learnerGoal: "Explain failure modes",
      lessonMinutes: 12,
      scheduleTime: "08:00",
      sourceMode: "hybrid",
      reviewBeforeLesson: true,
      showSpecificSources: true,
      templateId: undefined,
      templateVersion: undefined,
      attribution: seeded.attribution,
    });
    expect(values.sources).toEqual([
      { name: "Evidence one", url: "https://example.org/one", limit: 8 },
      { name: "Evidence two", url: "https://example.org/two", limit: 8 },
    ]);
  });

  it("keeps source-intent fields editable in hybrid mode without discovery", () => {
    const values = draftToFormValues(seeded, false);
    expect(values.sourceMode).toBe("provided");
    expect(values.showSpecificSources).toBe(true);
    expect(values.sources).toEqual([
      { name: "Evidence one", url: "https://example.org/one", limit: 8 },
      { name: "Evidence two", url: "https://example.org/two", limit: 8 },
    ]);
  });

  it("defaults missing fields and supplies the browser timezone later", () => {
    const values = draftToFormValues({ attribution: seeded.attribution }, true);
    expect(values).toMatchObject({
      name: "",
      topic: "",
      learnerLevel: "intermediate",
      learnerGoal: "",
      lessonMinutes: 12,
      scheduleTime: "08:00",
      sourceMode: "discovered",
      reviewBeforeLesson: false,
      showSpecificSources: false,
      sources: [],
    });
    expect(values.timeZone).not.toBe("");
  });
});
