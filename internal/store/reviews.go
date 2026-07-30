package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReviewAssessment string

const (
	ReviewNeedsWork ReviewAssessment = "needs_work"
	ReviewPartial   ReviewAssessment = "partial"
	ReviewSolid     ReviewAssessment = "solid"
)

func (s *Store) AssessReview(
	ctx context.Context,
	accountID, reviewID, idempotencyKey string,
	assessment ReviewAssessment,
	now time.Time,
) (WorkspaceReview, error) {
	if assessment != ReviewNeedsWork && assessment != ReviewPartial &&
		assessment != ReviewSolid {
		return WorkspaceReview{}, errors.New("Review assessment is invalid")
	}
	if _, err := uuid.Parse(idempotencyKey); err != nil {
		return WorkspaceReview{}, errors.New("Review idempotency key is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkspaceReview{}, err
	}
	defer rollback(tx)

	var existingReviewID string
	err = tx.QueryRow(ctx, `
		SELECT review_item_id::text
		FROM review_attempts
		WHERE account_id = $1 AND idempotency_key = $2
	`, accountID, idempotencyKey).Scan(&existingReviewID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceReview{}, fmt.Errorf("inspect Review Attempt: %w", err)
	}
	if err == nil {
		if existingReviewID != reviewID {
			return WorkspaceReview{}, ErrConflict
		}
		review, err := getReviewItem(ctx, tx, accountID, reviewID, false)
		if err != nil {
			return WorkspaceReview{}, err
		}
		return review, tx.Commit(ctx)
	}

	review, err := getReviewItem(ctx, tx, accountID, reviewID, true)
	if err != nil {
		return WorkspaceReview{}, err
	}
	nextStage, nextDue := scheduleReview(review.Stage, assessment, now)
	if _, err := tx.Exec(ctx, `
		INSERT INTO review_attempts (
			review_item_id, account_id, idempotency_key, assessment,
			previous_stage, next_stage, previous_due_at, next_due_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, review.ID, accountID, idempotencyKey, assessment, review.Stage,
		nextStage, review.DueAt, nextDue, now); err != nil {
		return WorkspaceReview{}, fmt.Errorf("record Review Attempt: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE review_items
		SET stage = $3, due_at = $4, last_reviewed_at = $5, updated_at = $5
		WHERE id = $1 AND account_id = $2
	`, review.ID, accountID, nextStage, nextDue, now); err != nil {
		return WorkspaceReview{}, fmt.Errorf("reschedule Review Item: %w", err)
	}
	review.Stage = nextStage
	review.DueAt = nextDue
	review.LastReviewedAt = &now
	if err := tx.Commit(ctx); err != nil {
		return WorkspaceReview{}, fmt.Errorf("commit Review Attempt: %w", err)
	}
	return review, nil
}

func getReviewItem(
	ctx context.Context,
	tx pgx.Tx,
	accountID, reviewID string,
	forUpdate bool,
) (WorkspaceReview, error) {
	query := `
		SELECT id::text, issue_id::text, objective, prompt, answer_rubric,
		       corrective_explanation, stage, due_at, last_reviewed_at
		FROM review_items
		WHERE id = $1 AND account_id = $2
		  AND due_at IS NOT NULL AND retired_at IS NULL
	`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var review WorkspaceReview
	err := tx.QueryRow(ctx, query, reviewID, accountID).Scan(
		&review.ID,
		&review.IssueID,
		&review.Objective,
		&review.Prompt,
		&review.AnswerRubric,
		&review.CorrectiveExplanation,
		&review.Stage,
		&review.DueAt,
		&review.LastReviewedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceReview{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceReview{}, fmt.Errorf("get Review Item: %w", err)
	}
	return review, nil
}

func scheduleReview(
	stage int,
	assessment ReviewAssessment,
	now time.Time,
) (int, time.Time) {
	intervals := []time.Duration{
		24 * time.Hour,
		3 * 24 * time.Hour,
		7 * 24 * time.Hour,
		21 * 24 * time.Hour,
		45 * 24 * time.Hour,
	}
	nextStage := 0
	switch assessment {
	case ReviewNeedsWork:
		nextStage = 0
	case ReviewPartial:
		nextStage = min(stage+1, len(intervals)-1)
	case ReviewSolid:
		nextStage = min(stage+2, len(intervals)-1)
	}
	return nextStage, now.Add(intervals[nextStage])
}
