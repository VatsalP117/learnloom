# Implementation Plan

## 1. Task Summary

Improve the substance of Learnloom's Dossiers rather than its deployment or
surface UX. The new pipeline should reason over richer source material, teach
toward a precise objective, deliberately connect to prior learning, provide
better retrieval/application work, and use an additional model pass to edit
against a measurable quality rubric. Newsletter owners may opt into a separate
AI Exploration section for explicitly synthetic analogies, deductions, and
experiments that never blend into the source-grounded lesson.

## 2. Current System Understanding

The current Daily Run passes up to eighteen RSS/Atom titles and summaries
through four free-form model stages: researcher, skeptic, teacher, and examiner.
Learning History is supplied mainly to avoid repetition. There is no source
curation before research, no article enrichment, no explicit learning
blueprint, no deterministic quality report, and no editorial rewrite. The
Dossier is version 1 and stores free-form lesson, critique, practice, and
sources. Rendering and delivery already derive from that canonical JSON.

Newsletters are stored in SQLite schema version 2. Generation settings are
adapted into the existing config inside the worker. The dashboard can create
Newsletters and edit email settings, but it has no content-generation settings.

## 3. Scope

### In Scope

- A curator model stage selecting a coherent set of source identifiers
- Full-text enrichment for selected HTTP(S) Source Items
- Bounded downloads, content-type/size checks, redirect validation, and
  private-address blocking for enrichment
- Graceful summary fallback when an article cannot be enriched
- A structured learning blueprint with objective, prerequisites, central
  mechanism, worked example, misconception, experiment, and continuity bridge
- Stronger teacher and examiner contracts
- An optional, clearly labelled AI Exploration stage
- A final structured editorial rewrite and boundary check
- Dossier version 2 containing blueprint, provenance, optional exploration, and
  deterministic quality evaluation
- Citation/source-identifier and required-structure validation
- SQLite schema version 3 with per-Newsletter AI Exploration enablement
- Dashboard create/detail controls for the opt-in
- Backward-compatible rendering of version 1 Dossiers
- Updated demo provider, configuration, documentation, and focused tests

### Out of Scope

- Deployment changes
- Notion or new delivery channels
- Authentication or public multi-user subscriber features
- Learner rating buttons or automatic curriculum mastery tracking
- Browser automation, paywalled-source bypass, JavaScript page rendering, or
  arbitrary document uploads
- A general-purpose web crawler
- Treating AI Exploration as verified fact or including it in core recall

## 4. Proposed Technical Approach

Introduce a focused source-enrichment module. After the curator returns strict
JSON containing three to five valid source identifiers, enrich only those
items. Every URL and redirect is restricted to HTTP(S), DNS results resolving
to private/reserved addresses are rejected, response bytes and elapsed time are
bounded, and non-HTML/text responses fall back to the feed summary. HTML is
reduced to article/main/body text using a conservative dependency-free
extractor. Individual enrichment failure is recorded in provenance and does
not abort the Dossier.

Evolve `buildDossier` into:

1. curator
2. source enrichment
3. structured learning blueprint
4. source-grounded research
5. skeptical audit
6. lesson
7. retrieval/application practice
8. optional AI Exploration
9. editorial rewrite
10. deterministic validation/evaluation

Structured stages return JSON parsed through one strict helper with code-fence
tolerance and field validation. The editorial stage returns final lesson,
critique, practice, and optional exploration separately. Its prompt explicitly
forbids moving synthetic content into the core. Programmatic validation rejects
unknown citation identifiers, missing core sections, insufficient retrieval
questions, missing answer keys, or exploration appearing when disabled.

Add `evaluateDossier` as a deterministic rubric rather than using another
subjective model call. Persist its checks and score in Dossier v2 so quality can
later be compared across prompt/model changes.

Migrate SQLite under the existing serialized migration path. Existing
Newsletters default AI Exploration off. Add a dedicated content-settings
operation and CSRF-protected dashboard route. The worker maps the Newsletter
setting into runtime config. Direct Daily Runs can opt in through
`content.aiExplorationEnabled`; default remains false.

## 5. Step-by-Step Execution Plan

1. Record task and plan; establish Dossier v2 and schema v3 contracts.
2. Add source curation/enrichment primitives and security/fallback tests.
3. Add structured-stage parsing, blueprint, expanded prompts, editor, and
   deterministic evaluation.
4. Update Demo Provider and pipeline fixtures for the complete stage sequence.
5. Add schema v3, content settings, worker mapping, and migration tests.
6. Add dashboard opt-in controls with CSRF/escaping tests.
7. Update Markdown/email rendering with a visually distinct AI Exploration
   section and backward compatibility.
8. Update examples, architecture, and operating documentation.
9. Run focused and complete checks, package an independent review, fix
   blockers, push, and open a draft PR.

## 6. Test Plan

- Curator accepts only existing source identifiers and enforces selection size.
- Enrichment rejects non-web/private targets and validates redirects.
- Downloads stop at the byte limit and unsupported content falls back safely.
- HTML extraction removes scripts/styles and returns useful article text.
- One enrichment failure does not abort other selected sources.
- Structured output parser handles plain/fenced JSON and rejects malformed
  contracts.
- Blueprint fields are present and bounded.
- Pipeline stage order is correct with exploration disabled and enabled.
- Editor receives the grounded/synthetic boundary contract.
- Unknown citation identifiers fail validation.
- Required lesson sections, retrieval questions, application, and answer key
  are validated.
- Quality evaluation is persisted in Dossier v2.
- AI Exploration is absent by default and clearly separate when enabled.
- Version 1 Dossiers continue rendering.
- Schema v2 migrates to v3 without data loss; concurrent initialization remains
  safe.
- Newsletter content setting maps into worker runtime config.
- Dashboard mutations require CSRF and display the current opt-in state.
- Existing generation, delivery, scheduling, and migration tests remain green.

## 7. Acceptance Criteria

- A generated Dossier uses a curated subset of richer Source Items.
- The core lesson has a precise objective, mechanism, example, misconception,
  practical experiment, citations, and retrieval/application practice.
- The Dossier intentionally bridges to Learning History when available.
- The final persisted lesson is the editor's validated rewrite.
- AI Exploration is disabled by default and only appears for opted-in
  Newsletters/direct configurations.
- Synthetic content is labelled, uncited, excluded from core recall, and stored
  separately in canonical JSON.
- A deterministic quality report accompanies every Dossier v2.
- Existing workspaces and old Dossier previews continue to work.
- Full automated/static checks and independent review pass.

## 8. Risks and Guardrails

- Article fetching expands SSRF exposure. Validate protocol, hostname, resolved
  addresses, and every redirect; retain the documented trusted-feed boundary.
- HTML extraction will not solve paywalls or client-rendered pages. Fall back to
  feed summaries rather than failing a run.
- More stages increase latency and token usage. The user explicitly prefers
  spending available model tokens on content quality; keep source and
  intermediate bounds enforced.
- JSON model output can be malformed. Fail with a stage-specific contract error
  rather than persisting a partially validated Dossier.
- Editorial rewriting can weaken citations. Validate references after editing.
- AI-generated material can leak into the core. Keep separate prompt fields,
  separate canonical storage, clear rendering, and deterministic boundary
  checks.
- Do not include AI Exploration in core retrieval questions or represent it as
  sourced evidence.

## 9. Executor Instructions

Use Node built-ins only unless a dependency becomes unavoidable and is approved
through a change request. Preserve existing Daily Run idempotency and delivery
behavior. Keep commits thin: enrichment, pipeline, settings, rendering/docs.
Do not weaken URL, prompt-injection, HTML-escaping, or migration protections.

