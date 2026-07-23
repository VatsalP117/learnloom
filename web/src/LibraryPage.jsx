import {
  ArrowRight,
  BookOpen,
  CheckCircle2,
  Clock3,
  Search,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import LearningShell, {
  AtelierError,
  AtelierLoading,
  formatShortDate,
} from "./LearningShell.jsx";
import { lessonState } from "./learningState.js";
import { useWorkspace } from "./useWorkspace.js";

export default function LibraryPage() {
  const workspace = useWorkspace();
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("all");
  const [, refreshState] = useState(0);

  useEffect(() => {
    const refresh = () => refreshState((value) => value + 1);
    window.addEventListener("learnloom:state", refresh);
    return () => window.removeEventListener("learnloom:state", refresh);
  }, []);

  const lessons = useMemo(
    () =>
      workspace.lessons.filter((lesson) => {
        if (lesson.status !== "generated") return false;
        const state = lessonState(lesson.id);
        const matchesFilter =
          filter === "all" ||
          (filter === "completed" && state.completed) ||
          (filter === "unread" && !state.completed && !state.progress) ||
          (filter === "in-progress" && !state.completed && state.progress > 0);
        const text = `${lesson.title} ${lesson.newsletter.name} ${lesson.newsletter.topic}`.toLowerCase();
        return matchesFilter && text.includes(query.trim().toLowerCase());
      }),
    [workspace.lessons, query, filter],
  );

  return (
    <LearningShell active="library">
      <section className="atelier-page library-page">
        <header className="atelier-page-heading">
          <p className="atelier-eyebrow">Your lasting archive</p>
          <h1>Library</h1>
          <p>Find a lesson again by title, stream, or subject.</p>
        </header>

        <div className="contextual-toolbar">
          <div className="atelier-filter-row" role="group" aria-label="Filter lessons">
            {[
              ["all", "All lessons"],
              ["unread", "Unread"],
              ["in-progress", "In progress"],
              ["completed", "Completed"],
            ].map(([value, label]) => (
              <button
                className={filter === value ? "current" : ""}
                type="button"
                onClick={() => setFilter(value)}
                key={value}
              >
                {label}
              </button>
            ))}
          </div>
          <label className="contextual-search">
            <Search size={15} />
            <span className="sr-only">Search lessons and topics</span>
            <input
              type="search"
              placeholder="Search lessons and topics"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
        </div>

        {workspace.loading ? <AtelierLoading label="Opening your library…" /> : null}
        {workspace.error ? <AtelierError message={workspace.error} onRetry={workspace.reload} /> : null}
        {!workspace.loading && !lessons.length ? (
          <div className="atelier-state-card">
            <Search size={20} />
            <strong>No lessons found.</strong>
            <p>Try another term or choose a different reading state.</p>
          </div>
        ) : null}

        <div className="lesson-library-grid">
          {lessons.map((lesson) => {
            const state = lessonState(lesson.id);
            return (
              <article className="lesson-library-card glass-panel" key={lesson.id}>
                <div className="lesson-library-meta">
                  <span className="atelier-chip">{lesson.newsletter.name}</span>
                  {state.completed ? (
                    <span><CheckCircle2 size={14} /> Completed</span>
                  ) : state.progress ? (
                    <span>{Math.round(state.progress)}% read</span>
                  ) : (
                    <span>Unread</span>
                  )}
                </div>
                <h2>{lesson.title}</h2>
                <p>{lesson.newsletter.topic}</p>
                <div className="lesson-library-footer">
                  <span><Clock3 size={13} />{lesson.newsletter.lessonMinutes} min</span>
                  <span><BookOpen size={13} />{formatShortDate(lesson.createdAt)}</span>
                  <a href={`/issues/${encodeURIComponent(lesson.id)}`}>
                    {state.progress ? "Continue" : "Open"} <ArrowRight size={14} />
                  </a>
                </div>
                {state.progress && !state.completed ? (
                  <div className="library-progress"><i style={{ width: `${state.progress}%` }} /></div>
                ) : null}
              </article>
            );
          })}
        </div>
        {workspace.hasMore ? (
          <div className="library-load-more">
            <button
              className="atelier-primary"
              type="button"
              disabled={workspace.loadingMore}
              onClick={workspace.loadMore}
            >
              {workspace.loadingMore ? "Loading older lessons…" : "Load older lessons"}
            </button>
          </div>
        ) : null}
      </section>
    </LearningShell>
  );
}
