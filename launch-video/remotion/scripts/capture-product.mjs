#!/usr/bin/env node
/**
 * capture-product.mjs — records short product clips of the running Learnloom demo.
 *
 * Launches Google Chrome headlessly via playwright-core, visits direct demo routes
 * (in-memory fixtures only), performs gentle scroll/click interactions, and records
 * one WebM per clip into public/product-clips/.
 *
 * Requirements:
 *   - The demo server must be running (default http://127.0.0.1:4173, override with
 *     LEARNLOOM_DEMO_URL). Start it with `npm run demo` from the repo root.
 *   - Google Chrome at /Applications/Google Chrome.app/Contents/MacOS/Google Chrome
 *     (override with CHROME_PATH).
 *
 * Usage:
 *   npm run capture
 *   LEARNLOOM_DEMO_URL=http://127.0.0.1:5173 npm run capture
 *
 * The output directory is wiped before each run so results are deterministic and
 * never mixed with stale clips. No packages are installed and nothing outside
 * public/product-clips is written.
 */

import { chromium } from "playwright-core";
import { access, mkdir, rename, rm, stat } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const projectDir = path.resolve(scriptDir, "..");
const clipsDir = path.join(projectDir, "public", "product-clips");

const DEMO_URL = (process.env.LEARNLOOM_DEMO_URL ?? "http://127.0.0.1:4173").replace(/\/+$/, "");
const CHROME_PATH =
  process.env.CHROME_PATH ?? "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";

const VIEWPORT = { width: 1440, height: 900 };
const NAV_TIMEOUT_MS = 20_000;
const WAIT_TIMEOUT_MS = 20_000;
const LOOPBACK_HOSTS = new Set(["127.0.0.1", "localhost", "::1", "[::1]"]);

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const log = (message) => console.log(`[capture] ${message}`);

/** Preflight: Chrome binary present, demo server reachable. Fails fast and clearly. */
async function preflight() {
  const demoHost = new URL(DEMO_URL).hostname;
  if (!LOOPBACK_HOSTS.has(demoHost)) {
    throw new Error(
      `Refusing to capture non-local URL ${DEMO_URL}. ` +
        "Run the Vite demo on localhost or 127.0.0.1.",
    );
  }

  try {
    await access(CHROME_PATH, fsConstants.X_OK);
  } catch {
    throw new Error(
      `Google Chrome not found at ${CHROME_PATH}.\n` +
        "Install Chrome or set CHROME_PATH to a Chrome/Chromium binary.",
    );
  }

  try {
    const response = await fetch(`${DEMO_URL}/`, {
      redirect: "follow",
      signal: AbortSignal.timeout(6_000),
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
  } catch (error) {
    throw new Error(
      `Demo server unavailable at ${DEMO_URL} (${error.message}).\n` +
        "Start it first, e.g. `npm run demo` from the repo root.",
    );
  }
  log(`demo server reachable at ${DEMO_URL}`);
}

/** Wait for fonts and a beat so first paint is stable before recording actions. */
async function settle(page) {
  await page.evaluate(() => document.fonts.ready);
  await sleep(700);
}

const CLIPS = [
  {
    name: "today",
    path: "/",
    async run(page) {
      await page.getByRole("heading", { level: 1 }).waitFor({ state: "visible" });
      await page
        .getByRole("heading", { name: "Active learning streams" })
        .waitFor({ state: "visible" });
      // Gentle scroll down the Today page to reveal the day's streams.
      for (const delta of [420, 420, 420]) {
        await page.mouse.wheel(0, delta);
        await sleep(1_100);
      }
      await sleep(1_400);
    },
  },
  {
    name: "streams",
    path: "/streams",
    async run(page) {
      await page.getByRole("heading", { name: "Streams", exact: true }).waitFor({ state: "visible" });
      await page.getByRole("heading", { name: "Active streams" }).waitFor({ state: "visible" });
      await page.locator("a.streams-card").first().waitFor({ state: "visible" });
      // Scroll the hero into view, then hover the first stream card.
      await page.mouse.wheel(0, 520);
      await sleep(1_300);
      await page.locator("a.streams-card").first().hover();
      await sleep(1_100);
      await page.mouse.wheel(0, 420);
      await sleep(1_100);
      await page.mouse.wheel(0, 420);
      await sleep(1_500);
    },
  },
  {
    name: "lesson",
    path: "/?demoIssue=ai-evaluation-issue-1",
    async run(page) {
      await page.locator("article.reader-paper").waitFor({ state: "visible" });
      await page.getByRole("heading", { level: 1 }).waitFor({ state: "visible" });
      await page
        .locator("section.reader-section h2")
        .first()
        .waitFor({ state: "visible" });
      // Reading rhythm: scroll through the lesson paper section by section.
      for (const delta of [480, 560, 640, 680, 680]) {
        await page.mouse.wheel(0, delta);
        await sleep(900);
      }
      await sleep(1_300);
    },
  },
  {
    name: "library",
    path: "/library",
    async run(page) {
      await page.getByRole("heading", { name: "Library", exact: true }).waitFor({ state: "visible" });
      await page.getByRole("group", { name: "Filter lessons" }).waitFor({ state: "visible" });
      await page.locator("a.lesson-library-card").first().waitFor({ state: "visible" });
      await sleep(1_300);
      // Show the filters working: In progress, then Completed.
      await page.getByRole("button", { name: "In progress" }).click();
      await sleep(1_200);
      await page.getByRole("button", { name: "Completed" }).click();
      await page.locator("a.lesson-library-card").first().waitFor({ state: "visible" });
      await sleep(1_600);
    },
  },
  {
    name: "review",
    path: "/review",
    async run(page) {
      await page.getByRole("heading", { name: "Spaced retrieval" }).waitFor({ state: "visible" });
      await page
        .getByRole("button", { name: "Reveal lesson context" })
        .waitFor({ state: "visible" });
      await sleep(1_400);
      // Reveal the lesson context and the recall rating controls.
      await page.getByRole("button", { name: "Reveal lesson context" }).click();
      await page.getByRole("button", { name: "Recalled solidly" }).waitFor({ state: "visible" });
      await sleep(1_800);
      await page.mouse.wheel(0, 260);
      await sleep(1_300);
    },
  },
  {
    name: "publishing",
    path: "/publishing",
    async run(page) {
      await page.getByRole("heading", { name: "Publishing", exact: true }).waitFor({ state: "visible" });
      await page
        .locator('[aria-label="Current publishing state"]')
        .waitFor({ state: "visible" });
      await sleep(1_300);
      // Scroll through identity + analytics, then toggle site visibility.
      await page.mouse.wheel(0, 520);
      await sleep(1_200);
      await page.mouse.wheel(0, 520);
      await sleep(1_200);
      await page.getByRole("button", { name: "Make private" }).scrollIntoViewIfNeeded();
      await sleep(600);
      await page.getByRole("button", { name: "Make private" }).click();
      await page.getByRole("button", { name: "Publish site" }).waitFor({ state: "visible" });
      await sleep(1_600);
    },
  },
];

async function recordClip(browser, clip) {
  const context = await browser.newContext({
    viewport: VIEWPORT,
    deviceScaleFactor: 1,
    recordVideo: { dir: clipsDir, size: VIEWPORT },
  });
  const page = await context.newPage();
  page.setDefaultTimeout(WAIT_TIMEOUT_MS);

  const url = `${DEMO_URL}${clip.path}`;
  const startedAt = Date.now();
  try {
    log(`[${clip.name}] opening ${url}`);
    await page.goto(url, { waitUntil: "load", timeout: NAV_TIMEOUT_MS });
    await settle(page);
    await clip.run(page);
    await sleep(400);
    log(
      `[${clip.name}] actions finished in ${((Date.now() - startedAt) / 1_000).toFixed(1)}s, ` +
        "closing context to finalize video",
    );
    await context.close();

    const video = page.video();
    if (!video) throw new Error("no video was recorded (recordVideo missing?)");
    const rawPath = await video.path();
    const finalPath = path.join(clipsDir, `${clip.name}.webm`);
    await rename(rawPath, finalPath);
    const { size } = await stat(finalPath);
    log(
      `[${clip.name}] wrote ${path.relative(projectDir, finalPath)} ` +
        `(${(size / 1_024).toFixed(0)} KiB)`,
    );
    return finalPath;
  } catch (error) {
    await context.close().catch(() => {});
    throw new Error(`clip "${clip.name}" failed: ${error.message}`);
  }
}

async function main() {
  await preflight();

  // Deterministic output: the clips directory is fully replaced each run.
  await rm(clipsDir, { recursive: true, force: true });
  await mkdir(clipsDir, { recursive: true });
  log(`output directory reset: ${path.relative(projectDir, clipsDir)}/`);
  log(`recording ${CLIPS.length} clips at ${VIEWPORT.width}x${VIEWPORT.height}`);

  const browser = await chromium.launch({
    executablePath: CHROME_PATH,
    headless: true,
    args: [
      "--mute-audio",
      "--disable-background-timer-throttling",
      "--disable-renderer-backgrounding",
      "--disable-backgrounding-occluded-windows",
      "--no-first-run",
      "--no-default-browser-check",
    ],
  });

  const written = [];
  try {
    for (const clip of CLIPS) {
      written.push(await recordClip(browser, clip));
    }
  } finally {
    await browser.close().catch(() => {});
  }

  log(`done — ${written.length} clip(s) recorded:`);
  for (const file of written) {
    console.log(`  ${path.relative(projectDir, file)}`);
  }
}

main().catch((error) => {
  console.error(`[capture] FAILED: ${error.message}`);
  process.exitCode = 1;
});
