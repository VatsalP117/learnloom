# Implementation Report

## Summary

Implemented Learnloom v0.3's local multi-newsletter test phase. A deep
SQLite Workspace now owns Newsletter validation, daily local schedules,
scheduled idempotency, manual Issue queueing, atomic claims, lifecycle
transitions, and dashboard projections. A separate worker maps claimed Issues
into the existing Daily Run while explicitly suppressing all delivery adapters.

Added a server-rendered dashboard with overview cards, Newsletter creation,
pause/resume, Run Now, Issue history, and escaped Dossier previews. Added
long-lived dashboard and worker CLI/Compose roles while preserving the existing
finite CLI job.

## Files Changed

- `CONTEXT.md`: Newsletter and Issue domain language
- `src/workspace.mjs`: SQLite schema, scheduling, queue, claims, and projections
- `src/newsletter-worker.mjs`: claimed-Issue to Daily Run adapter
- `src/dashboard.mjs`: local server-rendered dashboard and HTTP safety
- `src/daily-run.mjs`: explicit run ID override and returned canonical Dossier
- `src/paths.mjs`: stable SQLite Workspace path
- `bin/learn.mjs`: `serve` and `worker` runtime roles
- `compose.yaml`: loopback dashboard and worker services
- `package.json`: v0.3 scripts and Node 22.13 minimum
- `README.md`, `docs/*`: test-phase operation, security, and architecture
- `test/*`: Workspace, scheduling, concurrency, worker, dashboard, and
  deployment coverage

## Commands Run

- focused `node --test` runs for Workspace, worker, dashboard, and Daily Run
- `npm test`
- `npm run check`
- `docker compose config --quiet`
- CLI help and no-warning regression check
- local two-process dashboard plus worker smoke test with two Newsletters
- Docker image build
- non-root/read-only container dashboard health check
- container dashboard create/queue, separate worker generation, and preview
  verification
- `git diff --check`
- tracked credential-pattern scan

## Tests

- 64 Node test-runner tests pass.
- Syntax checks pass for every CLI, source, and test module.
- SQLite initialization, safety pragmas, forward-version rejection, schedule
  DST behavior, idempotent dispatch, pause state, and two-connection claims are
  covered.
- Worker tests prove delivery adapters are empty and Newsletter artifacts and
  Learning History paths are isolated.
- Dashboard tests cover escaping, CSRF, validation, methods, missing IDs, and
  safe generated previews.
- Compose binds the dashboard to host loopback and keeps non-root/read-only
  controls.
- A local smoke test generated RabbitMQ and PostgreSQL Issues from two
  processes and opened both previews.
- The rebuilt container generated and previewed a queued Issue through a
  separate worker sharing the durable volume.

## Deviations From Plan

- Existing config is not automatically imported as a Newsletter. The dashboard
  begins empty and uses installation settings as defaults, which avoids
  surprising scheduled work or accidental duplication.
- Editing Newsletter fields is deferred; creation, pause/resume, Run Now,
  history, and preview cover the requested test loop.
- The work began as a stack on the approved v0.2 branch. After v0.2 merged, the
  review package and pull request were refreshed directly against `main`.

## Known Risks

- Node's built-in SQLite remains marked experimental and emits a warning in
  dashboard/worker processes, though it no longer requires a feature flag on
  Node 22.13+.
- The dashboard has no authentication and must remain on loopback or behind a
  trusted access layer.
- A worker crash can leave an Issue `generating` and a Daily Run lock requiring
  deliberate operator cleanup.
- SQLite lifecycle updates and immutable artifact files are not one
  cross-resource transaction; an orphan artifact is possible if a process dies
  after Daily Run persistence but before Issue completion.
- Schedule calculation scans up to eight days of UTC minutes. This is simple
  and correct for a small single-user workspace, not optimized for thousands of
  Newsletters.
- Existing file-based Daily Run records and Learning History remain separate
  from SQLite and are not imported into dashboard history.

## Next Steps

- Run the local test phase with real topics and confirm the dashboard workflow.
- Add explicit retry/recovery controls for failed or abandoned Issues.
- Add authentication before any public dashboard exposure.
- Connect Newsletter-level recipients and the existing Resend adapter after
  the test phase is accepted.
