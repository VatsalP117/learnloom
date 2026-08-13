package dossier

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func buildLearningContract(
	curation domain.Curation,
	blueprint domain.LearningBlueprint,
	lesson, critique, practice string,
	sources []domain.SourceItem,
	history []domain.LearningHistoryEntry,
) (domain.LearningContract, error) {
	concepts := contractConcepts(blueprint)
	conceptChanges := contractConceptChanges(concepts, history)
	claims, err := evidenceClaims(lesson, sources)
	if err != nil {
		return domain.LearningContract{}, err
	}
	limitations, err := evidenceClaims(critique, sources)
	if err != nil {
		return domain.LearningContract{}, err
	}
	retrieval := retrievalPrompts(practice)
	conceptIDs := make([]string, 0, len(concepts))
	for _, concept := range concepts {
		if concept.Role == "core" {
			conceptIDs = append(conceptIDs, concept.ID)
		}
	}
	for index := range retrieval {
		retrieval[index].ConceptIDs = append([]string(nil), conceptIDs...)
	}
	if len(concepts) == 0 || len(claims) == 0 || len(limitations) == 0 || len(retrieval) < 3 {
		return domain.LearningContract{}, errors.New(
			"structured learning contract is incomplete",
		)
	}
	return domain.LearningContract{
		Version:               2,
		LessonType:            blueprint.LessonType,
		EvidenceStatus:        domain.EvidenceSourceBounded,
		SelectionRationale:    strings.TrimSpace(curation.Rationale),
		LearningObjective:     strings.TrimSpace(blueprint.LearningObjective),
		ContinuityBridge:      strings.TrimSpace(blueprint.ContinuityBridge),
		Concepts:              concepts,
		ConceptChanges:        conceptChanges,
		Misconception:         strings.TrimSpace(blueprint.Misconception),
		Claims:                claims,
		Limitations:           limitations,
		Retrieval:             retrieval,
		SuggestedNextConcepts: cleanUniqueStrings(blueprint.SuggestedNextConcepts),
		Application:           strings.TrimSpace(blueprint.PracticalExperiment),
	}, nil
}

func contractConceptChanges(
	concepts []domain.LearningConcept,
	history []domain.LearningHistoryEntry,
) []domain.ConceptChange {
	known := map[string]struct{}{}
	for _, entry := range history {
		for _, concept := range entry.ConceptStates {
			known[conceptID(concept.Label)] = struct{}{}
		}
		for _, concept := range entry.Concepts {
			known[conceptID(concept)] = struct{}{}
		}
	}
	changes := make([]domain.ConceptChange, 0, len(concepts))
	for _, concept := range concepts {
		if concept.Role != "core" {
			continue
		}
		change := "introduced"
		if _, exists := known[concept.ID]; exists {
			change = "reinforced"
		}
		changes = append(changes, domain.ConceptChange{
			ID: concept.ID, Label: concept.Label, Change: change,
		})
	}
	return changes
}

func contractConcepts(blueprint domain.LearningBlueprint) []domain.LearningConcept {
	var result []domain.LearningConcept
	seen := map[string]struct{}{}
	appendConcept := func(label, role string) {
		label = strings.TrimSpace(label)
		key := strings.ToLower(label)
		if label == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result = append(result, domain.LearningConcept{
			ID:    conceptID(label),
			Label: label,
			Role:  role,
		})
	}
	for _, concept := range blueprint.Concepts {
		appendConcept(concept, "core")
	}
	for _, prerequisite := range blueprint.Prerequisites {
		appendConcept(prerequisite, "prerequisite")
	}
	return result
}

func evidenceClaims(
	markdown string,
	sources []domain.SourceItem,
) ([]domain.EvidenceClaim, error) {
	known := make(map[string]struct{}, len(sources))
	for index, source := range sources {
		id := source.SourceID
		if id == "" {
			id = fmt.Sprintf("S%d", index+1)
		}
		known[id] = struct{}{}
	}
	var result []domain.EvidenceClaim
	for _, block := range strings.Split(markdown, "\n\n") {
		ids := sourceIDs(block)
		if len(ids) == 0 {
			continue
		}
		for _, id := range ids {
			if _, exists := known[id]; !exists {
				return nil, fmt.Errorf(
					"structured claim cites unknown Source Item %s",
					id,
				)
			}
		}
		text := plainText(block)
		if len([]rune(text)) < 20 {
			continue
		}
		result = append(result, domain.EvidenceClaim{
			ID:        fmt.Sprintf("claim-%d", len(result)+1),
			Text:      text,
			SourceIDs: ids,
		})
	}
	return result, nil
}

func retrievalPrompts(practice string) []domain.RetrievalPrompt {
	questions := extractQuestions(practice)
	answerKey := betweenFold(practice, "<summary>Answer key</summary>", "</details>")
	answers := numberedAnswers(answerKey)
	result := make([]domain.RetrievalPrompt, 0, len(questions))
	for index, question := range questions {
		answer := plainText(answers[index+1])
		if answer == "" {
			continue
		}
		result = append(result, domain.RetrievalPrompt{
			ID:                    fmt.Sprintf("retrieval-%d", index+1),
			Prompt:                question,
			AnswerRubric:          answer,
			CorrectiveExplanation: answer,
		})
	}
	return result
}

func conceptID(value string) string {
	var result strings.Builder
	separator := false
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			separator = false
			continue
		}
		separator = true
	}
	if result.Len() == 0 {
		return "concept"
	}
	return result.String()
}

func cleanUniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}
