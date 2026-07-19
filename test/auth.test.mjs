import assert from "node:assert/strict";
import test from "node:test";
import { resolveClerkConfig } from "../src/auth.mjs";

test("hosted Clerk configuration requires server keys", () => {
  assert.deepEqual(
    resolveClerkConfig({
      env: {
        CLERK_PUBLISHABLE_KEY: "pk_test_example",
        CLERK_SECRET_KEY: "sk_test_example",
        CLERK_JWT_KEY: "public-key",
      },
    }),
    {
      publishableKey: "pk_test_example",
      secretKey: "sk_test_example",
      jwtKey: "public-key",
    },
  );
  assert.throws(() => resolveClerkConfig({ env: {} }), /CLERK_PUBLISHABLE_KEY/);
  assert.throws(
    () =>
      resolveClerkConfig({
        env: { CLERK_PUBLISHABLE_KEY: "pk_test_example" },
      }),
    /CLERK_SECRET_KEY/,
  );
});
