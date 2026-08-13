package research

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateInterviewsPassesExactRoadmapGate(t *testing.T) {
	t.Parallel()
	corpus := passingInterviewCorpus()
	report, err := EvaluateInterviews(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if !report.ExitGatePassed || report.QualifiedInterviews != 15 ||
		report.WeeklyPain != 10 || report.SubstantialWorkarounds != 5 ||
		report.DesignPartnerInterest != 10 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if err := ValidateInterviewExitGate(report); err != nil {
		t.Fatal(err)
	}
}

func TestInterviewGateFailsBelowEveryThreshold(t *testing.T) {
	t.Parallel()
	corpus := passingInterviewCorpus()
	corpus.Interviews = corpus.Interviews[:14]
	for index := range corpus.Interviews {
		corpus.Interviews[index].PainFrequency = "monthly"
		corpus.Interviews[index].PaysForWorkaround = false
		corpus.Interviews[index].WorkaroundMinutesWeek = 30
		corpus.Interviews[index].DesignPartnerInterest = false
	}
	report, err := EvaluateInterviews(corpus)
	if err == nil || report.ProtocolVersion != "" {
		t.Fatalf("undersized corpus should fail structurally: report=%#v err=%v", report, err)
	}

	corpus = passingInterviewCorpus()
	for index := range corpus.Interviews {
		corpus.Interviews[index].PainFrequency = "monthly"
		corpus.Interviews[index].PaysForWorkaround = false
		corpus.Interviews[index].WorkaroundMinutesWeek = 30
		corpus.Interviews[index].DesignPartnerInterest = false
	}
	report, err = EvaluateInterviews(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if report.ExitGatePassed || ValidateInterviewExitGate(report) == nil {
		t.Fatalf("threshold miss passed: %#v", report)
	}
}

func TestEvaluateInterviewsRejectsInvalidOrIdentifyingCorpus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*InterviewCorpus)
		want   string
	}{
		{"duplicate", func(c *InterviewCorpus) { c.Interviews[1].ParticipantRef = c.Interviews[0].ParticipantRef }, "duplicated"},
		{"identity", func(c *InterviewCorpus) { c.Interviews[0].ParticipantRef = "person@example.com" }, "identity"},
		{"URL reference", func(c *InterviewCorpus) { c.Interviews[0].EvidenceReference = "https://notes.example/private" }, "identity"},
		{"post freeze", func(c *InterviewCorpus) { c.Interviews[0].InterviewedAt = c.FrozenAt.Add(time.Minute) }, "interviewedAt"},
		{"unknown code", func(c *InterviewCorpus) { c.Interviews[0].PrimaryPainCode = "feature_request" }, "unknown"},
		{"unknown reachability", func(c *InterviewCorpus) { c.Interviews[0].ReachabilityCode = "viral" }, "unknown"},
		{"unfrozen", func(c *InterviewCorpus) { c.FrozenAt = nil }, "frozen"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			corpus := passingInterviewCorpus()
			test.mutate(&corpus)
			_, err := EvaluateInterviews(corpus)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("wanted %q, got %v", test.want, err)
			}
		})
	}
}

func TestUnqualifiedInterviewsCannotSatisfyGate(t *testing.T) {
	t.Parallel()
	corpus := passingInterviewCorpus()
	for index := 0; index < 6; index++ {
		corpus.Interviews[index].RoleFit = false
	}
	report, err := EvaluateInterviews(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if report.QualifiedInterviews != 9 || report.ExitGatePassed {
		t.Fatalf("unqualified participants inflated gate: %#v", report)
	}
}

func passingInterviewCorpus() InterviewCorpus {
	frozenAt := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	corpus := InterviewCorpus{ProtocolVersion: InterviewProtocolVersion, FrozenAt: &frozenAt}
	personas := []string{"ai_engineer", "ml_platform_engineer", "ai_product_lead", "technical_founder"}
	for index := 0; index < 15; index++ {
		corpus.Interviews = append(corpus.Interviews, ProblemInterview{
			ParticipantRef: strings.Repeat("x", 8) + string(rune('a'+index)),
			InterviewedAt:  frozenAt.Add(-time.Duration(index+1) * time.Hour),
			Persona:        personas[index%len(personas)], RoleFit: true,
			RecentAttemptConfirmed: true,
			PainFrequency:          map[bool]string{true: "weekly", false: "monthly"}[index < 10],
			EconomicConsequence:    2,
			WorkaroundMinutesWeek:  map[bool]int{true: 120, false: 45}[index < 5],
			PaysForWorkaround:      false,
			WorkflowCode:           "mixed", PrimaryPainCode: "context_rebuilding",
			DesiredOutcomeCode:    "make_decision",
			UrgencyCode:           "active_now",
			ReachabilityCode:      "professional_community",
			PurchasingAuthority:   "self_serve",
			DesignPartnerInterest: index < 10,
			EvidenceReference:     "research/interview-" + string(rune('a'+index)),
		})
	}
	return corpus
}
