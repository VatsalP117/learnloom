# Implementation Plan

## 1. Task Summary

Evolve Learnloom from one config-defined learning stream into a testable,
single-user multi-newsletter workspace. Users can create topic-focused
Newsletters, choose a daily local schedule, pause or resume them, queue Run Now,
and inspect generated Issue history and Dossier previews through a small local
dashboard.

## 2. Current System Understanding

The existing deep Daily Run module reliably generates one profile/date Dossier,
persists immutable artifacts, prevents overlap, and records delivery outcomes.
Configuration combines installation-wide provider/storage choices with one
profile's topic and sources. File JSON records cannot list Newsletters, claim
queued Issues, calculate schedules, or answer dashboard queries. The container
is currently a finite batch process with no web listener.

## 3. Scope

### In Scope

- `Newsletter` and `Issue` domain language
- A concrete deep SQLite Workspace using Node's built-in `node:sqlite`
- Idempotent schema initialization with foreign keys, WAL, and busy timeout
- Newsletter create/list/detail and active/paused state
- Daily local schedule plus IANA timezone, stored as next UTC occurrence
- Scheduled Issue dispatch and atomic queued-Issue claiming
- Manual Run Now Issues distinct from scheduled Issues
- Worker adapter around the existing Daily Run
- Dashboard overview, create form, detail/history, and safe Issue preview
- CSRF protection on dashboard mutations
- Localhost-only listener by default
- Separate `serve` and `worker` CLI/runtime roles
- Demo-mode end-to-end operation with delivery adapters forcibly disabled
- Existing CLI behavior and file artifacts retained
- Tests and local/container verification

### Out of Scope

- Live Resend calls, recipient management, or sent status
- Notion delivery
- Authentication or public internet exposure
- Multiple user accounts, billing, or subscriber lists
- Editing every Newsletter field after creation
- Replacing existing immutable Dossier files or Learning History
- Importing historical JSON Daily Runs into SQLite
- A JavaScript SPA or frontend framework

## 4. Proposed Technical Approach

Add one deep SQLite Workspace module whose interface expresses domain
operations: create/list/get/toggle Newsletters, enqueue manual Issues, dispatch
due schedules, claim queued Issues, and record generated/failed outcomes. Its
implementation owns schema, validation, schedule arithmetic, transactions, and
dashboard projections for locality and leverage.

Map `Newsletter.id` to the existing `profileId` internally so each Newsletter
retains isolated Learning History and artifact paths. Extend Daily Run with an
explicit run ID override so manual Issues on the same local date remain
distinct while sharing Newsletter history. A Newsletter runner claims an Issue,
builds a validated runtime config from the installation config and Newsletter
snapshot, invokes Daily Run with `deliveries: []`, then persists the Issue
outcome.

The web module only calls Workspace operations and renders server-side HTML.
The worker periodically dispatches due schedules and drains queued Issues.
Dashboard POST requests use an in-memory CSRF token, and the listener defaults
to `127.0.0.1`.

Daily schedule calculation scans real UTC minutes for the next matching local
wall time. This naturally skips nonexistent DST times; scheduled uniqueness
ensures a repeated wall time creates one Issue.

## 5. Step-by-Step Execution Plan

1. Record domain terms, task, plan, and architecture decision.
2. Add stable workspace path and SQLite Workspace schema/domain operations.
3. Add schedule calculation, due dispatch, and atomic Issue claiming.
4. Adapt claimed Issues into the existing Daily Run with delivery disabled.
5. Add safe server-rendered dashboard routes and CSRF-protected mutations.
6. Add `serve` and `worker` CLI commands while preserving existing commands.
7. Add Compose test-phase web/worker roles and documentation.
8. Add focused database, worker, HTTP, scheduling, isolation, and regression
   tests.
9. Run full checks, end-to-end two-Newsletter demo, container validation,
   security scan, independent review, and publish a stacked draft PR.

## 6. Test Plan

- SQLite initialization is idempotent and enables required pragmas.
- Newsletter validation covers names, topics, sources, timezone, and time.
- Two Newsletters remain isolated in Issues, history paths, and artifacts.
- Due dispatch respects timezone and pause state and is idempotent.
- DST gaps skip and repeated times cannot duplicate scheduled Issues.
- Manual Run Now creates distinct queued Issues.
- Atomic claim prevents two workers from claiming one Issue.
- Worker records generated and safely truncated failed outcomes.
- Worker passes no delivery adapters in the test phase.
- Dashboard overview/detail/preview routes render escaped content.
- POST mutations require CSRF; unknown routes return 404.
- Existing 39 tests continue to pass.
- Docker web binds only to host loopback and shares the durable volume.

## 7. Acceptance Criteria

- A user can start demo web and worker processes locally.
- The dashboard can create at least two topic-focused Newsletters with separate
  daily schedules.
- Run Now queues work and returns immediately.
- The worker generates each Issue without invoking Resend.
- The dashboard shows active state, next run, counts, latest status, Issue
  history, and a safe Dossier preview.
- Scheduled dispatch cannot duplicate one Newsletter's daily Issue.
- Existing CLI/Daily Run flows remain operational.
- Automated and container checks pass with no tracked secrets.

## 8. Risks and Guardrails

- `node:sqlite` requires Node 22.13+ and is still marked experimental in current
  LTS documentation; tighten the engine requirement and keep all usage local to
  one module.
- Authentication is excluded, so bind to loopback and document that the
  dashboard must not be publicly exposed.
- Generation is slow, so HTTP only enqueues Issues.
- Existing enabled deliveries could cause email; the Newsletter runner always
  passes an explicit empty adapter list.
- SQLite and immutable artifact writes are not one cross-resource transaction;
  mark an Issue generated only after Daily Run returns persisted paths.
- Preserve the current config/CLI contract and do not migrate or delete legacy
  files.

## 9. Executor Instructions

Implement thin commits: Workspace first, worker second, dashboard third, runtime
packaging/docs last. Use Node built-ins only. Keep SQL and schedule invariants
inside the SQLite Workspace. Do not implement Resend, auth, or Notion in this
slice.
