package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,29}$`)

var reservedUsernames = map[string]struct{}{
	"admin": {}, "api": {}, "app": {}, "assets": {}, "auth": {},
	"blog": {}, "clerk": {}, "dashboard": {}, "docs": {}, "help": {},
	"learnloom": {}, "mail": {}, "root": {}, "status": {}, "support": {},
	"www": {},
}

func (s *Store) EnsureAccount(
	ctx context.Context,
	clerkUserID string,
) (domain.Account, error) {
	clerkUserID = strings.TrimSpace(clerkUserID)
	if clerkUserID == "" || len(clerkUserID) > 200 {
		return domain.Account{}, errors.New("Clerk user ID is invalid")
	}
	account, err := s.accountByClerkUserID(ctx, clerkUserID)
	if err == nil {
		if account.Status != domain.AccountActive {
			return domain.Account{}, ErrForbidden
		}
		return account, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, fmt.Errorf("load Account: %w", err)
	}

	now := time.Now().UTC()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (
			id, clerk_user_id, status, created_at, updated_at
		)
		VALUES ($1, $2, 'active', $3, $3)
		ON CONFLICT (clerk_user_id) DO NOTHING
		RETURNING id::text, clerk_user_id, COALESCE(primary_email, ''),
		          status, created_at, updated_at, deleted_at
	`, uuid.New(), clerkUserID, now)
	account, err = scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		account, err = s.accountByClerkUserID(ctx, clerkUserID)
	}
	if err != nil {
		return domain.Account{}, fmt.Errorf("ensure Account: %w", err)
	}
	if account.Status != domain.AccountActive {
		return domain.Account{}, ErrForbidden
	}
	return account, nil
}

func (s *Store) accountByClerkUserID(
	ctx context.Context,
	clerkUserID string,
) (domain.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx, `
		SELECT id::text, clerk_user_id, COALESCE(primary_email, ''),
		       status, created_at, updated_at, deleted_at
		FROM accounts
		WHERE clerk_user_id = $1
	`, clerkUserID))
}

func (s *Store) SyncAccountIdentity(
	ctx context.Context,
	clerkUserID, primaryEmail string,
	status domain.AccountStatus,
	identityEventAt int64,
) (domain.Account, error) {
	if status != domain.AccountActive && status != domain.AccountSuspended &&
		status != domain.AccountDeleted {
		return domain.Account{}, errors.New("Account status is invalid")
	}
	if identityEventAt < 1 {
		return domain.Account{}, errors.New("Clerk event timestamp is invalid")
	}
	now := time.Now().UTC()
	var deletedAt *time.Time
	if status == domain.AccountDeleted {
		deletedAt = &now
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (
			id, clerk_user_id, primary_email, status,
			identity_event_at, created_at, updated_at, deleted_at
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $6, $7)
		ON CONFLICT (clerk_user_id) DO UPDATE SET
			primary_email = COALESCE(NULLIF(EXCLUDED.primary_email, ''), accounts.primary_email),
			status = CASE
				WHEN accounts.status = 'deleted' THEN 'deleted'
				ELSE EXCLUDED.status
			END,
			identity_event_at = EXCLUDED.identity_event_at,
			updated_at = EXCLUDED.updated_at,
			deleted_at = CASE
				WHEN accounts.status = 'deleted' THEN accounts.deleted_at
				ELSE EXCLUDED.deleted_at
			END
		WHERE EXCLUDED.identity_event_at >= accounts.identity_event_at
		RETURNING id::text, clerk_user_id, COALESCE(primary_email, ''),
		          status, created_at, updated_at, deleted_at
	`, uuid.New(), clerkUserID, strings.TrimSpace(primaryEmail), status,
		identityEventAt, now, deletedAt)
	account, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		account, err = scanAccount(s.pool.QueryRow(ctx, `
			SELECT id::text, clerk_user_id, COALESCE(primary_email, ''),
			       status, created_at, updated_at, deleted_at
			FROM accounts WHERE clerk_user_id = $1
		`, clerkUserID))
	}
	if err != nil {
		return domain.Account{}, fmt.Errorf("sync Account identity: %w", err)
	}
	if account.Status != domain.AccountActive {
		if err := s.stopAccountWork(
			ctx,
			account.ID,
			account.Status == domain.AccountDeleted,
		); err != nil {
			return domain.Account{}, err
		}
	}
	return account, nil
}

func (s *Store) stopAccountWork(
	ctx context.Context,
	accountID string,
	deleteArtifacts bool,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `
		UPDATE newsletters
		SET active = false, updated_at = now()
		WHERE owner_account_id = $1
	`, accountID); err != nil {
		return fmt.Errorf("pause Account Newsletters: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE issues SET
			status = 'cancelled',
			claim_token = NULL,
			claim_expires_at = NULL,
			error = 'Account is unavailable'
		WHERE newsletter_id IN (
			SELECT id FROM newsletters WHERE owner_account_id = $1
		)
		AND status IN ('queued', 'generating')
	`, accountID); err != nil {
		return fmt.Errorf("cancel Account Issues: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE delivery_receipts SET
			status = 'cancelled',
			claim_token = NULL,
			claim_expires_at = NULL,
			error = 'Account is unavailable',
			updated_at = now()
		WHERE issue_id IN (
			SELECT i.id FROM issues i
			JOIN newsletters n ON n.id = i.newsletter_id
			WHERE n.owner_account_id = $1
		)
		AND status IN ('pending', 'failed', 'delivering')
	`, accountID); err != nil {
		return fmt.Errorf("cancel Account Delivery Receipts: %w", err)
	}
	if deleteArtifacts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_queue (account_id, available_at)
			VALUES ($1, now())
			ON CONFLICT (account_id) DO UPDATE SET
				available_at = LEAST(account_deletion_queue.available_at, EXCLUDED.available_at),
				completed_at = NULL
		`, accountID); err != nil {
			return fmt.Errorf("enqueue Account artifact deletion: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("stop Account work: %w", err)
	}
	return nil
}

func (s *Store) GetSite(
	ctx context.Context,
	accountID string,
) (*domain.PersonalSite, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, owner_account_id::text, username, display_name,
		       description, visibility, claimed_at, created_at, updated_at
		FROM personal_sites
		WHERE owner_account_id = $1
	`, accountID)
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get Personal Site: %w", err)
	}
	return &site, nil
}

func (s *Store) UsernameAvailable(ctx context.Context, username string) (bool, error) {
	username, err := normalizeUsername(username)
	if err != nil {
		return false, nil
	}
	var available bool
	if err := s.pool.QueryRow(ctx, `
		SELECT NOT EXISTS (
			SELECT 1 FROM personal_sites WHERE username = $1
		)
	`, username).Scan(&available); err != nil {
		return false, fmt.Errorf("check username: %w", err)
	}
	return available, nil
}

func (s *Store) ClaimSite(
	ctx context.Context,
	accountID, username, displayName string,
) (domain.PersonalSite, error) {
	username, err := normalizeUsername(username)
	if err != nil {
		return domain.PersonalSite{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len([]rune(displayName)) > 80 {
		return domain.PersonalSite{}, errors.New("display name must contain 1 to 80 characters")
	}
	now := time.Now().UTC()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO personal_sites (
			id, owner_account_id, username, display_name, description,
			visibility, claimed_at, created_at, updated_at
		)
		SELECT $1, a.id, $2, $3, '', 'private', $4, $4, $4
		FROM accounts a
		WHERE a.id = $5 AND a.status = 'active'
		ON CONFLICT DO NOTHING
		RETURNING id::text, owner_account_id::text, username, display_name,
		          description, visibility, claimed_at, created_at, updated_at
	`, uuid.New(), username, displayName, now, accountID)
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PersonalSite{}, ErrConflict
	}
	if err != nil {
		return domain.PersonalSite{}, fmt.Errorf("claim Personal Site: %w", err)
	}
	return site, nil
}

func (s *Store) UpdateSite(
	ctx context.Context,
	accountID string,
	visibility domain.SiteVisibility,
	displayName, description *string,
) (domain.PersonalSite, error) {
	if visibility != domain.SitePrivate && visibility != domain.SitePublic {
		return domain.PersonalSite{}, errors.New("Personal Site visibility is invalid")
	}
	if displayName != nil {
		normalized := strings.TrimSpace(*displayName)
		if normalized == "" || len([]rune(normalized)) > 80 {
			return domain.PersonalSite{}, errors.New("display name must contain 1 to 80 characters")
		}
		displayName = &normalized
	}
	if description != nil {
		normalized := strings.TrimSpace(*description)
		if len([]rune(normalized)) > 400 {
			return domain.PersonalSite{}, errors.New("description must not exceed 400 characters")
		}
		description = &normalized
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE personal_sites SET
			visibility = $2,
			display_name = COALESCE($3, display_name),
			description = COALESCE($4, description),
			updated_at = now()
		WHERE owner_account_id = $1
		RETURNING id::text, owner_account_id::text, username, display_name,
		          description, visibility, claimed_at, created_at, updated_at
	`, accountID, visibility, displayName, description)
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PersonalSite{}, ErrNotFound
	}
	if err != nil {
		return domain.PersonalSite{}, fmt.Errorf("update Personal Site: %w", err)
	}
	return site, nil
}

func (s *Store) GetPublicSite(
	ctx context.Context,
	username string,
) (domain.PersonalSite, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT s.id::text, s.owner_account_id::text, s.username, s.display_name,
		       s.description, s.visibility, s.claimed_at, s.created_at, s.updated_at
		FROM personal_sites s
		JOIN accounts a ON a.id = s.owner_account_id
		WHERE s.username = $1 AND s.visibility = 'public' AND a.status = 'active'
	`, strings.ToLower(username))
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PersonalSite{}, ErrNotFound
	}
	if err != nil {
		return domain.PersonalSite{}, fmt.Errorf("get public Personal Site: %w", err)
	}
	return site, nil
}

type scanner interface {
	Scan(...any) error
}

func scanAccount(row scanner) (domain.Account, error) {
	var account domain.Account
	err := row.Scan(
		&account.ID,
		&account.ClerkUserID,
		&account.PrimaryEmail,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.DeletedAt,
	)
	return account, err
}

func scanSite(row scanner) (domain.PersonalSite, error) {
	var site domain.PersonalSite
	err := row.Scan(
		&site.ID,
		&site.OwnerAccountID,
		&site.Username,
		&site.DisplayName,
		&site.Description,
		&site.Visibility,
		&site.ClaimedAt,
		&site.CreatedAt,
		&site.UpdatedAt,
	)
	return site, err
}

func normalizeUsername(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if !usernamePattern.MatchString(value) || strings.HasSuffix(value, "-") ||
		strings.Contains(value, "--") {
		return "", errors.New("username must be 3 to 30 lowercase characters and use letters, numbers, or single hyphens")
	}
	if _, reserved := reservedUsernames[value]; reserved {
		return "", errors.New("username is reserved")
	}
	return value, nil
}
