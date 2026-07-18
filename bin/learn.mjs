#!/usr/bin/env node

import { access, copyFile, mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { loadConfig, validateConfig } from "../src/config.mjs";
import { runDailyDossier } from "../src/daily-run.mjs";
import { checkDeliveries } from "../src/delivery.mjs";
import { resolveAppPaths } from "../src/paths.mjs";
import { checkProvider } from "../src/provider.mjs";
import {
  installSchedule,
  removeSchedule,
  scheduleStatus,
} from "../src/schedule.mjs";

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
  const force = commandArgs.includes("--force");
  const configPath = option(commandArgs, "--config") ?? "config.json";
  const config = demo ? demoConfig() : await loadConfig(configPath);
  const paths = resolveAppPaths(config);
  const result = await runDailyDossier({
    config,
    paths,
    demo,
    force,
    cwd: projectRoot,
    onEvent: printRunEvent,
  });
  process.stdout.write(
    `${result.generated ? "Dossier ready" : "Existing Dossier reused"}: ${result.outputPath}\n`,
  );
  if (result.deliveryErrors.length > 0) {
    for (const failure of result.deliveryErrors) {
      process.stderr.write(
        `Delivery ${failure.deliveryId} failed: ${failure.error.message}\n`,
      );
    }
    process.exitCode = 2;
  }
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
    const paths = resolveAppPaths(config);
    checks.push({ name: "Application home", ok: true, detail: paths.appHome });
  } catch (error) {
    checks.push({ name: "Configuration", ok: false, detail: error.message });
  }

  if (config) {
    checks.push(...(await checkProvider(config, { cwd: projectRoot })));
    checks.push(...checkDeliveries(config));
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
  const config = await loadConfig(configPath);
  const paths = resolveAppPaths(config);
  const hour = integerOption(commandArgs, "--hour", 9, 0, 23);
  const minute = integerOption(commandArgs, "--minute", 0, 0, 59);
  const logDirectory = paths.logsDirectory;
  const plistPath = await installSchedule({
    nodePath: process.execPath,
    cliPath,
    configPath,
    workingDirectory: projectRoot,
    logDirectory,
    hour,
    minute,
    environmentPath: process.env.PATH,
    learnloomHome: process.env.LEARNLOOM_HOME,
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
      deliveries: [],
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

function printRunEvent(event) {
  if (event.type === "fetch") process.stdout.write("Fetching configured feeds…\n");
  if (event.type === "warning") process.stderr.write(`Warning: ${event.message}\n`);
  if (event.type === "generation") {
    process.stdout.write(`Building a Dossier from ${event.itemCount} Source Items…\n`);
  }
  if (event.type === "stage") process.stdout.write(`  ${event.stage}\n`);
  if (event.type === "reuse") process.stdout.write(`Reusing Daily Run ${event.runId}…\n`);
  if (event.type === "delivery") {
    process.stdout.write(`Delivering through ${event.deliveryId}…\n`);
  }
  if (event.type === "delivery-skip") {
    process.stdout.write(`Delivery ${event.deliveryId} already completed; skipping.\n`);
  }
}

function pad(value) {
  return String(value).padStart(2, "0");
}

function printHelp() {
  process.stdout.write(`Learnloom

Usage:
  learn init [--config path] [--force]
  learn run [--config path] [--demo] [--force]
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
