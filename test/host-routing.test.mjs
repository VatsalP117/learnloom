import assert from "node:assert/strict";
import test from "node:test";
import {
  isPotentialSiteUsername,
  resolveDeploymentConfig,
  resolveRequestHost,
} from "../src/host-routing.mjs";

test("deployment configuration defaults to local mode", () => {
  assert.deepEqual(resolveDeploymentConfig({ env: {} }), { mode: "local" });
  assert.throws(
    () =>
      resolveDeploymentConfig({
        env: { LEARNLOOM_DEPLOYMENT_MODE: "production" },
      }),
    /either "local" or "hosted"/,
  );
});

test("hosted deployment configuration requires one exact HTTPS app origin", () => {
  const deployment = resolveDeploymentConfig({
    env: {
      LEARNLOOM_DEPLOYMENT_MODE: "hosted",
      LEARNLOOM_ROOT_DOMAIN: "LearnLoom.Blog",
      LEARNLOOM_APP_ORIGIN: "https://app.learnloom.blog",
    },
  });
  assert.deepEqual(deployment, {
    mode: "hosted",
    rootDomain: "learnloom.blog",
    appHostname: "app.learnloom.blog",
    appOrigin: "https://app.learnloom.blog",
    apexOrigin: "https://learnloom.blog",
    clerkFrontendApiOrigin: "https://clerk.learnloom.blog",
  });

  for (const rootDomain of [
    "",
    "localhost",
    "127.0.0.1",
    "*.learnloom.blog",
    "léarn.blog",
  ]) {
    assert.throws(
      () =>
        resolveDeploymentConfig({
          env: {
            LEARNLOOM_DEPLOYMENT_MODE: "hosted",
            LEARNLOOM_ROOT_DOMAIN: rootDomain,
            LEARNLOOM_APP_ORIGIN: "https://app.learnloom.blog",
          },
        }),
      /LEARNLOOM_ROOT_DOMAIN/,
    );
  }
  for (const appOrigin of [
    "",
    "http://app.learnloom.blog",
    "https://admin.learnloom.blog",
    "https://app.learnloom.blog:8443",
    "https://app.learnloom.blog/path",
  ]) {
    assert.throws(
      () =>
        resolveDeploymentConfig({
          env: {
            LEARNLOOM_DEPLOYMENT_MODE: "hosted",
            LEARNLOOM_ROOT_DOMAIN: "learnloom.blog",
            LEARNLOOM_APP_ORIGIN: appOrigin,
          },
        }),
      /LEARNLOOM_APP_ORIGIN/,
    );
  }
});

test("local host resolution preserves an exact allowlist and parses ports", () => {
  const local = { mode: "local" };
  assert.deepEqual(
    resolveRequestHost("localhost:3000", local),
    { kind: "local", hostname: "localhost" },
  );
  assert.deepEqual(
    resolveRequestHost("[::1]:3000", local),
    { kind: "local", hostname: "[::1]" },
  );
  assert.deepEqual(
    resolveRequestHost("app.lvh.me:3000", local, {
      allowedHosts: ["app.lvh.me"],
    }),
    { kind: "local", hostname: "app.lvh.me" },
  );
  for (const host of [
    "attacker.example",
    "user@localhost",
    "localhost/path",
    "localhost:not-a-port",
    "",
    null,
  ]) {
    assert.equal(resolveRequestHost(host, local).kind, "rejected");
  }
});

test("hosted requests classify apex, app, www, and one username label", () => {
  const deployment = resolveDeploymentConfig({
    env: {
      LEARNLOOM_DEPLOYMENT_MODE: "hosted",
      LEARNLOOM_ROOT_DOMAIN: "learnloom.blog",
      LEARNLOOM_APP_ORIGIN: "https://app.learnloom.blog",
    },
  });
  assert.equal(
    resolveRequestHost("learnloom.blog", deployment).kind,
    "apex",
  );
  assert.equal(
    resolveRequestHost("www.learnloom.blog", deployment).kind,
    "www",
  );
  assert.equal(
    resolveRequestHost("app.learnloom.blog", deployment).kind,
    "app",
  );
  assert.deepEqual(
    resolveRequestHost("vatsal.learnloom.blog:443", deployment),
    {
      kind: "site",
      hostname: "vatsal.learnloom.blog",
      username: "vatsal",
    },
  );

  for (const host of [
    "clerk.learnloom.blog",
    "admin.learnloom.blog",
    "a.b.learnloom.blog",
    "-bad.learnloom.blog",
    "ab.learnloom.blog",
    "outside.example",
  ]) {
    assert.equal(resolveRequestHost(host, deployment).kind, "rejected");
  }
});

test("potential site usernames use the claim grammar and reserved list", () => {
  for (const username of ["vatsal", "alice-2", "abc"]) {
    assert.equal(isPotentialSiteUsername(username), true);
  }
  for (const username of [
    "ab",
    "2alice",
    "alice-",
    "alice--two",
    "Alice",
    "app",
    "clerk",
    "a".repeat(31),
  ]) {
    assert.equal(isPotentialSiteUsername(username), false);
  }
});
