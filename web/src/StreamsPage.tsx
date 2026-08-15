import {
  ArrowRight,
  BookOpen,
  Clock3,
  LibraryBig,
  Pause,
  Plus,
  Sparkles,
} from "lucide-react";
import { useEffect, useState } from "react";
import LearningShell, {
  AtelierError,
  AtelierLoading,
} from "./LearningShell";
import { lessonState } from "./learningState";
import { artworkForStream } from "./todayArtwork";
import { useWorkspace } from "./useWorkspace";
import type { Issue, Newsletter } from "./types";

const heroSizes = "(min-width: 1100px) 40vw, 100vw";
const cardSizes = "(min-width: 1100px) 22vw, 45vw";

interface LessonProgressState {
  progress: number;
  completed: boolean;
}

export interface StreamProgress {
  /** Generated lessons in this stream. */
  lessonCount: number;
  /** Generated lessons marked completed. */
  completedCount: number;
  /** Average progress over generated lessons (completed counts as 100), clamped 0–100. */
  progress: number;
}

/** Mutually exclusive stream classification. */
export type StreamStatus = "active" | "paused" | "completed";

export function generatedLessonsFor(
  newsletter: Newsletter,
  lessons: Issue[],
): Issue[] {
  return lessons.filter(
    (lesson) =>
      lesson.newsletterId === newsletter.id && lesson.status === "generated",
  );
}

function clamp(value: number) {
  return Math.min(100, Math.max(0, value));
}

/**
 * Defensible stream summary: the average progress across generated
 * lessons, treating a completed lesson as 100, clamped to 0–100.
 */
export function streamProgress(
  newsletter: Newsletter,
  lessons: Issue[],
  stateFor: (issueId: string) => LessonProgressState = lessonState,
): StreamProgress {
  const streamLessons = generatedLessonsFor(newsletter, lessons);
  if (!streamLessons.length) {
    return { lessonCount: 0, completedCount: 0, progress: 0 };
  }
  let total = 0;
  let completedCount = 0;
  for (const lesson of streamLessons) {
    const state = stateFor(lesson.id);
    if (state.completed) {
      total += 100;
      completedCount += 1;
    } else {
      total += clamp(state.progress ?? 0);
    }
  }
  return {
    lessonCount: streamLessons.length,
    completedCount,
    progress: Math.round(clamp(total / streamLessons.length)),
  };
}

/**
 * A stream counts as completed only when it has at least one generated
 * lesson and every generated lesson is completed. Otherwise it is
 * active when the stream is active, paused otherwise.
 */
export function streamStatus(
  newsletter: Newsletter,
  lessons: Issue[],
  stateFor: (issueId: string) => LessonProgressState = lessonState,
): StreamStatus {
  const streamLessons = generatedLessonsFor(newsletter, lessons);
  const completed =
    streamLessons.length > 0 &&
    streamLessons.every((lesson) => stateFor(lesson.id).completed);
  if (completed) return "completed";
  return newsletter.active ? "active" : "paused";
}

/** Partitions streams into exclusive active/paused/completed lists, sorted by name. */
export function classifyStreams(
  newsletters: Newsletter[],
  lessons: Issue[],
  stateFor: (issueId: string) => LessonProgressState = lessonState,
): { active: Newsletter[]; paused: Newsletter[]; completed: Newsletter[] } {
  const active: Newsletter[] = [];
  const paused: Newsletter[] = [];
  const completed: Newsletter[] = [];
  for (const newsletter of newsletters) {
    const status = streamStatus(newsletter, lessons, stateFor);
    if (status === "active") active.push(newsletter);
    else if (status === "paused") paused.push(newsletter);
    else completed.push(newsletter);
  }
  const byName = (left: Newsletter, right: Newsletter) =>
    left.name.localeCompare(right.name);
  return {
    active: active.sort(byName),
    paused: paused.sort(byName),
    completed: completed.sort(byName),
  };
}

function hasIncompleteLesson(
  newsletter: Newsletter,
  lessons: Issue[],
  stateFor: (issueId: string) => LessonProgressState,
) {
  return generatedLessonsFor(newsletter, lessons).some(
    (lesson) => !stateFor(lesson.id).completed,
  );
}

function hasStartedLesson(
  newsletter: Newsletter,
  lessons: Issue[],
  stateFor: (issueId: string) => LessonProgressState,
) {
  return generatedLessonsFor(newsletter, lessons).some((lesson) => {
    const state = stateFor(lesson.id);
    return !state.completed && (state.progress ?? 0) > 0;
  });
}

/**
 * Stable ordering for active streams: streams with an incomplete
 * generated lesson first, then in-progress lessons, then name so the
 * featured pick and card order never flicker between reloads.
 */
export function rankActiveStreams(
  newsletters: Newsletter[],
  lessons: Issue[],
  excludeNewsletterId?: string | null,
  stateFor: (issueId: string) => LessonProgressState = lessonState,
): Newsletter[] {
  return newsletters
    .filter(
      (newsletter) =>
        streamStatus(newsletter, lessons, stateFor) === "active" &&
        newsletter.id !== excludeNewsletterId,
    )
    .sort((left, right) => {
      const leftIncomplete = hasIncompleteLesson(left, lessons, stateFor);
      const rightIncomplete = hasIncompleteLesson(right, lessons, stateFor);
      if (leftIncomplete !== rightIncomplete) return leftIncomplete ? -1 : 1;
      const leftStarted = hasStartedLesson(left, lessons, stateFor);
      const rightStarted = hasStartedLesson(right, lessons, stateFor);
      if (leftStarted !== rightStarted) return leftStarted ? -1 : 1;
      return left.name.localeCompare(right.name);
    });
}

/**
 * Featured current stream: the todayFocus stream when it is still an
 * active stream, otherwise a stable active fallback preferring streams
 * with incomplete generated lessons or in-progress work.
 */
export function selectFeaturedStream(
  newsletters: Newsletter[],
  lessons: Issue[],
  todayFocusNewsletterId?: string | null,
  stateFor: (issueId: string) => LessonProgressState = lessonState,
): Newsletter | undefined {
  const active = classifyStreams(newsletters, lessons, stateFor).active;
  if (!active.length) return undefined;
  const focused = active.find(
    (newsletter) => newsletter.id === todayFocusNewsletterId,
  );
  if (focused) return focused;
  return rankActiveStreams(active, lessons, null, stateFor)[0];
}

/** Most recent generated lesson that is still unfinished, if any. */
export function latestIncompleteLesson(
  newsletter: Newsletter,
  lessons: Issue[],
  stateFor: (issueId: string) => LessonProgressState = lessonState,
): Issue | undefined {
  return generatedLessonsFor(newsletter, lessons)
    .filter((lesson) => !stateFor(lesson.id).completed)
    .sort(
      (left, right) =>
        new Date(right.createdAt ?? 0).getTime() -
        new Date(left.createdAt ?? 0).getTime(),
    )[0];
}

export default function StreamsPage() {
  const workspace = useWorkspace();
  const [, refreshState] = useState(0);

  useEffect(() => {
    const refresh = () => refreshState((value) => value + 1);
    window.addEventListener("learnloom:state", refresh);
    return () => window.removeEventListener("learnloom:state", refresh);
  }, []);

  const { newsletters, lessons } = workspace;

  const classified = classifyStreams(newsletters, lessons);
  const featured = selectFeaturedStream(
    newsletters,
    lessons,
    workspace.snapshot?.todayFocus?.newsletterId,
  );

  const featuredArt = featured ? artworkForStream(featured.id) : null;
  const featuredProgress = featured ? streamProgress(featured, lessons) : null;
  const featuredLesson = featured ? latestIncompleteLesson(featured, lessons) : undefined;

  const activeEntries = rankActiveStreams(
    classified.active,
    lessons,
    featured?.id,
  ).map((newsletter) => ({
    newsletter,
    progress: streamProgress(newsletter, lessons),
    art: artworkForStream(newsletter.id),
  }));

  const paused = classified.paused;
  const completed = classified.completed;

  return (
    <LearningShell active="streams" variant="streams">
      <section className="streams-page">
        <header className="streams-heading">
          <div>
            <p className="atelier-eyebrow">Your learning journeys</p>
            <h1>Streams</h1>
            <p>
              Learning streams follow the questions you care about — one
              lesson at a time.
            </p>
          </div>
          <a className="atelier-primary" href="/newsletters/new">
            <Plus size={16} /> New learning stream
          </a>
        </header>

        {workspace.loading ? <AtelierLoading /> : null}
        {workspace.error ? (
          <AtelierError message={workspace.error} onRetry={workspace.reload} />
        ) : null}

        {!workspace.loading && !workspace.error && !newsletters.length ? (
          <section className="streams-empty">
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

        {featured && featuredArt ? (
          <article className="streams-hero">
            <div className="streams-hero-copy">
              <p className="atelier-eyebrow">Current stream</p>
              <h2>{featured.name}</h2>
              <p className="streams-hero-deck">{featured.topic}</p>
              <div className="streams-hero-progress">
                <div>
                  <strong>{featuredProgress?.progress ?? 0}% complete</strong>
                  <span>
                    {featuredProgress?.lessonCount ?? 0} lesson
                    {featuredProgress?.lessonCount === 1 ? "" : "s"}
                    {featured.lessonMinutes
                      ? ` · ${featured.lessonMinutes} min each`
                      : ""}
                  </span>
                </div>
                <span className="streams-hero-track">
                  <i style={{ width: `${featuredProgress?.progress ?? 0}%` }} />
                </span>
              </div>
              <a
                className="atelier-primary"
                href={
                  featuredLesson
                    ? `/issues/${encodeURIComponent(featuredLesson.id)}`
                    : `/newsletters/${encodeURIComponent(featured.id)}`
                }
              >
                {featuredLesson ? "Continue stream" : "Open stream"}
                <ArrowRight size={16} />
              </a>
            </div>
            <div className="streams-hero-art">
              <img
                src={featuredArt.hero}
                srcSet={featuredArt.heroSrcSet}
                sizes={heroSizes}
                alt=""
                loading="eager"
                fetchPriority="high"
                decoding="async"
              />
            </div>
          </article>
        ) : null}

        {activeEntries.length ? (
          <section
            className="streams-section"
            aria-labelledby="streams-active-heading"
          >
            <div className="streams-section-head">
              <h2 id="streams-active-heading">Active streams</h2>
            </div>
            <div className="streams-grid">
              {activeEntries.map(({ newsletter, progress, art }) => (
                <a
                  className="streams-card"
                  href={`/newsletters/${encodeURIComponent(newsletter.id)}`}
                  key={newsletter.id}
                >
                  <img
                    src={art.card}
                    srcSet={art.cardSrcSet}
                    sizes={cardSizes}
                    alt=""
                    loading="lazy"
                    decoding="async"
                  />
                  <div className="streams-card-copy">
                    <h3>{newsletter.name}</h3>
                    <p className="streams-card-topic">{newsletter.topic}</p>
                    <div className="streams-card-meta">
                      <span>
                        <BookOpen size={12} /> {progress.lessonCount} lesson
                        {progress.lessonCount === 1 ? "" : "s"}
                      </span>
                      {newsletter.lessonMinutes ? (
                        <span>
                          <Clock3 size={12} /> {newsletter.lessonMinutes} min
                        </span>
                      ) : null}
                    </div>
                    <div className="streams-card-progress">
                      <span>{progress.progress}%</span>
                      <span className="streams-card-track">
                        <i style={{ width: `${progress.progress}%` }} />
                      </span>
                    </div>
                  </div>
                </a>
              ))}
            </div>
          </section>
        ) : null}

        {paused.length || completed.length ? (
          <section
            className="streams-section"
            aria-labelledby="streams-inactive-heading"
          >
            <div className="streams-section-head">
              <h2 id="streams-inactive-heading">Paused &amp; completed</h2>
            </div>
            <div className="streams-row-list">
              {paused.map((newsletter) => (
                <a
                  className="streams-row"
                  href={`/newsletters/${encodeURIComponent(newsletter.id)}`}
                  key={newsletter.id}
                >
                  <span className="atelier-icon"><Pause size={16} /></span>
                  <div className="streams-row-copy">
                    <h3>{newsletter.name}</h3>
                  </div>
                  <span className="streams-row-status">
                    Paused · {streamProgress(newsletter, lessons).progress}%
                  </span>
                  <ArrowRight className="streams-row-arrow" size={16} />
                </a>
              ))}
              {completed.map((newsletter) => (
                <a
                  className="streams-row"
                  href={`/newsletters/${encodeURIComponent(newsletter.id)}`}
                  key={newsletter.id}
                >
                  <span className="atelier-icon"><LibraryBig size={16} /></span>
                  <div className="streams-row-copy">
                    <h3>{newsletter.name}</h3>
                  </div>
                  <span className="streams-row-status">Completed · 100%</span>
                  <ArrowRight className="streams-row-arrow" size={16} />
                </a>
              ))}
            </div>
          </section>
        ) : null}
      </section>
    </LearningShell>
  );
}
