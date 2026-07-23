import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ClerkProvider } from "@clerk/react";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import "@fontsource/newsreader/latin-500.css";
import "@fontsource/newsreader/latin-600.css";
import "@fontsource/newsreader/latin-700.css";
import HostedApp from "./HostedApp.jsx";
import DemoHostedApp from "./DemoHostedApp.jsx";
import MarketingLanding from "./MarketingLanding.jsx";
import { demoMode } from "./api.js";
import { rootDomain } from "./config.js";
import "./styles.css";
import "./redesign.css";

const clerkPublishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;
const hostname = window.location.hostname.toLowerCase();
const isMarketingPage =
  hostname === rootDomain ||
  hostname === `www.${rootDomain}` ||
  window.location.pathname === "/marketing";

createRoot(document.getElementById("root")).render(
  <StrictMode>
    {isMarketingPage ? (
      <MarketingLanding />
    ) : demoMode ? (
      <DemoHostedApp />
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
