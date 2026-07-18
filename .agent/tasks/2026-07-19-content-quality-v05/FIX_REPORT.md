# Fix Report — Independent Review

## DNS Rebinding and Address Classification

- Replaced the runtime enrichment fetch path with Node HTTP/HTTPS requests whose DNS callback returns only the previously validated address.
- Preserved the original hostname for HTTP Host and TLS SNI behavior.
- Verified the connected socket address is public and equals the pinned address.
- Repeated resolution, validation, pinning, and socket verification for every redirect.
- Conservatively limited IPv6 targets to global unicast and rejected IETF special-purpose, documentation, transition, unique-local, link-local, multicast, and private IPv4-mapped ranges.
- Retained an injected fetch path only as an explicit test Adapter; production does not supply it.
- Added tests for non-global IPv6, pinned lookup behavior, and a mismatched private remote socket.
- Exercised the production transport against a real HTTPS arXiv page.

## Substantive Quality Gate

- Required at least five meaningful words and 30 plain-text characters in every lesson section.
- Required citations to appear in the lesson itself, not only in critique or practice.
- Continued validating citation identifiers across all grounded sections.
- Required distinct, substantive retrieval questions.
- Scoped retrieval-question parsing to the Retrieval practice section only.
- Required a substantive application challenge.
- Required a numbered, substantive answer for every retrieval question.
- Scoped answer parsing to the answer key, required exact number correspondence,
  and rejected question-form answers.
- Required every lesson heading exactly once and in the prescribed order.
- Derived the score from the actual structure, grounding, practice, challenge, answer, continuity, exploration, and enrichment checks.
- Added an adversarial regression that previously could receive 100/100 with one-character sections, duplicate questions, and an empty answer key.

## AI Exploration Boundary

- Removed AI Exploration and its enabled state from the final grounded editor input.
- The editor now receives source-grounded material only.
- The independently generated exploration is attached after grounded editing and is still checked for prohibited source markers.
- Added a regression assertion that synthetic exploration prose never reaches the editor prompt.

## Validation

- `npm test`: 96 passed.
- `npm run check`: passed.
- `git diff --check`: passed.
- `docker compose build`: passed.
- Production pinned HTTPS transport successfully enriched a real arXiv page.
- A live DeepSeek run correctly rejected an editor response that still omitted required sections after its one bounded repair; no invalid Dossier was persisted or delivered.
