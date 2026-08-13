package store

import (
	"testing"
	"time"
)

func TestScoreTodayLessonPrioritizesSavedProgress(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	ready := todayLessonCandidate{
		IssueID: "ready", NewsletterID: "stream", Title: "Ready",
		LearnerGoal: "Understand the mechanism", LessonMinutes: 12,
		CreatedAt: now.Add(-48 * time.Hour), EvidenceSources: 4,
		Prerequisites: 2, SatisfiedPrereqs: 2,
	}
	inProgress := ready
	inProgress.IssueID = "in-progress"
	inProgress.Progress = 38
	lastProgress := now.Add(-time.Hour)
	inProgress.LastProgressAt = &lastProgress

	readyFocus := scoreTodayLesson(ready, now)
	inProgressFocus := scoreTodayLesson(inProgress, now)
	if inProgressFocus.Score <= readyFocus.Score ||
		inProgressFocus.ReasonCode != "continue_in_progress" ||
		inProgressFocus.Progress != 38 {
		t.Fatalf("ready=%#v inProgress=%#v", readyFocus, inProgressFocus)
	}
}

func TestScoreTodayLessonUsesEvidencePrerequisitesRelevanceAndNeglect(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	strong := scoreTodayLesson(todayLessonCandidate{
		IssueID: "strong", NewsletterID: "stream", Title: "Strong",
		LearnerGoal: "Apply the model", LessonMinutes: 12,
		CreatedAt: now.Add(-20 * 24 * time.Hour), Relevance: "very_relevant",
		Prerequisites: 2, SatisfiedPrereqs: 2, EvidenceSources: 5,
	}, now)
	weak := scoreTodayLesson(todayLessonCandidate{
		IssueID: "weak", NewsletterID: "stream", Title: "Weak",
		LearnerGoal: "Apply the model", LessonMinutes: 30,
		CreatedAt: now, Relevance: "not_relevant",
		Prerequisites: 2, SatisfiedPrereqs: 0, EvidenceSources: 0,
	}, now)
	if strong.Score <= weak.Score || strong.ReasonCode != "return_to_neglected_path" ||
		strong.Components["evidenceStrength"] <= weak.Components["evidenceStrength"] ||
		strong.Components["prerequisites"] <= weak.Components["prerequisites"] {
		t.Fatalf("strong=%#v weak=%#v", strong, weak)
	}
}
