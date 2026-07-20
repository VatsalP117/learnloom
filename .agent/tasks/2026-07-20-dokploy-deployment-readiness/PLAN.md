# Implementation Plan

## 1. Task Summary

Make the existing hosted Learnloom architecture deployable as a Dokploy Docker
Compose service on the user's VM and provide a complete operator runbook.

## 2. Current System Understanding

Learnloom is a single immutable image with `web`, `worker`, and `migrate`
roles. The backend already routes the apex, `www`, `app`, and one-level
username subdomains. Postgres is the mutable system of record, S3-compatible
storage holds generated artifacts, Clerk provides authentication, Resend sends
email, and the worker calls an OpenAI-compatible model.

The local Compose stack is development-oriented: it publishes loopback ports
and uses private HTTP/non-TLS service URLs. Production validation rejects those
URLs, so a self-contained production VM deployment needs an explicit,
restricted acknowledgement of private-network transport.

## 3. Scope

### In Scope

- Add an explicit production configuration switch permitting non-TLS Postgres
  and S3-compatible endpoints only when their host is local/private.
- Keep encrypted external dependencies mandatory by default.
- Add tests for the private-service exception and its rejection of public
  endpoints.
- Add a Dokploy Compose manifest with private stateful services, one-shot
  migrations, web/worker roles, health checks, and optional SearXNG discovery.
- Make the frontend root domain a build-time setting rather than a hardcoded
  hostname.
- Add a production environment template and a complete Dokploy deployment
  runbook covering DNS, wildcard TLS, Clerk, secrets, deployment, backups, and
  smoke tests.

### Out of Scope

- Connecting to or mutating the user's VM, DNS provider, Clerk, Resend, model,
  or Dokploy account.
- Changing authentication, database schema, or product behavior.
- Enabling public signup without completing the existing launch checklist.

## 4. Proposed Technical Approach

Introduce `ALLOW_INSECURE_PRIVATE_SERVICES`, defaulting to false. In production,
it may relax transport validation only for URLs whose hostname is loopback,
private/link-local IP, `localhost`, or a single-label container DNS name.
Public hostnames remain required to use TLS.

Create a separate `compose.dokploy.yaml` so local development behavior is not
changed. Attach only the web role to Dokploy's ingress network; keep Postgres,
MinIO, the worker, and discovery services on an internal network.

Pass `VITE_LEARNLOOM_ROOT_DOMAIN` into the frontend build and use it for host
classification and personal-site URL display.

## 5. Step-by-Step Execution Plan

1. Add and test private-service transport validation.
2. Parameterize frontend domain references and Docker build arguments.
3. Add the Dokploy Compose manifest and production environment template.
4. Write the Dokploy deployment guide.
5. Run Go, frontend, Compose, and container-build verification.
6. Review the final diff and record results.

## 6. Test Plan

- `go test ./internal/config`
- `go test -race ./...`
- `go vet ./...`
- `npm test`
- `npm run check`
- Render `compose.dokploy.yaml` with placeholder production variables.
- Build the production image when the local Docker daemon is available.

## 7. Acceptance Criteria

- Production defaults still reject unencrypted public dependencies.
- The explicit override accepts only private/local dependency hosts.
- The image builds with `learnloom.blog` as the frontend domain.
- Dokploy can route port 3000 to the web service without exposing stateful
  ports.
- Migrations complete before the long-running roles start.
- The runbook contains actionable DNS, wildcard certificate, provider,
  deployment, and verification steps.

## 8. Risks and Guardrails

- The private-service override must never accept a normal public DNS hostname.
- Wildcard DNS is not sufficient by itself; wildcard TLS must also be
  configured.
- Build-time Clerk and domain values require a rebuild when changed.
- Stateful named volumes require an explicit backup plan before launch.

## 9. Executor Instructions

Keep changes limited to deployment readiness. Do not commit secrets, change
schema/auth behavior, or expose Postgres, MinIO, SearXNG, or worker metrics to
the public internet.
