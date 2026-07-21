package dossier

import (
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestEvaluateQualityAcceptsCompleteDossier(t *testing.T) {
	t.Parallel()
	report, err := evaluateQuality(
		completeLesson(),
		"A skeptical review compares the evidence [S1].",
		completePractice(),
		nil,
		[]domain.SourceItem{{SourceID: "S1"}, {SourceID: "S2"}},
		domain.LearningBlueprint{ContinuityBridge: "Prior learning"},
		1,
		lessonWordBudgetFor(15),
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Score < 90 || !report.Checks["lessonTimeFit"] ||
		report.Metrics["lessonWordsMinimum"] != 450 ||
		report.Metrics["lessonWordsMaximum"] != 1350 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestEvaluateQualityRejectsLessonOutsideTimeBudget(t *testing.T) {
	t.Parallel()
	_, err := evaluateQuality(
		shortCompleteLesson(),
		"A skeptical review compares the evidence [S1].",
		completePractice(),
		nil,
		[]domain.SourceItem{{SourceID: "S1"}, {SourceID: "S2"}},
		domain.LearningBlueprint{},
		0,
		lessonWordBudgetFor(5),
	)
	if err == nil || !strings.Contains(err.Error(), "must contain 300 to 700 words") {
		t.Fatalf("expected actionable time-fit error, got %v", err)
	}
}

func TestLessonWordBudgetBounds(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		minutes int
		want    lessonWordBudget
	}{
		{minutes: 5, want: lessonWordBudget{minimum: 300, maximum: 700}},
		{minutes: 20, want: lessonWordBudget{minimum: 600, maximum: 1800}},
		{minutes: 90, want: lessonWordBudget{minimum: 1800, maximum: 3200}},
		{minutes: 0, want: lessonWordBudget{minimum: 600, maximum: 1800}},
	} {
		if got := lessonWordBudgetFor(test.minutes); got != test.want {
			t.Errorf("lessonWordBudgetFor(%d) = %#v, want %#v", test.minutes, got, test.want)
		}
	}
}

func TestMarkdownBodyWordCountExcludesHeadings(t *testing.T) {
	t.Parallel()
	if got := markdownBodyWordCount("## Three heading words\n\nOnly two"); got != 2 {
		t.Fatalf("markdownBodyWordCount() = %d, want 2", got)
	}
}

func TestEvaluateQualityRejectsExplorationCitation(t *testing.T) {
	t.Parallel()
	exploration := "A synthetic claim [S1]"
	_, err := evaluateQuality(
		"", "", "", &exploration, nil, domain.LearningBlueprint{}, 0,
		lessonWordBudgetFor(20),
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
