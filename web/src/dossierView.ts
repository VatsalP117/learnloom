export interface DossierSection {
  label: string;
  heading: string;
  paragraphs: Array<string | DossierParagraph>;
  callout?: string;
}

export interface DossierParagraph {
  text: string;
  sourceIds: string[];
}

export interface RetrievalItem {
  id: string;
  prompt: string;
  answerRubric?: string;
  correctiveExplanation?: string;
}

export interface EvidenceClaim {
  id: string;
  text: string;
  sourceIds: string[];
}

export interface NormalizedDossier {
  readTime: number;
  deck: string;
  objective: string;
  whyNow: string;
  continuityBridge: string;
  concepts: string[];
  sections: DossierSection[];
  retrieval: string[];
  retrievalItems: RetrievalItem[];
  application: string;
  claims: EvidenceClaim[];
}

interface RawDossier {
  sections?: DossierSection[];
  lesson?: string;
  critique?: string;
  practice?: string;
  curation?: { rationale?: string };
  blueprint?: { continuityBridge?: string; learningObjective?: string };
  learning?: {
    selectionRationale?: string;
    continuityBridge?: string;
    concepts?: Array<{ label?: string }>;
    retrieval?: RetrievalItem[];
    claims?: EvidenceClaim[];
    application?: string;
  };
  [key: string]: unknown;
}

function plainText(value: string) {
  return value
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[*_`>#]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

function dossierParagraph(value: string): DossierParagraph {
  return {
    text: plainText(value)
      .replace(/\[S\d+\]/g, "")
      .replace(/\s+/g, " ")
      .replace(/\s+([.,;:!?])/g, "$1")
      .trim(),
    sourceIds: [...value.matchAll(/\[(S\d+)\]/g)].map((match) => match[1]),
  };
}

function markdownSections(markdown: string, startIndex = 0): DossierSection[] {
  const chunks = markdown
    .split(/^##\s+/m)
    .map((chunk) => chunk.trim())
    .filter(Boolean);

  return chunks.map((chunk, index) => {
    const [heading, ...bodyLines] = chunk.split("\n");
    const paragraphs = bodyLines
      .join("\n")
      .split(/\n\s*\n/)
      .map(dossierParagraph)
      .filter((paragraph) =>
        paragraph.text &&
        !paragraph.text.startsWith("<details") &&
        !paragraph.text.startsWith("<summary") &&
        paragraph.text !== "</details>",
      );
    return {
      label: String(startIndex + index + 1).padStart(2, "0"),
      heading: plainText(heading),
      paragraphs,
    };
  });
}

function sectionBody(markdown: string, heading: string) {
  const escaped = heading.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = markdown.match(new RegExp(
    `(?:^|\\n)##\\s+${escaped}\\s*\\n([\\s\\S]*?)(?=\\n##\\s+|$)`,
    "i",
  ));
  return match ? plainText(match[1]) : "";
}

function retrievalQuestions(markdown: string) {
  const body = sectionBody(markdown, "Retrieval practice");
  return body
    .split(/(?=\d+\.\s+)/)
    .map((line) => line.replace(/^\d+\.\s*/, "").trim())
    .filter((line) => line.endsWith("?"));
}

export function normalizeDossier(
  dossier?: RawDossier | null,
  newsletter: { lessonMinutes?: number } = {},
): NormalizedDossier {
  if (Array.isArray(dossier?.sections)) {
    return dossier as unknown as NormalizedDossier;
  }

  const lessonSections = markdownSections(dossier?.lesson ?? "");
  const critiqueSections = markdownSections(
    dossier?.critique ?? "",
    lessonSections.length,
  );
  const retrieval = retrievalQuestions(dossier?.practice ?? "");
  const retrievalItems = dossier?.learning?.retrieval?.length
    ? dossier.learning.retrieval
    : retrieval.map((prompt, index) => ({
      id: `legacy-retrieval-${index + 1}`,
      prompt,
    }));

  return {
    readTime: newsletter.lessonMinutes ?? 10,
    deck:
      dossier?.curation?.rationale ||
      dossier?.blueprint?.continuityBridge ||
      "A source-grounded lesson prepared for your learning practice.",
    objective: dossier?.blueprint?.learningObjective ?? "",
    whyNow:
      dossier?.learning?.selectionRationale ||
      dossier?.curation?.rationale ||
      "",
    continuityBridge:
      dossier?.learning?.continuityBridge ||
      dossier?.blueprint?.continuityBridge ||
      "",
    concepts: dossier?.learning?.concepts
      ?.map((concept) => concept.label?.trim() ?? "")
      .filter(Boolean) ?? [],
    sections: [...lessonSections, ...critiqueSections],
    retrieval,
    retrievalItems,
    application:
      dossier?.learning?.application ||
      sectionBody(dossier?.practice ?? "", "Application challenge"),
    claims: dossier?.learning?.claims ?? [],
  };
}
