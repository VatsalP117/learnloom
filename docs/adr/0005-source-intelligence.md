# ADR-0005: Catalog-first autonomous source intelligence

Status: accepted

## Context

Learners should be able to create a daily learning stream from a topic without
first researching feeds and article URLs. Learner-provided and automatically
discovered sources still need identical URL safety, extraction, snapshot, and
citation behavior.

Paid search APIs would add recurring per-query cost. General browser agents
would add non-deterministic tool use and a wider security boundary.

## Decision

- Persist three source policies: `discovered`, `provided`, and `hybrid`.
- Use a self-hosted SearXNG JSON endpoint as the discovery adapter.
- Search only when the active catalog lacks the configured evidence target.
- Treat search results as candidates, never evidence.
- Rank and diversify candidates deterministically, resolve only a bounded top
  set, and activate only successfully extracted sources.
- Fetch every source through the native bounded, DNS-pinned HTTP path.
- Persist immutable normalized snapshots and freeze the exact ordered snapshot
  set on each Issue before model generation.
- Reuse frozen Issue evidence on retry without search or network access.
- Keep browser automation, authenticated scraping, CAPTCHA/paywall bypass, and
  paid search providers out of this decision.

## Consequences

Discovery can be self-hosted and disabled independently. Upstream engine
throttling is an expected degraded state, so a sufficient hybrid catalog may
continue without search. Discovered mode never generates when grounded
evidence remains below the hard minimum.

SearXNG adds an operator-managed service and Valkey cache. Both run only under
the Compose `discovery` profile and use pinned image versions.
