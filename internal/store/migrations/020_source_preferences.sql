ALTER TABLE source_specs
  ADD COLUMN preference text NOT NULL DEFAULT 'neutral'
    CHECK (preference IN ('neutral', 'preferred', 'blocked'));

CREATE INDEX source_specs_newsletter_preference
  ON source_specs(newsletter_id, preference, state);
