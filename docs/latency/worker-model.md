# Worker and dossier-generation latency

This is a design audit for asynchronous generation and delivery. It documents
where queue delay, database work, source work, model time, and persistence time
can accumulate. It proposes experiments; it does not change worker behavior.

## Worker scheduling facts

`Worker.Run` creates one ticker and invokes `Cycle`; after each cycle it waits
for the next ticker event. A long cycle does not add a full poll interval to its
own duration, because ticker events can already be pending. This distinction
matters when interpreting queue-age traces. See
[`worker.go`](../../internal/execution/worker.go#L206-L225).

`Cycle` runs lifecycle recovery and issue dispatch, then processes issues to
completion, then deliveries, then weekly recaps, public-follow deliveries,
deletion, artifact cleanup, and hourly operational cleanup. A returned error
stops the remainder of that cycle. See
[`worker.go`](../../internal/execution/worker.go#L231-L307).

`processIssues` claims up to `GlobalConcurrency` issues synchronously, starts a
goroutine for each claim, and waits for the entire batch. It does not refill a
finished slot while another issue in that batch is still running. Deliveries
cannot start until the wait completes. This is a code fact; the impact depends
on the duration distribution and workload mix. See
[`worker.go`](../../internal/execution/worker.go#L635-L676).

### P0: test bounded refill or separate queue loops

**Hypothesis:** one slow generation can leave capacity idle and delay delivery
and other queues behind it. This will be most visible with mixed fast/slow
issues, a full model semaphore, or a provider retry. The existing aggregate
worker duration metric cannot prove the straggler effect by itself.

**Proposal for architecture review:** choose one bounded design:

- refill an issue slot as each issue completes while retaining one scheduler's
  account/global claim limits; or
- run independently bounded loops for issue generation, ordinary delivery,
  recaps/follows, and cleanup, with an explicit database/pool budget.

Both designs need backpressure. Increasing goroutines without a database and
provider concurrency budget can increase tail latency. Keep claim renewal and
shutdown drain semantics explicit.

A refill loop that never returns while issues remain queued would starve the
later delivery and cleanup phases. If retaining `Cycle`, impose a finite claim
count or time budget before yielding to those phases; alternatively schedule
them independently. Test a continuously nonempty issue queue, not just a finite
batch. Keep database-enforced account fairness and budgets: a local slot limit
alone does not enforce limits across worker replicas.

**Experiment:** replay a staging workload with a deterministic mix of short,
long, retrying, source-blocked, and delivery-ready jobs. Compare queue age,
time-to-claim, active slots, delivery delay, account fairness, claim recovery,
database pool wait, and model semaphore wait. Use the same concurrency and
limits for control and treatment. The treatment must not claim above account or
global daily quotas.

## Issue-generation critical path

After a claim, `processIssue` starts a bounded issue context and claim-renewal
goroutine. It then loads learning history, learner state, and prepared source
items. If evidence-led rhythm has no novel evidence or review-before-lesson is
awaiting approval, generation can terminate before model work. Otherwise it
computes a generation fingerprint, loads matching checkpoints, invokes the
producer, stores the artifact, and completes the issue. See
[`worker.go`](../../internal/execution/worker.go#L679-L866).

The order is intentional: the generator needs learner context and source
evidence; completion needs a durable artifact. `CompleteIssue` performs a
transaction that ties issue status, usage, history, search data, concepts,
reviews, events, and delivery enqueue together. A latency change must not split
that transaction merely to make one statement appear faster. See
[`issues.go`](../../internal/store/issues.go#L799-L930).

## Model pipeline

`Generate` assembles a candidate bundle, then runs curator, blueprint,
researcher, skeptic, teacher, examiner, and editor stages. The explorer may run
concurrently with examiner; dependent stages remain ordered. It renders the
final artifact after quality evaluation and can preserve teacher/examiner
content if editor output fails. See
[`generator.go`](../../internal/dossier/generator.go#L182-L592).

### P1: optimize stages from traces

**Observed:** `runStage` records stage duration and calls `OnStage`;
`runStructured` can perform a second model call to repair a contract failure.
The model adapter limits concurrent calls, retries transient/provider errors,
and uses exponential backoff. It does not stream a partial result. See
[`stages.go`](../../internal/dossier/stages.go#L13-L92) and
[`model.go`](../../internal/dossier/model.go#L68-L242).

**Hypotheses to test separately:**

- one stage or provider retry dominates p95;
- prompts carry more source/history text than needed for the chosen output;
- structured repair is frequent enough to matter;
- waiting for complete model responses harms perceived latency even when total
  generation time is acceptable;
- model semaphore wait, rather than provider service time, is the bottleneck.

**Proposal for review:** instrument each stage with attempt number, queue wait,
provider time, response bytes, input/output tokens, retry cause, and contract
repair. Then consider smaller context windows, bounded output budgets, prompt
reuse, streaming where the UI can safely consume it, or reducing repair rate
through contract-safe prompt/schema changes. Each is a separate experiment.

Do not parallelize curator/blueprint/researcher/skeptic/teacher/editor without
showing that their inputs are independent. Do not remove skeptic, examiner,
quality evaluation, provenance, or contract repair solely to lower wall time.
The success vector is p95 plus quality, evidence coverage, token spend, retry
rate, and completion/error rate.

### Retry and timeout effects

The model client defaults to a ten-minute HTTP client timeout, a concurrency semaphore,
and configured retries with backoff. The worker issue timeout defaults to 45
minutes, so a provider stall or repeated retry can occupy a claim for a large
part of a batch. See [`model.go`](../../internal/dossier/model.go#L68-L165)
and [`worker.go`](../../internal/execution/worker.go#L679-L704).

Measure retries by provider/status/stage and distinguish provider service time
from backoff. A shorter timeout may reduce tail latency while increasing failed
lessons and replay cost; a longer timeout may reduce false retries but tie up
worker slots. Any policy change needs an explicit error-category and budget
experiment.

`Complete` acquires its semaphore before the retry loop and releases it only
when the whole call returns, including backoff sleeps. Its HTTP timeout covers
each HTTP request; it does not bound time waiting for a semaphore slot or all
retries combined. The issue context is the outer deadline. Releasing a slot
during backoff is a proposal only: it needs provider cooldown and retry
admission controls so additional traffic does not amplify a rate-limit incident.
See [`model.go`](../../internal/dossier/model.go#L104-L164).

Configuration defaults are not deployed settings. The repository defaults are
four generation slots per worker, four model calls per model instance, one
concurrent issue per account, two provider retries, and a two-second worker
ticker. See [`config.go`](../../internal/config/config.go#L181-L239). Source
parallelism can add more outbound requests inside each issue; replicas multiply
local limits. Budget database connections and provider demand across the entire
deployment before increasing any one limit.

## Checkpoints and stage recording

The worker supplies callbacks that synchronously save each checkpoint and record
each stage. Both operations use a separate background context with a three
second timeout. The callback runs as part of generation, so a slow pool or
database write can delay the next stage or the generator return. See
[`worker.go`](../../internal/execution/worker.go#L748-L832).

### P1: measure before asynchronous journaling

**Hypothesis:** checkpoint/stage writes add a meaningful fraction of model
generation wall time or contend with issue reads. This is not established by
the existence of the writes; model calls may dominate.

**Experiment:** add tracing in a staging-only branch around callback entry,
pool acquisition, SQL execution, and return. Report callback time as its own
histogram and correlate it with model stage p95, pool wait, and checkpoint
failure. Include process restarts and claim expiry in the test.

**Proposal if evidence supports it:** buffer only append-like stage telemetry,
or persist a durable bounded journal and flush it after a stage. Checkpoints
require more care: they are restart inputs keyed by issue and generation
fingerprint. A buffered design must define ordering, duplicate handling,
claim-token validation, flush-before-success behavior, and recovery after a
crash. Checkpoint loss may increase regeneration time, while a stale checkpoint
must never be applied to a different fingerprint.

`RecordIssueStage` also carries token/cost usage and claim-token validation;
it is not disposable logging. Any batching must preserve durable accounting,
ordering, and exactly the existing ownership/claim semantics. Do not replace
these callbacks with fire-and-forget goroutines. Measure checkpoint-hit and
checkpoint-miss runs separately; replay savings and fresh-generation speed are
different outcomes.

## Artifact and delivery path

Artifact storage already has a bounded in-process LRU (64 MiB by default) and
checks the cache before the object-store read; writes and successful cold reads
populate it. See [`artifact/store.go`](../../internal/artifact/store.go#L45-L90)
and [`artifact/store.go`](../../internal/artifact/store.go#L181-L225). Do not
recommend adding a basic artifact cache as if none exists. Measure hit rate,
evictions, cold-read latency, and concurrent misses first; a future singleflight
for the same key would need bounded memory and cancellation semantics.

The worker registers cleanup, writes the artifact, then completes the issue.
If completion fails, it deletes the artifact where possible; the cleanup queue
is the durable fallback. This artifact-first ordering protects readers from a
completed issue with no dossier. See
[`worker.go`](../../internal/execution/worker.go#L835-L920).

Delivery claims are processed concurrently after issue processing. The path
gets the artifact, renews the claim, renders HTML, calls the mail provider, and
completes the delivery. Provider acceptance followed by completion failure is
represented as `outcome_unknown`; retries must follow reconciliation policy.
See [`worker.go`](../../internal/execution/worker.go#L1020-L1153) and
[`deliveries.go`](../../internal/store/deliveries.go#L1-L220).

### P2: separate artifact and delivery measurements

Measure object-store connect, first-byte, read, render, provider request,
provider acceptance, completion transaction, and cleanup time separately.
`delivery_total` alone cannot identify which segment is slow. A faster object
store read or HTML render is useful only if it does not change provider
idempotency or cause a second send after an ambiguous response.

## Worker sequence for a staged rollout

1. Establish queue-age and stage-level baselines with issue and delivery clocks.
2. Test scheduling treatment under mixed duration and failure workloads.
3. Test model-stage changes one at a time and compare quality/cost.
4. Test callback buffering only with restart, claim-loss, and checkpoint
   fingerprint cases.
5. Test artifact/provider paths independently; retain the existing
   `outcome_unknown` behavior.
