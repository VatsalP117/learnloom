import {
  CalendarDays,
  Clock3,
  Search,
} from "lucide-react";
import { useState } from "react";
import LearningShell, {
  AtelierError,
  AtelierLoading,
  formatShortDate,
} from "./LearningShell";
import { type LibraryFilter, useLibrary } from "./useLibrary";

export default function LibraryPage() {
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<LibraryFilter>("all");
  const library = useLibrary(query, filter);

  return (
    <LearningShell active="library">
      <section className="atelier-page library-page">
        <header className="atelier-page-heading">
          <p className="atelier-eyebrow">Your lasting archive</p>
          <h1>Library</h1>
          <p>Find a lesson by title, concept, source, retrieval question, or stream.</p>
        </header>

        <div className="contextual-toolbar">
          <div className="atelier-filter-row" role="group" aria-label="Filter lessons">
            {([
              ["all", "All lessons"],
              ["unread", "Unread"],
              ["in-progress", "In progress"],
              ["completed", "Completed"],
            ] as Array<[LibraryFilter, string]>).map(([value, label]) => (
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
            <span className="sr-only">Search lessons, concepts, sources, and retrieval prompts</span>
            <input
              type="search"
              placeholder="Search concepts, sources, or questions"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
        </div>

        {library.loading ? <AtelierLoading label="Opening your library…" /> : null}
        {library.error ? <AtelierError message={library.error} onRetry={library.reload} /> : null}
        {!library.loading && !library.lessons.length ? (
          <div className="atelier-state-card">
            <Search size={20} />
            <strong>No lessons found.</strong>
            <p>Try another term or choose a different reading state.</p>
          </div>
        ) : null}

        <div className="lesson-library-grid">
          {library.lessons.map((lesson) => (
            <a
              className="lesson-library-card glass-panel"
              href={`/issues/${encodeURIComponent(lesson.id)}`}
              key={lesson.id}
            >
              <div className="lesson-library-meta">
                <span className="atelier-chip">{lesson.newsletter.name}</span>
              </div>
              <h2>{lesson.title}</h2>
              <div className="lesson-library-footer">
                <span><Clock3 size={13} />{lesson.newsletter.lessonMinutes} min</span>
                <span><CalendarDays size={13} />Generated {formatShortDate(lesson.createdAt)}</span>
              </div>
            </a>
          ))}
        </div>
        {library.hasMore ? (
          <div className="library-load-more">
            <button
              className="atelier-primary"
              type="button"
              disabled={library.loadingMore}
              onClick={library.loadMore}
            >
              {library.loadingMore ? "Loading older lessons…" : "Load older lessons"}
            </button>
          </div>
        ) : null}
      </section>
    </LearningShell>
  );
}
