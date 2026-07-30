CREATE TABLE account_notification_preferences (
  account_id uuid PRIMARY KEY
    REFERENCES accounts(id) ON DELETE CASCADE,
  weekly_recap boolean NOT NULL DEFAULT false,
  reentry_reminder boolean NOT NULL DEFAULT true,
  time_zone text NOT NULL DEFAULT 'UTC'
    CHECK (char_length(time_zone) BETWEEN 1 AND 80),
  updated_at timestamptz NOT NULL
);

CREATE TABLE weekly_recaps (
  id uuid PRIMARY KEY,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  week_start date NOT NULL,
  payload jsonb NOT NULL
    CHECK (jsonb_typeof(payload) = 'object'),
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN (
      'pending', 'delivering', 'delivered', 'failed', 'cancelled', 'unknown'
    )),
  attempt_count integer NOT NULL DEFAULT 0
    CHECK (attempt_count >= 0),
  available_at timestamptz NOT NULL,
  claim_token uuid,
  claim_expires_at timestamptz,
  external_id text,
  error text,
  created_at timestamptz NOT NULL,
  started_at timestamptz,
  completed_at timestamptz,
  updated_at timestamptz NOT NULL,
  UNIQUE (account_id, week_start),
  CHECK (
    (
      status = 'delivering'
      AND claim_token IS NOT NULL
      AND claim_expires_at IS NOT NULL
    )
    OR status <> 'delivering'
  )
);

CREATE INDEX weekly_recaps_claim
  ON weekly_recaps(available_at, created_at, id)
  WHERE status IN ('pending', 'failed');

CREATE INDEX weekly_recaps_expired_claim
  ON weekly_recaps(claim_expires_at, id)
  WHERE status = 'delivering';
