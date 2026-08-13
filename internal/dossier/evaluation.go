package dossier

import (
	"errors"
	"fmt"
	"strings"
)

type LessonEvaluationDimension string

const (
	EvalUsefulness        LessonEvaluationDimension = "usefulness"
	EvalContinuity        LessonEvaluationDimension = "continuity"
	EvalDifficultyFit     LessonEvaluationDimension = "difficulty_fit"
	EvalUnsupportedClaims LessonEvaluationDimension = "unsupported_claims"
	EvalRedundancy        LessonEvaluationDimension = "redundancy"
	EvalTimeFit           LessonEvaluationDimension = "time_fit"
)

var requiredLessonEvaluationDimensions = []LessonEvaluationDimension{
	EvalUsefulness,
	EvalContinuity,
	EvalDifficultyFit,
	EvalUnsupportedClaims,
	EvalRedundancy,
	EvalTimeFit,
}

type LessonEvaluationCorpus struct {
	Version     string                 `json:"version"`
	LabelStatus string                 `json:"labelStatus"`
	Cases       []LessonEvaluationCase `json:"cases"`
}

type LessonEvaluationCase struct {
	ID               string                    `json:"id"`
	Dimension        LessonEvaluationDimension `json:"dimension"`
	LearnerContext   string                    `json:"learnerContext"`
	PriorLearning    string                    `json:"priorLearning"`
	Evidence         string                    `json:"evidence"`
	CandidateLesson  string                    `json:"candidateLesson"`
	ExpectedPass     bool                      `json:"expectedPass"`
	ExpectedFindings []string                  `json:"expectedFindings"`
}

func ValidateLessonEvaluationCorpus(corpus LessonEvaluationCorpus) error {
	if strings.TrimSpace(corpus.Version) == "" {
		return errors.New("Lesson Evaluation corpus version is required")
	}
	if corpus.LabelStatus != "product_labeled" && corpus.LabelStatus != "human_reviewed" {
		return errors.New("Lesson Evaluation corpus label status is invalid")
	}
	if len(corpus.Cases) < len(requiredLessonEvaluationDimensions)*2 {
		return fmt.Errorf(
			"Lesson Evaluation corpus has %d cases; at least two per dimension are required",
			len(corpus.Cases),
		)
	}
	seen := make(map[string]struct{}, len(corpus.Cases))
	dimensions := make(map[LessonEvaluationDimension]int)
	passByDimension := make(map[LessonEvaluationDimension]bool)
	failByDimension := make(map[LessonEvaluationDimension]bool)
	for _, testCase := range corpus.Cases {
		if strings.TrimSpace(testCase.ID) == "" {
			return errors.New("Lesson Evaluation case ID is required")
		}
		if _, exists := seen[testCase.ID]; exists {
			return fmt.Errorf("Lesson Evaluation case ID %q is duplicated", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
		if !validLessonEvaluationDimension(testCase.Dimension) {
			return fmt.Errorf("Lesson Evaluation case %q has an invalid dimension", testCase.ID)
		}
		if strings.TrimSpace(testCase.LearnerContext) == "" ||
			strings.TrimSpace(testCase.Evidence) == "" ||
			strings.TrimSpace(testCase.CandidateLesson) == "" ||
			len(testCase.ExpectedFindings) == 0 {
			return fmt.Errorf("Lesson Evaluation case %q is incomplete", testCase.ID)
		}
		for _, finding := range testCase.ExpectedFindings {
			if strings.TrimSpace(finding) == "" {
				return fmt.Errorf("Lesson Evaluation case %q has an empty finding", testCase.ID)
			}
		}
		dimensions[testCase.Dimension]++
		if testCase.ExpectedPass {
			passByDimension[testCase.Dimension] = true
		} else {
			failByDimension[testCase.Dimension] = true
		}
	}
	for _, dimension := range requiredLessonEvaluationDimensions {
		if dimensions[dimension] < 2 || !passByDimension[dimension] ||
			!failByDimension[dimension] {
			return fmt.Errorf(
				"Lesson Evaluation dimension %q needs at least one passing and one failing case",
				dimension,
			)
		}
	}
	return nil
}

func validLessonEvaluationDimension(value LessonEvaluationDimension) bool {
	for _, dimension := range requiredLessonEvaluationDimensions {
		if value == dimension {
			return true
		}
	}
	return false
}
