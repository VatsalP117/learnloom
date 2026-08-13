package httpapp

import (
	"image"
	_ "image/png"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
)

func TestSocialPreviewMetadataIsCompleteAndStatic(t *testing.T) {
	t.Parallel()

	metadata := renderSocialImageMetadata("https://learnloom.blog/")
	assertSocialPreviewMetadata(t, metadata)
	for _, forbidden := range []string{
		"learner@example.com",
		"source excerpt",
		"unpublished title",
	} {
		if strings.Contains(metadata, forbidden) {
			t.Fatalf("social image metadata contains private content %q", forbidden)
		}
	}
}

func TestSocialPreviewAssetIsExactOpenGraphPNG(t *testing.T) {
	t.Parallel()

	file, err := os.Open("../../web/public/social-preview.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" || config.Width != 1200 || config.Height != 630 {
		t.Fatalf(
			"social preview = %s %dx%d; want png 1200x630",
			format,
			config.Width,
			config.Height,
		)
	}
}

func TestRepresentativeApexPagesHaveSocialPreviews(t *testing.T) {
	t.Parallel()

	marketingInput := []byte(`<!doctype html><html><head>` +
		`<meta name="description" content="Stay current without rebuilding context. Give Learnloom a topic; it ranks useful sources, teaches the next concept, and revisits it before it fades.">` +
		`<title>Learnloom · Knowledge Dossiers</title></head><body><div id="root"></div></body></html>`)
	homepage := string(decorateMarketingIndex(
		marketingInput,
		"https://learnloom.blog",
		"https://app.learnloom.blog",
	))

	guide, ok := authorityPageForPath("/guides/how-to-remember-what-you-read")
	if !ok {
		t.Fatal("representative guide is missing")
	}
	guideDocument := renderAuthorityDocument(
		guide,
		"https://learnloom.blog"+guide.Path,
		"https://app.learnloom.blog",
	)
	examplesDocument := renderExamplesDocument(
		nil,
		"https://learnloom.blog/examples",
		"https://app.learnloom.blog",
		"learnloom.blog",
	)

	for name, document := range map[string]string{
		"homepage": homepage,
		"guide":    guideDocument,
		"examples": examplesDocument,
	} {
		t.Run(name, func(t *testing.T) {
			assertSocialPreviewMetadata(t, document)
		})
	}
}

func TestLearnerHomeAndPublishedDossierHaveSafeSocialPreviews(t *testing.T) {
	t.Parallel()

	server := &Server{cfg: Config{ApexOrigin: "https://learnloom.blog"}}
	homeResponse := httptest.NewRecorder()
	server.sendReadingPage(
		homeResponse,
		httptest.NewRequest(http.MethodGet, "https://maya.learnloom.blog/", nil),
		"Maya's Garden",
		"Cities, systems, and evidence.",
		"https://maya.learnloom.blog",
		"<main>Public learner home</main>",
		true,
	)
	assertSocialPreviewMetadata(t, homeResponse.Body.String())

	site := domain.PersonalSite{DisplayName: "Maya & Co"}
	issue := store.PublicIssue{
		Title:          "Why cities remember rivers",
		NewsletterName: "Urban systems",
		CompletedAt:    time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC),
	}
	dossierMetadata := renderPublicDossierMetadata(
		site,
		issue,
		"https://maya.learnloom.blog/d/dossier-123/city-rivers",
		"https://maya.learnloom.blog",
		"https://learnloom.blog",
		"A source-grounded Knowledge Dossier about Urban systems.",
	)
	assertSocialPreviewMetadata(t, dossierMetadata)
	for _, expected := range []string{
		`property="og:title" content="Why cities remember rivers"`,
		`"name":"Maya \u0026 Co"`,
		`"articleSection":"Urban systems"`,
	} {
		if !strings.Contains(dossierMetadata, expected) {
			t.Fatalf("published Dossier metadata missing permitted public field %q", expected)
		}
	}
}

func TestApexServesStableSocialPreviewImage(t *testing.T) {
	t.Parallel()

	static := fstest.MapFS{
		"social-preview.png": &fstest.MapFile{Data: []byte("\x89PNG\r\n\x1a\n")},
	}
	server := &Server{cfg: Config{Static: fs.FS(static)}}
	request := httptest.NewRequest(
		http.MethodGet,
		"https://learnloom.blog/social-preview.png",
		nil,
	)
	response := httptest.NewRecorder()

	server.handleApex(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("social image status = %d", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("social image Content-Type = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=300, must-revalidate" {
		t.Fatalf("social image Cache-Control = %q", got)
	}
}

func assertSocialPreviewMetadata(t *testing.T, document string) {
	t.Helper()
	for _, expected := range []string{
		`property="og:image" content="https://learnloom.blog/social-preview.png"`,
		`property="og:image:secure_url" content="https://learnloom.blog/social-preview.png"`,
		`property="og:image:type" content="image/png"`,
		`property="og:image:width" content="1200"`,
		`property="og:image:height" content="630"`,
		`property="og:image:alt" content="Learnloom — give us a topic, and we’ll build the learning path."`,
		`name="twitter:card" content="summary_large_image"`,
		`name="twitter:image" content="https://learnloom.blog/social-preview.png"`,
		`name="twitter:image:alt" content="Learnloom — give us a topic, and we’ll build the learning path."`,
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("social preview metadata missing %q", expected)
		}
	}
}
