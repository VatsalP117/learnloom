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

test("Compose exposes the unauthenticated dashboard on host loopback only", async () => {
  const compose = await readFile(path.join(root, "compose.yaml"), "utf8");
  assert.match(compose, /dashboard:/);
  assert.match(compose, /profiles:\n\s+- dashboard/);
  assert.match(compose, /"127\.0\.0\.1:3000:3000"/);
  assert.match(compose, /worker:/);
  assert.match(compose, /no-new-privileges:true/);
});
