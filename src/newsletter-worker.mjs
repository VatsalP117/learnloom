import { validateConfig } from "./config.mjs";
import { runDailyDossier } from "./daily-run.mjs";
import { resolveAppPaths } from "./paths.mjs";

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
  return { dispatched, processed };
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
      storage: baseConfig.storage,
      limits: baseConfig.limits,
    },
    baseConfig.configPath,
  );
}
