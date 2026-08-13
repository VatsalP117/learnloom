# Phase 7 public growth surface — implementation slice, 2026-08-12

## Implemented in this slice

- Public Dossier queries now expose only a publication-safe owner/path
  projection: owner display name, path topic, path outcome, and related public
  Dossiers. No account email, private profile data, draft content, source body,
  or learner state is added to the public projection.
- Each public Dossier now includes:
  - owner introduction;
  - outcome-oriented path context;
  - link to the complete public path;
  - up to three related Dossiers from the same public path;
  - LinkedIn, X, and email share links;
  - a contextual “Build your own path” CTA using explicit UTM and source
    Dossier attribution.
- Follow-by-email is a durable double-opt-in lifecycle: public forms require a
  same-origin submission, honeypot, valid mailbox syntax, and per-visitor rate
  limit; confirmation expires after seven days; only confirmed followers
  receive future published Dossiers; every update includes an immediate
  unsubscribe link; and retries use durable idempotent delivery records.
- The CTA truthfully leads with the topic-to-path promise and explicitly says
  that the visitor's path starts private.
- Public Dossier views, share actions, path-start clicks, and attributed account
  creation now flow through first-party routes and owner-scoped aggregates.
  Attributed activation uses the converted account already recorded at signup
  and the authoritative transactional `activation_completed` milestone; no
  client-controlled activation event or new visitor identifier is introduced.
  Repeated actions are deduplicated per Dossier, action, channel, visitor, and
  UTC day. Common crawler/preview user agents are excluded.
- The referral identifier is a random, secure, HttpOnly, SameSite=Lax cookie
  shared only across Learnloom subdomains. The database stores an HMAC
  fingerprint rather than the cookie, IP address, user agent, or visitor
  identity. Owners see only 7/30/90-day aggregate counts.
- The public “Build your own path” action now redirects to the app origin's
  actual signup route. The former apex `/sign-up` target did not serve the
  signup application and would have broken conversion.
- Existing canonical URL, article metadata, social preview, correction,
  reporting, moderation hold, account/site/stream/state gates, and noindex
  behavior remain in place.

## Safeguards verified and closed

- `LL-711`: canonical public Dossiers already emit canonical URL, Open Graph
  article metadata, Twitter metadata, a verified 1200×630 PNG social image, and
  publication-safe JSON-LD. The focused social suite passed after the state
  migration.
- `LL-712`: public reports use a server-secret fingerprint rather than storing
  or exposing raw reporter IPs; publisher corrections are escaped and
  auditable; moderation holds remove a Dossier from public queries; indexing is
  opt-in; empty/private paths stay noindex; and operator hold/clear actions
  remain in the moderation audit trail. The focused moderation, reading,
  robots, sitemap, CSP, and noindex tests passed after the state migration.
  Schema 39 additionally preserves moderation actions after an actor Account is
  erased. The least-privilege `cmd/public-hold` workflow lets a distinct active
  operator hold one exact published public Dossier from a rights-holder case;
  it requires repeated target confirmation, cannot clear a hold or read private
  content, is retry-idempotent, and records a non-PII case reference in the
  durable audit trail. The operational procedure and remaining staging drill
  are in `docs/public-source-rights-response.md`.

## Verification

- Public-growth rendering test asserts owner, outcome, related Dossier, share,
  UTM, and source-Dossier attribution output.
- Full Go test suite: passed.
- `go vet ./...`: passed.
- Frontend/API contract/lint/type/build checks: passed.
- Frontend: 21 files / 65 tests passed.
- Migration 028 adds pseudonymous, owner-scoped public growth events and
  attributed signup conversion records with daily deduplication.
- Migration 029 adds pending/confirmed/unsubscribed follower state, hashed
  confirmation and unsubscribe tokens, durable confirmation/update delivery
  claims, and confirmed-follow attribution. Subscriber email addresses are
  operational delivery data and are never included in owner analytics or
  public projections.
- Integration coverage proves duplicate suppression, attributed conversion,
  aggregate values, and cross-owner isolation when `TEST_DATABASE_URL` is
  available. Unit coverage verifies cookie flags, HMAC stability, bot
  filtering, and valid analytics periods.
- Migration 035 adds an `activated_at` milestone constrained to occur after the
  attributed signup. Publishing analytics now includes activated learners.
- `git diff --check`: passed.
- Fresh PostgreSQL 17 migration through schema 39 and the exact operator-hold
  lifecycle (public removal, owner/operator separation, unknown-target denial,
  idempotent retry, single audit action): passed on 12 August 2026.

## Deliberately not marked complete

`LL-708` and `LL-709` are complete at repository level. The Phase 7 exit gate
remains open until visibility comprehension is observed with five real users
and live shared Dossiers produce measurable visitor-to-signup conversion.
`LL-710` remains open until three to five founder-curated public paths are live
and verified. Those requirements are not weakened by the completed surface.
The rights-holder staging drill is also still required before final public
launch; repository coverage does not claim that the support operation has been
performed by a real second person.
