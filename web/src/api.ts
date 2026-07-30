type TokenGetter = () => Promise<string | null>;

export type APIRequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

let tokenGetter: TokenGetter | null = null;
let csrfToken = "";

export const demoMode =
  import.meta.env.DEV && import.meta.env.VITE_DEMO_MODE === "true";

export function configureAPI(getToken: TokenGetter) {
  tokenGetter = getToken;
}

export function setCSRFToken(token?: string | null) {
  csrfToken = token ?? "";
}

export class APIError extends Error {
  readonly status: number;
  readonly code: APIProblemCode | "unknown_error";
  readonly requestId: string;

  constructor(
    message: string,
    status: number,
    code: APIProblemCode | "unknown_error",
    requestId = "",
  ) {
    super(message);
    this.name = "APIError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export async function apiFetch(path: string, options: APIRequestOptions = {}) {
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
  } else {
    body = options.body;
  }

  return fetch(path, {
    ...options,
    method,
    headers,
    body: body as BodyInit | null | undefined,
  });
}

export async function apiJSON<T = unknown>(
  path: string,
  options: APIRequestOptions = {},
): Promise<T> {
  if (demoMode) {
    const { demoResponse } = await import("./demoData");
    return demoResponse(path, options) as T;
  }
  const response = await apiFetch(path, options);
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    const code = isAPIProblemCode(body?.code) ? body.code : "unknown_error";
    throw new APIError(
      typeof body?.message === "string"
        ? body.message
        : "The request could not be completed.",
      response.status,
      code,
      response.headers.get("x-request-id") ?? "",
    );
  }
  return body as T;
}
import {
  isAPIProblemCode,
  type APIProblemCode,
} from "./api-contracts.generated";
