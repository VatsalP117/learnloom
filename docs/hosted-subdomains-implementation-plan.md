# Hosted Personal Subdomains — Technical Implementation Plan

Status: proposed  
Target branch: `codex/hosted-user-subdomains`  
Primary production domain: `learnloom.blog`

## 1. Outcome

Turn Learnloom from a single trusted-operator dashboard into a hosted,
multi-user product where a learner:

1. signs in with Google through Clerk;
2. claims a unique username;
3. receives a personal site at `<username>.learnloom.blog`;
4. manages private Newsletters from `app.learnloom.blog`; and
5. automatically sees every successfully generated Dossier appear on the
   personal site and in its archive.

The personal site is a first-class reading product, not a copy of the admin
dashboard. The dashboard remains the control plane. Username subdomains are the
reading plane.

## 2. Product decisions for the first release

These decisions keep the initial implementation coherent and secure.

- One Clerk user maps to one Learnloom account and one personal site.
- Authentication is required before username availability can be checked or a
  username can be claimed.
- A username can be claimed once in the MVP. Renames require operator support
  until redirects, cooldowns, and anti-squatting rules exist.
- A newly claimed site is `private` by default. The learner explicitly changes
  it to `public`.
- Generated Issues are publishable by default but inherit the site's
  visibility. A learner can hide an individual Issue without deleting it.
- Only completed, quality-gated Issues can appear on a personal site.
- Email remains available, but the hosted version initially sends only to the
  authenticated learner's verified primary email. Arbitrary recipient lists
  are not exposed in the hosted UI because they create a spam and unsubscribe
  management surface.
- The first release supports Learnloom-owned subdomains only. Custom domains
  are a later feature.
- SQLite and the single VM remain acceptable for a controlled pilot. Tenancy is
  designed so storage can later move to PostgreSQL/object storage without
  changing URLs or product concepts.
- The public site renderer remains server-rendered. This produces fast reading
  pages, meaningful metadata, predictable caching, and no requirement to run
  Clerk or the dashboard React bundle on ordinary public page views.

## 3. Domain and request topology

| Hostname | Purpose | Authentication |
| --- | --- | --- |
| `learnloom.blog` | Marketing/root landing page | Public |
| `www.learnloom.blog` | Redirect to apex | Public |
| `app.learnloom.blog` | Dashboard, onboarding, settings, authenticated API | Required except sign-in/up routes |
| `<username>.learnloom.blog` | Personal site, topic archive, Dossier reader | Public only when site is public; owner auth for private preview |
| `clerk.learnloom.blog` | Clerk Frontend API CNAME | Managed by Clerk |

The application must classify the hostname before routing the path. Paths alone
must never select a tenant.

Introduce `src/host-routing.mjs` with a pure resolver:

```text
resolveRequestHost(hostHeader, deploymentConfig) ->
  { kind: "local" }
  { kind: "apex" }
  { kind: "app" }
  { kind: "site", username }
  { kind: "rejected" }
```

Rules:

- Parse and normalize the `Host` header with the URL parser.
- Strip a valid port only through URL parsing; never split the header manually.
- Accept exactly one label before `learnloom.blog` for personal sites.
- Reject deeper hosts such as `a.b.learnloom.blog`.
- Reject Unicode ambiguity by accepting only the normalized ASCII username
  grammar.
- Maintain explicit local-development hosts.
- Do not trust `X-Forwarded-Host` from arbitrary clients. If proxy headers are
  introduced, trust them only from the configured reverse-proxy network.
- Exact infrastructure hosts take precedence over the wildcard.
- Unknown or malformed hosts return `421 Misdirected Request`.
- A syntactically valid but unclaimed username returns a normal branded `404`;
  it must not fall through to another account.

Recommended local development mapping:

```text
app.lvh.me:3000
vatsal.lvh.me:3000
```

`lvh.me` resolves subdomains to loopback and exercises real hostname routing.
Keep `localhost` support for existing tests and self-hosted compatibility.

## 4. Username policy and claiming

### 4.1 Normalization and validation

Store only a canonical lowercase ASCII username.

Proposed grammar:

```regex
^[a-z][a-z0-9-]{2,29}$
```

Additional validation:

- 3–30 characters;
- must start with a letter;
- must end with a letter or number;
- no consecutive hyphens;
- no leading/trailing whitespace;
- exact lowercase storage;
- reject reserved names.

Initial reserved names should include:

```text
admin, api, app, assets, auth, blog, clerk, dashboard, docs, help,
learnloom, mail, root, status, support, www
```

Also reserve offensive/impersonation-prone terms through an operator-managed
list. The database uniqueness constraint, not an availability endpoint, is the
final authority.

### 4.2 Claim flow

1. The user completes Google sign-in on `app.learnloom.blog`.
2. The authenticated `/onboarding` page calls `GET /api/usernames/:username`
   for advisory availability feedback.
3. The user submits `POST /api/me/site/claim`.
4. The server authenticates the Clerk session and starts
   `BEGIN IMMEDIATE`.
5. It ensures the account does not already own a site.
6. It validates and inserts the canonical username into a unique column.
7. A uniqueness conflict returns `409 username_taken`.
8. The transaction commits and the UI links to
   `https://<username>.learnloom.blog`.

The availability response must be rate-limited and may intentionally return the
same unavailable result for reserved and already-claimed names.

Do not rely on DNS changes during a claim. The wildcard is provisioned once,
so a committed database row makes the hostname live immediately.

## 5. Authentication design with Clerk

### 5.1 SDK integration

Use the current Clerk packages:

- `@clerk/react` in the Vite/React dashboard;
- `@clerk/backend` in the Node server.

Wrap the dashboard entry point in `ClerkProvider` using
`VITE_CLERK_PUBLISHABLE_KEY`. Add routed sign-in and sign-up pages under the app
host, plus signed-in/signed-out/loading boundaries. Google is the only enabled
social connection for the pilot.

Frontend build configuration:

```text
VITE_CLERK_PUBLISHABLE_KEY
```

The Vite value is embedded at build time, so staging and production should use
their corresponding Clerk instance and build artifact.

Server startup requires:

```text
CLERK_PUBLISHABLE_KEY
CLERK_SECRET_KEY
CLERK_JWT_KEY
LEARNLOOM_ROOT_DOMAIN=learnloom.blog
LEARNLOOM_APP_ORIGIN=https://app.learnloom.blog
```

The publishable key is safe to embed in the browser build. The secret key and
JWT verification key remain server-side. Prefer networkless request
verification with the Clerk JWT public key; use the secret key only for Clerk
Backend API operations that require it.

### 5.2 Server authentication

Add `src/auth.mjs` behind an injected interface so tests do not contact Clerk:

```text
authenticate(request, expectedOrigin) ->
  { accountId, clerkUserId, sessionId } | unauthenticated
```

The production implementation converts the Node request to a Fetch `Request`
and calls Clerk's `authenticateRequest()`.

For every verification:

- accept session tokens only;
- pass an explicit `authorizedParties` list;
- use `https://app.learnloom.blog` for dashboard requests;
- for a valid personal hostname, use the exact resolved origin rather than a
  broad wildcard;
- reject incomplete sessions;
- return JSON `401` for API calls and redirect browser pages to the central
  sign-in flow.

The app must never trust a Clerk user ID supplied in a form, query, path, or
custom header.

### 5.3 Authentication across subdomains

Configure the Clerk production instance on the root domain so authentication
can operate across first-level subdomains. The central sign-in experience lives
on `app.learnloom.blog`.

For a private personal-site preview, load only the minimal Clerk bootstrap
needed to establish the same-origin short-lived session on that subdomain, or
redirect through the central app and return to the personal URL. Do not place
the entire dashboard application on a user hostname.

Clerk recommends restricting Frontend API access to explicitly allowed
subdomains. Dynamic username hosts are an intentional wildcard use case, so the
risk must be controlled on our side:

- wildcard hosts always resolve to Learnloom infrastructure;
- users cannot upload or execute JavaScript;
- generated and user-entered text is always escaped;
- public pages receive a strict CSP with no third-party script execution;
- authentication verification uses the exact resolved origin;
- stale or unclaimed hosts never serve user-controlled content.

Before implementation is marked production-ready, validate the private-preview
flow with Clerk production keys on at least two claimed subdomains. Development
cookies do not reproduce all production behavior.

### 5.4 User provisioning and deletion

Create the local Learnloom account synchronously on the first authenticated
request:

```text
ensureAccount(clerkUserId)
```

This avoids making onboarding depend on eventually consistent webhooks. Store
only the Clerk identifier and Learnloom-owned profile/site fields. Fetch the
verified primary email from Clerk when email delivery is enabled; do not mirror
the full Clerk user record.

Add a signed, idempotent Clerk webhook endpoint for `user.deleted`. On receipt:

- mark the local account suspended/deleted;
- pause its Newsletters;
- make its site unavailable immediately;
- cancel pending email deliveries;
- enqueue artifact deletion according to the retention policy.

Verify the webhook signature with Clerk's official helper and keep the route
outside normal session authentication.

### 5.5 CSRF and browser security

The existing process-wide CSRF token is unsuitable for multiple users.

- Derive a per-session CSRF token with HMAC over the Clerk session ID and a
  server secret, or store a random per-session token server-side.
- Require the token on every state-changing form request.
- Require an exact `Origin: https://app.learnloom.blog` on dashboard mutations.
- Continue verifying Clerk's authorized-party claim.
- Prefer JSON APIs for new React flows and reject unsupported content types.
- Set `Secure`, `SameSite=Lax`, and narrow paths on any Learnloom-owned cookies.
- Add rate limits to sign-up-adjacent and mutation endpoints.

Update the app CSP for Clerk's documented Frontend API, image, worker,
Cloudflare challenge, and frame requirements. Keep personal-site CSP separate
and stricter:

```text
default-src 'none'
style-src 'self' 'unsafe-inline'
img-src 'self' data: https:
font-src 'self'
base-uri 'none'
form-action 'self' https://app.learnloom.blog
frame-ancestors 'none'
```

## 6. Multi-tenant data model

Advance the SQLite workspace schema from version 3 through explicit migrations.
Do not replace the existing database in place without migration tests.

### 6.1 New account and site tables

```sql
CREATE TABLE accounts (
  id TEXT PRIMARY KEY,
  clerk_user_id TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL
    CHECK (status IN ('active', 'suspended', 'deleted')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT
) STRICT;

CREATE TABLE sites (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL UNIQUE
    REFERENCES accounts(id) ON DELETE CASCADE,
  username TEXT NOT NULL COLLATE NOCASE UNIQUE,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL DEFAULT 'private'
    CHECK (visibility IN ('private', 'public')),
  claimed_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (username = lower(username))
) STRICT;
```

Use an internal random account ID rather than Clerk's ID as a general foreign
key. This decouples product data from the identity provider and prevents Clerk
identifiers from leaking into public paths.

### 6.2 Newsletter ownership and public topics

Add to `newsletters`:

```text
owner_account_id  FK accounts(id), indexed
public_slug       lowercase site-local slug
site_visible      boolean, default true
```

Create a unique index on `(owner_account_id, public_slug COLLATE NOCASE)`.
Public slugs are generated from the Newsletter name and receive a short stable
suffix only on collision. Renaming a Newsletter does not silently change its
public URL.

Every dashboard workspace method that reads or mutates user data must accept an
account scope. Examples:

```text
listNewslettersForAccount(accountId)
getNewsletterForAccount(accountId, newsletterId)
createNewsletterForAccount(accountId, input)
enqueueManualIssueForAccount(accountId, newsletterId)
setNewsletterEmailForAccount(accountId, newsletterId, settings)
getIssueForAccount(accountId, issueId)
```

The SQL statement itself must contain the owner predicate. A caller must not
load a row globally and check ownership later.

Worker-only methods remain global and are exposed through a separate interface:

```text
dispatchDue()
claimNextIssue()
claimNextDelivery()
```

Separating owner-scoped control-plane methods from worker methods makes an
accidental unscoped dashboard query easier to detect in review and tests.

### 6.3 Public Issue identity

Add to `issues`:

```text
public_id          random, stable, unique identifier
public_slug        title-derived presentation slug
publication_state CHECK ('published', 'hidden'), default 'published'
```

Use public URLs of the form:

```text
https://vatsal.learnloom.blog/d/<public_id>/<title-slug>
```

The random `public_id` is authoritative. The title slug is cosmetic; an
incorrect or missing slug redirects to the canonical URL. Do not expose the
current scheduled Issue IDs because they contain internal Newsletter IDs and
dates.

The public query must join:

```text
site -> account -> newsletter -> generated issue
```

and require all of:

- hostname username matches the site;
- account is active;
- site is public, unless the authenticated owner is previewing;
- Newsletter is site-visible;
- Issue status is `generated`;
- Issue publication state is `published`;
- both persisted artifact paths are present.

### 6.4 Legacy data migration

The existing database has Newsletters with no owner. Preserve them, but do not
adopt them automatically.

1. Add the ownership column as nullable during the migration.
2. Do not expose unowned rows from authenticated or public routes.
3. Do not add a migration command to the deprecated CLI surface.
4. If a real installation needs adoption, perform it as an audited, backed-up,
   one-time database migration with an explicit account and username mapping.
5. After pilot data is adopted or retired, a later table-rebuild migration can make
   `owner_account_id` physically `NOT NULL`.

Migration tests must start from schema versions 1, 2, and 3 and prove that
Newsletter, Issue, delivery, and artifact pointers remain intact.

## 7. Application routing and API surface

Split the current monolithic `src/dashboard.mjs` into small seams while
preserving behavior:

```text
src/server.mjs                 HTTP server and error boundary
src/host-routing.mjs           hostname classification
src/auth.mjs                   Clerk adapter and auth policy
src/control-routes.mjs         app.learnloom.blog routes
src/public-site-routes.mjs     username host routes
src/public-site-render.mjs     reading-page HTML and metadata
src/dashboard.mjs              dashboard projections/helpers during migration
```

### 7.1 App-host routes

Public:

```text
GET  /sign-in/*
GET  /sign-up/*
POST /webhooks/clerk
GET  /healthz
```

Authenticated:

```text
GET    /api/me
GET    /api/usernames/:username
POST   /api/me/site/claim
GET    /api/me/site
PATCH  /api/me/site
GET    /api/newsletters
POST   /api/newsletters
GET    /api/newsletters/:id
PATCH  /api/newsletters/:id
POST   /api/newsletters/:id/run
POST   /api/issues/:id/retry-delivery
PATCH  /api/issues/:id/publication
```

Existing form POST routes may remain temporarily for compatibility, but they
must use the same authentication and owner-scoped service methods.

Response rules:

- `401` unauthenticated;
- `403` authenticated but disallowed;
- `404` for another tenant's resource to avoid confirming its existence;
- `409` for username or slug conflicts;
- stable machine-readable error codes in JSON.

### 7.2 Personal-site routes

```text
GET /                         latest Dossiers plus topic navigation
GET /topics/:newsletterSlug  chronological topic archive
GET /d/:publicId/:titleSlug? canonical Dossier reader
GET /robots.txt              private/public indexing policy
GET /sitemap.xml             public sites only; phase after pilot
```

Private sites return a private-site page with a central sign-in/owner-preview
action. Do not return title, topic, dates, or Issue counts to anonymous users.

## 8. Rendering and publishing behavior

The canonical Dossier JSON remains the source of truth. Publishing must not
copy model output into a second content database.

Refactor `src/render.mjs` around a shared safe fragment renderer:

```text
renderDossierFragment(dossier)
renderDossierEmail(dossier, options)
renderDossierWebPage(dossier, site, newsletter, issue)
```

Requirements:

- escape every model-, feed-, and user-controlled string;
- allow only normalized HTTP(S) links;
- never execute raw HTML from Markdown;
- retain source citations, answer-key disclosure behavior, quality metadata,
  and model-output disclaimer;
- add site header, Newsletter/topic context, publication date, canonical URL,
  and previous/next navigation;
- use semantic headings and readable mobile typography;
- ensure one Dossier cannot inject content into another page or the global
  layout.

There is no publish queue. `completeIssue()` already atomically changes an Issue
to `generated`; the public query begins returning it after that commit. A
failed generation leaves the last successful Dossier as the latest visible
content.

Cache policy:

- dashboard/private responses: `Cache-Control: no-store`;
- public site home/archive: short edge cache, for example
  `public, max-age=60, stale-while-revalidate=300`;
- Dossier page: ETag derived from the immutable generation ID and a longer
  cache lifetime;
- visibility changes must purge or naturally expire cached pages within the
  stated privacy window. Until reliable active purge exists, keep the maximum
  public cache age at 60 seconds.

Public indexing remains opt-in with site visibility. Private pages send
`X-Robots-Tag: noindex, nofollow`. Public pages include canonical metadata, but
do not add a global sitemap until content-quality and abuse review is complete.

## 9. Dashboard and onboarding work

### 9.1 Authentication shell

- Add Clerk provider to `web/src/main.jsx`.
- Add sign-in, sign-up, loading, and auth-error screens.
- Replace the placeholder avatar with Clerk's user menu.
- Hide all Newsletter UI until authentication and account provisioning finish.
- Preserve the intended destination through sign-in.

### 9.2 Username onboarding

Create an onboarding gate shown when the account has no site:

- username field with normalized live preview;
- debounced advisory availability status;
- reserved/invalid explanations without exposing another user;
- explicit acknowledgement that the first release does not support self-serve
  renames;
- transactional claim button;
- success state linking to the new subdomain.

### 9.3 Site settings

Add a Personal site section:

- canonical site URL;
- display name;
- short description;
- private/public toggle with a confirmation before first publication;
- public preview/open-site action;
- explanation that hidden Issues remain in the private dashboard.

Add per-Newsletter `Show on personal site` and per-Issue `Published/Hidden`
controls.

### 9.4 Personal-site UX

The first reader release should contain:

- owner display name and description;
- latest Dossier card for each visible Newsletter;
- recent Dossiers across all topics;
- topic navigation;
- chronological archives;
- full Dossier reader with sources and retrieval practice;
- subtle Learnloom attribution;
- responsive layout and accessible keyboard/focus behavior.

Do not copy dashboard controls, generation status, email receipts, internal
errors, recipient addresses, source configuration, or learner-private settings
onto the public site.

## 10. Email integration

Extend email rendering/delivery with an optional canonical web URL:

- if the learner has claimed a site, include a clear `Read on Learnloom` link;
- public sites link directly to the Dossier URL;
- private sites link through the app sign-in flow and return to owner preview;
- the email still contains the full Dossier initially so the web feature does
  not reduce existing utility;
- attach no email address or Clerk identifier to the URL;
- use Issue/public IDs for campaign attribution rather than user PII.

For the hosted pilot, derive the permitted destination from Clerk's verified
primary email. Changing the delivery address requires changing/verifying it in
Clerk or completing a separate email-verification flow.

## 11. Security and abuse controls required before public beta

Moving from a trusted operator to arbitrary signed-in users changes several
trust assumptions beyond authentication.

### 11.1 Tenant isolation

- Every control-plane SQL query is owner-scoped.
- Add cross-tenant tests for every read and mutation endpoint.
- Public queries begin from the resolved site, never from a user-supplied owner
  ID.
- Do not serialize internal account IDs, Clerk IDs, filesystem paths, delivery
  external IDs, or provider errors to public pages.

### 11.2 Source URL SSRF

The current architecture treats feed configuration as trusted operator input.
Hosted users make it untrusted.

Before public signup:

- apply the same public-address DNS/IP validation used by article enrichment
  to initial feed requests;
- validate every redirect;
- reject loopback, link-local, private, multicast, metadata-service, and other
  non-public destinations for IPv4 and IPv6;
- defend against DNS rebinding by validating the address actually connected
  to where the runtime permits;
- bound feed response type, bytes, decompression, redirects, and timeout;
- never include secrets in source requests.

This is a launch blocker, not a later hardening item.

### 11.3 Cost and queue abuse

Enforce server-side pilot limits, configurable by environment:

```text
maximum Newsletters per account
maximum feeds per Newsletter
maximum manual runs per day
maximum generated Dossiers per account per day
maximum queued Issues per account
```

Claim and consume quota atomically when queueing an Issue. Scheduled and manual
generation must share the same spend limit. Initially make signup invite-only
or Clerk-allowlisted until generation economics are measured.

Add fair queue selection so one account cannot keep all others waiting. A
simple first policy is oldest eligible Issue while allowing at most one
generating Issue per account. Record structured quota-rejection metrics.

### 11.4 Content and account abuse

- Keep public visibility opt-in.
- Add report/contact metadata before broad discovery.
- Rate-limit username probes and public page requests at the edge.
- Reserve operator ability to suspend an account and immediately unpublish its
  site.
- Disallow arbitrary HTML, JavaScript, file uploads, and custom CSS.
- Preserve citations and the model-output disclaimer.
- Define deletion and retention behavior before accepting non-team users.

## 12. Deployment design

### 12.1 DNS and TLS

Reference deployment using Cloudflare:

```text
A/AAAA or CNAME  @       -> production ingress, proxied
A/AAAA or CNAME  app     -> production ingress, proxied
A/AAAA or CNAME  *       -> production ingress, proxied
CNAME            clerk   -> value supplied by Clerk
```

The exact `clerk` record overrides the wildcard. Follow Clerk's dashboard DNS
instructions for whether that record is proxied or DNS-only.

Cloudflare supports wildcard DNS records and its Universal SSL covers the apex
and first-level subdomains in a full zone. Use Full (strict) TLS to the origin
with either a valid wildcard origin certificate or an automated DNS-01
certificate. Never use Flexible TLS.

### 12.2 Container topology

Add a reverse-proxy/ingress service:

```text
Internet -> Cloudflare -> reverse proxy -> dashboard server
                                      \-> worker remains private
```

- Remove direct host publication of the dashboard port in hosted mode.
- Only the reverse proxy publishes 80/443.
- Forward the original `Host`.
- Add request/body/time limits.
- Keep the worker without inbound ports.
- Keep SQLite and artifacts on the durable volume.
- Back up the SQLite database, WAL-safe snapshot, and artifact tree together.
- Add readiness separate from liveness; readiness checks database access and
  required auth configuration.

The local/self-hosted mode should remain usable without Clerk. Introduce an
explicit server mode:

```text
LEARNLOOM_DEPLOYMENT_MODE=local|hosted
```

`local` preserves loopback behavior. `hosted` refuses to start without Clerk,
root-domain, app-origin, and proxy/TLS assumptions configured correctly.

### 12.3 Clerk production setup

Operator checklist:

1. Create separate Clerk development and production instances.
2. Set the production root domain to `learnloom.blog`.
3. Add Clerk's required DNS records.
4. Enable Google as the only social connection.
5. Create a production Google OAuth application with Clerk's authorized
   redirect URI and publish the consent screen for external users.
6. Configure allowed redirect URLs.
7. Configure the dynamic-subdomain/FAPI policy deliberately and document the
   risk acceptance.
8. Register and verify the deletion webhook.
9. Store production keys only in the runtime secret store.
10. Test sign-in, sign-out, token refresh, private preview, and account deletion
    on real HTTPS hosts.

## 13. Observability

Add structured logs with:

```text
request_id
route_name
host_kind
account_id (internal only)
site_id (internal only)
newsletter_id
issue_id
status_code
duration_ms
auth_outcome
quota_outcome
```

Never log session tokens, Clerk secrets, CSRF tokens, full email addresses,
model prompts, or Dossier bodies.

Initial metrics:

- sign-in success/failure;
- onboarding started/username claimed;
- active public/private sites;
- public page requests and unique coarse sessions;
- email-to-web clicks;
- manual/scheduled Issues queued, generated, failed;
- queue age by account;
- generation spend/quota rejections;
- cross-tenant authorization denials;
- webhook verification failures.

Use privacy-preserving aggregates. Do not store raw IP addresses merely to
count readers.

## 14. Test strategy

### 14.1 Unit tests

- hostname classification, malformed hosts, deeper labels, ports, and local
  development;
- username normalization, reserved names, grammar, and collision handling;
- account provisioning and one-site ownership;
- site visibility and per-Issue publication rules;
- owner-scoped workspace methods;
- public URL/canonical slug generation;
- auth adapter outcomes and exact authorized parties;
- quota accounting;
- email web-link generation;
- public/private cache and security headers.

### 14.2 Migration tests

- schema v1 -> latest;
- schema v2 -> latest;
- schema v3 -> latest;
- idempotent startup on latest schema;
- legacy adoption in one transaction;
- rollback leaves legacy data untouched;
- newer unknown schema is still rejected.

### 14.3 HTTP integration tests

For two accounts, Alice and Bob:

- Alice cannot list, open, mutate, queue, hide, or retry Bob's resources;
- cross-tenant IDs return `404`;
- Alice's hostname never renders Bob's Issue;
- an unclaimed hostname returns branded `404`;
- private sites reveal no content anonymously;
- owner private preview works after authentication;
- public site renders only generated/published content;
- title/description/model text cannot inject HTML or script;
- forged Host, Origin, auth, CSRF, and webhook requests fail;
- deleting/suspending an account unpublishes it immediately.

### 14.4 Browser tests

Use a browser automation suite against `*.lvh.me` locally and a staging domain:

- Google/Clerk test sign-in;
- username claim and conflict race;
- dashboard isolation;
- open personal subdomain;
- public/private switch;
- mobile reading layout;
- sign-out across app and personal hosts;
- private owner preview;
- email deep link return after sign-in.

### 14.5 Infrastructure tests

- wildcard DNS resolves a newly invented label;
- exact Clerk DNS record wins over wildcard;
- apex, app, and username certificates validate;
- origin rejects direct/unknown hosts;
- only ingress ports are publicly reachable;
- cached public content becomes private within the documented maximum;
- backup restore retains account/site/Newsletter/Issue/artifact consistency.

## 15. Phased implementation order

The authoritative, bounded release gate is
[the public launch checklist](public-launch-checklist.md). The phases below
retain broader implementation context and should not be treated as an
ever-growing blocker list.

### Phase 0 — Foundations

- Add deployment mode and hosted configuration validation.
- Add host resolver and host-aware route separation.
- Split control/public route modules.
- Preserve all local-mode tests.

Exit condition: local Learnloom still works, while hosted mode can safely
distinguish apex, app, and username hosts.

### Phase 1 — Accounts, tenancy, and Clerk

- Add Clerk React/backend adapters.
- Add accounts/sites schema migration.
- Add synchronous account provisioning.
- Add authenticated app shell.
- Scope all dashboard workspace methods and APIs to an account.
- Add deletion webhook.

Exit condition: two test users have isolated dashboards and no unauthenticated
dashboard route exposes data.

### Phase 2 — Username claiming

- Add username policy and reserved list.
- Add transactional claim service and onboarding UI.
- Add site settings and visibility.
- Keep pre-hosted records unowned and excluded from tenant queries. If adoption
  is ever required, handle it as an audited one-time database migration rather
  than adding to the deprecated CLI surface.

Exit condition: a signed-in user can atomically claim one stable hostname and
cannot claim or inspect another user's site settings.

### Phase 3 — Personal reading sites

- Add Newsletter public slugs and Issue public IDs/slugs.
- Add public-site queries and server renderer.
- Add home, topic archive, and Dossier reader.
- Add publication controls, canonical URLs, security headers, and cache policy.

Exit condition: a newly generated Issue appears automatically on the correct
public site, and private/hidden content is unavailable anonymously.

### Phase 4 — Email and product polish

- Add canonical web links to email.
- Add private deep-link sign-in return.
- Add onboarding and site-management polish.
- Add privacy-preserving engagement metrics.

Exit condition: email and dashboard reliably deep-link to the right Dossier
without exposing PII.

### Phase 5 — Public-beta hardening

- Close feed-fetch SSRF gaps.
- Add quotas, rate limits, fair queueing, suspension, and deletion.
- Add production DNS/TLS/ingress.
- Run browser and infrastructure suites on staging.
- Complete backup/restore and incident runbooks.

Exit condition: every launch blocker in sections 11 and 14 passes in staging.

## 16. Pull-request slicing

Keep changes reviewable:

1. Host routing and deployment-mode configuration.
2. Workspace schema migrations plus tests proving legacy rows remain unowned
   and excluded from hosted tenants.
3. Clerk adapter, authenticated app shell, and CSRF changes.
4. Owner-scoped workspace/API refactor.
5. Username claim service and onboarding UI.
6. Public IDs/slugs, public queries, and personal-site renderer.
7. Site/publication settings and email deep links.
8. SSRF, quotas, rate limits, and queue fairness.
9. Hosted deployment, DNS/TLS docs, observability, and staging checks.

No pull request should temporarily expose the existing unscoped dashboard to
the public internet. Hosted ingress is enabled only after authentication and
tenant scoping have landed.

## 17. Definition of done for the first hosted pilot

- A user signs in with Google through Clerk.
- A first authenticated request provisions exactly one local account.
- The user claims one valid, unique username transactionally.
- The user's dashboard shows only their Newsletters and Issues.
- A completed Issue appears at the claimed subdomain without manual
  publication or DNS work.
- The site can be private or public, and an Issue can be hidden.
- Email links to the canonical hosted Dossier.
- Another authenticated user cannot access any resource by guessing IDs.
- Anonymous users cannot discover private-site metadata.
- Feed and article fetching reject private-network destinations.
- Spend and queue limits are enforced server-side.
- Wildcard DNS/TLS, Clerk's exact DNS record, backups, deletion, and cache
  invalidation are verified in staging.
- `npm test` and `npm run check` pass.

## 18. External implementation references

- Clerk React quickstart:
  <https://clerk.com/docs/react/getting-started/quickstart>
- Clerk backend `authenticateRequest()`:
  <https://clerk.com/docs/reference/backend/authenticate-request>
- Clerk production domains and shared subdomain authentication:
  <https://clerk.com/docs/guides/development/deployment/production>
- Clerk subdomain allowlist:
  <https://clerk.com/docs/guides/dashboard/dns-domains/subdomain-allowlist>
- Clerk Google connection:
  <https://clerk.com/docs/guides/configure/auth-strategies/social-connections/google>
- Clerk user synchronization/webhook tradeoffs:
  <https://clerk.com/docs/guides/development/webhooks/syncing>
- Clerk CSP requirements:
  <https://clerk.com/docs/guides/secure/best-practices/csp-headers>
- Cloudflare wildcard DNS:
  <https://developers.cloudflare.com/dns/manage-dns-records/reference/wildcard-dns-records/>
- Cloudflare Universal SSL coverage:
  <https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/limitations/>
