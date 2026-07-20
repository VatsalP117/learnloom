import { describe, expect, it } from "vitest";
import {
  buildNewsletterPayload,
  canSubmitNewsletter,
} from "./newsletterForm.js";

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
  sources: [],
};

describe("Newsletter source mode payloads", () => {
  it("supports topic-only discovered creation and omits empty source rows", () => {
    const values = {
      ...defaults,
      sourceMode: "discovered",
      sources: [{ name: "", url: "", limit: 8 }],
    };
    expect(canSubmitNewsletter(values)).toBe(true);
    expect(buildNewsletterPayload(values)).toMatchObject({
      topic: "LLM inference",
      sourceMode: "discovered",
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
});
