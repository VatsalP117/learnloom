CREATE TABLE account_evidence_classifications (
  id bigserial PRIMARY KEY,
  account_id uuid NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  evidence_class text NOT NULL
    CHECK (evidence_class IN ('real_user', 'founder', 'test')),
  reason_code text NOT NULL
    CHECK (reason_code IN (
      'external_customer',
      'external_design_partner',
      'founder_or_team',
      'automated_test',
      'manual_test',
      'monitoring_probe',
      'correction'
    )),
  classified_by text NOT NULL
    CHECK (char_length(classified_by) BETWEEN 1 AND 100),
  evidence_reference text NOT NULL
    CHECK (char_length(evidence_reference) BETWEEN 1 AND 240),
  classified_at timestamptz NOT NULL DEFAULT now(),
  CHECK (
    (evidence_class = 'real_user' AND reason_code IN (
      'external_customer', 'external_design_partner', 'correction'
    )) OR
    (evidence_class = 'founder' AND reason_code IN (
      'founder_or_team', 'correction'
    )) OR
    (evidence_class = 'test' AND reason_code IN (
      'automated_test', 'manual_test', 'monitoring_probe', 'correction'
    ))
  )
);

CREATE INDEX account_evidence_classifications_current
  ON account_evidence_classifications(account_id, classified_at DESC, id DESC);

CREATE VIEW current_account_evidence_classifications AS
SELECT DISTINCT ON (account_id)
  account_id,
  evidence_class,
  reason_code,
  classified_by,
  evidence_reference,
  classified_at
FROM account_evidence_classifications
ORDER BY account_id, classified_at DESC, id DESC;

COMMENT ON TABLE account_evidence_classifications IS
  'Append-only operator classifications used to separate real-user launch evidence from founder and test traffic. Absence means unclassified and is excluded from real-user gates.';
