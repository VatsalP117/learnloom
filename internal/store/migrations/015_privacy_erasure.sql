CREATE TABLE deleted_identity_tombstones (
  identity_fingerprint text PRIMARY KEY
    CHECK (char_length(identity_fingerprint) = 64),
  identity_event_at bigint NOT NULL,
  deleted_at timestamptz NOT NULL
);

CREATE TABLE privacy_erasure_receipts (
  id uuid PRIMARY KEY,
  account_fingerprint text NOT NULL
    CHECK (char_length(account_fingerprint) = 64),
  artifact_erasure_completed boolean NOT NULL,
  database_erasure_completed boolean NOT NULL,
  completed_at timestamptz NOT NULL
);

CREATE INDEX privacy_erasure_receipts_completed
  ON privacy_erasure_receipts(completed_at DESC);
