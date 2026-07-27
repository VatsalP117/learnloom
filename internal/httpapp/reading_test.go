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

func TestRenderIssueCardsUsesPublicRouteAndEscapesContent(t *testing.T) {
	t.Parallel()

	output := renderIssueCards([]store.PublicIssue{{
		PublicID:       "dossier-123",
		PublicSlug:     "patient-learning",
		Title:          `Patient <learning>`,
		NewsletterName: `Systems & society`,
		CompletedAt:    time.Date(2026, time.July, 27, 9, 30, 0, 0, time.UTC),
	}})

	for _, expected := range []string{
		`href="/d/dossier-123/patient-learning"`,
		`Patient &lt;learning&gt;`,
		`Systems &amp; society`,
		`datetime="2026-07-27T09:30:00Z"`,
		`27 Jul 2026`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("renderIssueCards() missing %q in %s", expected, output)
		}
	}
}

func TestDecoratePublicDossierUpgradesStoredArtifact(t *testing.T) {
	t.Parallel()

	document := `<!doctype html><html><head><title>Stored</title></head>` +
		`<body style="margin:0;background:#f5f5f4;color:#0f172a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">` +
		`<main><div>Lesson</div></main></body></html>`
	site := domain.PersonalSite{DisplayName: `Vatsal & Co`}
	issue := store.PublicIssue{
		NewsletterName:       `Cities < rivers`,
		NewsletterPublicSlug: `cities-and-rivers`,
	}

	output := decoratePublicDossier(document, site, issue)

	for _, expected := range []string{
		`<style>` + readingArticleCSS + `</style></head>`,
		`<body class="public-dossier">`,
		`Vatsal &amp; Co`,
		`href="/topics/cities-and-rivers"`,
		`Cities &lt; rivers`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("decoratePublicDossier() missing %q", expected)
		}
	}
	if strings.Contains(output, `background:#f5f5f4`) {
		t.Fatal("decoratePublicDossier() left the legacy body styles in place")
	}
}

func TestReadingEmptyEscapesCopy(t *testing.T) {
	t.Parallel()

	output := renderReadingEmpty(`Nothing <yet>`, `Wait & see`)
	if !strings.Contains(output, `Nothing &lt;yet&gt;`) ||
		!strings.Contains(output, `Wait &amp; see`) {
		t.Fatalf("renderReadingEmpty() did not escape content: %s", output)
	}
}

func TestReadingNotFoundUsesBrandedResponsiveDocument(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "https://missing.learnloom.blog/", nil)
	response := httptest.NewRecorder()
	(&Server{}).readingNotFound(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("readingNotFound() status = %d", response.Code)
	}
	for _, expected := range []string{
		`name="viewport"`,
		`class="not-found"`,
		`This page has wandered off the path.`,
		`https://learnloom.blog`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("readingNotFound() missing %q", expected)
		}
	}
}
