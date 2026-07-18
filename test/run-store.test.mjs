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

test("FileRunStore reclaims stale leases without letting the old owner release the new lock", async () => {
  const paths = await fixturePaths();
  let timestamp = Date.parse("2026-07-18T00:00:00.000Z");
  const options = {
    now: () => new Date(timestamp),
    staleLockMs: 100,
    heartbeatMs: 60_000,
  };
  const oldStore = new FileRunStore(paths, options);
  const releaseOld = await oldStore.acquire("default-2026-07-18");
  timestamp += 101;
  const newStore = new FileRunStore(paths, options);
  const releaseNew = await newStore.acquire("default-2026-07-18");
  await releaseOld();
  const contender = new FileRunStore(paths, options);
  await assert.rejects(
    contender.acquire("default-2026-07-18"),
    /already active/,
  );
  await releaseNew();
});

async function fixturePaths() {
  const root = await mkdtemp(path.join(os.tmpdir(), "learnloom-run-store-"));
  return {
    locksDirectory: path.join(root, "locks"),
    runsDirectory: path.join(root, "runs"),
  };
}
