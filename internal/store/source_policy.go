package store

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

const (
	SourcePolicyExactURL          = "exact_url"
	SourcePolicyRegistrableDomain = "registrable_domain"
	SourcePolicyBlock             = "block"
	SourcePolicyUnblock           = "unblock"
)

func (s *Store) RecordSourceRetrievalPolicy(
	ctx context.Context,
	operatorAccountID, scope, value, action, caseReference, reason string,
	now time.Time,
) error {
	if _, err := uuid.Parse(operatorAccountID); err != nil {
		return errors.New("operator account ID is invalid")
	}
	normalized, err := normalizeSourcePolicyValue(scope, value)
	if err != nil {
		return err
	}
	if action != SourcePolicyBlock && action != SourcePolicyUnblock {
		return errors.New("source policy action is invalid")
	}
	caseReference = strings.TrimSpace(caseReference)
	reason = strings.TrimSpace(reason)
	if len(caseReference) < 3 || len(caseReference) > 80 ||
		len(reason) < 10 || len(reason) > 800 ||
		strings.ContainsAny(caseReference+reason, "\r\n") {
		return errors.New("source policy audit detail is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 0))
	`, scope, normalized); err != nil {
		return fmt.Errorf("lock source retrieval policy: %w", err)
	}
	var active bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND status = 'active')
	`, operatorAccountID).Scan(&active); err != nil {
		return fmt.Errorf("verify source-policy operator: %w", err)
	}
	if !active {
		return ErrForbidden
	}
	var currentAction string
	err = tx.QueryRow(ctx, `
		SELECT action
		FROM current_source_retrieval_policy
		WHERE scope = $1 AND value = $2
	`, scope, normalized).Scan(&currentAction)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read current source policy: %w", err)
	}
	if err == nil && currentAction == action {
		return ErrConflict
	}
	if action == SourcePolicyUnblock && errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO source_retrieval_policy_events (
		  id, scope, value, action, case_reference, reason,
		  actor_account_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.NewString(), scope, normalized, action, caseReference, reason,
		operatorAccountID, now); err != nil {
		return fmt.Errorf("record source retrieval policy: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) SourceURLAllowed(ctx context.Context, rawURL string) (bool, error) {
	exact, domain, err := sourcePolicyKeys(rawURL)
	if err != nil {
		return false, err
	}
	var blocked bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM current_source_retrieval_policy
		  WHERE action = 'block'
		    AND ((scope = 'exact_url' AND value = $1)
		      OR (scope = 'registrable_domain' AND value = $2))
		)
	`, exact, domain).Scan(&blocked); err != nil {
		return false, fmt.Errorf("check source retrieval policy: %w", err)
	}
	return !blocked, nil
}

func normalizeSourcePolicyValue(scope, value string) (string, error) {
	switch scope {
	case SourcePolicyExactURL:
		exact, _, err := sourcePolicyKeys(value)
		return exact, err
	case SourcePolicyRegistrableDomain:
		value = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(value, ".")))
		ascii, err := idna.Lookup.ToASCII(value)
		if err != nil {
			return "", errors.New("source policy domain is invalid")
		}
		registrable, err := publicsuffix.EffectiveTLDPlusOne(ascii)
		if err != nil || registrable != ascii {
			return "", errors.New("source policy domain must be a registrable domain")
		}
		return registrable, nil
	default:
		return "", errors.New("source policy scope is invalid")
	}
}

func sourcePolicyKeys(rawURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Hostname() == "" || parsed.User != nil {
		return "", "", errors.New("source policy URL is invalid")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if net.ParseIP(host) == nil {
		host, err = idna.Lookup.ToASCII(host)
		if err != nil {
			return "", "", errors.New("source policy URL host is invalid")
		}
	}
	registrable := ""
	if net.ParseIP(host) == nil {
		registrable, err = publicsuffix.EffectiveTLDPlusOne(host)
		if err != nil {
			return "", "", errors.New("source policy URL must use a registrable public domain or IP address")
		}
	}
	port := parsed.Port()
	if (parsed.Scheme == "https" && port == "443") || (parsed.Scheme == "http" && port == "80") {
		port = ""
	}
	parsed.Host = host
	if strings.Contains(host, ":") {
		parsed.Host = "[" + host + "]"
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), registrable, nil
}
