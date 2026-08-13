# Phase 5 learning-contract evidence — 12 August 2026

## Slice completed

Tasks `LL-501`–`LL-505` are implemented in the version 4 Dossier artifact and
version 2 learning contract.

- Every new Dossier stores one bounded lesson purpose: `foundation`, `update`,
  `deep_dive`, `synthesis`, `application`, or `review`.
- The Blueprint prompt must select a valid purpose. A first lesson is forced to
  `foundation`; old checkpoints receive a deterministic compatible fallback.
- The learning contract persists selection rationale, learning objective,
  continuity bridge, prerequisite/core concepts, suggested next concepts, and
  whether each core concept is `introduced` or `reinforced` against history.
- Source-linked factual blocks become structured claims. Source-linked audit
  constraints become a separate structured limitations list.
- The evidence boundary is qualitative (`source_bounded`), not a fabricated
  confidence percentage.
- Raw skeptical/editorial critique remains available inside the private
  artifact for quality operations, but new reader, Markdown, and email HTML
  projections render only the concise `Limits and verification` projection.
- The prior public Markdown `Quality gate: N/100` line is removed. Internal
  deterministic quality metrics remain available to the generation system.
- The lesson reader labels the lesson purpose and source-bounded evidence state.

Tasks `LL-506`, `LL-507`, `LL-509`, `LL-513`, and `LL-514` are also implemented:

- New paths default to a 12-minute lesson; the focused choices are 8, 12, and
  15 minutes, with 25 minutes available only as an explicit extended deep dive.
- Starter paths use 12- or 15-minute defaults.
- The quality gate measures the words in the final rendered lesson rather than
  trusting the requested prompt budget. It stores an estimated focused reading
  time and requires a materially tighter 70–120 words per requested minute.
- The Blueprint and fixed lesson contract require a central mechanism, worked
  example, important source-linked limitation, at least three answerable
  retrieval prompts, and a transfer application.
- Learner level, goal, available time, recent difficulty, relevance, recall
  confidence, and concept state are loaded into the curation/Blueprint and
  teaching/editor contexts.
- The reader opens with time, why this lesson was selected, its connection to
  prior learning, and its objective. Claim citations jump to the corresponding
  source detail and copyable citation.

Tasks `LL-515`–`LL-519` are implemented on schema version 23:

- Reading percentage is saved to the Account and restored from the server when
  the lesson opens on another device. Local state remains a resilience cache,
  not the source of truth.
- Retrieval drafts autosave after a short debounce and on blur. Drafts are
  owner-scoped, bounded to 2,000 characters, and keyed to a real generated
  prompt for the lesson.
- The answer rubric remains hidden until the learner saves an answer or chooses
  “Skip for now.” A reveal is immutable; a stale or different second reveal is
  rejected rather than rewriting the original attempt.
- A persisted non-skipped answer records first retrieval and activation inside
  the same transaction. Skipping reveals the reflection but records neither
  milestone and carries no penalty.
- Difficulty, relevance, and recall-confidence feedback already feed learner
  state. “Question this claim” notes now contribute up to three recent open
  questions to the next lesson's learner context.
- Completion shows the capability practiced, next suggested concept, and next
  review timing. It retains return-to-path context and adds owner-scoped
  previous/next lesson navigation.

Task `LL-510` is implemented as a collapsed evidence appendix. Claim mappings,
structured limitations, enriched source summaries, authorship/provenance, and
source links are available on demand without forcing research-process detail
into the core reading path.

Task `LL-508` now compares the model-generated learning objective and concept
representation with the five most recent lessons and requires substantial
overlap in the frozen canonical source URLs before declaring a duplicate. Old
history entries without URLs retain the conservative title-overlap fallback.
Changing only wording no longer bypasses the gate; materially changing the
concept direction or evidence portfolio does.

Task `LL-511` adds the versioned 12-case `lesson-eval-v1` corpus. It contains a
passing and failing case for usefulness, continuity, difficulty fit,
unsupported claims, semantic redundancy, and final rendered time fit. The
structural gate always runs; an opt-in configured-provider gate requires at
least 80% agreement without spending model budget during ordinary tests.

## Invariants

- A learning contract fails generation if it lacks concepts, source-mapped
  claims, source-mapped limitations, or three answerable retrieval prompts.
- Claim and limitation mappings reject unknown source identifiers.
- A model cannot invent an arbitrary lesson type; the enum is validated.
- The first lesson cannot be mislabeled as a later-stage update, synthesis, or
  review even if a model proposes it.
- Public projections never expose raw internal audit text and never expose the
  internal numerical quality score as learner confidence.
- Final rendered lesson length, not the model's promise, must pass the focused
  reading-time budget before the artifact can complete.
- Retrieval drafts can change only before reveal. Revealed answers and skips
  are immutable, and only a non-skipped persisted answer can activate.
- Retrieval and navigation reads join through the owning Newsletter; knowing an
  Issue or prompt identifier does not grant cross-Account access.

## Verification

- Full Go suite: passed.
- `go vet ./...`: passed.
- Frontend API contract, ESLint, typecheck, and production build: passed.
- Frontend tests: 20 files / 56 tests passed.
- `git diff --check`: passed.
- Render regression proves the learner-facing limitation is present while raw
  audit text and numerical certainty proxy are absent.
- PostgreSQL 16 migration 23, full store lifecycle, retrieval draft/reveal,
  immutable replay, activation, progress resume, feedback projection, and
  navigation checks: passed.
- Baseline SQL: passed on schema version 23.

## Evidence still required

- `LL-512`: named human review for every canonical starter-path lesson and the
  planned beta-output sample. Product or model labels do not satisfy this gate.
- Phase 5 exit gate: 8 of 10 design partners must rate sampled lessons as worth
  their time, unsupported-claim/citation gates must pass on their output, and
  measured first-lesson completion must exceed the activation threshold.
- A fresh rendered desktop/mobile walkthrough of the new answer-draft and
  completion panels remains pending. The local browser surface rejected the
  loopback preview during this verification run; unit, integration, type, and
  production-build gates passed, but they are not being misrepresented as a
  visual check.
