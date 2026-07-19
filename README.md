# Learnloom

Learnloom is a hosted learning service that turns trusted web sources into
durable Knowledge Dossiers. This repository is SaaS-only: it has no local CLI,
filesystem state mode, offline demo, or Command Code integration.

## Runtime

One Go binary exposes three explicit roles:

- `learnloom web` serves the marketing site, authenticated control plane,
  learner subdomains, health endpoints, and Clerk webhooks.
- `learnloom worker` schedules and claims Issues, fetches sources, generates
  Dossiers through an OpenAI-compatible API, stores artifacts, and delivers
  email.
- `learnloom migrate` applies embedded, transactional Postgres migrations.

Postgres is the system of record. S3-compatible object storage owns immutable
Dossier artifacts. Clerk owns identity, and Resend is the email boundary.

See [architecture](docs/architecture.md), [architecture decisions](docs/adr),
and the [production runbook](docs/operations.md).

## Development

Requirements: Go 1.25.12+, Node 24+, Docker with Compose.

```sh
cp .env.example .env
# Replace every placeholder in .env.
npm ci
npm run check
go test ./cmd/... ./internal/...
docker compose config
docker compose up --build
```

For local hostname routing, map `learnloom.test`, `app.learnloom.test`, and a
test learner subdomain to `127.0.0.1`, then terminate local TLS in a reverse
proxy. Clerk must be configured with the exact app origin and its webhook must
target `/webhooks/clerk`.

Run a real Postgres lifecycle test with:

```sh
TEST_DATABASE_URL='postgres://learnloom:password@localhost:5432/learnloom?sslmode=disable' \
  go test ./internal/store -run TestPostgresLifecycleIntegration -v
```

## Configuration

Configuration is environment-only. Startup fails if a role is missing a
required secret or dependency. The full local template is
[`.env.example`](.env.example); production secrets belong in a managed secret
store, never an image, repository, or Compose file. On AWS, omit static S3
credentials and use the workload's IAM role.

Any HTTPS service exposing OpenAI-compatible `/models` and `/chat/completions`
endpoints can be used through `MODEL_BASE_URL`, `MODEL_API_KEY`, and
`MODEL_NAME`. DeepSeek is the default, not a special provider implementation.

## Safety properties

- Account ownership is enforced in every control-plane query.
- Mutations require a Clerk bearer session, exact Origin, JSON content type,
  and session-bound CSRF token.
- Source acquisition rejects credentials, private/reserved addresses, unsafe
  redirects, and oversized responses.
- Work and delivery use expiring claims, bounded attempts, idempotency keys,
  and explicit unknown-delivery outcomes.
- Public artifacts are rendered with a restrictive CSP and escaped model text.
- Account deletion stops future work and deletes artifact objects through a
  durable queue.

## Verification

```sh
npm run check
go test -race ./cmd/... ./internal/...
go vet ./...
docker build --build-arg VITE_CLERK_PUBLISHABLE_KEY=pk_test_placeholder .
```

Deployment still requires real DNS/TLS, provider credentials, backups,
monitoring, and a staging smoke test. The release gates live in
[the launch checklist](docs/public-launch-checklist.md).
