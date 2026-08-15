package httpapp

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppCSPAllowsClerkRuntimeWorkers(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{
		ClerkFrontendOrigin: "https://clerk.learnloom.blog",
	}}
	response := httptest.NewRecorder()

	server.applyAppCSP(response)

	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "worker-src 'self' blob:") {
		t.Fatalf("CSP does not allow Clerk runtime workers: %q", policy)
	}
}

func TestAppCSPAllowsFontshareSatoshiAssets(t *testing.T) {
	t.Parallel()
	server := &Server{}
	response := httptest.NewRecorder()

	server.applyAppCSP(response)

	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "style-src 'self' 'unsafe-inline' https://api.fontshare.com") {
		t.Fatalf("CSP does not allow Fontshare stylesheets: %q", policy)
	}
	if !strings.Contains(policy, "font-src 'self' https://cdn.fontshare.com") {
		t.Fatalf("CSP does not allow Fontshare fonts: %q", policy)
	}
}

func TestPrivateCacheableJSONSupportsConditionalRequests(t *testing.T) {
	t.Parallel()
	firstRequest := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	firstResponse := httptest.NewRecorder()
	writePrivateCacheableJSON(
		firstResponse,
		firstRequest,
		http.StatusOK,
		map[string]string{"status": "ready"},
		"private, max-age=0, must-revalidate",
	)
	etag := firstResponse.Header().Get("ETag")
	if firstResponse.Code != http.StatusOK || etag == "" {
		t.Fatalf("status=%d etag=%q", firstResponse.Code, etag)
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	secondRequest.Header.Set("If-None-Match", etag)
	secondResponse := httptest.NewRecorder()
	writePrivateCacheableJSON(
		secondResponse,
		secondRequest,
		http.StatusOK,
		map[string]string{"status": "ready"},
		"private, max-age=0, must-revalidate",
	)
	if secondResponse.Code != http.StatusNotModified || secondResponse.Body.Len() != 0 {
		t.Fatalf("status=%d body=%q", secondResponse.Code, secondResponse.Body.String())
	}
}

func TestWriteProblemRejectsUndeclaredRuntimeCodes(t *testing.T) {
	t.Parallel()
	response := httptest.NewRecorder()
	writeProblem(response, http.StatusTeapot, "surprise_code", "internal detail")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, `"code":"internal_error"`) ||
		strings.Contains(body, "internal detail") {
		t.Fatalf("unsafe problem response: %s", body)
	}
}

func TestProductionResponseIdentifiesReleaseAndEnablesHSTS(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{
		Environment: "production", ReleaseVersion: "git-abc123",
	}, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "https://app.learnloom.blog/healthz", nil)

	server.ServeHTTP(response, request)

	if got := response.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("HSTS=%q", got)
	}
	if got := response.Header().Get("X-Learnloom-Release"); got != "git-abc123" {
		t.Fatalf("release header=%q", got)
	}
}
