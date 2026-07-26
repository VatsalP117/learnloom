# Architecture

Learnloom is a hosted modular monolith with separately scaled web and worker
processes. The package boundaries follow product responsibilities rather than
transport or framework layers.

```mermaid
flowchart LR
  Browser --> Web["Go web role"]
  Clerk --> Web
  Web --> PG[(Postgres)]
  Web --> S3[(S3 artifacts)]
  Worker["Go worker role"] --> PG
  Worker --> Sources["RSS / Atom / articles"]
  Worker --> Search["Self-hosted SearXNG"]
  Worker --> Model["OpenAI-compatible API"]
  Worker --> S3
  Worker --> Resend
  Migrate["Go migrate role"] --> PG
```

## Modules

- `internal/httpapp`: host classification, authentication, request policy,
  control-plane JSON, Clerk lifecycle webhooks, and public reading surfaces.
- `internal/store`: Postgres persistence, scheduling, fair Issue Claims,
  delivery receipts, runtime controls, rate limits, and deletion work.
- `internal/execution`: orchestration only. It renews Claims and coordinates
  Dossier generation, artifact persistence, transactional completion, and
  delivery.
- `internal/failure`: the deep Issue Failure module. It classifies stable
  codes, categories, stages, retryability, safe learner messages, and incident
  identifiers without exposing internal detail through reading interfaces.
- `internal/source`: bounded acquisition with SSRF and redirect defenses.
  It owns catalog-first SearXNG discovery, native resolution, immutable
  snapshots, and Issue evidence freezing. Search snippets are candidate
  metadata only and never become Dossier evidence.
- `internal/dossier`: the multi-stage Dossier production pipeline, contract
  repair, learner-history continuity, time-fit enforcement, deterministic
  quality gate, and safe rendering.
- `internal/artifact`: checksummed, opaque-key S3 persistence.
- `internal/delivery`: Resend adapter and stable idempotency semantics.
- `internal/domain`: shared hosted product vocabulary and state machines.

Dependencies point inward: adapters implement narrow behavior consumed by the
Dossier and execution modules. HTTP handlers never call model, source, or
email providers directly.

## State and concurrency

Postgres owns mutable state. Workers claim due Issues with `SKIP LOCKED`,
per-Account fairness, expiring leases, renewal tokens, classified attempt
limits, and recovery of abandoned Claims. Issue Attempts and their Dossier
stage summaries are append-only operational evidence; the Issue row is the
current learner-safe projection. Infrastructure Claim loss has its own bounded
budget and does not consume a content-generation attempt. Artifact bytes are
persisted before an Issue is transactionally marked generated. Delivery is a
separate Claim so a retry never spends model tokens again.

Validated curation, Learning Blueprint, research, skeptical review, and teacher
outputs are stored as Dossier Checkpoints. A checkpoint is reusable only when
its fingerprint matches the pipeline version, model, Newsletter settings,
Learning History, and frozen Issue evidence. Invalid structured checkpoints
are rejected against the current contract, and failed editor candidates are
never checkpointed.

Workers drain during deployment: they stop claiming, become unready, continue
renewing active Claims, and wait for the current cycle. Work that exceeds the
drain deadline is explicitly released and requeued without consuming a
content-generation attempt.

An expired or interrupted Delivery Claim is conservatively recorded as
`outcome_unknown`, because the email provider may have accepted the idempotent
request before the worker lost its Claim. Unknown delivery outcomes are never
automatically replayed.

The important state transitions are:

```text
Issue:    queued -> generating -> generated
                        |            |
                        +-> failed   +-> delivery pending

Delivery: pending -> sending -> sent
                         |  \-> outcome_unknown
                         +----> failed -> pending (explicit retry)
```

## Hosted boundary

The apex domain serves marketing, the `app` host serves the authenticated
control plane, and `<username>` hosts serve public Personal Sites. Hostnames
are classified before routing; arbitrary Host headers do not reach a default
tenant. Account identity comes only from verified Clerk sessions and is
included in all owner-scoped database operations.

The architectural decisions and rejected local-first alternatives are recorded
under [`docs/adr`](adr).
