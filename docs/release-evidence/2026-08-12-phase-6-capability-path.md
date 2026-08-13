# Phase 6 capability path and continuity evidence — 2026-08-12

## Scope

Tasks `LL-605` through `LL-610` are implemented on schema version 26. This is
repository and isolated PostgreSQL evidence, not the external design-partner
retention gate.

## Capability path

- The stream view now leads with the learner's stated outcome and projects
  durable capability milestones from completed concepts and review outcomes.
- Each milestone says what the learner can now explain or retrieve and shows
  its evidence: completed lessons, retrieval attempts, and calculated recall
  confidence.
- Current gaps distinguish concepts not yet completed, concepts awaiting their
  first recall, and weak concepts needing another retrieval.
- The path continues to show likely next directions and a completed capability
  timeline linked to the underlying Dossiers.
- Stream cards and stream headers no longer present generated-lesson counts as
  progress. They summarize established and solidly recalled capabilities.

## Retrieval and weekly continuity

- In-lesson retrieval remains immediate and still drives the first-cycle
  activation event.
- Completing a lesson now activates its durable review items for the next day,
  rather than making the same prompts immediately due again.
- PostgreSQL lifecycle assertions prove the queue is empty immediately after
  completion and appears at the one-day spacing boundary.
- Weekly recaps say “Capabilities gained,” derive statements from completion
  and recall evidence, connect two recently learned concepts (or one concept to
  a next direction), include one due retrieval prompt, and choose one executable
  next action.

## Gentle re-entry

- Today preserves the one-step re-entry recommendation and identifies the
  relevant stream.
- The learner can slow that stream to a weekly synthesis, pause it, or clear
  older unopened backlog in one click.
- Backlog reset is non-destructive: it keeps the newest candidate in Today,
  moves older unopened lessons out of scheduling pressure, and preserves every
  lesson in Library.
- Dismissals are durable, account-owned, and ignored by Today, retention
  backlog counts, weekly-recap next actions, and cadence auto-throttling.
- Reset restores the learner's desired cadence without overwriting history and
  records a durable rhythm recovery decision.

## Honest visualization audit

- Removed fabricated percentage bars from the marketing dashboard mockup and
  replaced them with qualitative capability states.
- The remaining reading percentages in Today and the Dossier reader come only
  from persisted `lesson_progress`; they are reading position, not claimed
  mastery.
- Confidence labels are backed by the review scheduler's calculated learner
  concept state.

## Verification

- `go test ./... -count=1`: passed.
- `go vet ./...`: passed.
- `npm run check`: passed (contract check, lint, TypeScript, production build).
- Frontend: 21 files / 63 tests passed.
- PostgreSQL 17 clean-schema lifecycle passed on schema 26, including delayed
  review activation, capability projection, rhythm behavior, source novelty,
  non-destructive re-entry reset, Library preservation, and cross-account
  denial.
- Product baseline SQL passed on schema version 26.
- `git diff --check`: passed.

## Remaining Phase 6 exit gate

Seven-day and week-four retention, plus interview attribution to continuity,
mastery, or saved time, still require the external design-partner cohort. No
repository implementation substitutes for that evidence.
