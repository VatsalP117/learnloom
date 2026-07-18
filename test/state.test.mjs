import assert from "node:assert/strict";
import { mkdtemp, readFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { loadHistory, saveRun } from "../src/state.mjs";

test("saveRun writes a dossier and replaces same-day bounded history", async () => {
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
      historyEntry: {
        ...base.historyEntry,
        generatedAt: "2026-07-18T01:00:00.000Z",
        lessonSummary: "Second",
      },
    },
    { historyPath, outputDirectory, historyLimit: 1 },
  );

  assert.equal(await readFile(path.join(outputDirectory, "2026-07-18.md"), "utf8"), "# Lesson\n");
  const history = await loadHistory(historyPath);
  assert.equal(history.length, 1);
  assert.equal(history[0].lessonSummary, "Second");
  assert.equal(history[0].date, "2026-07-18");
});

test("saveRun clears Learning History when historyLimit is zero", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learning-engine-zero-"));
  const historyPath = path.join(directory, "data", "history.json");
  await saveRun(
    {
      date: "2026-07-18",
      markdown: "# Lesson\n",
      historyEntry: {
        date: "2026-07-18",
        generatedAt: "2026-07-18T00:00:00.000Z",
        lessonSummary: "Not retained",
      },
    },
    {
      historyPath,
      outputDirectory: path.join(directory, "output"),
      historyLimit: 0,
    },
  );
  assert.deepEqual(await loadHistory(historyPath), []);
});
