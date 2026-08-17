ALTER TABLE billing_plans
  DROP CONSTRAINT billing_plans_id_check;

ALTER TABLE billing_plans
  ADD CONSTRAINT billing_plans_id_check
  CHECK (id IN ('none', 'free', 'essential', 'pro'));

ALTER TABLE billing_plans
  ALTER COLUMN generation_allowance DROP NOT NULL,
  ADD COLUMN stream_allowance integer
    CHECK (stream_allowance IS NULL OR stream_allowance >= 0);

INSERT INTO billing_plans (
  id, display_name, generation_allowance, period_days,
  stream_allowance, active, created_at, updated_at
) VALUES
  ('none', 'No active plan', 0, 30, 0, true, now(), now()),
  ('essential', 'Essential', NULL, 30, 3, true, now(), now());

UPDATE billing_plans
SET display_name = 'Pro', generation_allowance = NULL,
    stream_allowance = NULL, active = true, updated_at = now()
WHERE id = 'pro';

-- Keep the legacy row for historical generation reservations, but no account
-- may remain entitled through it after paid-only pricing launches.
UPDATE billing_plans
SET display_name = 'Legacy free (inactive)', stream_allowance = 0,
    active = false, updated_at = now()
WHERE id = 'free';

UPDATE account_billing
SET plan_id = 'none', entitlement_status = 'generation_paused', updated_at = now()
WHERE plan_id = 'free';

ALTER TABLE account_billing
  ALTER COLUMN plan_id SET DEFAULT 'none';

ALTER TABLE billing_checkout_sessions
  ADD COLUMN plan_id text REFERENCES billing_plans(id);

-- Existing pending checkouts were created before plan selection and always
-- targeted the former Pro price.
UPDATE billing_checkout_sessions SET plan_id = 'pro' WHERE plan_id IS NULL;

ALTER TABLE billing_checkout_sessions
  ALTER COLUMN plan_id SET NOT NULL;

COMMENT ON COLUMN billing_plans.generation_allowance IS
  'Generated-lesson allowance per billing period; NULL means unlimited. Essential and Pro are unlimited.';

COMMENT ON COLUMN billing_plans.stream_allowance IS
  'Total learning-stream allowance; NULL means unlimited. Essential allows 3 and Pro is unlimited.';
