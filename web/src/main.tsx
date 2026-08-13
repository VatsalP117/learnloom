import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import productBackdropDesktop from "./assets/learning-landscape-1920.avif?url";
import productBackdropMobile from "./assets/learning-landscape-960.avif?url";
import CalmLoader from "./CalmLoader";
import "./entry.css";

const CanonicalDossier = lazy(() => import("./CanonicalDossier"));
const LegalPage = lazy(() => import("./LegalPage"));
const ProductRoot = lazy(() => import("./ProductRoot"));
const isLegalPage = ["/privacy", "/terms"].includes(window.location.pathname);
const isExamplePage = ["/examples", "/examples/ai-evaluation"].includes(window.location.pathname);

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

if (!isLegalPage) {
  warmBackdrop({ desktop: productBackdropDesktop, mobile: productBackdropMobile }, "low");
}

const root = document.getElementById("root");
if (!root) throw new Error("The application root element is missing.");

createRoot(root).render(
  <StrictMode>
    <Suspense fallback={<CalmLoader label="Opening Learnloom…" />}>
      {isLegalPage ? <LegalPage /> : isExamplePage ? <CanonicalDossier /> : <ProductRoot />}
    </Suspense>
  </StrictMode>,
);
