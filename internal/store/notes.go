package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LessonNote struct {
	ID         string    `json:"id"`
	IssueID    string    `json:"issueId"`
	Kind       string    `json:"kind"`
	AnchorType string    `json:"anchorType"`
	AnchorID   string    `json:"anchorId"`
	Body       string    `json:"body"`
	QuotedText string    `json:"quotedText,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type LessonNoteInput struct {
	Kind       string
	AnchorType string
	AnchorID   string
	Body       string
	QuotedText string
}

func (s *Store) ListLessonNotes(
	ctx context.Context,
	accountID, issueID string,
) ([]LessonNote, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT note.id::text, note.issue_id::text, note.kind,
		       note.anchor_type, note.anchor_id, note.body, note.quoted_text,
		       note.created_at, note.updated_at
		FROM lesson_notes note
		JOIN issues issue ON issue.id = note.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE note.account_id = $1
		  AND note.issue_id = $2
		  AND newsletter.owner_account_id = $1
		ORDER BY note.updated_at DESC, note.id
	`, accountID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list Lesson Notes: %w", err)
	}
	defer rows.Close()
	notes := make([]LessonNote, 0)
	for rows.Next() {
		var note LessonNote
		if err := rows.Scan(
			&note.ID,
			&note.IssueID,
			&note.Kind,
			&note.AnchorType,
			&note.AnchorID,
			&note.Body,
			&note.QuotedText,
			&note.CreatedAt,
			&note.UpdatedAt,
		); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (s *Store) CreateLessonNote(
	ctx context.Context,
	accountID, issueID string,
	input LessonNoteInput,
	now time.Time,
) (LessonNote, error) {
	normalized, err := normalizeLessonNote(input)
	if err != nil {
		return LessonNote{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var note LessonNote
	err = s.pool.QueryRow(ctx, `
		INSERT INTO lesson_notes (
			id, account_id, issue_id, kind, anchor_type, anchor_id,
			body, quoted_text, created_at, updated_at
		)
		SELECT $3, newsletter.owner_account_id, issue.id, $4, $5, $6,
		       $7, $8, $9, $9
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE newsletter.owner_account_id = $1
		  AND issue.id = $2
		  AND issue.status = 'generated'
		RETURNING id::text, issue_id::text, kind, anchor_type, anchor_id,
		          body, quoted_text, created_at, updated_at
	`, accountID, issueID, uuid.New(), normalized.Kind, normalized.AnchorType,
		normalized.AnchorID, normalized.Body, normalized.QuotedText, now).Scan(
		&note.ID,
		&note.IssueID,
		&note.Kind,
		&note.AnchorType,
		&note.AnchorID,
		&note.Body,
		&note.QuotedText,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LessonNote{}, ErrNotFound
	}
	if err != nil {
		return LessonNote{}, fmt.Errorf("create Lesson Note: %w", err)
	}
	return note, nil
}

func (s *Store) DeleteLessonNote(
	ctx context.Context,
	accountID, noteID string,
) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM lesson_notes note
		USING issues issue, newsletters newsletter
		WHERE note.id = $2
		  AND note.account_id = $1
		  AND issue.id = note.issue_id
		  AND newsletter.id = issue.newsletter_id
		  AND newsletter.owner_account_id = $1
	`, accountID, noteID)
	if err != nil {
		return fmt.Errorf("delete Lesson Note: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func normalizeLessonNote(input LessonNoteInput) (LessonNoteInput, error) {
	input.Kind = strings.TrimSpace(input.Kind)
	input.AnchorType = strings.TrimSpace(input.AnchorType)
	input.AnchorID = strings.TrimSpace(input.AnchorID)
	input.Body = strings.TrimSpace(input.Body)
	input.QuotedText = strings.TrimSpace(input.QuotedText)
	if input.Kind != "note" && input.Kind != "question" &&
		input.Kind != "highlight" {
		return LessonNoteInput{}, errors.New("Lesson Note kind is invalid")
	}
	if input.AnchorType != "lesson" && input.AnchorType != "claim" &&
		input.AnchorType != "source" && input.AnchorType != "section" {
		return LessonNoteInput{}, errors.New("Lesson Note anchor is invalid")
	}
	if len([]rune(input.AnchorID)) > 120 {
		return LessonNoteInput{}, errors.New("Lesson Note anchor is too long")
	}
	if len([]rune(input.Body)) < 1 || len([]rune(input.Body)) > 4000 {
		return LessonNoteInput{}, errors.New("Lesson Note body must be from 1 to 4000 characters")
	}
	if len([]rune(input.QuotedText)) > 1200 {
		return LessonNoteInput{}, errors.New("Lesson Note quote is too long")
	}
	return input, nil
}
