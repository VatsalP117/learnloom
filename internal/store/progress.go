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

func (s *Store) CompleteLesson(
	ctx context.Context,
	accountID, issueID string,
	now time.Time,
) (LessonProgress, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var result LessonProgress
	err := s.pool.QueryRow(ctx, `
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
