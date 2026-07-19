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

type IssueClaim struct {
	Issue        domain.Issue
	AccountID    string
	PrimaryEmail string
	Token        string
	ExpiresAt    time.Time
}

type CompleteIssueInput struct {
	ClaimToken   string
	GenerationID string
	ArtifactKey  string
	Checksum     string
	Bytes        int
	Title        string
	History      domain.LearningHistoryEntry
	HistoryLimit int
	CompletedAt  time.Time
}

type DeliveryClaim struct {
	Issue        domain.Issue
	AccountID    string
	PrimaryEmail string
	SiteUsername string
	SitePublic   bool
	Receipt      domain.DeliveryReceipt
	Token        string
	ExpiresAt    time.Time
}

func (s *Store) EnqueueManualIssue(
	ctx context.Context,
	accountID, newsletterID string,
	dailyAccountLimit int,
) (domain.Issue, error) {
	if dailyAccountLimit < 1 {
		dailyAccountLimit = 5
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	defer rollback(tx)
	var allowed bool
	if err := tx.QueryRow(ctx, `
		SELECT a.status = 'active'
		FROM newsletters n
		JOIN accounts a ON a.id = n.owner_account_id
		WHERE n.id = $1 AND n.owner_account_id = $2
		FOR UPDATE OF n
	`, newsletterID, accountID).Scan(&allowed); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Issue{}, ErrNotFound
		}
		return domain.Issue{}, err
	}
	if !allowed {
		return domain.Issue{}, ErrForbidden
	}
	var today int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1
		  AND i.created_at >= date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
		  AND i.status <> 'cancelled'
	`, accountID).Scan(&today); err != nil {
		return domain.Issue{}, err
	}
	if today >= dailyAccountLimit {
		return domain.Issue{}, ErrQuotaExceeded
	}
	now := time.Now().UTC()
	id := uuid.New()
	publicID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO issues (
			id, newsletter_id, trigger, status, available_at, public_id,
			publication_state, created_at
		)
		VALUES ($1, $2, 'manual', 'queued', $3, $4, 'published', $3)
	`, id, newsletterID, now, publicID); err != nil {
		return domain.Issue{}, fmt.Errorf("enqueue manual Issue: %w", err)
	}
	issue, err := getIssueTx(ctx, tx, accountID, id.String())
	if err != nil {
		return domain.Issue{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Issue{}, err
	}
	return issue, nil
}

func (s *Store) DispatchDue(
	ctx context.Context,
	now time.Time,
	maximum int,
) (int, error) {
	if maximum < 1 {
		maximum = 100
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	rows, err := tx.Query(ctx, `
		SELECT n.id::text, n.time_zone, n.schedule_hour, n.schedule_minute,
		       n.next_run_at
		FROM newsletters n
		JOIN accounts a ON a.id = n.owner_account_id
		WHERE n.active AND a.status = 'active' AND n.next_run_at <= $1
		ORDER BY n.next_run_at, n.id
		FOR UPDATE OF n SKIP LOCKED
		LIMIT $2
	`, now, maximum)
	if err != nil {
		return 0, fmt.Errorf("select due Newsletters: %w", err)
	}
	type dueNewsletter struct {
		id     string
		zone   string
		hour   int
		minute int
		dueAt  time.Time
	}
	var due []dueNewsletter
	for rows.Next() {
		var item dueNewsletter
		if err := rows.Scan(&item.id, &item.zone, &item.hour, &item.minute, &item.dueAt); err != nil {
			rows.Close()
			return 0, err
		}
		due = append(due, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	dispatched := 0
	for _, newsletter := range due {
		location, err := time.LoadLocation(newsletter.zone)
		if err != nil {
			return 0, fmt.Errorf("Newsletter %s timezone: %w", newsletter.id, err)
		}
		localDate := newsletter.dueAt.In(location).Format(time.DateOnly)
		tag, err := tx.Exec(ctx, `
			INSERT INTO issues (
				id, newsletter_id, trigger, scheduled_local_date, status,
				available_at, public_id, publication_state, created_at
			)
			VALUES ($1, $2, 'scheduled', $3::date, 'queued', $4, $5, 'published', $4)
			ON CONFLICT (newsletter_id, scheduled_local_date)
				WHERE trigger = 'scheduled'
			DO NOTHING
		`, uuid.New(), newsletter.id, localDate, now, uuid.New())
		if err != nil {
			return 0, fmt.Errorf("dispatch Newsletter %s: %w", newsletter.id, err)
		}
		dispatched += int(tag.RowsAffected())
		next, err := NextOccurrence(now, newsletter.zone, newsletter.hour, newsletter.minute)
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE newsletters SET next_run_at = $2, updated_at = $3 WHERE id = $1
		`, newsletter.id, next, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return dispatched, nil
}

func (s *Store) RecoverExpiredClaims(
	ctx context.Context,
	now time.Time,
	maxIssueAttempts, maxDeliveryAttempts int,
) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	issues, err := tx.Exec(ctx, `
		UPDATE issues SET
			status = CASE WHEN attempt_count >= $2 THEN 'failed' ELSE 'queued' END,
			available_at = CASE
				WHEN attempt_count >= $2 THEN available_at
				ELSE $1 + make_interval(secs => LEAST(900, 15 * power(2, GREATEST(0, attempt_count - 1)))::int)
			END,
			claim_token = NULL,
			claim_expires_at = NULL,
			error = 'Worker claim expired before completion'
		WHERE status = 'generating' AND claim_expires_at <= $1
	`, now, maxIssueAttempts)
	if err != nil {
		return 0, fmt.Errorf("recover Issue Claims: %w", err)
	}
	deliveries, err := tx.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = CASE WHEN attempt_count >= $2 THEN 'failed' ELSE 'failed' END,
			available_at = CASE
				WHEN attempt_count >= $2 THEN available_at
				ELSE $1 + make_interval(secs => LEAST(3600, 30 * power(2, GREATEST(0, attempt_count - 1)))::int)
			END,
			claim_token = NULL,
			claim_expires_at = NULL,
			error = 'Worker claim expired before completion',
			updated_at = $1
		WHERE status = 'delivering' AND claim_expires_at <= $1
	`, now, maxDeliveryAttempts)
	if err != nil {
		return 0, fmt.Errorf("recover Delivery Receipt Claims: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return issues.RowsAffected() + deliveries.RowsAffected(), nil
}

func (s *Store) ClaimNextIssue(
	ctx context.Context,
	now time.Time,
	claimDuration time.Duration,
	accountConcurrency, dailyAccountLimit, dailyGlobalLimit int,
) (*IssueClaim, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var paused bool
	if err := tx.QueryRow(ctx, `
		SELECT generation_paused FROM runtime_controls WHERE id = true
	`).Scan(&paused); err != nil {
		return nil, err
	}
	if paused {
		return nil, ErrGenerationPaused
	}
	var globalToday int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM issues
		WHERE started_at >= date_trunc('day', $1 AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
	`, now).Scan(&globalToday); err != nil {
		return nil, err
	}
	if globalToday >= dailyGlobalLimit {
		return nil, ErrQuotaExceeded
	}
	var issueID string
	err = tx.QueryRow(ctx, `
		WITH account_activity AS (
			SELECT n.owner_account_id,
			       count(*) FILTER (WHERE i.status = 'generating') AS active_count,
			       count(*) FILTER (
			         WHERE i.started_at >= date_trunc('day', $1 AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
			       ) AS daily_count,
			       max(i.started_at) AS last_started_at
			FROM newsletters n
			LEFT JOIN issues i ON i.newsletter_id = n.id
			GROUP BY n.owner_account_id
		),
		candidates AS (
			SELECT i.id, n.owner_account_id,
			       row_number() OVER (
			         PARTITION BY n.owner_account_id ORDER BY i.available_at, i.created_at, i.id
			       ) AS account_rank,
			       aa.last_started_at
			FROM issues i
			JOIN newsletters n ON n.id = i.newsletter_id
			JOIN accounts a ON a.id = n.owner_account_id
			JOIN account_activity aa ON aa.owner_account_id = n.owner_account_id
			WHERE i.status = 'queued' AND i.available_at <= $1
			  AND (n.active OR i.trigger = 'manual') AND a.status = 'active'
			  AND aa.active_count < $2
			  AND aa.daily_count < $3
		)
		SELECT i.id::text
		FROM issues i
		JOIN candidates c ON c.id = i.id
		WHERE c.account_rank = 1
		ORDER BY c.last_started_at NULLS FIRST, i.created_at, i.id
		FOR UPDATE OF i SKIP LOCKED
		LIMIT 1
	`, now, accountConcurrency, dailyAccountLimit).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select Issue Claim: %w", err)
	}
	token := uuid.New()
	expires := now.Add(claimDuration)
	tag, err := tx.Exec(ctx, `
		UPDATE issues SET
			status = 'generating',
			attempt_count = attempt_count + 1,
			claim_token = $2,
			claim_expires_at = $3,
			started_at = $1,
			error = NULL
		WHERE id = $4 AND status = 'queued'
	`, now, token, expires, issueID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrClaimLost
	}
	issue, accountID, email, err := getWorkerIssue(ctx, tx, issueID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &IssueClaim{
		Issue: issue, AccountID: accountID, PrimaryEmail: email,
		Token: token.String(), ExpiresAt: expires,
	}, nil
}

func (s *Store) RenewIssueClaim(
	ctx context.Context,
	issueID, token string,
	expiresAt time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE issues SET claim_expires_at = $3
		WHERE id = $1 AND claim_token = $2 AND status = 'generating'
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

func (s *Store) LoadLearningHistory(
	ctx context.Context,
	newsletterID string,
	limit int,
) ([]domain.LearningHistoryEntry, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT entry FROM learning_history
		WHERE newsletter_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, newsletterID, limit)
	if err != nil {
		return nil, fmt.Errorf("load Learning History: %w", err)
	}
	defer rows.Close()
	var reversed []domain.LearningHistoryEntry
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var entry domain.LearningHistoryEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, fmt.Errorf("decode Learning History: %w", err)
		}
		reversed = append(reversed, entry)
	}
	history := make([]domain.LearningHistoryEntry, len(reversed))
	for index := range reversed {
		history[len(reversed)-1-index] = reversed[index]
	}
	return history, rows.Err()
}

func (s *Store) CompleteIssue(
	ctx context.Context,
	issueID string,
	input CompleteIssueInput,
) error {
	if input.CompletedAt.IsZero() {
		input.CompletedAt = time.Now().UTC()
	}
	history, err := json.Marshal(input.History)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var newsletterID string
	var emailEnabled bool
	var primaryEmail *string
	err = tx.QueryRow(ctx, `
		SELECT i.newsletter_id::text, n.email_enabled, a.primary_email
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		JOIN accounts a ON a.id = n.owner_account_id
		WHERE i.id = $1 AND i.status = 'generating'
		  AND i.claim_token = $2 AND i.claim_expires_at > $3
		FOR UPDATE OF i
	`, issueID, input.ClaimToken, input.CompletedAt).Scan(
		&newsletterID,
		&emailEnabled,
		&primaryEmail,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrClaimLost
	}
	if err != nil {
		return err
	}
	publicSlug := slugify(input.Title)
	if publicSlug == "" {
		publicSlug = "dossier"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE issues SET
			status = 'generated', dossier_title = $3, generation_id = $4,
			artifact_key = $5, artifact_sha256 = $6, artifact_bytes = $7,
			public_slug = $8, completed_at = $9,
			claim_token = NULL, claim_expires_at = NULL, error = NULL
		WHERE id = $1 AND claim_token = $2
	`, issueID, input.ClaimToken, input.Title, input.GenerationID,
		input.ArtifactKey, input.Checksum, input.Bytes, publicSlug, input.CompletedAt); err != nil {
		return fmt.Errorf("complete Issue: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO learning_history (
			newsletter_id, issue_id, local_date, entry, created_at
		)
		VALUES ($1, $2, $3::date, $4::jsonb, $5)
	`, newsletterID, issueID, input.History.Date, history, input.CompletedAt); err != nil {
		return fmt.Errorf("append Learning History: %w", err)
	}
	if input.HistoryLimit >= 0 {
		if _, err := tx.Exec(ctx, `
			DELETE FROM learning_history
			WHERE newsletter_id = $1 AND issue_id IN (
				SELECT issue_id FROM learning_history
				WHERE newsletter_id = $1
				ORDER BY created_at DESC
				OFFSET $2
			)
		`, newsletterID, input.HistoryLimit); err != nil {
			return fmt.Errorf("trim Learning History: %w", err)
		}
	}
	if emailEnabled && primaryEmail != nil && strings.TrimSpace(*primaryEmail) != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO delivery_receipts (
				issue_id, status, attempt_count, available_at,
				created_at, updated_at
			)
			VALUES ($1, 'pending', 0, $2, $2, $2)
			ON CONFLICT (issue_id) DO NOTHING
		`, issueID, input.CompletedAt); err != nil {
			return fmt.Errorf("enqueue Delivery Receipt: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) FailIssue(
	ctx context.Context,
	issueID, token string,
	cause error,
	maxAttempts int,
	now time.Time,
) error {
	message := safeStoreError(cause)
	var attempts int
	if err := s.pool.QueryRow(ctx, `
		UPDATE issues SET
			status = CASE WHEN attempt_count >= $4 THEN 'failed' ELSE 'queued' END,
			available_at = CASE
				WHEN attempt_count >= $4 THEN available_at
				ELSE $3 + make_interval(secs => LEAST(900, 15 * power(2, GREATEST(0, attempt_count - 1)))::int)
			END,
			claim_token = NULL, claim_expires_at = NULL, error = $5,
			completed_at = CASE WHEN attempt_count >= $4 THEN $3 ELSE NULL END
		WHERE id = $1 AND claim_token = $2 AND status = 'generating'
		RETURNING attempt_count
	`, issueID, token, now, maxAttempts, message).Scan(&attempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClaimLost
		}
		return fmt.Errorf("fail Issue: %w", err)
	}
	return nil
}

func (s *Store) ListIssues(
	ctx context.Context,
	accountID, newsletterID string,
	limit int,
) ([]domain.Issue, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, workerIssueSelect+`
		WHERE n.owner_account_id = $1 AND i.newsletter_id = $2
		ORDER BY i.created_at DESC
		LIMIT $3
	`, accountID, newsletterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list Issues: %w", err)
	}
	defer rows.Close()
	var issues []domain.Issue
	for rows.Next() {
		issue, _, _, err := scanWorkerIssue(rows)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	receipts, err := s.listDeliveries(ctx, accountID, newsletterID)
	if err != nil {
		return nil, err
	}
	for index := range issues {
		if receipt, exists := receipts[issues[index].ID]; exists {
			value := receipt
			issues[index].Delivery = &value
		}
	}
	return issues, nil
}

func (s *Store) GetIssue(
	ctx context.Context,
	accountID, issueID string,
) (domain.Issue, error) {
	issue, err := getIssueTx(ctx, s.pool, accountID, issueID)
	if err != nil {
		return domain.Issue{}, err
	}
	receipt, err := s.GetDelivery(ctx, accountID, issueID)
	if err != nil {
		return domain.Issue{}, err
	}
	issue.Delivery = receipt
	return issue, nil
}

func (s *Store) SetIssuePublication(
	ctx context.Context,
	accountID, issueID string,
	state domain.PublicationState,
) error {
	if state != domain.PublicationPublished && state != domain.PublicationHidden {
		return errors.New("Issue publication state is invalid")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE issues i SET publication_state = $3
		FROM newsletters n
		WHERE i.newsletter_id = n.id AND n.owner_account_id = $1 AND i.id = $2
	`, accountID, issueID, state)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func getIssueTx(
	ctx context.Context,
	queryer interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	accountID, issueID string,
) (domain.Issue, error) {
	row := queryer.QueryRow(ctx, workerIssueSelect+`
		WHERE n.owner_account_id = $1 AND i.id = $2
	`, accountID, issueID)
	issue, _, _, err := scanWorkerIssue(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Issue{}, ErrNotFound
	}
	return issue, err
}

func getWorkerIssue(
	ctx context.Context,
	queryer interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	issueID string,
) (domain.Issue, string, string, error) {
	return scanWorkerIssue(queryer.QueryRow(ctx, workerIssueSelect+`
		WHERE i.id = $1
	`, issueID))
}

const workerIssueSelect = `
	SELECT
		i.id::text, i.newsletter_id::text, i.trigger,
		i.scheduled_local_date::text, i.status, COALESCE(i.dossier_title, ''),
		COALESCE(i.generation_id::text, ''), COALESCE(i.artifact_key, ''),
		COALESCE(i.error, ''), 'dossier-' || i.public_id::text,
		COALESCE(i.public_slug, ''), i.publication_state, i.created_at,
		i.started_at, i.completed_at,
		n.id::text, n.owner_account_id::text, n.name, n.topic, n.learner_level,
		n.learner_goal, n.lesson_minutes, n.sources, n.schedule_hour,
		n.schedule_minute, n.time_zone, n.active, n.next_run_at,
		n.email_enabled, n.ai_exploration_enabled, n.public_slug,
		n.site_visible, n.created_at, n.updated_at,
		a.id::text, COALESCE(a.primary_email, '')
	FROM issues i
	JOIN newsletters n ON n.id = i.newsletter_id
	JOIN accounts a ON a.id = n.owner_account_id
`

func scanWorkerIssue(row scanner) (domain.Issue, string, string, error) {
	var issue domain.Issue
	var newsletter domain.Newsletter
	var scheduledDate *string
	var rawSources []byte
	var accountID, email string
	err := row.Scan(
		&issue.ID,
		&issue.NewsletterID,
		&issue.Trigger,
		&scheduledDate,
		&issue.Status,
		&issue.Title,
		&issue.GenerationID,
		&issue.ArtifactKey,
		&issue.Error,
		&issue.PublicID,
		&issue.PublicSlug,
		&issue.PublicationState,
		&issue.CreatedAt,
		&issue.StartedAt,
		&issue.CompletedAt,
		&newsletter.ID,
		&newsletter.OwnerAccountID,
		&newsletter.Name,
		&newsletter.Topic,
		&newsletter.LearnerLevel,
		&newsletter.LearnerGoal,
		&newsletter.LessonMinutes,
		&rawSources,
		&newsletter.ScheduleHour,
		&newsletter.ScheduleMinute,
		&newsletter.TimeZone,
		&newsletter.Active,
		&newsletter.NextRunAt,
		&newsletter.EmailEnabled,
		&newsletter.AIExplorationEnabled,
		&newsletter.PublicSlug,
		&newsletter.SiteVisible,
		&newsletter.CreatedAt,
		&newsletter.UpdatedAt,
		&accountID,
		&email,
	)
	if err != nil {
		return domain.Issue{}, "", "", err
	}
	if err := json.Unmarshal(rawSources, &newsletter.Sources); err != nil {
		return domain.Issue{}, "", "", err
	}
	issue.ScheduledLocalDate = scheduledDate
	issue.Newsletter = newsletter
	return issue, accountID, email, nil
}

func safeStoreError(err error) string {
	if err == nil {
		return "unknown error"
	}
	return truncateStore(strings.Join(strings.Fields(err.Error()), " "), 500)
}

func truncateStore(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum])
}
