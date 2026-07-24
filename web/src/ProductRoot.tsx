import { ClerkProvider } from "@clerk/react";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import "@fontsource/bricolage-grotesque/latin-500.css";
import "@fontsource/bricolage-grotesque/latin-600.css";
import "@fontsource/bricolage-grotesque/latin-700.css";
import DemoHostedApp from "./DemoHostedApp";
import HostedApp from "./HostedApp";
import { demoMode } from "./api";
import "./styles.css";
import "./redesign.css";

const clerkPublishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;

export default function ProductRoot() {
  if (demoMode) return <DemoHostedApp />;

  if (clerkPublishableKey) {
    return (
      <ClerkProvider publishableKey={clerkPublishableKey} afterSignOutUrl="/sign-in">
        <HostedApp />
      </ClerkProvider>
    );
  }

  return (
    <main className="auth-shell">
      <section className="claim-card">
        <h1>Learnloom is not configured</h1>
        <p>The hosted app requires a Clerk publishable key at build time.</p>
      </section>
    </main>
  );
}
