package store

import "testing"

func TestProjectCapabilitiesUsesCompletionAndRetrievalEvidence(t *testing.T) {
	t.Parallel()
	milestones, gaps, recall := projectCapabilities([]LearnerConceptState{
		{Key: "unseen", Label: "Unseen", ExposureCount: 1},
		{Key: "explained", Label: "Explained", CompletedCount: 1},
		{Key: "weak", Label: "Weak recall", CompletedCount: 1, ReviewAttemptCount: 1, ConfidenceScore: 25},
		{Key: "solid", Label: "Solid recall", CompletedCount: 1, ReviewAttemptCount: 2, ConfidenceScore: 85},
	}, 2)
	if len(milestones) != 3 || milestones[0].Stage != "explained" ||
		milestones[1].Stage != "retrieved" || milestones[2].Stage != "recalled_solidly" {
		t.Fatalf("milestones=%#v", milestones)
	}
	if len(gaps) != 3 || recall.DueCount != 2 || recall.PracticedConcepts != 2 ||
		recall.SolidConcepts != 1 || recall.Summary != "2 retrieval prompts due now" {
		t.Fatalf("gaps=%#v recall=%#v", gaps, recall)
	}
}
