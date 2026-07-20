ALTER TABLE newsletters ADD COLUMN source_mode text NOT NULL DEFAULT 'provided'
  CHECK (source_mode IN ('discovered', 'provided', 'hybrid'));

CREATE TABLE source_specs (
  id uuid PRIMARY KEY,
  newsletter_id uuid NOT NULL REFERENCES newsletters(id) ON DELETE CASCADE,
  origin text NOT NULL CHECK (origin IN ('provided', 'discovered')),
  state text NOT NULL CHECK (state IN ('candidate', 'active', 'unhealthy', 'rejected', 'disabled')),
  display_name text NOT NULL,
  input_url text NOT NULL,
  canonical_url text,
  scope text NOT NULL CHECK (scope IN ('exact', 'feed', 'site', 'document')),
  kind text CHECK (kind IN ('rss', 'atom', 'json_feed', 'html', 'text', 'pdf')),
  item_limit integer NOT NULL CHECK (item_limit BETWEEN 1 AND 50),
  discovery_reason text,
  discovery_query text,
  rank_score integer,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX source_specs_newsletter_state
  ON source_specs(newsletter_id, state);
CREATE UNIQUE INDEX source_specs_newsletter_canonical_unique
  ON source_specs(newsletter_id, canonical_url)
  WHERE canonical_url IS NOT NULL;

CREATE TABLE source_endpoints (
  id uuid PRIMARY KEY,
  source_spec_id uuid NOT NULL REFERENCES source_specs(id) ON DELETE CASCADE,
  endpoint_url text NOT NULL,
  canonical_url text NOT NULL,
  kind text NOT NULL CHECK (kind IN ('rss', 'atom', 'json_feed', 'html', 'text', 'pdf')),
  etag text,
  last_modified text,
  last_http_status integer,
  health text NOT NULL DEFAULT 'unknown'
    CHECK (health IN ('unknown', 'healthy', 'stale', 'failing', 'blocked')),
  consecutive_failures integer NOT NULL DEFAULT 0,
  last_checked_at timestamptz,
  last_success_at timestamptz,
  last_changed_at timestamptz,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (source_spec_id, canonical_url)
);

CREATE TABLE source_snapshots (
  id uuid PRIMARY KEY,
  source_endpoint_id uuid NOT NULL REFERENCES source_endpoints(id) ON DELETE CASCADE,
  item_key text NOT NULL,
  title text NOT NULL,
  canonical_url text NOT NULL,
  author text,
  published_at timestamptz,
  content text NOT NULL,
  content_source text NOT NULL,
  content_sha256 text NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}',
  fetched_at timestamptz NOT NULL,
  UNIQUE (source_endpoint_id, item_key, content_sha256)
);

CREATE TABLE issue_sources (
  issue_id uuid REFERENCES issues(id) ON DELETE CASCADE,
  source_snapshot_id uuid REFERENCES source_snapshots(id),
  position integer NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (issue_id, source_snapshot_id),
  UNIQUE (issue_id, position)
);

CREATE TABLE discovery_runs (
  id uuid PRIMARY KEY,
  newsletter_id uuid NOT NULL REFERENCES newsletters(id) ON DELETE CASCADE,
  issue_id uuid REFERENCES issues(id) ON DELETE SET NULL,
  reason text NOT NULL CHECK (reason IN (
    'initial', 'insufficient_items', 'stale_catalog', 'coverage_gap', 'manual_refresh'
  )),
  state text NOT NULL CHECK (state IN ('running', 'completed', 'degraded', 'failed')),
  query_bundle jsonb NOT NULL DEFAULT '{}',
  returned_candidates integer NOT NULL DEFAULT 0,
  rejected_candidates integer NOT NULL DEFAULT 0,
  resolved_candidates integer NOT NULL DEFAULT 0,
  activated_candidates integer NOT NULL DEFAULT 0,
  error text,
  started_at timestamptz,
  completed_at timestamptz
);

DO $$
DECLARE
  newsletter_record RECORD;
  source_record RECORD;
  source_json jsonb;
  source_spec_id uuid;
BEGIN
  FOR newsletter_record IN
    SELECT id, sources FROM newsletters
  LOOP
    IF newsletter_record.sources IS NOT NULL AND jsonb_array_length(newsletter_record.sources) > 0 THEN
      FOR source_record IN
        SELECT value, (row_number() OVER ()) - 1 AS idx
        FROM jsonb_array_elements(newsletter_record.sources)
      LOOP
        source_json := source_record.value;
        source_spec_id := gen_random_uuid();
        INSERT INTO source_specs (
          id, newsletter_id, origin, state, display_name, input_url,
          canonical_url, scope, kind, item_limit, created_at, updated_at
        )
        VALUES (
          source_spec_id,
          newsletter_record.id,
          'provided',
          'active',
          COALESCE(source_json->>'name', 'Source'),
          COALESCE(source_json->>'url', ''),
          COALESCE(source_json->>'url', ''),
          CASE
            WHEN source_json->>'url' LIKE '%/feed%' OR source_json->>'url' LIKE '%/rss%' OR source_json->>'url' LIKE '%/atom%' THEN 'feed'
            ELSE 'exact'
          END,
          CASE
            WHEN source_json->>'url' LIKE '%/feed%' OR source_json->>'url' LIKE '%/rss%' THEN 'rss'
            WHEN source_json->>'url' LIKE '%/atom%' THEN 'atom'
            ELSE NULL
          END,
          COALESCE((source_json->>'limit')::int, 8),
          now(),
          now()
        )
        ON CONFLICT DO NOTHING;
      END LOOP;
    END IF;
  END LOOP;
END $$;
