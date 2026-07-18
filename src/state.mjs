import { randomUUID } from "node:crypto";
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
  await mkdir(outputDirectory, { recursive: true, mode: 0o700 });
  await mkdir(path.dirname(historyPath), { recursive: true, mode: 0o700 });

  const generationId = options.generationId;
  const fileStem = generationId
    ? `${result.date}-${safeFilePart(generationId)}`
    : result.date;
  const outputPath = path.join(outputDirectory, `${fileStem}.md`);
  await atomicWrite(outputPath, result.markdown);
  const dossierPath = path.join(outputDirectory, `${fileStem}.json`);
  if (result.dossier) {
    await atomicWrite(dossierPath, `${JSON.stringify(result.dossier, null, 2)}\n`);
  }

  const history = await loadHistory(historyPath);
  const withoutSameRun = history.filter(
    (entry) =>
      entry.generatedAt !== result.historyEntry.generatedAt &&
      entry.date !== result.historyEntry.date,
  );
  withoutSameRun.push(result.historyEntry);
  const retainedHistory =
    historyLimit === 0 ? [] : withoutSameRun.slice(-historyLimit);
  await atomicWrite(historyPath, `${JSON.stringify(retainedHistory, null, 2)}\n`);

  return {
    outputPath,
    dossierPath: result.dossier ? dossierPath : null,
    historyPath,
  };
}

function safeFilePart(value) {
  if (typeof value !== "string" || !/^[a-zA-Z0-9_-]+$/.test(value)) {
    throw new Error("generationId must contain only letters, numbers, underscores, or hyphens.");
  }
  return value;
}

export async function loadJson(filePath, description = "JSON file") {
  try {
    return JSON.parse(await readFile(filePath, "utf8"));
  } catch (error) {
    if (error.code === "ENOENT") return null;
    throw new Error(`Could not read ${description} at ${filePath}: ${error.message}`);
  }
}

export async function writeJsonAtomic(filePath, value) {
  await mkdir(path.dirname(filePath), { recursive: true, mode: 0o700 });
  await atomicWrite(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

async function atomicWrite(filePath, contents) {
  const temporaryPath = `${filePath}.${process.pid}.${randomUUID()}.tmp`;
  await writeFile(temporaryPath, contents, { encoding: "utf8", mode: 0o600 });
  await rename(temporaryPath, filePath);
}
