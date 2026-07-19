import { createClerkClient } from "@clerk/backend";

export function resolveClerkConfig(options = {}) {
  const environment = options.env ?? process.env;
  const publishableKey = requireEnvironmentValue(
    environment,
    "CLERK_PUBLISHABLE_KEY",
  );
  const secretKey = requireEnvironmentValue(environment, "CLERK_SECRET_KEY");
  const jwtKey = optionalEnvironmentValue(environment.CLERK_JWT_KEY);
  return Object.freeze({ publishableKey, secretKey, jwtKey });
}

export function createClerkAuthenticator(config) {
  const clerk = createClerkClient(config);
  return {
    async authenticate(nodeRequest, expectedOrigin) {
      const request = toFetchRequest(nodeRequest, expectedOrigin);
      const state = await clerk.authenticateRequest(request, {
        acceptsToken: "session_token",
        authorizedParties: [expectedOrigin],
      });
      if (state.status === "handshake") {
        return { status: "handshake", headers: state.headers };
      }
      if (!state.isAuthenticated) {
        return {
          status: "unauthenticated",
          headers: state.headers,
          reason: state.reason,
        };
      }
      const auth = state.toAuth();
      if (!auth.userId || !auth.sessionId) {
        return { status: "unauthenticated", headers: state.headers };
      }
      return {
        status: "authenticated",
        clerkUserId: auth.userId,
        sessionId: auth.sessionId,
        headers: state.headers,
      };
    },
  };
}

function toFetchRequest(request, expectedOrigin) {
  const headers = new Headers();
  for (const [name, value] of Object.entries(request.headers)) {
    if (Array.isArray(value)) {
      for (const item of value) headers.append(name, item);
    } else if (value !== undefined) {
      headers.set(name, value);
    }
  }
  return new Request(new URL(request.url ?? "/", expectedOrigin), {
    method: request.method ?? "GET",
    headers,
  });
}

function requireEnvironmentValue(environment, name) {
  const value = optionalEnvironmentValue(environment[name]);
  if (!value) throw new Error(`${name} is required in hosted mode.`);
  return value;
}

function optionalEnvironmentValue(value) {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}
