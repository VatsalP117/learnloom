package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/jackc/pgx/v5"
)

type LearnerConceptState struct {
	NewsletterID       string     `json:"newsletterId"`
	Key                string     `json:"key"`
	Label              string     `json:"label"`
	Role               string     `json:"role"`
	ExposureCount      int        `json:"exposureCount"`
	CompletedCount     int        `json:"completedCount"`
	ReviewAttemptCount int        `json:"reviewAttemptCount"`
	ConfidenceScore    int        `json:"confidenceScore"`
	LastSeenAt         time.Time  `json:"lastSeenAt"`
	LastCompletedAt    *time.Time `json:"lastCompletedAt,omitempty"`
	LastReviewedAt     *time.Time `json:"lastReviewedAt,omitempty"`
}

type CurriculumProjection struct {
	Outcome               string                 `json:"outcome"`
	Concepts              []LearnerConceptState  `json:"concepts"`
	Milestones            []CapabilityMilestone  `json:"milestones"`
	CurrentGaps           []CurriculumGap        `json:"currentGaps"`
	Recall                CurriculumRecall       `json:"recall"`
	SuggestedNextConcepts []string               `json:"suggestedNextConcepts"`
	Timeline              []ConceptTimelineEntry `json:"timeline"`
}

type CapabilityMilestone struct {
	Key                string `json:"key"`
	Label              string `json:"label"`
	Statement          string `json:"statement"`
	Stage              string `json:"stage"`
	CompletedCount     int    `json:"completedCount"`
	ReviewAttemptCount int    `json:"reviewAttemptCount"`
	ConfidenceScore    int    `json:"confidenceScore"`
}

type CurriculumGap struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

type CurriculumRecall struct {
	DueCount          int    `json:"dueCount"`
	PracticedConcepts int    `json:"practicedConcepts"`
	SolidConcepts     int    `json:"solidConcepts"`
	Summary           string `json:"summary"`
}

type ConceptTimelineEntry struct {
	IssueID     string    `json:"issueId"`
	Title       string    `json:"title"`
	CompletedAt time.Time `json:"completedAt"`
	Concepts    []string  `json:"concepts"`
}

func (s *Store) ListLearnerConcepts(
	ctx context.Context,
	accountID, newsletterID string,
	limit int,
) ([]LearnerConceptState, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT state.newsletter_id::text, state.concept_key, state.label,
		       state.role, state.exposure_count, state.completed_count,
		       state.review_attempt_count, state.confidence_score,
		       state.last_seen_at, state.last_completed_at,
		       state.last_reviewed_at
		FROM learner_concept_state state
		JOIN newsletters newsletter ON newsletter.id = state.newsletter_id
		WHERE state.account_id = $1
		  AND state.newsletter_id = $2
		  AND newsletter.owner_account_id = $1
		ORDER BY
		  CASE state.role WHEN 'core' THEN 0 ELSE 1 END,
		  state.last_seen_at DESC,
		  state.label
		LIMIT $3
	`, accountID, newsletterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list Learner Concepts: %w", err)
	}
	defer rows.Close()
	concepts := make([]LearnerConceptState, 0)
	for rows.Next() {
		var concept LearnerConceptState
		if err := rows.Scan(
			&concept.NewsletterID,
			&concept.Key,
			&concept.Label,
			&concept.Role,
			&concept.ExposureCount,
			&concept.CompletedCount,
			&concept.ReviewAttemptCount,
			&concept.ConfidenceScore,
			&concept.LastSeenAt,
			&concept.LastCompletedAt,
			&concept.LastReviewedAt,
		); err != nil {
			return nil, err
		}
		concepts = append(concepts, concept)
	}
	return concepts, rows.Err()
}

func (s *Store) LoadLearnerState(
	ctx context.Context,
	accountID, newsletterID string,
	limit int,
) (domain.LearnerState, error) {
	concepts, err := s.ListLearnerConcepts(ctx, accountID, newsletterID, limit)
	if err != nil {
		return domain.LearnerState{}, err
	}
	state := domain.LearnerState{
		Concepts: make([]domain.LearnerConceptProgress, 0, len(concepts)),
	}
	for _, concept := range concepts {
		state.Concepts = append(state.Concepts, domain.LearnerConceptProgress{
			Label:              concept.Label,
			Role:               concept.Role,
			ExposureCount:      concept.ExposureCount,
			CompletedCount:     concept.CompletedCount,
			ReviewAttemptCount: concept.ReviewAttemptCount,
			ConfidenceScore:    concept.ConfidenceScore,
		})
	}

	var difficulty, relevance, recallConfidence *string
	err = s.pool.QueryRow(ctx, `
		SELECT feedback.difficulty, feedback.relevance,
		       feedback.recall_confidence
		FROM lesson_feedback feedback
		JOIN issues issue ON issue.id = feedback.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE feedback.account_id = $1
		  AND issue.newsletter_id = $2
		  AND newsletter.owner_account_id = $1
		ORDER BY feedback.updated_at DESC
		LIMIT 1
	`, accountID, newsletterID).Scan(
		&difficulty,
		&relevance,
		&recallConfidence,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.LearnerState{}, fmt.Errorf("load Learner feedback: %w", err)
	}
	if err == nil {
		state.Difficulty = stringValue(difficulty)
		state.Relevance = stringValue(relevance)
		state.RecallConfidence = stringValue(recallConfidence)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT note.body
		FROM lesson_notes note
		JOIN issues issue ON issue.id = note.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE note.account_id = $1
		  AND issue.newsletter_id = $2
		  AND newsletter.owner_account_id = $1
		  AND note.kind = 'question'
		  AND note.anchor_type = 'claim'
		ORDER BY note.updated_at DESC, note.id DESC
		LIMIT 3
	`, accountID, newsletterID)
	if err != nil {
		return domain.LearnerState{}, fmt.Errorf("load Learner claim questions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var question string
		if err := rows.Scan(&question); err != nil {
			return domain.LearnerState{}, err
		}
		state.OpenQuestions = append(state.OpenQuestions, question)
	}
	if err := rows.Err(); err != nil {
		return domain.LearnerState{}, err
	}
	return state, nil
}

func (s *Store) GetCurriculum(
	ctx context.Context,
	accountID, newsletterID string,
) (CurriculumProjection, error) {
	concepts, err := s.ListLearnerConcepts(ctx, accountID, newsletterID, 100)
	if err != nil {
		return CurriculumProjection{}, err
	}
	var outcome string
	var dueCount int
	err = s.pool.QueryRow(ctx, `
		SELECT newsletter.learner_goal,
		       count(DISTINCT review.id) FILTER (
		         WHERE review.due_at IS NOT NULL
		           AND review.retired_at IS NULL
		           AND review.due_at <= now()
		       )::int
		FROM newsletters newsletter
		LEFT JOIN issues issue ON issue.newsletter_id = newsletter.id
		LEFT JOIN review_items review ON review.issue_id = issue.id
		WHERE newsletter.id = $1 AND newsletter.owner_account_id = $2
		GROUP BY newsletter.id
	`, newsletterID, accountID).Scan(&outcome, &dueCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return CurriculumProjection{}, ErrNotFound
	}
	if err != nil {
		return CurriculumProjection{}, fmt.Errorf("load Curriculum outcome: %w", err)
	}
	milestones, gaps, recall := projectCapabilities(concepts, dueCount)
	timeline, err := s.listConceptTimeline(ctx, accountID, newsletterID, 12)
	if err != nil {
		return CurriculumProjection{}, err
	}
	var raw []byte
	err = s.pool.QueryRow(ctx, `
		SELECT history.entry
		FROM learning_history history
		JOIN newsletters newsletter ON newsletter.id = history.newsletter_id
		WHERE history.newsletter_id = $1
		  AND newsletter.owner_account_id = $2
		ORDER BY history.created_at DESC
		LIMIT 1
	`, newsletterID, accountID).Scan(&raw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return CurriculumProjection{
				Outcome: outcome, Concepts: concepts, Milestones: milestones,
				CurrentGaps: gaps, Recall: recall, Timeline: timeline,
			}, nil
		}
		return CurriculumProjection{}, fmt.Errorf("load Curriculum direction: %w", err)
	}
	var history domain.LearningHistoryEntry
	if err := json.Unmarshal(raw, &history); err != nil {
		return CurriculumProjection{}, fmt.Errorf("decode Curriculum direction: %w", err)
	}
	return CurriculumProjection{
		Outcome:               outcome,
		Concepts:              concepts,
		Milestones:            milestones,
		CurrentGaps:           gaps,
		Recall:                recall,
		SuggestedNextConcepts: history.SuggestedNextConcepts,
		Timeline:              timeline,
	}, nil
}

func projectCapabilities(
	concepts []LearnerConceptState,
	dueCount int,
) ([]CapabilityMilestone, []CurriculumGap, CurriculumRecall) {
	milestones := make([]CapabilityMilestone, 0, len(concepts))
	gaps := make([]CurriculumGap, 0, len(concepts))
	recall := CurriculumRecall{DueCount: dueCount}
	for _, concept := range concepts {
		if concept.CompletedCount == 0 {
			gaps = append(gaps, CurriculumGap{
				Key: concept.Key, Label: concept.Label,
				Reason: "Not yet completed in a lesson",
			})
			continue
		}
		stage := "explained"
		statement := "Can explain the core idea behind " + concept.Label
		if concept.ReviewAttemptCount > 0 {
			recall.PracticedConcepts++
			stage = "retrieved"
			statement = "Can retrieve " + concept.Label + " with some support"
		}
		if concept.ReviewAttemptCount > 0 && concept.ConfidenceScore >= 75 {
			recall.SolidConcepts++
			stage = "recalled_solidly"
			statement = "Can recall and explain " + concept.Label
		}
		milestones = append(milestones, CapabilityMilestone{
			Key: concept.Key, Label: concept.Label, Statement: statement,
			Stage: stage, CompletedCount: concept.CompletedCount,
			ReviewAttemptCount: concept.ReviewAttemptCount,
			ConfidenceScore:    concept.ConfidenceScore,
		})
		if concept.ReviewAttemptCount == 0 {
			gaps = append(gaps, CurriculumGap{
				Key: concept.Key, Label: concept.Label,
				Reason: "Ready for a first recall",
			})
		} else if concept.ConfidenceScore < 60 {
			gaps = append(gaps, CurriculumGap{
				Key: concept.Key, Label: concept.Label,
				Reason: "Needs another retrieval",
			})
		}
	}
	switch {
	case dueCount > 0:
		recall.Summary = fmt.Sprintf("%d retrieval prompt%s due now", dueCount, pluralSuffix(dueCount))
	case recall.PracticedConcepts == 0:
		recall.Summary = "Complete an in-lesson retrieval to begin strengthening recall"
	case recall.SolidConcepts > 0:
		recall.Summary = fmt.Sprintf("%d concept%s recalled solidly", recall.SolidConcepts, pluralSuffix(recall.SolidConcepts))
	default:
		recall.Summary = "Recall is developing through spaced retrieval"
	}
	return milestones, gaps, recall
}

func (s *Store) listConceptTimeline(
	ctx context.Context,
	accountID, newsletterID string,
	limit int,
) ([]ConceptTimelineEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT issue.id::text, COALESCE(issue.dossier_title, ''),
		       progress.completed_at,
		       array_agg(concept.label ORDER BY concept.role, concept.label)
		FROM lesson_progress progress
		JOIN issues issue ON issue.id = progress.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		JOIN issue_concepts concept ON concept.issue_id = issue.id
		WHERE progress.account_id = $1
		  AND issue.newsletter_id = $2
		  AND newsletter.owner_account_id = $1
		  AND progress.completed_at IS NOT NULL
		GROUP BY issue.id, progress.completed_at
		ORDER BY progress.completed_at DESC, issue.id
		LIMIT $3
	`, accountID, newsletterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list Concept timeline: %w", err)
	}
	defer rows.Close()
	timeline := make([]ConceptTimelineEntry, 0)
	for rows.Next() {
		var entry ConceptTimelineEntry
		if err := rows.Scan(
			&entry.IssueID,
			&entry.Title,
			&entry.CompletedAt,
			&entry.Concepts,
		); err != nil {
			return nil, err
		}
		timeline = append(timeline, entry)
	}
	return timeline, rows.Err()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
