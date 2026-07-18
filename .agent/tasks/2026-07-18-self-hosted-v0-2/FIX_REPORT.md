# Fix Report

## Review disposition

The first independent review requested changes for four release blockers.

## Fixes

1. **Crash consistency:** Daily Runs now allocate a generation ID before
   persistence and write immutable generation-versioned Markdown/JSON files.
   The run-record pointer is swapped only after those files exist. A
   fault-injection test proves a crash before the swap preserves the earlier
   delivered generation and receipt.
2. **Lock ownership:** automatic stale reclamation and heartbeat mutation were
   removed to eliminate ownership races. Release first atomically moves the
   lock to an owner-specific candidate and validates the moved token. A crashed
   process leaves a lock that requires deliberate operator cleanup. Regressions
   prove old locks are never stolen and an old owner cannot delete a
   replacement lock.
3. **VM timeout:** the systemd oneshot now uses `TimeoutStartSec=infinity`;
   provider requests and retries remain bounded at the application layer. A
   deployment contract test protects this setting.
4. **Transport security:** credential-bearing model endpoints require HTTPS.
   Plain HTTP is accepted only for a loopback address with explicit
   `allowInsecureHttp: true`.

## Additional hardening

- Forced generations use a distinct Resend idempotency key.
- Duplicate delivery IDs are rejected.
- Email subjects are stripped of control characters and bounded.
