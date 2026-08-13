CREATE TABLE source_retrieval_policy_events (
  id uuid PRIMARY KEY,
  scope text NOT NULL CHECK (scope IN ('exact_url', 'registrable_domain')),
  value text NOT NULL CHECK (char_length(value) BETWEEN 1 AND 2048),
  action text NOT NULL CHECK (action IN ('block', 'unblock')),
  case_reference text NOT NULL CHECK (char_length(case_reference) BETWEEN 3 AND 80),
  reason text NOT NULL CHECK (char_length(reason) BETWEEN 10 AND 800),
  actor_account_id uuid REFERENCES accounts(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL
);

CREATE INDEX source_retrieval_policy_events_current
  ON source_retrieval_policy_events(scope, value, created_at DESC, id DESC);

CREATE VIEW current_source_retrieval_policy AS
SELECT DISTINCT ON (scope, value)
  id, scope, value, action, case_reference, actor_account_id, created_at
FROM source_retrieval_policy_events
ORDER BY scope, value, created_at DESC, id DESC;

COMMENT ON TABLE source_retrieval_policy_events IS
  'Append-only operator policy for verified source retrieval blocks and reversals; reasons remain audit-only and are not exposed to learners.';
