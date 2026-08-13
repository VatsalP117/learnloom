package store

import "testing"

func TestProductEventContractAllowsOnlyExpectedSubjects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    ProductEventName
		subject string
		valid   bool
	}{
		{ProductEventSignupCompleted, "account", true},
		{ProductEventOnboardingStarted, "onboarding", true},
		{ProductEventSourcePolicySelected, "onboarding", true},
		{ProductEventSourcePreviewReached, "onboarding", true},
		{ProductEventOnboardingConfirmed, "onboarding", true},
		{ProductEventOnboardingAbandoned, "onboarding", true},
		{ProductEventPreparationWaitExited, "stream", true},
		{ProductEventStreamCreated, "stream", true},
		{ProductEventLessonGenerated, "lesson", true},
		{ProductEventLessonOpened, "lesson", true},
		{ProductEventLessonCompleted, "lesson", true},
		{ProductEventReviewAttempted, "review", true},
		{ProductEventFirstRetrieval, "review", true},
		{ProductEventActivationCompleted, "review", true},
		{ProductEventSearchIndexingEnabled, "site", true},
		{ProductEventLessonOpened, "account", false},
		{ProductEventName("arbitrary"), "lesson", false},
	}
	for _, test := range tests {
		if got := validProductEvent(test.name, test.subject); got != test.valid {
			t.Errorf("validProductEvent(%q, %q) = %v", test.name, test.subject, got)
		}
	}
}
