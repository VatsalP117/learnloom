# Frontend and HTTP latency paths

This document records the request path and proposals for interactive reads. It
contains no implementation. Statements under “Observed” are source facts;
statements under “Hypothesis” require measurement.

## Observed request path

The hosted app configures the API client, starts `App` and `AppGraph` dynamic
imports, preloads the workspace, and requests `/api/me` in the same effect.
The authenticated app is therefore already warming code and data while the
profile request is in flight. A proposal to make these four actions concurrent
would duplicate work that is already concurrent. See
[`web/src/HostedApp.tsx`](../../web/src/HostedApp.tsx#L100-L128).

`apiFetch` obtains a Clerk token before every request, attaches the bearer token
and JSON headers, and attaches CSRF protection to non-GET requests.
`apiJSON` always consumes a JSON response body. Any client cache must preserve
that authentication and error behavior. See [`web/src/api.ts`](../../web/src/api.ts#L40-L96).

The server middleware starts a request clock, classifies the host, routes the
request, and observes a route-template histogram in a deferred function. The
structured request log is emitted after the metric observation, so the two
durations should not be assumed to have identical boundaries. Existing request
metrics also do not include browser queueing, network transfer, or time spent
after the handler writes the response. See
[`internal/httpapp/server.go`](../../internal/httpapp/server.go#L199-L286).

For authenticated routes, account lookup happens in the middleware before the
route handler. The middleware also performs synchronous public referral
attribution when applicable. Origin/CSRF and content-type checks run before
mutating controls. See [`server.go`](../../internal/httpapp/server.go#L511-L560).

## `/api/workspace` assembly

`workspaceSnapshot` starts six `errgroup` branches concurrently:

1. `ListNewsletters` builds the stream/sidebar records.
2. `GetRetentionState` calculates account rhythm and backlog state.
3. `ListWorkspaceIssuesPage` returns the first issue page.
4. `ListWorkspaceReviews` returns a bounded review list.
5. `ListRecentLessonProgress` returns a bounded progress list.
6. `RefreshTodayFocus` recalculates and persists Today focus.

The response cannot be written until the slowest branch completes. The handler
uses a private `max-age=0, must-revalidate` directive. This is appropriate for
fresh personalized state, but it means a browser refresh still reaches the
origin. See [`control.go`](../../internal/httpapp/control.go#L907-L1000).

### P0: share retention computation within the request

**Observed:** `GetRetentionState` runs as one workspace branch. `RefreshTodayFocus`
calls `GetRetentionState` internally and then persists the selected focus. The
two calls can run concurrently for the same account and timestamp window. See
[`today.go`](../../internal/store/today.go#L50-L115) and
[`retention.go`](../../internal/store/retention.go#L26-L173).

**Hypothesis:** the duplicate reads add pool acquisition, SQL execution, and
database CPU to the request. They could also increase contention with issue
generation. The request's wall time is still the maximum of all branches, so
the improvement depends on whether retention is the slow branch or merely adds
load.

**Proposal for review:** compute one request-scoped retention value and pass it
to a Today-focus calculation, or have one coordinator own both operations. Keep
the existing account ID and `now` semantics explicit. The persistence write
must still be account-owned and preserve the current Today-focus upsert rules.

**Experiment:** on a staging branch, trace query spans for one workspace
request. Compare query count, pool wait, database CPU, p50/p95 handler time,
and Today-focus freshness with the shared-value design. Test cold accounts,
accounts with a review backlog, and accounts with generated issues. The oracle
is equal retention/Today output and exactly one `GetRetentionState` calculation
per request; that calculation can still execute multiple SQL statements.

### P1: narrow the workspace contract only where the UI permits

The workspace payload includes all newsletters, 24 issues, 8 reviews, and 24
recent progress records. The bounded issue/progress/review portions are good
latency choices. Preserve their cursors and limits. The first question is bytes
and serialization time, not whether another parallel goroutine is needed.

Measure response bytes before compression, compressed bytes, JSON marshal time,
and browser parse/hydration time. If a field is only needed by one route,
consider a smaller initial snapshot plus a route-specific request. This is an
API contract proposal requiring frontend call-site review. Do not split data
that is needed for the first paint into a serial waterfall.

`hydrateWorkspace` maps issues to newsletters and synchronizes lesson progress
on every hydrated snapshot. This is linear in the returned lists and is likely
small at current limits, but it should be included in a browser profile before
adding client-side memoization. See [`useWorkspace.ts`](../../web/src/useWorkspace.ts#L46-L76).

## Client cache and refresh behavior

`useWorkspace` uses module-level cache state with a five-minute freshness window.
It deduplicates a normal in-flight request, but `force=true` starts a new
request even when another request is active. `invalidateWorkspaceCache` clears
the value and timestamp but does not cancel the in-flight request or fence its
result. The cache also has no explicit account/session identity. See
[`useWorkspace.ts`](../../web/src/useWorkspace.ts#L6-L43) and
[`useWorkspace.ts`](../../web/src/useWorkspace.ts#L97-L145).

### P1/P2: safe cache improvements

**Hypothesis:** focus/visibility refreshes, mutation-triggered invalidations,
and onboarding transitions can produce overlapping requests or allow an older
response to repopulate a newer session's module cache. The frequency and user
impact are unmeasured.

**Proposal for review:** retain the current dedupe and add an explicit session
identity plus monotonically increasing request generation. A response may
populate the cache only if its session identity and generation still match.
Decide whether invalidation should abort the request or merely fence it. Keep
stale-while-revalidate optional and account-scoped; do not expose personalized
workspace data to a shared intermediary.

**Experiment:** instrument request generation, force refresh, abort, cache hit,
and response age. Exercise sign-out/sign-in as two accounts in one browser,
rapid focus events, and mutation followed immediately by reload. The oracle is
no stale account data and no duplicate request for a single normal load. Compare
origin request count and time-to-usable-workspace.

### ETags and conditional requests

Issue detail computes an ETag after loading issue, feedback, notes, progress,
retrievals, navigation, and site data. A matching request returns 304 before
fetching the artifact, which saves artifact transfer and JSON serialization.
The preceding database reads still happen. See
[`control.go`](../../internal/httpapp/control.go#L1880-L2007) and
[`control.go`](../../internal/httpapp/control.go#L2008-L2070).

Treat ETags as a transfer optimization until traces prove otherwise. A future
workspace validator would need a cheap version source; hashing a fully
assembled personalized payload would move work rather than remove it. Any
conditional response must retain account scoping, `Vary` semantics, and
mutation invalidation behavior.

### P1: measure independent issue-detail reads

Unlike workspace and newsletter detail, `issueDetail` loads feedback, notes,
progress, retrieval responses, navigation, and site sequentially after the
account-owned issue lookup. The lookup and generated/artifact checks must remain
first. The subsequent reads are candidates for a bounded parallel group or a
purpose-built aggregate query, subject to a consistency review. See
[`control.go`](../../internal/httpapp/control.go#L1880-L1955).

Measure individual statement and pool-wait durations before choosing either
design. Parallel reads can reduce summed round trips to the slowest branch
under spare capacity, but consume more connections per request and can worsen
p95 under load. Validate the same response and errors, cancellation when a
branch fails, and account isolation for both 200 and 304 requests. If the page
needs a consistent database snapshot, explicit transaction semantics matter
more than issuing independent queries concurrently.

## Route-specific observations

### Library search

The frontend waits 250 ms before changing the query, aborts the previous
request, and ignores stale versions. The API validates the query, uses a limit,
and accepts a keyset cursor. These behaviors already protect both perceived
latency and server load. See [`useLibrary.ts`](../../web/src/useLibrary.ts#L29-L95)
and [`control.go`](../../internal/httpapp/control.go#L1037-L1087).

The database search path prefers the `lesson_search_documents` full-text index
and has a substring fallback. Measure query plans for short/common queries,
empty queries, and deep cursors. Do not remove the fallback without product
agreement; instead determine whether it is the tail path and whether an
alternative indexed search preserves result semantics. See
[`issues.go`](../../internal/store/issues.go#L1293-L1402) and
[`012_lesson_search.sql`](../../internal/store/migrations/012_lesson_search.sql#L1-L30).

### Newsletter detail

The handler already launches eight reads concurrently, including newsletter
detail, up to 100 issues, all account lesson progress, sidebar newsletters,
source summary/catalog, curriculum, and site. See
[`control.go`](../../internal/httpapp/control.go#L1308-L1399).

`ListLessonProgress` is account-wide rather than newsletter-scoped, and the
detail payload also loads all newsletters for the sidebar. **Hypothesis:** as an
account accumulates history, these two payloads can dominate query rows,
serialization, and browser parse time even though the visible page is bounded.
Measure rows, bytes, marshal time, and UI use before proposing a scoped
progress query or a separately cached sidebar endpoint. Ownership and the
newsletter/account predicates must remain in every narrowed query.

### Static app and telemetry

The runtime embeds the built frontend. `serveAppIndex` sets an edge cache
directive before `serveIndex` applies the response headers; verify the final
header set at the edge and browser before changing it. The current app document
is an account-neutral bootstrap, with browser `no-store` and a separate
Cloudflare-specific cache directive. Keep that distinction: personalized
HTML/API data must not inherit public shell caching. See
[`server.go`](../../internal/httpapp/server.go#L498-L509) and
[`server.go`](../../internal/httpapp/server.go#L569-L582).

Web vitals are sent as keepalive POSTs after `/api/me` loads. Errors are ignored,
which is good for user flow, but each metric still passes through auth and the
server endpoint. Measure request volume and whether telemetry competes with
interactive traffic before batching or sampling. Do not disable the signal
without an observability replacement. See [`performance.ts`](../../web/src/performance.ts#L1-L31).

## Suggested frontend/API sequence

1. Capture browser navigation, LCP/INP, fetch timings, response bytes, and
   server request IDs for workspace, library, newsletter detail, and issue
   detail.
2. Measure `/api/workspace` branch durations and SQL spans. Test the retention
   sharing design first.
3. Measure newsletter detail rows and bytes; only then decide whether to split
   or scope account-wide payloads.
4. Add session identity and stale-response fencing to cache design review
   before any stale-while-revalidate behavior.
5. Verify final CDN/browser cache headers and compression with a production-like
   edge in staging. Compare origin work separately from transfer time.
