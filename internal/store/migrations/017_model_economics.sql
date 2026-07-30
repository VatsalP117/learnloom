ALTER TABLE issue_stage_attempts
  ADD COLUMN input_tokens integer NOT NULL DEFAULT 0
    CHECK (input_tokens >= 0),
  ADD COLUMN output_tokens integer NOT NULL DEFAULT 0
    CHECK (output_tokens >= 0),
  ADD COLUMN provider_retries integer NOT NULL DEFAULT 0
    CHECK (provider_retries >= 0),
  ADD COLUMN estimated_cost_microusd bigint NOT NULL DEFAULT 0
    CHECK (estimated_cost_microusd >= 0);

CREATE INDEX issue_stage_attempts_economics
  ON issue_stage_attempts(recorded_at)
  INCLUDE (input_tokens, output_tokens, provider_retries, estimated_cost_microusd);
