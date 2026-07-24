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
  sources: NewsletterSourceInput[];
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
    sources,
  };
}

function sourceLabel(rawURL: string) {
  try {
    return new URL(rawURL).hostname.replace(/^www\./, "");
  } catch {
    return rawURL;
  }
}
