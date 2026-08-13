\set ON_ERROR_STOP on
\pset pager off

-- Commercial launch evidence is restricted to accounts explicitly classified
-- as real users. Founder, test, and unclassified payments cannot satisfy a
-- conversion, retention, revenue, or margin gate.

\echo '=== billing evidence classification coverage ==='
SELECT
  COALESCE(classification.evidence_class, 'unclassified') AS evidence_class,
  count(DISTINCT lifecycle.account_id) AS lifecycle_accounts,
  count(DISTINCT revenue.account_id) AS revenue_accounts
FROM accounts account
LEFT JOIN current_account_evidence_classifications classification
  ON classification.account_id = account.id
LEFT JOIN billing_lifecycle_events lifecycle
  ON lifecycle.account_id = account.id
 AND lifecycle.occurred_at >= now() - interval '30 days'
LEFT JOIN billing_revenue_events revenue
  ON revenue.account_id = account.id
 AND revenue.occurred_at >= now() - interval '30 days'
GROUP BY COALESCE(classification.evidence_class, 'unclassified')
ORDER BY evidence_class;

\echo '=== billing lifecycle funnel (real-user evidence; trailing 30 days) ==='
SELECT
  event_name,
  count(*) AS events,
  count(DISTINCT account_id) AS accounts
FROM billing_lifecycle_events
WHERE occurred_at >= now() - interval '30 days'
  AND account_id IN (
    SELECT account_id FROM current_account_evidence_classifications
    WHERE evidence_class = 'real_user'
  )
GROUP BY event_name
ORDER BY event_name;

\echo '=== trial conversion and churn cohorts (real-user evidence only) ==='
WITH account_dates AS (
  SELECT
    account_id,
    min(occurred_at) FILTER (WHERE event_name = 'trial_started') AS trial_started_at,
    min(occurred_at) FILTER (WHERE event_name = 'payment_succeeded') AS first_paid_at,
    min(occurred_at) FILTER (WHERE event_name = 'subscription_canceled') AS canceled_at
  FROM billing_lifecycle_events
  WHERE account_id IN (
    SELECT account_id FROM current_account_evidence_classifications
    WHERE evidence_class = 'real_user'
  )
  GROUP BY account_id
)
SELECT
  date_trunc('month', trial_started_at)::date AS trial_cohort,
  count(*) AS trials,
  count(first_paid_at) AS converted,
  round(100.0 * count(first_paid_at) / NULLIF(count(*), 0), 1) AS conversion_percent,
  count(canceled_at) FILTER (WHERE first_paid_at IS NOT NULL) AS paid_cancellations
FROM account_dates
WHERE trial_started_at IS NOT NULL
GROUP BY 1
ORDER BY 1 DESC;

\echo '=== recognized cash revenue and complete recorded COGS by currency (trailing 30 days) ==='
WITH revenue AS (
  SELECT
    currency_code,
    sum(CASE WHEN event_type = 'payment' THEN amount_minor ELSE -amount_minor END) AS net_revenue_minor,
    sum(CASE WHEN event_type = 'payment' THEN COALESCE(provider_fee_minor, 0) ELSE 0 END) AS provider_fees_minor
  FROM billing_revenue_events
  WHERE occurred_at >= now() - interval '30 days'
    AND account_id IN (
      SELECT account_id FROM current_account_evidence_classifications
      WHERE evidence_class = 'real_user'
    )
  GROUP BY currency_code
), model_cogs AS (
  SELECT COALESCE(sum(stage.estimated_cost_microusd), 0) AS cost_microusd
  FROM issue_stage_attempts stage
	JOIN issue_attempts attempt ON attempt.id = stage.issue_attempt_id
	JOIN issues issue ON issue.id = attempt.issue_id
  JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
  JOIN account_billing billing ON billing.account_id = newsletter.owner_account_id
	WHERE stage.recorded_at >= now() - interval '30 days'
    AND billing.plan_id = 'pro'
    AND newsletter.owner_account_id IN (
      SELECT account_id FROM current_account_evidence_classifications
      WHERE evidence_class = 'real_user'
    )
), operational_cogs AS (
  SELECT
    COALESCE(sum(cost_microusd) FILTER (WHERE account_id IS NOT NULL), 0) AS attributed_microusd,
    COALESCE(sum(cost_microusd) FILTER (WHERE account_id IS NULL), 0) AS shared_unallocated_microusd
  FROM operational_cogs_events
  WHERE occurred_at >= now() - interval '30 days'
    AND (
      account_id IS NULL OR account_id IN (
        SELECT account_id FROM current_account_evidence_classifications
        WHERE evidence_class = 'real_user'
      )
    )
), cogs_completeness AS (
  SELECT
    count(DISTINCT category) FILTER (
      WHERE category IN ('search', 'email', 'storage', 'support', 'infrastructure')
    ) = 5 AS all_required_categories_recorded
  FROM operational_cogs_events
  WHERE occurred_at >= now() - interval '30 days'
)
SELECT
  revenue.currency_code,
  revenue.net_revenue_minor,
  revenue.provider_fees_minor,
  round(model_cogs.cost_microusd / 1000000.0, 4) AS model_cogs_usd,
  round(operational_cogs.attributed_microusd / 1000000.0, 4) AS attributed_operational_cogs_usd,
  round(operational_cogs.shared_unallocated_microusd / 1000000.0, 4) AS shared_unallocated_cogs_usd,
  cogs_completeness.all_required_categories_recorded AS cogs_categories_complete,
  CASE WHEN revenue.currency_code = 'USD'
    AND revenue.net_revenue_minor > 0
    AND cogs_completeness.all_required_categories_recorded
    AND operational_cogs.shared_unallocated_microusd = 0 THEN
    round(100.0 * (
	  revenue.net_revenue_minor / 100.0
	  - revenue.provider_fees_minor / 100.0
	  - model_cogs.cost_microusd / 1000000.0
	  - operational_cogs.attributed_microusd / 1000000.0
    ) / NULLIF(revenue.net_revenue_minor / 100.0, 0), 1)
  END AS attributed_gross_margin_before_shared_percent
FROM revenue CROSS JOIN model_cogs CROSS JOIN operational_cogs CROSS JOIN cogs_completeness
ORDER BY revenue.currency_code;

\echo '=== operational COGS completeness by category (real-user and shared only; trailing 30 days) ==='
SELECT
  category,
  count(*) AS entries,
  round(sum(cost_microusd) / 1000000.0, 4) AS cost_usd,
  count(*) FILTER (WHERE account_id IS NULL) AS unallocated_entries
FROM operational_cogs_events
WHERE occurred_at >= now() - interval '30 days'
  AND (
    account_id IS NULL OR account_id IN (
      SELECT account_id FROM current_account_evidence_classifications
      WHERE evidence_class = 'real_user'
    )
  )
GROUP BY category
ORDER BY category;

\echo '=== retained paid cohort unit economics (USD; trailing 90-day payment cohorts) ==='
WITH first_payment AS (
  SELECT account_id, min(occurred_at) AS first_paid_at
  FROM billing_revenue_events
  WHERE event_type = 'payment' AND currency_code = 'USD'
    AND account_id IN (
      SELECT account_id FROM current_account_evidence_classifications
      WHERE evidence_class = 'real_user'
    )
  GROUP BY account_id
), retained AS (
  SELECT
    first_payment.account_id,
    date_trunc('month', first_payment.first_paid_at)::date AS cohort_month,
    EXISTS (
      SELECT 1 FROM product_events activity
      WHERE activity.account_id = first_payment.account_id
        AND activity.event_name IN ('lesson_completed', 'review_attempted')
        AND activity.occurred_at >= first_payment.first_paid_at + interval '21 days'
        AND activity.occurred_at < first_payment.first_paid_at + interval '35 days'
    ) AS retained_week_four
  FROM first_payment
  WHERE first_payment.first_paid_at >= now() - interval '90 days'
), account_revenue AS (
  SELECT account_id,
    sum(CASE WHEN event_type = 'payment' THEN amount_minor ELSE -amount_minor END) AS revenue_minor,
    sum(CASE WHEN event_type = 'payment' THEN COALESCE(provider_fee_minor, 0) ELSE 0 END) AS fee_minor
  FROM billing_revenue_events
  WHERE currency_code = 'USD'
  GROUP BY account_id
), account_model AS (
  SELECT newsletter.owner_account_id AS account_id,
         sum(stage.estimated_cost_microusd) AS cost_microusd
  FROM issue_stage_attempts stage
  JOIN issue_attempts attempt ON attempt.id = stage.issue_attempt_id
  JOIN issues issue ON issue.id = attempt.issue_id
  JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
  GROUP BY newsletter.owner_account_id
), account_operations AS (
  SELECT account_id, sum(cost_microusd) AS cost_microusd
  FROM operational_cogs_events
  WHERE account_id IS NOT NULL
  GROUP BY account_id
)
SELECT
  retained.cohort_month,
  count(*) AS paid_accounts,
  count(*) FILTER (WHERE retained.retained_week_four) AS retained_paid_accounts,
  sum(account_revenue.revenue_minor) AS net_revenue_minor,
  round((sum(COALESCE(account_model.cost_microusd, 0))
    + sum(COALESCE(account_operations.cost_microusd, 0))) / 1000000.0, 4)
    AS attributed_nonpayment_cogs_usd,
  sum(account_revenue.fee_minor) AS provider_fees_minor
FROM retained
JOIN account_revenue USING (account_id)
LEFT JOIN account_model USING (account_id)
LEFT JOIN account_operations USING (account_id)
GROUP BY retained.cohort_month
ORDER BY retained.cohort_month DESC;

\echo 'GATE NOTE: gross margin is not launch-valid while required categories are missing or shared costs remain unallocated.'
