package store

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type ProductEventName string

const (
	ProductEventSignupCompleted       ProductEventName = "signup_completed"
	ProductEventOnboardingStarted     ProductEventName = "onboarding_started"
	ProductEventOnboardingIntent      ProductEventName = "onboarding_intent_completed"
	ProductEventOnboardingSources     ProductEventName = "onboarding_sources_completed"
	ProductEventSourcePolicySelected  ProductEventName = "source_policy_selected"
	ProductEventSourcePreviewReached  ProductEventName = "source_preview_reached"
	ProductEventOnboardingConfirmed   ProductEventName = "onboarding_confirmed"
	ProductEventOnboardingAbandoned   ProductEventName = "onboarding_abandoned"
	ProductEventPreparationWaitExited ProductEventName = "preparation_wait_exited"
	ProductEventStreamCreated         ProductEventName = "stream_created"
	ProductEventLessonGenerated       ProductEventName = "lesson_generated"
	ProductEventLessonOpened          ProductEventName = "lesson_opened"
	ProductEventLessonCompleted       ProductEventName = "lesson_completed"
	ProductEventReviewAttempted       ProductEventName = "review_attempted"
	ProductEventFirstRetrieval        ProductEventName = "first_retrieval_completed"
	ProductEventActivationCompleted   ProductEventName = "activation_completed"
	ProductEventSearchIndexingEnabled ProductEventName = "search_indexing_enabled"
)

func (s *Store) RecordProductEvent(
	ctx context.Context,
	accountID string,
	name ProductEventName,
	subjectType, subjectID string,
	occurredAt time.Time,
) error {
	if accountID == "" || subjectID == "" || utf8.RuneCountInString(subjectID) > 128 {
		return errors.New("product event subject is invalid")
	}
	if !validProductEvent(name, subjectType) {
		return errors.New("product event name or subject type is invalid")
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	tag, err := tx.Exec(ctx, `
		INSERT INTO product_events (
			account_id, event_name, subject_type, subject_id, occurred_at
		)
		SELECT a.id, $2, $3, $4, $5
		FROM accounts a
		WHERE a.id = $1 AND a.status = 'active'
		ON CONFLICT (account_id, event_name, subject_id) DO NOTHING
	`, accountID, name, subjectType, subjectID, occurredAt)
	if err != nil {
		return fmt.Errorf("record product event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM product_events
			  WHERE account_id = $1 AND event_name = $2 AND subject_id = $3
			)
		`, accountID, name, subjectID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	if name == ProductEventActivationCompleted {
		if _, err := tx.Exec(ctx, `
			UPDATE public_attribution_conversions SET activated_at = COALESCE(activated_at, $2)
			WHERE converted_account_id = $1 AND converted_at <= $2
		`, accountID, occurredAt); err != nil {
			return fmt.Errorf("record public activation attribution: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func insertProductEvent(
	ctx context.Context,
	tx pgx.Tx,
	accountID string,
	name ProductEventName,
	subjectType, subjectID string,
	occurredAt time.Time,
) error {
	if accountID == "" || subjectID == "" || utf8.RuneCountInString(subjectID) > 128 ||
		!validProductEvent(name, subjectType) {
		return errors.New("product event subject is invalid")
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO product_events (
		  account_id, event_name, subject_type, subject_id, occurred_at
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id, event_name, subject_id) DO NOTHING
	`, accountID, name, subjectType, subjectID, occurredAt)
	if err != nil {
		return fmt.Errorf("record product event: %w", err)
	}
	if name == ProductEventActivationCompleted {
		if _, err := tx.Exec(ctx, `
			UPDATE public_attribution_conversions SET activated_at = COALESCE(activated_at, $2)
			WHERE converted_account_id = $1 AND converted_at <= $2
		`, accountID, occurredAt); err != nil {
			return fmt.Errorf("record public activation attribution: %w", err)
		}
	}
	return nil
}

func (s *Store) RecordOwnedLessonEvent(
	ctx context.Context,
	accountID, issueID string,
	name ProductEventName,
	occurredAt time.Time,
) error {
	if name != ProductEventLessonOpened {
		return errors.New("client lesson event is invalid")
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO product_events (
			account_id, event_name, subject_type, subject_id, occurred_at
		)
		SELECT n.owner_account_id, $3, $4, i.id::text, $5
		FROM issues i
		JOIN newsletters n ON n.id = i.newsletter_id
		WHERE n.owner_account_id = $1
		  AND i.id = $2
		  AND i.status = 'generated'
		ON CONFLICT (account_id, event_name, subject_id) DO NOTHING
	`, accountID, issueID, name, productEventSubjectType(name), occurredAt)
	if err != nil {
		return fmt.Errorf("record owned lesson event: %w", err)
	}
	return nil
}

func (s *Store) RecordPreparationWaitExited(
	ctx context.Context,
	accountID, newsletterID string,
	occurredAt time.Time,
) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO product_events (
		  account_id, event_name, subject_type, subject_id, occurred_at
		)
		SELECT n.owner_account_id, 'preparation_wait_exited', 'stream', n.id::text, $3
		FROM newsletters n
		WHERE n.id = $2 AND n.owner_account_id = $1
		  AND EXISTS (
		    SELECT 1 FROM issues i
		    WHERE i.newsletter_id = n.id
		      AND i.status IN ('queued', 'generating', 'awaiting_approval')
		  )
		ON CONFLICT (account_id, event_name, subject_id) DO NOTHING
	`, accountID, newsletterID, occurredAt)
	if err != nil {
		return fmt.Errorf("record preparation wait exit: %w", err)
	}
	return nil
}

func validProductEvent(name ProductEventName, subjectType string) bool {
	return productEventSubjectType(name) == subjectType
}

func productEventSubjectType(name ProductEventName) string {
	switch name {
	case ProductEventSignupCompleted:
		return "account"
	case ProductEventOnboardingStarted, ProductEventOnboardingIntent,
		ProductEventOnboardingSources, ProductEventSourcePolicySelected,
		ProductEventSourcePreviewReached, ProductEventOnboardingConfirmed,
		ProductEventOnboardingAbandoned:
		return "onboarding"
	case ProductEventStreamCreated, ProductEventPreparationWaitExited:
		return "stream"
	case ProductEventLessonGenerated, ProductEventLessonOpened, ProductEventLessonCompleted:
		return "lesson"
	case ProductEventReviewAttempted, ProductEventFirstRetrieval,
		ProductEventActivationCompleted:
		return "review"
	case ProductEventSearchIndexingEnabled:
		return "site"
	default:
		return ""
	}
}
