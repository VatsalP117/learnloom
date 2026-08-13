import { BellRing, Check, CreditCard, Mail, Sparkles } from "lucide-react";
import { useEffect, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell";
import { apiJSON } from "./api";
import type {
  BillingEntitlementResponse,
  NotificationPreferences,
  Profile,
} from "./types";

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
  const [billing, setBilling] = useState<BillingEntitlementResponse | null>(null);
  const [billingBusy, setBillingBusy] = useState(false);
  const [feedbackReason, setFeedbackReason] = useState("");
  const [feedbackNote, setFeedbackNote] = useState("");
  const [feedbackSaved, setFeedbackSaved] = useState(false);

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

  useEffect(() => {
    const controller = new AbortController();
    apiJSON<BillingEntitlementResponse>("/api/me/billing", { signal: controller.signal })
      .then(setBilling)
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
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

  async function openBilling(action: "checkout" | "portal") {
    setBillingBusy(true);
    setError("");
    try {
      const response = await apiJSON<{ url: string }>(`/api/me/billing/${action}`, {
        method: "POST",
      });
      const destination = new URL(response.url);
      if (destination.protocol !== "https:") throw new Error("Billing returned an unsafe link.");
      window.location.assign(destination.href);
    } catch (requestError) {
      setError(requestError instanceof Error
        ? requestError.message
        : "Billing could not be opened.");
      setBillingBusy(false);
    }
  }

  async function saveBillingFeedback() {
    if (!billing || !feedbackReason) return;
    setBillingBusy(true);
    setFeedbackSaved(false);
    setError("");
    try {
      await apiJSON("/api/me/billing/feedback", {
        method: "POST",
        body: {
          context: billing.billing.planId === "free" ? "non_conversion" : "cancellation",
          reasonCode: feedbackReason,
          note: feedbackNote,
        },
      });
      setFeedbackSaved(true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Feedback could not be saved.");
    } finally {
      setBillingBusy(false);
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
          <>
          {billing ? (
            <section className="settings-panel billing-panel glass-panel" aria-labelledby="billing-heading">
              <div className="settings-panel-intro">
                <p className="atelier-eyebrow">Plan & usage</p>
                <h2 id="billing-heading">{billing.billing.planName}</h2>
                <p>
                  {billing.billing.generationRemaining} of {billing.billing.generationAllowance} lesson generations remain in this period.
                </p>
              </div>
              <div className="billing-usage" aria-label={`${billing.billing.generationUsed} generations used`}>
                <span style={{ width: `${Math.min(100, (billing.billing.generationUsed / Math.max(1, billing.billing.generationAllowance)) * 100)}%` }} />
              </div>
              <div className="billing-summary">
                <span><Sparkles size={16} /></span>
                <span>
                  <strong>{billing.billing.generationUsed} used</strong>
                  <small>Resets {new Date(billing.billing.periodEnd).toLocaleDateString()}</small>
                </span>
              </div>
              {billing.billing.entitlementStatus === "grace" ? (
                <p className="billing-notice" role="status">Payment needs attention. Generation remains available temporarily during the grace period.</p>
              ) : null}
              {billing.billing.entitlementStatus === "generation_paused" ? (
                <p className="billing-notice billing-notice-error" role="status">New lesson generation is paused. Your existing lessons remain available.</p>
              ) : null}
              <div className="settings-actions">
                {billing.billing.planId === "free" ? (
                  <button
                    className="atelier-primary"
                    type="button"
                    disabled={billingBusy || !billing.commerceAvailable}
                    onClick={() => openBilling("checkout")}
                  >
                    <CreditCard size={15} /> {billingBusy ? "Opening…" : "Start Pro"}
                  </button>
                ) : (
                  <button
                    className="atelier-secondary"
                    type="button"
                    disabled={billingBusy || !billing.commerceAvailable}
                    onClick={() => openBilling("portal")}
                  >
                    <CreditCard size={15} /> {billingBusy ? "Opening…" : "Manage billing"}
                  </button>
                )}
                {!billing.commerceAvailable ? <small>Paid plans are not available in this environment.</small> : null}
              </div>
              {(billing.billing.planId === "free" || ["canceled", "refunded"].includes(billing.billing.subscriptionStatus)) ? (
                <details className="billing-feedback">
                  <summary>{billing.billing.planId === "free" ? "Tell us why Pro is not right yet" : "Tell us why you left"}</summary>
                  <label>
                    <span>Primary reason</span>
                    <select value={feedbackReason} onChange={(event) => setFeedbackReason(event.target.value)}>
                      <option value="">Choose one</option>
                      <option value="too_expensive">Too expensive</option>
                      <option value="insufficient_value">Not enough learning value</option>
                      <option value="quality_concerns">Lesson or source quality</option>
                      <option value="reliability_concerns">Reliability problems</option>
                      <option value="allowance_too_low">Allowance is too low</option>
                      <option value="missing_feature">Missing a feature I need</option>
                      <option value="no_longer_needed">No longer needed</option>
                      <option value="other">Other</option>
                    </select>
                  </label>
                  <label>
                    <span>Anything else? <small>Optional · do not include sensitive information</small></span>
                    <textarea maxLength={1000} value={feedbackNote} onChange={(event) => setFeedbackNote(event.target.value)} />
                  </label>
                  <button className="atelier-secondary" type="button" disabled={!feedbackReason || billingBusy} onClick={saveBillingFeedback}>Save feedback</button>
                  {feedbackSaved ? <span className="settings-saved"><Check size={14} /> Thank you</span> : null}
                </details>
              ) : null}
            </section>
          ) : null}
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
          </>
        ) : null}
      </section>
    </LearningShell>
  );
}
