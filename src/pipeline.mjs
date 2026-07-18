import {
  evaluateDossierContent,
  parseStructuredStage,
  requiredLessonSections,
  validateBlueprint,
  validateCuration,
  validateEditorial,
} from "./content-quality.mjs";
import { renderDossierMarkdown } from "./render.mjs";
import { enrichSourceItems } from "./source-enrichment.mjs";

const STAGE_INSTRUCTIONS = Object.freeze({
  curator: [
    "Choose one coherent, high-value learning theme from the supplied Source Items.",
    "Select the three to five Source Item identifiers that best explain one mechanism; use fewer only when fewer exist.",
    "Prefer complementary evidence over several versions of the same announcement.",
    'Return strict JSON only: {"theme":"...","rationale":"...","selectedSourceIds":["S1","S2","S3"]}.',
  ].join(" "),
  blueprint: [
    "Design one lesson before prose is written. Fit the learner's level, goal, available time, and previous lessons.",
    "The objective must describe what the learner can explain or do afterward.",
    "Use previous learning to create a concrete continuity bridge rather than merely avoiding repetition.",
    'Return strict JSON only with string fields "learningObjective", "centralMechanism", "workedExample",',
    '"misconception", "practicalExperiment", "continuityBridge", plus a non-empty string array "prerequisites".',
  ].join(" "),
  researcher: [
    "Write a compact research brief that serves the supplied learning blueprint.",
    "Explain claims, mechanisms, boundary conditions, and implications using only the supplied sources.",
    "Cite Source Item identifiers like [S1]. Distinguish reported facts from inference.",
    "Call out disagreements or missing evidence. Prefer causal depth over news summary.",
  ].join(" "),
  skeptic: [
    "Audit the research brief against the enriched sources and learning blueprint.",
    "Identify weak evidence, missing context, alternative explanations, edge cases, and seductive unsupported claims.",
    "Preserve valid Source Item identifiers. Do not invent facts.",
    "Give the teacher exact constraints for a trustworthy lesson.",
  ].join(" "),
  teacher: [
    "Write the source-grounded core lesson only. Do not add speculative AI Exploration.",
    `Use these exact Markdown headings: ${requiredLessonSections()
      .map((heading) => `"## ${heading}"`)
      .join(", ")}.`,
    "The Two-minute recall must reconstruct an older idea when history exists, then connect it to today's objective.",
    "Explain the central mechanism step by step, include the planned worked example and misconception,",
    "and make the practical experiment executable within the learner's available time.",
    "Cite factual claims with supplied Source Item identifiers. End with a compact takeaway.",
  ].join(" "),
  examiner: [
    "Create retrieval practice for the source-grounded lesson only, never the optional AI Exploration.",
    'Use "## Retrieval practice" with at least three numbered short-answer questions ending in question marks.',
    'Then use "## Application challenge" for one realistic transfer task.',
    'Finish with <details>, <summary>Answer key</summary>, complete answers, and </details>.',
    "Test the learning objective, mechanism, misconception, and tradeoffs rather than trivia.",
  ].join(" "),
  exploration: [
    "Create an explicitly synthetic AI Exploration that extends the core lesson without pretending to be sourced.",
    "Include one novel analogy, one cross-domain connection or deduction, one hypothetical scenario,",
    "and one experiment or project idea. Clearly label uncertainty inside the prose.",
    "Do not use [S#] citation markers and do not rewrite the source-grounded lesson.",
  ].join(" "),
  editor: [
    "Act as the final learning editor. Rewrite for precision, explanatory depth, continuity, and signal-to-noise.",
    "Preserve every required lesson heading, valid citations, the practice contract, and the collapsed answer key.",
    "Remove generic filler and unsupported core claims. Keep AI Exploration separate, explicitly synthetic, and uncited.",
    "Never move a claim from AI Exploration into the core lesson.",
    'Return strict JSON only with string fields "lesson", "critique", "practice";',
    '"exploration" must be a string when enabled and null when disabled;',
    '"qualityNotes" must be an array of short strings.',
  ].join(" "),
});

export async function buildDossier(options) {
  const {
    config,
    items,
    history,
    provider,
    now = new Date(),
    onStage,
  } = options;
  if (!Array.isArray(items) || items.length === 0) {
    throw new Error("At least one source item is required.");
  }

  const learnerContext = formatLearnerContext(config, history);
  const candidateItems = items.map((item, index) => ({
    ...item,
    sourceId: `S${index + 1}`,
  }));
  const candidateBundle = formatSourceBundle(
    candidateItems,
    config.limits.maxItemCharacters,
  );
  const curation = validateCuration(
    await runStructuredStage(
      provider,
      "curator",
      STAGE_INSTRUCTIONS.curator,
      `${learnerContext}\n\n# Candidate sources\n\n${candidateBundle}`,
      onStage,
    ),
    candidateItems.length,
  );
  const curatedItems = curation.selectedSourceIds.map((sourceId, index) => {
    const source = candidateItems[Number(sourceId.slice(1)) - 1];
    return {
      ...source,
      originalSourceId: sourceId,
      sourceId: `S${index + 1}`,
    };
  });
  const enrichItemsFn =
    options.enrichItemsFn ??
    (config.provider.kind === "demo"
      ? demoEnrichSourceItems
      : enrichSourceItems);
  const enrichedItems = await enrichItemsFn(curatedItems, {
    fetchImpl: options.fetchImpl,
    lookupFn: options.lookupFn,
    maximumBytes: config.content?.maxArticleBytes,
    maximumCharacters: config.content?.maxArticleCharacters,
  });
  const sourceBundle = formatSourceBundle(
    enrichedItems,
    config.content?.maxArticleCharacters ??
      config.limits.maxItemCharacters,
  );

  const blueprint = validateBlueprint(
    await runStructuredStage(
      provider,
      "blueprint",
      STAGE_INSTRUCTIONS.blueprint,
      fitSections(config.limits.maxIntermediateCharacters, [
        ["Learner context", learnerContext, 2],
        ["Curated theme", JSON.stringify(curation, null, 2), 1],
        ["Enriched sources", sourceBundle, 5],
      ]),
      onStage,
    ),
  );
  const blueprintText = JSON.stringify(blueprint, null, 2);
  const research = await runStage(
    provider,
    "researcher",
    STAGE_INSTRUCTIONS.researcher,
    fitSections(config.limits.maxIntermediateCharacters, [
      ["Learner context", learnerContext, 1],
      ["Learning blueprint", blueprintText, 2],
      ["Enriched sources", sourceBundle, 6],
    ]),
    onStage,
  );
  const critique = await runStage(
    provider,
    "skeptic",
    STAGE_INSTRUCTIONS.skeptic,
    fitSections(config.limits.maxIntermediateCharacters, [
      ["Learning blueprint", blueprintText, 1],
      ["Enriched sources", sourceBundle, 5],
      ["Research brief", research, 3],
    ]),
    onStage,
  );
  const lesson = await runStage(
    provider,
    "teacher",
    STAGE_INSTRUCTIONS.teacher,
    fitSections(config.limits.maxIntermediateCharacters, [
      ["Learner context", learnerContext, 1],
      ["Learning blueprint", blueprintText, 2],
      ["Enriched sources", sourceBundle, 3],
      ["Research brief", research, 3],
      ["Skeptical review", critique, 2],
    ]),
    onStage,
  );
  const practice = await runStage(
    provider,
    "examiner",
    STAGE_INSTRUCTIONS.examiner,
    fitSections(config.limits.maxIntermediateCharacters, [
      ["Learner context", learnerContext, 1],
      ["Learning blueprint", blueprintText, 2],
      ["Source-grounded lesson", lesson, 6],
    ]),
    onStage,
  );
  const explorationEnabled =
    config.content?.aiExplorationEnabled === true;
  const exploration = explorationEnabled
    ? await runStage(
        provider,
        "exploration",
        STAGE_INSTRUCTIONS.exploration,
        fitSections(config.limits.maxIntermediateCharacters, [
          ["Learner context", learnerContext, 1],
          ["Learning blueprint", blueprintText, 2],
          ["Source-grounded lesson", lesson, 5],
          ["Skeptical review", critique, 2],
        ]),
        onStage,
      )
    : null;
  const editorial = validateEditorial(
    await runStructuredStage(
      provider,
      "editor",
      STAGE_INSTRUCTIONS.editor,
      fitSections(config.limits.maxIntermediateCharacters, [
        ["AI Exploration enabled", String(explorationEnabled), 1],
        ["Learning blueprint", blueprintText, 2],
        ["Enriched sources", sourceBundle, 3],
        ["Draft lesson", lesson, 5],
        ["Skeptical review", critique, 2],
        ["Draft practice", practice, 3],
        ["Draft AI Exploration", exploration ?? "Disabled", 2],
      ]),
      onStage,
    ),
    { explorationEnabled },
  );
  const quality = evaluateDossierContent({
    ...editorial,
    sources: enrichedItems,
    blueprint,
    historyCount: history.length,
  });

  const date = formatDate(now, config.timeZone);
  const dossier = {
    version: 2,
    profileId: config.profileId,
    date,
    title: curation.theme,
    generatedAt: now.toISOString(),
    model: config.provider.model,
    curation,
    blueprint,
    lesson: editorial.lesson,
    critique: editorial.critique,
    practice: editorial.practice,
    exploration: editorial.exploration,
    quality: {
      ...quality,
      editorNotes: editorial.qualityNotes,
    },
    sources: enrichedItems.map((item) => ({ ...item })),
  };
  const markdown = renderDossierMarkdown(dossier);

  return {
    date,
    dossier,
    markdown,
    stages: {
      curation,
      blueprint,
      research,
      critique,
      lesson,
      practice,
      exploration,
      editorial,
    },
    historyEntry: {
      date,
      generatedAt: now.toISOString(),
      sourceTitles: enrichedItems.map((item) => item.title),
      lessonSummary: truncate(stripMarkdown(editorial.lesson), 800),
      recallQuestions: extractQuestions(editorial.practice),
      learningObjective: blueprint.learningObjective,
      concepts: [
        blueprint.centralMechanism,
        ...blueprint.prerequisites,
      ].map((value) => truncate(value, 300)),
    },
  };
}

async function runStructuredStage(
  provider,
  stage,
  instruction,
  input,
  onStage,
) {
  return parseStructuredStage(
    await runStage(provider, stage, instruction, input, onStage),
    stage,
  );
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
      const sourceId = item.sourceId ?? `S${index + 1}`;
      const contents = truncate(
        item.summary || "No source text supplied.",
        maxItemCharacters,
      );
      return [
        `## [${sourceId}] ${item.title}`,
        `Source: ${item.source}`,
        `Published: ${item.publishedAt ?? "unknown"}`,
        `URL: ${item.canonicalUrl ?? item.url}`,
        `Content basis: ${
          item.contentSource === "article"
            ? "enriched article text"
            : "feed summary"
        }`,
        item.author ? `Author: ${item.author}` : null,
        `Source text: ${contents}`,
      ]
        .filter(Boolean)
        .join("\n");
    })
    .join("\n\n");
}

function formatLearnerContext(config, history) {
  const retainedHistory =
    config.limits.historyEntries === 0
      ? []
      : history.slice(-config.limits.historyEntries);
  const prior = retainedHistory.length
    ? retainedHistory
        .map((entry) => {
          const recall = Array.isArray(entry.recallQuestions)
            ? entry.recallQuestions.slice(0, 3).join(" | ")
            : "none recorded";
          return `- ${entry.date}: ${entry.lessonSummary}\n  Recall: ${recall}`;
        })
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
    "Build deliberately on prior learning when it is relevant. Do not merely repeat it.",
  ].join("\n");
}

function extractQuestions(markdown) {
  return markdown
    .split("\n")
    .map((line) => line.match(/^\s*\d+\.\s+(.+\?)\s*$/)?.[1])
    .filter(Boolean)
    .slice(0, 5);
}

function fitSections(maximum, sections) {
  const renderedHeaders = sections.map(([heading]) => `# ${heading}\n\n`);
  const separatorCharacters = Math.max(0, sections.length - 1) * 2;
  const available = Math.max(
    sections.length,
    maximum -
      renderedHeaders.reduce((total, header) => total + header.length, 0) -
      separatorCharacters,
  );
  const totalWeight = sections.reduce(
    (total, [, , weight]) => total + weight,
    0,
  );

  return sections
    .map(([heading, contents, weight], index) => {
      const allocation =
        index === sections.length - 1
          ? available -
            sections
              .slice(0, -1)
              .reduce(
                (total, [, , priorWeight]) =>
                  total +
                  Math.floor((available * priorWeight) / totalWeight),
                0,
              )
          : Math.floor((available * weight) / totalWeight);
      return `${renderedHeaders[index]}${truncate(contents, allocation)}`;
    })
    .join("\n\n");
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

function formatDate(date, timeZone) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

async function demoEnrichSourceItems(items) {
  return items.map((item) => ({
    ...item,
    contentSource: "feed-summary",
    canonicalUrl: item.url,
    author: null,
    enrichmentError: "demo mode uses bundled source summaries",
  }));
}
