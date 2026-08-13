CREATE TABLE lesson_retrieval_responses (
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  prompt_key text NOT NULL
    CHECK (char_length(prompt_key) BETWEEN 1 AND 80),
  response_text text NOT NULL DEFAULT ''
    CHECK (char_length(response_text) <= 2000),
  skipped boolean NOT NULL DEFAULT false,
  revealed_at timestamptz,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (account_id, issue_id, prompt_key),
  CHECK (
    (skipped AND response_text = '' AND revealed_at IS NOT NULL) OR
    (NOT skipped AND char_length(response_text) BETWEEN 1 AND 2000)
  )
);

CREATE INDEX lesson_retrieval_responses_recent
  ON lesson_retrieval_responses(account_id, updated_at DESC);
