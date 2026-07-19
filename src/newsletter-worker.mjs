import { readFile } from "node:fs/promises";
import { validateConfig } from "./config.mjs";
import { runDailyDossier } from "./daily-run.mjs";
import { ResendDelivery } from "./delivery.mjs";
import { resolveAppPaths } from "./paths.mjs";
import { loadJson } from "./state.mjs";

export async function processNextIssue(options) {
  const {
    workspace,
    baseConfig,
    demo = false,
    onEvent = () => {},
  } = options;
  const clock = options.clock ?? (() => new Date());
  const startedAt = options.now ?? clock();
  const issue = workspace.claimNextIssue(startedAt);
  if (!issue) return null;
  onEvent({
    type: "issue-claimed",
    issueId: issue.id,
    newsletterId: issue.newsletterId,
  });

  try {
    const config = newsletterRuntimeConfig(baseConfig, issue.newsletter);
    const paths = resolveAppPaths(config, {
      env: options.env,
      home: options.home,
    });
    const run = await (options.runDailyDossierFn ?? runDailyDossier)({
      config,
      paths,
      runId: issue.id,
      now: startedAt,
      demo,
      cwd: options.cwd,
      env: options.env,
      provider: options.provider,
      fetchImpl: options.fetchImpl,
      fetchSourcesFn: options.fetchSourcesFn,
      sleep: options.sleep,
      deliveries: [],
      onEvent: (event) =>
        onEvent({
          ...event,
          issueId: issue.id,
          newsletterId: issue.newsletterId,
        }),
    });
    const completed = workspace.completeIssue(
      issue.id,
      {
        title: run.dossier.title,
        generationId: run.record.generationId,
        artifactPath: run.record.artifactPath,
        dossierPath: run.record.dossierPath,
      },
      options.now ?? clock(),
    );
    onEvent({
      type: "issue-generated",
      issueId: issue.id,
      newsletterId: issue.newsletterId,
    });
    return completed;
  } catch (error) {
    const failed = workspace.failIssue(issue.id, error, options.now ?? clock());
    onEvent({
      type: "issue-failed",
      issueId: issue.id,
      newsletterId: issue.newsletterId,
      message: failed.error,
    });
    return failed;
  }
}

export async function runWorkerCycle(options) {
  const dispatchTime = options.now ?? new Date();
  const dispatched = options.workspace.dispatchDue(dispatchTime);
  const processed = [];
  const maximum = options.maximumIssues ?? 100;
  for (let count = 0; count < maximum; count += 1) {
    const issue = await processNextIssue(options);
    if (!issue) break;
    processed.push(issue);
  }
  const deliveries = [];
  const maximumDeliveries = options.maximumDeliveries ?? 100;
  for (let count = 0; count < maximumDeliveries; count += 1) {
    const delivery = await processNextDelivery(options);
    if (!delivery) break;
    deliveries.push(delivery);
  }
  return { dispatched, processed, deliveries };
}

export async function processNextDelivery(options) {
  const {
    workspace,
    baseConfig,
    onEvent = () => {},
  } = options;
  const clock = options.clock ?? (() => new Date());
  const startedAt = options.now ?? clock();
  const delivery = workspace.claimNextDelivery(startedAt);
  if (!delivery) return null;
  onEvent({
    type: "delivery-claimed",
    issueId: delivery.issue.id,
    newsletterId: delivery.issue.newsletterId,
    attemptCount: delivery.attemptCount,
  });

  try {
    const resendConfig = baseConfig.deliveries.find(
      (candidate) => candidate.kind === "resend" && candidate.enabled,
    );
    if (!resendConfig) {
      throw new Error(
        "No enabled Resend delivery is configured for Newsletter email.",
      );
    }
    const dossier = await loadJson(
      delivery.issue.dossierPath,
      "Newsletter Dossier",
    );
    if (!dossier || typeof dossier !== "object" || Array.isArray(dossier)) {
      throw new Error(
        `Newsletter Dossier is missing at ${delivery.issue.dossierPath}.`,
      );
    }
    const markdown = await readFile(delivery.issue.artifactPath, "utf8");
    const site = delivery.newsletter.ownerAccountId
      ? workspace.getSiteForAccount(delivery.newsletter.ownerAccountId)
      : null;
    const webUrl =
      options.deployment?.mode === "hosted" &&
      site?.visibility === "public" &&
      delivery.newsletter.siteVisible &&
      delivery.issue.publicationState === "published" &&
      delivery.issue.publicId &&
      delivery.issue.publicSlug
        ? `https://${site.username}.${options.deployment.rootDomain}/d/${encodeURIComponent(
            delivery.issue.publicId,
          )}/${encodeURIComponent(delivery.issue.publicSlug)}`
        : null;
    const adapter = new ResendDelivery(
      {
        ...resendConfig,
        id: "newsletter-email",
        to: delivery.newsletter.emailRecipients,
      },
      {
        env: options.env,
        fetchImpl: options.fetchImpl,
        resendEndpoint: options.resendEndpoint,
      },
    );
    const receipt = await adapter.deliver({
      runId: delivery.issue.id,
      generationId: delivery.issue.generationId,
      dossier,
      markdown,
      webUrl,
    });
    const completed = workspace.completeDelivery(
      delivery.issue.id,
      receipt.externalId,
      options.now ?? clock(),
    );
    onEvent({
      type: "delivery-complete",
      issueId: delivery.issue.id,
      newsletterId: delivery.issue.newsletterId,
      externalId: completed.externalId,
    });
    return completed;
  } catch (error) {
    const failed = error?.outcomeUnknown
      ? workspace.markDeliveryUnknown(
          delivery.issue.id,
          error,
          options.now ?? clock(),
        )
      : workspace.failDelivery(
          delivery.issue.id,
          error,
          options.now ?? clock(),
        );
    onEvent({
      type:
        failed.status === "unknown"
          ? "delivery-outcome-unknown"
          : "delivery-failed",
      issueId: delivery.issue.id,
      newsletterId: delivery.issue.newsletterId,
      message: failed.error,
    });
    return failed;
  }
}

export function newsletterRuntimeConfig(baseConfig, newsletter) {
  return validateConfig(
    {
      profileId: newsletter.id,
      timeZone: newsletter.timeZone,
      interests: [newsletter.topic],
      learner: {
        level: newsletter.learnerLevel,
        goal: newsletter.learnerGoal,
        lessonMinutes: newsletter.lessonMinutes,
      },
      sources: newsletter.sources,
      provider: baseConfig.provider,
      deliveries: [],
      content: {
        ...baseConfig.content,
        aiExplorationEnabled: newsletter.aiExplorationEnabled,
      },
      storage: baseConfig.storage,
      limits: baseConfig.limits,
    },
    baseConfig.configPath,
  );
}
