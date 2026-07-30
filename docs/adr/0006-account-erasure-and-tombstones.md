# ADR 0006: Account erasure and identity tombstones

Status: accepted
Date: 2026-07-30

## Context

Identity deletion previously revoked access and queued object-store cleanup,
but retained the Account's relational learning data indefinitely. Deleting the
Account row without another guard would let a delayed or replayed active
identity event recreate access.

## Decision

Deletion remains a two-boundary workflow:

1. The identity event immediately marks the Account deleted, removes public
   visibility, cancels pending work, and enqueues artifact erasure.
2. The worker idempotently removes the Account's object prefix.
3. Under the same database transaction that completes the deletion Claim,
   Learnloom records a SHA-256 identity tombstone, cascades the Account and all
   owned product data, and records a privacy-minimal erasure receipt.

The tombstone contains no email, Clerk ID, lesson text, or Account UUID. It is
retained while the service exists because it is required to honor the deletion
against stale identity events. Erasure receipts contain only a one-way Account
fingerprint and completion flags and expire after 400 days. Provider backup
expiration remains an external launch-evidence requirement.

## Consequences

- Active database content is erased only after object cleanup succeeds.
- Retrying object cleanup and completion remains safe.
- A stale active identity event cannot silently restore a deleted Account.
- Support can prove the two erasure boundaries completed without retaining the
  learner's content or direct identity.
- Restoring a database backup requires replaying deletion/tombstone state before
  restored services may accept traffic.
