const REQUIRED_LESSON_SECTIONS = [
  "Learning objective",
  "Two-minute recall",
  "Why this matters",
  "Mental model",
  "How it works",
  "Worked example",
  "Common misconception",
  "Practical experiment",
  "Takeaway",
];

export function parseStructuredStage(output, stage) {
  if (typeof output !== "string" || output.trim() === "") {
    throw new Error(`The ${stage} stage returned empty structured output.`);
  }
  let candidate = output.trim();
  const fence = candidate.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i);
  if (fence) candidate = fence[1];
  try {
    const value = JSON.parse(candidate);
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      throw new Error("root must be an object");
    }
    return value;
  } catch (error) {
    throw new Error(
      `The ${stage} stage returned invalid JSON: ${error.message}`,
    );
  }
}

export function validateCuration(value, itemCount) {
  const theme = requiredText(value.theme, "Curator theme", 500);
  const rationale = requiredText(
    value.rationale,
    "Curator rationale",
    1_000,
  );
  if (!Array.isArray(value.selectedSourceIds)) {
    throw new Error("Curator selectedSourceIds must be an array.");
  }
  const minimum = Math.min(3, itemCount);
  const maximum = Math.min(5, itemCount);
  const selectedSourceIds = [
    ...new Set(
      value.selectedSourceIds.map((sourceId) =>
        requiredText(sourceId, "Curator source ID", 10),
      ),
    ),
  ];
  if (
    selectedSourceIds.length < minimum ||
    selectedSourceIds.length > maximum
  ) {
    throw new Error(
      `Curator must select ${minimum} to ${maximum} Source Items.`,
    );
  }
  for (const sourceId of selectedSourceIds) {
    const match = /^S([1-9]\d*)$/.exec(sourceId);
    const index = match ? Number(match[1]) : 0;
    if (index < 1 || index > itemCount) {
      throw new Error(`Curator selected unknown Source Item ${sourceId}.`);
    }
  }
  return { theme, rationale, selectedSourceIds };
}

export function validateBlueprint(value) {
  const prerequisites = requiredTextArray(
    value.prerequisites,
    "Blueprint prerequisites",
    5,
  );
  return {
    learningObjective: requiredText(
      value.learningObjective,
      "Blueprint learning objective",
      600,
    ),
    prerequisites,
    centralMechanism: requiredText(
      value.centralMechanism,
      "Blueprint central mechanism",
      1_200,
    ),
    workedExample: requiredText(
      value.workedExample,
      "Blueprint worked example",
      1_200,
    ),
    misconception: requiredText(
      value.misconception,
      "Blueprint misconception",
      800,
    ),
    practicalExperiment: requiredText(
      value.practicalExperiment,
      "Blueprint practical experiment",
      1_200,
    ),
    continuityBridge: requiredText(
      value.continuityBridge,
      "Blueprint continuity bridge",
      1_200,
    ),
  };
}

export function validateEditorial(value, options) {
  const lesson = requiredText(value.lesson, "Editorial lesson", 60_000);
  const critique = requiredText(
    value.critique,
    "Editorial critique",
    30_000,
  );
  const practice = requiredText(value.practice, "Editorial practice", 30_000);
  let exploration = null;
  if (options.explorationEnabled) {
    exploration = requiredText(
      value.exploration,
      "Editorial AI Exploration",
      30_000,
    );
  } else if (value.exploration != null && String(value.exploration).trim()) {
    throw new Error(
      "Editorial output included AI Exploration when it was disabled.",
    );
  }
  const qualityNotes = Array.isArray(value.qualityNotes)
    ? value.qualityNotes
        .filter((note) => typeof note === "string")
        .map((note) => note.trim())
        .filter(Boolean)
        .slice(0, 10)
    : [];
  return { lesson, critique, practice, exploration, qualityNotes };
}

export function evaluateDossierContent(input) {
  const {
    lesson,
    critique,
    practice,
    exploration,
    sources,
    blueprint,
    historyCount = 0,
  } = input;
  const missingSections = REQUIRED_LESSON_SECTIONS.filter(
    (section) =>
      !new RegExp(`^#{1,4}\\s+${escapeRegExp(section)}\\s*$`, "im").test(
        lesson,
      ),
  );
  if (missingSections.length > 0) {
    throw new Error(
      `Editorial lesson is missing required sections: ${missingSections.join(
        ", ",
      )}.`,
    );
  }
  const knownSourceIds = new Set(
    sources.map((source, index) => source.sourceId ?? `S${index + 1}`),
  );
  const groundedText = `${lesson}\n${critique}\n${practice}`;
  const citedSourceIds = [
    ...new Set(
      [...groundedText.matchAll(/\[S(\d+)\]/g)].map(
        (match) => `S${match[1]}`,
      ),
    ),
  ];
  const unknownSourceIds = citedSourceIds.filter(
    (sourceId) => !knownSourceIds.has(sourceId),
  );
  if (unknownSourceIds.length > 0) {
    throw new Error(
      `Editorial output cites unknown Source Items: ${unknownSourceIds.join(
        ", ",
      )}.`,
    );
  }
  const requiredCitationCount = Math.min(2, sources.length);
  if (citedSourceIds.length < requiredCitationCount) {
    throw new Error(
      `Editorial output must cite at least ${requiredCitationCount} Source Items.`,
    );
  }
  const questions = practice
    .split("\n")
    .map((line) => line.match(/^\s*\d+\.\s+(.+\?)\s*$/)?.[1])
    .filter(Boolean);
  if (questions.length < 3) {
    throw new Error(
      "Editorial practice must contain at least three retrieval questions.",
    );
  }
  if (!/^#{1,4}\s+Application challenge\s*$/im.test(practice)) {
    throw new Error(
      "Editorial practice is missing an Application challenge section.",
    );
  }
  if (
    !/<details>/i.test(practice) ||
    !/<summary>\s*Answer key\s*<\/summary>/i.test(practice) ||
    !/<\/details>/i.test(practice)
  ) {
    throw new Error("Editorial practice is missing a collapsed answer key.");
  }
  if (exploration && /\[S\d+\]/.test(exploration)) {
    throw new Error("AI Exploration must not use source citation markers.");
  }

  const citationCoverage =
    knownSourceIds.size === 0 ? 0 : citedSourceIds.length / knownSourceIds.size;
  const enrichedCount = sources.filter(
    (source) => source.contentSource === "article",
  ).length;
  const checks = {
    requiredLessonSections: missingSections.length === 0,
    sourceGrounding: citedSourceIds.length >= requiredCitationCount,
    validCitationIdentifiers: unknownSourceIds.length === 0,
    retrievalPractice: questions.length >= 3,
    applicationChallenge: true,
    collapsedAnswerKey: true,
    continuity:
      historyCount === 0 || blueprint.continuityBridge.trim().length > 0,
    explorationBoundary: !exploration || !/\[S\d+\]/.test(exploration),
  };
  const score = Math.round(
    Math.min(25, citationCoverage * 25) +
      25 +
      20 +
      (historyCount === 0 || checks.continuity ? 10 : 0) +
      (checks.explorationBoundary ? 5 : 0) +
      (enrichedCount > 0 ? 15 : 8),
  );
  return {
    version: 1,
    score: Math.min(100, score),
    checks,
    metrics: {
      selectedSources: sources.length,
      enrichedSources: enrichedCount,
      citedSources: citedSourceIds.length,
      retrievalQuestions: questions.length,
    },
  };
}

export function requiredLessonSections() {
  return [...REQUIRED_LESSON_SECTIONS];
}

function requiredText(value, field, maximum) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} must be a non-empty string.`);
  }
  const normalized = value.trim();
  if (normalized.length > maximum) {
    throw new Error(`${field} must be at most ${maximum} characters.`);
  }
  return normalized;
}

function requiredTextArray(value, field, maximumItems) {
  if (!Array.isArray(value) || value.length === 0) {
    throw new Error(`${field} must be a non-empty array.`);
  }
  return value
    .slice(0, maximumItems)
    .map((item) => requiredText(item, field, 500));
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
