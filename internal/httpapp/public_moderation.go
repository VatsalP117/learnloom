package httpapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/store"
)

func (s *Server) handlePublicContentReport(
	response http.ResponseWriter,
	request *http.Request,
	host RequestHost,
	publicID string,
) {
	expectedOrigin := "https://" + host.Hostname
	if origin := strings.TrimSuffix(request.Header.Get("Origin"), "/"); origin != expectedOrigin {
		writeProblem(response, http.StatusForbidden, "origin_rejected", "The report origin is invalid.")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, s.cfg.MaxRequestBodyBytes)
	if err := request.ParseForm(); err != nil {
		writeProblem(response, http.StatusBadRequest, "invalid_report", "The report could not be read.")
		return
	}
	if request.PostForm.Get("website") != "" {
		writeProblem(response, http.StatusBadRequest, "invalid_report", "The report could not be submitted.")
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
	fingerprint := s.publicReporterFingerprint(request)
	allowed, err := s.store.AllowRequest(
		request.Context(),
		fingerprint,
		"public-content-report",
		time.Hour,
		5,
		time.Now().UTC(),
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if !allowed {
		writeProblem(response, http.StatusTooManyRequests, "rate_limited", "Please wait before sending another report.")
		return
	}
	if _, err := s.store.CreatePublicContentReport(
		request.Context(),
		site.Username,
		publicID,
		request.PostForm.Get("category"),
		request.PostForm.Get("details"),
		fingerprint,
		time.Now().UTC(),
	); err != nil {
		writeStoreError(response, err)
		return
	}
	back := "/d/" + url.PathEscape(issue.PublicID) + "/" + url.PathEscape(issue.PublicSlug)
	body := `<main id="main-content"><section class="report-thanks">` +
		`<p class="eyebrow">Report received</p><h1>Thank you for helping keep this accurate.</h1>` +
		`<p>The publisher can now review your report without receiving your identity or IP address.</p>` +
		`<a class="text-link" href="` + html.EscapeString(back) + `">Return to the Dossier <span>→</span></a>` +
		`</section></main>`
	s.sendReadingPage(
		response,
		request,
		"Report received",
		"Your report has been sent to the publisher.",
		expectedOrigin+back,
		body,
		false,
	)
	response.Header().Set("Cache-Control", "private, no-store")
}

func (s *Server) publicReporterFingerprint(request *http.Request) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.CSRFSecret))
	_, _ = fmt.Fprintf(mac, "%s\x00%s", clientAddress(request), request.UserAgent())
	return hex.EncodeToString(mac.Sum(nil))
}

func decoratePublicModeration(
	document string,
	issue store.PublicIssue,
	corrections []store.PublicCorrection,
) string {
	var panel strings.Builder
	panel.WriteString(`<aside class="public-trust" aria-label="Corrections and reporting">`)
	if len(corrections) > 0 {
		panel.WriteString(`<section class="public-corrections"><p class="trust-label">Publisher corrections</p>`)
		for _, correction := range corrections {
			panel.WriteString(`<article><p>`)
			panel.WriteString(html.EscapeString(correction.Body))
			panel.WriteString(`</p><time datetime="`)
			panel.WriteString(correction.CreatedAt.Format(time.RFC3339))
			panel.WriteString(`">`)
			panel.WriteString(correction.CreatedAt.Format("2 Jan 2006"))
			panel.WriteString(`</time></article>`)
		}
		panel.WriteString(`</section>`)
	}
	panel.WriteString(`<details class="public-report"><summary>Report a concern</summary>` +
		`<form method="post" action="/report/` + url.PathEscape(issue.PublicID) + `">` +
		`<label>What needs attention?<select name="category" required>` +
		`<option value="inaccurate">Factual accuracy</option>` +
		`<option value="citation">Citation or source</option>` +
		`<option value="harmful">Potentially harmful content</option>` +
		`<option value="other">Something else</option></select></label>` +
		`<label>Details <span>(optional)</span><textarea name="details" maxlength="2000" rows="4"></textarea></label>` +
		`<label class="report-honeypot" aria-hidden="true">Website<input name="website" tabindex="-1" autocomplete="off"></label>` +
		`<button type="submit">Send private report</button>` +
		`<small>Your identity and IP address are not shared with the publisher.</small>` +
		`</form></details></aside>`)
	style := `<style>.public-trust{max-width:820px;margin:0 auto 80px;padding:0 24px;color:#344039;font:14px/1.6 "Avenir Next",Avenir,"Segoe UI",sans-serif}.public-corrections,.public-report{padding:22px;border:1px solid rgba(23,33,27,.13);border-radius:14px;background:#fffef9}.public-corrections{margin-bottom:14px}.trust-label{margin:0 0 12px;color:#496b4c;font-size:10px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}.public-corrections article+article{margin-top:15px;padding-top:15px;border-top:1px solid rgba(23,33,27,.1)}.public-corrections article p{margin:0;white-space:pre-wrap}.public-corrections time{color:#69736c;font-size:11px}.public-report summary{cursor:pointer;font-weight:700}.public-report form{display:grid;gap:14px;margin-top:18px}.public-report label{display:grid;gap:5px;font-size:12px;font-weight:700}.public-report label span,.public-report small{color:#69736c;font-weight:400}.public-report select,.public-report textarea{width:100%;padding:10px;border:1px solid rgba(23,33,27,.2);border-radius:8px;background:#fff;font:inherit}.public-report button{width:max-content;padding:10px 15px;border:0;border-radius:8px;background:#1d2c22;color:#fff;font:inherit;font-weight:700}.report-honeypot{position:absolute!important;left:-10000px!important}.report-thanks{max-width:720px;margin:100px auto;padding:50px 32px}.report-thanks h1{font:400 48px/1.05 "Iowan Old Style",Georgia,serif}</style>`
	document = strings.Replace(document, "</head>", style+"</head>", 1)
	return strings.Replace(document, "</body>", panel.String()+"</body>", 1)
}
