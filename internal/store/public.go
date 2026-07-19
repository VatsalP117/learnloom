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

type PublicNewsletter struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Topic          string `json:"topic"`
	PublicSlug     string `json:"publicSlug"`
	GeneratedCount int    `json:"generatedCount"`
}

type PublicIssue struct {
	ID                   string    `json:"id"`
	PublicID             string    `json:"publicId"`
	PublicSlug           string    `json:"publicSlug"`
	Title                string    `json:"title"`
	ArtifactKey          string    `json:"-"`
	CompletedAt          time.Time `json:"completedAt"`
	NewsletterID         string    `json:"newsletterId"`
	NewsletterName       string    `json:"newsletterName"`
	NewsletterPublicSlug string    `json:"newsletterPublicSlug"`
}

func (s *Store) ListPublicNewsletters(
	ctx context.Context,
	username string,
) ([]PublicNewsletter, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT n.id::text, n.name, n.topic, n.public_slug,
		       count(i.id) FILTER (
		         WHERE i.status = 'generated' AND i.publication_state = 'published'
		       )::int
		FROM newsletters n
		JOIN accounts a ON a.id = n.owner_account_id
		JOIN personal_sites s ON s.owner_account_id = a.id
		LEFT JOIN issues i ON i.newsletter_id = n.id
		WHERE s.username = $1 AND s.visibility = 'public'
		  AND a.status = 'active' AND n.site_visible
		GROUP BY n.id
		ORDER BY n.created_at DESC
	`, strings.ToLower(username))
	if err != nil {
		return nil, fmt.Errorf("list public Newsletters: %w", err)
	}
	defer rows.Close()
	var result []PublicNewsletter
	for rows.Next() {
		var newsletter PublicNewsletter
		if err := rows.Scan(
			&newsletter.ID,
			&newsletter.Name,
			&newsletter.Topic,
			&newsletter.PublicSlug,
			&newsletter.GeneratedCount,
		); err != nil {
			return nil, err
		}
		result = append(result, newsletter)
	}
	return result, rows.Err()
}

func (s *Store) ListPublicIssues(
	ctx context.Context,
	username, newsletterSlug string,
	limit int,
) ([]PublicIssue, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT i.id::text, 'dossier-' || i.public_id::text, i.public_slug,
		       i.dossier_title, i.artifact_key, i.completed_at,
		       n.id::text, n.name, n.public_slug
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		JOIN accounts a ON a.id = n.owner_account_id
		JOIN personal_sites s ON s.owner_account_id = a.id
		WHERE s.username = $1 AND s.visibility = 'public'
		  AND a.status = 'active' AND n.site_visible
		  AND i.status = 'generated' AND i.publication_state = 'published'
		  AND ($2 = '' OR n.public_slug = $2)
		ORDER BY i.completed_at DESC, i.id
		LIMIT $3
	`, strings.ToLower(username), newsletterSlug, limit)
	if err != nil {
		return nil, fmt.Errorf("list public Issues: %w", err)
	}
	defer rows.Close()
	var result []PublicIssue
	for rows.Next() {
		var issue PublicIssue
		if err := rows.Scan(
			&issue.ID,
			&issue.PublicID,
			&issue.PublicSlug,
			&issue.Title,
			&issue.ArtifactKey,
			&issue.CompletedAt,
			&issue.NewsletterID,
			&issue.NewsletterName,
			&issue.NewsletterPublicSlug,
		); err != nil {
			return nil, err
		}
		result = append(result, issue)
	}
	return result, rows.Err()
}

func (s *Store) GetPublicIssue(
	ctx context.Context,
	username, publicID string,
) (PublicIssue, error) {
	rawID := strings.TrimPrefix(publicID, "dossier-")
	if _, err := uuid.Parse(rawID); err != nil {
		return PublicIssue{}, ErrNotFound
	}
	row := s.pool.QueryRow(ctx, `
		SELECT i.id::text, 'dossier-' || i.public_id::text, i.public_slug,
		       i.dossier_title, i.artifact_key, i.completed_at,
		       n.id::text, n.name, n.public_slug
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		JOIN accounts a ON a.id = n.owner_account_id
		JOIN personal_sites s ON s.owner_account_id = a.id
		WHERE s.username = $1 AND s.visibility = 'public'
		  AND a.status = 'active' AND n.site_visible
		  AND i.public_id = $2 AND i.status = 'generated'
		  AND i.publication_state = 'published'
	`, strings.ToLower(username), rawID)
	var issue PublicIssue
	err := row.Scan(
		&issue.ID,
		&issue.PublicID,
		&issue.PublicSlug,
		&issue.Title,
		&issue.ArtifactKey,
		&issue.CompletedAt,
		&issue.NewsletterID,
		&issue.NewsletterName,
		&issue.NewsletterPublicSlug,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicIssue{}, ErrNotFound
	}
	if err != nil {
		return PublicIssue{}, fmt.Errorf("get public Issue: %w", err)
	}
	return issue, nil
}
