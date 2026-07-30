# Learnloom technical ownership handbook

**Purpose:** architecture record, onboarding guide, operations manual, learning
curriculum, and evidence index for the repository at commit `e89ff26`.

## Why this is a documentation set

The repository has several different reading modes: a new engineer needs a
short system model; an operator needs lookup tables and runbooks; an owner
needs deep workflow traces and reconstructed history. One very large document
would make those jobs compete. This handbook therefore uses interconnected
Markdown chapters, with Mermaid diagrams plus equivalent prose.

All statements use these evidence labels:

- **Confirmed** — directly established by current code, configuration, tests,
  or a recorded ADR.
- **Strong inference** — multiple repository facts support the conclusion, but
  the motivation is not explicitly recorded.
- **Possible interpretation** — plausible but not adequately established.
- **Unknown** — the repository cannot answer the question.
- **Contradictory evidence** — current artifacts disagree.

No secret values were inspected or reproduced. Paths are repository-relative.

## Suggested reading order

### One-hour orientation

1. [Architecture and repository tour](01-architecture-and-repository.md):
   “two-minute explanation,” key concepts, diagrams, and startup.
2. [Journeys and API contracts](02-journeys-and-apis.md): create a stream,
   generate a lesson, read it, and publish it.
3. [Domain and data](03-domain-and-data.md): the Issue, Delivery, and evidence
   state machines.
4. [Risks and unknowns](07-quality-risks-and-unknowns.md): read the executive
   findings and highest-priority debt.

### One-day understanding

Read chapters 01–07 in order, then use the API and configuration tables as
references while tracing one real request in an IDE.

### One-week deep study

Complete the exercises in [Learning and ownership](09-learning-and-ownership.md),
run the system locally using chapter 08, and trace one SQL transaction, one
worker cycle, one browser journey, and one deployment.

### Full mastery

Work through the full question bank without its answers; perform a backup
restore rehearsal in an isolated environment; write one new ADR; and make one
small end-to-end feature using the safe-change guides.

## Handbook map

| Chapter | Scope | Priority |
|---|---|---|
| [01](01-architecture-and-repository.md) | Executive overview, repository map, system/context/component/deployment boundaries, entry points and startup | **Read first** |
| [02](02-journeys-and-apis.md) | End-to-end user journeys, frontend/backend architecture, complete HTTP contract catalogue | **Essential** |
| [03](03-domain-and-data.md) | Domain glossary, invariants, schema, migrations, transactions, ER diagram | **Essential** |
| [04](04-security-and-configuration.md) | Authentication, authorization, trust boundaries, security review, complete environment reference | **Security critical** |
| [05](05-integrations-deployment-and-operations.md) | Dependencies, external services, Docker/Dokploy, CI/CD, operations and incident playbooks | **Operationally critical** |
| [06](06-testing-resilience-and-observability.md) | Testing map, errors, retries, metrics, performance and scale | **Important** |
| [07](07-quality-risks-and-unknowns.md) | Maintainability assessment, debt register, contradictions, unknowns | **Important** |
| [08](08-change-guides-and-local-development.md) | Safe modification recipes and clean-machine setup | **Reference** |
| [09](09-learning-and-ownership.md) | Project-specific curriculum, question bank/answers, interview narratives, future ADRs, ownership checklist | **Deep dive** |
| [10](10-adr-catalogue.md) | Recorded and reconstructed architectural decisions | **Deep dive** |
| [11](11-coverage-report.md) | Investigation coverage, evidence limitations, final self-review | **Reference** |

## The first ten concepts to internalize

1. **One artifact, three roles.** `/learnloom web`, `worker`, and `migrate`
   are the same Go binary with different dependency graphs
   (`cmd/learnloom/main.go → main(), runWeb(), runWorker(), runMigrate()`).
2. **A modular monolith, not microservices.** Web and worker are separate
   processes but share one codebase, domain model, and Postgres schema
   (`docs/adr/0001-hosted-go-runtime.md`, `internal/*`).
3. **Postgres owns mutable truth.** Scheduling, claims, progress, quotas,
   identity projections, and delivery receipts are relational state
   (`internal/store/migrations/001_initial.sql` through `005_*`).
4. **S3 owns immutable lesson artifacts.** Postgres stores an opaque key and
   checksum; the object contains canonical Dossier JSON plus rendered Markdown
   and HTML (`internal/artifact/store.go → Put(), Get()`).
5. **An Issue is the unit of lesson work.** It is queued, claimed, generated or
   failed, and separately delivered (`internal/domain/domain.go → Issue`,
   `internal/store/issues.go`).
6. **Evidence is frozen before model work.** Network/search results become
   immutable snapshots and ordered `issue_sources`; retry does not silently
   change the lesson’s evidence (`internal/source/service.go → PrepareIssue()`).
7. **Generation is a pipeline, not one prompt.** Curator, blueprint,
   researcher, skeptic, teacher, examiner, optional exploration, and editor
   stages are validated and partly checkpointed
   (`internal/dossier/generator.go → Generate()`).
8. **Authentication and authorization are separate.** Clerk proves a session;
   every owner-facing SQL query must also constrain `owner_account_id`
   (`internal/httpapp/server.go → handleAuthenticated()`, `internal/store/*`).
9. **Hostnames are part of routing and tenancy.** Apex is marketing, `app` is
   authenticated control, and `<username>` is public reading
   (`internal/httpapp/host.go → ClassifyHost()`, `server.go → ServeHTTP()`).
10. **Claims make retries safe enough to scale workers horizontally.** Expiring
    tokens, `FOR UPDATE ... SKIP LOCKED`, attempt records, and conservative
    unknown delivery outcomes are the concurrency core
    (`internal/store/issues.go → ClaimNextIssue(), RecoverExpiredClaims()`).
