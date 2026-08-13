package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OnboardingDraftPayload struct {
	Name                 string                    `json:"name,omitempty"`
	Topic                string                    `json:"topic,omitempty"`
	LearnerLevel         string                    `json:"learnerLevel,omitempty"`
	LearnerGoal          string                    `json:"learnerGoal,omitempty"`
	LessonMinutes        int                       `json:"lessonMinutes,omitempty"`
	ScheduleTime         string                    `json:"scheduleTime,omitempty"`
	TimeZone             string                    `json:"timeZone,omitempty"`
	Active               bool                      `json:"active"`
	EmailEnabled         bool                      `json:"emailEnabled"`
	AIExplorationEnabled bool                      `json:"aiExplorationEnabled"`
	SourceMode           domain.SourceMode         `json:"sourceMode,omitempty"`
	SourceReviewMode     domain.SourceReviewMode   `json:"sourceReviewMode,omitempty"`
	Sources              []domain.SourceDefinition `json:"sources,omitempty"`
	TemplateID           string                    `json:"templateId,omitempty"`
	TemplateVersion      int                       `json:"templateVersion,omitempty"`
}

type OnboardingDraft struct {
	ID        string                 `json:"id"`
	Step      int                    `json:"step"`
	Revision  int64                  `json:"revision"`
	Payload   OnboardingDraftPayload `json:"payload"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

func (s *Store) GetOnboardingDraft(
	ctx context.Context,
	accountID string,
) (*OnboardingDraft, error) {
	var draft OnboardingDraft
	var raw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, step, revision, payload, updated_at
		FROM onboarding_drafts
		WHERE account_id = $1
	`, accountID).Scan(&draft.ID, &draft.Step, &draft.Revision, &raw, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get onboarding draft: %w", err)
	}
	if err := json.Unmarshal(raw, &draft.Payload); err != nil {
		return nil, fmt.Errorf("decode onboarding draft: %w", err)
	}
	return &draft, nil
}

func (s *Store) SaveOnboardingDraft(
	ctx context.Context,
	accountID, draftID string,
	expectedRevision int64,
	step int,
	payload OnboardingDraftPayload,
	now time.Time,
) (OnboardingDraft, error) {
	if _, err := uuid.Parse(draftID); err != nil {
		return OnboardingDraft{}, errors.New("onboarding draft ID is invalid")
	}
	if expectedRevision < 0 {
		return OnboardingDraft{}, errors.New("onboarding draft revision is invalid")
	}
	if err := validateOnboardingDraft(step, payload); err != nil {
		return OnboardingDraft{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return OnboardingDraft{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OnboardingDraft{}, err
	}
	defer rollback(tx)
	var draft OnboardingDraft
	if expectedRevision == 0 {
		err = tx.QueryRow(ctx, `
			INSERT INTO onboarding_drafts (
			  id, account_id, step, revision, payload, created_at, updated_at
			)
			SELECT $2::uuid, a.id, $3, 1, $4::jsonb, $5, $5
			FROM accounts a
			WHERE a.id = $1 AND a.status = 'active'
			  AND NOT EXISTS (
			    SELECT 1 FROM onboarding_draft_completions
			    WHERE draft_id = $2::uuid
			  )
			ON CONFLICT DO NOTHING
			RETURNING id::text, step, revision, updated_at
		`, accountID, draftID, step, raw, now).Scan(
			&draft.ID, &draft.Step, &draft.Revision, &draft.UpdatedAt,
		)
	} else {
		err = tx.QueryRow(ctx, `
			UPDATE onboarding_drafts SET
			  step = $4,
			  payload = $5::jsonb,
			  revision = revision + 1,
			  updated_at = $6
			WHERE account_id = $1 AND id = $2::uuid AND revision = $3
			  AND NOT EXISTS (
			    SELECT 1 FROM onboarding_draft_completions
			    WHERE draft_id = $2::uuid
			  )
			RETURNING id::text, step, revision, updated_at
		`, accountID, draftID, expectedRevision, step, raw, now).Scan(
			&draft.ID, &draft.Step, &draft.Revision, &draft.UpdatedAt,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return OnboardingDraft{}, ErrConflict
	}
	if err != nil {
		return OnboardingDraft{}, fmt.Errorf("save onboarding draft: %w", err)
	}
	draft.Payload = payload
	if err := recordOnboardingProgress(ctx, tx, accountID, draft, now); err != nil {
		return OnboardingDraft{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OnboardingDraft{}, fmt.Errorf("commit onboarding draft: %w", err)
	}
	return draft, nil
}

func (s *Store) DeleteOnboardingDraft(
	ctx context.Context,
	accountID, draftID string,
	expectedRevision int64,
	abandoned bool,
	now time.Time,
) error {
	if _, err := uuid.Parse(draftID); err != nil {
		return errors.New("onboarding draft ID is invalid")
	}
	if expectedRevision < 1 {
		return errors.New("onboarding draft revision is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var deletedID string
	err = tx.QueryRow(ctx, `
		DELETE FROM onboarding_drafts
		WHERE account_id = $1 AND id = $2::uuid AND revision = $3
		RETURNING id::text
	`, accountID, draftID, expectedRevision).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("delete onboarding draft: %w", err)
	}
	if abandoned && deletedID != "" {
		if err := insertProductEvent(
			ctx, tx, accountID, ProductEventOnboardingAbandoned,
			"onboarding", deletedID, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) RecordOnboardingPreviewReached(
	ctx context.Context,
	accountID, draftID string,
	now time.Time,
) error {
	if _, err := uuid.Parse(draftID); err != nil {
		return errors.New("onboarding draft ID is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO product_events (
		  account_id, event_name, subject_type, subject_id, occurred_at
		)
		SELECT d.account_id, 'source_preview_reached', 'onboarding', d.id::text, $3
		FROM onboarding_drafts d
		WHERE d.account_id = $1 AND d.id = $2::uuid
		ON CONFLICT (account_id, event_name, subject_id) DO NOTHING
	`, accountID, draftID, now)
	if err != nil {
		return fmt.Errorf("record onboarding preview: %w", err)
	}
	return nil
}

func recordOnboardingProgress(
	ctx context.Context,
	tx pgx.Tx,
	accountID string,
	draft OnboardingDraft,
	now time.Time,
) error {
	events := []struct {
		name      ProductEventName
		subjectID string
	}{
		{ProductEventOnboardingStarted, draft.ID},
	}
	if draft.Step >= 2 {
		events = append(events,
			struct {
				name      ProductEventName
				subjectID string
			}{ProductEventOnboardingIntent, draft.ID},
		)
		if draft.Payload.SourceMode != "" {
			events = append(events, struct {
				name      ProductEventName
				subjectID string
			}{ProductEventSourcePolicySelected, draft.ID + ":" + string(draft.Payload.SourceMode)})
		}
	}
	if draft.Step >= 3 {
		events = append(events,
			struct {
				name      ProductEventName
				subjectID string
			}{ProductEventOnboardingSources, draft.ID},
		)
	}
	for _, event := range events {
		if strings.HasSuffix(event.subjectID, ":") {
			continue
		}
		if err := insertProductEvent(
			ctx, tx, accountID, event.name, "onboarding", event.subjectID, now,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateOnboardingDraft(step int, payload OnboardingDraftPayload) error {
	if step < 1 || step > 3 {
		return errors.New("onboarding step must be from 1 to 3")
	}
	for _, field := range []struct {
		name  string
		value string
		limit int
	}{
		{"name", payload.Name, 80},
		{"topic", payload.Topic, 400},
		{"learner goal", payload.LearnerGoal, 500},
		{"time zone", payload.TimeZone, 120},
		{"template ID", payload.TemplateID, 80},
	} {
		if utf8.RuneCountInString(field.value) > field.limit {
			return fmt.Errorf("%s is too long", field.name)
		}
	}
	if payload.LearnerLevel != "" && payload.LearnerLevel != "beginner" &&
		payload.LearnerLevel != "intermediate" && payload.LearnerLevel != "advanced" {
		return errors.New("learner level is invalid")
	}
	if payload.LessonMinutes != 0 && (payload.LessonMinutes < 5 || payload.LessonMinutes > 90) {
		return errors.New("lesson minutes must be from 5 to 90")
	}
	if payload.ScheduleTime != "" {
		if _, err := time.Parse("15:04", payload.ScheduleTime); err != nil {
			return errors.New("schedule time must use HH:MM")
		}
	}
	if payload.SourceMode != "" && payload.SourceMode != domain.SourceModeDiscovered &&
		payload.SourceMode != domain.SourceModeProvided && payload.SourceMode != domain.SourceModeHybrid {
		return errors.New("source mode is invalid")
	}
	if payload.SourceReviewMode != "" && payload.SourceReviewMode != domain.SourceReviewAuto &&
		payload.SourceReviewMode != domain.SourceReviewBeforeLesson {
		return errors.New("source review mode is invalid")
	}
	if len(payload.Sources) > 12 {
		return errors.New("onboarding supports at most 12 sources")
	}
	for index, item := range payload.Sources {
		if utf8.RuneCountInString(item.Name) > 120 {
			return fmt.Errorf("source %d name is too long", index+1)
		}
		if item.URL == "" {
			continue
		}
		if utf8.RuneCountInString(item.URL) > 2048 {
			return fmt.Errorf("source %d URL is too long", index+1)
		}
		parsed, err := url.Parse(strings.TrimSpace(item.URL))
		if err != nil || parsed.Host == "" || parsed.User != nil ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("source %d URL is invalid", index+1)
		}
		host := strings.ToLower(parsed.Hostname())
		if host == "localhost" || strings.HasSuffix(host, ".localhost") {
			return fmt.Errorf("source %d URL must use a public host", index+1)
		}
		if address, parseErr := netip.ParseAddr(host); parseErr == nil &&
			(!address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
				address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified()) {
			return fmt.Errorf("source %d URL must use a public host", index+1)
		}
		if item.Limit != 0 && (item.Limit < 1 || item.Limit > 50) {
			return fmt.Errorf("source %d limit must be from 1 to 50", index+1)
		}
	}
	return nil
}
