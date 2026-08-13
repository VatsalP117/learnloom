import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  CheckCircle2,
  Clock3,
  Copy,
  Download,
  ExternalLink,
  Highlighter,
  Lightbulb,
  Map,
  MessageCircleQuestion,
  NotebookPen,
  ShieldCheck,
  Sparkles,
  Trash2,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import CalmLoader from "./CalmLoader";
import { AtelierError } from "./LearningShell";
import { apiFetch, apiJSON } from "./api";
import { normalizeDossier } from "./dossierView";
import { lessonState, updateLessonState } from "./learningState";
import {
  canRevealRetrieval,
  initialRetrievalState,
  resumeLessonProgress,
  type LessonRetrievalResponse,
  type RetrievalResponseState,
} from "./lessonSession";
import type { IssueModerationResponse } from "./types";

interface NoteDraft {
  kind: "note" | "question" | "highlight";
  anchorType: "lesson" | "claim" | "source" | "section";
  anchorId: string;
  body: string;
  quotedText: string;
}

interface LessonNavigationLink {
  issueId: string;
  title: string;
  createdAt: string;
}

interface LessonNavigation {
  previous: LessonNavigationLink | null;
  next: LessonNavigationLink | null;
  nextReviewAt: string | null;
}

interface IssueDetailSnapshot {
  issue: any;
  newsletter: any;
  dossier: any;
  sources: any[];
  feedback: any;
  notes: any[];
  lessonProgress?: {
    progress: number;
    completedAt?: string;
    updatedAt: string;
  } | null;
  retrievals?: LessonRetrievalResponse[];
  navigation?: LessonNavigation;
  site?: {
    visibility: string;
    searchIndexing: boolean;
  } | null;
}

const emptyRetrievalResponse: RetrievalResponseState = {
  response: "",
  skipped: false,
  revealed: false,
  busy: false,
  saving: false,
  error: "",
};

export default function IssueDetail({ issueId }) {
  const [snapshot, setSnapshot] = useState<IssueDetailSnapshot | null>(null);
  const [error, setError] = useState("");
  const [progress, setProgress] = useState(() => lessonState(issueId).progress ?? 0);
  const [completionError, setCompletionError] = useState("");
  const latestProgress = useRef(progress);
  const renderedProgress = useRef(Math.round(progress));
  const persistedProgress = useRef(Math.floor(progress));

  useEffect(() => {
    const controller = new AbortController();
    apiJSON<IssueDetailSnapshot>(`/api/issues/${encodeURIComponent(issueId)}`, {
      signal: controller.signal,
    })
      .then((nextSnapshot) => {
        setSnapshot(nextSnapshot);
        const savedProgress = resumeLessonProgress(
          lessonState(issueId).progress ?? 0,
          nextSnapshot.lessonProgress?.progress ?? 0,
        );
        latestProgress.current = savedProgress;
        renderedProgress.current = Math.round(savedProgress);
        persistedProgress.current = Math.floor(savedProgress);
        setProgress(savedProgress);
        void apiJSON(`/api/issues/${encodeURIComponent(issueId)}/opened`, {
          method: "POST",
        }).catch(() => {
          // Activation measurement must never interrupt reading.
        });
      })
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, [issueId]);

  useEffect(() => {
    const savedProgress = snapshot?.lessonProgress?.progress ?? 0;
    if (savedProgress < 1 || savedProgress >= 100) return undefined;
    const frame = window.requestAnimationFrame(() => {
      const available = document.documentElement.scrollHeight - window.innerHeight;
      if (available > 0) {
        window.scrollTo({ top: available * (savedProgress / 100) });
      }
    });
    return () => window.cancelAnimationFrame(frame);
  }, [snapshot?.lessonProgress?.progress]);

  useEffect(() => {
    if (!snapshot?.issue?.title) return undefined;
    document.title = `${snapshot.issue.title} · Learnloom`;
    return () => {
      document.title = "Learnloom";
    };
  }, [snapshot?.issue?.title]);

  useEffect(() => {
    let frame = 0;
    function measure() {
      if (frame) return;
      frame = window.requestAnimationFrame(() => {
        frame = 0;
        const available = document.documentElement.scrollHeight - window.innerHeight;
        if (available <= 0) return;
        const next = Math.round(
          Math.min(100, Math.max(0, (window.scrollY / available) * 100)),
        );
        latestProgress.current = next;
        if (next === renderedProgress.current) return;
        renderedProgress.current = next;
        setProgress(next);
      });
    }
    window.addEventListener("scroll", measure, { passive: true });
    measure();
    return () => {
      window.removeEventListener("scroll", measure);
      if (frame) window.cancelAnimationFrame(frame);
      void persistLessonProgress(issueId, latestProgress.current);
      updateLessonState(issueId, {
        progress: Math.max(lessonState(issueId).progress ?? 0, latestProgress.current),
        lastOpenedAt: new Date().toISOString(),
      });
    };
  }, [issueId]);

  useEffect(() => {
    const next = Math.min(99, Math.floor(progress));
    if (next < 1 || next <= persistedProgress.current) return undefined;
    const timeout = window.setTimeout(() => {
      persistedProgress.current = next;
      void persistLessonProgress(issueId, next);
    }, 1500);
    return () => window.clearTimeout(timeout);
  }, [issueId, progress]);

  if (!snapshot && !error) {
    return <CalmLoader label="Opening your lesson…" detail="Bringing you back to the page." />;
  }
  if (error) {
    return <div className="reader-loading"><AtelierError message={error} /></div>;
  }

  return (
    <LessonReader
      {...snapshot}
      dossier={normalizeDossier(snapshot.dossier, snapshot.newsletter)}
      progress={progress}
      completionError={completionError}
      onComplete={async () => {
        setCompletionError("");
        try {
          await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/complete`, {
            method: "POST",
          });
        } catch (requestError) {
          setCompletionError(
            requestError instanceof Error
              ? requestError.message
              : "The lesson could not be marked complete.",
          );
          return false;
        }
        latestProgress.current = 100;
        renderedProgress.current = 100;
        setProgress(100);
        updateLessonState(issueId, {
          progress: 100,
          completed: true,
          completedAt: new Date().toISOString(),
        });
        return true;
      }}
    />
  );
}

async function persistLessonProgress(issueId: string, progress: number) {
  const next = Math.min(99, Math.floor(progress));
  if (next < 1) return;
  try {
    await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/progress`, {
      method: "POST",
      body: { progress: next },
    });
  } catch {
    // Reading remains uninterrupted; local progress is retained for a later sync.
  }
}

function LessonReader({
  issue,
  dossier,
  newsletter,
  sources,
  feedback,
  notes: initialNotes = [],
  retrievals: initialRetrievals = [],
  navigation = { previous: null, next: null, nextReviewAt: null },
  lessonProgress = null,
  progress,
  completionError,
  onComplete,
  site = null,
}) {
  const [completed, setCompleted] = useState(() =>
    Boolean(lessonProgress?.completedAt || lessonState(issue.id).completed));
  const [completing, setCompleting] = useState(false);
  const [notes, setNotes] = useState(initialNotes);
  const [noteDraft, setNoteDraft] = useState<NoteDraft | null>(null);
  const lessonType = dossier.lessonType
    ? `${dossier.lessonType.replace("_", " ")} lesson`
    : "Today’s lesson";

  return (
    <div className="focus-reader">
      <div className="reader-progress" aria-label={`${Math.round(progress)}% read`}>
        <i style={{ width: `${progress}%` }} />
      </div>
      <header className="reader-toolbar">
        <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
          <ArrowLeft size={15} /> {newsletter.name}
        </a>
        <span>{Math.round(progress)}% read</span>
        <div>
          <LessonExportButton issueId={issue.id} />
          <a href="/library">Library <BookOpen size={14} /></a>
        </div>
      </header>

      <article className="reader-paper">
        <header className="reader-hero">
          <div className="reader-meta">
            <span><BookOpen size={14} /> {lessonType}</span>
            <span><Clock3 size={14} />{dossier.readTime} min</span>
            <span>{formatDate(issue.createdAt)}</span>
          </div>
          <p className="atelier-eyebrow">{newsletter.name}</p>
          <h1>{issue.title}</h1>
          <p className="reader-deck">{dossier.deck}</p>
          <div className="reader-grounding">
            <span><Check size={13} /> {dossier.evidenceStatus === "source_bounded" ? "Source-bounded evidence" : "Source-grounded"}</span>
            <span>Prepared from {sources.length} trusted sources</span>
            <span>{newsletter.learnerLevel} level</span>
            <span>{readerAudienceLabel(issue, newsletter, site)}</span>
          </div>
        </header>

        <div className="reader-layout">
          <main className="reader-content">
            <section className="reader-objective">
              <span><Lightbulb size={19} /></span>
              <div>
                <p className="atelier-eyebrow">Learning objective</p>
                <p>{dossier.objective}</p>
              </div>
            </section>

            {dossier.whyNow || dossier.continuityBridge ? (
              <section className="reader-orientation">
                {dossier.whyNow ? (
                  <div>
                    <p className="atelier-eyebrow">Why this lesson now</p>
                    <p>{dossier.whyNow}</p>
                  </div>
                ) : null}
                {dossier.continuityBridge ? (
                  <div>
                    <p className="atelier-eyebrow">Connection to prior learning</p>
                    <p>{dossier.continuityBridge}</p>
                  </div>
                ) : null}
                {dossier.concepts?.length ? (
                  <div className="reader-concepts">
                    <p className="atelier-eyebrow">Concepts in this lesson</p>
                    <p>{dossier.concepts.map((concept) => <span key={concept}>{concept}</span>)}</p>
                  </div>
                ) : null}
              </section>
            ) : null}

            {dossier.sections.map((section, index) => (
              <section className="reader-section" id={`section-${index + 1}`} key={section.heading}>
                <p className="atelier-eyebrow">{String(index + 1).padStart(2, "0")} · {section.label}</p>
                <h2>{section.heading}</h2>
                {section.paragraphs.map((paragraph, paragraphIndex) => {
                  const value = typeof paragraph === "string"
                    ? { text: paragraph, sourceIds: [] }
                    : paragraph;
                  return (
                    <p key={`${section.heading}-${paragraphIndex}`}>
                      {value.text}
                      {value.sourceIds.map((sourceID) => {
                        const sourceIndex = sources.findIndex((source) => source.id === sourceID);
                        if (sourceIndex < 0) return null;
                        return (
                          <a
                            className="reader-citation"
                            href={`#source-${sourceID}`}
                            title={`See source ${sourceIndex + 1}`}
                            aria-label={`See source ${sourceIndex + 1}`}
                            key={sourceID}
                          >
                            [{sourceIndex + 1}]
                          </a>
                        );
                      })}
                    </p>
                  );
                })}
                {section.callout ? <blockquote>{section.callout}</blockquote> : null}
              </section>
            ))}

            {dossier.claims?.length ? (
              <details className="reader-evidence-appendix">
                <summary>
                  <span><ShieldCheck size={16} /></span>
                  <span><strong>Deeper evidence appendix</strong><small>Inspect claim mappings, limitations, and source notes.</small></span>
                  <ArrowRight size={14} />
                </summary>
                <section className="reader-claims">
                <p className="atelier-eyebrow">Evidence map</p>
                <h2>Claims you can inspect</h2>
                <div>
                  {dossier.claims.map((claim) => {
                    const claimSources = claim.sourceIds
                      .map((sourceID) => sources.find((source) => source.id === sourceID))
                      .filter(Boolean);
                    const citation = [
                      claim.text,
                      ...claimSources.map((source) => `${source.name}: ${source.url}`),
                    ].join("\n");
                    return (
                      <article key={claim.id}>
                        <p>{claim.text}</p>
                        <div>
                          {claimSources.map((source) => (
                            <a href={`#source-${source.id}`} key={source.id}>
                              {source.id}
                            </a>
                          ))}
                        </div>
                        <footer>
                          <CopyButton text={citation} label="Copy citation" />
                          <button
                            type="button"
                            onClick={() => setNoteDraft({
                              kind: "question",
                              anchorType: "claim",
                              anchorId: claim.id,
                              quotedText: claim.text,
                              body: "",
                            })}
                          >
                            <MessageCircleQuestion size={13} /> Question this claim
                          </button>
                        </footer>
                      </article>
                    );
                  })}
                </div>
                {dossier.limitations?.length ? (
                  <div className="reader-appendix-limits">
                    <p className="atelier-eyebrow">Important limitations</p>
                    {dossier.limitations.map((limitation) => (
                      <p key={limitation.id}>{limitation.text}</p>
                    ))}
                  </div>
                ) : null}
                <div className="reader-appendix-sources">
                  <p className="atelier-eyebrow">Source notes</p>
                  {sources.map((source, index) => (
                    <article key={source.id ?? source.url}>
                      <strong>[{index + 1}] {source.name}</strong>
                      {source.summary ? <p>{source.summary}</p> : <p>No deeper source note was stored.</p>}
                      <a href={source.url} target="_blank" rel="noreferrer">Open source <ExternalLink size={12} /></a>
                    </article>
                  ))}
                </div>
              </section>
              </details>
            ) : null}

            <RetrievalSection
              issueId={issue.id}
              questions={dossier.retrievalItems?.length
                ? dossier.retrievalItems
                : dossier.retrieval}
              initialResponses={initialRetrievals}
            />

            <section className="reader-application">
              <p className="atelier-eyebrow"><Sparkles size={14} /> Try this in the world</p>
              <p>{dossier.application}</p>
            </section>

            <LessonFeedbackPanel issueId={issue.id} initialFeedback={feedback} />

            <section className="reader-complete">
              <span><CheckCircle2 size={23} /></span>
              <h2>{completed ? "Lesson complete." : "Close the loop."}</h2>
              <p>
                {completed
                  ? "This lesson is now part of your learning history."
                  : "Mark this lesson complete when you have finished the recall prompts or reflected on the central idea."}
              </p>
              {completed ? (
                <div className="reader-capability">
                  <div>
                    <p className="atelier-eyebrow">Capability gained</p>
                    <strong>{dossier.objective || "Explain the lesson’s central mechanism and its evidence boundary."}</strong>
                  </div>
                  <div>
                    <p className="atelier-eyebrow">Likely next direction</p>
                    <strong>{dossier.nextConcepts?.[0] || "Apply the model in a new context."}</strong>
                  </div>
                  <div>
                    <p className="atelier-eyebrow">Next review</p>
                    <strong>{navigation.nextReviewAt
                      ? formatRelativeReview(navigation.nextReviewAt)
                      : "Ready now in Review"}</strong>
                  </div>
                </div>
              ) : null}
              {!completed ? (
                <button
                  className="atelier-primary"
                  type="button"
                  disabled={completing}
                  onClick={async () => {
                    setCompleting(true);
                    if (!(await onComplete())) {
                      setCompleting(false);
                      return;
                    }
                    setCompleted(true);
                  }}
                >
                  {completing ? "Saving…" : "Mark lesson complete"} <Check size={15} />
                </button>
              ) : null}
              {completionError ? <p role="alert">{completionError}</p> : null}
              <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
                Return to this learning stream <ArrowRight size={15} />
              </a>
              <div className="reader-adjacent">
                {navigation.previous ? (
                  <a href={`/issues/${encodeURIComponent(navigation.previous.issueId)}`}>
                    <ArrowLeft size={14} /><span><small>Previous lesson</small>{navigation.previous.title}</span>
                  </a>
                ) : <span />}
                {navigation.next ? (
                  <a href={`/issues/${encodeURIComponent(navigation.next.issueId)}`}>
                    <span><small>Next lesson</small>{navigation.next.title}</span><ArrowRight size={14} />
                  </a>
                ) : <span />}
              </div>
            </section>
          </main>

          <aside className="reader-aside">
            <nav>
              <p className="atelier-eyebrow"><Map size={14} /> Lesson map</p>
              {dossier.sections.map((section, index) => (
                <a href={`#section-${index + 1}`} key={section.heading}>
                  <span>{String(index + 1).padStart(2, "0")}</span>{section.heading}
                </a>
              ))}
              <a href="#retrieval"><span>R</span>Pause and retrieve</a>
            </nav>
            <div className="reader-sources">
              <p className="atelier-eyebrow">Sources consulted</p>
              {sources.map((source, index) => (
                <div id={`source-${source.id ?? `S${index + 1}`}`} key={`${source.id}-${source.name}`}>
                  <a
                    href={source.url}
                    target="_blank"
                    rel="noreferrer"
                  >
                    <span><i>{index + 1}</i>{source.name}</span>
                    <ExternalLink size={13} />
                  </a>
                  <CopyButton
                    text={`${source.name}. ${source.url}`}
                    label={`Copy citation for ${source.name}`}
                    compact
                  />
                </div>
              ))}
              <p>
                These sources informed the lesson. Claim-level citation mapping is
                shown only when it is available in the generated artifact.
              </p>
            </div>
            <LessonNotes
              issueId={issue.id}
              notes={notes}
              setNotes={setNotes}
              draft={noteDraft}
              setDraft={setNoteDraft}
            />
            <PublisherTrustPanel issue={issue} />
          </aside>
        </div>
      </article>
    </div>
  );
}

export function readerAudienceLabel(issue, newsletter, site) {
  if (issue.publicationState === "private") return "Audience: only you";
  if (issue.publicationState !== "published") return "Audience: draft, only you";
  if (site?.visibility !== "public") return "Audience: not visible, site private";
  if (!newsletter.siteVisible) return "Audience: not visible, stream private";
  return site.searchIndexing ? "Audience: public + search" : "Audience: public by link";
}

function PublisherTrustPanel({ issue }) {
  const [moderation, setModeration] = useState(null);
  const [correction, setCorrection] = useState("");
  const [holdReason, setHoldReason] = useState("");
  const [resolutionReasons, setResolutionReasons] = useState({});
  const [status, setStatus] = useState("");

  const refresh = useCallback(async () => {
    const next = await apiJSON<IssueModerationResponse>(
      `/api/issues/${encodeURIComponent(issue.id)}/moderation`,
    );
    setModeration(next);
    setHoldReason(next.reason ?? "");
    setStatus("");
  }, [issue.id]);

  useEffect(() => {
    void refresh().catch((requestError) => setStatus(requestError.message));
  }, [refresh]);

  if (!moderation) {
    return (
      <section className="reader-trust">
        <p className="atelier-eyebrow"><ShieldCheck size={13} /> Publishing trust</p>
        <small>{status || "Loading publishing controls…"}</small>
      </section>
    );
  }

  async function publishCorrection(event) {
    event.preventDefault();
    setStatus("");
    try {
      await apiJSON(`/api/issues/${encodeURIComponent(issue.id)}/corrections`, {
        method: "POST",
        body: { body: correction },
      });
      setCorrection("");
      await refresh();
      setStatus("Correction published.");
    } catch (requestError) {
      setStatus(requestError.message);
    }
  }

  async function setModerationState(state) {
    setStatus("");
    try {
      await apiJSON(`/api/issues/${encodeURIComponent(issue.id)}/moderation`, {
        method: "POST",
        body: {
          state,
          reason: state === "held" ? holdReason : "Review completed.",
        },
      });
      await refresh();
      setStatus(state === "held" ? "Public page held." : "Public page cleared.");
    } catch (requestError) {
      setStatus(requestError.message);
    }
  }

  async function resolveReport(report, nextStatus) {
    const reason = (resolutionReasons[report.id] ?? "").trim();
    if (!reason) {
      setStatus("Add a resolution reason first.");
      return;
    }
    setStatus("");
    try {
      await apiJSON(`/api/reports/${encodeURIComponent(report.id)}/resolve`, {
        method: "POST",
        body: { status: nextStatus, reason },
      });
      await refresh();
      setStatus(nextStatus === "resolved" ? "Report resolved." : "Report dismissed.");
    } catch (requestError) {
      setStatus(requestError.message);
    }
  }

  return (
    <section className="reader-trust">
      <p className="atelier-eyebrow"><ShieldCheck size={13} /> Publishing trust</p>
      <p>
        {moderation.state === "held"
          ? "This page is held from public reading and search."
          : issue.publicationState === "published"
            ? "This lesson is public and eligible for reading."
            : "Publish this lesson before sharing corrections publicly."}
      </p>
      <label>
        Moderation reason
        <textarea
          maxLength={1000}
          rows={3}
          placeholder="Why should this page be held?"
          value={holdReason}
          onChange={(event) => setHoldReason(event.target.value)}
        />
      </label>
      <div className="reader-trust-actions">
        {moderation.state === "held" ? (
          <button type="button" onClick={() => setModerationState("clear")}>
            Clear hold
          </button>
        ) : (
          <button
            type="button"
            disabled={!holdReason.trim()}
            onClick={() => setModerationState("held")}
          >
            Hold public page
          </button>
        )}
      </div>

      <form onSubmit={publishCorrection}>
        <label>
          Public correction
          <textarea
            maxLength={2000}
            rows={4}
            placeholder="State what changed and what readers should know."
            value={correction}
            onChange={(event) => setCorrection(event.target.value)}
          />
        </label>
        <button type="submit" disabled={!correction.trim()}>Publish correction</button>
      </form>
      {moderation.corrections?.map((item) => (
        <article className="reader-correction" key={item.id}>
          <p>{item.body}</p>
          <button
            type="button"
            onClick={async () => {
              await apiJSON(`/api/corrections/${encodeURIComponent(item.id)}`, {
                method: "DELETE",
              });
              await refresh();
              setStatus("Correction retracted.");
            }}
          >
            Retract
          </button>
        </article>
      ))}

      {moderation.reports?.filter((report) => report.status === "open").map((report) => (
        <article className="reader-report" key={report.id}>
          <strong>{report.category}</strong>
          <p>{report.details || "No additional details supplied."}</p>
          <textarea
            aria-label={`Resolution reason for ${report.category} report`}
            maxLength={1000}
            rows={3}
            placeholder="Record what you checked and decided."
            value={resolutionReasons[report.id] ?? ""}
            onChange={(event) => setResolutionReasons((current) => ({
              ...current,
              [report.id]: event.target.value,
            }))}
          />
          <div>
            <button type="button" onClick={() => resolveReport(report, "resolved")}>
              Resolve
            </button>
            <button type="button" onClick={() => resolveReport(report, "dismissed")}>
              Dismiss
            </button>
          </div>
        </article>
      ))}
      {status ? <small role="status">{status}</small> : null}
    </section>
  );
}

function LessonExportButton({ issueId }) {
  const [busy, setBusy] = useState(false);
  return (
    <button
      type="button"
      disabled={busy}
      onClick={async () => {
        setBusy(true);
        try {
          const response = await apiFetch(
            `/api/issues/${encodeURIComponent(issueId)}/export?format=markdown`,
          );
          if (!response.ok) throw new Error("Export failed");
          const blob = await response.blob();
          const url = URL.createObjectURL(blob);
          const link = document.createElement("a");
          link.href = url;
          link.download = "learnloom-lesson.md";
          link.click();
          URL.revokeObjectURL(url);
        } finally {
          setBusy(false);
        }
      }}
    >
      <Download size={13} /> {busy ? "Exporting…" : "Export"}
    </button>
  );
}

function CopyButton({ text, label, compact = false }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      className={compact ? "reader-copy compact" : "reader-copy"}
      type="button"
      aria-label={label}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1500);
        } catch {
          setCopied(false);
        }
      }}
    >
      {copied ? <Check size={13} /> : <Copy size={13} />}
      {compact ? null : copied ? "Copied" : label}
    </button>
  );
}

function LessonNotes({ issueId, notes, setNotes, draft, setDraft }) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  function startNote() {
    setDraft({
      kind: "note",
      anchorType: "lesson",
      anchorId: "",
      quotedText: "",
      body: "",
    });
  }

  function startHighlight() {
    const quotedText = window.getSelection()?.toString().trim().slice(0, 1200) ?? "";
    if (!quotedText) {
      setError("Select a passage in the lesson first.");
      return;
    }
    setError("");
    setDraft({
      kind: "highlight",
      anchorType: "lesson",
      anchorId: "",
      quotedText,
      body: "",
    });
  }

  async function save() {
    if (!draft?.body.trim() || saving) return;
    setSaving(true);
    setError("");
    try {
      const note = await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/notes`, {
        method: "POST",
        body: draft,
      });
      setNotes((current) => [note, ...current]);
      setDraft(null);
    } catch (requestError) {
      setError(requestError instanceof Error
        ? requestError.message
        : "The note could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  async function remove(noteId) {
    setError("");
    try {
      await apiJSON(`/api/notes/${encodeURIComponent(noteId)}`, {
        method: "DELETE",
      });
      setNotes((current) => current.filter((note) => note.id !== noteId));
    } catch (requestError) {
      setError(requestError instanceof Error
        ? requestError.message
        : "The note could not be deleted.");
    }
  }

  return (
    <section className="reader-notes" id="lesson-notes">
      <p className="atelier-eyebrow"><NotebookPen size={13} /> Your notes</p>
      <div className="reader-note-actions">
        <button type="button" onClick={startNote}><NotebookPen size={13} /> Add note</button>
        <button type="button" onClick={startHighlight}><Highlighter size={13} /> Save selection</button>
      </div>
      {draft ? (
        <div className="reader-note-composer">
          {draft.quotedText ? <blockquote>{draft.quotedText}</blockquote> : null}
          <textarea
            autoFocus
            maxLength={4000}
            rows={4}
            placeholder={draft.kind === "question"
              ? "What do you want to verify or connect?"
              : draft.kind === "highlight"
                ? "Why is this passage worth keeping?"
                : "Capture an idea in your own words…"}
            value={draft.body}
            onChange={(event) => setDraft({ ...draft, body: event.target.value })}
          />
          <div>
            <button type="button" onClick={() => setDraft(null)}>Cancel</button>
            <button type="button" disabled={!draft.body.trim() || saving} onClick={save}>
              {saving ? "Saving…" : "Save"}
            </button>
          </div>
        </div>
      ) : null}
      <div className="reader-note-list">
        {notes.map((note) => (
          <article key={note.id}>
            <span>{note.kind}</span>
            {note.quotedText ? <blockquote>{note.quotedText}</blockquote> : null}
            <p>{note.body}</p>
            <button type="button" aria-label="Delete note" onClick={() => remove(note.id)}>
              <Trash2 size={12} />
            </button>
          </article>
        ))}
        {!notes.length && !draft ? <small>No notes yet. Keep only what will help you think later.</small> : null}
      </div>
      {error ? <p role="alert">{error}</p> : null}
    </section>
  );
}

export function LessonFeedbackPanel({ issueId, initialFeedback }) {
  const [values, setValues] = useState({
    difficulty: initialFeedback?.difficulty ?? "",
    relevance: initialFeedback?.relevance ?? "",
    recallConfidence: initialFeedback?.recallConfidence ?? "",
  });
  const [status, setStatus] = useState("");
  const [saving, setSaving] = useState(false);

  async function save() {
    if (!values.difficulty && !values.relevance && !values.recallConfidence) return;
    setSaving(true);
    setStatus("");
    try {
      await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/feedback`, {
        method: "POST",
        body: {
          difficulty: values.difficulty || undefined,
          relevance: values.relevance || undefined,
          recallConfidence: values.recallConfidence || undefined,
        },
      });
      setStatus("Saved to your learning history.");
    } catch (requestError) {
      setStatus(
        requestError instanceof Error
          ? requestError.message
          : "Your reflection could not be saved.",
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="reader-feedback">
      <p className="atelier-eyebrow">Shape what comes next</p>
      <h2>How did this lesson fit you?</h2>
      <p>These signals are private and record how the lesson fit you.</p>
      <FeedbackChoice
        label="Difficulty"
        value={values.difficulty}
        options={[
          ["too_basic", "Too basic"],
          ["right", "About right"],
          ["too_advanced", "Too advanced"],
        ]}
        onChange={(difficulty) => setValues((current) => ({ ...current, difficulty }))}
      />
      <FeedbackChoice
        label="Relevance"
        value={values.relevance}
        options={[
          ["not_relevant", "Not relevant"],
          ["somewhat_relevant", "Somewhat"],
          ["very_relevant", "Very relevant"],
        ]}
        onChange={(relevance) => setValues((current) => ({ ...current, relevance }))}
      />
      <FeedbackChoice
        label="Recall confidence"
        value={values.recallConfidence}
        options={[
          ["low", "Needs work"],
          ["medium", "Partial"],
          ["high", "Solid"],
        ]}
        onChange={(recallConfidence) =>
          setValues((current) => ({ ...current, recallConfidence }))}
      />
      <button
        className="atelier-primary"
        type="button"
        disabled={
          saving ||
          (!values.difficulty && !values.relevance && !values.recallConfidence)
        }
        onClick={save}
      >
        {saving ? "Saving…" : "Save reflection"}
      </button>
      {status ? <span role="status">{status}</span> : null}
    </section>
  );
}

function FeedbackChoice({ label, value, options, onChange }) {
  return (
    <fieldset>
      <legend>{label}</legend>
      <div>
        {options.map(([optionValue, optionLabel]) => (
          <button
            className={value === optionValue ? "selected" : ""}
            type="button"
            aria-pressed={value === optionValue}
            onClick={() => onChange(optionValue)}
            key={optionValue}
          >
            {optionLabel}
          </button>
        ))}
      </div>
    </fieldset>
  );
}

function RetrievalSection({ issueId, questions, initialResponses = [] }) {
  const [responses, setResponses] = useState(() => initialRetrievalState(initialResponses));
  const draftTimers = useRef<Record<string, number>>({});

  useEffect(() => () => {
    Object.values(draftTimers.current).forEach((timer) => window.clearTimeout(timer));
  }, []);

  function updateResponse(promptKey, patch) {
    setResponses((current) => ({
      ...current,
      [promptKey]: {
        response: "",
        skipped: false,
        revealed: false,
        busy: false,
        saving: false,
        error: "",
        ...current[promptKey],
        ...patch,
      },
    }));
  }

  function clearDraftTimer(promptKey) {
    if (draftTimers.current[promptKey]) {
      window.clearTimeout(draftTimers.current[promptKey]);
      delete draftTimers.current[promptKey];
    }
  }

  async function saveDraft(promptKey, response) {
    const value = response.trim();
    if (!canRevealRetrieval(value) || responses[promptKey]?.revealed) return;
    clearDraftTimer(promptKey);
    updateResponse(promptKey, { saving: true, error: "" });
    try {
      const saved = await apiJSON<LessonRetrievalResponse>(
        `/api/issues/${encodeURIComponent(issueId)}/retrievals/${encodeURIComponent(promptKey)}`,
        { method: "PUT", body: { response: value, skipped: false } },
      );
      updateResponse(promptKey, {
        response: saved.response ?? value,
        saving: false,
      });
    } catch (requestError) {
      updateResponse(promptKey, {
        saving: false,
        error: requestError instanceof Error
          ? requestError.message
          : "Your draft answer could not be saved.",
      });
    }
  }

  function scheduleDraft(promptKey, response) {
    clearDraftTimer(promptKey);
    if (!canRevealRetrieval(response)) return;
    draftTimers.current[promptKey] = window.setTimeout(() => {
      delete draftTimers.current[promptKey];
      void saveDraft(promptKey, response);
    }, 700);
  }

  async function reveal(item, skipped) {
    const promptKey = item.id;
    const current = responses[promptKey] ?? emptyRetrievalResponse;
    const response = skipped ? "" : (current.response ?? "").trim();
    if (!skipped && !response) return;
    clearDraftTimer(promptKey);
    updateResponse(promptKey, { busy: true, error: "" });
    try {
      const saved = await apiJSON<LessonRetrievalResponse>(
        `/api/issues/${encodeURIComponent(issueId)}/retrievals/${encodeURIComponent(promptKey)}`,
        { method: "POST", body: { response, skipped } },
      );
      updateResponse(promptKey, {
        response: saved.response ?? response,
        skipped: Boolean(saved.skipped),
        revealed: true,
        busy: false,
      });
    } catch (requestError) {
      updateResponse(promptKey, {
        busy: false,
        error: requestError instanceof Error
          ? requestError.message
          : "Your response could not be saved.",
      });
    }
  }

  return (
    <section className="reader-retrieval" id="retrieval">
      <p className="atelier-eyebrow">Pause and retrieve</p>
      <h2>Can you explain it without looking back?</h2>
      <p>Write what you remember before revealing the reflection. Your answer resumes on any device; skipping never counts against you.</p>
      <div>
        {questions.map((question, index) => {
          const item = typeof question === "string"
            ? { id: `question-${index + 1}`, prompt: question }
            : question;
          const state = responses[item.id] ?? emptyRetrievalResponse;
          return (
          <article key={item.id ?? item.prompt}>
            <span>{String(index + 1).padStart(2, "0")}</span>
            <p>{item.prompt}</p>
            {!state.revealed ? (
              <div className="retrieval-response">
                <label>
                  <span>Your answer</span>
                  <textarea
                    maxLength={2000}
                    rows={3}
                    value={state.response ?? ""}
                    onChange={(event) => {
                      updateResponse(item.id, { response: event.target.value, error: "" });
                      scheduleDraft(item.id, event.target.value);
                    }}
                    onBlur={(event) => void saveDraft(item.id, event.target.value)}
                    placeholder="A few words are enough. Name the mechanism and an important limit."
                  />
                </label>
                <div>
                  <button
                    type="button"
                    disabled={state.busy || state.saving || !canRevealRetrieval(state.response)}
                    onClick={() => void reveal(item, false)}
                  >
                    {state.busy ? "Saving…" : "Save and reveal"}
                  </button>
                  <button
                    type="button"
                    disabled={state.busy || state.saving}
                    onClick={() => void reveal(item, true)}
                  >
                    Skip for now
                  </button>
                </div>
                {state.saving ? <small>Saving draft…</small> : null}
                {state.error ? <small role="alert">{state.error}</small> : null}
              </div>
            ) : (
              <div className="retrieval-reveal">
                {state.skipped
                  ? <p>Skipped without penalty.</p>
                  : <p><strong>Your answer:</strong> {state.response}</p>}
                <small>
                  <strong>Reflection:</strong>{" "}
                {item.answerRubric || item.correctiveExplanation ||
                  "Return to the mechanism and evidence above. If your explanation names both the cause and its limits, you have the useful shape of the idea."}
                </small>
              </div>
            )}
          </article>
          );
        })}
      </div>
    </section>
  );
}

function formatDate(value) {
  return new Intl.DateTimeFormat("en", {
    month: "long",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value));
}

function formatRelativeReview(value) {
  const due = new Date(value);
  const days = Math.max(0, Math.ceil((due.getTime() - Date.now()) / 86_400_000));
  if (days === 0) return "Ready now in Review";
  if (days === 1) return "Tomorrow";
  return `In ${days} days`;
}
