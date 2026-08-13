package httpapp

import (
	"html"
	"strings"
)

func renderMarketingNav(appOrigin, currentPath string) string {
	links := []struct{ path, label string }{
		{path: "/", label: "Home"},
		{path: "/solutions", label: "Solutions"},
		{path: "/product/ai-learning-assistant", label: "Product"},
		{path: "/guides", label: "Guides"},
		{path: "/examples", label: "Examples"},
		{path: "/how-learnloom-works", label: "How it works"},
	}

	var body strings.Builder
	body.WriteString(`<header class="seo-nav"><a class="seo-brand" href="/" aria-label="Learnloom home"><span aria-hidden="true"><img src="/favicon.svg" alt=""></span><strong>Learnloom</strong></a><nav aria-label="Main navigation">`)
	for _, link := range links {
		current := ""
		if marketingSectionActive(currentPath, link.path) {
			current = ` aria-current="page"`
		}
		body.WriteString(`<a href="` + link.path + `"` + current + `>` + link.label + `</a>`)
	}
	body.WriteString(`</nav><div class="seo-nav-actions"><a class="nav-sign-in" href="`)
	body.WriteString(html.EscapeString(strings.TrimRight(appOrigin, "/") + "/sign-in"))
	body.WriteString(`">Sign in</a><a class="nav-cta" href="`)
	body.WriteString(html.EscapeString(strings.TrimRight(appOrigin, "/") + "/sign-up"))
	body.WriteString(`">Start learning <span aria-hidden="true">↗</span></a></div></header>`)
	return body.String()
}

func marketingSectionActive(currentPath, linkPath string) bool {
	switch linkPath {
	case "/":
		return currentPath == "/"
	case "/solutions":
		return currentPath == linkPath || strings.HasPrefix(currentPath, linkPath+"/")
	case "/product/ai-learning-assistant":
		return strings.HasPrefix(currentPath, "/product/")
	case "/guides":
		return currentPath == linkPath || strings.HasPrefix(currentPath, linkPath+"/")
	default:
		return currentPath == linkPath
	}
}

func renderMarketingFooter(appOrigin string) string {
	accountOrigin := strings.TrimRight(appOrigin, "/")
	return `<footer class="seo-footer"><div class="seo-footer-intro"><a class="seo-brand" href="/" aria-label="Learnloom home"><span aria-hidden="true"><img src="/favicon.svg" alt=""></span><strong>Learnloom</strong></a>` +
		`<p>Current sources, woven into durable understanding.</p></div><div class="seo-footer-links">` +
		`<div><strong>Product</strong><a href="/product/ai-learning-assistant">AI learning assistant</a><a href="/product/trusted-source-learning">Trusted-source learning</a><a href="/how-learnloom-works">How Learnloom works</a></div>` +
		`<div><strong>Learn</strong><a href="/guides">Learning guides</a><a href="/examples">Public examples</a><a href="/editorial-principles">Editorial principles</a></div>` +
		`<div><strong>Account</strong><a href="` + html.EscapeString(accountOrigin+"/sign-in") + `">Sign in</a><a href="` + html.EscapeString(accountOrigin+"/sign-up") + `">Get started</a></div>` +
		`<div><strong>Legal</strong><a href="/privacy">Privacy</a><a href="/terms">Terms</a></div></div>` +
		`<div class="seo-footer-bottom"><span>© 2026 Learnloom</span><span>Built for durable understanding.</span></div></footer>`
}
