package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) AllowRequest(
	ctx context.Context,
	bucketKey, action string,
	window time.Duration,
	limit int,
	now time.Time,
) (bool, error) {
	if bucketKey == "" || action == "" || window < time.Second || limit < 1 {
		return false, errors.New("rate limit parameters are invalid")
	}
	windowSeconds := int64(window / time.Second)
	windowStart := time.Unix((now.Unix()/windowSeconds)*windowSeconds, 0).UTC()
	var count int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO request_rate_buckets (
			bucket_key, action, window_start, request_count
		)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (bucket_key, action, window_start) DO UPDATE
		SET request_count = request_rate_buckets.request_count + 1
		RETURNING request_count
	`, bucketKey, action, windowStart).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("apply request rate limit: %w", err)
	}
	return count <= limit, nil
}

func (s *Store) BeginWebhook(
	ctx context.Context,
	id, eventType string,
	now time.Time,
) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events (id, event_type, received_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			event_type = EXCLUDED.event_type,
			received_at = EXCLUDED.received_at,
			error = NULL
		WHERE webhook_events.processed_at IS NULL
		  AND webhook_events.received_at <= EXCLUDED.received_at - interval '5 minutes'
	`, id, eventType, now)
	if err != nil {
		return false, fmt.Errorf("record webhook: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) CompleteWebhook(
	ctx context.Context,
	id string,
	processErr error,
	now time.Time,
) error {
	if processErr != nil {
		// Failed webhook work must be retryable. Successful event IDs remain
		// durable for idempotency; failed attempts are released for redelivery.
		_, err := s.pool.Exec(ctx, "DELETE FROM webhook_events WHERE id = $1", id)
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processed_at = $2, error = NULL WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("complete webhook: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

type DeletionClaim struct {
	AccountID string
	Token     string
	ExpiresAt time.Time
}

func (s *Store) ClaimAccountDeletion(
	ctx context.Context,
	now time.Time,
	duration time.Duration,
) (*DeletionClaim, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var accountID string
	err = tx.QueryRow(ctx, `
		SELECT account_id::text
		FROM account_deletion_queue
		WHERE completed_at IS NULL AND available_at <= $1
		  AND (claim_expires_at IS NULL OR claim_expires_at <= $1)
		ORDER BY available_at, account_id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, now).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	token := uuid.New()
	expires := now.Add(duration)
	if _, err := tx.Exec(ctx, `
		UPDATE account_deletion_queue SET
			claim_token = $2, claim_expires_at = $3,
			attempt_count = attempt_count + 1, error = NULL
		WHERE account_id = $1
	`, accountID, token, expires); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &DeletionClaim{
		AccountID: accountID, Token: token.String(), ExpiresAt: expires,
	}, nil
}

func (s *Store) CompleteAccountDeletion(
	ctx context.Context,
	accountID, token string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE account_deletion_queue SET
			completed_at = $3, claim_token = NULL, claim_expires_at = NULL,
			error = NULL
		WHERE account_id = $1 AND claim_token = $2
	`, accountID, token, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) FailAccountDeletion(
	ctx context.Context,
	accountID, token string,
	cause error,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE account_deletion_queue SET
			available_at = $3 + make_interval(
				secs => LEAST(86400, 60 * power(2, GREATEST(0, attempt_count - 1)))::int
			),
			claim_token = NULL, claim_expires_at = NULL, error = $4
		WHERE account_id = $1 AND claim_token = $2
	`, accountID, token, now, safeStoreError(cause))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) SetGenerationPaused(
	ctx context.Context,
	paused bool,
	reason string,
) error {
	if !paused {
		reason = ""
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE runtime_controls SET
			generation_paused = $1, pause_reason = NULLIF($2, ''), updated_at = now()
		WHERE id = true
	`, paused, reason)
	return err
}

// CleanupOperationalState removes ephemeral coordination history after its
// audit and replay windows have elapsed. Product records and delivery receipts
// are deliberately not included.
func (s *Store) CleanupOperationalState(
	ctx context.Context,
	before time.Time,
) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	rateTag, err := tx.Exec(ctx, `
		DELETE FROM request_rate_buckets WHERE window_start < $1
	`, before)
	if err != nil {
		return 0, fmt.Errorf("clean request rate buckets: %w", err)
	}
	webhookTag, err := tx.Exec(ctx, `
		DELETE FROM webhook_events
		WHERE processed_at IS NOT NULL AND processed_at < $1
	`, before)
	if err != nil {
		return 0, fmt.Errorf("clean webhook events: %w", err)
	}
	deletionTag, err := tx.Exec(ctx, `
		DELETE FROM account_deletion_queue
		WHERE completed_at IS NOT NULL AND completed_at < $1
	`, before)
	if err != nil {
		return 0, fmt.Errorf("clean deletion queue: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit operational cleanup: %w", err)
	}
	return rateTag.RowsAffected() +
		webhookTag.RowsAffected() +
		deletionTag.RowsAffected(), nil
}
