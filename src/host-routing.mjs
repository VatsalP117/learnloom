const DEPLOYMENT_MODES = new Set(["local", "hosted"]);
const USERNAME_PATTERN = /^[a-z][a-z0-9-]{1,28}[a-z0-9]$/;
const DNS_LABEL_PATTERN = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

export const RESERVED_SITE_NAMES = Object.freeze([
  "admin",
  "api",
  "app",
  "assets",
  "auth",
  "blog",
  "clerk",
  "dashboard",
  "docs",
  "help",
  "learnloom",
  "mail",
  "root",
  "status",
  "support",
  "www",
]);

const RESERVED_SITE_NAME_SET = new Set(RESERVED_SITE_NAMES);

export function resolveDeploymentConfig(options = {}) {
  const environment = options.env ?? process.env;
  const rawMode = environment.LEARNLOOM_DEPLOYMENT_MODE ?? "local";
  const mode = String(rawMode).trim().toLowerCase();
  if (!DEPLOYMENT_MODES.has(mode)) {
    throw new Error(
      'LEARNLOOM_DEPLOYMENT_MODE must be either "local" or "hosted".',
    );
  }
  if (mode === "local") {
    return Object.freeze({ mode: "local" });
  }

  const rootDomain = normalizeRootDomain(environment.LEARNLOOM_ROOT_DOMAIN);
  const expectedAppHostname = `app.${rootDomain}`;
  const appOrigin = normalizeAppOrigin(
    environment.LEARNLOOM_APP_ORIGIN,
    expectedAppHostname,
  );
  return Object.freeze({
    mode: "hosted",
    rootDomain,
    appHostname: expectedAppHostname,
    appOrigin,
    apexOrigin: `https://${rootDomain}`,
    clerkFrontendApiOrigin: `https://clerk.${rootDomain}`,
  });
}

export function resolveRequestHost(hostHeader, deployment, options = {}) {
  const hostname = parseHostHeader(hostHeader);
  if (!hostname) return Object.freeze({ kind: "rejected" });

  if (!deployment || deployment.mode === "local") {
    const allowedHosts = new Set(
      (options.allowedHosts ?? ["127.0.0.1", "localhost", "[::1]"]).map(
        normalizeAllowedHost,
      ),
    );
    return allowedHosts.has(hostname)
      ? Object.freeze({ kind: "local", hostname })
      : Object.freeze({ kind: "rejected" });
  }

  if (deployment.mode !== "hosted") {
    return Object.freeze({ kind: "rejected" });
  }
  if (hostname === deployment.rootDomain) {
    return Object.freeze({ kind: "apex", hostname });
  }
  if (hostname === `www.${deployment.rootDomain}`) {
    return Object.freeze({ kind: "www", hostname });
  }
  if (hostname === deployment.appHostname) {
    return Object.freeze({ kind: "app", hostname });
  }

  const suffix = `.${deployment.rootDomain}`;
  if (!hostname.endsWith(suffix)) {
    return Object.freeze({ kind: "rejected" });
  }
  const username = hostname.slice(0, -suffix.length);
  if (!isPotentialSiteUsername(username)) {
    return Object.freeze({ kind: "rejected" });
  }
  return Object.freeze({ kind: "site", hostname, username });
}

export function isPotentialSiteUsername(value) {
  return (
    typeof value === "string" &&
    USERNAME_PATTERN.test(value) &&
    !value.includes("--") &&
    !RESERVED_SITE_NAME_SET.has(value)
  );
}

function normalizeRootDomain(value) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(
      "LEARNLOOM_ROOT_DOMAIN is required when LEARNLOOM_DEPLOYMENT_MODE=hosted.",
    );
  }
  const rootDomain = value.trim().toLowerCase();
  if (
    rootDomain.length > 253 ||
    isIP(rootDomain) !== 0 ||
    !/^[a-z0-9.-]+$/.test(rootDomain) ||
    rootDomain.startsWith(".") ||
    rootDomain.endsWith(".") ||
    rootDomain.includes("..")
  ) {
    throw new Error("LEARNLOOM_ROOT_DOMAIN must be a valid ASCII DNS name.");
  }
  const labels = rootDomain.split(".");
  if (
    labels.length < 2 ||
    labels.some((label) => !DNS_LABEL_PATTERN.test(label))
  ) {
    throw new Error("LEARNLOOM_ROOT_DOMAIN must be a valid registrable domain.");
  }
  return rootDomain;
}

function normalizeAppOrigin(value, expectedHostname) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(
      "LEARNLOOM_APP_ORIGIN is required when LEARNLOOM_DEPLOYMENT_MODE=hosted.",
    );
  }
  let url;
  try {
    url = new URL(value);
  } catch {
    throw new Error("LEARNLOOM_APP_ORIGIN must be a valid HTTPS origin.");
  }
  if (
    url.protocol !== "https:" ||
    url.hostname.toLowerCase() !== expectedHostname ||
    url.port ||
    url.username ||
    url.password ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  ) {
    throw new Error(
      `LEARNLOOM_APP_ORIGIN must be exactly https://${expectedHostname}.`,
    );
  }
  return url.origin;
}

function parseHostHeader(hostHeader) {
  if (
    typeof hostHeader !== "string" ||
    hostHeader.length === 0 ||
    hostHeader.length > 255
  ) {
    return null;
  }
  try {
    const url = new URL(`http://${hostHeader}`);
    if (
      url.username ||
      url.password ||
      url.pathname !== "/" ||
      url.search ||
      url.hash
    ) {
      return null;
    }
    return url.hostname.toLowerCase();
  } catch {
    return null;
  }
}

function normalizeAllowedHost(value) {
  return String(value).trim().toLowerCase();
}
import { isIP } from "node:net";
