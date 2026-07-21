# Deferred AI Flow Improvements

These ideas are intentionally deferred for a later product iteration. They are
ordered by expected impact on learner satisfaction.

## 1. Independent Post-Editor Review

Add a structured `quality_reviewer` stage after editorial generation. It should
assess:

- alignment with the learner's goal and level;
- novelty relative to recent Learning History;
- evidence support and appropriate uncertainty;
- clarity of the central mechanism and worked example;
- consistency between the final lesson, retrieval questions, and answers.

If the reviewer reports a blocking issue, run one targeted revision using the
review findings. Do not regenerate the whole Dossier, and retain the existing
deterministic quality gate as the final hard contract.

## 2. Cumulative Spaced Retrieval

Include one or two retrieval questions from older Dossiers alongside questions
about the current lesson. Select them using concept relevance and time since
last exposure so practice builds durable, cumulative knowledge rather than
testing only today's material.

## 3. Citation-to-Claim Verification

The current quality gate verifies that citation identifiers exist, but not that
the cited Source Item supports the adjacent claim. Add a structured verifier
that flags unsupported, overstated, or contradictory claims and supplies exact
repair instructions to the final editor.

## 4. Generate AI Exploration From the Final Lesson

Generate optional AI Exploration only after the final source-grounded lesson is
accepted. This prevents synthetic analogies or deductions from drifting away
from editorial changes. Exploration can run alongside final validation because
it remains outside the grounded lesson contract.

## 5. Blueprint-Guided Source Excerpts

Replace character-prefix truncation with excerpt selection guided by the
Learning Blueprint. Preserve passages relevant to the central mechanism,
worked example, misconception, evidence limits, and practical experiment before
spending context on lower-value text.

## 6. Stage-Specific Model Budgets

Tune model behavior by stage:

- small deterministic budgets for curation and structured contracts;
- larger budgets for research and teaching;
- strict structured output for review and repair;
- explicit per-stage token and latency ceilings.

Maintain compatibility with the existing OpenAI-compatible provider boundary.

## 7. Partial Retry and Checkpointing

Persist safe intermediate results so a late editor or delivery-adjacent failure
does not repeat curation, research, and teaching calls. Resume from the latest
validated stage while preserving the Issue's frozen source evidence.

## Recommended Next Slice

Implement independent post-editor review with at most one targeted revision,
then add cumulative spaced retrieval. Together they offer the clearest direct
improvement to semantic quality and long-term learning without requiring UI
changes.
