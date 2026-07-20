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
