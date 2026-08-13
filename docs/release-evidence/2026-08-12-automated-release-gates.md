# Automated release gates — 2026-08-12

Scope: exact working tree through schema migration 39. These checks close the
automated section they cover; they do not replace staging, second-person, or
release-owner evidence.

## Clean frontend install

Using Node `v24.14.0`, a complete source snapshot excluding only `.git`, local
`.env`, `node_modules`, and generated `web/dist` was installed and tested:

- `npm ci` — pass; 323 packages installed, zero audit findings.
- `npm run check` — pass: API contract, ESLint, TypeScript, production build.
- `npm test` — pass: 21 files, 65 tests.

The temporary snapshot was deleted after the run. The authoritative worktree
and existing dependency directory were not replaced.

## Go and PostgreSQL

- `go test -race ./...` — pass.
- `go vet ./...` — pass.
- Full `internal/store` suite against a fresh PostgreSQL 17 schema through
  migration 39 — pass.
- All aggregate operator reports execute against the migrated database:
  `scripts/product-baseline.sql`, `scripts/design-partner-beta-report.sql`, and
  `scripts/billing-economics.sql`.
- The release parity verifier passes isolated end-to-end HTTP tests for healthy
  matching releases and rejects stale marketing, wrong/mutable release IDs,
  insecure origins, weak HSTS, and incorrect app/canonical behavior.
- The staging load runner passes isolated workload tests and fails on error,
  p95 latency, achieved-rate shortfall, cross-origin redirect, missing forecast,
  or accidental production targeting. No repository benchmark is represented
  as real staging capacity evidence.
- Stable generation/source/claim failure codes are centralized and each maps
  to a named behavioral fixture. The product baseline has a real-user-only
  unregistered-code gate that must return no rows before `LL-201` can close.

## Dependency remediation

The first scans found fixable issues and the gate remained open until they were
removed:

- npm transitive advisories in `brace-expansion`, `js-yaml`, `nanoid`, and
  `postcss` were resolved by lockfile-only dependency updates.
- Trivy initially reported ten High findings in `golang.org/x/crypto v0.51.0`
  and `golang.org/x/text v0.37.0`. They were upgraded to `v0.52.0` and
  `v0.39.0`; the full Go suite and vet passed afterward.

Current results:

- `npm audit` — zero vulnerabilities.
- Production image Trivy `0.69.3`, current database, High/Critical,
  fixed findings included — zero OS and zero Go-binary findings.

## Production image

- Exact worktree image `learnloom:prelaunch-audit` built successfully.
- Image manifest digest at verification:
  `sha256:1693c3153209897066df85807683498c438306f9ca66a6e8083add468118feb8`.
- Final image is pinned distroless Debian 12 and configured as
  `nonroot:nonroot`; executing `/learnloom` produced the expected CLI usage.
- The image contains the static frontend and stripped CGO-disabled Go binary;
  local `.env*` is excluded by `.dockerignore`.

## Secret scan

Gitleaks `v8.28.0` scanned a complete release source snapshot excluding only
the same non-release state as the Docker context: `.git`, `.env`, dependencies,
and generated output. Result: no leaks.

A deliberate scan of the raw working directory found two candidates only in
the local `.env`. That file is git-ignored and Docker-ignored; it was not
printed, changed, copied into evidence, or included in release scanning. This
is an expected operator secret store, not source material.

## Still open

- Real staging configuration and lifecycle tests.
- Authenticated runtime health using staging dependencies.
- A second-person security/privacy/recovery review.
- Explicit release-owner approval for public signup.
