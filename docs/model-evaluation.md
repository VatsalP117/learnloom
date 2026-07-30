# Model provider evaluation

The generation pipeline depends on the narrow `dossier.Completer` contract, so
provider selection is reversible. A provider change is allowed only after:

1. rates and the per-Issue reservation are configured from the provider's
   current contract;
2. readiness returns the configured model;
3. unit, generator, and PostgreSQL lifecycle tests pass;
4. the public, content-free evaluation corpus scores at least 0.80; and
5. a staging run stays inside latency, retry, quality, and daily-spend budgets.

Run the gated provider check explicitly:

```sh
MODEL_EVAL_ENABLED=true \
MODEL_BASE_URL='https://provider.example/v1' \
MODEL_API_KEY='replace-me' \
MODEL_NAME='configured-model' \
go test ./internal/dossier -run TestConfiguredProviderEvaluationCorpus -count=1
```

The corpus tests evidence bounding, causality language, citation identity, and
important limitations. Never add learner prompts, private lessons, emails, or
licensed source bodies to the corpus. Record provider, model revision, region,
date, score, p50/p95 latency, retries, and estimated cost in the release
evidence. Product plans remain deferred until completion, seven-day return,
and cost-per-retained-learner are based on real usage.
