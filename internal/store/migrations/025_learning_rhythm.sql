ALTER TABLE newsletters
  ADD COLUMN rhythm_mode text NOT NULL DEFAULT 'daily'
    CHECK (rhythm_mode IN ('evidence_led', 'daily', 'selected_weekdays', 'weekly_synthesis')),
  ADD COLUMN selected_weekdays smallint[] NOT NULL DEFAULT ARRAY[1, 2, 3, 4, 5]::smallint[],
  ADD COLUMN effective_rhythm_mode text NOT NULL DEFAULT 'daily'
    CHECK (effective_rhythm_mode IN ('evidence_led', 'daily', 'selected_weekdays', 'weekly_synthesis')),
  ADD COLUMN auto_throttle_enabled boolean NOT NULL DEFAULT true,
  ADD COLUMN unopened_lesson_limit smallint NOT NULL DEFAULT 3
    CHECK (unopened_lesson_limit BETWEEN 1 AND 20),
  ADD COLUMN rhythm_reason text NOT NULL DEFAULT '',
  ADD COLUMN rhythm_throttled_at timestamptz,
  ADD CONSTRAINT newsletters_selected_weekdays_check CHECK (
    cardinality(selected_weekdays) BETWEEN 1 AND 7
    AND selected_weekdays <@ ARRAY[1, 2, 3, 4, 5, 6, 7]::smallint[]
  );

ALTER TABLE issues
  ADD COLUMN requested_lesson_type text
    CHECK (requested_lesson_type IS NULL OR requested_lesson_type IN (
      'foundation', 'update', 'deep_dive', 'synthesis', 'application', 'review'
    ));

CREATE TABLE rhythm_decisions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  newsletter_id uuid NOT NULL REFERENCES newsletters(id) ON DELETE CASCADE,
  issue_id uuid REFERENCES issues(id) ON DELETE CASCADE,
  decision text NOT NULL CHECK (decision IN ('configure', 'dispatch', 'defer', 'throttle', 'recover')),
  desired_mode text NOT NULL
    CHECK (desired_mode IN ('evidence_led', 'daily', 'selected_weekdays', 'weekly_synthesis')),
  effective_mode text NOT NULL
    CHECK (effective_mode IN ('evidence_led', 'daily', 'selected_weekdays', 'weekly_synthesis')),
  unopened_count integer NOT NULL DEFAULT 0 CHECK (unopened_count >= 0),
  reason text NOT NULL,
  decided_at timestamptz NOT NULL
);

CREATE INDEX rhythm_decisions_newsletter_recent
  ON rhythm_decisions(newsletter_id, decided_at DESC, id DESC);
