package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NewsletterRecord struct {
	domain.Newsletter
	IssueCount     int `json:"issueCount"`
	GeneratedCount int `json:"generatedCount"`
	SentCount      int `json:"sentCount"`
}

type NewsletterInput struct {
	Name                 string
	Topic                string
	LearnerLevel         string
	LearnerGoal          string
	LessonMinutes        int
	Sources              []domain.SourceDefinition
	ScheduleHour         int
	ScheduleMinute       int
	TimeZone             string
	Active               bool
	EmailEnabled         bool
	AIExplorationEnabled bool
	SiteVisible          bool
}

func (s *Store) CreateNewsletter(
	ctx context.Context,
	accountID string,
	input NewsletterInput,
	maximumPerAccount int,
) (NewsletterRecord, error) {
	normalized, err := normalizeNewsletterInput(input)
	if err != nil {
		return NewsletterRecord{}, err
	}
	if maximumPerAccount < 1 {
		maximumPerAccount = 10
	}
	now := time.Now().UTC()
	next, err := NextOccurrence(now, normalized.TimeZone, normalized.ScheduleHour, normalized.ScheduleMinute)
	if err != nil {
		return NewsletterRecord{}, err
	}
	sources, _ := json.Marshal(normalized.Sources)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return NewsletterRecord{}, err
	}
	defer rollback(tx)
	var status domain.AccountStatus
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT status FROM accounts WHERE id = $1 FOR UPDATE
	`, accountID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NewsletterRecord{}, ErrForbidden
		}
		return NewsletterRecord{}, fmt.Errorf("inspect Account Newsletter quota: %w", err)
	}
	if status != domain.AccountActive {
		return NewsletterRecord{}, ErrForbidden
	}
	if err := tx.QueryRow(
		ctx,
		"SELECT count(*) FROM newsletters WHERE owner_account_id = $1",
		accountID,
	).Scan(&count); err != nil {
		return NewsletterRecord{}, fmt.Errorf("count Account Newsletters: %w", err)
	}
	if count >= maximumPerAccount {
		return NewsletterRecord{}, ErrQuotaExceeded
	}
	publicSlug, err := allocateNewsletterSlug(ctx, tx, accountID, normalized.Name)
	if err != nil {
		return NewsletterRecord{}, err
	}
	id := uuid.New()
	row := tx.QueryRow(ctx, `
		INSERT INTO newsletters (
			id, owner_account_id, name, topic, learner_level, learner_goal,
			lesson_minutes, sources, schedule_hour, schedule_minute, time_zone,
			active, next_run_at, email_enabled, ai_exploration_enabled,
			public_slug, site_visible, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $18
		)
		RETURNING id::text, owner_account_id::text, name, topic, learner_level,
		          learner_goal, lesson_minutes, sources, schedule_hour,
		          schedule_minute, time_zone, active, next_run_at, email_enabled,
		          ai_exploration_enabled, public_slug, site_visible, created_at,
		          updated_at, 0, 0, 0
	`, id, accountID, normalized.Name, normalized.Topic, normalized.LearnerLevel,
		normalized.LearnerGoal, normalized.LessonMinutes, sources,
		normalized.ScheduleHour, normalized.ScheduleMinute, normalized.TimeZone,
		normalized.Active, next, normalized.EmailEnabled,
		normalized.AIExplorationEnabled, publicSlug, normalized.SiteVisible, now)
	record, err := scanNewsletterRecord(row)
	if err != nil {
		return NewsletterRecord{}, fmt.Errorf("create Newsletter: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return NewsletterRecord{}, fmt.Errorf("commit Newsletter: %w", err)
	}
	return record, nil
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
	next, err := NextOccurrence(
		time.Now().UTC(),
		normalized.TimeZone,
		normalized.ScheduleHour,
		normalized.ScheduleMinute,
	)
	if err != nil {
		return NewsletterRecord{}, err
	}
	sources, _ := json.Marshal(normalized.Sources)
	tag, err := s.pool.Exec(ctx, `
		UPDATE newsletters SET
			name = $3, topic = $4, learner_level = $5, learner_goal = $6,
			lesson_minutes = $7, sources = $8::jsonb, schedule_hour = $9,
			schedule_minute = $10, time_zone = $11, active = $12,
			next_run_at = $13, email_enabled = $14,
			ai_exploration_enabled = $15, site_visible = $16, updated_at = now()
		WHERE owner_account_id = $1 AND id = $2
	`, accountID, newsletterID, normalized.Name, normalized.Topic,
		normalized.LearnerLevel, normalized.LearnerGoal, normalized.LessonMinutes,
		sources, normalized.ScheduleHour, normalized.ScheduleMinute,
		normalized.TimeZone, normalized.Active, next, normalized.EmailEnabled,
		normalized.AIExplorationEnabled, normalized.SiteVisible)
	if err != nil {
		return NewsletterRecord{}, fmt.Errorf("update Newsletter: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return NewsletterRecord{}, ErrNotFound
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
		if err := tx.QueryRow(ctx, `
			SELECT time_zone, schedule_hour, schedule_minute
			FROM newsletters WHERE owner_account_id = $1 AND id = $2
		`, accountID, newsletterID).Scan(&zone, &hour, &minute); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		var err error
		next, err = NextOccurrence(now, zone, hour, minute)
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
	       n.learner_level, n.learner_goal, n.lesson_minutes, n.sources,
	       n.schedule_hour, n.schedule_minute, n.time_zone, n.active,
	       n.next_run_at, n.email_enabled, n.ai_exploration_enabled,
	       n.public_slug, n.site_visible, n.created_at, n.updated_at,
	       count(DISTINCT i.id)::int,
	       count(DISTINCT i.id) FILTER (WHERE i.status = 'generated')::int,
	       count(DISTINCT d.issue_id) FILTER (WHERE d.status = 'delivered')::int
	FROM newsletters n
	LEFT JOIN issues i ON i.newsletter_id = n.id
	LEFT JOIN delivery_receipts d ON d.issue_id = i.id
`

func scanNewsletterRecord(row scanner) (NewsletterRecord, error) {
	var record NewsletterRecord
	var rawSources []byte
	err := row.Scan(
		&record.ID,
		&record.OwnerAccountID,
		&record.Name,
		&record.Topic,
		&record.LearnerLevel,
		&record.LearnerGoal,
		&record.LessonMinutes,
		&rawSources,
		&record.ScheduleHour,
		&record.ScheduleMinute,
		&record.TimeZone,
		&record.Active,
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
	)
	if err != nil {
		return NewsletterRecord{}, err
	}
	if err := json.Unmarshal(rawSources, &record.Sources); err != nil {
		return NewsletterRecord{}, fmt.Errorf("decode Newsletter sources: %w", err)
	}
	return record, nil
}

func normalizeNewsletterInput(input NewsletterInput) (NewsletterInput, error) {
	var err error
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
	if input.ScheduleHour < 0 || input.ScheduleHour > 23 ||
		input.ScheduleMinute < 0 || input.ScheduleMinute > 59 {
		return NewsletterInput{}, errors.New("Newsletter schedule is invalid")
	}
	if _, err := time.LoadLocation(input.TimeZone); err != nil {
		return NewsletterInput{}, errors.New("Newsletter timezone is invalid")
	}
	if len(input.Sources) == 0 || len(input.Sources) > 12 {
		return NewsletterInput{}, errors.New("Newsletter requires 1 to 12 sources")
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

func NextOccurrence(after time.Time, zone string, hour, minute int) (time.Time, error) {
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, errors.New("Newsletter timezone is invalid")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, errors.New("Newsletter schedule is invalid")
	}
	candidate := after.UTC().Truncate(time.Minute).Add(time.Minute)
	for count := 0; count < 8*24*60; count++ {
		local := candidate.In(location)
		if local.Hour() == hour && local.Minute() == minute {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}, errors.New("could not find next Newsletter occurrence")
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

var repeatedHyphens = regexp.MustCompile(`-+`)

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
