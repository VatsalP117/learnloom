# Source retrieval policy response control — 13 August 2026

Scope: repository implementation through schema migration 040. This evidence
covers the technical future-retrieval hold; it does not claim counsel approval,
a completed staging complaint drill, or a rights determination.

## Implemented boundary

- Migration 040 stores immutable `block` and `unblock` events for an exact URL
  or registrable domain. The current state is derived by a view; prior actions
  are not updated or deleted.
- Every action requires an active operator Account, a bounded non-identifying
  case reference and reason, and retains actor and timestamp for authorized
  audit. The current-policy view intentionally omits the reason.
- `cmd/source-policy` validates the exact target twice and defaults to dry-run.
  Applying an action requires an explicit `-apply`; reversing a decision is a
  new audited `unblock` event.
- Initial feed/article requests, discovered and cached candidate selection,
  feed-item article requests, and every redirect consult the same policy.
  Policy read failures fail closed. URL canonicalization strips fragments,
  normalizes IDNs/default ports, preserves paths and queries, and safely
  supports exact public IPv4/IPv6 targets.
- The rights-holder runbook uses the narrowest reviewed target and keeps
  requester identity and legal analysis in the approved case system rather
  than command arguments or repository state.

## Verification

Passed in the exact worktree:

```sh
npm run check
npm test
go vet ./...
go test -race ./internal/store ./cmd/source-policy ./cmd/learnloom
go test -race ./internal/source -run '<policy and non-listener source tests>'
git diff --check
```

Frontend result: 21 files and 68 tests passed. The changed Go packages and
policy/source selection tests passed with the race detector. The store
integration lifecycle is present and remains fail-closed when
`TEST_DATABASE_URL` is absent.

An attempted `go test ./...` in the managed desktop sandbox reached all
packages but could not execute pre-existing `httptest` cases because loopback
port binding is prohibited (`operation not permitted`). This is an environment
restriction rather than a test assertion failure. The last full unrestricted
race run through migration 039 remains recorded in
`2026-08-12-automated-release-gates.md`; migration 040 still requires a fresh
PostgreSQL integration run in an environment with `TEST_DATABASE_URL` before
release evidence is complete.

## Still open

- Run the synthetic rights-holder workflow in staging, including exact public
  hold, retrieval block, redirect enforcement, authorized unblock or permanent
  hold, public cache purge, and final probes.
- Run the complete race and PostgreSQL integration suites through migration 040
  outside the managed no-listener sandbox.
- Obtain counsel review and second-person security/privacy/recovery review.
