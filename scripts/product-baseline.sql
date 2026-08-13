\set ON_ERROR_STOP on
\pset pager off

-- Learnloom Phase 0 product baseline.
--
-- Read-only and content-free: this report emits aggregate counts, rates,
-- latency, failure taxonomy, and model economics. It never selects email,
-- topic, source, lesson text, or Account identifiers.
--
-- Run against an authorized database connection:
--   psql "$DATABASE_URL" -X -f scripts/product-baseline.sql
--
-- Launch-gate metrics below are fail closed: only accounts whose latest
-- append-only operator classification is `real_user` are included. Founder,
-- test, and unclassified traffic are reported separately and never promoted
-- into real-user evidence by an email/domain heuristic.

\echo '=== deployment/schema context ==='
SELECT
  now() AT TIME ZONE 'UTC' AS measured_at_utc,
  current_database() AS database_name,
  current_setting('server_version') AS postgres_version,
  COALESCE((SELECT max(version) FROM schema_migrations), 0) AS schema_version;

\echo '=== evidence classification coverage (all current accounts) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id)
    account_id,
    evidence_class,
    reason_code,
    classified_at
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), account_event_counts AS (
  SELECT account_id, count(*) AS trailing_30_day_events
  FROM product_events
  WHERE occurred_at >= now() - interval '30 days'
  GROUP BY account_id
)
SELECT
  COALESCE(classification.evidence_class, 'unclassified') AS evidence_class,
  count(*) AS accounts,
  count(*) FILTER (WHERE account.status = 'active') AS active_accounts,
  COALESCE(sum(events.trailing_30_day_events), 0) AS trailing_30_day_events
FROM accounts account
LEFT JOIN latest_classification classification
  ON classification.account_id = account.id
LEFT JOIN account_event_counts events ON events.account_id = account.id
GROUP BY COALESCE(classification.evidence_class, 'unclassified')
ORDER BY evidence_class;

\echo 'GATE NOTE: unclassified accounts are excluded from every real-user metric below.'

\echo '=== trailing 30-day funnel events (real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), event_counts AS (
  SELECT
    event_name,
    count(*) AS events,
    count(DISTINCT account_id) AS accounts
  FROM product_events
  WHERE occurred_at >= now() - interval '30 days'
    AND account_id IN (SELECT account_id FROM real_users)
  GROUP BY event_name
)
SELECT event_name, events, accounts
FROM event_counts
ORDER BY CASE event_name
  WHEN 'signup_completed' THEN 1
  WHEN 'onboarding_started' THEN 2
  WHEN 'onboarding_intent_completed' THEN 3
  WHEN 'onboarding_sources_completed' THEN 4
  WHEN 'source_preview_reached' THEN 5
  WHEN 'onboarding_confirmed' THEN 6
  WHEN 'stream_created' THEN 7
  WHEN 'lesson_generated' THEN 8
  WHEN 'preparation_wait_exited' THEN 9
  WHEN 'lesson_opened' THEN 10
  WHEN 'lesson_completed' THEN 11
  WHEN 'review_attempted' THEN 12
  WHEN 'first_retrieval_completed' THEN 13
  WHEN 'activation_completed' THEN 14
  WHEN 'search_indexing_enabled' THEN 15
  ELSE 99
END;

\echo '=== trailing 30-day Account funnel conversion (real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), cohort AS (
  SELECT account_id, min(occurred_at) AS signed_up_at
  FROM product_events
  WHERE event_name = 'signup_completed'
    AND occurred_at >= now() - interval '30 days'
    AND account_id IN (SELECT account_id FROM real_users)
  GROUP BY account_id
), milestones AS (
  SELECT
    cohort.account_id,
    cohort.signed_up_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'source_preview_reached'
    ) AS source_preview_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'stream_created'
    ) AS stream_created_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'lesson_generated'
    ) AS lesson_generated_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'lesson_opened'
    ) AS lesson_opened_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'lesson_completed'
    ) AS lesson_completed_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'review_attempted'
    ) AS review_attempted_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'activation_completed'
    ) AS activation_completed_at
  FROM cohort
  LEFT JOIN product_events events
    ON events.account_id = cohort.account_id
   AND events.occurred_at >= cohort.signed_up_at
  GROUP BY cohort.account_id, cohort.signed_up_at
), totals AS (
  SELECT
    count(*) AS signups,
    count(source_preview_at) AS reached_preview,
    count(stream_created_at) AS created_stream,
    count(lesson_generated_at) AS generated_lesson,
    count(lesson_opened_at) AS opened_lesson,
    count(lesson_completed_at) AS completed_lesson,
    count(review_attempted_at) AS attempted_review,
    count(activation_completed_at) AS activated
  FROM milestones
)
SELECT
  signups,
  reached_preview,
  round(100.0 * reached_preview / NULLIF(signups, 0), 1) AS signup_to_preview_pct,
  created_stream,
  round(100.0 * created_stream / NULLIF(signups, 0), 1) AS signup_to_stream_pct,
  generated_lesson,
  round(100.0 * generated_lesson / NULLIF(signups, 0), 1) AS signup_to_generated_pct,
  opened_lesson,
  round(100.0 * opened_lesson / NULLIF(signups, 0), 1) AS signup_to_opened_pct,
  completed_lesson,
  round(100.0 * completed_lesson / NULLIF(signups, 0), 1) AS signup_to_completed_pct,
  attempted_review,
  round(100.0 * attempted_review / NULLIF(signups, 0), 1) AS signup_to_review_pct,
  activated,
  round(100.0 * activated / NULLIF(signups, 0), 1) AS signup_to_activation_pct
FROM totals;

\echo '=== trailing 30-day onboarding journey (real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), starts AS (
  SELECT account_id, subject_id AS draft_id, min(occurred_at) AS started_at
  FROM product_events
  WHERE event_name = 'onboarding_started'
    AND occurred_at >= now() - interval '30 days'
    AND account_id IN (SELECT account_id FROM real_users)
  GROUP BY account_id, subject_id
), journey AS (
  SELECT
    starts.account_id,
    starts.draft_id,
    starts.started_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'onboarding_intent_completed'
    ) AS intent_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'onboarding_sources_completed'
    ) AS sources_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'source_preview_reached'
    ) AS preview_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'onboarding_confirmed'
    ) AS confirmed_at,
    min(events.occurred_at) FILTER (
      WHERE events.event_name = 'onboarding_abandoned'
    ) AS abandoned_at
  FROM starts
  LEFT JOIN product_events events
    ON events.account_id = starts.account_id
   AND events.occurred_at >= starts.started_at
   AND events.subject_id = starts.draft_id
  GROUP BY starts.account_id, starts.draft_id, starts.started_at
)
SELECT
  count(*) AS started,
  count(intent_at) AS completed_intent,
  count(sources_at) AS completed_sources,
  count(preview_at) AS reached_preview,
  count(confirmed_at) AS confirmed,
  count(abandoned_at) AS abandoned_or_expired,
  round(100.0 * count(preview_at) / NULLIF(count(*), 0), 1)
    AS start_to_preview_pct,
  round(100.0 * count(confirmed_at) / NULLIF(count(*), 0), 1)
    AS start_to_confirmed_pct,
  round(percentile_cont(0.5) WITHIN GROUP (
    ORDER BY extract(epoch FROM (preview_at - started_at)) / 60.0
  )::numeric, 2) AS preview_p50_minutes,
  round(percentile_cont(0.95) WITHIN GROUP (
    ORDER BY extract(epoch FROM (preview_at - started_at)) / 60.0
  )::numeric, 2) AS preview_p95_minutes
FROM journey;

\echo '=== trailing 30-day source-policy choices and preparation exits (real-user evidence only) ==='
SELECT
  CASE
    WHEN event_name = 'source_policy_selected'
      THEN split_part(subject_id, ':', 2)
    ELSE event_name
  END AS choice_or_event,
  count(*) AS events,
  count(DISTINCT account_id) AS accounts
FROM product_events
WHERE occurred_at >= now() - interval '30 days'
  AND account_id IN (
    SELECT account_id
    FROM (
      SELECT DISTINCT ON (account_id) account_id, evidence_class
      FROM account_evidence_classifications
      ORDER BY account_id, classified_at DESC, id DESC
    ) latest
    WHERE evidence_class = 'real_user'
  )
  AND event_name IN (
    'source_policy_selected', 'preparation_wait_exited'
  )
GROUP BY choice_or_event
ORDER BY choice_or_event;

\echo '=== time to preview (minutes) and first lesson milestones (hours; real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), first_events AS (
  SELECT
    account_id,
    min(occurred_at) FILTER (
      WHERE event_name = 'signup_completed'
    ) AS signup_at,
    min(occurred_at) FILTER (
      WHERE event_name = 'source_preview_reached'
    ) AS preview_at,
    min(occurred_at) FILTER (
      WHERE event_name = 'lesson_generated'
    ) AS generated_at,
    min(occurred_at) FILTER (
      WHERE event_name = 'lesson_opened'
    ) AS opened_at,
    min(occurred_at) FILTER (
      WHERE event_name = 'lesson_completed'
    ) AS completed_at
  FROM product_events
  WHERE account_id IN (SELECT account_id FROM real_users)
  GROUP BY account_id
), durations AS (
  SELECT
    extract(epoch FROM (preview_at - signup_at)) / 60.0 AS preview_minutes,
    extract(epoch FROM (generated_at - signup_at)) / 3600.0 AS generated_hours,
    extract(epoch FROM (opened_at - signup_at)) / 3600.0 AS opened_hours,
    extract(epoch FROM (completed_at - signup_at)) / 3600.0 AS completed_hours
  FROM first_events
  WHERE signup_at >= now() - interval '30 days'
)
SELECT
  round(percentile_cont(0.5) WITHIN GROUP (ORDER BY preview_minutes)::numeric, 2)
    AS preview_p50_minutes,
  round(percentile_cont(0.95) WITHIN GROUP (ORDER BY preview_minutes)::numeric, 2)
    AS preview_p95_minutes,
  round(percentile_cont(0.5) WITHIN GROUP (ORDER BY generated_hours)::numeric, 2)
    AS generated_p50_hours,
  round(percentile_cont(0.95) WITHIN GROUP (ORDER BY generated_hours)::numeric, 2)
    AS generated_p95_hours,
  round(percentile_cont(0.5) WITHIN GROUP (ORDER BY opened_hours)::numeric, 2)
    AS opened_p50_hours,
  round(percentile_cont(0.95) WITHIN GROUP (ORDER BY opened_hours)::numeric, 2)
    AS opened_p95_hours,
  round(percentile_cont(0.5) WITHIN GROUP (ORDER BY completed_hours)::numeric, 2)
    AS completed_p50_hours,
  round(percentile_cont(0.95) WITHIN GROUP (ORDER BY completed_hours)::numeric, 2)
    AS completed_p95_hours
FROM durations;

\echo '=== trailing 30-day terminal generation outcomes (real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), terminal AS (
  SELECT status, count(*) AS issues
  FROM issues issue
  JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
  WHERE issue.created_at >= now() - interval '30 days'
    AND issue.status IN ('generated', 'deferred', 'failed', 'cancelled')
    AND newsletter.owner_account_id IN (SELECT account_id FROM real_users)
  GROUP BY issue.status
), total AS (
  SELECT sum(issues) AS issues FROM terminal
)
SELECT
  terminal.status,
  terminal.issues,
  round(100.0 * terminal.issues / NULLIF(total.issues, 0), 1) AS pct
FROM terminal
CROSS JOIN total
ORDER BY terminal.status;

\echo '=== trailing 30-day failure taxonomy (real-user evidence only) ==='
SELECT
  COALESCE(issue.failure_category, 'unknown') AS failure_category,
  COALESCE(issue.failure_stage, 'unknown') AS failure_stage,
  COALESCE(issue.failure_code, 'unknown') AS failure_code,
  COALESCE(issue.failure_retryable, false) AS retryable,
  count(*) AS failed_issues,
  count(DISTINCT issue.newsletter_id) AS affected_streams
FROM issues issue
JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
WHERE issue.status = 'failed'
  AND issue.created_at >= now() - interval '30 days'
  AND newsletter.owner_account_id IN (
    SELECT account_id FROM (
      SELECT DISTINCT ON (account_id) account_id, evidence_class
      FROM account_evidence_classifications
      ORDER BY account_id, classified_at DESC, id DESC
    ) latest WHERE evidence_class = 'real_user'
  )
GROUP BY issue.failure_category, issue.failure_stage, issue.failure_code,
         issue.failure_retryable
ORDER BY failed_issues DESC, failure_category, failure_stage, failure_code;

\echo '=== trailing 30-day unregistered failure-code gate (real-user evidence only) ==='
WITH registered(code) AS (
  VALUES
    ('model_contract_unsatisfied'),
    ('model_provider_unavailable'),
    ('model_request_rejected'),
    ('model_output_truncated'),
    ('model_output_empty'),
    ('generation_interrupted'),
    ('no_new_learning_signal'),
    ('source_discovery_unavailable'),
    ('source_evidence_needs_attention'),
    ('no_worthwhile_evidence'),
    ('no_new_evidence'),
    ('worker_claim_expired')
)
SELECT
  COALESCE(issue.failure_code, 'unknown') AS failure_code,
  count(*) AS failed_issues
FROM issues issue
JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
LEFT JOIN registered ON registered.code = issue.failure_code
WHERE issue.status = 'failed'
  AND issue.created_at >= now() - interval '30 days'
  AND newsletter.owner_account_id IN (
    SELECT account_id FROM current_account_evidence_classifications
    WHERE evidence_class = 'real_user'
  )
  AND registered.code IS NULL
GROUP BY COALESCE(issue.failure_code, 'unknown')
ORDER BY failed_issues DESC, failure_code;

\echo 'GATE NOTE: any row above requires a safe fixture and registry update before LL-201 can close.'

\echo '=== trailing 30-day attempt/stage reliability (real-user evidence only) ==='
SELECT
  stage_attempt.stage,
  count(*) FILTER (WHERE stage_attempt.status = 'completed') AS completed_attempts,
  count(*) FILTER (WHERE stage_attempt.status = 'failed') AS failed_attempts,
  round(
    100.0 * count(*) FILTER (WHERE stage_attempt.status = 'failed') /
    NULLIF(count(*), 0),
    1
  ) AS failure_pct,
  round(percentile_cont(0.5) WITHIN GROUP (ORDER BY stage_attempt.duration_ms)::numeric, 0)
    AS p50_duration_ms,
  round(percentile_cont(0.95) WITHIN GROUP (ORDER BY stage_attempt.duration_ms)::numeric, 0)
    AS p95_duration_ms
FROM issue_stage_attempts stage_attempt
JOIN issue_attempts attempt ON attempt.id = stage_attempt.issue_attempt_id
JOIN issues issue ON issue.id = attempt.issue_id
JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
WHERE stage_attempt.recorded_at >= now() - interval '30 days'
  AND newsletter.owner_account_id IN (
    SELECT account_id FROM (
      SELECT DISTINCT ON (account_id) account_id, evidence_class
      FROM account_evidence_classifications
      ORDER BY account_id, classified_at DESC, id DESC
    ) latest WHERE evidence_class = 'real_user'
  )
GROUP BY stage_attempt.stage
ORDER BY stage_attempt.stage;

\echo '=== seven-day return for mature activated cohorts (real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), activation AS (
  SELECT account_id, min(occurred_at) AS activated_at
  FROM product_events
  WHERE event_name = 'activation_completed'
    AND account_id IN (SELECT account_id FROM real_users)
  GROUP BY account_id
), mature AS (
  SELECT *
  FROM activation
  WHERE activated_at >= now() - interval '90 days'
    AND activated_at < now() - interval '7 days'
), returns AS (
  SELECT
    mature.account_id,
    EXISTS (
      SELECT 1
      FROM product_events later
      WHERE later.account_id = mature.account_id
        AND later.event_name IN (
          'lesson_opened', 'lesson_completed', 'review_attempted'
        )
        AND later.occurred_at > mature.activated_at
        AND later.occurred_at <= mature.activated_at + interval '7 days'
    ) AS returned
  FROM mature
)
SELECT
  count(*) AS mature_activated_accounts,
  count(*) FILTER (WHERE returned) AS returned_accounts,
  round(
    100.0 * count(*) FILTER (WHERE returned) / NULLIF(count(*), 0),
    1
  ) AS seven_day_return_pct
FROM returns;

\echo '=== trailing 30-day model economics and outcome unit costs (real-user evidence only) ==='
WITH latest_classification AS (
  SELECT DISTINCT ON (account_id) account_id, evidence_class
  FROM account_evidence_classifications
  ORDER BY account_id, classified_at DESC, id DESC
), real_users AS (
  SELECT account_id FROM latest_classification WHERE evidence_class = 'real_user'
), economics AS (
  SELECT
    COALESCE(sum(input_tokens), 0) AS input_tokens,
    COALESCE(sum(output_tokens), 0) AS output_tokens,
    COALESCE(sum(provider_retries), 0) AS provider_retries,
    COALESCE(sum(estimated_cost_microusd), 0) AS cost_microusd
  FROM issue_stage_attempts stage_attempt
  JOIN issue_attempts attempt ON attempt.id = stage_attempt.issue_attempt_id
  JOIN issues issue ON issue.id = attempt.issue_id
  JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
  WHERE stage_attempt.recorded_at >= now() - interval '30 days'
    AND newsletter.owner_account_id IN (SELECT account_id FROM real_users)
), outcomes AS (
  SELECT
    count(*) FILTER (WHERE event_name = 'lesson_generated') AS generated,
    count(*) FILTER (WHERE event_name = 'lesson_opened') AS opened,
    count(*) FILTER (WHERE event_name = 'lesson_completed') AS completed,
    count(*) FILTER (WHERE event_name = 'review_attempted') AS reviewed
  FROM product_events
  WHERE occurred_at >= now() - interval '30 days'
    AND account_id IN (SELECT account_id FROM real_users)
)
SELECT
  input_tokens,
  output_tokens,
  provider_retries,
  round(cost_microusd / 1000000.0, 4) AS estimated_cost_usd,
  generated,
  opened,
  completed,
  reviewed,
  round(cost_microusd / 1000000.0 / NULLIF(generated, 0), 4)
    AS cost_per_generated_usd,
  round(cost_microusd / 1000000.0 / NULLIF(opened, 0), 4)
    AS cost_per_opened_usd,
  round(cost_microusd / 1000000.0 / NULLIF(completed, 0), 4)
    AS cost_per_completed_usd,
  round(cost_microusd / 1000000.0 / NULLIF(reviewed, 0), 4)
    AS cost_per_reviewed_usd
FROM economics
CROSS JOIN outcomes;

\echo '=== cost per seven-day retained lesson (mature real-user activation cohorts) ==='
WITH real_users AS (
  SELECT account_id
  FROM current_account_evidence_classifications
  WHERE evidence_class = 'real_user'
), activation AS (
  SELECT event.account_id, min(event.occurred_at) AS activated_at
  FROM product_events event
  WHERE event.event_name = 'activation_completed'
    AND event.account_id IN (SELECT account_id FROM real_users)
  GROUP BY event.account_id
), mature AS (
  SELECT account_id, activated_at
  FROM activation
  WHERE activated_at >= now() - interval '90 days'
    AND activated_at < now() - interval '7 days'
), retained_lessons AS (
  SELECT DISTINCT
    mature.account_id,
    event.subject_id AS issue_id
  FROM mature
  JOIN product_events event ON event.account_id = mature.account_id
  WHERE event.event_name IN ('lesson_completed', 'review_attempted')
    AND event.occurred_at > mature.activated_at
    AND event.occurred_at <= mature.activated_at + interval '7 days'
), cohort_cost AS (
  SELECT COALESCE(sum(stage.estimated_cost_microusd), 0) AS cost_microusd
  FROM mature
  JOIN newsletters newsletter ON newsletter.owner_account_id = mature.account_id
  JOIN issues issue ON issue.newsletter_id = newsletter.id
  JOIN issue_attempts attempt ON attempt.issue_id = issue.id
  JOIN issue_stage_attempts stage ON stage.issue_attempt_id = attempt.id
  WHERE stage.recorded_at >= mature.activated_at
    AND stage.recorded_at <= mature.activated_at + interval '7 days'
), totals AS (
  SELECT
    (SELECT count(*) FROM mature) AS mature_activated_accounts,
    count(DISTINCT retained_lessons.account_id) AS retained_accounts,
    count(*) AS retained_lessons
  FROM retained_lessons
)
SELECT
  totals.mature_activated_accounts,
  totals.retained_accounts,
  totals.retained_lessons,
  round(cohort_cost.cost_microusd / 1000000.0, 4) AS cohort_generation_cost_usd,
  round(
    cohort_cost.cost_microusd / 1000000.0 /
    NULLIF(totals.retained_lessons, 0),
    4
  ) AS cost_per_retained_lesson_usd
FROM totals
CROSS JOIN cohort_cost;

\echo '=== baseline complete ==='
