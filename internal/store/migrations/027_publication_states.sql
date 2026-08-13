ALTER TABLE issues DROP CONSTRAINT issues_publication_state_check;

UPDATE issues
SET publication_state = 'private'
WHERE publication_state = 'hidden';

ALTER TABLE issues
  ALTER COLUMN publication_state SET DEFAULT 'draft',
  ADD CONSTRAINT issues_publication_state_check
    CHECK (publication_state IN ('private', 'draft', 'published')),
  ADD COLUMN publication_updated_at timestamptz,
  ADD COLUMN first_publish_reviewed_at timestamptz,
  ADD COLUMN published_at timestamptz;

UPDATE issues
SET publication_updated_at = COALESCE(completed_at, created_at),
    first_publish_reviewed_at = CASE
      WHEN publication_state = 'published' THEN COALESCE(completed_at, created_at)
      ELSE NULL
    END,
    published_at = CASE
      WHEN publication_state = 'published' THEN COALESCE(completed_at, created_at)
      ELSE NULL
    END;

ALTER TABLE issues ALTER COLUMN publication_updated_at SET NOT NULL;
ALTER TABLE issues ALTER COLUMN publication_updated_at SET DEFAULT now();

ALTER TABLE newsletters
  ADD COLUMN lesson_publication_default text NOT NULL DEFAULT 'draft'
    CHECK (lesson_publication_default IN ('draft', 'published')),
  ADD COLUMN lesson_publication_default_reviewed_at timestamptz;

COMMENT ON COLUMN issues.publication_state IS
  'private: intentionally owner-only; draft: owner-only and ready for review; published: public only when site and stream gates also permit';

COMMENT ON COLUMN newsletters.lesson_publication_default IS
  'Desired state assigned to newly queued lessons; draft is the safe default.';
