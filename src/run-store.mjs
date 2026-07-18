import { mkdir, open, rm, stat } from "node:fs/promises";
import path from "node:path";
import { loadJson, writeJsonAtomic } from "./state.mjs";

const STALE_LOCK_MS = 2 * 60 * 60 * 1000;

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
    let handle;
    try {
      handle = await open(lockPath, "wx", 0o600);
    } catch (error) {
      if (error.code !== "EEXIST") throw error;
      const lockStat = await stat(lockPath);
      if (this.now().valueOf() - lockStat.mtimeMs <= STALE_LOCK_MS) {
        throw new Error(`Daily Run ${runId} is already active.`);
      }
      await rm(lockPath, { force: true });
      handle = await open(lockPath, "wx", 0o600);
    }
    await handle.writeFile(
      JSON.stringify({ runId, pid: process.pid, acquiredAt: this.now().toISOString() }),
      "utf8",
    );
    await handle.close();
    let released = false;
    return async () => {
      if (released) return;
      released = true;
      await rm(lockPath, { force: true });
    };
  }

  recordPath(runId) {
    return path.join(this.paths.runsDirectory, `${runId}.json`);
  }
}

