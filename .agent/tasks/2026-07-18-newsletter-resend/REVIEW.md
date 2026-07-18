# Codex Review

## Verdict

`REQUEST_CHANGES`

## Summary

The feature separation, CSRF coverage, and output escaping are sound, but three
delivery/migration edge cases must be resolved before publication.

## Blocking Issues

1. Concurrent version 1 to version 2 startup can race because `user_version` is
   read before the migration write lock. Read and migrate under one
   `BEGIN IMMEDIATE` transaction and test concurrent startup.
2. Disabling Newsletter email does not stop already-pending receipts. Add a
   durable cancelled state, cancel unclaimed pending/failed receipts
   atomically on disable, and require enablement for claim and retry. Re-enable
   must not revive historical cancelled receipts.
3. Transport timeouts/lost responses and unusable successful Resend responses
   are ambiguous, but are currently marked failed and exposed to ordinary
   retry. Record a non-retryable unknown outcome instead and test acceptance
   followed by response loss.

## Non-Blocking Suggestions

None required for this phase.

## Test Gaps

- Concurrent dashboard/worker schema migration startup
- Disable before claim, disable before retry, and re-enable behavior
- Ambiguous provider acceptance followed by response loss

## Risk Areas

- SQLite schema migration locking
- Owner intent when email is disabled
- Duplicate mail after a provider response is lost

## Exact Fix Instructions for Executor

1. Acquire the SQLite immediate transaction before reading `user_version`;
   re-read and conditionally migrate inside that transaction.
2. Add `cancelled` to Delivery Receipt states. On disabling, atomically cancel
   pending and failed receipts. Require an enabled Newsletter in claim/retry,
   leave an in-flight `delivering` receipt visible, and never revive cancelled
   receipts automatically.
3. Add an `unknown` Delivery Receipt state. Classify Resend request transport
   failures and unusable 2xx responses as ambiguous; record them as unknown and
   do not offer standard retry. Keep definite configuration/rejection failures
   retryable.
4. Add the focused tests above and rerun the complete suite.

