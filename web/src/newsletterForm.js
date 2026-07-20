export function usableSources(sources) {
  return sources.filter((source) => source.url.trim() !== "");
}

export function canSubmitNewsletter({ topic, sourceMode, sources }) {
  if (!topic.trim()) return false;
  if (sourceMode === "discovered") return true;
  return usableSources(sources).length > 0;
}

export function buildNewsletterPayload(values) {
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

function sourceLabel(rawURL) {
  try {
    return new URL(rawURL).hostname.replace(/^www\./, "");
  } catch {
    return rawURL;
  }
}
