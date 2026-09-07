# Latency improvement audit

Status: investigation and proposals only. Reviewed on 2026-09-08 against the
working tree based on commit `8a9f5a3118e2a5401869fbef74baa098a32817c6`. The working tree
also contains unrelated frontend, launch-video, and documentation work. Those
changes are preserved and are not treated as deployed performance evidence.

The audit answers where time can be removed or made less variable. It does not
claim a measured baseline, capacity number, or improvement. Every item marked
**Hypothesis** needs a staging trace, query plan, or load result before an
implementation decision.

## Read the detailed audits

- [Frontend and HTTP request paths](frontend-api.md): browser boot, workspace
  aggregation, API caching, payload shape, and read-route work.
- [Worker and dossier generation](worker-model.md): queue scheduling, claims,
  source preparation, model stages, persistence, and delivery.
- [Sources and database](sources-data.md): external fetch/discovery, snapshot
  writes, SQL query shape, pool contention, and operational aggregates.
- [Validation and rollout](validation-operations.md): instrumentation already
  present, load-harness limits, experiments, gates, and a staged roadmap.

## The current critical paths

### Authenticated workspace read

`HostedApp.OnboardingGate` starts the `/api/me` request, `App` and `AppGraph`
chunk imports, and workspace preload concurrently. This is already a useful
latency optimization; boot should not be described as a profile-then-workspace
sequence. The browser then calls `/api/workspace`, whose handler starts six
database branches in one `errgroup`: newsletters, retention, issue page,
reviews, recent progress, and Today focus. The response is serialized only
after all six finish. See [`HostedApp.tsx`](../../web/src/HostedApp.tsx#L106-L127)
and [`control.go`](../../internal/httpapp/control.go#L907-L1000).

The strongest concrete candidate is duplicate retention work: the workspace
branch calls `GetRetentionState`, while `RefreshTodayFocus` calls
`GetRetentionState` again before persisting Today focus. This is a code fact,
not a measured percentage of request time. See
[`today.go`](../../internal/store/today.go#L50-L68) and
[`retention.go`](../../internal/store/retention.go#L26-L173). A request-scoped
shared result or a Today calculation that accepts the already-computed state
should be measured first.

### Generated lesson

The worker polls on a ticker, dispatches due issues, claims up to
`GlobalConcurrency` issues, starts those issues concurrently, and waits for
the complete batch before processing deliveries and the remaining queues. A
slow issue therefore prevents completed slots from being refilled and delays delivery
work even if other work is ready. See
[`worker.go`](../../internal/execution/worker.go#L231-L307) and
[`worker.go`](../../internal/execution/worker.go#L635-L676).

Within an issue, history, learner state, source preparation, checkpoint load,
model generation, artifact storage, and the final completion transaction form
the critical path. Generation itself has dependent curator, blueprint,
researcher, skeptic, teacher, examiner, and editor stages. The examiner and
optional exploration stages already overlap where their inputs permit it. See
[`worker.go`](../../internal/execution/worker.go#L679-L866) and
[`generator.go`](../../internal/dossier/generator.go#L182-L592).

### Source preparation

Configured source endpoint fetches and article enrichment use bounded ordered
parallelism. Discovery search queries are parallel, but ranked candidates are
resolved and snapshotted one at a time. Feed snapshots also perform URL policy
checks and an insert per usable item. These are likely contributors to tail
latency when discovery is needed, but no gain is claimed without stage timings.
See [`discovery.go`](../../internal/source/discovery.go#L51-L188),
[`service.go`](../../internal/source/service.go#L174-L283), and
[`service.go`](../../internal/source/service.go#L630-L716).

## Priority order

P0/P1/P2 below indicate investigation order, not measured incident severity.
Establish the baseline first. Expected benefits are directional: retention
sharing removes duplicate work; scheduling reduces idle capacity and queue
coupling; discovery overlap reduces serial waiting. None establishes a
percentage gain without measurements. Retention sharing has a relatively small
change surface; scheduler changes and durable projections require substantially
more correctness and recovery testing.

| Priority | Opportunity | Evidence | First decision test | Main guardrail |
| --- | --- | --- | --- | --- |
| P0 | Share retention state inside `/api/workspace` and Today calculation | Duplicate calls are visible in code; cost is unmeasured | Trace one request and compare query count, pool wait, and p95 with a request-scoped shared value in a staging branch | Today focus must use the same account/time semantics and remain persisted atomically enough for existing behavior |
| P0 | Replace worker full-batch waiting with bounded refill or separate queue service loops | Full batch and sequential queue order are code facts; impact is unmeasured | Replay mixed fast/slow generation plus deliveries and compare queue age, fairness, claims, and completion order | Preserve account/global quotas, claim renewal, cancellation, retries, and `outcome_unknown` delivery handling |
| P1 | Reduce model critical-path work only after stage-level traces identify the slow stages | Stage sequence and retry/repair behavior are code facts | Compare stage p50/p95, token counts, retries, quality, and cost by pipeline version | Do not parallelize stages with prompt dependencies or weaken source/evidence requirements |
| P1 | Parallelize bounded discovery candidate resolution and batch safe snapshot writes | Ranked resolution and per-item writes are code facts | Replay the same candidate set with concurrency 1 vs bounded N; measure provider load and evidence equivalence | Keep URL policy checks, SSRF protections, deterministic ordering, and idempotent snapshots |
| P1 | Tune heavy authenticated read SQL and operational aggregates | Query shape is visible; table-size sensitivity is unmeasured | `EXPLAIN (ANALYZE, BUFFERS)` at representative row counts and compare pool wait during scrapes | Keep account predicates, keyset pagination, retention semantics, and atomic writes |
| P1 | Overlap independent issue-detail reads after ownership validation | Feedback, notes, progress, retrieval, navigation, and site reads are sequential | Compare serial reads with bounded fan-out or one aggregate query under spare and saturated pool capacity | Keep authorization first, account predicates, error cancellation, and required snapshot consistency |
| P2 | Add conditional/private caching or narrower payloads to read routes | ETags and private cache headers already exist on selected routes; transfer and hit rates are unmeasured | Measure bytes, 304 rate, freshness, and cache-key isolation per account | Never share personalized data across sessions/accounts; fence stale responses |
| P2 | Deployment and transport tuning | Keep-alive, bounded timeouts, static embedding, and cache directives already exist | Measure edge TTFB, origin time, connection reuse, and compressed bytes | Avoid caching personalized HTML/API responses or relaxing source-fetch safety |

## Existing latency work to retain

The repository already has several good foundations. The improvement plan should
extend them rather than duplicate them:

- HTTP request histograms use route templates, and the server records request
  IDs and duration. See [`server.go`](../../internal/httpapp/server.go#L199-L286)
  and [`metrics.go`](../../internal/httpapp/metrics.go#L1-L120).
- Worker phases and model stages have duration observations; operational
  snapshots expose pool acquisition count and aggregate acquisition duration.
  These are useful, but an aggregate acquisition duration is not a query p95.
  See [`worker.go`](../../internal/execution/worker.go#L1155-L1190) and
  [`operational.go`](../../internal/store/operational.go#L48-L179).
- Frontend boot starts authenticated chunks, workspace preload, and profile
  fetch together. Library search debounces input, aborts superseded requests,
  and fences stale responses. See [`useLibrary.ts`](../../web/src/useLibrary.ts#L29-L67).
- Workspace and issue detail responses use private cache directives/ETags in
  selected paths. A 304 still requires the current server-side data assembly
  before the ETag can be computed for issue detail, so transfer savings should
  not be counted as origin-query savings. See [`control.go`](../../internal/httpapp/control.go#L1381-L1398)
  and [`control.go`](../../internal/httpapp/control.go#L1880-L2007).
- Source HTTP uses bounded bodies, timeouts, conditional validators, keep-alive,
  redirect limits, and public-address checks. Any source latency work must
  preserve those controls. See [`http.go`](../../internal/source/http.go#L29-L128).
- Read APIs use limits/keyset cursors in workspace and library paths. Keep that
  shape when narrowing payloads or adding a route. See [`issues.go`](../../internal/store/issues.go#L1235-L1402).

## Non-negotiable invariants

Latency changes are acceptable only if they preserve the following:

1. **Ownership and privacy:** every personalized read and write remains scoped
   to the authenticated account. A cache key must include the account/session
   boundary, and `Vary: Authorization` alone must not be treated as proof that
   an intermediary is safe for private data.
2. **Source safety:** URL policy checks, DNS rebinding defenses, bounded response
   sizes, redirect restrictions, and timeouts remain on every fetch path. Cached
   snapshots may be reused only under the same policy and freshness rules.
3. **Claims and retry safety:** issue, delivery, deletion, and cleanup claims
   continue to renew, expire, and recover. A worker scheduler must not claim the
   same work concurrently or bypass account/global quota.
4. **Idempotent delivery:** provider acceptance followed by a lost response or
   database completion failure remains `outcome_unknown`; retries require the
   existing reconciliation policy. Faster retries must not duplicate mail.
5. **Atomic learning state:** `CompleteIssue` and `AssessReview` preserve their
   transaction boundaries for usage, history, progress, review state, search
   documents, and events. Faster individual statements must not expose partial
   learner state.
6. **Evidence and quality:** shorter or more concurrent model/source paths must
   retain source provenance, contract validation, checkpoint fingerprints,
   quality evaluation, budget reservations, and audit records.

## Proposed order of work

1. Establish the baseline and traces described in
   [validation and rollout](validation-operations.md), including separate
   browser, API/database, queue, source, and model clocks.
2. Test request-scoped retention sharing. This is low-scope and has a clear
   correctness oracle.
3. Test worker scheduling with a feature-gated bounded refill or independently
   scheduled delivery loop. Treat this as an architecture proposal until
   fairness and claim-recovery tests pass.
4. Use traces to choose between model prompt/stage work, discovery resolution,
   SQL plans, and payload/cache work. Do not optimize by intuition where a
   stage histogram or query plan can identify the wait.
5. Roll out one change at a time, compare aggregate and per-target percentiles,
   and retain an explicit rollback switch for scheduler/model/source changes.
