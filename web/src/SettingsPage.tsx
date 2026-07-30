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
          <p>Choose the prompts that help. Learnloom will not turn an unfinished queue into pressure.</p>
        </header>

        {loading ? <AtelierLoading label="Loading your preferences…" /> : null}
        {error ? <AtelierError message={error} /> : null}

        {!loading ? (
          <section className="settings-panel glass-panel">
            <label>
              <span><Mail size={18} /></span>
              <span>
                <strong>Weekly learning recap</strong>
                <small>Concepts learned, one connection, one retrieval prompt, and the best next action.</small>
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
            <label>
              <span><BellRing size={18} /></span>
              <span>
                <strong>Gentle re-entry mode</strong>
                <small>After time away, Today offers one useful action without listing a backlog.</small>
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
            <label className="settings-timezone">
              <span>Recap time zone</span>
              <input
                value={preferences.timeZone}
                onChange={(event) => setPreferences({
                  ...preferences,
                  timeZone: event.target.value,
                })}
              />
            </label>
            <div>
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
