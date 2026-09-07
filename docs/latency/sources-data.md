# Source acquisition and database latency

This document covers two external dependencies in the generation path:
source intelligence and Postgres. It separates optimizations already present
from proposals that need query plans or staging traces.

## Source acquisition already has bounded safety

The acquisition layer uses feed and article timeouts, bounded response sizes,
maximum characters, redirect limits, and a configured concurrency. Feed fetches
and article enrichment use ordered bounded parallelism. Conditional ETag and
Last-Modified requests, keep-alive, and public-address validation are already
implemented. See [`acquisition.go`](../../internal/source/acquisition.go#L17-L155)
and [`http.go`](../../internal/source/http.go#L29-L128).

Those controls are part of the latency design: an unsafe or unbounded “faster”
fetcher is not an acceptable optimization. Enrichment waits for all bounded
workers, so one slow article can determine the source-preparation tail. A feed
summary fallback exists when an article fails, which is a quality behavior that
must be included in any experiment.

## P1: discovery candidate resolution

Discovery search queries run concurrently up to the configured query count.
After ranking, `discover` resolves candidates and snapshots them in a
sequential loop, with extra endpoint reads/updates along the way, until the
active-source limit is satisfied. See
[`discovery.go`](../../internal/source/discovery.go#L51-L188).

**Hypothesis:** when discovery is needed, sequential candidate resolution is a
tail-latency hotspot. It is not safe to infer this from the loop alone because
the first ranked candidate may usually succeed and cached endpoints may be
fast.

**Experiment:** capture candidate count, candidate resolution duration,
endpoint-cache hit, policy-rejection, provider status, snapshot insert count,
and time to sufficient evidence. Replay the same ranked candidates with
bounded resolution concurrency 1, 2, and 4. Compare evidence set, order,
duplicates, provider request rate, and account-level source budgets.

**Proposal for review:** resolve a bounded candidate window concurrently, then
apply deterministic ranking/selection after results return. Keep per-account
discovery limits, URL policy checks, DNS rebinding protections, and
`DiscoveryMaxActive`. If writes are parallelized, use idempotent keys and do
not let a late low-ranked candidate change the selected evidence nondeterministically.

## Source snapshot writes and policy checks

`PrepareIssue` resolves configured source specs in bounded parallelism, selects
evidence, and freezes issue sources. A cache path still checks snapshot URLs
through the URL policy. Fresh endpoint fetches can autodiscover a feed,
upsert endpoint state, and snapshot items. `snapshotFeed` checks each item and
inserts each usable snapshot individually. See
[`service.go`](../../internal/source/service.go#L174-L283),
[`service.go`](../../internal/source/service.go#L432-L627), and
[`service.go`](../../internal/source/service.go#L630-L798).

**Hypotheses:**

- repeated per-snapshot/per-item policy calls add database round trips;
- one insert per feed item increases pool occupancy for large feeds;
- waiting for all enrichment items makes one slow origin dominate the issue;
- cached endpoint state reduces provider time but not all database/policy time.

Measure each separately. A safe future batch operation could validate a set of
URLs in one database interaction or insert snapshots in a transaction, but it
must retain per-item rejection/audit semantics, ownership, unique/idempotent
behavior, and the rule that only policy-approved URLs are persisted. A batch
must not turn one bad item into loss of all good evidence unless that is an
explicitly reviewed behavior.

For enrichment, compare bounded concurrency and an evidence sufficiency cutoff
only if the learner-facing contract permits returning after enough reliable
sources are ready. Preserve deterministic source ordering and the fallback
summary path. Never silently lower evidence quality to improve p95.

## Postgres pool and request behavior

`store.Open` creates a pgx pool, applies a statement timeout, and pings the
database. The operational snapshot exports `AcquireCount` and aggregate
`AcquireDuration`. These are useful saturation signals but do not provide
query-level p95 or identify the statement holding a connection. See
[`store.go`](../../internal/store/store.go#L1-L100) and
[`operational.go`](../../internal/store/operational.go#L48-L179).

### P1: query-plan and pool attribution

Before changing pool size or adding indexes, capture:

- pool acquired/idle/max connections and wait duration;
- query duration and rows by operation, with account identifiers redacted;
- `EXPLAIN (ANALYZE, BUFFERS)` for representative cardinalities;
- lock waits, statement timeout count, and database CPU/IO;
- concurrent worker model/source activity during the read workload.

Pool enlargement can reduce wait while making Postgres slower through CPU,
memory, or lock contention. Pool reduction can protect the database while
increasing queue delay. Treat the pool as a shared budget across HTTP,
workers, source writes, checkpoint callbacks, and operational scrapes.

## Authenticated read query candidates

### Newsletter list/detail

`newsletterSelect` joins issues and delivery receipts, counts distinct values,
aggregates source specs, and runs correlated concept-state counts. The query is
appropriate for a rich sidebar but its work grows with account streams and
issue history. See [`newsletters.go`](../../internal/store/newsletters.go#L722-L760).

Measure plans for zero, one, and many newsletters; large generated histories;
and concurrent workspace/detail loads. Candidate designs include a summary
projection maintained on writes, a bounded summary endpoint, or separate
sidebar/detail queries. These are data-model decisions: preserve account
ownership and consistency expectations before adopting a projection.

Newsletter detail launches many reads in parallel, but its account-wide
`ListLessonProgress` can return more rows than the visible newsletter needs.
See [`control.go`](../../internal/httpapp/control.go#L1335-L1348) and
[`progress.go`](../../internal/store/progress.go#L268-L334). Measure rows and
bytes first; a newsletter-scoped progress query may lower tail latency while
preserving the page contract.

### Today focus and retention

Retention performs several sequential reads: event history, return timing,
notification preference, backlog/due review counts, due review selection,
uncompleted generated issue selection, and newsletter fallback. See
[`retention.go`](../../internal/store/retention.go#L26-L173).

These reads are conditional: accounts without activity return early, active
accounts skip reentry/backlog selection, and finding a due review avoids later
fallback queries. Benchmark active and inactive accounts separately rather
than assigning the longest query chain to every workspace request.

Today lesson candidates use joins and lateral subqueries over progress, events,
concepts, and evidence, while the review candidate uses `count(*) OVER ()`
before selecting one row. See [`today.go`](../../internal/store/today.go#L117-L212)
and [`today.go`](../../internal/store/today.go#L298-L337).

The count window is a candidate for a plan comparison when many review items are
due. Removing it would change the displayed due count, so any optimization must
retain that product behavior, perhaps through a maintained count or a separate
bounded count query after measuring.

The duplicate retention call from workspace is documented in
[frontend/API audit](frontend-api.md#p0-share-retention-computation-within-the-request)
and should be tested before broader retention redesign.

### Library and issue pages

Workspace and library paths use bounded limits and keyset cursors. Keep those
properties. Library search uses full-text search with a substring fallback; run
plans for both paths and common/rare terms. See
[`issues.go`](../../internal/store/issues.go#L1235-L1402) and
[`012_lesson_search.sql`](../../internal/store/migrations/012_lesson_search.sql#L1-L30).

Issue detail performs multiple account-owned reads before computing its ETag and
fetching the artifact. A 304 avoids artifact and response serialization work,
but it does not avoid the preceding reads. See
[`control.go`](../../internal/httpapp/control.go#L1880-L2070).

## Operational snapshot overhead

`OperationalSnapshot` executes global aggregates over `issue_stage_attempts`
and `product_events`, in addition to queue, delivery, billing, retention, and
cleanup values. It also reads pool statistics. As operational tables grow,
this scrape can consume database CPU and connections while the metrics endpoint
is trying to observe the system. See
[`operational.go`](../../internal/store/operational.go#L48-L179).

**Hypothesis:** scrape cost becomes material at larger event volumes. This is
not evidence that the current query is slow at current scale.

**Experiment:** run the exact snapshot query with representative table sizes
using `EXPLAIN (ANALYZE, BUFFERS)`, while concurrent workspace and worker load is
active. Attribute pool wait and database CPU to the scrape. Measure freshness
and query duration at the current 15-second scrape interval described in
[`config.alloy`](../../infra/monitoring/config.alloy#L1-L80).

**Proposal for review:** if measured, move expensive historical aggregates to a
rollup, bounded time window, partition/retention policy, or precomputed metric
store. Keep gauges that must be current on a cheap path, and document any
eventual consistency. Do not add indexes based only on intuition; validate
write amplification and vacuum behavior.

## Safe database experiment sequence

1. Capture representative plans and operation-level timings.
2. Test request-scoped retention sharing.
3. Test scoped progress and summary payloads with equal ownership/output
   semantics.
4. Test source policy/snapshot batching in isolation with exact evidence
   equivalence and duplicate/retry cases.
5. Test operational rollup alternatives under scrape plus worker load.
6. Recheck pool sizing only after statement-level work is understood.
