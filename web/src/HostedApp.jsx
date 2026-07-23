import {
  ClerkFailed,
  ClerkLoading,
  RedirectToSignIn,
  Show,
  SignIn,
  SignUp,
  useAuth,
} from "@clerk/react";
import { ExternalLink, Globe2, X } from "lucide-react";
import { useEffect, useState } from "react";
import App from "./App.jsx";
import { apiJSON, configureAPI, setCSRFToken } from "./api.js";
import { personalSiteHost } from "./config.js";

export default function HostedApp() {
  const path = window.location.pathname;
  if (path.startsWith("/sign-in")) {
    return <AuthPage><SignIn path="/sign-in" routing="path" signUpUrl="/sign-up" fallbackRedirectUrl="/" /></AuthPage>;
  }
  if (path.startsWith("/sign-up")) {
    return <AuthPage><SignUp path="/sign-up" routing="path" signInUrl="/sign-in" fallbackRedirectUrl="/" /></AuthPage>;
  }
  return (
    <>
      <ClerkLoading><AuthPage><p>Loading your workspace…</p></AuthPage></ClerkLoading>
      <ClerkFailed><AuthPage><p>Authentication could not be loaded.</p></AuthPage></ClerkFailed>
      <Show when="signed-out"><RedirectToSignIn /></Show>
      <Show when="signed-in"><OnboardingGate /></Show>
    </>
  );
}

function OnboardingGate() {
  const { getToken } = useAuth();
  const [profile, setProfile] = useState(null);
  const [error, setError] = useState("");
  const [siteSetupOpen, setSiteSetupOpen] = useState(false);
  const showSiteControl = window.location.pathname !== "/newsletters/new";

  useEffect(() => {
    configureAPI(getToken);
    const controller = new AbortController();
    apiJSON("/api/me", { signal: controller.signal })
      .then((body) => {
        setCSRFToken(body.csrfToken);
        setProfile(body);
      })
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, [getToken]);

  if (error) return <AuthPage><p>{error}</p></AuthPage>;
  if (!profile) return <AuthPage><p>Preparing your workspace…</p></AuthPage>;
  return (
    <>
      <App capabilities={profile.capabilities ?? {}} />
      {profile.site && showSiteControl ? (
        <SiteControl
          site={profile.site}
          onUpdate={(site) => setProfile({ ...profile, site })}
        />
      ) : !profile.site && showSiteControl ? (
        <>
          <button className="site-setup-launcher" type="button" onClick={() => setSiteSetupOpen(true)}>
            <Globe2 size={16} />
            <span><strong>Personal site</strong><small>Set up when you’re ready to share</small></span>
          </button>
          {siteSetupOpen ? (
            <div className="site-setup-overlay" role="presentation" onMouseDown={() => setSiteSetupOpen(false)}>
              <ClaimUsername
                embedded
                onCancel={() => setSiteSetupOpen(false)}
                onClaim={(site) => {
                  setProfile({ ...profile, site });
                  setSiteSetupOpen(false);
                }}
              />
            </div>
          ) : null}
        </>
      ) : null}
    </>
  );
}

export function SiteControl({ site, onUpdate }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [expanded, setExpanded] = useState(false);
  const nextVisibility = site.visibility === "public" ? "private" : "public";

  async function toggleVisibility() {
    setBusy(true);
    setError("");
    try {
      const body = await apiJSON("/api/me/site/settings", {
        method: "POST",
        body: { visibility: nextVisibility },
      });
      onUpdate(body.site);
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <aside className={`site-control ${expanded ? "expanded" : ""}`} aria-label="Personal site controls">
      <button className="site-control-summary" type="button" onClick={() => setExpanded((current) => !current)} aria-expanded={expanded}>
        <Globe2 size={16} />
        <span><strong>Personal site</strong><small>{personalSiteHost(site.username)}</small></span>
        <i className={`site-dot ${site.visibility}`} />
      </button>
      {expanded ? (
        <div className="site-control-panel">
          <div><span className={`site-status ${site.visibility}`}>{site.visibility}</span><button type="button" onClick={() => setExpanded(false)} aria-label="Close personal site controls"><X size={15} /></button></div>
          <p>{site.visibility === "public" ? "Your published learning is visible at your personal address." : "Your site is private. Publish it whenever you have something ready to share."}</p>
          <div className="site-control-actions">
            {site.visibility === "public" && site.url ? (
              <a href={site.url} target="_blank" rel="noreferrer">View site <ExternalLink size={13} /></a>
            ) : null}
            <button type="button" disabled={busy} onClick={toggleVisibility}>
              {busy ? "Saving…" : nextVisibility === "public" ? "Publish site" : "Make private"}
            </button>
          </div>
          {error ? <p className="site-control-error">{error}</p> : null}
        </div>
      ) : null}
    </aside>
  );
}

function ClaimUsername({ onClaim, onCancel, embedded = false }) {
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [availability, setAvailability] = useState(null);

  useEffect(() => {
    const normalized = username.trim().toLowerCase();
    if (normalized.length < 3) {
      setAvailability(null);
      return;
    }
    const controller = new AbortController();
    const timer = setTimeout(() => {
      apiJSON(`/api/usernames/${encodeURIComponent(normalized)}`, {
        signal: controller.signal,
      })
        .then((body) => setAvailability(Boolean(body.available)))
        .catch((requestError) => {
          if (requestError.name !== "AbortError") setAvailability(null);
        });
    }, 300);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [username]);

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body = await apiJSON("/api/me/site/claim", {
        method: "POST",
        body: { username, displayName },
      });
      onClaim(body.site);
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy(false);
    }
  }

  const normalized = username.trim().toLowerCase();
  const card = (
      <section className="claim-card" onMouseDown={(event) => event.stopPropagation()}>
        {onCancel ? <button className="claim-close" type="button" onClick={onCancel} aria-label="Close"><X size={18} /></button> : null}
        <p className="overline">Claim your learning home</p>
        <h1>A lasting home for what you learn</h1>
        <p>Choose an address for the lessons you decide to publish. Your streams stay private unless you explicitly share them.</p>
        <form onSubmit={submit}>
          <label>
            <span>Username</span>
            <input required minLength={3} maxLength={30} pattern="[a-zA-Z][a-zA-Z0-9-]{1,28}[a-zA-Z0-9]" value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" />
          </label>
          <div className="subdomain-preview">
            {personalSiteHost(normalized || "yourname")}
          </div>
          {availability !== null ? (
            <div className={availability ? "username-available" : "username-unavailable"}>
              {availability ? "Available" : "Unavailable"}
            </div>
          ) : null}
          <label>
            <span>Display name</span>
            <input required maxLength={100} value={displayName} onChange={(event) => setDisplayName(event.target.value)} autoComplete="name" />
          </label>
          {error ? <p className="claim-error">{error}</p> : null}
          <button className="primary-button claim-submit" disabled={busy || availability === false}>{busy ? "Creating your site…" : "Create personal site"}</button>
        </form>
      </section>
  );
  return embedded ? card : <main className="auth-shell">{card}</main>;
}

function AuthPage({ children }) {
  return <main className="auth-shell">{children}</main>;
}
