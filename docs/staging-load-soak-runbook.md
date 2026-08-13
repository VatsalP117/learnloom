# Staging load and soak runbook

Status: repository harness ready; staging forecast and run evidence not yet
collected.

This procedure exists to test a declared launch forecast. It does not claim a
capacity number from source code or from a laptop benchmark.

## Before the run

The release owner must record, with an internal evidence reference:

- forecast peak requests per second and concurrent requests for the first
  launch cohort;
- the public and authenticated read routes represented by the mix;
- expected web and worker replica counts, database connection budget, and
  model/provider concurrency limits;
- the test duration and why it covers the expected burst or soak period;
- pass thresholds. The repository defaults are at least 90% achieved request
  rate, no more than 1% failed requests, and aggregate p95 no greater than two
  seconds, matching the existing HTTP latency alert boundary.

Do not guess these values merely to obtain a green result. If no defensible
forecast exists, the checklist item remains open.

## Read workload

Run against the dedicated staging hostname. Targets must be explicit GET URLs
without query strings. The runner refuses `learnloom.blog` production hosts
unless `-allow-production` is deliberately supplied; production load requires
separate written authorization.

```sh
export LOAD_VERIFY_BEARER_TOKEN='<short-lived dedicated staging test token>'

go run ./cmd/load-verify \
  -staging-host app.staging.example \
  -targets 'https://app.staging.example/healthz,https://app.staging.example/api/workspace' \
  -rate 10 \
  -concurrency 20 \
  -duration 30m \
  -min-rate-percent 90 \
  -max-error-percent 1 \
  -max-p95 2s \
  > /approved/evidence/location/load-report.json

unset LOAD_VERIFY_BEARER_TOKEN
```

Use a dedicated non-human test account and classify it as `test` with reason
`manual_test`; its activity must never enter real-user evidence. Do not put a
token into arguments, source control, or the JSON report. The runner bounds
response bodies, rejects credentials/query strings in targets, and refuses
cross-origin or insecure redirects.

The JSON report records requested rate, concurrency, duration, minimum and
actual request counts, achieved rate, failures, p50/p95/p99, per-target p95,
and the final pass decision. A nonzero exit or `passed: false` leaves the gate
open.

## Worker and dependency soak

The read runner is only one part of the staging gate. During the same declared
soak window:

1. Seed the forecast number of dedicated test accounts and queued lessons via
   the authenticated product API; never write queue rows around application
   invariants.
2. Include source discovery, first-lesson generation, artifact reads, review,
   and delivery using safe test recipients.
3. Observe queue oldest age, account fairness, claim recovery, Postgres pool
   saturation, provider latency/retries, artifact errors, delivery outcomes,
   model reservation/spend, and cleanup backlog in the versioned dashboard.
4. Confirm no account receives two consecutive retryable failures without the
   documented recovery path, and no budget/allowance invariant is bypassed.
5. Continue the provider-evaluation workload for the full 72-hour soak required
   by `LL-210`; a 30-minute HTTP pass does not substitute for it.

Attach the release SHA, schema version, topology, forecast reference, JSON
report, dashboard window, alert outcomes, queue drain time, and any incident
references to a dated release record. Remove the test accounts through the
normal deletion workflow afterward and verify artifact cleanup.
