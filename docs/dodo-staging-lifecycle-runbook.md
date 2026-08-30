# Dodo Payments staging lifecycle runbook

Use Dodo Payments test mode before enabling production commerce. Configure the
test API host, test-mode `pdt_` products, API key, and endpoint signing secret;
keep `PAID_COMMERCE_APPROVED=false` outside an approved production release.

Register `https://<staging-host>/webhooks/dodo` for subscription active,
updated, on-hold, paused, unpaused, renewed, plan-changed, cancelled, failed,
and expired events, plus payment succeeded/failed and refund succeeded.

Exercise both Essential and Pro:

1. Start checkout and confirm the returned URL uses
   `test.checkout.dodopayments.com`.
2. Complete a test payment. Confirm subscription access is activated only by a
   product-matched subscription event and the payment is recorded once.
3. Replay the same signed webhook and confirm it is idempotent; tamper with the
   body or use a stale timestamp and confirm rejection.
4. Put renewal on hold and confirm grace, then reactivate and confirm access.
5. Schedule and complete cancellation; existing lessons remain readable while
   new generation pauses after access ends.
6. Issue partial and full refunds. Confirm the financial adjustment is recorded
   without treating the refund itself as subscription cancellation.
7. Open the hosted customer portal and verify cancellation, invoices, and the
   return link. Simulate provider failure and confirm existing content remains
   available and new checkout creation fails closed.

Before production, repeat with live-mode product identifiers in a controlled
release, record the approval evidence reference, and verify the billing backlog
and entitlement dashboards remain clear.
