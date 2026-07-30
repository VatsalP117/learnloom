package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ProductEventName string

const (
	ProductEventSignupCompleted       ProductEventName = "signup_completed"
	ProductEventStreamCreated         ProductEventName = "stream_created"
	ProductEventLessonGenerated       ProductEventName = "lesson_generated"
	ProductEventLessonOpened          ProductEventName = "lesson_opened"
	ProductEventLessonCompleted       ProductEventName = "lesson_completed"
	ProductEventReviewAttempted       ProductEventName = "review_attempted"
	ProductEventSearchIndexingEnabled ProductEventName = "search_indexing_enabled"
)

func (s *Store) RecordProductEvent(
	ctx context.Context,
	accountID string,
	name ProductEventName,
	subjectType, subjectID string,
	occurredAt time.Time,
) error {
	if accountID == "" || subjectID == "" || len(subjectID) > 128 {
		return errors.New("product event subject is invalid")
	}
	if !validProductEvent(name, subjectType) {
		return errors.New("product event name or subject type is invalid")
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
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
	return nil
}

func (s *Store) RecordOwnedLessonEvent(
	ctx context.Context,
	accountID, issueID string,
	name ProductEventName,
	occurredAt time.Time,
) error {
	if name != ProductEventLessonOpened && name != ProductEventReviewAttempted {
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

func validProductEvent(name ProductEventName, subjectType string) bool {
	return productEventSubjectType(name) == subjectType
}

func productEventSubjectType(name ProductEventName) string {
	switch name {
	case ProductEventSignupCompleted:
		return "account"
	case ProductEventStreamCreated:
		return "stream"
	case ProductEventLessonGenerated, ProductEventLessonOpened, ProductEventLessonCompleted:
		return "lesson"
	case ProductEventReviewAttempted:
		return "review"
	case ProductEventSearchIndexingEnabled:
		return "site"
	default:
		return ""
	}
}
