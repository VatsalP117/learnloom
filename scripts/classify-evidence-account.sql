\set ON_ERROR_STOP on

-- Append an operator-reviewed evidence classification without storing an email,
-- name, or other identity in the audit record.
--
-- Required psql variables:
--   account_id          UUID selected through an authorized identity lookup
--   evidence_class      real_user | founder | test
--   reason_code         see migration 038 for the bounded taxonomy
--   classified_by       non-secret operator reference (for example founder)
--   evidence_reference  non-secret evidence pointer (for example beta/cohort-01)
--
-- Example:
--   psql "$DATABASE_URL" -X \
--     -v account_id=00000000-0000-0000-0000-000000000000 \
--     -v evidence_class=founder \
--     -v reason_code=founder_or_team \
--     -v classified_by=founder \
--     -v evidence_reference=ops/founder-account \
--     -f scripts/classify-evidence-account.sql

\if :{?account_id}
\else
  \echo 'account_id is required'
  DO $$ BEGIN RAISE EXCEPTION 'account_id is required'; END $$;
\endif
\if :{?evidence_class}
\else
  \echo 'evidence_class is required'
  DO $$ BEGIN RAISE EXCEPTION 'evidence_class is required'; END $$;
\endif
\if :{?reason_code}
\else
  \echo 'reason_code is required'
  DO $$ BEGIN RAISE EXCEPTION 'reason_code is required'; END $$;
\endif
\if :{?classified_by}
\else
  \echo 'classified_by is required'
  DO $$ BEGIN RAISE EXCEPTION 'classified_by is required'; END $$;
\endif
\if :{?evidence_reference}
\else
  \echo 'evidence_reference is required'
  DO $$ BEGIN RAISE EXCEPTION 'evidence_reference is required'; END $$;
\endif

BEGIN;

INSERT INTO account_evidence_classifications (
  account_id,
  evidence_class,
  reason_code,
  classified_by,
  evidence_reference
)
SELECT
  :'account_id'::uuid,
  :'evidence_class',
  :'reason_code',
  :'classified_by',
  :'evidence_reference'
FROM accounts
WHERE id = :'account_id'::uuid
  AND status <> 'deleted';

SELECT :ROW_COUNT = 1 AS classification_inserted \gset
\if :classification_inserted
\else
  ROLLBACK;
  \echo 'active or suspended account not found'
  DO $$ BEGIN RAISE EXCEPTION 'active or suspended account not found'; END $$;
\endif

COMMIT;

SELECT
  evidence_class,
  reason_code,
  classified_by,
  evidence_reference,
  classified_at AT TIME ZONE 'UTC' AS classified_at_utc
FROM account_evidence_classifications
WHERE account_id = :'account_id'::uuid
ORDER BY classified_at DESC, id DESC
LIMIT 1;
