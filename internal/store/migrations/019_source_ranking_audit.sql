ALTER TABLE source_specs
  ADD COLUMN source_role text
    CHECK (source_role IN (
      'official_primary', 'research', 'practitioner_explainer',
      'reporting_context', 'counterweight'
    )),
  ADD COLUMN ranking_version text,
  ADD COLUMN score_components jsonb NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(score_components) = 'object');

CREATE INDEX source_specs_newsletter_role
  ON source_specs(newsletter_id, source_role)
  WHERE source_role IS NOT NULL;
