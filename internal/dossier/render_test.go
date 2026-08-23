package dossier

import (
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestRenderHTMLUsesNeutralInlineDefaults(t *testing.T) {
	t.Parallel()
	exploration := "## Beyond the sources\n\nA speculative analogy."
	dossier := domain.Dossier{
		Date:     "2026-08-12",
		Title:    "A bounded lesson",
		Lesson:   "## Mental model\n\nA useful mechanism [S1].\n\nInline `code` stays neutral.",
		Practice: "## Retrieval practice\n\n1. What is the mechanism?",
		Learning: domain.LearningContract{
			EvidenceStatus: domain.EvidenceSourceBounded,
			Limitations: []domain.EvidenceClaim{{
				ID: "limitation-1", Text: "The evidence does not establish causality.", SourceIDs: []string{"S1"},
			}},
		},
		Exploration: &exploration,
		Quality: domain.QualityReport{Score: 97, Metrics: map[string]int{
			"enrichedSources": 1, "retrievalQuestions": 1,
		}},
	}
	output := RenderHTML(dossier, "https://example.com/d/1/slug")
	for _, oldToken := range []string{
		"047857", "f0c36a", "fff8e8", "9a5b13", "7c5a2d",
		"f5f5f4", "0f172a", "78716c", "e7e5e4", "border-radius:999px",
	} {
		if strings.Contains(output, oldToken) {
			t.Fatalf("RenderHTML still carries old palette token %q", oldToken)
		}
	}
	for _, expected := range []string{
		"background:#fff", "color:#2e3238", "background:#151719", "color:#5c626a",
		"background:rgba(21,23,25,.06)", "border-radius:3px",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("RenderHTML missing neutral default %q", expected)
		}
	}
}

func TestRenderHTMLPinsNativeReadingStackAndMetrics(t *testing.T) {
	t.Parallel()
	dossier := domain.Dossier{
		Date:   "2026-08-12",
		Title:  "A bounded lesson",
		Lesson: "## Mental model\n\nA useful mechanism.",
		Learning: domain.LearningContract{
			EvidenceStatus: domain.EvidenceSourceBounded,
		},
	}
	output := RenderHTML(dossier, "")

	for name, fragment := range map[string]string{
		"native sans stack": `-apple-system,system-ui,'Segoe UI',sans-serif`,
		"16px/1.65 base":    `font-size:16px;line-height:1.65`,
		"white page":        `background:#fff;color:#2e3238`,
		"760px outer main":  `max-width:760px;margin:0 auto;padding:32px 20px`,
		"720px content":     `box-sizing:border-box`,
		"h1 30/36/700":      `font-size:30px;line-height:36px;font-weight:700`,
		"h1 28px bottom":    `margin:0 0 28px`,
		"h3 18.72px bold":   `margin:24px 0 10px;font-size:18.72px;line-height:1.4;font-weight:700`,
		"paragraph flow":    `margin:0 0 14px;line-height:1.65`,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("RenderHTML lost the %s (%q)", name, fragment)
		}
	}
	for _, forbidden := range []string{
		"Iowan", "Palatino", "Georgia", "Manrope", "Avenir", "ui-sans-serif", "BlinkMacSystemFont",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("RenderHTML reintroduced the font %q", forbidden)
		}
	}
}

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
