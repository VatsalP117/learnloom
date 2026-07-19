package dossier

import (
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestEvaluateQualityAcceptsCompleteDossier(t *testing.T) {
	t.Parallel()
	var lesson strings.Builder
	for index, heading := range requiredLessonSections {
		lesson.WriteString("## " + heading + "\n\n")
		lesson.WriteString("This section explains a useful mechanism with enough concrete detail to support durable learning")
		if index < 2 {
			lesson.WriteString(" [S" + string(rune('1'+index)) + "]")
		}
		lesson.WriteString(".\n\n")
	}
	practice := `## Retrieval practice

1. How does the central mechanism produce its intended learning effect?
2. Which constraint most changes the mechanism under realistic conditions?
3. Why does the common misconception lead to an incorrect prediction?

## Application challenge

Apply the mechanism to a realistic project and explain which evidence would falsify your chosen approach.

<details>
<summary>Answer key</summary>

1. The mechanism creates the effect through repeated retrieval and corrective feedback.
2. The available evidence changes how confidently the mechanism can be applied.
3. The misconception ignores the causal step that connects practice with durable recall.

</details>`
	report, err := evaluateQuality(
		lesson.String(),
		"A skeptical review compares the evidence [S1].",
		practice,
		nil,
		[]domain.SourceItem{{SourceID: "S1"}, {SourceID: "S2"}},
		domain.LearningBlueprint{ContinuityBridge: "Prior learning"},
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Score < 90 {
		t.Fatalf("unexpected score: %d", report.Score)
	}
}

func TestEvaluateQualityRejectsExplorationCitation(t *testing.T) {
	t.Parallel()
	exploration := "A synthetic claim [S1]"
	_, err := evaluateQuality("", "", "", &exploration, nil, domain.LearningBlueprint{}, 0)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
