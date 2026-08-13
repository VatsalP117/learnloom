package httpapp

import (
	"strings"
	"testing"
)

func TestMarketingNavAlwaysProvidesExplicitHomeRoute(t *testing.T) {
	t.Parallel()

	document := renderMarketingNav("https://app.learnloom.blog", "/solutions/remember-what-you-read")
	for _, expected := range []string{
		`aria-label="Learnloom home"`,
		`href="/">Home</a>`,
		`href="/solutions" aria-current="page"`,
		`href="https://app.learnloom.blog/sign-in"`,
		`href="https://app.learnloom.blog/sign-up"`,
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("marketing navigation missing %q", expected)
		}
	}
}

func TestMarketingFooterUsesSharedPublicNavigation(t *testing.T) {
	t.Parallel()

	document := renderMarketingFooter("https://app.learnloom.blog/")
	for _, expected := range []string{
		`class="seo-footer"`,
		`href="/" aria-label="Learnloom home"`,
		`href="/how-learnloom-works"`,
		`href="/editorial-principles"`,
		`href="/privacy"`,
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("marketing footer missing %q", expected)
		}
	}
}
