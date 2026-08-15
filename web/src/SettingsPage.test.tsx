import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

const { apiJSONMock } = vi.hoisted(() => ({ apiJSONMock: vi.fn() }));

vi.mock("./api", () => ({ apiJSON: apiJSONMock }));

import SettingsPage from "./SettingsPage";

describe("redesigned Settings page render", () => {
  it("renders the redesigned shell and the loading route without network requests", () => {
    const markup = renderToStaticMarkup(<SettingsPage />);

    expect(markup).toContain('class="atelier-app atelier-today"');
    expect(markup).toContain("<h1>Prompts &amp; recaps</h1>");
    expect(markup).toContain("Loading your preferences…");
    // Server rendering never runs effects, so the two-area workspace stays
    // hidden and no /api/me or /api/me/billing request is ever attempted.
    expect(apiJSONMock).not.toHaveBeenCalled();
    expect(markup).not.toContain("settings-workspace");
  });
});
