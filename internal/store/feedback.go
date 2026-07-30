package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type LessonFeedback struct {
	IssueID          string    `json:"issueId"`
	Difficulty       *string   `json:"difficulty,omitempty"`
	Relevance        *string   `json:"relevance,omitempty"`
	RecallConfidence *string   `json:"recallConfidence,omitempty"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type LessonFeedbackInput struct {
	Difficulty       *string
	Relevance        *string
	RecallConfidence *string
}

func (s *Store) SaveLessonFeedback(
	ctx context.Context,
	accountID, issueID string,
	input LessonFeedbackInput,
	now time.Time,
) (LessonFeedback, error) {
	if err := validateLessonFeedback(input); err != nil {
		return LessonFeedback{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var result LessonFeedback
	err := s.pool.QueryRow(ctx, `
		INSERT INTO lesson_feedback (
			account_id, issue_id, difficulty, relevance, recall_confidence,
			created_at, updated_at
		)
		SELECT n.owner_account_id, i.id, $3, $4, $5, $6, $6
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1
		  AND i.id = $2
		  AND i.status = 'generated'
		ON CONFLICT (account_id, issue_id) DO UPDATE SET
			difficulty = COALESCE(EXCLUDED.difficulty, lesson_feedback.difficulty),
			relevance = COALESCE(EXCLUDED.relevance, lesson_feedback.relevance),
			recall_confidence = COALESCE(
				EXCLUDED.recall_confidence,
				lesson_feedback.recall_confidence
			),
			updated_at = EXCLUDED.updated_at
		RETURNING issue_id::text, difficulty, relevance, recall_confidence, updated_at
	`, accountID, issueID, input.Difficulty, input.Relevance,
		input.RecallConfidence, now).Scan(
		&result.IssueID,
		&result.Difficulty,
		&result.Relevance,
		&result.RecallConfidence,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LessonFeedback{}, ErrNotFound
	}
	if err != nil {
		return LessonFeedback{}, fmt.Errorf("save lesson feedback: %w", err)
	}
	return result, nil
}

func (s *Store) GetLessonFeedback(
	ctx context.Context,
	accountID, issueID string,
) (*LessonFeedback, error) {
	var result LessonFeedback
	err := s.pool.QueryRow(ctx, `
		SELECT lf.issue_id::text, lf.difficulty, lf.relevance,
		       lf.recall_confidence, lf.updated_at
		FROM lesson_feedback lf
		JOIN issues i ON i.id = lf.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE lf.account_id = $1
		  AND lf.issue_id = $2
		  AND n.owner_account_id = $1
	`, accountID, issueID).Scan(
		&result.IssueID,
		&result.Difficulty,
		&result.Relevance,
		&result.RecallConfidence,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get lesson feedback: %w", err)
	}
	return &result, nil
}

func validateLessonFeedback(input LessonFeedbackInput) error {
	if input.Difficulty == nil && input.Relevance == nil &&
		input.RecallConfidence == nil {
		return errors.New("at least one lesson feedback signal is required")
	}
	if input.Difficulty != nil &&
		*input.Difficulty != "too_basic" &&
		*input.Difficulty != "right" &&
		*input.Difficulty != "too_advanced" {
		return errors.New("lesson difficulty feedback is invalid")
	}
	if input.Relevance != nil &&
		*input.Relevance != "not_relevant" &&
		*input.Relevance != "somewhat_relevant" &&
		*input.Relevance != "very_relevant" {
		return errors.New("lesson relevance feedback is invalid")
	}
	if input.RecallConfidence != nil &&
		*input.RecallConfidence != "low" &&
		*input.RecallConfidence != "medium" &&
		*input.RecallConfidence != "high" {
		return errors.New("lesson recall confidence is invalid")
	}
	return nil
}
