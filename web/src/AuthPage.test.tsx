import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

vi.mock("@clerk/react", () => ({
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

import AuthPage from "./AuthPage";

describe("AuthPage", () => {
  it("enables the sign-in submit button when Clerk is idle", () => {
    const markup = renderToStaticMarkup(<AuthPage mode="sign-in" />);

    expect(markup).toContain("Sign in");
    expect(markup).toContain("Continue with Google");
    expect(markup).not.toContain("Just a moment");
    expect(markup).not.toContain('class="auth-submit" type="submit" disabled');
  });

  it("enables the sign-up submit button when Clerk is idle", () => {
    const markup = renderToStaticMarkup(<AuthPage mode="sign-up" />);

    expect(markup).toContain("Create account");
    expect(markup).toContain("Continue with Google");
    expect(markup).toContain('href="/terms"');
    expect(markup).toContain('href="/privacy"');
    expect(markup).not.toContain("Just a moment");
    expect(markup).not.toContain('class="auth-submit" type="submit" disabled');
  });
});
