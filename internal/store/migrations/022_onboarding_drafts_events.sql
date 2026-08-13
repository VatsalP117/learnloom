ALTER TABLE product_events DROP CONSTRAINT product_events_event_name_check;
ALTER TABLE product_events ADD CONSTRAINT product_events_event_name_check
  CHECK (event_name IN (
    'signup_completed',
    'onboarding_started',
    'onboarding_intent_completed',
    'onboarding_sources_completed',
    'source_policy_selected',
    'source_preview_reached',
    'onboarding_confirmed',
    'onboarding_abandoned',
    'preparation_wait_exited',
    'stream_created',
    'lesson_generated',
    'lesson_opened',
    'lesson_completed',
    'review_attempted',
    'first_retrieval_completed',
    'activation_completed',
    'search_indexing_enabled'
  ));

ALTER TABLE product_events DROP CONSTRAINT product_events_subject_type_check;
ALTER TABLE product_events ADD CONSTRAINT product_events_subject_type_check
  CHECK (subject_type IN (
    'account', 'onboarding', 'stream', 'lesson', 'review', 'site'
  ));

CREATE TABLE onboarding_drafts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL UNIQUE
    REFERENCES accounts(id) ON DELETE CASCADE,
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  step smallint NOT NULL DEFAULT 1 CHECK (step BETWEEN 1 AND 3),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(payload) = 'object')
    CHECK (octet_length(payload::text) <= 32768),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX onboarding_drafts_recent
  ON onboarding_drafts(updated_at DESC);

CREATE TABLE onboarding_draft_completions (
  draft_id uuid PRIMARY KEY,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  completed_at timestamptz NOT NULL
);

CREATE INDEX onboarding_draft_completions_account
  ON onboarding_draft_completions(account_id, completed_at DESC);
