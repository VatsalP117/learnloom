# Architecture

```text
RSS / Atom feeds
      │
      ▼
parse, normalize, deduplicate, bound input
      │
      ├────────────── recent local lesson history
      │                         │
      ▼                         ▼
 researcher ──► skeptic ──► teacher ──► examiner
      │             │            │           │
      └─────────────┴────────────┴───────────┘
                            │
                            ▼
                 Markdown + history JSON
```

## Trust boundaries

- Source XML and summaries are untrusted network input.
- Configuration URLs are restricted to HTTP and HTTPS.
- Source content is explicitly labeled as reference material in provider
  prompts and cannot request tool use.
- Command Code is spawned with `shell: false`, so prompt content is never
  interpreted by a shell.
- Provider sessions run with plan permissions and are instructed not to browse,
  edit files, or use tools.
- Command Code credentials are managed solely by its official CLI.
- The scheduler stores paths and timing, never credentials.

## Failure behavior

- A single failed feed becomes a warning.
- All feeds failing aborts before model calls.
- Any failed or empty model stage aborts the dossier.
- Markdown and history writes use temporary files followed by atomic renames.
- Re-running on the same date replaces that date's dossier but preserves each
  completed history entry.

## Deliberate MVP limits

- The model receives feed-provided summaries, not full article bodies.
- There is no email or chat delivery.
- Learner answers are not yet captured, so personalization uses lesson history
  rather than demonstrated mastery.
- RSS/Atom parsing supports common formats without trying to implement the full
  XML specification.

