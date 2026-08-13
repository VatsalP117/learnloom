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
			"The garden is taking root.",
			"Published learning streams will gather here.",
		))
	}
	body := renderReadingHeader(site, "home") +
		`<main id="main-content">` +
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
		`<main id="main-content"><header class="hero topic-hero">` +
		`<div class="hero-copy"><a class="back-link" href="/">← Back to the garden</a>` +
		`<p class="eyebrow"><span></span>Learning stream</p><h1>` +
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
		"default-src 'none'; style-src 'unsafe-inline'; img-src https: data:; "+
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
			`</style></head><body><main class="not-found"><div class="leaf-mark" aria-hidden="true"><i></i></div>` +
			`<p class="eyebrow">A quiet corner of the web</p><h1>This page has wandered off the path.</h1>` +
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

func renderReadingHeader(site domain.PersonalSite, current string) string {
	topicsClass := ""
	if current == "topics" {
		topicsClass = ` class="current"`
	}
	return `<a class="skip-link" href="#main-content">Skip to content</a>` +
		`<header class="site-header"><a class="site-brand" href="/" aria-label="` +
		html.EscapeString(site.DisplayName) + ` home"><span class="brand-mark" aria-hidden="true">✣</span>` +
		`<span><strong>` + html.EscapeString(site.DisplayName) +
		`</strong><small>Learning garden</small></span></a>` +
		`<nav aria-label="Personal site navigation"><a` + topicsClass +
		` href="/#topics">Topics</a><a href="/#latest">Archive</a>` +
		`<a href="/#about">About</a></nav></header>`
}

func renderReadingFooter(site domain.PersonalSite) string {
	return `<footer class="site-footer" id="about"><div><a class="site-brand" href="/">` +
		`<span class="brand-mark" aria-hidden="true">✣</span><span><strong>` +
		html.EscapeString(site.DisplayName) +
		`</strong><small>A personal learning garden</small></span></a>` +
		`<p>` + html.EscapeString(firstReadingText(
		site.Description,
		"A quiet place for ideas worth returning to.",
	)) + `</p></div><div class="footer-meta"><span>Grown with</span>` +
		`<a href="https://learnloom.blog">Learnloom ↗</a></div></footer>`
}

func renderReadingEmpty(title, description string) string {
	return `<div class="empty-state"><div class="leaf-mark" aria-hidden="true"><i></i></div>` +
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
		`<span class="brand-mark" aria-hidden="true">✣</span><span><strong>` +
		html.EscapeString(site.DisplayName) + `</strong><small>Learning garden</small></span></a>` +
		`<a class="article-back" href="/topics/` +
		url.PathEscape(issue.NewsletterPublicSlug) + `">` +
		html.EscapeString(issue.NewsletterName) + ` <span aria-hidden="true">↗</span></a></header>`
	legacyBody := `<body style="margin:0;background:#f5f5f4;color:#0f172a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">`
	if strings.Contains(document, legacyBody) {
		return strings.Replace(
			document,
			legacyBody,
			`<body class="public-dossier">`+header,
			1,
		)
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
		`<div class="public-build"><p class="growth-label">Build your own path</p>` +
		`<h2>Give Learnloom a topic. It finds and ranks useful sources, then builds connected lessons that remember where you are.</h2>` +
		`<a href="` + html.EscapeString(startURL) + `">Start your private learning path →</a></div></section>`
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

func (s *Server) publicVisitorFingerprint(
	response http.ResponseWriter,
	request *http.Request,
) string {
	value := ""
	if cookie, err := request.Cookie(publicReferralCookie); err == nil {
		value = strings.TrimSpace(cookie.Value)
	}
	if decoded, err := hex.DecodeString(value); err != nil || len(decoded) != 32 {
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
:root{color-scheme:light;--ink:#17211b;--muted:#5f6b63;--paper:#f7f6f0;--paper-warm:#f1eee4;--card:rgba(255,255,251,.78);--green:#496b4c;--green-deep:#1d2c22;--lime:#abc477;--line:rgba(23,33,27,.12);--serif:"Iowan Old Style","Palatino Linotype",Palatino,Georgia,serif;--sans:"Avenir Next",Avenir,"Segoe UI",ui-sans-serif,sans-serif}
*{box-sizing:border-box}html{scroll-behavior:smooth;background:var(--paper)}body{min-width:320px;margin:0;overflow-x:hidden;background:var(--paper);color:var(--ink);font:15px/1.65 var(--sans);-webkit-font-smoothing:antialiased}
body:before{position:fixed;z-index:-2;inset:0;content:"";background:radial-gradient(circle at 14% 8%,rgba(181,210,232,.62),transparent 32rem),radial-gradient(circle at 84% 13%,rgba(238,217,185,.54),transparent 30rem),linear-gradient(180deg,#eef4f1 0,#f7f6f0 31rem,#f7f6f0 100%)}
body:after{position:fixed;z-index:-1;inset:0;pointer-events:none;content:"";opacity:.22;background-image:url("data:image/svg+xml,%3Csvg viewBox='0 0 180 180' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='.9' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='.13'/%3E%3C/svg%3E")}
a{color:inherit}.skip-link{position:fixed;z-index:100;top:8px;left:8px;padding:10px 14px;border-radius:8px;background:var(--green-deep);color:#fff;transform:translateY(-150%)}.skip-link:focus{transform:none}
.site-header{position:relative;z-index:10;width:min(1280px,calc(100% - 72px));height:92px;display:flex;align-items:center;justify-content:space-between;margin:auto;border-bottom:1px solid rgba(23,33,27,.09)}
.site-brand{display:inline-flex;align-items:center;gap:11px;color:inherit;text-decoration:none}.site-brand>span:last-child{display:grid}.site-brand strong{font-size:15px;font-weight:750;letter-spacing:-.035em}.site-brand small{color:var(--muted);font-size:9px;font-weight:700;letter-spacing:.13em;text-transform:uppercase}
.brand-mark{width:34px;height:34px;display:inline-grid;place-items:center;flex:0 0 auto;border-radius:10px;background:var(--green-deep);color:#eef6ec;box-shadow:inset 0 0 0 1px rgba(255,255,255,.14),0 8px 22px rgba(19,36,24,.14);font-size:15px}
.site-header nav{display:flex;align-items:center;gap:30px}.site-header nav a{position:relative;color:rgba(23,33,27,.68);font-size:12px;font-weight:650;text-decoration:none}.site-header nav a:hover,.site-header nav a.current{color:var(--ink)}.site-header nav a:after{position:absolute;right:0;bottom:-9px;left:0;height:1px;content:"";background:var(--green);transform:scaleX(0);transition:transform .18s ease}.site-header nav a:hover:after,.site-header nav a.current:after{transform:scaleX(1)}
main{width:min(1180px,calc(100% - 72px));margin:auto}.hero{min-height:600px;display:grid;grid-template-columns:minmax(0,1.45fr) minmax(260px,.55fr);align-items:center;gap:clamp(52px,8vw,120px);padding:98px 0 90px}.hero-copy{max-width:790px}.eyebrow{margin:0 0 19px;color:#47644c;font-size:10px;font-weight:800;letter-spacing:.15em;text-transform:uppercase}.eyebrow>span{width:7px;height:7px;display:inline-block;margin:0 9px 1px 0;border-radius:50%;background:#87ac57;box-shadow:0 0 0 5px rgba(135,172,87,.12)}
.hero h1{max-width:850px;margin:0;color:var(--ink);font-family:var(--serif);font-size:clamp(58px,8.5vw,112px);font-weight:400;line-height:.9;letter-spacing:-.065em;text-wrap:balance}.hero-description{max-width:650px;margin:27px 0 0;color:#46564b;font-size:clamp(16px,1.7vw,20px);line-height:1.62;text-wrap:balance}.text-link,.back-link{display:inline-flex;align-items:center;gap:11px;margin-top:30px;color:#314538;font-size:12px;font-weight:750;text-decoration:none}.text-link span{width:31px;height:31px;display:grid;place-items:center;border:1px solid var(--line);border-radius:50%;transition:transform .18s ease}.text-link:hover span{transform:translateY(3px)}
.garden-note,.stream-note{position:relative;padding:30px;border:1px solid rgba(255,255,255,.72);border-radius:20px;background:rgba(255,255,251,.5);box-shadow:0 24px 70px rgba(35,64,43,.09);backdrop-filter:blur(18px)}.garden-note:before{position:absolute;inset:10px;border:1px solid rgba(23,33,27,.07);border-radius:13px;content:"";pointer-events:none}.leaf-mark{width:46px;height:46px;display:grid;place-items:center;border:1px solid rgba(23,32,26,.14);border-radius:50%;background:rgba(255,254,249,.58)}.leaf-mark i{width:9px;height:17px;display:block;border-radius:10px 2px 10px 2px;background:#527355;transform:rotate(-36deg)}.garden-note p{position:relative;margin:25px 0 28px;color:#46564b;font-family:var(--serif);font-size:20px;line-height:1.45}.garden-note dl{position:relative;display:grid;grid-template-columns:1fr 1fr;margin:0;padding-top:18px;border-top:1px solid var(--line)}.garden-note dl div{display:grid;gap:2px}.garden-note dt{color:var(--muted);font-size:9px;font-weight:750;letter-spacing:.12em;text-transform:uppercase}.garden-note dd{margin:0;font-family:var(--serif);font-size:30px}
.reading-section{padding:105px 0;border-top:1px solid var(--line)}.section-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:50px;margin-bottom:46px}.section-heading .eyebrow{margin-bottom:9px}.section-heading h2{margin:0;font-family:var(--serif);font-size:clamp(40px,5vw,66px);font-weight:400;line-height:1;letter-spacing:-.05em}.section-heading>p{max-width:390px;margin:0;color:var(--muted);font-size:13px}
.topics{display:grid;border-top:1px solid var(--line)}.topic-card{position:relative;min-height:120px;display:grid;grid-template-columns:55px minmax(0,1fr) auto 34px;align-items:center;gap:22px;padding:22px 4px;border-bottom:1px solid var(--line);text-decoration:none;transition:padding .18s ease,background .18s ease}.topic-card:hover{padding-right:18px;padding-left:18px;background:rgba(255,255,251,.46)}.topic-index,.issue-number{color:#819087;font-family:var(--serif);font-size:13px;font-style:italic}.topic-copy{display:grid;gap:4px}.topic-copy strong{font-family:var(--serif);font-size:clamp(23px,2.5vw,32px);font-weight:400;letter-spacing:-.03em}.topic-copy>span{overflow:hidden;color:var(--muted);font-size:12px;text-overflow:ellipsis;white-space:nowrap}.topic-count{display:grid;min-width:92px;color:var(--ink);font-family:var(--serif);font-size:23px;text-align:right}.topic-count small{color:var(--muted);font-family:var(--sans);font-size:8px;font-weight:750;letter-spacing:.11em;text-transform:uppercase}.arrow{transition:transform .18s ease}.topic-card:hover .arrow,.issue-link:hover .arrow{transform:translate(3px,-3px)}
.latest-section{padding-bottom:130px}.issues{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.issue-card{min-width:0;border:1px solid rgba(23,33,27,.1);border-radius:17px;background:var(--card);box-shadow:0 12px 38px rgba(29,48,34,.045);backdrop-filter:blur(12px);transition:transform .18s ease,box-shadow .18s ease}.issue-card:hover{transform:translateY(-4px);box-shadow:0 20px 55px rgba(29,48,34,.09)}.issue-link{height:100%;min-height:300px;display:flex;flex-direction:column;padding:25px;color:inherit;text-decoration:none}.issue-top{display:flex;align-items:center;gap:12px}.issue-topic{min-width:0;overflow:hidden;color:#527355;font-size:9px;font-weight:800;letter-spacing:.11em;text-overflow:ellipsis;text-transform:uppercase;white-space:nowrap}.issue-top .arrow{margin-left:auto}.issue-card h3{margin:45px 0 30px;font-family:var(--serif);font-size:clamp(23px,2.2vw,31px);font-weight:400;line-height:1.17;letter-spacing:-.035em;text-wrap:balance}.issue-meta{display:flex;align-items:center;justify-content:space-between;gap:14px;margin-top:auto;padding-top:18px;border-top:1px solid var(--line);color:var(--muted);font-size:9px;font-weight:750;letter-spacing:.06em;text-transform:uppercase}.issue-meta span{color:#314538}
.topic-hero{min-height:525px}.topic-hero h1{font-size:clamp(54px,7.5vw,96px)}.back-link{margin:0 0 40px}.stream-note{display:grid;justify-items:start;gap:4px}.stream-note>span,.stream-note small{color:var(--muted);font-size:9px;font-weight:750;letter-spacing:.12em;text-transform:uppercase}.stream-note strong{margin:12px 0 1px;font-family:var(--serif);font-size:72px;font-weight:400;line-height:1}
.empty-state{grid-column:1/-1;min-height:250px;display:grid;place-items:center;align-content:center;gap:8px;padding:35px;border:1px dashed rgba(23,33,27,.2);border-radius:17px;color:var(--muted);text-align:center}.empty-state .leaf-mark{margin-bottom:10px}.empty-state strong{color:var(--ink);font-family:var(--serif);font-size:23px;font-weight:400}.empty-state span{font-size:12px}
.site-footer{width:min(1280px,calc(100% - 72px));display:flex;align-items:flex-end;justify-content:space-between;gap:50px;margin:auto;padding:58px 0 64px;border-top:1px solid var(--line)}.site-footer>div:first-child>p{max-width:450px;margin:26px 0 0;color:var(--muted);font-size:12px}.footer-meta{display:grid;justify-items:end;gap:5px}.footer-meta span{color:var(--muted);font-size:9px;letter-spacing:.12em;text-transform:uppercase}.footer-meta a{font-family:var(--serif);font-size:18px;text-decoration:none}
.not-found{min-height:100vh;width:min(760px,calc(100% - 48px));display:grid;place-items:center;align-content:center;margin:auto;text-align:center}.not-found .eyebrow{margin-top:28px}.not-found h1{margin:0;font-family:var(--serif);font-size:clamp(45px,7vw,78px);font-weight:400;line-height:.98;letter-spacing:-.055em}.not-found>p:not(.eyebrow){color:var(--muted)}.button{min-height:48px;display:inline-flex;align-items:center;gap:12px;margin-top:18px;padding:0 20px;border-radius:999px;background:var(--green-deep);color:#fff;font-size:12px;font-weight:750;text-decoration:none;box-shadow:0 12px 32px rgba(18,35,23,.18)}
.topics-section{padding-top:64px}.topics-section + .latest-section{border-top:0;padding-top:56px}
@media(max-width:860px){.site-header,main,.site-footer{width:min(100% - 40px,680px)}.site-header{height:76px}.site-header nav{gap:18px}.site-header nav a:last-child{display:none}.hero{min-height:auto;grid-template-columns:1fr;gap:45px;padding:75px 0}.hero h1{font-size:clamp(52px,15vw,80px)}.garden-note,.stream-note{max-width:440px}.reading-section{padding:78px 0}.section-heading{display:grid;gap:20px}.issues{grid-template-columns:1fr 1fr}.topic-card{grid-template-columns:36px minmax(0,1fr) 70px 20px;gap:12px}.topic-copy>span{white-space:normal}.latest-section{padding-bottom:90px}}
@media(max-width:580px){.site-header nav{gap:14px}.site-header nav a{font-size:10px}.site-brand small{display:none}.hero{padding:58px 0 68px}.hero-description{font-size:15px}.garden-note{padding:25px}.reading-section{padding:66px 0}.section-heading h2{font-size:44px}.topics{border:0}.topic-card{grid-template-columns:30px minmax(0,1fr) 18px;min-height:105px}.topic-count{display:none}.issues{grid-template-columns:1fr}.issue-link{min-height:250px}.issue-card h3{margin:34px 0 25px;font-size:28px}.site-footer{align-items:flex-start;display:grid}.footer-meta{justify-items:start}.latest-section{padding-bottom:75px}}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}.site-header nav a:after,.topic-card,.arrow,.issue-card,.text-link span{transition:none}}
`

const readingArticleCSS = `
:root{--article-ink:#17211b;--article-muted:#5f6b63;--article-paper:#f7f6f0;--article-green:#496b4c;--article-deep:#1d2c22;--article-line:rgba(23,33,27,.12);--article-serif:"Iowan Old Style","Palatino Linotype",Palatino,Georgia,serif;--article-sans:"Avenir Next",Avenir,"Segoe UI",ui-sans-serif,sans-serif}
body.public-dossier{min-width:320px!important;margin:0!important;background:radial-gradient(circle at 16% 0,rgba(181,210,232,.45),transparent 31rem),linear-gradient(180deg,#eef3ef 0,#f7f6f0 35rem)!important;color:var(--article-ink)!important;font-family:var(--article-sans)!important;-webkit-font-smoothing:antialiased}
.public-dossier *{box-sizing:border-box}.article-header{width:min(1120px,calc(100% - 64px));height:88px;display:flex;align-items:center;justify-content:space-between;margin:auto;border-bottom:1px solid var(--article-line)}.article-brand{display:inline-flex;align-items:center;gap:10px;color:inherit;text-decoration:none}.article-brand>span:last-child{display:grid}.article-brand strong{font-size:14px;letter-spacing:-.03em}.article-brand small{color:var(--article-muted);font-size:8px;font-weight:700;letter-spacing:.12em;text-transform:uppercase}.public-dossier .brand-mark{width:32px;height:32px;display:inline-grid;place-items:center;border-radius:9px;background:var(--article-deep);color:#eef6ec;font-size:14px}.article-back{color:#314538;font-size:11px;font-weight:700;text-decoration:none}
body.public-dossier>main{max-width:820px!important;margin:0 auto!important;padding:72px 24px 110px!important}body.public-dossier>main>div{padding:clamp(30px,6vw,64px)!important;border:1px solid rgba(23,33,27,.1)!important;border-radius:20px!important;background:rgba(255,255,251,.78)!important;box-shadow:0 24px 70px rgba(35,64,43,.08)!important;backdrop-filter:blur(16px)}
body.public-dossier>main>div>p:first-child{color:var(--article-green)!important;font-size:10px!important;letter-spacing:.14em!important}body.public-dossier>main>div>h1{margin-bottom:36px!important;font-family:var(--article-serif)!important;font-size:clamp(38px,6vw,60px)!important;font-weight:400!important;line-height:1.02!important;letter-spacing:-.045em!important}body.public-dossier h2,body.public-dossier h3,body.public-dossier h4{font-family:var(--article-serif)!important;font-weight:400!important;letter-spacing:-.025em!important}body.public-dossier h2{font-size:29px!important}body.public-dossier p,body.public-dossier li{color:#344039;line-height:1.78!important}body.public-dossier hr{border-color:var(--article-line)!important;margin:42px 0!important}body.public-dossier a{color:var(--article-green)!important}body.public-dossier code{background:#eef0e8!important;color:#294232!important}body.public-dossier details{padding:18px 20px;border:1px solid var(--article-line);border-radius:12px;background:rgba(247,246,240,.7)}
.public-growth{width:min(980px,calc(100% - 48px));display:grid;grid-template-columns:1.2fr .8fr;gap:14px;margin:0 auto 90px;padding-top:12px}.public-growth>div{padding:25px;border:1px solid var(--article-line);border-radius:16px;background:rgba(255,255,251,.78);box-shadow:0 14px 45px rgba(35,64,43,.06)}.growth-label{margin:0 0 10px!important;color:var(--article-green)!important;font-size:9px!important;font-weight:800;letter-spacing:.13em;text-transform:uppercase}.public-growth h2{margin:0 0 10px!important;font-size:clamp(23px,3vw,34px)!important;line-height:1.15!important}.public-growth p{margin:0 0 13px}.public-growth a{font-size:11px;font-weight:750;text-decoration:none}.public-share{align-content:start;display:grid;grid-template-columns:repeat(3,auto);justify-content:start;gap:8px}.public-share .growth-label{grid-column:1/-1}.public-share a{padding:8px 10px;border:1px solid var(--article-line);border-radius:999px}.public-follow{grid-column:1/-1}.public-follow form{display:flex;gap:8px}.public-follow label{flex:1;color:var(--article-muted);font-size:9px}.public-follow input{width:100%;min-height:42px;margin-top:5px;padding:0 12px;border:1px solid var(--article-line);border-radius:8px;background:#fff;color:var(--article-ink)}.public-follow button{align-self:end;min-height:42px;padding:0 15px;border:0;border-radius:8px;background:var(--article-deep);color:#fff;font-weight:750;cursor:pointer}.public-follow>small{color:var(--article-muted);font-size:8px}.follow-honeypot{position:absolute!important;left:-10000px!important}.public-related,.public-build{grid-column:1/-1}.public-related>div{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px}.public-related>div>a{display:grid;gap:5px;padding:14px;border:1px solid var(--article-line);border-radius:11px;background:rgba(247,246,240,.7)}.public-related small,.public-related span{color:var(--article-muted);font-size:8px}.public-related strong{font-family:var(--article-serif);font-size:17px;font-weight:400;line-height:1.25}.public-build{color:#eef6ec;background:var(--article-deep)!important}.public-build .growth-label,.public-build p{color:#bbcabf!important}.public-build h2,.public-build a{color:#fff!important}.public-build a{display:inline-flex;margin-top:8px;padding:11px 14px;border-radius:999px;background:#eef6ec;color:var(--article-deep)!important}.growth-empty{color:var(--article-muted)!important;font-size:11px}
@media(max-width:700px){.public-growth{grid-template-columns:1fr}.public-follow,.public-related,.public-build{grid-column:auto}.public-follow form{display:grid}.public-related>div{grid-template-columns:1fr}.article-header{width:calc(100% - 40px);height:74px}.article-brand small{display:none}.article-back{max-width:140px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}body.public-dossier>main{padding:45px 18px 75px!important}body.public-dossier>main>div{padding:30px 23px!important;border-radius:16px!important}}
`

func firstReadingText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
