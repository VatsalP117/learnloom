# Evaluation and prioritized roadmap

Date: 2026-09-08. Status: proposed work; no experiments or application changes have been executed for this audit.

## Establish a baseline before changing generation

Retain the existing structural and configured-provider evaluation gates. Extend them with complete artifacts rather than replacing them with another broad quality score.

A practical first editorial pilot is 24 complete lessons: three subject families × two starting levels × two time budgets × two history conditions. Example families are a technical diagnostic skill, a quantitative conceptual distinction, and an explanatory/synthesis topic. History conditions should include a first lesson and a continuation with a specific weak prerequisite. This is a deliberately small coverage pilot, not a statistically powered proof of learning improvement.

Use public or explicitly permitted source material and synthetic learner profiles. Review saved artifacts first if suitable ones exist; do not incur model spend just to establish a document plan. When generation is authorized later, record all failures and retries, not only successful lessons.

Each evaluation bundle should contain:

- The exact learner brief, relevant history, and requested session budget.
- Frozen source content or permitted excerpts with snapshot identity and dates.
- What text and metadata each model stage actually received, including truncation.
- Blueprint, final lesson, critique, practice, rubric, and rendered output.
- Pipeline/prompt version, model identity, repair/fallback path, latency, and cost.
- Independent reviewer judgments, supporting passages, disagreement, and resolution.

Sensitive learner data and licensed source bodies do not belong in a public test corpus. Where source rights restrict redistribution, use permitted excerpts or reconstructible metadata and keep the distinction explicit. Do not label a case human-reviewed without a reviewer and date.

## Rubric with observable anchors

Use the following proposed 0–3 scale per dimension. Score 1 means a major weakness remains between the 0 and 2 anchors. Keep evidence sufficiency and instructional quality separate.

| Dimension | 0: unusable or misleading | 2: adequate | 3: strong |
| --- | --- | --- | --- |
| Goal usefulness | No observable payoff for the learner | A relevant task with success criteria | Clear payoff and a convincing changed-case demonstration |
| Mechanism | Assertions or definitions only | Explains causal/logical steps where appropriate | Makes assumptions, alternatives, and limits inspectable |
| Example | Missing, incorrect, or unworkable | Complete inputs and reasoned solution | Reveals a subtle distinction through a useful contrast |
| Grounding | Material unsupported claim | Central claims supported with correct scope | Support is traceable and uncertainty teaches something useful |
| Practice | Unanswerable, trivial, or misaligned | Tests the objective with a correct rubric | Distinguishes misconceptions and tests transfer |
| Correction | Wrong or merely repeats an answer | Explains the key error | Offers a targeted retry and respects valid alternatives |
| Difficulty | Essential prerequisites unexplained | Fits starting knowledge with scaffolding | Handles uncertainty without tedious repetition |
| Continuity | Repeats or assumes unlearned material | Builds on relevant prior material | Responds to specific evidence and adds a justified advance |
| Effort fit | Core task clearly exceeds the promise | Plausible total-session fit | Observed fit with optional depth separated |

**Proposed initial editorial acceptance:** at least 2 in every applicable dimension and no critical defect. This is a decision rule for the pilot, not a scientifically established threshold. Record “not applicable” with a reason; do not use it to evade practice or grounding checks.

Critical defects include a material false/unsupported claim, incorrect answer key, impossible task, or advice that exceeds the supported scope. A high score elsewhere cannot compensate. Count critical misses separately from average ratings.

## Test complete generation as well as the evaluator

The existing 12-case corpus is useful for checking that a judge recognizes obvious examples of the six dimensions. It does not test the teacher's complete output, actual rendered time, or learner performance. Keep it as a smoke test and add:

| Test case | Failure it should expose |
| --- | --- |
| All headings present, generic text throughout | Mistaking format compliance for depth |
| Real citation ID, unsupported conclusion | Mistaking reference validity for support |
| Three questions testing only one of several concepts | Over-crediting untested concepts |
| Correct answer, wrong explanation | Treating rubric and correction as equivalent |
| Editor changes an example but preserves old answers | Final artifact misalignment |
| Teacher/examiner fallback after editor failure | Bypassing semantic acceptance in recovery |
| Feed teaser points to a rich article | Assuming the linked article was seen |
| Unsupported PDF or empty/dynamic HTML | Confusing reachability with usable evidence |
| Extracted navigation overwhelms explanatory text | Mistaking text volume for teaching support |
| Several sites repeat one original report | Mistaking URL diversity for independent evidence |
| Key caveat lies beyond the retained source prefix | Losing support conditions during truncation |
| Same objective phrased with synonyms | Missing repetition through lexical differences |
| Same evidence, new valid application | Rejecting useful reinforcement as duplication |
| Long practical task inside a short reading budget | Mistaking body length for session fit |
| Confident self-rating with incorrect reasoning | Mistaking confidence for mastery |
| Skipped retrieval | Treating missing evidence as a wrong answer |
| Recently fetched old version of technical documentation | Confusing fetch freshness with applicability |

Include both passing and failing variants so the system cannot succeed by rejecting everything. Also include ambiguous cases where “needs review” is better than a forced binary judgment.

## Human calibration and comparison

For the first pilot, have two reviewers independently score each artifact without seeing the producing variant. Randomize display order for paired comparisons. Ask reviewers to inspect actual evidence for central claims, not just evaluate how convincing the prose sounds.

Record per-dimension agreement and disagreements. Adjudicate disputed critical defects before using cases to calibrate an automated judge. Preserve original labels and the reason for revisions. Keep a held-out set separate from examples used to tune prompts.

For model judges, report false acceptance of critical defects, false rejection of useful lessons, and per-dimension agreement. An 80% aggregate match can hide a serious grounding weakness. A second model is a fallible reviewer; it does not independently verify a source unless it receives and evaluates the source.

Compare baseline and candidate using the same learner briefs and frozen evidence. Run enough repeated generations to see whether gains survive output variability; choose the repeat count and spend cap before the run. Report failures and exclusions. A paired win rate on 24 lessons is preliminary evidence with wide uncertainty, not grounds for a precise product claim.

## Learning and usefulness pilot

After editorial acceptance, use a small consented learner pilot to discover whether the content is usable. Start with qualitative observation and a modest number of participants; determine a quantitative sample size later from baseline variance and the minimum worthwhile effect.

For each lesson:

1. Establish the relevant starting capability with a short parallel task or explanation.
2. Observe the core session, including time spent attempting and interpreting feedback.
3. Use a changed-case task after the lesson, graded with a prewritten rubric.
4. Recheck the capability after a delay, initially around seven days as a practical pilot choice.
5. Ask what was useful, confusing, unnecessary, and applicable to the learner's own interests.

Do not reuse the worked example as the outcome test. Keep task difficulty comparable across variants. A pretest can itself influence learning, so use the same procedure across conditions and acknowledge that limitation.

For a causal comparison, randomize at learner or stream level when feasible so repeated lessons do not contaminate independent lesson-level assignments. At low traffic, use clearly labeled qualitative or within-person evidence and account for order effects. Do not claim causal improvement from a simple before/after cohort.

## Metrics and interpretation

| Metric | Definition | Important caveat |
| --- | --- | --- |
| Supported-claim rate | Supported audited material claims / all audited material claims | Define sampling; do not audit only easy claims. |
| Critical defect rate | Artifacts with at least one critical defect / all evaluated artifacts | Show failure and fallback paths separately. |
| Task success | Responses meeting prewritten essential criteria / attempted assessment tasks | Also report non-attempts against assigned tasks. |
| Delayed success | Successful delayed tasks / completed delayed tasks | Report follow-up participation and missingness separately. |
| Session fit | Sessions completed within the promised core budget / observed sessions | Report distribution and interruptions, not only an average. |
| Personal usefulness | Learner can name a concrete application or explanatory payoff | Self-report complements performance; it does not replace it. |
| Repetition defect rate | Unjustified repeated objectives/tasks / reviewed continuation lessons | Legitimate review is not a defect. |
| Cost per accepted lesson | Total generation/review/repair cost / accepted lessons | Include discarded generations and reviewer cost. |
| Cost per successful learning session | Total attributable cost / sessions meeting the task criterion | Report the criterion and observation coverage. |

Track completion and seven-day return as product outcomes, separately from learning. Avoid optimizing solely for easy tasks, flattering feedback, or completion speed. More challenging but useful lessons may require more effort; evaluate fit and benefit together.

No current values for these metrics were measured by this audit.

## Prioritized work packages

Relative effort is an estimate of scope, not a delivery promise. Product, privacy, and architecture choices remain future decisions.

| Priority | Work package | Why now | Relative effort | Proposed acceptance evidence |
| --- | --- | --- | --- | --- |
| P0 | Full-artifact baseline and reviewer calibration | Establish which problems actually occur | Medium | Reviewed baseline bundles, disagreement log, defect inventory, source completeness audit |
| P0 | Hosted feed evidence completeness | Missing source detail constrains every later stage | Medium | Teaser feed/exact article/full-text feed/fetch failure cases retain accurate evidence status and sufficient support |
| P1 | Outcome–example–assessment alignment | Directly addresses generic usefulness | Small–medium | Paired review improves usefulness and practice without grounding regressions |
| P1 | Final claim and answer review | Prevents trustworthy-looking errors | Medium | Critical seeded defects detected on normal, repaired, and fallback artifacts |
| P1 | Total-session budgeting | Makes the time promise meaningful | Small–medium | Observed pilot fit, with tasks and optional depth accounted for |
| P2 | Per-question concepts and distinct corrective explanations | Makes practice and adaptation more specific | Medium | Unrelated concepts receive no credit; feedback addresses the actual misconception |
| P2 | Source metadata and final portfolio review | Preserve useful upstream provenance | Medium | Metadata survives to review; final evidence roles and caveats are inspectable |
| P2 | Structured continuity and justified review | Improves a sequence beyond lexical novelty | Medium | Rephrased duplicates rejected, supported new applications accepted in sequence cases |
| P3 | Response-aware adaptation and targeted retries | Needs trustworthy assessment signals first | Larger | Conservative decisions, data-minimization design, better delayed task results |
| P3 | Multiple lesson formats and capability paths | Useful only if simpler changes hit format limits | Larger | Evidence that format flexibility improves a specific lesson type |

### First wave: discover and correct the biggest bottleneck

Prepare the baseline and inspect source completeness before changing prompts. If most weak artifacts lack source detail, address evidence first. If sufficient evidence produces generic teaching, prioritize blueprint alignment. Do not change source selection, prompt design, model, and format simultaneously; the comparison would not explain what helped.

### Second wave: strengthen one complete lesson

Test aligned briefs, correct examples/answers, and session fit within the existing headings and pipeline. Keep a bounded comparison against the baseline. Hold publication changes if critical defects increase even when stylistic preference improves.

### Third wave: improve the next lesson

After concept attribution and assessment calibration are trustworthy, evaluate multi-lesson sequences, review choice, and response-aware correction. Do not build a large mastery model on top of coarse self-confidence scores.

## Future implementation surfaces and verification

This is a map for later work, not authorization to change these files now.

- `internal/source/service.go`, `acquisition.go`, `discovery.go`: evidence completeness, metadata, final selection. Verify bounded fetching, source policy, immutable snapshots, retry reuse, and summary fallback identity.
- `internal/dossier/generator.go`, `stages.go`, `quality.go`: briefs, prompt contracts, final review and repair. Verify every publication/fallback path, context limits, and meaningful defect cases.
- `internal/dossier/learning_contract.go`, `internal/domain/domain.go`: concept attribution and richer assessment contracts. Version changed contracts and retain compatibility with old artifacts.
- `internal/store/concepts.go`, `reviews.go`, `retrievals.go`: future learner evidence. Verify owner isolation, skip/reveal semantics, idempotency, and no accidental mastery projection.
- `web/src/NewsletterCreate.tsx`, `IssueDetail.tsx`, `ReviewPage.tsx`: future intent preview, effort labeling, and feedback. Test accessibility, honest loading/error states, and response recovery.
- Existing evaluation docs and test corpora: retain structural smoke tests while introducing full-artifact and sequence evaluations.

Changes to evidence, prompts, or contracts must participate in the existing generation fingerprint/version strategy so checkpoints cannot silently reuse incompatible work. Delivery retries must remain separate from generation. Any schema or public API changes need their own design review; this audit prescribes none.

## Defer until evidence justifies them

Do not start with a larger model, more autonomous agents, a vector database, a full curriculum graph, compulsory diagnostics, or unrestricted web retrieval. Each may eventually solve a demonstrated problem, but none substitutes for useful objectives, adequate evidence, and correct practice. Likewise, do not relax existing gates to improve generation completion statistics.

## Decisions this audit cannot settle

The baseline should determine which failure dominates. Learner observation should determine how much setup people tolerate and which tasks they value. A controlled comparison should determine whether a new judge or stage earns its cost. Product judgment should determine whether evidence-led streams offer optional review when sources are unchanged.

The immediate deliverable is an evidence-backed improvement plan. Claims about improved learning, retention, cost, or satisfaction require the future evaluations described here.
