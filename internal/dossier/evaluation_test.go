package dossier

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLessonEvaluationCorpusCoversLaunchQualityDimensions(t *testing.T) {
	t.Parallel()
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
	if len(corpus.Cases) != 12 {
		t.Fatalf("Lesson Evaluation corpus has %d cases, want 12", len(corpus.Cases))
	}
}

func TestLessonEvaluationCorpusRejectsOneSidedDimension(t *testing.T) {
	t.Parallel()
	corpus := LessonEvaluationCorpus{
		Version: "test", LabelStatus: "product_labeled",
	}
	for _, dimension := range requiredLessonEvaluationDimensions {
		for index := 0; index < 2; index++ {
			corpus.Cases = append(corpus.Cases, LessonEvaluationCase{
				ID:               string(dimension) + string(rune('a'+index)),
				Dimension:        dimension,
				LearnerContext:   "A learner context",
				Evidence:         "Source-bounded evidence [S1].",
				CandidateLesson:  "A candidate lesson.",
				ExpectedPass:     true,
				ExpectedFindings: []string{"A finding"},
			})
		}
	}
	if err := ValidateLessonEvaluationCorpus(corpus); err == nil {
		t.Fatal("one-sided corpus was accepted")
	}
}
