ALTER TABLE public_growth_events
  DROP CONSTRAINT public_growth_events_event_name_check;
ALTER TABLE public_growth_events
  ADD CONSTRAINT public_growth_events_event_name_check
  CHECK (event_name IN ('view', 'share', 'cta_click', 'follow'));

CREATE TABLE public_path_followers (
  id uuid PRIMARY KEY,
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  owner_account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  email text NOT NULL CHECK (char_length(email) BETWEEN 3 AND 320),
  email_hash text NOT NULL CHECK (char_length(email_hash) = 64),
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'confirmed', 'unsubscribed')),
  confirmation_token_hash text NOT NULL
    CHECK (char_length(confirmation_token_hash) = 64),
  unsubscribe_token_hash text NOT NULL
    CHECK (char_length(unsubscribe_token_hash) = 64),
  requested_at timestamptz NOT NULL,
  confirmed_at timestamptz,
  unsubscribed_at timestamptz,
  updated_at timestamptz NOT NULL,
  UNIQUE (newsletter_id, email_hash)
);

CREATE INDEX public_path_followers_owner_status
  ON public_path_followers(owner_account_id, status, updated_at DESC);

CREATE TABLE public_follow_unsubscribe_tokens (
  follower_id uuid NOT NULL
    REFERENCES public_path_followers(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE
    CHECK (char_length(token_hash) = 64),
  created_at timestamptz NOT NULL,
  PRIMARY KEY (follower_id, token_hash)
);

CREATE TABLE public_follow_deliveries (
  id uuid PRIMARY KEY,
  follower_id uuid NOT NULL
    REFERENCES public_path_followers(id) ON DELETE CASCADE,
  issue_id uuid REFERENCES issues(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('confirmation', 'update')),
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'delivering', 'delivered', 'failed', 'unknown')),
  token text NOT NULL CHECK (char_length(token) BETWEEN 32 AND 256),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  available_at timestamptz NOT NULL,
  claim_token uuid,
  claim_expires_at timestamptz,
  external_id text,
  error text,
  created_at timestamptz NOT NULL,
  completed_at timestamptz,
  updated_at timestamptz NOT NULL,
  CHECK (
    (status = 'delivering' AND claim_token IS NOT NULL AND claim_expires_at IS NOT NULL)
    OR status <> 'delivering'
  )
);

CREATE UNIQUE INDEX public_follow_deliveries_one_confirmation
  ON public_follow_deliveries(follower_id) WHERE kind = 'confirmation';
CREATE UNIQUE INDEX public_follow_deliveries_one_issue_update
  ON public_follow_deliveries(follower_id, issue_id) WHERE kind = 'update';

CREATE INDEX public_follow_deliveries_claim
  ON public_follow_deliveries(available_at, created_at, id)
  WHERE status IN ('pending', 'failed');
