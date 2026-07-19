package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResendUsesDeterministicIdempotency(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Idempotency-Key"); got != "learnloom/issue-1/gen-1" {
			t.Errorf("unexpected idempotency key %q", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"id":"email-1"}`))
	}))
	defer server.Close()
	resend, err := NewResend(Config{
		APIKey: "secret", From: "Learnloom <hello@example.com>", Endpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := resend.Deliver(context.Background(), Message{
		IssueID: "issue-1", GenerationID: "gen-1", To: "learner@example.com",
		Subject: "Lesson\nInjected", HTML: "<p>Lesson</p>", Text: "Lesson",
	})
	if err != nil || id != "email-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResendRedactsErrors(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte("bad secret"))
	}))
	defer server.Close()
	resend, _ := NewResend(Config{
		APIKey: "secret", From: "Learnloom <hello@example.com>", Endpoint: server.URL,
	})
	_, err := resend.Deliver(context.Background(), Message{
		IssueID: "i", GenerationID: "g", To: "learner@example.com",
	})
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("secret leaked: %v", err)
	}
}
