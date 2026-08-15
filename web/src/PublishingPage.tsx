import {
  ArrowRight,
  Check,
  ExternalLink,
  Eye,
  EyeOff,
  Globe2,
  LockKeyhole,
  Save,
  Search,
  TrendingUp,
} from "lucide-react";
import { useEffect, useState } from "react";
import LearningShell, { AtelierError, AtelierLoading } from "./LearningShell";
import { apiJSON } from "./api";
import { personalSiteHost } from "./config";
import { useWorkspace } from "./useWorkspace";
import type {
  SiteMutationResponse,
  PublicGrowthAnalytics,
  PublicGrowthAnalyticsResponse,
  UsernameAvailabilityResponse,
} from "./types";

export default function PublishingPage({ site, onSiteUpdate }) {
  const workspace = useWorkspace();
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState(site?.displayName ?? "");
  const [description, setDescription] = useState(site?.description ?? "");
  const [availability, setAvailability] = useState(null);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [analytics, setAnalytics] = useState<PublicGrowthAnalytics | null>(null);
  const [analyticsPeriod, setAnalyticsPeriod] = useState(30);
  const [analyticsError, setAnalyticsError] = useState("");

  useEffect(() => {
    setDisplayName(site?.displayName ?? "");
    setDescription(site?.description ?? "");
  }, [site?.displayName, site?.description]);

  useEffect(() => {
    if (!site) return undefined;
    const controller = new AbortController();
    setAnalyticsError("");
    apiJSON<PublicGrowthAnalyticsResponse>(
      `/api/me/public-analytics?days=${analyticsPeriod}`,
      { signal: controller.signal },
    )
      .then((body) => setAnalytics(body.analytics))
      .catch((requestError) => {
        if (requestError.name !== "AbortError") {
          setAnalyticsError("Public analytics are temporarily unavailable.");
        }
      });
    return () => controller.abort();
  }, [site, analyticsPeriod]);

  useEffect(() => {
    if (site) return undefined;
    const normalized = username.trim().toLowerCase();
    if (normalized.length < 3) {
      setAvailability(null);
      return undefined;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      apiJSON<UsernameAvailabilityResponse>(
        `/api/usernames/${encodeURIComponent(normalized)}`,
        {
          signal: controller.signal,
        },
      )
        .then((body) => setAvailability(Boolean(body.available)))
        .catch(() => setAvailability(null));
    }, 300);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [site, username]);

  async function claim(event) {
    event.preventDefault();
    setBusy("claim");
    setError("");
    try {
      const body = await apiJSON<SiteMutationResponse>("/api/me/site/claim", {
        method: "POST",
        body: { username, displayName },
      });
      onSiteUpdate?.(body.site);
      setNotice("Your personal learning address is ready. It remains private until you publish it.");
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  async function saveIdentity(event) {
    event.preventDefault();
    setBusy("identity");
    setError("");
    try {
      const body = await apiJSON<SiteMutationResponse>("/api/me/site/settings", {
        method: "POST",
        body: {
          visibility: site.visibility,
          displayName,
          description,
        },
      });
      onSiteUpdate?.(body.site);
      setNotice("Public identity saved.");
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  async function toggleVisibility() {
    setBusy("visibility");
    setError("");
    const visibility = site.visibility === "public" ? "private" : "public";
    try {
      const body = await apiJSON<SiteMutationResponse>("/api/me/site/settings", {
        method: "POST",
        body: { visibility },
      });
      onSiteUpdate?.(body.site);
      setNotice(
        visibility === "public"
          ? "Your site is public. Only streams and lessons you publish can appear."
          : "Your site is now private.",
      );
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  async function toggleSearchIndexing() {
    setBusy("search-indexing");
    setError("");
    const searchIndexing = !site.searchIndexing;
    try {
      const body = await apiJSON<SiteMutationResponse>("/api/me/site/settings", {
        method: "POST",
        body: { visibility: site.visibility, searchIndexing },
      });
      onSiteUpdate?.(body.site);
      setNotice(
        searchIndexing
          ? "Search indexing is enabled. Search engines may discover eligible published pages."
          : "Search indexing is disabled. Your public links still work, but pages ask search engines not to list them.",
      );
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy("");
    }
  }

  const eligibleStreamCount = workspace.newsletters.filter(
    (item) => item.siteVisible,
  ).length;

  return (
    <LearningShell active="publishing" redesigned>
      <section className="atelier-page publishing-page">
        <header className="atelier-page-heading">
          <p className="atelier-eyebrow">Share deliberately</p>
          <h1>Publishing</h1>
          <p>Shape your public learning identity and understand exactly what is visible.</p>
        </header>

        {notice ? (
          <div className="atelier-notice" role="status">
            <Check size={15} />
            <span>{notice}</span>
            <button type="button" onClick={() => setNotice("")}>Dismiss</button>
          </div>
        ) : null}
        {error ? <AtelierError message={error} /> : null}

        {!site ? (
          <section className="publishing-claim">
            <div>
              <span className="atelier-icon"><Globe2 size={19} /></span>
              <p className="atelier-eyebrow">Claim your learning home</p>
              <h2>A lasting address for what you choose to share.</h2>
              <p>
                Claiming an address does not publish your streams or lessons. Your
                site starts private.
              </p>
            </div>
            <form onSubmit={claim}>
              <label>
                <span>Address</span>
                <input
                  required
                  minLength={3}
                  maxLength={30}
                  pattern="[a-zA-Z][a-zA-Z0-9-]{1,28}[a-zA-Z0-9]"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="yourname"
                />
                <small>
                  {personalSiteHost(username.trim().toLowerCase() || "yourname")}
                  {availability === true ? " · Available" : availability === false ? " · Unavailable" : ""}
                </small>
              </label>
              <label>
                <span>Display name</span>
                <input
                  required
                  maxLength={80}
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="Your learning archive"
                />
              </label>
              <button
                className="atelier-primary"
                type="submit"
                disabled={busy === "claim" || availability === false}
              >
                {busy === "claim" ? "Creating…" : "Create private site"}
                <ArrowRight size={15} />
              </button>
            </form>
          </section>
        ) : (
          <>
            <div className="publishing-status" aria-label="Current publishing state">
              <span className={site.visibility === "public" ? "is-live" : ""}>
                {site.visibility === "public" ? <Eye size={13} /> : <EyeOff size={13} />}
                {site.visibility === "public" ? "Site public" : "Site private"}
              </span>
              <span className={site.searchIndexing ? "is-live" : ""}>
                <Search size={13} />
                {site.searchIndexing ? "Search indexing on" : "Search indexing off"}
              </span>
              <span>
                <LockKeyhole size={13} />
                {workspace.loading
                  ? "Checking streams…"
                  : eligibleStreamCount === 0
                    ? "No streams eligible"
                    : `${eligibleStreamCount} stream${eligibleStreamCount === 1 ? "" : "s"} eligible`}
              </span>
            </div>

            <div className="publishing-layout">
              <div className="publishing-main">
                <form className="publishing-identity" onSubmit={saveIdentity}>
                  <div className="publishing-card-heading">
                    <span className="atelier-icon"><Globe2 size={18} /></span>
                    <div>
                      <p className="atelier-eyebrow">Public identity</p>
                      <h2>{personalSiteHost(site.username)}</h2>
                    </div>
                  </div>
                  <label>
                    <span>Display name</span>
                    <input
                      required
                      maxLength={80}
                      value={displayName}
                      onChange={(event) => setDisplayName(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>Bio or description</span>
                    <textarea
                      maxLength={400}
                      rows={6}
                      value={description}
                      onChange={(event) => setDescription(event.target.value)}
                    />
                    <small>{description.length}/400</small>
                  </label>
                  <button className="atelier-primary" type="submit" disabled={busy === "identity"}>
                    <Save size={14} />{busy === "identity" ? "Saving…" : "Save identity"}
                  </button>
                </form>

                <article className="publishing-analytics">
                  <div className="publishing-analytics-heading">
                    <div>
                      <span className="atelier-icon"><TrendingUp size={17} /></span>
                      <div>
                        <p className="atelier-eyebrow">Public path analytics</p>
                        <h2>What sharing leads to.</h2>
                      </div>
                    </div>
                    <select
                      aria-label="Analytics period"
                      value={analyticsPeriod}
                      onChange={(event) => setAnalyticsPeriod(Number(event.target.value))}
                    >
                      <option value="7">7 days</option>
                      <option value="30">30 days</option>
                      <option value="90">90 days</option>
                    </select>
                  </div>
                  {analyticsError ? (
                    <p role="alert">{analyticsError}</p>
                  ) : analytics ? (
                    <>
                      <div className="publishing-analytics-grid">
                        <div><strong>{analytics.views}</strong><span>Dossier views</span></div>
                        <div><strong>{analytics.uniqueViewers}</strong><span>Unique readers</span></div>
                        <div><strong>{analytics.shares}</strong><span>Share actions</span></div>
                        <div><strong>{analytics.follows}</strong><span>Confirmed follows</span></div>
                        <div><strong>{analytics.ctaClicks}</strong><span>Path starts</span></div>
                        <div><strong>{analytics.attributedSignups}</strong><span>Signups</span></div>
                        <div><strong>{analytics.attributedActivations}</strong><span>Activated learners</span></div>
                      </div>
                      <small>
                        Counts are privacy-safe aggregates. Repeat activity is deduplicated by
                        Dossier, action, and day; visitor identities and raw IP addresses are never
                        shown.
                      </small>
                    </>
                  ) : <AtelierLoading label="Loading public analytics…" />}
                </article>
              </div>

              <div className="publishing-side">
                <article className="site-preview">
                  <div className="site-preview-heading">
                    <div>
                      <p className="atelier-eyebrow">Live preview</p>
                      <strong>{personalSiteHost(site.username)}</strong>
                    </div>
                    {site.visibility === "public" && site.url ? (
                      <a href={site.url} target="_blank" rel="noreferrer">
                        View site <ExternalLink size={13} />
                      </a>
                    ) : null}
                  </div>
                  <div className="site-preview-paper">
                    <div className="site-preview-paper-top">
                      <span>Personal learning archive</span>
                      <span className={`site-preview-state${site.visibility === "public" ? " is-live" : ""}`}>
                        {site.visibility === "public" ? <Eye size={12} /> : <EyeOff size={12} />}
                        {site.visibility === "public" ? "Public" : "Private"}
                      </span>
                    </div>
                    <h2>{displayName || site.displayName}</h2>
                    <p>{description || "A durable home for connected learning."}</p>
                    <div className="site-preview-topics">
                      {workspace.newsletters.filter((item) => item.siteVisible).slice(0, 2).map((item) => (
                        <span key={item.id}>{item.name}</span>
                      ))}
                    </div>
                  </div>
                </article>

                <article className="visibility-card">
                  <div>
                    <p className="atelier-eyebrow">Visibility ladder</p>
                    <h2>Know what people can see.</h2>
                  </div>
                  <div className="visibility-level is-site">
                    <span className="atelier-icon">
                      {site.visibility === "public" ? <Eye size={17} /> : <EyeOff size={17} />}
                    </span>
                    <div>
                      <strong>Site</strong>
                      <p>
                        {site.visibility === "public"
                          ? "Public. Eligible published content can be viewed."
                          : "Private. Nothing is publicly accessible."}
                      </p>
                    </div>
                    <button type="button" onClick={toggleVisibility} disabled={busy === "visibility"}>
                      {site.visibility === "public" ? "Make private" : "Publish site"}
                    </button>
                  </div>
                  <div className="visibility-level">
                    <span className="atelier-icon">
                      <Search size={17} />
                    </span>
                    <div>
                      <strong>Search discovery</strong>
                      <p>
                        {site.searchIndexing
                          ? "Enabled. Eligible published pages may appear in search results."
                          : "Off. Public links work, but pages request exclusion from search results."}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={toggleSearchIndexing}
                      disabled={
                        site.visibility !== "public" ||
                        busy === "search-indexing"
                      }
                    >
                      {site.searchIndexing ? "Disable indexing" : "Allow indexing"}
                    </button>
                  </div>
                  <div className="visibility-streams">
                    <span><LockKeyhole size={15} /> Streams</span>
                    {workspace.loading ? <AtelierLoading label="Checking stream visibility…" /> : null}
                    {workspace.newsletters.map((newsletter) => (
                      <a href={`/newsletters/${encodeURIComponent(newsletter.id)}`} key={newsletter.id}>
                        <span>{newsletter.name}</span>
                        <strong className={newsletter.siteVisible ? "is-eligible" : ""}>
                          {newsletter.siteVisible ? "Eligible to publish" : "Private"}
                        </strong>
                        <ArrowRight size={14} />
                      </a>
                    ))}
                  </div>
                  <p className="visibility-help">
                    A lesson appears publicly only when the site is public, its stream
                    is visible, and that lesson is published. Search discovery is a
                    separate opt-in for eligible public pages.
                  </p>
                </article>
              </div>
            </div>
          </>
        )}
      </section>
    </LearningShell>
  );
}
