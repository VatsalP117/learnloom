import { randomUUID } from "node:crypto";
import { mkdir, open, readFile, rename, rm } from "node:fs/promises";
import path from "node:path";
import { loadJson, writeJsonAtomic } from "./state.mjs";

export class FileRunStore {
  constructor(paths, options = {}) {
    this.paths = paths;
    this.now = options.now ?? (() => new Date());
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
    try {
      await createLease(lockPath, runId, token, this.now());
    } catch (error) {
      if (error.code !== "EEXIST") throw error;
      throw new Error(
        `Daily Run ${runId} is already active. If the process crashed, confirm no run is active before removing ${lockPath}.`,
      );
    }

    let released = false;
    return async () => {
      if (released) return;
      released = true;
      const candidatePath = `${lockPath}.release-${token}`;
      try {
        await rename(lockPath, candidatePath);
      } catch (error) {
        if (error.code === "ENOENT") return;
        throw error;
      }
      const moved = await readLease(candidatePath);
      if (moved?.token === token) {
        await rm(candidatePath, { force: true });
        return;
      }
      try {
        await rename(candidatePath, lockPath);
      } catch (error) {
        if (error.code === "EEXIST") {
          throw new Error(
            `Daily Run ${runId} lock ownership changed during release; preserved ${candidatePath}.`,
          );
        }
        throw error;
      }
    };
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
  };
  const handle = await open(lockPath, "wx", 0o600);
  try {
    await handle.writeFile(`${JSON.stringify(lease)}\n`, "utf8");
  } catch (error) {
    await rm(lockPath, { force: true });
    throw error;
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
