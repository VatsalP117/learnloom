CREATE TABLE today_focus_selections (
  account_id uuid PRIMARY KEY
    REFERENCES accounts(id) ON DELETE CASCADE,
  kind text NOT NULL
    CHECK (kind IN ('lesson', 'review', 'reentry', 'clear')),
  subject_id text NOT NULL
    CHECK (char_length(subject_id) BETWEEN 1 AND 128),
  newsletter_id uuid
    REFERENCES newsletters(id) ON DELETE CASCADE,
  reason_code text NOT NULL
    CHECK (char_length(reason_code) BETWEEN 1 AND 80),
  reason_text text NOT NULL
    CHECK (char_length(reason_text) BETWEEN 1 AND 500),
  score integer NOT NULL,
  score_components jsonb NOT NULL DEFAULT '{}'
    CHECK (jsonb_typeof(score_components) = 'object'),
  selected_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX today_focus_selection_timeline
  ON today_focus_selections(updated_at DESC);
