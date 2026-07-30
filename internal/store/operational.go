package store

import (
	"context"
	"fmt"
	"time"
)

type OperationalSnapshot struct {
	QueuedIssues           int64
	OldestQueuedIssueAge   time.Duration
	PendingDeliveries      int64
	OldestDeliveryAge      time.Duration
	PendingWeeklyRecaps    int64
	PendingDeletions       int64
	DatabaseAcquired       int32
	DatabaseIdle           int32
	DatabaseTotal          int32
	DatabaseMax            int32
	DatabaseAcquireCount   int64
	DatabaseAcquireNanos   int64
	ModelInputTokens       int64
	ModelOutputTokens      int64
	ModelProviderRetries   int64
	ModelCostMicroUSD      int64
	ModelCostTodayMicroUSD int64
	LessonsGenerated       int64
	LessonsOpened          int64
	LessonsCompleted       int64
	RetainedLearners       int64
}

func (s *Store) OperationalSnapshot(
	ctx context.Context,
	now time.Time,
) (OperationalSnapshot, error) {
	var snapshot OperationalSnapshot
	var issueAgeSeconds, deliveryAgeSeconds float64
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
		  )
	`, now).Scan(
		&snapshot.QueuedIssues,
		&issueAgeSeconds,
		&snapshot.PendingDeliveries,
		&deliveryAgeSeconds,
		&snapshot.PendingWeeklyRecaps,
		&snapshot.PendingDeletions,
		&snapshot.ModelInputTokens,
		&snapshot.ModelOutputTokens,
		&snapshot.ModelProviderRetries,
		&snapshot.ModelCostMicroUSD,
		&snapshot.ModelCostTodayMicroUSD,
		&snapshot.LessonsGenerated,
		&snapshot.LessonsOpened,
		&snapshot.LessonsCompleted,
		&snapshot.RetainedLearners,
	)
	if err != nil {
		return OperationalSnapshot{}, fmt.Errorf("load operational snapshot: %w", err)
	}
	snapshot.OldestQueuedIssueAge = max(0, time.Duration(issueAgeSeconds*float64(time.Second)))
	snapshot.OldestDeliveryAge = max(0, time.Duration(deliveryAgeSeconds*float64(time.Second)))
	stats := s.pool.Stat()
	snapshot.DatabaseAcquired = stats.AcquiredConns()
	snapshot.DatabaseIdle = stats.IdleConns()
	snapshot.DatabaseTotal = stats.TotalConns()
	snapshot.DatabaseMax = stats.MaxConns()
	snapshot.DatabaseAcquireCount = stats.AcquireCount()
	snapshot.DatabaseAcquireNanos = stats.AcquireDuration().Nanoseconds()
	return snapshot, nil
}
