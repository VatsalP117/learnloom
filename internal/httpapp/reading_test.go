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

func TestDecoratePublicGrowthShowsPathRelatedSharingAndAttributedCTA(t *testing.T) {
	t.Parallel()
	document := decoratePublicGrowth(
		"<html><body><main>Lesson</main></body></html>",
		domain.PersonalSite{DisplayName: "Alan"},
		store.PublicIssue{
			PublicID: "dossier-1", Title: "A useful lesson",
			NewsletterOutcome: "Make better evaluation decisions.",
			NewsletterTopic:   "AI evaluation", NewsletterPublicSlug: "ai-evaluation",
		},
		"https://alan.learnloom.blog/d/dossier-1/a-useful-lesson",
		[]store.PublicIssue{{
			PublicID: "dossier-2", PublicSlug: "next", Title: "The next lesson",
			CompletedAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		}},
	)
	for _, expected := range []string{
		"A path maintained by Alan",
		"Make better evaluation decisions.",
		"The next lesson",
		"Share this Dossier",
		"Follow this path",
		`method="post" action="/follow/dossier-1"`,
		"Double opt-in",
		"/go/dossier-1/start",
		"/go/dossier-1/linkedin",
		"Start a path like this",
		"private, personalized path",
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("public growth surface missing %q", expected)
		}
	}
}

func TestPublicReferralFingerprintIsStableButDoesNotExposeCookie(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{CSRFSecret: strings.Repeat("s", 32)}}
	first := server.publicReferralFingerprint("visitor-cookie")
	second := server.publicReferralFingerprint("visitor-cookie")
	other := server.publicReferralFingerprint("another-cookie")
	if len(first) != 64 || first != second || first == other {
		t.Fatalf("unexpected referral fingerprints %q %q %q", first, second, other)
	}
	if strings.Contains(first, "visitor-cookie") {
		t.Fatal("referral fingerprint exposed the visitor cookie")
	}
}

func TestLikelyAutomatedVisitorExcludesBotsAndPreviews(t *testing.T) {
	t.Parallel()
	for _, userAgent := range []string{"", "Googlebot/2.1", "Slackbot-LinkExpanding", "HeadlessChrome"} {
		if !isLikelyAutomatedVisitor(userAgent) {
			t.Errorf("%q should be classified as automated", userAgent)
		}
	}
	if isLikelyAutomatedVisitor("Mozilla/5.0 Safari/605.1.15") {
		t.Fatal("normal browser was classified as automated")
	}
}

func TestPublicVisitorCookieIsSecureHttpOnlyAndSharedAcrossSubdomains(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{
		RootDomain: "learnloom.blog", CSRFSecret: strings.Repeat("s", 32),
	}}
	request := httptest.NewRequest(http.MethodGet, "https://maya.learnloom.blog/d/id/slug", nil)
	request.RemoteAddr = "203.0.113.10:443"
	response := httptest.NewRecorder()
	fingerprint := server.publicVisitorFingerprint(response, request)
	if len(fingerprint) != 64 {
		t.Fatalf("fingerprint length = %d", len(fingerprint))
	}
	cookie := response.Header().Get("Set-Cookie")
	for _, expected := range []string{"ll_public_ref=", "Domain=learnloom.blog", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(cookie, expected) {
			t.Fatalf("visitor cookie missing %q: %s", expected, cookie)
		}
	}
}

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
	if got := response.Header().Get("X-Robots-Tag"); got != "noindex, follow" {
		t.Fatalf("readingNotFound() robots header = %q", got)
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

func TestReadingIndexabilityHeaders(t *testing.T) {
	t.Parallel()
	server := &Server{}

	indexable := httptest.NewRecorder()
	server.applyReadingHeaders(indexable, true)
	if got := indexable.Header().Get("X-Robots-Tag"); got != "index, follow" {
		t.Fatalf("indexable robots header = %q", got)
	}

	empty := httptest.NewRecorder()
	server.applyReadingHeaders(empty, false)
	if got := empty.Header().Get("X-Robots-Tag"); got != "noindex, follow" {
		t.Fatalf("empty robots header = %q", got)
	}
}

func TestEmptySitemapContainsNoIndexableURLs(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest(http.MethodGet, "https://maya.learnloom.blog/sitemap.xml", nil)
	response := httptest.NewRecorder()

	writeEmptySitemap(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "<url>") ||
		!strings.Contains(response.Body.String(), "<urlset") {
		t.Fatalf("unexpected empty sitemap: %s", response.Body.String())
	}
}

func TestPersonalRobotsAdvertisesSitemapOnlyAfterIndexingOptIn(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest(http.MethodGet, "https://maya.learnloom.blog/robots.txt", nil)

	disabled := httptest.NewRecorder()
	renderPersonalRobots(
		disabled,
		request,
		domain.PersonalSite{SearchIndexing: false},
		"https://maya.learnloom.blog",
	)
	if strings.Contains(disabled.Body.String(), "Sitemap:") {
		t.Fatalf("disabled robots advertised a sitemap: %s", disabled.Body.String())
	}

	enabled := httptest.NewRecorder()
	renderPersonalRobots(
		enabled,
		request,
		domain.PersonalSite{SearchIndexing: true},
		"https://maya.learnloom.blog",
	)
	if !strings.Contains(
		enabled.Body.String(),
		"Sitemap: https://maya.learnloom.blog/sitemap.xml",
	) {
		t.Fatalf("enabled robots omitted the sitemap: %s", enabled.Body.String())
	}
}

func TestRenderHomeHeroShowsEscapedIdentityAndPublicCounts(t *testing.T) {
	t.Parallel()

	output := renderHomeHero(domain.PersonalSite{
		DisplayName: "Maya & Co",
		Description: "Cities, <systems>, and evidence.",
	}, 3, 12)

	for _, expected := range []string{
		`class="reading-hero home-hero"`,
		`<h1>Maya &amp; Co</h1>`,
		`Cities, &lt;systems&gt;, and evidence.`,
		`A public learning archive`,
		`<dt>Streams</dt><dd>3</dd>`,
		`<dt>Dossiers</dt><dd>12</dd>`,
		`href="/#topics"`,
		`aria-hidden="true"`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("renderHomeHero() missing %q in %s", expected, output)
		}
	}
	if strings.Contains(output, "Maya & Co") || strings.Contains(output, "<systems>") {
		t.Fatal("renderHomeHero() rendered unescaped site fields")
	}
}

func TestRenderHomeHeroFallsBackToDescriptionWhenAbsent(t *testing.T) {
	t.Parallel()

	output := renderHomeHero(domain.PersonalSite{DisplayName: "Maya"}, 0, 0)
	if !strings.Contains(output, "A durable personal learning archive.") {
		t.Fatalf("renderHomeHero() description fallback missing: %s", output)
	}
}

func TestRenderReadingHeaderAndFooterEscapeSiteFields(t *testing.T) {
	t.Parallel()

	header := renderReadingHeader(domain.PersonalSite{DisplayName: `Ada <Lovelace> & Co`}, "home")
	for _, expected := range []string{
		`class="skip-link" href="#main-content"`,
		`aria-label="Ada &lt;Lovelace&gt; &amp; Co home"`,
		`<strong>Ada &lt;Lovelace&gt; &amp; Co</strong>`,
		`href="/#topics"`,
		`href="/#latest"`,
		`href="/#about"`,
		`aria-label="Personal site navigation"`,
	} {
		if !strings.Contains(header, expected) {
			t.Fatalf("renderReadingHeader() missing %q in %s", expected, header)
		}
	}

	footer := renderReadingFooter(domain.PersonalSite{
		DisplayName: `Ada <Lovelace>`,
		Description: `Notes & <observations> from the field.`,
	})
	for _, expected := range []string{
		`<footer class="site-footer" id="about">`,
		`Ada &lt;Lovelace&gt;`,
		`Notes &amp; &lt;observations&gt; from the field.`,
		`Published with`,
		`A personal learning archive`,
	} {
		if !strings.Contains(footer, expected) {
			t.Fatalf("renderReadingFooter() missing %q in %s", expected, footer)
		}
	}
	if strings.Contains(footer, "<observations>") {
		t.Fatal("renderReadingFooter() rendered unescaped description")
	}
}

func TestReadingSheetsRemainImageLightAndAssetFree(t *testing.T) {
	t.Parallel()

	for name, sheet := range map[string]string{
		"readingCSS":        readingCSS,
		"readingArticleCSS": readingArticleCSS,
	} {
		for _, forbidden := range []string{
			"backdrop-filter",
			"url(",
			"data:image",
			"@import",
			"@font-face",
			"javascript:",
		} {
			if strings.Contains(sheet, forbidden) {
				t.Fatalf("%s reintroduced %q", name, forbidden)
			}
		}
	}
}

func TestReadingSheetsUseOnlyNeutralPalette(t *testing.T) {
	t.Parallel()

	for name, sheet := range map[string]string{
		"readingCSS":        readingCSS,
		"readingArticleCSS": readingArticleCSS,
	} {
		for _, oldToken := range []string{
			"#047857", "#f0c36a", "#fff8e8", "#9a5b13", "#7c5a2d",
			"#496b4c", "#56634b", "#9c4e2c", "#315b44", "#172319", "#17241a",
			"#262e20", "#1d2c22", "#f6f3ea", "#fdfbf4", "#efe9db",
			"gradient", "999px", "moss", "rust",
		} {
			// readingArticleCSS may mention legacy inline values only inside
			// [style*="..."] attribute selectors that neutralize stored artifacts.
			if (strings.HasPrefix(oldToken, "#") || oldToken == "999px") && strings.Contains(sheet, `style*="`) {
				rest := sheet
				mentioned := false
				for {
					index := strings.Index(rest, oldToken)
					if index == -1 {
						break
					}
					mentioned = true
					selectorStart := strings.LastIndex(rest[:index], `style*="`)
					quoteEnd := strings.Index(rest[index+len(oldToken):], `"`)
					if selectorStart == -1 || quoteEnd == -1 {
						t.Fatalf("%s uses old palette token %q outside a legacy override selector", name, oldToken)
					}
					rest = rest[index+len(oldToken):]
				}
				if mentioned {
					continue
				}
			}
			if strings.Contains(sheet, oldToken) {
				t.Fatalf("%s still carries the old palette token %q", name, oldToken)
			}
		}
	}
}

func TestReadingSheetsNeverReturnSerifOrCustomDisplayFonts(t *testing.T) {
	t.Parallel()

	for name, sheet := range map[string]string{
		"readingCSS":        readingCSS,
		"readingArticleCSS": readingArticleCSS,
	} {
		for _, forbidden := range []string{
			"Iowan", "Palatino", "Georgia", "Manrope", "Avenir", "ui-sans-serif",
		} {
			if strings.Contains(sheet, forbidden) {
				t.Fatalf("%s reintroduced the font %q", name, forbidden)
			}
		}
		if !strings.Contains(sheet, `-apple-system,system-ui,"Segoe UI",sans-serif`) {
			t.Fatalf("%s dropped the native render stack", name)
		}
	}
}

func TestReadingSheetsPinNativeReadingMetrics(t *testing.T) {
	t.Parallel()

	for name, sheet := range map[string]string{
		"readingCSS":        readingCSS,
		"readingArticleCSS": readingArticleCSS,
	} {
		for _, expected := range []string{
			// White page, body text and 16px/1.65 base from the app artifact.
			`background:#fff`, `#2e3238`, `16px/1.65`,
		} {
			if !strings.Contains(sheet, expected) {
				t.Fatalf("%s lost the native reading metric %q", name, expected)
			}
		}
	}

	// The Dossier override must preserve the artifact's 760px outer main
	// (20px horizontal padding inside a border-box) instead of re-typing it.
	for _, expected := range []string{
		`max-width:760px!important`,
		`padding:32px 20px 96px!important`,
	} {
		if !strings.Contains(readingArticleCSS, expected) {
			t.Fatalf("readingArticleCSS lost the article metric %q", expected)
		}
	}
	// Heading metrics must not be overridden away from artifact defaults
	// (30px/36px/700 h1, 18.72px bold h3 in plain document flow).
	for _, forbidden := range []string{
		`h1{font-size`, `>div>h1{`, `h3{font-size`, `h4{font-size`, `font-weight:400`,
	} {
		if strings.Contains(readingArticleCSS, forbidden) {
			t.Fatalf("readingArticleCSS re-typed heading metrics (%q)", forbidden)
		}
	}
}
