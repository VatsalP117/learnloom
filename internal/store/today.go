package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type TodayFocus struct {
	Kind           string         `json:"kind"`
	SubjectID      string         `json:"subjectId"`
	NewsletterID   string         `json:"newsletterId,omitempty"`
	Title          string         `json:"title,omitempty"`
	NewsletterName string         `json:"newsletterName,omitempty"`
	LessonMinutes  int            `json:"lessonMinutes,omitempty"`
	Progress       int            `json:"progress,omitempty"`
	DueCount       int            `json:"dueCount,omitempty"`
	ReasonCode     string         `json:"reasonCode"`
	Reason         string         `json:"reason"`
	ActionLabel    string         `json:"actionLabel"`
	ActionURL      string         `json:"actionUrl"`
	Score          int            `json:"score"`
	Components     map[string]int `json:"components"`
	SelectedAt     time.Time      `json:"selectedAt"`
}

type todayLessonCandidate struct {
	IssueID              string
	NewsletterID         string
	Title                string
	NewsletterName       string
	LearnerGoal          string
	LessonMinutes        int
	Progress             int
	CreatedAt            time.Time
	LastProgressAt       *time.Time
	LastPathCompletionAt *time.Time
	Relevance            string
	Prerequisites        int
	SatisfiedPrereqs     int
	EvidenceSources      int
}

func (s *Store) RefreshTodayFocus(
	ctx context.Context,
	accountID string,
	now time.Time,
) (TodayFocus, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	lessons, err := s.todayLessonCandidates(ctx, accountID)
	if err != nil {
		return TodayFocus{}, err
	}
	retention, err := s.GetRetentionState(ctx, accountID, now)
	if err != nil {
		return TodayFocus{}, err
	}
	focuses := make([]TodayFocus, 0, len(lessons)+2)
	for _, lesson := range lessons {
		focuses = append(focuses, scoreTodayLesson(lesson, now))
	}
	if review, err := s.todayReviewCandidate(ctx, accountID, now); err != nil {
		return TodayFocus{}, err
	} else if review != nil {
		focuses = append(focuses, *review)
	}
	if retention.Inactive {
		focuses = append(focuses, TodayFocus{
			Kind:           "reentry",
			SubjectID:      "reentry",
			NewsletterID:   retention.ReentryNewsletterID,
			NewsletterName: retention.ReentryNewsletterName,
			ReasonCode:     "gentle_reentry",
			Reason:         fmt.Sprintf("You have been away for %d days. One small step is enough; there is no backlog to clear.", retention.DaysAway),
			ActionLabel:    firstTextStore(retention.ActionLabel, "Choose one useful step"),
			ActionURL:      firstTextStore(retention.ActionURL, "/streams"),
			Score:          260,
			Components:     map[string]int{"reentry": 260},
		})
	}
	if len(focuses) == 0 {
		focuses = append(focuses, TodayFocus{
			Kind:        "clear",
			SubjectID:   "clear",
			ReasonCode:  "queue_clear",
			Reason:      "There is no urgent learning step right now. Your completed work remains in the library.",
			ActionLabel: "Open your library",
			ActionURL:   "/library",
			Score:       0,
			Components:  map[string]int{"clear": 0},
		})
	}
	sort.SliceStable(focuses, func(i, j int) bool {
		if focuses[i].Score == focuses[j].Score {
			if focuses[i].Kind == focuses[j].Kind {
				return focuses[i].SubjectID < focuses[j].SubjectID
			}
			return todayKindOrder(focuses[i].Kind) < todayKindOrder(focuses[j].Kind)
		}
		return focuses[i].Score > focuses[j].Score
	})
	selected := focuses[0]
	if err := s.persistTodayFocus(ctx, accountID, &selected, now); err != nil {
		return TodayFocus{}, err
	}
	return selected, nil
}

func (s *Store) todayLessonCandidates(
	ctx context.Context,
	accountID string,
) ([]todayLessonCandidate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT issue.id::text, newsletter.id::text,
		       COALESCE(issue.dossier_title, ''), newsletter.name,
		       newsletter.learner_goal, newsletter.lesson_minutes,
		       COALESCE(progress.progress, 0), issue.created_at,
		       progress.updated_at,
		       path_completion.completed_at,
		       COALESCE(recent_feedback.relevance, ''),
		       COALESCE(prerequisite.total, 0),
		       COALESCE(prerequisite.satisfied, 0),
		       COALESCE(evidence.source_count, 0)
		FROM issues issue
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		LEFT JOIN lesson_progress progress
		  ON progress.account_id = newsletter.owner_account_id
		 AND progress.issue_id = issue.id
		LEFT JOIN LATERAL (
		  SELECT max(path_progress.completed_at) AS completed_at
		  FROM lesson_progress path_progress
		  JOIN issues path_issue ON path_issue.id = path_progress.issue_id
		  WHERE path_progress.account_id = newsletter.owner_account_id
		    AND path_issue.newsletter_id = newsletter.id
		    AND path_progress.completed_at IS NOT NULL
		) path_completion ON true
		LEFT JOIN LATERAL (
		  SELECT feedback.relevance
		  FROM lesson_feedback feedback
		  JOIN issues feedback_issue ON feedback_issue.id = feedback.issue_id
		  WHERE feedback.account_id = newsletter.owner_account_id
		    AND feedback_issue.newsletter_id = newsletter.id
		    AND feedback.relevance IS NOT NULL
		  ORDER BY feedback.updated_at DESC
		  LIMIT 1
		) recent_feedback ON true
		LEFT JOIN LATERAL (
		  SELECT count(*)::int AS total,
		         count(*) FILTER (
		           WHERE state.completed_count > 0 OR state.confidence_score >= 60
		         )::int AS satisfied
		  FROM issue_concepts concept
		  LEFT JOIN learner_concept_state state
		    ON state.account_id = concept.account_id
		   AND state.newsletter_id = concept.newsletter_id
		   AND state.concept_key = concept.concept_key
		  WHERE concept.issue_id = issue.id
		    AND concept.role = 'prerequisite'
		) prerequisite ON true
		LEFT JOIN LATERAL (
		  SELECT count(*)::int AS source_count
		  FROM issue_sources evidence_link
		  WHERE evidence_link.issue_id = issue.id
		) evidence ON true
		WHERE newsletter.owner_account_id = $1
		  AND (newsletter.active OR COALESCE(progress.progress, 0) > 0)
		  AND issue.status = 'generated'
		  AND progress.completed_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM lesson_backlog_dismissals dismissal
			WHERE dismissal.account_id = $1 AND dismissal.issue_id = issue.id
		  )
		ORDER BY issue.created_at, issue.id
		LIMIT 100
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list Today lesson candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]todayLessonCandidate, 0)
	for rows.Next() {
		var candidate todayLessonCandidate
		if err := rows.Scan(
			&candidate.IssueID,
			&candidate.NewsletterID,
			&candidate.Title,
			&candidate.NewsletterName,
			&candidate.LearnerGoal,
			&candidate.LessonMinutes,
			&candidate.Progress,
			&candidate.CreatedAt,
			&candidate.LastProgressAt,
			&candidate.LastPathCompletionAt,
			&candidate.Relevance,
			&candidate.Prerequisites,
			&candidate.SatisfiedPrereqs,
			&candidate.EvidenceSources,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func scoreTodayLesson(candidate todayLessonCandidate, now time.Time) TodayFocus {
	components := map[string]int{}
	if candidate.Progress > 0 && candidate.Progress < 100 {
		components["inProgress"] = 210
	}
	since := candidate.CreatedAt
	if candidate.LastProgressAt != nil {
		since = *candidate.LastProgressAt
	}
	components["neglected"] = min(40, max(0, int(now.Sub(since).Hours()/24))*2)
	switch candidate.Relevance {
	case "very_relevant":
		components["goalRelevance"] = 20
	case "somewhat_relevant":
		components["goalRelevance"] = 6
	case "not_relevant":
		components["goalRelevance"] = -40
	default:
		components["goalRelevance"] = 8
	}
	if candidate.Prerequisites == 0 {
		components["prerequisites"] = 6
	} else {
		components["prerequisites"] =
			(candidate.SatisfiedPrereqs*20)/candidate.Prerequisites - 10
	}
	components["evidenceStrength"] = min(20, candidate.EvidenceSources*4)
	if candidate.EvidenceSources == 0 {
		components["evidenceStrength"] = -30
	}
	if candidate.LessonMinutes <= 15 {
		components["timeFit"] = 12
	} else if candidate.LessonMinutes <= 25 {
		components["timeFit"] = 6
	}
	if candidate.LastPathCompletionAt == nil {
		components["pathNeglect"] = min(
			30, max(0, int(now.Sub(candidate.CreatedAt).Hours()/24)),
		)
	} else {
		components["pathNeglect"] = min(
			30, max(0, int(now.Sub(*candidate.LastPathCompletionAt).Hours()/24)),
		)
	}
	score := 100
	for _, value := range components {
		score += value
	}
	reasonCode := "goal_progress"
	reason := fmt.Sprintf(
		"This source-grounded lesson fits your %d-minute setting and is aligned with your goal: %s",
		candidate.LessonMinutes,
		terminalSentence(candidate.LearnerGoal),
	)
	if candidate.Progress > 0 {
		reasonCode = "continue_in_progress"
		reason = fmt.Sprintf(
			"Continue from your saved place at %d%% while this model is still fresh.",
			candidate.Progress,
		)
	} else if components["pathNeglect"] >= 14 {
		reasonCode = "return_to_neglected_path"
		reason = "This path has waited longer than your other active work. Returning now preserves continuity without creating a backlog."
	} else if candidate.Relevance == "very_relevant" {
		reasonCode = "high_relevance"
		reason = "You marked this path highly relevant, and its prerequisites and evidence are ready."
	}
	return TodayFocus{
		Kind:           "lesson",
		SubjectID:      candidate.IssueID,
		NewsletterID:   candidate.NewsletterID,
		Title:          candidate.Title,
		NewsletterName: candidate.NewsletterName,
		LessonMinutes:  candidate.LessonMinutes,
		Progress:       candidate.Progress,
		ReasonCode:     reasonCode,
		Reason:         reason,
		ActionLabel:    map[bool]string{true: "Resume lesson", false: "Begin lesson"}[candidate.Progress > 0],
		ActionURL:      "/issues/" + candidate.IssueID,
		Score:          score,
		Components:     components,
	}
}

func (s *Store) todayReviewCandidate(
	ctx context.Context,
	accountID string,
	now time.Time,
) (*TodayFocus, error) {
	var reviewID string
	var dueAt time.Time
	var dueCount int
	err := s.pool.QueryRow(ctx, `
		SELECT review.id::text, review.due_at, count(*) OVER ()::int
		FROM review_items review
		JOIN issues issue ON issue.id = review.issue_id
		JOIN newsletters newsletter ON newsletter.id = issue.newsletter_id
		WHERE review.account_id = $1
		  AND newsletter.owner_account_id = $1
		  AND review.due_at <= $2
		  AND review.retired_at IS NULL
		ORDER BY review.due_at, review.id
		LIMIT 1
	`, accountID, now).Scan(&reviewID, &dueAt, &dueCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get Today Review candidate: %w", err)
	}
	daysOverdue := max(0, int(now.Sub(dueAt).Hours()/24))
	urgency := 145 + min(50, daysOverdue*5) + min(20, dueCount*2)
	return &TodayFocus{
		Kind:        "review",
		SubjectID:   reviewID,
		DueCount:    dueCount,
		ReasonCode:  "retrieval_due",
		Reason:      fmt.Sprintf("%d retrieval prompt%s %s due. Recalling before reading something new protects continuity.", dueCount, pluralSuffix(dueCount), map[bool]string{true: "are", false: "is"}[dueCount != 1]),
		ActionLabel: "Start review",
		ActionURL:   "/review",
		Score:       urgency,
		Components:  map[string]int{"reviewUrgency": urgency},
	}, nil
}

func (s *Store) persistTodayFocus(
	ctx context.Context,
	accountID string,
	focus *TodayFocus,
	now time.Time,
) error {
	components, err := json.Marshal(focus.Components)
	if err != nil {
		return err
	}
	var newsletterID any
	if focus.NewsletterID != "" {
		newsletterID = focus.NewsletterID
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO today_focus_selections (
		  account_id, kind, subject_id, newsletter_id, reason_code,
		  reason_text, score, score_components, selected_at, updated_at
		)
		SELECT account.id, $2, $3, $4::uuid, $5, $6, $7, $8::jsonb, $9, $9
		FROM accounts account
		WHERE account.id = $1 AND account.status = 'active'
		ON CONFLICT (account_id) DO UPDATE SET
		  kind = EXCLUDED.kind,
		  subject_id = EXCLUDED.subject_id,
		  newsletter_id = EXCLUDED.newsletter_id,
		  reason_code = EXCLUDED.reason_code,
		  reason_text = EXCLUDED.reason_text,
		  score = EXCLUDED.score,
		  score_components = EXCLUDED.score_components,
		  selected_at = CASE
		    WHEN (today_focus_selections.kind, today_focus_selections.subject_id)
		      IS DISTINCT FROM (EXCLUDED.kind, EXCLUDED.subject_id)
		    THEN EXCLUDED.selected_at
		    ELSE today_focus_selections.selected_at
		  END,
		  updated_at = EXCLUDED.updated_at
		RETURNING selected_at
	`, accountID, focus.Kind, focus.SubjectID, newsletterID,
		focus.ReasonCode, focus.Reason, focus.Score, components, now).Scan(&focus.SelectedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return fmt.Errorf("persist Today focus: %w", err)
	}
	return nil
}

func todayKindOrder(kind string) int {
	switch kind {
	case "lesson":
		return 0
	case "review":
		return 1
	case "reentry":
		return 2
	default:
		return 3
	}
}

func terminalSentence(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "your learning goal."
	}
	if !strings.HasSuffix(value, ".") && !strings.HasSuffix(value, "!") &&
		!strings.HasSuffix(value, "?") {
		value += "."
	}
	return value
}

func firstTextStore(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
