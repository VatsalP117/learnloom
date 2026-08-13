package httpapp

import (
	"strings"
	"testing"
)

func TestBuildResearchPlanPreviewReflectsIntentWithoutInventingSpecificClaims(t *testing.T) {
	t.Parallel()
	preview := buildResearchPlanPreview(
		"large language model evaluation",
		"decide whether product claims are credible",
		"advanced",
	)
	if preview.MinimumMinutes != 5 || preview.MaximumMinutes != 15 {
		t.Fatalf("preparation range=%d-%d", preview.MinimumMinutes, preview.MaximumMinutes)
	}
	if len(preview.InitialConcepts) != 4 ||
		!strings.Contains(preview.InitialConcepts[0], "large language model evaluation") ||
		!strings.Contains(preview.InitialConcepts[3], "product claims") {
		t.Fatalf("concepts=%#v", preview.InitialConcepts)
	}
	if !strings.Contains(preview.LikelyFirstTitle, "large language model evaluation") ||
		!strings.Contains(preview.Objective, "Stress-test") {
		t.Fatalf("title=%q objective=%q", preview.LikelyFirstTitle, preview.Objective)
	}
}

func TestBuildResearchPlanPreviewBoundsVoiceInput(t *testing.T) {
	t.Parallel()
	preview := buildResearchPlanPreview(
		strings.Repeat("topic ", 100),
		strings.Repeat("goal ", 100),
		"beginner",
	)
	if len([]rune(preview.LikelyFirstTitle)) > 115 ||
		len([]rune(preview.InitialConcepts[3])) > 115 {
		t.Fatalf("unbounded preview=%#v", preview)
	}
}
