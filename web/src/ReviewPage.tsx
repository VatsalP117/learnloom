import {
  ArrowRight,
  BrainCircuit,
  Check,
  RotateCcw,
} from "lucide-react";
import { useMemo, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell";
import { apiJSON } from "./api";
import { invalidateWorkspaceCache, useWorkspace } from "./useWorkspace";

export default function ReviewPage() {
  const workspace = useWorkspace();
  const [activeIndex, setActiveIndex] = useState(0);
  const [contextOpen, setContextOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [assessmentError, setAssessmentError] = useState("");

  const queue = useMemo(
    () => {
      const lessonsByID = new Map(workspace.lessons.map((lesson) => [lesson.id, lesson]));
      return workspace.reviews.map((review) => {
        const lesson = lessonsByID.get(review.issueId);
        return {
          ...review,
          issueId: review.issueId,
          newsletter: lesson?.newsletter,
          issue: lesson,
        };
      });
    },
    [workspace.lessons, workspace.reviews],
  );
  const due = queue;
  const active = due[activeIndex % Math.max(due.length, 1)];

  async function assess(assessment) {
    if (!active || busy) return;
    setBusy(true);
    setAssessmentError("");
    try {
      await apiJSON(`/api/reviews/${encodeURIComponent(active.id)}/assess`, {
        method: "POST",
        body: {
          assessment,
          idempotencyKey: crypto.randomUUID(),
        },
      });
      setContextOpen(false);
      setActiveIndex(0);
      invalidateWorkspaceCache();
      await workspace.reload();
    } catch (requestError) {
      setAssessmentError(
        requestError instanceof Error
          ? requestError.message
          : "The review could not be saved.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <LearningShell active="review">
      <section className="atelier-page review-page">
        <header className="atelier-page-heading">
          <p className="atelier-eyebrow">Strengthen the thread</p>
          <h1>Spaced retrieval</h1>
          <p>Recall an idea before looking back. Honest effort matters more than a perfect answer.</p>
        </header>

        {workspace.loading ? <AtelierLoading label="Preparing your review queue…" /> : null}
        {workspace.error ? <AtelierError message={workspace.error} onRetry={workspace.reload} /> : null}

        {!workspace.loading && !active ? (
          <section className="review-complete glass-panel">
            <span><Check size={22} /></span>
            <h2>Your review queue is clear.</h2>
            <p>New prompts will appear as you complete more lessons.</p>
            <a href="/library">Return to your library <ArrowRight size={15} /></a>
          </section>
        ) : null}

        {active ? (
          <div className="review-layout">
            <article className="review-card glass-panel">
              <div className="review-card-top">
                <span className="atelier-chip"><BrainCircuit size={13} /> Active recall</span>
                <span>{due.length} prompt{due.length === 1 ? "" : "s"} due</span>
              </div>
              <p className="atelier-eyebrow">{active.newsletter?.name ?? "Recent lesson"}</p>
              <h2>{active.prompt}</h2>
              <p className="review-instruction">
                Explain it aloud or in your own notes. Then reveal the lesson context
                and rate your recall.
              </p>
              {!contextOpen ? (
                <button
                  className="atelier-primary"
                  type="button"
                  onClick={() => {
                    setContextOpen(true);
                    void apiJSON(
                      `/api/issues/${encodeURIComponent(active.issueId)}/review-attempted`,
                      { method: "POST" },
                    ).catch(() => {
                      // Activation measurement must never interrupt review.
                    });
                  }}
                >
                  Reveal lesson context
                </button>
              ) : (
                <div className="review-context">
                  <span>Learning objective</span>
                  <p>{active.objective}</p>
                  <div className="review-rubric">
                    <span>Useful answer</span>
                    <p>{active.answerRubric}</p>
                  </div>
                  <a href={`/issues/${encodeURIComponent(active.issueId)}`}>
                    Reopen the lesson <ArrowRight size={14} />
                  </a>
                  <div className="review-assessment">
                    <button type="button" disabled={busy} onClick={() => assess("needs_work")}>
                      <RotateCcw size={14} /> Needs another pass
                    </button>
                    <button type="button" disabled={busy} onClick={() => assess("partial")}>
                      Partial
                    </button>
                    <button type="button" disabled={busy} onClick={() => assess("solid")}>
                      <Check size={14} /> Recalled solidly
                    </button>
                  </div>
                  {assessmentError ? <p role="alert">{assessmentError}</p> : null}
                </div>
              )}
            </article>

            <aside className="review-summary">
              <article className="glass-panel">
                <p className="atelier-eyebrow">Learning rhythm</p>
                <div className="review-bars" aria-label="Recent review activity">
                  {[32, 48, 38, 72, 55, 84, 64].map((height, index) => (
                    <i style={{ height: `${height}%` }} key={index} />
                  ))}
                </div>
                <span>Return when a lesson feels slightly difficult to recall.</span>
              </article>
              <article className="glass-panel">
                <p className="atelier-eyebrow">This session</p>
                <strong>{due.length} due now</strong>
                <span>Each answer schedules its own next review.</span>
              </article>
            </aside>
          </div>
        ) : null}
      </section>
    </LearningShell>
  );
}
