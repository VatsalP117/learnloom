# Phase 6 adaptive Today evidence — 12 August 2026

## Status

Tasks `LL-601` and `LL-602` are implemented on schema version 24. The Phase 6
retention exit gate remains open.

## Authoritative priority model

`RefreshTodayFocus` evaluates eligible generated lessons, due retrievals, and
the gentle re-entry state from durable Account-owned data. A lesson score has
separate, inspectable components for:

- saved in-progress reading state;
- age since the learner last touched the lesson;
- explicit relevance feedback as the goal-relevance signal;
- prerequisite completion or recall confidence;
- count of frozen evidence sources;
- fit with the learner's configured lesson time; and
- days since that path last produced a completed lesson.

An unfinished lesson in a paused stream remains resumable; pausing future
generation does not erase work already in progress. Untouched lessons from a
paused stream are not promoted. Due reviews remain eligible even when future
generation is paused.

Priority order is score-based rather than first-ready ordering. Saved progress
receives the strongest lesson signal. A long absence creates one re-entry
action rather than a backlog. Review urgency grows with due count and days
overdue. Ties are deterministic.

## Stored explanation

The selected kind, subject, stream, reason code, learner-facing reason, total
score, and component map are persisted in `today_focus_selections`. Repeated
reads of the same choice preserve `selected_at`; changing the selected subject
starts a new selection interval. The workspace returns this authoritative
projection even when the chosen lesson is older than the first Issue page.

Today displays the stored explanation and action verb. It retains the old
client heuristic only as a backward-compatible demo/failure fallback, not as
the production source of truth.

## Safety and verification

- Candidate, progress, feedback, concept, evidence, review, and persistence
  queries all join through the owning Account or Newsletter.
- The selection table cascades on Account deletion and validates bounded kinds,
  subject IDs, reasons, and JSON score components.
- Unit regressions cover saved-progress priority and the effects of evidence,
  prerequisites, relevance, time fit, and path neglect.
- PostgreSQL 16 lifecycle regression proves lesson selection, durable reason,
  stable selection timestamp, completion-to-review transition, migration 24,
  and the product baseline query.
- Frontend regressions prove the stored selection overrides first-ready order
  and can hydrate an older selected lesson outside the initial workspace page.
- Full Go suite and `go vet ./...`: passed.
- Frontend API contract, ESLint, typecheck, production build, and 20 files / 58
  tests: passed.
- `git diff --check`: passed.
- Product baseline SQL: passed on schema version 24.

## Evidence still required

- Tasks `LL-603`–`LL-610`.
- Seven-day and week-four retention gates for the design-partner cohort, with
  interview evidence attributing return behavior to continuity, mastery, or
  saved time rather than novelty.
