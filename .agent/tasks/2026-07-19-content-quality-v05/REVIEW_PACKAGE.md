# Review Package — Content Quality v0.5

## Review Range

- Base: `origin/main` (`f81303d`)
- Head: `agent/content-quality-v05`
- Diff: `git diff origin/main...HEAD`

## Intended Behavior

- Build source-grounded lessons from a curated subset of feed items.
- Fetch article text only through a bounded public-web enrichment path.
- Require a learning blueprint and fixed editorial structure.
- Reject unknown citations and incomplete practice material deterministically.
- Permit one repair response for structured or editorial contract failures.
- Offer AI-generated exploration as a per-Newsletter, default-off option.
- Keep AI Exploration visibly and structurally separate from grounded content.
- Preserve prior workspace data and Dossier rendering.

## High-Risk Review Areas

1. SSRF, redirects, DNS/IP validation, response-size limits, and fallback behavior in `src/source-enrichment.mjs`.
2. Prompt budgeting, structured parsing/repair, citation checks, and quality-score behavior in `src/pipeline.mjs` and `src/content-quality.mjs`.
3. The provenance boundary between grounded lesson content and uncited AI Exploration.
4. SQLite v1/v2-to-v3 migration correctness and Newsletter setting isolation.
5. Dashboard CSRF handling and safe rendering.
6. Dossier v1 compatibility and Dossier v2 email/Markdown escaping.

## Evidence

- 95 automated tests pass.
- Syntax checks and whitespace validation pass.
- Compose configuration validates.
- Live Command Code run completed with three enriched sources and a 92/100 deterministic score after one editor repair.

## Exclusions

- VM deployment.
- Notion integration.
- Multi-user authentication or public dashboard exposure.
- General-purpose arbitrary URL crawling.
