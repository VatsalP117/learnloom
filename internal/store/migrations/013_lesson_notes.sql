CREATE TABLE lesson_notes (
  id uuid PRIMARY KEY,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  kind text NOT NULL
    CHECK (kind IN ('note', 'question', 'highlight')),
  anchor_type text NOT NULL
    CHECK (anchor_type IN ('lesson', 'claim', 'source', 'section')),
  anchor_id text NOT NULL DEFAULT ''
    CHECK (char_length(anchor_id) <= 120),
  body text NOT NULL
    CHECK (char_length(body) BETWEEN 1 AND 4000),
  quoted_text text NOT NULL DEFAULT ''
    CHECK (char_length(quoted_text) <= 1200),
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX lesson_notes_issue_timeline
  ON lesson_notes(account_id, issue_id, updated_at DESC);
