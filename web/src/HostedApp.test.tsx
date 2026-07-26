import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const authState = vi.hoisted(() => ({ signedIn: false }));

vi.mock("@clerk/react", () => ({
  AuthenticateWithRedirectCallback: () => null,
  ClerkFailed: () => null,
  ClerkLoaded: ({ children }: { children: ReactNode }) => children,
  ClerkLoading: () => null,
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
});
