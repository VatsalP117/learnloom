package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type LessonProgress struct {
	IssueID     string     `json:"issueId"`
	Progress    int        `json:"progress"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type LessonNavigationLink struct {
	IssueID   string    `json:"issueId"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

type LessonNavigation struct {
	Previous     *LessonNavigationLink `json:"previous,omitempty"`
	Next         *LessonNavigationLink `json:"next,omitempty"`
	NextReviewAt *time.Time            `json:"nextReviewAt,omitempty"`
}

func (s *Store) GetLessonNavigation(
	ctx context.Context,
	accountID, issueID string,
) (LessonNavigation, error) {
	var newsletterID string
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT issue.newsletter_id::text, issue.created_at
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE issue.id = $2
		  AND newsletter.owner_account_id = $1
		  AND issue.status = 'generated'
	`, accountID, issueID).Scan(&newsletterID, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return LessonNavigation{}, ErrNotFound
	}
	if err != nil {
		return LessonNavigation{}, fmt.Errorf("get Lesson navigation anchor: %w", err)
	}
	result := LessonNavigation{}
	result.Previous, err = s.adjacentLesson(
		ctx, accountID, newsletterID, issueID, createdAt, false,
	)
	if err != nil {
		return LessonNavigation{}, err
	}
	result.Next, err = s.adjacentLesson(
		ctx, accountID, newsletterID, issueID, createdAt, true,
	)
	if err != nil {
		return LessonNavigation{}, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT min(review.due_at)
		FROM review_items review
		JOIN issues issue ON issue.id = review.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE review.account_id = $1
		  AND review.issue_id = $2
		  AND newsletter.owner_account_id = $1
		  AND review.retired_at IS NULL
	`, accountID, issueID).Scan(&result.NextReviewAt)
	if err != nil {
		return LessonNavigation{}, fmt.Errorf("get Lesson next Review: %w", err)
	}
	return result, nil
}

func (s *Store) adjacentLesson(
	ctx context.Context,
	accountID, newsletterID, issueID string,
	createdAt time.Time,
	newer bool,
) (*LessonNavigationLink, error) {
	comparison := "<"
	ordering := "DESC"
	if newer {
		comparison = ">"
		ordering = "ASC"
	}
	query := fmt.Sprintf(`
		SELECT issue.id::text, COALESCE(issue.dossier_title, ''), issue.created_at
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE newsletter.owner_account_id = $1
		  AND issue.newsletter_id = $2
		  AND issue.id <> $3
		  AND issue.status = 'generated'
		  AND (issue.created_at, issue.id) %s ($4, $3::uuid)
		ORDER BY issue.created_at %s, issue.id %s
		LIMIT 1
	`, comparison, ordering, ordering)
	var result LessonNavigationLink
	err := s.pool.QueryRow(
		ctx, query, accountID, newsletterID, issueID, createdAt,
	).Scan(&result.IssueID, &result.Title, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get adjacent Lesson: %w", err)
	}
	return &result, nil
}

func (s *Store) GetLessonProgress(
	ctx context.Context,
	accountID, issueID string,
) (*LessonProgress, error) {
	var result LessonProgress
	err := s.pool.QueryRow(ctx, `
		SELECT progress.issue_id::text, progress.progress,
		       progress.completed_at, progress.updated_at
		FROM lesson_progress progress
		JOIN issues issue ON issue.id = progress.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE progress.account_id = $1
		  AND progress.issue_id = $2
		  AND newsletter.owner_account_id = $1
	`, accountID, issueID).Scan(
		&result.IssueID,
		&result.Progress,
		&result.CompletedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get Lesson Progress: %w", err)
	}
	return &result, nil
}

func (s *Store) SaveLessonProgress(
	ctx context.Context,
	accountID, issueID string,
	progress int,
	now time.Time,
) (LessonProgress, error) {
	if progress < 1 || progress > 99 {
		return LessonProgress{}, ErrConflict
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var result LessonProgress
	err := s.pool.QueryRow(ctx, `
		INSERT INTO lesson_progress (
			account_id, issue_id, progress, updated_at
		)
		SELECT n.owner_account_id, i.id, $3, $4
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1
		  AND i.id = $2
		  AND i.status = 'generated'
		ON CONFLICT (account_id, issue_id) DO UPDATE SET
			progress = GREATEST(lesson_progress.progress, EXCLUDED.progress),
			updated_at = EXCLUDED.updated_at
		RETURNING issue_id::text, progress, completed_at, updated_at
	`, accountID, issueID, progress, now).Scan(
		&result.IssueID,
		&result.Progress,
		&result.CompletedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LessonProgress{}, ErrConflict
	}
	if err != nil {
		return LessonProgress{}, fmt.Errorf("save lesson progress: %w", err)
	}
	return result, nil
}

func (s *Store) CompleteLesson(
	ctx context.Context,
	accountID, issueID string,
	now time.Time,
) (LessonProgress, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LessonProgress{}, err
	}
	defer rollback(tx)
	var result LessonProgress
	err = tx.QueryRow(ctx, `
		INSERT INTO lesson_progress (
			account_id, issue_id, progress, completed_at, updated_at
		)
		SELECT n.owner_account_id, i.id, 100, $3, $3
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1
		  AND i.id = $2
		  AND i.status = 'generated'
		ON CONFLICT (account_id, issue_id) DO UPDATE SET
			progress = 100,
			completed_at = COALESCE(lesson_progress.completed_at, EXCLUDED.completed_at),
			updated_at = EXCLUDED.updated_at
		RETURNING issue_id::text, progress, completed_at, updated_at
	`, accountID, issueID, now).Scan(
		&result.IssueID,
		&result.Progress,
		&result.CompletedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LessonProgress{}, ErrConflict
	}
	if err != nil {
		return LessonProgress{}, fmt.Errorf("complete lesson: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE review_items
		SET due_at = COALESCE(due_at, $3::timestamptz + interval '1 day'),
		    updated_at = $3
		WHERE account_id = $1
		  AND issue_id = $2
		  AND retired_at IS NULL
	`, accountID, issueID, now); err != nil {
		return LessonProgress{}, fmt.Errorf("activate lesson reviews: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE learner_concept_state state
		SET completed_count = LEAST(
		      state.exposure_count,
		      state.completed_count + 1
		    ),
		    last_completed_at = $3,
		    updated_at = $3
		FROM issue_concepts concept
		WHERE concept.issue_id = $2
		  AND concept.account_id = $1
		  AND state.account_id = concept.account_id
		  AND state.newsletter_id = concept.newsletter_id
		  AND state.concept_key = concept.concept_key
	`, accountID, issueID, now); err != nil {
		return LessonProgress{}, fmt.Errorf("complete Learner Concepts: %w", err)
	}
	if err := insertProductEvent(
		ctx, tx, accountID, ProductEventLessonCompleted,
		"lesson", issueID, now,
	); err != nil {
		return LessonProgress{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LessonProgress{}, fmt.Errorf("commit lesson completion: %w", err)
	}
	return result, nil
}

func (s *Store) ListLessonProgress(
	ctx context.Context,
	accountID string,
) ([]LessonProgress, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT issue_id::text, progress, completed_at, updated_at
		FROM lesson_progress
		WHERE account_id = $1
		ORDER BY updated_at DESC
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list lesson progress: %w", err)
	}
	defer rows.Close()
	items := make([]LessonProgress, 0)
	for rows.Next() {
		var item LessonProgress
		if err := rows.Scan(
			&item.IssueID,
			&item.Progress,
			&item.CompletedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListRecentLessonProgress(
	ctx context.Context,
	accountID string,
	limit int,
) ([]LessonProgress, error) {
	if limit < 1 || limit > 100 {
		limit = 24
	}
	rows, err := s.pool.Query(ctx, `
		SELECT lp.issue_id::text, lp.progress, lp.completed_at, lp.updated_at
		FROM lesson_progress lp
		JOIN issues i ON i.id = lp.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE lp.account_id = $1
		  AND n.owner_account_id = $1
		ORDER BY i.created_at DESC, i.id DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent lesson progress: %w", err)
	}
	defer rows.Close()
	items := make([]LessonProgress, 0, limit)
	for rows.Next() {
		var item LessonProgress
		if err := rows.Scan(
			&item.IssueID,
			&item.Progress,
			&item.CompletedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
