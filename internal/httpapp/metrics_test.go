package httpapp

import "testing"

func TestMetricRouteBoundsDynamicPaths(t *testing.T) {
	tests := map[string]string{
		"/api/usernames/vatsal":            "/api/usernames/:username",
		"/api/reviews/review-1/assess":     "/api/reviews/:review_id/assess",
		"/api/newsletters/newsletter-1":    "/api/newsletters/:newsletter_id",
		"/api/newsletters/newsletter-1/go": "/api/newsletters/:newsletter_id/:action",
		"/api/issues/issue-1/notes":        "/api/issues/:issue_id/:action",
		"/issues/issue-1":                  "/issues/:issue_id",
	}
	host := RequestHost{Kind: HostApp}
	for input, expected := range tests {
		if actual := metricRoute(host, input); actual != expected {
			t.Errorf("metricRoute(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestMetricRouteBoundsPersonalSites(t *testing.T) {
	host := RequestHost{Kind: HostSite, Username: "vatsal"}
	for input, expected := range map[string]string{
		"/topics/observability": "/topics/:topic_slug",
		"/d/public-id/slug":     "/d/:dossier_id/:slug",
		"/report/public-id":     "/report/:dossier_id",
	} {
		if actual := metricRoute(host, input); actual != expected {
			t.Errorf("metricRoute(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestMetricStatus(t *testing.T) {
	for status, expected := range map[int]string{200: "2xx", 404: "4xx", 503: "5xx", 0: "unknown"} {
		if actual := metricStatus(status); actual != expected {
			t.Errorf("metricStatus(%d)=%q, want %q", status, actual, expected)
		}
	}
}
