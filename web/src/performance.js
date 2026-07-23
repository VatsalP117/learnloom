import { onCLS, onINP, onLCP } from "web-vitals";
import { apiFetch } from "./api.js";

let started = false;

export function startWebVitals() {
  if (started) return;
  started = true;
  const report = (metric) => {
    apiFetch("/api/performance/vitals", {
      method: "POST",
      keepalive: true,
      body: {
        name: metric.name,
        value: metric.value,
        rating: metric.rating,
        navigationType: metric.navigationType,
        page: performancePage(window.location.pathname),
      },
    }).catch(() => {});
  };
  onCLS(report);
  onINP(report);
  onLCP(report);
}

export function performancePage(pathname) {
  if (/^\/issues\/[^/]+/.test(pathname)) return "/issues/:id";
  if (/^\/newsletters\/[^/]+/.test(pathname)) return "/newsletters/:id";
  return pathname || "/";
}
