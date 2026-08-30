import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

const { apiJSONMock } = vi.hoisted(() => ({ apiJSONMock: vi.fn() }));

vi.mock("./api", () => ({ apiJSON: apiJSONMock }));

import SettingsPage from "./SettingsPage";
import type { NotificationPreferences } from "./types";

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

  it("renders the preferences workspace immediately from initial notifications without fetching /api/me", () => {
    const initialNotifications: NotificationPreferences = {
      configured: true,
      weeklyRecap: true,
      reentryReminder: false,
      timeZone: "America/New_York",
    };

    const markup = renderToStaticMarkup(
      <SettingsPage initialNotifications={initialNotifications} />,
    );

    expect(markup).toContain('class="atelier-app atelier-today"');
    expect(markup).toContain("settings-workspace");
    expect(markup).not.toContain("Loading your preferences…");
    expect(markup).toContain('aria-label="Recap time zone" value="America/New_York"');
    // Weekly recap is checked; gentle re-entry (false) is not.
    expect(markup).toContain('type="checkbox" checked=""');
    expect(markup.match(/checked=""/g) ?? []).toHaveLength(1);
    expect(markup).toContain("Save preferences");
    // The seeded profile values replace the duplicate first-mount /api/me
    // request; nothing in this render path touches the network mock.
    expect(apiJSONMock).not.toHaveBeenCalled();
  });

  it("falls back to defaults with the local time zone for unconfigured preferences", () => {
    const localTimeZone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";

    const markup = renderToStaticMarkup(
      <SettingsPage
        initialNotifications={{
          configured: false,
          weeklyRecap: false,
          reentryReminder: true,
          timeZone: "UTC",
        }}
      />,
    );

    expect(markup).toContain("settings-workspace");
    expect(markup).not.toContain("Loading your preferences…");
    expect(markup).toContain(`aria-label="Recap time zone" value="${localTimeZone}"`);
    // Only the default (reentry reminder) is checked.
    expect(markup.match(/checked=""/g) ?? []).toHaveLength(1);
    expect(apiJSONMock).not.toHaveBeenCalled();
  });
});
