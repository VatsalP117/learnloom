import { randomUUID } from "node:crypto";
import { readFile } from "node:fs/promises";
import { createDeliveryAdapters } from "./delivery.mjs";
import { DEMO_ITEMS } from "./demo-data.mjs";
import { fetchSources } from "./feeds.mjs";
import { buildDossier } from "./pipeline.mjs";
import { createProvider } from "./provider.mjs";
import { FileRunStore } from "./run-store.mjs";
import { loadHistory, loadJson, saveRun } from "./state.mjs";

export async function runDailyDossier(options) {
  const {
    config,
    paths,
    now = new Date(),
    demo = false,
    force = false,
    onEvent = () => {},
  } = options;
  const date = formatLocalDate(now, config.timeZone);
  const runId = options.runId ?? `${config.profileId}-${date}`;
  const store = options.runStore ?? new FileRunStore(paths);
  const release = await store.acquire(runId);

  try {
    let record = await store.load(runId);
    let dossier;
    let markdown;
    let generated = false;
    let outputPath;

    if (!force && record?.dossierPath && record?.artifactPath) {
      const reusable = await loadReusableArtifacts(record);
      if (reusable) {
        ({ dossier, markdown } = reusable);
        outputPath = record.artifactPath;
        onEvent({ type: "reuse", runId, outputPath });
      }
    }

    if (!dossier || !markdown) {
      try {
        const items = demo
          ? DEMO_ITEMS
          : await loadSourceItems(config, options.fetchSourcesFn ?? fetchSources, onEvent);
        const history = await loadHistory(paths.historyPath);
        onEvent({ type: "generation", itemCount: items.length });
        const provider =
          options.provider ?? createProvider(config, {
            demo,
            cwd: options.cwd,
            env: options.env,
            fetchImpl: options.fetchImpl,
            sleep: options.sleep,
          });
        const result = await buildDossier({
          config,
          items,
          history,
          provider,
          now,
          onStage: (stage) => onEvent({ type: "stage", stage }),
        });
        const generationId = randomUUID();
        const saved = await saveRun(result, {
          historyPath: paths.historyPath,
          outputDirectory: paths.outputDirectory,
          historyLimit: config.limits.historyEntries,
          generationId,
        });
        onEvent({ type: "persisted", runId, generationId, outputPath: saved.outputPath });
        dossier = result.dossier;
        markdown = result.markdown;
        outputPath = saved.outputPath;
        generated = true;
        record = {
          version: 1,
          runId,
          profileId: config.profileId,
          date,
          createdAt: record?.createdAt ?? now.toISOString(),
          updatedAt: now.toISOString(),
          generationStatus: "complete",
          generationId,
          artifactPath: saved.outputPath,
          dossierPath: saved.dossierPath,
          deliveries: {},
        };
        await store.save(record);
      } catch (error) {
        if (!record?.artifactPath) {
          record = {
            ...record,
            version: 1,
            runId,
            profileId: config.profileId,
            date,
            createdAt: record?.createdAt ?? now.toISOString(),
            updatedAt: now.toISOString(),
            generationStatus: "failed",
            generationError: safeRecordError(error),
            deliveries: record?.deliveries ?? {},
          };
        } else {
          record.lastGenerationError = safeRecordError(error);
          record.lastGenerationFailedAt = now.toISOString();
        }
        await store.save(record);
        throw error;
      }
    }

    const adapters =
      options.deliveries ??
      createDeliveryAdapters(config, {
        env: options.env,
        fetchImpl: options.fetchImpl,
        resendEndpoint: options.resendEndpoint,
      });
    const deliveryErrors = [];
    record.deliveries ??= {};
    for (const adapter of adapters) {
      const priorDeliveryStatus = record.deliveries[adapter.id]?.status;
      if (["delivered", "unknown"].includes(priorDeliveryStatus)) {
        onEvent({
          type:
            priorDeliveryStatus === "unknown"
              ? "delivery-unknown-skip"
              : "delivery-skip",
          deliveryId: adapter.id,
        });
        continue;
      }
      onEvent({ type: "delivery", deliveryId: adapter.id });
      try {
        const receipt = await adapter.deliver({
          runId,
          generationId: record.generationId ?? dossier.generatedAt ?? "legacy",
          dossier,
          markdown,
        });
        record.deliveries[adapter.id] = {
          status: "delivered",
          externalId: receipt.externalId ?? null,
          deliveredAt: new Date().toISOString(),
        };
      } catch (error) {
        record.deliveries[adapter.id] = {
          status: error?.outcomeUnknown ? "unknown" : "failed",
          error: safeRecordError(error),
          failedAt: new Date().toISOString(),
        };
        deliveryErrors.push({ deliveryId: adapter.id, error });
      }
      await store.save(record);
    }

    record.status = Object.values(record.deliveries).some(
      (delivery) => delivery.status === "unknown",
    )
      ? "delivery_unknown"
      : deliveryErrors.length
        ? "delivery_failed"
        : "complete";
    await store.save(record);
    return {
      runId,
      date,
      generated,
      outputPath,
      dossier,
      record,
      deliveryErrors,
    };
  } finally {
    await release();
  }
}

async function loadReusableArtifacts(record) {
  try {
    const dossier = await loadJson(record.dossierPath, "Dossier");
    if (!dossier) return null;
    const markdown = await readFile(record.artifactPath, "utf8");
    return { dossier, markdown };
  } catch (error) {
    if (error.code === "ENOENT") return null;
    throw error;
  }
}

async function loadSourceItems(config, fetchSourcesFn, onEvent) {
  onEvent({ type: "fetch" });
  const result = await fetchSourcesFn(config);
  result.warnings.forEach((warning) => onEvent({ type: "warning", message: warning }));
  return result.items;
}

function safeRecordError(error) {
  return String(error?.message ?? "Unknown error").slice(0, 500);
}

function formatLocalDate(date, timeZone) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}
