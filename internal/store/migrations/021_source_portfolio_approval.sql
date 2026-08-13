ALTER TABLE newsletters
  ADD COLUMN source_review_mode text NOT NULL DEFAULT 'auto'
    CHECK (source_review_mode IN ('auto', 'review')),
  ADD COLUMN source_approved_at timestamptz;

ALTER TABLE issues DROP CONSTRAINT issues_status_check;
ALTER TABLE issues ADD CONSTRAINT issues_status_check CHECK (
  status IN (
    'queued', 'generating', 'awaiting_approval', 'generated',
    'failed', 'deferred', 'cancelled'
  )
);

ALTER TABLE issue_attempts DROP CONSTRAINT issue_attempts_status_check;
ALTER TABLE issue_attempts ADD CONSTRAINT issue_attempts_status_check CHECK (
  status IN (
    'running', 'completed', 'awaiting_approval', 'failed',
    'deferred', 'abandoned'
  )
);

CREATE INDEX issues_awaiting_source_approval
  ON issues(newsletter_id, created_at DESC)
  WHERE status = 'awaiting_approval';
