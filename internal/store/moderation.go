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

type PublicCorrection struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type ContentReport struct {
	ID               string     `json:"id"`
	Category         string     `json:"category"`
	Details          string     `json:"details"`
	Status           string     `json:"status"`
	ResolutionReason string     `json:"resolutionReason"`
	CreatedAt        time.Time  `json:"createdAt"`
	ResolvedAt       *time.Time `json:"resolvedAt"`
}

type ModerationAction struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

type IssueModeration struct {
	State       string             `json:"state"`
	Reason      string             `json:"reason"`
	Corrections []PublicCorrection `json:"corrections"`
	Reports     []ContentReport    `json:"reports"`
	Actions     []ModerationAction `json:"actions"`
}

type OperatorModerationResult struct {
	AlreadyHeld bool
}

// HoldPublicIssueByOperator removes an exact public Dossier from public reads
// without granting the operator access to the owner's private lesson data.
// The operator must be a distinct active Account so the existing moderation
// audit trail records who applied the hold.
func (s *Store) HoldPublicIssueByOperator(
	ctx context.Context,
	operatorAccountID, username, publicID, caseReference, reason string,
	now time.Time,
) (OperatorModerationResult, error) {
	if _, err := uuid.Parse(operatorAccountID); err != nil {
		return OperatorModerationResult{}, errors.New("operator account ID is invalid")
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" || len(username) > 63 {
		return OperatorModerationResult{}, errors.New("public username is invalid")
	}
	rawPublicID := strings.TrimPrefix(strings.TrimSpace(publicID), "dossier-")
	if _, err := uuid.Parse(rawPublicID); err != nil {
		return OperatorModerationResult{}, errors.New("public Dossier ID is invalid")
	}
	caseReference = strings.TrimSpace(caseReference)
	reason = strings.TrimSpace(reason)
	if len(caseReference) < 3 || len(caseReference) > 80 ||
		len(reason) < 10 || len(reason) > 800 {
		return OperatorModerationResult{}, errors.New("moderation case reference or reason is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	auditReason := "Rights-holder review pending · " + caseReference + ": " + reason
	if len(auditReason) > 1000 {
		return OperatorModerationResult{}, errors.New("moderation reason is invalid")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OperatorModerationResult{}, err
	}
	defer rollback(tx)
	var operatorActive bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM accounts WHERE id = $1 AND status = 'active'
		)
	`, operatorAccountID).Scan(&operatorActive); err != nil {
		return OperatorModerationResult{}, fmt.Errorf("verify moderation operator: %w", err)
	}
	if !operatorActive {
		return OperatorModerationResult{}, ErrForbidden
	}

	var issueID, ownerAccountID, currentState string
	err = tx.QueryRow(ctx, `
		SELECT issue.id::text, newsletter.owner_account_id::text,
		       issue.moderation_state
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN accounts account ON account.id = newsletter.owner_account_id
		JOIN personal_sites site
		  ON site.owner_account_id = newsletter.owner_account_id
		WHERE site.username = $1
		  AND site.visibility = 'public'
		  AND account.status = 'active'
		  AND newsletter.site_visible
		  AND issue.public_id = $2::uuid
		  AND issue.status = 'generated'
		  AND issue.publication_state = 'published'
		FOR UPDATE OF issue
	`, username, rawPublicID).Scan(&issueID, &ownerAccountID, &currentState)
	if errors.Is(err, pgx.ErrNoRows) {
		return OperatorModerationResult{}, ErrNotFound
	}
	if err != nil {
		return OperatorModerationResult{}, fmt.Errorf("find public Dossier for moderation: %w", err)
	}
	if ownerAccountID == operatorAccountID {
		return OperatorModerationResult{}, ErrForbidden
	}
	if currentState == "held" {
		if err := tx.Commit(ctx); err != nil {
			return OperatorModerationResult{}, err
		}
		return OperatorModerationResult{AlreadyHeld: true}, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE issues
		SET moderation_state = 'held', moderation_reason = $2
		WHERE id = $1
	`, issueID, auditReason); err != nil {
		return OperatorModerationResult{}, fmt.Errorf("hold public Dossier: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO public_moderation_actions (
		  id, issue_id, actor_account_id, action, reason, created_at
		) VALUES ($1, $2, $3, 'publication_held', $4, $5)
	`, uuid.NewString(), issueID, operatorAccountID, auditReason, now); err != nil {
		return OperatorModerationResult{}, fmt.Errorf("audit operator moderation hold: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OperatorModerationResult{}, fmt.Errorf("commit operator moderation hold: %w", err)
	}
	return OperatorModerationResult{}, nil
}

func (s *Store) CreatePublicContentReport(
	ctx context.Context,
	username, publicID, category, details, fingerprint string,
	now time.Time,
) (string, error) {
	category = strings.TrimSpace(category)
	details = strings.TrimSpace(details)
	if category != "inaccurate" && category != "citation" &&
		category != "harmful" && category != "other" {
		return "", errors.New("report category is invalid")
	}
	if len(details) > 2000 || len(fingerprint) < 16 || len(fingerprint) > 128 {
		return "", errors.New("report details are invalid")
	}
	rawID := strings.TrimPrefix(publicID, "dossier-")
	if _, err := uuid.Parse(rawID); err != nil {
		return "", ErrNotFound
	}
	id := uuid.NewString()
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO public_content_reports (
		  id, issue_id, category, details, reporter_fingerprint, created_at
		)
		SELECT $1, i.id, $4, $5, $6, $7
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		JOIN accounts a ON a.id = n.owner_account_id
		JOIN personal_sites ps ON ps.owner_account_id = a.id
		WHERE ps.username = $2 AND ps.visibility = 'public'
		  AND a.status = 'active' AND n.site_visible
		  AND i.public_id = $3 AND i.status = 'generated'
		  AND i.publication_state = 'published'
	`, id, strings.ToLower(username), rawID, category, details, fingerprint, now)
	if err != nil {
		return "", fmt.Errorf("create public content report: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", ErrNotFound
	}
	return id, nil
}

func (s *Store) ListPublicCorrections(
	ctx context.Context,
	issueID string,
) ([]PublicCorrection, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, body, created_at
		FROM public_corrections
		WHERE issue_id = $1 AND status = 'published'
		ORDER BY created_at, id
	`, issueID)
	if err != nil {
		return nil, fmt.Errorf("list public corrections: %w", err)
	}
	defer rows.Close()
	var corrections []PublicCorrection
	for rows.Next() {
		var correction PublicCorrection
		if err := rows.Scan(&correction.ID, &correction.Body, &correction.CreatedAt); err != nil {
			return nil, err
		}
		corrections = append(corrections, correction)
	}
	return corrections, rows.Err()
}

func (s *Store) GetIssueModeration(
	ctx context.Context,
	accountID, issueID string,
) (IssueModeration, error) {
	var result IssueModeration
	err := s.pool.QueryRow(ctx, `
		SELECT i.moderation_state, i.moderation_reason
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1 AND i.id = $2
	`, accountID, issueID).Scan(&result.State, &result.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return IssueModeration{}, ErrNotFound
	}
	if err != nil {
		return IssueModeration{}, fmt.Errorf("get issue moderation: %w", err)
	}
	corrections, err := s.ListPublicCorrections(ctx, issueID)
	if err != nil {
		return IssueModeration{}, err
	}
	result.Corrections = corrections
	reportRows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.category, r.details, r.status,
		       r.resolution_reason, r.created_at, r.resolved_at
		FROM public_content_reports r
		JOIN issues i ON i.id = r.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1 AND r.issue_id = $2
		ORDER BY (r.status = 'open') DESC, r.created_at DESC
	`, accountID, issueID)
	if err != nil {
		return IssueModeration{}, fmt.Errorf("list content reports: %w", err)
	}
	for reportRows.Next() {
		var report ContentReport
		if err := reportRows.Scan(
			&report.ID, &report.Category, &report.Details, &report.Status,
			&report.ResolutionReason, &report.CreatedAt, &report.ResolvedAt,
		); err != nil {
			reportRows.Close()
			return IssueModeration{}, err
		}
		result.Reports = append(result.Reports, report)
	}
	reportRows.Close()
	actionRows, err := s.pool.Query(ctx, `
		SELECT a.id::text, a.action, a.reason, a.created_at
		FROM public_moderation_actions a
		JOIN issues i ON i.id = a.issue_id
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1 AND a.issue_id = $2
		ORDER BY a.created_at DESC
		LIMIT 50
	`, accountID, issueID)
	if err != nil {
		return IssueModeration{}, fmt.Errorf("list moderation actions: %w", err)
	}
	defer actionRows.Close()
	for actionRows.Next() {
		var action ModerationAction
		if err := actionRows.Scan(&action.ID, &action.Action, &action.Reason, &action.CreatedAt); err != nil {
			return IssueModeration{}, err
		}
		result.Actions = append(result.Actions, action)
	}
	return result, actionRows.Err()
}

func (s *Store) AddPublicCorrection(
	ctx context.Context,
	accountID, issueID, body string,
	now time.Time,
) (PublicCorrection, error) {
	body = strings.TrimSpace(body)
	if len(body) < 1 || len(body) > 2000 {
		return PublicCorrection{}, errors.New("correction must be between 1 and 2000 characters")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicCorrection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	correction := PublicCorrection{ID: uuid.NewString(), Body: body, CreatedAt: now}
	tag, err := tx.Exec(ctx, `
		INSERT INTO public_corrections (
		  id, issue_id, owner_account_id, body, created_at
		)
		SELECT $3, i.id, $1, $4, $5
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1 AND i.id = $2
	`, accountID, issueID, correction.ID, correction.Body, now)
	if err != nil {
		return PublicCorrection{}, fmt.Errorf("add public correction: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return PublicCorrection{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO public_moderation_actions (
		  id, issue_id, actor_account_id, action, created_at
		) VALUES ($1, $2, $3, 'correction_published', $4)
	`, uuid.NewString(), issueID, accountID, now); err != nil {
		return PublicCorrection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicCorrection{}, err
	}
	return correction, nil
}

func (s *Store) RetractPublicCorrection(
	ctx context.Context,
	accountID, correctionID string,
	now time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var issueID string
	err = tx.QueryRow(ctx, `
		UPDATE public_corrections c
		SET status = 'retracted', retracted_at = $3
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE c.issue_id = i.id AND n.owner_account_id = $1
		  AND c.id = $2 AND c.status = 'published'
		RETURNING c.issue_id::text
	`, accountID, correctionID, now).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO public_moderation_actions (
		  id, issue_id, actor_account_id, action, created_at
		) VALUES ($1, $2, $3, 'correction_retracted', $4)
	`, uuid.NewString(), issueID, accountID, now)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ResolvePublicContentReport(
	ctx context.Context,
	accountID, reportID, status, reason string,
	now time.Time,
) error {
	reason = strings.TrimSpace(reason)
	if (status != "resolved" && status != "dismissed") || len(reason) < 1 || len(reason) > 1000 {
		return errors.New("report resolution is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var issueID string
	err = tx.QueryRow(ctx, `
		UPDATE public_content_reports r
		SET status = $3, resolution_reason = $4, resolved_at = $5
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE r.issue_id = i.id AND n.owner_account_id = $1
		  AND r.id = $2 AND r.status = 'open'
		RETURNING r.issue_id::text
	`, accountID, reportID, status, reason, now).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO public_moderation_actions (
		  id, issue_id, report_id, actor_account_id, action, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.NewString(), issueID, reportID, accountID, "report_"+status, reason, now)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetIssueModerationState(
	ctx context.Context,
	accountID, issueID, state, reason string,
	now time.Time,
) error {
	reason = strings.TrimSpace(reason)
	if state != "clear" && state != "held" {
		return errors.New("moderation state is invalid")
	}
	if state == "held" && reason == "" || len(reason) > 1000 {
		return errors.New("moderation reason is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE issues i
		SET moderation_state = $3, moderation_reason = $4
		FROM newsletters n
		WHERE i.newsletter_id = n.id AND n.owner_account_id = $1 AND i.id = $2
	`, accountID, issueID, state, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	action := "publication_cleared"
	if state == "held" {
		action = "publication_held"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO public_moderation_actions (
		  id, issue_id, actor_account_id, action, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.NewString(), issueID, accountID, action, reason, now)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
