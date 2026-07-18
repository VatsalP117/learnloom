# Fix Report

## Issues Fixed

1. Overdue scheduled Issues now derive their local date from the stored
   `nextRunAt` occurrence rather than the worker's current time. A regression
   proves a missed yesterday schedule does not queue today's Issue early.
2. Dashboard requests now enforce an explicit Host allowlist before rendering
   any page or CSRF token. Loopback hosts are allowed by default, trusted proxy
   names require repeated `--allowed-host` options, and forged Host coverage
   prevents localhost DNS rebinding.
3. Worker completion and failure timestamps now come from a fresh clock reading
   after generation. Injected clocks keep tests deterministic.
4. SQLite claims now serialize generating Issues per Newsletter, preventing
   concurrent workers from racing on one Newsletter's shared Learning History.
   Different Newsletters remain independently claimable.

## Files Changed

- `src/workspace.mjs`, `test/workspace.test.mjs`
- `src/dashboard.mjs`, `test/dashboard.test.mjs`
- `src/newsletter-worker.mjs`, `test/newsletter-worker.test.mjs`
- `bin/learn.mjs`, `docs/dashboard-test-phase.md`

## Commands Run

- focused Workspace, dashboard, and worker tests
- `npm run check`
- `git diff --check`

## Remaining Issues

None from the first independent review.
