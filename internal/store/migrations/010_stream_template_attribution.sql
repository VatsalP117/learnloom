ALTER TABLE newsletters
ADD COLUMN stream_template_id text,
ADD COLUMN stream_template_version integer,
ADD CONSTRAINT newsletters_template_attribution_complete
CHECK (
  (stream_template_id IS NULL AND stream_template_version IS NULL)
  OR (
    stream_template_id ~ '^[a-z0-9][a-z0-9-]{1,79}$'
    AND stream_template_version > 0
  )
);

CREATE INDEX newsletters_template_activation
  ON newsletters(stream_template_id, stream_template_version, created_at)
  WHERE stream_template_id IS NOT NULL;
