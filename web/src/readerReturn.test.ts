import { afterEach, describe, expect, it, vi } from "vitest";
import {
  DEFAULT_STREAM_LABEL,
  MAX_READER_DEPTH,
  MAX_READER_LABEL,
  READER_RETURN_VERSION,
  handleReaderReturnClick,
  parseReaderReturnState,
  readerNavigationState,
  readerReturnLabel,
  streamLabelFromTitle,
} from "./readerReturn";

const ORIGIN = "https://app.learnloom.blog";

function baseState(overrides: Record<string, unknown> = {}) {
  return {
    v: READER_RETURN_VERSION,
    href: "/streams",
    label: "Streams",
    depth: 1,
    ...overrides,
  };
}

function navigation(overrides: Record<string, unknown> = {}) {
  return {
    destination: "/issues/lesson-1",
    currentPathname: "/streams",
    currentSearch: "",
    currentHash: "",
    origin: ORIGIN,
    historyState: null,
    documentTitle: "Learnloom",
    ...overrides,
  };
}

describe("parseReaderReturnState", () => {
  it("accepts every allowlisted origin route, with search and hash", () => {
    const cases = [
      { href: "/", label: "Today" },
      { href: "/streams", label: "Streams" },
      { href: "/streams?tab=archived#done", label: "Streams" },
      { href: "/library", label: "Library" },
      { href: "/review", label: "Review" },
      { href: "/newsletters/quantum-42", label: "Quantum Field Theory" },
      { href: `${ORIGIN}/streams`, label: "Streams" },
    ];
    for (const entry of cases) {
      const parsed = parseReaderReturnState(baseState(entry), ORIGIN);
      expect(parsed).not.toBeNull();
      expect(parsed?.href).toBe(
        new URL(entry.href, ORIGIN).pathname +
          new URL(entry.href, ORIGIN).search +
          new URL(entry.href, ORIGIN).hash,
      );
      expect(parsed?.label).toBe(entry.label);
    }
  });

  it("rejects external, foreign-protocol, and reader origins", () => {
    const rejectedHrefs = [
      "https://evil.example/streams",
      "https://app.learnloom.blog.evil.example/streams",
      "//evil.example/streams",
      "javascript:alert(1)",
      "data:text/html,phish",
      "https://app.learnloom.blog:8443/streams",
      "/issues/lesson-1",
      "/issues/lesson-1?tab=open",
      "/?demoIssue=lesson-1",
    ];
    for (const href of rejectedHrefs) {
      expect(
        parseReaderReturnState(baseState({ href }), ORIGIN),
        href,
      ).toBeNull();
    }
  });

  it("rejects same-app pathnames outside the return allowlist", () => {
    const rejectedPathnames = [
      "/publishing",
      "/settings",
      "/welcome/abc",
      "/newsletters/",
      "/newsletters/a/b",
      "/streams/extra",
      "/terms",
    ];
    for (const pathname of rejectedPathnames) {
      expect(
        parseReaderReturnState(baseState({ href: pathname }), ORIGIN),
        pathname,
      ).toBeNull();
    }
  });

  it("rejects malformed, unbounded, or foreign-shaped records", () => {
    const malformed = [
      null,
      undefined,
      "streams",
      42,
      { v: "learnloom-reader-v0", href: "/streams", label: "Streams", depth: 1 },
      baseState({ v: undefined }),
      baseState({ href: undefined }),
      baseState({ href: "" }),
      baseState({ href: "x".repeat(2049) }),
      baseState({ label: undefined }),
      baseState({ label: "   " }),
      baseState({ label: "x".repeat(MAX_READER_LABEL + 1) }),
      baseState({ depth: undefined }),
      baseState({ depth: 0 }),
      baseState({ depth: -1 }),
      baseState({ depth: 1.5 }),
      baseState({ depth: "1" }),
      baseState({ depth: MAX_READER_DEPTH + 1 }),
    ];
    for (const state of malformed) {
      expect(parseReaderReturnState(state, ORIGIN), JSON.stringify(state)).toBeNull();
    }
  });

  it("accepts the bounded depth and returns a normalized same-app href", () => {
    const parsed = parseReaderReturnState(
      baseState({ href: `${ORIGIN}/library#lesson-2`, depth: MAX_READER_DEPTH }),
      ORIGIN,
    );
    expect(parsed).toEqual({
      v: READER_RETURN_VERSION,
      href: "/library#lesson-2",
      label: "Streams",
      depth: MAX_READER_DEPTH,
    });
  });
});

describe("readerNavigationState", () => {
  it("creates a depth-1 state when a non-reader route opens a reader", () => {
    const state = readerNavigationState(
      navigation({ currentPathname: "/streams" }),
    );
    expect(state).toEqual({
      v: READER_RETURN_VERSION,
      href: "/streams",
      label: "Streams",
      depth: 1,
    });
  });

  it("captures search and hash of the origin location", () => {
    const state = readerNavigationState(
      navigation({
        currentPathname: "/library",
        currentSearch: "?scope=all",
        currentHash: "#lesson-2",
      }),
    );
    expect(state?.href).toBe("/library?scope=all#lesson-2");
    expect(state?.label).toBe("Library");
    expect(state?.depth).toBe(1);
  });

  it("treats the demo reader target as a reader entry from Today", () => {
    const state = readerNavigationState(
      navigation({
        destination: "/?demoIssue=ai-evaluation-issue-1",
        currentPathname: "/",
      }),
    );
    expect(state).toEqual({
      v: READER_RETURN_VERSION,
      href: "/",
      label: "Today",
      depth: 1,
    });
  });

  it("derives the stream name from the document title on stream origins", () => {
    const state = readerNavigationState(
      navigation({
        currentPathname: "/newsletters/quantum-42",
        documentTitle: "Quantum Field Theory · Learnloom",
      }),
    );
    expect(state?.href).toBe("/newsletters/quantum-42");
    expect(state?.label).toBe("Quantum Field Theory");
    expect(state?.depth).toBe(1);
  });

  it("preserves origin and increments depth for reader-to-reader navigation", () => {
    const state = readerNavigationState(
      navigation({
        destination: "/issues/lesson-2",
        currentPathname: "/issues/lesson-1",
        currentSearch: "",
        currentHash: "",
        historyState: baseState({ href: "/streams", label: "Streams", depth: 1 }),
      }),
    );
    expect(state).toEqual({
      v: READER_RETURN_VERSION,
      href: "/streams",
      label: "Streams",
      depth: 2,
    });
  });

  it("preserves origin across the demo reader chain", () => {
    const state = readerNavigationState(
      navigation({
        destination: "/?demoIssue=lesson-2",
        currentPathname: "/",
        currentSearch: "?demoIssue=lesson-1",
        currentHash: "",
        historyState: baseState({ href: "/review", label: "Review", depth: 2 }),
      }),
    );
    expect(state?.href).toBe("/review");
    expect(state?.label).toBe("Review");
    expect(state?.depth).toBe(3);
  });

  it("drops contextual state when a reader chain exceeds the bounded depth", () => {
    const state = readerNavigationState(
      navigation({
        destination: "/issues/lesson-2",
        currentPathname: "/issues/lesson-1",
        historyState: baseState({ depth: MAX_READER_DEPTH }),
      }),
    );
    expect(state).toBeNull();
  });

  it("carries no reader state when a reader's own state is invalid", () => {
    const state = readerNavigationState(
      navigation({
        destination: "/issues/lesson-2",
        currentPathname: "/issues/lesson-1",
        historyState: { v: "learnloom-reader-v0", href: "/streams", label: "Streams", depth: 1 },
      }),
    );
    expect(state).toBeNull();
  });

  it("carries no reader state for non-reader destinations", () => {
    const destinations = [
      "/",
      "/streams",
      "/library",
      "/review",
      "/newsletters/quantum-42",
      "/publishing",
    ];
    for (const destination of destinations) {
      const state = readerNavigationState(
        navigation({
          destination,
          currentPathname: "/issues/lesson-1",
          historyState: baseState(),
        }),
      );
      expect(state, destination).toBeNull();
    }
  });

  it("carries no reader state for malformed destinations", () => {
    expect(readerNavigationState(navigation({ destination: "::not-a-url" }))).toBeNull();
    expect(readerNavigationState(navigation({ destination: "https://evil.example/issues/x" }))).toBeNull();
  });

  it("does not capture a return origin outside the allowlist", () => {
    expect(readerNavigationState(navigation({ currentPathname: "/settings" }))).toBeNull();
    expect(readerNavigationState(navigation({ currentPathname: "/publishing" }))).toBeNull();
  });
});

describe("route labels", () => {
  it("maps the main learning routes to concise labels", () => {
    expect(readerReturnLabel("/")).toBe("Today");
    expect(readerReturnLabel("/streams")).toBe("Streams");
    expect(readerReturnLabel("/library")).toBe("Library");
    expect(readerReturnLabel("/review")).toBe("Review");
  });

  it("extracts the stream name from the document title", () => {
    expect(readerReturnLabel("/newsletters/quantum-42", "Quantum Field Theory · Learnloom"))
      .toBe("Quantum Field Theory");
  });

  it("falls back to the generic stream label", () => {
    expect(readerReturnLabel("/newsletters/quantum-42")).toBe(DEFAULT_STREAM_LABEL);
    expect(readerReturnLabel("/newsletters/quantum-42", "Learnloom")).toBe(DEFAULT_STREAM_LABEL);
    expect(readerReturnLabel("/newsletters/quantum-42", "Quantum Field Theory")).toBe(DEFAULT_STREAM_LABEL);
    expect(readerReturnLabel("/newsletters/quantum-42", `x${"y".repeat(MAX_READER_LABEL)}· Learnloom`))
      .toBe(DEFAULT_STREAM_LABEL);
    expect(readerReturnLabel("/publishing")).toBe(DEFAULT_STREAM_LABEL);
    expect(readerReturnLabel("/welcome/new-learner")).toBe(DEFAULT_STREAM_LABEL);
  });

  it("derives stream labels with exact suffix and bounded length", () => {
    expect(streamLabelFromTitle("My Stream · Learnloom")).toBe("My Stream");
    expect(streamLabelFromTitle("  Spaced Stream · Learnloom")).toBe("Spaced Stream");
    expect(streamLabelFromTitle("")).toBe(DEFAULT_STREAM_LABEL);
    expect(streamLabelFromTitle(" · Learnloom")).toBe(DEFAULT_STREAM_LABEL);
    expect(streamLabelFromTitle("x".repeat(MAX_READER_LABEL + 1) + " · Learnloom")).toBe(DEFAULT_STREAM_LABEL);
  });
});

describe("handleReaderReturnClick", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function clickEvent(overrides: Record<string, unknown> = {}) {
    return {
      defaultPrevented: false,
      button: 0,
      metaKey: false,
      ctrlKey: false,
      shiftKey: false,
      altKey: false,
      preventDefault: vi.fn(),
      ...overrides,
    };
  }

  function stubWindow() {
    const go = vi.fn();
    vi.stubGlobal("window", { location: { origin: ORIGIN }, history: { go } });
    return go;
  }

  it("intercepts an unmodified primary click with valid contextual state", () => {
    const go = stubWindow();
    const event = clickEvent();

    const consumed = handleReaderReturnClick(
      event,
      baseState({ href: "/streams", label: "Streams", depth: 3 }),
    );

    expect(consumed).toBe(true);
    expect(event.preventDefault).toHaveBeenCalledOnce();
    expect(go).toHaveBeenCalledWith(-3);
  });

  it("leaves modifier clicks native", () => {
    const go = stubWindow();
    for (const modifier of ["metaKey", "ctrlKey", "shiftKey", "altKey"]) {
      const event = clickEvent({ [modifier]: true });
      expect(handleReaderReturnClick(event, baseState())).toBe(false);
      expect(event.preventDefault).not.toHaveBeenCalled();
      expect(go).not.toHaveBeenCalled();
    }
  });

  it("leaves non-primary buttons and already-handled events native", () => {
    const go = stubWindow();
    const middleClick = clickEvent({ button: 1 });
    expect(handleReaderReturnClick(middleClick, baseState())).toBe(false);
    expect(middleClick.preventDefault).not.toHaveBeenCalled();

    const handled = clickEvent({ defaultPrevented: true });
    expect(handleReaderReturnClick(handled, baseState())).toBe(false);
    expect(handled.preventDefault).not.toHaveBeenCalled();
    expect(go).not.toHaveBeenCalled();
  });

  it("stays native without valid contextual state", () => {
    const go = stubWindow();
    const event = clickEvent();

    expect(handleReaderReturnClick(event, null)).toBe(false);
    expect(handleReaderReturnClick(event, baseState({ v: "other" }))).toBe(false);
    expect(handleReaderReturnClick(event, baseState({ href: "/publishing" }))).toBe(false);
    expect(event.preventDefault).not.toHaveBeenCalled();
    expect(go).not.toHaveBeenCalled();
  });
});
