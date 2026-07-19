import { Plus, Sparkles, Trash2 } from "lucide-react";
import { useState } from "react";
import { ErrorState, Footer, Topbar } from "./App.jsx";
import { apiJSON } from "./api.js";

const defaultSource = () => ({ name: "", url: "", limit: 8 });

export default function NewsletterCreate() {
  const [sources, setSources] = useState([defaultSource()]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  function updateSource(index, field, value) {
    setSources((current) =>
      current.map((source, position) =>
        position === index ? { ...source, [field]: value } : source,
      ),
    );
  }

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      const body = await apiJSON("/api/newsletters", {
        method: "POST",
        body: {
          name: data.get("name"),
          topic: data.get("topic"),
          learnerLevel: data.get("learnerLevel"),
          learnerGoal: data.get("learnerGoal"),
          lessonMinutes: Number(data.get("lessonMinutes")),
          scheduleTime: data.get("scheduleTime"),
          timeZone: data.get("timeZone"),
          active: data.get("active") === "on",
          emailEnabled: data.get("emailEnabled") === "on",
          aiExplorationEnabled: data.get("aiExplorationEnabled") === "on",
          siteVisible: data.get("siteVisible") === "on",
          sources: sources.map((source) => ({
            ...source,
            limit: Number(source.limit),
          })),
        },
      });
      window.location.assign(
        `/newsletters/${encodeURIComponent(body.newsletter.id)}?created=1`,
      );
    } catch (requestError) {
      setError(requestError.message);
      setBusy(false);
    }
  }

  return (
    <div className="app">
      <Topbar onMenu={() => {}} />
      <main className="content create-content">
        <div className="content-inner create-inner">
          <a className="back-link" href="/">Newsletters <span>/</span></a>
          <section className="create-heading">
            <p className="overline">New learning stream</p>
            <h1>Create a Knowledge Dossier</h1>
            <p>Define the question, the learner, and the sources that deserve attention.</p>
          </section>
          {error ? <ErrorState message={error} /> : null}
          <form className="newsletter-form" onSubmit={submit}>
            <fieldset>
              <legend>Learning brief</legend>
              <label>
                <span>Name</span>
                <input name="name" required maxLength="120" placeholder="Distributed systems field notes" />
              </label>
              <label>
                <span>Topic</span>
                <textarea name="topic" required maxLength="400" rows="3" placeholder="The systems question this Dossier should investigate" />
              </label>
              <div className="form-grid">
                <label>
                  <span>Learner level</span>
                  <select name="learnerLevel" defaultValue="intermediate">
                    <option value="beginner">Beginner</option>
                    <option value="intermediate">Intermediate</option>
                    <option value="advanced">Advanced</option>
                  </select>
                </label>
                <label>
                  <span>Lesson length</span>
                  <input name="lessonMinutes" type="number" min="5" max="90" defaultValue="20" required />
                </label>
              </div>
              <label>
                <span>Learning goal</span>
                <textarea name="learnerGoal" required maxLength="500" rows="3" placeholder="What should become newly understandable or actionable?" />
              </label>
            </fieldset>

            <fieldset>
              <legend>Trusted sources</legend>
              <p className="field-help">Use direct article, publication, RSS, or Atom URLs. Learnloom validates every fetch.</p>
              <div className="source-editor">
                {sources.map((source, index) => (
                  <div className="source-row" key={index}>
                    <input aria-label={`Source ${index + 1} name`} required maxLength="120" placeholder="Source name" value={source.name} onChange={(event) => updateSource(index, "name", event.target.value)} />
                    <input aria-label={`Source ${index + 1} URL`} required type="url" placeholder="https://example.com/feed.xml" value={source.url} onChange={(event) => updateSource(index, "url", event.target.value)} />
                    <input aria-label={`Source ${index + 1} item limit`} required type="number" min="1" max="50" value={source.limit} onChange={(event) => updateSource(index, "limit", event.target.value)} />
                    <button type="button" aria-label={`Remove source ${index + 1}`} disabled={sources.length === 1} onClick={() => setSources((current) => current.filter((_, position) => position !== index))}><Trash2 size={16} /></button>
                  </div>
                ))}
              </div>
              <button className="add-source" type="button" disabled={sources.length >= 12} onClick={() => setSources((current) => [...current, defaultSource()])}><Plus size={16} />Add source</button>
            </fieldset>

            <fieldset>
              <legend>Schedule & publishing</legend>
              <div className="form-grid">
                <label>
                  <span>Daily time</span>
                  <input name="scheduleTime" type="time" defaultValue="08:00" required />
                </label>
                <label>
                  <span>Time zone</span>
                  <input name="timeZone" required defaultValue={Intl.DateTimeFormat().resolvedOptions().timeZone} />
                </label>
              </div>
              <label className="switch-row"><span><strong>Active schedule</strong><small>Generate future Issues automatically.</small></span><input name="active" type="checkbox" defaultChecked /></label>
              <label className="switch-row"><span><strong>Email delivery</strong><small>Send to your verified primary email.</small></span><input name="emailEnabled" type="checkbox" /></label>
              <label className="switch-row"><span><strong>AI exploration</strong><small>Allow clearly marked model-generated extensions.</small></span><input name="aiExplorationEnabled" type="checkbox" /></label>
              <label className="switch-row"><span><strong>Show on personal site</strong><small>New Issues remain individually publishable.</small></span><input name="siteVisible" type="checkbox" /></label>
            </fieldset>

            <div className="form-actions">
              <a href="/">Cancel</a>
              <button className="primary-button" disabled={busy}>
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
