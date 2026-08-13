package store

import (
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestNextOccurrenceHandlesDST(t *testing.T) {
	t.Parallel()
	after := time.Date(2026, 3, 8, 6, 0, 0, 0, time.UTC)
	next, err := NextOccurrence(after, "America/New_York", 9, 0)
	if err != nil {
		t.Fatal(err)
	}
	if local := next.In(mustLocation(t, "America/New_York")); local.Hour() != 9 {
		t.Fatalf("unexpected local occurrence: %s", local)
	}
}

func TestNextRhythmOccurrenceUsesSelectedWeekdays(t *testing.T) {
	t.Parallel()
	after := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC) // Wednesday.
	next, err := NextRhythmOccurrence(
		after,
		"UTC",
		8,
		0,
		domain.RhythmSelectedWeekdays,
		[]int{1, 5},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next=%s, want %s", next, want)
	}
}

func TestNextRhythmOccurrenceUsesFirstDayForWeeklySynthesis(t *testing.T) {
	t.Parallel()
	after := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC) // Tuesday.
	next, err := NextRhythmOccurrence(
		after,
		"UTC",
		9,
		30,
		domain.RhythmWeeklySynthesis,
		[]int{4},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 13, 9, 30, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next=%s, want %s", next, want)
	}
}

func TestNormalizeRhythmInputRejectsUnsafeValuesAndDeduplicatesDays(t *testing.T) {
	t.Parallel()
	input, err := normalizeRhythmInput(RhythmInput{
		Mode: domain.RhythmSelectedWeekdays, SelectedWeekdays: []int{5, 1, 5},
		AutoThrottleEnabled: true, UnopenedLessonLimit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(input.SelectedWeekdays) != 2 || input.SelectedWeekdays[0] != 1 ||
		input.SelectedWeekdays[1] != 5 {
		t.Fatalf("weekdays=%v", input.SelectedWeekdays)
	}
	if _, err := normalizeRhythmInput(RhythmInput{
		Mode: domain.RhythmDaily, SelectedWeekdays: []int{8}, UnopenedLessonLimit: 3,
	}); err == nil {
		t.Fatal("invalid weekday was accepted")
	}
}

func TestSlugify(t *testing.T) {
	t.Parallel()
	if got := slugify("  AI & Learning—Daily! "); got != "ai-learning-daily" {
		t.Fatalf("unexpected slug %q", got)
	}
}

func TestApplyCreateDefaultsSupportsTopicOnlyInput(t *testing.T) {
	input := applyCreateDefaults(NewsletterInput{Topic: "LLM inference"})
	if input.Name != "LLM inference" ||
		input.LearnerLevel != "intermediate" ||
		input.LearnerGoal != "Build a practical understanding of LLM inference." ||
		input.LessonMinutes != 12 {
		t.Fatalf("defaults=%#v", input)
	}
}

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return location
}
