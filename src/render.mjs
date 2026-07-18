export function renderDossierMarkdown(dossier) {
  const pipelineDescription =
    dossier.version >= 2
      ? "curation, source enrichment, blueprint, research, skepticism, teaching, practice, and editorial validation"
      : "researcher, skeptic, teacher, and examiner passes";
  const sections = [
    `# Learning Dossier — ${dossier.date}`,
    "",
    `> Generated from ${dossier.sources.length} source items through ${pipelineDescription}.`,
    "",
    demoteTopLevelHeadings(dossier.lesson),
    "",
    demoteTopLevelHeadings(dossier.critique),
    "",
    demoteTopLevelHeadings(dossier.practice),
    "",
  ];
  if (dossier.exploration) {
    sections.push(
      "## AI Exploration",
      "",
      "> Opt-in synthetic exploration. These analogies, deductions, and scenarios extend beyond the cited sources and may be speculative.",
      "",
      demoteTopLevelHeadings(dossier.exploration),
      "",
    );
  }
  sections.push(
    "## Source Index",
    "",
    ...dossier.sources.map(
      (item, index) =>
        `${index + 1}. **[${
          item.sourceId ?? `S${index + 1}`
        }] ${escapeMarkdown(item.title)}** — ${
          item.source
        }  \n   ${item.canonicalUrl ?? item.url}`,
    ),
    ...(dossier.quality
      ? [
          "",
          `Quality gate: ${dossier.quality.score}/100 · ${
            dossier.quality.metrics.enrichedSources
          } enriched source${
            dossier.quality.metrics.enrichedSources === 1 ? "" : "s"
          } · ${dossier.quality.metrics.retrievalQuestions} retrieval questions`,
        ]
      : []),
    "",
    "---",
    "",
    `Generated at ${dossier.generatedAt} · Model output can be wrong; verify important claims at the linked sources.`,
    "",
  );
  return sections.join("\n");
}

export function renderDossierEmail(dossier, markdown = renderDossierMarkdown(dossier)) {
  const sections = [
    renderMarkdownFragment(dossier.lesson),
    renderMarkdownFragment(dossier.critique),
    renderMarkdownFragment(dossier.practice),
  ];
  const exploration = dossier.exploration
    ? `<section style="margin:32px 0 0;padding:24px;border:1px solid #f0c36a;border-radius:12px;background:#fff8e8">
        <p style="margin:0 0 8px;color:#9a5b13;font-size:11px;font-weight:800;letter-spacing:.12em;text-transform:uppercase">AI Exploration · Opt-in</p>
        <p style="margin:0 0 18px;color:#7c5a2d;font-size:13px;line-height:1.5">Synthetic analogies, deductions, and scenarios that extend beyond the cited sources. They may be speculative.</p>
        ${renderMarkdownFragment(dossier.exploration)}
      </section>`
    : "";
  const sources = dossier.sources
    .map((item, index) => {
      const url = safeHttpUrl(item.canonicalUrl ?? item.url);
      const title = escapeHtml(item.title);
      const source = escapeHtml(item.source);
      const label = `[${item.sourceId ?? `S${index + 1}`}] ${title}`;
      return `<li style="margin:0 0 10px"><strong>${label}</strong> — ${source}${
        url
          ? `<br><a href="${escapeAttribute(url)}" style="color:#047857">${escapeHtml(url)}</a>`
          : ""
      }</li>`;
    })
    .join("");
  const title = escapeHtml(dossier.title);
  const date = escapeHtml(dossier.date);
  const html = `<!doctype html>
<html>
  <body style="margin:0;background:#f5f5f4;color:#0f172a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">
    <main style="max-width:720px;margin:0 auto;padding:32px 20px">
      <div style="background:#ffffff;border:1px solid #e7e5e4;border-radius:16px;padding:32px">
        <p style="margin:0 0 8px;color:#047857;font-size:12px;font-weight:700;letter-spacing:.12em;text-transform:uppercase">Learnloom · ${date}</p>
        <h1 style="margin:0 0 28px;font-size:30px;line-height:1.2">${title}</h1>
        ${sections.join('<hr style="border:0;border-top:1px solid #e7e5e4;margin:32px 0">')}
        ${exploration}
        <hr style="border:0;border-top:1px solid #e7e5e4;margin:32px 0">
        <h2 style="font-size:20px">Sources</h2>
        <ol style="padding-left:22px">${sources}</ol>
        <p style="margin-top:28px;color:#78716c;font-size:12px">Model output can be wrong. Verify important claims at the linked sources.</p>
      </div>
    </main>
  </body>
</html>`;
  return { html, text: markdown };
}

function renderMarkdownFragment(markdown) {
  const blocks = [];
  let listType = null;
  const closeList = () => {
    if (listType) {
      blocks.push(`</${listType}>`);
      listType = null;
    }
  };

  for (const rawLine of markdown.split("\n")) {
    const line = rawLine.trim();
    if (!line) {
      closeList();
      continue;
    }
    if (/^<\/?details>$/i.test(line)) {
      closeList();
      continue;
    }
    const summary = line.match(/^<summary>([\s\S]+)<\/summary>$/i);
    if (summary) {
      closeList();
      blocks.push(`<h3 style="margin:24px 0 10px">${formatInline(summary[1])}</h3>`);
      continue;
    }
    const heading = line.match(/^(#{1,4})\s+(.+)$/);
    if (heading) {
      closeList();
      const level = Math.min(heading[1].length + 1, 4);
      blocks.push(`<h${level} style="margin:24px 0 10px">${formatInline(heading[2])}</h${level}>`);
      continue;
    }
    const unordered = line.match(/^[-*]\s+(.+)$/);
    const ordered = line.match(/^\d+\.\s+(.+)$/);
    if (unordered || ordered) {
      const wantedType = unordered ? "ul" : "ol";
      if (listType !== wantedType) {
        closeList();
        listType = wantedType;
        blocks.push(`<${wantedType} style="padding-left:24px">`);
      }
      blocks.push(`<li style="margin:0 0 8px">${formatInline((unordered || ordered)[1])}</li>`);
      continue;
    }
    closeList();
    blocks.push(`<p style="margin:0 0 14px;line-height:1.65">${formatInline(line)}</p>`);
  }
  closeList();
  return blocks.join("\n");
}

function formatInline(value) {
  return escapeHtml(value)
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/`([^`]+)`/g, "<code style=\"background:#f5f5f4;padding:1px 4px;border-radius:4px\">$1</code>");
}

function safeHttpUrl(value) {
  try {
    const url = new URL(value);
    return ["http:", "https:"].includes(url.protocol) ? url.toString() : null;
  } catch {
    return null;
  }
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttribute(value) {
  return escapeHtml(value);
}

function demoteTopLevelHeadings(markdown) {
  return markdown.replace(/^# /gm, "## ");
}

function escapeMarkdown(value) {
  return value.replace(/([\\[\]*_])/g, "\\$1");
}
