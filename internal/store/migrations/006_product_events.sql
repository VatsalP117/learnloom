CREATE TABLE product_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  event_name text NOT NULL
    CHECK (event_name IN (
      'signup_completed',
      'stream_created',
      'lesson_generated',
      'lesson_opened',
      'lesson_completed',
      'review_attempted',
      'search_indexing_enabled'
    )),
  subject_type text NOT NULL
    CHECK (subject_type IN ('account', 'stream', 'lesson', 'review', 'site')),
  subject_id text NOT NULL
    CHECK (char_length(subject_id) BETWEEN 1 AND 128),
  occurred_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (account_id, event_name, subject_id)
);

CREATE INDEX product_events_funnel
  ON product_events(event_name, occurred_at DESC);

CREATE INDEX product_events_account_timeline
  ON product_events(account_id, occurred_at DESC);
