# Proposed content creation flow

Date: 2026-09-08. Status: design proposal. These are candidate changes, not existing behavior or an approved architecture change.

## 1. Turn the topic into an editable learning brief

Keep initial setup short. After a topic is entered, suggest a useful outcome in plain language:

> You want to learn production RAG evaluation. A useful starting point is distinguishing missing retrieval evidence from incorrect use of retrieved evidence. We will use small examples; you do not need your own system logs.

Let the learner accept, edit, or choose a broader exploratory direction. Ask a follow-up only when the answer changes the lesson: “Have you inspected retrieved passages before?” is useful; collecting a long biography is usually not.

The brief should contain:

- **Desired capability:** an observable action, explanation, or distinction.
- **Context:** where it matters, with a synthetic example if no personal context is available.
- **Starting point:** prerequisites the learner reports, plus uncertainty about them.
- **Time:** total core-session budget and whether hands-on work is feasible.
- **Scope:** what this session will and will not establish.
- **Evidence need:** the claims or examples that sources must support.

Use progressive disclosure for source settings and advanced preferences. The existing source preview should eventually explain the intended learning payoff, not only show source availability. A proposed plan must be labeled as a plan until the supporting evidence is resolved.

If a learner enters only “economics,” offer a concrete first question, such as explaining why a slowing rate of price increase can coexist with rising prices. Do not silently assume a career goal. Avoid blocking creation solely because the learner cannot yet articulate a precise outcome.

## 2. Select the next capability before committing to a theme

Consider a small set of candidate lessons using current intent, prior exposure, explicit feedback, unresolved questions, and available evidence. The initial version can be a structured editorial decision within existing stages.

| Candidate | Choose when | Evidence needed |
| --- | --- | --- |
| Foundation | A prerequisite is missing or starting knowledge is uncertain | A clear explanation and simple discriminating example |
| Deep dive | The learner knows the basic distinction but cannot explain the mechanism | Mechanism, conditions, and a nontrivial example |
| Application | The learner can explain but has not used the concept | Task inputs, solution reasoning, and evaluation criteria |
| Review | A specific capability needs retrieval or correction | Prior material plus a changed question or context |
| Synthesis | Several learned ideas need connecting | A comparison or decision requiring those ideas together |
| Update | New evidence changes a prior conclusion or method | The earlier claim, new evidence, and precise change |

These types already exist in the domain. The proposal is to strengthen their selection criteria and editorial treatment.

Prefer an explicit reason such as “You identified the concept correctly but missed its boundary condition” over “This is personalized for you.” Where there is only self-report, say so. An unread generated lesson must not be treated as a mastered prerequisite.

Do not invent novelty merely to maintain daily output. A useful review can repeat a concept while changing what the learner must retrieve or apply. Distinguish legitimate reinforcement from a paraphrased duplicate.

## 3. Acquire evidence for the teaching task

Preserve the catalog-first, bounded native retrieval path and freeze evidence before generation. Add a task-oriented assessment of whether the selected evidence is sufficient.

The audit found that hosted feed preparation stores bounded feed text and the prepared generation path skips linked-article enrichment. Inspect the text actually frozen for each Issue. Where it is only a teaser, the proposed improvement is bounded linked-article resolution before evidence freezing, with an explicit fallback status if the article cannot be obtained. Do not assume all feeds are short or that a long body is automatically sufficient.

A good evidence packet should answer:

1. Which source supports the central mechanism?
2. What supports the worked example's factual premises?
3. What limits the conclusion?
4. Are apparently independent sources repeating the same original report?
5. Is the material current enough for this claim?
6. Can the requested lesson be taught honestly from this evidence?

Freshness should depend on the claim. A foundational mathematical explanation and a current software API do not have the same recency needs. Track publication date, retrieval date, and applicable version separately. Several recent summaries should not displace an older primary explanation solely because it is older.

Prefer complementary evidence roles: explanation, example, boundary, and disagreement where relevant. Do not manufacture disagreement or require equal weight for weak claims. Source count is an operational constraint, not a measure of truth.

### Proposed claim ledger

Before prose, capture the few claims that materially drive the lesson:

| Field | Purpose |
| --- | --- |
| Claim and claim type | Distinguish sourced fact, interpretation, and explicitly synthetic illustration. |
| Snapshot and passage locator | Let a reviewer inspect the actual support. |
| Support relationship | Direct support, inference with assumptions, contradiction, or missing support. |
| Scope | Population, version, conditions, and date when relevant. |
| Teaching role | Mechanism, example premise, limitation, or application. |
| Disposition | Keep, narrow, qualify, or remove. |

This is a proposed content contract, not a prescribed database schema. Start with editorial artifacts in a pilot. Validate that recorded passages actually exist; an LLM-generated locator is not self-validating.

If evidence is thin, narrow the lesson, offer a supported foundation/review, or explain that a new lesson cannot yet be supported. Any alternative should be selected explicitly and remain within the stream's intent. Do not quietly convert insufficient evidence into confident general advice.

## 4. Design the assessment before drafting the explanation

Use backward design: write what successful performance looks like, then design the lesson needed to reach it. This is a proposed editorial method, not a claim that every learner needs a formal examination.

For each objective, identify:

- A short retrieval question that tests the mechanism or distinction.
- A misconception question that distinguishes plausible wrong reasoning.
- A changed-case task that checks application rather than copying.
- Essential answer criteria and acceptable alternatives.
- The concept each prompt tests.
- A corrective explanation for likely errors.

Drafting practice early exposes objectives that are vague or too broad. The examiner can still refine questions after prose, but should preserve alignment with the agreed objective. Separate a writing preference from a factual error when assessing open answers.

Avoid three versions of “define the term.” Avoid asking for an unstated fact. Do not make personal data, paid tools, or a working production environment necessary to complete a basic lesson; supply synthetic inputs or an offline alternative.

## 5. Draft one coherent learning experience

The existing blueprint is the starting point. Strengthen it with an observable outcome, success criteria, example inputs, the reasoning steps to demonstrate, an evidence boundary, and an effort budget.

A useful sequence is:

1. Show the situation and the question the learner will answer.
2. Retrieve a relevant prerequisite, with a brief recovery explanation available.
3. Explain one central mechanism using a concrete representation.
4. Work through a case, including why alternatives are less suitable.
5. Contrast a plausible misconception or boundary case.
6. Ask the learner to solve a changed case.
7. Give feedback and name the next useful question.

Keep existing required headings during an initial experiment so editorial improvements can be evaluated without changing the rendering contract. Different visible formats can come later if the fixed structure is shown to obstruct a particular lesson type.

Use diagrams, tables, code, or equations when they carry reasoning. A table of case evidence can be more useful than another paragraph. Decorative imagery is not evidence of teaching value. Keep accessible textual explanations and test the actual rendered output if richer formats are introduced.

## 6. Budget the entire session

Proposed illustrative allocation for a 12-minute session:

| Activity | Initial allocation |
| --- | --- |
| Recall and goal | 1 minute |
| Mechanism and worked example | 5 minutes |
| Independent attempt | 3 minutes |
| Feedback and takeaway | 2 minutes |
| Transition/buffer | 1 minute |

These are planning assumptions to measure, not established reading speeds. Optional source reading, extended critique, and longer experiments should have their own effort labels. A “practical experiment” that takes 30 minutes should not be hidden inside the core 12-minute promise.

When over budget, reduce the number of concepts, shorten duplicated setup, or move optional depth out of the core. Do not remove reasoning steps or answer criteria simply to meet a word target. When under budget, do not pad: check whether the task is too shallow or whether the lesson honestly needs less time.

## 7. Review the exact publishable artifact

Retain the deterministic gate. Add editorial review of the final lesson, critique, questions, answers, and relevant source passages together.

The reviewer should return specific findings: claim text, supporting or conflicting passage, severity, and a proposed bounded correction. “Needs more depth” is not enough. A failed grounding check should lead to a narrower claim, not an extra citation marker.

Review responsibilities:

- **Evidence:** claims do not exceed the source's scope; hypothetical numbers are labeled.
- **Teaching:** the example demonstrates the promised reasoning.
- **Assessment:** the answer follows from the taught material and changed-case inputs.
- **Continuity:** the lesson adds capability or justified retrieval, not cosmetic novelty.
- **Session fit:** the core task can plausibly be completed in the promised time.

A separate reviewer call is only one possible implementation. First test whether structured review in the existing skeptic/editor stages detects failures. A different prompt or model is not automatically an independent source of truth; calibrate against human labels.

Allow bounded repairs with a recorded reason. Recheck all affected artifacts after a change. The fallback path needs the same semantic acceptance conditions as the normal path; preserving old practice should trigger an alignment check with the final lesson.

## 8. Turn learner interaction into useful next-step evidence

Keep response capture and explicit skips. A skip means unknown, not incorrect. A revealed answer is not proof of independent performance. Proposed future assessment fields should distinguish: before/after reveal, self-rating versus rubric judgment, assessed concept, and uncertainty.

Proposed initial correction experience:

> Your answer correctly identifies missing evidence. The part to revisit is whether the provided passages are sufficient to answer the question. Compare the passage content before deciding which component to change.

Give the learner a small retry with changed inputs. Avoid a full replacement lecture after every error. Model feedback should be framed as revisable and allow “the question is ambiguous” or “the answer seems wrong.” Handle content defects separately from learner errors.

For future adaptation, distinguish:

| Signal | Safe inference | Unsafe inference |
| --- | --- | --- |
| Lesson generated | Content exists | Learner has studied it |
| Lesson completed | Learner marked/read the session | Capability mastered |
| Answer attempted | Learner engaged | Answer correct |
| Self-rated solid | Learner reports confidence | Independent mastery established |
| Correct changed-case answer | Evidence for that task and concept | All lesson concepts mastered |
| Delayed success | Evidence of retention in this setting | Permanent mastery |

Do not send additional raw learner responses to a model by default as part of this documentation work. Any future assessment flow needs an explicit data-minimization and ownership design. Existing confidence should remain interpretable as its actual source of evidence.

## 9. Improve sequences, not just individual lessons

Evaluate short learning sequences with a beginning, a visible advance, and a return to earlier knowledge. A useful three-session sequence could introduce a distinction, apply it to a more ambiguous case, then combine it with a prior concept.

Maintain separate records of material delivered and capability demonstrated. A concise continuity note should capture the earlier task, observed difficulty, and next useful step; a prefix of the last lesson is not necessarily a useful learning summary.

Update content when material evidence changes. Preserve the original artifact and evidence identity; design a visible correction/version relationship rather than silently rewriting what the learner previously read. Details of storage, notification, and source-rights handling require a future design review.
