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

type PublicGrowthEvent string

const (
	PublicGrowthView     PublicGrowthEvent = "view"
	PublicGrowthShare    PublicGrowthEvent = "share"
	PublicGrowthCTAClick PublicGrowthEvent = "cta_click"
)

type PublicGrowthAnalytics struct {
	PeriodDays            int `json:"periodDays"`
	Views                 int `json:"views"`
	UniqueViewers         int `json:"uniqueViewers"`
	Shares                int `json:"shares"`
	Follows               int `json:"follows"`
	CTAClicks             int `json:"ctaClicks"`
	AttributedSignups     int `json:"attributedSignups"`
	AttributedActivations int `json:"attributedActivations"`
	PublishedDossiers     int `json:"publishedDossiers"`
}

func (s *Store) RecordPublicGrowthEvent(
	ctx context.Context,
	username, publicID string,
	event PublicGrowthEvent,
	channel, visitorFingerprint string,
	now time.Time,
) error {
	if event != PublicGrowthView && event != PublicGrowthShare && event != PublicGrowthCTAClick {
		return errors.New("public growth event is invalid")
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if event != PublicGrowthShare {
		channel = ""
	} else if channel != "linkedin" && channel != "x" && channel != "email" {
		return errors.New("public share channel is invalid")
	}
	if len(visitorFingerprint) != 64 {
		return errors.New("public visitor fingerprint is invalid")
	}
	rawID := strings.TrimPrefix(publicID, "dossier-")
	if _, err := uuid.Parse(rawID); err != nil {
		return ErrNotFound
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO public_growth_events (
		  issue_id, owner_account_id, event_name, channel,
		  visitor_fingerprint, visitor_day, occurred_at
		)
		SELECT issue.id, newsletter.owner_account_id, $3, $4, $5, $6, $7
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN accounts account ON account.id = newsletter.owner_account_id
		JOIN personal_sites site ON site.owner_account_id = account.id
		WHERE site.username = $1 AND site.visibility = 'public'
		  AND account.status = 'active' AND newsletter.site_visible
		  AND issue.public_id = $2 AND issue.status = 'generated'
		  AND issue.publication_state = 'published'
		  AND issue.moderation_state = 'clear'
		ON CONFLICT (issue_id, event_name, channel, visitor_fingerprint, visitor_day)
		DO NOTHING
	`, strings.ToLower(username), rawID, event, channel, visitorFingerprint,
		now.UTC().Format("2006-01-02"), now)
	if err != nil {
		return fmt.Errorf("record public growth event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		err = s.pool.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM public_growth_events event
			  JOIN issues issue ON issue.id = event.issue_id
			  JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
			  JOIN personal_sites site ON site.owner_account_id = newsletter.owner_account_id
			  WHERE site.username = $1 AND issue.public_id = $2
			    AND event.event_name = $3 AND event.channel = $4
			    AND event.visitor_fingerprint = $5 AND event.visitor_day = $6
			)
		`, strings.ToLower(username), rawID, event, channel, visitorFingerprint,
			now.UTC().Format("2006-01-02")).Scan(&exists)
		if err != nil {
			return fmt.Errorf("verify public growth event: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

func (s *Store) RecordPublicSignupConversion(
	ctx context.Context,
	accountID, referralFingerprint string,
	now time.Time,
) error {
	if len(referralFingerprint) != 64 {
		return errors.New("public referral fingerprint is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO public_attribution_conversions (
		  issue_id, owner_account_id, converted_account_id,
		  referral_fingerprint, converted_at
		)
		SELECT event.issue_id, event.owner_account_id, $1, $2, $3
		FROM public_growth_events event
		JOIN accounts converted ON converted.id = $1
		WHERE event.event_name = 'cta_click'
		  AND event.visitor_fingerprint = $2
		  AND event.occurred_at >= $3::timestamptz - interval '30 days'
		  AND converted.created_at >= event.occurred_at
		  AND converted.created_at <= $3::timestamptz
		ORDER BY event.occurred_at DESC
		LIMIT 1
		ON CONFLICT (converted_account_id) DO NOTHING
	`, accountID, referralFingerprint, now)
	if err != nil {
		return fmt.Errorf("record public signup conversion: %w", err)
	}
	return nil
}

func (s *Store) GetPublicGrowthAnalytics(
	ctx context.Context,
	accountID string,
	periodDays int,
	now time.Time,
) (PublicGrowthAnalytics, error) {
	if periodDays != 7 && periodDays != 30 && periodDays != 90 {
		return PublicGrowthAnalytics{}, errors.New("public analytics period is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	analytics := PublicGrowthAnalytics{PeriodDays: periodDays}
	err := s.pool.QueryRow(ctx, `
		WITH growth AS (
		  SELECT event_name, visitor_fingerprint
		  FROM public_growth_events
		  WHERE owner_account_id = $1
		    AND occurred_at >= $2::timestamptz - ($3 * interval '1 day')
		), conversions AS (
		  SELECT count(*)::int AS signups,
		         count(*) FILTER (WHERE activated_at IS NOT NULL)::int AS activations
		  FROM public_attribution_conversions
		  WHERE owner_account_id = $1
		    AND converted_at >= $2::timestamptz - ($3 * interval '1 day')
		), published AS (
		  SELECT count(*)::int AS count
		  FROM issues issue
		  JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		  WHERE newsletter.owner_account_id = $1
		    AND issue.status = 'generated'
		    AND issue.publication_state = 'published'
		    AND issue.moderation_state = 'clear'
		)
		SELECT
		  count(*) FILTER (WHERE event_name = 'view')::int,
		  count(DISTINCT visitor_fingerprint) FILTER (WHERE event_name = 'view')::int,
		  count(*) FILTER (WHERE event_name = 'share')::int,
		  count(*) FILTER (WHERE event_name = 'cta_click')::int,
		  count(*) FILTER (WHERE event_name = 'follow')::int,
		  (SELECT signups FROM conversions),
		  (SELECT activations FROM conversions),
		  (SELECT count FROM published)
		FROM growth
	`, accountID, now, periodDays).Scan(
		&analytics.Views, &analytics.UniqueViewers, &analytics.Shares,
		&analytics.CTAClicks, &analytics.Follows, &analytics.AttributedSignups,
		&analytics.AttributedActivations, &analytics.PublishedDossiers,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return analytics, nil
	}
	if err != nil {
		return PublicGrowthAnalytics{}, fmt.Errorf("load public growth analytics: %w", err)
	}
	return analytics, nil
}
