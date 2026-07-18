# Fix Report

## Review disposition

The first independent review requested changes for four release blockers.

## Fixes

1. **Crash consistency:** Daily Runs now allocate a generation ID before
   persistence and write immutable generation-versioned Markdown/JSON files.
   The run-record pointer is swapped only after those files exist. A
   fault-injection test proves a crash before the swap preserves the earlier
   delivered generation and receipt.
2. **Lock ownership:** the run lock is now an owner-token lease with a
   heartbeat. Stale reclamation compares the observed lease after an atomic
   rename, and release removes a lock only when ownership still matches. A race
   regression proves an old owner cannot remove a reclaimed lease.
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
