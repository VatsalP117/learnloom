package store

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) SetSourcePreference(
	ctx context.Context,
	accountID, newsletterID, sourceID string,
	preference domain.SourcePreference,
) error {
	switch preference {
	case domain.SourcePreferenceNeutral,
		domain.SourcePreferencePreferred,
		domain.SourcePreferenceBlocked:
	default:
		return errors.New("source preference is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var origin domain.SourceOrigin
	var mode domain.SourceMode
	if err := tx.QueryRow(ctx, `
		SELECT ss.origin, n.source_mode
		FROM source_specs ss
		JOIN newsletters n ON n.id = ss.newsletter_id
		WHERE ss.id = $1 AND ss.newsletter_id = $2 AND n.owner_account_id = $3
		FOR UPDATE OF ss, n
	`, sourceID, newsletterID, accountID).Scan(&origin, &mode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("inspect source preference: %w", err)
	}
	if preference == domain.SourcePreferenceBlocked &&
		origin == domain.SourceOriginProvided && mode == domain.SourceModeProvided {
		var alternatives int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM source_specs
			WHERE newsletter_id = $1 AND origin = 'provided' AND id <> $2
			  AND preference <> 'blocked' AND state = 'active'
		`, newsletterID, sourceID).Scan(&alternatives); err != nil {
			return err
		}
		if alternatives == 0 {
			return ErrConflict
		}
	}
	state := domain.SourceStateActive
	if preference == domain.SourcePreferenceBlocked ||
		(origin == domain.SourceOriginDiscovered && mode == domain.SourceModeProvided) ||
		(origin == domain.SourceOriginProvided && mode == domain.SourceModeDiscovered) {
		state = domain.SourceStateDisabled
	}
	if _, err := tx.Exec(ctx, `
		UPDATE source_specs
		SET preference = $2, state = $3, updated_at = now()
		WHERE id = $1
	`, sourceID, preference, state); err != nil {
		return fmt.Errorf("set source preference: %w", err)
	}
	if preference == domain.SourcePreferenceBlocked &&
		origin == domain.SourceOriginProvided && mode == domain.SourceModeHybrid {
		var remaining int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM source_specs
			WHERE newsletter_id = $1 AND origin = 'provided'
			  AND preference <> 'blocked' AND state = 'active'
		`, newsletterID).Scan(&remaining); err != nil {
			return err
		}
		if remaining == 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE newsletters SET source_mode = 'discovered', updated_at = now()
				WHERE id = $1
			`, newsletterID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE source_specs SET state = 'active', updated_at = now()
				WHERE newsletter_id = $1 AND origin = 'discovered'
				  AND preference <> 'blocked'
			`, newsletterID); err != nil {
				return err
			}
		}
	}
	if err := resetAwaitingSourcePortfolio(
		ctx,
		tx,
		newsletterID,
		time.Now().UTC(),
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetNewsletterSourceMode(
	ctx context.Context,
	accountID, newsletterID string,
	mode domain.SourceMode,
) error {
	switch mode {
	case domain.SourceModeDiscovered, domain.SourceModeProvided, domain.SourceModeHybrid:
	default:
		return errors.New("source mode is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT true FROM newsletters
		WHERE id = $1 AND owner_account_id = $2
		FOR UPDATE
	`, newsletterID, accountID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if mode == domain.SourceModeProvided || mode == domain.SourceModeHybrid {
		var provided int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM source_specs
			WHERE newsletter_id = $1 AND origin = 'provided'
			  AND preference <> 'blocked'
		`, newsletterID).Scan(&provided); err != nil {
			return err
		}
		if provided == 0 {
			return ErrConflict
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE newsletters SET source_mode = $2, updated_at = now()
		WHERE id = $1
	`, newsletterID, mode); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE source_specs SET
		  state = CASE
		    WHEN preference = 'blocked' THEN 'disabled'
		    WHEN origin = 'provided' AND $2 = 'discovered' THEN 'disabled'
		    WHEN origin = 'discovered' AND $2 = 'provided' THEN 'disabled'
		    ELSE 'active'
		  END,
		  updated_at = now()
		WHERE newsletter_id = $1 AND state IN ('active', 'disabled')
	`, newsletterID, mode); err != nil {
		return err
	}
	if err := resetAwaitingSourcePortfolio(
		ctx,
		tx,
		newsletterID,
		time.Now().UTC(),
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ReplaceProvidedSource(
	ctx context.Context,
	accountID, newsletterID, sourceID string,
	replacement domain.SourceDefinition,
) (string, error) {
	var err error
	replacement.Name, err = boundedText(replacement.Name, "source name", 120)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(strings.TrimSpace(replacement.URL))
	if err != nil || parsed.Host == "" || parsed.User != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("source URL is invalid")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", errors.New("source URL must use a public host")
	}
	if address, parseErr := netip.ParseAddr(host); parseErr == nil &&
		(!address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
			address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified()) {
		return "", errors.New("source URL must use a public host")
	}
	replacement.URL = parsed.String()
	if replacement.Limit < 1 || replacement.Limit > 50 {
		return "", errors.New("source limit must be from 1 to 50")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer rollback(tx)
	var origin domain.SourceOrigin
	var mode domain.SourceMode
	if err := tx.QueryRow(ctx, `
		SELECT ss.origin, n.source_mode
		FROM source_specs ss
		JOIN newsletters n ON n.id = ss.newsletter_id
		WHERE ss.id = $1 AND ss.newsletter_id = $2 AND n.owner_account_id = $3
		FOR UPDATE OF ss, n
	`, sourceID, newsletterID, accountID).Scan(&origin, &mode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if origin != domain.SourceOriginProvided {
		return "", ErrConflict
	}
	if _, err := tx.Exec(ctx, `
		UPDATE source_specs
		SET state = 'disabled', preference = 'blocked', updated_at = now()
		WHERE id = $1
	`, sourceID); err != nil {
		return "", err
	}
	scope, kind := sourceShape(replacement.URL)
	state := domain.SourceStateActive
	if mode == domain.SourceModeDiscovered {
		state = domain.SourceStateDisabled
	}
	var replacementID string
	err = tx.QueryRow(ctx, `
		SELECT id::text FROM source_specs
		WHERE newsletter_id = $1 AND canonical_url = $2
		FOR UPDATE
	`, newsletterID, replacement.URL).Scan(&replacementID)
	now := time.Now().UTC()
	if err == nil {
		if _, err := tx.Exec(ctx, `
			UPDATE source_specs SET
			  origin = 'provided', state = $2, preference = 'neutral',
			  display_name = $3, input_url = $4, scope = $5,
			  kind = $6, item_limit = $7, updated_at = $8
			WHERE id = $1
		`, replacementID, state, replacement.Name, replacement.URL, scope,
			kind, replacement.Limit, now); err != nil {
			return "", err
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		replacementID = uuid.NewString()
		if _, err := tx.Exec(ctx, `
			INSERT INTO source_specs (
			  id, newsletter_id, origin, state, display_name, input_url,
			  canonical_url, scope, kind, item_limit, preference,
			  created_at, updated_at
			)
			VALUES ($1, $2, 'provided', $3, $4, $5, $5, $6, $7, $8,
			        'neutral', $9, $9)
		`, replacementID, newsletterID, state, replacement.Name,
			replacement.URL, scope, kind, replacement.Limit, now); err != nil {
			return "", err
		}
	} else {
		return "", err
	}
	if err := resetAwaitingSourcePortfolio(ctx, tx, newsletterID, now); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return replacementID, nil
}

func sourceShape(rawURL string) (domain.SourceScope, any) {
	scope := domain.SourceScopeExact
	var kind any
	lower := strings.ToLower(rawURL)
	switch {
	case strings.Contains(lower, "/feed"), strings.Contains(lower, "/rss"):
		scope, kind = domain.SourceScopeFeed, "rss"
	case strings.Contains(lower, "/atom"):
		scope, kind = domain.SourceScopeFeed, "atom"
	}
	return scope, kind
}

func (s *Store) AwaitSourceApproval(
	ctx context.Context,
	issueID, token string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		WITH waiting_issue AS (
		  UPDATE issues i SET
		    status = 'awaiting_approval',
		    attempt_count = GREATEST(0, attempt_count - 1),
		    claim_token = NULL,
		    claim_expires_at = NULL,
		    completed_at = NULL
		  FROM newsletters n
		  WHERE i.id = $1::uuid AND i.claim_token = $2::uuid
		    AND i.status = 'generating' AND i.newsletter_id = n.id
		    AND n.source_review_mode = 'review'
		    AND n.source_approved_at IS NULL
		  RETURNING i.id
		)
		UPDATE issue_attempts SET
		  status = 'awaiting_approval', completed_at = $3::timestamptz
		WHERE id = $2::uuid AND issue_id = (SELECT id FROM waiting_issue)
		  AND status = 'running'
	`, issueID, token, now)
	if err != nil {
		return fmt.Errorf("await source approval: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) SetNewsletterSourceReviewMode(
	ctx context.Context,
	accountID, newsletterID string,
	mode domain.SourceReviewMode,
	now time.Time,
) error {
	if mode != domain.SourceReviewAuto && mode != domain.SourceReviewBeforeLesson {
		return errors.New("source review mode is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	approvedAt := any(nil)
	if mode == domain.SourceReviewAuto {
		approvedAt = now
	}
	tag, err := tx.Exec(ctx, `
		UPDATE newsletters SET
		  source_review_mode = $3,
		  source_approved_at = $4,
		  updated_at = $5
		WHERE id = $1 AND owner_account_id = $2
	`, newsletterID, accountID, mode, approvedAt, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	if mode == domain.SourceReviewAuto {
		if err := queueAwaitingSourceIssues(ctx, tx, newsletterID, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ApproveSourcePortfolio(
	ctx context.Context,
	accountID, newsletterID string,
	now time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	tag, err := tx.Exec(ctx, `
		UPDATE newsletters SET source_approved_at = $3, updated_at = $3
		WHERE id = $1 AND owner_account_id = $2
		  AND source_review_mode = 'review'
	`, newsletterID, accountID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	if err := queueAwaitingSourceIssues(ctx, tx, newsletterID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func queueAwaitingSourceIssues(
	ctx context.Context,
	tx pgx.Tx,
	newsletterID string,
	now time.Time,
) error {
	if _, err := tx.Exec(ctx, `
		UPDATE issues SET
		  status = 'queued', available_at = $2,
		  completed_at = NULL, error = NULL, public_error = NULL
		WHERE newsletter_id = $1 AND status = 'awaiting_approval'
	`, newsletterID, now); err != nil {
		return fmt.Errorf("queue approved source portfolio: %w", err)
	}
	return nil
}

func resetAwaitingSourcePortfolio(
	ctx context.Context,
	tx pgx.Tx,
	newsletterID string,
	now time.Time,
) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM issue_sources links
		USING issues i
		WHERE links.issue_id = i.id AND i.newsletter_id = $1
		  AND i.status = 'awaiting_approval'
	`, newsletterID); err != nil {
		return fmt.Errorf("clear awaiting source portfolio: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE issues SET status = 'queued', available_at = $2
		WHERE newsletter_id = $1 AND status = 'awaiting_approval'
	`, newsletterID, now); err != nil {
		return fmt.Errorf("rebuild awaiting source portfolio: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE newsletters SET
		  source_approved_at = CASE
		    WHEN source_review_mode = 'review' THEN NULL
		    ELSE source_approved_at
		  END,
		  updated_at = $2
		WHERE id = $1
	`, newsletterID, now); err != nil {
		return err
	}
	return nil
}
