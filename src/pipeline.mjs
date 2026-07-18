const STAGE_INSTRUCTIONS = Object.freeze({
  researcher: [
    "Select one coherent, high-value theme from the supplied source bundle.",
    "Write a compact research brief containing the important claims, mechanisms, and implications.",
    "Cite source identifiers like [S1]. Distinguish reported facts from your inference.",
    "Prefer depth over a list of unrelated news.",
  ].join(" "),
  skeptic: [
    "Audit the research brief against the supplied sources.",
    "Identify weak evidence, missing context, alternative explanations, and seductive but unsupported claims.",
    "Preserve source identifiers. Do not invent facts.",
  ].join(" "),
  teacher: [
    "Turn the research brief and skeptical review into one clear lesson for the described learner.",
    "Include: the core idea, a step-by-step explanation, one concrete analogy, why it matters,",
    "a practical experiment, and what remains uncertain. Aim for the requested lesson duration.",
    "Use source identifiers for factual claims.",
  ].join(" "),
  examiner: [
    "Create retrieval practice from the lesson, not trivia.",
    "Include three short-answer questions, one application challenge, and a collapsed answer key",
    "using HTML details/summary tags. Test understanding of mechanisms and tradeoffs.",
  ].join(" "),
});

export async function buildDossier({ config, items, history, provider, now = new Date(), onStage }) {
  if (!Array.isArray(items) || items.length === 0) {
    throw new Error("At least one source item is required.");
  }

  const sourceBundle = formatSourceBundle(items, config.limits.maxItemCharacters);
  const learnerContext = formatLearnerContext(config, history);

  const research = await runStage(
    provider,
    "researcher",
    STAGE_INSTRUCTIONS.researcher,
    `${learnerContext}\n\n# Sources\n\n${sourceBundle}`,
    onStage,
  );
  const critique = await runStage(
    provider,
    "skeptic",
    STAGE_INSTRUCTIONS.skeptic,
    truncate(
      `${learnerContext}\n\n# Sources\n\n${sourceBundle}\n\n# Research brief\n\n${research}`,
      config.limits.maxIntermediateCharacters,
    ),
    onStage,
  );
  const lesson = await runStage(
    provider,
    "teacher",
    STAGE_INSTRUCTIONS.teacher,
    truncate(
      `${learnerContext}\n\n# Research brief\n\n${research}\n\n# Skeptical review\n\n${critique}`,
      config.limits.maxIntermediateCharacters,
    ),
    onStage,
  );
  const practice = await runStage(
    provider,
    "examiner",
    STAGE_INSTRUCTIONS.examiner,
    truncate(`${learnerContext}\n\n# Lesson\n\n${lesson}`, config.limits.maxIntermediateCharacters),
    onStage,
  );

  const date = formatDate(now, config.timeZone);
  const markdown = [
    `# Learning Dossier — ${date}`,
    "",
    `> Generated from ${items.length} source items through researcher, skeptic, teacher, and examiner passes.`,
    "",
    lesson,
    "",
    critique,
    "",
    practice,
    "",
    "## Source Index",
    "",
    ...items.map(
      (item, index) =>
        `${index + 1}. **[S${index + 1}] ${escapeMarkdown(item.title)}** — ${
          item.source
        }  \n   ${item.url}`,
    ),
    "",
    "---",
    "",
    `Generated at ${now.toISOString()} · Model output can be wrong; verify important claims at the linked sources.`,
    "",
  ].join("\n");

  return {
    date,
    markdown,
    stages: { research, critique, lesson, practice },
    historyEntry: {
      date,
      generatedAt: now.toISOString(),
      sourceTitles: items.slice(0, 8).map((item) => item.title),
      lessonSummary: truncate(stripMarkdown(lesson), 800),
      recallQuestions: extractQuestions(practice),
    },
  };
}

async function runStage(provider, stage, instruction, input, onStage) {
  onStage?.(stage);
  const output = await provider.complete({ stage, instruction, input });
  if (typeof output !== "string" || output.trim() === "") {
    throw new Error(`The ${stage} stage returned empty output.`);
  }
  return output.trim();
}

export function formatSourceBundle(items, maxItemCharacters) {
  return items
    .map((item, index) => {
      const summary = truncate(item.summary || "No summary supplied by feed.", maxItemCharacters);
      return [
        `## [S${index + 1}] ${item.title}`,
        `Source: ${item.source}`,
        `Published: ${item.publishedAt ?? "unknown"}`,
        `URL: ${item.url}`,
        `Feed summary: ${summary}`,
      ].join("\n");
    })
    .join("\n\n");
}

function formatLearnerContext(config, history) {
  const prior = history.length
    ? history
        .slice(-config.limits.historyEntries)
        .map((entry) => `- ${entry.date}: ${entry.lessonSummary}`)
        .join("\n")
    : "- No previous lessons yet.";
  return [
    "# Learner",
    `Interests: ${config.interests.join(", ")}`,
    `Level: ${config.learner.level}`,
    `Goal: ${config.learner.goal}`,
    `Available time: ${config.learner.lessonMinutes} minutes`,
    "",
    "# Previous lessons",
    prior,
    "",
    "Avoid merely repeating a previous lesson. Build on it or choose a distinct theme.",
  ].join("\n");
}

function extractQuestions(markdown) {
  return markdown
    .split("\n")
    .map((line) => line.match(/^\s*\d+\.\s+(.+\?)\s*$/)?.[1])
    .filter(Boolean)
    .slice(0, 5);
}

function stripMarkdown(value) {
  return value
    .replace(/<[^>]+>/g, " ")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/^[#>*+\-\d.\s]+/gm, "")
    .replace(/[*_`~]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

function truncate(value, maximum) {
  if (value.length <= maximum) return value;
  return `${value.slice(0, maximum - 16).trimEnd()}\n[truncated]`;
}

function escapeMarkdown(value) {
  return value.replace(/([\\[\]*_])/g, "\\$1");
}

function formatDate(date, timeZone) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

