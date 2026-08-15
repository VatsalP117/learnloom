import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  Clock3,
  Globe,
  Plus,
  Sparkles,
  Target,
  Trash2,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import LearningShell from "./LearningShell";
import { apiJSON } from "./api";
import { firstLessonPreparation } from "./preparation";
import type {
  NewsletterCreateResponse,
  OnboardingDraftResponse,
  SourcePortfolioPreviewResponse,
  SourceValidationResponse,
} from "./types";
import {
  buildNewsletterPayload,
  canSubmitNewsletter,
  usableSources,
} from "./newsletterForm";
import { streamTemplates, type StreamTemplate } from "./streamTemplates";

const defaultSource = () => ({ name: "", url: "", limit: 8 });
const topicIdeas = [
  "How AI systems learn and fail",
  "Production RAG evaluation",
  "The economics of clean energy",
];
const steps = [
  { number: 1, label: "Learning intent" },
  { number: 2, label: "Sources" },
  { number: 3, label: "Preview & begin" },
];

interface SourceValidation {
  status: "ready" | "unavailable";
  itemCount: number;
  message?: string;
}

export default function NewsletterCreate({ sourceDiscovery = false }) {
  const [step, setStep] = useState(1);
  const [sourceMode, setSourceMode] = useState(
    sourceDiscovery ? "discovered" : "provided",
  );
  const [sources, setSources] = useState(
    sourceDiscovery ? [] : [defaultSource()],
  );
  const [busy, setBusy] = useState(false);
  const [validatingSources, setValidatingSources] = useState(false);
  const [sourceValidation, setSourceValidation] = useState<SourceValidation[]>([]);
  const [portfolioPreview, setPortfolioPreview] = useState<SourcePortfolioPreviewResponse | null>(null);
  const [previewError, setPreviewError] = useState("");
  const [reviewBeforeLesson, setReviewBeforeLesson] = useState(false);
  const [showSpecificSources, setShowSpecificSources] = useState(!sourceDiscovery);
  const [selectedTemplate, setSelectedTemplate] = useState<StreamTemplate | null>(null);
  const [error, setError] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [draftReady, setDraftReady] = useState(false);
  const [draftStatus, setDraftStatus] = useState("");
  const restoredDraft = useRef(false);
  const onboardingDraftID = useRef<string>(crypto.randomUUID());
  const onboardingDraftRevision = useRef(0);
  const draftSaveQueue = useRef<Promise<void>>(Promise.resolve());
  const draftCompleted = useRef(false);

  const [topic, setTopic] = useState("");
  const [name, setName] = useState("");
  const [learnerLevel, setLearnerLevel] = useState("intermediate");
  const [learnerGoal, setLearnerGoal] = useState("");
  const [lessonMinutes, setLessonMinutes] = useState(12);
  const [scheduleTime, setScheduleTime] = useState("08:00");
  const [timeZone, setTimeZone] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone);
  const [active, setActive] = useState(true);
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [aiExplorationEnabled, setAIExplorationEnabled] = useState(false);
  const siteVisible = false;

  const validSources = useMemo(() => usableSources(sources), [sources]);
  const sourceReady = canSubmitNewsletter({ topic, sourceMode, sources });
  const stepReady = step === 1 ? topic.trim().length > 0 : step === 2 ? sourceReady : true;
  const showSources = sourceMode === "provided" || sourceMode === "hybrid";

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "auto" });
  }, [step]);

  useEffect(() => {
    const controller = new AbortController();
    let activeRequest = true;
    apiJSON<OnboardingDraftResponse>("/api/onboarding/draft", { signal: controller.signal })
      .then(async ({ draft }) => {
        if (!draft) {
          if (activeRequest) setDraftReady(true);
          return;
        }
        restoredDraft.current = true;
        onboardingDraftID.current = draft.id;
        onboardingDraftRevision.current = draft.revision;
        const payload = draft.payload;
        const restoredMode = payload.sourceMode ?? (sourceDiscovery ? "discovered" : "provided");
        setName(payload.name ?? "");
        setTopic(payload.topic ?? "");
        setLearnerLevel(payload.learnerLevel ?? "intermediate");
        setLearnerGoal(payload.learnerGoal ?? "");
        setLessonMinutes(payload.lessonMinutes ?? 12);
        setScheduleTime(payload.scheduleTime ?? "08:00");
        setTimeZone(payload.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone);
        setActive(payload.active ?? true);
        setEmailEnabled(payload.emailEnabled ?? false);
        setAIExplorationEnabled(payload.aiExplorationEnabled ?? false);
        setSourceMode(restoredMode);
        setReviewBeforeLesson(payload.sourceReviewMode === "review");
        setShowSpecificSources(restoredMode !== "discovered");
        const restoredTemplate = streamTemplates.find((template) =>
          template.id === payload.templateId && template.version === payload.templateVersion,
        );
        setSelectedTemplate(restoredTemplate ?? null);
        setSources(payload.sources?.length
          ? payload.sources.map((source) => ({ ...source, limit: source.limit ?? 8 }))
          : restoredMode === "discovered" ? [] : [defaultSource()]);
        const restoredStep = Math.max(1, Math.min(3, draft.step));
        setStep(restoredStep);
        setDraftStatus(`Restored setup saved ${formatDraftTime(draft.updatedAt)}.`);
        if (restoredStep === 3 && sourceDiscovery && restoredMode !== "provided" && payload.topic) {
          try {
            const preview = await apiJSON<SourcePortfolioPreviewResponse>(
              "/api/source-portfolio/preview",
              {
                method: "POST",
                body: {
                  topic: payload.topic,
                  learnerGoal: payload.learnerGoal ?? "",
                  learnerLevel: payload.learnerLevel ?? "intermediate",
                  onboardingDraftId: draft.id,
                },
                signal: controller.signal,
              },
            );
            setPortfolioPreview(preview);
          } catch (requestError) {
            if (requestError.name !== "AbortError") setPreviewError(requestError.message);
          }
        }
        if (activeRequest) setDraftReady(true);
      })
      .catch((requestError) => {
        if (requestError.name !== "AbortError") {
          setDraftStatus("Setup will remain on this device until syncing is available.");
        }
      });
    return () => {
      activeRequest = false;
      controller.abort();
    };
  }, [sourceDiscovery]);

  const saveDraft = useCallback((stepValue = step) => {
    const meaningful = stepValue > 1 || Boolean(
      topic.trim() || learnerGoal.trim() || name.trim() ||
      sources.some((source) => source.url.trim()),
    );
    if (!draftReady || !meaningful || draftCompleted.current) return Promise.resolve();
    const body = {
      draftId: onboardingDraftID.current,
      expectedRevision: onboardingDraftRevision.current,
      step: stepValue,
      payload: {
        name,
        topic,
        learnerLevel,
        learnerGoal,
        lessonMinutes,
        scheduleTime,
        timeZone,
        active,
        emailEnabled,
        aiExplorationEnabled,
        sourceMode,
        sourceReviewMode: reviewBeforeLesson ? "review" : "auto",
        sources,
        templateId: selectedTemplate?.id,
        templateVersion: selectedTemplate?.version,
      },
    };
    const queued = draftSaveQueue.current
      .catch(() => {})
      .then(async () => {
        if (draftCompleted.current) return;
        body.expectedRevision = onboardingDraftRevision.current;
        const result = await apiJSON<OnboardingDraftResponse>("/api/onboarding/draft", {
          method: "PUT",
          body,
        });
        if (result.draft) onboardingDraftRevision.current = result.draft.revision;
        setDraftStatus("Setup saved.");
      })
      .catch((requestError) => {
        setDraftStatus(requestError?.code === "conflict"
          ? "This setup changed elsewhere. Reload to use the latest version."
          : "Couldn’t sync setup yet. We’ll retry after your next change.");
        throw requestError;
      });
    const settled = queued.catch(() => {});
    draftSaveQueue.current = settled;
    return settled;
  }, [
    active,
    aiExplorationEnabled,
    draftReady,
    emailEnabled,
    learnerGoal,
    learnerLevel,
    lessonMinutes,
    name,
    reviewBeforeLesson,
    scheduleTime,
    selectedTemplate,
    sourceMode,
    sources,
    step,
    timeZone,
    topic,
  ]);

  useEffect(() => {
    if (!draftReady) return undefined;
    if (restoredDraft.current) {
      restoredDraft.current = false;
      return undefined;
    }
    const timeout = window.setTimeout(() => {
      void saveDraft();
    }, 600);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [draftReady, saveDraft]);

  function addSource() {
    setSourceValidation([]);
    setSources((current) => [...current, defaultSource()]);
  }

  function removeSource(index) {
    setSourceValidation([]);
    setSources((current) => current.filter((_, position) => position !== index));
  }

  function updateSource(index, field, value) {
    setSourceValidation([]);
    setSources((current) =>
      current.map((source, position) =>
        position === index ? { ...source, [field]: value } : source,
      ),
    );
  }

  function handleModeChange(mode) {
    setSourceValidation([]);
    setPortfolioPreview(null);
    setPreviewError("");
    setSourceMode(mode);
    if (mode !== "discovered") setShowSpecificSources(true);
    if (mode === "discovered") {
      setSources([]);
    } else if (sources.length === 0) {
      setSources([defaultSource()]);
    }
  }

  function applyTemplate(template) {
    setSelectedTemplate(template);
    setName(template.name);
    setTopic(template.topic);
    setLearnerGoal(template.learnerGoal);
    setLearnerLevel(template.learnerLevel);
    setLessonMinutes(template.lessonMinutes);
    setSourceMode("provided");
    setShowSpecificSources(true);
    setSources(template.sources.map((source) => ({ ...source })));
    setSourceValidation([]);
    setError("");
  }

  async function validateProvidedSources() {
    if (!showSources) return true;
    setValidatingSources(true);
    setError("");
    try {
      const result = await apiJSON<SourceValidationResponse>("/api/sources/validate", {
        method: "POST",
        body: { sources: validSources, onboardingDraftId: onboardingDraftID.current },
      });
      const validations = result.sources ?? [];
      setSourceValidation(validations);
      if (validations.some((validation) => validation.status !== "ready")) {
        setError("One or more sources could not be read. Fix or replace them before continuing.");
        return false;
      }
      return true;
    } catch (requestError) {
      setError(requestError.message);
      return false;
    } finally {
      setValidatingSources(false);
    }
  }

  async function continueSetup() {
    if (!stepReady) return;
    await saveDraft(step === 1 ? 2 : step);
    if (step === 2 && !(await validateProvidedSources())) return;
    if (step === 2 && sourceDiscovery && sourceMode !== "provided") {
      setValidatingSources(true);
      setPreviewError("");
      try {
        const preview = await apiJSON<SourcePortfolioPreviewResponse>(
          "/api/source-portfolio/preview",
          {
            method: "POST",
            body: {
              topic: topic.trim(),
              learnerGoal: learnerGoal.trim(),
              learnerLevel,
              onboardingDraftId: onboardingDraftID.current,
            },
          },
        );
        setPortfolioPreview(preview);
      } catch (requestError) {
        setPortfolioPreview(null);
        setPreviewError(requestError.message);
      } finally {
        setValidatingSources(false);
      }
    }
    if (step === 2) await saveDraft(3);
    setError("");
    setStep((current) => Math.min(3, current + 1));
  }

  async function submit(event) {
    event.preventDefault();
    if (step < 3) {
      await continueSetup();
      return;
    }

    setBusy(true);
    setError("");
    try {
      draftCompleted.current = true;
      await draftSaveQueue.current;
      const body = buildNewsletterPayload({
        name,
        topic,
        learnerLevel,
        learnerGoal,
        lessonMinutes,
        scheduleTime,
        timeZone,
        active,
        emailEnabled,
        aiExplorationEnabled,
        siteVisible,
        sourceMode,
        sourceReviewMode: reviewBeforeLesson ? "review" : "auto",
        sources: validSources,
        templateId: selectedTemplate?.id,
        templateVersion: selectedTemplate?.version,
        onboardingDraftId: onboardingDraftID.current,
        onboardingDraftRevision: onboardingDraftRevision.current,
      });
      const result = await apiJSON<NewsletterCreateResponse>(
        "/api/newsletters",
        { method: "POST", body },
      );
      window.location.assign(
        `/welcome/${encodeURIComponent(result.newsletter.id)}`,
      );
    } catch (requestError) {
      draftCompleted.current = false;
      setError(requestError.message);
      setBusy(false);
    }
  }

  async function discardSetup() {
    setBusy(true);
    draftCompleted.current = true;
    try {
      await draftSaveQueue.current;
      const params = new URLSearchParams({
        reason: "abandoned",
        draftId: onboardingDraftID.current,
        expectedRevision: String(onboardingDraftRevision.current),
      });
      await apiJSON(
        `/api/onboarding/draft?${params.toString()}`,
        { method: "DELETE" },
      );
    } catch {
      // Leaving remains available even if the best-effort discard cannot sync.
    }
    window.location.assign("/streams");
  }

  return (
    <LearningShell active="streams" redesigned>
      <section className="atelier-page create-page">
        <div className="create-inner">
          <a className="atelier-back" href="/streams"><ArrowLeft size={14} /> Back to your streams</a>
          <section className="create-heading">
            <p className="atelier-eyebrow">Create a learning stream</p>
            <h1>{step === 1 ? "What should become clearer?" : step === 2 ? "Where should we learn from?" : "Preview your learning path."}</h1>
            <p>
              {step === 1
                ? "Start with a subject or question. You can be broad; Learnloom will build continuity over time."
                : step === 2
                  ? "Choose the information environment Learnloom should curate for you."
                  : "See how Learnloom will build the evidence base, then choose how much control you want."}
            </p>
          </section>

          <ol className="setup-steps" aria-label="Learning stream setup progress">
            {steps.map((item) => (
              <li className={`${step === item.number ? "current" : ""} ${step > item.number ? "complete" : ""}`} key={item.number}>
                <button type="button" disabled={item.number > step} onClick={() => setStep(item.number)}>
                  <span>{step > item.number ? <Check size={14} /> : item.number}</span>
                  <strong>{item.label}</strong>
                </button>
              </li>
            ))}
          </ol>

          {draftStatus ? <p className="onboarding-draft-status" role="status"><Check size={13} />{draftStatus}</p> : null}

          {error ? <div className="create-error" role="alert">{error}</div> : null}
          <form className="newsletter-form setup-form" onSubmit={submit}>
            {step === 1 ? (
              <fieldset className="setup-panel">
                <legend className="sr-only">Learning intent</legend>
                <div className="template-picker">
                  <div>
                    <span className="atelier-eyebrow">Start with a focused path</span>
                    <p>Choose a ready source environment, then make it yours.</p>
                  </div>
                  <div className="template-grid">
                    {streamTemplates.map((template) => (
                      <button type="button" key={template.id} onClick={() => applyTemplate(template)}>
                        <strong>{template.name}</strong>
                        <span>{template.outcome}</span>
                        <small>{template.sources.length} curated starting sources · {template.lessonMinutes} min</small>
                      </button>
                    ))}
                  </div>
                  {selectedTemplate ? (
                    <article className="template-sample">
                      <span className="atelier-eyebrow">A sample Dossier direction</span>
                      <strong>{selectedTemplate.sample.objective}</strong>
                      <div>
                        {selectedTemplate.sample.concepts.map((concept) => (
                          <span key={concept}>{concept}</span>
                        ))}
                      </div>
                      <p><b>Retrieval:</b> {selectedTemplate.sample.retrievalPrompt}</p>
                      <small>The actual lesson will use current items available when your stream runs.</small>
                    </article>
                  ) : null}
                </div>
                <label className="hero-field">
                  <span>Subject or question</span>
                  <textarea
                    name="topic"
                    required
                    maxLength={400}
                    rows={4}
                    autoFocus
                    placeholder="What do you want to understand over time?"
                    value={topic}
                    onChange={(event) => setTopic(event.target.value)}
                  />
                  <small>{topic.length}/400</small>
                </label>
                <div className="topic-ideas" aria-label="Topic examples">
                  <span>Try an example</span>
                  {topicIdeas.map((idea) => (
                    <button type="button" key={idea} onClick={() => setTopic(idea)}>{idea}</button>
                  ))}
                </div>
                <div className="intent-grid">
                  <label>
                    <span>Your current level</span>
                    <select name="learnerLevel" value={learnerLevel} onChange={(event) => setLearnerLevel(event.target.value)}>
                      <option value="beginner">Beginner — build the foundations</option>
                      <option value="intermediate">Intermediate — connect the pieces</option>
                      <option value="advanced">Advanced — challenge my model</option>
                    </select>
                  </label>
                  <label>
                    <span>What would progress feel like? <em>Optional</em></span>
                    <textarea
                      name="learnerGoal"
                      maxLength={500}
                      rows={3}
                      placeholder="e.g. I want to explain the trade-offs clearly and make better decisions."
                      value={learnerGoal}
                      onChange={(event) => setLearnerGoal(event.target.value)}
                    />
                  </label>
                </div>
              </fieldset>
            ) : null}

            {step === 2 ? (
              <fieldset className="setup-panel">
                <legend className="sr-only">Source policy</legend>
                <div className={`mode-options mode-options-grid ${busy ? "busy" : ""}`}>
                  {sourceDiscovery ? (
                    <label className={`mode-card ${sourceMode === "discovered" ? "selected" : ""}`}>
                      <input type="radio" name="sourceModeRadio" value="discovered" checked={sourceMode === "discovered"} onChange={() => handleModeChange("discovered")} disabled={busy} />
                      <span className="mode-icon"><Sparkles size={20} /></span>
                      <div className="mode-body"><strong>Find strong sources for me</strong><small>Learnloom discovers and validates sources around your learning intent.</small></div>
                      <span className="mode-check"><Check size={14} /></span>
                    </label>
                  ) : null}
                  {!sourceDiscovery || showSpecificSources ? (
                    <label className={`mode-card ${sourceMode === "provided" ? "selected" : ""}`}>
                      <input type="radio" name="sourceModeRadio" value="provided" checked={sourceMode === "provided"} onChange={() => handleModeChange("provided")} disabled={busy} />
                      <span className="mode-icon"><BookOpen size={20} /></span>
                      <div className="mode-body"><strong>Use sources I trust</strong><small>Add publications, feeds, organizations, or pages you already value.</small></div>
                      <span className="mode-check"><Check size={14} /></span>
                    </label>
                  ) : null}
                  {sourceDiscovery && showSpecificSources ? (
                    <label className={`mode-card ${sourceMode === "hybrid" ? "selected" : ""}`}>
                      <input type="radio" name="sourceModeRadio" value="hybrid" checked={sourceMode === "hybrid"} onChange={() => handleModeChange("hybrid")} disabled={busy} />
                      <span className="mode-icon"><Globe size={20} /></span>
                      <div className="mode-body"><strong>Start with mine, fill the gaps</strong><small>Your sources stay central; Learnloom adds evidence when coverage is thin.</small></div>
                      <span className="mode-check"><Check size={14} /></span>
                    </label>
                  ) : null}
                </div>

                {sourceDiscovery ? (
                  <button
                    className="specific-sources-toggle"
                    type="button"
                    onClick={() => {
                      if (showSpecificSources) handleModeChange("discovered");
                      setShowSpecificSources((current) => !current);
                    }}
                  >{showSpecificSources ? "Let Learnloom choose the evidence base" : "Use specific sources instead"}</button>
                ) : null}

                {!sourceDiscovery ? (
                  <div className="source-guidance"><BookOpen size={18} /><p><strong>You’re in control of the source list.</strong><span>Add at least one feed, publication, research organization, or article page. You can add and remove sources later.</span></p></div>
                ) : null}

                {showSources ? (
                  <div className="source-section">
                    <div className="source-section-heading">
                      <div><strong>Your trusted sources</strong><span>One is enough to begin.</span></div>
                      <button className="add-source" type="button" disabled={busy || sources.length >= 12} onClick={addSource}><Plus size={16} />Add source</button>
                    </div>
                    <div className="source-editor">
                      {sources.map((source, index) => (
                        <div key={index}>
                          <div className="source-row guided-source-row">
                            <span className="source-number">{index + 1}</span>
                            <label><span>Source URL</span><input aria-label={`Source ${index + 1} URL`} required type="url" placeholder="https://publication.com or feed.xml" value={source.url} onChange={(event) => updateSource(index, "url", event.target.value)} /></label>
                            <label><span>Label <em>Optional</em></span><input aria-label={`Source ${index + 1} name`} maxLength={120} placeholder="Publication name" value={source.name} onChange={(event) => updateSource(index, "name", event.target.value)} /></label>
                            <button className="remove-source" type="button" aria-label={`Remove source ${index + 1}`} disabled={busy || sources.length === 1} onClick={() => removeSource(index)}><Trash2 size={16} /></button>
                          </div>
                          {sourceValidation[index] ? (
                            <div className={`source-validation ${sourceValidation[index].status}`}>
                              {sourceValidation[index].status === "ready"
                                ? <><Check size={14} /><span>Ready · {sourceValidation[index].itemCount} recent items found</span></>
                                : <><span aria-hidden="true">!</span><span>{sourceValidation[index].message}</span></>}
                            </div>
                          ) : null}
                        </div>
                      ))}
                    </div>
                    <p className="source-validation-note">
                      Sources are checked safely before creation. Learnloom reads public feed metadata only.
                    </p>
                  </div>
                ) : null}
              </fieldset>
            ) : null}

            {step === 3 ? (
              <div className="review-layout">
                <fieldset className="setup-panel rhythm-panel">
                  <legend className="sr-only">Source portfolio and learning rhythm</legend>
                  <section className="portfolio-preview" aria-labelledby="portfolio-preview-heading">
                    {portfolioPreview?.researchPlan ? (
                      <section className="research-plan-preview" aria-labelledby="research-plan-heading">
                        <span className="atelier-eyebrow">Initial learning arc</span>
                        <h2 id="research-plan-heading">{portfolioPreview.researchPlan.likelyFirstLesson}</h2>
                        <p>{portfolioPreview.researchPlan.objective}</p>
                        <ol>
                          {portfolioPreview.researchPlan.initialConcepts.map((concept, index) => (
                            <li key={concept}><span>{index + 1}</span>{concept}</li>
                          ))}
                        </ol>
                        <small>
                          Likely first lesson · {portfolioPreview.researchPlan.minimumPreparationMinutes}–{portfolioPreview.researchPlan.maximumPreparationMinutes} min to prepare. The path will refine as sources are resolved.
                        </small>
                      </section>
                    ) : null}
                    <div className="portfolio-preview-heading">
                      <div>
                        <span className="atelier-eyebrow">Provisional evidence portfolio</span>
                        <h2 id="portfolio-preview-heading">What Learnloom will investigate first</h2>
                      </div>
                      {sourceMode !== "provided" ? <span>{portfolioPreview?.items.length ?? 0} candidates</span> : null}
                    </div>
                    {sourceMode === "provided" ? (
                      <div className="provided-preview-list">
                        {validSources.map((source) => (
                          <article key={source.url}>
                            <BookOpen size={16} />
                            <div><strong>{source.name || new URL(source.url).hostname}</strong><small>Your trusted source · validated before creation</small></div>
                          </article>
                        ))}
                      </div>
                    ) : portfolioPreview?.items.length ? (
                      <div className="portfolio-preview-list">
                        {portfolioPreview.items.map((source) => (
                          <article key={`${source.role}:${source.url}`}>
                            <span>{sourceRoleLabel(source.role)}</span>
                            <div><strong>{source.title || source.registrableDomain}</strong><small>{source.selectionReason}</small></div>
                            <a href={source.url} target="_blank" rel="noreferrer">Inspect</a>
                          </article>
                        ))}
                      </div>
                    ) : (
                      <div className="portfolio-preview-unavailable">
                        <strong>{previewError ? "Preview temporarily unavailable" : "No candidates surfaced yet"}</strong>
                        <p>{previewError || "Learnloom will run a deeper search after you begin. You can still review the final portfolio before any lesson is written."}</p>
                        <button type="button" onClick={() => { setStep(2); setPreviewError(""); }}>Adjust sources</button>
                      </div>
                    )}
                    {portfolioPreview?.missingRoles.length ? (
                      <p className="portfolio-gap-note">
                        Coverage is still thin for {portfolioPreview.missingRoles.map(sourceRoleLabel).join(", ")}. Learnloom will keep searching before writing.
                      </p>
                    ) : null}
                    <small>These are search candidates, not citations yet. Learnloom resolves, checks, and freezes the exact evidence used in each lesson.</small>
                  </section>

                  <label className="approval-choice">
                    <input type="checkbox" checked={reviewBeforeLesson} onChange={(event) => setReviewBeforeLesson(event.target.checked)} />
                    <span><strong>Let me approve the sources before lesson one</strong><small>Learnloom will pause after building the final portfolio. You can prefer, block, or replace sources, then approve generation.</small></span>
                  </label>

                  <div className="rhythm-intro"><span><Clock3 size={20} /></span><div><strong>A small, steady practice</strong><p>We’ll prepare one focused lesson each day. Twelve minutes is enough for a mechanism, example, limitation, and active recall.</p></div></div>
                  <div className="form-grid">
                    <label>
                      <span>Lesson length</span>
                      <select name="lessonMinutes" value={lessonMinutes} onChange={(event) => setLessonMinutes(Number(event.target.value))}>
                        <option value="8">8 min — compact lesson</option>
                        <option value="12">12 min — focused understanding</option>
                        <option value="15">15 min — fuller example</option>
                        <option value="25">25 min — extended deep dive</option>
                      </select>
                    </label>
                    <label>
                      <span>Ready each day at</span>
                      <input name="scheduleTime" type="time" value={scheduleTime} onChange={(event) => setScheduleTime(event.target.value)} />
                    </label>
                  </div>
                  <label className="delivery-choice">
                    <span className="delivery-choice-icon"><BookOpen size={18} /></span>
                    <span><strong>Keep lessons in Learnloom</strong><small>Your archive is always available here.</small></span>
                    <input type="radio" name="deliveryChoice" checked={!emailEnabled} onChange={() => setEmailEnabled(false)} />
                  </label>
                  <label className="delivery-choice">
                    <span className="delivery-choice-icon"><Sparkles size={18} /></span>
                    <span><strong>Also send them by email</strong><small>A gentle prompt when each lesson is ready.</small></span>
                    <input type="radio" name="deliveryChoice" checked={emailEnabled} onChange={() => setEmailEnabled(true)} />
                  </label>

                  <button className="more-options-toggle" type="button" onClick={() => setShowAdvanced((current) => !current)} aria-expanded={showAdvanced}>
                    {showAdvanced ? "Hide" : "Show"} optional settings
                  </button>
                  {showAdvanced ? (
                    <div className="optional-settings">
                      <label><span>Stream name <em>Optional</em></span><input name="name" maxLength={80} placeholder="We’ll generate one from your topic" value={name} onChange={(event) => setName(event.target.value)} /></label>
                      <label><span>Time zone</span><input name="timeZone" value={timeZone} onChange={(event) => setTimeZone(event.target.value)} /></label>
                      <label className="switch-row"><span><strong>Active schedule</strong><small>Prepare future lessons automatically.</small></span><input name="active" type="checkbox" checked={active} onChange={(event) => setActive(event.target.checked)} /></label>
                      <label className="switch-row"><span><strong>AI exploration</strong><small>Allow clearly marked ideas beyond sourced claims.</small></span><input name="aiExplorationEnabled" type="checkbox" checked={aiExplorationEnabled} onChange={(event) => setAIExplorationEnabled(event.target.checked)} /></label>
                    </div>
                  ) : null}
                </fieldset>

                <aside className="setup-review" aria-label="Learning stream summary">
                  <p className="atelier-eyebrow">Ready to begin</p>
                  <div className="review-icon"><Target size={22} /></div>
                  <h2>{name.trim() || topic.trim()}</h2>
                  <p>{learnerGoal.trim() || `Build a ${learnerLevel}-level understanding through connected, source-grounded lessons.`}</p>
                  <dl>
                    <div><dt>Sources</dt><dd>{sourceMode === "discovered" ? "Curated by Learnloom" : `${validSources.length} trusted source${validSources.length === 1 ? "" : "s"}`}</dd></div>
                    <div><dt>Before lesson one</dt><dd>{reviewBeforeLesson ? "Wait for my approval" : "Begin automatically"}</dd></div>
                    <div><dt>Rhythm</dt><dd>Daily at {scheduleTime}</dd></div>
                    <div><dt>Lesson</dt><dd>{lessonMinutes} minutes</dd></div>
                    <div><dt>Delivery</dt><dd>{emailEnabled ? "Learnloom + email" : "Learnloom archive"}</dd></div>
                  </dl>
                  <small>
                    Your first lesson is {firstLessonPreparation.shortLabel.toLowerCase()}.
                    You can leave safely while Learnloom prepares it.
                  </small>
                </aside>
              </div>
            ) : null}

            <div className="form-actions setup-actions">
              {step > 1 ? <button className="create-secondary" type="button" onClick={() => setStep((current) => current - 1)}><ArrowLeft size={15} />Back</button> : <button className="create-secondary" type="button" onClick={discardSetup}>Discard setup</button>}
              <span>Step {step} of 3</span>
              {step < 3 ? (
                <button className="atelier-primary" disabled={!stepReady || validatingSources} type="submit">
                  {validatingSources ? "Checking sources…" : "Continue"} <ArrowRight size={16} />
                </button>
              ) : (
                <button className="atelier-primary create-submit" disabled={busy || !sourceReady} type="submit"><Sparkles size={17} />{busy ? "Building your path…" : "Build my learning path"}</button>
              )}
            </div>
          </form>
        </div>
      </section>
    </LearningShell>
  );
}

function sourceRoleLabel(role: string) {
  return ({
    official_primary: "Official source",
    research: "Research",
    practitioner_explainer: "Practitioner",
    reporting_context: "Context",
    counterweight: "Counterpoint",
  })[role] ?? "Supporting source";
}

function formatDraftTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "recently";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}
