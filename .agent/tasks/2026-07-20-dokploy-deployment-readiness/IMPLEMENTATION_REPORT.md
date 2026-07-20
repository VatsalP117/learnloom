# Implementation Report

## Summary

Prepared Learnloom for a self-contained production deployment on an existing
Dokploy VM.

- Added a Dokploy Compose stack for the web, worker, migration, Postgres,
  MinIO, SearXNG, and Valkey roles.
- Added an explicit production-only private-service transport exception that
  remains off by default and cannot relax checks for public dependency hosts.
- Parameterized frontend domain behavior at build time.
- Added a production environment template and a complete DNS, wildcard TLS,
  Clerk, Dokploy, backup, verification, and troubleshooting guide.
- Corrected the production runbook's health endpoint names.

## Files Changed

- `compose.dokploy.yaml`: production Dokploy topology.
- `.env.dokploy.example`: production variable inventory.
- `docs/dokploy-deployment.md`: operator runbook.
- `infra/searxng/Dockerfile`: packages the checked-in SearXNG configuration.
- `internal/config/config.go` and tests: private dependency transport opt-in.
- `web/src/config.js` and frontend consumers: build-time root domain.
- `Dockerfile`, `compose.yaml`, `.env.example`: build argument wiring.
- `README.md`, `docs/operations.md`: deployment links and health corrections.

## Commands Run

- `gofmt -w internal/config/config.go internal/config/config_test.go`
- `go test ./internal/config`
- `npm test`
- `npm run check`
- `docker compose --env-file .env.dokploy.example -f compose.dokploy.yaml config`
- `go test -race ./...`
- `go vet ./...`
- `docker build ... -t learnloom:deployment-check .`
- `docker build -t learnloom-searxng:deployment-check infra/searxng`
- `git diff --check`

## Tests

All commands passed:

- Go race suite: pass
- Go vet: pass
- Frontend tests: 5 tests across 2 files passed
- ESLint and Vite production build: pass
- Dokploy Compose rendering: pass
- Main production image build: pass; final image runs as `nonroot:nonroot`
- SearXNG configuration image build: pass
- Whitespace validation: pass

## Deviations From Plan

No product, authentication, database schema, or public API changes were made.
The Dokploy stack uses private Compose-network transport for self-hosted
Postgres and MinIO, guarded by the new explicit configuration switch.

## Known Risks

- Wildcard routing and certificate attachment vary across Dokploy versions.
  The runbook requires Preview Compose verification and includes a Traefik file
  fallback.
- Stateful Postgres and MinIO remain on the same VM; off-VM backups are
  mandatory before public signup.
- Real provider credentials, DNS, TLS, webhook delivery, and end-to-end
  generation cannot be verified without access to the user's services.

## Next Steps

Merge/push the branch, populate Dokploy secrets, configure DNS/TLS and Clerk,
deploy, then execute the documented external smoke test and launch checklist.
