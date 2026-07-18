import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import path from "node:path";

export async function loadHistory(filePath = "data/history.json") {
  const absolutePath = path.resolve(filePath);
  try {
    const value = JSON.parse(await readFile(absolutePath, "utf8"));
    if (!Array.isArray(value)) {
      throw new Error("history root must be an array");
    }
    return value.filter((entry) => entry && typeof entry === "object");
  } catch (error) {
    if (error.code === "ENOENT") return [];
    throw new Error(`Could not read learning history at ${absolutePath}: ${error.message}`);
  }
}

export async function saveRun(result, options = {}) {
  const outputDirectory = path.resolve(options.outputDirectory ?? "output");
  const historyPath = path.resolve(options.historyPath ?? "data/history.json");
  const historyLimit = options.historyLimit ?? 100;
  await mkdir(outputDirectory, { recursive: true });
  await mkdir(path.dirname(historyPath), { recursive: true });

  const outputPath = path.join(outputDirectory, `${result.date}.md`);
  await atomicWrite(outputPath, result.markdown);

  const history = await loadHistory(historyPath);
  const withoutSameRun = history.filter(
    (entry) => entry.generatedAt !== result.historyEntry.generatedAt,
  );
  withoutSameRun.push(result.historyEntry);
  await atomicWrite(
    historyPath,
    `${JSON.stringify(withoutSameRun.slice(-historyLimit), null, 2)}\n`,
  );

  return { outputPath, historyPath };
}

async function atomicWrite(filePath, contents) {
  const temporaryPath = `${filePath}.${process.pid}.tmp`;
  await writeFile(temporaryPath, contents, "utf8");
  await rename(temporaryPath, filePath);
}

