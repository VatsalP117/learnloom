import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

test("systemd permits the bounded multi-stage model run to finish", async () => {
  const service = await readFile(
    path.join(root, "deploy", "learnloom.service"),
    "utf8",
  );
  assert.match(service, /^Type=oneshot$/m);
  assert.match(service, /^TimeoutStartSec=infinity$/m);
});

test("Compose exposes the local dashboard on host loopback only", async () => {
  const compose = await readFile(path.join(root, "compose.yaml"), "utf8");
  assert.match(compose, /dashboard:/);
  assert.match(compose, /profiles:\n\s+- dashboard/);
  assert.match(compose, /"127\.0\.0\.1:3000:3000"/);
  assert.match(compose, /worker:/);
  assert.match(compose, /no-new-privileges:true/);
});

test("Docker image installs Clerk runtime dependencies and accepts the public build key", async () => {
  const dockerfile = await readFile(path.join(root, "Dockerfile"), "utf8");
  assert.match(dockerfile, /ARG VITE_CLERK_PUBLISHABLE_KEY/);
  assert.match(dockerfile, /npm ci --omit=dev/);
  assert.match(dockerfile, /package-lock\.json/);
});
