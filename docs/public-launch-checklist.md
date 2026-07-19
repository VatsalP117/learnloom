# Public launch checklist

This is the short launch gate for hosted Learnloom. Do not enable public
signup or send production traffic to wildcard user subdomains until every
blocker below is checked.

The detailed design remains in
[the hosted-subdomains implementation plan](hosted-subdomains-implementation-plan.md).
This file intentionally limits the pre-launch list to security, privacy,
reliability, and cost controls that can materially harm users or the service.

## Launch blockers

### 1. Prevent SSRF through user-provided feeds

**Why it matters:** authenticated users can configure RSS or Atom URLs. A
server-side fetch must not be usable to reach loopback, private networks,
cloud metadata endpoints, or a private destination reached through a redirect
or DNS change.

- [ ] Apply the existing public-address validation and pinned-connection model
      to initial feed requests and every redirect.
- [ ] Reject credentials, non-HTTP(S) schemes, private/reserved IP ranges, and
      unsupported response types.
- [ ] Bound response bytes, redirects, and timeouts.
- [ ] Add tests for IPv4, IPv6, encoded addresses, redirects, and DNS rebinding
      behavior.

**Exit evidence:** feed-fetch tests prove private destinations are rejected,
and a staging request cannot reach the instance metadata service or internal
services.

### 2. Enforce quotas, rate limits, and fair queueing

**Why it matters:** each generation can consume model tokens, network capacity,
email quota, and worker time. Authentication alone does not prevent accidental
or deliberate cost exhaustion.

- [ ] Limit username checks and claims by session/IP at the ingress.
- [ ] Limit manual generation and concurrent work per account.
- [ ] Set maximum active Newsletters and daily generations per account.
- [ ] Make the worker queue fair so one account cannot starve others.
- [ ] Add global spend/circuit-breaker controls and actionable operator alerts.

**Exit evidence:** automated tests cover account limits and queue fairness;
staging demonstrates throttled requests return a stable `429` without creating
work.

### 3. Complete account suspension and deletion

**Why it matters:** Clerk is the identity source, while Learnloom stores
account, site, Newsletter, Issue, delivery, and artifact data locally. Deleted
or suspended identities must not retain access or leave public content online
indefinitely.

- [ ] Verify Clerk webhook signatures and handle relevant user lifecycle events
      idempotently.
- [ ] Immediately make suspended/deleted accounts and sites unavailable.
- [ ] Define and implement artifact deletion or documented retention.
- [ ] Ensure retries and workers cannot process deleted-account work.
- [ ] Test duplicate, delayed, and out-of-order webhook delivery.

**Exit evidence:** deleting a staging Clerk user revokes dashboard access,
removes the public site, stops queued work, and applies the documented data
retention policy.

### 4. Validate production identity, DNS, TLS, and ingress

**Why it matters:** hostname routing and Clerk authorization depend on exact
origins. A proxy or DNS mistake can break authentication, expose the control
plane, or route tenant traffic incorrectly.

- [ ] Configure apex, `www`, `app`, Clerk, and wildcard DNS records.
- [ ] Confirm TLS coverage for `learnloom.blog` and
      `*.learnloom.blog` end-to-end.
- [ ] Configure Clerk's production domain, Google connection, allowed
      subdomains, and exact redirect URLs.
- [ ] Preserve and validate the original `Host`; do not trust arbitrary
      forwarded-host headers.
- [ ] Confirm unknown and reserved hosts fail closed.
- [ ] Run two-user browser tests covering sign-in, claim, publish, private mode,
      cross-tenant denial, canonical redirects, and sign-out.

**Exit evidence:** the complete browser journey passes on staging using real
production-style DNS and Clerk configuration.

### 5. Prove recovery, privacy changes, and operations

**Why it matters:** SQLite and persisted Dossier artifacts are durable only if
they are backed up together and can be restored. Public caches must also stop
serving content promptly when a site or Dossier becomes private.

- [ ] Back up the SQLite database and artifact storage as one recoverable data
      set.
- [ ] Perform and time a staging restore; document recovery point and recovery
      time objectives.
- [ ] Verify migration rollback/recovery from a pre-migration backup.
- [ ] Verify privacy changes expire or purge every public cache within the
      documented window.
- [ ] Add health, queue-depth, failure-rate, delivery, storage, and spend
      monitoring with alerts.
- [ ] Write a concise incident runbook covering auth outage, queue runaway,
      email failure, database corruption, and compromised credentials.

**Exit evidence:** a restore drill succeeds and an operator can detect and
contain each documented incident using the runbook.

## Final launch sign-off

- [ ] `npm test` passes.
- [ ] `npm run check` passes.
- [ ] All five blocker sections have attached staging evidence.
- [ ] A second person reviews the security and recovery evidence.
- [ ] Public signup remains disabled until the release owner records approval.

## Explicitly not launch blockers

These can follow the first controlled public release:

- custom domains;
- engagement analytics;
- richer site themes and editor polish;
- removing the deprecated compatibility CLI after service entrypoints replace
  its current server and worker duties;
- automatic adoption of pre-hosted local data. Legacy rows remain unowned and
  excluded unless an installation performs an audited one-time migration.
