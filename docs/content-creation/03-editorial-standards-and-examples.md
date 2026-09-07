# Editorial standards and examples

Date: 2026-09-08. All lesson fragments and numbers below are synthetic. They illustrate proposed standards, not actual Learnloom output, empirical findings, or production recommendations.

## The minimum useful lesson

A lesson needs one answer to “What can I now do or explain?” It also needs a visible chain connecting the evidence, explanation, worked example, independent task, and feedback. A section heading is only a container for that chain.

| Element | Weak version | Useful version |
| --- | --- | --- |
| Objective | Understand evaluation | Classify a failed answer using its retrieved passages and identify the next diagnostic check. |
| Relevance | This is important in the modern world | This prevents changing the answer generator when the needed evidence was never retrieved. |
| Mechanism | Retrieval improves answers | Explain the separate evidence-availability and evidence-use questions. |
| Example | A team improves its system | Give the question, passages, answer, decision steps, and uncertainty. |
| Misconception | There are common pitfalls | Show a plausible wrong conclusion and the observation that refutes it. |
| Practice | List the terms | Diagnose a changed case and justify the diagnosis. |
| Feedback | Here is the correct answer | Name the missing distinction, explain it, and offer a short retry. |
| Takeaway | Keep learning and experimenting | State the reusable decision rule and where it stops being sufficient. |

Do not turn this table into a mandate for identical prose. A historical explanation, mathematical derivation, design critique, and practical debugging lesson need different examples and standards of evidence.

## Example A: a useful diagnostic lesson

### Weak sketch

> RAG combines retrieval and generation. Evaluate retrieval precision, recall, and answer quality. Good evaluation helps you improve your system. Try evaluating your RAG pipeline.

This gives vocabulary but no inspectable inputs, decision procedure, success criteria, or bounded task. The learner cannot tell whether they performed the exercise well.

### Proposed learning brief

- **Learner:** knows that a system retrieves passages before answering; has not diagnosed failures systematically.
- **Outcome:** given a failed answer and its retrieved passages, distinguish insufficient evidence from failure to use available evidence.
- **Prerequisite:** can identify what information would answer a question.
- **Boundary:** diagnosing these examples does not establish the best fix or expected production improvement.
- **Evidence requirement:** a published explanation of the component distinction and any empirical claims used in the final lesson. The fictional inputs below do not establish empirical performance.

### Worked case

Question: “What is the return window for this item?”

Synthetic case A:

- Retrieved text: “Unused items can be returned within 30 days of delivery.”
- System answer: “The return window is 60 days.”
- No other conflicting passage is provided.

Reasoning to model explicitly:

1. Identify the requested fact: the return window.
2. Check whether the supplied evidence contains that fact: it states 30 days.
3. Compare the answer with that evidence: 60 days contradicts it.
4. Classify this example as failure to use available evidence correctly.
5. State the limit: one case does not explain why the component failed or how often it fails.

Synthetic case B:

- Retrieved text: “Items ship within two business days.”
- System answer: “The return window is 60 days.”

The supplied passage cannot answer the question. This case has missing retrieval evidence and an unsupported answer. It can involve more than one failure; do not force mutually exclusive labels if the evidence permits both.

### Misconception

> Every wrong answer means the answer model needs to be replaced.

A wrong answer alone does not isolate the failing part. Inspect evidence availability before making that diagnosis. Even when evidence is available, further investigation is required before choosing an intervention.

### Independent transfer task

New question: “Does the warranty cover water damage?”

Retrieved text: “The warranty covers manufacturing defects. Accidental liquid damage is excluded.”

System answer: “All water damage is covered.”

Ask the learner to classify the failure, quote the decisive phrase from the provided case, and name one fact they would need before recommending a system-wide change. This changes the content while preserving the diagnostic distinction.

### Proposed answer rubric

- Essential: notices that the relevant evidence is present.
- Essential: identifies the contradiction with the exclusion of accidental liquid damage.
- Essential: does not infer a proven system-wide fix from one example.
- Acceptable: calls it an evidence-use or answer-grounding failure with equivalent reasoning.
- Insufficient: only says “the model hallucinated,” without examining the passage.
- Ambiguity note: the case concerns accidental liquid damage; do not overgeneralize the exclusion to every possible water-related defect.

### Corrective explanation

If the learner blames retrieval because the answer is wrong, point to the retrieved exclusion and ask: “What additional passage was necessary to reject this answer?” If none was needed, the specific mistake is evidence use. Give a new case with an absent exclusion to test whether the distinction now holds.

This is more informative than repeating the model answer. It also makes room for a learner to identify ambiguous wording in the question itself.

## Example B: useful conceptual learning without a workplace task

### Weak sketch

> Rates of change and levels are different. Understanding this is important when reading charts.

### Better teaching fragment

Use a deliberately simplified price index: 100, then 110, then 115. The level rises in both intervals. The second increase is smaller: approximately 4.5%, versus 10% in the first interval. Ask the learner to explain why a slower rate of increase does not imply a falling level.

Then give a new sequence, 200, 220, 231, and ask the learner to compare levels and percentage changes. The useful outcome is an explanation of a distinction, not a personal financial decision.

The answer should recognize increases of 10% and 5%, respectively, for that second sequence: 200 to 220 is 10%; 220 to 231 is 5%. Require reasoning with the correct denominator. A corrective explanation should address confusing a smaller positive increase with a negative change.

A real lesson must establish the relevant definitions from sources and distinguish this synthetic index from measured inflation data. It should not append policy or investment conclusions that the example cannot support.

## Example C: useful synthesis under disagreement

Weak: summarize two articles and end with “there are pros and cons.”

Better: identify the question on which the sources differ, compare definitions and assumptions, and explain what observation would favor one account. The independent task asks the learner to classify a new piece of evidence as supporting one view, both, or neither, with a reason.

If the apparent disagreement comes from different time periods or populations, teach that distinction. Do not create a balanced debate where one source lacks support. A strong outcome can be “explain why these claims are not actually contradictory.”

## Reusable blueprint template

This is a proposed editorial template. It is intentionally Markdown, not a new runtime schema.

```text
Working title:
Learner's desired capability:
Situation or motivating question:
Prior capability actually evidenced:
Unknown prerequisite and recovery explanation:
One-session objective:
What successful performance looks like:
Central mechanism:
Evidence supporting that mechanism:
Important limitation or disagreement:
Worked example inputs:
Worked example reasoning steps:
Synthetic versus sourced details:
Plausible misconception:
Changed-case task:
Essential answer criteria:
Acceptable alternative reasoning:
Concept tested by each question:
Corrective explanation for likely errors:
Core-session effort estimate:
Optional extension and its separate effort:
Why this lesson follows the previous one:
What should be checked after a delay:
```

If the outcome, task, and answer criteria do not line up, revise the brief before generating more prose. If sources cannot support the mechanism, revise the scope before drafting.

## Stage-specific editorial directions

These are proposed instructions to test, not replacement prompts ready for deployment.

**Curator:** Explain what capability the selected evidence can teach. Name the supporting and limiting evidence, and state if the task cannot be supported. Prefer complementary content over superficially different URLs. Retain relevant source caveats.

**Blueprint:** Define success in terms of learner performance. Specify example inputs and reasoning steps, not just a description of an example. Mark uncertain prerequisites. Draft the changed-case task and rubric before the lesson.

**Researcher:** Connect each central claim to a verified passage and preserve its scope. Separate sourced observations, inference, and fictional teaching inputs. Identify missing support explicitly.

**Skeptic:** Find a specific unsupported inference, missing condition, misleading example, or ambiguity. If none is found, do not manufacture one. Return a disposition that the teacher/editor can act on.

**Teacher:** Make the reasoning visible. Explain why the tempting wrong answer fails. Define necessary terminology close to its first use. Remove generic openings and repeated summaries. Do not invent facts to make an example feel concrete.

**Examiner:** Test the promised capability and map each question to its actual concept. Use new inputs for transfer. Write answer criteria separately from feedback. Confirm that the learner has been taught enough to attempt the task.

**Editor:** Review the final lesson and answers together. Preserve limitations and distinguish synthetic data. Remove redundancy before compressing reasoning. Return unresolved problems rather than declaring quality through praise.

## Editorial rejection examples

Reject or repair a candidate when:

- The objective says “choose” but the lesson only defines terms.
- The worked example jumps directly from inputs to a conclusion.
- A question's answer requires information absent from the lesson and supplied task.
- A citation exists but its source does not support the claim's scope.
- A synthetic example is described as a measured result.
- The learner is asked to “try this” without inputs, completion criteria, or feedback.
- A review repeats identical wording while claiming to teach a new capability.
- The takeaway recommends an intervention that neither evidence nor task supports.

These are reviewer decision rules. Their frequency in current output is unknown until complete lessons are sampled.
