/**
 * Contextual return navigation for the Dossier reader.
 *
 * The document-level SPA interceptor in App.tsx records a small,
 * validated history-state record whenever a reader route is opened from
 * elsewhere in the app. The reader's top-left control then offers a
 * single-step return to that origin, skipping adjacent lessons added by
 * previous/next navigation, and letting native history traversal restore
 * the origin's scroll position.
 *
 * Everything in this module is pure and window-safe so it can be unit
 * tested directly and used from static-markup renders.
 */

export const READER_RETURN_VERSION = "learnloom-reader-v1";

/** Upper bound for a reader chain: history depth never grows past this. */
export const MAX_READER_DEPTH = 100;

/** Upper bound for an origin label: short, render-safe, and easy to scan. */
export const MAX_READER_LABEL = 80;

/** Generic label when a stream name cannot be derived safely. */
export const DEFAULT_STREAM_LABEL = "Learning stream";

const ISSUE_PATH_RE = /^\/issues\/[a-z0-9_-]+$/;

/**
 * Same-app routes a contextual return may target. Everything else —
 * reader pages, publishing, settings, legal, or welcome journeys — is
 * rejected so the parent-stream anchor remains the fallback.
 */
const RETURN_PATH_RE =
  /^\/(?:streams|library|review|newsletters\/[a-z0-9_-]+)?$/;

/** The document-title shape NewsletterDetail sets while a stream is open. */
const STREAM_TITLE_SUFFIX = " · Learnloom";

export interface ReaderReturnState {
  v: typeof READER_RETURN_VERSION;
  href: string;
  label: string;
  depth: number;
}

/** Inputs the SPA interceptor supplies when computing a pushed entry's state. */
export interface ReaderNavigationInput {
  /** Destination href (same-app pathname + search + hash). */
  destination: string;
  currentPathname: string;
  currentSearch: string;
  currentHash: string;
  /** Same-app origin used for URL validation. */
  origin: string;
  /** The untrusted current history.state value. */
  historyState: unknown;
  documentTitle: string;
}

/** Shape of the click event the contextual return control consumes. */
export interface ReaderReturnClickEvent {
  defaultPrevented: boolean;
  button: number;
  metaKey: boolean;
  ctrlKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
  preventDefault: () => void;
}

/** True when the URL opens the Dossier reader (direct or demo target). */
export function isReaderTarget(url: URL): boolean {
  return (
    ISSUE_PATH_RE.test(url.pathname) ||
    (url.pathname === "/" && url.searchParams.has("demoIssue"))
  );
}

/** True when the current same-app location is itself the reader. */
export function isReaderLocation(pathname: string, search: string): boolean {
  return (
    ISSUE_PATH_RE.test(pathname) ||
    (pathname === "/" && new URLSearchParams(search).has("demoIssue"))
  );
}

/**
 * Validates an untrusted history.state value. Returns null for anything
 * malformed, foreign, reader-shaped, or outside the return allowlist;
 * the returned record is safe to render and to navigate with.
 */
export function parseReaderReturnState(
  state: unknown,
  origin = currentOrigin(),
): ReaderReturnState | null {
  if (typeof state !== "object" || state === null) return null;
  const record = state as Record<string, unknown>;
  if (record.v !== READER_RETURN_VERSION) return null;
  if (typeof record.label !== "string") return null;
  const label = record.label.trim();
  if (!label || label.length > MAX_READER_LABEL) return null;
  const depth = record.depth;
  if (
    typeof depth !== "number" ||
    !Number.isInteger(depth) ||
    depth < 1 ||
    depth > MAX_READER_DEPTH
  ) {
    return null;
  }
  const href = returnHref(record.href, origin);
  if (!href) return null;
  return { v: READER_RETURN_VERSION, href, label, depth };
}

/**
 * The history state for a pushed same-app entry. Reader destinations
 * remember their origin: the first entry records the current location at
 * depth 1, while reader-to-reader navigation preserves the origin and
 * extends the depth so the return action skips the whole chain in one
 * step. Every other navigation carries no reader state.
 */
export function readerNavigationState(
  input: ReaderNavigationInput,
): ReaderReturnState | null {
  const {
    destination,
    currentPathname,
    currentSearch,
    currentHash,
    origin,
    historyState,
    documentTitle,
  } = input;
  let next: URL;
  try {
    next = new URL(destination, origin);
  } catch {
    return null;
  }
  if (!isReaderTarget(next)) return null;
  if (next.origin !== origin) return null;
  if (isReaderLocation(currentPathname, currentSearch)) {
    const previous = parseReaderReturnState(historyState, origin);
    return previous && previous.depth < MAX_READER_DEPTH
      ? nextReaderReturnState(previous)
      : null;
  }
  return parseReaderReturnState(
    firstReaderReturnState(
      `${currentPathname}${currentSearch}${currentHash}`,
      readerReturnLabel(currentPathname, documentTitle),
    ),
    origin,
  );
}

/** State for the first entry into a reader: capture the origin at depth 1. */
export function firstReaderReturnState(
  href: string,
  label: string,
): ReaderReturnState {
  return { v: READER_RETURN_VERSION, href, label, depth: 1 };
}

/** State for reader-to-reader navigation: preserve origin, extend depth. */
export function nextReaderReturnState(
  previous: ReaderReturnState,
): ReaderReturnState {
  return {
    ...previous,
    depth: previous.depth + 1,
  };
}

/** Concise route label for a captured origin route. */
export function readerReturnLabel(
  pathname: string,
  documentTitle = "",
): string {
  if (pathname === "/") return "Today";
  if (pathname === "/streams") return "Streams";
  if (pathname === "/library") return "Library";
  if (pathname === "/review") return "Review";
  if (/^\/newsletters\/[a-z0-9_-]+$/.test(pathname)) {
    return streamLabelFromTitle(documentTitle);
  }
  return DEFAULT_STREAM_LABEL;
}

/**
 * Derives a stream name from the `{name} · Learnloom` document title
 * NewsletterDetail sets. Anything outside that exact, bounded shape
 * falls back to the generic stream label.
 */
export function streamLabelFromTitle(documentTitle: string): string {
  if (typeof documentTitle !== "string") return DEFAULT_STREAM_LABEL;
  if (!documentTitle.endsWith(STREAM_TITLE_SUFFIX)) return DEFAULT_STREAM_LABEL;
  const name = documentTitle.slice(0, -STREAM_TITLE_SUFFIX.length).trim();
  if (!name || name.length > MAX_READER_LABEL) return DEFAULT_STREAM_LABEL;
  return name;
}

/**
 * Intercepts an unmodified primary click on the contextual return
 * control and travels back to the origin in one history step, skipping
 * adjacent Dossier entries so native history restores the origin scroll.
 * Modifier clicks, non-primary buttons, already-handled events, and any
 * invalid state stay native. Returns true when the click was consumed.
 */
export function handleReaderReturnClick(
  event: ReaderReturnClickEvent,
  historyState: unknown,
): boolean {
  if (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return false;
  }
  const state = parseReaderReturnState(historyState);
  if (!state) return false;
  event.preventDefault();
  if (typeof window !== "undefined") window.history.go(-state.depth);
  return true;
}

/** Same-app origin used as the URL validation base when none is supplied. */
function currentOrigin(): string {
  return typeof window === "undefined" ? "" : window.location.origin;
}

function returnHref(href: unknown, origin: string): string | null {
  if (typeof href !== "string" || href.length === 0 || href.length > 2048) {
    return null;
  }
  let url: URL;
  try {
    url = new URL(href, origin);
  } catch {
    return null;
  }
  if (url.origin !== origin) return null;
  if (url.protocol !== "http:" && url.protocol !== "https:") return null;
  if (!RETURN_PATH_RE.test(url.pathname)) return null;
  // The demo reader lives on the root path; it is never a return origin.
  if (url.pathname === "/" && url.searchParams.has("demoIssue")) return null;
  return `${url.pathname}${url.search}${url.hash}`;
}
