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
import CalmLoader from "./CalmLoader";
import LearningShell, { AtelierError, formatShortDate } from "./LearningShell";
import { apiJSON, demoMode } from "./api";
import { lessonState, syncLessonProgress } from "./learningState";
import { invalidateWorkspaceCache } from "./useWorkspace";
import { firstLessonPreparation } from "./preparation";
import type { Newsletter, NewsletterDetailResponse, SourceValidationResponse } from "./types";

export default function NewsletterDetail({ newsletterId }) {
  const [snapshot, setSnapshot] = useState<NewsletterDetailResponse | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState(noticeFromLocation());
  const [busy, setBusy] = useState("");
  const [replacement, setReplacement] = useState(null);
  const [publishReview, setPublishReview] = useState(null);
  const [selectedLessons, setSelectedLessons] = useState([]);

  const load = useCallback(async ({ signal }: { signal?: AbortSignal } = {}) => {
    setError("");
    try {
      const nextSnapshot = await apiJSON<NewsletterDetailResponse>(
        `/api/newsletters/${encodeURIComponent(newsletterId)}`,
        { signal },
      );
      syncLessonProgress(nextSnapshot.lessonProgress);
      setSnapshot(nextSnapshot);
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

  useEffect(() => {
    const createdJourney = new URLSearchParams(window.location.search).has("created");
    const currentNewsletter = snapshot?.newsletter;
    const waiting = snapshot?.issues?.some((issue) =>
      ["queued", "generating", "awaiting_approval"].includes(issue.status),
    );
    if (!createdJourney || !currentNewsletter || !waiting) return undefined;
    const recordExit = () => {
      void apiJSON(
        `/api/newsletters/${encodeURIComponent(currentNewsletter.id)}/preparation-wait-exited`,
        { method: "POST", keepalive: true },
      ).catch(() => {
        // Funnel measurement must never prevent a safe exit.
      });
    };
    window.addEventListener("pagehide", recordExit, { once: true });
    return () => window.removeEventListener("pagehide", recordExit);
  }, [snapshot]);

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

  function updateRhythm(
    newsletter: Newsletter,
    mode = newsletter.rhythmMode ?? "daily",
    selectedWeekdays = newsletter.selectedWeekdays ?? [1, 2, 3, 4, 5],
    autoThrottleEnabled = newsletter.autoThrottleEnabled ?? true,
  ) {
    return submit(
      `/api/newsletters/${encodeURIComponent(newsletter.id)}/rhythm`,
      {
        mode,
        selectedWeekdays,
        autoThrottleEnabled,
        unopenedLessonLimit: newsletter.unopenedLessonLimit ?? 3,
      },
      rhythmConfirmation(mode),
    );
  }

  async function broadenSourcesAndRetry(issue) {
    if (!snapshot?.newsletter) return;
    const current = snapshot.newsletter;
    setBusy(`broaden:${issue.id}`);
    setError("");
    try {
      await apiJSON(`/api/newsletters/${encodeURIComponent(current.id)}`, {
        method: "PUT",
        body: {
          name: current.name,
          topic: current.topic,
          learnerLevel: current.learnerLevel,
          learnerGoal: current.learnerGoal,
          lessonMinutes: current.lessonMinutes,
          sourceMode: "hybrid",
          sourceReviewMode: current.sourceReviewMode,
          sources: current.sources ?? [],
          scheduleTime: current.scheduleTime,
          timeZone: current.timeZone,
          active: current.active,
          emailEnabled: current.emailEnabled,
          aiExplorationEnabled: current.aiExplorationEnabled,
          siteVisible: current.siteVisible,
        },
      });
      await apiJSON(`/api/issues/${encodeURIComponent(issue.id)}/retry-generation`, {
        method: "POST",
        body: {},
      });
      invalidateWorkspaceCache(newsletterId);
      await load();
      setNotice("Learnloom will now keep your sources and discover additional evidence around them.");
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  async function replaceSource(event) {
    event.preventDefault();
    if (!snapshot?.newsletter || !replacement) return;
    setBusy(`replace:${replacement.id}`);
    setError("");
    try {
      const validation = await apiJSON<SourceValidationResponse>("/api/sources/validate", {
        method: "POST",
        body: {
          sources: [{
            name: replacement.name,
            url: replacement.url,
            limit: 8,
          }],
        },
      });
      if (validation.sources?.[0]?.status !== "ready") {
        throw new Error(validation.sources?.[0]?.message ?? "The replacement source could not be read.");
      }
      await apiJSON(
        `/api/newsletters/${encodeURIComponent(snapshot.newsletter.id)}/sources/${encodeURIComponent(replacement.id)}/replace`,
        {
          method: "POST",
          body: { name: replacement.name, url: replacement.url, limit: 8 },
        },
      );
      invalidateWorkspaceCache(newsletterId);
      setReplacement(null);
      await load();
      setNotice("Future lessons will use the replacement source. Earlier lessons keep their original evidence.");
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  async function changePublication(issue, state) {
    if (state === "published") {
      setPublishReview(issue);
      return;
    }
    await submit(
      `/api/issues/${encodeURIComponent(issue.id)}/publication`,
      { state, audienceConfirmed: false },
      state === "private"
        ? "Lesson is private and removed from publishing review."
        : "Lesson unpublished and kept as a draft.",
    );
  }

  async function confirmPublish() {
    if (!publishReview) return;
    await submit(
      `/api/issues/${encodeURIComponent(publishReview.id)}/publication`,
      { state: "published", audienceConfirmed: true },
      "Lesson published. Anyone with access to your public site can read it.",
    );
    setPublishReview(null);
  }

  async function applyBulkPublication(state) {
    if (!selectedLessons.length) return;
    if (state === "published" && !window.confirm(`Publish ${selectedLessons.length} selected lesson${selectedLessons.length === 1 ? "" : "s"}? Anyone can read them whenever your site and stream are public.`)) return;
    await submit(
      `/api/newsletters/${encodeURIComponent(newsletter.id)}/bulk-publication`,
      { issueIds: selectedLessons, state, audienceConfirmed: state === "published" },
      `${selectedLessons.length} lesson${selectedLessons.length === 1 ? "" : "s"} changed to ${state}.`,
    );
    setSelectedLessons([]);
  }

  if (!snapshot && !error) {
    return <CalmLoader label="Opening this learning stream…" detail="Bringing your stream into focus." />;
  }

  const newsletter = snapshot?.newsletter;
  const issues = snapshot?.issues ?? [];
  const latest = issues[0];
  const generated = issues.filter((issue) => issue.status === "generated");
  const latestGenerated = latestGeneratedIssue(issues);
  const preparing = issues.find((issue) => ["queued", "generating"].includes(issue.status));
  const awaitingApproval = issues.find((issue) => issue.status === "awaiting_approval");
  const latestPresentation = latestGenerated
    ? lessonPresentation(lessonState(latestGenerated.id))
    : null;
  const emptyPresentation = streamEmptyPresentation(latest);

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
                  <span>{newsletterPathSummary(newsletter)}</span>
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
                  disabled={Boolean(busy) || Boolean(preparing) || Boolean(awaitingApproval)}
                  onClick={() =>
                    submit(
                      `/api/newsletters/${encodeURIComponent(newsletter.id)}/run`,
                      {},
                      `${firstLessonPreparation.shortLabel}. You can leave this page.`,
                    )
                  }
                >
                  <RefreshCw className={busy.endsWith("/run") ? "spin" : ""} size={15} />
                  {awaitingApproval ? "Waiting for source approval" : preparing ? "Preparing lesson…" : generated.length ? "Prepare a lesson now" : "Prepare first lesson"}
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
                    to your learning history. {firstLessonPreparation.explanation}
                  </p>
                </div>
              </section>
            ) : null}

            {awaitingApproval ? (
              <section className="stream-source-approval glass-panel">
                <span className="atelier-icon"><Check size={18} /></span>
                <div>
                  <p className="atelier-eyebrow">Your source portfolio is ready</p>
                  <h2>Review the evidence mix before lesson one is written.</h2>
                  <p>Prefer, block, or replace any source below. Changes rebuild this portfolio; earlier lessons, if any, keep their frozen evidence.</p>
                </div>
                <button
                  className="atelier-primary"
                  type="button"
                  disabled={Boolean(busy)}
                  onClick={() => submit(
                    `/api/newsletters/${encodeURIComponent(newsletter.id)}/approve-source-portfolio`,
                    {},
                    `${firstLessonPreparation.shortLabel}. Your approved sources are now being turned into a lesson.`,
                  )}
                >Approve sources & begin</button>
              </section>
            ) : null}

            <div className="stream-overview-grid">
              <div className="stream-overview-main">
                {latestGenerated ? (
                  <article className="latest-lesson-card glass-panel">
                    <div className="latest-lesson-heading">
                      <span className="atelier-chip"><BookOpen size={13} /> Latest lesson</span>
                      <span><Clock3 size={13} />{newsletter.lessonMinutes} min</span>
                    </div>
                    <h2>{latestGenerated.title}</h2>
                    <p>{latestPresentation?.description}</p>
                    <div className="latest-lesson-footer">
                      <span>
                        {formatShortDate(latestGenerated.createdAt)} · {latestPresentation?.status}
                      </span>
                      <a className="atelier-primary" href={lessonHref(latestGenerated.id)}>
                        {latestPresentation?.cta}
                        <ArrowRight size={15} />
                      </a>
                    </div>
                  </article>
                ) : (
                  <article className="latest-lesson-card glass-panel">
                    <span className="atelier-icon"><Sparkles size={18} /></span>
                    <h2>{emptyPresentation.title}</h2>
                    <p>{emptyPresentation.description}</p>
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
                  {issues.some((issue) => issue.status === "generated") ? (
                    <div className="bulk-publication-bar">
                      <button
                        type="button"
                        onClick={() => setSelectedLessons(
                          selectedLessons.length
                            ? []
                            : issues.filter((issue) => issue.status === "generated").map((issue) => issue.id),
                        )}
                      >{selectedLessons.length ? "Clear selection" : "Select all generated"}</button>
                      <span>{selectedLessons.length} selected</span>
                      <button type="button" disabled={!selectedLessons.length || Boolean(busy)} onClick={() => applyBulkPublication("private")}>Private</button>
                      <button type="button" disabled={!selectedLessons.length || Boolean(busy)} onClick={() => applyBulkPublication("draft")}>Draft</button>
                      <button type="button" disabled={!selectedLessons.length || Boolean(busy)} onClick={() => applyBulkPublication("published")}>Publish</button>
                    </div>
                  ) : null}
                  {issues.length ? (
                    <div className="stream-lesson-list">
                      {issues.map((issue, index) => {
                        const presentation = lessonPresentation(lessonState(issue.id));
                        const status = issue.status === "generated" && presentation.status !== "Unread"
                          ? presentation.status
                          : issue.status === "deferred"
                            ? "Waiting for stronger evidence"
                            : humanize(issue.status);
                        return (
                          <article className="stream-lesson-row glass-panel" key={issue.id}>
                            {issue.status === "generated" ? (
                              <label className="lesson-selection">
                                <input
                                  type="checkbox"
                                  checked={selectedLessons.includes(issue.id)}
                                  onChange={(event) => setSelectedLessons((current) => event.target.checked
                                    ? [...current, issue.id]
                                    : current.filter((id) => id !== issue.id))}
                                />
                                <span className="sr-only">Select {issue.title}</span>
                              </label>
                            ) : <span className="stream-lesson-index">{String(index + 1).padStart(2, "0")}</span>}
                            <div>
                              <span>{formatShortDate(issue.createdAt)} · {status}</span>
                              <h3>{issue.title ?? issueFallbackTitle(issue.status)}</h3>
                              {issue.status === "generated" ? (
                                <p className={`lesson-audience ${issue.publicationState ?? "draft"}`}>
                                  {publicationAudience(issue, snapshot.site, newsletter)}
                                </p>
                              ) : null}
                              {issue.error ? <p className="row-error">{issue.error}</p> : null}
                            </div>
                            <div className="stream-lesson-actions">
                              {issue.status === "generated" ? (
                                <>
                                  <a href={lessonHref(issue.id)}>
                                    {presentation.historyCta} <ArrowRight size={14} />
                                  </a>
                                  <button
                                    type="button"
                                    disabled={Boolean(busy)}
                                    onClick={() => changePublication(
                                      issue,
                                      issue.publicationState === "published" ? "draft" : "published",
                                    )}
                                  >
                                    {issue.publicationState === "published" ? "Unpublish" : "Review & publish"}
                                  </button>
                                  {issue.publicationState === "draft" ? (
                                    <button type="button" disabled={Boolean(busy)} onClick={() => changePublication(issue, "private")}>
                                      Keep private
                                    </button>
                                  ) : null}
                                </>
                              ) : null}
                              {recoveryAction(issue, newsletter.sourceMode)?.kind === "retry" ? (
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
                              {recoveryAction(issue, newsletter.sourceMode)?.kind === "broaden_sources" ? (
                                <button
                                  type="button"
                                  disabled={Boolean(busy)}
                                  onClick={() => broadenSourcesAndRetry(issue)}
                                >
                                  <WandSparkles size={14} /> Broaden sources and retry
                                </button>
                              ) : null}
                              {recoveryAction(issue, newsletter.sourceMode)?.kind === "contact_support" ? (
                                <a href={supportHref(issue)}>
                                  Contact support <ArrowRight size={14} />
                                </a>
                              ) : null}
                            </div>
                          </article>
                        );
                      })}
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

                {snapshot.curriculum ? (
                  <article className="stream-curriculum glass-panel">
                    <p className="atelier-eyebrow">Your capability path</p>
                    <h2>{snapshot.curriculum.outcome || newsletter.learnerGoal}</h2>
                    <p className="curriculum-recall-summary">
                      {snapshot.curriculum.recall?.summary || "Your first capability milestone will appear after a completed lesson."}
                    </p>
                    {(snapshot.curriculum.milestones ?? []).length ? (
                      <div className="capability-milestones">
                        {snapshot.curriculum.milestones.slice(0, 8).map((milestone) => (
                          <span key={milestone.key}>
                            <strong>{milestone.statement}</strong>
                            <small>{capabilityEvidence(milestone)}</small>
                          </span>
                        ))}
                      </div>
                    ) : (
                      <p className="curriculum-empty">Complete the first lesson to establish a capability milestone.</p>
                    )}
                    {(snapshot.curriculum.currentGaps ?? []).length ? (
                      <section className="curriculum-gaps">
                        <p className="atelier-eyebrow">Current gaps</p>
                        {(snapshot.curriculum.currentGaps ?? []).slice(0, 5).map((gap) => (
                          <p key={gap.key}><strong>{gap.label}</strong><span>{gap.reason}</span></p>
                        ))}
                      </section>
                    ) : null}
                    {snapshot.curriculum.suggestedNextConcepts?.length ? (
                      <section>
                        <p className="atelier-eyebrow">Likely next directions</p>
                        <p>{snapshot.curriculum.suggestedNextConcepts.join(" · ")}</p>
                      </section>
                    ) : null}
                    {snapshot.curriculum.timeline?.length ? (
                      <section className="curriculum-timeline">
                        <p className="atelier-eyebrow">Capability timeline</p>
                        {snapshot.curriculum.timeline.slice(0, 5).map((entry) => (
                          <a href={`/issues/${encodeURIComponent(entry.issueId)}`} key={entry.issueId}>
                            <strong>{entry.title}</strong>
                            <small>{entry.concepts.join(" · ")}</small>
                          </a>
                        ))}
                      </section>
                    ) : null}
                  </article>
                ) : null}

                <article className="stream-sources glass-panel">
                  <div className="section-heading-row">
                    <div>
                      <p className="atelier-eyebrow">Source material</p>
                      <h2>{snapshot.sourceSummary?.healthy ?? 0} ready</h2>
                    </div>
                    <BookOpen size={17} />
                  </div>
                  <div className="source-control-list">
                    {(snapshot.sourceCatalog ?? []).map((source) => (
                      <article className="source-control-row" key={source.id}>
                        <i className={`source-health ${source.health}`} />
                        <span>
                          <a href={source.canonicalUrl} target="_blank" rel="noreferrer">
                            <strong>{source.displayName}</strong><ExternalLink size={12} />
                          </a>
                          <small>
                            {sourceRoleLabel(source.role, source.origin)} · {source.kind || source.scope} · {source.health}
                          </small>
                          {source.discoveryReason ? <em>{source.discoveryReason}</em> : null}
                        </span>
                        <div className="source-control-actions">
                          {source.preference === "blocked" ? (
                            <button
                              type="button"
                              disabled={Boolean(busy)}
                              onClick={() => submit(
                                `/api/newsletters/${encodeURIComponent(newsletter.id)}/sources/${encodeURIComponent(source.id)}/preference`,
                                { preference: "neutral" },
                                "Source allowed for future lessons.",
                              )}
                            >Allow</button>
                          ) : (
                            <>
                              <button
                                type="button"
                                disabled={Boolean(busy)}
                                onClick={() => submit(
                                  `/api/newsletters/${encodeURIComponent(newsletter.id)}/sources/${encodeURIComponent(source.id)}/preference`,
                                  { preference: source.preference === "preferred" ? "neutral" : "preferred" },
                                  source.preference === "preferred"
                                    ? "Source returned to normal priority."
                                    : "Source preferred for future lessons.",
                                )}
                              >{source.preference === "preferred" ? "Preferred" : "Prefer"}</button>
                              <button
                                type="button"
                                disabled={Boolean(busy)}
                                onClick={() => submit(
                                  `/api/newsletters/${encodeURIComponent(newsletter.id)}/sources/${encodeURIComponent(source.id)}/preference`,
                                  { preference: "blocked" },
                                  "Source blocked from future lessons. Earlier lessons are unchanged.",
                                )}
                              >Block</button>
                              {source.origin === "provided" ? (
                                <button
                                  type="button"
                                  disabled={Boolean(busy)}
                                  onClick={() => setReplacement({
                                    id: source.id,
                                    name: source.displayName,
                                    url: "",
                                  })}
                                >Replace</button>
                              ) : null}
                            </>
                          )}
                        </div>
                        {replacement?.id === source.id ? (
                          <form className="source-replacement" onSubmit={replaceSource}>
                            <label>
                              <span>Replacement name</span>
                              <input
                                maxLength={120}
                                value={replacement.name}
                                onChange={(event) => setReplacement({ ...replacement, name: event.target.value })}
                              />
                            </label>
                            <label>
                              <span>Replacement URL</span>
                              <input
                                type="url"
                                required
                                placeholder="https://…"
                                value={replacement.url}
                                onChange={(event) => setReplacement({ ...replacement, url: event.target.value })}
                              />
                            </label>
                            <div>
                              <button type="button" onClick={() => setReplacement(null)}>Cancel</button>
                              <button className="atelier-primary" disabled={Boolean(busy)} type="submit">
                                {busy === `replace:${source.id}` ? "Checking…" : "Use replacement"}
                              </button>
                            </div>
                          </form>
                        ) : null}
                      </article>
                    ))}
                  </div>
                  <section className="source-policy-controls">
                    <p className="atelier-eyebrow">Source policy</p>
                    <div>
                      <button
                        className={newsletter.sourceMode === "provided" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => submit(
                          `/api/newsletters/${encodeURIComponent(newsletter.id)}/source-mode`,
                          { mode: "provided" },
                          "Future lessons will use only sources you chose.",
                        )}
                      >Use only mine</button>
                      <button
                        className={newsletter.sourceMode === "hybrid" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => submit(
                          `/api/newsletters/${encodeURIComponent(newsletter.id)}/source-mode`,
                          { mode: "hybrid" },
                          "Learnloom will keep your sources and fill evidence gaps.",
                        )}
                      >Fill gaps</button>
                      <button
                        className={newsletter.sourceMode === "discovered" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => submit(
                          `/api/newsletters/${encodeURIComponent(newsletter.id)}/source-mode`,
                          { mode: "discovered" },
                          "Learnloom will establish the source portfolio for future lessons.",
                        )}
                      >Find for me</button>
                    </div>
                    <div className="source-review-controls">
                      <button
                        className={newsletter.sourceReviewMode !== "review" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => submit(
                          `/api/newsletters/${encodeURIComponent(newsletter.id)}/source-review-mode`,
                          { mode: "auto" },
                          "Future source portfolios can proceed automatically.",
                        )}
                      >Begin automatically</button>
                      <button
                        className={newsletter.sourceReviewMode === "review" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => submit(
                          `/api/newsletters/${encodeURIComponent(newsletter.id)}/source-review-mode`,
                          { mode: "review" },
                          "Learnloom will pause after preparing future source portfolios.",
                        )}
                      >Ask before writing</button>
                    </div>
                  </section>
                  {(snapshot.sourceSummary?.needsAttention ?? 0) > 0 ? (
                    <p className="source-warning">
                      {snapshot.sourceSummary.needsAttention} source needs attention before the next lesson.
                    </p>
                  ) : null}
                </article>

                <article className="stream-rhythm glass-panel">
                  <p className="atelier-eyebrow">Rhythm and control</p>
                  <dl>
                    <div><dt><Clock3 size={14} /> Schedule</dt><dd>{rhythmScheduleLabel(newsletter)}</dd></div>
                    <div><dt><Mail size={14} /> Delivery</dt><dd>{newsletter.emailEnabled ? "Learnloom + email" : "Learnloom only"}</dd></div>
                    <div><dt><WandSparkles size={14} /> AI Exploration</dt><dd>{newsletter.aiExplorationEnabled ? "Clearly labeled" : "Off"}</dd></div>
                    <div><dt><Globe2 size={14} /> Personal site</dt><dd>{newsletter.siteVisible ? "Eligible to publish" : "Private"}</dd></div>
                  </dl>
                  <section className="rhythm-controls" aria-label="Learning rhythm">
                    <p>Prepare the next lesson</p>
                    <div className="rhythm-mode-grid">
                      {([
                        ["evidence_led", "When evidence changes"],
                        ["daily", "Every day"],
                        ["selected_weekdays", "Selected days"],
                        ["weekly_synthesis", "Weekly synthesis"],
                      ] as const).map(([mode, label]) => (
                        <button
                          className={newsletter.rhythmMode === mode ? "selected" : ""}
                          type="button"
                          disabled={Boolean(busy)}
                          key={mode}
                          onClick={() => updateRhythm(newsletter, mode)}
                        >{label}</button>
                      ))}
                    </div>
                    {["selected_weekdays", "weekly_synthesis"].includes(newsletter.rhythmMode ?? "") ? (
                      <div className="rhythm-weekdays" aria-label="Scheduled weekdays">
                        {weekdayChoices.map(({ value, label }) => (
                          <button
                            className={(newsletter.selectedWeekdays ?? []).includes(value) ? "selected" : ""}
                            type="button"
                            disabled={Boolean(busy)}
                            key={value}
                            aria-label={label}
                            onClick={() => updateRhythm(
                              newsletter,
                              newsletter.rhythmMode,
                              nextSelectedWeekdays(
                                newsletter.selectedWeekdays ?? [1],
                                value,
                                newsletter.rhythmMode === "weekly_synthesis",
                              ),
                            )}
                          >{label.slice(0, 1)}</button>
                        ))}
                      </div>
                    ) : null}
                    <label className="rhythm-throttle-control">
                      <input
                        type="checkbox"
                        checked={newsletter.autoThrottleEnabled ?? true}
                        disabled={Boolean(busy)}
                        onChange={(event) => updateRhythm(
                          newsletter,
                          newsletter.rhythmMode,
                          newsletter.selectedWeekdays,
                          event.target.checked,
                        )}
                      />
                      <span>Slow down automatically when {newsletter.unopenedLessonLimit ?? 3} lessons are waiting</span>
                    </label>
                    {newsletter.rhythmReason ? (
                      <p className={newsletter.rhythmThrottledAt ? "rhythm-reason throttled" : "rhythm-reason"}>
                        {newsletter.rhythmReason}
                      </p>
                    ) : null}
                  </section>
                  <section className="publication-default-controls" aria-label="New lesson publishing default">
                    <p>New completed lessons</p>
                    <div>
                      <button
                        className={newsletter.lessonPublicationDefault !== "published" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => submit(
                          `/api/newsletters/${encodeURIComponent(newsletter.id)}/publication-default`,
                          { state: "draft", audienceConfirmed: false },
                          "New lessons will wait as drafts for your review.",
                        )}
                      >Keep as drafts <small>Recommended</small></button>
                      <button
                        className={newsletter.lessonPublicationDefault === "published" ? "selected" : ""}
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => {
                          if (window.confirm("Automatically publish every future completed lesson? Anyone can read it whenever your site and this stream are public.")) {
                            void submit(
                              `/api/newsletters/${encodeURIComponent(newsletter.id)}/publication-default`,
                              { state: "published", audienceConfirmed: true },
                              "Future completed lessons will publish automatically while the site and stream remain public.",
                            );
                          }
                        }}
                      >Publish automatically</button>
                    </div>
                  </section>
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
      {publishReview ? (
        <div className="publish-review-backdrop" role="presentation" onClick={() => setPublishReview(null)}>
          <section className="publish-review-modal" role="dialog" aria-modal="true" aria-labelledby="publish-review-title" onClick={(event) => event.stopPropagation()}>
            <p className="atelier-eyebrow">First-publish review</p>
            <h2 id="publish-review-title">Who will be able to read this lesson?</h2>
            <div className="publish-audience-preview">
              <strong>{publishReview.title}</strong>
              <p>{publicationAudience({ ...publishReview, publicationState: "published" }, snapshot.site, newsletter)}</p>
            </div>
            <ul>
              <li>{snapshot.site?.visibility === "public" ? "Your learning site is public." : "Your site is private; publish will be saved but the lesson stays inaccessible until you make the site public."}</li>
              <li>{newsletter.siteVisible ? "This stream is allowed on the site." : "This stream is private; publish will be saved but the lesson stays inaccessible until you allow the stream."}</li>
              <li>Search discovery is {snapshot.site?.searchIndexing ? "enabled" : "off"}; this does not change link access.</li>
            </ul>
            <a href={`/issues/${encodeURIComponent(publishReview.id)}`} target="_blank" rel="noreferrer">Preview the exact lesson <ExternalLink size={13} /></a>
            <div>
              <button type="button" onClick={() => setPublishReview(null)}>Cancel</button>
              <button className="atelier-primary" type="button" disabled={Boolean(busy)} onClick={confirmPublish}>Confirm publish</button>
            </div>
          </section>
        </div>
      ) : null}
    </LearningShell>
  );
}

const weekdayChoices = [
  { value: 1, label: "Monday" },
  { value: 2, label: "Tuesday" },
  { value: 3, label: "Wednesday" },
  { value: 4, label: "Thursday" },
  { value: 5, label: "Friday" },
  { value: 6, label: "Saturday" },
  { value: 7, label: "Sunday" },
];

export function nextSelectedWeekdays(current: number[], weekday: number, single = false) {
  if (single) return [weekday];
  const selected = new Set(current);
  if (selected.has(weekday) && selected.size > 1) selected.delete(weekday);
  else selected.add(weekday);
  return [...selected].sort((left, right) => left - right);
}

export function rhythmScheduleLabel(newsletter: Newsletter) {
  const time = newsletter.scheduleTime ?? "08:00";
  const weekdays = newsletter.selectedWeekdays ?? [1, 2, 3, 4, 5];
  const mode = newsletter.effectiveRhythmMode ?? newsletter.rhythmMode ?? "daily";
  if (newsletter.rhythmThrottledAt) return `Slowed to weekly · ${time}`;
  if (mode === "evidence_led") return `Evidence-led check · ${time}`;
  if (mode === "weekly_synthesis") {
    return `${weekdayChoices.find(({ value }) => value === weekdays[0])?.label ?? "Weekly"} · ${time}`;
  }
  if (mode === "selected_weekdays") return `${weekdays.length} days/week · ${time}`;
  return `Daily · ${time}`;
}

function rhythmConfirmation(mode: string) {
  switch (mode) {
    case "evidence_led": return "Learnloom will only write when refreshed sources add something worthwhile.";
    case "selected_weekdays": return "Your selected learning days are saved.";
    case "weekly_synthesis": return "Learnloom will prepare one connecting synthesis each week.";
    default: return "Your daily learning rhythm is saved.";
  }
}

function humanize(value) {
  if (!value) return "Waiting";
  return value.charAt(0).toUpperCase() + value.slice(1).replaceAll("_", " ");
}

export function sourceRoleLabel(role, origin = "discovered") {
  if (origin === "provided") return "Chosen by you";
  switch (role) {
    case "official_primary": return "Official or primary";
    case "research": return "Research";
    case "practitioner_explainer": return "Practitioner explanation";
    case "reporting_context": return "Reporting and context";
    case "counterweight": return "Counterweight";
    default: return "Discovered source";
  }
}

export function capabilityEvidence(milestone: {
  completedCount: number;
  reviewAttemptCount: number;
  confidenceScore: number;
}) {
  if (milestone.reviewAttemptCount > 0) {
    return `${milestone.reviewAttemptCount} retrieval ${milestone.reviewAttemptCount === 1 ? "attempt" : "attempts"} · ${recallConfidenceLabel(milestone.confidenceScore)}`;
  }
  return `${milestone.completedCount} completed ${milestone.completedCount === 1 ? "lesson" : "lessons"} · retrieval not attempted yet`;
}

export function newsletterPathSummary(newsletter: Newsletter) {
  const capabilities = newsletter.capabilityCount ?? 0;
  const recalled = newsletter.recalledCapabilityCount ?? 0;
  if (recalled > 0) return `${recalled} recalled capabilities · ${capabilities} established`;
  if (capabilities > 0) return `${capabilities} capabilities established`;
  return "Your first capability milestone is ahead";
}

export function publicationAudience(issue, site, newsletter) {
  const state = issue.publicationState ?? "draft";
  if (state === "private") return "Private · only you can read this lesson";
  if (state === "draft") return "Draft · only you can read it until you publish";
  if (site?.visibility !== "public") return "Published, but not visible · your site is private";
  if (!newsletter.siteVisible) return "Published, but not visible · this stream is private";
  return site.searchIndexing
    ? "Public · anyone can read it and search engines may index it"
    : "Public by link · anyone can read it; search discovery is off";
}

function recallConfidenceLabel(score: number) {
  if (score >= 75) return "solid recall";
  if (score >= 50) return "developing recall";
  return "recall needs reinforcement";
}

export function lessonPresentation(state: { progress: number; completed: boolean }) {
  if (state.completed) {
    return {
      status: "Completed",
      cta: "Review lesson",
      historyCta: "Review",
      description: "You completed this lesson. Revisit it whenever you want to refresh the idea.",
    };
  }
  if (state.progress > 0) {
    return {
      status: `${Math.round(state.progress)}% read`,
      cta: "Continue lesson",
      historyCta: "Continue",
      description: "Continue this lesson from where you left off.",
    };
  }
  return {
    status: "Unread",
    cta: "Open lesson",
    historyCta: "Read",
    description: "Begin a focused lesson grounded in your source library.",
  };
}

export function streamEmptyPresentation(issue) {
  switch (issue?.status) {
    case "queued":
    case "generating":
      return {
        title: "Your first lesson is being prepared.",
        description: firstLessonPreparation.explanation,
      };
    case "deferred":
      return {
        title: "Waiting for stronger evidence.",
        description: issue.error ?? "Learnloom will check again at the next scheduled time.",
      };
    case "awaiting_approval":
      return {
        title: "Your sources are ready to review.",
        description: "Check the evidence portfolio below, make any changes, then approve it to begin lesson generation.",
      };
    case "failed":
      return {
        title: "Your first lesson needs attention.",
        description: issue.error ?? "Use the recovery action below to continue.",
      };
    default:
      return {
        title: "Your first lesson hasn’t been requested yet.",
        description: `Prepare it when you are ready. ${firstLessonPreparation.explanation}`,
      };
  }
}

export function latestGeneratedIssue(issues) {
  return issues.find((issue) => issue.status === "generated");
}

export function recoveryAction(issue, sourceMode) {
  if (issue?.status !== "failed") return null;
  if (issue.failureCategory === "user_actionable" && sourceMode === "provided") {
    return { kind: "broaden_sources", label: "Broaden sources and retry" };
  }
  if (issue.failureRetryable !== false) {
    return { kind: "retry", label: "Retry lesson" };
  }
  return { kind: "contact_support", label: "Contact support" };
}

function issueFallbackTitle(status) {
  if (status === "awaiting_approval") return "Sources ready for approval";
  if (status === "deferred") return "No lesson prepared";
  if (status === "failed") return "Lesson needs attention";
  if (status === "queued" || status === "generating") return "Lesson being prepared";
  return "Lesson update";
}

function supportHref(issue) {
  const subject = encodeURIComponent(`Lesson preparation help · ${issue.incidentId ?? issue.id}`);
  return `mailto:support@learnloom.blog?subject=${subject}`;
}

function noticeFromLocation() {
  const params = new URLSearchParams(window.location.search);
  return params.has("created")
    ? `Your stream is ready. ${firstLessonPreparation.shortLabel}; you can safely leave while it is prepared.`
    : "";
}

function lessonHref(issueId) {
  return demoMode ? `/?demoIssue=${encodeURIComponent(issueId)}` : `/issues/${encodeURIComponent(issueId)}`;
}
