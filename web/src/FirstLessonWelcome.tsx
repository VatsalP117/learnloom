import {
  ArrowRight,
  Bell,
  BookOpen,
  Check,
  Clock3,
  ExternalLink,
  Sparkles,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import CalmLoader from "./CalmLoader";
import LearningShell, { AtelierError } from "./LearningShell";
import { apiJSON, demoMode } from "./api";
import { firstLessonPreparation } from "./preparation";
import type { Newsletter, NewsletterDetailResponse } from "./types";

export default function FirstLessonWelcome({ newsletterId }) {
  const [snapshot, setSnapshot] = useState<NewsletterDetailResponse | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");

  const load = useCallback(async ({ signal }: { signal?: AbortSignal } = {}) => {
    try {
      const result = await apiJSON<NewsletterDetailResponse>(
        `/api/newsletters/${encodeURIComponent(newsletterId)}`,
        { signal },
      );
      setSnapshot(result);
      setError("");
    } catch (requestError) {
      if (requestError.name !== "AbortError") setError(requestError.message);
    }
  }, [newsletterId]);

  const issues = snapshot?.issues ?? [];
  const generated = issues.find((issue) => issue.status === "generated");
  const awaitingApproval = issues.find((issue) => issue.status === "awaiting_approval");
  const waiting = issues.some((issue) => ["queued", "generating"].includes(issue.status));

  useEffect(() => {
    const controller = new AbortController();
    void load({ signal: controller.signal });
    return () => controller.abort();
  }, [load]);

  useEffect(() => {
    if (!waiting || generated || awaitingApproval) return undefined;
    const interval = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(interval);
  }, [awaitingApproval, generated, load, waiting]);

  useEffect(() => {
    if (!snapshot?.newsletter || (!waiting && !awaitingApproval)) return undefined;
    const recordExit = () => {
      void apiJSON(
        `/api/newsletters/${encodeURIComponent(snapshot.newsletter.id)}/preparation-wait-exited`,
        { method: "POST", keepalive: true },
      ).catch(() => {});
    };
    window.addEventListener("pagehide", recordExit, { once: true });
    return () => window.removeEventListener("pagehide", recordExit);
  }, [awaitingApproval, snapshot, waiting]);

  const plan = snapshot?.newsletter ? orientationPlan(snapshot.newsletter) : null;

  async function enableEmail() {
    if (!snapshot?.newsletter) return;
    setBusy("email");
    setError("");
    try {
      await apiJSON(
        `/api/newsletters/${encodeURIComponent(snapshot.newsletter.id)}/delivery`,
        { method: "POST", body: { enabled: true } },
      );
      await load();
      setNotice("We’ll email this lesson and future lessons in this path when they are ready.");
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  async function leaveSafely() {
    if (snapshot?.newsletter && (waiting || awaitingApproval)) {
      await apiJSON(
        `/api/newsletters/${encodeURIComponent(snapshot.newsletter.id)}/preparation-wait-exited`,
        { method: "POST" },
      ).catch(() => {});
    }
    window.location.assign("/");
  }

  if (!snapshot && !error) {
    return <CalmLoader label="Building your learning path…" detail="Opening your first learning orientation." />;
  }

  if (!snapshot?.newsletter) {
    return (
      <LearningShell active="today">
        <section className="atelier-page"><AtelierError message={error || "This learning path is unavailable."} onRetry={load} /></section>
      </LearningShell>
    );
  }

  const newsletter = snapshot.newsletter;
  return (
    <LearningShell active="today">
      <section className="atelier-page first-welcome-page">
        {error ? <AtelierError message={error} onRetry={load} /> : null}
        <header className="first-welcome-hero">
          <span className="atelier-icon"><Sparkles size={19} /></span>
          <p className="atelier-eyebrow">Your learning path is underway</p>
          <h1>{generated ? "Your first lesson is ready." : awaitingApproval ? "Your evidence portfolio is ready." : "Start thinking before the lesson arrives."}</h1>
          <p>
            {generated
              ? "Open the lesson when you have a focused few minutes."
              : awaitingApproval
                ? "You asked to check the sources first. Review the evidence mix, make any changes, then approve it."
                : firstLessonPreparation.explanation}
          </p>
          <div className="first-welcome-actions">
            {generated ? (
              <a className="atelier-primary" href={lessonHref(generated.id)}>Open first lesson <ArrowRight size={15} /></a>
            ) : awaitingApproval ? (
              <a className="atelier-primary" href={`/newsletters/${encodeURIComponent(newsletter.id)}#source-portfolio`}>Review sources <ArrowRight size={15} /></a>
            ) : (
              <button className="atelier-primary" type="button" onClick={leaveSafely}>Go to Today—you can leave safely <ArrowRight size={15} /></button>
            )}
            <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>Tune this path <ExternalLink size={14} /></a>
          </div>
        </header>

        {notice ? <div className="atelier-notice" role="status"><Check size={15} /><span>{notice}</span></div> : null}

        <div className="first-welcome-grid">
          <article className="welcome-plan-card glass-panel">
            <p className="atelier-eyebrow">Likely first lesson</p>
            <h2>{plan.title}</h2>
            <p>{plan.objective}</p>
            <ol>
              {plan.concepts.map((concept, index) => (
                <li key={concept}><span>{index + 1}</span><strong>{concept}</strong></li>
              ))}
            </ol>
            <small>The path will refine as Learnloom resolves the evidence and connects later lessons to what you complete.</small>
          </article>

          <aside className="welcome-preflight glass-panel">
            <p className="atelier-eyebrow">Two-minute preflight</p>
            <h2>Capture your current model mentally.</h2>
            <p>Before reading, try answering these without searching:</p>
            <ul>
              <li>What do you currently think is the central mechanism?</li>
              <li>What evidence would change your mind?</li>
              <li>Where do you expect the simple explanation to break?</li>
            </ul>
            <small>You’ll revisit these after the lesson. The contrast is where durable learning starts.</small>
          </aside>
        </div>

        <section className="welcome-status glass-panel" aria-live="polite">
          <div>
            <span className={waiting || awaitingApproval || generated ? "complete" : ""}><Check size={13} /></span>
            <strong>Learning intent</strong>
            <small>{newsletter.learnerGoal}</small>
          </div>
          <div>
            <span className={awaitingApproval || generated ? "complete" : "active"}><BookOpen size={13} /></span>
            <strong>Evidence portfolio</strong>
            <small>{awaitingApproval ? "Ready for your review" : generated ? "Resolved and frozen" : "Finding independent roles"}</small>
          </div>
          <div>
            <span className={generated ? "complete" : ""}><Clock3 size={13} /></span>
            <strong>First lesson</strong>
            <small>{generated ? "Ready to open" : "Preparing in the background"}</small>
          </div>
        </section>

        {!generated && !newsletter.emailEnabled ? (
          <section className="welcome-notification glass-panel">
            <span><Bell size={17} /></span>
            <div><strong>Want an email when it is ready?</strong><small>This also enables email delivery for future lessons in this path. Your Learnloom library remains the permanent home.</small></div>
            <button type="button" disabled={busy === "email" || !snapshot.resendConfigured} onClick={enableEmail}>
              {snapshot.resendConfigured ? busy === "email" ? "Enabling…" : "Email me" : "Email unavailable"}
            </button>
          </section>
        ) : null}
      </section>
    </LearningShell>
  );
}

export function orientationPlan(newsletter: Newsletter) {
  const topic = boundedOrientationText(newsletter.topic, 90);
  const objective = newsletter.learnerLevel === "beginner"
    ? "Build clear foundations, explain the basic mechanism, and avoid common misconceptions."
    : newsletter.learnerLevel === "advanced"
      ? "Stress-test the dominant model, compare evidence, and identify consequential edge cases."
      : "Explain the core mechanisms, test the evidence, and recognize important limitations.";
  return {
    title: `Build a working model of ${topic}`,
    objective,
    concepts: [
      `Foundations and boundaries of ${topic}`,
      "Core mechanisms and causal relationships",
      "Evidence quality and counterarguments",
      newsletter.learnerGoal ? `Apply the model to: ${boundedOrientationText(newsletter.learnerGoal, 100)}` : "Practical applications and failure modes",
    ],
  };
}

function boundedOrientationText(value: string, maximum: number) {
  const clean = value.trim();
  const runes = Array.from(clean);
  return runes.length <= maximum
    ? clean
    : `${runes.slice(0, maximum - 1).join("").trim()}…`;
}

function lessonHref(issueId: string) {
  return demoMode ? `/?demoIssue=${encodeURIComponent(issueId)}` : `/issues/${encodeURIComponent(issueId)}`;
}
