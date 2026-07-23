import {
  ArrowRight,
  BookOpen,
  Clock3,
  Pause,
  Plus,
  Search,
  Sparkles,
} from "lucide-react";
import { useMemo, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell.jsx";
import { useWorkspace } from "./useWorkspace.js";

export default function StreamsPage() {
  const workspace = useWorkspace({ includeDetails: false });
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("all");
  const visible = useMemo(
    () =>
      workspace.newsletters.filter((newsletter) => {
        const text = `${newsletter.name} ${newsletter.topic}`.toLowerCase();
        const matchesSearch = text.includes(query.trim().toLowerCase());
        const matchesFilter =
          filter === "all" ||
          (filter === "active" && newsletter.active) ||
          (filter === "paused" && !newsletter.active);
        return matchesSearch && matchesFilter;
      }),
    [workspace.newsletters, query, filter],
  );

  return (
    <LearningShell active="streams">
      <section className="atelier-page streams-page">
        <header className="atelier-page-heading with-actions">
          <div>
            <p className="atelier-eyebrow">Subjects you are following</p>
            <h1>Learning streams</h1>
            <p>Tune the questions, sources, and rhythm behind your lessons.</p>
          </div>
          <a className="atelier-primary" href="/newsletters/new">
            <Plus size={15} /> New stream
          </a>
        </header>

        <div className="contextual-toolbar">
          <div className="atelier-filter-row" role="group" aria-label="Filter streams">
            {["all", "active", "paused"].map((value) => (
              <button
                className={filter === value ? "current" : ""}
                type="button"
                onClick={() => setFilter(value)}
                key={value}
              >
                {value[0].toUpperCase() + value.slice(1)}
              </button>
            ))}
          </div>
          <label className="contextual-search">
            <Search size={15} />
            <span className="sr-only">Search learning streams</span>
            <input
              type="search"
              placeholder="Search learning streams"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
        </div>

        {workspace.loading ? <AtelierLoading /> : null}
        {workspace.error ? <AtelierError message={workspace.error} onRetry={workspace.reload} /> : null}
        {!workspace.loading && !visible.length ? (
          <div className="atelier-state-card">
            <Search size={20} />
            <strong>No streams match this view.</strong>
            <p>Try a different search or filter.</p>
          </div>
        ) : null}

        <div className="stream-grid">
          {visible.map((newsletter) => (
            <article className="stream-card glass-panel" key={newsletter.id}>
              <div className="stream-card-top">
                <span className="atelier-icon">
                  {newsletter.active ? <Sparkles size={17} /> : <Pause size={17} />}
                </span>
                <span className={`atelier-status ${newsletter.active ? "active" : ""}`}>
                  {newsletter.active ? "Active" : "Paused"}
                </span>
              </div>
              <p className="atelier-eyebrow">{newsletter.learnerLevel} practice</p>
              <h2>{newsletter.name}</h2>
              <p>{newsletter.topic}</p>
              <dl>
                <div>
                  <dt><Clock3 size={13} /> Next rhythm</dt>
                  <dd>{newsletter.active ? `Daily at ${newsletter.scheduleTime}` : "Not scheduled"}</dd>
                </div>
                <div>
                  <dt><BookOpen size={13} /> Learning history</dt>
                  <dd>{newsletter.generatedCount} lessons</dd>
                </div>
              </dl>
              <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
                Open stream <ArrowRight size={15} />
              </a>
            </article>
          ))}
          {!query && filter === "all" ? (
            <a className="stream-create-card" href="/newsletters/new">
              <Plus size={20} />
              <strong>Follow another question</strong>
              <span>Build a new learning thread from sources you trust.</span>
            </a>
          ) : null}
        </div>
      </section>
    </LearningShell>
  );
}
