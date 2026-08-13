# Phase 0 baseline evidence — 11 August 2026

## Scope

This record separates facts proved by the repository, behavior observed on the
live Learnloom product, and production facts that still require authorized
operational evidence.

## Repository baseline

- Repository: `VatsalP117/learnloom`
- Local branch inspected: `main`
- Inspected commit before Phase 0 edits: `cdc1b43` (`Add Grafana Cloud monitoring`)
- Embedded migration ledger: 20 contiguous migrations
- Production Compose defaults autonomous source discovery to enabled and points
  the worker at the internal SearXNG service.
- Source policies implemented: discovered, provided, and hybrid.
- Product events implemented: signup, stream creation, lesson generation/open/
  completion, review attempt, and search-indexing enablement.
- Durable model economics implemented: input/output tokens, provider retries,
  stage latency, and estimated micro-USD cost.
- Durable reviews, learner concepts, lesson feedback, weekly recaps, search,
  notes, moderation, privacy erasure, budget admission, and Grafana monitoring
  exist in the inspected repository.

## Live behavior observed before Phase 0 edits

- Marketing led with “sources you trust” and personal publishing rather than
  topic-only autonomy.
- The authenticated creation flow exposed autonomous, provided-only, and hybrid
  source modes.
- The public “How Learnloom works” page incorrectly called autonomous discovery
  a planned capability.
- The inspected founder stream showed 27 terminal attempts with 16 usable
  lessons and 11 visible failures (40.7% visible failure rate). This is a
  single-Account observation, not a production-wide aggregate.
- The homepage example address `maya.learnloom.blog` returned a not-found page,
  and the public examples gallery contained no curated examples.

## Phase 0 repository changes begun

- Repositioned the homepage and server-rendered fallback around “Give us a
  topic. We’ll build the learning path.”
- Made autonomous discovery, learner-provided sources, and hybrid gap-filling
  truthful and explicit in product-transparency copy.
- Updated metadata, social alt text, app empty states, solution CTAs, and tests
  to match the shipped capability.
- Added `scripts/product-baseline.sql` for aggregate activation, latency,
  failure, retention, and unit-economics measurement.
- Added schema 38 append-only account evidence classifications and an
  account-ID-only operator script. Product, design-partner, billing, revenue,
  retention, failure, and cost reports now include only the latest explicit
  `real_user` classification in launch-gate aggregates. Founder, test, and
  unclassified coverage is reported separately; missing classification fails
  closed instead of using email heuristics.
- Corrected seven-day return to require a subsequent meaningful action strictly
  after activation and no later than day seven. The previous lower-bound-only
  query could incorrectly count a much later return as seven-day retention.
- Added a cohort-aligned retained-lesson unit-cost section: for mature activated
  real users, it divides generation cost incurred during days 0–7 by distinct
  lessons completed or reviewed during that same post-activation window. The
  report exposes mature accounts, retained accounts, retained lessons, and
  cohort cost; an empty
  retention denominator returns null rather than a manufactured value.
- Applied all 20 migrations to a clean PostgreSQL 16 database and executed
  the complete baseline report successfully against the empty schema. This
  validates report syntax and schema compatibility; it is not production
  evidence.
- Added durable evidence deferral semantics and a stage-specific source/model
  recovery matrix. See `2026-08-11-phase-2-reliability.md`.

## Verification

- `npm run check`: passed
- `npm test`: 16 files and 43 tests passed
- `go test ./cmd/... ./internal/...`: passed
- `go vet ./...`: passed
- `git diff --check`: passed

## Production evidence still required

- [ ] Exact deployed commit and image digest
- [ ] Applied production schema version
- [ ] Deployed model and pipeline versions (names only; no credentials)
- [ ] Trailing 30-day aggregate funnel
- [ ] Production-wide generation outcome and failure taxonomy
- [ ] Time to first generated/opened/completed lesson
- [ ] Seven-day return for a mature activated cohort
- [ ] Cost per generated, opened, completed, and reviewed lesson
- [x] Repository test/founder exclusion method (schema 38; production rows
      still require authorized operator classification before export)
- [ ] Current queue, alert, backup, restore, and release evidence references

Use the read-only aggregate report:

```sh
psql "$DATABASE_URL" -X -f scripts/product-baseline.sql
```

Production access is not inferred from repository access. Fill these items only
from an authorized operational session, and keep small-cohort or identifying
raw output outside the repository.
