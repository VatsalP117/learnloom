CREATE TABLE lesson_search_documents (
  issue_id uuid PRIMARY KEY
    REFERENCES issues(id) ON DELETE CASCADE,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  newsletter_id uuid NOT NULL
    REFERENCES newsletters(id) ON DELETE CASCADE,
  title text NOT NULL,
  concepts text[] NOT NULL DEFAULT '{}',
  source_titles text[] NOT NULL DEFAULT '{}',
  retrieval_prompts text[] NOT NULL DEFAULT '{}',
  search_text text NOT NULL,
  document tsvector GENERATED ALWAYS AS (
    to_tsvector('english'::regconfig, search_text)
  ) STORED,
  created_at timestamptz NOT NULL
);

CREATE INDEX lesson_search_documents_account
  ON lesson_search_documents(account_id, created_at DESC);

CREATE INDEX lesson_search_documents_fts
  ON lesson_search_documents USING gin(document);

INSERT INTO lesson_search_documents (
  issue_id, account_id, newsletter_id, title, concepts,
  source_titles, retrieval_prompts, search_text, created_at
)
SELECT
  history.issue_id,
  newsletter.owner_account_id,
  history.newsletter_id,
  COALESCE(issue.dossier_title, ''),
  ARRAY(
    SELECT jsonb_array_elements_text(
      COALESCE(history.entry->'concepts', '[]'::jsonb)
    )
  ),
  ARRAY(
    SELECT jsonb_array_elements_text(
      COALESCE(history.entry->'sourceTitles', '[]'::jsonb)
    )
  ),
  ARRAY(
    SELECT prompt->>'prompt'
    FROM jsonb_array_elements(
      COALESCE(history.entry->'retrievalPrompts', '[]'::jsonb)
    ) prompt
    WHERE prompt->>'prompt' IS NOT NULL
  ),
  concat_ws(
    ' ',
    issue.dossier_title,
    newsletter.name,
    newsletter.topic,
    (
      SELECT string_agg(value, ' ')
      FROM jsonb_array_elements_text(
        COALESCE(history.entry->'concepts', '[]'::jsonb)
      ) value
    ),
    (
      SELECT string_agg(value, ' ')
      FROM jsonb_array_elements_text(
        COALESCE(history.entry->'sourceTitles', '[]'::jsonb)
      ) value
    ),
    (
      SELECT string_agg(prompt->>'prompt', ' ')
      FROM jsonb_array_elements(
        COALESCE(history.entry->'retrievalPrompts', '[]'::jsonb)
      ) prompt
    )
  ),
  history.created_at
FROM learning_history history
JOIN issues issue ON issue.id = history.issue_id
JOIN newsletters newsletter ON newsletter.id = history.newsletter_id
WHERE issue.status = 'generated'
ON CONFLICT (issue_id) DO NOTHING;
