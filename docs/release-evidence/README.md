# Release evidence

This directory records dated, non-secret evidence for product and operational
launch gates. Repository documents may describe an intended mechanism; evidence
must identify what was actually measured, where it was measured, and which
items remain externally unverified.

Do not commit:

- secrets or connection strings;
- learner emails, Account identifiers, topics, source sets, or lesson text;
- private provider payloads;
- screenshots containing personal or operationally sensitive information.

Aggregate, content-free reports may be recorded when cohort sizes are large
enough to avoid identifying a learner. Keep raw production output in the
approved operational system when a small cohort could be identifiable.

The Phase 0 aggregate report is generated with:

```sh
psql "$DATABASE_URL" -X -f scripts/product-baseline.sql
```

Before using any product, beta, or billing aggregate as launch evidence,
classify each participating account through an authorized operator session:

```sh
psql "$DATABASE_URL" -X \
  -v account_id='<account UUID from the restricted identity lookup>' \
  -v evidence_class='real_user' \
  -v reason_code='external_design_partner' \
  -v classified_by='founder' \
  -v evidence_reference='beta/cohort-01' \
  -f scripts/classify-evidence-account.sql
```

Use `founder` with `founder_or_team`, or `test` with `automated_test`,
`manual_test`, or `monitoring_probe`, for excluded traffic. Classifications are
append-only; correct a mistake by appending a new row with reason `correction`.
Absence means `unclassified`, which is excluded from every real-user gate.
Never classify by an email-domain pattern or let the browser choose a class.

Attach the command date, deployed commit, environment, schema version, and a
redacted/aggregate result or internal evidence reference to the applicable
dated record.

Current implementation evidence:

- `2026-08-11-phase-0-baseline.md`
- `2026-08-12-phase-1-research-readiness.md`
- `2026-08-11-phase-2-reliability.md`
- `2026-08-11-phase-3-source-intelligence.md`
- `2026-08-11-phase-4-activation.md`
- `2026-08-12-phase-5-learning-contract.md`
- `2026-08-12-phase-6-adaptive-today.md`
- `2026-08-12-phase-6-learning-rhythm.md`
- `2026-08-12-phase-6-capability-path.md`
- `2026-08-12-phase-7-publishing-safety.md`
- `2026-08-12-phase-7-public-growth-surface.md`
- `2026-08-12-phase-8-truth-first-marketing.md`
- `2026-08-12-phase-9-billing-foundation.md`
- `2026-08-12-automated-release-gates.md`
- `2026-08-12-production-parity-check.md`
- `2026-08-12-artifact-cleanup-safety.md`
- `2026-08-12-real-user-evidence-integrity.md`
- `2026-08-13-source-retrieval-policy.md`
- `starter-path-editorial-review.md`
- `starter-path-review-v3.json`
