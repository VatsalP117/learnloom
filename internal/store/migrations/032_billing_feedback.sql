CREATE TABLE billing_feedback (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  context text NOT NULL CHECK (context IN ('non_conversion', 'cancellation')),
  reason_code text NOT NULL CHECK (reason_code IN (
    'too_expensive', 'insufficient_value', 'quality_concerns',
    'reliability_concerns', 'allowance_too_low', 'missing_feature',
    'no_longer_needed', 'other'
  )),
  note text CHECK (note IS NULL OR char_length(note) BETWEEN 1 AND 1000),
  subscription_status text NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  UNIQUE (account_id, context)
);

CREATE INDEX billing_feedback_reason_created
  ON billing_feedback(context, reason_code, created_at DESC);
