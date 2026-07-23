package httpapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestDecodeNewsletterInputSupportsTopicOnlyDefaults(t *testing.T) {
	server := &Server{cfg: Config{MaxRequestBodyBytes: 1 << 20}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/newsletters",
		strings.NewReader(`{
			"topic":"LLM inference",
			"sourceMode":"discovered",
			"timeZone":"Asia/Kolkata"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	input, ok := server.decodeNewsletterInput(response, request)
	if !ok {
		t.Fatalf("decode failed: status=%d body=%s", response.Code, response.Body.String())
	}
	if input.SourceMode != domain.SourceModeDiscovered ||
		input.ScheduleHour != 8 || input.ScheduleMinute != 0 ||
		!input.Active || input.SiteVisible || len(input.Sources) != 0 {
		t.Fatalf("input=%#v", input)
	}
}

func TestClerkSessionTokenSupportsAPIsAndPageNavigations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		authorization string
		cookie        string
		want          string
	}{
		{
			name:          "bearer token for API request",
			authorization: "Bearer api-token",
			cookie:        "cookie-token",
			want:          "api-token",
		},
		{
			name:   "session cookie for page navigation",
			cookie: "cookie-token",
			want:   "cookie-token",
		},
		{
			name:          "empty bearer falls back to session cookie",
			authorization: "Bearer ",
			cookie:        "cookie-token",
			want:          "cookie-token",
		},
		{
			name:          "unrelated authorization scheme is ignored",
			authorization: "Basic credentials",
			want:          "",
		},
		{name: "anonymous request", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/issues/issue-1", nil)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "__session", Value: test.cookie})
			}
			if got := clerkSessionToken(request); got != test.want {
				t.Fatalf("clerkSessionToken() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDecodeNewsletterInputKeepsBackwardCompatibleProvidedMode(t *testing.T) {
	server := &Server{cfg: Config{MaxRequestBodyBytes: 1 << 20}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/newsletters",
		strings.NewReader(`{
			"topic":"LLM inference",
			"timeZone":"UTC",
			"scheduleTime":"09:30",
			"active":false,
			"sources":[{"name":"Docs","url":"https://example.com/docs","limit":8}]
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	input, ok := server.decodeNewsletterInput(response, request)
	if !ok {
		t.Fatalf("decode failed: status=%d body=%s", response.Code, response.Body.String())
	}
	if input.SourceMode != domain.SourceModeProvided ||
		input.ScheduleHour != 9 || input.ScheduleMinute != 30 || input.Active {
		t.Fatalf("input=%#v", input)
	}
}
