import {
  ClerkFailed,
  ClerkLoading,
  RedirectToSignIn,
  Show,
  useAuth,
} from "@clerk/react";
import { useEffect, useState } from "react";
import App from "./App.jsx";
import AuthPage from "./AuthPage.jsx";
import { apiJSON, configureAPI, setCSRFToken } from "./api.js";

export default function HostedApp() {
  const path = window.location.pathname;
  if (path.startsWith("/sign-in")) {
    return <AuthPage mode="sign-in" />;
  }
  if (path.startsWith("/sign-up")) {
    return <AuthPage mode="sign-up" />;
  }
  return (
    <>
      <ClerkLoading><AuthPage status="Loading your workspace…" /></ClerkLoading>
      <ClerkFailed><AuthPage status="Authentication could not be loaded." /></ClerkFailed>
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
        import("./performance.js")
          .then(({ startWebVitals }) => startWebVitals())
          .catch(() => {});
      })
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, [getToken]);

  if (error) return <AuthPage status={error} />;
  if (!profile) return <AuthPage status="Preparing your workspace…" />;
  return (
    <App
      capabilities={profile.capabilities ?? {}}
      site={profile.site}
      onSiteUpdate={(site) => setProfile({ ...profile, site })}
    />
  );
}
