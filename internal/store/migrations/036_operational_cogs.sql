CREATE TABLE operational_cogs_events (
  external_reference text PRIMARY KEY
    CHECK (char_length(external_reference) BETWEEN 1 AND 128),
  account_id uuid REFERENCES accounts(id) ON DELETE SET NULL,
  category text NOT NULL CHECK (category IN (
    'search', 'email', 'storage', 'support', 'infrastructure', 'other'
  )),
  cost_microusd bigint NOT NULL CHECK (cost_microusd >= 0),
  occurred_at timestamptz NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX operational_cogs_account_occurred
  ON operational_cogs_events(account_id, occurred_at DESC);
CREATE INDEX operational_cogs_category_occurred
  ON operational_cogs_events(category, occurred_at DESC);
