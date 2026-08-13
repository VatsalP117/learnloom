package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NewsletterRecord struct {
	domain.Newsletter
	IssueCount              int `json:"issueCount"`
	GeneratedCount          int `json:"generatedCount"`
	SentCount               int `json:"sentCount"`
	CapabilityCount         int `json:"capabilityCount"`
	RecalledCapabilityCount int `json:"recalledCapabilityCount"`
	CurrentGapCount         int `json:"currentGapCount"`
}

type NewsletterInput struct {
	Name                    string
	Topic                   string
	LearnerLevel            string
	LearnerGoal             string
	LessonMinutes           int
	SourceMode              domain.SourceMode
	SourceReviewMode        domain.SourceReviewMode
	Sources                 []domain.SourceDefinition
	ScheduleHour            int
	ScheduleMinute          int
	TimeZone                string
	Active                  bool
	EmailEnabled            bool
	AIExplorationEnabled    bool
	SiteVisible             bool
	TemplateID              string
	TemplateVersion         int
	OnboardingDraftID       string
	OnboardingDraftRevision int64
}

type RhythmInput struct {
	Mode                domain.RhythmMode
	SelectedWeekdays    []int
	AutoThrottleEnabled bool
	UnopenedLessonLimit int
}

type CreateNewsletterResult struct {
	Newsletter NewsletterRecord
	FirstIssue domain.Issue
}

func (s *Store) CreateNewsletter(
	ctx context.Context,
	accountID string,
	input NewsletterInput,
	maximumPerAccount int,
	dailyAccountLimit int,
) (CreateNewsletterResult, error) {
	if input.Name == "" || input.LearnerLevel == "" || input.LearnerGoal == "" || input.LessonMinutes == 0 {
		input = applyCreateDefaults(input)
	}
	normalized, err := normalizeNewsletterInput(input)
	if err != nil {
		return CreateNewsletterResult{}, err
	}
	if maximumPerAccount < 1 {
		maximumPerAccount = 10
	}
	now := time.Now().UTC()
	next, err := NextRhythmOccurrence(
		now,
		normalized.TimeZone,
		normalized.ScheduleHour,
		normalized.ScheduleMinute,
		domain.RhythmDaily,
		defaultSelectedWeekdays(),
	)
	if err != nil {
		return CreateNewsletterResult{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateNewsletterResult{}, err
	}
	defer rollback(tx)
	var status domain.AccountStatus
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT status FROM accounts WHERE id = $1 FOR UPDATE
	`, accountID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CreateNewsletterResult{}, ErrForbidden
		}
		return CreateNewsletterResult{}, fmt.Errorf("inspect Account Newsletter quota: %w", err)
	}
	if status != domain.AccountActive {
		return CreateNewsletterResult{}, ErrForbidden
	}
	if err := tx.QueryRow(
		ctx,
		"SELECT count(*) FROM newsletters WHERE owner_account_id = $1",
		accountID,
	).Scan(&count); err != nil {
		return CreateNewsletterResult{}, fmt.Errorf("count Account Newsletters: %w", err)
	}
	if count >= maximumPerAccount {
		return CreateNewsletterResult{}, ErrQuotaExceeded
	}
	publicSlug, err := allocateNewsletterSlug(ctx, tx, accountID, normalized.Name)
	if err != nil {
		return CreateNewsletterResult{}, err
	}
	id := uuid.New()
	var templateID, templateVersion any
	if normalized.TemplateID != "" {
		templateID = normalized.TemplateID
		templateVersion = normalized.TemplateVersion
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO newsletters (
			id, owner_account_id, name, topic, learner_level, learner_goal,
			lesson_minutes, source_mode, schedule_hour, schedule_minute,
			time_zone, active, next_run_at, email_enabled, ai_exploration_enabled,
			public_slug, site_visible, stream_template_id,
			stream_template_version, source_review_mode, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $21
		)
		RETURNING id::text, owner_account_id::text, name, topic, learner_level,
		          learner_goal, lesson_minutes, source_mode, source_review_mode,
		          source_approved_at, '[]'::jsonb, schedule_hour,
		          schedule_minute, time_zone, rhythm_mode, selected_weekdays,
		          effective_rhythm_mode, auto_throttle_enabled,
		          unopened_lesson_limit, rhythm_reason, rhythm_throttled_at,
		          active, lesson_publication_default,
		          lesson_publication_default_reviewed_at,
		          next_run_at, email_enabled,
		          ai_exploration_enabled, public_slug, site_visible, created_at,
		          updated_at, 0, 0, 0, 0, 0, 0
	`, id, accountID, normalized.Name, normalized.Topic, normalized.LearnerLevel,
		normalized.LearnerGoal, normalized.LessonMinutes, normalized.SourceMode,
		normalized.ScheduleHour, normalized.ScheduleMinute, normalized.TimeZone,
		normalized.Active, next, normalized.EmailEnabled,
		normalized.AIExplorationEnabled, publicSlug, normalized.SiteVisible,
		templateID, templateVersion, normalized.SourceReviewMode, now)
	record, err := scanNewsletterRecord(row)
	if err != nil {
		return CreateNewsletterResult{}, fmt.Errorf("create Newsletter: %w", err)
	}
	record.Sources = normalized.Sources

	for index, source := range normalized.Sources {
		scope := domain.SourceScopeExact
		kind := (*string)(nil)
		lowerURL := strings.ToLower(source.URL)
		if strings.Contains(lowerURL, "/feed") || strings.Contains(lowerURL, "/rss") {
			scope = domain.SourceScopeFeed
			k := "rss"
			kind = &k
		} else if strings.Contains(lowerURL, "/atom") {
			scope = domain.SourceScopeFeed
			k := "atom"
			kind = &k
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO source_specs (
				id, newsletter_id, origin, state, display_name, input_url,
				canonical_url, scope, kind, item_limit, created_at, updated_at
			)
			VALUES ($1, $2, 'provided', 'active', $3, $4, $5, $6, $7, $8, $9, $9)
		`, uuid.New(), id, source.Name, source.URL, source.URL, scope, kind, source.Limit, now)
		if err != nil {
			return CreateNewsletterResult{}, fmt.Errorf("backfill source spec %d: %w", index+1, err)
		}
	}

	issueID := uuid.New()
	publicID := uuid.New()
	if err := reserveGenerationUsageTx(
		ctx, tx, accountID, issueID.String(), now,
	); err != nil {
		return CreateNewsletterResult{}, err
	}

	var todayCount int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1
		  AND i.created_at >= date_trunc('day', $2 AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
		  AND i.status <> 'cancelled'
	`, accountID, now).Scan(&todayCount); err != nil {
		return CreateNewsletterResult{}, fmt.Errorf("check daily Issue quota: %w", err)
	}
	if dailyAccountLimit < 1 {
		dailyAccountLimit = 5
	}
	if todayCount >= dailyAccountLimit {
		return CreateNewsletterResult{}, ErrQuotaExceeded
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO issues (
			id, newsletter_id, trigger, status, available_at, public_id,
			publication_state, created_at
		)
		VALUES ($1, $2, 'manual', 'queued', $3, $4, 'draft', $3)
	`, issueID, id, now, publicID); err != nil {
		return CreateNewsletterResult{}, fmt.Errorf("enqueue first Issue: %w", err)
	}
	issue, err := getIssueTx(ctx, tx, accountID, issueID.String())
	if err != nil {
		return CreateNewsletterResult{}, fmt.Errorf("load first Issue: %w", err)
	}
	if normalized.OnboardingDraftID != "" {
		var completedDraftID string
		if err := tx.QueryRow(ctx, `
			DELETE FROM onboarding_drafts
			WHERE account_id = $1 AND id = $2::uuid AND revision = $3
			RETURNING id::text
		`, accountID, normalized.OnboardingDraftID, normalized.OnboardingDraftRevision).Scan(
			&completedDraftID,
		); errors.Is(err, pgx.ErrNoRows) {
			return CreateNewsletterResult{}, ErrConflict
		} else if err != nil {
			return CreateNewsletterResult{}, fmt.Errorf("complete onboarding draft: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO onboarding_draft_completions (
			  draft_id, account_id, newsletter_id, completed_at
			)
			VALUES ($1::uuid, $2, $3, $4)
		`, completedDraftID, accountID, id, now); err != nil {
			return CreateNewsletterResult{}, fmt.Errorf("tombstone onboarding draft: %w", err)
		}
		if err := insertProductEvent(
			ctx, tx, accountID, ProductEventOnboardingConfirmed,
			"onboarding", completedDraftID, now,
		); err != nil {
			return CreateNewsletterResult{}, err
		}
	}
	if err := insertProductEvent(
		ctx, tx, accountID, ProductEventStreamCreated,
		"stream", record.ID, now,
	); err != nil {
		return CreateNewsletterResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateNewsletterResult{}, fmt.Errorf("commit Newsletter: %w", err)
	}
	return CreateNewsletterResult{Newsletter: record, FirstIssue: issue}, nil
}

func (s *Store) ListNewsletters(
	ctx context.Context,
	accountID string,
) ([]NewsletterRecord, error) {
	rows, err := s.pool.Query(ctx, newsletterSelect+`
		WHERE n.owner_account_id = $1
		GROUP BY n.id
		ORDER BY n.created_at DESC
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list Newsletters: %w", err)
	}
	defer rows.Close()
	var records []NewsletterRecord
	for rows.Next() {
		record, err := scanNewsletterRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan Newsletter: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) GetNewsletter(
	ctx context.Context,
	accountID, newsletterID string,
) (NewsletterRecord, error) {
	row := s.pool.QueryRow(ctx, newsletterSelect+`
		WHERE n.owner_account_id = $1 AND n.id = $2
		GROUP BY n.id
	`, accountID, newsletterID)
	record, err := scanNewsletterRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return NewsletterRecord{}, ErrNotFound
	}
	if err != nil {
		return NewsletterRecord{}, fmt.Errorf("get Newsletter: %w", err)
	}
	return record, nil
}

func (s *Store) UpdateNewsletter(
	ctx context.Context,
	accountID, newsletterID string,
	input NewsletterInput,
) (NewsletterRecord, error) {
	normalized, err := normalizeNewsletterInput(input)
	if err != nil {
		return NewsletterRecord{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return NewsletterRecord{}, err
	}
	defer rollback(tx)
	var effectiveMode domain.RhythmMode
	var selectedWeekdays []int16
	if err := tx.QueryRow(ctx, `
		SELECT effective_rhythm_mode, selected_weekdays
		FROM newsletters
		WHERE owner_account_id = $1 AND id = $2
		FOR UPDATE
	`, accountID, newsletterID).Scan(&effectiveMode, &selectedWeekdays); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NewsletterRecord{}, ErrNotFound
		}
		return NewsletterRecord{}, err
	}
	next, err := NextRhythmOccurrence(
		time.Now().UTC(),
		normalized.TimeZone,
		normalized.ScheduleHour,
		normalized.ScheduleMinute,
		effectiveMode,
		weekdayInts(selectedWeekdays),
	)
	if err != nil {
		return NewsletterRecord{}, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE newsletters SET
			name = $3, topic = $4, learner_level = $5, learner_goal = $6,
			lesson_minutes = $7, source_mode = $8, schedule_hour = $9,
			schedule_minute = $10, time_zone = $11, active = $12,
			next_run_at = $13, email_enabled = $14,
			ai_exploration_enabled = $15, site_visible = $16,
			source_review_mode = $17, updated_at = now()
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, normalized.Name, normalized.Topic,
		normalized.LearnerLevel, normalized.LearnerGoal, normalized.LessonMinutes,
		normalized.SourceMode, normalized.ScheduleHour, normalized.ScheduleMinute,
		normalized.TimeZone, normalized.Active, next, normalized.EmailEnabled,
		normalized.AIExplorationEnabled, normalized.SiteVisible,
		normalized.SourceReviewMode)
	if err != nil {
		return NewsletterRecord{}, fmt.Errorf("update Newsletter: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return NewsletterRecord{}, ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE source_specs SET state = 'disabled', updated_at = now()
		WHERE newsletter_id = $1 AND origin = 'provided'
	`, newsletterID); err != nil {
		return NewsletterRecord{}, fmt.Errorf("disable source specs: %w", err)
	}
	discoveredState := domain.SourceStateActive
	if normalized.SourceMode == domain.SourceModeProvided {
		discoveredState = domain.SourceStateDisabled
	}
	if _, err := tx.Exec(ctx, `
		UPDATE source_specs SET
		  state = CASE WHEN preference = 'blocked' THEN 'disabled' ELSE $2 END,
		  updated_at = now()
		WHERE newsletter_id = $1 AND origin = 'discovered'
		  AND state IN ('active', 'disabled')
	`, newsletterID, discoveredState); err != nil {
		return NewsletterRecord{}, fmt.Errorf("reconcile discovered source specs: %w", err)
	}

	now := time.Now().UTC()
	for _, source := range normalized.Sources {
		scope := domain.SourceScopeExact
		kind := (*string)(nil)
		lowerURL := strings.ToLower(source.URL)
		if strings.Contains(lowerURL, "/feed") || strings.Contains(lowerURL, "/rss") {
			scope = domain.SourceScopeFeed
			k := "rss"
			kind = &k
		} else if strings.Contains(lowerURL, "/atom") {
			scope = domain.SourceScopeFeed
			k := "atom"
			kind = &k
		}
		var existingID string
		err := tx.QueryRow(ctx, `
			SELECT id::text FROM source_specs
			WHERE newsletter_id = $1 AND canonical_url = $2 AND origin = 'provided'
		`, newsletterID, source.URL).Scan(&existingID)
		if err == nil {
			if _, err := tx.Exec(ctx, `
				UPDATE source_specs SET
					state = CASE WHEN preference = 'blocked' THEN 'disabled' ELSE 'active' END,
					display_name = $2, input_url = $3,
					scope = $4, kind = $5, item_limit = $6, updated_at = $7
				WHERE id = $1
			`, existingID, source.Name, source.URL, scope, kind, source.Limit, now); err != nil {
				return NewsletterRecord{}, fmt.Errorf("update source spec: %w", err)
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return NewsletterRecord{}, fmt.Errorf("find source spec: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO source_specs (
				id, newsletter_id, origin, state, display_name, input_url,
				canonical_url, scope, kind, item_limit, created_at, updated_at
			)
			VALUES ($1, $2, 'provided', 'active', $3, $4, $5, $6, $7, $8, $9, $9)
		`, uuid.New(), newsletterID, source.Name, source.URL,
			source.URL, scope, kind, source.Limit, now)
		if err != nil {
			return NewsletterRecord{}, fmt.Errorf("insert source spec: %w", err)
		}
	}
	if err := resetAwaitingSourcePortfolio(ctx, tx, newsletterID, now); err != nil {
		return NewsletterRecord{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return NewsletterRecord{}, err
	}
	return s.GetNewsletter(ctx, accountID, newsletterID)
}

func (s *Store) SetNewsletterActive(
	ctx context.Context,
	accountID, newsletterID string,
	active bool,
) error {
	now := time.Now().UTC()
	var next time.Time
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if active {
		var zone string
		var hour, minute int
		var mode domain.RhythmMode
		var weekdays []int16
		if err := tx.QueryRow(ctx, `
			SELECT time_zone, schedule_hour, schedule_minute,
			       effective_rhythm_mode, selected_weekdays
			FROM newsletters WHERE owner_account_id = $1 AND id = $2
		`, accountID, newsletterID).Scan(&zone, &hour, &minute, &mode, &weekdays); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		var err error
		next, err = NextRhythmOccurrence(now, zone, hour, minute, mode, weekdayInts(weekdays))
		if err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `
		UPDATE newsletters SET
			active = $3,
			next_run_at = CASE WHEN $3 THEN $4 ELSE next_run_at END,
			updated_at = $5
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, active, next, now)
	if err != nil {
		return fmt.Errorf("set Newsletter active state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !active {
		if _, err := tx.Exec(ctx, `
			UPDATE issues SET
				status = 'cancelled', completed_at = $2,
				error = 'Newsletter schedule was paused'
			WHERE newsletter_id = $1 AND trigger = 'scheduled'
			  AND status = 'queued'
		`, newsletterID, now); err != nil {
			return fmt.Errorf("cancel scheduled Issues: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) SetNewsletterRhythm(
	ctx context.Context,
	accountID, newsletterID string,
	input RhythmInput,
	now time.Time,
) (NewsletterRecord, error) {
	input, err := normalizeRhythmInput(input)
	if err != nil {
		return NewsletterRecord{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return NewsletterRecord{}, err
	}
	defer rollback(tx)
	var zone string
	var hour, minute int
	if err := tx.QueryRow(ctx, `
		SELECT time_zone, schedule_hour, schedule_minute
		FROM newsletters
		WHERE owner_account_id = $1 AND id = $2
		FOR UPDATE
	`, accountID, newsletterID).Scan(&zone, &hour, &minute); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NewsletterRecord{}, ErrNotFound
		}
		return NewsletterRecord{}, err
	}
	unopened, err := unopenedLessonCount(ctx, tx, newsletterID)
	if err != nil {
		return NewsletterRecord{}, err
	}
	effective := input.Mode
	reason := rhythmPreferenceReason(input.Mode, input.SelectedWeekdays)
	var throttledAt *time.Time
	decision := "configure"
	if input.AutoThrottleEnabled && input.Mode != domain.RhythmWeeklySynthesis &&
		unopened >= input.UnopenedLessonLimit {
		effective = domain.RhythmWeeklySynthesis
		reason = backlogRhythmReason(unopened, input.UnopenedLessonLimit)
		value := now.UTC()
		throttledAt = &value
		decision = "throttle"
	}
	next, err := NextRhythmOccurrence(
		now,
		zone,
		hour,
		minute,
		effective,
		input.SelectedWeekdays,
	)
	if err != nil {
		return NewsletterRecord{}, err
	}
	weekdays := weekdaySmallInts(input.SelectedWeekdays)
	tag, err := tx.Exec(ctx, `
		UPDATE newsletters SET
			rhythm_mode = $3,
			selected_weekdays = $4,
			effective_rhythm_mode = $5,
			auto_throttle_enabled = $6,
			unopened_lesson_limit = $7,
			rhythm_reason = $8,
			rhythm_throttled_at = $9,
			next_run_at = $10,
			updated_at = $11
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, input.Mode, weekdays, effective,
		input.AutoThrottleEnabled, input.UnopenedLessonLimit, reason,
		throttledAt, next, now)
	if err != nil {
		return NewsletterRecord{}, fmt.Errorf("set Newsletter rhythm: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return NewsletterRecord{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO rhythm_decisions (
			newsletter_id, decision, desired_mode, effective_mode,
			unopened_count, reason, decided_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newsletterID, decision, input.Mode, effective, unopened, reason, now); err != nil {
		return NewsletterRecord{}, fmt.Errorf("record Newsletter rhythm decision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return NewsletterRecord{}, err
	}
	return s.GetNewsletter(ctx, accountID, newsletterID)
}

func (s *Store) SetNewsletterEmail(
	ctx context.Context,
	accountID, newsletterID string,
	enabled bool,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	tag, err := tx.Exec(ctx, `
		UPDATE newsletters SET email_enabled = $3, updated_at = now()
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !enabled {
		if _, err := tx.Exec(ctx, `
			UPDATE delivery_receipts d SET
				status = 'cancelled', claim_token = NULL, claim_expires_at = NULL,
				error = 'Newsletter email is disabled', updated_at = now()
			FROM issues i
			WHERE d.issue_id = i.id AND i.newsletter_id = $1
			AND d.status IN ('pending', 'failed')
		`, newsletterID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) SetNewsletterContent(
	ctx context.Context,
	accountID, newsletterID string,
	aiExploration bool,
) error {
	return s.updateNewsletterBoolean(
		ctx,
		accountID,
		newsletterID,
		"ai_exploration_enabled",
		aiExploration,
	)
}

func (s *Store) SetNewsletterSiteVisible(
	ctx context.Context,
	accountID, newsletterID string,
	visible bool,
) error {
	return s.updateNewsletterBoolean(
		ctx,
		accountID,
		newsletterID,
		"site_visible",
		visible,
	)
}

func (s *Store) SetNewsletterPublicationDefault(
	ctx context.Context,
	accountID, newsletterID string,
	state domain.PublicationState,
	audienceConfirmed bool,
	now time.Time,
) error {
	if state != domain.PublicationDraft && state != domain.PublicationPublished {
		return errors.New("lesson publication default must be draft or published")
	}
	if state == domain.PublicationPublished && !audienceConfirmed {
		return errors.New("automatic publishing requires audience confirmation")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE newsletters SET
			lesson_publication_default = $3,
			lesson_publication_default_reviewed_at = CASE
			  WHEN $3::text = 'published' THEN $4::timestamptz
			  ELSE NULL::timestamptz
			END,
			updated_at = $4
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, state, now)
	if err != nil {
		return fmt.Errorf("set lesson publication default: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) updateNewsletterBoolean(
	ctx context.Context,
	accountID, newsletterID, column string,
	value bool,
) error {
	if column != "ai_exploration_enabled" && column != "site_visible" {
		return errors.New("unsupported Newsletter setting")
	}
	query := fmt.Sprintf(`
		UPDATE newsletters SET %s = $3, updated_at = now()
		WHERE owner_account_id = $1 AND id = $2
	`, column)
	tag, err := s.pool.Exec(ctx, query, accountID, newsletterID, value)
	if err != nil {
		return fmt.Errorf("update Newsletter setting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const newsletterSelect = `
	SELECT n.id::text, n.owner_account_id::text, n.name, n.topic,
	       n.learner_level, n.learner_goal, n.lesson_minutes, n.source_mode,
	       n.source_review_mode, n.source_approved_at,
	       ` + providedSourcesProjection + `, n.schedule_hour, n.schedule_minute,
	       n.time_zone, n.rhythm_mode, n.selected_weekdays,
	       n.effective_rhythm_mode, n.auto_throttle_enabled,
	       n.unopened_lesson_limit, n.rhythm_reason, n.rhythm_throttled_at, n.active,
	       n.lesson_publication_default, n.lesson_publication_default_reviewed_at,
	       n.next_run_at, n.email_enabled, n.ai_exploration_enabled,
	       n.public_slug, n.site_visible, n.created_at, n.updated_at,
	       count(DISTINCT i.id)::int,
	       count(DISTINCT i.id) FILTER (WHERE i.status = 'generated')::int,
	       count(DISTINCT d.issue_id) FILTER (WHERE d.status = 'delivered')::int,
	       COALESCE((
	         SELECT count(*)::int FROM learner_concept_state capability
	         WHERE capability.newsletter_id = n.id
	           AND capability.completed_count > 0
	       ), 0),
	       COALESCE((
	         SELECT count(*)::int FROM learner_concept_state capability
	         WHERE capability.newsletter_id = n.id
	           AND capability.review_attempt_count > 0
	           AND capability.confidence_score >= 75
	       ), 0),
	       COALESCE((
	         SELECT count(*)::int FROM learner_concept_state capability
	         WHERE capability.newsletter_id = n.id
	           AND (
	             capability.completed_count = 0
	             OR capability.review_attempt_count = 0
	             OR capability.confidence_score < 60
	           )
	       ), 0)
	FROM newsletters n
	LEFT JOIN issues i ON i.newsletter_id = n.id
	LEFT JOIN delivery_receipts d ON d.issue_id = i.id
`

const providedSourcesProjection = `
	COALESCE((
	  SELECT jsonb_agg(
	    jsonb_build_object(
	      'name', ss.display_name,
	      'url', ss.input_url,
	      'limit', ss.item_limit
	    )
	    ORDER BY ss.created_at, ss.id
	  )
	  FROM source_specs ss
	  WHERE ss.newsletter_id = n.id
	    AND ss.origin = 'provided'
	    AND ss.state = 'active'
	), '[]'::jsonb)
`

func scanNewsletterRecord(row scanner) (NewsletterRecord, error) {
	var record NewsletterRecord
	var rawSources []byte
	var selectedWeekdays []int16
	err := row.Scan(
		&record.ID,
		&record.OwnerAccountID,
		&record.Name,
		&record.Topic,
		&record.LearnerLevel,
		&record.LearnerGoal,
		&record.LessonMinutes,
		&record.SourceMode,
		&record.SourceReviewMode,
		&record.SourceApprovedAt,
		&rawSources,
		&record.ScheduleHour,
		&record.ScheduleMinute,
		&record.TimeZone,
		&record.RhythmMode,
		&selectedWeekdays,
		&record.EffectiveRhythmMode,
		&record.AutoThrottleEnabled,
		&record.UnopenedLessonLimit,
		&record.RhythmReason,
		&record.RhythmThrottledAt,
		&record.Active,
		&record.LessonPublicationDefault,
		&record.LessonPublicationDefaultReviewedAt,
		&record.NextRunAt,
		&record.EmailEnabled,
		&record.AIExplorationEnabled,
		&record.PublicSlug,
		&record.SiteVisible,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.IssueCount,
		&record.GeneratedCount,
		&record.SentCount,
		&record.CapabilityCount,
		&record.RecalledCapabilityCount,
		&record.CurrentGapCount,
	)
	if err != nil {
		return NewsletterRecord{}, err
	}
	if len(rawSources) == 0 || string(rawSources) == "null" {
		record.Sources = nil
	} else if err := json.Unmarshal(rawSources, &record.Sources); err != nil {
		return NewsletterRecord{}, fmt.Errorf("decode Newsletter sources: %w", err)
	}
	record.SelectedWeekdays = weekdayInts(selectedWeekdays)
	return record, nil
}

func normalizeNewsletterInput(input NewsletterInput) (NewsletterInput, error) {
	var err error
	if input.OnboardingDraftID != "" {
		if _, parseErr := uuid.Parse(input.OnboardingDraftID); parseErr != nil {
			return NewsletterInput{}, errors.New("onboardingDraftId is invalid")
		}
		if input.OnboardingDraftRevision < 1 {
			return NewsletterInput{}, errors.New("onboardingDraftRevision is invalid")
		}
	} else if input.OnboardingDraftRevision != 0 {
		return NewsletterInput{}, errors.New("onboardingDraftRevision requires onboardingDraftId")
	}
	input.Name, err = boundedText(input.Name, "Newsletter name", 80)
	if err != nil {
		return NewsletterInput{}, err
	}
	input.Topic, err = boundedText(input.Topic, "Newsletter topic", 400)
	if err != nil {
		return NewsletterInput{}, err
	}
	input.LearnerLevel, err = boundedText(input.LearnerLevel, "learner level", 120)
	if err != nil {
		return NewsletterInput{}, err
	}
	input.LearnerGoal, err = boundedText(input.LearnerGoal, "learner goal", 500)
	if err != nil {
		return NewsletterInput{}, err
	}
	if input.LessonMinutes < 5 || input.LessonMinutes > 90 {
		return NewsletterInput{}, errors.New("lesson minutes must be from 5 to 90")
	}
	if input.TemplateID == "" {
		if input.TemplateVersion != 0 {
			return NewsletterInput{}, errors.New("template version requires a template ID")
		}
	} else if !streamTemplateIDPattern.MatchString(input.TemplateID) ||
		input.TemplateVersion < 1 {
		return NewsletterInput{}, errors.New("stream template attribution is invalid")
	}
	if input.ScheduleHour < 0 || input.ScheduleHour > 23 ||
		input.ScheduleMinute < 0 || input.ScheduleMinute > 59 {
		return NewsletterInput{}, errors.New("Newsletter schedule is invalid")
	}
	if _, err := time.LoadLocation(input.TimeZone); err != nil {
		return NewsletterInput{}, errors.New("Newsletter timezone is invalid")
	}
	if input.SourceMode == "" {
		input.SourceMode = domain.SourceModeProvided
	}
	if input.SourceReviewMode == "" {
		input.SourceReviewMode = domain.SourceReviewAuto
	}
	if input.SourceReviewMode != domain.SourceReviewAuto &&
		input.SourceReviewMode != domain.SourceReviewBeforeLesson {
		return NewsletterInput{}, errors.New("sourceReviewMode must be auto or review")
	}
	switch input.SourceMode {
	case domain.SourceModeDiscovered:
		if len(input.Sources) != 0 {
			return NewsletterInput{}, errors.New("discovered mode must not include provided sources")
		}
	case domain.SourceModeProvided:
		if len(input.Sources) == 0 {
			return NewsletterInput{}, errors.New("provided mode requires at least one source")
		}
	case domain.SourceModeHybrid:
		if len(input.Sources) == 0 {
			return NewsletterInput{}, errors.New("hybrid mode requires at least one provided source")
		}
	default:
		return NewsletterInput{}, errors.New("sourceMode must be discovered, provided, or hybrid")
	}
	if len(input.Sources) > 12 {
		return NewsletterInput{}, errors.New("Newsletter supports at most 12 provided sources")
	}
	for index := range input.Sources {
		input.Sources[index].Name, err = boundedText(
			input.Sources[index].Name,
			fmt.Sprintf("source %d name", index+1),
			120,
		)
		if err != nil {
			return NewsletterInput{}, err
		}
		parsed, parseErr := url.Parse(strings.TrimSpace(input.Sources[index].URL))
		if parseErr != nil || parsed.Host == "" || parsed.User != nil ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") {
			return NewsletterInput{}, fmt.Errorf("source %d URL is invalid", index+1)
		}
		input.Sources[index].URL = parsed.String()
		if input.Sources[index].Limit < 1 || input.Sources[index].Limit > 50 {
			return NewsletterInput{}, fmt.Errorf("source %d limit must be from 1 to 50", index+1)
		}
	}
	return input, nil
}

func applyCreateDefaults(input NewsletterInput) NewsletterInput {
	if input.Name == "" {
		name := strings.TrimSpace(input.Topic)
		if len([]rune(name)) > 60 {
			name = string([]rune(name)[:60])
		}
		input.Name = name
	}
	if input.LearnerLevel == "" {
		input.LearnerLevel = "intermediate"
	}
	if input.LearnerGoal == "" {
		input.LearnerGoal = "Build a practical understanding of " + input.Topic + "."
		const maxGoal = 500
		if len([]rune(input.LearnerGoal)) > maxGoal {
			input.LearnerGoal = string([]rune(input.LearnerGoal)[:maxGoal-1]) + "."
		}
	}
	if input.LessonMinutes == 0 {
		input.LessonMinutes = 12
	}
	for i := range input.Sources {
		if input.Sources[i].Name == "" {
			parsed, err := url.Parse(strings.TrimSpace(input.Sources[i].URL))
			if err == nil && parsed.Host != "" {
				input.Sources[i].Name = strings.TrimPrefix(parsed.Host, "www.")
			} else if input.Sources[i].URL != "" {
				input.Sources[i].Name = input.Sources[i].URL
			}
		}
	}
	return input
}

func NextOccurrence(after time.Time, zone string, hour, minute int) (time.Time, error) {
	return NextRhythmOccurrence(
		after,
		zone,
		hour,
		minute,
		domain.RhythmDaily,
		defaultSelectedWeekdays(),
	)
}

func NextRhythmOccurrence(
	after time.Time,
	zone string,
	hour, minute int,
	mode domain.RhythmMode,
	selectedWeekdays []int,
) (time.Time, error) {
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, errors.New("Newsletter timezone is invalid")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, errors.New("Newsletter schedule is invalid")
	}
	input, err := normalizeRhythmInput(RhythmInput{
		Mode:                mode,
		SelectedWeekdays:    selectedWeekdays,
		AutoThrottleEnabled: true,
		UnopenedLessonLimit: 3,
	})
	if err != nil {
		return time.Time{}, err
	}
	allowed := make(map[int]bool, len(input.SelectedWeekdays))
	for _, weekday := range input.SelectedWeekdays {
		allowed[weekday] = true
	}
	weeklyDay := input.SelectedWeekdays[0]
	candidate := after.UTC().Truncate(time.Minute).Add(time.Minute)
	for count := 0; count < 8*24*60; count++ {
		local := candidate.In(location)
		isoWeekday := int(local.Weekday())
		if isoWeekday == 0 {
			isoWeekday = 7
		}
		dayAllowed := true
		switch input.Mode {
		case domain.RhythmSelectedWeekdays:
			dayAllowed = allowed[isoWeekday]
		case domain.RhythmWeeklySynthesis:
			dayAllowed = isoWeekday == weeklyDay
		}
		if dayAllowed && local.Hour() == hour && local.Minute() == minute {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}, errors.New("could not find next Newsletter occurrence")
}

func normalizeRhythmInput(input RhythmInput) (RhythmInput, error) {
	switch input.Mode {
	case domain.RhythmEvidenceLed, domain.RhythmDaily,
		domain.RhythmSelectedWeekdays, domain.RhythmWeeklySynthesis:
	default:
		return RhythmInput{}, errors.New("rhythm mode is invalid")
	}
	if input.UnopenedLessonLimit < 1 || input.UnopenedLessonLimit > 20 {
		return RhythmInput{}, errors.New("unopened lesson limit must be from 1 to 20")
	}
	if len(input.SelectedWeekdays) == 0 {
		input.SelectedWeekdays = defaultSelectedWeekdays()
	}
	seen := make(map[int]bool, len(input.SelectedWeekdays))
	weekdays := make([]int, 0, len(input.SelectedWeekdays))
	for _, weekday := range input.SelectedWeekdays {
		if weekday < 1 || weekday > 7 {
			return RhythmInput{}, errors.New("selected weekdays must use ISO values 1 through 7")
		}
		if !seen[weekday] {
			weekdays = append(weekdays, weekday)
			seen[weekday] = true
		}
	}
	if len(weekdays) == 0 {
		return RhythmInput{}, errors.New("at least one weekday is required")
	}
	slices.Sort(weekdays)
	input.SelectedWeekdays = weekdays
	return input, nil
}

func defaultSelectedWeekdays() []int { return []int{1, 2, 3, 4, 5} }

func weekdaySmallInts(values []int) []int16 {
	result := make([]int16, len(values))
	for index, value := range values {
		result[index] = int16(value)
	}
	return result
}

func weekdayInts(values []int16) []int {
	result := make([]int, len(values))
	for index, value := range values {
		result[index] = int(value)
	}
	return result
}

func unopenedLessonCount(ctx context.Context, tx pgx.Tx, newsletterID string) (int, error) {
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int
		FROM issues i
		WHERE i.newsletter_id = $1
		  AND i.status = 'generated'
		  AND NOT EXISTS (
			SELECT 1
			FROM product_events event
			WHERE event.event_name = 'lesson_opened'
			  AND event.subject_type = 'lesson'
			  AND event.subject_id = i.id::text
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM lesson_backlog_dismissals dismissal
			WHERE dismissal.newsletter_id = i.newsletter_id
			  AND dismissal.issue_id = i.id
		  )
	`, newsletterID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unopened lessons: %w", err)
	}
	return count, nil
}

func rhythmPreferenceReason(mode domain.RhythmMode, weekdays []int) string {
	switch mode {
	case domain.RhythmEvidenceLed:
		return "Learnloom checks for meaningful new evidence before preparing another lesson."
	case domain.RhythmSelectedWeekdays:
		return fmt.Sprintf("Lessons are prepared on %d selected day(s) each week.", len(weekdays))
	case domain.RhythmWeeklySynthesis:
		return "One weekly synthesis connects the strongest learning signals."
	default:
		return "A focused lesson is prepared each day."
	}
}

func backlogRhythmReason(unopened, limit int) string {
	return fmt.Sprintf(
		"Rhythm slowed because %d lessons are waiting. Open one to restore your preferred rhythm (limit %d).",
		unopened,
		limit,
	)
}

func allocateNewsletterSlug(
	ctx context.Context,
	tx pgx.Tx,
	accountID, name string,
) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "newsletter"
	}
	for suffix := 1; suffix <= 1000; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}
		var available bool
		if err := tx.QueryRow(ctx, `
			SELECT NOT EXISTS (
				SELECT 1 FROM newsletters
				WHERE owner_account_id = $1 AND public_slug = $2
			)
		`, accountID, candidate).Scan(&available); err != nil {
			return "", err
		}
		if available {
			return candidate, nil
		}
	}
	return "", errors.New("could not allocate a unique Newsletter slug")
}

var (
	repeatedHyphens         = regexp.MustCompile(`-+`)
	streamTemplateIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,79}$`)
)

func slugify(value string) string {
	var output strings.Builder
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(char) || unicode.IsNumber(char):
			output.WriteRune(char)
		default:
			output.WriteByte('-')
		}
	}
	result := strings.Trim(repeatedHyphens.ReplaceAllString(output.String(), "-"), "-")
	runes := []rune(result)
	if len(runes) > 60 {
		result = strings.Trim(string(runes[:60]), "-")
	}
	return result
}

func boundedText(value, field string, maximum int) (string, error) {
	value = strings.TrimSpace(value)
	length := len([]rune(value))
	if length < 1 || length > maximum {
		return "", fmt.Errorf("%s must contain 1 to %d characters", field, maximum)
	}
	return value, nil
}
