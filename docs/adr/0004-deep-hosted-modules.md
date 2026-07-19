# ADR-0004: Deep hosted modules

Status: accepted

## Context

The Node implementation mixes Dossier production across many callers, places
Issue transitions in both worker and persistence code, and combines host
classification, authentication, tenant scope, JSON, public rendering, static
assets, and legacy HTML in one HTTP implementation.

## Decision

The Go backend is organized around these deep modules:

- Dossier production owns Source Item selection, enrichment, Learning Blueprint,
  model stages, quality validation, provenance, and rendering.
- Issue execution owns dispatch, fair claiming, claim renewal, generation,
  persistence, Delivery Receipt progression, recovery, backoff, quotas, and
  spend circuit breaking.
- Hosted request policy owns hostname classification, Clerk authentication,
  Account scope, origin checks, CSRF, request identifiers, rate limits, and
  stable error mapping.
- Control owns authenticated Newsletter and Personal Site behavior.
- Reading owns public Personal Site, archive, Dossier, sitemap, and robots
  behavior.

The module interface is the test surface. Implementations may use internal test
seams, but shallow pass-through packages are not introduced.

## Consequences

- Authorization policy cannot drift between control handlers.
- Issue lifecycle tests cover recovery and idempotency through one interface.
- The JavaScript-to-Go migration can proceed by domain behavior rather than by
  translating files one-for-one.
