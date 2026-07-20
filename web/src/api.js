let tokenGetter = null;
let csrfToken = "";

export const demoMode =
  import.meta.env.DEV && import.meta.env.VITE_DEMO_MODE === "true";

export function configureAPI(getToken) {
  tokenGetter = getToken;
}

export function setCSRFToken(token) {
  csrfToken = token ?? "";
}

export async function apiFetch(path, options = {}) {
  if (!tokenGetter) {
    throw new Error("Authentication is not ready.");
  }

  const token = await tokenGetter();
  if (!token) {
    throw new Error("Your session has expired. Sign in again.");
  }

  const method = (options.method ?? "GET").toUpperCase();
  const headers = new Headers(options.headers);
  headers.set("authorization", `Bearer ${token}`);
  headers.set("accept", "application/json");

  let body = options.body;
  if (!["GET", "HEAD"].includes(method)) {
    headers.set("content-type", "application/json");
    headers.set("x-csrf-token", csrfToken);
    if (body !== undefined && typeof body !== "string") {
      body = JSON.stringify(body);
    }
  }

  return fetch(path, { ...options, method, headers, body });
}

export async function apiJSON(path, options) {
  if (demoMode) {
    const { demoResponse } = await import("./demoData.js");
    return demoResponse(path, options);
  }
  const response = await apiFetch(path, options);
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(body?.message ?? "The request could not be completed.");
  }
  return body;
}
