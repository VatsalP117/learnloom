import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  CheckCircle2,
  Clock3,
  Copy,
  ExternalLink,
  Highlighter,
  Lightbulb,
  Map,
  MessageCircleQuestion,
  NotebookPen,
  Sparkles,
  Trash2,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import CalmLoader from "./CalmLoader";
import { AtelierError } from "./LearningShell";
import { apiJSON } from "./api";
import { normalizeDossier } from "./dossierView";
import { lessonState, updateLessonState } from "./learningState";

interface NoteDraft {
  kind: "note" | "question" | "highlight";
  anchorType: "lesson" | "claim" | "source" | "section";
  anchorId: string;
  body: string;
  quotedText: string;
}

export default function IssueDetail({ issueId }) {
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState("");
  const [progress, setProgress] = useState(() => lessonState(issueId).progress ?? 0);
  const [completionError, setCompletionError] = useState("");
  const latestProgress = useRef(progress);
  const renderedProgress = useRef(Math.round(progress));
  const persistedProgress = useRef(Math.floor(progress));

  useEffect(() => {
    const controller = new AbortController();
    apiJSON(`/api/issues/${encodeURIComponent(issueId)}`, {
      signal: controller.signal,
    })
      .then((nextSnapshot) => {
        setSnapshot(nextSnapshot);
        void apiJSON(`/api/issues/${encodeURIComponent(issueId)}/opened`, {
          method: "POST",
        }).catch(() => {
          // Activation measurement must never interrupt reading.
        });
      })
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
    let frame = 0;
    function measure() {
      if (frame) return;
      frame = window.requestAnimationFrame(() => {
        frame = 0;
        const available = document.documentElement.scrollHeight - window.innerHeight;
        if (available <= 0) return;
        const next = Math.round(
          Math.min(100, Math.max(0, (window.scrollY / available) * 100)),
        );
        latestProgress.current = next;
        if (next === renderedProgress.current) return;
        renderedProgress.current = next;
        setProgress(next);
      });
    }
    window.addEventListener("scroll", measure, { passive: true });
    measure();
    return () => {
      window.removeEventListener("scroll", measure);
      if (frame) window.cancelAnimationFrame(frame);
      void persistLessonProgress(issueId, latestProgress.current);
      updateLessonState(issueId, {
        progress: Math.max(lessonState(issueId).progress ?? 0, latestProgress.current),
        lastOpenedAt: new Date().toISOString(),
      });
    };
  }, [issueId]);

  useEffect(() => {
    const next = Math.min(99, Math.floor(progress));
    if (next < 1 || next <= persistedProgress.current) return undefined;
    const timeout = window.setTimeout(() => {
      persistedProgress.current = next;
      void persistLessonProgress(issueId, next);
    }, 1500);
    return () => window.clearTimeout(timeout);
  }, [issueId, progress]);

  if (!snapshot && !error) {
    return <CalmLoader label="Opening your lesson…" detail="Bringing you back to the page." />;
  }
  if (error) {
    return <div className="reader-loading"><AtelierError message={error} /></div>;
  }

  return (
    <LessonReader
      {...snapshot}
      dossier={normalizeDossier(snapshot.dossier, snapshot.newsletter)}
      progress={progress}
      completionError={completionError}
      onComplete={async () => {
        setCompletionError("");
        try {
          await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/complete`, {
            method: "POST",
          });
        } catch (requestError) {
          setCompletionError(
            requestError instanceof Error
              ? requestError.message
              : "The lesson could not be marked complete.",
          );
          return false;
        }
        latestProgress.current = 100;
        renderedProgress.current = 100;
        setProgress(100);
        updateLessonState(issueId, {
          progress: 100,
          completed: true,
          completedAt: new Date().toISOString(),
        });
        return true;
      }}
    />
  );
}

async function persistLessonProgress(issueId: string, progress: number) {
  const next = Math.min(99, Math.floor(progress));
  if (next < 1) return;
  try {
    await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/progress`, {
      method: "POST",
      body: { progress: next },
    });
  } catch {
    // Reading remains uninterrupted; local progress is retained for a later sync.
  }
}

function LessonReader({
  issue,
  dossier,
  newsletter,
  sources,
  feedback,
  notes: initialNotes = [],
  progress,
  completionError,
  onComplete,
}) {
  const [completed, setCompleted] = useState(() => lessonState(issue.id).completed);
  const [completing, setCompleting] = useState(false);
  const [notes, setNotes] = useState(initialNotes);
  const [noteDraft, setNoteDraft] = useState<NoteDraft | null>(null);

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

            {dossier.whyNow || dossier.continuityBridge ? (
              <section className="reader-orientation">
                {dossier.whyNow ? (
                  <div>
                    <p className="atelier-eyebrow">Why this lesson now</p>
                    <p>{dossier.whyNow}</p>
                  </div>
                ) : null}
                {dossier.continuityBridge ? (
                  <div>
                    <p className="atelier-eyebrow">Connection to prior learning</p>
                    <p>{dossier.continuityBridge}</p>
                  </div>
                ) : null}
                {dossier.concepts?.length ? (
                  <div className="reader-concepts">
                    <p className="atelier-eyebrow">Concepts in this lesson</p>
                    <p>{dossier.concepts.map((concept) => <span key={concept}>{concept}</span>)}</p>
                  </div>
                ) : null}
              </section>
            ) : null}

            {dossier.sections.map((section, index) => (
              <section className="reader-section" id={`section-${index + 1}`} key={section.heading}>
                <p className="atelier-eyebrow">{String(index + 1).padStart(2, "0")} · {section.label}</p>
                <h2>{section.heading}</h2>
                {section.paragraphs.map((paragraph, paragraphIndex) => {
                  const value = typeof paragraph === "string"
                    ? { text: paragraph, sourceIds: [] }
                    : paragraph;
                  return (
                    <p key={`${section.heading}-${paragraphIndex}`}>
                      {value.text}
                      {value.sourceIds.map((sourceID) => {
                        const sourceIndex = sources.findIndex((source) => source.id === sourceID);
                        if (sourceIndex < 0) return null;
                        return (
                          <a
                            className="reader-citation"
                            href={`#source-${sourceID}`}
                            title={`See source ${sourceIndex + 1}`}
                            aria-label={`See source ${sourceIndex + 1}`}
                            key={sourceID}
                          >
                            [{sourceIndex + 1}]
                          </a>
                        );
                      })}
                    </p>
                  );
                })}
                {section.callout ? <blockquote>{section.callout}</blockquote> : null}
              </section>
            ))}

            {dossier.claims?.length ? (
              <section className="reader-claims">
                <p className="atelier-eyebrow">Evidence map</p>
                <h2>Claims you can inspect</h2>
                <div>
                  {dossier.claims.map((claim) => {
                    const claimSources = claim.sourceIds
                      .map((sourceID) => sources.find((source) => source.id === sourceID))
                      .filter(Boolean);
                    const citation = [
                      claim.text,
                      ...claimSources.map((source) => `${source.name}: ${source.url}`),
                    ].join("\n");
                    return (
                      <article key={claim.id}>
                        <p>{claim.text}</p>
                        <div>
                          {claimSources.map((source) => (
                            <a href={`#source-${source.id}`} key={source.id}>
                              {source.id}
                            </a>
                          ))}
                        </div>
                        <footer>
                          <CopyButton text={citation} label="Copy citation" />
                          <button
                            type="button"
                            onClick={() => setNoteDraft({
                              kind: "question",
                              anchorType: "claim",
                              anchorId: claim.id,
                              quotedText: claim.text,
                              body: "",
                            })}
                          >
                            <MessageCircleQuestion size={13} /> Question this claim
                          </button>
                        </footer>
                      </article>
                    );
                  })}
                </div>
              </section>
            ) : null}

            <RetrievalSection
              questions={dossier.retrievalItems?.length
                ? dossier.retrievalItems
                : dossier.retrieval}
            />

            <section className="reader-application">
              <p className="atelier-eyebrow"><Sparkles size={14} /> Try this in the world</p>
              <p>{dossier.application}</p>
            </section>

            <LessonFeedbackPanel issueId={issue.id} initialFeedback={feedback} />

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
                  disabled={completing}
                  onClick={async () => {
                    setCompleting(true);
                    if (!(await onComplete())) {
                      setCompleting(false);
                      return;
                    }
                    setCompleted(true);
                  }}
                >
                  {completing ? "Saving…" : "Mark lesson complete"} <Check size={15} />
                </button>
              ) : null}
              {completionError ? <p role="alert">{completionError}</p> : null}
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
                <div id={`source-${source.id ?? `S${index + 1}`}`} key={`${source.id}-${source.name}`}>
                  <a
                    href={source.url}
                    target="_blank"
                    rel="noreferrer"
                  >
                    <span><i>{index + 1}</i>{source.name}</span>
                    <ExternalLink size={13} />
                  </a>
                  <CopyButton
                    text={`${source.name}. ${source.url}`}
                    label={`Copy citation for ${source.name}`}
                    compact
                  />
                </div>
              ))}
              <p>
                These sources informed the lesson. Claim-level citation mapping is
                shown only when it is available in the generated artifact.
              </p>
            </div>
            <LessonNotes
              issueId={issue.id}
              notes={notes}
              setNotes={setNotes}
              draft={noteDraft}
              setDraft={setNoteDraft}
            />
          </aside>
        </div>
      </article>
    </div>
  );
}

function CopyButton({ text, label, compact = false }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      className={compact ? "reader-copy compact" : "reader-copy"}
      type="button"
      aria-label={label}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1500);
        } catch {
          setCopied(false);
        }
      }}
    >
      {copied ? <Check size={13} /> : <Copy size={13} />}
      {compact ? null : copied ? "Copied" : label}
    </button>
  );
}

function LessonNotes({ issueId, notes, setNotes, draft, setDraft }) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  function startNote() {
    setDraft({
      kind: "note",
      anchorType: "lesson",
      anchorId: "",
      quotedText: "",
      body: "",
    });
  }

  function startHighlight() {
    const quotedText = window.getSelection()?.toString().trim().slice(0, 1200) ?? "";
    if (!quotedText) {
      setError("Select a passage in the lesson first.");
      return;
    }
    setError("");
    setDraft({
      kind: "highlight",
      anchorType: "lesson",
      anchorId: "",
      quotedText,
      body: "",
    });
  }

  async function save() {
    if (!draft?.body.trim() || saving) return;
    setSaving(true);
    setError("");
    try {
      const note = await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/notes`, {
        method: "POST",
        body: draft,
      });
      setNotes((current) => [note, ...current]);
      setDraft(null);
    } catch (requestError) {
      setError(requestError instanceof Error
        ? requestError.message
        : "The note could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  async function remove(noteId) {
    setError("");
    try {
      await apiJSON(`/api/notes/${encodeURIComponent(noteId)}`, {
        method: "DELETE",
      });
      setNotes((current) => current.filter((note) => note.id !== noteId));
    } catch (requestError) {
      setError(requestError instanceof Error
        ? requestError.message
        : "The note could not be deleted.");
    }
  }

  return (
    <section className="reader-notes" id="lesson-notes">
      <p className="atelier-eyebrow"><NotebookPen size={13} /> Your notes</p>
      <div className="reader-note-actions">
        <button type="button" onClick={startNote}><NotebookPen size={13} /> Add note</button>
        <button type="button" onClick={startHighlight}><Highlighter size={13} /> Save selection</button>
      </div>
      {draft ? (
        <div className="reader-note-composer">
          {draft.quotedText ? <blockquote>{draft.quotedText}</blockquote> : null}
          <textarea
            autoFocus
            maxLength={4000}
            rows={4}
            placeholder={draft.kind === "question"
              ? "What do you want to verify or connect?"
              : draft.kind === "highlight"
                ? "Why is this passage worth keeping?"
                : "Capture an idea in your own words…"}
            value={draft.body}
            onChange={(event) => setDraft({ ...draft, body: event.target.value })}
          />
          <div>
            <button type="button" onClick={() => setDraft(null)}>Cancel</button>
            <button type="button" disabled={!draft.body.trim() || saving} onClick={save}>
              {saving ? "Saving…" : "Save"}
            </button>
          </div>
        </div>
      ) : null}
      <div className="reader-note-list">
        {notes.map((note) => (
          <article key={note.id}>
            <span>{note.kind}</span>
            {note.quotedText ? <blockquote>{note.quotedText}</blockquote> : null}
            <p>{note.body}</p>
            <button type="button" aria-label="Delete note" onClick={() => remove(note.id)}>
              <Trash2 size={12} />
            </button>
          </article>
        ))}
        {!notes.length && !draft ? <small>No notes yet. Keep only what will help you think later.</small> : null}
      </div>
      {error ? <p role="alert">{error}</p> : null}
    </section>
  );
}

export function LessonFeedbackPanel({ issueId, initialFeedback }) {
  const [values, setValues] = useState({
    difficulty: initialFeedback?.difficulty ?? "",
    relevance: initialFeedback?.relevance ?? "",
    recallConfidence: initialFeedback?.recallConfidence ?? "",
  });
  const [status, setStatus] = useState("");
  const [saving, setSaving] = useState(false);

  async function save() {
    if (!values.difficulty && !values.relevance && !values.recallConfidence) return;
    setSaving(true);
    setStatus("");
    try {
      await apiJSON(`/api/issues/${encodeURIComponent(issueId)}/feedback`, {
        method: "POST",
        body: {
          difficulty: values.difficulty || undefined,
          relevance: values.relevance || undefined,
          recallConfidence: values.recallConfidence || undefined,
        },
      });
      setStatus("Saved to your learning history.");
    } catch (requestError) {
      setStatus(
        requestError instanceof Error
          ? requestError.message
          : "Your reflection could not be saved.",
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="reader-feedback">
      <p className="atelier-eyebrow">Shape what comes next</p>
      <h2>How did this lesson fit you?</h2>
      <p>These signals are private and record how the lesson fit you.</p>
      <FeedbackChoice
        label="Difficulty"
        value={values.difficulty}
        options={[
          ["too_basic", "Too basic"],
          ["right", "About right"],
          ["too_advanced", "Too advanced"],
        ]}
        onChange={(difficulty) => setValues((current) => ({ ...current, difficulty }))}
      />
      <FeedbackChoice
        label="Relevance"
        value={values.relevance}
        options={[
          ["not_relevant", "Not relevant"],
          ["somewhat_relevant", "Somewhat"],
          ["very_relevant", "Very relevant"],
        ]}
        onChange={(relevance) => setValues((current) => ({ ...current, relevance }))}
      />
      <FeedbackChoice
        label="Recall confidence"
        value={values.recallConfidence}
        options={[
          ["low", "Needs work"],
          ["medium", "Partial"],
          ["high", "Solid"],
        ]}
        onChange={(recallConfidence) =>
          setValues((current) => ({ ...current, recallConfidence }))}
      />
      <button
        className="atelier-primary"
        type="button"
        disabled={
          saving ||
          (!values.difficulty && !values.relevance && !values.recallConfidence)
        }
        onClick={save}
      >
        {saving ? "Saving…" : "Save reflection"}
      </button>
      {status ? <span role="status">{status}</span> : null}
    </section>
  );
}

function FeedbackChoice({ label, value, options, onChange }) {
  return (
    <fieldset>
      <legend>{label}</legend>
      <div>
        {options.map(([optionValue, optionLabel]) => (
          <button
            className={value === optionValue ? "selected" : ""}
            type="button"
            aria-pressed={value === optionValue}
            onClick={() => onChange(optionValue)}
            key={optionValue}
          >
            {optionLabel}
          </button>
        ))}
      </div>
    </fieldset>
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
        {questions.map((question, index) => {
          const item = typeof question === "string"
            ? { id: `question-${index + 1}`, prompt: question }
            : question;
          return (
          <article key={item.id ?? item.prompt}>
            <span>{String(index + 1).padStart(2, "0")}</span>
            <p>{item.prompt}</p>
            <button
              type="button"
              aria-expanded={Boolean(open[index])}
              onClick={() => setOpen((current) => ({ ...current, [index]: !current[index] }))}
            >
              {open[index] ? "Hide reflection" : "I’ve thought it through"}
            </button>
            {open[index] ? (
              <small>
                {item.answerRubric || item.correctiveExplanation ||
                  "Return to the mechanism and evidence above. If your explanation names both the cause and its limits, you have the useful shape of the idea."}
              </small>
            ) : null}
          </article>
          );
        })}
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
