import {
  ArrowRight,
  BrainCircuit,
  Check,
  RotateCcw,
} from "lucide-react";
import { useMemo, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell.jsx";
import { reviewState, updateReviewState } from "./learningState.js";
import { useWorkspace } from "./useWorkspace.js";

export default function ReviewPage() {
  const workspace = useWorkspace();
  const [activeIndex, setActiveIndex] = useState(0);
  const [contextOpen, setContextOpen] = useState(false);
  const [, refreshState] = useState(0);

  const queue = useMemo(
    () =>
      workspace.reviews.flatMap((review) => {
        const lesson = workspace.lessons.find((item) => item.id === review.issueId);
        return (review.questions ?? []).map((question, index) => ({
          id: `${review.issueId}:${index}`,
          issueId: review.issueId,
          question,
          objective: review.objective,
          newsletter: lesson?.newsletter,
          issue: lesson,
        }));
      }),
    [workspace.lessons, workspace.reviews],
  );
  const due = queue.filter((item) => reviewState(item.id).status !== "mastered");
  const active = due[activeIndex % Math.max(due.length, 1)];
  const mastered = queue.length - due.length;

  function assess(status) {
    if (!active) return;
    updateReviewState(active.id, { status, reviewedAt: new Date().toISOString() });
    setContextOpen(false);
    setActiveIndex(0);
    refreshState((value) => value + 1);
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
              <h2>{active.question}</h2>
              <p className="review-instruction">
                Explain it aloud or in your own notes. Then reveal the lesson context
                and rate your recall.
              </p>
              {!contextOpen ? (
                <button className="atelier-primary" type="button" onClick={() => setContextOpen(true)}>
                  Reveal lesson context
                </button>
              ) : (
                <div className="review-context">
                  <span>Learning objective</span>
                  <p>{active.objective}</p>
                  <a href={`/issues/${encodeURIComponent(active.issueId)}`}>
                    Reopen the lesson <ArrowRight size={14} />
                  </a>
                  <div className="review-assessment">
                    <button type="button" onClick={() => assess("needs-work")}>
                      <RotateCcw size={14} /> Needs another pass
                    </button>
                    <button type="button" onClick={() => assess("mastered")}>
                      <Check size={14} /> Recalled it
                    </button>
                  </div>
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
                <strong>{mastered} recalled</strong>
                <span>{due.length} still worth revisiting</span>
              </article>
            </aside>
          </div>
        ) : null}
      </section>
    </LearningShell>
  );
}
