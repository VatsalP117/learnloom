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

## Deep modules and seams

- The **Daily Run** module owns ordering, reuse, failure policy, persistence,
  delivery retry, and locking behind one interface.
- The model seam has Command Code, OpenAI-compatible HTTP, and deterministic
  demo adapters.
- The delivery seam has a Resend adapter and injected deterministic adapters
  in tests. Additional destinations do not alter generation.
- The file run-store implementation owns Daily Run identity, locks, and
  Delivery Receipts. Its interface can later gain a SQLite adapter.
- Markdown and email are renderings of the canonical **Dossier**, not the
  source of truth.

## Trust seams

- Source XML and summaries are untrusted network input.
- Model endpoints require HTTPS. Loopback HTTP requires explicit insecure-local
  opt-in; remote plaintext HTTP is rejected before a credential can be sent.
- Source content is labeled as reference material and model adapters expose no
  tools.
- Command Code is spawned with `shell: false`.
- Direct model and Resend credentials come only from named environment
  variables and are never persisted.
- Email rendering escapes model/source text and allows only HTTP(S) links.
- The container runs non-root, opens no port, and excludes local secrets and
  state from its build context.

## Failure behavior

- A single failed feed becomes a warning; all feeds failing aborts before model
  calls.
- A failed generation creates no delivery attempt.
- Each canonical Dossier is persisted to immutable, generation-versioned paths
  before its run-record pointer is atomically swapped and delivery begins.
- A failed destination records a Delivery Receipt and can retry independently.
- Successful destinations are not repeated on a same-day rerun.
- An owner-token lease with heartbeat rejects overlapping execution and lets a
  later process safely reclaim a lock left by a crashed process.
- Atomic writes use unique temporary names and restrictive file modes.

## Deliberate v0.2 limits

- Daily Run records are atomic JSON rather than SQLite.
- One trusted operator configures feeds; private-network URL blocking is absent.
- Source material uses feed summaries rather than extracted article bodies.
- Resend is the only external delivery adapter.
- Learner feedback and Notion delivery are not yet implemented.
