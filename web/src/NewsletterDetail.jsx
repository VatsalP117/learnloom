import {
  AlertCircle,
  ArrowRight,
  BookOpen,
  BrainCircuit,
  Check,
  CircleDashed,
  Clock3,
  ExternalLink,
  FileCheck2,
  FlaskConical,
  Lightbulb,
  Globe2,
  Mail,
  Pause,
  Play,
  RefreshCw,
  RotateCcw,
  Settings2,
  Sparkles,
  WandSparkles,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { ErrorState, Footer, Sidebar, Topbar } from "./App.jsx";
import { apiJSON, demoMode } from "./api.js";

const pipelineSteps = [
  { label: "Curation", icon: BookOpen },
  { label: "Enrichment", icon: Sparkles },
  { label: "Continuity", icon: BrainCircuit },
  { label: "Evidence Review", icon: FileCheck2 },
  { label: "Lesson Design", icon: Lightbulb },
  { label: "Quality Gate", icon: Check },
];

function NewsletterDetail({ newsletterId }) {
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState(noticeFromLocation());
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const load = useCallback(async () => {
    const body = await apiJSON(
      `/api/newsletters/${encodeURIComponent(newsletterId)}`,
    );
    setSnapshot(body);
  }, [newsletterId]);

  useEffect(() => {
    const controller = new AbortController();
    apiJSON(`/api/newsletters/${encodeURIComponent(newsletterId)}`, {
      signal: controller.signal,
    })
      .then(setSnapshot)
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, [newsletterId]);

  useEffect(() => {
    if (!snapshot?.newsletter.name) return;
    document.title = `${snapshot.newsletter.name} · Learnloom`;
    return () => {
      document.title = "Learnloom · Knowledge Dossiers";
    };
  }, [snapshot?.newsletter.name]);

  async function submit(action, fields, successMessage) {
    setBusy(action);
    setError("");
    try {
      await apiJSON(action, {
        method: "POST",
        body: fields,
      });
      await load();
      setNotice(successMessage);
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  const newsletters = snapshot?.newsletters ?? [];
  const newsletter = snapshot?.newsletter;
  const issues = snapshot?.issues ?? [];
  const latestIssue = issues[0] ?? null;

  return (
    <div className="app">
      <Topbar onMenu={() => setSidebarOpen(true)} />
      <div className="shell">
        <Sidebar
          newsletters={newsletters}
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
          currentNewsletterId={newsletterId}
        />
        <main className="content detail-content">
          <div className="content-inner detail-inner">
            {!snapshot && !error ? <DetailSkeleton /> : null}
            {error ? <ErrorState message={error} /> : null}
            {newsletter ? (
              <>
                <DetailHeader
                  newsletter={newsletter}
                  hasIssues={issues.length > 0}
                  busy={busy}
                  onRun={() =>
                    submit(
                      `/api/newsletters/${encodeURIComponent(newsletter.id)}/run`,
                      {},
                      "Issue queued. The worker will pick it up shortly.",
                    )
                  }
                  onToggle={() =>
                    submit(
                      `/api/newsletters/${encodeURIComponent(newsletter.id)}/active`,
                      { active: !newsletter.active },
                      newsletter.active
                        ? "Newsletter paused."
                        : "Newsletter resumed.",
                    )
                  }
                />
                {notice ? (
                  <div className="detail-notice" role="status">
                    <Check size={16} />
                    <span>{notice}</span>
                    <button onClick={() => setNotice("")} aria-label="Dismiss notification">
                      ×
                    </button>
                  </div>
                ) : null}
                <section className="detail-grid">
                  <PipelineCard
                    latestIssue={latestIssue}
                    active={newsletter.active}
                  />
                  <SourceCatalogCard
                    mode={newsletter.sourceMode}
                    summary={snapshot.sourceSummary}
                    sources={snapshot.sourceCatalog ?? []}
                  />
                  <BlueprintCard newsletter={newsletter} />
                  <DeliveryCard
                    newsletter={newsletter}
                    resendConfigured={snapshot.resendConfigured}
                    busy={busy}
                    onSave={(settings) =>
                      submit(
                        `/api/newsletters/${encodeURIComponent(newsletter.id)}/delivery`,
                        settings,
                        "Email delivery settings saved.",
                      )
                    }
                    onSaveContent={(settings) =>
                      submit(
                        `/api/newsletters/${encodeURIComponent(newsletter.id)}/content`,
                        settings,
                        "Content settings saved for future Issues.",
                      )
                    }
                    onSaveSiteVisibility={(visible) =>
                      submit(
                        `/api/newsletters/${encodeURIComponent(newsletter.id)}/site`,
                        { visible },
                        visible
                          ? "This learning stream will appear on your public site."
                          : "This learning stream is hidden from your public site.",
                      )
                    }
                  />
                  <IssueHistory
                    issues={issues}
                    busy={busy}
                    onRetryGeneration={(issue) =>
                      submit(
                        `/api/issues/${encodeURIComponent(issue.id)}/retry-generation`,
                        {},
                        "Issue queued for generation again.",
                      )
                    }
                    onRetry={(issue) =>
                      submit(
                        `/api/issues/${encodeURIComponent(issue.id)}/retry-delivery`,
                        {},
                        "Email delivery queued for another attempt.",
                      )
                    }
                    onPublicationChange={(issue) =>
                      submit(
                        `/api/issues/${encodeURIComponent(issue.id)}/publication`,
                        {
                          state:
                            issue.publicationState === "published"
                              ? "hidden"
                              : "published",
                        },
                        issue.publicationState === "published"
                          ? "Dossier hidden from your public site."
                          : "Dossier published to your site.",
                      )
                    }
                  />
                </section>
              </>
            ) : null}
          </div>
          <Footer />
        </main>
      </div>
    </div>
  );
}

function DetailHeader({ newsletter, hasIssues, busy, onRun, onToggle }) {
  return (
    <section className="detail-header">
      <div>
        <a className="back-link" href="/">Learning streams <span>/</span></a>
        <div className="detail-title-row">
          <h1>{newsletter.name}</h1>
          <span className={`detail-status ${newsletter.active ? "active" : ""}`}>
            <i />{newsletter.active ? "Active" : "Paused"}
          </span>
        </div>
        <p>{newsletter.topic}</p>
      </div>
      <div className="detail-actions">
        <button
          className="secondary-action"
          type="button"
          disabled={Boolean(busy)}
          onClick={onToggle}
        >
          {newsletter.active ? <Pause size={16} /> : <Play size={16} />}
          {newsletter.active ? "Pause" : "Resume"}
        </button>
        <button
          className="primary-action"
          type="button"
          disabled={Boolean(busy)}
          onClick={onRun}
        >
          <RefreshCw className={busy.endsWith("/run") ? "spin" : ""} size={17} />
          {hasIssues ? "Create a lesson now" : "Create your first lesson"}
        </button>
      </div>
    </section>
  );
}

function PipelineCard({ latestIssue, active }) {
  const completedSteps = pipelineProgress(latestIssue);
  const stateLabel = latestIssue
    ? humanize(latestIssue.status)
    : active
      ? "Awaiting first run"
      : "Paused";

  return (
    <article className="detail-card pipeline-card">
      <div className="detail-card-heading">
        <span>How your lesson is made</span>
        <div className="pipeline-current">
          <i className={latestIssue?.status ?? "idle"} />
          {stateLabel}
        </div>
      </div>
      <div className="pipeline-steps">
        {pipelineSteps.map(({ label, icon: Icon }, index) => {
          const complete = index < completedSteps;
          const current =
            latestIssue?.status === "generating" && index === completedSteps;
          return (
            <div
              className={`pipeline-step ${complete ? "complete" : ""} ${current ? "current" : ""}`}
              key={label}
            >
              <div className="step-track"><span /></div>
              <span>{label}</span>
              <Icon size={18} />
            </div>
          );
        })}
      </div>
      {latestIssue?.status === "failed" ? (
        <div className="pipeline-error">
          <AlertCircle size={15} />
          {latestIssue.error ?? "Generation failed. Queue a new run to retry."}
        </div>
      ) : null}
    </article>
  );
}

function SourceCatalogCard({ mode, summary, sources }) {
  const modeLabel = {
    discovered: "Find sources for me",
    provided: "Provided sources only",
    hybrid: "Provided + discovered",
  }[mode] ?? mode;

  return (
    <article className="detail-card source-catalog-card">
      <div className="detail-card-heading">
        <span>Your source library</span>
        <span className="blueprint-meta">{modeLabel}</span>
      </div>
      <div className="source-summary-row">
        <span>{summary?.provided ?? 0} provided</span>
        <span>{summary?.discovered ?? 0} discovered</span>
        <span>{summary?.healthy ?? 0} ready</span>
        {(summary?.needsAttention ?? 0) > 0
          ? <span className="needs-attention">{summary.needsAttention} need attention</span>
          : null}
      </div>
      {sources.length ? (
        <div className="source-catalog-list">
          {sources.map((source) => (
            <div className="source-catalog-item" key={source.id}>
              <span className={`source-health ${source.health}`} aria-label={source.health} />
              <div>
                <strong>{source.displayName}</strong>
                <small>
                  <span className={`source-origin ${source.origin}`}>{source.origin}</span>
                  {source.kind || source.scope}
                  {source.discoveryReason ? ` · ${source.discoveryReason}` : ""}
                  {source.state !== "active" ? ` · ${source.state}` : ""}
                </small>
                {source.error ? <em>{source.error}</em> : null}
              </div>
              <a href={source.canonicalUrl} target="_blank" rel="noreferrer" aria-label={`Open ${source.displayName}`}>
                <ExternalLink size={14} />
              </a>
            </div>
          ))}
        </div>
      ) : (
        <p className="source-catalog-empty">
          Your sources will appear here after the first lesson begins.
        </p>
      )}
    </article>
  );
}

function BlueprintCard({ newsletter }) {
  const sourceHosts = newsletter.sources
    .slice(0, 3)
    .map((source) => {
      try {
        return new URL(source.url).hostname.replace(/^www\./, "");
      } catch {
        return source.name;
      }
    });

  return (
    <article className="detail-card blueprint-card">
      <div className="detail-card-heading">
        <span>Learning Blueprint</span>
        <span className="blueprint-meta">{newsletter.lessonMinutes} min lesson</span>
      </div>
      <div className="blueprint-copy">
        <span className="blueprint-icon"><FlaskConical size={22} /></span>
        <h2>{newsletter.learnerGoal}</h2>
        <p>
          Designed for {newsletter.learnerLevel}-level learning. Each lesson synthesizes
          trusted sources into a focused explanation, skeptical review, retrieval practice, and
          an application exercise.
        </p>
        <div className="source-pills">
          {sourceHosts.map((source) => <span key={source}>{source}</span>)}
        </div>
      </div>
    </article>
  );
}

function DeliveryCard({
  newsletter,
  resendConfigured,
  busy,
  onSave,
  onSaveContent,
  onSaveSiteVisibility,
}) {
  const [editing, setEditing] = useState(false);
  const [emailEnabled, setEmailEnabled] = useState(newsletter.emailEnabled);

  useEffect(() => {
    setEmailEnabled(newsletter.emailEnabled);
  }, [newsletter.emailEnabled]);

  return (
    <article className="detail-card delivery-card">
      <div className="detail-card-heading">
        <span>Schedule & Delivery</span>
        <button
          className="small-icon-button"
          type="button"
          onClick={() => setEditing((value) => !value)}
          aria-label="Edit delivery preferences"
        >
          <Settings2 size={16} />
        </button>
      </div>
      <div className="delivery-items">
        <div className="delivery-item">
          <span><Clock3 size={20} /></span>
          <div><small>Frequency</small><strong>{formatSchedule(newsletter)}</strong></div>
        </div>
        <div className="delivery-item">
          <span><Mail size={20} /></span>
          <div>
            <small>Recipient</small>
            <strong>{newsletter.emailEnabled
              ? newsletter.emailRecipients.join(", ")
              : "Email delivery is off"}</strong>
          </div>
        </div>
        <div className="delivery-item">
          <span><WandSparkles size={20} /></span>
          <div>
            <small>AI Exploration</small>
            <strong>{newsletter.aiExplorationEnabled ? "Included" : "Source-grounded only"}</strong>
          </div>
        </div>
      </div>
      {editing ? (
        <div className="delivery-editor">
          <label className="switch-row">
            <span>
              <strong>Email delivery</strong>
              <small>{resendConfigured ? "Sender configured" : "Resend sender not configured"}</small>
            </span>
            <input
              type="checkbox"
              checked={emailEnabled}
              onChange={(event) => setEmailEnabled(event.target.checked)}
            />
          </label>
          <p className="field-help">
            Delivery uses your verified Account email
            {newsletter.emailRecipients[0] ? ` (${newsletter.emailRecipients[0]})` : ""}.
          </p>
          <button
            type="button"
            disabled={Boolean(busy)}
            onClick={() => {
              onSave({ enabled: emailEnabled });
            }}
          >
            Save delivery
          </button>
        </div>
      ) : null}
      <button
        className="exploration-toggle"
        type="button"
        disabled={Boolean(busy)}
        onClick={() =>
          onSaveContent({
            aiExplorationEnabled: !newsletter.aiExplorationEnabled,
          })
        }
      >
        <span className={`ios-toggle ${newsletter.aiExplorationEnabled ? "on" : ""}`}><i /></span>
        {newsletter.aiExplorationEnabled ? "Disable AI Exploration" : "Enable AI Exploration"}
      </button>
      <button
        className="exploration-toggle"
        type="button"
        disabled={Boolean(busy)}
        onClick={() => onSaveSiteVisibility(!newsletter.siteVisible)}
      >
        <Globe2 size={17} />
        {newsletter.siteVisible
          ? "Hide this stream from your site"
          : "Show this stream on your site"}
      </button>
    </article>
  );
}

function IssueHistory({
  issues,
  busy,
  onRetryGeneration,
  onRetry,
  onPublicationChange,
}) {
  return (
    <article className="detail-card history-card">
      <div className="detail-card-heading history-heading">
        <div>
          <span>Lesson archive</span>
          <small>{issues.length} lesson{issues.length === 1 ? "" : "s"}</small>
        </div>
        <span className="history-continuity">Each lesson builds on the last <ArrowRight size={14} /></span>
      </div>
      {issues.length === 0 ? (
        <div className="history-empty">
          <CircleDashed size={24} />
          <strong>Your first lesson is waiting to be made</strong>
          <span>Create it now, or let Learnloom prepare it at the scheduled time.</span>
        </div>
      ) : (
        <div className="history-table-wrap">
          <table className="history-table">
            <thead>
              <tr>
                <th>Issue</th>
                <th>Trigger</th>
                <th>Generation</th>
                <th>Delivery</th>
                <th>Created</th>
                <th><span className="sr-only">Action</span></th>
              </tr>
            </thead>
            <tbody>
              {issues.map((issue) => (
                <tr key={issue.id}>
                  <td data-label="Issue">
                    <strong>{issue.title ?? `${humanize(issue.trigger)} Issue`}</strong>
                    <small>{issue.scheduledLocalDate ?? issue.id.slice(-8)}</small>
                  </td>
                  <td data-label="Trigger"><StatusPill status={issue.trigger} /></td>
                  <td data-label="Generation">
                    <StatusPill status={issue.status} />
                    {issue.error ? <small className="row-error">{issue.error}</small> : null}
                  </td>
                  <td data-label="Delivery">
                    {issue.delivery ? (
                      <>
                        <StatusPill status={issue.delivery.status} />
                        {issue.delivery.error ? (
                          <small className="row-error">{issue.delivery.error}</small>
                        ) : null}
                      </>
                    ) : <span className="muted-cell">Email off</span>}
                  </td>
                  <td data-label="Created">{formatDate(issue.createdAt)}</td>
                  <td className="row-action" data-label="Action">
                    {issue.status === "generated" ? (
                      <>
                        <a href={lessonHref(issue.id)}>
                          View <ExternalLink size={14} />
                        </a>
                        <button
                          type="button"
                          disabled={Boolean(busy)}
                          onClick={() => onPublicationChange(issue)}
                        >
                          {issue.publicationState === "published"
                            ? "Hide"
                            : "Publish"}
                        </button>
                      </>
                    ) : null}
                    {issue.status === "failed" ? (
                      <button
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => onRetryGeneration(issue)}
                      >
                        <RotateCcw size={14} />
                        Retry
                      </button>
                    ) : null}
                    {issue.delivery?.status === "failed" ? (
                      <button
                        type="button"
                        disabled={Boolean(busy)}
                        onClick={() => onRetry(issue)}
                      >
                        <RotateCcw size={14} /> Retry
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </article>
  );
}

function StatusPill({ status }) {
  return <span className={`table-pill ${status}`}>{humanize(status)}</span>;
}

function DetailSkeleton() {
  return (
    <div className="detail-skeleton">
      <div />
      <div />
      <div />
    </div>
  );
}

function pipelineProgress(issue) {
  if (!issue || issue.status === "queued" || issue.status === "failed") return 0;
  if (issue.status === "generating") return 4;
  if (issue.status === "generated") return 6;
  return 0;
}

function humanize(value) {
  if (!value) return "None";
  return value.charAt(0).toUpperCase() + value.slice(1).replaceAll("_", " ");
}

function formatSchedule(newsletter) {
  return `Daily at ${newsletter.scheduleTime} · ${newsletter.timeZone}`;
}

function formatDate(value) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function noticeFromLocation() {
  const search = new URLSearchParams(window.location.search);
  if (search.has("created")) return "Your learning stream is ready. Create a lesson now, or let it begin on schedule.";
  if (search.has("queued")) return "Your lesson is queued and will be ready shortly.";
  if (search.get("delivery") === "saved") return "Email delivery settings saved.";
  if (search.get("delivery") === "retried") return "Email delivery queued for another attempt.";
  if (search.get("content") === "saved") return "Content settings saved for future Issues.";
  return "";
}

function lessonHref(issueId) {
  return demoMode
    ? `/?demoIssue=${encodeURIComponent(issueId)}`
    : `/issues/${encodeURIComponent(issueId)}`;
}

export default NewsletterDetail;
