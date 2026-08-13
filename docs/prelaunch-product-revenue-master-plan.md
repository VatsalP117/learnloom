# Learnloom pre-launch product and revenue master plan

Status: proposed execution plan
Date: 11 August 2026
Purpose: turn Learnloom from a capable learning system into a focused, dependable, monetizable product before broad public launch.

## 1. The decision

Learnloom will not launch as a generic AI learning assistant, a read-later tool,
or a publishing platform. It will launch as:

> **An autonomous learning system for professionals in fast-moving fields. Give
> Learnloom a topic and an outcome; it establishes a credible information
> environment, builds a progressive learning path, teaches the next useful
> concept, and revisits it before it fades.**

The initial wedge is English-speaking AI/software professionals who need to
stay current without living in feeds, repeatedly prompting a chatbot, or
manually maintaining a curriculum.

This wedge is intentionally narrow. The architecture remains general enough
for other fields, but the launch message, starter paths, examples, interviews,
and distribution will target one buyer with one expensive problem.

The personal learning site remains strategically valuable, but it becomes an
optional proof, identity, and acquisition layer after private learning value is
established.

## 2. What is and is not guaranteed

No product plan can guarantee revenue. The plan can remove avoidable causes of
failure and require evidence before further investment. Revenue probability is
maximized by proving, in order:

1. Learnloom can reliably produce trustworthy source portfolios and lessons.
2. The first session creates value quickly enough for a new user to continue.
3. Learners return because capability is accumulating, not because content is
   being generated.
4. A defined customer chooses to pay at a sustainable gross margin.
5. At least one repeatable acquisition channel converts profitably.

Every phase below has an exit gate. Missing a gate causes diagnosis and
iteration; it does not permit moving forward based on optimism.

## 3. The product promise

### Primary promise

> **Give us a topic. We will build the learning path.**

Learnloom will:

- discover candidate sources;
- classify their role in the information environment;
- rank relevance, authority signals, freshness, usability, and diversity;
- show the learner which sources were selected and why;
- build an initial curriculum from the learner's outcome and current level;
- prepare a short lesson that visibly builds on prior learning;
- attach claims to inspectable source evidence;
- ask for retrieval or application before completion;
- schedule later review based on learner feedback;
- adapt future lesson choice, depth, and rhythm.

### Source-control promise

Users who already know their sources can:

- use only those sources;
- make them the preferred core and let Learnloom fill gaps;
- add, remove, block, or prioritize sources later;
- inspect why an automatically discovered source was included;
- report or replace weak evidence.

### Trust boundary

Learnloom must not claim to certify truth. It discovers, ranks, safely resolves,
and evaluates sources using explainable signals. Important conclusions remain
linked to evidence, limitations remain visible, and learners retain source
control.

## 4. The core loop

```text
intent
  -> credible source environment
  -> visible learning path
  -> next useful lesson
  -> retrieval/application
  -> learner-state update
  -> adapted next lesson or review
```

The product's unit of value is a **completed learning cycle**:

```text
lesson understood + retrieval attempted + learner state updated
```

Generated lessons are inventory and cost. They are not the north-star metric.

## 5. Experience principles

1. **Autonomy first, control available.** A topic is enough to begin; expert
   configuration is available after value is visible.
2. **Capability over content.** The UI shows what the learner can now explain,
   evaluate, or do—not how many documents were generated.
3. **Evidence before fluency.** Clear prose cannot compensate for weak or
   untraceable evidence.
4. **Continuity must be visible.** Every lesson explains why it was selected,
   what it builds on, and what it unlocks.
5. **Quality beats cadence.** “No worthwhile lesson today” is better than a
   forced, repetitive, or poorly grounded lesson.
6. **Private by default.** Publishing is an intentional action after learning,
   never an implicit consequence of generation.
7. **Short by default, depth on demand.** The core session should usually fit
   into 8–15 minutes; evidence and deeper exploration remain accessible.
8. **No operational leakage.** Learners see useful preparation and recovery
   states, not internal pipeline terminology or raw editorial audits.
9. **A calm product can still be decisive.** Calmness must not hide the primary
   action, value, state, or consequence.

## 6. Changes we are willing to make

The current implementation is not a constraint when it conflicts with the
target product. The program explicitly authorizes:

- replacing the homepage message and information hierarchy;
- changing onboarding from configuration-first to preview-first;
- adding a real source-authority and coverage model;
- replacing fixed daily generation with evidence-led scheduling;
- changing the Dossier contract, length, and rendering hierarchy;
- moving curriculum selection and Today ranking to durable server state;
- replacing `published/hidden` with `private/draft/published` semantics;
- changing database schemas and API contracts through migrations;
- adding billing, entitlements, trial, and lifecycle infrastructure;
- demoting or removing features that distract from activation or retention;
- delaying public launch until paid usage and reliability gates are met.

## 7. Success metrics and launch gates

### North-star metric

**Weekly completed learning cycles per activated learner.**

### Reliability gates

- At least 98% of scheduled learning decisions end in either a usable lesson or
  an explicit evidence-led deferral—not an opaque failure.
- At least 95% of requested first lessons become available inside the published
  preparation-time SLO.
- No learner encounters two consecutive retryable failures without automatic
  recovery or a specific corrective path.
- Source and generation failures are classified and measurable by stage.
- Citation identifiers in displayed lessons resolve to frozen source evidence.

### Source-intelligence gates

- A human-labeled evaluation set covers at least 50 representative launch-ICP
  topics.
- At least 80% precision among the top five selected sources on the evaluation
  set, measured against documented relevance and authority criteria.
- Every selected source has an explainable role: primary/official, research,
  practitioner explanation, or counterweight.
- No single registrable domain supplies more than the configured share of a
  source portfolio unless the learner explicitly chooses a bounded source.
- Weak coverage defers generation or asks for help rather than inventing
  confidence.

### Activation gates

- At least 60% of qualified sign-ups reach a source/path preview.
- At least 50% open the first lesson.
- At least 40% complete the first lesson's retrieval/application step.
- Median time from sign-up to useful preview is under two minutes for starter
  paths and under five minutes for arbitrary topics.

### Retention gates

- At least 30% of activated design partners complete another meaningful
  learning action within seven days.
- At least 20% complete a meaningful action in week four.
- Fewer than 30% of generated lessons remain unopened after seven days; if this
  threshold is missed, generation cadence must reduce automatically.

### Commercial gates

- At least 10 design partners voluntarily pay before broad launch.
- At least 30 paying customers and one complete month of payment/entitlement
  lifecycle evidence before paid acquisition.
- AI, search, email, storage, and payment COGS remain below 25% of recognized
  subscription revenue at the tested usage allowance.
- Early monthly paid churn trends below 8%; the target after product-market fit
  is below 5%.
- No paid channel scales until measured contribution margin and payback are
  acceptable.

## 8. Workstreams

The program has eight connected workstreams:

1. Product strategy and research
2. Reliability and release safety
3. Source intelligence
4. Curriculum, lesson, and mastery
5. Activation and product UX
6. Publishing and product-led growth
7. Billing and unit economics
8. Distribution and launch

Work is executed by phase, not by completing an entire workstream in isolation.

## 9. Phase-by-phase execution plan

Calendar estimates assume one founder using agent-assisted implementation.
They are planning ranges, not promises. Evidence gates, not dates, determine
progression.

### Phase 0 — establish the launch baseline

Target: 2–3 days.

#### Tasks

- [ ] `LL-001` Record the exact production commit, migration version, model,
  discovery configuration, infrastructure topology, and deployed feature set.
- [ ] `LL-002` Deploy or stage the current `main` revision so repository and
  live behavior no longer contradict one another.
- [ ] `LL-003` Export a baseline funnel from existing product events: signup,
  stream creation, lesson generation/open/completion, review, seven-day return.
- [ ] `LL-004` Export failure rate and failure classes for the last 30 days.
- [ ] `LL-005` Export model tokens, latency, retries, and cost per generated,
  opened, completed, and retained lesson.
  The schema-38 report now includes cohort-aligned seven-day retained-lesson
  cost and a strict retention window; this remains open until an authorized
  real-user production export is retained.
- [ ] `LL-006` Separate founder/test traffic from real-user evidence.
  Append-only operator classification and fail-closed product, reliability,
  beta, revenue, and cost reports are implemented. This remains open until
  existing production accounts are reviewed/classified and exported.
- [x] `LL-007` Create a release evidence folder for browser, source-safety,
  restore, load, provider-evaluation, and payment-lifecycle proof.
- [x] `LL-008` Fix the marketing/product contradiction that calls autonomous
  discovery “planned” when production exposes it.

#### Exit gate

We can explain current production reliability, activation, retention, and cost
from data rather than screenshots or intuition.

### Phase 1 — choose the buyer through problem evidence

Target: 1 week, overlapping Phase 0.

#### Tasks

- [ ] `LL-101` Recruit 15 interviewees who work in AI engineering, AI product,
  ML infrastructure, developer tooling, or adjacent fast-moving technical work.
  The versioned protocol, privacy-safe frozen corpus, and fail-closed evaluator
  are ready; recruitment remains external and unperformed.
- [ ] `LL-102` Conduct problem interviews about the last subject they tried to
  follow, current source workflow, time lost, learning failure, and purchasing
  authority. Do not pitch features until the current workflow is understood.
- [ ] `LL-103` Score interviewees on pain frequency, economic consequence,
  existing spend, urgency, and reachable distribution.
  Stable codes and the roadmap's exact aggregate thresholds are implemented;
  real interview records remain required.
- [ ] `LL-104` Select one launch persona and one primary job-to-be-done.
- [ ] `LL-105` Write the rejected personas and reasons so marketing does not
  drift back toward “everyone who likes learning.”
  A decision/rejection template is ready and must be completed only after the
  frozen aggregate is evaluated.
- [ ] `LL-106` Recruit 10 design partners from the selected persona and obtain
  permission for weekly observation and feedback.

#### Default hypothesis to test

**AI/software professionals who need to stay current on a defined technical
area and turn updates into usable professional judgment.**

#### Exit gate

At least 10 of 15 interviewees experience the problem weekly, at least five
already spend money or substantial time on a workaround, and 10 agree to a
structured design-partner program. Otherwise the wedge is revised before major
UX work.

### Phase 2 — make generation dependable

Target: 1–2 weeks.

#### Tasks

- [ ] `LL-201` Reproduce every active production failure class with a safe
  fixture or synthetic provider behavior.
  Every currently emitted stable repository code has a named behavioral
  fixture and the baseline flags unregistered/legacy/unknown production codes.
  This remains open until the 30-day production taxonomy proves which classes
  are active and identifies no additional production-only class.
- [x] `LL-202` Distinguish a legitimate evidence deferral from system failure.
  A weak-news day should become “waiting for stronger evidence,” not “failed.”
- [x] `LL-203` Add automatic retry/backoff or checkpoint resume for every
  retryable source/model failure class.
- [ ] `LL-204` Add provider fallback only after the same evaluation corpus,
  privacy terms, cost reservation, and quality gates pass.
- [x] `LL-205` Prevent duplicate same-day/manual work and repeated near-identical
  lessons using concept, source, and objective similarity checks.
- [x] `LL-206` Set and display an honest first-lesson preparation-time range.
- [x] `LL-207` Remove contradictory empty and “first lesson” states from
  established streams.
- [x] `LL-208` Add a learner-facing recovery action tied to each safe failure
  category: retry, replace source, broaden source policy, or contact support.
- [ ] `LL-209` Alert on failure-rate regression, consecutive Account failures,
  queue age, provider truncation, and depleted model budget.
  Repository metrics and rules are complete; keep this open until staging rules
  and notification delivery are exercised successfully.
- [ ] `LL-210` Run the provider evaluation corpus and a 72-hour staging soak.

#### Exit gate

Reliability gates in section 7 pass in staging and for the design-partner
cohort. Broad launch work stops if they regress.

### Phase 3 — build credible source intelligence

Target: 2–3 weeks.

The existing deterministic ranking is a strong safety/relevance foundation,
but it does not yet justify a broad “we find the most trustworthy sources”
claim.

#### Data and evaluation

- [x] `LL-301` Create a versioned, public-safe source evaluation corpus across
  at least 50 launch-ICP topics.
- [x] `LL-302` Define labeling criteria: topical relevance, source authority,
  primaryness, recency, explanatory usefulness, independence, accessibility,
  and counterweight value.
- [ ] `LL-303` Have a human label the top candidates and calculate precision@5,
  domain diversity, role coverage, and unsafe/unusable rejection.
  The evaluator and 50-topic seed are ready; this remains open until blinded
  human labels replace `awaiting_human_labels` and the retained metrics pass.

Public-source retrieval follows [ADR-0007](adr/0007-public-source-retrieval-policy.md):
a missing `robots.txt` file is not a blocker, while authentication/paywall/
CAPTCHA bypass, credentialed or private-network targets, unsafe redirects, and
claims of unrestricted republication rights remain prohibited. Counsel review
and a tested rights-holder complaint/removal response remain launch evidence;
there is no runtime source-policy approval flag. Migration 040 adds an
append-only operator response control for exact-URL and registrable-domain
retrieval blocks and audited reversals; enforcement covers initial requests
and redirect targets and fails closed when policy cannot be checked.

#### Discovery model

- [x] `LL-304` Replace the fixed three-query bundle with topic-aware query
  planning that seeks source roles rather than keyword variations.
- [x] `LL-305` Classify candidate roles: official/primary, peer-reviewed or
  research, practitioner/explainer, reporting/context, and counterweight.
- [x] `LL-306` Add authority signals appropriate to the source class. Examples:
  official-domain match, canonical organization ownership, DOI/repository
  evidence, author/organization identity, update date, citations, and stable
  documentation structure.
- [x] `LL-307` Keep search-engine rank as one signal, never the trust decision.
- [x] `LL-308` Add negative signals for SEO farms, copied/aggregated material,
  anonymous unsupported advice, stale versioned documentation, thin pages, and
  misleading title/snippet matches.
- [x] `LL-309` Score portfolio coverage and independence, not only individual
  URLs.
- [x] `LL-310` Store score components, role, selection reason, and ranking
  version for auditability.
- [x] `LL-311` Add learner source controls: prefer, block, replace, “use only
  these,” and “fill gaps.”
- [x] `LL-312` Re-evaluate stale or failing sources without silently changing
  the evidence frozen for an existing lesson.

#### Experience

- [x] `LL-313` Show a source portfolio preview with “why selected” explanations
  before the first lesson.
- [x] `LL-314` Let the learner approve the portfolio or begin immediately with
  a reversible default.
- [x] `LL-315` Explain weak coverage and request a source or broader policy
  rather than generating from a poor portfolio.

#### Exit gate

The source-intelligence gates pass on the labeled corpus and design partners
prefer the selected portfolio to their normal starting workflow in at least
8 of 10 observed tasks.

### Phase 4 — replace configuration-first onboarding with first value

Target: 2 weeks.

#### New journey

```text
topic/outcome
  -> source and path preview
  -> first short lesson
  -> first retrieval/application
  -> schedule and refine
  -> optional publishing later
```

#### Tasks

- [x] `LL-401` Replace the signup CTA with “Build my learning path.”
- [x] `LL-402` Ask for one topic/question, one desired capability/outcome, and
  current familiarity. Keep time available optional or defaulted.
- [x] `LL-403` Default to autonomous discovery. Move provided-only and hybrid
  control to “Use specific sources” without hiding the trust choice.
- [x] `LL-404` Generate a fast research-plan preview: source roles, initial
  concepts, likely first lesson, and preparation estimate.
- [ ] `LL-405` Offer 5–8 human-reviewed starter paths for the selected ICP with
  precomputed source portfolios and sample lessons.
- [x] `LL-406` Allow a visitor to inspect one complete canonical Dossier before
  signup.
- [x] `LL-407` Allow onboarding to be abandoned and resumed across devices.
- [x] `LL-408` Prepare the first lesson automatically after confirmation; do
  not land the learner on a status-management screen.
- [x] `LL-409` While waiting, show a useful orientation, allow safe exit, and
  notify when ready.
- [x] `LL-410` Ask for cadence, email, and detailed source tuning after the
  first value preview, not before it.
- [x] `LL-411` Ask about a public site only after the first completed lesson.
- [x] `LL-412` Instrument each step, source-policy choice, wait abandonment,
  first open, first retrieval, and activation.

#### Exit gate

Activation gates pass for at least 30 qualified new users. If arbitrary-topic
latency misses the gate, starter paths remain the primary beta entry rather
than disguising the delay.

### Phase 5 — turn Dossiers into adaptive learning sessions

Target: 2–3 weeks.

#### Learning contract

- [x] `LL-501` Add a durable lesson type: foundation, update, deep dive,
  synthesis, application, or review.
- [x] `LL-502` Require a selection rationale, prior-learning connection,
  prerequisites, concepts changed, and suggested next concepts.
- [x] `LL-503` Require claim-to-source mappings for material factual claims.
- [x] `LL-504` Store confidence/limitation metadata without presenting a fake
  numerical certainty score to learners.
- [x] `LL-505` Keep internal skeptical/editorial audits in structured metadata;
  never render the raw audit as the core public lesson.

#### Lesson generation and quality

- [x] `LL-506` Make 8–15 minutes the default core lesson and test real reading
  time rather than prompt-requested word count alone.
- [x] `LL-507` Require one central mechanism, one worked example, one important
  limitation or counterexample, and two high-quality retrieval/application
  prompts.
- [x] `LL-508` Add semantic duplication checks against prior stream concepts,
  objectives, and source evidence.
- [x] `LL-509` Use learner level, feedback, recall state, and available time in
  curation and teaching stages.
- [x] `LL-510` Generate an optional deeper evidence appendix instead of forcing
  every reader through research-process detail.
- [x] `LL-511` Expand the evaluation corpus to include usefulness, continuity,
  difficulty fit, unsupported claims, redundancy, and time fit.
- [ ] `LL-512` Add human review for every canonical starter-path lesson and for
  a sampled percentage of beta output.

#### Learning session UX

- [x] `LL-513` Open with “why this now,” “builds on,” objective, and time.
- [x] `LL-514` Attach source detail to claims on demand.
- [x] `LL-515` Autosave reading position and learner responses durably.
- [x] `LL-516` Require an answer-before-reveal retrieval or application step
  for activation, while allowing the learner to skip without punishment.
- [x] `LL-517` Capture “too basic,” “too advanced,” “not relevant,” and
  “question this claim” feedback in the learning state.
- [x] `LL-518` End with capability gained, next likely direction, and next
  review timing.
- [x] `LL-519` Provide previous/next navigation and return-to-path context.

#### Exit gate

At least 8 of 10 design partners rate sampled lessons as worth their time, the
unsupported-claim and citation gates pass, and first-lesson completion exceeds
the activation threshold.

### Phase 6 — make progress and Today genuinely adaptive

Target: 2 weeks.

The repository already stores reviews, concept state, feedback, retention, and
curriculum metadata. This phase turns that foundation into the product's felt
advantage.

#### Tasks

- [x] `LL-601` Move Today selection from first-ready ordering to a durable,
  tested priority model using in-progress state, review urgency, goal relevance,
  prerequisites, evidence strength, available time, and neglected paths.
- [x] `LL-602` Store and explain the selection reason in learner language.
- [x] `LL-603` Add an evidence-led rhythm that may defer generation, plus daily,
  selected-weekday, and weekly-synthesis choices.
- [x] `LL-604` Automatically reduce cadence when unopened lessons accumulate.
- [x] `LL-605` Add a path view showing outcome, concepts covered, current gaps,
  confidence/recall, and likely next directions.
- [x] `LL-606` Replace lesson counts as progress with “you can now…” capability
  milestones backed by completed/reviewed concepts.
- [x] `LL-607` Make the first retrieval immediate; use the durable review queue
  for later spacing.
- [x] `LL-608` Make weekly recap content reflect capability, one connection,
  one retrieval prompt, and one next action.
- [x] `LL-609` Preserve the gentle re-entry model and add one-click rhythm
  reduction, pause, or reset.
- [x] `LL-610` Remove decorative/fabricated progress visualizations that are not
  calculated from real learner state.

#### Exit gate

Seven-day and week-four retention gates pass for the design-partner cohort, and
interviews attribute return behavior to continuity, mastery, or saved time—not
merely novelty.

### Phase 7 — simplify publishing and make it a growth loop

Target: 1–2 weeks.

#### State model

- [x] `LL-701` Replace binary `published/hidden` lesson semantics with
  `private/draft/published` through an explicit migration and compatibility
  plan.
- [x] `LL-702` Default all new content to private or draft. Never auto-publish
  merely because a site and stream are visible.
- [x] `LL-703` Present one site mode: private workspace or public learning site.
- [x] `LL-704` Present one default for completed lessons: keep as drafts or
  publish automatically; drafts are the recommended initial default.
- [x] `LL-705` Move search indexing into an advanced discovery/SEO section that
  does not alter link privacy.
- [x] `LL-706` Show the effective audience and state on every lesson.
- [x] `LL-707` Add first-publish review, exact visitor preview, bulk actions,
  and clear unpublish behavior.

#### Public growth surface

- [x] `LL-708` Add owner introduction, outcome-oriented path descriptions,
  follow-by-email, related/next Dossier, share controls, and a contextual
  “Build your own path” CTA.
- [x] `LL-709` Add privacy-safe owner analytics: views, follows, shares, and
  visitor-to-signup conversion.
  Views, confirmed follows, shares, path-start clicks, and attributed signup
  conversion are pseudonymous, bot-filtered, daily-deduplicated, owner-scoped,
  and visible in 7/30/90-day aggregates.
- [ ] `LL-710` Create three to five founder-curated public learning paths with
  no broken or fictitious subdomains.
- [x] `LL-711` Add canonical social previews for real public Dossiers.
- [x] `LL-712` Maintain moderation, correction, noindex, and operator-removal
  safeguards before scaling public content.

#### Exit gate

Five observed users can correctly predict who can see a lesson without help;
no accidental publication occurs in testing; and shared canonical Dossiers
produce measurable visitor-to-signup conversion.

### Phase 8 — rebuild marketing around the proven product

Target: 1–2 weeks after Phases 3–6 have real evidence.

#### Message hierarchy

1. Pain: keeping up produces links, not command of the field.
2. Promise: give Learnloom a topic; it builds and maintains the learning path.
3. Mechanism: source intelligence + progressive lessons + retrieval +
   continuity.
4. Proof: a real source portfolio, lesson sequence, and mastery progression.
5. Control: bring sources, constrain the boundary, inspect evidence.
6. Optional identity: publish selected learning to a personal site.

#### Tasks

- [x] `LL-801` Replace the hero, title, metadata, social copy, and CTA with the
  topic-to-curriculum promise.
- [x] `LL-802` Demonstrate one complete sequence: topic → source portfolio →
  lesson → retrieval → adapted next lesson.
- [x] `LL-803` Make a real Dossier available before signup.
- [x] `LL-804` Add a factual comparison against ChatGPT, NotebookLM, Readwise,
  Recall, and feed/read-later workflows.
- [x] `LL-805` Add pricing and an explicit trial/beta explanation.
- [x] `LL-806` Add source methodology, privacy, correction, and model-limitation
  trust content near the conversion path.
- [ ] `LL-807` Add outcome-based testimonials only: time saved, decisions
  improved, concepts recalled, or professional confidence gained.
- [x] `LL-808` Remove unsupported mockups, fictitious examples, and promises not
  present in the deployed product.
- [ ] `LL-809` Create persona/problem pages only when each has a distinct buyer,
  query intent, real example, and conversion path.
- [ ] `LL-810` Run five-second comprehension and unmoderated funnel tests with
  at least 20 target users.

#### Exit gate

At least 80% of tested target users can explain who Learnloom is for, the main
outcome, and its difference without prompting. Qualified visitor-to-signup
conversion reaches the target range before paid acquisition.

### Phase 9 — add billing, entitlements, and commercial operations

Target: 2 weeks.

#### Pricing hypothesis

Test one simple paid plan before introducing complexity:

- **Free:** one starter path, a bounded number of learning cycles, private
  archive, and enough review to experience the loop.
- **Pro hypothesis:** test $12, $15, and $19 monthly willingness-to-pay; offer a
  founding annual price only after cost and retention evidence exists.
- **Later professional/team plan:** deferred until interviews reveal a distinct
  buyer, collaboration need, and support model.

The final allowance must be derived from recorded cost and desired gross
margin, not the phrase “unlimited AI.”

#### Tasks

- [x] `LL-901` Select a payment provider after reviewing supported countries,
  tax/VAT handling, payouts, subscriptions, webhooks, refunds, and customer
  portal behavior.
- [x] `LL-902` Model plans, entitlements, trial state, subscription state,
  allowance period, and usage reservations in durable storage.
- [x] `LL-903` Make payment webhooks signed, idempotent, replay-safe, and
  auditable.
- [x] `LL-904` Enforce entitlements server-side before work is created; never
  rely on hidden UI.
- [x] `LL-905` Reserve usage atomically so concurrent manual/scheduled work
  cannot overspend an allowance.
- [x] `LL-906` Define grace, payment failure, downgrade, cancellation, refund,
  and reactivation semantics.
- [x] `LL-907` Preserve access to learned content after downgrade; stop or
  reduce new paid generation according to the published policy.
- [x] `LL-908` Add a self-service billing portal and clear in-product usage
  explanation.
- [ ] `LL-909` Add invoice/tax/legal copy appropriate to the business entity and
  sales jurisdictions.
  Production checkout now fails closed behind explicit commerce approval and a
  non-secret evidence reference. Provisional source terms now disclose the
  accepted missing-`robots.txt` retrieval policy, access-control boundary,
  attribution/republication limits, and rights-holder contact path;
  counsel/accountant/entity review remains external and unperformed.
- [x] `LL-910` Instrument trial start, paywall exposure, checkout, payment,
  cancellation, refund, and reason.
- [ ] `LL-911` Run sandbox and staging lifecycle tests for every payment state,
  duplicate/out-of-order webhook, and provider outage.
- [x] `LL-912` Add revenue, COGS, gross-margin, trial-conversion, churn, and
  cohort dashboards.

#### Exit gate

Ten design partners voluntarily pay; the full payment lifecycle is proven in
staging; plan allowance maintains the gross-margin gate under representative
usage.

### Phase 10 — paid design-partner beta

Target: minimum 4 weeks; do not compress the retention observation window.

#### Tasks

- [ ] `LL-1001` Onboard 20–30 selected design partners personally.
- [ ] `LL-1002` Observe first setup and first lesson without coaching unless
  blocked.
- [ ] `LL-1003` Hold weekly outcome interviews and review behavioral data.
- [ ] `LL-1004` Sample source portfolios and lessons for quality every week.
- [ ] `LL-1005` Ask for payment after the learner has experienced a completed
  learning cycle, not before proof of value.
- [ ] `LL-1006` Record non-conversion and cancellation reasons using a stable
  taxonomy plus verbatim notes.
- [ ] `LL-1007` Fix the largest activation or retention constraint each week;
  avoid adding unrelated features.
- [ ] `LL-1008` Collect permissioned outcome stories and public examples from
  successful users.
- [ ] `LL-1009` Calculate cost and gross margin by retained paid cohort.

#### Exit gate

Commercial gates pass, at least 30 paying customers exist, and week-four usage
demonstrates repeated learning value. Otherwise remain in beta and address the
dominant constraint.

### Phase 11 — launch distribution system

Target: begins during beta; scales only after commercial gates.

#### Founder-led channel

- [ ] `LL-1101` Publish weekly demonstrations using real source portfolios,
  lessons, and progress—not generic AI-learning advice.
- [ ] `LL-1102` Share useful artifacts in AI engineering/product communities
  where the underlying lesson answers a real question. No mass posting.
- [ ] `LL-1103` Build relationships with technical newsletter authors,
  educators, researchers, and community operators.
- [ ] `LL-1104` Offer reviewed public paths that partners can share with their
  audience while preserving source and correction transparency.

#### Product-led channel

- [x] `LL-1105` Measure public Dossier view → follow → signup → activation.
- [ ] `LL-1106` Add referrals only after activated users naturally share; test
  rewards tied to a useful product allowance rather than cash first.
- [ ] `LL-1107` Let experts publish curated source policies/path templates only
  after moderation and attribution models are ready.

#### Search and authority channel

- [ ] `LL-1108` Replace generic SEO volume with original worked examples,
  source-evaluation methodology, comparisons, and domain-expert review.
- [ ] `LL-1109` Configure Search Console/Bing and review query → activation
  monthly.
- [ ] `LL-1110` Scale only pages with distinct intent, substantive evidence,
  real output, and measurable activation.

#### Paid channel

- [ ] `LL-1111` Do not buy broad traffic before paid retention and gross margin
  are known.
- [ ] `LL-1112` Test narrow high-intent campaigns only after an organic/founder
  page converts the same persona.
- [ ] `LL-1113` Stop campaigns whose cohort payback or retained activation is
  below the agreed threshold even when signup volume looks attractive.

#### Exit gate for scaling

At least one channel produces repeatable activated paid customers with
acceptable contribution margin. Scale the winning channel; do not distribute
effort evenly across every channel.

### Phase 12 — public launch and post-launch cadence

#### Final launch requirements

- [ ] All repository and staging gates in `docs/public-launch-checklist.md`
  pass with dated evidence.
- [ ] Reliability, activation, retention, commercial, and gross-margin gates in
  this plan pass.
- [ ] Real examples, pricing, support, privacy, terms, refund, billing, and
  cancellation experiences are live.
- [ ] Incident response, backups, restore, alerts, spend controls, payment
  reconciliation, and correction/moderation responsibilities have owners.
- [ ] The live site and repository describe the same shipped product.

#### Weekly operating cadence after launch

- Review acquisition → activation → learning cycle → retention → payment by
  cohort.
- Review source and lesson quality samples.
- Review failures, latency, spend, gross margin, and support volume.
- Speak with at least five active, inactive, trial, or churned users.
- Choose one funnel constraint for the next week.
- Publish one real proof artifact or customer outcome.

## 10. Critical path

```text
production baseline
  -> buyer evidence
  -> reliability
  -> source-intelligence quality
  -> first-value onboarding
  -> adaptive lesson/mastery loop
  -> paid design-partner retention
  -> billing proof
  -> repeatable distribution
  -> public launch
```

Marketing, publishing polish, and paid acquisition cannot compensate for a
failure earlier on this path.

## 11. Parallel work that is safe

While reliability and source intelligence are being implemented, the following
can proceed without creating expensive rework:

- interview recruitment and problem research;
- canonical starter-path topic/source curation;
- source and lesson evaluation-corpus labeling;
- staging/release evidence collection;
- billing-provider/legal research;
- real example and testimonial permission workflow;
- analytics query/dashboard preparation.

Homepage implementation should wait until the source and learning promises are
stable enough to demonstrate truthfully.

## 12. Explicit non-goals before launch

Unless interviews overturn the launch hypothesis, defer:

- team workspaces and classroom management;
- native mobile apps;
- community-generated template marketplace;
- arbitrary free-text automated grading;
- gamified streaks and broad social networking;
- elaborate public-site themes;
- many pricing tiers;
- enterprise SSO and procurement features;
- paid acquisition at scale;
- expansion to every professional or hobby-learning category.

These features can generate activity without proving the core learning or
revenue loop.

## 13. First ten implementation issues

The first implementation sequence after plan approval is:

1. `LL-001` Production/repository parity inventory.
2. `LL-003` Baseline activation/retention query and dashboard.
3. `LL-004` Generation failure-rate and classification report.
4. `LL-005` Cost per generated/opened/completed/retained lesson report.
5. `LL-008` Correct discovery messaging contradiction.
6. `LL-201` Reproduce top production failure classes.
7. `LL-202` Add evidence-deferral semantics.
8. `LL-203` Complete automatic retry/recovery matrix.
9. `LL-301` Create the source-intelligence evaluation corpus.
10. `LL-302` Write human source-labeling criteria.

The interview work in Phase 1 starts concurrently on day one.

## 14. Definition of “best product before launch”

Learnloom is ready when a target professional can:

1. arrive with only a topic and professional outcome;
2. understand the promise immediately;
3. inspect and trust the proposed information environment;
4. receive a concise, useful lesson reliably;
5. see why the lesson matters now and how it connects;
6. attempt retrieval or application and have the system adapt;
7. feel growing professional capability over several weeks;
8. control or publish their work without privacy confusion;
9. decide that the saved time and accumulated command are worth paying for;
10. explain Learnloom to another person in one sentence.

That is the launch product. Everything else is supporting infrastructure,
future expansion, or distraction.
