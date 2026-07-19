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
	var topics strings.Builder
	for _, newsletter := range newsletters {
		fmt.Fprintf(
			&topics,
			`<a class="topic" href="/topics/%s"><strong>%s</strong><span>%d Dossiers</span></a>`,
			url.PathEscape(newsletter.PublicSlug),
			html.EscapeString(newsletter.Name),
			newsletter.GeneratedCount,
		)
	}
	if len(newsletters) == 0 {
		topics.WriteString("<p>No published learning streams yet.</p>")
	}
	body := `<header class="hero"><p>Personal learning archive</p><h1>` +
		html.EscapeString(site.DisplayName) + `</h1><div>` +
		html.EscapeString(site.Description) + `</div></header>
<section><h2>Topics</h2><div class="topics">` + topics.String() + `</div></section>
<section><h2>Latest Dossiers</h2><div class="issues">` +
		renderIssueCards(issues) + `</div></section>`
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
	body := `<header class="hero compact"><a href="/">← ` +
		html.EscapeString(site.DisplayName) + `</a><p>Learning stream</p><h1>` +
		html.EscapeString(selected.Name) + `</h1><div>` +
		html.EscapeString(selected.Topic) + `</div></header>
<section><h2>Archive</h2><div class="issues">` + renderIssueCards(issues) + `</div></section>`
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
		`<style>` + readingCSS + `</style></head><body><main>` + body +
		`</main><footer>Learnloom · Intelligence, made durable.</footer></body></html>`
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
		_, _ = response.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>Not found · Learnloom</title></head><body><main><h1>Not found</h1><p>This learning page is unavailable.</p></main></body></html>`))
	}
}

func renderIssueCards(issues []store.PublicIssue) string {
	if len(issues) == 0 {
		return "<p>No published Dossiers yet.</p>"
	}
	var result strings.Builder
	for _, issue := range issues {
		href := "/d/" + url.PathEscape(issue.PublicID) + "/" + url.PathEscape(issue.PublicSlug)
		fmt.Fprintf(
			&result,
			`<article><p>%s</p><h3><a href="%s">%s</a></h3><time datetime="%s">%s</time></article>`,
			html.EscapeString(issue.NewsletterName),
			href,
			html.EscapeString(issue.Title),
			issue.CompletedAt.Format(time.RFC3339),
			issue.CompletedAt.Format("2 January 2006"),
		)
	}
	return result.String()
}

const readingCSS = `
:root{color-scheme:light;--ink:#17211b;--muted:#68736c;--paper:#f7f6f0;--card:#fff;--accent:#176b4d}
*{box-sizing:border-box}body{margin:0;background:var(--paper);color:var(--ink);font:16px/1.65 ui-sans-serif,system-ui,sans-serif}
main,footer{max-width:980px;margin:auto;padding:40px 24px}.hero{padding:72px 0 48px}.hero.compact{padding-bottom:24px}
.hero p{color:var(--accent);font-weight:700;text-transform:uppercase;letter-spacing:.12em;font-size:12px}.hero h1{font:700 clamp(42px,8vw,76px)/1.03 Georgia,serif;margin:12px 0}
h2{font:700 30px/1.2 Georgia,serif;margin-top:48px}.topics,.issues{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:16px}
.topic,article{display:block;background:var(--card);border:1px solid #deddd5;border-radius:14px;padding:22px;color:inherit;text-decoration:none}
.topic span,article p,time{display:block;color:var(--muted);font-size:13px}.topic strong,article h3{font-size:19px;margin:4px 0}
article a{color:var(--ink);text-decoration:none}article a:hover,.topic:hover strong{color:var(--accent)}footer{color:var(--muted);font-size:13px}
`

func firstReadingText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
