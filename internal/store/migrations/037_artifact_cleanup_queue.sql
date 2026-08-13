CREATE TABLE artifact_cleanup_queue (
  artifact_key text PRIMARY KEY
    CHECK (char_length(artifact_key) BETWEEN 1 AND 1000),
  issue_id uuid NOT NULL,
  available_at timestamptz NOT NULL,
  claim_token uuid,
  claim_expires_at timestamptz,
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (
    (claim_token IS NULL AND claim_expires_at IS NULL) OR
    (claim_token IS NOT NULL AND claim_expires_at IS NOT NULL)
  )
);

CREATE INDEX artifact_cleanup_queue_available
  ON artifact_cleanup_queue(available_at, created_at)
  WHERE claim_token IS NULL;

COMMENT ON TABLE artifact_cleanup_queue IS
  'Durable deletion intents for immutable artifacts uploaded before their Issue transaction commits.';
