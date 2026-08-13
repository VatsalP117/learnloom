package dossier

import (
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestRenderKeepsInternalAuditOutOfLearnerOutput(t *testing.T) {
	t.Parallel()
	dossier := domain.Dossier{
		Date:     "2026-08-12",
		Title:    "A bounded lesson",
		Lesson:   "## Mental model\n\nA useful mechanism [S1].",
		Critique: "INTERNAL AUDIT: rewrite the weak causal claim [S1].",
		Practice: "## Retrieval practice\n\n1. What is the mechanism?",
		Learning: domain.LearningContract{
			EvidenceStatus: domain.EvidenceSourceBounded,
			Limitations: []domain.EvidenceClaim{{
				ID: "limitation-1", Text: "The evidence does not establish causality.", SourceIDs: []string{"S1"},
			}},
		},
		Quality: domain.QualityReport{Score: 97, Metrics: map[string]int{
			"enrichedSources": 1, "retrievalQuestions": 1,
		}},
	}

	for name, output := range map[string]string{
		"markdown": RenderMarkdown(dossier),
		"html":     RenderHTML(dossier, ""),
	} {
		if strings.Contains(output, "INTERNAL AUDIT") {
			t.Fatalf("%s exposed the internal skeptical audit", name)
		}
		if !strings.Contains(output, "The evidence does not establish causality") {
			t.Fatalf("%s omitted the learner-facing limitation", name)
		}
		if strings.Contains(output, "97/100") {
			t.Fatalf("%s exposed a numerical certainty proxy", name)
		}
	}
}
