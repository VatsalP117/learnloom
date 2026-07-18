# Fix Report

## Issues Fixed

1. Moved schema version detection and migration under one `BEGIN IMMEDIATE`
   transaction. SQLite's busy timeout is installed before WAL activation, and
   WAL activation has a bounded busy retry. Concurrent dashboard/worker startup
   now serializes before reading `user_version`.
2. Added durable `cancelled` Delivery Receipts. Disabling email atomically
   cancels pending and failed receipts; claim and retry require an enabled
   Newsletter; re-enable does not revive old receipts; an in-flight receipt
   stays visible.
3. Added durable, non-retryable `unknown` Delivery Receipts for transport loss
   or an unusable successful Resend response. Both Newsletter workers and the
   legacy Daily Run refuse ordinary retry after an ambiguous outcome.

## Files Changed

- `src/workspace.mjs`
- `src/delivery.mjs`
- `src/newsletter-worker.mjs`
- `src/daily-run.mjs`
- `src/dashboard.mjs`
- `bin/learn.mjs`
- `test/workspace.test.mjs`
- `test/newsletter-worker.test.mjs`
- `test/daily-run.test.mjs`
- `README.md`
- `docs/dashboard-test-phase.md`
- `docs/architecture.md`

## Commands Run

- `node --test test/workspace.test.mjs`
- `node --test test/workspace.test.mjs test/newsletter-worker.test.mjs
  test/delivery.test.mjs test/dashboard.test.mjs`
- `npm test`
- `npm run check`
- `git diff --check`
- Twenty consecutive runs of the two-process schema migration test

## Remaining Issues

Automatic reconciliation of an `unknown` Resend outcome and stale in-flight
claim recovery remain deliberately out of scope. Neither state is exposed to
ordinary retry.
