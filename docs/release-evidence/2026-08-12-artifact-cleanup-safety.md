# Durable artifact-cleanup safety — 12 August 2026

## Risk closed

Dossier artifacts are immutable S3 objects uploaded before the PostgreSQL Issue
completion transaction. A database failure after upload previously triggered a
best-effort delete only; a simultaneous object-store outage could strand
learner content and create untracked storage cost.

Schema migration 37 adds a durable cleanup intent for the exact future object
key. The worker registers that key before upload. Successful Issue completion
deletes the intent in the same transaction that records the artifact. Failed or
ambiguous upload/completion leaves the intent for an independent cleanup claim.

## Safety invariants

- Keys are constructed by the artifact package's validated namespace helper;
  arbitrary or traversal-like parts are rejected.
- Cleanup claims are expiring, `SKIP LOCKED`, idempotent, and retried with
  bounded exponential backoff.
- A key referenced by any Issue is excluded at claim time, so a delayed cleanup
  cannot delete a legitimately committed artifact.
- An originating Issue with an unexpired generation claim also blocks cleanup,
  protecting slow but healthy uploads before the artifact reference commits.
- An immediate successful compensation delete removes the queued intent; an
  ambiguous or failed delete leaves it durable.
- Queue depth, oldest overdue age, and successful cleanup count are exported as
  metrics. `LearnloomArtifactCleanupStale` warns when an unreferenced object is
  more than 15 minutes overdue for ten minutes.
- Account deletion remains a separate prefix-wide erasure workflow.

## Verification

- Artifact namespace unit tests: passed.
- Worker cleanup success/retry/metric regressions: passed.
- Migration ledger and migration 37 structure tests: passed.
- Fresh PostgreSQL 17 through migration 37: passed.
- Real PostgreSQL lifecycle proves committed cleanup removal, referenced-key
  protection, live-generation-lease protection, unreferenced-key claim, and
  claim completion.
- HTTP/worker metric and alert-rule regressions: passed.

## Staging evidence still required

Trigger an object-store deletion failure after a deliberately failed Issue
completion, retain the alert notification, restore object-store availability,
and verify the queue drains and the object is absent. This repository evidence
does not substitute for that staged failure exercise.
