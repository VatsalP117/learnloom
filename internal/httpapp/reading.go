package httpapp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/google/uuid"
)

func (s *Server) handleReading(
	response http.ResponseWriter,
	request *http.Request,
	host RequestHost,
) {
	route := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if request.Method == http.MethodPost {
		if len(route) == 2 && route[0] == "report" {
			s.handlePublicContentReport(response, request, host, route[1])
			return
		}
		if len(route) == 2 && route[0] == "follow" {
			s.handlePublicPathFollow(response, request, host, route[1])
			return
		}
	}
	if request.Method == http.MethodGet && len(route) == 3 && route[0] == "go" {
		s.handlePublicGrowthClick(response, request, host, route[1], route[2])
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		methodNotAllowed(response, http.MethodGet, http.MethodHead)
		return
	}
	if isFaviconPath(request.URL.Path) {
		s.serveStatic(response, request, strings.TrimPrefix(request.URL.Path, "/"))
		return
	}
	site, err := s.store.GetPublicSite(request.Context(), host.Username)
	if err != nil {
		s.readingNotFound(response, request)
		return
	}
	origin := "https://" + host.Hostname
	switch request.URL.Path {
	case "/robots.txt":
		renderPersonalRobots(response, request, site, origin)
		return
	case "/sitemap.xml":
		s.renderSitemap(response, request, site, origin)
		return
	case "/":
		s.renderPublicHome(response, request, site, origin)
		return
	}
	if len(route) == 2 && route[0] == "topics" {
		s.renderPublicTopic(response, request, site, origin, route[1])
		return
	}
	if len(route) >= 2 && route[0] == "d" {
		s.renderPublicDossier(response, request, site, origin, route)
		return
	}
	s.readingNotFound(response, request)
}

func renderPersonalRobots(
	response http.ResponseWriter,
	request *http.Request,
	site domain.PersonalSite,
	origin string,
) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300")
	if request.Method != http.MethodHead {
		fmt.Fprint(response, "User-agent: *\nAllow: /\n")
		if site.SearchIndexing {
			fmt.Fprintf(response, "Sitemap: %s/sitemap.xml\n", origin)
		}
	}
}

func (s *Server) renderPublicHome(
	response http.ResponseWriter,
	request *http.Request,
	site domain.PersonalSite,
	origin string,
) {
	newsletters, err := s.store.ListPublicNewsletters(request.Context(), site.Username)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	issues, err := s.store.ListPublicIssues(request.Context(), site.Username, "", 24)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	streamCount := len(newsletters)
	dossierCount := 0
	for _, newsletter := range newsletters {
		dossierCount += newsletter.GeneratedCount
	}
	var topics strings.Builder
	for index, newsletter := range newsletters {
		fmt.Fprintf(
			&topics,
			`<a class="topic-card" href="/topics/%s">`+
				`<span class="topic-index">%02d</span><span class="topic-copy">`+
				`<strong>%s</strong><span>%s</span></span>`+
				`<span class="topic-count">%d <small>Dossiers</small></span>`+
				`<span class="arrow" aria-hidden="true">↗</span></a>`,
			url.PathEscape(newsletter.PublicSlug),
			index+1,
			html.EscapeString(newsletter.Name),
			html.EscapeString(newsletter.Topic),
			newsletter.GeneratedCount,
		)
	}
	if len(newsletters) == 0 {
		topics.WriteString(renderReadingEmpty(
			"The archive is just beginning.",
			"Published learning streams will gather here.",
		))
	}
	body := renderReadingHeader(site, "home") +
		`<main id="main-content">` + renderHomeHero(site, streamCount, dossierCount) +
		`<section class="reading-section topics-section" id="topics"><div class="section-heading">` +
		`<div><p class="eyebrow">Paths of inquiry</p><h2>Learning streams</h2></div>` +
		`<p>Each stream follows one question over time, building context instead of adding noise.</p>` +
		`</div><div class="topics">` + topics.String() + `</div></section>` +
		`<section class="reading-section latest-section" id="latest"><div class="section-heading">` +
		`<div><p class="eyebrow">Recently tended</p><h2>Latest Dossiers</h2></div>` +
		`<p>Source-grounded lessons shaped for patient, lasting understanding.</p>` +
		`</div><div class="issues">` + renderIssueCards(issues) + `</div></section></main>` +
		renderReadingFooter(site)
	s.sendReadingPage(
		response,
		request,
		site.DisplayName,
		firstReadingText(site.Description, "A durable personal learning archive."),
		origin,
		body,
		site.SearchIndexing && len(issues) > 0,
	)
}

func (s *Server) renderPublicTopic(
	response http.ResponseWriter,
	request *http.Request,
	site domain.PersonalSite,
	origin, slug string,
) {
	newsletters, err := s.store.ListPublicNewsletters(request.Context(), site.Username)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	var selected *store.PublicNewsletter
	for index := range newsletters {
		if newsletters[index].PublicSlug == slug {
			selected = &newsletters[index]
			break
		}
	}
	if selected == nil {
		s.readingNotFound(response, request)
		return
	}
	issues, err := s.store.ListPublicIssues(request.Context(), site.Username, slug, 100)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	body := renderReadingHeader(site, "topics") +
		`<main id="main-content"><header class="reading-hero topic-hero">` +
		`<div class="hero-copy"><a class="back-link" href="/">← Back to the archive</a>` +
		`<p class="eyebrow"><span aria-hidden="true"></span>Learning stream</p><h1>` +
		html.EscapeString(selected.Name) + `</h1><p class="hero-description">` +
		html.EscapeString(selected.Topic) + `</p></div>` +
		`<aside class="stream-note"><span>Published archive</span><strong>` +
		fmt.Sprint(len(issues)) + `</strong><small>Dossiers in this stream</small></aside></header>` +
		`<section class="reading-section latest-section"><div class="section-heading">` +
		`<div><p class="eyebrow">The full thread</p><h2>Archive</h2></div>` +
		`<p>Read from the newest entry or wander back through the questions that shaped this stream.</p>` +
		`</div><div class="issues">` + renderIssueCards(issues) + `</div></section></main>` +
		renderReadingFooter(site)
	s.sendReadingPage(
		response,
		request,
		selected.Name,
		selected.Topic,
		origin+"/topics/"+url.PathEscape(selected.PublicSlug),
		body,
		site.SearchIndexing && len(issues) > 0,
	)
}

func (s *Server) renderPublicDossier(
	response http.ResponseWriter,
	request *http.Request,
	site domain.PersonalSite,
	origin string,
	route []string,
) {
	issue, err := s.store.GetPublicIssue(request.Context(), site.Username, route[1])
	if err != nil {
		s.readingNotFound(response, request)
		return
	}
	canonicalPath := "/d/" + url.PathEscape(issue.PublicID) + "/" + url.PathEscape(issue.PublicSlug)
	if request.URL.Path != canonicalPath {
		http.Redirect(response, request, origin+canonicalPath, http.StatusPermanentRedirect)
		return
	}
	if request.Method == http.MethodGet && !isLikelyAutomatedVisitor(request.UserAgent()) {
		fingerprint := s.publicVisitorFingerprint(response, request)
		if err := s.store.RecordPublicGrowthEvent(
			request.Context(), site.Username, issue.PublicID,
			store.PublicGrowthView, "", fingerprint, time.Now().UTC(),
		); err != nil {
			s.logger.WarnContext(request.Context(), "record public Dossier view", "error", err)
		}
	}
	artifactValue, err := s.artifacts.Get(request.Context(), issue.ArtifactKey)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	related, err := s.store.ListRelatedPublicIssues(
		request.Context(), site.Username, issue.NewsletterID, issue.ID, 3,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	canonical := origin + canonicalPath
	description := "A source-grounded Knowledge Dossier about " + issue.NewsletterName + "."
	metadata := renderPublicDossierMetadata(
		site,
		issue,
		canonical,
		origin,
		s.cfg.ApexOrigin,
		description,
	)
	document := strings.Replace(
		artifactValue.HTML,
		"</head>",
		`<link rel="canonical" href="`+html.EscapeString(canonical)+`">`+
			metadata+"</head>",
		1,
	)
	document = decoratePublicDossier(document, site, issue)
	document = decoratePublicGrowth(document, site, issue, canonical, related)
	corrections, err := s.store.ListPublicCorrections(request.Context(), issue.ID)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	document = decoratePublicModeration(document, issue, corrections)
	s.applyReadingHeaders(response, site.SearchIndexing)
	response.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func renderPublicDossierMetadata(
	site domain.PersonalSite,
	issue store.PublicIssue,
	canonical, authorOrigin, apexOrigin, description string,
) string {
	// PersonalSite and PublicIssue are the store's publication-safe projections.
	// Do not accept an account, artifact body, source excerpt, or draft issue here.
	articleSchema, _ := json.Marshal(map[string]any{
		"@context":         "https://schema.org",
		"@type":            "Article",
		"headline":         issue.Title,
		"description":      description,
		"datePublished":    issue.CompletedAt.Format(time.RFC3339),
		"dateModified":     issue.CompletedAt.Format(time.RFC3339),
		"mainEntityOfPage": canonical,
		"articleSection":   issue.NewsletterName,
		"author": map[string]any{
			"@type": "Person",
			"name":  site.DisplayName,
			"url":   authorOrigin,
		},
		"publisher": map[string]any{
			"@type": "Organization",
			"name":  "Learnloom",
			"url":   "https://learnloom.blog/",
		},
	})
	return `<meta name="description" content="` + html.EscapeString(description) + `">` +
		`<meta property="og:type" content="article">` +
		`<meta property="og:site_name" content="Learnloom">` +
		`<meta property="og:title" content="` + html.EscapeString(issue.Title) + `">` +
		`<meta property="og:description" content="` + html.EscapeString(description) + `">` +
		`<meta property="og:url" content="` + html.EscapeString(canonical) + `">` +
		`<meta property="article:published_time" content="` + issue.CompletedAt.Format(time.RFC3339) + `">` +
		renderSocialImageMetadata(apexOrigin) +
		`<meta name="twitter:title" content="` + html.EscapeString(issue.Title) + `">` +
		`<meta name="twitter:description" content="` + html.EscapeString(description) + `">` +
		`<script type="application/ld+json">` + string(articleSchema) + `</script>`
}

func (s *Server) renderSitemap(
	response http.ResponseWriter,
	request *http.Request,
	site domain.PersonalSite,
	origin string,
) {
	if !site.SearchIndexing {
		writeEmptySitemap(response, request)
		return
	}
	newsletters, err := s.store.ListPublicNewsletters(request.Context(), site.Username)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	issues, err := s.store.ListPublicIssues(request.Context(), site.Username, "", 49000)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	var locations []string
	if len(issues) > 0 {
		locations = append(locations, origin)
	}
	for _, newsletter := range newsletters {
		if newsletter.GeneratedCount > 0 {
			locations = append(locations, origin+"/topics/"+url.PathEscape(newsletter.PublicSlug))
		}
	}
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, location := range locations {
		body.WriteString("<url><loc>" + html.EscapeString(location) + "</loc></url>")
	}
	for _, issue := range issues {
		location := origin + "/d/" + url.PathEscape(issue.PublicID) + "/" + url.PathEscape(issue.PublicSlug)
		body.WriteString("<url><loc>" + html.EscapeString(location) + "</loc><lastmod>" +
			issue.CompletedAt.Format(time.RFC3339) + "</lastmod></url>")
	}
	body.WriteString("</urlset>")
	response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(body.String()))
	}
}

func writeEmptySitemap(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(
			`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`,
		))
	}
}

func (s *Server) sendReadingPage(
	response http.ResponseWriter,
	request *http.Request,
	title, description, canonical, body string,
	indexable bool,
) {
	document := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<link rel="icon" href="/favicon.svg" type="image/svg+xml">` +
		`<title>` + html.EscapeString(title) + ` · Learnloom</title>` +
		`<meta name="description" content="` + html.EscapeString(description) + `">` +
		`<link rel="canonical" href="` + html.EscapeString(canonical) + `">` +
		`<meta property="og:type" content="website">` +
		`<meta property="og:site_name" content="Learnloom">` +
		`<meta property="og:title" content="` + html.EscapeString(title) + ` · Learnloom">` +
		`<meta property="og:description" content="` + html.EscapeString(description) + `">` +
		`<meta property="og:url" content="` + html.EscapeString(canonical) + `">` +
		renderSocialImageMetadata(s.cfg.ApexOrigin) +
		`<meta name="twitter:title" content="` + html.EscapeString(title) + ` · Learnloom">` +
		`<meta name="twitter:description" content="` + html.EscapeString(description) + `">` +
		`<style>` + readingCSS + `</style></head><body>` + body + `</body></html>`
	s.applyReadingHeaders(response, indexable)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func (s *Server) applyReadingHeaders(response http.ResponseWriter, indexable bool) {
	response.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; img-src 'self' https: data:; "+
			"font-src 'self'; base-uri 'none'; form-action 'self'; "+
			"frame-ancestors 'none'; object-src 'none'",
	)
	if indexable {
		response.Header().Set("X-Robots-Tag", "index, follow")
	} else {
		response.Header().Set("X-Robots-Tag", "noindex, follow")
	}
}

func (s *Server) readingNotFound(
	response http.ResponseWriter,
	request *http.Request,
) {
	s.applyReadingHeaders(response, false)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=30")
	response.WriteHeader(http.StatusNotFound)
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8">` +
			`<meta name="viewport" content="width=device-width,initial-scale=1">` +
			`<link rel="icon" href="/favicon.svg" type="image/svg+xml">` +
			`<title>Not found · Learnloom</title><style>` + readingCSS +
			`</style></head><body><main class="not-found" id="main-content">` +
			`<a class="skip-link" href="#main-content">Skip to content</a>` +
			`<img class="leaf-mark" src="/favicon.svg" alt="" aria-hidden="true">` +
			`<p class="eyebrow"><span aria-hidden="true"></span>A quiet corner of the web</p>` +
			`<h1>This page has wandered off the path.</h1>` +
			`<p>The learning page you were looking for is unavailable.</p>` +
			`<a class="button" href="https://learnloom.blog">Return to Learnloom <span>↗</span></a>` +
			`</main></body></html>`))
	}
}

func renderIssueCards(issues []store.PublicIssue) string {
	if len(issues) == 0 {
		return renderReadingEmpty(
			"No Dossiers yet.",
			"The first published lesson will appear here.",
		)
	}
	var result strings.Builder
	for index, issue := range issues {
		href := "/d/" + url.PathEscape(issue.PublicID) + "/" + url.PathEscape(issue.PublicSlug)
		fmt.Fprintf(
			&result,
			`<article class="issue-card"><a class="issue-link" href="%s">`+
				`<div class="issue-top"><span class="issue-number">%02d</span>`+
				`<span class="issue-topic">%s</span><span class="arrow" aria-hidden="true">↗</span></div>`+
				`<h3>%s</h3><div class="issue-meta"><time datetime="%s">%s</time>`+
				`<span>Read Dossier</span></div></a></article>`,
			href,
			index+1,
			html.EscapeString(issue.NewsletterName),
			html.EscapeString(issue.Title),
			issue.CompletedAt.Format(time.RFC3339),
			issue.CompletedAt.Format("02 Jan 2006"),
		)
	}
	return result.String()
}

func renderHomeHero(site domain.PersonalSite, streamCount, dossierCount int) string {
	return `<section class="reading-hero home-hero"><div class="hero-copy">` +
		`<p class="eyebrow"><span aria-hidden="true"></span>A public learning archive</p>` +
		`<h1>` + html.EscapeString(site.DisplayName) + `</h1>` +
		`<p class="hero-description">` + html.EscapeString(firstReadingText(
		site.Description,
		"A durable personal learning archive.",
	)) + `</p>` +
		`<a class="text-link" href="/#topics">Explore the archive <span aria-hidden="true">↓</span></a>` +
		`</div><aside class="archive-note"><p class="note-label">The public archive</p>` +
		`<dl class="archive-stats"><div><dt>Streams</dt><dd>` + fmt.Sprint(streamCount) + `</dd></div>` +
		`<div><dt>Dossiers</dt><dd>` + fmt.Sprint(dossierCount) + `</dd></div></dl>` +
		`<p class="note-foot">Questions followed over time, shaped into source-grounded lessons.</p>` +
		`</aside></section>`
}

func renderReadingHeader(site domain.PersonalSite, current string) string {
	topicsClass := ""
	if current == "topics" {
		topicsClass = ` class="current"`
	}
	return `<a class="skip-link" href="#main-content">Skip to content</a>` +
		`<header class="site-header"><a class="site-brand" href="/" aria-label="` +
		html.EscapeString(site.DisplayName) + ` home"><span class="brand-mark" aria-hidden="true"><img src="/favicon.svg" alt=""></span>` +
		`<span><strong>` + html.EscapeString(site.DisplayName) +
		`</strong><small>Learning archive</small></span></a>` +
		`<nav aria-label="Personal site navigation"><a` + topicsClass +
		` href="/#topics">Streams</a><a href="/#latest">Archive</a>` +
		`<a href="/#about">About</a></nav></header>`
}

func renderReadingFooter(site domain.PersonalSite) string {
	return `<footer class="site-footer" id="about"><div><a class="site-brand" href="/">` +
		`<span class="brand-mark" aria-hidden="true"><img src="/favicon.svg" alt=""></span><span><strong>` +
		html.EscapeString(site.DisplayName) +
		`</strong><small>A personal learning archive</small></span></a>` +
		`<p>` + html.EscapeString(firstReadingText(
		site.Description,
		"A quiet place for ideas worth returning to.",
	)) + `</p></div><div class="footer-meta"><span>Published with</span>` +
		`<a href="https://learnloom.blog">Learnloom ↗</a></div></footer>`
}

func renderReadingEmpty(title, description string) string {
	return `<div class="empty-state"><img class="leaf-mark" src="/favicon.svg" alt="" aria-hidden="true">` +
		`<strong>` + html.EscapeString(title) + `</strong><span>` +
		html.EscapeString(description) + `</span></div>`
}

func decoratePublicDossier(
	document string,
	site domain.PersonalSite,
	issue store.PublicIssue,
) string {
	document = strings.Replace(
		document,
		"</head>",
		`<link rel="icon" href="/favicon.svg" type="image/svg+xml"><style>`+readingArticleCSS+`</style></head>`,
		1,
	)
	if strings.Contains(document, `class="public-dossier"`) {
		return document
	}
	header := `<header class="article-header"><a class="article-brand" href="/">` +
		`<span class="brand-mark" aria-hidden="true"><img src="/favicon.svg" alt=""></span><span><strong>` +
		html.EscapeString(site.DisplayName) + `</strong><small>Learning archive</small></span></a>` +
		`<a class="article-back" href="/topics/` +
		url.PathEscape(issue.NewsletterPublicSlug) + `">` +
		html.EscapeString(issue.NewsletterName) + ` <span aria-hidden="true">↗</span></a></header>`
	legacyBody := `<body style="margin:0;background:#f5f5f4;color:#0f172a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">`
	neutralBody := `<body style="margin:0;background:#fff;color:#2e3238;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">`
	nativeBody := `<body style="margin:0;background:#fff;color:#2e3238;font-family:-apple-system,system-ui,'Segoe UI',sans-serif;font-size:16px;line-height:1.65">`
	for _, storedBody := range []string{legacyBody, neutralBody, nativeBody} {
		if strings.Contains(document, storedBody) {
			return strings.Replace(
				document,
				storedBody,
				`<body class="public-dossier">`+header,
				1,
			)
		}
	}
	return strings.Replace(document, "<body>", `<body class="public-dossier">`+header, 1)
}

func decoratePublicGrowth(
	document string,
	site domain.PersonalSite,
	issue store.PublicIssue,
	canonical string,
	related []store.PublicIssue,
) string {
	var relatedHTML strings.Builder
	for _, item := range related {
		href := "/d/" + url.PathEscape(item.PublicID) + "/" + url.PathEscape(item.PublicSlug)
		fmt.Fprintf(
			&relatedHTML,
			`<a href="%s"><small>%s</small><strong>%s</strong><span>Read Dossier →</span></a>`,
			href,
			html.EscapeString(item.CompletedAt.Format("02 Jan 2006")),
			html.EscapeString(item.Title),
		)
	}
	if relatedHTML.Len() == 0 {
		relatedHTML.WriteString(`<p class="growth-empty">More Dossiers will appear as this path develops.</p>`)
	}
	startURL := "/go/" + url.PathEscape(issue.PublicID) + "/start"
	panel := `<section class="public-growth" aria-label="Continue this learning path">` +
		`<div class="public-path-context"><p class="growth-label">A path maintained by ` + html.EscapeString(site.DisplayName) + `</p>` +
		`<h2>` + html.EscapeString(issue.NewsletterOutcome) + `</h2>` +
		`<p>` + html.EscapeString(issue.NewsletterTopic) + `</p>` +
		`<a href="/topics/` + url.PathEscape(issue.NewsletterPublicSlug) + `">See the complete path →</a></div>` +
		`<div class="public-follow"><p class="growth-label">Follow this path</p>` +
		`<p>Get an email when ` + html.EscapeString(site.DisplayName) + ` publishes the next Dossier in this path.</p>` +
		`<form method="post" action="/follow/` + url.PathEscape(issue.PublicID) + `">` +
		`<label>Email address<input type="email" name="email" maxlength="320" autocomplete="email" required></label>` +
		`<label class="follow-honeypot" aria-hidden="true">Website<input name="website" tabindex="-1" autocomplete="off"></label>` +
		`<button type="submit">Send confirmation</button></form>` +
		`<small>Double opt-in. Unsubscribe in one click.</small></div>` +
		`<div class="public-share"><p class="growth-label">Share this Dossier</p>` +
		`<a href="/go/` + url.PathEscape(issue.PublicID) + `/linkedin" rel="noreferrer">LinkedIn</a>` +
		`<a href="/go/` + url.PathEscape(issue.PublicID) + `/x" rel="noreferrer">X</a>` +
		`<a href="/go/` + url.PathEscape(issue.PublicID) + `/email">Email</a></div>` +
		`<div class="public-related"><p class="growth-label">Continue reading</p><div>` + relatedHTML.String() + `</div></div>` +
		`<div class="public-build"><p class="growth-label">Start a path like this</p>` +
		`<h2>Give Learnloom a topic. This Dossier's sources become the seeds of a private, personalized path built just for you.</h2>` +
		`<a href="` + html.EscapeString(startURL) + `">Start a path like this →</a></div></section>`
	return strings.Replace(document, "</body>", panel+"</body>", 1)
}

func (s *Server) handlePublicPathFollow(
	response http.ResponseWriter,
	request *http.Request,
	host RequestHost,
	publicID string,
) {
	expectedOrigin := "https://" + host.Hostname
	if origin := strings.TrimSuffix(request.Header.Get("Origin"), "/"); origin != expectedOrigin {
		writeProblem(response, http.StatusForbidden, "origin_rejected", "The follow origin is invalid.")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, s.cfg.MaxRequestBodyBytes)
	if err := request.ParseForm(); err != nil || request.PostForm.Get("website") != "" {
		writeProblem(response, http.StatusBadRequest, "invalid_follow", "The follow request could not be read.")
		return
	}
	site, err := s.store.GetPublicSite(request.Context(), host.Username)
	if err != nil {
		s.readingNotFound(response, request)
		return
	}
	issue, err := s.store.GetPublicIssue(request.Context(), site.Username, publicID)
	if err != nil {
		s.readingNotFound(response, request)
		return
	}
	fingerprint := s.publicVisitorFingerprint(response, request)
	allowed, err := s.store.AllowRequest(
		request.Context(), fingerprint, "public-path-follow", time.Hour, 5, time.Now().UTC(),
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if !allowed {
		writeProblem(response, http.StatusTooManyRequests, "rate_limited", "Please wait before requesting another confirmation.")
		return
	}
	if err := s.store.RequestPublicPathFollow(
		request.Context(), site.Username, issue.PublicID,
		request.PostForm.Get("email"), time.Now().UTC(),
	); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.readingNotFound(response, request)
			return
		}
		writeProblem(response, http.StatusBadRequest, "invalid_follow", "Enter a valid email address.")
		return
	}
	body := `<main id="main-content"><section class="report-thanks">` +
		`<p class="eyebrow">Check your inbox</p><h1>Confirm this path follow.</h1>` +
		`<p>We sent a confirmation link. Nothing will be delivered unless you confirm it.</p>` +
		`<a class="text-link" href="/d/` + url.PathEscape(issue.PublicID) + `/` + url.PathEscape(issue.PublicSlug) + `">Return to the Dossier <span>→</span></a>` +
		`</section></main>`
	s.sendReadingPage(response, request, "Confirm your follow", "Check your inbox to confirm.", expectedOrigin, body, false)
	response.Header().Set("Cache-Control", "private, no-store")
}

func (s *Server) handlePublicGrowthClick(
	response http.ResponseWriter,
	request *http.Request,
	host RequestHost,
	publicID, action string,
) {
	site, err := s.store.GetPublicSite(request.Context(), host.Username)
	if err != nil {
		s.readingNotFound(response, request)
		return
	}
	issue, err := s.store.GetPublicIssue(request.Context(), site.Username, publicID)
	if err != nil {
		s.readingNotFound(response, request)
		return
	}
	canonical := "https://" + host.Hostname + "/d/" + url.PathEscape(issue.PublicID) + "/" + url.PathEscape(issue.PublicSlug)
	event := store.PublicGrowthShare
	channel := action
	var destination string
	switch action {
	case "linkedin":
		destination = "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(canonical)
	case "x":
		destination = "https://twitter.com/intent/tweet?text=" +
			url.QueryEscape(issue.Title+" — a Learnloom Dossier") + "&url=" + url.QueryEscape(canonical)
	case "email":
		destination = "mailto:?subject=" + url.QueryEscape(issue.Title+" — a Learnloom Dossier") +
			"&body=" + url.QueryEscape(canonical)
	case "start":
		event = store.PublicGrowthCTAClick
		channel = ""
		destination = strings.TrimRight(s.cfg.AppOrigin, "/") +
			"/sign-up?utm_source=public_dossier&utm_medium=referral&utm_campaign=build_path&source_dossier=" +
			url.QueryEscape(issue.PublicID)
	default:
		s.readingNotFound(response, request)
		return
	}
	fingerprint := s.publicVisitorFingerprint(response, request)
	if !isLikelyAutomatedVisitor(request.UserAgent()) {
		if err := s.store.RecordPublicGrowthEvent(
			request.Context(), site.Username, issue.PublicID,
			event, channel, fingerprint, time.Now().UTC(),
		); err != nil {
			s.logger.WarnContext(request.Context(), "record public growth click", "error", err)
		}
	}
	response.Header().Set("Cache-Control", "private, no-store")
	http.Redirect(response, request, destination, http.StatusFound)
}

const publicReferralCookie = "ll_public_ref"

// validPublicReferralCookieValue reports whether the referral cookie carries
// the exact 32-byte hex value the reading surface sets.
func validPublicReferralCookieValue(value string) bool {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == 32
}

func (s *Server) publicVisitorFingerprint(
	response http.ResponseWriter,
	request *http.Request,
) string {
	value := ""
	if cookie, err := request.Cookie(publicReferralCookie); err == nil {
		value = strings.TrimSpace(cookie.Value)
	}
	if !validPublicReferralCookieValue(value) {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			raw = []byte(uuid.NewString() + uuid.NewString())[:32]
		}
		value = hex.EncodeToString(raw)
		http.SetCookie(response, &http.Cookie{
			Name: publicReferralCookie, Value: value, Path: "/",
			Domain: "." + s.cfg.RootDomain, MaxAge: 30 * 24 * 60 * 60,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
	}
	return s.publicReferralFingerprint(value)
}

func (s *Server) publicReferralFingerprint(value string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.CSRFSecret))
	_, _ = mac.Write([]byte("learnloom-public-referral\x00" + value))
	return hex.EncodeToString(mac.Sum(nil))
}

func isLikelyAutomatedVisitor(userAgent string) bool {
	value := strings.ToLower(userAgent)
	for _, marker := range []string{"bot", "crawler", "spider", "preview", "slurp", "headless"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return value == ""
}

const readingCSS = `
:root{color-scheme:light;--ink:#151719;--body:#2e3238;--muted:#5c626a;--faint:#8a9099;--line:rgba(21,23,25,.12);--font:-apple-system,system-ui,"Segoe UI",sans-serif}
*{box-sizing:border-box}
html{scroll-behavior:smooth;background:#fff}
body{min-width:320px;margin:0;overflow-x:hidden;background:#fff;color:var(--ink);font:16px/1.65 var(--font);-webkit-font-smoothing:antialiased}
a{color:inherit}
:focus-visible{outline:2px solid var(--ink);outline-offset:2px}
.skip-link{position:fixed;z-index:100;top:8px;left:8px;min-height:44px;display:inline-flex;align-items:center;padding:0 16px;border-radius:3px;background:var(--ink);color:#fff;transform:translateY(-160%);font-size:13px;font-weight:650}
.skip-link:focus{transform:none}
.site-header{position:relative;z-index:10;width:min(1120px,calc(100% - 64px));height:88px;display:flex;align-items:center;justify-content:space-between;margin:auto;border-bottom:1px solid var(--line)}
.site-brand{min-height:44px;display:inline-flex;align-items:center;gap:11px;color:inherit;text-decoration:none}
.site-brand>span:last-child{display:grid}
.site-brand strong{font-size:15px;font-weight:700;letter-spacing:-.02em}
.site-brand small{color:var(--muted);font-size:8px;font-weight:650;letter-spacing:.14em;text-transform:uppercase}
.brand-mark{width:32px;height:32px;display:inline-grid;place-items:center;flex:0 0 auto;overflow:hidden;border:1px solid var(--line);border-radius:3px;background:#fff}
.brand-mark img{width:100%;height:100%;display:block;filter:grayscale(100%)}
.site-header nav{display:flex;align-items:center;gap:26px}
.site-header nav a{min-height:44px;display:inline-flex;align-items:center;color:var(--body);font-size:12px;font-weight:650;text-decoration:none}
.site-header nav a:hover{color:var(--ink);text-decoration:underline}
.site-header nav a.current{color:var(--ink);box-shadow:inset 0 -2px 0 var(--ink)}
main{width:min(1120px,calc(100% - 64px));margin:auto}
.reading-hero{padding:72px 0 64px;border-bottom:1px solid var(--line)}
.home-hero,.topic-hero{display:grid;grid-template-columns:minmax(0,1.4fr) minmax(240px,.6fr);align-items:center;gap:clamp(48px,7vw,110px)}
.hero-copy{max-width:780px}
.eyebrow{margin:0 0 20px;color:var(--muted);font-size:10px;font-weight:700;letter-spacing:.18em;text-transform:uppercase}
.eyebrow>span{width:24px;height:1px;display:inline-block;margin:0 10px 3px 0;background:var(--faint)}
.reading-hero h1{margin:0;font-size:clamp(30px,4.4vw,46px);font-weight:700;line-height:1.1;letter-spacing:-.02em;text-wrap:balance}
.topic-hero h1{font-size:clamp(28px,4vw,40px)}
.hero-description{max-width:640px;margin:18px 0 0;color:var(--body);font-size:17px;line-height:1.6;text-wrap:balance}
.text-link,.back-link{min-height:44px;display:inline-flex;align-items:center;gap:10px;margin-top:34px;color:var(--ink);font-size:12px;font-weight:700;text-decoration:none}
.text-link span{width:30px;height:30px;display:grid;place-items:center;border:1px solid var(--line);border-radius:2px;font-size:11px}
.text-link:hover,.back-link:hover{text-decoration:underline}
.back-link{margin:0 0 42px;color:var(--muted)}
.back-link:hover{color:var(--ink)}
.archive-note{padding:28px;border:1px solid var(--line);background:#fff}
.note-label{margin:0 0 18px;color:var(--muted);font-size:9px;font-weight:700;letter-spacing:.16em;text-transform:uppercase}
.archive-stats{display:grid;grid-template-columns:1fr 1fr;gap:24px;margin:0;padding-top:18px;border-top:1px solid var(--line)}
.archive-stats div{display:grid;gap:3px}
.archive-stats dt{color:var(--muted);font-size:9px;font-weight:700;letter-spacing:.13em;text-transform:uppercase}
.archive-stats dd{margin:0;font-size:26px;font-weight:700;line-height:1}
.note-foot{margin:20px 0 0;color:var(--muted);font-size:12px;line-height:1.6}
.reading-section{padding:72px 0;border-top:1px solid var(--line)}
.reading-hero+.reading-section{border-top:0}
.section-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:48px;margin-bottom:44px}
.section-heading .eyebrow{margin-bottom:10px}
.section-heading h2{margin:0;font-size:clamp(24px,3vw,34px);font-weight:700;line-height:1.15;letter-spacing:-.02em}
.section-heading>p{max-width:400px;margin:0;color:var(--muted);font-size:13px;line-height:1.6}
.topics{display:grid;border-top:1px solid var(--line)}
.topic-card{position:relative;min-height:112px;display:grid;grid-template-columns:52px minmax(0,1fr) auto 30px;align-items:center;gap:22px;padding:20px 2px;border-bottom:1px solid var(--line);text-decoration:none}
.topic-index{color:var(--muted);font-size:12px;font-weight:650}
.topic-copy{display:grid;gap:3px;min-width:0}
.topic-copy strong{font-size:clamp(18px,2vw,22px);font-weight:700;letter-spacing:-.01em}
.topic-card:hover .topic-copy strong{text-decoration:underline}
.topic-copy>span{overflow:hidden;color:var(--muted);font-size:12px;text-overflow:ellipsis;white-space:nowrap}
.topic-count{display:grid;min-width:96px;color:var(--ink);font-size:18px;font-weight:650;text-align:right}
.topic-count small{color:var(--muted);font-size:8px;font-weight:650;letter-spacing:.12em;text-transform:uppercase}
.arrow{color:var(--muted);font-size:13px}
.topic-card:hover .arrow{color:var(--ink)}
.issues{display:grid;grid-template-columns:1fr;gap:0;border-top:1px solid var(--line)}
.issue-card{min-width:0;border:0;border-bottom:1px solid var(--line);border-radius:0;background:#fff}
.issue-card:hover h3{text-decoration:underline}
.issue-link{height:100%;min-height:150px;display:grid;grid-template-columns:180px minmax(0,1fr) 180px;align-items:center;gap:32px;padding:34px 6px;color:inherit;text-decoration:none}
.issue-top{display:grid;gap:6px;align-content:start}
.issue-number{color:var(--muted);font-size:12px;font-weight:650}
.issue-topic{color:var(--ink);font-size:9px;font-weight:700;letter-spacing:.13em;text-transform:uppercase}
.issue-top .arrow{display:none}
.issue-card h3{margin:0;font-size:clamp(18px,1.9vw,23px);font-weight:700;line-height:1.25;letter-spacing:-.01em;text-wrap:balance}
.issue-meta{display:grid;justify-items:end;gap:6px;margin:0;padding:0;border:0;color:var(--muted);font-size:9px;font-weight:650;letter-spacing:.08em;text-transform:uppercase}
.issue-meta span{color:var(--ink);font-weight:700}
.latest-section{padding-bottom:96px}
.stream-note{display:grid;justify-items:start;gap:5px;padding:26px;border:1px solid var(--line);background:#fff}
.stream-note>span,.stream-note small{color:var(--muted);font-size:9px;font-weight:650;letter-spacing:.13em;text-transform:uppercase}
.stream-note strong{margin:14px 0 2px;font-size:40px;font-weight:700;line-height:1}
.leaf-mark{width:40px;height:40px;display:block;border-radius:0;filter:grayscale(100%)}
.empty-state{grid-column:1/-1;min-height:230px;display:grid;place-items:center;align-content:center;gap:7px;padding:36px;border:1px dashed rgba(21,23,25,.25);color:var(--muted);text-align:center}
.empty-state .leaf-mark{margin-bottom:12px}
.empty-state strong{color:var(--ink);font-size:17px;font-weight:700}
.empty-state span{font-size:13px}
.site-footer{width:min(1120px,calc(100% - 64px));display:flex;align-items:flex-end;justify-content:space-between;gap:48px;margin:auto;padding:54px 0 60px;border-top:1px solid var(--line)}
.site-footer>div:first-child>p{max-width:440px;margin:24px 0 0;color:var(--muted);font-size:12px;line-height:1.6}
.footer-meta{display:grid;justify-items:end;gap:5px}
.footer-meta span{color:var(--muted);font-size:9px;letter-spacing:.13em;text-transform:uppercase}
.footer-meta a{min-height:44px;display:inline-flex;align-items:center;font-size:14px;font-weight:650;text-decoration:none}
.footer-meta a:hover{text-decoration:underline}
.not-found{min-height:100vh;width:min(720px,calc(100% - 48px));display:grid;place-items:center;align-content:center;margin:auto;text-align:center}
.not-found .eyebrow{margin-top:30px}
.not-found h1{margin:0;font-size:clamp(28px,4.5vw,40px);font-weight:700;line-height:1.12;letter-spacing:-.02em;text-wrap:balance}
.not-found>p:not(.eyebrow){color:var(--muted)}
.button{min-height:48px;display:inline-flex;align-items:center;gap:11px;margin-top:20px;padding:0 22px;border-radius:3px;background:var(--ink);color:#fff;font-size:12px;font-weight:700;text-decoration:none}
.button:hover{background:var(--body)}
.report-thanks{max-width:720px;margin:96px auto;padding:0 24px}
.report-thanks h1{margin:0 0 28px;font-size:30px;line-height:36px;font-weight:700}
.report-thanks>p:not(.eyebrow){color:var(--body)}
@media(max-width:860px){.site-header,main,.site-footer{width:min(100% - 40px,680px)}.site-header{height:76px}.site-header nav{gap:18px}.site-header nav a:last-child{display:none}.reading-hero{padding:56px 0 48px}.home-hero,.topic-hero{grid-template-columns:1fr;gap:44px}.archive-note,.stream-note{max-width:440px}.reading-section{padding:56px 0}.section-heading{display:grid;gap:18px}.issue-link{grid-template-columns:140px minmax(0,1fr) auto;gap:20px}.topic-card{grid-template-columns:36px minmax(0,1fr) auto 20px;gap:12px}.topic-copy>span{white-space:normal}.latest-section{padding-bottom:72px}}
@media(max-width:580px){.site-header nav{gap:14px}.site-header nav a{font-size:10px}.site-brand small{display:none}.reading-hero h1{font-size:clamp(28px,9vw,36px)}.hero-description{font-size:16px}.reading-section{padding:48px 0}.section-heading h2{font-size:28px}.topics{border:0}.topic-card{grid-template-columns:30px minmax(0,1fr) 18px;min-height:100px}.topic-count{display:none}.issue-link{grid-template-columns:1fr;gap:8px;min-height:0;padding:26px 2px}.issue-top{display:flex;align-items:baseline;gap:10px}.issue-meta{display:flex;justify-items:start;gap:16px}.issue-card h3{margin:0;font-size:21px}.site-footer{align-items:flex-start;display:grid}.footer-meta{justify-items:start}.latest-section{padding-bottom:56px}.stream-note strong{font-size:36px}}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
`

const readingArticleCSS = `
:root{color-scheme:light;--a-ink:#151719;--a-body:#2e3238;--a-muted:#5c626a;--a-faint:#8a9099;--a-line:rgba(21,23,25,.12);--a-font:-apple-system,system-ui,"Segoe UI",sans-serif}
body.public-dossier{min-width:320px!important;margin:0!important;background:#fff!important;color:var(--a-body)!important;font:16px/1.65 var(--a-font)!important;-webkit-font-smoothing:antialiased}
.public-dossier *{box-sizing:border-box}
.public-dossier :focus-visible{outline:2px solid var(--a-ink);outline-offset:2px}
.article-header{width:min(1120px,calc(100% - 64px));height:86px;display:flex;align-items:center;justify-content:space-between;margin:auto;border-bottom:1px solid var(--a-line)}
.article-brand{min-height:44px;display:inline-flex;align-items:center;gap:10px;color:inherit;text-decoration:none}
.article-brand>span:last-child{display:grid}
.article-brand strong{font-size:14px;font-weight:700;letter-spacing:-.02em}
.article-brand small{color:var(--a-muted);font-size:8px;font-weight:650;letter-spacing:.13em;text-transform:uppercase}
.public-dossier .brand-mark{width:30px;height:30px;display:inline-grid;place-items:center;overflow:hidden;border:1px solid var(--a-line);border-radius:3px;background:#fff}
.public-dossier .brand-mark img{width:100%;height:100%;display:block;filter:grayscale(100%)}
.article-back{min-height:44px;display:inline-flex;align-items:center;color:var(--a-muted);font-size:11px;font-weight:650;text-decoration:none}
.article-back:hover{color:var(--a-ink);text-decoration:underline}
body.public-dossier>main{max-width:760px!important;margin:0 auto!important;padding:32px 20px 96px!important}
body.public-dossier>main>div{padding:12px 0 32px!important;border:0!important;border-radius:0!important;background:#fff!important;box-shadow:none!important}
body.public-dossier>main>div>p:first-child{color:var(--a-muted)!important;font-size:10px!important;font-weight:700!important;letter-spacing:.16em!important;text-transform:uppercase!important}
body.public-dossier p,body.public-dossier li{line-height:1.65!important}
body.public-dossier hr{border-color:var(--a-line)!important}
body.public-dossier main a{color:var(--a-ink);text-decoration:underline;text-decoration-thickness:1px;text-underline-offset:2px}
body.public-dossier main a:hover{text-decoration-thickness:2px}
body.public-dossier code{background:rgba(21,23,25,.06)!important;color:var(--a-ink)!important;border-radius:3px!important;padding:1px 4px!important}
body.public-dossier main section[style*="f0c36a"],body.public-dossier main section[style*="fff8e8"]{background:#fff!important;border-color:var(--a-line)!important;border-radius:0!important}
body.public-dossier main section[style*="fff8e8"] p{color:var(--a-body)!important}
body.public-dossier main [style*="color:#047857"]{color:var(--a-ink)!important}
body.public-dossier main [style*="background:#047857"]{background:var(--a-ink)!important;border-radius:3px!important;color:#fff!important;text-decoration:none!important}
body.public-dossier main [style*="border-radius:999px"]{border-radius:3px!important}
.public-growth{width:min(920px,calc(100% - 48px));display:grid;grid-template-columns:1.15fr .85fr;gap:0 44px;margin:0 auto 84px;padding-top:56px;border-top:1px solid var(--a-line)}
.public-growth>div{padding:0;border:0;border-radius:0;background:#fff;box-shadow:none}
.public-path-context{grid-column:1;grid-row:1}
.public-follow{grid-column:2;grid-row:1}
.public-share{grid-column:1/-1;grid-row:2}
.public-related{grid-column:1/-1;grid-row:3}
.public-build{grid-column:1/-1;grid-row:4}
.growth-label{margin:0 0 10px!important;color:var(--a-muted)!important;font-size:9px!important;font-weight:700;letter-spacing:.14em;text-transform:uppercase}
.public-growth h2{margin:0 0 9px!important;color:var(--a-ink)!important;font-size:clamp(21px,2.6vw,26px)!important;line-height:1.2!important}
.public-growth p{margin:0 0 12px;color:var(--a-muted)}
.public-growth a{font-size:12px;font-weight:700;text-decoration:none}
.public-growth a:hover{text-decoration:underline}
.public-share{display:flex;flex-wrap:wrap;justify-content:flex-start}
.public-share .growth-label{flex:0 0 100%}
.public-share a{min-height:44px;display:inline-flex;align-items:center;padding:0 16px;border-left:1px solid var(--a-line);color:var(--a-ink)}
.public-share a:first-of-type{padding-left:0;border-left:0}
.public-follow form{display:flex;gap:10px;align-items:flex-end}
.public-follow label{flex:1;color:var(--a-muted);font-size:9px}
.public-follow input{width:100%;min-height:44px;margin-top:6px;padding:0 12px;border:1px solid rgba(21,23,25,.2);border-radius:3px;background:#fff;color:var(--a-ink)}
.public-follow input:focus{border-color:var(--a-ink)}
.public-follow button{align-self:end;min-height:44px;padding:0 16px;border:0;border-radius:3px;background:var(--a-ink);color:#fff;font-weight:700;cursor:pointer}
.public-follow button:hover{background:var(--a-body)}
.public-follow>small{color:var(--a-muted);font-size:8px}
.follow-honeypot{position:absolute!important;left:-10000px!important}
.public-share,.public-related,.public-build{margin-top:36px;border-top:1px solid var(--a-line);padding-top:30px}
.public-related>div{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:0 36px}
.public-related>div>a{min-height:44px;display:grid;align-content:center;gap:4px;padding:2px 0;color:var(--a-ink)}
.public-related small,.public-related span{color:var(--a-muted);font-size:8px;letter-spacing:.08em;text-transform:uppercase}
.public-related strong{font-size:15px;font-weight:650;line-height:1.3}
.public-build a{min-height:44px;display:inline-flex;align-items:center;margin-top:14px;padding:0 18px;border:1px solid var(--a-ink);border-radius:3px;background:var(--a-ink);color:#fff!important;text-decoration:none}
.public-build a:hover{background:var(--a-body)}
.growth-empty{color:var(--a-muted)!important;font-size:11px}
@media(max-width:700px){.public-growth{grid-template-columns:1fr;gap:0;margin-bottom:64px;padding-top:40px}.public-path-context,.public-follow,.public-share,.public-related,.public-build{grid-column:auto;grid-row:auto}.public-follow form{display:grid}.public-related>div{grid-template-columns:1fr;gap:0}.public-related>div>a{padding:12px 0}.article-header{width:calc(100% - 40px);height:72px}.article-brand small{display:none}.article-back{max-width:150px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}body.public-dossier>main{padding:32px 20px 72px!important}body.public-dossier>main>div{padding:12px 0 24px!important}}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
`

func firstReadingText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
