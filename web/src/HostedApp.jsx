import {
  ClerkFailed,
  ClerkLoading,
  RedirectToSignIn,
  Show,
  SignIn,
  SignUp,
  useAuth,
} from "@clerk/react";
import { useEffect, useState } from "react";
import App from "./App.jsx";
import { apiJSON, configureAPI, setCSRFToken } from "./api.js";

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
    <App
      capabilities={profile.capabilities ?? {}}
      site={profile.site}
      onSiteUpdate={(site) => setProfile({ ...profile, site })}
    />
  );
}

function AuthPage({ children }) {
  return <main className="auth-shell">{children}</main>;
}
