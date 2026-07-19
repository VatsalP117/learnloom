# Production operations

## Deployment order

1. Provision managed Postgres and an encrypted, private S3 bucket.
2. Configure Clerk production domains and a signed webhook endpoint.
3. Configure a verified Resend sender and OpenAI-compatible model endpoint.
4. Run the immutable image as `migrate`; only then roll out `web` and `worker`.
5. Route apex, app, and wildcard learner DNS through TLS to `web`.
6. Verify `/health/live`, `/health/ready`, and `/metrics` before shifting traffic.

Run one migration job at a time. The migration role takes a Postgres advisory
lock and applies embedded migrations transactionally.

## Monitoring

Alert on sustained readiness failure, web 5xx rate, rate-limit spikes, queued
Issue age, exhausted Issue attempts, unknown delivery outcomes, delivery
failure rate, worker Claim recovery, model latency/error rate, Postgres pool
saturation, and S3 errors. Logs are JSON and include request IDs without model
prompts, source bodies, tokens, or secrets.

## Backup and restore

- Enable continuous Postgres backups and point-in-time recovery.
- Enable S3 versioning, server-side encryption, lifecycle retention, and
  cross-region replication when the recovery objective requires it.
- Test restore quarterly into an isolated account: restore Postgres, restore or
  remap the artifact bucket, run readiness checks, and preview sampled Dossiers.
- Never replay delivery rows in `sent` or `outcome_unknown` state during restore.

## Incident controls

Use the durable generation control to stop new model Claims without
redeploying. Scale workers to zero to pause scheduling and delivery if Claim
churn or provider behavior is unsafe. Rotate a compromised provider credential,
restart the affected role, and audit logs by request ID and Account ID.

For an account deletion incident, verify the Account is inactive first, inspect
its deletion queue row, and retry only the artifact deletion phase. Database
deletion and artifact deletion are intentionally independently observable.

## Rollback

Deployments are immutable. Roll back `web` and `worker` to the previous image;
do not reverse a database migration in place. Schema changes must remain
compatible with the immediately previous application image until a rollout is
complete, even though no compatibility with the removed local product is kept.
