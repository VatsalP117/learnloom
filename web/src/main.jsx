import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { rootDomain } from "./config.js";
import "./entry.css";

const MarketingLanding = lazy(() => import("./MarketingLanding.jsx"));
const ProductRoot = lazy(() => import("./ProductRoot.jsx"));
const hostname = window.location.hostname.toLowerCase();
const isMarketingPage =
  hostname === rootDomain ||
  hostname === `www.${rootDomain}` ||
  window.location.pathname === "/marketing";

createRoot(document.getElementById("root")).render(
  <StrictMode>
    <Suspense fallback={<main className="entry-loading">Opening Learnloom…</main>}>
      {isMarketingPage ? <MarketingLanding /> : <ProductRoot />}
    </Suspense>
  </StrictMode>,
);
