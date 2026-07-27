CREATE TABLE lesson_progress (
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  progress smallint NOT NULL DEFAULT 0
    CHECK (progress BETWEEN 0 AND 100),
  completed_at timestamptz,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (account_id, issue_id)
);

CREATE INDEX lesson_progress_recent
  ON lesson_progress(account_id, updated_at DESC);
