# Phase 2 reliability evidence — 11 August 2026

## Scope

This record covers the deterministic failure-state and recovery work completed
before production-wide failure measurements are available. It proves repository
behavior; it does not claim that the current production failure distribution is
known.

## Recovery matrix

| Failure | Stable code | Category | Automatic behavior | Terminal learner state |
|---|---|---|---|---|
| Discovery service returns no results because all queries fail | `source_discovery_unavailable` | infrastructure | Durable Issue retry with exponential backoff | Safe delayed message after exhaustion |
| Discovery works but no worthwhile independent evidence exists | `no_worthwhile_evidence` | insufficient evidence | Do not burn generation retries; defer until the next scheduled evidence check | Waiting for stronger evidence |
| Learner-provided sources cannot supply usable evidence | `source_evidence_needs_attention` | user actionable | Stop automatic retries | Replace or update sources, then retry |
| Model timeout, network interruption, HTTP 408/429/5xx, or invalid provider response | `model_provider_unavailable` | provider | Provider retry/backoff, then durable Issue retry/backoff | Safe delayed message after exhaustion |
| Model HTTP request is permanently rejected | `model_request_rejected` | provider | Stop automatic retries | Safe failure message; internal incident retained |
| Model reaches its output token limit | `model_output_truncated` | content quality | Durable Issue retry from validated checkpoints | Safe failure message after exhaustion |
| Model returns an empty completion | `model_output_empty` | content quality | Durable Issue retry from validated checkpoints | Safe failure message after exhaustion |
| Structured stage fails validation and its repair attempt | `model_contract_unsatisfied` | content quality | Durable Issue retry from validated checkpoints | Safe failure message after exhaustion |
| Worker generation is interrupted | `generation_interrupted` | infrastructure | Release during a controlled drain; otherwise retry durably | Safe delayed message after exhaustion |
| Candidate objective, concepts, and evidence repeat recent learning | `no_new_learning_signal` | insufficient evidence | Stop before downstream generation and defer | Waiting for a genuinely new learning signal |
| Evidence-led cadence finds no changed source evidence | `no_new_evidence` | insufficient evidence | Do not spend model budget; defer | Waiting for stronger/new evidence |
| A worker claim expires | `worker_claim_expired` | infrastructure | Recover the claim through PostgreSQL and retry within the bounded attempt policy | Safe delayed/failure projection according to attempts |

All public projections are fixed safe messages. Provider response bodies,
credentials, private source diagnostics, and internal validation errors remain
in operational detail only.

## Proved behavior

- Autonomous and hybrid discovery outages no longer masquerade as weak evidence.
- A weak-news day is a durable `deferred` Issue and `deferred` attempt, not a
  failed generation or worker-cycle failure.
- Model HTTP retryability survives the provider boundary, so permanent 4xx
  request rejection cannot consume the full Issue retry budget.
- Successful curator, blueprint, researcher, skeptic, and teacher stages are
  checkpointed and restored only when the current fingerprint and validation
  contract still match.
- Structured stages attempt one targeted contract repair before returning a
  stable, stage-specific quality failure.
- Repeated manual requests return the already queued or generating Issue instead
  of creating duplicate work.
- A candidate whose objective, concepts, and source portfolio all substantially
  overlap a recent lesson becomes an evidence-led deferral before teaching,
  practice, editing, artifact storage, or publication.
- Learning history now retains the Blueprint's actual concepts in addition to
  its central mechanism and prerequisites, strengthening continuity and novelty
  checks.
- First-lesson setup, confirmation, and waiting states now publish one bounded
  expectation: plan for 5–15 minutes, with an explicit caveat for source
  availability and automatic recovery. This is the launch SLO to validate, not
  a claim derived from missing production latency data.
- Stream overview remains anchored to the latest usable lesson when a newer
  attempt fails or defers, eliminating the contradictory “first lesson” state
  on established streams.
- Failed content/provider/infrastructure work offers retry when safe;
  learner-provided source failure offers an executable switch to hybrid
  discovery plus retry; permanent failures provide a support link carrying
  only the incident ID.
- Aggregate monitoring now covers the 30-minute generation failure ratio,
  Accounts with two consecutive terminal failures, queue age, provider output
  truncation, and model-budget depletion. Account identifiers are not emitted.
- Failure code, category, stage, retryability, incident ID, provider retries,
  stage duration, token usage, and estimated model cost are durable.
- The control API exposes only safe failure fields, including the evidence
  deferral state.

## Regression fixtures

- Weak discovered evidence becomes `no_worthwhile_evidence`.
- Discovery query outage becomes retryable `source_discovery_unavailable`.
- Hybrid discovery outage remains retryable when provided evidence is
  insufficient.
- Evidence deferral bypasses `FailIssue` and does not fail the worker cycle.
- HTTP 400 becomes non-retryable `model_request_rejected`.
- HTTP 503 becomes retryable `model_provider_unavailable` with a delayed public
  message.
- Empty model output becomes retryable `model_output_empty` content quality.
- Invalid structured output receives one repair attempt and retains
  `model_contract_unsatisfied`, `content_quality`, and its exact stage.
- Migration 18 is applied and the real PostgreSQL lifecycle covers deferred
  Issue persistence, safe projection, attempt state, and retry conflict.
- The PostgreSQL 16 lifecycle also proves that a repeated manual request while
  an Issue is queued returns the existing Issue instead of inserting another.
- A deterministic novelty fixture rejects three-way objective/concept/source
  overlap while allowing a genuinely changed concept direction.
- Frontend fixtures cover queued, deferred, established-stream, retry,
  broaden-source, and contact-support presentations.
- The alert-rule regression fixture protects every launch-critical alert name
  and metric dependency.
- A canonical stable-code registry now covers every classified code emitted by
  model, source, novelty, rhythm, and claim-recovery paths. Its regression
  manifest names a deterministic behavioral fixture for every code, so adding
  a new stable class without adding coverage fails the suite.
- Additional fixtures prove deadline interruption, provided-source correction,
  novelty deferral before downstream model stages, evidence-led deferral,
  output truncation retry, and expired-claim recovery.

## Remaining operational evidence

- Export the trailing 30-day production failure taxonomy using
  `scripts/product-baseline.sql`.
- Reproduce any additional active production-only class with a safe fixture.
- Confirm the export contains no unexplained `internal_error`,
  `legacy_internal_error`, `unknown`, or unregistered stable code before
  completing `LL-201`.
- Run the provider evaluation corpus and 72-hour staging soak.
- Prove the reliability exit gates for the design-partner cohort before broad
  launch.
- Provision the updated rules in staging, trigger the truncation and
  consecutive-failure fixtures, and retain notification delivery and recovery
  screenshots before completing `LL-209`.

## Verification

- `go test ./cmd/... ./internal/...`: passed
- `go vet ./...`: passed
- `npm run check`: passed
- `npm test`: 16 files and 47 tests passed
- `TestPostgresLifecycleIntegration` against disposable PostgreSQL 16: passed
- Consecutive-Account failure aggregation against disposable PostgreSQL 16:
  passed
- Prometheus validation of the inner Grafana/Mimir rule groups: 27 rules found,
  syntax valid
- `git diff --check`: passed
