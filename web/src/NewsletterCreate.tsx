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
import { useEffect, useMemo, useState } from "react";
import LearningShell from "./LearningShell";
import { apiJSON } from "./api";
import {
  buildNewsletterPayload,
  canSubmitNewsletter,
  usableSources,
} from "./newsletterForm";

const defaultSource = () => ({ name: "", url: "", limit: 8 });
const topicIdeas = [
  "How AI systems learn and fail",
  "Climate adaptation in cities",
  "The economics of clean energy",
];
const steps = [
  { number: 1, label: "Learning intent" },
  { number: 2, label: "Sources" },
  { number: 3, label: "Your rhythm" },
];

export default function NewsletterCreate({ sourceDiscovery = false }) {
  const [step, setStep] = useState(1);
  const [sourceMode, setSourceMode] = useState(
    sourceDiscovery ? "discovered" : "provided",
  );
  const [sources, setSources] = useState(
    sourceDiscovery ? [] : [defaultSource()],
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);

  const [topic, setTopic] = useState("");
  const [name, setName] = useState("");
  const [learnerLevel, setLearnerLevel] = useState("intermediate");
  const [learnerGoal, setLearnerGoal] = useState("");
  const [lessonMinutes, setLessonMinutes] = useState(20);
  const [scheduleTime, setScheduleTime] = useState("08:00");
  const [timeZone, setTimeZone] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone);
  const [active, setActive] = useState(true);
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [aiExplorationEnabled, setAIExplorationEnabled] = useState(false);
  const [siteVisible, setSiteVisible] = useState(false);

  const validSources = useMemo(() => usableSources(sources), [sources]);
  const sourceReady = canSubmitNewsletter({ topic, sourceMode, sources });
  const stepReady = step === 1 ? topic.trim().length > 0 : step === 2 ? sourceReady : true;
  const showSources = sourceMode === "provided" || sourceMode === "hybrid";

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "auto" });
  }, [step]);

  function addSource() {
    setSources((current) => [...current, defaultSource()]);
  }

  function removeSource(index) {
    setSources((current) => current.filter((_, position) => position !== index));
  }

  function updateSource(index, field, value) {
    setSources((current) =>
      current.map((source, position) =>
        position === index ? { ...source, [field]: value } : source,
      ),
    );
  }

  function handleModeChange(mode) {
    setSourceMode(mode);
    if (mode === "discovered") {
      setSources([]);
    } else if (sources.length === 0) {
      setSources([defaultSource()]);
    }
  }

  function continueSetup() {
    if (stepReady) {
      setError("");
      setStep((current) => Math.min(3, current + 1));
    }
  }

  async function submit(event) {
    event.preventDefault();
    if (step < 3) {
      continueSetup();
      return;
    }

    setBusy(true);
    setError("");
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
      sources: validSources,
    });

    try {
      const result = await apiJSON("/api/newsletters", { method: "POST", body });
      window.location.assign(
        `/newsletters/${encodeURIComponent(result.newsletter.id)}?created=1`,
      );
    } catch (requestError) {
      setError(requestError.message);
      setBusy(false);
    }
  }

  return (
    <LearningShell active="streams">
      <section className="atelier-page create-page">
        <div className="create-inner">
          <a className="atelier-back" href="/streams"><ArrowLeft size={14} /> Back to your streams</a>
          <section className="create-heading">
            <p className="atelier-eyebrow">Create a learning stream</p>
            <h1>{step === 1 ? "What should become clearer?" : step === 2 ? "Where should we learn from?" : "Make it fit your life."}</h1>
            <p>
              {step === 1
                ? "Start with a subject or question. You can be broad; Learnloom will build continuity over time."
                : step === 2
                  ? "Choose the information environment Learnloom should curate for you."
                  : "Choose a pace you can keep. You can change every setting later."}
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

          {error ? <div className="create-error" role="alert">{error}</div> : null}
          <form className="newsletter-form setup-form" onSubmit={submit}>
            {step === 1 ? (
              <fieldset className="setup-panel">
                <legend className="sr-only">Learning intent</legend>
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
                  <label className={`mode-card ${sourceMode === "provided" ? "selected" : ""}`}>
                    <input type="radio" name="sourceModeRadio" value="provided" checked={sourceMode === "provided"} onChange={() => handleModeChange("provided")} disabled={busy} />
                    <span className="mode-icon"><BookOpen size={20} /></span>
                    <div className="mode-body"><strong>Use sources I trust</strong><small>Add publications, feeds, organizations, or pages you already value.</small></div>
                    <span className="mode-check"><Check size={14} /></span>
                  </label>
                  {sourceDiscovery ? (
                    <label className={`mode-card ${sourceMode === "hybrid" ? "selected" : ""}`}>
                      <input type="radio" name="sourceModeRadio" value="hybrid" checked={sourceMode === "hybrid"} onChange={() => handleModeChange("hybrid")} disabled={busy} />
                      <span className="mode-icon"><Globe size={20} /></span>
                      <div className="mode-body"><strong>Start with mine, fill the gaps</strong><small>Your sources stay central; Learnloom adds evidence when coverage is thin.</small></div>
                      <span className="mode-check"><Check size={14} /></span>
                    </label>
                  ) : null}
                </div>

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
                        <div className="source-row guided-source-row" key={index}>
                          <span className="source-number">{index + 1}</span>
                          <label><span>Source URL</span><input aria-label={`Source ${index + 1} URL`} required type="url" placeholder="https://publication.com or feed.xml" value={source.url} onChange={(event) => updateSource(index, "url", event.target.value)} /></label>
                          <label><span>Label <em>Optional</em></span><input aria-label={`Source ${index + 1} name`} maxLength={120} placeholder="Publication name" value={source.name} onChange={(event) => updateSource(index, "name", event.target.value)} /></label>
                          <button className="remove-source" type="button" aria-label={`Remove source ${index + 1}`} disabled={busy || sources.length === 1} onClick={() => removeSource(index)}><Trash2 size={16} /></button>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </fieldset>
            ) : null}

            {step === 3 ? (
              <div className="review-layout">
                <fieldset className="setup-panel rhythm-panel">
                  <legend className="sr-only">Learning rhythm</legend>
                  <div className="rhythm-intro"><span><Clock3 size={20} /></span><div><strong>A small, steady practice</strong><p>We’ll prepare one focused lesson each day. Twenty minutes is a good default for depth without overload.</p></div></div>
                  <div className="form-grid">
                    <label>
                      <span>Lesson length</span>
                      <select name="lessonMinutes" value={lessonMinutes} onChange={(event) => setLessonMinutes(Number(event.target.value))}>
                        <option value="10">10 min — quick orientation</option>
                        <option value="20">20 min — focused understanding</option>
                        <option value="30">30 min — deeper study</option>
                        <option value="45">45 min — extended lesson</option>
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
                      <label><span>Stream name <em>Optional</em></span><input name="name" maxLength={120} placeholder="We’ll generate one from your topic" value={name} onChange={(event) => setName(event.target.value)} /></label>
                      <label><span>Time zone</span><input name="timeZone" value={timeZone} onChange={(event) => setTimeZone(event.target.value)} /></label>
                      <label className="switch-row"><span><strong>Active schedule</strong><small>Prepare future lessons automatically.</small></span><input name="active" type="checkbox" checked={active} onChange={(event) => setActive(event.target.checked)} /></label>
                      <label className="switch-row"><span><strong>AI exploration</strong><small>Allow clearly marked ideas beyond sourced claims.</small></span><input name="aiExplorationEnabled" type="checkbox" checked={aiExplorationEnabled} onChange={(event) => setAIExplorationEnabled(event.target.checked)} /></label>
                      <label className="switch-row"><span><strong>Show on personal site</strong><small>Make this stream available to publish.</small></span><input name="siteVisible" type="checkbox" checked={siteVisible} onChange={(event) => setSiteVisible(event.target.checked)} /></label>
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
                    <div><dt>Rhythm</dt><dd>Daily at {scheduleTime}</dd></div>
                    <div><dt>Lesson</dt><dd>{lessonMinutes} minutes</dd></div>
                    <div><dt>Delivery</dt><dd>{emailEnabled ? "Learnloom + email" : "Learnloom archive"}</dd></div>
                  </dl>
                  <small>Nothing here is permanent. You can tune your stream after it’s created.</small>
                </aside>
              </div>
            ) : null}

            <div className="form-actions setup-actions">
              {step > 1 ? <button className="create-secondary" type="button" onClick={() => setStep((current) => current - 1)}><ArrowLeft size={15} />Back</button> : <a className="create-secondary" href="/streams">Cancel</a>}
              <span>Step {step} of 3</span>
              {step < 3 ? (
                <button className="atelier-primary" disabled={!stepReady} type="submit">Continue <ArrowRight size={16} /></button>
              ) : (
                <button className="atelier-primary create-submit" disabled={busy || !sourceReady} type="submit"><Sparkles size={17} />{busy ? "Creating your stream…" : "Create learning stream"}</button>
              )}
            </div>
          </form>
        </div>
      </section>
    </LearningShell>
  );
}
