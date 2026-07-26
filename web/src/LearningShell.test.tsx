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
});
