import {
  AuthenticateWithRedirectCallback,
  ClerkFailed,
  ClerkLoaded,
  ClerkLoading,
  RedirectToSignIn,
  Show,
  useAuth,
  useClerk,
} from "@clerk/react";
import { useEffect, useState } from "react";
import App from "./App";
import AuthPage from "./AuthPage";
import CalmLoader from "./CalmLoader";
import { SessionActionsProvider } from "./LearningShell";
import { apiJSON, configureAPI, setCSRFToken } from "./api";
import type { Profile } from "./types";

export default function HostedApp() {
  const path = window.location.pathname;
  if (path === "/sso-callback") {
    return <SSOCallbackRoute />;
  }
  if (path.startsWith("/sign-in")) {
    return <AuthRoute mode="sign-in" />;
  }
  if (path.startsWith("/sign-up")) {
    return <AuthRoute mode="sign-up" />;
  }
  return (
    <>
      <ClerkLoading><CalmLoader label="Returning to your learning space…" /></ClerkLoading>
      <ClerkFailed><AuthPage status="Authentication could not be loaded." /></ClerkFailed>
      <Show when="signed-out"><RedirectToSignIn /></Show>
      <Show when="signed-in"><OnboardingGate /></Show>
    </>
  );
}

function SSOCallbackRoute() {
  return (
    <>
      <AuthPage
        status="Finishing your secure sign-in…"
        statusDetail="Google has confirmed your identity. We’re opening your learning space."
      />
      <AuthenticateWithRedirectCallback
        signInUrl="/sign-in"
        signUpUrl="/sign-up"
        signInFallbackRedirectUrl="/"
        signUpFallbackRedirectUrl="/"
      />
    </>
  );
}

function AuthRoute({ mode }: { mode: "sign-in" | "sign-up" }) {
  return (
    <>
      <ClerkLoading><AuthPage status="Loading secure authentication…" /></ClerkLoading>
      <ClerkFailed>
        <AuthPage
          status="Authentication could not be loaded."
          statusDetail="Refresh the page to try again."
          statusKind="error"
        />
      </ClerkFailed>
      <ClerkLoaded>
        <Show when="signed-out"><AuthPage mode={mode} /></Show>
        <Show when="signed-in"><AuthenticatedRedirect /></Show>
      </ClerkLoaded>
    </>
  );
}

function AuthenticatedRedirect() {
  useEffect(() => {
    window.location.replace("/");
  }, []);

  return (
    <AuthPage
      status="You’re already signed in."
      statusDetail="Opening your learning workspace…"
    />
  );
}

function OnboardingGate() {
  const { getToken } = useAuth();
  const clerk = useClerk();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    configureAPI(getToken);
    const controller = new AbortController();
    apiJSON<Profile>("/api/me", { signal: controller.signal })
      .then((body) => {
        setCSRFToken(body.csrfToken);
        setProfile(body);
        import("./performance")
          .then(({ startWebVitals }) => startWebVitals())
          .catch(() => {});
      })
      .catch((requestError) => {
        if (requestError.name !== "AbortError") setError(requestError.message);
      });
    return () => controller.abort();
  }, [getToken]);

  if (error) return <AuthPage status={error} />;
  if (!profile) {
    return (
      <CalmLoader
        label="Preparing your workspace…"
        detail="Restoring your streams and recent lessons."
      />
    );
  }
  return (
    <SessionActionsProvider onSignOut={() => clerk.signOut({ redirectUrl: "/sign-in" })}>
      <App
        capabilities={profile.capabilities ?? {}}
        site={profile.site}
        onSiteUpdate={(site) => setProfile({ ...profile, site })}
      />
    </SessionActionsProvider>
  );
}
