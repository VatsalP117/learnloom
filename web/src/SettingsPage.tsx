import { BellRing, Check, Mail } from "lucide-react";
import { useEffect, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell";
import { apiJSON } from "./api";
import type { NotificationPreferences, Profile } from "./types";

const defaultPreferences: NotificationPreferences = {
  weeklyRecap: false,
  reentryReminder: true,
  timeZone: "UTC",
};

export default function SettingsPage() {
  const [preferences, setPreferences] = useState(defaultPreferences);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    apiJSON<Profile>("/api/me", { signal: controller.signal })
      .then((profile) => {
        const notifications = profile.notifications;
        setPreferences(notifications?.configured
          ? notifications
          : {
              ...defaultPreferences,
              timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
            });
      })
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      })
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, []);

  async function save() {
    setBusy(true);
    setSaved(false);
    setError("");
    try {
      const response = await apiJSON<{ notifications: NotificationPreferences }>(
        "/api/me/notifications",
        { method: "POST", body: preferences },
      );
      setPreferences(response.notifications);
      setSaved(true);
    } catch (requestError) {
      setError(requestError instanceof Error
        ? requestError.message
        : "Notification preferences could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <LearningShell active="settings">
      <section className="atelier-page settings-page">
        <header className="atelier-page-heading">
          <p className="atelier-eyebrow">Your learning rhythm</p>
          <h1>Prompts & recaps</h1>
          <p>Choose the small signals that keep your learning moving, without turning an unfinished queue into pressure.</p>
        </header>

        {loading ? <AtelierLoading label="Loading your preferences…" /> : null}
        {error ? <AtelierError message={error} /> : null}

        {!loading ? (
          <section className="settings-panel glass-panel">
            <div className="settings-panel-intro">
              <p className="atelier-eyebrow">Your preferences</p>
              <h2>Stay connected without the pressure</h2>
              <p>Weekly recaps keep the thread visible. Re-entry mode makes it easier to return after time away.</p>
            </div>
            <label className="settings-toggle">
              <span><Mail size={18} /></span>
              <span>
                <strong>Weekly learning recap</strong>
                <small>A weekly email with concepts learned, one connection, a retrieval prompt, and a next action.</small>
              </span>
              <input
                type="checkbox"
                checked={preferences.weeklyRecap}
                onChange={(event) => setPreferences({
                  ...preferences,
                  weeklyRecap: event.target.checked,
                })}
              />
            </label>
            <label className="settings-timezone">
              <span>
                <strong>Recap time zone</strong>
                <small>Used to schedule your weekly recap.</small>
              </span>
              <input
                aria-label="Recap time zone"
                value={preferences.timeZone}
                onChange={(event) => setPreferences({
                  ...preferences,
                  timeZone: event.target.value,
                })}
              />
            </label>
            <label className="settings-toggle">
              <span><BellRing size={18} /></span>
              <span>
                <strong>Gentle re-entry mode</strong>
                <small>When you return after a break, Today offers one useful next step instead of a backlog.</small>
              </span>
              <input
                type="checkbox"
                checked={preferences.reentryReminder}
                onChange={(event) => setPreferences({
                  ...preferences,
                  reentryReminder: event.target.checked,
                })}
              />
            </label>
            <div className="settings-actions">
              <button className="atelier-primary" type="button" disabled={busy} onClick={save}>
                {busy ? "Saving…" : "Save preferences"}
              </button>
              {saved ? <span className="settings-saved"><Check size={14} /> Saved</span> : null}
            </div>
          </section>
        ) : null}
      </section>
    </LearningShell>
  );
}
