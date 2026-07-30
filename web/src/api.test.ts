import { afterEach, describe, expect, it, vi } from "vitest";
import { apiJSON, configureAPI, setCSRFToken } from "./api";

describe("apiJSON", () => {
  it("attaches bearer and CSRF credentials to mutations", async () => {
    configureAPI(async () => "session-token");
    setCSRFToken("csrf-token");
    const fetchMock = vi.fn(async (_path, request) => {
      expect(request.method).toBe("POST");
      expect(request.headers.get("authorization")).toBe("Bearer session-token");
      expect(request.headers.get("x-csrf-token")).toBe("csrf-token");
      expect(request.headers.get("content-type")).toBe("application/json");
      expect(request.body).toBe('{"active":true}');
      return new Response('{"active":true}', {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiJSON("/api/newsletters/example/active", {
      method: "POST",
      body: { active: true },
    })).resolves.toEqual({ active: true });
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it("exposes safe API problem messages", async () => {
    configureAPI(async () => "session-token");
    vi.stubGlobal("fetch", vi.fn(async () =>
      new Response('{"message":"Too many requests."}', {
        status: 429,
        headers: { "content-type": "application/json" },
      })));

    await expect(apiJSON("/api/newsletters")).rejects.toThrow("Too many requests.");
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    setCSRFToken("");
  });

  it("preserves declared problem codes, status, and request ID", async () => {
    configureAPI(async () => "token");
    vi.stubGlobal("fetch", vi.fn(async () => new Response(
      JSON.stringify({ code: "quota_exceeded", message: "Try later." }),
      {
        status: 429,
        headers: {
          "content-type": "application/json",
          "x-request-id": "request-123",
        },
      },
    )));

    await expect(apiJSON("/api/test")).rejects.toMatchObject({
      name: "APIError",
      code: "quota_exceeded",
      status: 429,
      requestId: "request-123",
      message: "Try later.",
    });
  });

  it("does not trust undeclared server problem codes", async () => {
    configureAPI(async () => "token");
    vi.stubGlobal("fetch", vi.fn(async () => new Response(
      JSON.stringify({ code: "provider_stack_trace", message: "Safe message." }),
      { status: 500, headers: { "content-type": "application/json" } },
    )));

    await expect(apiJSON("/api/test")).rejects.toMatchObject({
      code: "unknown_error",
      status: 500,
    });
  });
});
