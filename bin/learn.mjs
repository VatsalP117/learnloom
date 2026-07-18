#!/usr/bin/env node

import { access, copyFile, mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { loadConfig, validateConfig } from "../src/config.mjs";
import { DEMO_ITEMS } from "../src/demo-data.mjs";
import { fetchSources } from "../src/feeds.mjs";
import { buildDossier } from "../src/pipeline.mjs";
import { createProvider, runProcess } from "../src/provider.mjs";
import {
  installSchedule,
  removeSchedule,
  scheduleStatus,
} from "../src/schedule.mjs";
import { loadHistory, saveRun } from "../src/state.mjs";

const cliPath = fileURLToPath(import.meta.url);
const projectRoot = path.resolve(path.dirname(cliPath), "..");
const command = process.argv[2] ?? "help";
const args = process.argv.slice(3);

try {
  if (command === "init") {
    await initialize(args);
  } else if (command === "run") {
    await run(args);
  } else if (command === "doctor") {
    await doctor(args);
  } else if (command === "schedule") {
    await schedule(args);
  } else if (["help", "--help", "-h"].includes(command)) {
    printHelp();
  } else {
    throw new Error(`Unknown command "${command}". Run "node bin/learn.mjs help".`);
  }
} catch (error) {
  process.stderr.write(`Error: ${error.message}\n`);
  process.exitCode = 1;
}

async function initialize(commandArgs) {
  const destination = path.resolve(option(commandArgs, "--config") ?? "config.json");
  const force = commandArgs.includes("--force");
  if (!force) {
    await access(destination)
      .then(() => {
        throw new Error(`${destination} already exists. Use --force to replace it.`);
      })
      .catch((error) => {
        if (error.code !== "ENOENT") throw error;
      });
  }
  await mkdir(path.dirname(destination), { recursive: true });
  await copyFile(path.join(projectRoot, "config.example.json"), destination);
  process.stdout.write(`Created ${destination}\nEdit your interests and sources, then run "npm run doctor".\n`);
}

async function run(commandArgs) {
  const demo = commandArgs.includes("--demo");
  const configPath = option(commandArgs, "--config") ?? "config.json";
  const config = demo ? demoConfig() : await loadConfig(configPath);
  const historyPath = path.resolve("data/history.json");
  const history = await loadHistory(historyPath);
  let items;
  let warnings = [];

  if (demo) {
    items = DEMO_ITEMS;
  } else {
    process.stdout.write("Fetching configured feeds…\n");
    const fetched = await fetchSources(config);
    items = fetched.items;
    warnings = fetched.warnings;
  }

  warnings.forEach((warning) => process.stderr.write(`Warning: ${warning}\n`));
  process.stdout.write(`Building a dossier from ${items.length} source items…\n`);
  const provider = createProvider(config, { demo, cwd: projectRoot });
  const result = await buildDossier({
    config,
    items,
    history,
    provider,
    onStage: (stage) => process.stdout.write(`  ${stage}\n`),
  });
  const saved = await saveRun(result, {
    historyPath,
    outputDirectory: "output",
    historyLimit: config.limits.historyEntries || 100,
  });
  process.stdout.write(`Dossier ready: ${saved.outputPath}\n`);
}

async function doctor(commandArgs) {
  const configPath = option(commandArgs, "--config") ?? "config.json";
  const checks = [];
  checks.push({
    name: "Node.js",
    ok: Number(process.versions.node.split(".")[0]) >= 22,
    detail: process.version,
  });

  let config;
  try {
    config = await loadConfig(configPath);
    checks.push({ name: "Configuration", ok: true, detail: config.configPath });
  } catch (error) {
    checks.push({ name: "Configuration", ok: false, detail: error.message });
  }

  const executable = config?.provider.executable ?? "cmd";
  let status;
  try {
    status = await runProcess(executable, ["status", "--json"], {
      timeoutMs: 30_000,
      acceptedExitCodes: [0, 1],
    });
    const parsed = safeJson(status.stdout);
    const authenticated = inferAuthentication(parsed, status.stdout);
    checks.push({
      name: "Command Code login",
      ok: authenticated,
      detail: authenticated ? "authenticated" : "CLI found, but login was not confirmed",
    });
  } catch (error) {
    checks.push({ name: "Command Code CLI", ok: false, detail: error.message });
  }

  if (status) {
    try {
      const models = await runProcess(executable, ["--list-models"], { timeoutMs: 60_000 });
      const wantedModel = config?.provider.model ?? "deepseek-v4-pro";
      const available = models.stdout.toLowerCase().includes(wantedModel.toLowerCase());
      checks.push({
        name: "DeepSeek model",
        ok: available,
        detail: available ? wantedModel : `${wantedModel} was not present in cmd --list-models`,
      });
    } catch (error) {
      checks.push({ name: "Model discovery", ok: false, detail: error.message });
    }
  }

  for (const check of checks) {
    process.stdout.write(`${check.ok ? "✓" : "✗"} ${check.name}: ${check.detail}\n`);
  }
  if (checks.some((check) => !check.ok)) {
    process.exitCode = 1;
  } else {
    process.stdout.write("Ready for a live run.\n");
  }
}

async function schedule(commandArgs) {
  const action = commandArgs[0] ?? "status";
  if (action === "status") {
    const result = await scheduleStatus();
    process.stdout.write(
      `${result.loaded ? "✓" : result.installed ? "!" : "✗"} Schedule: ${
        result.loaded ? "installed and loaded" : result.installed ? "installed but not loaded" : "not installed"
      }\n${result.plistPath}\n`,
    );
    return;
  }
  if (action === "remove") {
    const removed = await removeSchedule();
    process.stdout.write(`Removed ${removed}\n`);
    return;
  }
  if (action !== "install") {
    throw new Error('Schedule action must be "install", "status", or "remove".');
  }

  const configPath = path.resolve(option(commandArgs, "--config") ?? "config.json");
  await loadConfig(configPath);
  const hour = integerOption(commandArgs, "--hour", 9, 0, 23);
  const minute = integerOption(commandArgs, "--minute", 0, 0, 59);
  const logDirectory = path.resolve("data/logs");
  const plistPath = await installSchedule({
    nodePath: process.execPath,
    cliPath,
    configPath,
    workingDirectory: projectRoot,
    logDirectory,
    hour,
    minute,
    environmentPath: process.env.PATH,
  });
  process.stdout.write(
    `Installed ${plistPath}\nLearnloom will run daily at ${pad(hour)}:${pad(minute)} local time.\n`,
  );
}

function demoConfig() {
  return validateConfig(
    {
      timeZone: "Asia/Kolkata",
      interests: ["learning science", "adaptive AI systems"],
      learner: {
        level: "technically experienced",
        goal: "build durable understanding",
        lessonMinutes: 15,
      },
      sources: [{ name: "Demo", url: "https://example.com/feed", limit: 10 }],
      provider: { kind: "demo" },
    },
    "<demo>",
  );
}

function option(commandArgs, name) {
  const index = commandArgs.indexOf(name);
  if (index === -1) return undefined;
  if (!commandArgs[index + 1] || commandArgs[index + 1].startsWith("--")) {
    throw new Error(`${name} requires a value.`);
  }
  return commandArgs[index + 1];
}

function integerOption(commandArgs, name, fallback, minimum, maximum) {
  const raw = option(commandArgs, name);
  if (raw === undefined) return fallback;
  const value = Number(raw);
  if (!Number.isInteger(value) || value < minimum || value > maximum) {
    throw new Error(`${name} must be an integer from ${minimum} to ${maximum}.`);
  }
  return value;
}

function safeJson(value) {
  try {
    return JSON.parse(value.trim());
  } catch {
    return null;
  }
}

function inferAuthentication(parsed, raw) {
  if (parsed) {
    const serialized = JSON.stringify(parsed).toLowerCase();
    if (serialized.includes('"authenticated":true') || serialized.includes('"loggedin":true')) {
      return true;
    }
    if (serialized.includes('"authenticated":false') || serialized.includes('"loggedin":false')) {
      return false;
    }
  }
  const normalized = raw.toLowerCase();
  return (
    normalized.includes("authenticated") &&
    !normalized.includes("not authenticated") &&
    !normalized.includes("false")
  );
}

function pad(value) {
  return String(value).padStart(2, "0");
}

function printHelp() {
  process.stdout.write(`Learnloom

Usage:
  learn init [--config path] [--force]
  learn run [--config path] [--demo]
  learn doctor [--config path]
  learn schedule install [--config path] [--hour 9] [--minute 0]
  learn schedule status
  learn schedule remove

Quick start:
  npm run demo
  node bin/learn.mjs init
  npm run doctor
  npm start
`);
}
