# 08 — How to change Learnloom safely and develop locally

## Clean-machine local development

### Prerequisites

- Git.
- Go 1.25.12 or newer compatible patch version (`go.mod`, Dockerfile).
- Node.js 24+ and npm (`package.json`).
- Docker Engine with Compose v2.
- A Clerk development instance, model API key, and Resend test/verified sender
  for full-stack behavior. Source discovery additionally needs SearXNG profile.
- Local DNS and TLS proxy capable of apex/app/wildcard hosts. The repository
  names `learnloom.test`, `app.learnloom.test`, and a learner subdomain but does
  not provide proxy config.

### Install and configure

```sh
git clone https://github.com/VatsalP117/learnloom.git
cd learnloom
cp .env.example .env
npm ci
```

Replace every placeholder in `.env`; do not commit it. Key local relationships:

- root/app/Vite domains must agree;
- Vite and server Clerk publishable keys must match the same Clerk instance;
- database URL must address Compose Postgres from the process that uses it;
- MinIO credentials/bucket/endpoint/path-style must agree;
- Clerk allowed origin is exact HTTPS app origin; webhook points to
  `https://app.<root>/webhooks/clerk`;
- set `SOURCE_DISCOVERY_ENABLED=false` unless starting the discovery profile.

Map the apex, app, and one valid username host to `127.0.0.1`. Configure a local
trusted certificate/reverse proxy to forward them to `127.0.0.1:3000` while
preserving Host. Exact commands are OS/proxy-specific and absent from repo.

### Verify code before services

```sh
npm run check
npm test
go test ./cmd/... ./internal/...
go vet ./...
docker compose config
```

### Start

```sh
docker compose up --build
```

For discovery:

```sh
docker compose --profile discovery up --build
```

> [!WARNING] Current-revision blocker
> Migration 005 exists but the expected schema is hardcoded to 4, so the stack
> will not be healthy after migrations. Fix TD-001 before running this flow.

**Current-revision warning:** migration 005 versus expected schema 4 prevents
readiness after migrations. Fix TD-001 first or the stack will not be healthy.

### Real integration tests

Postgres:

```sh
TEST_DATABASE_URL='postgres://learnloom:password@localhost:5432/learnloom?sslmode=disable' \
  go test ./internal/store -run TestPostgresLifecycleIntegration -v
```

S3 integration requires `TEST_S3_ENDPOINT`, `TEST_S3_BUCKET`,
`TEST_S3_ACCESS_KEY_ID`, and `TEST_S3_SECRET_ACCESS_KEY`; inspect
`internal/artifact/integration_test.go` and use a disposable bucket.

### Frontend-only modes

`npm run dev` starts Vite but authenticated APIs still need a compatible web
origin/proxy. `npm run demo` provides development-only fixture behavior and is
useful for visual work, not backend correctness or integration validation.

### Reset and common errors

- To reset **disposable local** data, use `docker compose down` and explicitly
  decide whether to add `--volumes`; that deletes local Postgres/MinIO/Valkey
  data and is irreversible. Never use this against production contexts.
- “configuration: X is required”: Compose did not receive `.env` value.
- TLS/origin/CSRF failures: verify exact `https://app.<root>`, proxy Host and
  rebuilt Vite environment.
- authentication loop: Clerk allowed origins/callback/publishable key mismatch.
- discovery insufficient: confirm discovery profile/SearXNG, then inspect
  source endpoint/discovery-run evidence; do not disable SSRF checks.
- readiness schema mismatch: compare `schema_migrations` max to
  `currentSchemaVersion`.
- artifact unavailable: confirm MinIO bucket creation and path-style endpoint.

## Universal safe-change workflow

```mermaid
flowchart LR
  A["Identify owning capability + read tests/ADR"] --> B["Trace UI DTO → handler validation → owner-scoped store → schema/side effect"]
  B --> C["State invariants + failure/retry behavior"]
  C --> D["Smallest cohesive change — no generic layers"]
  D --> E["Regression at lowest level + cross-boundary integration"]
  E --> F["Format / static / unit / integration / build"]
  F --> G["Schema/config/deploy? Rehearse migration + rollback, update handbook"]
```

1. Identify the owning capability and read its tests/ADR.
2. Trace UI DTO → handler validation → owner-scoped store operation → schema/
   external side effect → response projection.
3. State invariants and failure/retry behavior before editing.
4. Make the smallest cohesive change; do not add generic layers.
5. Add regression at the lowest useful level plus cross-boundary integration.
6. Run format/static/unit/integration/build checks.
7. For schema/config/deploy changes, rehearse migration/rollback and update this
   handbook/ADR.

## Add a new authenticated page

Likely files: `web/src/App.tsx`, a PascalCase page, `LearningShell.tsx` for nav,
DTO in `types.ts`, hooks/API, CSS and colocated tests.

Preserve same-origin anchor interception and lazy-load large pages. Decide
whether data belongs in the workspace snapshot, a separate paginated endpoint,
or local-only state. Do not read owner IDs from URL/browser as authorization.
Test direct-load (Go index fallback), back/forward navigation, signed-out
redirect, loading/error/empty/accessibility, and mobile layout. Build with
production Vite vars.

## Add an API endpoint

1. Define method/path/request/response/errors/auth/idempotency.
2. Add route in `handleControl()` or a public host handler. All `/api` routes
   automatically get Clerk; mutations get Origin/CSRF/JSON.
3. Decode through `decodeJSON`; validate sizes/enums. Use a dedicated response
   DTO rather than exposing internal secrets/domain fields.
4. Add an owner-scoped store method. Include Account ID in the SQL predicate,
   even if upstream loaded an owned parent.
5. Put multi-row invariant in one transaction; map expected sentinel errors.
6. Add cross-tenant integration and HTTP policy/contract tests; update chapter
   02. Rate-limit abuse/cost-heavy actions.

## Add a database field

Use next numbered embedded migration. Prefer additive nullable/defaulted change
that old code tolerates. Update scan/select/insert/update/domain/DTO/tests.
Critically, update or derive `currentSchemaVersion`. Test empty and populated
upgrade, `Store.Ready()`, constraints, old-image compatibility and rollback
plan. Deploy expand → code → contract; never remove/rename in the same release.

## Add a table

Define owner/data lifecycle, PK/FK/delete behavior, constraints and indexes
from actual queries. If tenant data, decide how every query proves ownership
and how Account deletion handles it. Add repository methods with transaction
boundaries, migration lifecycle/integration tests, backup/retention/privacy
documentation, and expected schema version.

## Add or change a business rule

Locate every enforcement layer: form utility, handler, store normalization,
transaction predicate, DB constraint, worker/generator quality. Pick one
canonical enforcement point and add defense-in-depth only where it prevents
invalid stored state or improves UX. Test disagreement and migration of
existing rows. Do not rely on frontend checks.

## Add a background job

Current scheduling model is a worker polling a durable Postgres state machine.
Add explicit status/available/claim token/expiry/attempt/idempotency fields,
`SKIP LOCKED` claim, lease renewal if long-running, known versus ambiguous
outcomes, backoff/dead-letter/manual retry, drain behavior, metrics and cleanup.
Wire in `runWorker()`. Do not put long work in web handlers. Decide whether it
shares worker concurrency or needs separate role via ADR.

## Add an external integration

Write an ADR when it creates production variability. Define data sent,
credentials/rotation, endpoint validation, timeout, retryable statuses,
idempotency, ambiguous outcome, rate/cost budget, redaction, readiness and
provider outage behavior. Construct adapter in `main.go`, expose a narrow
capability interface at the consumer, and use fake HTTP server tests. Add config
validation and production secret injection—never a browser `VITE_*` secret.

## Add an environment variable

Add typed field, `Load` read/default, strict role validation, tests, `.env.example`,
both relevant Compose files, Docker build arg only if intentionally public, and
chapter 04. Decide missing/malformed behavior and rotation. Avoid two names for
one value unless build-time/server-time separation is necessary.

## Change authentication

This spans `ProductRoot`, `HostedApp`, `AuthPage`, `api.ts`, CSP, server Clerk
middleware/token extraction, Account projection/webhook, config and provider
dashboard. Preserve the distinction between authentication and SQL ownership,
mutation CSRF/Origin, verified-email delivery, event ordering/idempotency,
deletion/suspension, and public behavior. Require a recorded ADR, threat model,
cross-tenant suite, session expiry/refresh/logout/SSO tests and staged migration.

## Change source acquisition/discovery

Read ADR-0005 and all `internal/source` tests. Never turn search snippets into
evidence or allow a generic HTTP client around `secureTransport`. Preserve
public-address checking for all DNS answers and redirects, byte/time/content
bounds, immutable snapshots, hard evidence minimum, and retry freeze. Add
malicious URL/DNS/redirect fixtures and repository integration tests.

## Change Dossier/model pipeline

Increment `PipelineVersion` whenever checkpoint semantics/contracts/prompts make
old output unsafe. Decide which settings enter `GenerationFingerprint`. Add
contract validators and quality tests before prompt changes. Preserve citation
provenance, duration budget, learner history, editor fallback, safe rendering,
and stage failure classification. Measure quality/cost on representative
fixtures; model name changes also invalidate checkpoints.

## Change deployment

Update Docker/Compose/docs together. Validate `docker compose config`, build
image, migration against production-like copy, drain worker, all health hosts,
wildcard route/TLS, rollback compatibility, secrets and backup. Do not expose
worker metrics/Postgres/SearXNG. Add resource bounds based on measurements.

## Upgrade a major dependency

Read release/migration/security notes from the primary upstream project. Update
lockfile using the package manager, not hand edits. Run all CI commands plus
targeted integration/E2E. Clerk: auth/CSP/callback; pgx: transactions/pool;
AWS: endpoint/checksum; React/Vite/TS: build/router/hydration; PostgreSQL:
migrations/query plans/backup restore. Roll out staging first and retain a
schema-compatible rollback image.
