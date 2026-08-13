# Paddle staging lifecycle runbook

This runbook is required evidence for `LL-911`. Do not run it against live
prices or a production customer. Do not copy webhook payloads, API keys,
customer email addresses, or payment details into repository evidence.

## Preconditions

- Dedicated staging hostname, database, object store, Clerk instance, and
  Paddle sandbox account.
- `PADDLE_API_BASE_URL=https://sandbox-api.paddle.com`.
- Sandbox-only API key, webhook secret, and recurring Pro price.
- Immutable `LEARNLOOM_RELEASE_VERSION` visible in the response header.
- Schema version 36 and clean health/readiness.
- Test account whose account ID is attached by the server-created checkout;
  never hand-edit `custom_data` to manufacture success evidence.

## State matrix

For each row, record UTC time, release SHA, Paddle event IDs (in the restricted
operator record), HTTP result, resulting plan/status/entitlement, allowance,
and whether existing lessons remain accessible.

| Scenario | Required result |
| --- | --- |
| Free account | 3/30-day allowance; fourth creation rejected before work exists |
| Checkout open/retry | one pending checkout reused for 30 minutes; no second surfaced payment link |
| Trial starts | Pro/trialing/active; 30 allowance; one trial event |
| Transaction completes | Pro/active; pre-tax revenue and Paddle fee recorded once |
| Duplicate webhook | 204; no duplicate lifecycle, revenue, or usage row |
| Same event ID, changed body | rejected; original receipt preserved |
| Older webhook after newer state | audited but cannot roll state backward |
| Payment failure | past_due/grace; generation available only until grace expiry |
| Paused | generation_paused; existing lessons and review remain readable |
| Resumed | active; reactivation event; allowance restored for current period |
| Cancellation | generation_paused/Free policy; content remains readable |
| Partial approved refund | refund ledger entry; subscription state unchanged |
| Full refund plus cancel | refund and cancellation recorded separately; generation paused |
| Unrelated Paddle price | event audited/ignored; no Pro entitlement |
| Unknown Learnloom account | non-2xx/retryable processing failure; no processed receipt |
| Provider API outage | checkout/portal returns safe service error; no entitlement granted |
| Webhook delivery outage | Paddle retries; later delivery applies exactly once |

## Reconciliation

After the matrix, run in an authorized read-only database session:

```sh
psql "$STAGING_DATABASE_URL" -X -f scripts/billing-economics.sql
```

Verify directly in the restricted operator session:

- every relevant Paddle event ID has one processed receipt;
- every completed transaction has one payment revenue fact;
- every approved refund has one refund fact;
- payment subtotal excludes collected tax and provider fee is separate;
- no ignored/unrelated event changes entitlement;
- no account has more than one pending checkout;
- the public aggregate evidence contains no account/customer identifiers.

## Provider outage exercise

Temporarily point staging at a controlled HTTPS endpoint that returns 503, or
use an approved network fault at the staging boundary. Do not modify production
DNS. Checkout and portal endpoints must fail without creating an entitlement,
revenue row, or misleading success UI. Restore the sandbox endpoint and prove a
new checkout succeeds.

## Evidence boundary

Repository evidence may contain aggregate counts, state names, release SHA,
schema version, pass/fail, and internal evidence references. Keep event IDs,
customer IDs, emails, transaction URLs/tokens, payloads, and screenshots with
payment details only in the restricted operational evidence system.

`LL-911` closes only when every row passes on the exact staged release and the
operator attaches the dated evidence reference. Unit/integration tests do not
substitute for this provider lifecycle exercise.
