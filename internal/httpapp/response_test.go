package httpapp

import (
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
