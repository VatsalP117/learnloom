import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import MarketingLanding from "./MarketingLanding";

const root = document.getElementById("root");
if (!root) throw new Error("The marketing root element is missing.");

createRoot(root).render(
  <StrictMode>
    <MarketingLanding />
  </StrictMode>,
);
