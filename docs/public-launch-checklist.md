# Public launch checklist

Do not enable public signup until every staging-evidence item is attached to a
release.

## Implemented release gates

- [x] Source requests pin validated public IPs, revalidate redirects, reject
  credentials/private networks, and bound bytes and time.
- [x] Owner-scoped Postgres queries enforce Account isolation.
- [x] Issue generation uses fair, expiring Claims with recovery and bounded
  attempts.
- [x] Generation, artifact persistence, and delivery are separate durable
  phases; delivery uses stable idempotency keys and records unknown outcomes.
- [x] Clerk bearer authentication, signed lifecycle webhooks, exact-host
  routing, exact-Origin CSRF checks, and JSON-only mutations fail closed.
- [x] Username, creation, and manual-generation limits are durable.
- [x] Account deletion disables access and work before queued S3 cleanup.
- [x] Health/readiness, metrics, structured logs, migrations, and an operations
  runbook exist.

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
  revocation, public-site removal, stopped work, and eventual artifact cleanup.
- [ ] Exercise expired Issue and delivery Claims and confirm automatic recovery.
- [ ] Resolve an intentionally ambiguous provider response into
  `outcome_unknown` without an automatic duplicate email.
- [ ] Configure Postgres point-in-time recovery and S3 versioning/encryption,
  then complete and time an isolated restore drill.
- [ ] Configure alerts for readiness, 5xx rate, queue age, Claim recovery,
  exhausted attempts, delivery outcomes, model errors/latency, Postgres pool
  saturation, S3 errors, and spend controls.
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
