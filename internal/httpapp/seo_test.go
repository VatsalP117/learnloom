package httpapp

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	// Shaped like the real current marketing document (web/marketing.html).
	currentDocument := []byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="Give Learnloom a topic. It builds a learning path and gives every lesson a lasting home at your own Learnloom address." />
    <meta name="theme-color" content="#dcecf4" />
    <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
    <title>Learnloom | Give us a topic. We’ll build your learning home.</title>
    <style>html, body, #root { min-width: 320px; min-height: 100%; }</style>
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>`)
	// A drifted legacy document must normalize identically: decoration is
	// structural, not tied to obsolete copy.
	legacyDrifted := []byte(`<!doctype html><head>` +
		`<meta name="description" content="Stay current without rebuilding context. Give Learnloom a topic; it ranks useful sources, teaches the next concept, and revisits it before it fades.">` +
		`<title>Learnloom · Knowledge Dossiers</title></head><body><div id="root"></div></body>`)

	for _, tc := range []struct {
		name  string
		input []byte
	}{
		{name: "current marketing document", input: currentDocument},
		{name: "legacy drifted document", input: legacyDrifted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := string(decorateMarketingIndex(tc.input, "https://learnloom.blog"))
			output = string(decorateMarketingIndex([]byte(output), "https://learnloom.blog"))

			for _, expected := range []string{
				`<title>Learnloom | Stay current. Build understanding that compounds.</title>`,
				`<meta name="description" content="For professionals in fast-moving fields, Learnloom turns credible current sources into connected lessons, active recall, and a learning path that compounds.">`,
				`rel="canonical" href="https://learnloom.blog/"`,
				`property="og:title" content="Learnloom | Stay current. Build understanding that compounds."`,
				`property="og:description" content="For professionals in fast-moving fields, Learnloom turns credible current sources into connected lessons, active recall, and a learning path that compounds."`,
				`property="og:url" content="https://learnloom.blog/"`,
				`property="og:image" content="https://learnloom.blog/social-preview.png"`,
				`name="twitter:card" content="summary_large_image"`,
				`name="twitter:title" content="Learnloom | Stay current. Build understanding that compounds."`,
				`name="twitter:image" content="https://learnloom.blog/social-preview.png"`,
			} {
				if !strings.Contains(output, expected) {
					t.Fatalf("decorated index missing %q: %s", expected, output)
				}
			}
			if strings.Count(output, "<title>") != 1 {
				t.Fatalf("decorated index must have exactly one title: %s", output)
			}
			if strings.Count(output, `name="description"`) != 1 {
				t.Fatalf("decorated index must have exactly one description: %s", output)
			}
			if strings.Count(output, `rel="canonical"`) != 1 {
				t.Fatalf("decorated index must have exactly one canonical: %s", output)
			}
			if strings.Count(output, `type="application/ld+json"`) != 1 {
				t.Fatalf("decorated index must inject schema metadata once: %s", output)
			}
			if strings.Count(output, `property="og:title"`) != 1 ||
				strings.Count(output, `name="twitter:card"`) != 1 {
				t.Fatalf("decorated index must be idempotent: %s", output)
			}
			if strings.Contains(output, "seo-prerender") {
				t.Fatalf("marketing index must not inject temporary visible UI: %s", output)
			}
		})
	}
}

func TestMarketingHomepageUsesDedicatedFrontendDocument(t *testing.T) {
	t.Parallel()
	static := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte(`<!doctype html><title>Product app</title><div id="app-entry"></div>`)},
		"marketing.html": &fstest.MapFile{Data: []byte(`<!doctype html><html><head><title>Learnloom | Stay current. Build understanding that compounds.</title></head><body><div id="marketing-entry"></div></body></html>`)},
	}
	server := &Server{cfg: Config{
		Static:     fs.FS(static),
		ApexOrigin: "https://learnloom.blog",
	}}
	request := httptest.NewRequest(http.MethodGet, "https://learnloom.blog/", nil)
	response := httptest.NewRecorder()

	server.serveMarketingIndex(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `id="marketing-entry"`) {
		t.Fatalf("marketing response = %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), `id="app-entry"`) || strings.Contains(response.Body.String(), "seo-prerender") {
		t.Fatalf("marketing response used the shared app bootstrap: %s", response.Body.String())
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
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Cloudflare-CDN-Cache-Control"); got != "public, max-age=300, stale-while-revalidate=60" {
		t.Fatalf("Cloudflare-CDN-Cache-Control = %q", got)
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

func TestAppHostRobotsDisallowsCrawlers(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{}}

	getRequest := httptest.NewRequest(http.MethodGet, "https://app.learnloom.blog/robots.txt", nil)
	getResponse := httptest.NewRecorder()
	server.handleApp(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d", getResponse.Code)
	}
	if got := getResponse.Body.String(); got != "User-agent: *\nDisallow: /\n" {
		t.Fatalf("GET body = %q", got)
	}
	if got := getResponse.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := getResponse.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("Cache-Control = %q", got)
	}

	headRequest := httptest.NewRequest(http.MethodHead, "https://app.learnloom.blog/robots.txt", nil)
	headResponse := httptest.NewRecorder()
	server.handleApp(headResponse, headRequest)
	if headResponse.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d", headResponse.Code)
	}
	if headResponse.Body.Len() != 0 {
		t.Fatalf("HEAD body = %q", headResponse.Body.String())
	}

	postRequest := httptest.NewRequest(http.MethodPost, "https://app.learnloom.blog/robots.txt", nil)
	postResponse := httptest.NewRecorder()
	server.handleApp(postResponse, postRequest)
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d", postResponse.Code)
	}
}

func TestLegalPagesGetCrawlMetadataWithNoindex(t *testing.T) {
	t.Parallel()
	// Shaped like the real shared SPA document (web/index.html) served for
	// legal paths on the apex host.
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(`<!doctype html><html lang="en"><head>` +
			`<meta name="description" content="Stay current without rebuilding context. Give Learnloom a topic; it ranks useful sources, teaches the next concept, and revisits it before it fades.">` +
			`<title>Learnloom | Give us a topic. We’ll build the learning path.</title>` +
			`</head><body><div id="root"></div><script type="module" src="/src/main.tsx"></script></body></html>`)},
	}
	server := &Server{cfg: Config{
		Static:     fs.FS(static),
		ApexOrigin: "https://learnloom.blog",
	}}

	for _, tc := range []struct {
		path        string
		title       string
		description string
		h1          string
	}{
		{
			path:        "/privacy",
			title:       "Privacy Policy | Learnloom",
			description: "Learn how Learnloom handles account, learning, source, publishing, delivery, and technical information, and the choices available to you.",
			h1:          "Privacy Policy",
		},
		{
			path:        "/terms",
			title:       "Terms of Service | Learnloom",
			description: "Read the terms governing Learnloom accounts, source-grounded learning, publishing, subscriptions, acceptable use, and service availability.",
			h1:          "Terms of Service",
		},
	} {
		t.Run(tc.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://learnloom.blog"+tc.path, nil)
			response := httptest.NewRecorder()
			server.handleApex(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("%s status = %d", tc.path, response.Code)
			}
			if got := response.Header().Get("X-Robots-Tag"); got != "noindex, follow" {
				t.Fatalf("%s X-Robots-Tag = %q", tc.path, got)
			}
			body := response.Body.String()
			if strings.Count(body, "<title>") != 1 ||
				!strings.Contains(body, "<title>"+tc.title+"</title>") {
				t.Fatalf("%s title mismatch: %s", tc.path, body)
			}
			if strings.Count(body, `name="description"`) != 1 {
				t.Fatalf("%s description count = %d: %s", tc.path, strings.Count(body, `name="description"`), body)
			}
			if !strings.Contains(body, `<meta name="description" content="`+tc.description+`">`) {
				t.Fatalf("%s missing description copy: %s", tc.path, body)
			}
			canonical := "https://learnloom.blog" + tc.path
			for _, expected := range []string{
				`rel="canonical" href="` + canonical + `"`,
				`property="og:title" content="` + tc.title + `"`,
				`property="og:description" content="` + tc.description + `"`,
				`property="og:url" content="` + canonical + `"`,
				`property="og:image" content="https://learnloom.blog/social-preview.png"`,
				`name="twitter:card" content="summary_large_image"`,
				`name="twitter:title" content="` + tc.title + `"`,
				`name="twitter:image" content="https://learnloom.blog/social-preview.png"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("%s missing %q: %s", tc.path, expected, body)
				}
			}
			if !strings.Contains(
				body,
				`<div id="root"><h1>`+tc.h1+`</h1><p>`+tc.description+`</p>`,
			) {
				t.Fatalf("%s missing visible #root fallback: %s", tc.path, body)
			}
			if strings.Count(body, `rel="canonical"`) != 1 {
				t.Fatalf("%s must have exactly one canonical: %s", tc.path, body)
			}
		})
	}
}

func TestMarketingHomepageCachePolicy(t *testing.T) {
	t.Parallel()
	static := fstest.MapFS{
		"marketing.html": &fstest.MapFile{Data: []byte(`<!doctype html><html><head><title>Learnloom | Stay current. Build understanding that compounds.</title></head><body><div id="root"></div></body></html>`)},
	}
	server := &Server{cfg: Config{
		Static:     fs.FS(static),
		ApexOrigin: "https://learnloom.blog",
	}}
	request := httptest.NewRequest(http.MethodGet, "https://learnloom.blog/", nil)
	response := httptest.NewRecorder()

	server.serveMarketingIndex(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	browserMaxAge := cacheMaxAgeSeconds(response.Header().Get("Cache-Control"))
	cdnMaxAge := cacheMaxAgeSeconds(response.Header().Get("Cloudflare-CDN-Cache-Control"))
	if browserMaxAge <= 0 || cdnMaxAge <= browserMaxAge {
		t.Fatalf(
			"expected a short public browser cache and a longer CDN cache: Cache-Control=%q Cloudflare-CDN-Cache-Control=%q",
			response.Header().Get("Cache-Control"),
			response.Header().Get("Cloudflare-CDN-Cache-Control"),
		)
	}
}

func cacheMaxAgeSeconds(value string) int {
	for _, directive := range strings.Split(value, ",") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			if seconds, err := strconv.Atoi(strings.TrimPrefix(directive, "max-age=")); err == nil {
				return seconds
			}
		}
	}
	return -1
}
