\set ON_ERROR_STOP on
\pset pager off

-- Aggregate-only beta operating report. It emits no account IDs, emails,
-- topics, source URLs, lesson text, interview notes, or opaque note references.
-- Every beta gate is restricted to accounts explicitly classified as real
-- users. A missing classification is intentionally excluded.

\echo '=== design-partner evidence classification coverage ==='
SELECT
  COALESCE(classification.evidence_class, 'unclassified') AS evidence_class,
  count(*) AS participants
FROM design_partner_participants participant
LEFT JOIN current_account_evidence_classifications classification
  ON classification.account_id = participant.account_id
GROUP BY COALESCE(classification.evidence_class, 'unclassified')
ORDER BY evidence_class;

\echo '=== cohort recruitment and value milestones ==='
SELECT
  cohort_label,
  count(*) AS selected,
  count(*) FILTER (WHERE research_consent_at IS NOT NULL) AS consented,
  count(*) FILTER (WHERE onboarded_at IS NOT NULL) AS onboarded,
  count(*) FILTER (WHERE value_cycle_completed_at IS NOT NULL) AS completed_value_cycle,
  count(*) FILTER (WHERE payment_asked_at IS NOT NULL) AS asked_after_value,
  count(*) FILTER (WHERE status = 'active') AS active,
  count(*) FILTER (WHERE status = 'churned') AS churned
FROM design_partner_participants
WHERE account_id IN (
  SELECT account_id FROM current_account_evidence_classifications
  WHERE evidence_class = 'real_user'
)
GROUP BY cohort_label
ORDER BY cohort_label;

\echo '=== uncoached setup observation ==='
SELECT
  date_trunc('week', occurred_at)::date AS week,
  count(*) AS observed,
  count(*) FILTER (WHERE NOT coached) AS uncoached,
  count(*) FILTER (WHERE coached) AS coached,
  round(100.0 * count(*) FILTER (WHERE blocked_reason_code IS NULL)
    / NULLIF(count(*), 0), 1) AS unblocked_percent
FROM design_partner_sessions
WHERE session_type = 'setup_observation'
  AND account_id IN (
    SELECT account_id FROM current_account_evidence_classifications
    WHERE evidence_class = 'real_user'
  )
GROUP BY 1
ORDER BY 1 DESC;

\echo '=== setup blockers (stable taxonomy) ==='
SELECT blocked_reason_code, count(*) AS sessions
FROM design_partner_sessions
WHERE session_type = 'setup_observation' AND blocked_reason_code IS NOT NULL
  AND account_id IN (
    SELECT account_id FROM current_account_evidence_classifications
    WHERE evidence_class = 'real_user'
  )
GROUP BY blocked_reason_code
ORDER BY sessions DESC, blocked_reason_code;

\echo '=== weekly outcome interviews ==='
SELECT
  date_trunc('week', occurred_at)::date AS week,
  count(*) AS interviews,
  round(avg(outcome_score), 2) AS average_outcome_score,
  round(100.0 * count(*) FILTER (WHERE worth_time)
    / NULLIF(count(*) FILTER (WHERE worth_time IS NOT NULL), 0), 1) AS worth_time_percent
FROM design_partner_sessions
WHERE session_type = 'weekly_outcome_interview'
  AND account_id IN (
    SELECT account_id FROM current_account_evidence_classifications
    WHERE evidence_class = 'real_user'
  )
GROUP BY 1
ORDER BY 1 DESC;

\echo '=== weekly source and lesson quality sample ==='
SELECT
  date_trunc('week', sampled_at)::date AS week,
  count(*) AS sampled,
  round(avg(source_quality_score), 2) AS source_quality,
  round(avg(lesson_quality_score), 2) AS lesson_quality,
  sum(unsupported_claim_count) AS unsupported_claims,
  sum(citation_issue_count) AS citation_issues,
  round(100.0 * count(*) FILTER (WHERE time_fit) / NULLIF(count(*), 0), 1) AS time_fit_percent,
  count(*) FILTER (WHERE verdict = 'block') AS blocked
FROM design_partner_quality_samples sample
JOIN issues issue ON issue.id = sample.issue_id
JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
WHERE newsletter.owner_account_id IN (
  SELECT account_id FROM current_account_evidence_classifications
  WHERE evidence_class = 'real_user'
)
GROUP BY 1
ORDER BY 1 DESC;

\echo '=== beta week-four repeated value ==='
WITH cohorts AS (
  SELECT account_id, cohort_label, onboarded_at
  FROM design_partner_participants
  WHERE onboarded_at IS NOT NULL AND onboarded_at <= now() - interval '28 days'
    AND account_id IN (
      SELECT account_id FROM current_account_evidence_classifications
      WHERE evidence_class = 'real_user'
    )
), activity AS (
  SELECT
    cohorts.account_id,
    cohorts.cohort_label,
    count(DISTINCT date_trunc('week', events.occurred_at)) FILTER (
      WHERE events.event_name IN ('lesson_completed', 'review_attempted')
        AND events.occurred_at >= cohorts.onboarded_at
        AND events.occurred_at < cohorts.onboarded_at + interval '28 days'
    ) AS active_weeks
  FROM cohorts
  LEFT JOIN product_events events ON events.account_id = cohorts.account_id
  GROUP BY cohorts.account_id, cohorts.cohort_label
)
SELECT
  cohort_label,
  count(*) AS mature_partners,
  count(*) FILTER (WHERE active_weeks >= 2) AS repeated_value,
  round(100.0 * count(*) FILTER (WHERE active_weeks >= 2)
    / NULLIF(count(*), 0), 1) AS repeated_value_percent
FROM activity
GROUP BY cohort_label
ORDER BY cohort_label;

\echo '=== non-conversion and cancellation reasons ==='
SELECT context, reason_code, count(*) AS responses
FROM billing_feedback
WHERE account_id IN (
  SELECT account_id FROM current_account_evidence_classifications
  WHERE evidence_class = 'real_user'
)
GROUP BY context, reason_code
ORDER BY context, responses DESC, reason_code;

\echo 'Payment and margin evidence: run scripts/billing-economics.sql alongside this report.'
