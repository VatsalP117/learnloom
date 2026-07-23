import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  CircleHelp,
  Clock3,
  ExternalLink,
  Lightbulb,
  Map,
  Sparkles,
} from "lucide-react";
import { useEffect, useState } from "react";
import { ErrorState, Footer, Sidebar, Topbar } from "./App.jsx";
import { apiJSON } from "./api.js";

function IssueDetail({ issueId }) {
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(false);

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
    if (!snapshot?.issue?.title) return;
    document.title = `${snapshot.issue.title} · Learnloom`;
    return () => {
      document.title = "Learnloom · Knowledge Dossiers";
    };
  }, [snapshot?.issue?.title]);

  const newsletters = snapshot?.newsletters ?? [];
  const issue = snapshot?.issue;
  const dossier = snapshot?.dossier;

  return (
    <div className="app">
      <Topbar onMenu={() => setSidebarOpen(true)} />
      <div className="shell">
        <Sidebar
          newsletters={newsletters}
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
          currentNewsletterId={snapshot?.newsletter?.id}
        />
        <main className="content lesson-content">
          <div className="content-inner lesson-inner">
            {!snapshot && !error ? <LessonSkeleton /> : null}
            {error ? <ErrorState message={error} /> : null}
            {issue && dossier ? (
              <LessonReader
                issue={issue}
                dossier={dossier}
                newsletter={snapshot.newsletter}
                sources={snapshot.sources ?? []}
              />
            ) : null}
          </div>
          <Footer />
        </main>
      </div>
    </div>
  );
}

function LessonReader({ issue, dossier, newsletter, sources }) {
  return (
    <article className="lesson-reader">
      <header className="lesson-header">
        <a className="back-link" href={`/newsletters/${encodeURIComponent(newsletter.id)}`}>
          <ArrowLeft size={14} /> {newsletter.name} <span>/</span> Lesson 01
        </a>
        <div className="lesson-meta-row">
          <span className="lesson-kicker"><BookOpen size={14} /> Today’s lesson</span>
          <span><Clock3 size={14} /> {dossier.readTime} min read</span>
          <span>{formatDate(issue.createdAt)}</span>
        </div>
        <h1>{issue.title}</h1>
        <p className="lesson-deck">{dossier.deck}</p>
        <div className="lesson-source-strip">
          <span className="source-grounded"><Check size={13} /> Source-grounded</span>
          <span>Built from {sources.length} trusted sources</span>
          <span>Designed for {newsletter.learnerLevel}-level learning</span>
        </div>
      </header>

      <div className="lesson-layout">
        <div className="lesson-main-column">
          <section className="lesson-objective">
            <div className="lesson-objective-icon"><Lightbulb size={19} /></div>
            <div>
              <span className="lesson-label">Today’s learning objective</span>
              <p>{dossier.objective}</p>
            </div>
          </section>

          {dossier.sections.map((section, index) => (
            <LessonSection key={section.heading} section={section} index={index + 1} />
          ))}

          <section className="lesson-retrieval" id="retrieval">
            <div className="lesson-section-heading">
              <span className="lesson-label">Pause and retrieve</span>
              <CircleHelp size={19} />
            </div>
            <h2>Can you explain it without looking back?</h2>
            <div className="retrieval-grid">
              {dossier.retrieval.map((question, index) => (
                <div className="retrieval-card" key={question}>
                  <span>0{index + 1}</span>
                  <p>{question}</p>
                </div>
              ))}
            </div>
          </section>

          <section className="lesson-application">
            <span className="lesson-label"><Sparkles size={14} /> Try this in the world</span>
            <p>{dossier.application}</p>
          </section>
        </div>

        <aside className="lesson-aside">
          <div className="lesson-map-card">
            <div className="lesson-aside-heading"><Map size={16} /> Lesson map</div>
            {dossier.sections.map((section, index) => (
              <a href={`#lesson-section-${index + 1}`} key={section.heading}>
                <span>0{index + 1}</span>{section.heading}<ArrowRight size={13} />
              </a>
            ))}
            <a href="#retrieval"><span>04</span>Test your model<ArrowRight size={13} /></a>
          </div>
          <div className="lesson-sources-card">
            <span className="lesson-label">Sources in this lesson</span>
            {sources.map((source) => (
              <a href={source.url} key={source.name} target="_blank" rel="noreferrer">
                <span>{source.name}</span><ExternalLink size={13} />
              </a>
            ))}
            <p>Every claim stays attached to the source it came from.</p>
          </div>
        </aside>
      </div>
    </article>
  );
}

function LessonSection({ section, index }) {
  return (
    <section className="lesson-section" id={`lesson-section-${index}`}>
      <div className="lesson-section-heading">
        <span className="lesson-label">0{index} · {section.label}</span>
        {section.icon ? <span className="lesson-section-icon">{section.icon}</span> : null}
      </div>
      <h2>{section.heading}</h2>
      {section.paragraphs.map((paragraph) => <p key={paragraph}>{paragraph}</p>)}
      {section.callout ? (
        <blockquote className="lesson-callout">
          <span>“</span>
          <p>{section.callout}</p>
        </blockquote>
      ) : null}
    </section>
  );
}

function LessonSkeleton() {
  return (
    <div className="lesson-skeleton" aria-label="Loading lesson">
      <span /><span /><span /><span />
    </div>
  );
}

function formatDate(value) {
  if (!value) return "Today";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value));
}

export default IssueDetail;
