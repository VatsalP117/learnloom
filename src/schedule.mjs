import { access, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { runProcess } from "./provider.mjs";

export const LAUNCH_AGENT_LABEL = "app.learnloom.morning";

export function buildLaunchAgent({
  nodePath,
  cliPath,
  configPath,
  workingDirectory,
  logDirectory,
  hour = 9,
  minute = 0,
  environmentPath = process.env.PATH,
  learnloomHome,
}) {
  const args = [nodePath, cliPath, "run", "--config", configPath];
  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">',
    '<plist version="1.0">',
    "<dict>",
    "  <key>Label</key>",
    `  <string>${escapeXml(LAUNCH_AGENT_LABEL)}</string>`,
    "  <key>ProgramArguments</key>",
    "  <array>",
    ...args.map((argument) => `    <string>${escapeXml(argument)}</string>`),
    "  </array>",
    "  <key>WorkingDirectory</key>",
    `  <string>${escapeXml(workingDirectory)}</string>`,
    "  <key>EnvironmentVariables</key>",
    "  <dict>",
    "    <key>PATH</key>",
    `    <string>${escapeXml(environmentPath)}</string>`,
    ...(learnloomHome
      ? [
          "    <key>LEARNLOOM_HOME</key>",
          `    <string>${escapeXml(learnloomHome)}</string>`,
        ]
      : []),
    "  </dict>",
    "  <key>StartCalendarInterval</key>",
    "  <dict>",
    "    <key>Hour</key>",
    `    <integer>${hour}</integer>`,
    "    <key>Minute</key>",
    `    <integer>${minute}</integer>`,
    "  </dict>",
    "  <key>StandardOutPath</key>",
    `  <string>${escapeXml(path.join(logDirectory, "launchd.out.log"))}</string>`,
    "  <key>StandardErrorPath</key>",
    `  <string>${escapeXml(path.join(logDirectory, "launchd.err.log"))}</string>`,
    "</dict>",
    "</plist>",
    "",
  ].join("\n");
}

export async function installSchedule(options) {
  assertMacOS();
  const launchAgentsDirectory = path.join(os.homedir(), "Library", "LaunchAgents");
  const plistPath = path.join(launchAgentsDirectory, `${LAUNCH_AGENT_LABEL}.plist`);
  await mkdir(launchAgentsDirectory, { recursive: true });
  await mkdir(options.logDirectory, { recursive: true });

  const plist = buildLaunchAgent(options);
  await writeFile(plistPath, plist, "utf8");

  const domain = `gui/${process.getuid()}`;
  await runProcess("launchctl", ["bootout", domain, plistPath], {
    timeoutMs: 15_000,
  }).catch(() => {});
  await runProcess("launchctl", ["bootstrap", domain, plistPath], {
    timeoutMs: 15_000,
  });

  return plistPath;
}

export async function removeSchedule() {
  assertMacOS();
  const plistPath = schedulePath();
  const domain = `gui/${process.getuid()}`;
  await runProcess("launchctl", ["bootout", domain, plistPath], {
    timeoutMs: 15_000,
  }).catch(() => {});
  await rm(plistPath, { force: true });
  return plistPath;
}

export async function scheduleStatus() {
  const plistPath = schedulePath();
  let installed = true;
  try {
    await access(plistPath);
  } catch {
    installed = false;
  }

  let loaded = false;
  if (process.platform === "darwin") {
    loaded = await runProcess(
      "launchctl",
      ["print", `gui/${process.getuid()}/${LAUNCH_AGENT_LABEL}`],
      { timeoutMs: 15_000 },
    )
      .then(() => true)
      .catch(() => false);
  }
  return { installed, loaded, plistPath };
}

export async function readSchedule() {
  return readFile(schedulePath(), "utf8");
}

function schedulePath() {
  return path.join(os.homedir(), "Library", "LaunchAgents", `${LAUNCH_AGENT_LABEL}.plist`);
}

function assertMacOS() {
  if (process.platform !== "darwin") {
    throw new Error("Automatic scheduling currently supports macOS launchd only.");
  }
}

function escapeXml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&apos;");
}
