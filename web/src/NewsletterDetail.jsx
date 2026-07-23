import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  Clock3,
  ExternalLink,
  Globe2,
  Mail,
  Pause,
  Play,
  RefreshCw,
  RotateCcw,
  Sparkles,
  WandSparkles,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading, formatShortDate } from "./LearningShell.jsx";
import { apiJSON, demoMode } from "./api.js";
import { lessonState } from "./learningState.js";
import { invalidateWorkspaceCache } from "./useWorkspace.js";

export default function NewsletterDetail({ newsletterId }) {
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState(noticeFromLocation());
  const [busy, setBusy] = useState("");

  const load = useCallback(async ({ signal } = {}) => {
    setError("");
    try {
      setSnapshot(
        await apiJSON(`/api/newsletters/${encodeURIComponent(newsletterId)}`, { signal }),
      );
    } catch (requestError) {
      if (requestError.name === "AbortError") return;
      setError(requestError.message);
    }
  }, [newsletterId]);

  useEffect(() => {
    const controller = new AbortController();
    load({ signal: controller.signal });
    return () => controller.abort();
  }, [load]);

  useEffect(() => {
    if (!snapshot?.newsletter?.name) return undefined;
    document.title = `${snapshot.newsletter.name} · Learnloom`;
    return () => {
      document.title = "Learnloom";
    };
  }, [snapshot?.newsletter?.name]);

  async function submit(action, body, successMessage) {
    setBusy(action);
    setError("");
    try {
      await apiJSON(action, { method: "POST", body });
      invalidateWorkspaceCache(newsletterId);
      await load();
      setNotice(successMessage);
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  if (!snapshot && !error) {
    return <LearningShell active="streams"><AtelierLoading label="Opening this learning stream…" /></LearningShell>;
  }

  const newsletter = snapshot?.newsletter;
  const issues = snapshot?.issues ?? [];
  const latest = issues[0];
  const generated = issues.filter((issue) => issue.status === "generated");
  const preparing = issues.find((issue) => ["queued", "generating"].includes(issue.status));

  return (
    <LearningShell active="streams">
      <section className="atelier-page stream-overview-page">
        {error ? <AtelierError message={error} onRetry={load} /> : null}
        {newsletter ? (
          <>
            <a className="atelier-back" href="/streams"><ArrowLeft size={14} /> All streams</a>
            <header className="stream-overview-header">
              <div>
                <div className="stream-status-row">
                  <span className={`atelier-status ${newsletter.active ? "active" : ""}`}>
                    {newsletter.active ? "Active" : "Paused"}
                  </span>
                  <span>{newsletter.generatedCount} lessons in your history</span>
                </div>
                <h1>{newsletter.name}</h1>
                <p>{newsletter.topic}</p>
              </div>
              <div className="stream-header-actions">
                <button
                  type="button"
                  disabled={Boolean(busy)}
                  onClick={() =>
                    submit(
                      `/api/newsletters/${encodeURIComponent(newsletter.id)}/active`,
                      { active: !newsletter.active },
                      newsletter.active ? "Stream paused." : "Stream resumed.",
                    )
                  }
                >
                  {newsletter.active ? <Pause size={15} /> : <Play size={15} />}
                  {newsletter.active ? "Pause" : "Resume"}
                </button>
                <button
                  className="atelier-primary"
                  type="button"
                  disabled={Boolean(busy) || Boolean(preparing)}
                  onClick={() =>
                    submit(
                      `/api/newsletters/${encodeURIComponent(newsletter.id)}/run`,
                      {},
                      "Your lesson is being prepared. You can leave this page.",
                    )
                  }
                >
                  <RefreshCw className={busy.endsWith("/run") ? "spin" : ""} size={15} />
                  {preparing ? "Preparing lesson…" : generated.length ? "Prepare a lesson now" : "Prepare first lesson"}
                </button>
              </div>
            </header>

            {notice ? (
              <div className="atelier-notice" role="status">
                <Check size={15} />
                <span>{notice}</span>
                <button type="button" onClick={() => setNotice("")}>Dismiss</button>
              </div>
            ) : null}

            {preparing ? (
              <section className="stream-preparing glass-panel">
                <span className="atelier-spinner" />
                <div>
                  <p className="atelier-eyebrow">Quietly working in the background</p>
                  <h2>Your next lesson is being prepared.</h2>
                  <p>
                    Learnloom is selecting useful material, checking evidence, and connecting it
                    to your learning history. You can safely leave this page.
                  </p>
                </div>
              </section>
            ) : null}

            <div className="stream-overview-grid">
              <div className="stream-overview-main">
                {latest?.status === "generated" ? (
                  <article className="latest-lesson-card glass-panel">
                    <div className="latest-lesson-heading">
                      <span className="atelier-chip"><BookOpen size={13} /> Latest lesson</span>
                      <span><Clock3 size={13} />{newsletter.lessonMinutes} min</span>
                    </div>
                    <h2>{latest.title}</h2>
                    <p>
                      Continue this thread with a focused lesson grounded in your source library.
                    </p>
                    <div className="latest-lesson-footer">
                      <span>{formatShortDate(latest.createdAt)}</span>
                      <a className="atelier-primary" href={lessonHref(latest.id)}>
                        {lessonState(latest.id).progress ? "Continue lesson" : "Open lesson"}
                        <ArrowRight size={15} />
                      </a>
                    </div>
                  </article>
                ) : (
                  <article className="latest-lesson-card glass-panel">
                    <span className="atelier-icon"><Sparkles size={18} /></span>
                    <h2>Your first lesson will begin the thread.</h2>
                    <p>Prepare it now, or let Learnloom follow your scheduled rhythm.</p>
                  </article>
                )}

                <section className="stream-history">
                  <div className="section-heading-row">
                    <div>
                      <p className="atelier-eyebrow">Learning history</p>
                      <h2>Lessons in this thread</h2>
                    </div>
                    <span>Newest first</span>
                  </div>
                  {issues.length ? (
                    <div className="stream-lesson-list">
                      {issues.map((issue, index) => (
                        <article className="stream-lesson-row glass-panel" key={issue.id}>
                          <span className="stream-lesson-index">{String(index + 1).padStart(2, "0")}</span>
                          <div>
                            <span>{formatShortDate(issue.createdAt)} · {humanize(issue.status)}</span>
                            <h3>{issue.title ?? "Lesson in preparation"}</h3>
                            {issue.error ? <p className="row-error">{issue.error}</p> : null}
                          </div>
                          <div className="stream-lesson-actions">
                            {issue.status === "generated" ? (
                              <>
                                <a href={lessonHref(issue.id)}>Read <ArrowRight size={14} /></a>
                                <button
                                  type="button"
                                  disabled={Boolean(busy)}
                                  onClick={() =>
                                    submit(
                                      `/api/issues/${encodeURIComponent(issue.id)}/publication`,
                                      {
                                        state: issue.publicationState === "published" ? "hidden" : "published",
                                      },
                                      issue.publicationState === "published"
                                        ? "Lesson hidden from your personal site."
                                        : "Lesson published to your personal site.",
                                    )
                                  }
                                >
                                  {issue.publicationState === "published" ? "Hide from site" : "Publish"}
                                </button>
                              </>
                            ) : null}
                            {issue.status === "failed" ? (
                              <button
                                type="button"
                                disabled={Boolean(busy)}
                                onClick={() =>
                                  submit(
                                    `/api/issues/${encodeURIComponent(issue.id)}/retry-generation`,
                                    {},
                                    "Lesson queued for generation again.",
                                  )
                                }
                              >
                                <RotateCcw size={14} /> Retry
                              </button>
                            ) : null}
                          </div>
                        </article>
                      ))}
                    </div>
                  ) : (
                    <div className="atelier-state-card">
                      <strong>No lessons yet.</strong>
                      <p>The archive will grow here as the stream develops.</p>
                    </div>
                  )}
                </section>
              </div>

              <aside className="stream-overview-side">
                <article className="stream-blueprint glass-panel">
                  <p className="atelier-eyebrow">Learning direction</p>
                  <h2>{newsletter.learnerGoal || "Build a connected understanding over time."}</h2>
                  <p>
                    Designed for {newsletter.learnerLevel}-level learning in about
                    {" "}{newsletter.lessonMinutes} minutes.
                  </p>
                </article>

                <article className="stream-sources glass-panel">
                  <div className="section-heading-row">
                    <div>
                      <p className="atelier-eyebrow">Source material</p>
                      <h2>{snapshot.sourceSummary?.healthy ?? 0} ready</h2>
                    </div>
                    <BookOpen size={17} />
                  </div>
                  <div>
                    {(snapshot.sourceCatalog ?? []).map((source) => (
                      <a href={source.canonicalUrl} target="_blank" rel="noreferrer" key={source.id}>
                        <i className={`source-health ${source.health}`} />
                        <span>
                          <strong>{source.displayName}</strong>
                          <small>{source.origin} · {source.kind || source.scope} · {source.health}</small>
                        </span>
                        <ExternalLink size={13} />
                      </a>
                    ))}
                  </div>
                  {(snapshot.sourceSummary?.needsAttention ?? 0) > 0 ? (
                    <p className="source-warning">
                      {snapshot.sourceSummary.needsAttention} source needs attention before the next lesson.
                    </p>
                  ) : null}
                </article>

                <article className="stream-rhythm glass-panel">
                  <p className="atelier-eyebrow">Rhythm and control</p>
                  <dl>
                    <div><dt><Clock3 size={14} /> Schedule</dt><dd>Daily at {newsletter.scheduleTime}</dd></div>
                    <div><dt><Mail size={14} /> Delivery</dt><dd>{newsletter.emailEnabled ? "Learnloom + email" : "Learnloom only"}</dd></div>
                    <div><dt><WandSparkles size={14} /> AI Exploration</dt><dd>{newsletter.aiExplorationEnabled ? "Clearly labeled" : "Off"}</dd></div>
                    <div><dt><Globe2 size={14} /> Personal site</dt><dd>{newsletter.siteVisible ? "Eligible to publish" : "Private"}</dd></div>
                  </dl>
                  <button
                    type="button"
                    disabled={Boolean(busy)}
                    onClick={() =>
                      submit(
                        `/api/newsletters/${encodeURIComponent(newsletter.id)}/content`,
                        { aiExplorationEnabled: !newsletter.aiExplorationEnabled },
                        newsletter.aiExplorationEnabled
                          ? "Future lessons will remain source-grounded only."
                          : "Clearly labeled AI Exploration is enabled for future lessons.",
                      )
                    }
                  >
                    {newsletter.aiExplorationEnabled ? "Disable AI Exploration" : "Enable AI Exploration"}
                  </button>
                  <button
                    type="button"
                    disabled={Boolean(busy)}
                    onClick={() =>
                      submit(
                        `/api/newsletters/${encodeURIComponent(newsletter.id)}/site`,
                        { visible: !newsletter.siteVisible },
                        newsletter.siteVisible
                          ? "This stream is private on your personal site."
                          : "This stream can now be published on your personal site.",
                      )
                    }
                  >
                    {newsletter.siteVisible ? "Keep this stream private" : "Allow this stream on my site"}
                  </button>
                </article>
              </aside>
            </div>
          </>
        ) : null}
      </section>
    </LearningShell>
  );
}

function humanize(value) {
  if (!value) return "Waiting";
  return value.charAt(0).toUpperCase() + value.slice(1).replaceAll("_", " ");
}

function noticeFromLocation() {
  const params = new URLSearchParams(window.location.search);
  return params.has("created")
    ? "Your learning stream is ready. Prepare the first lesson now or let it begin on schedule."
    : "";
}

function lessonHref(issueId) {
  return demoMode ? `/?demoIssue=${encodeURIComponent(issueId)}` : `/issues/${encodeURIComponent(issueId)}`;
}
