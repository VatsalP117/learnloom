CREATE TABLE lesson_backlog_dismissals (
  account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  newsletter_id uuid NOT NULL REFERENCES newsletters(id) ON DELETE CASCADE,
  reason text NOT NULL CHECK (reason IN ('reentry_reset')),
  dismissed_at timestamptz NOT NULL,
  PRIMARY KEY (account_id, issue_id)
);

CREATE INDEX lesson_backlog_dismissals_newsletter
  ON lesson_backlog_dismissals(newsletter_id, dismissed_at DESC);
