CREATE TABLE design_partner_participants (
  account_id uuid PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
  cohort_label text NOT NULL CHECK (char_length(cohort_label) BETWEEN 1 AND 64),
  status text NOT NULL CHECK (status IN (
    'invited', 'enrolled', 'active', 'declined', 'withdrawn', 'churned', 'completed'
  )),
  persona_match boolean NOT NULL,
  research_consent_at timestamptz,
  consent_withdrawn_at timestamptz,
  onboarded_at timestamptz,
  value_cycle_completed_at timestamptz,
  payment_asked_at timestamptz,
  outcome_story_permission_at timestamptz,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  CHECK (research_consent_at IS NOT NULL OR status IN ('invited', 'declined')),
  CHECK (onboarded_at IS NULL OR research_consent_at IS NOT NULL),
  CHECK (payment_asked_at IS NULL OR value_cycle_completed_at IS NOT NULL),
  CHECK (consent_withdrawn_at IS NULL OR research_consent_at IS NOT NULL)
);

CREATE INDEX design_partner_cohort_status
  ON design_partner_participants(cohort_label, status);

CREATE TABLE design_partner_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL
    REFERENCES design_partner_participants(account_id) ON DELETE CASCADE,
  session_type text NOT NULL CHECK (session_type IN (
    'setup_observation', 'weekly_outcome_interview'
  )),
  occurred_at timestamptz NOT NULL,
  coached boolean NOT NULL DEFAULT false,
  blocked_reason_code text CHECK (blocked_reason_code IS NULL OR blocked_reason_code IN (
    'authentication', 'intent_clarity', 'source_portfolio', 'generation_wait',
    'lesson_quality', 'retrieval_flow', 'navigation', 'billing', 'other'
  )),
  outcome_score integer CHECK (outcome_score BETWEEN 1 AND 5),
  worth_time boolean,
  notes_reference text CHECK (
    notes_reference IS NULL OR char_length(notes_reference) BETWEEN 1 AND 256
  ),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX design_partner_sessions_account_occurred
  ON design_partner_sessions(account_id, occurred_at DESC);

CREATE TABLE design_partner_quality_samples (
  issue_id uuid PRIMARY KEY REFERENCES issues(id) ON DELETE CASCADE,
  sampled_at timestamptz NOT NULL,
  evaluator_label text NOT NULL CHECK (char_length(evaluator_label) BETWEEN 1 AND 64),
  source_quality_score integer NOT NULL CHECK (source_quality_score BETWEEN 1 AND 4),
  lesson_quality_score integer NOT NULL CHECK (lesson_quality_score BETWEEN 1 AND 4),
  unsupported_claim_count integer NOT NULL CHECK (unsupported_claim_count >= 0),
  citation_issue_count integer NOT NULL CHECK (citation_issue_count >= 0),
  time_fit boolean NOT NULL,
  verdict text NOT NULL CHECK (verdict IN ('pass', 'revise', 'block')),
  notes_reference text CHECK (
    notes_reference IS NULL OR char_length(notes_reference) BETWEEN 1 AND 256
  ),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX design_partner_quality_sampled_at
  ON design_partner_quality_samples(sampled_at DESC, verdict);
