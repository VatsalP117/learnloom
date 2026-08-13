CREATE TABLE public_growth_events (
  id bigserial PRIMARY KEY,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  owner_account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  event_name text NOT NULL
    CHECK (event_name IN ('view', 'share', 'cta_click')),
  channel text NOT NULL DEFAULT ''
    CHECK (channel IN ('', 'linkedin', 'x', 'email')),
  visitor_fingerprint text NOT NULL
    CHECK (char_length(visitor_fingerprint) = 64),
  visitor_day date NOT NULL,
  occurred_at timestamptz NOT NULL,
  UNIQUE (issue_id, event_name, channel, visitor_fingerprint, visitor_day)
);

CREATE INDEX public_growth_events_owner_period
  ON public_growth_events(owner_account_id, occurred_at DESC);

CREATE TABLE public_attribution_conversions (
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  owner_account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  converted_account_id uuid NOT NULL UNIQUE
    REFERENCES accounts(id) ON DELETE CASCADE,
  referral_fingerprint text NOT NULL
    CHECK (char_length(referral_fingerprint) = 64),
  converted_at timestamptz NOT NULL,
  PRIMARY KEY (issue_id, converted_account_id)
);

CREATE INDEX public_attribution_conversions_owner_period
  ON public_attribution_conversions(owner_account_id, converted_at DESC);
