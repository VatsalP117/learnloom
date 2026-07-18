# Implementation Report — Content Quality v0.5

## Outcome

LearnLoom now turns curated feed items into a validated learning dossier through a staged content pipeline:

1. Curate a coherent theme and 3–5 source items.
2. Fetch bounded article text with public-network and redirect checks.
3. Build a learning blueprint before drafting.
4. Research, challenge, teach, and examine the topic in separate stages.
5. Optionally generate a clearly separated AI Exploration.
6. Edit the sections into a fixed learning structure.
7. Apply deterministic quality checks and allow one targeted repair.

AI Exploration is opt-in per Newsletter and disabled by default. It is stored and rendered separately from the source-grounded lesson, carries a clear disclaimer, and cannot contain source citations.

## Major Changes

- Added safe, bounded source enrichment with independent feed-summary fallback.
- Added structured contracts for curation, blueprint, and final editorial output.
- Added deterministic checks for lesson structure, citation validity, retrieval practice, application work, continuity, and the AI/source boundary.
- Added a single repair attempt for malformed structured responses and editorial quality-gate failures.
- Added Dossier schema version 2 while preserving rendering compatibility for existing Dossiers.
- Added SQLite schema version 3 with a per-Newsletter `ai_exploration_enabled` preference.
- Added dashboard controls on Newsletter creation and detail pages.
- Added separate Markdown and email presentation for AI Exploration.
- Updated configuration, architecture, dashboard, and project documentation for v0.5.

## Validation Evidence

- `npm test`: 96 tests passed.
- `npm run check`: all JavaScript syntax checks passed.
- `git diff --check`: passed.
- `docker compose config --quiet`: passed.
- `docker compose build`: built `learnloom:0.5` successfully.
- `npm run doctor`: configuration, Command Code authentication, and configured model checks passed.

## Live Command Code Exercise

A live forced run was executed with the configured Command Code provider and `deepseek-v4-pro`.

- The first run exposed that deterministic editorial validation occurred after the structured-response repair boundary.
- Validation was moved inside the editor contract so a failing draft receives one actionable editor repair.
- A regression test was added for this exact behavior.
- The second run completed after one editor retry and persisted:
  - Dossier version: 2
  - Selected sources: 3
  - Enriched sources: 3
  - Quality score: 92/100
  - Retrieval questions: 4
  - AI Exploration: disabled, as configured

The generated live artifact is intentionally excluded from Git.

After independent review, the gate was strengthened and another live run was
attempted. The editor omitted two required sections even after its one bounded
repair, so the run failed closed: no invalid Dossier was persisted or delivered.
The editor contract was then made explicit about exact, ordered headings. A
subsequent live run completed successfully with three enriched and cited sources,
four matched retrieval questions and answers, and a 100/100 deterministic score.

## Security and Trust Boundaries

- Enrichment accepts HTTP(S) only and rejects credentialed URLs.
- DNS results and redirect targets are checked against private, loopback, link-local, and reserved address ranges.
- Downloads are time-, size-, redirect-, and content-type-bounded.
- Model text remains untrusted and is escaped in email/dashboard HTML.
- Source identifiers are deterministically checked; AI Exploration cannot claim source citations.
- This remains a trusted-feed reader, not a general-purpose web crawler.

## Compatibility

- Existing schema v1 and v2 workspaces migrate forward without losing Newsletter or email settings.
- Existing Dossier v1 artifacts continue to render.
- AI Exploration remains off for existing and newly created Newsletters unless explicitly enabled.
