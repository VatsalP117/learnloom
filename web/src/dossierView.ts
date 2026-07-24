export interface DossierSection {
  label: string;
  heading: string;
  paragraphs: string[];
  callout?: string;
}

export interface NormalizedDossier {
  readTime: number;
  deck: string;
  objective: string;
  sections: DossierSection[];
  retrieval: string[];
  application: string;
}

interface RawDossier {
  sections?: DossierSection[];
  lesson?: string;
  critique?: string;
  practice?: string;
  curation?: { rationale?: string };
  blueprint?: { continuityBridge?: string; learningObjective?: string };
  [key: string]: unknown;
}

function plainText(value: string) {
  return value
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[*_`>#]/g, "")
    .replace(/\s+/g, " ")
    .trim();
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
      .map(plainText)
      .filter((paragraph) =>
        paragraph &&
        !paragraph.startsWith("<details") &&
        !paragraph.startsWith("<summary") &&
        paragraph !== "</details>",
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

  return {
    readTime: newsletter.lessonMinutes ?? 10,
    deck:
      dossier?.curation?.rationale ||
      dossier?.blueprint?.continuityBridge ||
      "A source-grounded lesson prepared for your learning practice.",
    objective: dossier?.blueprint?.learningObjective ?? "",
    sections: [...lessonSections, ...critiqueSections],
    retrieval: retrievalQuestions(dossier?.practice ?? ""),
    application: sectionBody(dossier?.practice ?? "", "Application challenge"),
  };
}
