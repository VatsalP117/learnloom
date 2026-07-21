# Implementation Plan

## 1. Task Summary

Strengthen backend Dossier personalization so generated lessons remain
relevant to the learner, build on prior lessons, and fit the configured lesson
duration after the final editorial rewrite.

## 2. Current System Understanding

The generator builds a learner context and supplies it to curation, blueprint,
research, teaching, and practice. The context currently reduces each history
entry to a lesson summary and up to three recall questions, even though history
also stores the learning objective, concepts, and source titles. The final
editor receives the blueprint and drafts but not the learner context, so the
last rewrite can erase level, goal, and time personalization. The quality gate
validates structure, citations, and practice, but does not validate lesson
length against `LessonMinutes`.

## 3. Scope

### In Scope

- Include prior learning objectives, concepts, and source titles in the
  bounded learner context.
- Add an explicit, deterministic lesson word budget derived from configured
  lesson minutes.
- Give the final editor the learner context and exact word-budget contract.
- Validate the final lesson against the word budget and record useful metrics.
- Add focused tests for context completeness, budget calculation, repair
  feedback, and quality-gate behavior.
- Update generation architecture documentation.

### Out of Scope

- UI changes or new learner settings.
- Database/schema, authentication, billing, infrastructure, or public API
  changes.
- A new model provider or dependency.
- Subjective model-scored quality grading.

## 4. Proposed Technical Approach

Introduce an internal `lessonWordBudget` value with broad, bounded minimum and
maximum word counts based on `LessonMinutes`. Render that contract into the
learner context so every relevant stage sees it, and pass the same budget to
the deterministic quality evaluator. Add the learner context to the editor's
weighted input so its rewrite cannot legitimately ignore the profile.

Enrich each retained history entry with its objective, concepts, source
titles, summary, and recall questions while keeping existing history-count and
field-count bounds. The existing intermediate-character limiter remains the
hard prompt-size boundary.

## 5. Step-by-Step Execution Plan

1. Add word-budget calculation and human-readable context formatting.
2. Enrich bounded learning-history formatting.
3. Pass learner context and budget into the final editor.
4. Extend deterministic quality evaluation with time-fit validation and
   metrics.
5. Update unit fixtures and add focused regression tests.
6. Run focused and repository-wide Go checks, then review the diff against
   `main`.

## 6. Test Plan

- `go test ./internal/dossier`
- `go test -race ./cmd/... ./internal/...`
- `go vet ./...`
- `git diff --check`

## 7. Acceptance Criteria

- The final editor input includes learner level, goal, available time, exact
  lesson word range, and recent structured learning history.
- Lessons below or above the broad configured time-fit range are rejected with
  actionable repair feedback.
- Valid Dossiers expose lesson word count and budget bounds in quality metrics.
- Existing source-grounding, practice, rendering, and generation behavior
  remains intact.
- All relevant checks pass.

## 8. Risks and Guardrails

- The range must be broad enough to avoid brittle failures from normal model
  variance, especially for short and long configured lessons.
- Prompt growth must stay bounded by existing history retention and section
  fitting.
- No new external calls, schema fields, dependencies, or UI behavior.

## 9. Executor Instructions

Keep the implementation inside the Dossier module and its documentation. Use
the existing structured-output repair loop for actionable time-fit repair.
Avoid unrelated refactors.
