CREATE TABLE billing_checkout_sessions (
  transaction_id text PRIMARY KEY CHECK (transaction_id LIKE 'txn_%'),
  account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  status text NOT NULL CHECK (status IN ('pending', 'completed', 'expired')),
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX billing_checkout_one_pending_per_account
  ON billing_checkout_sessions(account_id) WHERE status = 'pending';

CREATE INDEX billing_checkout_sessions_updated
  ON billing_checkout_sessions(updated_at DESC);
