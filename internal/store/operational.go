package store

import (
	"context"
	"fmt"
	"time"
)

type OperationalSnapshot struct {
	QueuedIssues                int64
	OldestQueuedIssueAge        time.Duration
	PendingDeliveries           int64
	OldestDeliveryAge           time.Duration
	PendingWeeklyRecaps         int64
	PendingDeletions            int64
	PendingArtifactCleanups     int64
	OldestArtifactCleanupAge    time.Duration
	DatabaseAcquired            int32
	DatabaseIdle                int32
	DatabaseTotal               int32
	DatabaseMax                 int32
	DatabaseAcquireCount        int64
	DatabaseAcquireNanos        int64
	ModelInputTokens            int64
	ModelOutputTokens           int64
	ModelProviderRetries        int64
	ModelCostMicroUSD           int64
	ModelCostTodayMicroUSD      int64
	LessonsGenerated            int64
	LessonsDeferred             int64
	LessonsOpened               int64
	LessonsCompleted            int64
	RetainedLearners            int64
	AccountsConsecutiveFailures int64
	PendingBillingWebhooks      int64
	OldestBillingWebhookAge     time.Duration
	BillingAccountsInGrace      int64
	BillingAccountsPaused       int64
	PendingBillingCheckouts     int64
}

func (s *Store) OperationalSnapshot(
	ctx context.Context,
	now time.Time,
) (OperationalSnapshot, error) {
	var snapshot OperationalSnapshot
	var issueAgeSeconds, deliveryAgeSeconds, artifactCleanupAgeSeconds, billingWebhookAgeSeconds float64
	err := s.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM issues WHERE status = 'queued'),
		  COALESCE((SELECT extract(epoch FROM ($1 - min(available_at)))
		            FROM issues WHERE status = 'queued'), 0),
		  (SELECT count(*) FROM delivery_receipts
		   WHERE status IN ('pending', 'failed')),
		  COALESCE((SELECT extract(epoch FROM ($1 - min(available_at)))
		            FROM delivery_receipts
		            WHERE status IN ('pending', 'failed')), 0),
		  (SELECT count(*) FROM weekly_recaps
		   WHERE status IN ('pending', 'failed')),
		  (SELECT count(*) FROM account_deletion_queue
		   WHERE completed_at IS NULL),
		  (SELECT count(*) FROM artifact_cleanup_queue cleanup
		   WHERE cleanup.available_at <= $1
		     AND NOT EXISTS (
		       SELECT 1 FROM issues issue
		       WHERE issue.artifact_key = cleanup.artifact_key
		     )),
		  COALESCE((SELECT extract(epoch FROM ($1 - min(cleanup.available_at)))
		            FROM artifact_cleanup_queue cleanup
		            WHERE cleanup.available_at <= $1
		              AND NOT EXISTS (
		                SELECT 1 FROM issues issue
		                WHERE issue.artifact_key = cleanup.artifact_key
		              )), 0),
		  (SELECT COALESCE(sum(input_tokens), 0) FROM issue_stage_attempts),
		  (SELECT COALESCE(sum(output_tokens), 0) FROM issue_stage_attempts),
		  (SELECT COALESCE(sum(provider_retries), 0) FROM issue_stage_attempts),
		  (SELECT COALESCE(sum(estimated_cost_microusd), 0)
		   FROM issue_stage_attempts),
		  (SELECT COALESCE(sum(estimated_cost_microusd), 0)
		   FROM issue_stage_attempts
		   WHERE recorded_at >=
		     date_trunc('day', $1 AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'),
		  (SELECT count(*) FROM product_events
		   WHERE event_name = 'lesson_generated'),
		  (SELECT count(*) FROM issues WHERE status = 'deferred'),
		  (SELECT count(*) FROM product_events
		   WHERE event_name = 'lesson_opened'),
		  (SELECT count(*) FROM product_events
		   WHERE event_name = 'lesson_completed'),
		  (
		    SELECT count(*)
		    FROM (
		      SELECT first.account_id
		      FROM (
		        SELECT account_id, min(occurred_at) AS first_completed_at
		        FROM product_events
		        WHERE event_name = 'lesson_completed'
		        GROUP BY account_id
		      ) first
		      WHERE first.first_completed_at <= $1 - interval '7 days'
		        AND EXISTS (
		          SELECT 1 FROM product_events later
		          WHERE later.account_id = first.account_id
		            AND later.event_name IN (
		              'lesson_opened', 'lesson_completed', 'review_attempted'
		            )
		            AND later.occurred_at >=
		              first.first_completed_at + interval '7 days'
		        )
		    ) retained
		  ),
		  (
		    SELECT count(*)
		    FROM (
		      SELECT owner_account_id
		      FROM (
		        SELECT n.owner_account_id, i.status,
		               row_number() OVER (
		                 PARTITION BY n.owner_account_id
		                 ORDER BY i.created_at DESC, i.id DESC
		               ) AS position
		        FROM issues i
		        JOIN newsletters n ON n.id = i.newsletter_id
		        WHERE i.status IN ('generated', 'failed', 'deferred')
		      ) recent
		      WHERE position <= 2
		      GROUP BY owner_account_id
		      HAVING count(*) = 2 AND bool_and(status = 'failed')
		    ) consecutive
		  ),
		  (SELECT count(*) FROM billing_webhook_events WHERE processed_at IS NULL),
		  COALESCE((SELECT extract(epoch FROM ($1 - min(received_at)))
		            FROM billing_webhook_events WHERE processed_at IS NULL), 0),
		  (SELECT count(*) FROM account_billing WHERE entitlement_status = 'grace'),
		  (SELECT count(*) FROM account_billing WHERE entitlement_status = 'generation_paused'),
		  (SELECT count(*) FROM billing_checkout_sessions WHERE status = 'pending')
	`, now).Scan(
		&snapshot.QueuedIssues,
		&issueAgeSeconds,
		&snapshot.PendingDeliveries,
		&deliveryAgeSeconds,
		&snapshot.PendingWeeklyRecaps,
		&snapshot.PendingDeletions,
		&snapshot.PendingArtifactCleanups,
		&artifactCleanupAgeSeconds,
		&snapshot.ModelInputTokens,
		&snapshot.ModelOutputTokens,
		&snapshot.ModelProviderRetries,
		&snapshot.ModelCostMicroUSD,
		&snapshot.ModelCostTodayMicroUSD,
		&snapshot.LessonsGenerated,
		&snapshot.LessonsDeferred,
		&snapshot.LessonsOpened,
		&snapshot.LessonsCompleted,
		&snapshot.RetainedLearners,
		&snapshot.AccountsConsecutiveFailures,
		&snapshot.PendingBillingWebhooks,
		&billingWebhookAgeSeconds,
		&snapshot.BillingAccountsInGrace,
		&snapshot.BillingAccountsPaused,
		&snapshot.PendingBillingCheckouts,
	)
	if err != nil {
		return OperationalSnapshot{}, fmt.Errorf("load operational snapshot: %w", err)
	}
	snapshot.OldestQueuedIssueAge = max(0, time.Duration(issueAgeSeconds*float64(time.Second)))
	snapshot.OldestDeliveryAge = max(0, time.Duration(deliveryAgeSeconds*float64(time.Second)))
	snapshot.OldestArtifactCleanupAge = max(0, time.Duration(artifactCleanupAgeSeconds*float64(time.Second)))
	snapshot.OldestBillingWebhookAge = max(0, time.Duration(billingWebhookAgeSeconds*float64(time.Second)))
	stats := s.pool.Stat()
	snapshot.DatabaseAcquired = stats.AcquiredConns()
	snapshot.DatabaseIdle = stats.IdleConns()
	snapshot.DatabaseTotal = stats.TotalConns()
	snapshot.DatabaseMax = stats.MaxConns()
	snapshot.DatabaseAcquireCount = stats.AcquireCount()
	snapshot.DatabaseAcquireNanos = stats.AcquireDuration().Nanoseconds()
	return snapshot, nil
}
