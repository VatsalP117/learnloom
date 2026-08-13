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

func TestConfiguredProviderLessonQualityCorpus(t *testing.T) {
	if os.Getenv("MODEL_EVAL_ENABLED") != "true" {
		t.Skip("MODEL_EVAL_ENABLED is not true")
	}
	raw, err := os.ReadFile("testdata/lesson-evaluation-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus LessonEvaluationCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLessonEvaluationCorpus(corpus); err != nil {
		t.Fatal(err)
	}
	model, err := NewOpenAIModel(ModelConfig{
		BaseURL:          os.Getenv("MODEL_BASE_URL"),
		APIKey:           os.Getenv("MODEL_API_KEY"),
		Model:            os.Getenv("MODEL_NAME"),
		StructuredOutput: true,
		MaxTokens:        350, Timeout: 2 * time.Minute, Retries: 1,
		MaxConcurrency:                 1,
		InputMicroUSDPerMillionTokens:  1,
		OutputMicroUSDPerMillionTokens: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	passed := 0
	for _, testCase := range corpus.Cases {
		input, _ := json.Marshal(struct {
			Dimension       LessonEvaluationDimension `json:"dimension"`
			LearnerContext  string                    `json:"learnerContext"`
			PriorLearning   string                    `json:"priorLearning"`
			Evidence        string                    `json:"evidence"`
			CandidateLesson string                    `json:"candidateLesson"`
		}{
			Dimension:       testCase.Dimension,
			LearnerContext:  testCase.LearnerContext,
			PriorLearning:   testCase.PriorLearning,
			Evidence:        testCase.Evidence,
			CandidateLesson: testCase.CandidateLesson,
		})
		result, err := model.Complete(context.Background(), CompletionRequest{
			Stage:       "lesson-quality-evaluation",
			Structured:  true,
			Instruction: "Act as a strict lesson evaluator. Return JSON only with boolean field \"pass\" and string array \"findings\". Judge only the named quality dimension using the learner context, prior learning, supplied evidence, and candidate lesson.",
			Input:       string(input),
		})
		if err != nil {
			t.Errorf("%s: %v", testCase.ID, err)
			continue
		}
		var judgment struct {
			Pass     bool     `json:"pass"`
			Findings []string `json:"findings"`
		}
		if err := json.Unmarshal([]byte(result.Output), &judgment); err != nil {
			t.Errorf("%s: invalid JSON: %v", testCase.ID, err)
			continue
		}
		if judgment.Pass == testCase.ExpectedPass && len(judgment.Findings) > 0 {
			passed++
		} else {
			t.Errorf("%s: judgment=%#v expectedPass=%v", testCase.ID, judgment, testCase.ExpectedPass)
		}
	}
	if score := float64(passed) / float64(len(corpus.Cases)); score < 0.8 {
		t.Fatalf("Lesson Quality provider evaluation score %.2f is below 0.80", score)
	}
}
