# Phase 3 source-intelligence evidence — 11 August 2026

## Status

The repository now has a versioned ranking model, auditable score components,
role-aware portfolio selection, a public-safe 50-topic evaluation seed, a
human-labeling protocol, and deterministic metric calculation. The corpus is
still marked `awaiting_human_labels`; therefore the 80% precision gate and
Phase 3 exit gate have not passed.

## Implemented ranking contract

- Ranking version: `source-rank-v2`.
- Planned and classified roles: official/primary, research,
  practitioner/explainer, reporting/context, and counterweight.
- Query planning seeks those evidence roles instead of repeating three keyword
  variations.
- Candidate intake is round-robin across successful role queries before the
  global candidate cap is applied. A high-volume early official/research query
  cannot starve practitioner, counterweight, or reporting candidates from
  preview, production discovery, or the human-evaluation snapshot.
- Search position contributes at most ten positive points and cannot by itself
  make a candidate eligible.
- Eligibility requires topical relevance, accessibility, and role-specific
  authority, primaryness, usefulness, or counterweight evidence.
- Components cover search position, relevance, authority, primaryness,
  recency, usefulness, portfolio independence, accessibility, counterweight
  value, and negative signals.
- Current negative signals cover zero-overlap title/snippet matches, thin
  snippets, stale non-primary material, promotional list-title patterns, and
  tag/category/author/archive surfaces.
- Resolved-page signals retain extracted authorship, cross-domain canonical
  changes, syndication markers, unattributed prescriptive advice, and stale
  versioned material. Highly similar text shingles remove copied or syndicated
  evidence, with quality signals taking precedence over learner preference.
- Portfolio selection seeks the four essential roles before filling remaining
  positions and caps a registrable domain at 25% of the configured eight-source
  portfolio.
- Learners see a role and plain-language selection reason, never a fabricated
  numeric “trust score.”

## Public retrieval policy

[ADR-0007](../adr/0007-public-source-retrieval-policy.md) records the product
owner's decision that a missing `robots.txt` file does not block retrieval of a
public HTTP(S) source. This does not weaken the existing network boundary or
authorize access-control bypass: authenticated pages, paywalls, CAPTCHA
challenges, credentialed URLs, non-public addresses, and unsafe redirects
remain out of scope. Retrieval is also not represented as ownership or an
unrestricted republication license. Attribution, bounded instructional
synthesis, public correction/reporting, and rights-holder review/removal remain
part of the launch contract.

No production feature flag or approval reference is required for this policy.
Counsel review of the provisional source/legal copy and a tested complaint
response remain external launch evidence under `LL-909` and the final launch
operations gate.

Migration 040 and `cmd/source-policy` add the operational response boundary:
an active operator can append an exact-URL or registrable-domain block and a
later audited unblock. The native fetch path checks current policy before an
initial request, before discovered/feed/article retrieval, and again for every
redirect. Policy-store failures fail closed. The current-policy view excludes
the private case reason, while the immutable event retains actor, case
reference, reason, and timestamp for authorized review.

## Durable audit trail

Migration 19 adds `source_role`, `ranking_version`, and checked JSON
`score_components` to discovered source specifications. Re-ranking an existing
canonical URL updates its role, reason, query, score, components, and ranking
version without changing evidence already frozen for an Issue.

Migration 20 adds reversible `neutral`, `preferred`, and `blocked` source
preferences. Owners can prefer, block, or replace a provided source and choose
provided-only, hybrid gap-filling, or autonomous discovery. Replacement retires
the old source specification and creates or reactivates a new one; it never
updates or deletes snapshots frozen into earlier Issues.

Migration 21 adds an optional, owner-controlled source approval boundary. In
`review` mode the worker resolves and freezes the source portfolio, transitions
the Issue to `awaiting_approval`, releases its claim without consuming a model
attempt, and does not call the lesson model or artifact store. The learner can
prefer, block, or replace sources; those changes invalidate only the pending
portfolio and preserve evidence frozen into earlier lessons. Approval requeues
the Issue, while switching to automatic mode is an explicit reversible action.

Before creation, autonomous and hybrid onboarding now requests a bounded,
read-only portfolio preview. It exposes role, domain, and a plain-language
selection reason, identifies missing evidence roles, labels candidates as
provisional, and never persists page bodies or presents a numeric trust score.

## Evaluation assets

- Corpus: `internal/source/testdata/source-evaluation-v1.json`
- Protocol: `docs/source-evaluation/README.md`
- Evaluator: `go run ./cmd/source-eval`
- Candidate capture: `go run ./cmd/source-eval -capture -output <new-path>`
- Seed validation:

  ```sh
  go run ./cmd/source-eval \
    -corpus internal/source/testdata/source-evaluation-v1.json \
    -validate-only
  ```

The evaluator requires at least 50 topics and calculates precision@5, domain
diversity, maximum observed domain share, required-role coverage from the
human-assigned role, unsafe/unusable rejection, selected sources
without a role, and topics with fewer than five ranked candidates. It rejects
out-of-range human scores, duplicate ranks, recommended sources that violate
the documented rubric, and insufficiently primary “official” recommendations.
The capture command runs the real query planner/ranker, freezes selected and
rejected public-safe candidate metadata, leaves every human judgment empty,
and refuses to overwrite a prior snapshot unless `-force` is explicit. The
`-require-gates` evaluator exits non-zero when labels are absent or any launch
threshold fails.

Human role labels are distinct from system-predicted roles, so the ranker
cannot grade its own role coverage. `human_adjudicated` now requires
CLI-generated hashes from two exact label-set files, immutable snapshot
equality, decision-level agreement measurement, and an explicit resolution
note for every recommendation/safety/usability/role disagreement. Domain-share
compliance is an explicit fail-closed metric (`maxDomainShare=0.25`) rather
than an inference from average diversity.
Captured corpora now carry a canonical candidate-snapshot SHA-256 spanning the
ranking/policy/topic/candidate fields. Labeled and adjudicated inputs must match
that hash, preventing silent post-capture edits across all copies. Every
candidate also requires `humanReviewed=true` and a bounded labeler note, so an
untouched all-zero score block cannot masquerade as an intentional review.

## Verification

- Seed validation: 50 representative topics structurally valid.
- Authority-versus-search-rank regression: passed.
- Essential-role and registrable-domain portfolio regression: passed.
- Fair-intake regression: a five-candidate cap retains the first candidate from
  each of the five planned source-role queries instead of consuming one query
  at a time.
- Human-label metric and rubric validation fixtures: passed.
- Migrations 19–21 and ranking/approval persistence against PostgreSQL 16:
  passed.
- Cross-Account source-catalog isolation remains covered by the PostgreSQL
  lifecycle.
- Full Go test suite and `go vet ./...`: passed.
- Frontend check/build and 16 files / 50 tests: passed.
- Baseline SQL executed successfully on schema version 21 under PostgreSQL
  16.14.
- `git diff --check`: passed.
- Source-control PostgreSQL lifecycle: prefer, block, replace, mode transition,
  cross-Account denial, and frozen-evidence preservation passed on schema 20.
- Resolved authorship, unattributed-advice, cross-domain canonical, and copied
  evidence regressions passed.
- The real PostgreSQL approval lifecycle passed: claim, pause before model
  generation, owner isolation, approval, and requeue. The worker regression
  separately proves the producer is never called before approval.
- The three-step onboarding journey was inspected in the local product at the
  desktop viewport and 390 px mobile width. Source roles, provisional status,
  optional approval, and the “Build my learning path” action remained visible
  and correctly ordered.

## Evidence still required

- Capture frozen real candidates for all 50 topics using one search snapshot
  and `source-rank-v2`.
- Obtain two blinded human label sets and independent adjudication through the
  hash-bound workflow; repository fixtures do not prove human independence.
- Retain evaluator output proving at least 0.80 precision@5, complete required
  role coverage, domain-share compliance, and complete unsafe/unusable
  rejection.
- Run ten observed design-partner source-selection tasks and win at least eight
  comparisons against the learner’s normal starting workflow.
