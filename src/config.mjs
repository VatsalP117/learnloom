import { readFile } from "node:fs/promises";
import path from "node:path";

const DEFAULT_LIMITS = Object.freeze({
  maxItems: 18,
  maxItemCharacters: 1800,
  maxIntermediateCharacters: 24000,
  historyEntries: 14,
});
const PROVIDER_KINDS = new Set(["commandcode", "openai-compatible", "demo"]);
const ENVIRONMENT_NAME = /^[A-Z_][A-Z0-9_]*$/;
const PROFILE_ID = /^[a-z0-9][a-z0-9_-]{0,63}$/;

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
  if (!PROVIDER_KINDS.has(providerKind)) {
    throw new Error(
      'provider.kind must be "commandcode", "openai-compatible", or "demo".',
    );
  }

  const timeZone = value.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
  try {
    new Intl.DateTimeFormat("en", { timeZone }).format();
  } catch {
    throw new Error(`Invalid timeZone: ${timeZone}`);
  }

  const learner = value.learner ?? {};
  const limits = value.limits ?? {};
  const profileId = value.profileId ?? "default";
  if (typeof profileId !== "string" || !PROFILE_ID.test(profileId)) {
    throw new Error(
      "profileId must start with a lowercase letter or number and contain only lowercase letters, numbers, underscores, or hyphens (maximum 64 characters).",
    );
  }
  const storage = value.storage ?? {};
  const content = value.content ?? {};
  if (!content || typeof content !== "object" || Array.isArray(content)) {
    throw new Error("content must be an object.");
  }
  const aiExplorationEnabled = content.aiExplorationEnabled ?? false;
  if (typeof aiExplorationEnabled !== "boolean") {
    throw new Error("content.aiExplorationEnabled must be a boolean.");
  }
  const deliveries = validateDeliveries(value.deliveries ?? []);
  const providerConfig = {
    kind: providerKind,
    executable: requireString(provider.executable ?? "cmd", "provider.executable"),
    model: requireString(provider.model ?? "deepseek-v4-pro", "provider.model"),
    timeoutSeconds: boundedInteger(
      provider.timeoutSeconds ?? 600,
      5,
      1800,
      "provider.timeoutSeconds",
    ),
    retries: boundedInteger(provider.retries ?? 2, 0, 5, "provider.retries"),
  };
  if (providerKind === "openai-compatible") {
    const allowInsecureHttp = provider.allowInsecureHttp ?? false;
    if (typeof allowInsecureHttp !== "boolean") {
      throw new Error("provider.allowInsecureHttp must be a boolean.");
    }
    providerConfig.baseUrl = requireProviderUrl(
      provider.baseUrl ?? "https://api.deepseek.com",
      "provider.baseUrl",
      allowInsecureHttp,
    ).replace(/\/+$/, "");
    providerConfig.allowInsecureHttp = allowInsecureHttp;
    providerConfig.apiKeyEnv = environmentName(
      provider.apiKeyEnv ?? "DEEPSEEK_API_KEY",
      "provider.apiKeyEnv",
    );
    providerConfig.maxTokens = boundedInteger(
      provider.maxTokens ?? 8192,
      256,
      65536,
      "provider.maxTokens",
    );
  }

  return {
    configPath: path.resolve(configPath),
    profileId,
    timeZone,
    interests,
    learner: {
      level: learner.level ?? "curious generalist",
      goal: learner.goal ?? "develop durable, practical understanding",
      lessonMinutes: boundedInteger(learner.lessonMinutes ?? 15, 5, 90, "learner.lessonMinutes"),
    },
    sources,
    provider: providerConfig,
    deliveries,
    content: {
      aiExplorationEnabled,
      maxArticleBytes: boundedInteger(
        content.maxArticleBytes ?? 512 * 1024,
        16 * 1024,
        2 * 1024 * 1024,
        "content.maxArticleBytes",
      ),
      maxArticleCharacters: boundedInteger(
        content.maxArticleCharacters ?? 16_000,
        1_000,
        50_000,
        "content.maxArticleCharacters",
      ),
    },
    storage: {
      dataDirectory: optionalPath(storage.dataDirectory ?? "data", "storage.dataDirectory"),
      outputDirectory: optionalPath(
        storage.outputDirectory ?? "output",
        "storage.outputDirectory",
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

function validateDeliveries(value) {
  if (!Array.isArray(value)) {
    throw new Error("deliveries must be an array.");
  }
  const deliveries = value.map((delivery, index) => {
    const field = `deliveries[${index}]`;
    if (!delivery || typeof delivery !== "object" || Array.isArray(delivery)) {
      throw new Error(`${field} must be an object.`);
    }
    if (delivery.kind !== "resend") {
      throw new Error(`${field}.kind must be "resend".`);
    }
    const enabled = delivery.enabled ?? true;
    if (typeof enabled !== "boolean") {
      throw new Error(`${field}.enabled must be a boolean.`);
    }
    const result = {
      id: delivery.id ?? `resend-${index + 1}`,
      kind: "resend",
      enabled,
      apiKeyEnv: environmentName(
        delivery.apiKeyEnv ?? "RESEND_API_KEY",
        `${field}.apiKeyEnv`,
      ),
      from: requireString(delivery.from, `${field}.from`),
      to: normalizeRecipients(delivery.to, `${field}.to`),
      subjectPrefix: requireString(
        delivery.subjectPrefix ?? "Learnloom",
        `${field}.subjectPrefix`,
      ),
    };
    if (!/^[a-z0-9][a-z0-9_-]{0,63}$/.test(result.id)) {
      throw new Error(`${field}.id must be a lowercase slug.`);
    }
    return result;
  });
  const ids = new Set();
  for (const delivery of deliveries) {
    if (ids.has(delivery.id)) {
      throw new Error(`deliveries contains duplicate id "${delivery.id}".`);
    }
    ids.add(delivery.id);
  }
  return deliveries;
}

function normalizeRecipients(value, field) {
  const recipients = typeof value === "string" ? [value] : value;
  if (
    !Array.isArray(recipients) ||
    recipients.length === 0 ||
    recipients.some((recipient) => typeof recipient !== "string" || !recipient.includes("@"))
  ) {
    throw new Error(`${field} must be an email address or a non-empty array of addresses.`);
  }
  return recipients.map((recipient) => recipient.trim());
}

function requireWebUrl(value, field) {
  const normalized = requireString(value, field);
  let parsed;
  try {
    parsed = new URL(normalized);
  } catch {
    throw new Error(`${field} must be a valid URL.`);
  }
  if (!["http:", "https:"].includes(parsed.protocol)) {
    throw new Error(`${field} must use HTTP or HTTPS.`);
  }
  return parsed.toString();
}

function requireProviderUrl(value, field, allowInsecureHttp) {
  const parsed = new URL(requireWebUrl(value, field));
  if (parsed.protocol === "https:") return parsed.toString();
  if (!allowInsecureHttp || !isLoopbackHost(parsed.hostname)) {
    throw new Error(
      `${field} must use HTTPS. Plain HTTP requires provider.allowInsecureHttp=true and a loopback host.`,
    );
  }
  return parsed.toString();
}

function isLoopbackHost(hostname) {
  return (
    hostname === "localhost" ||
    hostname === "[::1]" ||
    /^127(?:\.\d{1,3}){3}$/.test(hostname)
  );
}

function environmentName(value, field) {
  const normalized = requireString(value, field);
  if (!ENVIRONMENT_NAME.test(normalized)) {
    throw new Error(`${field} must be an uppercase environment variable name.`);
  }
  return normalized;
}

function optionalPath(value, field) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} must be a non-empty path.`);
  }
  return value.trim();
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
