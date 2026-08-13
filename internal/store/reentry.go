package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/jackc/pgx/v5"
)

type BacklogResetResult struct {
	DismissedCount int              `json:"dismissedCount"`
	Newsletter     NewsletterRecord `json:"newsletter"`
}

func (s *Store) ResetNewsletterBacklog(
	ctx context.Context,
	accountID, newsletterID string,
	now time.Time,
) (BacklogResetResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BacklogResetResult{}, err
	}
	defer rollback(tx)
	var zone string
	var hour, minute int
	var desiredMode domain.RhythmMode
	var weekdays []int16
	if err := tx.QueryRow(ctx, `
		SELECT time_zone, schedule_hour, schedule_minute,
		       rhythm_mode, selected_weekdays
		FROM newsletters
		WHERE owner_account_id = $1 AND id = $2
		FOR UPDATE
	`, accountID, newsletterID).Scan(
		&zone, &hour, &minute, &desiredMode, &weekdays,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BacklogResetResult{}, ErrNotFound
		}
		return BacklogResetResult{}, err
	}
	tag, err := tx.Exec(ctx, `
		WITH candidates AS (
			SELECT issue.id,
			       row_number() OVER (ORDER BY issue.created_at DESC, issue.id DESC) AS position
			FROM issues issue
			LEFT JOIN lesson_progress progress
			  ON progress.account_id = $1 AND progress.issue_id = issue.id
			WHERE issue.newsletter_id = $2
			  AND issue.status = 'generated'
			  AND progress.completed_at IS NULL
			  AND NOT EXISTS (
				SELECT 1 FROM product_events event
				WHERE event.account_id = $1
				  AND event.event_name = 'lesson_opened'
				  AND event.subject_type = 'lesson'
				  AND event.subject_id = issue.id::text
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM lesson_backlog_dismissals dismissal
				WHERE dismissal.account_id = $1 AND dismissal.issue_id = issue.id
			  )
		)
		INSERT INTO lesson_backlog_dismissals (
			account_id, issue_id, newsletter_id, reason, dismissed_at
		)
		SELECT $1, id, $2, 'reentry_reset', $3
		FROM candidates
		WHERE position > 1
		ON CONFLICT (account_id, issue_id) DO NOTHING
	`, accountID, newsletterID, now)
	if err != nil {
		return BacklogResetResult{}, fmt.Errorf("reset Newsletter backlog: %w", err)
	}
	dismissed := int(tag.RowsAffected())
	unopened, err := unopenedLessonCount(ctx, tx, newsletterID)
	if err != nil {
		return BacklogResetResult{}, err
	}
	next, err := NextRhythmOccurrence(
		now, zone, hour, minute, desiredMode, weekdayInts(weekdays),
	)
	if err != nil {
		return BacklogResetResult{}, err
	}
	reason := "Older unopened lessons were moved out of Today and remain available in your library."
	if _, err := tx.Exec(ctx, `
		UPDATE newsletters SET
			effective_rhythm_mode = rhythm_mode,
			rhythm_throttled_at = NULL,
			rhythm_reason = $3,
			next_run_at = $4,
			updated_at = $5
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, reason, next, now); err != nil {
		return BacklogResetResult{}, fmt.Errorf("restore Newsletter rhythm after reset: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO rhythm_decisions (
			newsletter_id, decision, desired_mode, effective_mode,
			unopened_count, reason, decided_at
		)
		VALUES ($1, 'recover', $2, $2, $3, $4, $5)
	`, newsletterID, desiredMode, unopened, reason, now); err != nil {
		return BacklogResetResult{}, fmt.Errorf("record backlog reset decision: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM today_focus_selections WHERE account_id = $1
	`, accountID); err != nil {
		return BacklogResetResult{}, fmt.Errorf("clear Today focus after reset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return BacklogResetResult{}, err
	}
	record, err := s.GetNewsletter(ctx, accountID, newsletterID)
	if err != nil {
		return BacklogResetResult{}, err
	}
	return BacklogResetResult{DismissedCount: dismissed, Newsletter: record}, nil
}
