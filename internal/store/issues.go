package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/failure"
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

type IssueAttemptContext struct {
	WorkerID          string
	DeploymentVersion string
	ModelName         string
	PipelineVersion   string
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

type WorkspaceReview struct {
	ID                    string     `json:"id"`
	IssueID               string     `json:"issueId"`
	Objective             string     `json:"objective"`
	Prompt                string     `json:"prompt"`
	AnswerRubric          string     `json:"answerRubric"`
	CorrectiveExplanation string     `json:"correctiveExplanation"`
	Stage                 int        `json:"stage"`
	DueAt                 time.Time  `json:"dueAt"`
	LastReviewedAt        *time.Time `json:"lastReviewedAt,omitempty"`
}

type WorkspaceIssueCursor struct {
	CreatedAt time.Time
	IssueID   string
}

type LibraryFilter string

const (
	LibraryAll        LibraryFilter = "all"
	LibraryUnread     LibraryFilter = "unread"
	LibraryInProgress LibraryFilter = "in-progress"
	LibraryCompleted  LibraryFilter = "completed"
)

type LibraryLesson struct {
	ID               string          `json:"id"`
	NewsletterID     string          `json:"newsletterId"`
	Title            string          `json:"title"`
	Status           string          `json:"status"`
	PublicationState string          `json:"publicationState"`
	CreatedAt        time.Time       `json:"createdAt"`
	Newsletter       LibraryStream   `json:"newsletter"`
	Progress         *LessonProgress `json:"progress,omitempty"`
	Concepts         []string        `json:"concepts,omitempty"`
	SourceTitles     []string        `json:"sourceTitles,omitempty"`
}

type LibraryStream struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Topic         string `json:"topic"`
	LearnerLevel  string `json:"learnerLevel"`
	LessonMinutes int    `json:"lessonMinutes"`
}

func (s *Store) LoadIssueCheckpoints(
	ctx context.Context,
	issueID, fingerprint string,
) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT stage, output
		FROM issue_generation_checkpoints
		WHERE issue_id = $1::uuid AND fingerprint = $2
	`, issueID, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("load Issue checkpoints: %w", err)
	}
	defer rows.Close()
	checkpoints := map[string]string{}
	for rows.Next() {
		var stage, output string
		if err := rows.Scan(&stage, &output); err != nil {
			return nil, err
		}
		checkpoints[stage] = output
	}
	return checkpoints, rows.Err()
}

func (s *Store) SaveIssueCheckpoint(
	ctx context.Context,
	issueID, token, fingerprint, stage, output, pipelineVersion string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO issue_generation_checkpoints (
			issue_id, fingerprint, stage, output, pipeline_version,
			created_at, updated_at
		)
		SELECT i.id, $3, $4, $5, $6, $7, $7
		FROM issues i
		WHERE i.id = $1::uuid AND i.claim_token = $2::uuid
		  AND i.status = 'generating'
		ON CONFLICT (issue_id, fingerprint, stage) DO UPDATE SET
			output = EXCLUDED.output,
			pipeline_version = EXCLUDED.pipeline_version,
			updated_at = EXCLUDED.updated_at
	`, issueID, token, fingerprint, stage, output, pipelineVersion, now)
	if err != nil {
		return fmt.Errorf("save Issue checkpoint: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
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
	maxIssueAttempts, _ int,
) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	var issues int64
	err = tx.QueryRow(ctx, `
		WITH candidates AS MATERIALIZED (
			SELECT id, claim_token, claim_loss_count, gen_random_uuid() AS incident_id
			FROM issues
			WHERE status = 'generating' AND claim_expires_at <= $1
			FOR UPDATE
		),
		closed_attempts AS (
			UPDATE issue_attempts a SET
				status = 'abandoned',
				completed_at = $1,
				failure_code = 'worker_claim_expired',
				failure_category = 'infrastructure',
				failure_retryable = true,
				internal_error = 'Worker claim expired before completion',
				incident_id = c.incident_id
			FROM candidates c
			WHERE a.id = c.claim_token AND a.status = 'running'
		),
		recovered AS (
			UPDATE issues i SET
				status = CASE
					WHEN c.claim_loss_count + 1 >= $2 * 2 THEN 'failed'
					ELSE 'queued'
				END,
				attempt_count = GREATEST(0, i.attempt_count - 1),
				claim_loss_count = c.claim_loss_count + 1,
				available_at = $1 + make_interval(
					secs => LEAST(900, 15 * power(2, GREATEST(0, c.claim_loss_count)))::int
				),
				claim_token = NULL,
				claim_expires_at = NULL,
				error = 'Worker claim expired before completion',
				failure_code = 'worker_claim_expired',
				failure_category = 'infrastructure',
				failure_stage = NULL,
				failure_retryable = true,
				public_error = CASE
					WHEN c.claim_loss_count + 1 >= $2 * 2
					THEN 'We couldn’t prepare this lesson. We’ve been notified, and you can retry now.'
					ELSE NULL
				END,
				incident_id = c.incident_id,
				completed_at = CASE
					WHEN c.claim_loss_count + 1 >= $2 * 2 THEN $1
					ELSE NULL
				END
			FROM candidates c
			WHERE i.id = c.id
			RETURNING i.id
		)
		SELECT count(*) FROM recovered
	`, now, maxIssueAttempts).Scan(&issues)
	if err != nil {
		return 0, fmt.Errorf("recover Issue Claims: %w", err)
	}
	deliveries, err := tx.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = 'unknown',
			claim_token = NULL,
			claim_expires_at = NULL,
			error = 'Worker claim expired before completion',
			completed_at = $1,
			updated_at = $1
		WHERE status = 'delivering' AND claim_expires_at <= $1
	`, now)
	if err != nil {
		return 0, fmt.Errorf("recover Delivery Receipt Claims: %w", err)
	}
	recaps, err := tx.Exec(ctx, `
		UPDATE weekly_recaps SET
			status = 'unknown',
			claim_token = NULL,
			claim_expires_at = NULL,
			error = 'Worker claim expired before completion',
			completed_at = $1,
			updated_at = $1
		WHERE status = 'delivering' AND claim_expires_at <= $1
	`, now)
	if err != nil {
		return 0, fmt.Errorf("recover weekly Recap Claims: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return issues + deliveries.RowsAffected() + recaps.RowsAffected(), nil
}

func (s *Store) ClaimNextIssue(
	ctx context.Context,
	now time.Time,
	claimDuration time.Duration,
	accountConcurrency, dailyAccountLimit, dailyGlobalLimit int,
	attemptContext IssueAttemptContext,
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
	var attemptNumber int
	err = tx.QueryRow(ctx, `
		UPDATE issues SET
			status = 'generating',
			attempt_count = attempt_count + 1,
			claim_token = $2,
			claim_expires_at = $3,
			started_at = $1,
			error = NULL,
			failure_code = NULL,
			failure_category = NULL,
			failure_stage = NULL,
			failure_retryable = NULL,
			public_error = NULL,
			incident_id = NULL
		WHERE id = $4 AND status = 'queued'
		RETURNING attempt_count
	`, now, token, expires, issueID).Scan(&attemptNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClaimLost
		}
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO issue_attempts (
			id, issue_id, attempt_number, status, started_at, last_renewed_at,
			worker_id, deployment_version, model_name, pipeline_version
		)
		VALUES ($1, $2, $3, 'running', $4, $4, $5, $6, $7, $8)
	`, token, issueID, attemptNumber, now,
		fallbackAttemptValue(attemptContext.WorkerID),
		fallbackAttemptValue(attemptContext.DeploymentVersion),
		fallbackAttemptValue(attemptContext.ModelName),
		fallbackAttemptValue(attemptContext.PipelineVersion)); err != nil {
		return nil, fmt.Errorf("record Issue attempt: %w", err)
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

func fallbackAttemptValue(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return truncateStore(value, 200)
	}
	return "unknown"
}

func (s *Store) RenewIssueClaim(
	ctx context.Context,
	issueID, token string,
	expiresAt time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		WITH renewed AS (
			UPDATE issues SET claim_expires_at = $3
			WHERE id = $1 AND claim_token = $2 AND status = 'generating'
			  AND claim_expires_at > now()
			RETURNING claim_token
		)
		UPDATE issue_attempts SET last_renewed_at = now()
		WHERE id = (SELECT claim_token FROM renewed)
	`, issueID, token, expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Store) ReleaseIssueClaim(
	ctx context.Context,
	issueID, token string,
	cause error,
	now time.Time,
) error {
	detail := failure.Describe(cause)
	tag, err := s.pool.Exec(ctx, `
		WITH released AS (
			UPDATE issues SET
				status = 'queued',
				attempt_count = GREATEST(0, attempt_count - 1),
				available_at = $3::timestamptz + interval '15 seconds',
				claim_token = NULL,
				claim_expires_at = NULL,
				error = $4,
				failure_code = $5,
				failure_category = $6,
				failure_stage = NULLIF($7, ''),
				failure_retryable = true,
				public_error = NULL,
				incident_id = $8::uuid,
				completed_at = NULL
			WHERE id = $1::uuid AND claim_token = $2::uuid AND status = 'generating'
			RETURNING id
		)
		UPDATE issue_attempts SET
			status = 'abandoned',
			completed_at = $3::timestamptz,
			failure_code = $5,
			failure_category = $6,
			failure_stage = NULLIF($7, ''),
			failure_retryable = true,
			internal_error = $4,
			incident_id = $8::uuid
		WHERE id = $2::uuid AND issue_id = (SELECT id FROM released)
		  AND status = 'running'
	`, issueID, token, now, truncateStore(detail.Internal, 500),
		detail.Code, detail.Category, detail.Stage, detail.IncidentID)
	if err != nil {
		return fmt.Errorf("release Issue Claim: %w", err)
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
	var accountID string
	var emailEnabled bool
	var primaryEmail *string
	err = tx.QueryRow(ctx, `
		SELECT i.newsletter_id::text, n.owner_account_id::text,
		       n.email_enabled, a.primary_email
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		JOIN accounts a ON a.id = n.owner_account_id
		WHERE i.id = $1 AND i.status = 'generating'
		  AND i.claim_token = $2 AND i.claim_expires_at > $3
		FOR UPDATE OF i
	`, issueID, input.ClaimToken, input.CompletedAt).Scan(
		&newsletterID,
		&accountID,
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
			claim_token = NULL, claim_expires_at = NULL, error = NULL,
			failure_code = NULL, failure_category = NULL, failure_stage = NULL,
			failure_retryable = NULL, public_error = NULL, incident_id = NULL
		WHERE id = $1 AND claim_token = $2
	`, issueID, input.ClaimToken, input.Title, input.GenerationID,
		input.ArtifactKey, input.Checksum, input.Bytes, publicSlug, input.CompletedAt); err != nil {
		return fmt.Errorf("complete Issue: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE issue_attempts SET
			status = 'completed', completed_at = $3, last_renewed_at = $3
		WHERE id = $2 AND issue_id = $1 AND status = 'running'
	`, issueID, input.ClaimToken, input.CompletedAt); err != nil {
		return fmt.Errorf("complete Issue attempt: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO learning_history (
			newsletter_id, issue_id, local_date, entry, created_at
		)
		VALUES ($1, $2, $3::date, $4::jsonb, $5)
	`, newsletterID, issueID, input.History.Date, history, input.CompletedAt); err != nil {
		return fmt.Errorf("append Learning History: %w", err)
	}
	retrievalPrompts := make([]string, 0, len(input.History.RetrievalPrompts))
	for _, prompt := range input.History.RetrievalPrompts {
		retrievalPrompts = append(retrievalPrompts, prompt.Prompt)
	}
	searchText := strings.Join([]string{
		input.Title,
		strings.Join(input.History.Concepts, " "),
		strings.Join(input.History.SourceTitles, " "),
		strings.Join(retrievalPrompts, " "),
	}, " ")
	if _, err := tx.Exec(ctx, `
		INSERT INTO lesson_search_documents (
			issue_id, account_id, newsletter_id, title, concepts,
			source_titles, retrieval_prompts, search_text, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (issue_id) DO UPDATE SET
			title = EXCLUDED.title,
			concepts = EXCLUDED.concepts,
			source_titles = EXCLUDED.source_titles,
			retrieval_prompts = EXCLUDED.retrieval_prompts,
			search_text = EXCLUDED.search_text
	`, issueID, accountID, newsletterID, input.Title, input.History.Concepts,
		input.History.SourceTitles, retrievalPrompts, searchText,
		input.CompletedAt); err != nil {
		return fmt.Errorf("project Lesson search document: %w", err)
	}
	for _, concept := range input.History.ConceptStates {
		if _, err := tx.Exec(ctx, `
			INSERT INTO issue_concepts (
				account_id, issue_id, newsletter_id, concept_key,
				label, role, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (issue_id, concept_key) DO NOTHING
		`, accountID, issueID, newsletterID, concept.ID, concept.Label,
			concept.Role, input.CompletedAt); err != nil {
			return fmt.Errorf("record Issue Concept: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO learner_concept_state (
				account_id, newsletter_id, concept_key, label, role,
				exposure_count, last_seen_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, 1, $6, $6)
			ON CONFLICT (account_id, newsletter_id, concept_key) DO UPDATE SET
				label = EXCLUDED.label,
				role = EXCLUDED.role,
				exposure_count = learner_concept_state.exposure_count + 1,
				last_seen_at = EXCLUDED.last_seen_at,
				updated_at = EXCLUDED.updated_at
		`, accountID, newsletterID, concept.ID, concept.Label, concept.Role,
			input.CompletedAt); err != nil {
			return fmt.Errorf("project Learner Concept exposure: %w", err)
		}
	}
	for _, prompt := range input.History.RetrievalPrompts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO review_items (
				account_id, issue_id, prompt_key, prompt, answer_rubric,
				corrective_explanation, objective, concept_keys,
				scheduler_version, stage, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1, 0, $9, $9)
			ON CONFLICT (issue_id, prompt_key) DO NOTHING
		`, accountID, issueID, prompt.ID, prompt.Prompt, prompt.AnswerRubric,
			prompt.CorrectiveExplanation, input.History.LearningObjective,
			prompt.ConceptIDs, input.CompletedAt); err != nil {
			return fmt.Errorf("create Review Item: %w", err)
		}
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
	if _, err := tx.Exec(ctx, `
		DELETE FROM issue_generation_checkpoints WHERE issue_id = $1
	`, issueID); err != nil {
		return fmt.Errorf("delete completed Issue checkpoints: %w", err)
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
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.RecordProductEvent(
		ctx,
		accountID,
		ProductEventLessonGenerated,
		"lesson",
		issueID,
		input.CompletedAt,
	)
	return nil
}

func (s *Store) FailIssue(
	ctx context.Context,
	issueID, token string,
	cause error,
	maxAttempts int,
	now time.Time,
) error {
	detail := failure.Describe(cause)
	message := truncateStore(detail.Internal, 500)
	var attempts int
	if err := s.pool.QueryRow(ctx, `
		WITH failed_attempt AS (
			UPDATE issue_attempts SET
				status = 'failed',
				completed_at = $3::timestamptz,
				failure_code = $5,
				failure_category = $6,
				failure_stage = NULLIF($7, ''),
				failure_retryable = $8,
				internal_error = $9,
				incident_id = $10::uuid
			WHERE id = $2::uuid AND issue_id = $1::uuid AND status = 'running'
		)
		UPDATE issues SET
			status = CASE
				WHEN NOT $8 OR attempt_count >= $4 THEN 'failed'
				ELSE 'queued'
			END,
			available_at = CASE
				WHEN NOT $8 OR attempt_count >= $4 THEN available_at
				ELSE $3::timestamptz + make_interval(secs => LEAST(900, 15 * power(2, GREATEST(0, attempt_count - 1)))::int)
			END,
			claim_token = NULL, claim_expires_at = NULL, error = $9,
			failure_code = $5,
			failure_category = $6,
			failure_stage = NULLIF($7, ''),
			failure_retryable = $8,
			public_error = CASE
				WHEN NOT $8 OR attempt_count >= $4 THEN $11
				ELSE NULL
			END,
			incident_id = $10::uuid,
			completed_at = CASE
				WHEN NOT $8 OR attempt_count >= $4 THEN $3::timestamptz
				ELSE NULL
			END
		WHERE id = $1 AND claim_token = $2 AND status = 'generating'
		RETURNING attempt_count
	`, issueID, token, now, maxAttempts, detail.Code, detail.Category,
		detail.Stage, detail.Retryable, message, detail.IncidentID,
		detail.PublicMessage).Scan(&attempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClaimLost
		}
		return fmt.Errorf("fail Issue: %w", err)
	}
	return nil
}

func (s *Store) RecordIssueStage(
	ctx context.Context,
	issueID, token, stage string,
	duration time.Duration,
	stageErr error,
	now time.Time,
) error {
	status := "completed"
	var detail failure.Detail
	if stageErr != nil {
		status = "failed"
		detail = failure.Describe(stageErr)
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO issue_stage_attempts (
			issue_attempt_id, stage, status, duration_ms,
			failure_code, internal_error, recorded_at
		)
		SELECT a.id, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8
		FROM issue_attempts a
		WHERE a.id = $2::uuid AND a.issue_id = $1::uuid
		ON CONFLICT (issue_attempt_id, stage) DO UPDATE SET
			status = EXCLUDED.status,
			duration_ms = EXCLUDED.duration_ms,
			failure_code = EXCLUDED.failure_code,
			internal_error = EXCLUDED.internal_error,
			recorded_at = EXCLUDED.recorded_at
	`, issueID, token, stage, status, max(0, duration.Milliseconds()),
		detail.Code, truncateStore(detail.Internal, 500), now)
	if err != nil {
		return fmt.Errorf("record Issue stage: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrClaimLost
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

func (s *Store) ListWorkspaceIssuesPage(
	ctx context.Context,
	accountID string,
	limit int,
	cursor *WorkspaceIssueCursor,
) ([]domain.Issue, *WorkspaceIssueCursor, error) {
	if limit < 1 || limit > 100 {
		limit = 40
	}
	hasCursor := cursor != nil
	cursorCreatedAt := time.Time{}
	cursorIssueID := uuid.Nil.String()
	if cursor != nil {
		cursorCreatedAt = cursor.CreatedAt
		cursorIssueID = cursor.IssueID
	}
	rows, err := s.pool.Query(ctx, workerIssueColumns+`
		FROM newsletters n
		JOIN accounts a ON a.id = n.owner_account_id
		CROSS JOIN LATERAL (
			SELECT candidate.*
			FROM issues candidate
			WHERE candidate.newsletter_id = n.id
			  AND (
				NOT $2::boolean OR
				(candidate.created_at, candidate.id) < ($3::timestamptz, $4::uuid)
			  )
			ORDER BY candidate.created_at DESC, candidate.id DESC
			LIMIT $5
		) i
		WHERE n.owner_account_id = $1
		ORDER BY i.created_at DESC, i.id DESC
		LIMIT $5
	`, accountID, hasCursor, cursorCreatedAt, cursorIssueID, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list workspace Issues: %w", err)
	}
	defer rows.Close()
	issues := make([]domain.Issue, 0)
	for rows.Next() {
		issue, _, _, err := scanWorkerIssue(rows)
		if err != nil {
			return nil, nil, err
		}
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(issues) <= limit {
		return issues, nil, nil
	}
	issues = issues[:limit]
	last := issues[len(issues)-1]
	next := &WorkspaceIssueCursor{CreatedAt: last.CreatedAt, IssueID: last.ID}
	return issues, next, nil
}

func (s *Store) ListLibraryLessonsPage(
	ctx context.Context,
	accountID, search string,
	filter LibraryFilter,
	limit int,
	cursor *WorkspaceIssueCursor,
) ([]LibraryLesson, *WorkspaceIssueCursor, error) {
	if limit < 1 || limit > 100 {
		limit = 24
	}
	switch filter {
	case LibraryAll, LibraryUnread, LibraryInProgress, LibraryCompleted:
	default:
		filter = LibraryAll
	}
	hasCursor := cursor != nil
	cursorCreatedAt := time.Time{}
	cursorIssueID := uuid.Nil.String()
	if cursor != nil {
		cursorCreatedAt = cursor.CreatedAt
		cursorIssueID = cursor.IssueID
	}
	rows, err := s.pool.Query(ctx, `
		SELECT
			i.id::text,
			i.newsletter_id::text,
			COALESCE(i.dossier_title, ''),
			i.status,
			i.publication_state,
			i.created_at,
			n.id::text,
			n.name,
			n.topic,
			n.learner_level,
			n.lesson_minutes,
			lp.issue_id::text,
			COALESCE(lp.progress, 0),
			lp.completed_at,
			lp.updated_at,
			COALESCE(search.concepts, '{}'),
			COALESCE(search.source_titles, '{}')
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		LEFT JOIN lesson_search_documents search ON search.issue_id = i.id
		LEFT JOIN lesson_progress lp
		  ON lp.account_id = n.owner_account_id AND lp.issue_id = i.id
		WHERE n.owner_account_id = $1
		  AND i.status = 'generated'
		  AND (
			$2 = '' OR
			search.document @@ websearch_to_tsquery('english', $2) OR
			strpos(
			  lower(concat_ws(' ', i.dossier_title, n.name, n.topic, search.search_text)),
			  lower($2)
			) > 0
		  )
		  AND (
			$3 = 'all' OR
			($3 = 'completed' AND lp.completed_at IS NOT NULL) OR
			($3 = 'in-progress' AND lp.completed_at IS NULL AND COALESCE(lp.progress, 0) > 0) OR
			($3 = 'unread' AND lp.completed_at IS NULL AND COALESCE(lp.progress, 0) = 0)
		  )
		  AND (
			NOT $4::boolean OR
			(i.created_at, i.id) < ($5::timestamptz, $6::uuid)
		  )
		ORDER BY i.created_at DESC, i.id DESC
		LIMIT $7
	`, accountID, search, string(filter), hasCursor, cursorCreatedAt, cursorIssueID, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list Library lessons: %w", err)
	}
	defer rows.Close()
	lessons := make([]LibraryLesson, 0, limit+1)
	for rows.Next() {
		var (
			lesson      LibraryLesson
			progressID  *string
			progress    int
			completedAt *time.Time
			updatedAt   *time.Time
		)
		if err := rows.Scan(
			&lesson.ID,
			&lesson.NewsletterID,
			&lesson.Title,
			&lesson.Status,
			&lesson.PublicationState,
			&lesson.CreatedAt,
			&lesson.Newsletter.ID,
			&lesson.Newsletter.Name,
			&lesson.Newsletter.Topic,
			&lesson.Newsletter.LearnerLevel,
			&lesson.Newsletter.LessonMinutes,
			&progressID,
			&progress,
			&completedAt,
			&updatedAt,
			&lesson.Concepts,
			&lesson.SourceTitles,
		); err != nil {
			return nil, nil, err
		}
		if progressID != nil && updatedAt != nil {
			lesson.Progress = &LessonProgress{
				IssueID:     *progressID,
				Progress:    progress,
				CompletedAt: completedAt,
				UpdatedAt:   *updatedAt,
			}
		}
		lessons = append(lessons, lesson)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(lessons) <= limit {
		return lessons, nil, nil
	}
	lessons = lessons[:limit]
	last := lessons[len(lessons)-1]
	return lessons, &WorkspaceIssueCursor{
		CreatedAt: last.CreatedAt,
		IssueID:   last.ID,
	}, nil
}

func (s *Store) ListWorkspaceReviews(
	ctx context.Context,
	accountID string,
	limit int,
	now time.Time,
) ([]WorkspaceReview, error) {
	if limit < 1 || limit > 100 {
		limit = 8
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, issue_id::text, objective, prompt, answer_rubric,
		       corrective_explanation, stage, due_at, last_reviewed_at
		FROM review_items
		WHERE account_id = $1
		  AND due_at IS NOT NULL
		  AND due_at <= $3
		  AND retired_at IS NULL
		ORDER BY due_at, created_at
		LIMIT $2
	`, accountID, limit, now)
	if err != nil {
		return nil, fmt.Errorf("list workspace reviews: %w", err)
	}
	defer rows.Close()
	reviews := make([]WorkspaceReview, 0)
	for rows.Next() {
		var review WorkspaceReview
		if err := rows.Scan(
			&review.ID,
			&review.IssueID,
			&review.Objective,
			&review.Prompt,
			&review.AnswerRubric,
			&review.CorrectiveExplanation,
			&review.Stage,
			&review.DueAt,
			&review.LastReviewedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviews, nil
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

func (s *Store) RetryIssue(
	ctx context.Context,
	accountID, issueID string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE issues i SET
			status = 'queued',
			attempt_count = 0,
			available_at = $3::timestamptz,
			claim_token = NULL,
			claim_expires_at = NULL,
			error = NULL,
			failure_code = NULL,
			failure_category = NULL,
			failure_stage = NULL,
			failure_retryable = NULL,
			public_error = NULL,
			incident_id = NULL,
			started_at = NULL,
			completed_at = NULL
		FROM newsletters n
		JOIN accounts a ON a.id = n.owner_account_id
		WHERE i.newsletter_id = n.id
		  AND n.owner_account_id = $1
		  AND i.id = $2
		  AND i.status = 'failed'
		  AND a.status = 'active'
	`, accountID, issueID, now)
	if err != nil {
		return fmt.Errorf("retry Issue: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
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

const workerIssueColumns = `
	SELECT
		i.id::text, i.newsletter_id::text, i.trigger,
		i.scheduled_local_date::text, i.status, COALESCE(i.dossier_title, ''),
		COALESCE(i.generation_id::text, ''), COALESCE(i.artifact_key, ''),
		COALESCE(i.artifact_sha256, ''), COALESCE(i.artifact_bytes, 0),
		COALESCE(i.public_error, ''), COALESCE(i.failure_code, ''),
		COALESCE(i.failure_category, ''), COALESCE(i.failure_stage, ''),
		COALESCE(i.failure_retryable, false), COALESCE(i.incident_id::text, ''),
		'dossier-' || i.public_id::text,
		COALESCE(i.public_slug, ''), i.publication_state, i.created_at,
		i.started_at, i.completed_at,
		n.id::text, n.owner_account_id::text, n.name, n.topic, n.learner_level,
		n.learner_goal, n.lesson_minutes, n.source_mode,
		` + providedSourcesProjection + `, n.schedule_hour,
		n.schedule_minute, n.time_zone, n.active, n.next_run_at,
		n.email_enabled, n.ai_exploration_enabled, n.public_slug,
		n.site_visible, n.created_at, n.updated_at,
		a.id::text, COALESCE(a.primary_email, '')
`

const workerIssueSelect = workerIssueColumns + `
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
		&issue.ArtifactSHA256,
		&issue.ArtifactBytes,
		&issue.Error,
		&issue.FailureCode,
		&issue.FailureCategory,
		&issue.FailureStage,
		&issue.FailureRetryable,
		&issue.IncidentID,
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
		&newsletter.SourceMode,
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
	if len(rawSources) == 0 || string(rawSources) == "null" {
		newsletter.Sources = nil
	} else if err := json.Unmarshal(rawSources, &newsletter.Sources); err != nil {
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
