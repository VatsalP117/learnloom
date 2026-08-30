import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import MarketingLanding from "./MarketingLanding";
import { appOrigin } from "./config";

const root = document.getElementById("root");
if (!root) throw new Error("The marketing root element is missing.");

// Warm the connection to the hosted app before the visitor clicks Sign in or
// Get started, so the first navigation to /sign-in or /sign-up skips the
// DNS + TCP + TLS handshake. The https: guard keeps this inert if a build
// environment ever provides a non-HTTPS origin.
if (appOrigin.startsWith("https://")) {
  const preconnect = document.createElement("link");
  preconnect.rel = "preconnect";
  preconnect.href = appOrigin;
  document.head.append(preconnect);
}

createRoot(root).render(
  <StrictMode>
    <MarketingLanding />
  </StrictMode>,
);
