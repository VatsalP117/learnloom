package dossier

import (
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestBuildLearningContractPreservesEvidenceAndAnswerRubrics(t *testing.T) {
	t.Parallel()
	contract, err := buildLearningContract(
		domain.Curation{Rationale: "This mechanism advances the learner's goal."},
		domain.LearningBlueprint{
			LearningObjective:     "Explain retrieval strength.",
			Prerequisites:         []string{"Recall"},
			Concepts:              []string{"Retrieval strength", "Feedback"},
			SuggestedNextConcepts: []string{"Spacing effects"},
			Misconception:         "Rereading is equivalent.",
			PracticalExperiment:   "Compare recall with rereading.",
			ContinuityBridge:      "Builds on recognition.",
		},
		"## Mechanism\n\nRetrieval plus feedback improves later access [S1] [S2].",
		"## Limits\n\nEffects depend on corrective feedback [S2].",
		"## Retrieval practice\n\n1. Why does retrieval help?\n2. Why does feedback matter?\n3. When can retrieval fail?\n\n"+
			"## Application challenge\n\nCompare two study sessions.\n\n"+
			"<details>\n<summary>Answer key</summary>\n\n"+
			"1. It strengthens later access.\n2. It corrects errors.\n"+
			"3. It can stabilize errors without feedback.\n</details>",
		[]domain.SourceItem{{SourceID: "S1"}, {SourceID: "S2"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Version != 1 || len(contract.Concepts) != 3 ||
		len(contract.Claims) != 2 || len(contract.Retrieval) != 3 {
		t.Fatalf("contract=%#v", contract)
	}
	if contract.Claims[0].SourceIDs[0] != "S1" ||
		contract.Retrieval[0].AnswerRubric != "It strengthens later access." ||
		len(contract.Retrieval[0].ConceptIDs) != 2 {
		t.Fatalf("evidence or rubric was not preserved: %#v", contract)
	}
}
