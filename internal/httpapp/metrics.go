package httpapp

import (
	"net/http"
	"strings"
)

func metricMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return method
	default:
		return "OTHER"
	}
}

func metricStatus(status int) string {
	if status < 100 || status > 599 {
		return "unknown"
	}
	return string(rune('0'+status/100)) + "xx"
}

func metricRoute(host RequestHost, requestPath string) string {
	switch host.Kind {
	case HostWWW:
		return "www_redirect"
	case HostApex:
		return apexMetricRoute(requestPath)
	case HostApp:
		return appMetricRoute(requestPath)
	case HostSite:
		return siteMetricRoute(requestPath)
	default:
		return "invalid_host"
	}
}

func apexMetricRoute(requestPath string) string {
	switch {
	case strings.HasPrefix(requestPath, "/assets/"):
		return "/assets/:asset"
	case isFaviconPath(requestPath):
		return "/favicon"
	case requestPath == socialImagePath:
		return socialImagePath
	case requestPath == "/", requestPath == "/robots.txt",
		requestPath == "/sitemap.xml", requestPath == "/examples",
		requestPath == "/marketing", requestPath == "/privacy",
		requestPath == "/terms":
		return requestPath
	case isApexPage(requestPath):
		return requestPath
	default:
		return "apex_page"
	}
}

func appMetricRoute(requestPath string) string {
	switch {
	case strings.HasPrefix(requestPath, "/assets/"):
		return "/assets/:asset"
	case isFaviconPath(requestPath):
		return "/favicon"
	case strings.HasPrefix(requestPath, "/sign-in"):
		return "/sign-in/*"
	case strings.HasPrefix(requestPath, "/sign-up"):
		return "/sign-up/*"
	}
	segments := splitMetricPath(requestPath)
	if len(segments) == 0 {
		return "/"
	}
	if segments[0] == "issues" && len(segments) == 2 {
		return "/issues/:issue_id"
	}
	if segments[0] != "api" {
		switch requestPath {
		case "/healthz", "/readyz", "/webhooks/clerk":
			return requestPath
		default:
			return "app_page"
		}
	}
	if len(segments) == 3 && segments[1] == "usernames" {
		return "/api/usernames/:username"
	}
	if len(segments) == 4 && segments[1] == "reviews" && segments[3] == "assess" {
		return "/api/reviews/:review_id/assess"
	}
	if len(segments) >= 3 && segments[1] == "newsletters" {
		if len(segments) == 3 {
			return "/api/newsletters/:newsletter_id"
		}
		if len(segments) == 4 {
			return "/api/newsletters/:newsletter_id/:action"
		}
	}
	if len(segments) >= 3 && segments[1] == "issues" {
		if len(segments) == 3 {
			return "/api/issues/:issue_id"
		}
		if len(segments) == 4 {
			return "/api/issues/:issue_id/:action"
		}
	}
	if len(segments) == 3 && segments[1] == "notes" {
		return "/api/notes/:note_id"
	}
	if len(segments) == 3 && segments[1] == "corrections" {
		return "/api/corrections/:correction_id"
	}
	if len(segments) == 4 && segments[1] == "reports" && segments[3] == "resolve" {
		return "/api/reports/:report_id/resolve"
	}
	switch requestPath {
	case "/api/me", "/api/me/notifications", "/api/me/site/claim",
		"/api/me/site/settings", "/api/newsletters", "/api/sources/validate",
		"/api/workspace", "/api/library", "/api/performance/vitals", "/api/issues":
		return requestPath
	default:
		return "/api/other"
	}
}

func siteMetricRoute(requestPath string) string {
	segments := splitMetricPath(requestPath)
	switch {
	case isFaviconPath(requestPath):
		return "/favicon"
	case requestPath == "/", requestPath == "/robots.txt", requestPath == "/sitemap.xml":
		return requestPath
	case len(segments) == 2 && segments[0] == "topics":
		return "/topics/:topic_slug"
	case len(segments) >= 2 && segments[0] == "d":
		return "/d/:dossier_id/:slug"
	case len(segments) == 2 && segments[0] == "report":
		return "/report/:dossier_id"
	default:
		return "personal_site_page"
	}
}

func splitMetricPath(value string) []string {
	trimmed := strings.Trim(value, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}
