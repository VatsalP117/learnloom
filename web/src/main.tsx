import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import productBackdropDesktop from "./assets/learning-landscape-1920.avif?url";
import productBackdropMobile from "./assets/learning-landscape-960.avif?url";
import { rootDomain } from "./config";
import "./entry.css";

const MarketingLanding = lazy(() => import("./MarketingLanding"));
const ProductRoot = lazy(() => import("./ProductRoot"));
const hostname = window.location.hostname.toLowerCase();
const isMarketingPage =
  hostname === rootDomain ||
  hostname === `www.${rootDomain}` ||
  window.location.pathname === "/marketing";

if (!isMarketingPage) {
  const preload = document.createElement("link");
  preload.rel = "preload";
  preload.as = "image";
  preload.type = "image/avif";
  preload.href = window.matchMedia("(max-width: 820px)").matches
    ? productBackdropMobile
    : productBackdropDesktop;
  document.head.append(preload);
}

const root = document.getElementById("root");
if (!root) throw new Error("The application root element is missing.");

createRoot(root).render(
  <StrictMode>
    <Suspense fallback={<main className="entry-loading">Opening Learnloom…</main>}>
      {isMarketingPage ? <MarketingLanding /> : <ProductRoot />}
    </Suspense>
  </StrictMode>,
);
