ALTER TABLE issues
  ADD COLUMN moderation_state text NOT NULL DEFAULT 'clear'
    CHECK (moderation_state IN ('clear', 'held')),
  ADD COLUMN moderation_reason text NOT NULL DEFAULT ''
    CHECK (char_length(moderation_reason) <= 1000);

CREATE TABLE public_content_reports (
  id uuid PRIMARY KEY,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  category text NOT NULL
    CHECK (category IN ('inaccurate', 'citation', 'harmful', 'other')),
  details text NOT NULL DEFAULT ''
    CHECK (char_length(details) <= 2000),
  reporter_fingerprint text NOT NULL
    CHECK (char_length(reporter_fingerprint) BETWEEN 16 AND 128),
  status text NOT NULL DEFAULT 'open'
    CHECK (status IN ('open', 'resolved', 'dismissed')),
  resolution_reason text NOT NULL DEFAULT ''
    CHECK (char_length(resolution_reason) <= 1000),
  created_at timestamptz NOT NULL,
  resolved_at timestamptz
);

CREATE INDEX public_content_reports_owner_queue
  ON public_content_reports(issue_id, status, created_at DESC);

CREATE TABLE public_corrections (
  id uuid PRIMARY KEY,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  owner_account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  body text NOT NULL
    CHECK (char_length(body) BETWEEN 1 AND 2000),
  status text NOT NULL DEFAULT 'published'
    CHECK (status IN ('published', 'retracted')),
  created_at timestamptz NOT NULL,
  retracted_at timestamptz
);

CREATE INDEX public_corrections_issue_published
  ON public_corrections(issue_id, created_at)
  WHERE status = 'published';

CREATE TABLE public_moderation_actions (
  id uuid PRIMARY KEY,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  report_id uuid
    REFERENCES public_content_reports(id) ON DELETE SET NULL,
  actor_account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  action text NOT NULL
    CHECK (action IN (
      'correction_published',
      'correction_retracted',
      'report_resolved',
      'report_dismissed',
      'publication_held',
      'publication_cleared'
    )),
  reason text NOT NULL DEFAULT ''
    CHECK (char_length(reason) <= 1000),
  created_at timestamptz NOT NULL
);

CREATE INDEX public_moderation_actions_issue_audit
  ON public_moderation_actions(issue_id, created_at DESC);
