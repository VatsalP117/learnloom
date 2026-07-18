import assert from "node:assert/strict";
import { access, mkdtemp, readFile, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { validateConfig } from "../src/config.mjs";
import { runDailyDossier } from "../src/daily-run.mjs";
import { resolveAppPaths } from "../src/paths.mjs";
import { DemoProvider } from "../src/provider.mjs";

test("Daily Run persists a Dossier before delivery", async () => {
  const fixture = await fixtureConfig();
  let deliveryObservedArtifacts = false;
  const delivery = {
    id: "test-email",
    async deliver({ dossier }) {
      await access(path.join(fixture.paths.outputDirectory, `${dossier.date}.md`));
      await access(path.join(fixture.paths.outputDirectory, `${dossier.date}.json`));
      deliveryObservedArtifacts = true;
      return { externalId: "sent-1" };
    },
  };
  const result = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  assert.equal(result.generated, true);
  assert.equal(deliveryObservedArtifacts, true);
  assert.equal(result.record.status, "complete");
  assert.equal(result.record.deliveries["test-email"].externalId, "sent-1");
});

test("Daily Run retries failed delivery without regenerating the Dossier", async () => {
  const fixture = await fixtureConfig();
  let providerCalls = 0;
  let deliveryCalls = 0;
  const demo = new DemoProvider();
  const provider = {
    async complete(input) {
      providerCalls += 1;
      return demo.complete(input);
    },
  };
  const delivery = {
    id: "test-email",
    async deliver() {
      deliveryCalls += 1;
      if (deliveryCalls === 1) throw new Error("temporary email failure");
      return { externalId: "sent-2" };
    },
  };
  const first = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider,
    deliveries: [delivery],
  });
  const second = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider,
    deliveries: [delivery],
  });
  assert.equal(first.record.status, "delivery_failed");
  assert.equal(second.generated, false);
  assert.equal(second.record.status, "complete");
  assert.equal(providerCalls, 4);
  assert.equal(deliveryCalls, 2);
});

test("Daily Run skips an already delivered destination", async () => {
  const fixture = await fixtureConfig();
  let deliveryCalls = 0;
  const delivery = {
    id: "test-email",
    async deliver() {
      deliveryCalls += 1;
      return { externalId: "sent-once" };
    },
  };
  await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  const result = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  assert.equal(result.generated, false);
  assert.equal(deliveryCalls, 1);
  const recordPath = path.join(
    fixture.paths.runsDirectory,
    "test-profile-2026-07-18.json",
  );
  assert.match(await readFile(recordPath, "utf8"), /"status": "delivered"/);
});

test("Daily Run regenerates when its recorded artifact is missing", async () => {
  const fixture = await fixtureConfig();
  let providerCalls = 0;
  const demo = new DemoProvider();
  const provider = {
    async complete(input) {
      providerCalls += 1;
      return demo.complete(input);
    },
  };
  const first = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider,
    deliveries: [],
  });
  await rm(first.outputPath);
  const second = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider,
    deliveries: [],
  });
  assert.equal(second.generated, true);
  assert.equal(providerCalls, 8);
});

async function fixtureConfig() {
  const directory = await mkdtemp(path.join(os.tmpdir(), "learnloom-daily-run-"));
  const config = validateConfig(
    {
      profileId: "test-profile",
      timeZone: "UTC",
      interests: ["learning"],
      sources: [{ name: "Demo", url: "https://example.com/feed" }],
      provider: { kind: "demo" },
      storage: { dataDirectory: "data", outputDirectory: "output" },
    },
    path.join(directory, "config.json"),
  );
  return {
    config,
    paths: resolveAppPaths(config, { env: {} }),
    now: new Date("2026-07-18T06:00:00.000Z"),
  };
}
