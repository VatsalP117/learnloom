ALTER TABLE public_attribution_conversions
  ADD COLUMN activated_at timestamptz,
  ADD CONSTRAINT public_attribution_activation_after_conversion
    CHECK (activated_at IS NULL OR activated_at >= converted_at);

CREATE INDEX public_attribution_activated_owner_period
  ON public_attribution_conversions(owner_account_id, activated_at DESC)
  WHERE activated_at IS NOT NULL;
