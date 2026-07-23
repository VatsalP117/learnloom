import {
  Atom,
  BookOpen,
  BrainCircuit,
  ChevronRight,
  Clock3,
  LibraryBig,
  Menu,
  Plus,
  Search,
  Sparkles,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";
import IssueDetail from "./IssueDetail.jsx";
import NewsletterDetail from "./NewsletterDetail.jsx";
import NewsletterCreate from "./NewsletterCreate.jsx";
import { apiJSON } from "./api.js";

const iconCycle = [Atom, BrainCircuit, BookOpen, LibraryBig];

function App({ capabilities = {} }) {
  if (window.location.pathname === "/newsletters/new") {
    return <NewsletterCreate sourceDiscovery={Boolean(capabilities.sourceDiscovery)} />;
  }
  const detailMatch = /^\/newsletters\/([a-z0-9_-]+)$/.exec(
    window.location.pathname,
  );
  const issueMatch = /^\/issues\/([a-z0-9_-]+)$/.exec(window.location.pathname);
  const demoIssue = new URLSearchParams(window.location.search).get("demoIssue");
  if (issueMatch) {
    return <IssueDetail issueId={issueMatch[1]} />;
  }
  if (demoIssue) {
    return <IssueDetail issueId={demoIssue} />;
  }
  return detailMatch ? (
    <NewsletterDetail newsletterId={detailMatch[1]} />
  ) : (
    <DashboardHome />
  );
}

function DashboardHome() {
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("all");

  useEffect(() => {
    const controller = new AbortController();
    apiJSON("/api/newsletters", { signal: controller.signal })
      .then(setSnapshot)
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, []);

  const newsletters = snapshot?.newsletters ?? [];
  const filteredNewsletters = newsletters.filter((newsletter) => {
    const matchesFilter =
      filter === "all" ||
      (filter === "active" && newsletter.active) ||
      (filter === "paused" && !newsletter.active);
    const searchText = `${newsletter.name} ${newsletter.topic}`.toLowerCase();
    return matchesFilter && searchText.includes(query.trim().toLowerCase());
  });
  const nextNewsletter = newsletters.find((newsletter) => newsletter.active);

  return (
    <div className="app">
      <Topbar onMenu={() => setSidebarOpen(true)} />
      <div className="shell">
        <Sidebar
          newsletters={newsletters}
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
        />
        <main className="content">
          <div className="content-inner">
            <section className="page-heading">
              <div>
                <p className="overline">Your learning practice</p>
                <h1>Learning streams</h1>
                <p className="intro">
                  Follow the subjects you care about. Learnloom keeps the thread,
                  filters the noise, and prepares the next lesson.
                </p>
              </div>
              {snapshot && <Summary summary={snapshot.summary} />}
            </section>

            {error ? <ErrorState message={error} /> : null}
            {!snapshot && !error ? <LoadingGrid /> : null}
            {snapshot && newsletters.length === 0 ? <EmptyState /> : null}
            {snapshot && newsletters.length > 0 ? (
              <>
                {nextNewsletter ? <NextUp newsletter={nextNewsletter} /> : null}
                <section className="stream-library" aria-labelledby="stream-library-heading">
                  <div className="library-toolbar">
                    <div>
                      <p className="overline">Your library</p>
                      <h2 id="stream-library-heading">Keep following your curiosity</h2>
                    </div>
                    <div className="library-controls">
                      <label className="stream-search">
                        <Search size={16} />
                        <span className="sr-only">Search learning streams</span>
                        <input
                          type="search"
                          placeholder="Search streams"
                          value={query}
                          onChange={(event) => setQuery(event.target.value)}
                        />
                      </label>
                      <div className="filter-group" aria-label="Filter learning streams">
                        {["all", "active", "paused"].map((value) => (
                          <button
                            className={filter === value ? "selected" : ""}
                            type="button"
                            key={value}
                            onClick={() => setFilter(value)}
                          >
                            {value.charAt(0).toUpperCase() + value.slice(1)}
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                  {filteredNewsletters.length ? (
                    <div className="dossier-grid" aria-label="Learning streams">
                      {filteredNewsletters.map((newsletter, index) => (
                        <DossierCard
                          key={newsletter.id}
                          newsletter={newsletter}
                          icon={iconCycle[index % iconCycle.length]}
                        />
                      ))}
                      {newsletters.length < 3 && filter === "all" && !query ? <CreateStreamCard /> : null}
                    </div>
                  ) : (
                    <div className="filter-empty">
                      <Search size={20} />
                      <strong>No streams match that view</strong>
                      <span>Try another search or clear the filter.</span>
                      <button type="button" onClick={() => { setQuery(""); setFilter("all"); }}>
                        Show all streams
                      </button>
                    </div>
                  )}
                </section>
              </>
            ) : null}
          </div>
          <Footer />
        </main>
      </div>
    </div>
  );
}

function Topbar({ onMenu }) {
  return (
    <header className="topbar">
      <div className="brand-group">
        <button className="icon-button mobile-menu" onClick={onMenu} aria-label="Open menu">
          <Menu size={20} />
        </button>
        <a className="brand" href="/">
          <span className="brand-mark"><Sparkles size={17} strokeWidth={2.2} /></span>
          <span>Learnloom</span>
        </a>
        <nav className="main-nav" aria-label="Primary navigation">
          <a className="active" href="/">Learning streams</a>
        </nav>
      </div>
      <div className="topbar-actions">
        <a className="primary-button" href="/newsletters/new" aria-label="Create a new learning stream">
          <Plus size={17} strokeWidth={2.5} />
          <span>New learning stream</span>
        </a>
      </div>
    </header>
  );
}

function Sidebar({ newsletters, open, onClose, currentNewsletterId }) {
  return (
    <>
      <button
        className={`scrim ${open ? "visible" : ""}`}
        onClick={onClose}
        aria-label="Close menu"
      />
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <div className="sidebar-mobile-head">
          <span>Workspace</span>
          <button className="icon-button" onClick={onClose} aria-label="Close menu">
            <X size={19} />
          </button>
        </div>
        <a className={`sidebar-home ${!currentNewsletterId ? "current" : ""}`} href="/">
          <LibraryBig size={17} />
          <span>All learning streams</span>
        </a>
        <p className="sidebar-label">Active now</p>
        <nav className="topic-list" aria-label="Active topics">
          {newsletters.filter((item) => item.active).slice(0, 6).map((item, index) => {
            const Icon = iconCycle[index % iconCycle.length];
            return (
              <a
                className={item.id === currentNewsletterId ? "current" : ""}
                key={item.id}
                href={`/newsletters/${encodeURIComponent(item.id)}`}
              >
                <Icon size={17} />
                <span>{item.name}</span>
              </a>
            );
          })}
          {newsletters.filter((item) => item.active).length === 0 ? (
            <span className="sidebar-empty">No active topics yet</span>
          ) : null}
        </nav>
        <div className="sidebar-bottom">
          <a className="archive-button" href="/newsletters/new">
            <Plus size={16} />
            New learning stream
          </a>
          <p className="sidebar-promise">A focused lesson from sources you trust, ready on your schedule.</p>
        </div>
      </aside>
    </>
  );
}

function Summary({ summary }) {
  return (
    <dl className="summary" aria-label="Workspace summary">
      <div><dt>Streams</dt><dd>{summary.newsletters}</dd></div>
      <div><dt>Active</dt><dd>{summary.active}</dd></div>
      <div><dt>Lessons</dt><dd>{summary.generated}</dd></div>
    </dl>
  );
}

function NextUp({ newsletter }) {
  const Icon = iconCycle[0];
  return (
    <section className="next-up" aria-labelledby="next-up-title">
      <div className="next-up-icon"><Icon size={23} /></div>
      <div className="next-up-copy">
        <p className="overline">Next in your rhythm</p>
        <h2 id="next-up-title">{newsletter.name}</h2>
        <p>{newsletter.topic}</p>
      </div>
      <div className="next-up-meta">
        <span><Clock3 size={15} />{formatSchedule(newsletter.scheduleTime, newsletter.timeZone)}</span>
        <span>{newsletter.generatedCount} lesson{newsletter.generatedCount === 1 ? "" : "s"} in your archive</span>
      </div>
      <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
        Open stream <ChevronRight size={16} />
      </a>
    </section>
  );
}

function DossierCard({ newsletter, icon: Icon }) {
  return (
    <article className="dossier-card">
      <div className="card-top">
        <span className="category"><Icon size={13} />Learning stream</span>
        <span className={`state ${newsletter.active ? "active" : ""}`}>
          <i />{newsletter.active ? "Active" : "Paused"}
        </span>
      </div>
      <div className="card-copy">
        <div className="topic-icon"><Icon size={22} /></div>
        <h2>{newsletter.name}</h2>
        <p>{newsletter.topic}</p>
      </div>
      <dl className="card-details">
        <div>
          <dt>Next lesson</dt>
          <dd>{formatSchedule(newsletter.scheduleTime, newsletter.timeZone)}</dd>
        </div>
        <div>
          <dt>Learning history</dt>
          <dd>{newsletter.generatedCount} lesson{newsletter.generatedCount === 1 ? "" : "s"}</dd>
        </div>
      </dl>
      <div className="card-footer">
        <span className="delivery-state">
          {newsletter.emailEnabled ? "Email delivery on" : "Read in Learnloom"}
        </span>
        <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
          Open stream <ChevronRight size={16} />
        </a>
      </div>
    </article>
  );
}

function CreateStreamCard() {
  return (
    <a className="create-stream-card" href="/newsletters/new">
      <span><Plus size={20} /></span>
      <strong>Follow a new subject</strong>
      <small>Set the question, sources, and learning rhythm.</small>
    </a>
  );
}

function EmptyState() {
  return (
    <section className="empty-state">
      <div className="empty-visual" aria-hidden="true">
        <span className="empty-orbit empty-orbit-outer" />
        <span className="empty-orbit empty-orbit-inner" />
        <span className="empty-spark empty-spark-one" />
        <span className="empty-spark empty-spark-two" />
        <span className="empty-icon"><LibraryBig size={30} /></span>
      </div>
      <div className="empty-copy">
        <p className="overline">Your learning starts here</p>
        <h2>Turn a question into a lasting body of knowledge.</h2>
        <p>
          Choose a topic you care about. Learnloom will gather trusted sources
          and shape them into thoughtful, connected lessons.
        </p>
        <div className="empty-benefits" aria-label="What your first dossier includes">
          <span>Trusted sources</span>
          <i />
          <span>A learning schedule</span>
          <i />
          <span>Your own archive</span>
        </div>
        <a className="primary-button" href="/newsletters/new">
          <Plus size={17} />
          Create your first learning stream
        </a>
        <small>It takes about two minutes to set up.</small>
      </div>
    </section>
  );
}

function ErrorState({ message }) {
  return (
    <section className="error-state">
      <strong>We couldn’t open your workspace.</strong>
      <span>{message}</span>
      <button type="button" onClick={() => window.location.reload()}>Try again</button>
    </section>
  );
}

function LoadingGrid() {
  return (
    <section className="dossier-grid" aria-label="Loading dossiers">
      {[0, 1, 2].map((item) => <div className="card-skeleton" key={item} />)}
    </section>
  );
}

function Footer() {
  return (
    <footer>
      <span>Learnloom</span>
      <span>Understanding, built one lesson at a time.</span>
    </footer>
  );
}

function formatSchedule(time, timeZone) {
  if (!time) return "Not scheduled";
  return `Daily, ${time} · ${shortTimeZone(timeZone)}`;
}

function shortTimeZone(timeZone) {
  if (!timeZone) return "Local";
  return timeZone.split("/").at(-1).replaceAll("_", " ");
}

export default App;
export { ErrorState, Footer, Sidebar, Topbar };
