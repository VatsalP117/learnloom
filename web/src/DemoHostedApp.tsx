import { useState } from "react";
import App from "./App";
import { demoSite } from "./demoData";
import type { Site } from "./types";

export default function DemoHostedApp() {
  const [site, setSite] = useState<Site>(demoSite);
  return <App site={site} onSiteUpdate={setSite} />;
}
