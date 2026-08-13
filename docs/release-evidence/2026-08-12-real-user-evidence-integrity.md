# Real-user evidence integrity — 12 August 2026

## Claim proved by the repository

Schema 38 provides an append-only, operator-controlled classification history
for each account. Its latest entry is one of `real_user`, `founder`, or `test`;
absence is deliberately `unclassified`. The browser and public API cannot set
or change this value, and the audit table stores no email or Clerk identity.

`scripts/product-baseline.sql`, `scripts/design-partner-beta-report.sql`, and
`scripts/billing-economics.sql` expose classification coverage and restrict
launch-gate product, reliability, retention, beta, payment, revenue, and cost
aggregates to explicit real users. Founder/test activity cannot become customer
evidence through an identity naming convention or an omitted filter.

Corrections append a new row with bounded reason `correction`; history is not
overwritten. `scripts/classify-evidence-account.sql` accepts an exact account
UUID resolved separately in an authorized workflow and rolls back if that
active or suspended account does not exist.

## Verification

- Migration ledger structure and evidence-report boundary tests: passed.
- Fresh PostgreSQL 17 migration lifecycle through schema 38: passed.
- All three aggregate reports on an empty schema: passed and returned no
  real-user evidence.
- Seeded one real user, one founder, and one test account: coverage reported all
  three separately while the product funnel counted exactly the real user.
- Seeded two mature real-user activations: a meaningful action on day two
  counted, while an otherwise identical first return after day seven did not.
  The report returned one of two retained accounts (50%) and one retained
  lesson, proving the seven-day upper bound and retained-cost denominator.
- Seeded $10 of external-customer revenue and $90 of founder revenue: coverage
  exposed both, while recognized launch revenue and retained paid-cohort
  economics included only the $10 real-user payment. Gross margin remained
  null because representative COGS categories were absent.
- Missing-account operator classification: exited nonzero, rolled back, and
  left the classification row count unchanged.

## Production evidence still required

This mechanism does not classify existing production accounts automatically.
An authorized operator must review every production evidence account, append
the appropriate classification, record an internal evidence reference, and
then run the aggregate reports. Until that happens, `LL-006` remains open and
unclassified traffic remains excluded.
