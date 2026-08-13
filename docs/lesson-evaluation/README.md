# Learnloom lesson-evaluation protocol

`lesson-eval-v1` is the versioned prelaunch corpus for Dossier quality. It
contains one passing and one failing case for each launch-critical dimension:

- usefulness for the learner's stated outcome;
- continuity with prior learning;
- difficulty fit;
- unsupported factual or causal claims;
- semantic redundancy; and
- final rendered reading-time fit.

The corpus lives at
`internal/dossier/testdata/lesson-evaluation-v1.json`. Its labels are
product-contract labels, not human editorial approval. Human review remains a
separate release gate under `LL-512`.

## Local structural gate

```bash
go test ./internal/dossier -run TestLessonEvaluationCorpus -count=1
```

This always runs. It rejects missing dimensions, one-sided dimensions,
duplicate IDs, incomplete cases, or an invalid label status.

## Configured-model gate

```bash
MODEL_EVAL_ENABLED=true \
MODEL_BASE_URL='https://provider.example/v1' \
MODEL_API_KEY='...' \
MODEL_NAME='...' \
go test ./internal/dossier -run TestConfiguredProviderLessonQualityCorpus -count=1
```

The configured model must return structured judgments matching at least 80% of
the corpus. This gate is intentionally separate from ordinary unit tests so it
cannot spend model budget accidentally.

## Expansion rules

- Add production failures; do not edit an old case merely to make a model pass.
- Preserve both passing and failing cases per dimension.
- Increment the corpus version for substantive rubric or label changes.
- Record human adjudication separately rather than changing
  `labelStatus` to `human_reviewed` without a named reviewer and date.
