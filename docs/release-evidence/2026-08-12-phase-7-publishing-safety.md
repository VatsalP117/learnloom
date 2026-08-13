# Phase 7 publishing safety evidence — 2026-08-12

## Scope

Tasks `LL-701` through `LL-707` are implemented on schema version 27. This
evidence covers repository behavior and a clean PostgreSQL 17 database; the
five-user comprehension exit gate still requires observed users.

## State and compatibility model

- Migration 27 changes lesson states from `published/hidden` to
  `private/draft/published`.
- Legacy `hidden` rows become `private`. Existing published rows remain
  published and are backfilled with publication timestamps, avoiding an
  unexpected outage for deliberate public pages.
- New lesson rows default to `draft` at the schema level and in first, manual,
  and scheduled queue paths.
- `private` means intentionally owner-only; `draft` means owner-only and ready
  for review; `published` still requires the public-site, stream, moderation,
  and account gates before any public read succeeds.
- Publication changes retain first-review time separately from the current
  public timestamp so unpublish/re-publish remains auditable.

## Owner experience

- The publishing page presents one site choice: private workspace or public
  learning site. Search indexing remains a separate discovery control and its
  copy explicitly says it does not alter link access.
- Every generated lesson in stream history shows its content state and
  effective audience, including blocked-by-private-site and
  blocked-by-private-stream cases.
- The Dossier reader also shows the effective audience, so visibility is not
  hidden in settings.
- First publish opens an audience-review dialog that states the exact site,
  stream, and search gates and links to the authenticated exact lesson preview.
- Publishing and auto-publishing require an explicit confirmation bit at the
  backend, not merely a client dialog.
- The safe recommended future-lesson default is Draft. Auto-publish is an
  explicit alternative with a warning and a durable review timestamp.
- Owners can bulk-select generated lessons and move them to Private, Draft, or
  Published. Bulk publish is all-or-nothing, ownership checked, limited to 100
  lessons, and also requires audience confirmation.
- Unpublish changes a lesson to Draft and removes public access immediately;
  “Keep private” removes it from publication review without deleting it.

## Verification

- Clean PostgreSQL 17 schema version 27 lifecycle passed.
- A dedicated integration test proves:
  - first and future lessons default to Draft;
  - Draft cannot be loaded through the public Dossier query;
  - Publish without audience confirmation is rejected;
  - confirmed Publish is publicly readable;
  - Unpublish immediately returns public not-found;
  - auto-publish cannot be enabled without explicit confirmation;
  - cross-account publication mutation is denied.
- Product baseline SQL passed on schema version 27.
- Go store/HTTP suites passed.
- API contract, ESLint, TypeScript, production build, and focused publishing UI
  tests passed.

## Remaining Phase 7 work

The public growth surface (`LL-708`–`LL-712`) and observed five-user audience
comprehension gate remain. No repository test claims those external outcomes.
