# Production parity check — 2026-08-12

Read-only checks were performed against the public service. No deployment or
provider configuration was changed.

## Proven live

- `https://learnloom.blog/` returns HTTP 200 over TLS.
- `https://www.learnloom.blog/` returns HTTP 308 to the apex.
- `https://app.learnloom.blog/healthz` returns HTTP 200 and `{"status":"ok"}`.
- `https://app.learnloom.blog/readyz` returns HTTP 200 and
  `{"status":"ready"}`.
- Responses pass through Cloudflare, use `cache-control: no-store`, and include
  the application CSP, permissions policy, referrer policy, request ID, and
  `X-Content-Type-Options`.

## Contradictions / open parity gate

Production is not running the current roadmap worktree:

- live entry asset: `assets/index-BFd8EMkG.js`;
- live marketing asset: `assets/MarketingLanding-DP1_Qjyu.js`;
- live marketing contains the old “trusted sources” and fictional Urban
  Systems material;
- it does not contain the current topic-first headline, canonical AI-evaluation
  example, or design-partner pricing markers;
- current local production builds produce different content-addressed assets.

No release/version response header was present, so the exact deployed commit
cannot be proven from the service. The repository now emits
`X-Learnloom-Release` when `LEARNLOOM_RELEASE_VERSION` is configured to a real
value; the next deployment must set it to the immutable git SHA.

`Strict-Transport-Security` was absent from the sampled live responses. The
repository now emits one-year HSTS with `includeSubDomains` in production. Edge
behavior, preload suitability, and wildcard renewal still require staging/live
verification before the HSTS checklist item can close.

The repository now includes `cmd/release-verify`, a read-only fail-closed
verifier for immutable release identity, HSTS, cache policy, health/readiness,
app indexing policy, canonical `www` redirect, and current-versus-legacy
marketing markers. Its HTTP behavior is covered by isolated end-to-end tests.
A passing run against staging/live is still required; repository tests are not
deployment evidence.

Configuration validation now also refuses every staging or production runtime
role when `LEARNLOOM_RELEASE_VERSION` is missing, `unknown`, a branch/tag, an
abbreviated SHA, uppercase, or otherwise not a full immutable 40- or
64-character lowercase hexadecimal revision. This prevents a new unverifiable
deployment; development retains the `unknown` default.

## Required next production action

Stage the reviewed migration-38+ build with an immutable release SHA, apply all
migrations, run the full staging checklist (including Paddle sandbox), then
deploy only after approval. After deployment, verify the release header, asset
markers, HSTS, health/readiness, cache purge, and repository/live feature parity.

Therefore `LL-001`, `LL-002`, and the final live-parity launch requirement remain
open. Availability is not parity.
