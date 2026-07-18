# Implementation Report

## Summary

Implemented Learnloom v0.2 as a cloneable self-hosted Daily Run. The CLI now
delegates generation, persistence, reuse, locking, and delivery to one deep
module. Users can call DeepSeek or another OpenAI-compatible endpoint with a
key held only in the environment, retain local Command Code subscription use,
and deliver persisted Dossiers through Resend with durable Delivery Receipts.

Added stable profile-scoped storage, same-day reuse, failed-delivery retry
without regeneration, safe email rendering, Docker/Compose packaging, a host
systemd timer, VM documentation, environment/config examples, and an MIT
license.

## Files Changed

- `CONTEXT.md`: Daily Run, Dossier, Source Item, Learning History, and Delivery
  Receipt vocabulary
- `src/daily-run.mjs`: deep idempotent run lifecycle
- `src/provider.mjs`: Command Code, HTTP, and demo model adapters plus diagnostics
- `src/delivery.mjs`: Resend adapter and delivery diagnostics
- `src/render.mjs`: canonical Markdown and escaped email rendering
- `src/run-store.mjs`: atomic Daily Run records and overlap locks
- `src/paths.mjs`: stable application and profile-scoped paths
- `src/config.mjs`: BYO provider, delivery, profile, and storage validation
- `bin/learn.mjs`: shallow CLI entry adapter using the Daily Run module
- `Dockerfile`, `compose.yaml`, `deploy/*`: finite non-root VM runtime
- `.env.example`, `config.example.json`, `.dockerignore`: public onboarding and
  secret-safe image context
- `README.md`, `docs/*`, `LICENSE`: self-hosting and public release documentation
- `test/*`: provider, delivery, Daily Run, run-store, rendering, path, and
  regression coverage

## Commands Run

- `npm test`
- `npm run check`
- isolated `npm run demo` twice with one `LEARNLOOM_HOME`
- `node bin/learn.mjs doctor`
- `node bin/learn.mjs schedule status`
- `docker compose config --quiet`
- `docker build -t learnloom:0.2 .`
- direct non-root/read-only `docker run` demo with a named volume
- Docker artifact, canonical Dossier, Daily Run record, user, entrypoint, and
  command inspection
- `git diff --check`
- tracked-file secret-pattern and ignored-file checks

## Tests

- 32 Node test-runner tests pass.
- Syntax checks pass for every CLI, source, and test module.
- The clean-room demo generates once and reuses the same Daily Run on rerun.
- Current local Command Code diagnostics pass after the provider-aware refactor.
- Existing macOS launchd schedule remains installed and loaded.
- Compose configuration validates.
- The Docker image builds successfully from a 139 KB secret-safe context.
- The container runs as `node` (uid 1000) with a read-only root filesystem,
  persists Markdown/JSON Dossiers and the run ledger to its volume, and exits.
- No tracked environment/config secret files or key-shaped credentials were
  found.

## Deviations From Plan

- The v0.2 run ledger remains atomic JSON as planned rather than adding SQLite.
- Docker Desktop's configured credential helper hung while reading an unrelated
  registry credential. The public Node image was pulled using an empty temporary
  Docker client config, after which the image and runtime contract validated.
  Compose configuration validated, and an equivalent direct container run
  supplied runtime evidence.
- Notion remains explicitly out of scope for the next delivery-adapter slice.

## Known Risks

- Resend behavior is covered through contract tests but was not called live
  because no Resend key/domain was supplied.
- OpenAI-compatible behavior is contract-tested; the existing live integration
  remains Command Code because no direct DeepSeek key was supplied.
- Feed URLs are trusted operator configuration and can address private networks;
  this is unsuitable for untrusted multi-tenant input.
- Atomic JSON is appropriate for one scheduler/profile but is less queryable
  than the planned future SQLite adapter.
- Resend idempotency keys expire after the provider's retention window; the
  durable local Delivery Receipt remains the primary duplicate-send guard.

## Next Steps

- Configure a verified Resend domain and perform one live email smoke test.
- Make the GitHub repository public after review approval.
- Add a Notion delivery adapter that upserts by deterministic Daily Run ID.
- Add learner ratings/notes and spaced-repetition scheduling.

