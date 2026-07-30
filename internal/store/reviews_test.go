package store

import (
	"testing"
	"time"
)

func TestScheduleReviewUsesTransparentConfidenceIntervals(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		assessment ReviewAssessment
		stage      int
		wantStage  int
		wantDue    time.Time
	}{
		{ReviewNeedsWork, 3, 0, now.Add(24 * time.Hour)},
		{ReviewPartial, 0, 1, now.Add(3 * 24 * time.Hour)},
		{ReviewSolid, 0, 2, now.Add(7 * 24 * time.Hour)},
		{ReviewSolid, 4, 4, now.Add(45 * 24 * time.Hour)},
	}
	for _, test := range tests {
		stage, due := scheduleReview(test.stage, test.assessment, now)
		if stage != test.wantStage || !due.Equal(test.wantDue) {
			t.Errorf(
				"scheduleReview(%d, %q) = %d/%s, want %d/%s",
				test.stage,
				test.assessment,
				stage,
				due,
				test.wantStage,
				test.wantDue,
			)
		}
	}
}
