import { Plus, BookOpen, Globe, Sparkles, Trash2, ChevronDown, ChevronUp } from "lucide-react";
import { useState, useMemo } from "react";
import { ErrorState, Footer, Topbar } from "./App.jsx";
import { apiJSON } from "./api.js";
import {
  buildNewsletterPayload,
  canSubmitNewsletter,
  usableSources,
} from "./newsletterForm.js";

const defaultSource = () => ({ name: "", url: "", limit: 8 });

export default function NewsletterCreate({ sourceDiscovery = false }) {
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

  const validSources = useMemo(
    () => usableSources(sources),
    [sources],
  );

  function canSubmit() {
    return canSubmitNewsletter({ topic, sourceMode, sources });
  }

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
    } else if (mode === "provided" || mode === "hybrid") {
      if (sources.length === 0) {
        setSources([defaultSource()]);
      }
    }
  }

  async function submit(event) {
    event.preventDefault();
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

  const showSources = sourceMode === "provided" || sourceMode === "hybrid";

  return (
    <div className="app">
      <Topbar onMenu={() => {}} />
      <main className="content create-content">
        <div className="content-inner create-inner">
          <a className="back-link" href="/">Newsletters <span>/</span></a>
          <section className="create-heading">
            <p className="overline">New learning stream</p>
            <h1>What do you want to learn?</h1>
            <p>Enter a topic and choose how sources should be handled. Advanced settings can be tuned after creation.</p>
          </section>
          {error ? <ErrorState message={error} /> : null}
          <form className="newsletter-form" onSubmit={submit}>
            <fieldset>
              <legend>Learning topic</legend>
              <label>
                <span>Topic</span>
                <textarea
                  name="topic"
                  required
                  maxLength="400"
                  rows="3"
                  placeholder="e.g. LLM inferencing, distributed systems, renaissance art history"
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                />
              </label>
            </fieldset>

            <fieldset>
              <legend>Source policy</legend>
              <p className="field-help">Choose how Learnloom finds and uses sources for your daily Knowledge Dossiers.</p>
              <div className={`mode-options ${busy ? "busy" : ""}`}>
                <label className={`mode-card ${sourceMode === "discovered" ? "selected" : ""} ${!sourceDiscovery ? "disabled" : ""}`}>
                  <input type="radio" name="sourceModeRadio" value="discovered" checked={sourceMode === "discovered"} onChange={() => handleModeChange("discovered")} disabled={busy || !sourceDiscovery} />
                  <span className="mode-icon"><Sparkles size={20} /></span>
                  <div className="mode-body">
                    <strong>Find sources for me</strong>
                    <small>{sourceDiscovery ? "Learnloom discovers, validates, and selects relevant sources automatically." : "Automatic discovery is not enabled on this deployment."}</small>
                  </div>
                </label>
                <label className={`mode-card ${sourceMode === "provided" ? "selected" : ""}`}>
                  <input type="radio" name="sourceModeRadio" value="provided" checked={sourceMode === "provided"} onChange={() => handleModeChange("provided")} disabled={busy} />
                  <span className="mode-icon"><BookOpen size={20} /></span>
                  <div className="mode-body">
                    <strong>I'll provide them</strong>
                    <small>Supply specific RSS feeds, article URLs, or publications. Only your sources are used.</small>
                  </div>
                </label>
                <label className={`mode-card ${sourceMode === "hybrid" ? "selected" : ""} ${!sourceDiscovery ? "disabled" : ""}`}>
                  <input type="radio" name="sourceModeRadio" value="hybrid" checked={sourceMode === "hybrid"} onChange={() => handleModeChange("hybrid")} disabled={busy || !sourceDiscovery} />
                  <span className="mode-icon"><Globe size={20} /></span>
                  <div className="mode-body">
                    <strong>Use mine and find more when helpful</strong>
                    <small>Your sources are prioritized. Learnloom adds discovered sources to fill coverage gaps.</small>
                  </div>
                </label>
              </div>
            </fieldset>

            {showSources ? (
              <fieldset>
                <legend>Provided sources</legend>
                <p className="field-help">Add RSS feeds, Atom feeds, article pages, or publication URLs. Learnloom validates every fetch.</p>
                <div className="source-editor">
                  {sources.map((source, index) => (
                    <div className="source-row" key={index}>
                      <input aria-label={`Source ${index + 1} name`} maxLength="120" placeholder="optional label" value={source.name} onChange={(event) => updateSource(index, "name", event.target.value)} />
                      <input aria-label={`Source ${index + 1} URL`} required type="url" placeholder="https://example.com/feed.xml" value={source.url} onChange={(event) => updateSource(index, "url", event.target.value)} />
                      <input aria-label={`Source ${index + 1} item limit`} type="number" min="1" max="50" value={source.limit} onChange={(event) => updateSource(index, "limit", event.target.value)} />
                      <button type="button" aria-label={`Remove source ${index + 1}`} disabled={busy} onClick={() => removeSource(index)}><Trash2 size={16} /></button>
                    </div>
                  ))}
                </div>
                <button className="add-source" type="button" disabled={busy || sources.length >= 12} onClick={addSource}><Plus size={16} />Add source</button>
              </fieldset>
            ) : null}

            <fieldset className="advanced-fieldset">
              <legend className="sr-only">Learning preferences</legend>
              <button
                type="button"
                className="advanced-toggle"
                onClick={() => setShowAdvanced((v) => !v)}
                aria-expanded={showAdvanced}
              >
                <span className="advanced-toggle-label">Learning preferences</span>
                {showAdvanced ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
              </button>
              <div className={`advanced-body ${showAdvanced ? "visible" : ""}`} hidden={!showAdvanced}>
                <label>
                  <span>Stream name</span>
                  <input name="name" maxLength="120" placeholder="Generated from topic if left empty" value={name} onChange={(e) => setName(e.target.value)} />
                </label>
                <div className="form-grid">
                  <label>
                    <span>Learner level</span>
                    <select name="learnerLevel" value={learnerLevel} onChange={(e) => setLearnerLevel(e.target.value)}>
                      <option value="beginner">Beginner</option>
                      <option value="intermediate">Intermediate</option>
                      <option value="advanced">Advanced</option>
                    </select>
                  </label>
                  <label>
                    <span>Lesson length (minutes)</span>
                    <input name="lessonMinutes" type="number" min="5" max="90" value={lessonMinutes} onChange={(e) => setLessonMinutes(Number(e.target.value))} />
                  </label>
                </div>
                <label>
                  <span>Learning goal</span>
                  <textarea name="learnerGoal" maxLength="500" rows="2" placeholder="What should become newly understandable?" value={learnerGoal} onChange={(e) => setLearnerGoal(e.target.value)} />
                </label>
                <div className="form-grid">
                  <label>
                    <span>Schedule time</span>
                    <input name="scheduleTime" type="time" value={scheduleTime} onChange={(e) => setScheduleTime(e.target.value)} />
                  </label>
                  <label>
                    <span>Time zone</span>
                    <input name="timeZone" value={timeZone} onChange={(e) => setTimeZone(e.target.value)} />
                  </label>
                </div>
                <label className="switch-row"><span><strong>Active schedule</strong><small>Generate future Issues automatically.</small></span><input name="active" type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} /></label>
                <label className="switch-row"><span><strong>Email delivery</strong><small>Send to your verified primary email.</small></span><input name="emailEnabled" type="checkbox" checked={emailEnabled} onChange={(e) => setEmailEnabled(e.target.checked)} /></label>
                <label className="switch-row"><span><strong>AI exploration</strong><small>Allow clearly marked model-generated extensions.</small></span><input name="aiExplorationEnabled" type="checkbox" checked={aiExplorationEnabled} onChange={(e) => setAIExplorationEnabled(e.target.checked)} /></label>
                <label className="switch-row"><span><strong>Show on personal site</strong><small>New Issues remain individually publishable.</small></span><input name="siteVisible" type="checkbox" checked={siteVisible} onChange={(e) => setSiteVisible(e.target.checked)} /></label>
              </div>
            </fieldset>

            <div className="form-actions">
              <a href="/">Cancel</a>
              <button className="primary-button" disabled={busy || !canSubmit()} type="submit">
                <Sparkles size={17} />{busy ? "Creating…" : "Create Dossier"}
              </button>
            </div>
          </form>
        </div>
        <Footer />
      </main>
    </div>
  );
}
