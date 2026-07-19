package store

import (
	"testing"
	"time"
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

func TestSlugify(t *testing.T) {
	t.Parallel()
	if got := slugify("  AI & Learning—Daily! "); got != "ai-learning-daily" {
		t.Fatalf("unexpected slug %q", got)
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
