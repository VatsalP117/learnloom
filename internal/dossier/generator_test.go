package dossier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type fakeSources struct{}

func (fakeSources) Fetch(
	_ context.Context,
	_ []domain.SourceDefinition,
	_ int,
) ([]domain.SourceItem, []string, error) {
	now := time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC)
	return []domain.SourceItem{
		{SourceID: "S1", Source: "One", Title: "First", URL: "https://example.com/1", CanonicalURL: "https://example.com/1", Summary: "First source summary", PublishedAt: &now, ContentSource: "feed-summary"},
		{SourceID: "S2", Source: "Two", Title: "Second", URL: "https://example.com/2", CanonicalURL: "https://example.com/2", Summary: "Second source summary", PublishedAt: &now, ContentSource: "feed-summary"},
		{SourceID: "S3", Source: "Three", Title: "Third", URL: "https://example.com/3", CanonicalURL: "https://example.com/3", Summary: "Third source summary", PublishedAt: &now, ContentSource: "feed-summary"},
	}, []string{"optional source unavailable"}, nil
}

func (fakeSources) Enrich(
	_ context.Context,
	items []domain.SourceItem,
) ([]domain.SourceItem, error) {
	for index := range items {
		items[index].ContentSource = "article"
		items[index].Summary = strings.Repeat("Enriched source evidence. ", 30)
	}
	return items, nil
}

type fakeModel struct {
	mu             sync.Mutex
	responses      map[string]string
	requests       *[]CompletionRequest
	parallelActive int
	parallelPeak   int
}

type sequenceModel struct {
	requests  []CompletionRequest
	responses []string
}

func (m *sequenceModel) Complete(_ context.Context, request CompletionRequest) (string, error) {
	m.requests = append(m.requests, request)
	if len(m.responses) == 0 {
		return "", fmt.Errorf("unexpected stage %s", request.Stage)
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return response, nil
}

func (f *fakeModel) Complete(_ context.Context, request CompletionRequest) (string, error) {
	f.mu.Lock()
	if f.requests != nil {
		*f.requests = append(*f.requests, request)
	}
	value, exists := f.responses[request.Stage]
	isParallelStage := request.Stage == "examiner" || request.Stage == "exploration"
	if isParallelStage {
		f.parallelActive++
		f.parallelPeak = max(f.parallelPeak, f.parallelActive)
	}
	f.mu.Unlock()
	if isParallelStage {
		time.Sleep(20 * time.Millisecond)
		f.mu.Lock()
		f.parallelActive--
		f.mu.Unlock()
	}
	if !exists {
		return "", fmt.Errorf("missing stage %s", request.Stage)
	}
	return value, nil
}

func TestGeneratorProducesValidatedArtifact(t *testing.T) {
	t.Parallel()
	lesson := completeLesson()
	practice := completePractice()
	var requests []CompletionRequest
	editor, _ := json.Marshal(editorialOutput{
		Lesson: lesson, Critique: "Evidence is useful but remains bounded by the supplied Source Items [S1].",
		Practice: practice, QualityNotes: []string{"Preserved the evidence contract."},
	})
	model := fakeModel{responses: map[string]string{
		"curator":     `{"theme":"Retrieval and feedback","rationale":"The sources explain one complementary mechanism.","selectedSourceIds":["S1","S2","S3"]}`,
		"blueprint":   `{"learningObjective":"Explain the mechanism","prerequisites":["Recall"],"centralMechanism":"Repeated retrieval plus feedback","workedExample":"A learner recalls then corrects an answer","misconception":"Rereading is equivalent","practicalExperiment":"Compare recall with rereading","continuityBridge":"Builds on the prior lesson"}`,
		"researcher":  "Research grounded in [S1] and [S2].",
		"skeptic":     "The evidence has limits [S1].",
		"teacher":     lesson,
		"examiner":    practice,
		"exploration": "A synthetic analogy and experiment without source citations.",
		"editor":      string(editor),
	}, requests: &requests}
	generator, err := NewGenerator(fakeSources{}, &model, GenerationConfig{
		ModelName: "deepseek-chat",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := generator.Generate(context.Background(), GenerateRequest{
		Newsletter: domain.Newsletter{
			ID: "newsletter-1", Topic: "learning science", LearnerLevel: "experienced",
			LearnerGoal: "build durable knowledge", LessonMinutes: 15,
			TimeZone:             "Asia/Kolkata",
			AIExplorationEnabled: true,
			Sources:              []domain.SourceDefinition{{Name: "Example", URL: "https://example.com/feed", Limit: 3}},
		},
		History: []domain.LearningHistoryEntry{{
			Date:              "2026-07-18",
			LearningObjective: "Compare recognition with independent recall",
			Concepts:          []string{"retrieval strength", "storage strength"},
			SourceTitles:      []string{"Prior evidence review"},
			LessonSummary:     "Recognition can overstate what a learner can retrieve unaided.",
			RecallQuestions:   []string{"Why can familiarity mislead a learner?"},
		}},
		Now: time.Date(2026, 7, 19, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Artifact.Dossier.Date != "2026-07-19" ||
		result.Artifact.Dossier.Quality.Score < 90 ||
		!strings.Contains(result.Artifact.Markdown, "## Source Index") ||
		!strings.Contains(result.Artifact.HTML, "Retrieval and feedback") {
		t.Fatalf("unexpected result: %#v", result.Artifact.Dossier)
	}
	if len(result.Warnings) != 1 || len(result.History.RecallQuestions) != 3 {
		t.Fatalf("missing warnings or history: %#v", result)
	}
	if result.Artifact.Dossier.Exploration == nil || model.parallelPeak != 2 {
		t.Fatalf("exploration=%v peak parallel stages=%d", result.Artifact.Dossier.Exploration, model.parallelPeak)
	}
	if result.Artifact.Dossier.Quality.Metrics["lessonWords"] < 450 ||
		result.Artifact.Dossier.Quality.Metrics["lessonWordsMaximum"] != 1350 {
		t.Fatalf("missing time-fit quality metrics: %#v", result.Artifact.Dossier.Quality.Metrics)
	}
	var editorInput string
	for _, request := range requests {
		if request.Stage == "editor" {
			editorInput = request.Input
			break
		}
	}
	for _, wanted := range []string{
		"Level: experienced",
		"Goal: build durable knowledge",
		"Lesson body budget: 450-1350 words",
		"Objective: Compare recognition with independent recall",
		"Concepts: retrieval strength | storage strength",
		"Sources: Prior evidence review",
	} {
		if !strings.Contains(editorInput, wanted) {
			t.Fatalf("editor input is missing %q:\n%s", wanted, editorInput)
		}
	}
}

func TestEditorReceivesActionableTimeFitRepair(t *testing.T) {
	t.Parallel()
	practice := completePractice()
	encode := func(lesson string) string {
		value, err := json.Marshal(editorialOutput{
			Lesson: lesson, Critique: "Evidence remains bounded [S1].", Practice: practice,
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(value)
	}
	model := &sequenceModel{responses: []string{
		encode(shortCompleteLesson()),
		encode(completeLesson()),
	}}
	budget := lessonWordBudgetFor(15)
	_, err := runStructured(
		context.Background(),
		model,
		"editor",
		stageInstructions()["editor"],
		"original editor input",
		nil,
		func(value editorialOutput) error {
			if err := value.validate(); err != nil {
				return err
			}
			_, err := evaluateQuality(
				value.Lesson,
				value.Critique,
				value.Practice,
				nil,
				[]domain.SourceItem{{SourceID: "S1"}, {SourceID: "S2"}},
				domain.LearningBlueprint{},
				0,
				budget,
			)
			return err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.requests) != 2 ||
		!strings.Contains(model.requests[1].Input, "Contract repair") ||
		!strings.Contains(model.requests[1].Input, "must contain 450 to 1350 words") {
		t.Fatalf("missing actionable repair request: %#v", model.requests)
	}
}

func completeLesson() string {
	var lesson strings.Builder
	for index, heading := range requiredLessonSections {
		lesson.WriteString("## " + heading + "\n\n")
		for paragraph := 0; paragraph < 4; paragraph++ {
			lesson.WriteString("This section explains a useful causal mechanism with enough concrete detail to support durable understanding")
			if index == 0 && paragraph == 0 {
				lesson.WriteString(" [S1]")
			}
			if index == 1 && paragraph == 0 {
				lesson.WriteString(" [S2]")
			}
			lesson.WriteString(". ")
		}
		lesson.WriteString("\n\n")
	}
	return lesson.String()
}

func shortCompleteLesson() string {
	var lesson strings.Builder
	for index, heading := range requiredLessonSections {
		lesson.WriteString("## " + heading + "\n\n")
		lesson.WriteString("Concise but substantive mechanism explanation for durable learning")
		if index == 0 {
			lesson.WriteString(" [S1]")
		}
		if index == 1 {
			lesson.WriteString(" [S2]")
		}
		lesson.WriteString(".\n\n")
	}
	return lesson.String()
}

func completePractice() string {
	return `## Retrieval practice

1. How does repeated retrieval produce a stronger and more durable learning effect?
2. Which realistic constraint most changes whether the mechanism works as intended?
3. Why does passive rereading create a misleading feeling of successful learning?

## Application challenge

Apply retrieval and feedback to a current project, then identify evidence that would falsify your chosen approach.

<details>
<summary>Answer key</summary>

1. Repeated retrieval strengthens access routes while feedback corrects errors before they become stable.
2. Delayed corrective feedback can allow a confidently recalled error to become more persistent.
3. Familiarity rises during rereading even when independent recall remains weak and unreliable.

</details>`
}
