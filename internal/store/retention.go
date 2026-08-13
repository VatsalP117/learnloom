package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

type RetentionState struct {
	ActivatedAt            *time.Time `json:"activatedAt,omitempty"`
	ReturnedAfterSevenDays bool       `json:"returnedAfterSevenDays"`
	LastActivityAt         *time.Time `json:"lastActivityAt,omitempty"`
	Inactive               bool       `json:"inactive"`
	DaysAway               int        `json:"daysAway"`
	SuppressedBacklogCount int        `json:"-"`
	ActionLabel            string     `json:"actionLabel,omitempty"`
	ActionURL              string     `json:"actionUrl,omitempty"`
	ReentryNewsletterID    string     `json:"reentryNewsletterId,omitempty"`
	ReentryNewsletterName  string     `json:"reentryNewsletterName,omitempty"`
}

func (s *Store) GetRetentionState(
	ctx context.Context,
	accountID string,
	now time.Time,
) (RetentionState, error) {
	var state RetentionState
	err := s.pool.QueryRow(ctx, `
		SELECT
		  min(occurred_at) FILTER (WHERE event_name = 'activation_completed'),
		  max(occurred_at) FILTER (
		    WHERE event_name IN (
		      'lesson_opened', 'lesson_completed', 'review_attempted',
		      'activation_completed'
		    )
		  )
		FROM product_events
		WHERE account_id = $1
	`, accountID).Scan(&state.ActivatedAt, &state.LastActivityAt)
	if err != nil {
		return RetentionState{}, fmt.Errorf("load retention milestones: %w", err)
	}
	if state.ActivatedAt != nil {
		err = s.pool.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1
			  FROM product_events
			  WHERE account_id = $1
			    AND occurred_at >= $2::timestamptz + interval '7 days'
			)
		`, accountID, *state.ActivatedAt).Scan(&state.ReturnedAfterSevenDays)
		if err != nil {
			return RetentionState{}, err
		}
	}
	if state.LastActivityAt == nil || now.Before(*state.LastActivityAt) {
		return state, nil
	}
	var reentryEnabled bool
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE((
		  SELECT reentry_reminder
		  FROM account_notification_preferences
		  WHERE account_id = $1
		), true)
	`, accountID).Scan(&reentryEnabled)
	if err != nil {
		return RetentionState{}, err
	}
	state.DaysAway = int(math.Floor(now.Sub(*state.LastActivityAt).Hours() / 24))
	state.Inactive = reentryEnabled && state.DaysAway >= 7
	if !state.Inactive {
		return state, nil
	}
	err = s.pool.QueryRow(ctx, `
		SELECT
		  (
		    SELECT count(*)
		    FROM issues issue
		    JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		    LEFT JOIN lesson_progress progress
		      ON progress.account_id = newsletter.owner_account_id
		      AND progress.issue_id = issue.id
		    WHERE newsletter.owner_account_id = $1
		      AND newsletter.active
		      AND issue.status = 'generated'
		      AND progress.completed_at IS NULL
		      AND NOT EXISTS (
		        SELECT 1 FROM lesson_backlog_dismissals dismissal
		        WHERE dismissal.account_id = $1 AND dismissal.issue_id = issue.id
		      )
		  ) + (
		    SELECT count(*)
		    FROM review_items review
		    WHERE review.account_id = $1
		      AND review.due_at IS NOT NULL
		      AND review.retired_at IS NULL
		      AND review.due_at <= $2
		  )
	`, accountID, now).Scan(&state.SuppressedBacklogCount)
	if err != nil {
		return RetentionState{}, err
	}
	var reviewID string
	err = s.pool.QueryRow(ctx, `
		SELECT review.id::text, newsletter.id::text, newsletter.name
		FROM review_items review
		JOIN issues issue ON issue.id = review.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE review.account_id = $1
		  AND review.due_at IS NOT NULL
		  AND review.retired_at IS NULL
		  AND review.due_at <= $2
		ORDER BY review.due_at, review.created_at
		LIMIT 1
	`, accountID, now).Scan(
		&reviewID, &state.ReentryNewsletterID, &state.ReentryNewsletterName,
	)
	if err == nil {
		state.ActionLabel = "Recall one idea"
		state.ActionURL = "/review"
		return state, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return RetentionState{}, err
	}
	var issueID string
	err = s.pool.QueryRow(ctx, `
		SELECT issue.id::text, newsletter.id::text, newsletter.name
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		LEFT JOIN lesson_progress progress
		  ON progress.account_id = newsletter.owner_account_id
		  AND progress.issue_id = issue.id
		WHERE newsletter.owner_account_id = $1
		  AND newsletter.active
		  AND issue.status = 'generated'
		  AND progress.completed_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM lesson_backlog_dismissals dismissal
			WHERE dismissal.account_id = $1 AND dismissal.issue_id = issue.id
		  )
		ORDER BY issue.created_at DESC
		LIMIT 1
	`, accountID).Scan(
		&issueID, &state.ReentryNewsletterID, &state.ReentryNewsletterName,
	)
	if err == nil {
		state.ActionLabel = "Open one worthwhile lesson"
		state.ActionURL = "/issues/" + issueID
		return state, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return RetentionState{}, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT id::text, name
		FROM newsletters
		WHERE owner_account_id = $1 AND active
		ORDER BY updated_at DESC, id
		LIMIT 1
	`, accountID).Scan(&state.ReentryNewsletterID, &state.ReentryNewsletterName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return RetentionState{}, err
	}
	state.ActionLabel = "Choose a gentler rhythm"
	state.ActionURL = "/streams"
	return state, nil
}
