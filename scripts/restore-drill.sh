#!/usr/bin/env bash
set -euo pipefail

: "${SOURCE_DATABASE_URL:?set SOURCE_DATABASE_URL}"
: "${RESTORE_DATABASE_URL:?set RESTORE_DATABASE_URL}"
: "${RESTORE_EVIDENCE_PATH:?set RESTORE_EVIDENCE_PATH}"
: "${RESTORE_ARTIFACT_SOURCE_FILE:?set RESTORE_ARTIFACT_SOURCE_FILE}"
: "${RESTORE_ARTIFACT_RESTORED_FILE:?set RESTORE_ARTIFACT_RESTORED_FILE}"

restore_database="$(
  psql "$RESTORE_DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
    -c "SELECT current_database()"
)"
case "$restore_database" in
  *restore*|*drill*) ;;
  *)
    echo "refusing restore target without 'restore' or 'drill' in its database name" >&2
    exit 2
    ;;
esac

existing_tables="$(
  psql "$RESTORE_DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
    -c "SELECT count(*) FROM pg_tables WHERE schemaname = 'public'"
)"
if [[ "$existing_tables" != "0" ]]; then
  echo "restore target must be an empty isolated database" >&2
  exit 2
fi

drill_directory="$(mktemp -d)"
trap 'rm -rf "$drill_directory"' EXIT
dump_path="$drill_directory/learnloom.dump"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
started_epoch="$(date +%s)"

pg_dump --format=custom --no-owner --no-privileges \
  --file="$dump_path" "$SOURCE_DATABASE_URL"
pg_restore --exit-on-error --no-owner --no-privileges \
  --dbname="$RESTORE_DATABASE_URL" "$dump_path"

source_version="$(
  psql "$SOURCE_DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
    -c "SELECT COALESCE(max(version), 0) FROM schema_migrations"
)"
restored_version="$(
  psql "$RESTORE_DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
    -c "SELECT COALESCE(max(version), 0) FROM schema_migrations"
)"
if [[ "$source_version" != "$restored_version" ]]; then
  echo "schema version mismatch after restore" >&2
  exit 1
fi

source_accounts="$(
  psql "$SOURCE_DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
    -c "SELECT count(*) FROM accounts"
)"
restored_accounts="$(
  psql "$RESTORE_DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
    -c "SELECT count(*) FROM accounts"
)"
if [[ "$source_accounts" != "$restored_accounts" ]]; then
  echo "account count mismatch after restore" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  source_artifact_sha="$(sha256sum "$RESTORE_ARTIFACT_SOURCE_FILE" | awk '{print $1}')"
  restored_artifact_sha="$(sha256sum "$RESTORE_ARTIFACT_RESTORED_FILE" | awk '{print $1}')"
else
  source_artifact_sha="$(shasum -a 256 "$RESTORE_ARTIFACT_SOURCE_FILE" | awk '{print $1}')"
  restored_artifact_sha="$(shasum -a 256 "$RESTORE_ARTIFACT_RESTORED_FILE" | awk '{print $1}')"
fi
if [[ "$source_artifact_sha" != "$restored_artifact_sha" ]]; then
  echo "artifact checksum mismatch after restore" >&2
  exit 1
fi

completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
elapsed_seconds="$(( $(date +%s) - started_epoch ))"
mkdir -p "$(dirname "$RESTORE_EVIDENCE_PATH")"
{
  echo "# Learnloom restore drill"
  echo
  echo "- Started: $started_at"
  echo "- Completed: $completed_at"
  echo "- Duration: ${elapsed_seconds}s"
  echo "- Restore database: $restore_database"
  echo "- Schema version: $restored_version"
  echo "- Account rows compared: $restored_accounts"
  echo "- Sample restored object SHA-256: $restored_artifact_sha"
  echo "- Result: passed"
} >"$RESTORE_EVIDENCE_PATH"
