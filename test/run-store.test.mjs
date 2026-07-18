import assert from "node:assert/strict";
import { mkdtemp, mkdir, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { FileRunStore } from "../src/run-store.mjs";

test("FileRunStore prevents overlapping Daily Runs and releases its lock", async () => {
  const paths = await fixturePaths();
  const store = new FileRunStore(paths);
  const release = await store.acquire("default-2026-07-18");
  await assert.rejects(
    store.acquire("default-2026-07-18"),
    /already active/,
  );
  await release();
  const releaseAgain = await store.acquire("default-2026-07-18");
  await releaseAgain();
});

test("FileRunStore rejects a corrupted or mismatched record", async () => {
  const paths = await fixturePaths();
  await mkdir(paths.runsDirectory, { recursive: true });
  await writeFile(
    path.join(paths.runsDirectory, "default-2026-07-18.json"),
    JSON.stringify({ version: 1, runId: "different-run" }),
  );
  const store = new FileRunStore(paths);
  await assert.rejects(store.load("default-2026-07-18"), /Invalid Daily Run record/);
});

async function fixturePaths() {
  const root = await mkdtemp(path.join(os.tmpdir(), "learnloom-run-store-"));
  return {
    locksDirectory: path.join(root, "locks"),
    runsDirectory: path.join(root, "runs"),
  };
}
