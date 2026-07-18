import assert from "node:assert/strict";
import { mkdtemp, mkdir, rename, writeFile } from "node:fs/promises";
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

test("FileRunStore never steals an old lock automatically", async () => {
  const paths = await fixturePaths();
  let timestamp = Date.parse("2026-07-18T00:00:00.000Z");
  const oldStore = new FileRunStore(paths, {
    now: () => new Date(timestamp),
  });
  const releaseOld = await oldStore.acquire("default-2026-07-18");
  timestamp += 24 * 60 * 60 * 1000;
  const newStore = new FileRunStore(paths, {
    now: () => new Date(timestamp),
  });
  await assert.rejects(
    newStore.acquire("default-2026-07-18"),
    /already active/,
  );
  await releaseOld();
});

test("FileRunStore release restores a lock owned by another process", async () => {
  const paths = await fixturePaths();
  const runId = "default-2026-07-18";
  const lockPath = path.join(paths.locksDirectory, `${runId}.lock`);
  const oldStore = new FileRunStore(paths);
  const releaseOld = await oldStore.acquire(runId);
  await rename(lockPath, `${lockPath}.operator-stale-copy`);
  const newStore = new FileRunStore(paths);
  const releaseNew = await newStore.acquire("default-2026-07-18");
  await releaseOld();
  const contender = new FileRunStore(paths);
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
