package dossier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	responses map[string]string
}

func (f fakeModel) Complete(_ context.Context, request CompletionRequest) (string, error) {
	value, exists := f.responses[request.Stage]
	if !exists {
		return "", fmt.Errorf("missing stage %s", request.Stage)
	}
	return value, nil
}

func TestGeneratorProducesValidatedArtifact(t *testing.T) {
	t.Parallel()
	lesson := completeLesson()
	practice := completePractice()
	editor, _ := json.Marshal(editorialOutput{
		Lesson: lesson, Critique: "Evidence is useful but remains bounded by the supplied Source Items [S1].",
		Practice: practice, QualityNotes: []string{"Preserved the evidence contract."},
	})
	model := fakeModel{responses: map[string]string{
		"curator":    `{"theme":"Retrieval and feedback","rationale":"The sources explain one complementary mechanism.","selectedSourceIds":["S1","S2","S3"]}`,
		"blueprint":  `{"learningObjective":"Explain the mechanism","prerequisites":["Recall"],"centralMechanism":"Repeated retrieval plus feedback","workedExample":"A learner recalls then corrects an answer","misconception":"Rereading is equivalent","practicalExperiment":"Compare recall with rereading","continuityBridge":"Builds on the prior lesson"}`,
		"researcher": "Research grounded in [S1] and [S2].",
		"skeptic":    "The evidence has limits [S1].",
		"teacher":    lesson,
		"examiner":   practice,
		"editor":     string(editor),
	}}
	generator, err := NewGenerator(fakeSources{}, model, GenerationConfig{
		ModelName: "deepseek-chat",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := generator.Generate(context.Background(), GenerateRequest{
		Newsletter: domain.Newsletter{
			ID: "newsletter-1", Topic: "learning science", LearnerLevel: "experienced",
			LearnerGoal: "build durable knowledge", LessonMinutes: 15,
			TimeZone: "Asia/Kolkata",
			Sources:  []domain.SourceDefinition{{Name: "Example", URL: "https://example.com/feed", Limit: 3}},
		},
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
}

func completeLesson() string {
	var lesson strings.Builder
	for index, heading := range requiredLessonSections {
		lesson.WriteString("## " + heading + "\n\n")
		lesson.WriteString("This section explains a useful causal mechanism with enough concrete detail to support durable understanding")
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
