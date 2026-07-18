# Implementation Plan

## 1. Task Summary

Deepen Learnloom's Daily Run, add an OpenAI-compatible HTTP model adapter,
introduce durable idempotent delivery with Resend, stabilize storage paths, and
package the finite process for Docker plus a host systemd timer.

## 2. Current System Understanding

The CLI currently composes the complete run lifecycle. Model invocation supports
Command Code and a deterministic demo adapter. Generated Markdown and Learning
History use caller-relative paths. Delivery is absent, scheduling is macOS-only,
and the repository has no container, VM guide, public license, or environment
template.

## 3. Scope

### In Scope

- A deep Daily Run module used by the CLI
- A canonical Dossier retained alongside Markdown
- OpenAI-compatible HTTP model adapter with environment-only key lookup,
  bounded retries, timeouts, model discovery diagnostics, and useful errors
- Command Code and demo compatibility
- Stable application paths derived from `LEARNLOOM_HOME` or the configuration
  directory
- Durable per-day run records and Delivery Receipts
- Resend email adapter with safe HTML/text rendering and idempotency keys
- Failed-delivery retry without regenerating the Dossier
- Dockerfile, Compose file, systemd service/timer examples, and VM guide
- `.env.example`, permissive MIT license, public-oriented README/config
- Focused unit and end-to-end tests

### Out of Scope

- Notion delivery
- Multi-user hosting, login, billing, or a web dashboard
- PostgreSQL or a network database
- Full article extraction and non-feed Source Item adapters
- Inbound email or one-click learner feedback
- Automatic cloud provisioning

## 4. Proposed Technical Approach

Keep Node.js built-ins only. The Daily Run module owns source retrieval,
generation, persistence, and delivery ordering. It persists the Dossier before
external delivery and uses a deterministic profile/date run ID. Repeated runs
reuse the artifact and retry only unfinished deliveries unless forced.

The model seam has three adapters: Command Code process, OpenAI-compatible
HTTP, and deterministic demo. Keys are named by configuration but read only
from the environment. The delivery seam initially has Resend and a deterministic
test adapter. Run records remain atomic JSON for this single-process release;
their interface is shaped so SQLite can replace the implementation later.

The container runs once and exits. A host systemd timer invokes it daily and a
persistent volume owns application state.

## 5. Step-by-Step Execution Plan

1. Record domain language, task scope, and acceptance criteria.
2. Deepen configuration and application paths.
3. Add the HTTP model adapter and provider-aware diagnostics.
4. Refactor orchestration behind the Daily Run module and persist canonical
   Dossiers plus per-day run records.
5. Add Resend rendering/delivery and retry semantics.
6. Update CLI commands and offline demo.
7. Add Docker, Compose, systemd, environment, licensing, and deployment docs.
8. Run focused tests, full checks, container validation, and security scans.
9. Build a review package, review against `main`, fix blockers, and publish a
   draft pull request.

## 6. Test Plan

- Configuration tests for provider, storage, delivery, and environment names
- HTTP provider tests for success, auth absence, malformed responses, retryable
  status codes, model diagnostics, and secret-safe errors
- Resend tests for HTML escaping, request shape, idempotency, and failure modes
- Daily Run integration tests for generation, persistence, same-day reuse,
  failed-delivery retry, and force regeneration
- Path tests independent of the caller's working directory
- Existing feed, pipeline, schedule, and demo tests
- Syntax checks and `git diff --check`
- Docker image build and container demo/doctor where available
- Tracked-file scan for key-shaped secrets

## 7. Acceptance Criteria

- A new user can clone, copy configuration/environment examples, set a model
  key, and pass `doctor`.
- Direct DeepSeek and other OpenAI-compatible endpoints require no Command Code
  installation.
- A generated Dossier is persisted before email delivery.
- Re-running the same date does not regenerate or duplicate successful email.
- A failed email can retry without spending model tokens again.
- The container is finite and stores durable files in a mounted application home.
- VM scheduling instructions work through systemd.
- Existing local Command Code usage remains supported.
- All automated checks pass and no secrets are tracked.

## 8. Risks and Guardrails

- Provider schemas vary: support the common Chat Completions contract and keep
  provider-specific behavior inside the adapter.
- Network retries can duplicate side effects: retry model reads, but rely on
  Resend's deterministic idempotency key for email.
- Markdown-to-HTML conversion can introduce injection: escape source/model text
  before adding minimal formatting.
- Run-record corruption can break retries: use atomic writes and validate data.
- Concurrent invocations are not a supported v0.2 mode; document one scheduler
  per profile and fail safely if an active run lock exists.
- Never include environment values in logs, errors, artifacts, or tests.
- Deployment and provider configuration changes are explicitly authorized by
  this task; no cloud resources or paid services are created.

## 9. Executor Instructions

Implement thin commits: configuration/provider first, Daily Run/delivery second,
deployment/docs third. Preserve current local behavior. Add no runtime dependency
unless a correctness requirement cannot be met with Node built-ins. Do not add
Notion opportunistically.

