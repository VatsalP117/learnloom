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
    async deliver({ dossier, generationId }) {
      const stem = `${dossier.date}-${generationId}`;
      await access(path.join(fixture.paths.outputDirectory, `${stem}.md`));
      await access(path.join(fixture.paths.outputDirectory, `${stem}.json`));
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
  assert.equal(typeof result.record.generationId, "string");
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

test("Daily Run does not retry an ambiguous delivery outcome", async () => {
  const fixture = await fixtureConfig();
  let deliveryCalls = 0;
  const delivery = {
    id: "test-email",
    async deliver() {
      deliveryCalls += 1;
      const error = new Error("response lost after transmission");
      error.outcomeUnknown = true;
      throw error;
    },
  };
  const first = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  const second = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  assert.equal(first.record.status, "delivery_unknown");
  assert.equal(second.record.status, "delivery_unknown");
  assert.equal(
    second.record.deliveries["test-email"].status,
    "unknown",
  );
  assert.equal(deliveryCalls, 1);
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

test("forced Daily Run creates a new delivery generation", async () => {
  const fixture = await fixtureConfig();
  const generations = [];
  const delivery = {
    id: "test-email",
    async deliver({ generationId }) {
      generations.push(generationId);
      return { externalId: `sent-${generations.length}` };
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
  const forced = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    force: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  assert.equal(forced.generated, true);
  assert.equal(generations.length, 2);
  assert.notEqual(generations[0], generations[1]);
});

test("crash before record swap preserves the previously delivered generation", async () => {
  const fixture = await fixtureConfig();
  let deliveryCalls = 0;
  const delivery = {
    id: "test-email",
    async deliver() {
      deliveryCalls += 1;
      return { externalId: `sent-${deliveryCalls}` };
    },
  };
  const first = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  await assert.rejects(
    runDailyDossier({
      config: fixture.config,
      paths: fixture.paths,
      demo: true,
      force: true,
      now: fixture.now,
      provider: new DemoProvider(),
      deliveries: [delivery],
      onEvent(event) {
        if (event.type === "persisted") throw new Error("simulated crash");
      },
    }),
    /simulated crash/,
  );
  const recovered = await runDailyDossier({
    config: fixture.config,
    paths: fixture.paths,
    demo: true,
    now: fixture.now,
    provider: new DemoProvider(),
    deliveries: [delivery],
  });
  assert.equal(recovered.generated, false);
  assert.equal(recovered.record.generationId, first.record.generationId);
  assert.equal(recovered.outputPath, first.outputPath);
  assert.equal(deliveryCalls, 1);
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
