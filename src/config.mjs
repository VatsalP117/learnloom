import { readFile } from "node:fs/promises";
import path from "node:path";

const DEFAULT_LIMITS = Object.freeze({
  maxItems: 18,
  maxItemCharacters: 1800,
  maxIntermediateCharacters: 24000,
  historyEntries: 14,
});

export async function loadConfig(filePath = "config.json") {
  const absolutePath = path.resolve(filePath);
  let raw;
  try {
    raw = await readFile(absolutePath, "utf8");
  } catch (error) {
    if (error.code === "ENOENT") {
      throw new Error(
        `Configuration not found at ${absolutePath}. Run "npm run demo" or "node bin/learn.mjs init".`,
      );
    }
    throw error;
  }

  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    throw new Error(`Invalid JSON in ${absolutePath}: ${error.message}`);
  }

  return validateConfig(parsed, absolutePath);
}

export function validateConfig(value, configPath = "config.json") {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error("Configuration must be a JSON object.");
  }

  const interests = requireStringArray(value.interests, "interests");
  if (interests.length === 0) {
    throw new Error("Configuration must include at least one interest.");
  }

  if (!Array.isArray(value.sources) || value.sources.length === 0) {
    throw new Error("Configuration must include at least one source.");
  }

  const sources = value.sources.map((source, index) => {
    if (!source || typeof source !== "object") {
      throw new Error(`sources[${index}] must be an object.`);
    }
    const name = requireString(source.name, `sources[${index}].name`);
    const url = requireString(source.url, `sources[${index}].url`);
    let parsedUrl;
    try {
      parsedUrl = new URL(url);
    } catch {
      throw new Error(`sources[${index}].url must be a valid URL.`);
    }
    if (!["http:", "https:"].includes(parsedUrl.protocol)) {
      throw new Error(`sources[${index}].url must use HTTP or HTTPS.`);
    }
    return {
      name,
      url: parsedUrl.toString(),
      limit: boundedInteger(source.limit ?? 10, 1, 50, `sources[${index}].limit`),
    };
  });

  const provider = value.provider ?? {};
  const providerKind = provider.kind ?? "commandcode";
  if (!["commandcode", "demo"].includes(providerKind)) {
    throw new Error('provider.kind must be either "commandcode" or "demo".');
  }

  const timeZone = value.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
  try {
    new Intl.DateTimeFormat("en", { timeZone }).format();
  } catch {
    throw new Error(`Invalid timeZone: ${timeZone}`);
  }

  const learner = value.learner ?? {};
  const limits = value.limits ?? {};

  return {
    configPath: path.resolve(configPath),
    timeZone,
    interests,
    learner: {
      level: learner.level ?? "curious generalist",
      goal: learner.goal ?? "develop durable, practical understanding",
      lessonMinutes: boundedInteger(learner.lessonMinutes ?? 15, 5, 90, "learner.lessonMinutes"),
    },
    sources,
    provider: {
      kind: providerKind,
      executable: provider.executable ?? "cmd",
      model: provider.model ?? "deepseek-v4-pro",
      timeoutSeconds: boundedInteger(
        provider.timeoutSeconds ?? 600,
        30,
        1800,
        "provider.timeoutSeconds",
      ),
    },
    limits: {
      maxItems: boundedInteger(
        limits.maxItems ?? DEFAULT_LIMITS.maxItems,
        1,
        100,
        "limits.maxItems",
      ),
      maxItemCharacters: boundedInteger(
        limits.maxItemCharacters ?? DEFAULT_LIMITS.maxItemCharacters,
        200,
        10000,
        "limits.maxItemCharacters",
      ),
      maxIntermediateCharacters: boundedInteger(
        limits.maxIntermediateCharacters ?? DEFAULT_LIMITS.maxIntermediateCharacters,
        2000,
        100000,
        "limits.maxIntermediateCharacters",
      ),
      historyEntries: boundedInteger(
        limits.historyEntries ?? DEFAULT_LIMITS.historyEntries,
        0,
        100,
        "limits.historyEntries",
      ),
    },
  };
}

function requireString(value, field) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} must be a non-empty string.`);
  }
  return value.trim();
}

function requireStringArray(value, field) {
  if (!Array.isArray(value) || value.some((item) => typeof item !== "string")) {
    throw new Error(`${field} must be an array of strings.`);
  }
  return value.map((item) => item.trim()).filter(Boolean);
}

function boundedInteger(value, minimum, maximum, field) {
  if (!Number.isInteger(value) || value < minimum || value > maximum) {
    throw new Error(`${field} must be an integer from ${minimum} to ${maximum}.`);
  }
  return value;
}

