package store

import (
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestNewsletterSourceModeValidation(t *testing.T) {
	t.Parallel()

	validTimeZone := "UTC"
	validSchedule := 9
	validSource := domain.SourceDefinition{
		Name: "Test", URL: "https://example.com/feed.xml", Limit: 5,
	}

	for _, test := range []struct {
		name  string
		input NewsletterInput
		valid bool
	}{
		{
			name: "discovered mode with empty sources is valid",
			input: NewsletterInput{
				Name: "Discovered", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode: domain.SourceModeDiscovered, Sources: nil,
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: true,
		},
		{
			name: "discovered mode with sources is invalid",
			input: NewsletterInput{
				Name: "Discovered", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode:   domain.SourceModeDiscovered,
				Sources:      []domain.SourceDefinition{validSource},
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: false,
		},
		{
			name: "provided mode with one source is valid",
			input: NewsletterInput{
				Name: "Provided", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode:   domain.SourceModeProvided,
				Sources:      []domain.SourceDefinition{validSource},
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: true,
		},
		{
			name: "provided mode with empty sources is invalid",
			input: NewsletterInput{
				Name: "Provided", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode: domain.SourceModeProvided, Sources: nil,
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: false,
		},
		{
			name: "hybrid mode with one source is valid",
			input: NewsletterInput{
				Name: "Hybrid", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode:   domain.SourceModeHybrid,
				Sources:      []domain.SourceDefinition{validSource},
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: true,
		},
		{
			name: "hybrid mode with empty sources is invalid",
			input: NewsletterInput{
				Name: "Hybrid", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode: domain.SourceModeHybrid, Sources: nil,
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: false,
		},
		{
			name: "empty mode with sources defaults to provided",
			input: NewsletterInput{
				Name: "Default", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode:   "",
				Sources:      []domain.SourceDefinition{validSource},
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: true,
		},
		{
			name: "invalid mode string is rejected",
			input: NewsletterInput{
				Name: "Bad Mode", Topic: "AI", LearnerLevel: "intermediate",
				LearnerGoal: "learn", LessonMinutes: 20,
				SourceMode:   "custom",
				Sources:      []domain.SourceDefinition{validSource},
				ScheduleHour: validSchedule, TimeZone: validTimeZone,
			},
			valid: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeNewsletterInput(test.input)
			if test.valid && err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected invalid, got nil error")
			}
		})
	}
}
