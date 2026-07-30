# 09 — Learning path, comprehension bank, interview narratives, and ownership

## Project-specific learning curriculum

### Module 1 — Run and identify the three roles

- **Concept:** process role and composition root.
- **Why here:** the same artifact becomes web, worker or migrate; confusing
  deployment unit with process responsibility leads to bad debugging.
- **Files:** `cmd/learnloom/main.go`, `metrics.go`, Dockerfile, Compose.
- **Prerequisite:** process, environment variable, port.
- **Plain language:** one toolbox is launched in three modes. Each mode opens
  only what it needs.
- **Deep:** trace `LoadFor(role)`, dependency graph, signal context, HTTP
  shutdown and worker drain.
- **Questions:** Which role talks to model? Who applies schema? What does web do
  on SIGTERM?
- **Exercise:** draw startup from memory; intentionally omit a required
  non-secret local variable and interpret startup error.
- **Modification:** add a non-secret release field to worker metrics.
- **Verify:** tests/build and explain why migrate does not need S3.

### Module 2 — Follow one authenticated request

- **Concept:** authentication versus tenant authorization, middleware.
- **Why:** this is the primary security boundary.
- **Files:** `HostedApp.tsx`, `api.ts`, `server.go`, `control.go`,
  `store/accounts.go`, one owner query.
- **Prerequisite:** HTTP headers/cookies/JWT/HMAC.
- **Plain:** Clerk proves who; Account ID plus SQL proves what they own; CSRF
  proves the mutation came from the intended app session.
- **Deep:** authorized party, token extractor, Account projection, exact Origin,
  session-HMAC CSRF, owner predicate and problem mapping.
- **Questions:** Why is auth not enough? Can cookie fallback mutate cross-site?
- **Exercise:** trace `/api/issues/{id}/complete` to its owner join.
- **Modification:** add a read-only profile field.
- **Verify:** write a cross-account test and identify every trust transition.

### Module 3 — Understand the relational model and migrations

- **Concept:** FKs, unique/check/partial indexes, transactions, migration ledger.
- **Why:** Postgres is state, queue and concurrency coordinator.
- **Files:** all migrations, `migrate.go`, `store.go`, `newsletters.go`.
- **Prerequisite:** SQL/ACID.
- **Plain:** constraints prevent impossible rows even if two requests race.
- **Deep:** advisory lock, Read Committed, row locks, `SKIP LOCKED`, keyset
  cursor and current-schema readiness.
- **Questions:** Why is scheduled-day index partial? What rolls back on stream
  quota?
- **Exercise:** map every ER edge; inspect the version 4/5 bug.
- **Modification:** add a harmless indexed metadata field using expand pattern.
- **Verify:** apply from empty DB, readiness, populated upgrade, query plan.

### Module 4 — Master Issue and Delivery state machines

- **Concept:** durable queue, lease, idempotency, ambiguous outcome.
- **Why:** central reliability design.
- **Files:** `store/issues.go`, `deliveries.go`, `execution/worker.go`,
  worker/integration tests.
- **Plain:** workers borrow a job for a limited time; only the matching borrower
  may finish it. Email is a second job.
- **Deep:** fairness CTE, renewal, claim-loss budget, attempts, backoff,
  artifact-before-completion, outcome unknown.
- **Questions:** Why decrement attempt on expired claim? Why not retry unknown?
- **Exercise:** simulate crash before/after provider acceptance on paper.
- **Modification:** expose oldest queued age metric.
- **Verify:** deterministic fake-clock/state transition tests.

### Module 5 — Trace source safety and evidence

- **Concept:** SSRF, DNS rebinding, normalized immutable evidence.
- **Why:** users influence URLs and content becomes model grounding.
- **Files:** `source/http.go`, `service.go`, migrations 002, tests.
- **Plain:** search suggests doors; Learnloom still checks and enters each door
  through one guarded path, then photographs exact evidence.
- **Deep:** URL credential/scheme checks, all-address public validation, pinned
  dial, redirect policy, bounds, ETag/304/staleness, discovery threshold,
  snapshot identity/freeze.
- **Questions:** Why check all DNS answers? Why reuse Issue sources on retry?
- **Exercise:** classify malicious URLs and follow a 304.
- **Modification:** add a safe supported content type.
- **Verify:** malicious redirect/DNS/body/stale tests.

### Module 6 — Understand Dossier generation

- **Concept:** staged LLM workflow, structured contract, deterministic quality.
- **Why:** central product differentiator and largest cost.
- **Files:** `dossier/generator.go`, `model.go`, `quality.go`, `render.go`.
- **Plain:** several specialized passes research/teach/challenge/edit; ordinary
  code checks the result before publication.
- **Deep:** context weighting/truncation, curation ID remap, fingerprint,
  checkpoint validation, repair, parallel exploration, editor fallback,
  citation/time/answer gates.
- **Questions:** Which stages checkpoint? What invalidates reuse? Why is model
  structured mode insufficient?
- **Exercise:** trace one source ID through curation to citation.
- **Modification:** add a quality metric without changing prompt.
- **Verify:** generator fixture plus failed editor recovery and cost estimate.

### Module 7 — Frontend state and rendering boundaries

- **Concept:** SPA routing, server DTO, local/durable state, SSR public pages.
- **Why:** two presentation architectures coexist intentionally.
- **Files:** `main.tsx`, `App.tsx`, `HostedApp.tsx`, hooks, `learningState.ts`,
  `reading.go`.
- **Plain:** private product paints in the browser; public pages arrive complete
  for readers/crawlers.
- **Deep:** Clerk gating, module cache/ETag, abort/versioning, History API,
  local monotonic progress, CSP/canonical/JSON-LD.
- **Questions:** What happens on direct `/issues/id` load? Why can review differ
  across devices?
- **Exercise:** trace first render and cache refresh.
- **Modification:** add a page with a paginated read endpoint.
- **Verify:** direct/back/forward/auth/loading/error/mobile/a11y.

### Module 8 — Operate and architect

- **Concept:** build/deploy/migrate/drain/restore/SLO/threat model/ADR.
- **Why:** ownership includes failure, not just feature work.
- **Files:** Docker/Compose/CI, chapters 04–07, existing ADRs.
- **Prerequisite:** prior modules.
- **Plain:** prove you can release, detect failure, stop cost, recover data and
  explain trade-offs.
- **Deep:** expand-contract, pool math, per-process limits, coordinated recovery,
  secret rotation, SLO/error budget and decision reversibility.
- **Exercise:** staging deploy + restore game day; write incident report.
- **Modification:** implement TD-001 and its invariant test, then one
  observability metric.
- **Verify:** another engineer can follow your runbook without undocumented
  knowledge.

## Question bank — stop before the answers

### Basic

1. What product object does the UI call a stream but backend call a Newsletter?
2. What are the three executable roles?
3. Which store owns mutable state and which owns immutable lessons?
4. Who owns identity?
5. Which hostname serves the authenticated app?
6. What is an Issue?
7. Why does a generated Issue store both artifact key and checksum?
8. What is the difference between public visibility and search indexing?
9. Which frontend state is durable and which remains browser-only?
10. Where are migrations embedded?
11. What creates the first Issue?
12. Does web call the model?

### Intermediate

13. Trace bearer token to Account ID.
14. List every mutation request protection.
15. How does SQL prevent a second scheduled lesson for one local day?
16. Why does `CreateNewsletter` lock Account?
17. How is worker fairness approximated?
18. What happens when Issue lease renewal transiently fails?
19. When can hybrid mode avoid SearXNG?
20. Why can a retry make zero network source requests?
21. What enters the generation fingerprint?
22. Which model failures retry at HTTP level?
23. What transaction creates a Delivery Receipt?
24. Why is public Dossier URL based on public UUID and slug?
25. How do ETags and local caches differ?
26. What does Account suspension do to in-flight work?
27. Why is `unknown` email terminal?
28. Which database query enforces progress ownership?

### Advanced/trade-off

29. Defend Postgres queue versus SQS at current scale.
30. What is the first database query likely to degrade with large Issue volume?
31. What changes when adding a second worker replica?
32. Why does current modular monolith not imply one process?
33. Which invariants are DB constraints versus transition SQL?
34. How could source JSON and source specs diverge?
35. Why is a separate public server renderer reasonable?
36. What would RLS add and complicate?
37. When should `PipelineVersion` change?
38. Could old image safely roll back after migration 005?
39. What are the two-store restore hazards?
40. What is accidental versus essential complexity in the model pipeline?

### Debugging / “what if”

41. Migrate succeeds, but both web and worker say schema expected 4. Cause?
42. A lesson stays generating after worker crash. What transitions it?
43. Resend accepted email but DB commit timed out. What should happen?
44. A feed returns 304 but snapshots are 40 days old. What happens?
45. An attacker submits a hostname resolving to public and private IPs.
46. User B guesses User A’s Issue UUID.
47. Site is public but indexing false. What do robots/sitemap/headers do?
48. A stream is paused immediately after its manual Issue is created.
49. Model editor breaks answer key but teacher/examiner output is valid.
50. CSRF secret changes during rolling deployment.
51. Frontend works via Vite demo but production auth loops.
52. Worker count doubles but model account permits four concurrent calls.
53. S3 upload succeeds and DB completion fails.
54. A Clerk deletion webhook is delivered twice.
55. Browser local progress says 80, server says 30.

### Architecture/security/deployment

56. Draw all trust boundaries.
57. Identify all unauthenticated endpoints and their exposure.
58. What personal data goes to the model and Resend?
59. How would you test every owner boundary?
60. Which secrets require rebuild versus restart?
61. Why can’t a database-only backup fully restore?
62. What must be checked before an image rollback?
63. Which CI step should have caught migration drift but does not?
64. What observability question is impossible today?
65. When is a separate queue justified?

## Answers

1. Newsletter. 2. web/worker/migrate. 3. Postgres/S3. 4. Clerk; Accounts is a
projection. 5. `app.<root>`. 6. One scheduled/manual lesson work unit and its
state projection. 7. Location plus integrity/cache identity. 8. Public permits
link access; indexing separately opts into crawler discovery. 9. lesson
progress is DB-durable; review state is localStorage-only. 10. `store/migrations`
via `go:embed`. 11. the same transaction as stream creation. 12. no, worker.

13. Clerk middleware verifies subject/session; `EnsureAccount` loads/inserts
projection; session context passes Account ID to owner SQL. 14. auth, exact
Origin, HMAC CSRF, JSON content type/body bounds/strict decode. 15. partial
unique `(newsletter_id, scheduled_local_date)` where scheduled. 16. serialize
status/quota with concurrent creates. 17. one candidate/account, below
account limits, ordered least-recent account then oldest work. 18. renewal
retries until current expiry; claim loss/expiry cancels. 19. when selected
provided evidence reaches target/hard sufficiency per mode. 20. frozen
`issue_sources` load snapshots. 21. pipeline/model, key stream settings,
history, prepared evidence. 22. request timeout, 429, 5xx; not ordinary 4xx or
truncated output. 23. Issue completion. 24. stable opaque public identity plus
readable canonical slug. 25. HTTP validation versus in-process browser/S3
caches. 26. disables streams, cancels queued/generating Issues and deliveries.
27. duplicate mail risk after ambiguous provider acceptance. 28. progress SQL
joins Issue→Newsletter and matches Account.

29. atomic state+queue and low ops outweigh broker scale now; revisit on
measured contention/fan-out. 30. claim account-activity aggregation. 31. DB and
model pools/concurrency multiply; claims remain safe; global config is not
truly global. 32. one code/deploy model can run separate processes. 33. see
chapter 03; status checks are DB, permitted edges mostly SQL. 34. bypass
reconciliation write or failed future migration. 35. crawlable HTML without a
second Node server and interactive SPA overhead. 36. defense-in-depth tenant
policy, but session/account context and migration complexity. 37. when old
checkpoint meaning/contract/prompt is unsafe despite same inputs. 38. only if
old code tolerates column/schema; current old expected version check rejects 5.
39. inconsistent point-in-time keys, missing objects, email replay. 40.
pedagogy/contracts are essential; file size/prompt plumbing/duplicated DTOs may
be accidental.

41. confirmed `currentSchemaVersion` not updated for migration 005. 42. cycle
`RecoverExpiredClaims` after lease expiry. 43. `unknown`, reconcile manually.
44. evidence rejected as stale. 45. fetch rejected because every address must
be public. 46. owner-scoped SQL returns not found. 47. accessible but
`noindex`, no advertised/filled sitemap. 48. manual Issue still claimable.
49. quality fallback can preserve valid practice/drafts. 50. mutations routed
across versions intermittently fail until profile/token refresh; coordinate.
51. check baked Vite key/domain, Clerk origins/callback/CSP, exact host. 52.
per-process model semaphore permits up to eight; needs external/global budget.
53. object delete attempted; possible orphan if deletion fails. 54. processed
event ID returns 204 without repeat; failed event deletes row for retry. 55.
monotonic client merge retains 80; reset cannot propagate.

56. browser/edge/web, Clerk webhook, DB, S3, public web, search, model, Resend.
57. apex/public sites/static, health/ready/metrics, Clerk webhook (signed).
58. learner goal/level/topic/history/source evidence to model; verified email
and lesson to Resend. 59. Account A/B fixtures, enumerate every store/API
operation, assert B receives no A data/change. 60. `VITE_*` rebuild; server
secrets restart/rollout. 61. Dossier bytes are in S3. 62. migration/schema
forward compatibility and delivery state. 63. missing all-migrations
`Store.Ready()` assertion. 64. historical SLO/error/queue/provider trends,
among others. 65. proven DB contention, throughput/fan-out/decoupling need that
outweighs new operations/consistency complexity.

## Interview-style explanations

### Frontend

Emphasize React 19/Vite/TypeScript, hostname surface selection, Clerk state
gating, custom History router, module cache + ETag, server pagination, durable/
local progress merge, lazy chunks, responsive asset warming and public SSR
boundary. Expect: why no React Query/router, cache invalidation, error
boundaries, accessibility, type safety, XSS/CSP. Be candid that custom state is
small but weakly typed and review is local-only.

### Backend

Emphasize standard-library HTTP, centralized auth/CSRF/host policy, concrete
pgx transactional store, claims/fairness/leases, source SSRF boundary, staged
generation/checkpoints, artifact-before-completion and separate delivery.
Expect: isolation levels, idempotency, ambiguous outcomes, retries, ownership,
why Postgres queue, transaction boundaries.

### Full stack/product engineering

Tell one vertical story: learner creates stream → atomic first Issue → worker
freezes sources/generates/stores → SPA reads/progresses → public site/email.
Explain product decisions (grounding, duration, continuity, indexing consent)
and their code invariants. Admit implicit API DTO debt and operational gaps.

### System design

Frame as modular monolith with web/worker separation, Postgres+S3, external
identity/model/email/search. Discuss failure domains, horizontal claims,
capacity limits, recovery set, scaling sequence and when *not* to add a broker.
Expect RPO/RTO, multi-region, model cost, queue hotspots, CDN and tenant safety.

### DevOps

Explain multi-stage distroless build, one image/three roles, migrate-first,
worker drain, Traefik host rules, private networks, R2, health/readiness/metrics
and CI. Clearly state deploy trigger/backups/alerts are external/unknown and
current schema mismatch is a release blocker.

### Security review

Lead with Clerk authentication separate from owner SQL, exact Origin/session
CSRF, signed/idempotent webhook, source DNS pinning, restrictive CSP/escaping,
secret redaction and nonroot image. Then risks: no RLS/cross-tenant matrix,
public metrics, limited rate limits, retention gap, external edge/IAM settings.

### Honest role wording

Say: “I own and am now validating a system developed with substantial AI
assistance. The repository records these decisions; where historical motivation
was not explicit, I reconstructed and tested it rather than claiming I
personally made it.” Then explain what you can now defend, what you would change,
and the evidence. Do not claim scale, uptime, restore success or security audit
that has not occurred.

## Final project ownership checklist

### Architecture and code

- [ ] I can draw context/container/deployment diagrams from memory.
- [ ] I can explain why this is a modular monolith with separate processes.
- [ ] I can locate each composition/runtime entry point.
- [ ] I can explain every `internal` package and its dependency direction.
- [ ] I can trace sign-in, create, scheduled generation, read, publish, email,
      progress, retry and deletion end to end.
- [ ] I can distinguish frontend presentation policy from backend invariants.

### Data and reliability

- [ ] I can draw the ER model and both state machines.
- [ ] I can name every table, owner, key constraint and retention behavior.
- [ ] I can explain each transaction boundary and ownership predicate.
- [ ] I can explain claim, renewal, recovery, attempt and checkpoint semantics.
- [ ] I can explain why frozen evidence and unknown delivery exist.
- [ ] I can predict crash behavior at each external side effect.

### Security

- [ ] I can trace Clerk token verification and webhook lifecycle separately.
- [ ] I can explain auth versus authorization and test cross-tenant isolation.
- [ ] I can explain Origin/CSRF/CSP/SSRF/escaping/rate-limit boundaries.
- [ ] I know what personal data each provider receives and how deletion works.
- [ ] I can identify secrets, browser-public config and rotation constraints.

### Delivery and operations

- [ ] I can set up from a clean machine without an AI agent.
- [ ] I can build/test/migrate/start/verify all roles.
- [ ] I can explain and execute worker drain and generation pause.
- [ ] I know the actual production topology, region, resource and owners.
- [ ] I have personally completed a schema-compatible deploy and rollback drill.
- [ ] I have personally restored Postgres+artifacts and recorded RPO/RTO.
- [ ] I can find logs/metrics/alerts and debug every playbook in chapter 05.
- [ ] I have resolved and regression-tested the schema version blocker.

### Architecture ownership

- [ ] I can explain all recorded/reconstructed ADRs and confidence.
- [ ] I can defend good choices without pretending alternatives are impossible.
- [ ] I can name current risks/debt in priority order.
- [ ] I know what evidence would trigger Postgres/broker/CDN/auth changes.
- [ ] I can write an ADR with context, alternatives, consequences and revisit
      conditions.
- [ ] I can review an AI change for distant invariants (migration version,
      owner predicates, DTOs, config/Compose/docs, retry semantics).
- [ ] I can safely add a field, endpoint, page, job, provider and business rule.
- [ ] I can say “unknown” when repository evidence ends and know how to resolve it.
