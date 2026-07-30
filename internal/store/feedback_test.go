package store

import "testing"

func TestValidateLessonFeedback(t *testing.T) {
	t.Parallel()
	right := "right"
	invalid := "perfect"
	if err := validateLessonFeedback(LessonFeedbackInput{Difficulty: &right}); err != nil {
		t.Fatalf("valid feedback rejected: %v", err)
	}
	if err := validateLessonFeedback(LessonFeedbackInput{}); err == nil {
		t.Fatal("empty feedback was accepted")
	}
	if err := validateLessonFeedback(LessonFeedbackInput{Difficulty: &invalid}); err == nil {
		t.Fatal("invalid difficulty was accepted")
	}
}
