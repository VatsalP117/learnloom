import {
  BookOpen,
  Clock3,
  Pause,
  Plus,
  Search,
  Sparkles,
} from "lucide-react";
import { useMemo, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell";
import { useWorkspace } from "./useWorkspace";
import type { Newsletter } from "./types";

export default function StreamsPage() {
  const workspace = useWorkspace();
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
            <a
              className="stream-card glass-panel"
              href={`/newsletters/${encodeURIComponent(newsletter.id)}`}
              key={newsletter.id}
            >
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
                  <dd>{newsletter.active ? streamRhythmSummary(newsletter) : "Not scheduled"}</dd>
                </div>
                <div>
                  <dt><BookOpen size={13} /> Capability path</dt>
                  <dd>{streamCapabilitySummary(newsletter)}</dd>
                </div>
              </dl>
            </a>
          ))}
          {!query && filter === "all" ? (
            <a className="stream-create-card" href="/newsletters/new">
              <Plus size={20} />
              <strong>Follow another question</strong>
              <span>Give Learnloom a question and build a connected learning path.</span>
            </a>
          ) : null}
        </div>
      </section>
    </LearningShell>
  );
}

export function streamCapabilitySummary(newsletter: Newsletter) {
  const capabilities = newsletter.capabilityCount ?? 0;
  const recalled = newsletter.recalledCapabilityCount ?? 0;
  if (recalled > 0) return `${recalled} recalled · ${capabilities} established`;
  if (capabilities > 0) return `${capabilities} established · recall next`;
  return "First milestone ahead";
}

export function streamRhythmSummary(newsletter: Newsletter) {
  const time = newsletter.scheduleTime ?? "08:00";
  if (newsletter.rhythmThrottledAt) return `Slowed to weekly · ${time}`;
  const mode = newsletter.effectiveRhythmMode ?? newsletter.rhythmMode ?? "daily";
  if (mode === "evidence_led") return `Evidence-led · ${time}`;
  if (mode === "weekly_synthesis") return `Weekly synthesis · ${time}`;
  if (mode === "selected_weekdays") {
    return `${newsletter.selectedWeekdays?.length ?? 1} days/week · ${time}`;
  }
  return `Daily · ${time}`;
}
