package dossier

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConfiguredProviderEvaluationCorpus(t *testing.T) {
	if os.Getenv("MODEL_EVAL_ENABLED") != "true" {
		t.Skip("MODEL_EVAL_ENABLED is not true")
	}
	var cases []struct {
		ID            string   `json:"id"`
		Evidence      string   `json:"evidence"`
		Question      string   `json:"question"`
		RequiredTerms []string `json:"requiredTerms"`
		SourceID      string   `json:"sourceId"`
	}
	raw, err := os.ReadFile("testdata/model_eval_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL:          os.Getenv("MODEL_BASE_URL"),
		APIKey:           os.Getenv("MODEL_API_KEY"),
		Model:            os.Getenv("MODEL_NAME"),
		StructuredOutput: true,
		MaxTokens:        300, Timeout: 2 * time.Minute, Retries: 1,
		MaxConcurrency:                 1,
		InputMicroUSDPerMillionTokens:  1,
		OutputMicroUSDPerMillionTokens: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	passed := 0
	for _, testCase := range cases {
		result, err := model.Complete(context.Background(), CompletionRequest{
			Stage:      "provider-evaluation",
			Structured: true,
			Instruction: "Return JSON with an answer string and sourceIds array. " +
				"Use only the supplied evidence, distinguish observation from causation, " +
				"and state important limits.",
			Input: testCase.Evidence + "\n\nQuestion: " + testCase.Question,
		})
		if err != nil {
			t.Errorf("%s: %v", testCase.ID, err)
			continue
		}
		var answer struct {
			Answer    string   `json:"answer"`
			SourceIDs []string `json:"sourceIds"`
		}
		if err := json.Unmarshal([]byte(result.Output), &answer); err != nil {
			t.Errorf("%s: invalid JSON: %v", testCase.ID, err)
			continue
		}
		text := strings.ToLower(answer.Answer)
		termsPresent := true
		for _, term := range testCase.RequiredTerms {
			termsPresent = termsPresent && strings.Contains(text, strings.ToLower(term))
		}
		sourcePresent := false
		for _, sourceID := range answer.SourceIDs {
			sourcePresent = sourcePresent || sourceID == testCase.SourceID
		}
		if termsPresent && sourcePresent {
			passed++
		} else {
			t.Errorf("%s: answer did not satisfy evidence contract", testCase.ID)
		}
	}
	if score := float64(passed) / float64(len(cases)); score < 0.8 {
		t.Fatalf("provider evaluation score %.2f is below 0.80", score)
	}
}
