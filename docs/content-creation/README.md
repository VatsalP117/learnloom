# Making Learnloom's lessons more useful

Date: 2026-09-08. Status: proposal and repository audit; no application changes.

## Recommendation

Make the unit of content creation a **small capability the learner can demonstrate**. Use that capability to choose evidence, write an example, design practice, and decide what should come next. A coherent summary with the right headings is a useful foundation, but it does not establish that the learner can explain, decide, diagnose, or do something better afterward.

Start by reviewing complete lessons and the evidence they actually received. Use that baseline to improve source completeness and strengthen the existing blueprint and editorial process, then evaluate the result against learner tasks. Adding more generation stages or more prose should wait until an evaluation shows which failure they fix.

## Reading guide

- [Current flow and evidence-backed gaps](01-current-flow-and-gaps.md): what exists, where checks stop, and what remains unknown.
- [Proposed creation flow](02-proposed-creation-flow.md): intent, lesson selection, evidence, drafting, feedback, and failure behavior.
- [Editorial standards and examples](03-editorial-standards-and-examples.md): what a useful lesson looks like, with concrete before/after examples and reusable briefs.
- [Evaluation and prioritized roadmap](04-evaluation-and-roadmap.md): how to establish improvement, proposed sequencing, and acceptance criteria.

## What “better” means

A good Learnloom lesson should leave the learner able to perform a specific task with appropriate limits on their confidence. For example, “Given a small set of failed answers and retrieved passages, distinguish missing evidence from failure to use available evidence” is more useful than “Understand RAG evaluation.”

Usefulness includes conceptual learning: interpreting a graph, explaining a mechanism, or distinguishing two plausible explanations can be the task. Do not force every subject into a workplace checklist. Curious learners should still get a clear intellectual payoff.

Assess six separate properties:

| Property | Evidence to seek |
| --- | --- |
| Relevance | The learner recognizes a situation where the capability matters. |
| Understanding | They can explain why the mechanism works, not just repeat its name. |
| Transfer | They can apply it to a changed example without copying the solution. |
| Trustworthiness | Claims, uncertainty, and source support match. |
| Fit | Prerequisites, scope, and total effort fit this learner and session. |
| Durability | They can retrieve or apply it after a delay. |

These should remain separate measurements. A single quality score can conceal a severe weakness behind good formatting or many citations. Completion, return visits, and satisfaction matter, but do not independently demonstrate learning.

## First priorities

1. Establish a reviewed baseline of complete generated lessons and their source evidence.
2. Verify that hosted feed evidence contains enough detail, resolving linked articles safely when needed.
3. Require the blueprint to connect one observable outcome to its example and assessment.
4. Review claim support and the final example–answer alignment, including fallback outputs.
5. Budget the complete learning session, including practice.
6. Improve concept-specific practice and correction before making adaptation more aggressive.

The detailed roadmap proposes acceptance criteria rather than claiming these changes have already improved outcomes.

## Scope and evidence limits

This audit inspected local source code, tests, and existing documentation. It did not generate lessons with a configured provider, access production lessons or learner data, conduct learner interviews, or measure actual failure rates. Code establishes what the system enforces; it cannot establish how often the model produces weak lessons. Examples in this document set are synthetic editorial illustrations, not quotations from Learnloom output.

Existing evaluation documents and release evidence are useful context, but historical reports do not establish current end-to-end performance. Product recommendations here are proposals, not accepted ADRs, migration plans, or commitments to ship.

The design also draws on the [IES practice guide on organizing instruction and study](https://ies.ed.gov/ncee/wwc/PracticeGuide/1), which recommends spacing, worked examples combined with problem solving, abstract/concrete connections, retrieval, and explanatory questions. Its recommendations have differing evidence ratings and educational contexts. Applying them to short, personalized adult lessons is a hypothesis to validate, not a guaranteed effect size.

## Boundaries to preserve

Retain source safety, immutable evidence, account ownership, durable retries, separate delivery, and explicit separation of synthetic exploration. Start within the existing modular monolith. Nothing in these recommendations requires browser-based scraping, a new agent platform, new model providers, or a new database.
