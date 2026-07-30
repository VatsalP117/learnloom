CREATE TABLE lesson_feedback (
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  difficulty text
    CHECK (difficulty IN ('too_basic', 'right', 'too_advanced')),
  relevance text
    CHECK (relevance IN ('not_relevant', 'somewhat_relevant', 'very_relevant')),
  recall_confidence text
    CHECK (recall_confidence IN ('low', 'medium', 'high')),
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (account_id, issue_id),
  CHECK (
    difficulty IS NOT NULL OR
    relevance IS NOT NULL OR
    recall_confidence IS NOT NULL
  )
);

CREATE INDEX lesson_feedback_recent
  ON lesson_feedback(account_id, updated_at DESC);
