ALTER TABLE review_items
ADD COLUMN concept_keys text[] NOT NULL DEFAULT '{}',
ADD CONSTRAINT review_items_concept_keys_bounded
CHECK (cardinality(concept_keys) <= 12);

CREATE TABLE issue_concepts (
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  issue_id uuid NOT NULL
    REFERENCES issues(id) ON DELETE CASCADE,
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  concept_key text NOT NULL
    CHECK (char_length(concept_key) BETWEEN 1 AND 240),
  label text NOT NULL
    CHECK (char_length(label) BETWEEN 1 AND 300),
  role text NOT NULL
    CHECK (role IN ('core', 'prerequisite')),
  created_at timestamptz NOT NULL,
  PRIMARY KEY (issue_id, concept_key)
);

CREATE INDEX issue_concepts_account_stream
  ON issue_concepts(account_id, newsletter_id);

CREATE TABLE learner_concept_state (
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  concept_key text NOT NULL
    CHECK (char_length(concept_key) BETWEEN 1 AND 240),
  label text NOT NULL
    CHECK (char_length(label) BETWEEN 1 AND 300),
  role text NOT NULL
    CHECK (role IN ('core', 'prerequisite')),
  exposure_count integer NOT NULL DEFAULT 0
    CHECK (exposure_count >= 0),
  completed_count integer NOT NULL DEFAULT 0
    CHECK (completed_count >= 0 AND completed_count <= exposure_count),
  review_attempt_count integer NOT NULL DEFAULT 0
    CHECK (review_attempt_count >= 0),
  confidence_score smallint NOT NULL DEFAULT 0
    CHECK (confidence_score BETWEEN 0 AND 100),
  last_seen_at timestamptz NOT NULL,
  last_completed_at timestamptz,
  last_reviewed_at timestamptz,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (account_id, newsletter_id, concept_key)
);

CREATE INDEX learner_concept_state_stream
  ON learner_concept_state(account_id, newsletter_id, updated_at DESC);
