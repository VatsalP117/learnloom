# Phase 1 research readiness — 12 August 2026

## Repository mechanism

The repository now contains a precommitted `launch-icp-interviews-v1` protocol,
fixed behavioral questions, bounded coding definitions, privacy boundaries, a
blank corpus template, a persona decision/rejection template, and
`cmd/interview-eval`.

The evaluator requires a frozen batch of at least 15 records, rejects duplicate
participant references, post-freeze interviews, unknown codes, invalid scores,
and obvious identity/URL material in references, and excludes people without
both role fit and a concrete recent attempt from the qualified denominator.

The fail-closed gate implements the roadmap's exact thresholds: 15 qualified
interviews, 10 weekly-pain signals, five paid or two-hour weekly workarounds,
and 10 explicit design-partner interests. Persona-level aggregates are emitted
without names, quotes, employers, topics, or contact details.

## Verification

- Exact-threshold passing fixture: passed.
- Below-threshold and undersized fixtures: rejected.
- Duplicate, unfrozen/post-freeze, unknown-code, and identifying-reference
  fixtures: rejected.
- Unqualified participants cannot inflate the exit gate: passed.

## Human evidence still required

No interview, consent, persona selection, rejected-persona decision, or
design-partner recruitment is claimed. `LL-101`–`LL-106` and the Phase 1 exit
gate remain open until the real frozen corpus, restricted qualitative evidence,
decision record, and participant consent/recruitment records exist.
