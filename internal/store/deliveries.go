package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ClaimNextDelivery(
	ctx context.Context,
	now time.Time,
	claimDuration time.Duration,
	maxAttempts int,
) (*DeliveryClaim, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var issueID string
	err = tx.QueryRow(ctx, `
		SELECT d.issue_id::text
		FROM delivery_receipts d
		JOIN issues i ON i.id = d.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		JOIN accounts a ON a.id = n.owner_account_id
		WHERE d.status IN ('pending', 'failed') AND d.available_at <= $1
		  AND d.attempt_count < $2 AND n.email_enabled
		  AND a.status = 'active' AND a.primary_email IS NOT NULL
		ORDER BY d.available_at, d.created_at, d.issue_id
		FOR UPDATE OF d SKIP LOCKED
		LIMIT 1
	`, now, maxAttempts).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select Delivery Receipt Claim: %w", err)
	}
	token := uuid.New()
	expires := now.Add(claimDuration)
	if _, err := tx.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = 'delivering', attempt_count = attempt_count + 1,
			claim_token = $2, claim_expires_at = $3, started_at = $1,
			error = NULL, updated_at = $1
		WHERE issue_id = $4
	`, now, token, expires, issueID); err != nil {
		return nil, err
	}
	issue, accountID, email, err := getWorkerIssue(ctx, tx, issueID)
	if err != nil {
		return nil, err
	}
	var receipt domain.DeliveryReceipt
	var username *string
	var siteVisibility *domain.SiteVisibility
	if err := tx.QueryRow(ctx, `
		SELECT d.issue_id::text, d.status, d.attempt_count,
		       COALESCE(d.external_id, ''), COALESCE(d.error, ''),
		       d.created_at, d.started_at, d.completed_at, d.available_at,
		       s.username, s.visibility
		FROM delivery_receipts d
		LEFT JOIN personal_sites s ON s.owner_account_id = $2
		WHERE d.issue_id = $1
	`, issueID, accountID).Scan(
		&receipt.IssueID,
		&receipt.Status,
		&receipt.AttemptCount,
		&receipt.ExternalID,
		&receipt.Error,
		&receipt.CreatedAt,
		&receipt.StartedAt,
		&receipt.CompletedAt,
		&receipt.NextAttempt,
		&username,
		&siteVisibility,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	claim := &DeliveryClaim{
		Issue: issue, AccountID: accountID, PrimaryEmail: email,
		Receipt: receipt, Token: token.String(), ExpiresAt: expires,
	}
	if username != nil {
		claim.SiteUsername = *username
	}
	claim.SitePublic = siteVisibility != nil && *siteVisibility == domain.SitePublic
	return claim, nil
}

func (s *Store) RenewDeliveryClaim(
	ctx context.Context,
	issueID, token string,
	expiresAt time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE delivery_receipts SET claim_expires_at = $3, updated_at = now()
		WHERE issue_id = $1 AND claim_token = $2 AND status = 'delivering'
		  AND claim_expires_at > now()
	`, issueID, token, expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) CompleteDelivery(
	ctx context.Context,
	issueID, token, externalID string,
	now time.Time,
) error {
	if strings.TrimSpace(externalID) == "" {
		return errors.New("delivery external ID is required")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = 'delivered', external_id = $3, completed_at = $4,
			claim_token = NULL, claim_expires_at = NULL, error = NULL,
			updated_at = $4
		WHERE issue_id = $1 AND claim_token = $2 AND status = 'delivering'
		  AND claim_expires_at > $4
	`, issueID, token, externalID, now)
	if err != nil {
		return fmt.Errorf("complete Delivery Receipt: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) FailDelivery(
	ctx context.Context,
	issueID, token string,
	cause error,
	maxAttempts int,
	now time.Time,
) error {
	message := safeStoreError(cause)
	tag, err := s.pool.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = 'failed',
			available_at = $3 + make_interval(
				secs => LEAST(3600, 30 * power(2, GREATEST(0, attempt_count - 1)))::int
			),
			claim_token = NULL, claim_expires_at = NULL, error = $5,
			completed_at = CASE WHEN attempt_count >= $4 THEN $3 ELSE NULL END,
			updated_at = $3
		WHERE issue_id = $1 AND claim_token = $2 AND status = 'delivering'
	`, issueID, token, now, maxAttempts, message)
	if err != nil {
		return fmt.Errorf("fail Delivery Receipt: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) MarkDeliveryUnknown(
	ctx context.Context,
	issueID, token string,
	cause error,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = 'unknown', claim_token = NULL, claim_expires_at = NULL,
			error = $3, completed_at = $4, updated_at = $4
		WHERE issue_id = $1 AND claim_token = $2 AND status = 'delivering'
	`, issueID, token, safeStoreError(cause), now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) RetryDelivery(
	ctx context.Context,
	accountID, issueID string,
	maxAttempts int,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE delivery_receipts d SET
			status = 'pending', available_at = now(), completed_at = NULL,
			error = NULL, updated_at = now()
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE d.issue_id = i.id AND d.issue_id = $2
		  AND n.owner_account_id = $1 AND n.email_enabled
		  AND d.status = 'failed' AND d.attempt_count < $3
	`, accountID, issueID, maxAttempts)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) GetDelivery(
	ctx context.Context,
	accountID, issueID string,
) (*domain.DeliveryReceipt, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT d.issue_id::text, d.status, d.attempt_count,
		       COALESCE(d.external_id, ''), COALESCE(d.error, ''),
		       d.created_at, d.started_at, d.completed_at, d.available_at
		FROM delivery_receipts d
		JOIN issues i ON i.id = d.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1 AND d.issue_id = $2
	`, accountID, issueID)
	var receipt domain.DeliveryReceipt
	err := row.Scan(
		&receipt.IssueID,
		&receipt.Status,
		&receipt.AttemptCount,
		&receipt.ExternalID,
		&receipt.Error,
		&receipt.CreatedAt,
		&receipt.StartedAt,
		&receipt.CompletedAt,
		&receipt.NextAttempt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (s *Store) listDeliveries(
	ctx context.Context,
	accountID, newsletterID string,
) (map[string]domain.DeliveryReceipt, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.issue_id::text, d.status, d.attempt_count,
		       COALESCE(d.external_id, ''), COALESCE(d.error, ''),
		       d.created_at, d.started_at, d.completed_at, d.available_at
		FROM delivery_receipts d
		JOIN issues i ON i.id = d.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1 AND n.id = $2
	`, accountID, newsletterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]domain.DeliveryReceipt{}
	for rows.Next() {
		var receipt domain.DeliveryReceipt
		if err := rows.Scan(
			&receipt.IssueID,
			&receipt.Status,
			&receipt.AttemptCount,
			&receipt.ExternalID,
			&receipt.Error,
			&receipt.CreatedAt,
			&receipt.StartedAt,
			&receipt.CompletedAt,
			&receipt.NextAttempt,
		); err != nil {
			return nil, err
		}
		result[receipt.IssueID] = receipt
	}
	return result, rows.Err()
}
