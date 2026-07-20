import {
  Archive,
  Atom,
  BookOpen,
  BrainCircuit,
  Check,
  ChevronRight,
  CircleHelp,
  Clock3,
  FileText,
  History,
  LibraryBig,
  Menu,
  Plus,
  Search,
  Sparkles,
  X,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
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
                <p className="overline">Your intelligence workspace</p>
                <h1>Knowledge Dossiers</h1>
                <p className="intro">
                  Automated research journeys designed for intellectual depth
                  and archival precision.
                </p>
              </div>
              {snapshot && <Summary summary={snapshot.summary} />}
            </section>

            {error ? <ErrorState message={error} /> : null}
            {!snapshot && !error ? <LoadingGrid /> : null}
            {snapshot && newsletters.length === 0 ? <EmptyState /> : null}
            {snapshot && newsletters.length > 0 ? (
              <section className="dossier-grid" aria-label="Knowledge dossiers">
                {newsletters.map((newsletter, index) => (
                  <DossierCard
                    key={newsletter.id}
                    newsletter={newsletter}
                    icon={iconCycle[index % iconCycle.length]}
                  />
                ))}
                <CreateCard />
              </section>
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
          <a className="active" href="/">Newsletters</a>
          <a href="#history">Learning History</a>
          <a href="#archive">Archive</a>
          <a href="#settings">Settings</a>
        </nav>
      </div>
      <div className="topbar-actions">
        <button className="search-button" type="button" aria-label="Search">
          <Search size={17} />
          <span>Search</span>
          <kbd>⌘ K</kbd>
        </button>
        <a className="primary-button" href="/newsletters/new">
          <Plus size={17} strokeWidth={2.5} />
          <span>Create Newsletter</span>
        </a>
        <div className="avatar" aria-label="Account menu">VP</div>
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
        <p className="sidebar-label">Active topics</p>
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
          <a className="archive-button" href="#archive">
            <Archive size={16} />
            Manage Archive
          </a>
          <nav className="utility-links" aria-label="Resources">
            <a href="#status"><Clock3 size={16} />System Status</a>
            <a href="#docs"><FileText size={16} />Documentation</a>
            <a href="#help"><CircleHelp size={16} />Help & feedback</a>
          </nav>
        </div>
      </aside>
    </>
  );
}

function Summary({ summary }) {
  return (
    <dl className="summary" aria-label="Workspace summary">
      <div><dt>Dossiers</dt><dd>{summary.newsletters}</dd></div>
      <div><dt>Active</dt><dd>{summary.active}</dd></div>
      <div><dt>Issues</dt><dd>{summary.generated}</dd></div>
    </dl>
  );
}

function DossierCard({ newsletter, icon: Icon }) {
  const progress = useMemo(() => {
    if (newsletter.issueCount === 0) return 0;
    return Math.round((newsletter.generatedCount / newsletter.issueCount) * 100);
  }, [newsletter.generatedCount, newsletter.issueCount]);
  const recipient = newsletter.emailEnabled
    ? newsletter.emailRecipients[0] ?? "Recipient pending"
    : "Email off";

  return (
    <article className="dossier-card">
      <div className="card-top">
        <span className="category"><Icon size={13} />Dossier</span>
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
          <dt>Schedule</dt>
          <dd>{formatSchedule(newsletter.scheduleTime, newsletter.timeZone)}</dd>
        </div>
        <div>
          <dt>Recipient</dt>
          <dd title={recipient}>{recipient}</dd>
        </div>
      </dl>
      <div className="pipeline">
        <div className="pipeline-label">
          <span>Pipeline</span>
          <strong>{progress}%</strong>
        </div>
        <div className="pipeline-track" aria-label={`${progress}% generated`}>
          <span style={{ width: `${progress}%` }} />
        </div>
      </div>
      <div className="card-footer">
        <span className={progress === 100 ? "verified" : "pending"}>
          {progress === 100 ? <Check size={14} /> : <History size={14} />}
          {progress === 100 ? "Quality verified" : `${newsletter.generatedCount} generated`}
        </span>
        <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
          View Blueprint <ChevronRight size={16} />
        </a>
      </div>
    </article>
  );
}

function CreateCard() {
  return (
    <a className="create-card" href="/newsletters/new">
      <span className="create-icon"><Plus size={24} /></span>
      <strong>Create a new dossier</strong>
      <span>Choose a topic, trusted sources, and a delivery schedule.</span>
    </a>
  );
}

function EmptyState() {
  return (
    <section className="empty-state">
      <span className="empty-icon"><LibraryBig size={28} /></span>
      <p className="overline">A clean slate</p>
      <h2>Build your first knowledge dossier</h2>
      <p>Start with one question you want to understand more deeply.</p>
      <a className="primary-button" href="/newsletters/new"><Plus size={17} />Create Newsletter</a>
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
      <nav aria-label="Footer navigation">
        <a href="#status">System Status</a>
        <a href="#docs">Documentation</a>
        <a href="#privacy">Privacy</a>
      </nav>
      <span>Intelligence, made durable.</span>
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
