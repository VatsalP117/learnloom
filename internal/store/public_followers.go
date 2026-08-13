package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PublicFollowClaim struct {
	ID              string
	FollowerID      string
	Email           string
	NewsletterName  string
	SiteUsername    string
	Token           string
	Kind            string
	IssueTitle      string
	IssuePublicID   string
	IssuePublicSlug string
	ClaimToken      string
	ExpiresAt       time.Time
}

func (s *Store) RequestPublicPathFollow(
	ctx context.Context,
	username, publicID, email string,
	now time.Time,
) error {
	email, err := normalizeFollowerEmail(email)
	if err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	confirmationToken, err := randomPublicToken()
	if err != nil {
		return err
	}
	unsubscribeToken, err := randomPublicToken()
	if err != nil {
		return err
	}
	rawID := strings.TrimPrefix(publicID, "dossier-")
	if _, err := uuid.Parse(rawID); err != nil {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var followerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO public_path_followers (
		  id, newsletter_id, owner_account_id, email, email_hash, status,
		  confirmation_token_hash, unsubscribe_token_hash,
		  requested_at, updated_at
		)
		SELECT $3, newsletter.id, newsletter.owner_account_id, $4, $5, 'pending',
		       $6, $7, $8, $8
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN accounts account ON account.id = newsletter.owner_account_id
		JOIN personal_sites site ON site.owner_account_id = account.id
		WHERE site.username = $1 AND site.visibility = 'public'
		  AND account.status = 'active' AND newsletter.site_visible
		  AND issue.public_id = $2 AND issue.status = 'generated'
		  AND issue.publication_state = 'published'
		  AND issue.moderation_state = 'clear'
		ON CONFLICT (newsletter_id, email_hash) DO UPDATE SET
		  email = EXCLUDED.email,
		  status = CASE WHEN public_path_followers.status = 'confirmed'
		                THEN 'confirmed' ELSE 'pending' END,
		  confirmation_token_hash = CASE WHEN public_path_followers.status = 'confirmed'
		                THEN public_path_followers.confirmation_token_hash
		                ELSE EXCLUDED.confirmation_token_hash END,
		  unsubscribe_token_hash = CASE WHEN public_path_followers.status = 'confirmed'
		                THEN public_path_followers.unsubscribe_token_hash
		                ELSE EXCLUDED.unsubscribe_token_hash END,
		  requested_at = CASE WHEN public_path_followers.status = 'confirmed'
		                THEN public_path_followers.requested_at ELSE EXCLUDED.requested_at END,
		  unsubscribed_at = CASE WHEN public_path_followers.status = 'confirmed'
		                THEN public_path_followers.unsubscribed_at ELSE NULL END,
		  updated_at = EXCLUDED.updated_at
		RETURNING id::text
	`, strings.ToLower(username), rawID, uuid.New(), email,
		sha256Hex(strings.ToLower(email)), sha256Hex(confirmationToken),
		sha256Hex(unsubscribeToken), now).Scan(&followerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("request public path follow: %w", err)
	}
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM public_path_followers WHERE id = $1`, followerID).Scan(&status); err != nil {
		return err
	}
	if status == "pending" {
		_, err = tx.Exec(ctx, `
			INSERT INTO public_follow_deliveries (
			  id, follower_id, issue_id, kind, status, token, available_at, created_at, updated_at
			)
			VALUES ($1, $2, NULL, 'confirmation', 'pending', $3, $4, $4, $4)
			ON CONFLICT (follower_id) WHERE kind = 'confirmation' DO UPDATE SET
			  status = 'pending', token = EXCLUDED.token, attempt_count = 0,
			  available_at = EXCLUDED.available_at, claim_token = NULL,
			  claim_expires_at = NULL, external_id = NULL, error = NULL,
			  completed_at = NULL, updated_at = EXCLUDED.updated_at
		`, uuid.New(), followerID, confirmationToken, now)
		if err != nil {
			return fmt.Errorf("queue public follow confirmation: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) DispatchPublicFollowUpdates(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT follower.id::text, issue.id::text
		FROM public_path_followers follower
		JOIN issues issue ON issue.newsletter_id = follower.newsletter_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN personal_sites site ON site.owner_account_id = follower.owner_account_id
		WHERE follower.status = 'confirmed'
		  AND issue.status = 'generated' AND issue.publication_state = 'published'
		  AND issue.moderation_state = 'clear' AND newsletter.site_visible
		  AND site.visibility = 'public'
		  AND issue.published_at > follower.confirmed_at
		  AND NOT EXISTS (
		    SELECT 1 FROM public_follow_deliveries delivery
		    WHERE delivery.follower_id = follower.id
		      AND delivery.issue_id = issue.id AND delivery.kind = 'update'
		  )
		ORDER BY issue.published_at, follower.id
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, fmt.Errorf("list public follow updates: %w", err)
	}
	type candidate struct{ followerID, issueID string }
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.followerID, &item.issueID); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	created := 0
	for _, item := range candidates {
		token, err := randomPublicToken()
		if err != nil {
			return created, err
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return created, err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO public_follow_deliveries (
			  id, follower_id, issue_id, kind, status, token,
			  available_at, created_at, updated_at
			)
			SELECT $1, follower.id, $3, 'update', 'pending', $4, $5, $5, $5
			FROM public_path_followers follower
			WHERE follower.id = $2 AND follower.status = 'confirmed'
			ON CONFLICT (follower_id, issue_id) WHERE kind = 'update' DO NOTHING
		`, uuid.New(), item.followerID, item.issueID, token, now)
		if err == nil && tag.RowsAffected() == 1 {
			_, err = tx.Exec(ctx, `
				INSERT INTO public_follow_unsubscribe_tokens (
				  follower_id, token_hash, created_at
				)
				VALUES ($1, $2, $3)
				ON CONFLICT (token_hash) DO NOTHING
			`, item.followerID, sha256Hex(token), now)
		}
		if err != nil {
			rollback(tx)
			return created, err
		}
		if err := tx.Commit(ctx); err != nil {
			return created, err
		}
		created += int(tag.RowsAffected())
	}
	return created, nil
}

func (s *Store) ConfirmPublicPathFollow(ctx context.Context, token string, now time.Time) error {
	if len(token) < 32 || len(token) > 256 {
		return ErrNotFound
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
		WITH confirmed AS (
		  UPDATE public_path_followers SET
		    status = 'confirmed', confirmed_at = COALESCE(confirmed_at, $2),
		    unsubscribed_at = NULL, updated_at = $2
		  WHERE confirmation_token_hash = $1
		    AND requested_at >= $2::timestamptz - interval '7 days'
		  RETURNING id, newsletter_id, owner_account_id, confirmation_token_hash
		)
		INSERT INTO public_growth_events (
		  issue_id, owner_account_id, event_name, channel,
		  visitor_fingerprint, visitor_day, occurred_at
		)
		SELECT issue.id, confirmed.owner_account_id, 'follow', '',
		       confirmed.confirmation_token_hash,
		       ($2 AT TIME ZONE 'UTC')::date, $2
		FROM confirmed
		JOIN LATERAL (
		  SELECT id FROM issues
		  WHERE newsletter_id = confirmed.newsletter_id
		    AND status = 'generated' AND publication_state = 'published'
		  ORDER BY completed_at DESC NULLS LAST, created_at DESC
		  LIMIT 1
		) issue ON true
		ON CONFLICT DO NOTHING
	`, sha256Hex(token), now)
	if err != nil {
		return fmt.Errorf("confirm public path follow: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var confirmed bool
		err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM public_path_followers
			  WHERE confirmation_token_hash = $1 AND status = 'confirmed'
			)
		`, sha256Hex(token)).Scan(&confirmed)
		if err != nil {
			return err
		}
		if !confirmed {
			return ErrNotFound
		}
	}
	return nil
}

func (s *Store) ClaimNextPublicFollowDelivery(
	ctx context.Context,
	now time.Time,
	claimDuration time.Duration,
	maxAttempts int,
) (*PublicFollowClaim, error) {
	if claimDuration < time.Minute || maxAttempts < 1 {
		return nil, errors.New("public follow claim settings are invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var deliveryID string
	err = tx.QueryRow(ctx, `
		SELECT delivery.id::text
		FROM public_follow_deliveries delivery
		JOIN public_path_followers follower ON follower.id = delivery.follower_id
		JOIN accounts owner ON owner.id = follower.owner_account_id
		WHERE delivery.status IN ('pending', 'failed')
		  AND delivery.available_at <= $1 AND delivery.attempt_count < $2
		  AND (
		    (delivery.kind = 'confirmation' AND follower.status = 'pending')
		    OR (delivery.kind = 'update' AND follower.status = 'confirmed')
		  )
		  AND owner.status = 'active'
		ORDER BY delivery.available_at, delivery.created_at, delivery.id
		FOR UPDATE OF delivery SKIP LOCKED
		LIMIT 1
	`, now, maxAttempts).Scan(&deliveryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim public follow delivery candidate: %w", err)
	}
	claimToken := uuid.NewString()
	expiresAt := now.Add(claimDuration)
	var claim PublicFollowClaim
	var issueID *string
	err = tx.QueryRow(ctx, `
		UPDATE public_follow_deliveries delivery SET
		  status = 'delivering', attempt_count = attempt_count + 1,
		  claim_token = $2, claim_expires_at = $3,
		  error = NULL, updated_at = $1
		WHERE delivery.id = $4
		RETURNING delivery.id::text, delivery.follower_id::text,
		          delivery.issue_id::text, delivery.token, delivery.kind
	`, now, claimToken, expiresAt, deliveryID).Scan(
		&claim.ID, &claim.FollowerID, &issueID, &claim.Token, &claim.Kind,
	)
	if err != nil {
		return nil, fmt.Errorf("claim public follow delivery: %w", err)
	}
	err = tx.QueryRow(ctx, `
		SELECT follower.email, newsletter.name, site.username,
		       COALESCE(issue.dossier_title, ''),
		       COALESCE('dossier-' || issue.public_id::text, ''),
		       COALESCE(issue.public_slug, '')
		FROM public_path_followers follower
		JOIN newsletters newsletter ON newsletter.id = follower.newsletter_id
		JOIN personal_sites site ON site.owner_account_id = follower.owner_account_id
		LEFT JOIN issues issue ON issue.id = $2
		WHERE follower.id = $1
	`, claim.FollowerID, issueID).Scan(
		&claim.Email, &claim.NewsletterName, &claim.SiteUsername,
		&claim.IssueTitle, &claim.IssuePublicID, &claim.IssuePublicSlug,
	)
	if err != nil {
		return nil, fmt.Errorf("load public follow delivery: %w", err)
	}
	claim.ClaimToken = claimToken
	claim.ExpiresAt = expiresAt
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &claim, nil
}

func (s *Store) RenewPublicFollowDeliveryClaim(
	ctx context.Context,
	id, token string,
	expiresAt time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE public_follow_deliveries SET claim_expires_at = $3, updated_at = now()
		WHERE id = $1 AND claim_token = $2 AND status = 'delivering'
		  AND claim_expires_at > now()
	`, id, token, expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) CompletePublicFollowDelivery(
	ctx context.Context,
	id, token, externalID string,
	now time.Time,
) error {
	if strings.TrimSpace(externalID) == "" {
		return errors.New("public follow external ID is required")
	}
	return s.transitionPublicFollowDelivery(ctx, id, token, "delivered", externalID, "", now)
}

func (s *Store) FailPublicFollowDelivery(
	ctx context.Context,
	id, token string,
	cause error,
	maxAttempts int,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE public_follow_deliveries SET
		  status = 'failed',
		  available_at = $3::timestamptz + make_interval(
		    secs => LEAST(3600, 30 * power(2, GREATEST(0, attempt_count - 1)))::int
		  ),
		  claim_token = NULL, claim_expires_at = NULL,
		  error = $5,
		  completed_at = CASE WHEN attempt_count >= $4 THEN $3::timestamptz ELSE NULL END,
		  updated_at = $3
		WHERE id = $1 AND claim_token = $2 AND status = 'delivering'
	`, id, token, now, maxAttempts, safeStoreError(cause))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) MarkPublicFollowDeliveryUnknown(
	ctx context.Context,
	id, token string,
	cause error,
	now time.Time,
) error {
	return s.transitionPublicFollowDelivery(ctx, id, token, "unknown", "", safeStoreError(cause), now)
}

func (s *Store) transitionPublicFollowDelivery(
	ctx context.Context,
	id, token, status, externalID, errorText string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE public_follow_deliveries SET
		  status = $3, external_id = NULLIF($4, ''), error = NULLIF($5, ''),
		  claim_token = NULL, claim_expires_at = NULL,
		  completed_at = $6, updated_at = $6
		WHERE id = $1 AND claim_token = $2 AND status = 'delivering'
	`, id, token, status, externalID, errorText, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) UnsubscribePublicPathFollow(ctx context.Context, token string, now time.Time) error {
	if len(token) < 32 || len(token) > 256 {
		return ErrNotFound
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE public_path_followers follower SET
		  status = 'unsubscribed', unsubscribed_at = $2, updated_at = $2
		WHERE follower.unsubscribe_token_hash = $1
		   OR EXISTS (
		     SELECT 1 FROM public_follow_unsubscribe_tokens token
		     WHERE token.follower_id = follower.id AND token.token_hash = $1
		   )
	`, sha256Hex(token), now)
	if err != nil {
		return fmt.Errorf("unsubscribe public path follow: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func normalizeFollowerEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || strings.ToLower(address.Address) != value || len(value) > 320 {
		return "", errors.New("email address is invalid")
	}
	return value, nil
}

func randomPublicToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
