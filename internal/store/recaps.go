package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NotificationPreferences struct {
	Configured      bool      `json:"configured"`
	WeeklyRecap     bool      `json:"weeklyRecap"`
	ReentryReminder bool      `json:"reentryReminder"`
	TimeZone        string    `json:"timeZone"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type WeeklyRecapPayload struct {
	WeekStart        string   `json:"weekStart"`
	LessonsCompleted int      `json:"lessonsCompleted"`
	Concepts         []string `json:"concepts"`
	Capabilities     []string `json:"capabilities"`
	Connection       string   `json:"connection"`
	ReviewPrompt     string   `json:"reviewPrompt,omitempty"`
	ActionLabel      string   `json:"actionLabel"`
	ActionURL        string   `json:"actionUrl"`
}

type WeeklyRecapClaim struct {
	ID           string
	AccountID    string
	PrimaryEmail string
	WeekStart    string
	Payload      WeeklyRecapPayload
	Token        string
	ExpiresAt    time.Time
}

func (s *Store) GetNotificationPreferences(
	ctx context.Context,
	accountID string,
) (NotificationPreferences, error) {
	var preferences NotificationPreferences
	err := s.pool.QueryRow(ctx, `
		SELECT weekly_recap, reentry_reminder, time_zone, updated_at
		FROM account_notification_preferences
		WHERE account_id = $1
	`, accountID).Scan(
		&preferences.WeeklyRecap,
		&preferences.ReentryReminder,
		&preferences.TimeZone,
		&preferences.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return NotificationPreferences{ReentryReminder: true, TimeZone: "UTC"}, nil
	}
	if err != nil {
		return NotificationPreferences{}, fmt.Errorf("get Notification Preferences: %w", err)
	}
	preferences.Configured = true
	return preferences, nil
}

func (s *Store) UpdateNotificationPreferences(
	ctx context.Context,
	accountID string,
	weeklyRecap, reentryReminder bool,
	timeZone string,
	now time.Time,
) (NotificationPreferences, error) {
	if len(timeZone) < 1 || len(timeZone) > 80 {
		return NotificationPreferences{}, errors.New("notification timezone is invalid")
	}
	if _, err := time.LoadLocation(timeZone); err != nil {
		return NotificationPreferences{}, errors.New("notification timezone is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var preferences NotificationPreferences
	err := s.pool.QueryRow(ctx, `
		INSERT INTO account_notification_preferences (
			account_id, weekly_recap, reentry_reminder, time_zone, updated_at
		)
		SELECT id, $2, $3, $4, $5
		FROM accounts
		WHERE id = $1 AND status = 'active'
		ON CONFLICT (account_id) DO UPDATE SET
			weekly_recap = EXCLUDED.weekly_recap,
			reentry_reminder = EXCLUDED.reentry_reminder,
			time_zone = EXCLUDED.time_zone,
			updated_at = EXCLUDED.updated_at
		RETURNING weekly_recap, reentry_reminder, time_zone, updated_at
	`, accountID, weeklyRecap, reentryReminder, timeZone, now).Scan(
		&preferences.WeeklyRecap,
		&preferences.ReentryReminder,
		&preferences.TimeZone,
		&preferences.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return NotificationPreferences{}, ErrForbidden
	}
	if err != nil {
		return NotificationPreferences{}, fmt.Errorf("update Notification Preferences: %w", err)
	}
	preferences.Configured = true
	return preferences, nil
}

func (s *Store) DispatchWeeklyRecaps(
	ctx context.Context,
	now time.Time,
	limit int,
) (int, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT preferences.account_id::text, preferences.time_zone
		FROM account_notification_preferences preferences
		JOIN accounts account ON account.id = preferences.account_id
		WHERE preferences.weekly_recap
		  AND account.status = 'active'
		  AND account.primary_email IS NOT NULL
		  AND (timezone(preferences.time_zone, $1))::time >= time '08:00'
		  AND NOT EXISTS (
		    SELECT 1
		    FROM weekly_recaps recap
		    WHERE recap.account_id = preferences.account_id
		      AND recap.week_start = date_trunc(
		        'week',
		        timezone(preferences.time_zone, $1)
		      )::date
		  )
		  AND EXISTS (
		    SELECT 1
		    FROM lesson_progress progress
		    WHERE progress.account_id = preferences.account_id
		      AND progress.completed_at >= (
		        date_trunc('week', timezone(preferences.time_zone, $1))
		        - interval '7 days'
		      ) AT TIME ZONE preferences.time_zone
		      AND progress.completed_at < date_trunc(
		        'week',
		        timezone(preferences.time_zone, $1)
		      ) AT TIME ZONE preferences.time_zone
		  )
		ORDER BY preferences.account_id
		LIMIT $2
	`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("list weekly Recap candidates: %w", err)
	}
	type candidate struct {
		accountID string
		timeZone  string
	}
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.accountID, &item.timeZone); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	dispatched := 0
	for _, candidate := range candidates {
		location, err := time.LoadLocation(candidate.timeZone)
		if err != nil {
			continue
		}
		local := now.In(location)
		weekStartLocal := time.Date(
			local.Year(),
			local.Month(),
			local.Day()-int((int(local.Weekday())+6)%7),
			0, 0, 0, 0,
			location,
		)
		weekStart := weekStartLocal.Format(time.DateOnly)
		payload, ok, err := s.buildWeeklyRecap(
			ctx,
			candidate.accountID,
			weekStartLocal.AddDate(0, 0, -7).UTC(),
			weekStartLocal.UTC(),
			now,
		)
		if err != nil {
			return dispatched, err
		}
		if !ok {
			continue
		}
		payload.WeekStart = weekStart
		encoded, err := json.Marshal(payload)
		if err != nil {
			return dispatched, err
		}
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO weekly_recaps (
				id, account_id, week_start, payload, status, available_at,
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4::jsonb, 'pending', $5, $5, $5)
			ON CONFLICT (account_id, week_start) DO NOTHING
		`, uuid.New(), candidate.accountID, weekStart, encoded, now)
		if err != nil {
			return dispatched, fmt.Errorf("enqueue weekly Recap: %w", err)
		}
		dispatched += int(tag.RowsAffected())
	}
	return dispatched, nil
}

func (s *Store) buildWeeklyRecap(
	ctx context.Context,
	accountID string,
	periodStart, periodEnd, now time.Time,
) (WeeklyRecapPayload, bool, error) {
	var payload WeeklyRecapPayload
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int
		FROM lesson_progress
		WHERE account_id = $1
		  AND completed_at >= $2
		  AND completed_at < $3
	`, accountID, periodStart, periodEnd).Scan(&payload.LessonsCompleted)
	if err != nil {
		return WeeklyRecapPayload{}, false, err
	}
	if payload.LessonsCompleted == 0 {
		return WeeklyRecapPayload{}, false, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT state.label, state.review_attempt_count, state.confidence_score
		FROM learner_concept_state state
		WHERE state.account_id = $1
		  AND state.last_completed_at >= $2
		  AND state.last_completed_at < $3
		ORDER BY state.last_completed_at DESC, state.label
		LIMIT 5
	`, accountID, periodStart, periodEnd)
	if err != nil {
		return WeeklyRecapPayload{}, false, err
	}
	for rows.Next() {
		var concept string
		var reviewAttempts, confidence int
		if err := rows.Scan(&concept, &reviewAttempts, &confidence); err != nil {
			rows.Close()
			return WeeklyRecapPayload{}, false, err
		}
		payload.Concepts = append(payload.Concepts, concept)
		capability := "Can explain the core idea behind " + concept
		if reviewAttempts > 0 && confidence >= 75 {
			capability = "Can recall and explain " + concept
		} else if reviewAttempts > 0 {
			capability = "Can retrieve " + concept + " with some support"
		}
		payload.Capabilities = append(payload.Capabilities, capability)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return WeeklyRecapPayload{}, false, err
	}

	var historyRaw []byte
	err = s.pool.QueryRow(ctx, `
		SELECT history.entry
		FROM learning_history history
		JOIN lesson_progress progress ON progress.issue_id = history.issue_id
		WHERE progress.account_id = $1
		  AND progress.completed_at >= $2
		  AND progress.completed_at < $3
		ORDER BY progress.completed_at DESC
		LIMIT 1
	`, accountID, periodStart, periodEnd).Scan(&historyRaw)
	if err == nil {
		var history domain.LearningHistoryEntry
		if json.Unmarshal(historyRaw, &history) == nil {
			switch {
			case len(payload.Concepts) >= 2:
				payload.Connection = fmt.Sprintf(
					"Connect %s with %s: where does one explain or constrain the other?",
					payload.Concepts[0], payload.Concepts[1],
				)
			case len(payload.Concepts) == 1 && len(history.SuggestedNextConcepts) > 0:
				payload.Connection = fmt.Sprintf(
					"Connect %s with the next direction, %s.",
					payload.Concepts[0], history.SuggestedNextConcepts[0],
				)
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return WeeklyRecapPayload{}, false, err
	}
	if payload.Connection == "" {
		payload.Connection = "Connect one recent idea to a question you care about."
	}

	err = s.pool.QueryRow(ctx, `
		SELECT review.prompt
		FROM review_items review
		WHERE review.account_id = $1
		  AND review.due_at IS NOT NULL
		  AND review.retired_at IS NULL
		  AND review.due_at <= $2
		ORDER BY review.due_at, review.created_at
		LIMIT 1
	`, accountID, now).Scan(&payload.ReviewPrompt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WeeklyRecapPayload{}, false, err
	}
	if payload.ReviewPrompt != "" {
		payload.ActionLabel = "Review what is due"
		payload.ActionURL = "/review"
		return payload, true, nil
	}

	var issueID string
	err = s.pool.QueryRow(ctx, `
		SELECT issue.id::text
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
	`, accountID).Scan(&issueID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WeeklyRecapPayload{}, false, err
	}
	if issueID != "" {
		payload.ActionLabel = "Continue your next lesson"
		payload.ActionURL = "/issues/" + issueID
	} else {
		payload.ActionLabel = "Tune your learning rhythm"
		payload.ActionURL = "/streams"
	}
	return payload, true, nil
}

func (s *Store) ClaimNextWeeklyRecap(
	ctx context.Context,
	now time.Time,
	claimDuration time.Duration,
	maxAttempts int,
) (*WeeklyRecapClaim, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var recapID string
	err = tx.QueryRow(ctx, `
		SELECT recap.id::text
		FROM weekly_recaps recap
		JOIN accounts account ON account.id = recap.account_id
		JOIN account_notification_preferences preferences
		  ON preferences.account_id = recap.account_id
		WHERE recap.status IN ('pending', 'failed')
		  AND recap.available_at <= $1
		  AND recap.attempt_count < $2
		  AND account.status = 'active'
		  AND account.primary_email IS NOT NULL
		  AND preferences.weekly_recap
		ORDER BY recap.available_at, recap.created_at, recap.id
		FOR UPDATE OF recap SKIP LOCKED
		LIMIT 1
	`, now, maxAttempts).Scan(&recapID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	token := uuid.New()
	expiresAt := now.Add(claimDuration)
	var claim WeeklyRecapClaim
	var raw []byte
	err = tx.QueryRow(ctx, `
		UPDATE weekly_recaps recap SET
			status = 'delivering',
			attempt_count = attempt_count + 1,
			claim_token = $2,
			claim_expires_at = $3,
			started_at = $1,
			error = NULL,
			updated_at = $1
		FROM accounts account
		WHERE recap.id = $4
		  AND account.id = recap.account_id
		RETURNING recap.id::text, recap.account_id::text,
		          account.primary_email, recap.week_start::text, recap.payload
	`, now, token, expiresAt, recapID).Scan(
		&claim.ID,
		&claim.AccountID,
		&claim.PrimaryEmail,
		&claim.WeekStart,
		&raw,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &claim.Payload); err != nil {
		return nil, err
	}
	claim.Token = token.String()
	claim.ExpiresAt = expiresAt
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &claim, nil
}

func (s *Store) RenewWeeklyRecapClaim(
	ctx context.Context,
	recapID, token string,
	expiresAt time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE weekly_recaps SET claim_expires_at = $3, updated_at = now()
		WHERE id = $1 AND claim_token = $2 AND status = 'delivering'
		  AND claim_expires_at > now()
	`, recapID, token, expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) CompleteWeeklyRecap(
	ctx context.Context,
	recapID, token, externalID string,
	now time.Time,
) error {
	if strings.TrimSpace(externalID) == "" {
		return errors.New("weekly Recap external ID is required")
	}
	return s.transitionWeeklyRecap(
		ctx,
		recapID,
		token,
		"delivered",
		externalID,
		"",
		now,
	)
}

func (s *Store) FailWeeklyRecap(
	ctx context.Context,
	recapID, token string,
	cause error,
	maxAttempts int,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE weekly_recaps SET
			status = 'failed',
			available_at = $3::timestamptz + make_interval(
				secs => LEAST(3600, 30 * power(2, GREATEST(0, attempt_count - 1)))::int
			),
			claim_token = NULL,
			claim_expires_at = NULL,
			error = $5,
			completed_at = CASE WHEN attempt_count >= $4 THEN $3::timestamptz ELSE NULL END,
			updated_at = $3
		WHERE id = $1 AND claim_token = $2 AND status = 'delivering'
	`, recapID, token, now, maxAttempts, safeStoreError(cause))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) MarkWeeklyRecapUnknown(
	ctx context.Context,
	recapID, token string,
	cause error,
	now time.Time,
) error {
	return s.transitionWeeklyRecap(
		ctx,
		recapID,
		token,
		"unknown",
		"",
		safeStoreError(cause),
		now,
	)
}

func (s *Store) transitionWeeklyRecap(
	ctx context.Context,
	recapID, token, status, externalID, message string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE weekly_recaps SET
			status = $3,
			external_id = NULLIF($4, ''),
			error = NULLIF($5, ''),
			completed_at = $6,
			claim_token = NULL,
			claim_expires_at = NULL,
			updated_at = $6
		WHERE id = $1 AND claim_token = $2 AND status = 'delivering'
		  AND claim_expires_at > $6
	`, recapID, token, status, externalID, message, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}
