# Phase 6 learning rhythm evidence — 2026-08-12

## Scope

Tasks `LL-603` and `LL-604` are implemented on schema version 25. This evidence
covers repository behavior and an isolated PostgreSQL 17 validation database;
it is not production deployment or retention evidence.

## Product behavior

- A learner can choose evidence-led, daily, selected-weekday, or weekly
  synthesis rhythm from the stream page.
- Desired rhythm and effective rhythm are separate durable fields. Automatic
  slowdown therefore never overwrites the learner's preference.
- Selected weekdays use ISO weekday values and are normalized, deduplicated,
  ownership-scoped, and used when calculating the next local occurrence.
- Weekly synthesis dispatches carry `requested_lesson_type = 'synthesis'`.
  That request participates in the generation fingerprint, appears in the
  blueprint context, and overrides an inconsistent model-selected lesson type.
- Evidence-led dispatch refreshes and freezes the candidate source portfolio,
  compares its snapshot IDs with the most recent generated lesson, and defers
  before model generation when the set is unchanged. First lessons remain
  eligible because there is no prior evidence set.
- A deferral records the stable `no_new_evidence` / `evidence_rhythm` failure
  classification and a durable rhythm decision. The learner sees the existing
  honest “nothing worthwhile yet” state rather than a failed lesson.
- Unopened backlog is calculated from generated lessons without an authoritative
  `lesson_opened` event. At the configured threshold (default three), due work
  is skipped, effective rhythm becomes weekly, and the reason is stored and
  shown. Once the backlog is opened, the next due decision restores the desired
  rhythm automatically.
- Manual lesson generation remains available and is not silently rate-shaped by
  the scheduled backlog policy.

## Data and safety

- Migration `025_learning_rhythm.sql` adds desired/effective rhythm, weekday,
  automatic-throttle, threshold, reason, and throttle timestamp fields.
- `rhythm_decisions` preserves configure, dispatch, defer, throttle, and recover
  decisions with the backlog count and learner-facing reason.
- The rhythm mutation endpoint is account-owned and returns `not found` for a
  cross-account stream ID.
- Source novelty comparison uses frozen snapshot identity, so a repeated HTTP
  200 response with identical content does not masquerade as new evidence.
- Existing general stream edits preserve the current effective rhythm and
  recompute the next run using the new time/timezone.

## Verification

- `go test ./... -count=1`: passed.
- Focused worker/store/source/Dossier suites: passed, including evidence-led
  pre-model deferral and weekly requested-type propagation.
- `npm run check`: passed (API contract check, ESLint, TypeScript, production
  build).
- `NewsletterDetail.test.ts`: 10 tests passed, including weekday selection and
  visible automatic slowdown.
- PostgreSQL 17 clean-schema lifecycle: passed on schema version 25.
- PostgreSQL rhythm integration: passed selected weekdays, weekly synthesis
  intent, automatic throttle, skipped backlog dispatch, recovery, and
  cross-account denial.
- PostgreSQL source comparison: passed unchanged and changed frozen-evidence
  cases.
- `scripts/product-baseline.sql`: passed on schema version 25.

## Remaining Phase 6 gate

The Phase 6 retention exit gate still requires the paid design-partner cohort
and interviews. Repository completion of `LL-603` and `LL-604` does not claim
that external evidence.
