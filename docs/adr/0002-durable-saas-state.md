# ADR-0002: Postgres and S3-compatible durable state

Status: accepted

## Context

Issue lifecycle data currently spans SQLite, JSON Daily Run records, JSON
Learning History, filesystem locks, and absolute artifact paths persisted in
database rows. Those implementations cannot provide coherent transactions,
horizontal worker coordination, or a single backup and restore story.

Only one production database implementation and one production artifact
implementation are needed.

## Decision

Postgres is the concrete transactional implementation for Accounts, Personal
Sites, Newsletters, Issues, Issue Claims, Learning History, Delivery Receipts,
quotas, and webhook idempotency.

S3-compatible object storage is the concrete implementation for immutable
Dossier Artifacts. Postgres stores opaque object keys and content metadata,
never local filesystem paths.

Database migrations are embedded in the Go binary, serialized with a Postgres
advisory lock, and applied by the `migrate` role. Web and worker startup fail
when the schema is not current.

No SQLite, filesystem-state, or legacy-data adapter is retained.

## Consequences

- Transactional invariants have one locality.
- Workers coordinate through time-bounded Issue Claims.
- Postgres and the object bucket form one documented recovery set.
- Local development requires Postgres and S3-compatible object storage.
