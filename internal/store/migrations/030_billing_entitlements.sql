CREATE TABLE billing_plans (
  id text PRIMARY KEY CHECK (id IN ('free', 'pro')),
  display_name text NOT NULL,
  generation_allowance integer NOT NULL CHECK (generation_allowance >= 0),
  period_days integer NOT NULL CHECK (period_days BETWEEN 1 AND 366),
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

INSERT INTO billing_plans (
  id, display_name, generation_allowance, period_days, created_at, updated_at
) VALUES
  ('free', 'Free', 3, 30, now(), now()),
  ('pro', 'Pro', 30, 30, now(), now());

CREATE TABLE account_billing (
  account_id uuid PRIMARY KEY
    REFERENCES accounts(id) ON DELETE CASCADE,
  provider text NOT NULL DEFAULT 'none'
    CHECK (provider IN ('none', 'paddle')),
  provider_customer_id text UNIQUE,
  provider_subscription_id text UNIQUE,
  plan_id text NOT NULL DEFAULT 'free'
    REFERENCES billing_plans(id),
  subscription_status text NOT NULL DEFAULT 'free'
    CHECK (subscription_status IN (
      'free', 'trialing', 'active', 'past_due', 'paused', 'canceled', 'refunded'
    )),
  entitlement_status text NOT NULL DEFAULT 'active'
    CHECK (entitlement_status IN ('active', 'grace', 'generation_paused')),
  current_period_start timestamptz NOT NULL,
  current_period_end timestamptz NOT NULL,
  trial_ends_at timestamptz,
  grace_ends_at timestamptz,
  cancel_at_period_end boolean NOT NULL DEFAULT false,
  canceled_at timestamptz,
  provider_event_at timestamptz,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  CHECK (current_period_end > current_period_start)
);

CREATE TABLE generation_usage_reservations (
  issue_id uuid PRIMARY KEY
    REFERENCES issues(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  plan_id text NOT NULL REFERENCES billing_plans(id),
  period_start timestamptz NOT NULL,
  units integer NOT NULL DEFAULT 1 CHECK (units > 0),
  state text NOT NULL DEFAULT 'reserved'
    CHECK (state IN ('reserved', 'consumed', 'released')),
  reserved_at timestamptz NOT NULL,
  consumed_at timestamptz,
  released_at timestamptz
);

CREATE INDEX generation_usage_account_period
  ON generation_usage_reservations(account_id, period_start, state);

CREATE TABLE billing_lifecycle_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  event_name text NOT NULL
    CHECK (event_name IN (
      'trial_started', 'paywall_exposed', 'checkout_started',
      'payment_succeeded', 'payment_failed', 'subscription_canceled',
      'subscription_reactivated', 'refund_issued'
    )),
  provider_event_id text,
  occurred_at timestamptz NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(metadata) = 'object')
    CHECK (octet_length(metadata::text) <= 4096),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX billing_lifecycle_provider_event
  ON billing_lifecycle_events(provider_event_id, event_name)
  WHERE provider_event_id IS NOT NULL;

CREATE TABLE billing_webhook_events (
  provider text NOT NULL CHECK (provider IN ('paddle')),
  event_id text NOT NULL,
  event_type text NOT NULL,
  event_occurred_at timestamptz NOT NULL,
  received_at timestamptz NOT NULL,
  processed_at timestamptz,
  payload_sha256 text NOT NULL CHECK (char_length(payload_sha256) = 64),
  error text,
  PRIMARY KEY (provider, event_id)
);

ALTER TABLE rhythm_decisions DROP CONSTRAINT rhythm_decisions_decision_check;
ALTER TABLE rhythm_decisions ADD CONSTRAINT rhythm_decisions_decision_check
  CHECK (decision IN (
    'configure', 'dispatch', 'defer', 'throttle', 'recover',
    'entitlement_deferred'
  ));
