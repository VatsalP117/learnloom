ALTER TABLE issues
  ADD COLUMN failure_code text,
  ADD COLUMN failure_category text,
  ADD COLUMN failure_stage text,
  ADD COLUMN failure_retryable boolean,
  ADD COLUMN public_error text,
  ADD COLUMN incident_id uuid,
  ADD COLUMN claim_loss_count integer NOT NULL DEFAULT 0
    CHECK (claim_loss_count >= 0);

UPDATE issues
SET
  failure_code = 'legacy_internal_error',
  failure_category = 'internal',
  failure_retryable = true,
  public_error = 'We couldn’t prepare this lesson. We’ve been notified, and you can retry now.',
  incident_id = gen_random_uuid()
WHERE status = 'failed' AND error IS NOT NULL;

CREATE TABLE issue_attempts (
  id uuid PRIMARY KEY,
  issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  attempt_number integer NOT NULL CHECK (attempt_number >= 1),
  status text NOT NULL
    CHECK (status IN ('running', 'completed', 'failed', 'abandoned')),
  started_at timestamptz NOT NULL,
  last_renewed_at timestamptz NOT NULL,
  completed_at timestamptz,
  failure_code text,
  failure_category text,
  failure_stage text,
  failure_retryable boolean,
  internal_error text,
  incident_id uuid,
  worker_id text NOT NULL DEFAULT 'unknown',
  deployment_version text NOT NULL DEFAULT 'unknown',
  model_name text NOT NULL DEFAULT 'unknown',
  pipeline_version text NOT NULL DEFAULT 'unknown'
);

CREATE INDEX issue_attempts_issue_started
  ON issue_attempts(issue_id, started_at DESC);

-- Keep rolling deployments compatible with Claims created by the previous
-- application image before this migration was applied.
INSERT INTO issue_attempts (
  id, issue_id, attempt_number, status, started_at, last_renewed_at,
  worker_id, deployment_version, model_name, pipeline_version
)
SELECT
  claim_token,
  id,
  GREATEST(1, attempt_count),
  'running',
  COALESCE(started_at, created_at),
  COALESCE(started_at, created_at),
  'rolling-deployment-backfill',
  'pre-v3',
  'unknown',
  'pre-v3'
FROM issues
WHERE status = 'generating' AND claim_token IS NOT NULL
ON CONFLICT (id) DO NOTHING;

CREATE TABLE issue_stage_attempts (
  issue_attempt_id uuid NOT NULL
    REFERENCES issue_attempts(id) ON DELETE CASCADE,
  stage text NOT NULL,
  status text NOT NULL CHECK (status IN ('completed', 'failed')),
  duration_ms bigint NOT NULL CHECK (duration_ms >= 0),
  failure_code text,
  internal_error text,
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY (issue_attempt_id, stage)
);

CREATE TABLE issue_generation_checkpoints (
  issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  fingerprint text NOT NULL,
  stage text NOT NULL,
  output text NOT NULL,
  pipeline_version text NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (issue_id, fingerprint, stage)
);

CREATE INDEX issue_generation_checkpoints_issue
  ON issue_generation_checkpoints(issue_id, updated_at DESC);
