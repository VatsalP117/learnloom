import {
  ClerkFailed,
  ClerkLoading,
  RedirectToSignIn,
  Show,
  SignIn,
  SignUp,
} from "@clerk/react";
import { useEffect, useState } from "react";
import App from "./App.jsx";

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
  const [profile, setProfile] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    fetch("/api/me", { signal: controller.signal })
      .then(async (response) => {
        const body = await response.json();
        if (!response.ok) throw new Error(body.error ?? "Your account could not be loaded.");
        return body;
      })
      .then(setProfile)
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, []);

  if (error) return <AuthPage><p>{error}</p></AuthPage>;
  if (!profile) return <AuthPage><p>Preparing your workspace…</p></AuthPage>;
  if (!profile.site) {
    return <ClaimUsername csrfToken={profile.csrfToken} onClaim={(site) => setProfile({ ...profile, site })} />;
  }
  return (
    <>
      <App />
      <SiteControl
        csrfToken={profile.csrfToken}
        site={profile.site}
        onUpdate={(site) => setProfile({ ...profile, site })}
      />
    </>
  );
}

function SiteControl({ csrfToken, site, onUpdate }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const nextVisibility = site.visibility === "public" ? "private" : "public";

  async function toggleVisibility() {
    setBusy(true);
    setError("");
    try {
      const response = await fetch("/api/me/site/settings", {
        method: "POST",
        headers: { "content-type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({
          _csrf: csrfToken,
          visibility: nextVisibility,
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error ?? "Site visibility could not be updated.");
      }
      onUpdate(body.site);
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <aside className="site-control" aria-label="Public site controls">
      <div>
        <span className={`site-status ${site.visibility}`}>{site.visibility}</span>
        <strong>{site.username}.learnloom.blog</strong>
      </div>
      <div className="site-control-actions">
        {site.visibility === "public" && site.url ? (
          <a href={site.url} target="_blank" rel="noreferrer">View site</a>
        ) : null}
        <button type="button" disabled={busy} onClick={toggleVisibility}>
          {busy
            ? "Saving…"
            : nextVisibility === "public"
              ? "Publish site"
              : "Make private"}
        </button>
      </div>
      {error ? <p>{error}</p> : null}
    </aside>
  );
}

function ClaimUsername({ csrfToken, onClaim }) {
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
      fetch(`/api/usernames/${encodeURIComponent(normalized)}`, {
        signal: controller.signal,
      })
        .then((response) => response.json())
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
      const response = await fetch("/api/me/site/claim", {
        method: "POST",
        headers: { "content-type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ _csrf: csrfToken, username, displayName }),
      });
      const body = await response.json();
      if (!response.ok) throw new Error(body.error ?? "That username could not be claimed.");
      onClaim(body.site);
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setBusy(false);
    }
  }

  const normalized = username.trim().toLowerCase();
  return (
    <main className="auth-shell">
      <section className="claim-card">
        <p className="overline">Claim your learning home</p>
        <h1>Choose your Learnloom address</h1>
        <p>Your Dossiers will live at a personal subdomain after you choose to publish them.</p>
        <form onSubmit={submit}>
          <label>
            <span>Username</span>
            <input required minLength={3} maxLength={30} pattern="[a-zA-Z][a-zA-Z0-9-]{1,28}[a-zA-Z0-9]" value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" />
          </label>
          <div className="subdomain-preview">{normalized || "yourname"}.learnloom.blog</div>
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
          <button className="primary-button claim-submit" disabled={busy}>{busy ? "Claiming…" : "Claim username"}</button>
        </form>
      </section>
    </main>
  );
}

function AuthPage({ children }) {
  return <main className="auth-shell">{children}</main>;
}
