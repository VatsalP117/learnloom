package source

import (
	"net/url"
	"strings"
	"testing"
)

func TestReadabilityFixtureEvaluation(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://example.com/guide")
	content := strings.Repeat(
		"Grounded explanations connect the mechanism to a practical worked example. ",
		18,
	)
	fixtures := []struct {
		name string
		html string
	}{
		{"semantic article", `<html><body><nav>Navigation noise</nav><article><h1>Guide</h1><p>` + content + `</p></article><footer>Footer noise</footer></body></html>`},
		{"main documentation", `<html><body><header>Header noise</header><main><h1>Documentation</h1><p>` + content + `</p></main></body></html>`},
		{"content class", `<html><body><aside class="sidebar">Sidebar noise</aside><div class="article-content"><h1>Lesson</h1><p>` + content + `</p></div></body></html>`},
		{"blog post", `<html><body><div class="menu">Menu noise</div><div class="post"><h1>Post</h1><p>` + content + `</p></div><div class="comments">Comment noise</div></body></html>`},
		{"nested sections", `<html><body><main><section><h1>Research review</h1><section><p>` + content + `</p></section></section></main></body></html>`},
		{"metadata and scripts", `<html><head><meta name="author" content="Ada"><script>script noise</script></head><body><article><h1>Technical note</h1><p>` + content + `</p></article></body></html>`},
		{"utility-heavy page", `<html><body><form>Form noise</form><div class="toolbar">Toolbar noise</div><div id="content"><h1>Tutorial</h1><p>` + content + `</p></div></body></html>`},
		{"reference with table", `<html><body><article><h1>Reference</h1><p>` + content + `</p><table><tr><td>Parameter</td><td>Meaningful value</td></tr></table></article></body></html>`},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			readable, _, _ := extractArticle(fixture.html, base)
			fallback, _, _ := extractArticleFallback(fixture.html, base)
			if !strings.Contains(readable, "Grounded explanations") {
				t.Fatalf("readability lost the fixture's main content")
			}
			if strings.Contains(strings.ToLower(readable), "script noise") {
				t.Fatalf("active script content leaked into extracted text")
			}
			t.Logf(
				"readability_chars=%d fallback_chars=%d",
				len([]rune(readable)),
				len([]rune(fallback)),
			)
		})
	}
}
