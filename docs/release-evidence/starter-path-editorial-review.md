# Starter-path editorial review

Status: **awaiting founder or qualified human review**

This is the release record for `LL-405`. Automated research prepared the
candidate portfolios, but it cannot sign the human-review gate. Do not change a
catalog entry to `human_reviewed` until every required check below has a named
reviewer, date, and decision.

## Approval standard

For each path, the reviewer must:

1. confirm the outcome describes a useful capability for the selected ICP;
2. open every source and confirm it is the named official or primary resource;
3. confirm the portfolio covers the first lesson's central concepts and
   includes meaningful uncertainty or counterweight where relevant;
4. reject promotional, stale, inaccessible, or misleading sources;
5. read the sample objective and retrieval prompt for answerability;
6. run one generated sample lesson and check material claims against sources;
7. record `approve`, `revise`, or `reject`, plus a short rationale.

## Review matrix

| Candidate path | Sources | Catalog status | Reviewer | Reviewed on | Decision | Notes |
| --- | ---: | --- | --- | --- | --- | --- |
| Evaluate an AI system | 4 | founder_review_pending | — | — | pending | NIST framework/profile/AIRC portfolio |
| Build reliable RAG | 4 | founder_review_pending | — | — | pending | OpenAI/Microsoft implementation guidance plus the RAG paper and eval guidance |
| Engineer reliable AI agents | 4 | founder_review_pending | — | — | pending | Anthropic/OpenAI guidance, MCP architecture, and the ReAct paper |
| Threat-model an LLM app | 4 | founder_review_pending | — | — | pending | OWASP, NIST, MITRE, and Google security frameworks |
| Engineer model context | 4 | founder_review_pending | — | — | pending | Anthropic/OpenAI guidance, MCP architecture, and Lost in the Middle |
| Operate AI inference | 4 | founder_review_pending | — | — | pending | OpenTelemetry/OpenAI guidance plus vLLM research and TensorRT-LLM docs |

Catalog version 3 deliberately removed five broad professional-development
paths that did not serve the selected AI/software launch wedge directly. Source
URLs returned successful responses in the automated pre-review check on
2026-08-12 (the final human source and generated-lesson checks remain open).

## Release action after approval

- Update only approved entries in `web/src/streamTemplates.ts` to
  `editorialStatus: "human_reviewed"`.
- Revise or remove rejected sources and increment a template version whenever
  the learning outcome, topic, or substantive portfolio changes.
- Run the full frontend test/check suite and record the result here.
- Mark `LL-405` complete only when at least five paths are approved and their
  sample lessons have passed the claim-to-source check.

## Review packet and commands

The machine-readable review packet is
`docs/release-evidence/starter-path-review-v3.json`. It intentionally begins
with every decision set to `pending` and every human attestation set to false.

Validate that the packet is structurally complete without implying approval:

```sh
go run ./cmd/starter-review -validate-only
```

For each reviewed path, save a public-safe rendered lesson or redacted internal
evidence reference under `docs/release-evidence/starter-lessons/`, then record:

- the named reviewer and ISO review date;
- `approve`, `revise`, or `reject`;
- the lesson artifact path;
- the four required attestations;
- one entry per material claim checked, using only a short claim label, source
  URL, disposition (`supported`, `revised`, or `removed`), and concise note;
- a rationale in `notes`.

Do not copy provider payloads, source bodies, learner data, or secrets into the
packet. After revision, regenerate the lesson and repeat the claim checks.

The actual release gate is fail-closed:

```sh
go run ./cmd/starter-review
```

It fails until at least five paths are approved with complete named human
evidence. A successful structural check does not satisfy `LL-405` or `LL-512`.
