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
  const [checkoutStatus, setCheckoutStatus] = useState<"" | "activating" | "active">("");

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
    const checkoutComplete = new URLSearchParams(window.location.search).get("checkout") === "complete";
    if (checkoutComplete) setCheckoutStatus("activating");
    async function refreshBilling() {
      for (let attempt = 0; attempt < (checkoutComplete ? 6 : 1); attempt += 1) {
        const next = await apiJSON<BillingEntitlementResponse>("/api/me/billing", { signal: controller.signal });
        setBilling(next);
        if (["essential", "pro"].includes(next.billing.planId)) {
          if (checkoutComplete) setCheckoutStatus("active");
          return;
        }
        if (checkoutComplete) await new Promise((resolve) => window.setTimeout(resolve, 2_000));
      }
    }
    void refreshBilling().catch((requestError) => {
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

  async function openBilling(action: "checkout" | "portal", planId?: "essential" | "pro") {
    setBillingBusy(true);
    setError("");
    try {
      const response = await apiJSON<{ url: string }>(`/api/me/billing/${action}`, {
        method: "POST",
        body: action === "checkout" ? { planId } : undefined,
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
          context: billing.billing.planId === "none" ? "non_conversion" : "cancellation",
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
    <LearningShell active="settings" redesigned>
      <section className="atelier-page settings-page">
        <header className="atelier-page-heading">
          <p className="atelier-eyebrow">Your learning rhythm</p>
          <h1>Prompts & recaps</h1>
          <p>Choose the small signals that keep your learning moving, without turning an unfinished queue into pressure.</p>
        </header>

        {loading ? <AtelierLoading label="Loading your preferences…" /> : null}
        {error ? <AtelierError message={error} /> : null}

        {!loading ? (
          <div className="settings-workspace">
          {billing ? (
            <section className="settings-panel billing-panel glass-panel" aria-labelledby="billing-heading">
              <div className="settings-panel-intro">
                <p className="atelier-eyebrow">Plan & usage</p>
                <h2 id="billing-heading">{billing.billing.planName}</h2>
                <p>
                  {billing.billing.generationUnlimited
                    ? "Unlimited lesson generations, with fair-use safeguards."
                    : "Generation is available after you choose a paid plan."}
                </p>
              </div>
              <div className="billing-summary">
                <span><Sparkles size={16} /></span>
                <span>
                  <strong>{billing.billing.streamUnlimited ? "Unlimited learning streams" : `${billing.billing.streamUsed} of ${billing.billing.streamAllowance ?? 0} streams used`}</strong>
                  <small>{billing.billing.generationUsed} lessons generated this period</small>
                </span>
              </div>
              {checkoutStatus === "activating" ? <p className="billing-notice" role="status">Payment completed. Paddle is activating your plan; this usually takes a few seconds.</p> : null}
              {checkoutStatus === "active" ? <p className="billing-notice" role="status">Your paid plan is active.</p> : null}
              {billing.billing.cancelAtPeriodEnd ? <p className="billing-notice" role="status">Your subscription is scheduled to end after the current billing period. Existing lessons will remain readable.</p> : null}
              {billing.billing.entitlementStatus === "grace" ? (
                <p className="billing-notice" role="status">Payment needs attention. Generation remains available temporarily during the grace period.</p>
              ) : null}
              {billing.billing.entitlementStatus === "generation_paused" ? (
                <p className="billing-notice billing-notice-error" role="status">New lesson generation is paused. Your existing lessons remain available.</p>
              ) : null}
              <div className="settings-actions">
                {billing.billing.planId === "none" ? (
                  <>
                    <button className="atelier-secondary" type="button" disabled={billingBusy || !billing.commerceAvailable} onClick={() => openBilling("checkout", "essential")}>
                      <CreditCard size={15} /> {billingBusy ? "Opening…" : "Start Essential · $9/mo"}
                    </button>
                    <button className="atelier-primary" type="button" disabled={billingBusy || !billing.commerceAvailable} onClick={() => openBilling("checkout", "pro")}>
                      <CreditCard size={15} /> {billingBusy ? "Opening…" : "Start Pro · $19/mo"}
                    </button>
                  </>
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
              {(billing.billing.planId === "none" || ["canceled", "refunded"].includes(billing.billing.subscriptionStatus)) ? (
                <details className="billing-feedback">
                  <summary>{billing.billing.planId === "none" ? "Tell us why a paid plan is not right yet" : "Tell us why you left"}</summary>
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
          </div>
        ) : null}
      </section>
    </LearningShell>
  );
}
