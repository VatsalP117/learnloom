import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  CheckCircle2,
  Clock3,
  ExternalLink,
  Lightbulb,
  Map,
  Sparkles,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { AtelierError, AtelierLoading } from "./LearningShell.jsx";
import { apiJSON } from "./api.js";
import { normalizeDossier } from "./dossierView.js";
import { lessonState, updateLessonState } from "./learningState.js";

export default function IssueDetail({ issueId }) {
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState("");
  const [progress, setProgress] = useState(() => lessonState(issueId).progress ?? 0);
  const latestProgress = useRef(progress);

  useEffect(() => {
    const controller = new AbortController();
    apiJSON(`/api/issues/${encodeURIComponent(issueId)}`, {
      signal: controller.signal,
    })
      .then(setSnapshot)
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, [issueId]);

  useEffect(() => {
    if (!snapshot?.issue?.title) return undefined;
    document.title = `${snapshot.issue.title} · Learnloom`;
    return () => {
      document.title = "Learnloom";
    };
  }, [snapshot?.issue?.title]);

  useEffect(() => {
    function measure() {
      const available = document.documentElement.scrollHeight - window.innerHeight;
      if (available <= 0) return;
      const next = Math.min(100, Math.max(0, (window.scrollY / available) * 100));
      latestProgress.current = next;
      setProgress(next);
    }
    window.addEventListener("scroll", measure, { passive: true });
    measure();
    return () => {
      window.removeEventListener("scroll", measure);
      updateLessonState(issueId, {
        progress: Math.max(lessonState(issueId).progress ?? 0, latestProgress.current),
        lastOpenedAt: new Date().toISOString(),
      });
    };
  }, [issueId]);

  if (!snapshot && !error) {
    return <div className="reader-loading"><AtelierLoading label="Opening your lesson…" /></div>;
  }
  if (error) {
    return <div className="reader-loading"><AtelierError message={error} /></div>;
  }

  return (
    <LessonReader
      {...snapshot}
      dossier={normalizeDossier(snapshot.dossier, snapshot.newsletter)}
      progress={progress}
      onComplete={() => {
        latestProgress.current = 100;
        setProgress(100);
        updateLessonState(issueId, {
          progress: 100,
          completed: true,
          completedAt: new Date().toISOString(),
        });
      }}
    />
  );
}

function LessonReader({ issue, dossier, newsletter, sources, progress, onComplete }) {
  const [completed, setCompleted] = useState(() => lessonState(issue.id).completed);

  return (
    <div className="focus-reader">
      <div className="reader-progress" aria-label={`${Math.round(progress)}% read`}>
        <i style={{ width: `${progress}%` }} />
      </div>
      <header className="reader-toolbar">
        <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
          <ArrowLeft size={15} /> {newsletter.name}
        </a>
        <span>{Math.round(progress)}% read</span>
        <a href="/library">Library <BookOpen size={14} /></a>
      </header>

      <article className="reader-paper">
        <header className="reader-hero">
          <div className="reader-meta">
            <span><BookOpen size={14} /> Today’s lesson</span>
            <span><Clock3 size={14} />{dossier.readTime} min</span>
            <span>{formatDate(issue.createdAt)}</span>
          </div>
          <p className="atelier-eyebrow">{newsletter.name}</p>
          <h1>{issue.title}</h1>
          <p className="reader-deck">{dossier.deck}</p>
          <div className="reader-grounding">
            <span><Check size={13} /> Source-grounded</span>
            <span>Prepared from {sources.length} trusted sources</span>
            <span>{newsletter.learnerLevel} level</span>
          </div>
        </header>

        <div className="reader-layout">
          <main className="reader-content">
            <section className="reader-objective">
              <span><Lightbulb size={19} /></span>
              <div>
                <p className="atelier-eyebrow">Learning objective</p>
                <p>{dossier.objective}</p>
              </div>
            </section>

            {dossier.sections.map((section, index) => (
              <section className="reader-section" id={`section-${index + 1}`} key={section.heading}>
                <p className="atelier-eyebrow">{String(index + 1).padStart(2, "0")} · {section.label}</p>
                <h2>{section.heading}</h2>
                {section.paragraphs.map((paragraph) => <p key={paragraph}>{paragraph}</p>)}
                {section.callout ? <blockquote>{section.callout}</blockquote> : null}
              </section>
            ))}

            <RetrievalSection questions={dossier.retrieval} />

            <section className="reader-application">
              <p className="atelier-eyebrow"><Sparkles size={14} /> Try this in the world</p>
              <p>{dossier.application}</p>
            </section>

            <section className="reader-complete">
              <span><CheckCircle2 size={23} /></span>
              <h2>{completed ? "Lesson complete." : "Close the loop."}</h2>
              <p>
                {completed
                  ? "This lesson is now part of your learning history."
                  : "Mark this lesson complete when you have finished the recall prompts or reflected on the central idea."}
              </p>
              {!completed ? (
                <button
                  className="atelier-primary"
                  type="button"
                  onClick={() => {
                    setCompleted(true);
                    onComplete();
                  }}
                >
                  Mark lesson complete <Check size={15} />
                </button>
              ) : null}
              <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
                Return to this learning stream <ArrowRight size={15} />
              </a>
            </section>
          </main>

          <aside className="reader-aside">
            <nav>
              <p className="atelier-eyebrow"><Map size={14} /> Lesson map</p>
              {dossier.sections.map((section, index) => (
                <a href={`#section-${index + 1}`} key={section.heading}>
                  <span>{String(index + 1).padStart(2, "0")}</span>{section.heading}
                </a>
              ))}
              <a href="#retrieval"><span>R</span>Pause and retrieve</a>
            </nav>
            <div className="reader-sources">
              <p className="atelier-eyebrow">Sources consulted</p>
              {sources.map((source, index) => (
                <a href={source.url} target="_blank" rel="noreferrer" key={source.name}>
                  <span><i>{index + 1}</i>{source.name}</span>
                  <ExternalLink size={13} />
                </a>
              ))}
              <p>
                These sources informed the lesson. Claim-level citation mapping is
                shown only when it is available in the generated artifact.
              </p>
            </div>
          </aside>
        </div>
      </article>
    </div>
  );
}

function RetrievalSection({ questions }) {
  const [open, setOpen] = useState({});
  return (
    <section className="reader-retrieval" id="retrieval">
      <p className="atelier-eyebrow">Pause and retrieve</p>
      <h2>Can you explain it without looking back?</h2>
      <p>Answer aloud or write a few words. Reveal each reflection only after trying.</p>
      <div>
        {questions.map((question, index) => (
          <article key={question}>
            <span>{String(index + 1).padStart(2, "0")}</span>
            <p>{question}</p>
            <button
              type="button"
              aria-expanded={Boolean(open[index])}
              onClick={() => setOpen((current) => ({ ...current, [index]: !current[index] }))}
            >
              {open[index] ? "Hide reflection" : "I’ve thought it through"}
            </button>
            {open[index] ? (
              <small>
                Return to the mechanism and evidence above. If your explanation names
                both the cause and its limits, you have the useful shape of the idea.
              </small>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}

function formatDate(value) {
  return new Intl.DateTimeFormat("en", {
    month: "long",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value));
}
