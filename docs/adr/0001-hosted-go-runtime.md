# ADR-0001: Hosted Go runtime only

Status: accepted

## Context

Learnloom began as a local, self-hosted command-line tool. It is now a hosted
SaaS product. The compatibility launcher still exposes initialization, direct
Daily Runs, diagnostics, local scheduling, demo execution, HTTP serving, and
worker polling through one shallow interface.

The product is not live, so backward compatibility has no value.

## Decision

The production backend is implemented in Go.

The runtime exposes exactly three process roles:

- `web` serves the marketing surface, authenticated control surface, public
  Personal Sites, health checks, and Clerk webhooks;
- `worker` dispatches and advances Issues and Delivery Receipts;
- `migrate` applies database migrations and exits.

The local CLI, macOS scheduler, finite Daily Run, self-hosted mode, installation
JSON configuration, filesystem home, and deterministic production demo are
deleted.

React remains the browser implementation and is built as a static artifact
served by the Go web role.

## Consequences

- Hosted behavior is the only runtime interface and test surface.
- Configuration comes from validated environment variables.
- Docker and deployment manifests invoke explicit process roles.
- Local development uses the same Postgres and object-storage implementations
  as production.
