import assert from "node:assert/strict";
import { mkdtemp, readFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { loadHistory, saveRun } from "../src/state.mjs";

test("saveRun writes a dossier and appends bounded history", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learning-engine-"));
  const historyPath = path.join(directory, "data", "history.json");
  const outputDirectory = path.join(directory, "output");
  const base = {
    date: "2026-07-18",
    markdown: "# Lesson\n",
    historyEntry: {
      date: "2026-07-18",
      generatedAt: "2026-07-18T00:00:00.000Z",
      lessonSummary: "First",
    },
  };
  await saveRun(base, { historyPath, outputDirectory, historyLimit: 1 });
  await saveRun(
    {
      ...base,
      date: "2026-07-19",
      historyEntry: {
        ...base.historyEntry,
        date: "2026-07-19",
        generatedAt: "2026-07-19T00:00:00.000Z",
        lessonSummary: "Second",
      },
    },
    { historyPath, outputDirectory, historyLimit: 1 },
  );

  assert.equal(await readFile(path.join(outputDirectory, "2026-07-18.md"), "utf8"), "# Lesson\n");
  const history = await loadHistory(historyPath);
  assert.equal(history.length, 1);
  assert.equal(history[0].lessonSummary, "Second");
});

