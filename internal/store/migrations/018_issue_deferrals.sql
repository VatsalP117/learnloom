ALTER TABLE issues
  DROP CONSTRAINT issues_status_check,
  ADD CONSTRAINT issues_status_check CHECK (
    status IN ('queued', 'generating', 'generated', 'failed', 'deferred', 'cancelled')
  );

ALTER TABLE issue_attempts
  DROP CONSTRAINT issue_attempts_status_check,
  ADD CONSTRAINT issue_attempts_status_check CHECK (
    status IN ('running', 'completed', 'failed', 'deferred', 'abandoned')
  );

CREATE INDEX issues_deferred_timeline
  ON issues(newsletter_id, completed_at DESC)
  WHERE status = 'deferred';
