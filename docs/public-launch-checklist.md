# Public launch checklist

Do not enable public signup until every staging-evidence item is attached to a
release.

## Implemented release gates

- [x] Source requests pin validated public IPs, revalidate redirects, reject
  credentials/private networks, and bound bytes and time.
- [x] Owner-scoped Postgres queries enforce Account isolation.
- [x] Issue generation uses fair, expiring Claims with recovery and bounded
  attempts.
- [x] Issue failures use classified internal detail and learner-safe
  projections; Issue Attempts and Dossier stages remain durably inspectable.
- [x] Worker shutdown drains active Claims and forced release does not consume
  a content-generation attempt.
- [x] Generation, artifact persistence, and delivery are separate durable
  phases; delivery uses stable idempotency keys and records unknown outcomes.
- [x] Clerk bearer authentication, signed lifecycle webhooks, exact-host
  routing, exact-Origin CSRF checks, and JSON-only mutations fail closed.
- [x] Username, creation, and manual-generation limits are durable.
- [x] Account deletion disables access and work before queued S3 cleanup,
  then transactionally erases active database learning data and records a
  privacy-minimal receipt.
- [x] Health/readiness, metrics, structured logs, migrations, and an operations
  runbook exist.
- [x] Web metrics run on a separate operations listener and CI exercises the
  canonical artifact lifecycle against a pinned S3-compatible service.
- [x] Provider tokens, retries, stage latency, estimated spend, and learning
  outcomes feed durable recording rules and a versioned operations dashboard;
  claim admission enforces the configured daily reservation budget.
- [x] A restore harness refuses non-drill databases and verifies schema, row,
  and sampled artifact checksum evidence.

## Required staging evidence

- [ ] Configure apex, `www`, `app`, Clerk, and wildcard DNS with end-to-end TLS.
- [ ] Complete a two-Account browser test covering sign-in, username Claim,
  creation, manual generation, publication, private mode, cross-tenant denial,
  delivery, retry, and sign-out.
- [ ] Prove a staging source request cannot reach cloud metadata or internal
  services, including through redirects and DNS changes.
- [ ] Prove request limits return stable `429` responses without creating work,
  and demonstrate fair worker progress across Accounts.
- [ ] Delete and suspend a real staging Clerk user; verify immediate access
  revocation, public-site removal, stopped work, artifact cleanup, relational
  erasure, and a non-identifying erasure receipt.
- [ ] Exercise expired Issue and delivery Claims and confirm automatic recovery.
- [ ] Resolve an intentionally ambiguous provider response into
  `outcome_unknown` without an automatic duplicate email.
- [ ] Configure Postgres point-in-time recovery and S3 versioning/encryption,
  then complete and time an isolated restore drill.
- [ ] Configure alerts for readiness, 5xx rate, queue age, Claim recovery,
  exhausted attempts, delivery outcomes, model errors/latency, Postgres pool
  saturation, S3 errors, and spend controls.
- [ ] Import the versioned Prometheus rules and Grafana dashboard, route alerts,
  and test warning and critical delivery in staging.
- [ ] Verify HSTS at the public edge, wildcard DNS/TLS renewal, scoped provider
  IAM, staging credential isolation, and cache purge after a privacy change.
- [ ] Verify privacy changes are not served by any public cache past the
  documented purge window.
- [ ] Run load and soak tests at the expected launch concurrency.

## Automated sign-off

- [ ] `npm ci && npm run check && npm test` passes.
- [ ] `go test -race ./...` and `go vet ./...` pass.
- [ ] The real Postgres lifecycle integration test passes.
- [ ] The production container builds, runs as non-root, and passes an image
  vulnerability scan.
- [ ] Dependency and secret scans pass.
- [ ] A second person reviews security, privacy, and recovery evidence.
- [ ] The release owner explicitly approves public signup.
