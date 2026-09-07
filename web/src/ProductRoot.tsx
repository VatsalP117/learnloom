import { ClerkProvider } from "@clerk/react";
import DemoHostedApp from "./DemoHostedApp";
import HostedApp from "./HostedApp";
import { demoMode } from "./api";

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
