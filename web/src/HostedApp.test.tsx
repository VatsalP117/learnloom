import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const authState = vi.hoisted(() => ({ loaded: true, signedIn: false }));

vi.mock("@clerk/react", () => ({
  AuthenticateWithRedirectCallback: () => null,
  ClerkFailed: () => null,
  ClerkLoaded: ({ children }: { children: ReactNode }) =>
    authState.loaded ? children : null,
  ClerkLoading: ({ children }: { children: ReactNode }) =>
    authState.loaded ? null : children,
  RedirectToSignIn: () => null,
  Show: ({ children, when }: { children: ReactNode; when: string }) => {
    const visible =
      (when === "signed-in" && authState.signedIn) ||
      (when === "signed-out" && !authState.signedIn);
    return visible ? children : null;
  },
  useAuth: () => ({ getToken: vi.fn() }),
  useClerk: () => ({ signOut: vi.fn() }),
  useSignIn: () => ({
    errors: { fields: {}, global: null, raw: null },
    fetchStatus: "idle",
    signIn: {},
  }),
  useSignUp: () => ({
    errors: { fields: {}, global: null, raw: null },
    fetchStatus: "idle",
    signUp: {},
  }),
}));

import HostedApp from "./HostedApp";

describe("HostedApp authentication routes", () => {
  beforeEach(() => {
    authState.loaded = true;
    authState.signedIn = false;
    Object.defineProperty(globalThis, "window", {
      configurable: true,
      value: {
        location: {
          pathname: "/sign-in",
          replace: vi.fn(),
        },
      },
    });
  });

  it("shows the sign-in form to signed-out users", () => {
    const markup = renderToStaticMarkup(<HostedApp />);

    expect(markup).toContain("Continue with Google");
    expect(markup).not.toContain("already signed in");
  });

  it("does not start another sign-in flow for signed-in users", () => {
    authState.signedIn = true;

    const markup = renderToStaticMarkup(<HostedApp />);

    expect(markup).toContain("already signed in");
    expect(markup).not.toContain("Continue with Google");
  });

  it("does not show the sign-in shell while an existing session is loading", () => {
    authState.loaded = false;
    Object.defineProperty(globalThis, "window", {
      configurable: true,
      value: {
        location: {
          pathname: "/newsletters/stream-1",
          replace: vi.fn(),
        },
      },
    });

    const markup = renderToStaticMarkup(<HostedApp />);

    expect(markup).toContain('class="calm-loader calm-loader-screen"');
    expect(markup).not.toContain('class="custom-auth-shell"');
  });

  it("keeps the same calm screen while the signed-in profile is loading", () => {
    authState.signedIn = true;
    Object.defineProperty(globalThis, "window", {
      configurable: true,
      value: {
        location: {
          pathname: "/newsletters/stream-1",
          replace: vi.fn(),
        },
      },
    });

    const markup = renderToStaticMarkup(<HostedApp />);

    expect(markup).toContain("Preparing your workspace");
    expect(markup).toContain('class="calm-loader calm-loader-screen"');
    expect(markup).not.toContain('class="custom-auth-shell"');
  });
});
