import { randomUUID } from "node:crypto";
import { mkdir, open, readFile, rename, rm } from "node:fs/promises";
import path from "node:path";
import { loadJson, writeJsonAtomic } from "./state.mjs";

const DEFAULT_STALE_LOCK_MS = 5 * 60 * 1000;
const DEFAULT_HEARTBEAT_MS = 30 * 1000;

export class FileRunStore {
  constructor(paths, options = {}) {
    this.paths = paths;
    this.now = options.now ?? (() => new Date());
    this.staleLockMs = options.staleLockMs ?? DEFAULT_STALE_LOCK_MS;
    this.heartbeatMs = options.heartbeatMs ?? DEFAULT_HEARTBEAT_MS;
  }

  async load(runId) {
    const record = await loadJson(this.recordPath(runId), "Daily Run record");
    if (record === null) return null;
    if (record.version !== 1 || record.runId !== runId) {
      throw new Error(`Invalid Daily Run record for ${runId}.`);
    }
    return record;
  }

  async save(record) {
    record.updatedAt = this.now().toISOString();
    await writeJsonAtomic(this.recordPath(record.runId), record);
  }

  async acquire(runId) {
    await mkdir(this.paths.locksDirectory, { recursive: true, mode: 0o700 });
    const lockPath = path.join(this.paths.locksDirectory, `${runId}.lock`);
    const token = randomUUID();
    let lease;
    try {
      lease = await createLease(lockPath, runId, token, this.now());
    } catch (error) {
      if (error.code !== "EEXIST") throw error;
      lease = await this.reclaim(lockPath, runId, token);
    }

    let released = false;
    let refreshPromise = Promise.resolve();
    const heartbeat = setInterval(() => {
      refreshPromise = refreshPromise
        .then(() => this.refresh(lockPath, lease))
        .catch(() => {});
    }, this.heartbeatMs);
    heartbeat.unref?.();

    return async () => {
      if (released) return;
      released = true;
      clearInterval(heartbeat);
      await refreshPromise;
      const current = await readLease(lockPath);
      if (current?.token === token) {
        await rm(lockPath, { force: true });
      }
    };
  }

  async refresh(lockPath, lease) {
    const current = await readLease(lockPath);
    if (current?.token !== lease.token) return;
    lease.heartbeatAt = this.now().toISOString();
    await writeJsonAtomic(lockPath, lease);
  }

  async reclaim(lockPath, runId, token) {
    const observed = await readLease(lockPath);
    if (!observed || !isStale(observed, this.now(), this.staleLockMs)) {
      throw new Error(`Daily Run ${runId} is already active.`);
    }

    const candidatePath = `${lockPath}.reclaim-${token}`;
    try {
      await rename(lockPath, candidatePath);
    } catch (error) {
      if (error.code === "ENOENT") {
        try {
          return await createLease(lockPath, runId, token, this.now());
        } catch (createError) {
          if (createError.code === "EEXIST") {
            throw new Error(`Daily Run ${runId} is already active.`);
          }
          throw createError;
        }
      }
      throw error;
    }

    const moved = await readLease(candidatePath);
    if (
      moved?.token !== observed.token ||
      moved?.heartbeatAt !== observed.heartbeatAt
    ) {
      try {
        await rename(candidatePath, lockPath);
      } catch (error) {
        if (error.code !== "EEXIST") throw error;
        await rm(candidatePath, { force: true });
      }
      throw new Error(`Daily Run ${runId} is already active.`);
    }

    await rm(candidatePath, { force: true });
    try {
      return await createLease(lockPath, runId, token, this.now());
    } catch (error) {
      if (error.code === "EEXIST") {
        throw new Error(`Daily Run ${runId} is already active.`);
      }
      throw error;
    }
  }

  recordPath(runId) {
    return path.join(this.paths.runsDirectory, `${runId}.json`);
  }
}

async function createLease(lockPath, runId, token, now) {
  const timestamp = now.toISOString();
  const lease = {
    version: 1,
    runId,
    token,
    pid: process.pid,
    acquiredAt: timestamp,
    heartbeatAt: timestamp,
  };
  const handle = await open(lockPath, "wx", 0o600);
  try {
    await handle.writeFile(`${JSON.stringify(lease)}\n`, "utf8");
  } finally {
    await handle.close();
  }
  return lease;
}

async function readLease(lockPath) {
  try {
    return JSON.parse(await readFile(lockPath, "utf8"));
  } catch (error) {
    if (error.code === "ENOENT") return null;
    throw error;
  }
}

function isStale(lease, now, staleLockMs) {
  const heartbeat = Date.parse(lease.heartbeatAt);
  return Number.isFinite(heartbeat) && now.valueOf() - heartbeat > staleLockMs;
}
