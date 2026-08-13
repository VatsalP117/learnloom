import {
  ArrowRight,
  BookOpen,
  BrainCircuit,
  CheckCircle2,
  Clock3,
  Sparkles,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import LearningShell, {
  AtelierError,
  AtelierLoading,
  formatShortDate,
} from "./LearningShell";
import { lessonState } from "./learningState";
import { apiJSON } from "./api";
import { useWorkspace } from "./useWorkspace";

export default function TodayPage() {
  const workspace = useWorkspace();
  const [, refreshState] = useState(0);
  const [reentryBusy, setReentryBusy] = useState("");
  const [reentryNotice, setReentryNotice] = useState("");
  const [reentryError, setReentryError] = useState("");

  useEffect(() => {
    const refresh = () => refreshState((value) => value + 1);
    window.addEventListener("learnloom:state", refresh);
    return () => window.removeEventListener("learnloom:state", refresh);
  }, []);

  const { primary, secondary, focus, reason, actionLabel, actionUrl, dueCount } = useMemo(
    () => resolveTodaySelection(
      workspace.snapshot?.todayFocus,
      workspace.lessons,
      workspace.reviews,
      lessonState,
      workspace.snapshot?.retention,
    ),
    [workspace.lessons, workspace.reviews, workspace.snapshot?.retention, workspace.snapshot?.todayFocus],
  );
  const primaryState = primary ? {
    ...lessonState(primary.id),
    progress: Math.max(
      lessonState(primary.id).progress ?? 0,
      workspace.snapshot?.todayFocus?.kind === "lesson"
        ? workspace.snapshot.todayFocus.progress ?? 0
        : 0,
    ),
  } : null;
  const reviewFirst = focus === "review";
  const reentryFirst = focus === "reentry";
  const specialFocus = reviewFirst || reentryFirst;
  const sideLesson = specialFocus ? primary : secondary;
  const reentryNewsletterId = workspace.snapshot?.todayFocus?.newsletterId ||
    workspace.snapshot?.retention?.reentryNewsletterId;
  const reentryNewsletter = workspace.newsletters.find(({ id }) => id === reentryNewsletterId);

  async function handleReentryControl(action: "reduce" | "pause" | "reset") {
    if (!reentryNewsletterId || !reentryNewsletter) return;
    setReentryBusy(action);
    setReentryError("");
    try {
      if (action === "reduce") {
        await apiJSON(`/api/newsletters/${encodeURIComponent(reentryNewsletterId)}/rhythm`, {
          method: "POST",
          body: {
            mode: "weekly_synthesis",
            selectedWeekdays: [reentryNewsletter.selectedWeekdays?.[0] ?? 1],
            autoThrottleEnabled: reentryNewsletter.autoThrottleEnabled ?? true,
            unopenedLessonLimit: reentryNewsletter.unopenedLessonLimit ?? 3,
          },
        });
        setReentryNotice("This stream now prepares one synthesis each week.");
      } else if (action === "pause") {
        await apiJSON(`/api/newsletters/${encodeURIComponent(reentryNewsletterId)}/active`, {
          method: "POST",
          body: { active: false },
        });
        setReentryNotice("This stream is paused. Your history is unchanged.");
      } else {
        const result = await apiJSON<{ dismissedCount: number }>(
          `/api/newsletters/${encodeURIComponent(reentryNewsletterId)}/reset-backlog`,
          { method: "POST", body: {} },
        );
        setReentryNotice(result.dismissedCount > 0
          ? `${result.dismissedCount} older lesson${result.dismissedCount === 1 ? "" : "s"} moved out of Today. They remain in your library.`
          : "Your Today queue is already clear.");
      }
      await workspace.reload();
    } catch (error) {
      setReentryError(error.message);
    } finally {
      setReentryBusy("");
    }
  }

  return (
    <LearningShell active="today">
      <section className="today-page">
        <header className="atelier-page-heading today-heading">
          <p className="atelier-eyebrow">Your learning practice</p>
          <h1>{greeting()}.</h1>
          <p>Choose one worthwhile step. Learnloom will keep the larger thread.</p>
        </header>

        {workspace.loading ? <AtelierLoading /> : null}
        {workspace.error ? (
          <AtelierError message={workspace.error} onRetry={workspace.reload} />
        ) : null}

        {!workspace.loading && !workspace.error && !workspace.newsletters.length ? (
          <section className="today-empty glass-panel">
            <span><Sparkles size={24} /></span>
            <p className="atelier-eyebrow">Your first thread</p>
            <h2>Turn a question into a learning practice.</h2>
            <p>
              Start with a question. Learnloom will establish the source
              environment, build the path, and prepare your first lesson.
            </p>
            <a className="atelier-primary" href="/newsletters/new">
              Create your first stream <ArrowRight size={16} />
            </a>
          </section>
        ) : null}

        {primary || specialFocus ? (
          <div className="today-grid">
            {reentryFirst ? (
              <article className="today-feature today-feature-reentry glass-panel">
                <div className="today-feature-top">
                  <span className="atelier-chip"><Sparkles size={13} /> A gentle return</span>
                  <span>No backlog to clear</span>
                </div>
                <div className="today-feature-copy">
                  <p className="atelier-eyebrow">Welcome back</p>
                  <h2>Begin with one useful step.</h2>
                  <p>
                    {reason || "Your learning history is still here. Choose one action now; the rest can wait."}
                  </p>
                </div>
                <a
                  className="atelier-primary"
                  href={actionUrl ?? workspace.snapshot?.retention?.actionUrl ?? "/streams"}
                >
                  {actionLabel ?? workspace.snapshot?.retention?.actionLabel ?? "Choose your next step"}
                  <ArrowRight size={16} />
                </a>
                {reentryNewsletter ? (
                  <div className="reentry-controls" aria-label={`Re-entry controls for ${reentryNewsletter.name}`}>
                    <p>Or make the return gentler for <strong>{reentryNewsletter.name}</strong>:</p>
                    <div>
                      <button type="button" disabled={Boolean(reentryBusy)} onClick={() => handleReentryControl("reduce")}>Slow to weekly</button>
                      <button type="button" disabled={Boolean(reentryBusy)} onClick={() => handleReentryControl("pause")}>Pause stream</button>
                      <button type="button" disabled={Boolean(reentryBusy)} onClick={() => handleReentryControl("reset")}>Clear older backlog</button>
                    </div>
                    {reentryNotice ? <small className="reentry-notice">{reentryNotice}</small> : null}
                    {reentryError ? <small className="reentry-error">{reentryError}</small> : null}
                  </div>
                ) : null}
              </article>
            ) : reviewFirst ? (
              <article className="today-feature today-feature-review glass-panel">
                <div className="today-feature-top">
                  <span className="atelier-chip"><BrainCircuit size={13} /> Review due</span>
                  <span>{dueCount || workspace.reviews.length} prompt{(dueCount || workspace.reviews.length) === 1 ? "" : "s"}</span>
                </div>
                <div className="today-feature-copy">
                  <p className="atelier-eyebrow">Best next step</p>
                  <h2>Strengthen what is ready to be recalled.</h2>
                  <p>
                    {reason || "A short retrieval pass now will make recent ideas easier to use later."}
                  </p>
                </div>
                <a className="atelier-primary" href={actionUrl ?? "/review"}>
                  {actionLabel ?? "Start review"} <ArrowRight size={16} />
                </a>
              </article>
            ) : (
              <article className="today-feature glass-panel">
                <div className="today-feature-top">
                  <span className="atelier-chip">
                    {primaryState?.progress > 0 ? "Continue learning" : "Ready for you"}
                  </span>
                  <span><Clock3 size={14} />{primary.newsletter.lessonMinutes} min</span>
                </div>
                <div className="today-feature-copy">
                  <p className="atelier-eyebrow">{primary.newsletter.name}</p>
                  <h2>{primary.title}</h2>
                  <p>
                    {reason || (primaryState?.progress > 0
                      ? "Pick up where you left off. Your place has been saved."
                      : `A source-grounded lesson designed for ${primary.newsletter.learnerLevel}-level learning.`)}
                  </p>
                </div>
                <div className="today-progress">
                  <div>
                    <span>Reading progress</span>
                    <strong>{Math.round(primaryState?.progress ?? 0)}%</strong>
                  </div>
                  <span><i style={{ width: `${primaryState?.progress ?? 0}%` }} /></span>
                </div>
                <a className="atelier-primary" href={actionUrl ?? `/issues/${encodeURIComponent(primary.id)}`}>
                  {actionLabel ?? (primaryState?.progress > 0 ? "Resume lesson" : "Begin lesson")}
                  <ArrowRight size={16} />
                </a>
              </article>
            )}

            <aside className="today-side">
              {sideLesson ? (
                <article className="today-synthesis glass-panel">
                  <span className="atelier-icon"><BookOpen size={17} /></span>
                  <p className="atelier-eyebrow">
                    {reentryFirst ? "Ready when you are" : reviewFirst ? "After review" : "Another thread"}
                  </p>
                  <h3>{sideLesson.newsletter.name}</h3>
                  <p>{sideLesson.title}</p>
                  <a href={`/issues/${encodeURIComponent(sideLesson.id)}`}>
                    Open lesson <ArrowRight size={14} />
                  </a>
                </article>
              ) : (
                <article className="today-synthesis glass-panel">
                  <span className="atelier-icon"><CheckCircle2 size={17} /></span>
                  <p className="atelier-eyebrow">A clear queue</p>
                  <h3>You are caught up.</h3>
                  <p>Your next lesson will appear here when it is ready.</p>
                </article>
              )}
              <article className={`today-review glass-panel${workspace.reviews.length ? " is-due" : ""}`}>
                <BrainCircuit size={18} />
                <div>
                  <p className="atelier-eyebrow">Recall</p>
                  <strong>
                    {workspace.reviews.length
                      ? `${workspace.reviews.length} prompt${workspace.reviews.length === 1 ? "" : "s"} due`
                      : "Strengthen what you learned"}
                  </strong>
                  <span>
                    {workspace.reviews.length
                      ? "A short retrieval pass is ready."
                      : "Review questions from recent lessons."}
                  </span>
                </div>
                <a href="/review" aria-label="Open review"><ArrowRight size={16} /></a>
              </article>
            </aside>
          </div>
        ) : !workspace.loading && !workspace.error && workspace.newsletters.length ? (
          <section className="today-empty glass-panel">
            <span><CheckCircle2 size={24} /></span>
            <p className="atelier-eyebrow">A clear queue</p>
            <h2>You are caught up.</h2>
            <p>Your completed lesson is in the library. The next one will appear here when it is ready.</p>
            <a className="atelier-primary" href="/library">
              Open your library <ArrowRight size={16} />
            </a>
          </section>
        ) : null}

        {!workspace.loading && workspace.newsletters.length ? (
          <section className="today-footer-row">
            <div>
              <p className="atelier-eyebrow">Your rhythm</p>
              <strong>{workspace.newsletters.filter((item) => item.active).length} active learning streams</strong>
              <span>Latest archive update {formatShortDate(workspace.lessons[0]?.createdAt)}</span>
            </div>
            <a href="/streams">Tune your streams <ArrowRight size={15} /></a>
          </section>
        ) : null}
      </section>
    </LearningShell>
  );
}

export function selectTodayLessons(lessons, stateFor = lessonState) {
  const ready = lessons.filter(
    (lesson) =>
      lesson.status === "generated" &&
      lesson.newsletter?.active &&
      !stateFor(lesson.id).completed,
  );
  const primary = ready.find((lesson) => stateFor(lesson.id).progress > 0)
    ?? ready[0];
  return {
    primary,
    secondary: ready.find((lesson) => lesson.id !== primary?.id),
  };
}

export function selectTodayFocus(
  lessons,
  reviews = [],
  stateFor = lessonState,
  retention = undefined,
) {
  const selected = selectTodayLessons(lessons, stateFor);
  const hasInProgressLesson = selected.primary
    ? stateFor(selected.primary.id).progress > 0
    : false;
  return {
    ...selected,
    focus: hasInProgressLesson
      ? "lesson"
      : retention?.inactive
        ? "reentry"
        : reviews.length > 0
          ? "review"
          : "lesson",
  };
}

export function resolveTodaySelection(
  authoritative,
  lessons,
  reviews = [],
  stateFor = lessonState,
  retention = undefined,
) {
  const fallback = selectTodayFocus(lessons, reviews, stateFor, retention);
  if (!authoritative?.kind) return {
    ...fallback,
    reason: "",
    actionLabel: "",
    actionUrl: "",
    dueCount: 0,
  };

  let primary = fallback.primary;
  if (authoritative.kind === "lesson") {
    primary = lessons.find((lesson) => lesson.id === authoritative.subjectId) ?? {
      id: authoritative.subjectId,
      title: authoritative.title,
      status: "generated",
      newsletterId: authoritative.newsletterId,
      newsletter: {
        id: authoritative.newsletterId,
        name: authoritative.newsletterName,
        lessonMinutes: authoritative.lessonMinutes,
        active: true,
      },
    };
  }
  return {
    primary,
    secondary: [fallback.primary, fallback.secondary]
      .find((lesson) => lesson && lesson.id !== primary?.id),
    focus: authoritative.kind === "clear" ? "clear" : authoritative.kind,
    reason: authoritative.reason,
    actionLabel: authoritative.actionLabel,
    actionUrl: authoritative.actionUrl,
    dueCount: authoritative.dueCount ?? 0,
  };
}

function greeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}
