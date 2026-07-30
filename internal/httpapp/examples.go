package httpapp

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
)

type featuredExample struct {
	Site   domain.PersonalSite
	Issues []store.PublicIssue
}

func (s *Server) loadFeaturedExamples(ctx context.Context) ([]featuredExample, error) {
	examples := make([]featuredExample, 0, len(s.cfg.FeaturedSites))
	for _, username := range s.cfg.FeaturedSites {
		site, err := s.store.GetPublicSite(ctx, username)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !site.SearchIndexing {
			continue
		}
		issues, err := s.store.ListPublicIssues(ctx, username, "", 3)
		if err != nil {
			return nil, err
		}
		if len(issues) == 0 {
			continue
		}
		examples = append(examples, featuredExample{Site: site, Issues: issues})
	}
	return examples, nil
}

func (s *Server) renderExamplesPage(
	response http.ResponseWriter,
	request *http.Request,
) {
	examples, err := s.loadFeaturedExamples(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	canonical := strings.TrimRight(s.cfg.ApexOrigin, "/") + "/examples"
	document := renderExamplesDocument(
		examples,
		canonical,
		s.cfg.AppOrigin,
		s.cfg.RootDomain,
	)
	s.applyAppCSP(response)
	if len(examples) > 0 {
		response.Header().Set("X-Robots-Tag", "index, follow")
	} else {
		response.Header().Set("X-Robots-Tag", "noindex, follow")
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=3600")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func renderExamplesDocument(
	examples []featuredExample,
	canonical, appOrigin, rootDomain string,
) string {
	const title = "Personal learning examples | Learnloom"
	const description = "Explore selected public learning homes and source-grounded Knowledge Dossiers created with Learnloom."
	var body strings.Builder
	body.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	body.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	body.WriteString(`<link rel="icon" href="/favicon.svg" type="image/svg+xml">`)
	body.WriteString(renderExamplesHead(
		examples,
		title,
		description,
		canonical,
		rootDomain,
		strings.TrimSuffix(canonical, "/examples"),
	))
	body.WriteString(`<style>` + seoCSS + examplesCSS + `</style></head><body>`)
	body.WriteString(`<header class="seo-nav"><a class="seo-brand" href="/"><span>✣</span>Learnloom</a>`)
	body.WriteString(`<nav aria-label="Main navigation"><a href="/solutions">Solutions</a><a href="/product/ai-learning-assistant">Product</a><a href="/guides">Guides</a><a href="/examples">Examples</a><a href="/how-learnloom-works">How it works</a></nav>`)
	body.WriteString(`<a class="nav-cta" href="` + html.EscapeString(strings.TrimRight(appOrigin, "/")+"/sign-up") + `">Start learning <span>↗</span></a></header>`)
	body.WriteString(`<main><section class="examples-hero"><p class="eyebrow">Learning in public</p><h1>See what sustained curiosity can become.</h1><p>Selected learning homes show how current sources can become connected, source-grounded lessons over time. Every featured learner has chosen both public visibility and search discovery.</p></section>`)
	if len(examples) == 0 {
		body.WriteString(`<section class="examples-empty"><p class="eyebrow">The gallery is being prepared</p><h2>Curated examples will appear here.</h2><p>Public sites are never added automatically. Learnloom features only selected learning homes whose owners have enabled search discovery.</p></section>`)
	} else {
		body.WriteString(`<section class="examples-grid" aria-label="Featured learning homes">`)
		for _, example := range examples {
			siteOrigin := "https://" + example.Site.Username + "." + rootDomain
			description := firstReadingText(
				example.Site.Description,
				"A public archive of connected, source-grounded learning.",
			)
			body.WriteString(`<article class="example-card"><header><p class="eyebrow">Personal learning home</p><h2>` +
				html.EscapeString(example.Site.DisplayName) + `</h2><p>` +
				html.EscapeString(description) + `</p><a href="` +
				html.EscapeString(siteOrigin) + `">Visit ` +
				html.EscapeString(example.Site.Username+"."+rootDomain) +
				` <span>↗</span></a></header><div class="example-lessons"><strong>Recent Knowledge Dossiers</strong>`)
			for _, issue := range example.Issues {
				issueURL := siteOrigin + "/d/" + url.PathEscape(issue.PublicID) + "/" +
					url.PathEscape(issue.PublicSlug)
				body.WriteString(`<a href="` + html.EscapeString(issueURL) + `"><span>` +
					html.EscapeString(issue.NewsletterName) + `</span><b>` +
					html.EscapeString(issue.Title) + `</b><i>Read Dossier ↗</i></a>`)
			}
			body.WriteString(`</div></article>`)
		}
		body.WriteString(`</section>`)
	}
	body.WriteString(`<section class="seo-cta"><p class="eyebrow">Build your own learning home</p><h2>Let each lesson become part of something larger.</h2><p>Choose a subject, bring the sources you trust, and decide privately what you want to share.</p><a class="primary" href="`)
	body.WriteString(html.EscapeString(strings.TrimRight(appOrigin, "/") + "/sign-up"))
	body.WriteString(`">Create your learning stream <span>↗</span></a></section></main>`)
	body.WriteString(renderSEOFooter(appOrigin))
	body.WriteString(`</body></html>`)
	return body.String()
}

func renderExamplesHead(
	examples []featuredExample,
	title, description, canonical, rootDomain, apexOrigin string,
) string {
	parts := make([]map[string]any, 0, len(examples))
	for _, example := range examples {
		parts = append(parts, map[string]any{
			"@type": "CollectionPage",
			"name":  example.Site.DisplayName,
			"url":   "https://" + example.Site.Username + "." + rootDomain,
		})
	}
	schema := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        title,
		"description": description,
		"url":         canonical,
		"hasPart":     parts,
	}
	encoded, _ := json.Marshal(schema)
	return `<title>` + html.EscapeString(title) + `</title>` +
		`<meta name="description" content="` + html.EscapeString(description) + `">` +
		`<link rel="canonical" href="` + html.EscapeString(canonical) + `">` +
		`<meta property="og:type" content="website">` +
		`<meta property="og:site_name" content="Learnloom">` +
		`<meta property="og:title" content="` + html.EscapeString(title) + `">` +
		`<meta property="og:description" content="` + html.EscapeString(description) + `">` +
		`<meta property="og:url" content="` + html.EscapeString(canonical) + `">` +
		renderSocialImageMetadata(apexOrigin) +
		`<meta name="twitter:title" content="` + html.EscapeString(title) + `">` +
		`<meta name="twitter:description" content="` + html.EscapeString(description) + `">` +
		`<script type="application/ld+json">` + string(encoded) + `</script>`
}

const examplesCSS = `
.examples-hero{padding:clamp(85px,10vw,145px) max(24px,calc((100vw - 1120px)/2));background:radial-gradient(circle at 82% 18%,#cfe8df 0,transparent 30%),#f7f5ee;border-bottom:1px solid rgba(16,37,33,.12)}
.examples-hero h1{max-width:900px;margin:0 0 30px;font-family:Georgia,serif;font-size:clamp(48px,7vw,84px);font-weight:500;line-height:1.03;letter-spacing:-.045em}
.examples-hero>p:not(.eyebrow){max-width:730px;color:#47605a;font-size:20px;line-height:1.7}
.examples-grid{max-width:1120px;margin:0 auto;padding:clamp(70px,9vw,115px) 24px;display:grid;gap:28px}
.example-card{display:grid;grid-template-columns:minmax(260px,.7fr) minmax(0,1.3fr);gap:50px;padding:36px;border:1px solid rgba(16,37,33,.14);border-radius:20px;background:#fff}
.example-card header h2{margin:0 0 15px;font-family:Georgia,serif;font-size:34px;font-weight:500;letter-spacing:-.03em}.example-card header>p:not(.eyebrow){color:#536a64;line-height:1.65}.example-card header>a{display:inline-block;margin-top:18px;color:#1d5d4e;font-size:14px;font-weight:800;text-decoration:none}
.example-lessons{display:grid;gap:10px}.example-lessons>strong{margin-bottom:5px;font-size:11px;letter-spacing:.08em;text-transform:uppercase}.example-lessons>a{display:grid;grid-template-columns:1fr auto;gap:6px 18px;padding:18px;border-radius:12px;background:#f1f4ee;color:inherit;text-decoration:none}.example-lessons span{color:#557068;font-size:11px;text-transform:uppercase;letter-spacing:.06em}.example-lessons b{grid-column:1/2;font-size:16px;line-height:1.4}.example-lessons i{grid-column:2;grid-row:1/3;align-self:center;color:#397064;font-size:11px;font-style:normal;font-weight:800}
.examples-empty{max-width:800px;margin:0 auto;padding:120px 24px;text-align:center}.examples-empty h2{font-family:Georgia,serif;font-size:48px;font-weight:500;margin:0 0 20px}.examples-empty>p:last-child{color:#536a64;line-height:1.7}
@media(max-width:780px){.example-card{grid-template-columns:1fr;padding:24px;gap:34px}.example-lessons>a{grid-template-columns:1fr}.example-lessons i{grid-column:1;grid-row:auto}.examples-hero h1{font-size:48px}}
`
