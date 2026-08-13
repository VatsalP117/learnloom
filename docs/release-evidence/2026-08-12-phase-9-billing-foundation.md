# Phase 9 billing foundation evidence — 2026-08-12

## Scope and status

This evidence closes the repository-backed foundation in `LL-901` through
`LL-908`. It does **not** close the Phase 9 exit gate. Paddle account approval,
India payout verification, real sandbox/staging lifecycle runs, entity-specific
legal review, ten voluntary design-partner payments, and representative gross
margin remain external evidence gates.

## Provider decision (`LL-901`)

Paddle is the conditional launch provider because it is merchant of record and
supports subscription tax/VAT handling, compliant invoices, refunds, lifecycle
webhooks, trials, and a hosted customer portal. The decision remains conditional
on account approval and founder-specific payout/India availability verification;
configuration may remain unset without weakening entitlement enforcement.

Primary references reviewed:

- Paddle VAT/tax handling: https://www.paddle.com/help/sell/tax/how-paddle-handles-vat-on-your-behalf
- Seller/merchant-of-record handbook: https://www.paddle.com/seller-guides/seller-handbook
- Subscription trials: https://developer.paddle.com/concepts/subscriptions/trials/
- Webhook signature verification: https://developer.paddle.com/webhooks/about/signature-verification/
- Customer portal sessions: https://developer.paddle.com/api-reference/customer-portals/create-customer-portal-session/
- Checkout transactions: https://developer.paddle.com/api-reference/transactions/create-transaction/

## Durable commercial model (`LL-902`–`LL-907`)

- Migration `030_billing_entitlements.sql` adds Free/Pro plans, subscription and
  entitlement state, allowance periods, usage reservations, lifecycle events,
  and webhook receipts.
- Migration `031_billing_economics.sql` adds cancellation/revenue facts and a
  replay-safe revenue ledger.
- Free currently allows 3 generation units per 30 days; Pro hypothesis allows
  30. These remain test allowances, not an unlimited promise.
- The first lesson, manual generation, and scheduled generation reserve a unit
  atomically in their domain transaction. Duplicate active work does not consume
  another unit. Rejected work is not created.
- Trial/active remain enabled; past-due enters a seven-day grace window;
  paused/canceled/refunded stop new generation. Existing lessons are untouched.
- Lifecycle projection is monotonic by provider event timestamp. Exact replays
  are idempotent; an event ID replayed with a changed payload is rejected.
- Signed irrelevant Paddle events are retained as processed audit receipts.
- Completed transaction and subscription lifecycle events grant Pro only when
  their signed Paddle entity contains the server-configured Pro price. Account
  custom data cannot activate an unrelated Paddle product.
- Approved Paddle `adjustment.created`/`adjustment.updated` refund events create
  financial facts without assuming a partial refund canceled the subscription;
  cancellation remains authoritative only from subscription lifecycle events.

## Checkout, portal, usage, and copy (`LL-805`, `LL-908`)

- Authenticated POST `/api/me/billing/checkout` creates an automatically
  collected transaction with server-owned price ID and account attribution.
- Checkout redirects are accepted only on the configured app host.
- Paddle API version 1 is pinned. Because Paddle does not offer arbitrary
  client idempotency keys, a durable single-pending-checkout record reuses a
  transaction for 30 minutes, collapses concurrent successful creates to the
  already selected checkout, and closes it from the completed transaction
  webhook.
- Authenticated POST `/api/me/billing/portal` creates a temporary Paddle-hosted
  portal session; only HTTPS Paddle hosts are accepted.
- Settings shows plan, allowance, usage, reset date, grace/paused semantics, and
  checkout/management actions.
- Marketing labels $15/month as a design-partner price hypothesis and explains
  Free/Pro allowances, cancellation, preservation of learned content, taxes,
  invoices, and merchant-of-record behavior.
- Terms disclose recurring billing, taxes/invoices, grace, cancellation,
  post-cancellation content access, and refund escalation. Entity/jurisdiction
  counsel review remains open under `LL-909`.
- Provider credentials no longer authorize production sales by themselves.
  Production checkout/portal requires `PAID_COMMERCE_APPROVED=true` and a
  bounded non-secret approval evidence reference; production accepts only the
  live Paddle API and staging accepts only the sandbox API. This keeps `LL-909`
  and `LL-911` fail closed while allowing sandbox lifecycle work.

## Economics instrumentation

- `scripts/billing-economics.sql` reports lifecycle funnel, trial cohorts,
  conversion, cancellations, recognized payment/refund cash facts, model COGS,
  and a partial margin including model COGS and Paddle fees. Revenue uses the
  pre-tax subtotal rather than customer total, so collected tax is not
  misrepresented as Learnloom revenue.
- Migration 36 adds an append-only, reference-idempotent operational COGS
  ledger for search, email, storage, support, infrastructure, and other costs.
  Gross margin is emitted as `NULL` until every required category has entries
  and shared costs are allocated; incomplete accounting cannot appear as a
  passing margin.
- The economics report also groups week-four retained paid accounts by first
  payment cohort with net revenue, provider fees, model cost, and attributed
  operational cost.
- The report explicitly refuses to treat model-only margin as the launch gate;
  search, email, storage, payment fees, tax adjustments, and support COGS remain
  required before `LL-912` can close.

## Verification

Against a fresh PostgreSQL 17 schema through migration 36:

- `TEST_DATABASE_URL=... go test ./internal/store` — pass.
- `go test ./...` — pass.
- `go vet ./...` — pass.
- `pnpm check` — pass (API contract, ESLint, TypeScript, production build).
- `pnpm test` — 21 files / 65 tests pass.

Focused coverage includes allowance exhaustion, duplicate-work idempotency,
period rollover, trial/past-due/refund projections, stale and duplicate webhook
events, changed-payload replay rejection, ignored-event audit retention, Paddle
signature freshness/tamper checks, checkout payload attribution, provider error
boundaries, return-host validation, and Paddle-only portal URLs.

In-app browser review of the local production frontend found:

- desktop viewport 1280×720: pricing section spans the viewport with no
  horizontal overflow;
- mobile viewport 390×844: both pricing cards are 346px wide inside 22px page
  gutters, the document scroll width is exactly 390px, and no horizontal
  overflow occurs;
- pricing, allowance, merchant-of-record, tax/invoice, cancellation, and
  post-cancellation access copy is visible and internally consistent;
- no browser console warnings or errors were observed.

## Feedback instrumentation

- Settings collects a stable non-conversion/cancellation reason taxonomy plus
  an optional 1,000-character note that explicitly warns against sensitive
  information.
- The server validates the taxonomy and accepts cancellation reasons only after
  an authoritative canceled/refunded billing state; free accounts can submit
  non-conversion reasons. Updates replace the prior response for that context.

## Repository tasks closed

- `LL-910`: authoritative trial, paywall, checkout, payment, failure,
  cancellation, reactivation, refund, and stable reason instrumentation is in
  durable storage. Production event-volume validation remains a deployment
  gate, not an absent instrument.
- `LL-912`: the aggregate economics report covers lifecycle funnel, trial
  conversion, churn/cancellation, revenue/refunds, provider fees, model COGS,
  every named operational COGS category, accounting completeness, gross margin,
  and retained paid cohorts. Margin remains intentionally null until accounting
  is complete.

## Open Phase 9 gates

- `LL-909`: entity- and jurisdiction-specific counsel/accountant review.
- `LL-911`: real Paddle sandbox and staging lifecycle/outage exercise.
- Production commerce approval must remain false until both records exist and
  the approved seller/refund/support copy is deployed.
- Exit gate: ten voluntary paying design partners and representative gross
  margin below the 25% COGS ceiling.
