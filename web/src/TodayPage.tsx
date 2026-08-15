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
import { artworkForStream } from "./todayArtwork";
import { useWorkspace } from "./useWorkspace";
import type { Issue, Newsletter } from "./types";

const cardSizes = "(min-width: 1100px) 23vw, 45vw";
const heroSizes = "(min-width: 1100px) 40vw, 100vw";

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

  const heroStreamId = !specialFocus
    ? primary?.newsletter?.id
    : reviewFirst
      ? sideLesson?.newsletter?.id
      : reentryNewsletter?.id;
  const heroArt = artworkForStream(
    !specialFocus
      ? heroStreamId
      : reviewFirst
        ? workspace.lessons.find((lesson) => lesson.id === workspace.reviews[0]?.issueId)
            ?.newsletterId ?? heroStreamId
        : heroStreamId,
  );
  const anotherThread = useMemo(
    () => selectAnotherThread(workspace.newsletters, workspace.lessons, heroStreamId, lessonState),
    [workspace.newsletters, workspace.lessons, heroStreamId],
  );
  const anotherThreadArt = artworkForStream(anotherThread?.id);
  const streamEntries = useMemo(
    () => {
      const candidates = rankTodayStreams(
        workspace.newsletters,
        workspace.lessons,
        heroStreamId,
        lessonState,
        4,
      );
      const withoutSideCard = candidates.filter(
        (entry) => entry.newsletter.id !== anotherThread?.id,
      );
      return (withoutSideCard.length ? withoutSideCard : candidates).slice(0, 3);
    },
    [workspace.newsletters, workspace.lessons, heroStreamId, anotherThread?.id],
  );
  const activeStreamCount = workspace.newsletters.filter((item) => item.active).length;

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
    <LearningShell active="today" redesigned>
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
          <section className="today-empty">
            <span className="atelier-icon"><Sparkles size={24} /></span>
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
              <article className="today-hero today-hero-reentry">
                <div className="today-hero-copy">
                  <div className="today-hero-meta">
                    <span className="atelier-chip"><Sparkles size={13} /> A gentle return</span>
                    <span className="today-hero-time">No backlog to clear</span>
                  </div>
                  <p className="atelier-eyebrow">Welcome back</p>
                  <h2>Begin with one useful step.</h2>
                  <p className="today-hero-deck">
                    {reason || "Your learning history is still here. Choose one action now; the rest can wait."}
                  </p>
                  <a
                    className="atelier-primary"
                    href={actionUrl || workspace.snapshot?.retention?.actionUrl || "/streams"}
                  >
                    {actionLabel || workspace.snapshot?.retention?.actionLabel || "Choose your next step"}
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
                </div>
                <div className="today-hero-art">
                  <img
                    src={heroArt.hero}
                    srcSet={heroArt.heroSrcSet}
                    sizes={heroSizes}
                    alt=""
                    loading="eager"
                    fetchPriority="high"
                    decoding="async"
                  />
                </div>
              </article>
            ) : reviewFirst ? (
              <article className="today-hero today-hero-review">
                <div className="today-hero-copy">
                  <div className="today-hero-meta">
                    <span className="atelier-chip"><BrainCircuit size={13} /> Review due</span>
                    <span className="today-hero-time">{dueCount || workspace.reviews.length} prompt{(dueCount || workspace.reviews.length) === 1 ? "" : "s"}</span>
                  </div>
                  <p className="atelier-eyebrow">Best next step</p>
                  <h2>Strengthen what is ready to be recalled.</h2>
                  <p className="today-hero-deck">
                    {reason || "A short retrieval pass now will make recent ideas easier to use later."}
                  </p>
                  <a className="atelier-primary" href={actionUrl || "/review"}>
                    {actionLabel || "Start review"} <ArrowRight size={16} />
                  </a>
                </div>
                <div className="today-hero-art">
                  <img
                    src={heroArt.hero}
                    srcSet={heroArt.heroSrcSet}
                    sizes={heroSizes}
                    alt=""
                    loading="eager"
                    fetchPriority="high"
                    decoding="async"
                  />
                </div>
              </article>
            ) : (
              <article className="today-hero">
                <div className="today-hero-copy">
                  <div className="today-hero-meta">
                    <span className="atelier-chip">
                      {primaryState?.progress > 0 ? "Continue learning" : "Ready for you"}
                    </span>
                    {primary.newsletter.lessonMinutes ? (
                      <span className="today-hero-time"><Clock3 size={14} />{primary.newsletter.lessonMinutes} min</span>
                    ) : null}
                  </div>
                  <p className="atelier-eyebrow">{primary.newsletter.name}</p>
                  <h2>{primary.title}</h2>
                  <p className="today-hero-deck">
                    {reason || (primaryState?.progress > 0
                      ? "Pick up where you left off. Your place has been saved."
                      : primary.newsletter.learnerLevel
                        ? `A source-grounded lesson designed for ${primary.newsletter.learnerLevel}-level learning.`
                        : "A source-grounded lesson prepared for your learning path.")}
                  </p>
                  <div className="today-hero-progress">
                    <div>
                      <span>Reading progress</span>
                      <strong>{Math.round(primaryState?.progress ?? 0)}%</strong>
                    </div>
                    <span className="today-hero-track"><i style={{ width: `${primaryState?.progress ?? 0}%` }} /></span>
                  </div>
                  <a className="atelier-primary" href={actionUrl || `/issues/${encodeURIComponent(primary.id)}`}>
                    {actionLabel || (primaryState?.progress > 0 ? "Resume lesson" : "Begin lesson")}
                    <ArrowRight size={16} />
                  </a>
                </div>
                <div className="today-hero-art">
                  <img
                    src={heroArt.hero}
                    srcSet={heroArt.heroSrcSet}
                    sizes={heroSizes}
                    alt=""
                    loading="eager"
                    fetchPriority="high"
                    decoding="async"
                  />
                </div>
              </article>
            )}

            <aside className="today-side" aria-label="Today's secondary actions">
              {anotherThread ? (
                <article className="today-thread-card">
                  <div className="today-thread-copy">
                    <span className="atelier-icon"><BookOpen size={17} /></span>
                    <p className="atelier-eyebrow">Another thread</p>
                    <h3>{anotherThread.name}</h3>
                    <p className="today-thread-topic">{anotherThread.topic}</p>
                    <a className="today-card-link" href={`/newsletters/${encodeURIComponent(anotherThread.id)}`}>
                      Open stream <ArrowRight size={14} />
                    </a>
                  </div>
                  <img
                    className="today-thread-art"
                    src={anotherThreadArt.card}
                    srcSet={anotherThreadArt.cardSrcSet}
                    sizes="(min-width: 1100px) 16vw, 40vw"
                    alt=""
                    loading="lazy"
                    decoding="async"
                  />
                </article>
              ) : (
                <article className="today-thread-card is-clear">
                  <span className="atelier-icon"><CheckCircle2 size={17} /></span>
                  <p className="atelier-eyebrow">A clear queue</p>
                  <h3>You are caught up.</h3>
                  <p className="today-thread-topic">No other active streams need attention right now.</p>
                </article>
              )}
              <article className={`today-recall-card${workspace.reviews.length ? " is-due" : ""}`}>
                <span className="atelier-icon"><BrainCircuit size={18} /></span>
                <div className="today-recall-copy">
                  <p className="atelier-eyebrow">Recall</p>
                  <strong>
                    {workspace.reviews.length
                      ? `${workspace.reviews.length} prompt${workspace.reviews.length === 1 ? "" : "s"} due`
                      : "Nothing due right now"}
                  </strong>
                  <span>
                    {workspace.reviews.length
                      ? "A short retrieval pass is ready."
                      : "Review questions from recent lessons will appear here."}
                  </span>
                </div>
                <div className="today-recall-count" aria-label={`${workspace.reviews.length} review prompts due`}>
                  <strong>{workspace.reviews.length}</strong>
                  <span>Due</span>
                </div>
                <a href="/review" aria-label="Open review"><ArrowRight size={16} /></a>
              </article>
            </aside>
          </div>
        ) : !workspace.loading && !workspace.error && workspace.newsletters.length ? (
          <section className="today-empty">
            <span className="atelier-icon"><CheckCircle2 size={24} /></span>
            <p className="atelier-eyebrow">A clear queue</p>
            <h2>You are caught up.</h2>
            <p>Your completed lesson is in the library. The next one will appear here when it is ready.</p>
            <a className="atelier-primary" href="/library">
              Open your library <ArrowRight size={16} />
            </a>
          </section>
        ) : null}

        {!workspace.loading && workspace.newsletters.length ? (
          <section className="today-streams" aria-labelledby="today-streams-heading">
            <div className="today-streams-head">
              <div>
                <p className="atelier-eyebrow" id="today-streams-heading">Your rhythm</p>
                <h2>Active learning streams</h2>
                <p>{activeStreamCount} active · Latest update {formatShortDate(workspace.lessons[0]?.createdAt)}</p>
              </div>
              <a className="today-view-all" href="/streams">
                View all streams <ArrowRight size={15} />
              </a>
            </div>
            {streamEntries.length ? (
              <div className="today-streams-row">
                {streamEntries.map((entry) => {
                  const art = artworkForStream(entry.newsletter.id);
                  const href = `/issues/${encodeURIComponent(entry.lesson.id)}`;
                  return (
                    <a className="today-stream-card" href={href} key={entry.newsletter.id}>
                      <img
                        src={art.card}
                        srcSet={art.cardSrcSet}
                        sizes={cardSizes}
                        alt=""
                        loading="lazy"
                        decoding="async"
                      />
                      <h3>{entry.newsletter.name}</h3>
                      <p className="today-stream-lesson">{entry.lesson.title}</p>
                      <div className="today-stream-meta">
                        <span>{entry.progress > 0 ? `${Math.round(entry.progress)}% complete` : "Not started"}</span>
                        {entry.remainingMinutes ? <span>{entry.remainingMinutes} min left</span> : null}
                      </div>
                      <span className="today-stream-track" aria-hidden="true">
                        <i style={{ width: `${entry.progress}%` }} />
                      </span>
                    </a>
                  );
                })}
              </div>
            ) : null}
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

export interface TodayStreamEntry {
  newsletter: Newsletter;
  /** Most recent generated lesson that is still unfinished. */
  lesson: Issue;
  progress: number;
  remainingMinutes?: number;
}

/**
 * Ranks the other active streams for the bottom strip: streams with an
 * unfinished generated lesson first, ordered by how recent that lesson is,
 * then in-progress lessons, then name for determinism. Streams without any
 * unfinished generated lesson are excluded — their progress is not shown
 * because there is nothing real to measure.
 */
export function rankTodayStreams(
  newsletters: Newsletter[],
  lessons: Issue[],
  excludeNewsletterId?: string | null,
  stateFor: (issueId: string) => { progress: number; completed: boolean } = lessonState,
  limit = 3,
): TodayStreamEntry[] {
  const entries = newsletters
    .filter((newsletter) => newsletter.active && newsletter.id !== excludeNewsletterId)
    .map((newsletter): TodayStreamEntry | null => {
      const streamLessons = lessons
        .filter(
          (lesson) =>
            lesson.newsletterId === newsletter.id &&
            lesson.status === "generated" &&
            !stateFor(lesson.id).completed,
        )
        .sort(
          (left, right) =>
            new Date(right.createdAt ?? 0).getTime() - new Date(left.createdAt ?? 0).getTime(),
        );
      const lesson = streamLessons[0];
      if (!lesson) return null;
      const progress = stateFor(lesson.id).progress;
      return {
        newsletter,
        lesson,
        progress,
        remainingMinutes:
          newsletter.lessonMinutes && progress < 100
            ? Math.max(1, Math.round(newsletter.lessonMinutes * (1 - progress / 100)))
            : undefined,
      };
    })
    .filter((entry): entry is TodayStreamEntry => entry !== null)
    .sort((left, right) => {
      const leftTime = new Date(left.lesson.createdAt ?? 0).getTime();
      const rightTime = new Date(right.lesson.createdAt ?? 0).getTime();
      if (leftTime !== rightTime) return rightTime - leftTime;
      if ((left.progress > 0 ? 1 : 0) !== (right.progress > 0 ? 1 : 0)) {
        return right.progress > 0 ? 1 : -1;
      }
      return left.newsletter.name.localeCompare(right.newsletter.name);
    });
  return entries.slice(0, limit);
}

/**
 * Highest-priority active stream other than the hero stream for the
 * "Another thread" card. Prefers the top of the strip ranking; falls back
 * to any remaining active stream by name so a brand-new thread is still
 * reachable.
 */
export function selectAnotherThread(
  newsletters: Newsletter[],
  lessons: Issue[],
  heroNewsletterId?: string | null,
  stateFor: (issueId: string) => { progress: number; completed: boolean } = lessonState,
): Newsletter | undefined {
  const ranked = rankTodayStreams(newsletters, lessons, heroNewsletterId, stateFor, 1);
  if (ranked[0]) return ranked[0].newsletter;
  return newsletters
    .filter((newsletter) => newsletter.active && newsletter.id !== heroNewsletterId)
    .sort((left, right) => left.name.localeCompare(right.name))[0];
}

function greeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}
