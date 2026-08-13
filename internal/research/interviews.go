package research

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const InterviewProtocolVersion = "launch-icp-interviews-v1"

type InterviewCorpus struct {
	ProtocolVersion string             `json:"protocolVersion"`
	FrozenAt        *time.Time         `json:"frozenAt,omitempty"`
	Interviews      []ProblemInterview `json:"interviews"`
}

type ProblemInterview struct {
	ParticipantRef         string    `json:"participantRef"`
	InterviewedAt          time.Time `json:"interviewedAt"`
	Persona                string    `json:"persona"`
	RoleFit                bool      `json:"roleFit"`
	RecentAttemptConfirmed bool      `json:"recentAttemptConfirmed"`
	PainFrequency          string    `json:"painFrequency"`
	EconomicConsequence    int       `json:"economicConsequence"`
	WorkaroundMinutesWeek  int       `json:"workaroundMinutesWeek"`
	PaysForWorkaround      bool      `json:"paysForWorkaround"`
	WorkflowCode           string    `json:"workflowCode"`
	PrimaryPainCode        string    `json:"primaryPainCode"`
	DesiredOutcomeCode     string    `json:"desiredOutcomeCode"`
	UrgencyCode            string    `json:"urgencyCode"`
	ReachabilityCode       string    `json:"reachabilityCode"`
	PurchasingAuthority    string    `json:"purchasingAuthority"`
	DesignPartnerInterest  bool      `json:"designPartnerInterest"`
	EvidenceReference      string    `json:"evidenceReference"`
}

type InterviewReport struct {
	ProtocolVersion        string           `json:"protocolVersion"`
	QualifiedInterviews    int              `json:"qualifiedInterviews"`
	WeeklyPain             int              `json:"weeklyPain"`
	SubstantialWorkarounds int              `json:"substantialWorkarounds"`
	DesignPartnerInterest  int              `json:"designPartnerInterest"`
	ExitGatePassed         bool             `json:"exitGatePassed"`
	PersonaBreakdown       []PersonaSummary `json:"personaBreakdown"`
}

type PersonaSummary struct {
	Persona                string `json:"persona"`
	Qualified              int    `json:"qualified"`
	WeeklyPain             int    `json:"weeklyPain"`
	SubstantialWorkarounds int    `json:"substantialWorkarounds"`
	DesignPartnerInterest  int    `json:"designPartnerInterest"`
	EconomicScoreTotal     int    `json:"economicScoreTotal"`
}

var allowedPersonas = map[string]bool{
	"ai_engineer": true, "ml_platform_engineer": true,
	"ai_product_lead": true, "technical_founder": true,
}

var allowedFrequencies = map[string]bool{
	"daily": true, "weekly": true, "monthly": true, "less_often": true,
}

var allowedWorkflows = map[string]bool{
	"search_and_tabs": true, "chat_assistant": true, "newsletter_or_feed": true,
	"read_later_or_notes": true, "course_or_training": true,
	"colleague_or_expert": true, "mixed": true, "none": true,
}

var allowedPains = map[string]bool{
	"source_discovery": true, "source_judgment": true, "context_rebuilding": true,
	"learning_sequence": true, "retention_and_recall": true,
	"application_to_work": true, "time_fragmentation": true, "other": true,
}

var allowedOutcomes = map[string]bool{
	"make_decision": true, "ship_system": true, "evaluate_risk": true,
	"explain_to_others": true, "stay_current": true, "build_capability": true,
	"other": true,
}

var allowedUrgency = map[string]bool{
	"active_now": true, "next_quarter": true,
	"when_triggered": true, "no_deadline": true,
}

var allowedReachability = map[string]bool{
	"direct_network": true, "professional_community": true,
	"newsletter_or_creator": true, "search_or_content": true,
	"partner_channel": true, "unknown": true,
}

var allowedPurchasingAuthority = map[string]bool{
	"self_serve": true, "expense_budget": true, "manager_approval": true,
	"procurement": true, "no_budget": true, "unknown": true,
}

func EvaluateInterviews(corpus InterviewCorpus) (InterviewReport, error) {
	if corpus.ProtocolVersion != InterviewProtocolVersion {
		return InterviewReport{}, fmt.Errorf("interview protocol must be %s", InterviewProtocolVersion)
	}
	if corpus.FrozenAt == nil || corpus.FrozenAt.IsZero() {
		return InterviewReport{}, errors.New("interview corpus must be frozen before evaluation")
	}
	if len(corpus.Interviews) < 15 {
		return InterviewReport{}, fmt.Errorf("interview corpus has %d interviews; at least 15 are required", len(corpus.Interviews))
	}
	seen := map[string]bool{}
	byPersona := map[string]*PersonaSummary{}
	report := InterviewReport{ProtocolVersion: corpus.ProtocolVersion}
	for index, interview := range corpus.Interviews {
		if err := validateInterview(index, interview, *corpus.FrozenAt); err != nil {
			return InterviewReport{}, err
		}
		if seen[interview.ParticipantRef] {
			return InterviewReport{}, fmt.Errorf("participantRef %q is duplicated", interview.ParticipantRef)
		}
		seen[interview.ParticipantRef] = true
		if !interview.RoleFit || !interview.RecentAttemptConfirmed {
			continue
		}
		report.QualifiedInterviews++
		summary := byPersona[interview.Persona]
		if summary == nil {
			summary = &PersonaSummary{Persona: interview.Persona}
			byPersona[interview.Persona] = summary
		}
		summary.Qualified++
		summary.EconomicScoreTotal += interview.EconomicConsequence
		weekly := interview.PainFrequency == "daily" || interview.PainFrequency == "weekly"
		if weekly {
			report.WeeklyPain++
			summary.WeeklyPain++
		}
		substantial := interview.PaysForWorkaround || interview.WorkaroundMinutesWeek >= 120
		if substantial {
			report.SubstantialWorkarounds++
			summary.SubstantialWorkarounds++
		}
		if interview.DesignPartnerInterest {
			report.DesignPartnerInterest++
			summary.DesignPartnerInterest++
		}
	}
	for _, persona := range []string{
		"ai_engineer", "ml_platform_engineer", "ai_product_lead", "technical_founder",
	} {
		if summary := byPersona[persona]; summary != nil {
			report.PersonaBreakdown = append(report.PersonaBreakdown, *summary)
		}
	}
	report.ExitGatePassed = report.QualifiedInterviews >= 15 && report.WeeklyPain >= 10 &&
		report.SubstantialWorkarounds >= 5 && report.DesignPartnerInterest >= 10
	return report, nil
}

func ValidateInterviewExitGate(report InterviewReport) error {
	if !report.ExitGatePassed {
		return fmt.Errorf(
			"ICP evidence gate failed: qualified=%d weekly=%d substantial_workarounds=%d design_partner_interest=%d",
			report.QualifiedInterviews, report.WeeklyPain,
			report.SubstantialWorkarounds, report.DesignPartnerInterest,
		)
	}
	return nil
}

func validateInterview(index int, value ProblemInterview, frozenAt time.Time) error {
	prefix := fmt.Sprintf("interview %d", index+1)
	if strings.TrimSpace(value.ParticipantRef) == "" || len(value.ParticipantRef) > 64 {
		return fmt.Errorf("%s has an invalid participantRef", prefix)
	}
	if value.InterviewedAt.IsZero() || value.InterviewedAt.After(frozenAt) {
		return fmt.Errorf("%s has an invalid interviewedAt", prefix)
	}
	if !allowedPersonas[value.Persona] || !allowedFrequencies[value.PainFrequency] ||
		!allowedWorkflows[value.WorkflowCode] || !allowedPains[value.PrimaryPainCode] ||
		!allowedOutcomes[value.DesiredOutcomeCode] || !allowedUrgency[value.UrgencyCode] ||
		!allowedReachability[value.ReachabilityCode] ||
		!allowedPurchasingAuthority[value.PurchasingAuthority] {
		return fmt.Errorf("%s contains an unknown bounded code", prefix)
	}
	if value.EconomicConsequence < 0 || value.EconomicConsequence > 3 ||
		value.WorkaroundMinutesWeek < 0 || value.WorkaroundMinutesWeek > 10_080 {
		return fmt.Errorf("%s has an invalid score or workaround time", prefix)
	}
	if strings.TrimSpace(value.EvidenceReference) == "" || len(value.EvidenceReference) > 160 {
		return fmt.Errorf("%s has an invalid evidenceReference", prefix)
	}
	for _, forbidden := range []string{"@", "http://", "https://"} {
		if strings.Contains(value.ParticipantRef, forbidden) || strings.Contains(value.EvidenceReference, forbidden) {
			return fmt.Errorf("%s contains identity or URL material in a reference", prefix)
		}
	}
	return nil
}
