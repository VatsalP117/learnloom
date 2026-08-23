import type {
  OnboardingAttribution,
  OnboardingDraftPayload,
} from "./types";

export interface NewsletterSourceInput {
  name: string;
  url: string;
  limit: number | string;
}

export interface NewsletterFormValues {
  name: string;
  topic: string;
  learnerLevel: string;
  learnerGoal: string;
  lessonMinutes: number;
  scheduleTime: string;
  timeZone: string;
  active: boolean;
  emailEnabled: boolean;
  aiExplorationEnabled: boolean;
  siteVisible: boolean;
  sourceMode: string;
  sourceReviewMode: "auto" | "review";
  sources: NewsletterSourceInput[];
  templateId?: string;
  templateVersion?: number;
  onboardingDraftId?: string;
  onboardingDraftRevision?: number;
  /** Display-only seeded provenance; never sent on final creation. */
  attribution?: OnboardingAttribution | null;
}

export interface DraftFormValues {
  name: string;
  topic: string;
  learnerLevel: string;
  learnerGoal: string;
  lessonMinutes: number;
  scheduleTime: string;
  timeZone: string;
  active: boolean;
  emailEnabled: boolean;
  aiExplorationEnabled: boolean;
  sourceMode: "discovered" | "provided" | "hybrid";
  reviewBeforeLesson: boolean;
  showSpecificSources: boolean;
  sources: NewsletterSourceInput[];
  templateId?: string;
  templateVersion?: number;
  attribution?: OnboardingAttribution | null;
}

/**
 * Maps a restored onboarding draft payload onto editable form values.
 * Seeded hybrid mode degrades to provided when source discovery is off so
 * seeded sources stay editable, and display-only attribution is carried
 * through for the banner and autosave persistence.
 */
export function draftToFormValues(
  payload: OnboardingDraftPayload,
  sourceDiscovery: boolean,
): DraftFormValues {
  const sourceMode = payload.sourceMode ?? (sourceDiscovery ? "discovered" : "provided");
  const restoredMode = sourceDiscovery || sourceMode !== "hybrid" ? sourceMode : "provided";
  const sources = payload.sources?.length
    ? payload.sources.map((source) => ({ ...source, limit: source.limit ?? 8 }))
    : restoredMode === "discovered"
      ? []
      : [{ name: "", url: "", limit: 8 }];
  return {
    name: payload.name ?? "",
    topic: payload.topic ?? "",
    learnerLevel: payload.learnerLevel ?? "intermediate",
    learnerGoal: payload.learnerGoal ?? "",
    lessonMinutes: payload.lessonMinutes ?? 12,
    scheduleTime: payload.scheduleTime ?? "08:00",
    timeZone: payload.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone,
    active: payload.active ?? true,
    emailEnabled: payload.emailEnabled ?? false,
    aiExplorationEnabled: payload.aiExplorationEnabled ?? false,
    sourceMode: restoredMode,
    reviewBeforeLesson: payload.sourceReviewMode === "review",
    showSpecificSources: restoredMode !== "discovered",
    sources,
    templateId: payload.templateId,
    templateVersion: payload.templateVersion,
    attribution: payload.attribution ?? null,
  };
}

export function usableSources(sources: NewsletterSourceInput[]) {
  return sources.filter((source) => source.url.trim() !== "");
}

export function canSubmitNewsletter({
  topic,
  sourceMode,
  sources,
}: Pick<NewsletterFormValues, "topic" | "sourceMode" | "sources">) {
  if (!topic.trim()) return false;
  if (sourceMode === "discovered") return true;
  return usableSources(sources).length > 0;
}

export function buildNewsletterPayload(values: NewsletterFormValues) {
  const sources = values.sourceMode === "discovered"
    ? []
    : usableSources(values.sources).map((source) => ({
        name: source.name.trim() || sourceLabel(source.url),
        url: source.url.trim(),
        limit: Number(source.limit),
      }));
  return {
    name: values.name.trim() || undefined,
    topic: values.topic.trim(),
    learnerLevel: values.learnerLevel,
    learnerGoal: values.learnerGoal.trim() || undefined,
    lessonMinutes: values.lessonMinutes,
    scheduleTime: values.scheduleTime,
    timeZone: values.timeZone,
    active: values.active,
    emailEnabled: values.emailEnabled,
    aiExplorationEnabled: values.aiExplorationEnabled,
    siteVisible: values.siteVisible,
    sourceMode: values.sourceMode,
    sourceReviewMode: values.sourceReviewMode,
    sources,
    templateId: values.templateId,
    templateVersion: values.templateVersion,
    onboardingDraftId: values.onboardingDraftId,
    onboardingDraftRevision: values.onboardingDraftRevision,
  };
}

function sourceLabel(rawURL: string) {
  try {
    return new URL(rawURL).hostname.replace(/^www\./, "");
  } catch {
    return rawURL;
  }
}
