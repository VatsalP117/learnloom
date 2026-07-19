package dossier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAIModelCompletesAndRetries(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret-value" {
			t.Errorf("missing authorization")
		}
		if attempts.Add(1) == 1 {
			response.WriteHeader(http.StatusTooManyRequests)
			_, _ = response.Write([]byte(`{"error":{"message":"slow down"}}`))
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"result"}}]}`))
	}))
	defer server.Close()

	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
		Retries: 1, MaxTokens: 100, MaxConcurrency: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	model.client = server.Client()
	model.sleep = func(context.Context, time.Duration) error { return nil }
	output, err := model.Complete(context.Background(), CompletionRequest{
		Stage: "teacher", Instruction: "teach", Input: "sources",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output != "result" || attempts.Load() != 2 {
		t.Fatalf("unexpected output=%q attempts=%d", output, attempts.Load())
	}
}

func TestOpenAIModelRedactsCredential(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte(`{"error":{"message":"bad secret-value"}}`))
	}))
	defer server.Close()
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	model.client = server.Client()
	_, err = model.Complete(context.Background(), CompletionRequest{Stage: "test"})
	if err == nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("credential was not redacted: %v", err)
	}
}
