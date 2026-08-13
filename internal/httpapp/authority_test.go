package httpapp

import (
	"html"
	"strings"
	"testing"
)

func TestAuthorityPageCatalogRendersSubstantiveCanonicalArticles(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, page := range authorityPages {
		if page.Path == "" || page.Category == "" || page.SchemaType == "" || page.Title == "" ||
			page.Description == "" || page.Lead == "" ||
			len(page.Sections) < 2 || len(page.Takeaways) < 3 {
			t.Fatalf("authority page is incomplete: %#v", page)
		}
		if seen[page.Path] {
			t.Fatalf("duplicate authority path %q", page.Path)
		}
		seen[page.Path] = true

		for _, path := range page.Related {
			if _, ok := publicPageTitle(path); !ok {
				t.Fatalf("%s has unresolved related page %q", page.Path, path)
			}
		}
		for _, reference := range page.References {
			if reference.Title == "" || reference.Citation == "" ||
				!strings.HasPrefix(reference.URL, "https://") {
				t.Fatalf("%s has invalid reference %#v", page.Path, reference)
			}
		}

		canonical := "https://learnloom.blog" + page.Path
		document := renderAuthorityDocument(
			page,
			canonical,
			"https://app.learnloom.blog",
		)
		for _, expected := range []string{
			`<html lang="en">`,
			`<article class="authority-article">`,
			`<h1>` + html.EscapeString(page.Title) + `</h1>`,
			`rel="canonical" href="` + canonical + `"`,
			`"@type":"` + page.SchemaType + `"`,
			`href="https://app.learnloom.blog/sign-up"`,
		} {
			if !strings.Contains(document, expected) {
				t.Fatalf("%s missing %q", page.Path, expected)
			}
		}
		if page.SchemaType == "Article" &&
			!strings.Contains(document, `property="og:type" content="article"`) {
			t.Fatalf("%s should use the article Open Graph type", page.Path)
		}
		if len(page.References) > 0 &&
			(!strings.Contains(document, "Research references") ||
				!strings.Contains(document, `"citation":[`)) {
			t.Fatalf("%s does not expose references in HTML and Article data", page.Path)
		}
	}
}

func TestHowLearnloomWorksStatesCurrentDiscoveryOptions(t *testing.T) {
	t.Parallel()

	page, ok := authorityPageForPath("/how-learnloom-works")
	if !ok {
		t.Fatal("How Learnloom works page is missing")
	}
	document := renderAuthorityDocument(
		page,
		"https://learnloom.blog/how-learnloom-works",
		"https://app.learnloom.blog",
	)
	for _, expected := range []string{
		"a topic is enough to begin",
		"provide only sources they trust",
		"let Learnloom fill gaps",
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("How Learnloom works missing discovery option %q", expected)
		}
	}
}
