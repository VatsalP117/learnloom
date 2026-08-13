package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ArtifactCleanupClaim struct {
	Key       string
	Token     string
	ExpiresAt time.Time
}

func (s *Store) RegisterArtifactCleanup(
	ctx context.Context,
	key, issueID string,
	now time.Time,
) error {
	if key == "" {
		return errors.New("artifact cleanup key is required")
	}
	if _, err := uuid.Parse(issueID); err != nil {
		return errors.New("artifact cleanup issue ID is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO artifact_cleanup_queue (
		  artifact_key, issue_id, available_at, created_at
		)
		VALUES ($1, $2, $3::timestamptz + interval '15 minutes', $3)
		ON CONFLICT (artifact_key) DO NOTHING
	`, key, issueID, now)
	if err != nil {
		return fmt.Errorf("register artifact cleanup: %w", err)
	}
	return nil
}

func (s *Store) ClaimArtifactCleanup(
	ctx context.Context,
	now time.Time,
	duration time.Duration,
) (*ArtifactCleanupClaim, error) {
	if duration < time.Minute {
		return nil, errors.New("artifact cleanup claim duration is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var key string
	err = tx.QueryRow(ctx, `
		SELECT cleanup.artifact_key
		FROM artifact_cleanup_queue cleanup
		WHERE cleanup.available_at <= $1
		  AND (cleanup.claim_expires_at IS NULL OR cleanup.claim_expires_at <= $1)
		  AND NOT EXISTS (
		    SELECT 1 FROM issues issue
		    WHERE issue.artifact_key = cleanup.artifact_key
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM issues issue
		    WHERE issue.id = cleanup.issue_id
		      AND issue.status = 'generating'
		      AND issue.claim_expires_at > $1
		  )
		ORDER BY cleanup.available_at, cleanup.created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, now).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim artifact cleanup: %w", err)
	}
	token := uuid.NewString()
	expiresAt := now.Add(duration)
	if _, err := tx.Exec(ctx, `
		UPDATE artifact_cleanup_queue SET
		  claim_token = $2, claim_expires_at = $3,
		  attempt_count = attempt_count + 1, error = NULL
		WHERE artifact_key = $1
	`, key, token, expiresAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &ArtifactCleanupClaim{Key: key, Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Store) CompleteArtifactCleanup(
	ctx context.Context,
	key, token string,
) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM artifact_cleanup_queue
		WHERE artifact_key = $1 AND claim_token = $2
	`, key, token)
	if err != nil {
		return fmt.Errorf("complete artifact cleanup: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) CancelArtifactCleanup(ctx context.Context, key string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM artifact_cleanup_queue
		WHERE artifact_key = $1 AND NOT EXISTS (
		  SELECT 1 FROM issues WHERE artifact_key = $1
		)
	`, key)
	if err != nil {
		return fmt.Errorf("cancel artifact cleanup: %w", err)
	}
	return nil
}

func (s *Store) FailArtifactCleanup(
	ctx context.Context,
	key, token string,
	cause error,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE artifact_cleanup_queue SET
		  available_at = $3::timestamptz + make_interval(
		    secs => LEAST(86400, 60 * power(2, GREATEST(0, attempt_count - 1)))::int
		  ),
		  claim_token = NULL, claim_expires_at = NULL, error = $4
		WHERE artifact_key = $1 AND claim_token = $2
	`, key, token, now, safeStoreError(cause))
	if err != nil {
		return fmt.Errorf("fail artifact cleanup: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}
