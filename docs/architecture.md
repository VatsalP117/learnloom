# Architecture

```text
CLI / launchd / systemd / Docker
                 │
                 ▼
             Daily Run
                 │
     ┌───────────┼────────────┐
     ▼           ▼            ▼
 Source Items  Model       Learning History
     │        adapter           │
     └───────────┼──────────────┘
                 ▼
              Dossier
          (canonical JSON)
                 │
        persist Markdown + JSON
                 │
                 ▼
       delivery adapter(s)
          Resend email
                 │
                 ▼
         Delivery Receipt
```

The Dossier v2 content path is:

```text
Candidate Source Items
        │
        ▼
     Curator ──► selected Source Items ──► bounded article enrichment
                                                   │
                                                   ▼
Learning History ──► Learning Blueprint ──► research ──► skepticism
                                                   │
                                                   ▼
                                    lesson + retrieval practice
                                                   │
                              optional AI Exploration (separate)
                                                   │
                                                   ▼
                                    editor + deterministic gate
                                                   │
                                                   ▼
                                             Dossier v2
```

The multi-newsletter path adds scheduling and dashboard control without
changing the Daily Run's generation responsibilities:

```text
Dashboard ──► SQLite Workspace ◄──────── Worker
                   │                      │
                   │ claim Issue          ▼
                   └─────────────────► Daily Run ──► Dossier
                   │                      │
                   │ pending receipt      │ persisted artifacts
                   └──────────────────────┴──► Resend
```

## Deep modules and seams

- The **Daily Run** module owns ordering, reuse, failure policy, persistence,
  delivery retry, and locking behind one interface.
- The model seam has Command Code, OpenAI-compatible HTTP, and deterministic
  demo adapters.
- The delivery seam has a Resend adapter and injected deterministic adapters
  in tests. Additional destinations do not alter generation.
- The file run-store implementation owns Daily Run identity, locks, and
  Delivery Receipts. Its interface can later gain a SQLite adapter.
- The concrete **SQLite Workspace** module owns Newsletter validation,
  schedules, Issue and Delivery Receipt queueing/claiming, lifecycle
  transitions, and dashboard projections behind one interface. It does not
  replace immutable Dossier files or the Daily Run file store.
- The web and worker modules are thin adapters. HTTP queues an Issue; only the
  worker invokes Daily Run.
- Markdown and email are renderings of the canonical **Dossier**, not the
  source of truth.
- The source-enrichment module owns URL/address/redirect validation, bounded
  retrieval, conservative HTML text extraction, and per-item fallback.
- The content-quality module owns structured model contracts, citation and
  teaching-structure validation, the sourced/synthetic boundary, and the
  deterministic Dossier quality report.

## Trust seams

- Source XML and summaries are untrusted network input.
- Enriched article URLs and every redirect must resolve only to public
  addresses. Downloads are bounded by type, bytes, characters, redirects, and
  timeout; page scripts are never executed.
- Model endpoints require HTTPS. Loopback HTTP requires explicit insecure-local
  opt-in; remote plaintext HTTP is rejected before a credential can be sent.
- Source content is labeled as reference material and model adapters expose no
  tools.
- Command Code is spawned with `shell: false`.
- Direct model and Resend credentials come only from named environment
  variables and are never persisted.
- Email rendering escapes model/source text and allows only HTTP(S) links.
- Every container role runs non-root and excludes local secrets and state from
  its build context. Only the dashboard role listens, and Compose publishes it
  to host loopback.
- The dashboard adds CSRF protection and browser security headers, binds
  to host loopback under Compose, and must not be exposed publicly because
  authentication is not implemented.

## Failure behavior

- A single failed feed becomes a warning; all feeds failing aborts before model
  calls.
- A failed generation creates no delivery attempt.
- Each canonical Dossier is persisted to immutable, generation-versioned paths
  before its run-record pointer is atomically swapped and delivery begins.
- A failed destination records a Delivery Receipt and can retry independently.
- Newsletter generation and email have separate queues. Completing generation
  and enqueueing its email receipt is one SQLite transaction.
- Failed Newsletter email is manually retried from persisted artifacts and
  never regenerates the Issue.
- Ambiguous provider outcomes are recorded as non-retryable `unknown` receipts
  because the email may already have been accepted.
- Disabling Newsletter email atomically cancels receipts not yet in flight.
- Successful destinations are not repeated on a same-day rerun.
- An owner-token lock rejects overlapping execution and is never reclaimed
  automatically. After a crash, an operator must confirm the process is gone
  before removing the stale lock.
- Atomic writes use unique temporary names and restrictive file modes.

## Deliberate v0.5 limits

- Daily Run records are atomic JSON rather than SQLite.
- One trusted operator configures feeds; private-network URL blocking is absent.
- Full-text extraction is conservative and falls back for paywalled,
  client-rendered, thin, or unsupported pages.
- Learning History supports continuity and recall context but does not yet
  track demonstrated mastery or learner ratings.
- Resend is the only external delivery adapter.
- Learner feedback and Notion delivery are not yet implemented.
- A worker crash can leave an Issue in `generating`; automatic recovery is not
  implemented.
- A worker crash can leave a Delivery Receipt in `delivering`; automatic stale
  claim recovery and backoff are not implemented.
- Newsletter recipients are a trusted owner's addresses, not public subscriber
  lists; unsubscribe management is not implemented.
