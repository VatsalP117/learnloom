package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type LessonRetrievalResponse struct {
	IssueID    string     `json:"issueId"`
	PromptKey  string     `json:"promptKey"`
	Response   string     `json:"response,omitempty"`
	Skipped    bool       `json:"skipped"`
	RevealedAt *time.Time `json:"revealedAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type LessonRetrievalInput struct {
	PromptKey string
	Response  string
	Skipped   bool
}

func validateLessonRetrievalInput(input LessonRetrievalInput, allowSkip bool) (LessonRetrievalInput, error) {
	input.PromptKey = strings.TrimSpace(input.PromptKey)
	input.Response = strings.TrimSpace(input.Response)
	if input.PromptKey == "" || utf8.RuneCountInString(input.PromptKey) > 80 {
		return LessonRetrievalInput{}, errors.New("retrieval prompt key is invalid")
	}
	responseLength := utf8.RuneCountInString(input.Response)
	if input.Skipped {
		if !allowSkip || input.Response != "" {
			return LessonRetrievalInput{}, errors.New("skipped retrieval is invalid")
		}
	} else if responseLength < 3 || responseLength > 2000 {
		return LessonRetrievalInput{}, errors.New("retrieval response is invalid")
	}
	return input, nil
}

func (s *Store) SaveLessonRetrievalDraft(
	ctx context.Context,
	accountID, issueID string,
	input LessonRetrievalInput,
	now time.Time,
) (LessonRetrievalResponse, error) {
	input.Skipped = false
	input, err := validateLessonRetrievalInput(input, false)
	if err != nil {
		return LessonRetrievalResponse{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var result LessonRetrievalResponse
	err = s.pool.QueryRow(ctx, `
		INSERT INTO lesson_retrieval_responses (
		  account_id, issue_id, prompt_key, response_text, skipped,
		  revealed_at, created_at, updated_at
		)
		SELECT newsletter.owner_account_id, issue.id, review.prompt_key,
		       $4, false, NULL, $5, $5
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN review_items review ON review.issue_id = issue.id
		WHERE newsletter.owner_account_id = $1
		  AND issue.id = $2
		  AND issue.status = 'generated'
		  AND review.prompt_key = $3
		ON CONFLICT (account_id, issue_id, prompt_key) DO UPDATE SET
		  response_text = EXCLUDED.response_text,
		  updated_at = EXCLUDED.updated_at
		WHERE lesson_retrieval_responses.revealed_at IS NULL
		RETURNING issue_id::text, prompt_key, response_text, skipped,
		          revealed_at, updated_at
	`, accountID, issueID, input.PromptKey, input.Response, now).Scan(
		&result.IssueID,
		&result.PromptKey,
		&result.Response,
		&result.Skipped,
		&result.RevealedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LessonRetrievalResponse{}, ErrConflict
	}
	if err != nil {
		return LessonRetrievalResponse{}, fmt.Errorf("save Lesson Retrieval Draft: %w", err)
	}
	return result, nil
}

func (s *Store) RevealLessonRetrieval(
	ctx context.Context,
	accountID, issueID string,
	input LessonRetrievalInput,
	now time.Time,
) (LessonRetrievalResponse, error) {
	var err error
	input, err = validateLessonRetrievalInput(input, true)
	if err != nil {
		return LessonRetrievalResponse{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LessonRetrievalResponse{}, err
	}
	defer rollback(tx)

	var result LessonRetrievalResponse
	err = tx.QueryRow(ctx, `
		INSERT INTO lesson_retrieval_responses (
		  account_id, issue_id, prompt_key, response_text, skipped,
		  revealed_at, created_at, updated_at
		)
		SELECT newsletter.owner_account_id, issue.id, review.prompt_key,
		       $4, $5, $6, $6, $6
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN review_items review ON review.issue_id = issue.id
		WHERE newsletter.owner_account_id = $1
		  AND issue.id = $2
		  AND issue.status = 'generated'
		  AND review.prompt_key = $3
		ON CONFLICT (account_id, issue_id, prompt_key) DO UPDATE SET
		  response_text = EXCLUDED.response_text,
		  skipped = EXCLUDED.skipped,
		  revealed_at = EXCLUDED.revealed_at,
		  updated_at = EXCLUDED.updated_at
		WHERE lesson_retrieval_responses.revealed_at IS NULL
		RETURNING issue_id::text, prompt_key, response_text, skipped,
		          revealed_at, updated_at
	`, accountID, issueID, input.PromptKey, input.Response, input.Skipped, now).Scan(
		&result.IssueID,
		&result.PromptKey,
		&result.Response,
		&result.Skipped,
		&result.RevealedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT response.issue_id::text, response.prompt_key,
			       response.response_text, response.skipped,
			       response.revealed_at, response.updated_at
			FROM lesson_retrieval_responses response
			JOIN issues issue ON issue.id = response.issue_id
			JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
			WHERE response.account_id = $1
			  AND response.issue_id = $2
			  AND response.prompt_key = $3
			  AND newsletter.owner_account_id = $1
		`, accountID, issueID, input.PromptKey).Scan(
			&result.IssueID,
			&result.PromptKey,
			&result.Response,
			&result.Skipped,
			&result.RevealedAt,
			&result.UpdatedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return LessonRetrievalResponse{}, ErrNotFound
		}
		if err != nil {
			return LessonRetrievalResponse{}, fmt.Errorf("read Lesson Retrieval: %w", err)
		}
		if result.RevealedAt == nil || result.Response != input.Response || result.Skipped != input.Skipped {
			return LessonRetrievalResponse{}, ErrConflict
		}
	} else if err != nil {
		return LessonRetrievalResponse{}, fmt.Errorf("reveal Lesson Retrieval: %w", err)
	}

	if !result.Skipped {
		for _, event := range []struct {
			name      ProductEventName
			subjectID string
		}{
			{ProductEventFirstRetrieval, "first"},
			{ProductEventActivationCompleted, "first-cycle"},
		} {
			if err := insertProductEvent(
				ctx, tx, accountID, event.name, "review", event.subjectID, now,
			); err != nil {
				return LessonRetrievalResponse{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return LessonRetrievalResponse{}, fmt.Errorf("commit Lesson Retrieval: %w", err)
	}
	return result, nil
}

func (s *Store) ListLessonRetrievalResponses(
	ctx context.Context,
	accountID, issueID string,
) ([]LessonRetrievalResponse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT response.issue_id::text, response.prompt_key,
		       response.response_text, response.skipped,
		       response.revealed_at, response.updated_at
		FROM lesson_retrieval_responses response
		JOIN issues issue ON issue.id = response.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE response.account_id = $1
		  AND response.issue_id = $2
		  AND newsletter.owner_account_id = $1
		ORDER BY response.created_at, response.prompt_key
	`, accountID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list Lesson Retrieval Responses: %w", err)
	}
	defer rows.Close()
	responses := make([]LessonRetrievalResponse, 0)
	for rows.Next() {
		var response LessonRetrievalResponse
		if err := rows.Scan(
			&response.IssueID,
			&response.PromptKey,
			&response.Response,
			&response.Skipped,
			&response.RevealedAt,
			&response.UpdatedAt,
		); err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return responses, rows.Err()
}
