CREATE TABLE review_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  prompt_key text NOT NULL
    CHECK (char_length(prompt_key) BETWEEN 1 AND 80),
  prompt text NOT NULL
    CHECK (char_length(prompt) BETWEEN 1 AND 1000),
  answer_rubric text NOT NULL
    CHECK (char_length(answer_rubric) BETWEEN 1 AND 4000),
  corrective_explanation text NOT NULL
    CHECK (char_length(corrective_explanation) BETWEEN 1 AND 4000),
  objective text NOT NULL
    CHECK (char_length(objective) BETWEEN 1 AND 1000),
  scheduler_version smallint NOT NULL DEFAULT 1
    CHECK (scheduler_version = 1),
  stage smallint NOT NULL DEFAULT 0
    CHECK (stage BETWEEN 0 AND 4),
  due_at timestamptz,
  last_reviewed_at timestamptz,
  retired_at timestamptz,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  UNIQUE (issue_id, prompt_key)
);

CREATE INDEX review_items_due
  ON review_items(account_id, due_at)
  WHERE due_at IS NOT NULL AND retired_at IS NULL;

CREATE TABLE review_attempts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  review_item_id uuid NOT NULL
    REFERENCES review_items(id) ON DELETE CASCADE,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  idempotency_key uuid NOT NULL,
  assessment text NOT NULL
    CHECK (assessment IN ('needs_work', 'partial', 'solid')),
  previous_stage smallint NOT NULL,
  next_stage smallint NOT NULL,
  previous_due_at timestamptz,
  next_due_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL,
  UNIQUE (account_id, idempotency_key)
);

CREATE INDEX review_attempts_item_timeline
  ON review_attempts(review_item_id, created_at DESC);
