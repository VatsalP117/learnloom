# Phase 4 activation evidence — 11 August 2026

## Status

Activation work has started but the Phase 4 exit gate has not passed. The
configuration-first source step has been replaced with an autonomous default,
a useful evidence preview, a fast initial learning arc, durable resumability,
and step-level instrumentation. The post-confirmation waiting experience now
delivers useful orientation and live readiness rather than stream management.
Canonical pre-signup lesson proof is now implemented. Human editorial approval
of the starter-path catalog and a qualified-user activation cohort remain open.

## Journey implemented

The current onboarding sequence is:

```text
topic + desired outcome + familiarity
  -> autonomous discovery by default
  -> provisional role-aware evidence portfolio
  -> initial concepts + likely first lesson
  -> optional approval boundary
  -> cadence and delivery after the preview
  -> build the learning path
```

- The primary marketing and confirmation action is “Build my learning path.”
- A learner can begin with only an idea. Source discovery is the default when
  configured; “Use specific sources” reveals provided-only and gap-filling
  alternatives.
- The preview explains official/primary, research, practitioner, context, and
  counterweight roles and why each candidate was selected.
- Missing evidence roles are described as weak coverage. Candidates are
  explicitly provisional until resolution and freezing.
- The same bounded response builds an instant research-plan preview from the
  learner's topic, outcome, and familiarity. It names four initial concept
  directions, a likely first lesson, an objective calibrated by familiarity,
  and the honest 5–15 minute preparation range. It does not make a second model
  call or invent topic-specific factual claims before evidence is resolved.
- “Begin automatically” is the default. A learner can instead require source
  approval before any lesson-model generation and can reverse that preference
  later.
- Cadence and email appear after the evidence preview. Personal-site publishing
  is no longer requested during onboarding and remains private by default.
- The demo environment exercises autonomous discovery and the same preview
  journey instead of falling back to the legacy provided-source form.
- Confirmation routes to a dedicated first-lesson orientation while the first
  Issue remains automatically queued. It shows the likely learning arc, a
  two-minute prediction/evidence preflight, live intent/evidence/lesson stages,
  and the honest preparation range instead of exposing stream administration.
- The orientation polls only while generation is active, changes to an “open
  first lesson” action when ready, and routes review-mode learners to the source
  portfolio when approval is required.
- Leaving is explicitly safe and records an ownership-scoped preparation exit.
  Learners can opt into verified email delivery for the first and future
  lessons; otherwise the path remains available on Today when they return.
- Incomplete setup is stored as an owner-scoped, bounded JSON draft with a
  stable draft ID and monotonic revision. It resumes on another authenticated
  device at the saved step and makes explicit discard available.
- Autosaves are serialized and use compare-and-set revisions. Stale tabs cannot
  overwrite a newer revision. Stream creation writes a per-draft completion
  tombstone in the same transaction and consumes only the submitted draft at
  its exact revision, so an in-flight save cannot resurrect a completed draft,
  a stale tab cannot confirm newer setup state, and another Account's draft
  cannot be consumed. Explicit discard is also revision-aware and cannot erase
  a newer tab's changes.
- Saved template ID/version is restored only when it still matches a known
  starter template, preserving template attribution across devices.
- Empty page visits do not persist a draft. Inactive drafts expire after 30
  days; expiry records a content-free abandonment milestone before deleting
  the topic/outcome payload. Completion tombstones expire after the same replay
  window.
- Visitors can open `/examples` or `/examples/ai-evaluation` without an account
  and inspect a complete AI-evaluation Dossier. It contains a decision-oriented
  objective, claim-level links to five visible sources, a worked example,
  limitations, retrieval practice, and an application exercise.
- Catalog version 3 contains six outcome-oriented AI-engineering candidate
  paths—evaluation, RAG, agents, LLM security, context engineering, and
  inference operations—with four primary or official starting sources each.
  Every entry is explicitly review-pending; none is represented as
  human-reviewed before the founder review checklist is signed.

## Safety and lifecycle invariants

- Preview search is authenticated, rate-limited, bounded, read-only, and uses
  a 15-second request deadline.
- The preview returns only public-safe candidate metadata; it does not persist
  source bodies.
- Approval is owner-scoped. Cross-Account approval returns not found.
- Source mutations rebuild a pending portfolio and cannot rewrite evidence
  frozen into earlier lessons.
- A full stream update preserves an omitted review preference and invalidates a
  prior review approval before rebuilding changed sources.
- The worker releases the generation claim and decrements the model-attempt
  count while waiting, so approval time does not look like a model failure or
  consume the learner’s attempt allowance.
- Preview completion is recorded by the server only after a discovered
  portfolio succeeds or all provided sources validate; the client cannot set
  the milestone in its draft payload.
- Source-policy selection begins only after the intent step. Successful review
  assessment is the sole retrieval/activation authority; merely revealing
  context no longer counts as a review attempt.
- Onboarding progress events, confirmation, stream creation, retrieval,
  activation, signup, Clerk identity lifecycle work cancellation, and public
  search-indexing changes are written in the same database transaction as
  their owning state change. First-open and preparation-exit events remain
  ownership-scoped and idempotent.
- Lesson-generation and lesson-completion milestones are written inside their
  owning transactions. Signup and indexing events are atomic with their owning
  state changes rather than being silently discarded or committed afterward.

## Verification

- Full Go suite: passed.
- `go vet ./...`: passed.
- Frontend API contract, ESLint, typecheck, and production build: passed.
- Frontend tests: 21 files / 66 tests passed.
- `git diff --check`: passed.
- PostgreSQL 16 migration and complete store integration suite: passed on
  schema version 23.
- Baseline SQL: passed on schema version 23 and now reports intent, source,
  preview, confirmation, policy, preparation-exit, first-retrieval, and
  activation milestones by stable journey rather than conflating an Account's
  later restart.
- Desktop and 390 px mobile visual/DOM walkthrough: passed through topic entry,
  autonomous source policy, portfolio preview, approval choice, and final CTA.
- Browser resume walkthrough: passed after leaving the setup route and opening
  it again; step, topic, desired outcome, policy, and saved timestamp restored.
- Post-confirmation orientation: waiting and ready states passed in the demo;
  the waiting state also passed at desktop and 390 px mobile widths.
- Canonical Dossier: semantic DOM and visual walkthrough passed at desktop and
  390 px mobile widths; the public routes require no authentication.
- PostgreSQL concurrency regressions: stale save/create/discard revisions
  rejected, stale create fully rolled back, unrelated Account isolated, exact
  draft revision consumed on creation, and post-completion save rejected by
  the tombstone.

## Evidence still required

- Complete and sign the human editorial review of the six prepared starter
  paths (`LL-405`). Candidate source portfolios and the review matrix exist,
  but approval is intentionally not inferred from automated research.
- Prove the instrumented activation thresholds with at least 30 qualified new
  users (the Phase 4 exit gate).
