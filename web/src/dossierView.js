function plainText(value) {
  return value
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[*_`>#]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

function markdownSections(markdown, startIndex = 0) {
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

function sectionBody(markdown, heading) {
  const escaped = heading.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = markdown.match(new RegExp(
    `(?:^|\\n)##\\s+${escaped}\\s*\\n([\\s\\S]*?)(?=\\n##\\s+|$)`,
    "i",
  ));
  return match ? plainText(match[1]) : "";
}

function retrievalQuestions(markdown) {
  const body = sectionBody(markdown, "Retrieval practice");
  return body
    .split(/(?=\d+\.\s+)/)
    .map((line) => line.replace(/^\d+\.\s*/, "").trim())
    .filter((line) => line.endsWith("?"));
}

export function normalizeDossier(dossier, newsletter = {}) {
  if (Array.isArray(dossier?.sections)) return dossier;

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
