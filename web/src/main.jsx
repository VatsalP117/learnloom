import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ClerkProvider } from "@clerk/react";
import "@fontsource/inter/latin-400.css";
import "@fontsource/inter/latin-500.css";
import "@fontsource/inter/latin-600.css";
import "@fontsource/inter/latin-700.css";
import App from "./App.jsx";
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
    ) : <App />}
  </StrictMode>,
);
