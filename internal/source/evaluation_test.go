package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestCaptureSourceEvaluationTopicFreezesSystemMetadataWithoutHumanLabels(t *testing.T) {
	t.Parallel()
	candidates, err := CaptureSourceEvaluationTopic(
		context.Background(), previewSearcher{},
		SourceEvaluationTopic{ID: "ai-capture", Topic: "inference systems"},
		5, 30, 8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 5 {
		t.Fatalf("captured %d candidates, want 5", len(candidates))
	}
	seenRanks := make(map[int]bool)
	for _, candidate := range candidates {
		if candidate.Title == "" || candidate.URL == "" || candidate.Snippet == "" ||
			candidate.RegistrableDomain == "" || candidate.SystemRole == "" {
			t.Fatalf("incomplete captured candidate=%#v", candidate)
		}
		if candidate.TopicalRelevance != 0 || candidate.HumanReviewed || candidate.Recommended || candidate.Unsafe ||
			candidate.Unusable || candidate.LabelerNotes != "" {
			t.Fatalf("capture invented human labels=%#v", candidate)
		}
		if candidate.SystemRank > 0 {
			seenRanks[candidate.SystemRank] = true
		}
	}
	for rank := 1; rank <= 5; rank++ {
		if !seenRanks[rank] {
			t.Fatalf("captured ranks=%v, missing %d", seenRanks, rank)
		}
	}
}

func TestVersionedSourceEvaluationSeedHasFiftyRepresentativeTopics(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/source-evaluation-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus SourceEvaluationCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSourceCorpusSeed(corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Version != "source-eval-v1" || corpus.LabelStatus != "awaiting_human_labels" ||
		len(corpus.Topics) != 50 {
		t.Fatalf("corpus=%s status=%s topics=%d", corpus.Version, corpus.LabelStatus, len(corpus.Topics))
	}
}

func TestEvaluateSourceCorpusCalculatesLaunchMetrics(t *testing.T) {
	t.Parallel()
	corpus := SourceEvaluationCorpus{
		Version: "source-eval-v1", LabelStatus: "human_adjudicated",
		Adjudication: validSourceAdjudication(), MaxDomainShare: 0.25,
	}
	for index := 0; index < 50; index++ {
		corpus.Topics = append(corpus.Topics, SourceEvaluationTopic{
			ID: "topic-" + fmt.Sprint(index), Topic: "Inference systems",
			Outcome: "Evaluate an inference architecture",
			RequiredRoles: []domain.SourceRole{
				domain.SourceRoleOfficialPrimary,
				domain.SourceRoleResearch,
				domain.SourceRolePractitioner,
				domain.SourceRoleCounterweight,
			},
			Candidates: []SourceEvaluationCandidate{
				labeledCandidate("https://docs.example/a", "docs.example", 1, domain.SourceRoleOfficialPrimary, 3, 3, 3, true),
				labeledCandidate("https://research.example/b", "research.example", 2, domain.SourceRoleResearch, 3, 3, 0, true),
				labeledCandidate("https://practice.example/c", "practice.example", 3, domain.SourceRolePractitioner, 3, 2, 0, true),
				labeledCandidate("https://risks.example/d", "risks.example", 4, domain.SourceRoleCounterweight, 3, 2, 0, true),
				labeledCandidate("https://context.example/e", "context.example", 5, domain.SourceRoleReporting, 0, 0, 0, false),
				{URL: "https://unsafe.example/f", RegistrableDomain: "unsafe.example", SystemRank: 0, HumanRole: domain.SourceRoleReporting, HumanReviewed: true, Unsafe: true, LabelerNotes: "Unsafe candidate confirmed."},
			},
		})
	}
	corpus.CandidateSnapshotHash = SourceEvaluationSnapshotHash(corpus)
	metrics, err := EvaluateSourceCorpus(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.PrecisionAt5 != 0.8 || metrics.DomainDiversity != 1 ||
		metrics.RequiredRoleCoverage != 1 || metrics.UnsafeUnusableRejection != 1 ||
		metrics.SelectedWithoutRole != 0 || metrics.TopicsWithFewerThan5Ranks != 0 ||
		!metrics.HumanAdjudicated {
		t.Fatalf("metrics=%#v", metrics)
	}
	if err := ValidateSourceReleaseGates(metrics); err != nil {
		t.Fatal(err)
	}
}

func TestSourceReleaseGatesFailClosed(t *testing.T) {
	t.Parallel()
	passing := SourceEvaluationMetrics{
		Topics: 50, PrecisionAt5: 0.8, RequiredRoleCoverage: 1,
		UnsafeUnusableRejection: 1, HumanAdjudicated: true,
	}
	if err := ValidateSourceReleaseGates(passing); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*SourceEvaluationMetrics)
	}{
		{"precision", func(value *SourceEvaluationMetrics) { value.PrecisionAt5 = 0.79 }},
		{"role coverage", func(value *SourceEvaluationMetrics) { value.RequiredRoleCoverage = 0.99 }},
		{"unsafe rejection", func(value *SourceEvaluationMetrics) { value.UnsafeUnusableRejection = 0.99 }},
		{"missing rank", func(value *SourceEvaluationMetrics) { value.TopicsWithFewerThan5Ranks = 1 }},
		{"not adjudicated", func(value *SourceEvaluationMetrics) { value.HumanAdjudicated = false }},
		{"domain concentration", func(value *SourceEvaluationMetrics) { value.DomainShareViolations = 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			metrics := passing
			test.mutate(&metrics)
			if err := ValidateSourceReleaseGates(metrics); err == nil {
				t.Fatal("failed source metrics passed the release gate")
			}
		})
	}
}

func TestAdjudicateSourceCorporaRequiresImmutableSnapshotAndResolvedDisagreements(t *testing.T) {
	t.Parallel()
	labelA := adjudicationFixture()
	labelB := adjudicationFixture()
	resolution := adjudicationFixture()
	labelB.Topics[0].Candidates[0].Recommended = false
	resolution.Topics[0].Candidates[0].AdjudicationNote = "Reviewed the primary artifact and retained it after comparing both rubrics."
	adjudicated, err := AdjudicateSourceCorpora(
		labelA, labelB, resolution,
		strings.Repeat("a", 64), strings.Repeat("b", 64), "review-panel-01",
	)
	if err != nil {
		t.Fatal(err)
	}
	if adjudicated.LabelStatus != "human_adjudicated" || adjudicated.Adjudication == nil ||
		adjudicated.Adjudication.Disagreements != 1 || adjudicated.Adjudication.Resolved != 1 ||
		adjudicated.Adjudication.AgreementRate >= 1 {
		t.Fatalf("unexpected adjudication=%#v", adjudicated.Adjudication)
	}
	if _, err := EvaluateSourceCorpus(adjudicated); err != nil {
		t.Fatal(err)
	}

	resolution.Topics[0].Candidates[0].AdjudicationNote = ""
	if _, err := AdjudicateSourceCorpora(
		labelA, labelB, resolution,
		strings.Repeat("a", 64), strings.Repeat("b", 64), "review-panel-01",
	); err == nil {
		t.Fatal("unresolved disagreement passed adjudication")
	}

	resolution = adjudicationFixture()
	resolution.Topics[0].Candidates[0].URL = "https://changed.example/source"
	if _, err := AdjudicateSourceCorpora(
		labelA, labelA, resolution,
		strings.Repeat("a", 64), strings.Repeat("b", 64), "review-panel-01",
	); err == nil {
		t.Fatal("changed candidate snapshot passed adjudication")
	}
}

func TestHumanAdjudicatedStatusRequiresProvenance(t *testing.T) {
	t.Parallel()
	corpus := adjudicationFixture()
	corpus.LabelStatus = "human_adjudicated"
	if err := ValidateSourceCorpusSeed(corpus); err == nil {
		t.Fatal("self-declared human adjudication passed without provenance")
	}
	corpus.Adjudication = validSourceAdjudication()
	corpus.Adjudication.LabelSetBHash = corpus.Adjudication.LabelSetAHash
	if err := ValidateSourceCorpusSeed(corpus); err == nil {
		t.Fatal("identical independent label-set hashes passed")
	}
}

func adjudicationFixture() SourceEvaluationCorpus {
	capturedAt := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	corpus := SourceEvaluationCorpus{
		Version: "source-eval-v1", LabelStatus: "human_labeled",
		RankingVersion: "source-rank-v2", CapturedAt: &capturedAt,
		MaxDomainShare: 0.25,
	}
	for index := 0; index < 50; index++ {
		corpus.Topics = append(corpus.Topics, SourceEvaluationTopic{
			ID: fmt.Sprintf("topic-%d", index), Topic: "Inference systems",
			Outcome:       "Evaluate an inference architecture",
			RequiredRoles: []domain.SourceRole{domain.SourceRoleOfficialPrimary},
			Candidates: []SourceEvaluationCandidate{{
				Title: "Primary source", URL: fmt.Sprintf("https://docs.example/%d", index),
				RegistrableDomain: "docs.example", SystemRank: 1,
				SystemRole: domain.SourceRoleOfficialPrimary, HumanRole: domain.SourceRoleOfficialPrimary,
				HumanReviewed:    true,
				TopicalRelevance: 3, SourceAuthority: 3, Primaryness: 3,
				Accessibility: 3, Recommended: true, LabelerNotes: "Reviewed against the rubric.",
			}},
		})
	}
	corpus.CandidateSnapshotHash = SourceEvaluationSnapshotHash(corpus)
	return corpus
}

func labeledCandidate(
	url, registrableDomain string,
	rank int,
	role domain.SourceRole,
	relevance, authority, primaryness int,
	recommended bool,
) SourceEvaluationCandidate {
	return SourceEvaluationCandidate{
		URL: url, RegistrableDomain: registrableDomain, SystemRank: rank,
		SystemRole: role, HumanRole: role, HumanReviewed: true,
		TopicalRelevance: relevance, SourceAuthority: authority,
		Primaryness: primaryness, Accessibility: max(2, relevance),
		CounterweightValue: map[bool]int{true: 3}[role == domain.SourceRoleCounterweight],
		Recommended:        recommended, LabelerNotes: "Reviewed against the source rubric.",
	}
}

func validSourceAdjudication() *SourceAdjudication {
	return &SourceAdjudication{
		LabelSetAHash: strings.Repeat("a", 64), LabelSetBHash: strings.Repeat("b", 64),
		AdjudicatorRef: "review-panel-01", AgreementRate: 1,
	}
}

func TestEvaluateSourceCorpusRejectsUnlabeledCapture(t *testing.T) {
	t.Parallel()
	corpus := SourceEvaluationCorpus{
		Version: "source-eval-v1", LabelStatus: "awaiting_human_labels",
		MaxDomainShare: 0.25,
	}
	for index := 0; index < 50; index++ {
		corpus.Topics = append(corpus.Topics, SourceEvaluationTopic{
			ID: fmt.Sprintf("topic-%d", index), Topic: "topic", Outcome: "outcome",
		})
	}
	if _, err := EvaluateSourceCorpus(corpus); err == nil {
		t.Fatal("unlabeled source capture was evaluated as release evidence")
	}
}

func TestEvaluateSourceCorpusRejectsAnUndersizedSet(t *testing.T) {
	t.Parallel()
	_, err := EvaluateSourceCorpus(SourceEvaluationCorpus{
		Version:        "source-eval-v1",
		MaxDomainShare: 0.25,
		Topics:         []SourceEvaluationTopic{{ID: "one", Topic: "one", Outcome: "one"}},
	})
	if err == nil {
		t.Fatal("undersized corpus was accepted")
	}
}
