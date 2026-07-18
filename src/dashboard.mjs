import { randomUUID, timingSafeEqual } from "node:crypto";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { renderDossierEmail } from "./render.mjs";
import { loadJson } from "./state.mjs";

const MAX_FORM_BYTES = 64 * 1024;

export function createDashboardServer(options) {
  const csrfToken = options.csrfToken ?? randomUUID();
  const server = createServer(async (request, response) => {
    try {
      await handleRequest(request, response, { ...options, csrfToken });
    } catch (error) {
      if (!response.headersSent) {
        sendHtml(
          response,
          error.statusCode ?? 500,
          page(
            "Something went wrong",
            `<section class="empty"><p>${escapeHtml(
              error.statusCode ? error.message : "The dashboard could not complete that request.",
            )}</p><a class="button secondary" href="/">Back to dashboard</a></section>`,
          ),
        );
      } else {
        response.end();
      }
      options.onError?.(error);
    }
  });
  return { server, csrfToken };
}

async function handleRequest(request, response, options) {
  const method = request.method ?? "GET";
  const url = new URL(request.url ?? "/", "http://localhost");
  requireAllowedHost(request.headers.host, options.allowedHosts);

  if (url.pathname === "/healthz") {
    if (method !== "GET") return methodNotAllowed(response, ["GET"]);
    return sendText(response, 200, "ok\n");
  }

  if (url.pathname === "/") {
    if (method !== "GET") return methodNotAllowed(response, ["GET"]);
    return sendHtml(response, 200, renderOverview(options.workspace));
  }

  if (url.pathname === "/newsletters/new") {
    if (method !== "GET") return methodNotAllowed(response, ["GET"]);
    return sendHtml(
      response,
      200,
      renderNewsletterForm(options.baseConfig, options.csrfToken),
    );
  }

  if (url.pathname === "/newsletters") {
    if (method !== "POST") return methodNotAllowed(response, ["POST"]);
    const form = await requireForm(request, options.csrfToken);
    let newsletter;
    try {
      newsletter = options.workspace.createNewsletter(
        newsletterFromForm(form, options.baseConfig),
      );
    } catch (error) {
      throw httpError(400, error.message);
    }
    return redirect(response, `/newsletters/${encodeURIComponent(newsletter.id)}`);
  }

  const newsletterMatch = /^\/newsletters\/([a-z0-9_-]+)$/.exec(url.pathname);
  if (newsletterMatch) {
    if (method !== "GET") return methodNotAllowed(response, ["GET"]);
    const newsletter = options.workspace.getNewsletter(newsletterMatch[1]);
    if (!newsletter) return notFound(response);
    const issues = options.workspace.listIssues(newsletter.id);
    return sendHtml(
      response,
      200,
      renderNewsletterDetail(newsletter, issues, options.csrfToken, url),
    );
  }

  const runMatch = /^\/newsletters\/([a-z0-9_-]+)\/run$/.exec(url.pathname);
  if (runMatch) {
    if (method !== "POST") return methodNotAllowed(response, ["POST"]);
    await requireForm(request, options.csrfToken);
    if (!options.workspace.getNewsletter(runMatch[1])) return notFound(response);
    const issue = options.workspace.enqueueManualIssue(runMatch[1]);
    return redirect(
      response,
      `/newsletters/${encodeURIComponent(runMatch[1])}?queued=${encodeURIComponent(
        issue.id,
      )}`,
    );
  }

  const toggleMatch = /^\/newsletters\/([a-z0-9_-]+)\/toggle$/.exec(url.pathname);
  if (toggleMatch) {
    if (method !== "POST") return methodNotAllowed(response, ["POST"]);
    await requireForm(request, options.csrfToken);
    const newsletter = options.workspace.getNewsletter(toggleMatch[1]);
    if (!newsletter) return notFound(response);
    options.workspace.setNewsletterActive(newsletter.id, !newsletter.active);
    return redirect(
      response,
      `/newsletters/${encodeURIComponent(newsletter.id)}`,
    );
  }

  const issueMatch = /^\/issues\/([a-z0-9_-]+)$/.exec(url.pathname);
  if (issueMatch) {
    if (method !== "GET") return methodNotAllowed(response, ["GET"]);
    const issue = options.workspace.getIssue(issueMatch[1]);
    if (!issue) return notFound(response);
    const newsletter = options.workspace.getNewsletter(issue.newsletterId);
    return sendHtml(
      response,
      200,
      await renderIssuePreview(newsletter, issue),
    );
  }

  return notFound(response);
}

function renderOverview(workspace) {
  const newsletters = workspace.listNewsletters();
  const generated = newsletters.reduce(
    (total, newsletter) => total + newsletter.generatedCount,
    0,
  );
  const active = newsletters.filter((newsletter) => newsletter.active).length;
  const cards = newsletters.length
    ? newsletters.map(renderNewsletterCard).join("")
    : `<section class="empty">
        <p class="eyebrow">A quiet beginning</p>
        <h2>Create your first learning stream</h2>
        <p>Choose a focused topic, a daily time, and the sources you trust.</p>
        <a class="button" href="/newsletters/new">Create Newsletter</a>
      </section>`;
  return page(
    "Your Newsletters",
    `<header class="hero">
      <div>
        <p class="eyebrow">Learning control room</p>
        <h1>Your Newsletters</h1>
        <p class="lede">Small, recurring lessons built around the questions you want to stay close to.</p>
      </div>
      <a class="button" href="/newsletters/new">New Newsletter</a>
    </header>
    <section class="stats" aria-label="Workspace totals">
      ${stat("Newsletters", newsletters.length)}
      ${stat("Active", active)}
      ${stat("Generated issues", generated)}
    </section>
    <section class="card-grid">${cards}</section>`,
  );
}

function renderNewsletterCard(newsletter) {
  return `<article class="newsletter-card">
    <div class="card-top">
      ${statusPill(newsletter.active ? "active" : "paused")}
      <span class="schedule">${escapeHtml(newsletter.scheduleTime)} · ${escapeHtml(
        newsletter.timeZone,
      )}</span>
    </div>
    <h2>${escapeHtml(newsletter.name)}</h2>
    <p>${escapeHtml(newsletter.topic)}</p>
    <dl class="card-metrics">
      <div><dt>Generated</dt><dd>${newsletter.generatedCount}</dd></div>
      <div><dt>Latest</dt><dd>${escapeHtml(
        humanStatus(newsletter.latestStatus),
      )}</dd></div>
    </dl>
    <div class="next-run">Next: ${escapeHtml(
      newsletter.active
        ? formatTimestamp(newsletter.nextRunAt, newsletter.timeZone)
        : "Paused",
    )}</div>
    <a class="card-link" href="/newsletters/${encodeURIComponent(
      newsletter.id,
    )}">Open Newsletter <span aria-hidden="true">→</span></a>
  </article>`;
}

function renderNewsletterForm(baseConfig, csrfToken) {
  const defaultSources = baseConfig.sources.map((source) => source.url).join("\n");
  return page(
    "Create Newsletter",
    `<header class="page-header">
      <div><a class="back" href="/">← Dashboard</a><p class="eyebrow">New learning stream</p><h1>Create Newsletter</h1></div>
    </header>
    <form class="form-card" method="post" action="/newsletters">
      ${csrfField(csrfToken)}
      <label><span>Name</span><input required maxlength="100" name="name" placeholder="RabbitMQ Deep Dive"></label>
      <label><span>Topic and focus</span><textarea required maxlength="500" name="topic" rows="4" placeholder="RabbitMQ architecture, delivery guarantees, backpressure, and operating healthy clusters"></textarea></label>
      <div class="form-row">
        <label><span>Daily time</span><input required type="time" name="scheduleTime" value="10:00"></label>
        <label><span>Timezone</span><input required name="timeZone" value="${escapeAttribute(
          baseConfig.timeZone,
        )}"></label>
      </div>
      <label><span>RSS or Atom feeds</span><textarea name="sourceUrls" rows="5" placeholder="One URL per line">${escapeHtml(
        defaultSources,
      )}</textarea><small>One URL per line. Leave the installation defaults in place for the first test.</small></label>
      <div class="form-actions"><a class="button secondary" href="/">Cancel</a><button class="button" type="submit">Create Newsletter</button></div>
    </form>`,
  );
}

function renderNewsletterDetail(newsletter, issues, csrfToken, url) {
  const queued = url.searchParams.get("queued");
  const notice = queued
    ? `<div class="notice">Issue queued. The worker will pick it up shortly.</div>`
    : "";
  const issueRows = issues.length
    ? issues.map(renderIssueRow).join("")
    : `<div class="empty compact"><p>No Issues yet. Queue the first one when you are ready.</p></div>`;
  return page(
    newsletter.name,
    `<header class="detail-hero">
      <div>
        <a class="back" href="/">← Dashboard</a>
        <div class="title-line">${statusPill(newsletter.active ? "active" : "paused")}<span>${escapeHtml(
          newsletter.scheduleTime,
        )} · ${escapeHtml(newsletter.timeZone)}</span></div>
        <h1>${escapeHtml(newsletter.name)}</h1>
        <p class="lede">${escapeHtml(newsletter.topic)}</p>
      </div>
      <div class="action-stack">
        <form method="post" action="/newsletters/${encodeURIComponent(
          newsletter.id,
        )}/run">${csrfField(
          csrfToken,
        )}<button class="button" type="submit">Run now</button></form>
        <form method="post" action="/newsletters/${encodeURIComponent(
          newsletter.id,
        )}/toggle">${csrfField(
          csrfToken,
        )}<button class="button secondary" type="submit">${
          newsletter.active ? "Pause" : "Resume"
        }</button></form>
      </div>
    </header>
    ${notice}
    <section class="stats">
      ${stat("Issues", newsletter.issueCount)}
      ${stat("Generated", newsletter.generatedCount)}
      ${stat(
        "Next run",
        newsletter.active
          ? formatTimestamp(newsletter.nextRunAt, newsletter.timeZone)
          : "Paused",
      )}
    </section>
    <section class="history">
      <div class="section-heading"><div><p class="eyebrow">Archive</p><h2>Issue history</h2></div></div>
      <div class="issue-list">${issueRows}</div>
    </section>`,
  );
}

function renderIssueRow(issue) {
  const title =
    issue.title ??
    (issue.trigger === "manual" ? "Manual Issue" : "Scheduled Issue");
  const href =
    issue.status === "generated"
      ? `href="/issues/${encodeURIComponent(issue.id)}"`
      : "";
  return `<a class="issue-row ${
    issue.status === "generated" ? "" : "disabled"
    }" ${href}>
    <div>${statusPill(issue.status)}<strong>${escapeHtml(
      title,
    )}</strong></div>
    <div class="issue-meta"><span>${escapeHtml(
      issue.trigger,
    )}</span><span>${escapeHtml(formatTimestamp(issue.createdAt))}</span><span aria-hidden="true">→</span></div>
  </a>`;
}

async function renderIssuePreview(newsletter, issue) {
  let content;
  if (issue.status === "generated" && issue.dossierPath && issue.artifactPath) {
    const dossier = await loadJson(issue.dossierPath, "Issue Dossier");
    const markdown = await readFile(issue.artifactPath, "utf8");
    if (!dossier) throw httpError(404, "The Dossier artifact was not found.");
    const rendered = renderDossierEmail(dossier, markdown).html;
    content = /<body[^>]*>([\s\S]*)<\/body>/i.exec(rendered)?.[1] ?? rendered;
  } else {
    content = `<section class="empty"><p>This Issue is ${escapeHtml(
      issue.status,
    )} and does not have a Dossier preview yet.</p></section>`;
  }
  return page(
    issue.title ?? "Issue preview",
    `<header class="page-header">
      <div><a class="back" href="/newsletters/${encodeURIComponent(
        newsletter.id,
      )}">← ${escapeHtml(newsletter.name)}</a><p class="eyebrow">Dossier preview</p><h1>${escapeHtml(
        issue.title ?? "Issue preview",
      )}</h1><div class="title-line">${statusPill(issue.status)}<span>${escapeHtml(
        formatTimestamp(issue.completedAt ?? issue.createdAt),
      )}</span></div></div>
    </header>
    <article class="dossier">${content}</article>`,
  );
}

function newsletterFromForm(form, baseConfig) {
  const sourceLines = String(form.get("sourceUrls") ?? "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  const sourceUrls = sourceLines.length
    ? sourceLines
    : baseConfig.sources.map((source) => source.url);
  return {
    name: form.get("name"),
    topic: form.get("topic"),
    scheduleTime: form.get("scheduleTime"),
    timeZone: form.get("timeZone"),
    learnerLevel: baseConfig.learner.level,
    learnerGoal: `build durable, practical understanding of ${form.get("topic")}`,
    lessonMinutes: baseConfig.learner.lessonMinutes,
    sources: sourceUrls.map((url) => {
      let name = "Configured feed";
      try {
        name = new URL(url).hostname;
      } catch {
        // The Workspace returns the user-facing validation error.
      }
      return { name, url, limit: 10 };
    }),
  };
}

async function requireForm(request, csrfToken) {
  const contentType = request.headers["content-type"] ?? "";
  if (!contentType.startsWith("application/x-www-form-urlencoded")) {
    throw httpError(415, "Expected a form submission.");
  }
  const chunks = [];
  let length = 0;
  for await (const chunk of request) {
    length += chunk.length;
    if (length > MAX_FORM_BYTES) throw httpError(413, "Form submission is too large.");
    chunks.push(chunk);
  }
  const form = new URLSearchParams(Buffer.concat(chunks).toString("utf8"));
  if (!tokensEqual(form.get("_csrf"), csrfToken)) {
    throw httpError(403, "The form token expired. Reload the page and try again.");
  }
  return form;
}

function tokensEqual(actual, expected) {
  if (typeof actual !== "string" || actual.length !== expected.length) return false;
  return timingSafeEqual(Buffer.from(actual), Buffer.from(expected));
}

function requireAllowedHost(hostHeader, configuredHosts) {
  if (typeof hostHeader !== "string" || hostHeader.length > 255) {
    throw httpError(421, "The request Host is not allowed.");
  }
  let hostname;
  try {
    hostname = new URL(`http://${hostHeader}`).hostname.toLowerCase();
  } catch {
    throw httpError(421, "The request Host is not allowed.");
  }
  const allowedHosts = new Set(
    (configuredHosts ?? ["127.0.0.1", "localhost", "[::1]"]).map((host) =>
      String(host).toLowerCase(),
    ),
  );
  if (!allowedHosts.has(hostname)) {
    throw httpError(421, "The request Host is not allowed.");
  }
}

function page(title, body) {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHtml(title)} · Learnloom</title>
  <style>${STYLES}</style>
</head>
<body>
  <nav><a class="brand" href="/"><span>LL</span> Learnloom</a><span class="test-label">Test phase · local only</span></nav>
  <main>${body}</main>
  <footer>Learnloom shapes source material into durable understanding.</footer>
</body>
</html>`;
}

function stat(label, value) {
  return `<div><dt>${escapeHtml(label)}</dt><dd>${escapeHtml(String(value))}</dd></div>`;
}

function statusPill(status) {
  return `<span class="pill ${escapeAttribute(status)}">${escapeHtml(
    humanStatus(status),
  )}</span>`;
}

function humanStatus(status) {
  if (!status) return "No Issues";
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function csrfField(token) {
  return `<input type="hidden" name="_csrf" value="${escapeAttribute(token)}">`;
}

function formatTimestamp(value, timeZone) {
  if (!value) return "Not yet";
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return "Unknown";
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone,
  }).format(date);
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function escapeAttribute(value) {
  return escapeHtml(value).replaceAll("`", "&#096;");
}

function redirect(response, location) {
  response.writeHead(303, {
    location,
    "cache-control": "no-store",
  });
  response.end();
}

function sendHtml(response, status, html) {
  response.writeHead(status, {
    "content-type": "text/html; charset=utf-8",
    "cache-control": "no-store",
    "x-content-type-options": "nosniff",
    "content-security-policy":
      "default-src 'none'; style-src 'unsafe-inline'; img-src data:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'",
    "referrer-policy": "no-referrer",
  });
  response.end(html);
}

function sendText(response, status, text) {
  response.writeHead(status, {
    "content-type": "text/plain; charset=utf-8",
    "cache-control": "no-store",
    "x-content-type-options": "nosniff",
  });
  response.end(text);
}

function methodNotAllowed(response, allowed) {
  response.setHeader("allow", allowed.join(", "));
  return sendText(response, 405, "Method not allowed\n");
}

function notFound(response) {
  return sendHtml(
    response,
    404,
    page(
      "Not found",
      '<section class="empty"><h1>That page wandered off.</h1><a class="button secondary" href="/">Back to dashboard</a></section>',
    ),
  );
}

function httpError(statusCode, message) {
  const error = new Error(message);
  error.statusCode = statusCode;
  return error;
}

const STYLES = `
:root{--ink:#14211d;--muted:#65736d;--paper:#f4f1e8;--card:#fffdf8;--line:#d9d8ce;--green:#176b50;--green2:#dff1e7;--navy:#132d29;--amber:#b96b27;--red:#a8433b}
*{box-sizing:border-box}body{margin:0;background:var(--paper);color:var(--ink);font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}a{color:inherit}nav{height:72px;display:flex;align-items:center;justify-content:space-between;padding:0 max(24px,calc((100vw - 1120px)/2));border-bottom:1px solid var(--line);background:rgba(244,241,232,.92)}.brand{text-decoration:none;font-weight:760;letter-spacing:-.02em}.brand span{display:inline-grid;place-items:center;width:34px;height:34px;margin-right:9px;border-radius:10px;background:var(--navy);color:white;font-size:12px}.test-label{font-size:12px;color:var(--muted);text-transform:uppercase;letter-spacing:.12em}main{max-width:1120px;margin:0 auto;padding:70px 24px 100px}footer{max-width:1120px;margin:0 auto;padding:30px 24px 50px;border-top:1px solid var(--line);color:var(--muted);font-size:13px}.hero,.detail-hero,.page-header{display:flex;align-items:flex-end;justify-content:space-between;gap:32px;margin-bottom:44px}.hero h1,.detail-hero h1,.page-header h1{font-family:Georgia,serif;font-size:clamp(42px,6vw,74px);line-height:.98;letter-spacing:-.045em;margin:10px 0 18px}.page-header h1{font-size:clamp(40px,5vw,62px)}.eyebrow{text-transform:uppercase;letter-spacing:.16em;font-size:11px;font-weight:800;color:var(--green);margin:0}.lede{max-width:700px;color:var(--muted);font-size:18px;line-height:1.6;margin:0}.button{display:inline-flex;align-items:center;justify-content:center;border:1px solid var(--green);border-radius:999px;padding:12px 20px;background:var(--green);color:white;font:inherit;font-weight:700;text-decoration:none;cursor:pointer;white-space:nowrap}.button:hover{filter:brightness(.94)}.button.secondary{background:transparent;color:var(--green)}.stats{display:grid;grid-template-columns:repeat(3,1fr);border:1px solid var(--line);border-radius:16px;background:rgba(255,253,248,.65);margin-bottom:36px;overflow:hidden}.stats>div{padding:22px 24px;border-right:1px solid var(--line)}.stats>div:last-child{border:0}.stats dt{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.1em}.stats dd{font-family:Georgia,serif;font-size:28px;margin:7px 0 0}.card-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}.newsletter-card{display:flex;flex-direction:column;min-height:310px;padding:28px;border:1px solid var(--line);border-radius:18px;background:var(--card);box-shadow:0 8px 30px rgba(20,33,29,.045)}.card-top,.title-line{display:flex;align-items:center;gap:12px;justify-content:space-between;color:var(--muted);font-size:12px}.newsletter-card h2{font-family:Georgia,serif;font-size:29px;line-height:1.1;margin:28px 0 10px}.newsletter-card>p{color:var(--muted);line-height:1.55;margin:0}.pill{display:inline-flex;border-radius:999px;padding:5px 9px;background:#e9ebe6;color:#57625d;font-size:10px;font-weight:850;text-transform:uppercase;letter-spacing:.1em}.pill.active,.pill.generated{background:var(--green2);color:var(--green)}.pill.queued,.pill.generating{background:#fff0d8;color:var(--amber)}.pill.failed{background:#f9dfdc;color:var(--red)}.card-metrics{display:flex;gap:34px;margin:auto 0 18px}.card-metrics dt{font-size:11px;text-transform:uppercase;letter-spacing:.09em;color:var(--muted)}.card-metrics dd{font-weight:750;margin:5px 0 0}.next-run{border-top:1px solid var(--line);padding-top:16px;color:var(--muted);font-size:13px}.card-link{display:flex;justify-content:space-between;margin-top:16px;color:var(--green);font-weight:750;text-decoration:none}.empty{padding:70px 30px;text-align:center;border:1px dashed #b8beb6;border-radius:18px;background:rgba(255,253,248,.5)}.empty.compact{padding:36px}.empty h2{font-family:Georgia,serif;font-size:34px;margin:10px}.empty p{color:var(--muted);margin:12px auto 26px;max-width:500px}.back{display:inline-block;margin-bottom:26px;color:var(--muted);font-size:13px;text-decoration:none}.form-card{max-width:780px;padding:32px;border:1px solid var(--line);border-radius:20px;background:var(--card)}label{display:block;margin-bottom:24px}label>span{display:block;margin-bottom:8px;font-size:13px;font-weight:750}input,textarea{width:100%;border:1px solid #bdc2ba;border-radius:10px;background:#fff;padding:12px 14px;color:var(--ink);font:inherit}input:focus,textarea:focus{outline:3px solid rgba(23,107,80,.16);border-color:var(--green)}textarea{resize:vertical;line-height:1.5}small{display:block;color:var(--muted);margin-top:7px}.form-row{display:grid;grid-template-columns:1fr 2fr;gap:18px}.form-actions{display:flex;justify-content:flex-end;gap:12px;margin-top:32px}.detail-hero{align-items:center}.action-stack{display:flex;gap:10px}.title-line{justify-content:flex-start;margin:16px 0}.notice{margin:-12px 0 28px;padding:14px 18px;border-radius:10px;background:var(--green2);color:var(--green);font-weight:650}.section-heading{display:flex;justify-content:space-between;align-items:end;margin:50px 0 18px}.section-heading h2{font-family:Georgia,serif;font-size:34px;margin:6px 0}.issue-list{border-top:1px solid var(--line)}.issue-row{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:20px 8px;border-bottom:1px solid var(--line);text-decoration:none}.issue-row>div:first-child{display:flex;align-items:center;gap:14px}.issue-row:not(.disabled):hover{background:rgba(255,253,248,.7)}.issue-row.disabled{cursor:default}.issue-meta{display:flex;gap:20px;color:var(--muted);font-size:12px}.dossier{max-width:820px;margin:0 auto;padding:42px;border:1px solid var(--line);border-radius:18px;background:var(--card);line-height:1.65}.dossier h1,.dossier h2,.dossier h3{font-family:Georgia,serif;line-height:1.2}.dossier a{color:var(--green)}.dossier pre{white-space:pre-wrap}
@media(max-width:760px){nav{padding:0 18px}.test-label{display:none}main{padding:44px 18px 70px}.hero,.detail-hero,.page-header{display:block}.hero .button{margin-top:24px}.stats{grid-template-columns:1fr}.stats>div{border-right:0;border-bottom:1px solid var(--line)}.card-grid{grid-template-columns:1fr}.form-row{grid-template-columns:1fr}.action-stack{margin-top:24px}.issue-row{align-items:flex-start;flex-direction:column}.issue-meta{width:100%;justify-content:space-between}.dossier{padding:24px}}
`;
