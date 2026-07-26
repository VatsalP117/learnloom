import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import marketingHeroDesktop from "./assets/learnloom-hero-landscape.avif?url";
import marketingHeroMobile from "./assets/learnloom-hero-landscape-960.avif?url";
import marketingHeroTablet from "./assets/learnloom-hero-landscape-1440.avif?url";
import productBackdropDesktop from "./assets/learning-landscape-1920.avif?url";
import productBackdropMobile from "./assets/learning-landscape-960.avif?url";
import CalmLoader from "./CalmLoader";
import { rootDomain } from "./config";
import "./entry.css";

const MarketingLanding = lazy(() => import("./MarketingLanding"));
const LegalPage = lazy(() => import("./LegalPage"));
const ProductRoot = lazy(() => import("./ProductRoot"));
const hostname = window.location.hostname.toLowerCase();
const isLegalPage = ["/privacy", "/terms"].includes(window.location.pathname);
const isMarketingPage =
  hostname === rootDomain ||
  hostname === `www.${rootDomain}` ||
  window.location.pathname === "/marketing";

type BackdropSources = {
  desktop: string;
  mobile: string;
  tablet?: string;
};

type NavigatorConnection = {
  effectiveType?: string;
  saveData?: boolean;
};

function selectBackdrop(sources: BackdropSources) {
  if (window.matchMedia("(max-width: 680px)").matches) return sources.mobile;
  if (sources.tablet && window.matchMedia("(max-width: 1100px)").matches) return sources.tablet;
  return sources.desktop;
}

function warmBackdrop(sources: BackdropSources, priority: "high" | "low") {
  const connection = (navigator as Navigator & { connection?: NavigatorConnection }).connection;
  if (connection?.saveData || connection?.effectiveType === "slow-2g" || connection?.effectiveType === "2g") {
    return;
  }

  const preload = document.createElement("link");
  preload.rel = connection?.effectiveType === "3g" ? "prefetch" : "preload";
  preload.as = "image";
  preload.type = "image/avif";
  preload.href = selectBackdrop(sources);
  preload.setAttribute("fetchpriority", preload.rel === "prefetch" ? "low" : priority);
  document.head.append(preload);
}

if (isMarketingPage) {
  warmBackdrop(
    { desktop: marketingHeroDesktop, tablet: marketingHeroTablet, mobile: marketingHeroMobile },
    "high",
  );
} else if (!isLegalPage) {
  warmBackdrop({ desktop: productBackdropDesktop, mobile: productBackdropMobile }, "low");
}

const root = document.getElementById("root");
if (!root) throw new Error("The application root element is missing.");

createRoot(root).render(
  <StrictMode>
    <Suspense fallback={<CalmLoader label="Opening Learnloom…" />}>
      {isLegalPage ? <LegalPage /> : isMarketingPage ? <MarketingLanding /> : <ProductRoot />}
    </Suspense>
  </StrictMode>,
);
