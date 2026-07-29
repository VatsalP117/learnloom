package httpapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
)

func TestRenderExamplesDocumentUsesOnlyProvidedCuratedEntries(t *testing.T) {
	t.Parallel()
	examples := []featuredExample{{
		Site: domain.PersonalSite{
			Username:       "maya",
			DisplayName:    "Maya & Co",
			Description:    "Cities, systems, and evidence.",
			Visibility:     domain.SitePublic,
			SearchIndexing: true,
		},
		Issues: []store.PublicIssue{{
			PublicID:       "dossier-123",
			PublicSlug:     "city-rivers",
			Title:          "Why cities remember rivers",
			NewsletterName: "Urban systems",
			CompletedAt:    time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC),
		}},
	}}

	document := renderExamplesDocument(
		examples,
		"https://learnloom.blog/examples",
		"https://app.learnloom.blog",
		"learnloom.blog",
	)

	for _, expected := range []string{
		`<h2>Maya &amp; Co</h2>`,
		`https://maya.learnloom.blog/d/dossier-123/city-rivers`,
		`Why cities remember rivers`,
		`rel="canonical" href="https://learnloom.blog/examples"`,
		`"@type":"CollectionPage"`,
		`"url":"https://maya.learnloom.blog"`,
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("examples document missing %q", expected)
		}
	}
}

func TestRenderEmptyExamplesDocumentExplainsCuration(t *testing.T) {
	t.Parallel()
	document := renderExamplesDocument(
		nil,
		"https://learnloom.blog/examples",
		"https://app.learnloom.blog",
		"learnloom.blog",
	)
	if !strings.Contains(document, "Public sites are never added automatically") {
		t.Fatal("empty gallery does not explain the curation boundary")
	}
}

func TestEmptyExamplesPageIsNoIndex(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{
		ApexOrigin: "https://learnloom.blog",
		AppOrigin:  "https://app.learnloom.blog",
		RootDomain: "learnloom.blog",
	}}
	request := httptest.NewRequest(http.MethodGet, "https://learnloom.blog/examples", nil)
	response := httptest.NewRecorder()

	server.renderExamplesPage(response, request)

	if got := response.Header().Get("X-Robots-Tag"); got != "noindex, follow" {
		t.Fatalf("X-Robots-Tag = %q", got)
	}
}
