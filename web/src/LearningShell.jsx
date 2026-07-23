import {
  BookOpen,
  BrainCircuit,
  CalendarDays,
  CircleHelp,
  Globe2,
  LibraryBig,
  Menu,
  Plus,
  Settings2,
  Sparkles,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";

const navigation = [
  { href: "/", label: "Today", icon: CalendarDays, key: "today" },
  { href: "/streams", label: "Streams", icon: Sparkles, key: "streams" },
  { href: "/library", label: "Library", icon: LibraryBig, key: "library" },
  { href: "/review", label: "Review", icon: BrainCircuit, key: "review" },
  { href: "/publishing", label: "Publishing", icon: Globe2, key: "publishing" },
];

export default function LearningShell({
  active,
  children,
  immersive = false,
}) {
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    setMenuOpen(false);
  }, [active]);

  if (immersive) return children;

  return (
    <div className="atelier-app">
      <button
        className={`atelier-scrim ${menuOpen ? "visible" : ""}`}
        type="button"
        aria-label="Close navigation"
        onClick={() => setMenuOpen(false)}
      />
      <aside className={`atelier-sidebar ${menuOpen ? "open" : ""}`}>
        <div className="atelier-brand-row">
          <a className="atelier-brand" href="/">
            <span><Sparkles size={15} /></span>
            <strong>Learnloom</strong>
            <small>Digital atelier</small>
          </a>
          <button
            className="atelier-mobile-close"
            type="button"
            onClick={() => setMenuOpen(false)}
            aria-label="Close navigation"
          >
            <X size={18} />
          </button>
        </div>

        <a className="atelier-new-entry" href="/newsletters/new">
          <Plus size={15} />
          New learning stream
        </a>

        <nav className="atelier-nav" aria-label="Primary navigation">
          {navigation.map(({ href, label, icon: Icon, key }) => (
            <a className={active === key ? "current" : ""} href={href} key={key}>
              <Icon size={16} />
              <span>{label}</span>
            </a>
          ))}
        </nav>

        <div className="atelier-sidebar-bottom">
          <a href="/publishing"><Settings2 size={15} />Site settings</a>
          <a href="mailto:support@learnloom.blog"><CircleHelp size={15} />Support</a>
        </div>
      </aside>

      <div className="atelier-stage">
        <header className="atelier-topbar">
          <button
            className="atelier-menu"
            type="button"
            onClick={() => setMenuOpen(true)}
            aria-label="Open navigation"
          >
            <Menu size={19} />
          </button>
          <a className="atelier-mobile-brand" href="/">Learnloom</a>
          <div className="atelier-top-actions">
            <a href="/library" aria-label="Open library"><BookOpen size={16} /></a>
          </div>
        </header>
        <main className="atelier-main">{children}</main>
      </div>
    </div>
  );
}

export function AtelierLoading({ label = "Preparing your learning home…" }) {
  return (
    <div className="atelier-state-card" aria-live="polite">
      <span className="atelier-spinner" />
      <strong>{label}</strong>
    </div>
  );
}

export function AtelierError({ message, onRetry }) {
  return (
    <div className="atelier-state-card error" role="alert">
      <strong>We couldn’t load this part of your learning home.</strong>
      <p>{message}</p>
      {onRetry ? <button type="button" onClick={onRetry}>Try again</button> : null}
    </div>
  );
}

export function formatShortDate(value) {
  if (!value) return "Recently";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
  }).format(new Date(value));
}
