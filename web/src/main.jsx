import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ClerkProvider } from "@clerk/react";
import "@fontsource/inter/latin-400.css";
import "@fontsource/inter/latin-500.css";
import "@fontsource/inter/latin-600.css";
import "@fontsource/inter/latin-700.css";
import HostedApp from "./HostedApp.jsx";
import MarketingLanding from "./MarketingLanding.jsx";
import "./styles.css";

const clerkPublishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;
const hostname = window.location.hostname.toLowerCase();
const isMarketingPage =
  hostname === "learnloom.blog" ||
  hostname === "www.learnloom.blog" ||
  window.location.pathname === "/marketing";

createRoot(document.getElementById("root")).render(
  <StrictMode>
    {isMarketingPage ? (
      <MarketingLanding />
    ) : clerkPublishableKey ? (
      <ClerkProvider publishableKey={clerkPublishableKey} afterSignOutUrl="/sign-in">
        <HostedApp />
      </ClerkProvider>
    ) : (
      <main className="auth-shell">
        <section className="claim-card">
          <h1>Learnloom is not configured</h1>
          <p>The hosted app requires a Clerk publishable key at build time.</p>
        </section>
      </main>
    )}
  </StrictMode>,
);
