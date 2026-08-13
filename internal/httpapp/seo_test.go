package httpapp

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSEOPageCatalogHasUniqueIndexablePages(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, page := range seoPages {
		if page.Path == "" || page.Title == "" || page.Description == "" ||
			len(page.Body) < 2 || len(page.Steps) < 3 || len(page.Benefits) < 3 {
			t.Fatalf("SEO page is incomplete: %#v", page)
		}
		if seen[page.Path] {
			t.Fatalf("duplicate SEO path %q", page.Path)
		}
		seen[page.Path] = true

		document := renderSEODocument(
			page,
			"https://learnloom.blog"+page.Path,
			"https://app.learnloom.blog",
		)
		for _, expected := range []string{
			`<html lang="en">`,
			`<h1>` + page.Title + `</h1>`,
			`rel="canonical" href="https://learnloom.blog` + page.Path + `"`,
			`type="application/ld+json"`,
			`href="https://app.learnloom.blog/sign-up"`,
		} {
			if !strings.Contains(document, expected) {
				t.Fatalf("%s missing %q", page.Path, expected)
			}
		}
	}
}

func TestApexRobotsAndSitemapAdvertiseCanonicalPages(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{ApexOrigin: "https://learnloom.blog"}}

	robotsRequest := httptest.NewRequest(http.MethodGet, "https://learnloom.blog/robots.txt", nil)
	robotsResponse := httptest.NewRecorder()
	server.renderApexRobots(robotsResponse, robotsRequest)
	if !strings.Contains(
		robotsResponse.Body.String(),
		"Sitemap: https://learnloom.blog/sitemap.xml",
	) {
		t.Fatalf("robots.txt did not advertise the sitemap: %s", robotsResponse.Body.String())
	}

	sitemapRequest := httptest.NewRequest(http.MethodGet, "https://learnloom.blog/sitemap.xml", nil)
	sitemapResponse := httptest.NewRecorder()
	server.renderApexSitemap(sitemapResponse, sitemapRequest)
	for _, expected := range []string{
		"<loc>https://learnloom.blog/</loc>",
		"<loc>https://learnloom.blog/solutions/remember-what-you-read</loc>",
		"<loc>https://learnloom.blog/product/ai-learning-assistant</loc>",
		"<loc>https://learnloom.blog/guides/how-to-remember-what-you-read</loc>",
		"<loc>https://learnloom.blog/how-learnloom-works</loc>",
	} {
		if !strings.Contains(sitemapResponse.Body.String(), expected) {
			t.Fatalf("sitemap missing %q: %s", expected, sitemapResponse.Body.String())
		}
	}
}

func TestMarketingIndexGetsCanonicalMetadata(t *testing.T) {
	t.Parallel()
	input := []byte(`<!doctype html><html><head>` +
		`<meta name="description" content="Stay current without rebuilding context. Give Learnloom a topic; it ranks useful sources, teaches the next concept, and revisits it before it fades.">` +
		`<title>Learnloom · Knowledge Dossiers</title></head><body><div id="root"></div></body></html>`)

	output := string(decorateMarketingIndex(
		input,
		"https://learnloom.blog",
		"https://app.learnloom.blog",
	))

	for _, expected := range []string{
		`<title>Learnloom | Give us a topic. We’ll build the learning path.</title>`,
		`rel="canonical" href="https://learnloom.blog/"`,
		`property="og:title"`,
		`type="application/ld+json"`,
		`Give us a topic. We’ll build the learning path.`,
		`href="/solutions/remember-what-you-read"`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("decorated index missing %q: %s", expected, output)
		}
	}
	if strings.Count(output, `name="description"`) != 1 {
		t.Fatalf("decorated index should have one description: %s", output)
	}
}

func TestAppIndexIsNoIndex(t *testing.T) {
	t.Parallel()
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Learnloom</title>")},
	}
	server := &Server{cfg: Config{Static: fs.FS(static)}}
	request := httptest.NewRequest(http.MethodGet, "https://app.learnloom.blog/", nil)
	response := httptest.NewRecorder()

	server.serveAppIndex(response, request)

	if got := response.Header().Get("X-Robots-Tag"); got != "noindex, nofollow" {
		t.Fatalf("X-Robots-Tag = %q", got)
	}
}

func TestApexSEOPageTrailingSlashRedirectsToCanonicalURL(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{ApexOrigin: "https://learnloom.blog"}}
	request := httptest.NewRequest(
		http.MethodGet,
		"https://learnloom.blog/solutions/remember-what-you-read/",
		nil,
	)
	response := httptest.NewRecorder()

	server.handleApex(response, request)

	if response.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d", response.Code)
	}
	if location := response.Header().Get("Location"); location !=
		"https://learnloom.blog/solutions/remember-what-you-read" {
		t.Fatalf("Location = %q", location)
	}
}

func TestApexAuthorityPageTrailingSlashRedirectsToCanonicalURL(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{ApexOrigin: "https://learnloom.blog"}}
	request := httptest.NewRequest(
		http.MethodGet,
		"https://learnloom.blog/guides/how-to-remember-what-you-read/",
		nil,
	)
	response := httptest.NewRecorder()

	server.handleApex(response, request)

	if response.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d", response.Code)
	}
	if location := response.Header().Get("Location"); location !=
		"https://learnloom.blog/guides/how-to-remember-what-you-read" {
		t.Fatalf("Location = %q", location)
	}
}

func TestNormalizeFeaturedSitesRejectsInvalidAndReservedNames(t *testing.T) {
	t.Parallel()
	got, err := normalizeFeaturedSites([]string{"Maya", "ada", "maya"})
	if err != nil || strings.Join(got, ",") != "maya,ada" {
		t.Fatalf("normalizeFeaturedSites() = %#v, %v", got, err)
	}
	for _, values := range [][]string{{"a"}, {"api"}, {"bad--name"}} {
		if _, err := normalizeFeaturedSites(values); err == nil {
			t.Fatalf("normalizeFeaturedSites(%#v) succeeded", values)
		}
	}
	tooMany := make([]string, 25)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("site-%02d", index)
	}
	if _, err := normalizeFeaturedSites(tooMany); err == nil {
		t.Fatal("normalizeFeaturedSites accepted more than 24 usernames")
	}
}

func TestApexExamplesTrailingSlashRedirectsToCanonicalURL(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{ApexOrigin: "https://learnloom.blog"}}
	request := httptest.NewRequest(
		http.MethodGet,
		"https://learnloom.blog/examples/",
		nil,
	)
	response := httptest.NewRecorder()

	server.handleApex(response, request)

	if response.Code != http.StatusPermanentRedirect ||
		response.Header().Get("Location") != "https://learnloom.blog/examples" {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
}
