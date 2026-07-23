import { Menu, Plus, Sparkles } from "lucide-react";
import IssueDetail from "./IssueDetail.jsx";
import LibraryPage from "./LibraryPage.jsx";
import NewsletterCreate from "./NewsletterCreate.jsx";
import NewsletterDetail from "./NewsletterDetail.jsx";
import PublishingPage from "./PublishingPage.jsx";
import ReviewPage from "./ReviewPage.jsx";
import StreamsPage from "./StreamsPage.jsx";
import TodayPage from "./TodayPage.jsx";

export default function App({ capabilities = {}, site = null, onSiteUpdate }) {
  const path = window.location.pathname;

  if (path === "/newsletters/new") {
    return <NewsletterCreate sourceDiscovery={Boolean(capabilities.sourceDiscovery)} />;
  }

  const detailMatch = /^\/newsletters\/([a-z0-9_-]+)$/.exec(path);
  if (detailMatch) {
    return <NewsletterDetail newsletterId={detailMatch[1]} />;
  }

  const issueMatch = /^\/issues\/([a-z0-9_-]+)$/.exec(path);
  const demoIssue = new URLSearchParams(window.location.search).get("demoIssue");
  if (issueMatch || demoIssue) {
    return <IssueDetail issueId={issueMatch?.[1] ?? demoIssue} />;
  }

  if (path === "/streams") return <StreamsPage />;
  if (path === "/library") return <LibraryPage />;
  if (path === "/review") return <ReviewPage />;
  if (path === "/publishing") {
    return <PublishingPage site={site} onSiteUpdate={onSiteUpdate} />;
  }
  return <TodayPage />;
}

function Topbar({ onMenu }) {
  return (
    <header className="create-topbar">
      <button className="create-menu-button" type="button" onClick={onMenu} aria-label="Open navigation">
        <Menu size={18} />
      </button>
      <a className="create-brand" href="/">
        <span><Sparkles size={15} /></span>
        <strong>Learnloom</strong>
      </a>
      <nav>
        <a href="/">Today</a>
        <a href="/streams">Learning streams</a>
      </nav>
      <a className="primary-button" href="/newsletters/new">
        <Plus size={16} /> New learning stream
      </a>
    </header>
  );
}

function ErrorState({ message }) {
  return (
    <section className="error-state" role="alert">
      <strong>We couldn’t complete that request.</strong>
      <span>{message}</span>
      <button type="button" onClick={() => window.location.reload()}>Try again</button>
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

export { ErrorState, Footer, Topbar };
