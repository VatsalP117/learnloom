import {
  ArrowRight,
  BookOpen,
  BrainCircuit,
  CheckCircle2,
  Clock3,
  Sparkles,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import LearningShell, {
  AtelierError,
  AtelierLoading,
  formatShortDate,
} from "./LearningShell";
import { lessonState } from "./learningState";
import { useWorkspace } from "./useWorkspace";

export default function TodayPage() {
  const workspace = useWorkspace();
  const [, refreshState] = useState(0);

  useEffect(() => {
    const refresh = () => refreshState((value) => value + 1);
    window.addEventListener("learnloom:state", refresh);
    return () => window.removeEventListener("learnloom:state", refresh);
  }, []);

  const readyLessons = useMemo(
    () =>
      workspace.lessons.filter(
        (lesson) =>
          lesson.status === "generated" &&
          lesson.newsletter?.active &&
          !lessonState(lesson.id).completed,
      ),
    [workspace.lessons],
  );
  const primary = readyLessons.find((lesson) => lessonState(lesson.id).progress > 0)
    ?? readyLessons[0]
    ?? workspace.lessons.find((lesson) => lesson.status === "generated");
  const secondary = readyLessons.find((lesson) => lesson.id !== primary?.id);
  const primaryState = primary ? lessonState(primary.id) : null;

  return (
    <LearningShell active="today">
      <section className="today-page">
        <header className="atelier-page-heading today-heading">
          <p className="atelier-eyebrow">Your learning practice</p>
          <h1>{greeting()}.</h1>
          <p>Choose one worthwhile step. Learnloom will keep the larger thread.</p>
        </header>

        {workspace.loading ? <AtelierLoading /> : null}
        {workspace.error ? (
          <AtelierError message={workspace.error} onRetry={workspace.reload} />
        ) : null}

        {!workspace.loading && !workspace.error && !workspace.newsletters.length ? (
          <section className="today-empty glass-panel">
            <span><Sparkles size={24} /></span>
            <p className="atelier-eyebrow">Your first thread</p>
            <h2>Turn a question into a learning practice.</h2>
            <p>
              Choose a subject, the sources you trust, and a rhythm that fits your
              life. Your first lesson can begin as soon as the stream is ready.
            </p>
            <a className="atelier-primary" href="/newsletters/new">
              Create your first stream <ArrowRight size={16} />
            </a>
          </section>
        ) : null}

        {primary ? (
          <div className="today-grid">
            <article className="today-feature glass-panel">
              <div className="today-feature-top">
                <span className="atelier-chip">
                  {primaryState?.progress > 0 ? "Continue learning" : "Ready for you"}
                </span>
                <span><Clock3 size={14} />{primary.newsletter.lessonMinutes} min</span>
              </div>
              <div className="today-feature-copy">
                <p className="atelier-eyebrow">{primary.newsletter.name}</p>
                <h2>{primary.title}</h2>
                <p>
                  {primaryState?.progress > 0
                    ? "Pick up where you left off. Your place has been saved."
                    : `A source-grounded lesson designed for ${primary.newsletter.learnerLevel}-level learning.`}
                </p>
              </div>
              <div className="today-progress">
                <div>
                  <span>Reading progress</span>
                  <strong>{Math.round(primaryState?.progress ?? 0)}%</strong>
                </div>
                <span><i style={{ width: `${primaryState?.progress ?? 0}%` }} /></span>
              </div>
              <a className="atelier-primary" href={`/issues/${encodeURIComponent(primary.id)}`}>
                {primaryState?.progress > 0 ? "Resume lesson" : "Begin lesson"}
                <ArrowRight size={16} />
              </a>
            </article>

            <aside className="today-side">
              {secondary ? (
                <article className="today-synthesis glass-panel">
                  <span className="atelier-icon"><BookOpen size={17} /></span>
                  <p className="atelier-eyebrow">Another thread</p>
                  <h3>{secondary.newsletter.name}</h3>
                  <p>{secondary.title}</p>
                  <a href={`/issues/${encodeURIComponent(secondary.id)}`}>
                    Open lesson <ArrowRight size={14} />
                  </a>
                </article>
              ) : (
                <article className="today-synthesis glass-panel">
                  <span className="atelier-icon"><CheckCircle2 size={17} /></span>
                  <p className="atelier-eyebrow">A clear queue</p>
                  <h3>You are caught up.</h3>
                  <p>Your next lesson will appear here when it is ready.</p>
                </article>
              )}
              <article className="today-review glass-panel">
                <BrainCircuit size={18} />
                <div>
                  <p className="atelier-eyebrow">Recall</p>
                  <strong>Strengthen what you learned</strong>
                  <span>Review questions from recent lessons.</span>
                </div>
                <a href="/review" aria-label="Open review"><ArrowRight size={16} /></a>
              </article>
            </aside>
          </div>
        ) : null}

        {!workspace.loading && workspace.newsletters.length ? (
          <section className="today-footer-row">
            <div>
              <p className="atelier-eyebrow">Your rhythm</p>
              <strong>{workspace.newsletters.filter((item) => item.active).length} active learning streams</strong>
              <span>Latest archive update {formatShortDate(workspace.lessons[0]?.createdAt)}</span>
            </div>
            <a href="/streams">Tune your streams <ArrowRight size={15} /></a>
          </section>
        ) : null}
      </section>
    </LearningShell>
  );
}

function greeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}
