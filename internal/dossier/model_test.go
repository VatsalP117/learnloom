package dossier

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/failure"
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
		_, _ = response.Write([]byte(
			`{"choices":[{"message":{"role":"assistant","content":"result"}}],` +
				`"usage":{"prompt_tokens":100,"completion_tokens":50}}`,
		))
	}))
	defer server.Close()

	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
		Retries: 1, MaxTokens: 100, MaxConcurrency: 1,
		InputMicroUSDPerMillionTokens:  1_000_000,
		OutputMicroUSDPerMillionTokens: 2_000_000,
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
	if output.Output != "result" || attempts.Load() != 2 ||
		output.Usage.Retries != 1 || output.Usage.InputTokens != 100 ||
		output.Usage.OutputTokens != 50 ||
		output.Usage.EstimatedCostMicroUSD != 200 {
		t.Fatalf("unexpected output=%#v attempts=%d", output, attempts.Load())
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

func TestOpenAIModelRequestsConfiguredStructuredOutput(t *testing.T) {
	t.Parallel()
	var responseFormat string
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body struct {
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		responseFormat = body.ResponseFormat.Type
		_, _ = response.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
		StructuredOutput: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	model.client = server.Client()
	if _, err := model.Complete(context.Background(), CompletionRequest{
		Stage: "editor", Structured: true,
	}); err != nil {
		t.Fatal(err)
	}
	if responseFormat != "json_object" {
		t.Fatalf("response_format=%q", responseFormat)
	}
}

func TestOpenAIModelClassifiesTokenTruncation(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"partial"},"finish_reason":"length"}]}`))
	}))
	defer server.Close()
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	model.client = server.Client()
	_, err = model.Complete(context.Background(), CompletionRequest{Stage: "editor"})
	if !errors.Is(err, ErrOutputTruncated) {
		t.Fatalf("err=%v", err)
	}
}

func TestStageExecutionFailureDoesNotRetryPermanentProviderRequest(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	model.client = server.Client()
	_, err = model.Complete(context.Background(), CompletionRequest{Stage: "teacher"})
	detail := failure.Describe(stageExecutionFailure("teacher", err))
	if detail.Code != "model_request_rejected" || detail.Category != failure.CategoryProvider ||
		detail.Stage != "teacher" || detail.Retryable {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestStageExecutionFailureRetriesTransientProviderRequest(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL: server.URL, APIKey: "secret-value", Model: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	model.client = server.Client()
	_, err = model.Complete(context.Background(), CompletionRequest{Stage: "skeptic"})
	detail := failure.Describe(stageExecutionFailure("skeptic", err))
	if detail.Code != "model_provider_unavailable" || detail.Category != failure.CategoryProvider ||
		detail.Stage != "skeptic" || !detail.Retryable || detail.PublicMessage != failure.PublicDelayed {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestStageExecutionFailureClassifiesEmptyOutputAsContentQuality(t *testing.T) {
	t.Parallel()
	detail := failure.Describe(stageExecutionFailure("researcher", ErrEmptyOutput))
	if detail.Code != "model_output_empty" || detail.Category != failure.CategoryContentQuality ||
		detail.Stage != "researcher" || !detail.Retryable {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestStageExecutionFailureClassifiesDeadlineAsRetryableInterruption(t *testing.T) {
	t.Parallel()
	detail := failure.Describe(stageExecutionFailure("teacher", context.DeadlineExceeded))
	if detail.Code != "generation_interrupted" ||
		detail.Category != failure.CategoryInfrastructure ||
		detail.Stage != "teacher" || !detail.Retryable ||
		detail.PublicMessage != failure.PublicDelayed {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}
