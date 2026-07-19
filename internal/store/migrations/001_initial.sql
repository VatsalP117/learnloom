CREATE TABLE accounts (
  id uuid PRIMARY KEY,
  clerk_user_id text NOT NULL UNIQUE,
  primary_email text,
  status text NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'suspended', 'deleted')),
  identity_event_at bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  deleted_at timestamptz
);

CREATE TABLE personal_sites (
  id uuid PRIMARY KEY,
  owner_account_id uuid NOT NULL UNIQUE
    REFERENCES accounts(id) ON DELETE CASCADE,
  username text NOT NULL UNIQUE,
  display_name text NOT NULL,
  description text NOT NULL DEFAULT '',
  visibility text NOT NULL DEFAULT 'private'
    CHECK (visibility IN ('private', 'public')),
  claimed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  CHECK (username = lower(username))
);

CREATE TABLE newsletters (
  id uuid PRIMARY KEY,
  owner_account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  name text NOT NULL,
  topic text NOT NULL,
  learner_level text NOT NULL,
  learner_goal text NOT NULL,
  lesson_minutes integer NOT NULL CHECK (lesson_minutes BETWEEN 5 AND 90),
  sources jsonb NOT NULL CHECK (jsonb_typeof(sources) = 'array'),
  schedule_hour integer NOT NULL CHECK (schedule_hour BETWEEN 0 AND 23),
  schedule_minute integer NOT NULL CHECK (schedule_minute BETWEEN 0 AND 59),
  time_zone text NOT NULL,
  active boolean NOT NULL DEFAULT true,
  next_run_at timestamptz NOT NULL,
  email_enabled boolean NOT NULL DEFAULT false,
  ai_exploration_enabled boolean NOT NULL DEFAULT false,
  public_slug text NOT NULL,
  site_visible boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  UNIQUE (owner_account_id, public_slug)
);

CREATE INDEX newsletters_owner_created
  ON newsletters(owner_account_id, created_at DESC);
CREATE INDEX newsletters_due
  ON newsletters(next_run_at, id)
  WHERE active;

CREATE TABLE issues (
  id uuid PRIMARY KEY,
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  trigger text NOT NULL CHECK (trigger IN ('scheduled', 'manual')),
  scheduled_local_date date,
  status text NOT NULL
    CHECK (status IN ('queued', 'generating', 'generated', 'failed', 'cancelled')),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  available_at timestamptz NOT NULL,
  claim_token uuid,
  claim_expires_at timestamptz,
  dossier_title text,
  generation_id uuid,
  artifact_key text,
  artifact_sha256 text,
  artifact_bytes integer,
  error text,
  public_id uuid NOT NULL UNIQUE,
  public_slug text,
  publication_state text NOT NULL DEFAULT 'published'
    CHECK (publication_state IN ('published', 'hidden')),
  created_at timestamptz NOT NULL,
  started_at timestamptz,
  completed_at timestamptz,
  CHECK (
    (status = 'generating' AND claim_token IS NOT NULL AND claim_expires_at IS NOT NULL)
    OR status <> 'generating'
  ),
  CHECK (
    (status = 'generated' AND generation_id IS NOT NULL AND artifact_key IS NOT NULL)
    OR status <> 'generated'
  )
);

CREATE UNIQUE INDEX issues_one_scheduled_per_day
  ON issues(newsletter_id, scheduled_local_date)
  WHERE trigger = 'scheduled';
CREATE INDEX issues_claim
  ON issues(available_at, created_at, id)
  WHERE status = 'queued';
CREATE INDEX issues_expired_claim
  ON issues(claim_expires_at, id)
  WHERE status = 'generating';
CREATE INDEX issues_newsletter_history
  ON issues(newsletter_id, created_at DESC);

CREATE TABLE delivery_receipts (
  issue_id uuid PRIMARY KEY
    REFERENCES issues(id) ON DELETE CASCADE,
  status text NOT NULL
    CHECK (status IN ('pending', 'delivering', 'delivered', 'failed', 'cancelled', 'unknown')),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  available_at timestamptz NOT NULL,
  claim_token uuid,
  claim_expires_at timestamptz,
  external_id text,
  error text,
  created_at timestamptz NOT NULL,
  started_at timestamptz,
  completed_at timestamptz,
  updated_at timestamptz NOT NULL,
  CHECK (
    (status = 'delivering' AND claim_token IS NOT NULL AND claim_expires_at IS NOT NULL)
    OR status <> 'delivering'
  )
);

CREATE INDEX delivery_receipts_claim
  ON delivery_receipts(available_at, created_at, issue_id)
  WHERE status IN ('pending', 'failed');
CREATE INDEX delivery_receipts_expired_claim
  ON delivery_receipts(claim_expires_at, issue_id)
  WHERE status = 'delivering';

CREATE TABLE learning_history (
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL UNIQUE
    REFERENCES issues(id) ON DELETE CASCADE,
  local_date date NOT NULL,
  entry jsonb NOT NULL CHECK (jsonb_typeof(entry) = 'object'),
  created_at timestamptz NOT NULL,
  PRIMARY KEY (newsletter_id, local_date, issue_id)
);

CREATE INDEX learning_history_recent
  ON learning_history(newsletter_id, created_at DESC);

CREATE TABLE webhook_events (
  id text PRIMARY KEY,
  event_type text NOT NULL,
  received_at timestamptz NOT NULL,
  processed_at timestamptz,
  error text
);

CREATE TABLE account_deletion_queue (
  account_id uuid PRIMARY KEY,
  available_at timestamptz NOT NULL,
  attempt_count integer NOT NULL DEFAULT 0,
  claim_token uuid,
  claim_expires_at timestamptz,
  completed_at timestamptz,
  error text
);

CREATE TABLE request_rate_buckets (
  bucket_key text NOT NULL,
  action text NOT NULL,
  window_start timestamptz NOT NULL,
  request_count integer NOT NULL CHECK (request_count >= 0),
  PRIMARY KEY (bucket_key, action, window_start)
);

CREATE TABLE runtime_controls (
  id boolean PRIMARY KEY DEFAULT true CHECK (id),
  generation_paused boolean NOT NULL DEFAULT false,
  pause_reason text,
  updated_at timestamptz NOT NULL
);

INSERT INTO runtime_controls (id, generation_paused, updated_at)
VALUES (true, false, now());
