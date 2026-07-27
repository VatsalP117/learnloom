package httpapp

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
)

func (s *Server) handleReading(
	response http.ResponseWriter,
	request *http.Request,
	host RequestHost,
) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		methodNotAllowed(response, http.MethodGet, http.MethodHead)
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
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.Header().Set("Cache-Control", "public, max-age=300")
		if request.Method != http.MethodHead {
			fmt.Fprintf(response, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", origin)
		}
		return
	case "/sitemap.xml":
		s.renderSitemap(response, request, site, origin)
		return
	case "/":
		s.renderPublicHome(response, request, site, origin)
		return
	}
	route := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
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
	dossierCount := 0
	var topics strings.Builder
	for index, newsletter := range newsletters {
		dossierCount += newsletter.GeneratedCount
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
		`<header class="hero home-hero">` +
		`<div class="hero-copy"><p class="eyebrow"><span></span>Personal learning garden</p>` +
		`<h1>` + html.EscapeString(site.DisplayName) + `</h1>` +
		`<p class="hero-description">` +
		html.EscapeString(firstReadingText(
			site.Description,
			"A quiet place for ideas worth returning to.",
		)) + `</p>` +
		`<a class="text-link" href="#latest">Explore the latest Dossiers <span aria-hidden="true">↓</span></a></div>` +
		`<aside class="garden-note" aria-label="Archive summary">` +
		`<div class="leaf-mark" aria-hidden="true"><i></i></div>` +
		`<p>A growing record of careful reading, source-grounded lessons, and ideas made durable.</p>` +
		`<dl><div><dt>Topics</dt><dd>` + fmt.Sprint(len(newsletters)) +
		`</dd></div><div><dt>Dossiers</dt><dd>` + fmt.Sprint(dossierCount) +
		`</dd></div></dl></aside></header>` +
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
	artifactValue, err := s.artifacts.Get(request.Context(), issue.ArtifactKey)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	canonical := origin + canonicalPath
	document := strings.Replace(
		artifactValue.HTML,
		"</head>",
		`<link rel="canonical" href="`+html.EscapeString(canonical)+`">`+
			`<meta property="og:type" content="article">`+
			`<meta property="og:url" content="`+html.EscapeString(canonical)+`">`+
			"</head>",
		1,
	)
	document = decoratePublicDossier(document, site, issue)
	s.applyReadingHeaders(response)
	response.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func (s *Server) renderSitemap(
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
	issues, err := s.store.ListPublicIssues(request.Context(), site.Username, "", 200)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	locations := []string{origin}
	for _, newsletter := range newsletters {
		locations = append(locations, origin+"/topics/"+url.PathEscape(newsletter.PublicSlug))
	}
	for _, issue := range issues {
		locations = append(
			locations,
			origin+"/d/"+url.PathEscape(issue.PublicID)+"/"+url.PathEscape(issue.PublicSlug),
		)
	}
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, location := range locations {
		body.WriteString("<url><loc>" + html.EscapeString(location) + "</loc></url>")
	}
	body.WriteString("</urlset>")
	response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(body.String()))
	}
}

func (s *Server) sendReadingPage(
	response http.ResponseWriter,
	request *http.Request,
	title, description, canonical, body string,
) {
	document := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + html.EscapeString(title) + ` · Learnloom</title>` +
		`<meta name="description" content="` + html.EscapeString(description) + `">` +
		`<link rel="canonical" href="` + html.EscapeString(canonical) + `">` +
		`<style>` + readingCSS + `</style></head><body>` + body + `</body></html>`
	s.applyReadingHeaders(response)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func (s *Server) applyReadingHeaders(response http.ResponseWriter) {
	response.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; img-src https: data:; "+
			"font-src 'self'; base-uri 'none'; form-action 'none'; "+
			"frame-ancestors 'none'; object-src 'none'",
	)
	response.Header().Set("X-Robots-Tag", "index, follow")
}

func (s *Server) readingNotFound(
	response http.ResponseWriter,
	request *http.Request,
) {
	s.applyReadingHeaders(response)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=30")
	response.WriteHeader(http.StatusNotFound)
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8">` +
			`<meta name="viewport" content="width=device-width,initial-scale=1">` +
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
		`<style>`+readingArticleCSS+`</style></head>`,
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
.home-hero{min-height:360px;grid-template-columns:minmax(0,1.6fr) minmax(220px,.6fr);gap:clamp(35px,6vw,90px);padding:54px 0 46px}.home-hero .hero-copy{max-width:700px}.home-hero h1{font-size:clamp(46px,6vw,78px);line-height:.94}.home-hero .hero-description{max-width:560px;margin-top:16px;font-size:clamp(14px,1.25vw,17px);line-height:1.55}.home-hero .text-link{margin-top:20px}.home-hero .garden-note{max-width:300px;margin-left:auto;padding:22px}.home-hero .leaf-mark{width:38px;height:38px}.home-hero .garden-note p{margin:17px 0 20px;font-size:16px;line-height:1.4}.home-hero .garden-note dd{font-size:25px}.home-hero + .reading-section{padding-top:64px}.home-hero ~ .reading-section .section-heading h2{font-size:clamp(36px,4vw,54px)}.home-hero ~ .latest-section{padding-bottom:92px}.home-hero ~ .topics-section{padding-top:78px}
@media(max-width:860px){.site-header,main,.site-footer{width:min(100% - 40px,680px)}.site-header{height:76px}.site-header nav{gap:18px}.site-header nav a:last-child{display:none}.hero{min-height:auto;grid-template-columns:1fr;gap:45px;padding:75px 0}.hero h1{font-size:clamp(52px,15vw,80px)}.garden-note,.stream-note{max-width:440px}.reading-section{padding:78px 0}.section-heading{display:grid;gap:20px}.issues{grid-template-columns:1fr 1fr}.topic-card{grid-template-columns:36px minmax(0,1fr) 70px 20px;gap:12px}.topic-copy>span{white-space:normal}.latest-section{padding-bottom:90px}}
@media(max-width:580px){.site-header nav{gap:14px}.site-header nav a{font-size:10px}.site-brand small{display:none}.hero{padding:58px 0 68px}.hero-description{font-size:15px}.garden-note{padding:25px}.reading-section{padding:66px 0}.section-heading h2{font-size:44px}.topics{border:0}.topic-card{grid-template-columns:30px minmax(0,1fr) 18px;min-height:105px}.topic-count{display:none}.issues{grid-template-columns:1fr}.issue-link{min-height:250px}.issue-card h3{margin:34px 0 25px;font-size:28px}.site-footer{align-items:flex-start;display:grid}.footer-meta{justify-items:start}.latest-section{padding-bottom:75px}}
@media(max-width:860px){.home-hero{gap:28px;padding:42px 0 36px}.home-hero .garden-note{max-width:440px;margin-left:0}}
@media(max-width:580px){.home-hero{padding:36px 0 46px}.home-hero h1{font-size:clamp(45px,12vw,64px)}.home-hero .garden-note{padding:20px}.home-hero .garden-note p{font-size:15px}}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}.site-header nav a:after,.topic-card,.arrow,.issue-card,.text-link span{transition:none}}
`

const readingArticleCSS = `
:root{--article-ink:#17211b;--article-muted:#5f6b63;--article-paper:#f7f6f0;--article-green:#496b4c;--article-deep:#1d2c22;--article-line:rgba(23,33,27,.12);--article-serif:"Iowan Old Style","Palatino Linotype",Palatino,Georgia,serif;--article-sans:"Avenir Next",Avenir,"Segoe UI",ui-sans-serif,sans-serif}
body.public-dossier{min-width:320px!important;margin:0!important;background:radial-gradient(circle at 16% 0,rgba(181,210,232,.45),transparent 31rem),linear-gradient(180deg,#eef3ef 0,#f7f6f0 35rem)!important;color:var(--article-ink)!important;font-family:var(--article-sans)!important;-webkit-font-smoothing:antialiased}
.public-dossier *{box-sizing:border-box}.article-header{width:min(1120px,calc(100% - 64px));height:88px;display:flex;align-items:center;justify-content:space-between;margin:auto;border-bottom:1px solid var(--article-line)}.article-brand{display:inline-flex;align-items:center;gap:10px;color:inherit;text-decoration:none}.article-brand>span:last-child{display:grid}.article-brand strong{font-size:14px;letter-spacing:-.03em}.article-brand small{color:var(--article-muted);font-size:8px;font-weight:700;letter-spacing:.12em;text-transform:uppercase}.public-dossier .brand-mark{width:32px;height:32px;display:inline-grid;place-items:center;border-radius:9px;background:var(--article-deep);color:#eef6ec;font-size:14px}.article-back{color:#314538;font-size:11px;font-weight:700;text-decoration:none}
body.public-dossier>main{max-width:820px!important;margin:0 auto!important;padding:72px 24px 110px!important}body.public-dossier>main>div{padding:clamp(30px,6vw,64px)!important;border:1px solid rgba(23,33,27,.1)!important;border-radius:20px!important;background:rgba(255,255,251,.78)!important;box-shadow:0 24px 70px rgba(35,64,43,.08)!important;backdrop-filter:blur(16px)}
body.public-dossier>main>div>p:first-child{color:var(--article-green)!important;font-size:10px!important;letter-spacing:.14em!important}body.public-dossier>main>div>h1{margin-bottom:36px!important;font-family:var(--article-serif)!important;font-size:clamp(38px,6vw,60px)!important;font-weight:400!important;line-height:1.02!important;letter-spacing:-.045em!important}body.public-dossier h2,body.public-dossier h3,body.public-dossier h4{font-family:var(--article-serif)!important;font-weight:400!important;letter-spacing:-.025em!important}body.public-dossier h2{font-size:29px!important}body.public-dossier p,body.public-dossier li{color:#344039;line-height:1.78!important}body.public-dossier hr{border-color:var(--article-line)!important;margin:42px 0!important}body.public-dossier a{color:var(--article-green)!important}body.public-dossier code{background:#eef0e8!important;color:#294232!important}body.public-dossier details{padding:18px 20px;border:1px solid var(--article-line);border-radius:12px;background:rgba(247,246,240,.7)}
@media(max-width:580px){.article-header{width:calc(100% - 40px);height:74px}.article-brand small{display:none}.article-back{max-width:140px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}body.public-dossier>main{padding:45px 18px 75px!important}body.public-dossier>main>div{padding:30px 23px!important;border-radius:16px!important}}
`

func firstReadingText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
