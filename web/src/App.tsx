import { lazy, Suspense } from "react";
import CalmLoader from "./CalmLoader";
import type { NotificationPreferences, Site } from "./types";

// The authenticated app graph lives in its own chunk (AppGraph) so signed-out
// visitors on /sign-in and /sign-up never download it; HostedApp lazy-loads
// this wrapper and starts the fetch in parallel with /api/me.
const AppGraph = lazy(() => import("./AppGraph"));

interface AppProps {
  capabilities?: { sourceDiscovery?: boolean };
  site?: Site | null;
  onSiteUpdate?: (site: Site) => void;
  initialNotifications?: NotificationPreferences;
  onNotificationsUpdate?: (notifications: NotificationPreferences) => void;
}

export default function App({
  capabilities = {},
  site = null,
  onSiteUpdate,
  initialNotifications,
  onNotificationsUpdate,
}: AppProps) {
  return (
    <Suspense fallback={<CalmLoader label="Preparing your workspace…" />}>
      <AppGraph
        capabilities={capabilities}
        site={site}
        onSiteUpdate={onSiteUpdate}
        initialNotifications={initialNotifications}
        onNotificationsUpdate={onNotificationsUpdate}
      />
    </Suspense>
  );
}
