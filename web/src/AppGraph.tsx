import { Menu, Plus } from "lucide-react";
import { lazy, Suspense, useEffect, useState, type MouseEvent, type ReactNode } from "react";
import BrandMark from "./BrandMark";
import CalmLoader from "./CalmLoader";
import LibraryPage from "./LibraryPage";
import PublishingPage from "./PublishingPage";
import ReviewPage from "./ReviewPage";
import StreamsPage from "./StreamsPage";
import TodayPage from "./TodayPage";
import type { NotificationPreferences, Site } from "./types";
import { readerNavigationState } from "./readerReturn";

const IssueDetail = lazy(() => import("./IssueDetail"));
const FirstLessonWelcome = lazy(() => import("./FirstLessonWelcome"));
const NewsletterCreate = lazy(() => import("./NewsletterCreate"));
const NewsletterDetail = lazy(() => import("./NewsletterDetail"));
const SettingsPage = lazy(() => import("./SettingsPage"));

interface AppGraphProps {
  capabilities?: { sourceDiscovery?: boolean };
  site?: Site | null;
  onSiteUpdate?: (site: Site) => void;
  initialNotifications?: NotificationPreferences;
  onNotificationsUpdate?: (notifications: NotificationPreferences) => void;
}

export default function AppGraph({
  capabilities = {},
  site = null,
  onSiteUpdate,
  initialNotifications,
  onNotificationsUpdate,
}: AppGraphProps) {
  const [location, setLocation] = useState(
    () => `${window.location.pathname}${window.location.search}${window.location.hash}`,
  );
  const path = new URL(location, window.location.origin).pathname;

  useEffect(() => {
    const updateLocation = () => {
      setLocation(`${window.location.pathname}${window.location.search}${window.location.hash}`);
    };
    const navigate = (event: globalThis.MouseEvent) => {
      if (
        event.defaultPrevented ||
        event.button !== 0 ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) return;

      const anchor = (event.target as Element | null)?.closest("a");
      if (!anchor || anchor.target || anchor.hasAttribute("download")) return;

      const next = new URL(anchor.href, window.location.href);
      if (
        next.origin !== window.location.origin ||
        !["http:", "https:"].includes(next.protocol) ||
        (next.pathname === window.location.pathname &&
          next.search === window.location.search &&
          next.hash)
      ) return;

      const href = `${next.pathname}${next.search}${next.hash}`;
      event.preventDefault();
      window.history.pushState(
        readerNavigationState({
          destination: href,
          currentPathname: window.location.pathname,
          currentSearch: window.location.search,
          currentHash: window.location.hash,
          origin: window.location.origin,
          historyState: window.history.state,
          documentTitle: document.title,
        }),
        "",
        href,
      );
      updateLocation();
      window.scrollTo({ top: 0 });
    };

    window.addEventListener("popstate", updateLocation);
    document.addEventListener("click", navigate);
    return () => {
      window.removeEventListener("popstate", updateLocation);
      document.removeEventListener("click", navigate);
    };
  }, []);

  useEffect(() => {
    // Warm the lazy Settings chunk after the app mounts (and on first
    // pointer/focus intent toward the Settings link) so the first visit to
    // /settings opens without a chunk-load pause. The timer is idle work and
    // never delays initial app rendering.
    let warmed = false;
    const warmSettingsChunk = () => {
      if (warmed) return;
      warmed = true;
      void import("./SettingsPage");
    };
    const warmTimer = window.setTimeout(warmSettingsChunk, 1_500);
    const warmOnIntent = (event: Event) => {
      if ((event.target as Element | null)?.closest?.('a[href="/settings"]')) {
        warmSettingsChunk();
      }
    };
    document.addEventListener("pointerover", warmOnIntent);
    document.addEventListener("focusin", warmOnIntent);
    return () => {
      window.clearTimeout(warmTimer);
      document.removeEventListener("pointerover", warmOnIntent);
      document.removeEventListener("focusin", warmOnIntent);
    };
  }, []);

  if (path === "/newsletters/new") {
    return routePage(
      <NewsletterCreate sourceDiscovery={Boolean(capabilities.sourceDiscovery)} />,
    );
  }

  const welcomeMatch = /^\/welcome\/([a-z0-9_-]+)$/.exec(path);
  if (welcomeMatch) {
    return routePage(<FirstLessonWelcome newsletterId={welcomeMatch[1]} />);
  }

  const detailMatch = /^\/newsletters\/([a-z0-9_-]+)$/.exec(path);
  if (detailMatch) {
    return routePage(<NewsletterDetail newsletterId={detailMatch[1]} />);
  }

  const issueMatch = /^\/issues\/([a-z0-9_-]+)$/.exec(path);
  const demoIssue = new URLSearchParams(window.location.search).get("demoIssue");
  if (issueMatch || demoIssue) {
    return routePage(<IssueDetail issueId={issueMatch?.[1] ?? demoIssue} />);
  }

  if (path === "/streams") return routePage(<StreamsPage />);
  if (path === "/library") return routePage(<LibraryPage />);
  if (path === "/review") return routePage(<ReviewPage />);
  if (path === "/publishing") {
    return routePage(<PublishingPage site={site} onSiteUpdate={onSiteUpdate} />);
  }
  if (path === "/settings") {
    return routePage(
      <SettingsPage
        initialNotifications={initialNotifications}
        onNotificationsUpdate={onNotificationsUpdate}
      />,
    );
  }
  return routePage(<TodayPage />);
}

function routePage(page: ReactNode) {
  return (
    <Suspense fallback={<CalmLoader label="Preparing this space…" />}>
      {page}
    </Suspense>
  );
}

function Topbar({ onMenu }: { onMenu: (event: MouseEvent<HTMLButtonElement>) => void }) {
  return (
    <header className="create-topbar">
      <button className="create-menu-button" type="button" onClick={onMenu} aria-label="Open navigation">
        <Menu size={18} />
      </button>
      <a className="create-brand" href="/">
        <span><BrandMark /></span>
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

function ErrorState({ message }: { message: string }) {
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