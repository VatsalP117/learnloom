package source

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type SourceEvaluationCorpus struct {
	Version               string                  `json:"version"`
	LabelStatus           string                  `json:"labelStatus"`
	RankingVersion        string                  `json:"rankingVersion,omitempty"`
	CapturedAt            *time.Time              `json:"capturedAt,omitempty"`
	CandidateSnapshotHash string                  `json:"candidateSnapshotHash,omitempty"`
	Adjudication          *SourceAdjudication     `json:"adjudication,omitempty"`
	MaxDomainShare        float64                 `json:"maxDomainShare"`
	DefaultRequiredRoles  []domain.SourceRole     `json:"defaultRequiredRoles"`
	Topics                []SourceEvaluationTopic `json:"topics"`
}

type SourceAdjudication struct {
	LabelSetAHash  string  `json:"labelSetAHash"`
	LabelSetBHash  string  `json:"labelSetBHash"`
	AdjudicatorRef string  `json:"adjudicatorRef"`
	AgreementRate  float64 `json:"agreementRate"`
	Disagreements  int     `json:"disagreements"`
	Resolved       int     `json:"resolved"`
}

type SourceEvaluationTopic struct {
	ID            string                      `json:"id"`
	Topic         string                      `json:"topic"`
	Outcome       string                      `json:"outcome"`
	RequiredRoles []domain.SourceRole         `json:"requiredRoles"`
	Candidates    []SourceEvaluationCandidate `json:"candidates"`
}

type SourceEvaluationCandidate struct {
	Title                 string            `json:"title,omitempty"`
	URL                   string            `json:"url"`
	Snippet               string            `json:"snippet,omitempty"`
	RegistrableDomain     string            `json:"registrableDomain"`
	PublishedAt           *time.Time        `json:"publishedAt,omitempty"`
	SystemRank            int               `json:"systemRank"`
	SystemRole            domain.SourceRole `json:"systemRole"`
	HumanRole             domain.SourceRole `json:"humanRole,omitempty"`
	HumanReviewed         bool              `json:"humanReviewed"`
	TopicalRelevance      int               `json:"topicalRelevance"`
	SourceAuthority       int               `json:"sourceAuthority"`
	Primaryness           int               `json:"primaryness"`
	Recency               int               `json:"recency"`
	ExplanatoryUsefulness int               `json:"explanatoryUsefulness"`
	Independence          int               `json:"independence"`
	Accessibility         int               `json:"accessibility"`
	CounterweightValue    int               `json:"counterweightValue"`
	Recommended           bool              `json:"recommended"`
	Unsafe                bool              `json:"unsafe"`
	Unusable              bool              `json:"unusable"`
	LabelerNotes          string            `json:"labelerNotes"`
	AdjudicationNote      string            `json:"adjudicationNote,omitempty"`
}

func CaptureSourceEvaluationTopic(
	ctx context.Context,
	searcher Searcher,
	topic SourceEvaluationTopic,
	maxQueries, maxCandidates, maxActive int,
) ([]SourceEvaluationCandidate, error) {
	if searcher == nil {
		return nil, errors.New("source evaluation capture requires a search provider")
	}
	queries := discoveryQueries(topic.Topic)
	if maxQueries < 1 || maxQueries > len(queries) {
		maxQueries = len(queries)
	}
	queries = queries[:maxQueries]
	if maxCandidates < 10 {
		maxCandidates = 30
	}
	if maxActive < 5 {
		maxActive = 8
	}
	outcomes := parallelMapOrdered(
		ctx, queries, min(5, len(queries)),
		func(ctx context.Context, query discoveryQuery) ([]SearchCandidate, error) {
			return searcher.Search(ctx, SearchRequest{
				Query: query.Query, Language: "all", Category: "general", Page: 1,
			})
		},
	)
	var queryErrors int
	for _, outcome := range outcomes {
		if outcome.err != nil {
			queryErrors++
		}
	}
	raw := interleaveDiscoveryCandidates(queries, outcomes, maxCandidates)
	if len(raw) == 0 {
		return nil, fmt.Errorf("source evaluation topic %q returned no candidates (%d query errors)", topic.ID, queryErrors)
	}
	selected, _ := rankDiscoveryCandidates(
		topic.Topic, raw, nil, maxCandidates, maxActive,
	)
	selectedRanks := make(map[string]int, len(selected))
	for index, candidate := range selected {
		selectedRanks[candidate.URL] = index + 1
	}
	seen := make(map[string]struct{}, maxCandidates)
	captured := make([]SourceEvaluationCandidate, 0, maxCandidates)
	appendCandidate := func(candidate discoveryCandidate, systemRank int) {
		normalized, domainName, err := gateCandidateURL(candidate.SearchCandidate.URL)
		if err != nil {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		candidate.URL = normalized
		candidate.Domain = domainName
		candidate.Role = classifyCandidateRole(candidate)
		captured = append(captured, SourceEvaluationCandidate{
			Title: strings.TrimSpace(candidate.Title), URL: normalized,
			Snippet: strings.TrimSpace(candidate.Snippet), RegistrableDomain: domainName,
			PublishedAt: candidate.PublishedAt, SystemRank: systemRank,
			SystemRole: candidate.Role,
		})
	}
	for _, candidate := range selected {
		appendCandidate(candidate, selectedRanks[candidate.URL])
	}
	for _, candidate := range raw {
		normalized, _, err := gateCandidateURL(candidate.SearchCandidate.URL)
		if err != nil {
			continue
		}
		if _, selected := selectedRanks[normalized]; selected {
			continue
		}
		appendCandidate(candidate, 0)
	}
	if len(captured) < 5 {
		return nil, fmt.Errorf("source evaluation topic %q retained %d candidates; at least 5 are required", topic.ID, len(captured))
	}
	return captured, nil
}

type SourceEvaluationMetrics struct {
	Version                   string  `json:"version"`
	Topics                    int     `json:"topics"`
	PrecisionAt5              float64 `json:"precisionAt5"`
	DomainDiversity           float64 `json:"domainDiversity"`
	RequiredRoleCoverage      float64 `json:"requiredRoleCoverage"`
	UnsafeUnusableRejection   float64 `json:"unsafeUnusableRejection"`
	SelectedWithoutRole       int     `json:"selectedWithoutRole"`
	TopicsWithFewerThan5Ranks int     `json:"topicsWithFewerThan5Ranks"`
	HumanAdjudicated          bool    `json:"humanAdjudicated"`
	AdjudicationAgreement     float64 `json:"adjudicationAgreement"`
	MaxObservedDomainShare    float64 `json:"maxObservedDomainShare"`
	DomainShareViolations     int     `json:"domainShareViolations"`
}

func EvaluateSourceCorpus(corpus SourceEvaluationCorpus) (SourceEvaluationMetrics, error) {
	if err := ValidateSourceCorpusSeed(corpus); err != nil {
		return SourceEvaluationMetrics{}, err
	}
	if corpus.LabelStatus != "human_labeled" && corpus.LabelStatus != "human_adjudicated" {
		return SourceEvaluationMetrics{}, errors.New("source evaluation corpus is not human labeled")
	}
	metrics := SourceEvaluationMetrics{
		Version: corpus.Version, Topics: len(corpus.Topics),
		HumanAdjudicated: corpus.LabelStatus == "human_adjudicated",
	}
	if corpus.Adjudication != nil {
		metrics.AdjudicationAgreement = corpus.Adjudication.AgreementRate
	}
	var selected, recommended, domainSlots int
	var requiredRoles, coveredRoles int
	var unsafeUnusable, rejectedUnsafeUnusable int
	for _, topic := range corpus.Topics {
		candidates := append([]SourceEvaluationCandidate(nil), topic.Candidates...)
		seenRanks := make(map[int]struct{})
		for _, candidate := range candidates {
			if err := validateHumanSourceLabel(topic.ID, candidate); err != nil {
				return SourceEvaluationMetrics{}, err
			}
			if candidate.SystemRank > 0 {
				if _, exists := seenRanks[candidate.SystemRank]; exists {
					return SourceEvaluationMetrics{}, fmt.Errorf(
						"topic %q contains duplicate system rank %d",
						topic.ID,
						candidate.SystemRank,
					)
				}
				seenRanks[candidate.SystemRank] = struct{}{}
			}
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			left, right := candidates[i].SystemRank, candidates[j].SystemRank
			if left <= 0 {
				return false
			}
			if right <= 0 {
				return true
			}
			return left < right
		})
		selectedRoles := make(map[domain.SourceRole]bool)
		domains := make(map[string]bool)
		rankedDomains := make(map[string]int)
		rankedCandidates := 0
		topFive := 0
		for _, candidate := range candidates {
			if candidate.Unsafe || candidate.Unusable {
				unsafeUnusable++
				if candidate.SystemRank <= 0 || candidate.SystemRank > 5 {
					rejectedUnsafeUnusable++
				}
			}
			if candidate.SystemRank < 1 || candidate.SystemRank > 5 {
				if candidate.SystemRank > 0 {
					rankedCandidates++
					rankedDomains[candidate.RegistrableDomain]++
				}
				continue
			}
			rankedCandidates++
			rankedDomains[candidate.RegistrableDomain]++
			topFive++
			selected++
			if candidate.Recommended {
				recommended++
			}
			if candidate.HumanRole == "" {
				metrics.SelectedWithoutRole++
			} else if candidate.Recommended {
				selectedRoles[candidate.HumanRole] = true
			}
			if candidate.RegistrableDomain != "" {
				domains[candidate.RegistrableDomain] = true
			}
		}
		if topFive < 5 {
			metrics.TopicsWithFewerThan5Ranks++
		}
		domainSlots += topFive
		for _, count := range rankedDomains {
			share := float64(count) / float64(rankedCandidates)
			if share > metrics.MaxObservedDomainShare {
				metrics.MaxObservedDomainShare = share
			}
			if share > corpus.MaxDomainShare {
				metrics.DomainShareViolations++
				break
			}
		}
		metrics.DomainDiversity += float64(len(domains))
		roles := topic.RequiredRoles
		if len(roles) == 0 {
			roles = corpus.DefaultRequiredRoles
		}
		for _, role := range roles {
			requiredRoles++
			if selectedRoles[role] {
				coveredRoles++
			}
		}
	}
	if selected == 0 {
		return SourceEvaluationMetrics{}, errors.New("source evaluation corpus has no ranked candidates")
	}
	metrics.PrecisionAt5 = float64(recommended) / float64(selected)
	if domainSlots > 0 {
		metrics.DomainDiversity /= float64(domainSlots)
	}
	if requiredRoles > 0 {
		metrics.RequiredRoleCoverage = float64(coveredRoles) / float64(requiredRoles)
	}
	if unsafeUnusable == 0 {
		metrics.UnsafeUnusableRejection = 1
	} else {
		metrics.UnsafeUnusableRejection = float64(rejectedUnsafeUnusable) / float64(unsafeUnusable)
	}
	return metrics, nil
}

func validateHumanSourceLabel(topicID string, candidate SourceEvaluationCandidate) error {
	parsed, parseErr := url.Parse(candidate.URL)
	if strings.TrimSpace(candidate.URL) == "" || parseErr != nil || parsed.Scheme != "https" ||
		parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("topic %q contains a candidate without a URL", topicID)
	}
	for name, value := range map[string]int{
		"topicalRelevance":      candidate.TopicalRelevance,
		"sourceAuthority":       candidate.SourceAuthority,
		"primaryness":           candidate.Primaryness,
		"recency":               candidate.Recency,
		"explanatoryUsefulness": candidate.ExplanatoryUsefulness,
		"independence":          candidate.Independence,
		"accessibility":         candidate.Accessibility,
		"counterweightValue":    candidate.CounterweightValue,
	} {
		if value < 0 || value > 3 {
			return fmt.Errorf("topic %q candidate %q has invalid %s=%d", topicID, candidate.URL, name, value)
		}
	}
	if candidate.Recommended && (candidate.TopicalRelevance < 2 ||
		candidate.SourceAuthority < 2 || candidate.Accessibility < 2 ||
		candidate.Unsafe || candidate.Unusable) {
		return fmt.Errorf("topic %q candidate %q violates the recommended-source rubric", topicID, candidate.URL)
	}
	if candidate.HumanRole != domain.SourceRoleOfficialPrimary &&
		candidate.HumanRole != domain.SourceRoleResearch &&
		candidate.HumanRole != domain.SourceRolePractitioner &&
		candidate.HumanRole != domain.SourceRoleReporting &&
		candidate.HumanRole != domain.SourceRoleCounterweight {
		return fmt.Errorf("topic %q candidate %q has an invalid human role", topicID, candidate.URL)
	}
	if !candidate.HumanReviewed || strings.TrimSpace(candidate.LabelerNotes) == "" ||
		len(candidate.LabelerNotes) > 500 {
		return fmt.Errorf(
			"topic %q candidate %q is missing a completed human review and bounded note",
			topicID, candidate.URL,
		)
	}
	if candidate.Recommended && candidate.HumanRole == domain.SourceRoleOfficialPrimary &&
		candidate.Primaryness < 2 {
		return fmt.Errorf("topic %q official candidate %q is not sufficiently primary", topicID, candidate.URL)
	}
	return nil
}

func AdjudicateSourceCorpora(
	labelSetA, labelSetB, resolution SourceEvaluationCorpus,
	labelSetAHash, labelSetBHash, adjudicatorRef string,
) (SourceEvaluationCorpus, error) {
	for name, corpus := range map[string]SourceEvaluationCorpus{
		"label set A": labelSetA,
		"label set B": labelSetB,
		"resolution":  resolution,
	} {
		if err := ValidateSourceCorpusSeed(corpus); err != nil {
			return SourceEvaluationCorpus{}, fmt.Errorf("%s: %w", name, err)
		}
		if corpus.LabelStatus != "human_labeled" {
			return SourceEvaluationCorpus{}, fmt.Errorf("%s must have labelStatus human_labeled", name)
		}
	}
	if labelSetA.Version != labelSetB.Version || labelSetA.Version != resolution.Version ||
		labelSetA.RankingVersion != labelSetB.RankingVersion ||
		labelSetA.RankingVersion != resolution.RankingVersion ||
		!sameCapturedAt(labelSetA.CapturedAt, labelSetB.CapturedAt) ||
		!sameCapturedAt(labelSetA.CapturedAt, resolution.CapturedAt) {
		return SourceEvaluationCorpus{}, errors.New("source label sets do not share the same captured corpus version")
	}
	if labelSetA.CandidateSnapshotHash != labelSetB.CandidateSnapshotHash ||
		labelSetA.CandidateSnapshotHash != resolution.CandidateSnapshotHash {
		return SourceEvaluationCorpus{}, errors.New("source label sets do not reference the same captured candidate snapshot")
	}
	if len(labelSetA.Topics) != len(labelSetB.Topics) || len(labelSetA.Topics) != len(resolution.Topics) {
		return SourceEvaluationCorpus{}, errors.New("source label sets contain different topic counts")
	}

	var decisions, agreements, disagreements, resolved int
	for topicIndex := range labelSetA.Topics {
		aTopic := labelSetA.Topics[topicIndex]
		bTopic := labelSetB.Topics[topicIndex]
		rTopic := &resolution.Topics[topicIndex]
		if !sameEvaluationTopicSnapshot(aTopic, bTopic) || !sameEvaluationTopicSnapshot(aTopic, *rTopic) {
			return SourceEvaluationCorpus{}, fmt.Errorf("topic index %d does not share one immutable candidate snapshot", topicIndex)
		}
		for candidateIndex := range aTopic.Candidates {
			a := aTopic.Candidates[candidateIndex]
			b := bTopic.Candidates[candidateIndex]
			r := &rTopic.Candidates[candidateIndex]
			for _, candidate := range []SourceEvaluationCandidate{a, b, *r} {
				if err := validateHumanSourceLabel(aTopic.ID, candidate); err != nil {
					return SourceEvaluationCorpus{}, err
				}
			}
			candidateDisagreements := 0
			for _, equal := range []bool{
				a.Recommended == b.Recommended,
				a.Unsafe == b.Unsafe,
				a.Unusable == b.Unusable,
				a.HumanRole == b.HumanRole,
			} {
				decisions++
				if equal {
					agreements++
				} else {
					disagreements++
					candidateDisagreements++
				}
			}
			if candidateDisagreements > 0 {
				if strings.TrimSpace(r.AdjudicationNote) == "" || len(r.AdjudicationNote) > 500 {
					return SourceEvaluationCorpus{}, fmt.Errorf(
						"topic %q candidate %q needs a bounded adjudication note",
						aTopic.ID, a.URL,
					)
				}
				resolved += candidateDisagreements
			}
		}
	}
	if decisions == 0 {
		return SourceEvaluationCorpus{}, errors.New("source adjudication contains no candidate decisions")
	}
	resolution.LabelStatus = "human_adjudicated"
	resolution.Adjudication = &SourceAdjudication{
		LabelSetAHash: labelSetAHash, LabelSetBHash: labelSetBHash,
		AdjudicatorRef: strings.TrimSpace(adjudicatorRef),
		AgreementRate:  float64(agreements) / float64(decisions),
		Disagreements:  disagreements, Resolved: resolved,
	}
	if err := validateSourceAdjudication(resolution.Adjudication); err != nil {
		return SourceEvaluationCorpus{}, err
	}
	return resolution, nil
}

func sameCapturedAt(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameEvaluationTopicSnapshot(left, right SourceEvaluationTopic) bool {
	if left.ID != right.ID || left.Topic != right.Topic || left.Outcome != right.Outcome ||
		len(left.RequiredRoles) != len(right.RequiredRoles) || len(left.Candidates) != len(right.Candidates) {
		return false
	}
	for index := range left.RequiredRoles {
		if left.RequiredRoles[index] != right.RequiredRoles[index] {
			return false
		}
	}
	for index := range left.Candidates {
		if evaluationCandidateSnapshot(left.Candidates[index]) != evaluationCandidateSnapshot(right.Candidates[index]) {
			return false
		}
	}
	return true
}

func evaluationCandidateSnapshot(candidate SourceEvaluationCandidate) string {
	publishedAt := ""
	if candidate.PublishedAt != nil {
		publishedAt = candidate.PublishedAt.UTC().Format(time.RFC3339Nano)
	}
	return strings.Join([]string{
		candidate.Title, candidate.URL, candidate.Snippet, candidate.RegistrableDomain,
		publishedAt, fmt.Sprint(candidate.SystemRank), string(candidate.SystemRole),
	}, "\x00")
}

func SourceEvaluationSnapshotHash(corpus SourceEvaluationCorpus) string {
	hash := sha256.New()
	write := func(value string) {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	write(corpus.Version)
	write(corpus.RankingVersion)
	write(fmt.Sprintf("%.6f", corpus.MaxDomainShare))
	for _, role := range corpus.DefaultRequiredRoles {
		write(string(role))
	}
	for _, topic := range corpus.Topics {
		write(topic.ID)
		write(topic.Topic)
		write(topic.Outcome)
		for _, role := range topic.RequiredRoles {
			write(string(role))
		}
		for _, candidate := range topic.Candidates {
			write(evaluationCandidateSnapshot(candidate))
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func ValidateSourceReleaseGates(metrics SourceEvaluationMetrics) error {
	if !metrics.HumanAdjudicated {
		return errors.New("source release corpus has not completed independent human adjudication")
	}
	if metrics.Topics < 50 {
		return fmt.Errorf("source release has %d topics; at least 50 are required", metrics.Topics)
	}
	if metrics.PrecisionAt5 < 0.80 {
		return fmt.Errorf("source precision@5 %.3f is below 0.80", metrics.PrecisionAt5)
	}
	if metrics.RequiredRoleCoverage < 1 {
		return fmt.Errorf("source required-role coverage %.3f is below 1.0", metrics.RequiredRoleCoverage)
	}
	if metrics.UnsafeUnusableRejection < 1 {
		return fmt.Errorf("unsafe/unusable rejection %.3f is below 1.0", metrics.UnsafeUnusableRejection)
	}
	if metrics.DomainShareViolations != 0 {
		return fmt.Errorf(
			"source release has %d topics exceeding the registrable-domain share limit",
			metrics.DomainShareViolations,
		)
	}
	if metrics.SelectedWithoutRole != 0 || metrics.TopicsWithFewerThan5Ranks != 0 {
		return fmt.Errorf(
			"source release has %d selected candidates without roles and %d topics with fewer than 5 ranks",
			metrics.SelectedWithoutRole, metrics.TopicsWithFewerThan5Ranks,
		)
	}
	return nil
}

func ValidateSourceCorpusSeed(corpus SourceEvaluationCorpus) error {
	if strings.TrimSpace(corpus.Version) == "" {
		return errors.New("source evaluation corpus version is required")
	}
	if len(corpus.Topics) < 50 {
		return fmt.Errorf(
			"source evaluation corpus has %d topics; at least 50 are required",
			len(corpus.Topics),
		)
	}
	if corpus.MaxDomainShare <= 0 || corpus.MaxDomainShare > 1 {
		return errors.New("source evaluation maxDomainShare must be between 0 and 1")
	}
	if corpus.LabelStatus == "human_adjudicated" {
		if err := validateSourceAdjudication(corpus.Adjudication); err != nil {
			return err
		}
	}
	if (corpus.LabelStatus == "awaiting_human_labels" && corpus.CapturedAt != nil) ||
		corpus.LabelStatus == "human_labeled" || corpus.LabelStatus == "human_adjudicated" {
		if corpus.CandidateSnapshotHash != SourceEvaluationSnapshotHash(corpus) {
			return errors.New("source evaluation candidate snapshot hash is missing or does not match the corpus")
		}
	}
	seen := make(map[string]struct{}, len(corpus.Topics))
	for _, topic := range corpus.Topics {
		if strings.TrimSpace(topic.ID) == "" || strings.TrimSpace(topic.Topic) == "" ||
			strings.TrimSpace(topic.Outcome) == "" {
			return fmt.Errorf("topic %q is missing its ID, topic, or outcome", topic.ID)
		}
		if _, exists := seen[topic.ID]; exists {
			return fmt.Errorf("source evaluation topic ID %q is duplicated", topic.ID)
		}
		seen[topic.ID] = struct{}{}
	}
	return nil
}

func validateSourceAdjudication(value *SourceAdjudication) error {
	if value == nil {
		return errors.New("human-adjudicated source corpus is missing adjudication provenance")
	}
	for name, hash := range map[string]string{
		"label set A": value.LabelSetAHash,
		"label set B": value.LabelSetBHash,
	} {
		if len(hash) != 64 {
			return fmt.Errorf("%s SHA-256 is invalid", name)
		}
		for _, char := range hash {
			if !strings.ContainsRune("0123456789abcdef", char) {
				return fmt.Errorf("%s SHA-256 is invalid", name)
			}
		}
	}
	if value.LabelSetAHash == value.LabelSetBHash {
		return errors.New("independent source label sets must have different hashes")
	}
	if strings.TrimSpace(value.AdjudicatorRef) == "" || len(value.AdjudicatorRef) > 100 {
		return errors.New("source adjudicator reference is invalid")
	}
	if value.AgreementRate < 0 || value.AgreementRate > 1 || value.Disagreements < 0 ||
		value.Resolved < 0 || value.Resolved != value.Disagreements {
		return errors.New("source adjudication counts or agreement rate are invalid")
	}
	return nil
}
