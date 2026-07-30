# Learnloom product and technical roadmap

This is the execution roadmap for moving Learnloom from a reliable recurring
lesson generator into an adaptive learning system. Work is ordered by data
dependency, not by visual surface. A later phase must not simulate a capability
whose underlying state is still browser-only or unmeasured.

## Product outcome

The core loop is:

1. A learner states a concrete capability they want to build.
2. Learnloom establishes a credible information environment.
3. It selects a worthwhile next lesson and explains why.
4. The learner understands, retrieves, and applies the idea.
5. Learnloom records concepts, confidence, difficulty, and relevance.
6. Review and the next lesson adapt to that evidence.

The north-star metric is the number of weekly learners who complete a lesson
or review and return for another meaningful learning action within seven days.
Generated lessons, registered Accounts, and public pages are supporting
metrics.

## Sequencing principles

- Durable state precedes adaptive UI.
- Structured learning data precedes concept search, review, and curriculum
  visualization.
- Product telemetry precedes funnel optimization.
- Evidence quality wins over forced frequency.
- Measurement failure must never interrupt learning.
- Private lesson text, source content, email, and raw learning goals never
  leave the product through analytics.
- Public publishing remains explicitly opt-in at the site, stream, lesson, and
  search-indexing levels.

## Phase 0 — Safety and measurement foundation

Status: implemented in migrations 006 and 007.

- Derive readiness from a validated, contiguous migration ledger.
- Record content-free activation milestones for sign-up, stream creation,
  lesson generation/open/completion, review attempts, and search indexing.
- Persist private lesson-fit signals: difficulty, relevance, and recall
  confidence.
- Hydrate feedback across devices.
- Keep analytics writes non-blocking.

Acceptance:

- A newly migrated database is ready without a distant version constant.
- The activation funnel can be reconstructed with first-party SQL.
- Feedback survives a browser or device change.
- A disposable Postgres lifecycle test covers the new state.

## Phase 1 — Structured learning contract

Product:

- Explain why a lesson was selected now.
- Show how it connects to prior learning and what it unlocks next.
- Give every retrieval question a useful answer rubric and corrective
  explanation.
- Attach source references to claims in the reader when the artifact contains
  them.

Technical:

- Version a structured learning contract containing concepts, prerequisites,
  continuity, selection rationale, claims, citation references, retrieval
  prompts, answer rubrics, misconceptions, and suggested next concepts.
- Preserve the contract in canonical Dossier artifacts; do not reparse rendered
  HTML.
- Validate source identifiers against the frozen evidence set.
- Provide backward-compatible projections for existing artifacts.

Acceptance:

- Newly generated artifacts contain validated structured learning metadata.
- Existing artifacts still render.
- Retrieval rubrics and continuity are available through the private lesson API.
- Unsupported citation mappings are represented as unavailable, never invented.

## Phase 2 — Durable review and learner state

Product:

- Replace the browser-only mastered flag with a real due queue.
- Let learners assess recall as needs work, partial, or solid.
- Show when an item is next due without punitive overdue language.
- Keep review state consistent across devices.

Technical:

- Add durable review items and append-only review attempts.
- Start with a transparent three-stage interval algorithm.
- Store scheduler and prompt-contract versions.
- Add learner concept state derived from lesson completion, feedback, and review
  attempts.
- Make rescheduling idempotent and Account-owned.

Acceptance:

- Completing a lesson creates review items.
- An assessment deterministically changes the next due date.
- The same queue appears on two devices.
- Replaying a request cannot create duplicate attempts or items.

## Phase 3 — Adaptive curriculum and Today

Product:

- Give each stream a visible evolving curriculum: goal, concepts covered,
  prerequisites, current gaps, and likely next steps.
- Rank Today by in-progress work, review urgency, goal relevance, evidence
  strength, prerequisites, available time, and neglected streams.
- Explain every recommendation in learner language.
- Support daily, selected weekdays, weekly synthesis, and evidence-led rhythm.
- Allow “no worthwhile lesson today.”

Technical:

- Build compact learner-state and curriculum projections for the generator.
- Feed feedback and concept mastery into curation and blueprint stages.
- Store a selection score and reason for prepared lessons.
- Generalize scheduling without breaking local-time and DST guarantees.
- Add evidence-strength gates before scheduled generation spends model tokens.

Acceptance:

- Future lessons visibly respond to at least difficulty and recall evidence.
- Today selection is deterministic and tested.
- A weak-evidence cycle can defer work without being recorded as a failure.
- Existing daily streams retain their behavior through migration.

## Phase 4 — Activation, templates, and retention

Product:

- Offer focused starter templates with a concrete outcome, credible source
  environment, rhythm, and sample Dossier.
- Validate and preview source URLs during onboarding.
- Define activation as completing first retrieval, not creating a stream.
- Show honest preparation states and allow the learner to leave safely.
- Send a weekly recap of concepts learned, one useful connection, one review,
  and the best next action.
- After inactivity, recommend one re-entry action, collapse backlog pressure,
  and offer a gentler rhythm.

Technical:

- Version templates as data with explicit ownership and update policy.
- Add bounded source-validation endpoints using the existing safe acquisition
  boundary.
- Preserve attribution from landing page through activation without private
  content.
- Create idempotent recap jobs and deliveries.
- Add notification preferences and re-entry projections.

Acceptance:

- A new learner can reach a meaningful sample before supplying expert source
  configuration.
- Source failures are visible before stream creation.
- Activation and seven-day return are queryable.
- Recaps and re-entry never create duplicate email.

## Phase 5 — Trust, retrieval, and library depth

Status: implemented in migrations 012 through 014.

Product:

- Add inline citation interactions, copyable citations, and “question this
  claim.”
- Search concepts, sources, retrieval prompts, and titles across streams.
- Add notes, highlights, saved material, and export only after concept and
  citation identity are stable.
- Show capability-oriented progress and a concept timeline.
- Keep publishing a deliberate optional growth loop with preview, reporting,
  corrections, and moderation.

Technical:

- Add claim/citation indexes and full-text search projections.
- Store learner-created notes separately from immutable generated artifacts.
- Add correction/report workflows and least-privilege operator tools.
- Enforce index eligibility and auditable moderation reasons for public pages.

Acceptance:

- Search success can be measured without logging query text externally.
- Every displayed claim citation resolves to a real frozen source item.
- A public correction does not mutate private learning history.

## Phase 6 — Platform hardening and economics

Contracts and maintainability:

- Replace permissive cross-layer `any` DTOs with endpoint-specific generated
  types and runtime problem codes.
- Split oversized HTTP/store/generator files by state machine or capability,
  without introducing pass-through service layers.
- Remove transitional source JSON after an expand/contract migration.

Reliability and operations:

- Add MinIO/S3 integration coverage, query-plan fixtures, load budgets, and
  restore evidence.
- Move metrics behind an operational boundary.
- Add durable dashboards and alerts for queue age, failure classes, provider
  latency, pool saturation, delivery ambiguity, and spend.
- Define privacy retention, database erasure/anonymization, auditability, and
  least-privilege support workflows.
- Validate HSTS, backups, wildcard routing, provider IAM, staging isolation,
  and public-cache purge behavior.

Economics:

- Record model input/output tokens, latency, retries, and estimated cost by
  stage.
- Measure cost per generated, opened, completed, and retained lesson.
- Add global provider-budget enforcement before scaling beyond one worker.
- Define product plans only after real completion/retention and cost evidence
  exists.

Acceptance:

- API drift fails generation or CI rather than a user session.
- A dated restore drill and staging launch evidence exist.
- Operators can explain queue health and cost per meaningful learning action.
- Account deletion behavior matches the published privacy promise.

## Explicit deferrals

These are not part of the core adaptive-learning roadmap until the earlier
phases demonstrate retention:

- collaborative classrooms, teams, and mentor workflows;
- rich concept-map visualization;
- marketplace or community template submission;
- automated grading of unrestricted free text;
- native mobile applications;
- aggressive public-content growth or monetization experiments.

They may be revisited only with a concrete user need, an owner, a safety model,
and a measurable connection to learning outcomes.
