import { useState } from "react";
import App from "./App.jsx";
import { demoSite } from "./demoData.js";

export default function DemoHostedApp() {
  const [site, setSite] = useState(demoSite);
  return <App site={site} onSiteUpdate={setSite} />;
}
