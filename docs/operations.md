# Production operations

## Deployment order

1. Provision managed Postgres and an encrypted, private S3 bucket.
2. Configure Clerk production domains and a signed webhook endpoint.
3. Configure a verified Resend sender and OpenAI-compatible model endpoint.
4. Run the immutable image as `migrate`; only then roll out `web` and `worker`.
5. Route apex, app, and wildcard learner DNS through TLS to `web`.
6. Verify public `/healthz` and `/readyz`, then scrape `/metrics` from the
   private web operations listener (`WEB_METRICS_ADDR`, default `:9091`) and
   worker operations listener (`WORKER_METRICS_ADDR`, default `:9090`) before
   shifting traffic. Do not route either metrics listener through the public
   ingress.

For the self-contained VM deployment, follow the
[Dokploy deployment guide](dokploy-deployment.md).

Run one migration job at a time. The migration role takes a Postgres advisory
lock and applies embedded migrations transactionally.

## Monitoring

Alert on sustained readiness failure, web 5xx rate, rate-limit spikes, queued
Issue age, exhausted Issue attempts, unknown delivery outcomes, delivery
failure rate, worker Claim recovery, model latency/error rate, Postgres pool
saturation, and S3 errors. Logs are JSON and include request IDs without model
prompts, source bodies, tokens, or secrets.

Worker metrics also expose active Issues, drain state, recovered Claims,
renewal failures, and Claims explicitly released during shutdown. A draining
worker returns `503` from its readiness endpoint so deployment traffic and new
work move elsewhere while existing Claims continue.

Web operations metrics expose Issue, delivery, recap, account-deletion, and
orphan-artifact cleanup depth; oldest Issue, delivery, and overdue artifact
cleanup age; and Postgres acquired, idle, total, and maximum connections plus
aggregate acquisition wait. Alert from these bounded, content-free signals
rather than from learner text or query strings.

Model stage metrics persist provider-reported input/output tokens, retries,
latency, and estimated micro-USD cost. Configure the two per-million-token
rates from the current provider agreement, not from an old dashboard. The
worker serializes claim admission and reserves
`MODEL_MAX_ESTIMATED_COST_MICRO_USD_PER_ISSUE` against
`GLOBAL_DAILY_MODEL_BUDGET_MICRO_USD`; once the reservation cap is reached, no
new generation Claim starts. In-flight Claims may finish. Import
`infra/monitoring/learnloom-rules.yaml` and
`infra/monitoring/learnloom-dashboard.json` into the operational monitoring
stack, and keep their alert routing private.

## Backup and restore

- Enable continuous Postgres backups and point-in-time recovery.
- Enable S3 versioning, server-side encryption, lifecycle retention, and
  cross-region replication when the recovery objective requires it.
- Test restore quarterly into an isolated account: restore Postgres, restore or
  remap the artifact bucket, run readiness checks, and preview sampled Dossiers.
- Use `scripts/restore-drill.sh` to refuse non-drill targets, compare schema and
  row evidence, verify a restored artifact checksum, and write a dated evidence
  record under `docs/restore-evidence/`.
- Never replay delivery rows in `sent` or `outcome_unknown` state during restore.

## Incident controls

Use the durable generation control to stop new model Claims without
redeploying. Scale workers to zero to pause scheduling and delivery if Claim
churn or provider behavior is unsafe. Rotate a compromised provider credential,
restart the affected role, and audit logs by request ID and Account ID.

Issue Failure internal detail remains in Postgres and structured logs. Learner
interfaces receive only the safe message, stable failure code/category/stage,
retryability, and incident identifier. Investigate an incident through its
Issue Attempt and stage rows; never copy internal detail into learner support
messages.

For an account deletion incident, verify the Account is inactive first and
inspect its deletion queue row. Retry the idempotent artifact phase; successful
artifact cleanup is followed by transactional relational erasure, identity
tombstoning, and a privacy-minimal receipt.

For an orphan-artifact alert, inspect only the cleanup key, attempt count, and
error. Confirm no generated Issue references the key and the originating
generation claim is expired before any manual deletion. Restore object-store
access and let the idempotent queue drain; do not bulk-delete an Account prefix
unless processing an authorized Account-erasure request.

## Rollback

Deployments are immutable. Roll back `web` and `worker` to the previous image;
do not reverse a database migration in place. Schema changes must remain
compatible with the immediately previous application image until a rollout is
complete, even though no compatibility with the removed local product is kept.
## Autonomous source discovery

Source discovery is disabled by default. To run the self-hosted discovery
profile:

```sh
docker compose --profile discovery up -d searxng searxng-valkey
docker compose --profile discovery up -d web worker
```

Set `SOURCE_DISCOVERY_ENABLED=true`, keep `SEARXNG_BASE_URL` pointed at the
operator-controlled SearXNG instance, and replace `SEARXNG_SECRET`. The
same discovery flag is passed to `web` so the creation screen only offers
discovered and hybrid modes when the worker capability is actually available. The
SearXNG configuration explicitly enables JSON output; a `403` from `/search`
usually means JSON was removed from `search.formats`.

Useful checks:

```sh
docker compose --profile discovery config
curl --get http://127.0.0.1:8080/search \
  --data-urlencode 'q=LLM inference official documentation' \
  --data 'format=json'
```

Container health is separate from functional discovery checks. The SearXNG
service runs a non-invasive health check against its local `/healthz`
endpoint (`wget --spider http://127.0.0.1:8080/healthz`); with the discovery
profile enabled, `web` and `worker` wait for that health check to pass before
starting. It only confirms the HTTP service is answering — it never runs a
real search query. The `curl ... /search?format=json` command above is the
separate, one-time functional check that discovery actually returns JSON
results; treat a green `/healthz` and a `200` JSON search as distinct signals.

Do not expose SearXNG publicly unless it has an appropriate reverse proxy and
rate limits. Search outages do not affect provided-only streams. Hybrid
streams continue only when their provided catalog already satisfies the hard
evidence minimum; otherwise the Issue fails without calling the model.

## Curated public examples

The web role accepts `FEATURED_SITE_USERNAMES` as a comma-separated allowlist
for the apex `/examples` gallery. A configured site is still excluded unless
its owner has made it public, explicitly enabled search indexing, and published
at least one Dossier.

Review the site and recent Dossiers before adding a username. Treat owner
consent as revocable: disabling search indexing or making the site private
removes it without an operator change. Remove unsafe, low-quality, stale, or
disputed entries from the allowlist and redeploy the web role.
