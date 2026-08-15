import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import LearningShell, { SessionActionsProvider } from "./LearningShell";

describe("LearningShell session controls", () => {
  it("shows logout when the authenticated session action is available", () => {
    const markup = renderToStaticMarkup(
      <SessionActionsProvider onSignOut={vi.fn()}>
        <LearningShell active="today">
          <p>Today</p>
        </LearningShell>
      </SessionActionsProvider>,
    );

    expect(markup).toContain("Log out");
  });

  it("keeps Clerk-only session controls out of demo mode", () => {
    const markup = renderToStaticMarkup(
      <LearningShell active="today">
        <p>Today</p>
      </LearningShell>,
    );

    expect(markup).not.toContain("Log out");
  });

  it("opting into the redesigned shell scopes the warm-neutral dashboard", () => {
    const markup = renderToStaticMarkup(
      <LearningShell active="library" redesigned>
        <p>Library</p>
      </LearningShell>,
    );

    expect(markup).toContain('class="atelier-app atelier-today"');
    expect(markup).toContain("New learning stream");
    expect(markup).toContain("Library");
  });

  it("default shell stays untouched unless redesigned is opted in", () => {
    const markup = renderToStaticMarkup(
      <LearningShell active="today">
        <p>Today</p>
      </LearningShell>,
    );

    expect(markup).not.toContain("atelier-today");
  });
});
