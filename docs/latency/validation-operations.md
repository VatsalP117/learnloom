# Latency validation, observability, and rollout

Latency work needs clocks that agree on what is being measured. This repository
already has several useful signals and a staging runbook. This document records
their boundaries and proposes a validation sequence; it does not add or alter
instrumentation.

## Existing signals

### HTTP

The server has request counters and duration histograms with route-template
labels. The deferred observation is inside the handler boundary; it does not
include all browser/network time, and it is observed before the final
structured log write. Use it for origin handler latency, not end-user TTFB or
LCP. See [`server.go`](../../internal/httpapp/server.go#L199-L286) and
[`metrics.go`](../../internal/httpapp/metrics.go#L1-L120).

### Browser

The frontend reports CLS, INP, and LCP to `/api/performance/vitals`, with a
normalized page route and keepalive requests. These signals show user-facing
behavior but do not directly identify a particular API/database span. See
[`performance.ts`](../../web/src/performance.ts#L1-L31).

### Worker and model

Worker phase timers and model-stage duration observations are already present.
Use stage labels and issue IDs carefully: high-cardinality identifiers should
remain in traces/log correlation rather than metric labels. Pool acquisition
count/duration and operational queue gauges exist, but acquisition duration is
an aggregate, not a p95 query metric. See
[`worker.go`](../../internal/execution/worker.go#L1155-L1190) and
[`operational.go`](../../internal/store/operational.go#L48-L179).

### Staging harness

The staging runbook requires a declared forecast, dedicated test accounts,
explicit targets, and a report with aggregate p50/p95/p99 plus per-target p95.
It also requires worker/dependency observation and a 72-hour provider-evaluation
soak for `LL-210`. See [`staging-load-soak-runbook.md`](../staging-load-soak-runbook.md).

## Load-harness boundaries

`internal/loadverify/run.go` starts a request clock after the worker dequeues a
job. A full jobs channel can block the feeder, reducing the offered rate. The
report's achieved rate counts completions against the configured duration,
while `CompletedAt` includes drain time. The final pass decision gates
aggregate request count, aggregate error percentage, and aggregate p95. It
reports per-target p95 but does not gate it. See
[`run.go`](../../internal/loadverify/run.go#L75-L186).

Therefore the harness is useful for bounded origin HTTP verification, but it is
not a browser test, queue-wait test, source-discovery test, or model-stage
benchmark. It does not include generator queue delay in the request duration.
Report both offered and achieved rate, per-target distributions, queue age,
and worker/model/source timings in a latency review.

## Baseline evidence package

Use explicit timing boundaries to avoid counting overlapping work twice:

| Clock | Start → end | Interpretation |
| --- | --- | --- |
| Usable workspace | Navigation/auth transition → first actionable workspace | Includes code, auth, data, parse, and render; LCP alone is not this clock |
| Origin request | `ServeHTTP` entry → deferred observation | Includes handler work; excludes much of browser/network time |
| Queue eligibility wait | `available_at` → claim | Separates deliberate scheduling/retry delay from runnable backlog |
| Lesson readiness | User submission/issue creation → generated artifact committed | Includes admission, scheduling, generation, and retries; specify which submission timestamp is available |
| Generation attempt | Claim/attempt start → completion or failure | Report successful, failed, deferred, and checkpoint-reused attempts separately |
| Model stage | Stage entry → observer duration | Includes semaphore wait, retries and structured repair; excludes the observer's subsequent database callback |
| Email provider acceptance | Delivery becomes eligible → provider accepts | Separate receipt completion and inbox arrival; acceptance does not establish inbox delivery time |

Existing oldest-queue gauges use `available_at`, clamp negative ages to zero,
and represent the oldest queued item, not a per-job wait distribution or total
user wait. Do not infer p95 queue time from those gauges. Existing stage timers
also cannot distinguish semaphore wait from network/service time without added
spans. See [`operational.go`](../../internal/store/operational.go#L48-L179) and
[`stages.go`](../../internal/dossier/stages.go#L13-L92).

Before an optimization, record:

- release SHA and schema version;
- staging topology: web/worker replicas, pool limits, model semaphore,
  source/discovery limits, object store, and edge path;
- route mix and forecast rate/concurrency;
- browser traces for onboarding, workspace, library, newsletter detail, and
  issue detail;
- HTTP route-template p50/p95/p99 and error rate;
- SQL operation timings, rows, query plans, pool wait, locks, and statement
  timeouts;
- queue created/available/claimed/started/completed timestamps and oldest age;
- source cache hit, fetch, policy, discovery, enrichment, and snapshot timing;
- model queue/semaphore wait, provider time, retries, token use, contract
  repairs, and quality/evidence outcomes;
- artifact read/write and delivery provider timing, including ambiguous
  outcomes;
- operational scrape duration and database cost.

Do not combine these into one “latency” number. A route p95 can improve while
queue or model p95 worsens, and a 304 can reduce bytes while origin SQL remains
unchanged.

## Experiment designs

### Experiment A: workspace retention sharing

Use identical accounts and timestamps. Compare control and treatment for query
count, branch duration, pool wait, handler p50/p95, Today output, and retention
output. Include cold and active accounts. Success requires equal data and one
retention calculation per request; rollback if errors, persistence races, or
freshness change.

### Experiment B: worker scheduling

Seed isolated test accounts with a deterministic mix of short lessons, long
lessons, provider retries, source timeouts, and ready deliveries. Compare full
batch scheduling with a bounded refill or split loops. Gate on account fairness,
quota, claim uniqueness, claim recovery, graceful drain, queue oldest age,
delivery delay, model/provider concurrency, pool saturation, and errors.

### Experiment C: model stage changes

Replay the same frozen source set, learner history, and learner settings. Hold
the model fixed unless it is the explicit experimental variable. Record each
treatment's pipeline/prompt/model version and generation fingerprint separately;
do not reuse a control checkpoint for changed prompts or pipeline behavior.
Use fresh isolated attempts for cold-generation comparisons and a separate
matching-fingerprint retry cohort for checkpoint reuse. Compare per-stage p50/p95,
retries, repair frequency, token/cost,
quality scores, evidence coverage, artifact validity, and completion rate. Do
not compare only total duration because a faster stage can shift time or quality
to a later stage.

### Experiment D: discovery and snapshot concurrency

Replay the same ranked candidate list with bounded concurrency levels. Compare
time to sufficient evidence, selected sources and ordering, policy decisions,
duplicate rows, provider rate, DB pool wait, and failure/retry behavior. A
faster result that changes evidence or violates source limits fails the test.

### Experiment E: SQL and operational aggregates

Run representative plans at current and projected row counts, with concurrent
HTTP and worker activity. Compare execution/IO, rows, locks, pool wait, and
scrape freshness. Only introduce an index, rollup, projection, or partition
after measuring write/vacuum cost and consistency requirements.

## Percentile and gate policy

Keep aggregate and per-target results separate. The existing load verifier gates
aggregate p95, while the runbook reports per-target p95. For a route-mix change,
require each critical target to meet its own declared p95/error threshold in
addition to the aggregate gate; otherwise a fast health route can hide a slow
workspace route. For worker/model/source work, use stage-specific and queue-age
percentiles rather than applying the HTTP gate to unrelated clocks.

Do not claim a gain from a single run. Require repeated control/treatment runs,
the same data shape, warmed/cold labels, confidence intervals or run-to-run
spread, and no regression in quality, ownership, safety, idempotency, or cost.

Record sample counts and failures alongside every percentile. Do not average
percentiles across replicas or add stage p95 values to estimate end-to-end p95.
Use request/job-level elapsed times for end-to-end results, and aggregate
histogram buckets before computing service quantiles. The current histogram
buckets limit percentile precision; inspect bucket resolution before interpreting
a small apparent improvement. Preserve environment/service labels when comparing
deployments, and keep success and failure distributions separate.

## Staged roadmap

### Stage 0: instrument and baseline

Use existing metrics and add only trace/span correlation needed to join browser
request IDs, origin route spans, SQL operations, worker issue IDs, source runs,
model attempts, artifact operations, and delivery claims. Confirm the clocks and
sampling policy before evaluating proposals.

### Stage 1: low-risk request work

Test request-scoped retention sharing, query/payload measurements, and safe
cache stale-response fencing. Verify account/session isolation and current
freshness semantics. Consider conditional transfer only after measuring origin
work separately.

### Stage 2: worker scheduling

Run the bounded scheduling experiment behind an explicit rollout control. Check
fairness, claim renewals, quotas, drain, and delivery ambiguity before increasing
concurrency or separating loops.

### Stage 3: source/database work

Apply only the measured candidate: bounded discovery resolution, safe snapshot
batching, scoped progress, query/index/rollup change, or pool tuning. Re-run
source equivalence and database plan tests together with worker load.

### Stage 4: model and edge work

Evaluate prompt/context, structured repair, streaming, provider retry, artifact
transport, compression, and edge-cache changes one at a time. Keep model quality,
evidence provenance, privacy, and final artifact correctness as release gates.

### Stage 5: soak and rollback review

Repeat the staging read gate and the required worker/provider soak. Attach the
release SHA, schema, topology, reports, dashboards, queue drain, alert outcomes,
and incident references to the release record. Roll back if any latency gain
comes with a material error, quality, privacy, quota, duplicate-delivery, or
database-saturation regression.
