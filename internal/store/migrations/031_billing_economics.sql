ALTER TABLE billing_lifecycle_events
  ADD COLUMN reason text CHECK (reason IS NULL OR char_length(reason) BETWEEN 1 AND 500),
  ADD COLUMN currency_code text CHECK (
    currency_code IS NULL OR currency_code ~ '^[A-Z]{3}$'
  ),
  ADD COLUMN amount_minor bigint CHECK (amount_minor IS NULL OR amount_minor >= 0);

CREATE TABLE billing_revenue_events (
  provider_event_id text PRIMARY KEY,
  account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  event_type text NOT NULL CHECK (event_type IN ('payment', 'refund')),
  currency_code text NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
  amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
  provider_fee_minor bigint CHECK (provider_fee_minor IS NULL OR provider_fee_minor >= 0),
  occurred_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX billing_revenue_account_occurred
  ON billing_revenue_events(account_id, occurred_at DESC);
