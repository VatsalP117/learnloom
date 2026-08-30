-- Dodo Payments becomes the active billing provider and merchant of record.
-- Historical Paddle rows stay intact as audited history; every new billing
-- write uses the 'dodo' provider. Paddle identifiers inside historical rows
-- (txn_ checkout sessions, ctm_/sub_ references) are not rewritten.

ALTER TABLE account_billing
  DROP CONSTRAINT account_billing_provider_check;

ALTER TABLE account_billing
  ADD CONSTRAINT account_billing_provider_check
  CHECK (provider IN ('none', 'paddle', 'dodo'));

ALTER TABLE billing_webhook_events
  DROP CONSTRAINT billing_webhook_events_provider_check;

ALTER TABLE billing_webhook_events
  ADD CONSTRAINT billing_webhook_events_provider_check
  CHECK (provider IN ('paddle', 'dodo'));

-- Dodo hosted checkout sessions use cks_ IDs. The relaxed constraint keeps
-- historical txn_ rows addressable while new sessions must use cks_.
ALTER TABLE billing_checkout_sessions
  DROP CONSTRAINT billing_checkout_sessions_transaction_id_check;

ALTER TABLE billing_checkout_sessions
  ADD CONSTRAINT billing_checkout_sessions_transaction_id_check
  CHECK (transaction_id LIKE 'cks_%' OR transaction_id LIKE 'txn_%');
