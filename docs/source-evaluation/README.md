# Learnloom source-evaluation protocol

## Status

`source-eval-v1` defines 50 representative topics for the initial
AI/software-professional launch hypothesis. The topic seed is public-safe and
versioned. Candidate capture and human labeling are deliberately incomplete;
the product must not claim that the source-intelligence gate has passed until
the completed corpus and evaluator output are retained as release evidence.

Validate the seed:

```sh
go run ./cmd/source-eval \
  -corpus internal/source/testdata/source-evaluation-v1.json \
  -validate-only
```

Freeze a candidate snapshot with the same SearXNG provider and ranking version
for all 50 topics. The output path must be new unless replacement is explicitly
authorized with `-force`:

```sh
SEARXNG_BASE_URL='https://search.internal.example' \
go run ./cmd/source-eval \
  -corpus internal/source/testdata/source-evaluation-v1.json \
  -capture \
  -output docs/source-evaluation/source-eval-v1-captured.json
```

Capture stores only public search metadata: title, canonicalized HTTPS URL,
short search snippet, registrable domain, publication date when supplied,
system rank, and system role. It does not fetch or store article bodies,
cookies, credentials, learner data, or private provider diagnostics. Human
scores, `humanRole`, recommendations, safety flags, and notes remain empty.
Make two byte-identical copies of the immutable capture for independent blinded
labeling; create a new version instead of recapturing selected topics after
labels are visible.

Each labeler sets `labelStatus` to `human_labeled`, completes every bounded
score, sets `humanReviewed=true`, assigns `humanRole` independently of
`systemRole`, and records a short evidence-based note for every candidate. They
must not see or edit the other label set. A single
label set may be evaluated for draft feedback, but it cannot pass release
gates.

After both label sets are frozen, create a third `human_labeled` resolved copy
from the same candidate snapshot. For every disagreement in `recommended`,
`unsafe`, `unusable`, or `humanRole`, choose the adjudicated value and add a
bounded `adjudicationNote`. Then run:

```sh
go run ./cmd/source-eval \
  -corpus internal/source/testdata/source-evaluation-v1.json \
  -adjudicate \
  -label-set-a /restricted/label-set-a.json \
  -label-set-b /restricted/label-set-b.json \
  -resolution /restricted/resolution.json \
  -adjudicator-ref review-panel-01 \
  -output /restricted/source-eval-v1-adjudicated.json
```

The command verifies that all three files share the same immutable capture,
computes SHA-256 provenance from the exact input files, measures decision-level
agreement, and refuses unresolved disagreements. Keep reviewer identity and
assignment proof in the restricted research system.

Candidate capture also writes `candidateSnapshotHash`, a canonical SHA-256 over
the corpus version, ranking version, domain-share policy, topic definitions,
required roles, and captured candidate metadata. Every human-labeled and
adjudicated input recomputes this value during validation. Editing a URL, rank,
system role, title, snippet, date, topic, outcome, or policy after capture makes
the corpus invalid even if all label files were changed together.

After adjudication, omit `-validate-only` to calculate precision@5, domain
diversity, maximum observed domain share, required-role coverage based on
`humanRole`, unsafe/unusable rejection, missing roles, and topics with fewer
than five ranked selections.

After labeling and adjudication, make launch signoff fail closed unless every
required metric passes:

```sh
go run ./cmd/source-eval \
  -corpus /restricted/source-eval-v1-adjudicated.json \
  -require-gates
```

## Capture protocol

For every topic, use a clean discovery index snapshot and the same ranking
version. Retain at least the top ten selected or rejected candidates so unsafe
and unusable rejection can be measured rather than inferred. Store only public
web metadata: canonical URL, registrable domain, system rank, role, title,
snippet, and publication date. Never include learner data, credentials,
paywalled article text, search cookies, or copied source bodies.

Freeze the raw candidate set before labeling. The human labeler must not see
the system score, score components, or selection reason until their labels are
submitted. They may see system rank afterward for adjudication.

## Human labeling rubric

Score each dimension from 0 to 3, assign `humanRole`, and add a short
evidence-based note. Do not copy `systemRole` without evaluating it.

| Dimension | 0 | 1 | 2 | 3 |
|---|---|---|---|---|
| Topical relevance | unrelated or title-only match | adjacent | directly supports part of the outcome | directly supports the central outcome |
| Source authority | unidentified or unsupported | identifiable but weak basis | credible for its source class | canonical or strongly accountable for its class |
| Primaryness | copied/aggregated | commentary only | near-primary synthesis | specification, original research/data, or accountable first-party material |
| Recency | stale for this question | date unclear or aging | current enough | current and explicitly maintained; timeless primary work may also score 3 |
| Explanatory usefulness | thin or promotional | fragments only | useful explanation | mechanism, examples, boundaries, and usable detail |
| Independence | duplicates another candidate | same organization or copied chain | meaningfully independent | distinct evidence, methods, incentives, and organization |
| Accessibility | blocked, deceptive, or unreadable | severe friction or incomplete | readable with minor friction | stable, readable, and directly accessible |
| Counterweight value | reinforces consensus only | mentions caveats | substantive limitations | strong contrary evidence, risks, failure modes, or boundary conditions |

`recommended=true` requires topical relevance of at least 2, authority of at
least 2 for the assigned role, accessibility of at least 2, no unsafe flag,
and no unusable flag. Primaryness and counterweight are role-dependent; a
practitioner explainer need not be primary, while an official/primary source
must score at least 2 for primaryness.

Set `unsafe=true` for malicious redirects, credential harvesting, malware,
deceptive downloads, or source-safety violations. Set `unusable=true` for
inaccessible, empty, copied, machine-spun, materially misleading, or
irreparably stale evidence. A disagreeable or critical source is not unsafe.

## Roles

- `official_primary`: maintained documentation, standards, original data, or
  an accountable organization speaking for its own system.
- `research`: peer-reviewed work, reputable preprints, proceedings, systematic
  reviews, or research repositories.
- `practitioner_explainer`: implementation guidance, case studies, worked
  examples, or technically accountable field experience.
- `reporting_context`: independent reporting, interviews, market or historical
  context; useful but not a substitute for primary evidence.
- `counterweight`: limitations, risks, negative findings, failure modes,
  alternative explanations, or credible criticism.

## Release gates

- at least 50 fully labeled topics;
- precision@5 of at least 0.80;
- every recommended top-five portfolio covers official/primary, research,
  practitioner explanation, and counterweight unless the labeler documents why
  a role genuinely does not exist;
- no selected source lacks a role;
- all known unsafe/unusable candidates are rejected;
- `maxDomainShare` is 0.25 and no ranked portfolio exceeds it;
- two humans adjudicate disagreements on `recommended`, `unsafe`, `unusable`,
  and source role.

Passing synthetic regression fixtures is necessary for implementation safety
but does not satisfy the human-labeled release gate.
