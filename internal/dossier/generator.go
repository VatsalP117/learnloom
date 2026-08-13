package dossier

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/failure"
	"golang.org/x/sync/errgroup"
)

type SourceAcquirer interface {
	Fetch(context.Context, []domain.SourceDefinition, int) ([]domain.SourceItem, []string, error)
	Enrich(context.Context, []domain.SourceItem) ([]domain.SourceItem, error)
}

type GenerationConfig struct {
	ModelName                 string
	MaxItems                  int
	MaxItemCharacters         int
	MaxArticleCharacters      int
	MaxIntermediateCharacters int
	HistoryEntries            int
}

type Generator struct {
	sources SourceAcquirer
	model   Completer
	cfg     GenerationConfig
}

type GenerateRequest struct {
	Newsletter          domain.Newsletter
	History             []domain.LearningHistoryEntry
	LearnerState        domain.LearnerState
	Now                 time.Time
	RequestedLessonType domain.LessonType
	OnStage             StageObserver
	OnCheckpoint        func(stage, output string)
	PreparedItems       []domain.SourceItem
	Warnings            []string
	Checkpoints         map[string]string
}

type StageObserver func(
	stage string,
	duration time.Duration,
	usage ModelUsage,
	err error,
)

const PipelineVersion = "dossier-v4"

func GenerationFingerprint(request GenerateRequest, modelName string) (string, error) {
	payload := struct {
		PipelineVersion string `json:"pipelineVersion"`
		ModelName       string `json:"modelName"`
		Newsletter      struct {
			ID                   string                    `json:"id"`
			Topic                string                    `json:"topic"`
			LearnerLevel         string                    `json:"learnerLevel"`
			LearnerGoal          string                    `json:"learnerGoal"`
			LessonMinutes        int                       `json:"lessonMinutes"`
			SourceMode           domain.SourceMode         `json:"sourceMode"`
			Sources              []domain.SourceDefinition `json:"sources"`
			TimeZone             string                    `json:"timeZone"`
			AIExplorationEnabled bool                      `json:"aiExplorationEnabled"`
		} `json:"newsletter"`
		History             []domain.LearningHistoryEntry `json:"history"`
		LearnerState        domain.LearnerState           `json:"learnerState"`
		PreparedItems       []domain.SourceItem           `json:"preparedItems"`
		RequestedLessonType domain.LessonType             `json:"requestedLessonType,omitempty"`
	}{
		PipelineVersion:     PipelineVersion,
		ModelName:           modelName,
		History:             request.History,
		LearnerState:        request.LearnerState,
		PreparedItems:       request.PreparedItems,
		RequestedLessonType: request.RequestedLessonType,
	}
	payload.Newsletter.ID = request.Newsletter.ID
	payload.Newsletter.Topic = request.Newsletter.Topic
	payload.Newsletter.LearnerLevel = request.Newsletter.LearnerLevel
	payload.Newsletter.LearnerGoal = request.Newsletter.LearnerGoal
	payload.Newsletter.LessonMinutes = request.Newsletter.LessonMinutes
	payload.Newsletter.SourceMode = request.Newsletter.SourceMode
	payload.Newsletter.Sources = request.Newsletter.Sources
	payload.Newsletter.TimeZone = request.Newsletter.TimeZone
	payload.Newsletter.AIExplorationEnabled = request.Newsletter.AIExplorationEnabled
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode generation fingerprint: %w", err)
	}
	checksum := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", checksum[:]), nil
}

func restoreStructuredCheckpoint[T any](
	checkpoints map[string]string,
	stage string,
	validate func(T) error,
) (T, bool) {
	var zero T
	raw := strings.TrimSpace(checkpoints[stage])
	if raw == "" {
		return zero, false
	}
	var value T
	if err := decodeStructured(raw, &value); err != nil {
		return zero, false
	}
	if err := validate(value); err != nil {
		return zero, false
	}
	return value, true
}

func restoreTextCheckpoint(checkpoints map[string]string, stage string) (string, bool) {
	value := strings.TrimSpace(checkpoints[stage])
	return value, value != ""
}

func saveStructuredCheckpoint[T any](request GenerateRequest, stage string, value T) {
	if request.OnCheckpoint == nil {
		return
	}
	encoded, err := json.Marshal(value)
	if err == nil {
		request.OnCheckpoint(stage, string(encoded))
	}
}

func saveTextCheckpoint(request GenerateRequest, stage, value string) {
	if request.OnCheckpoint != nil && strings.TrimSpace(value) != "" {
		request.OnCheckpoint(stage, value)
	}
}

type GenerateResult struct {
	Artifact domain.DossierArtifact
	History  domain.LearningHistoryEntry
	Warnings []string
}

func NewGenerator(
	sources SourceAcquirer,
	model Completer,
	cfg GenerationConfig,
) (*Generator, error) {
	if sources == nil || model == nil {
		return nil, errors.New("Dossier production requires Source Item and model implementations")
	}
	if cfg.ModelName == "" {
		return nil, errors.New("Dossier production requires a model name")
	}
	if cfg.MaxItems == 0 {
		cfg.MaxItems = 18
	}
	if cfg.MaxItemCharacters == 0 {
		cfg.MaxItemCharacters = 1800
	}
	if cfg.MaxArticleCharacters == 0 {
		cfg.MaxArticleCharacters = 16_000
	}
	if cfg.MaxIntermediateCharacters == 0 {
		cfg.MaxIntermediateCharacters = 24_000
	}
	if cfg.HistoryEntries == 0 {
		cfg.HistoryEntries = 14
	}
	return &Generator{sources: sources, model: model, cfg: cfg}, nil
}

func (g *Generator) Generate(
	ctx context.Context,
	request GenerateRequest,
) (GenerateResult, error) {
	now := request.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var items []domain.SourceItem
	warnings := slices.Clone(request.Warnings)
	var err error

	if len(request.PreparedItems) > 0 {
		items = request.PreparedItems
	} else {
		if len(request.Newsletter.Sources) == 0 {
			return GenerateResult{}, errors.New("Newsletter requires at least one Source Item definition")
		}
		items, warnings, err = g.sources.Fetch(
			ctx,
			request.Newsletter.Sources,
			g.cfg.MaxItems,
		)
		if err != nil {
			return GenerateResult{}, err
		}
	}
	learnerContext := g.learnerContext(
		request.Newsletter,
		request.History,
		request.LearnerState,
	)
	wordBudget := lessonWordBudgetFor(request.Newsletter.LessonMinutes)
	candidateCharacters := max(300, min(
		g.cfg.MaxItemCharacters,
		int(float64(g.cfg.MaxIntermediateCharacters)*0.7)/len(items),
	))
	candidateBundle := formatSourceBundle(items, candidateCharacters)

	curation, restored := restoreStructuredCheckpoint(
		request.Checkpoints,
		"curator",
		func(value domain.Curation) error { return validateCuration(value, len(items)) },
	)
	if !restored {
		curation, err = runStructured(
			ctx,
			g.model,
			"curator",
			stageInstructions()["curator"],
			learnerContext+"\n\n# Candidate sources\n\n"+candidateBundle,
			request.OnStage,
			func(value domain.Curation) error { return validateCuration(value, len(items)) },
		)
		if err == nil {
			saveStructuredCheckpoint(request, "curator", curation)
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	curated := make([]domain.SourceItem, 0, len(curation.SelectedSourceIDs))
	for index, sourceID := range curation.SelectedSourceIDs {
		sourceIndex, _ := strconv.Atoi(strings.TrimPrefix(sourceID, "S"))
		item := items[sourceIndex-1]
		item.OriginalID = sourceID
		item.SourceID = fmt.Sprintf("S%d", index+1)
		curated = append(curated, item)
	}
	enriched := curated
	if len(request.PreparedItems) == 0 {
		var enrichErr error
		enriched, enrichErr = g.sources.Enrich(ctx, curated)
		if enrichErr != nil {
			return GenerateResult{}, enrichErr
		}
	}
	sourceCharacters := max(1000, min(
		g.cfg.MaxArticleCharacters,
		int(float64(g.cfg.MaxIntermediateCharacters)*0.45)/len(enriched),
	))
	sourceBundle := formatSourceBundle(enriched, sourceCharacters)

	blueprint, restored := restoreStructuredCheckpoint(
		request.Checkpoints,
		"blueprint",
		validateBlueprint,
	)
	if !restored {
		blueprint, err = runStructured(
			ctx,
			g.model,
			"blueprint",
			stageInstructions()["blueprint"],
			fitSections(g.cfg.MaxIntermediateCharacters, []weightedSection{
				{"Learner context", learnerContext, 2},
				{"Requested lesson type", string(request.RequestedLessonType), 1},
				{"Curated theme", prettyJSON(curation), 1},
				{"Enriched sources", sourceBundle, 5},
			}),
			request.OnStage,
			validateBlueprint,
		)
		if err == nil {
			saveStructuredCheckpoint(request, "blueprint", blueprint)
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	if validLessonType(request.RequestedLessonType) {
		blueprint.LessonType = request.RequestedLessonType
	} else {
		blueprint.LessonType = resolveLessonType(blueprint.LessonType, len(request.History))
	}
	if repeatsRecentLearningSignal(blueprint, curated, request.History) {
		return GenerateResult{}, failure.New(
			failure.CodeNoNewLearningSignal,
			failure.CategoryInsufficientEvidence,
			"novelty_gate",
			false,
			failure.PublicNoEvidence,
			errors.New("candidate objective, concepts, and source portfolio repeat a recent lesson"),
		)
	}
	blueprintText := prettyJSON(blueprint)
	research, restored := restoreTextCheckpoint(request.Checkpoints, "researcher")
	if !restored {
		research, err = g.runStage(ctx, "researcher", fitSections(
			g.cfg.MaxIntermediateCharacters,
			[]weightedSection{
				{"Learner context", learnerContext, 1},
				{"Learning blueprint", blueprintText, 2},
				{"Enriched sources", sourceBundle, 6},
			},
		), request.OnStage)
		if err == nil {
			saveTextCheckpoint(request, "researcher", research)
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	critique, restored := restoreTextCheckpoint(request.Checkpoints, "skeptic")
	if !restored {
		critique, err = g.runStage(ctx, "skeptic", fitSections(
			g.cfg.MaxIntermediateCharacters,
			[]weightedSection{
				{"Learning blueprint", blueprintText, 1},
				{"Enriched sources", sourceBundle, 5},
				{"Research brief", research, 3},
			},
		), request.OnStage)
		if err == nil {
			saveTextCheckpoint(request, "skeptic", critique)
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	lesson, restored := restoreTextCheckpoint(request.Checkpoints, "teacher")
	if !restored {
		lesson, err = g.runStage(ctx, "teacher", fitSections(
			g.cfg.MaxIntermediateCharacters,
			[]weightedSection{
				{"Learner context", learnerContext, 1},
				{"Learning blueprint", blueprintText, 2},
				{"Enriched sources", sourceBundle, 3},
				{"Research brief", research, 3},
				{"Skeptical review", critique, 2},
			},
		), request.OnStage)
		if err == nil {
			saveTextCheckpoint(request, "teacher", lesson)
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	practiceInput := fitSections(
		g.cfg.MaxIntermediateCharacters,
		[]weightedSection{
			{"Learner context", learnerContext, 1},
			{"Learning blueprint", blueprintText, 2},
			{"Source-grounded lesson", lesson, 6},
		},
	)
	var practice string
	var exploration *string
	if request.Newsletter.AIExplorationEnabled {
		explorationInput := fitSections(
			g.cfg.MaxIntermediateCharacters,
			[]weightedSection{
				{"Learner context", learnerContext, 1},
				{"Learning blueprint", blueprintText, 2},
				{"Source-grounded lesson", lesson, 5},
				{"Skeptical review", critique, 2},
			},
		)
		group, groupContext := errgroup.WithContext(ctx)
		group.Go(func() error {
			var stageErr error
			practice, stageErr = g.runStage(
				groupContext,
				"examiner",
				practiceInput,
				request.OnStage,
			)
			return stageErr
		})
		group.Go(func() error {
			value, stageErr := g.runStage(
				groupContext,
				"exploration",
				explorationInput,
				request.OnStage,
			)
			if stageErr == nil {
				exploration = &value
			}
			return stageErr
		})
		if err := group.Wait(); err != nil {
			return GenerateResult{}, err
		}
	} else {
		practice, err = g.runStage(ctx, "examiner", practiceInput, request.OnStage)
		if err != nil {
			return GenerateResult{}, err
		}
	}

	editorial, err := runStructured(
		ctx,
		g.model,
		"editor",
		stageInstructions()["editor"],
		fitSectionsWithRequired(
			g.cfg.MaxIntermediateCharacters,
			[]weightedSection{
				{"Learner context and time-fit contract", learnerContext, 2},
				{"Learning blueprint", blueprintText, 2},
				{"Enriched sources", sourceBundle, 3},
				{"Draft lesson", lesson, 5},
				{"Skeptical review", critique, 2},
			},
			weightedSection{"Draft practice", practice, 3},
		),
		request.OnStage,
		func(value editorialOutput) error {
			if err := value.validate(); err != nil {
				return err
			}
			_, candidateErr := evaluateQuality(
				value.Lesson,
				value.Critique,
				value.Practice,
				nil,
				enriched,
				blueprint,
				len(request.History),
				wordBudget,
			)
			if candidateErr == nil {
				return nil
			}
			// The examiner's practice is already a distinct stage output. If
			// the editor damages only that practice contract, preserve the
			// original rather than rejecting an otherwise valid lesson.
			_, preservedErr := evaluateQuality(
				value.Lesson,
				value.Critique,
				practice,
				nil,
				enriched,
				blueprint,
				len(request.History),
				wordBudget,
			)
			if preservedErr == nil {
				return nil
			}
			return candidateErr
		},
	)
	if err != nil {
		// The teacher and examiner outputs are independently produced and are
		// often still valid when the final editor damages citations, time fit,
		// or the answer key. Prefer that already-validated pair to failing the
		// whole Issue after repeated editor-only contract drift.
		if _, fallbackErr := evaluateQuality(
			lesson,
			critique,
			practice,
			exploration,
			enriched,
			blueprint,
			len(request.History),
			wordBudget,
		); fallbackErr == nil {
			editorial = editorialOutput{
				Lesson:   lesson,
				Critique: critique,
				Practice: practice,
				QualityNotes: []string{
					"Preserved validated teacher and examiner outputs after editor contract drift.",
				},
			}
			err = nil
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	finalPractice := editorial.Practice
	quality, err := evaluateQuality(
		editorial.Lesson,
		editorial.Critique,
		finalPractice,
		exploration,
		enriched,
		blueprint,
		len(request.History),
		wordBudget,
	)
	if err != nil {
		preservedQuality, preservedErr := evaluateQuality(
			editorial.Lesson,
			editorial.Critique,
			practice,
			exploration,
			enriched,
			blueprint,
			len(request.History),
			wordBudget,
		)
		if preservedErr == nil {
			finalPractice = practice
			quality = preservedQuality
			editorial.QualityNotes = append(
				editorial.QualityNotes,
				"Preserved validated examiner practice after editor contract drift.",
			)
			err = nil
		}
	}
	if err != nil {
		return GenerateResult{}, err
	}
	quality.EditorNotes = slices.Clone(editorial.QualityNotes)
	learning, err := buildLearningContract(
		curation,
		blueprint,
		editorial.Lesson,
		editorial.Critique,
		finalPractice,
		enriched,
		request.History,
	)
	if err != nil {
		return GenerateResult{}, err
	}
	date, err := localDate(now, request.Newsletter.TimeZone)
	if err != nil {
		return GenerateResult{}, err
	}
	dossier := domain.Dossier{
		Version:     4,
		ProfileID:   request.Newsletter.ID,
		Date:        date,
		Title:       curation.Theme,
		LessonType:  blueprint.LessonType,
		GeneratedAt: now.UTC(),
		Model:       g.cfg.ModelName,
		Curation:    curation,
		Blueprint:   blueprint,
		Learning:    learning,
		Lesson:      editorial.Lesson,
		Critique:    editorial.Critique,
		Practice:    finalPractice,
		Exploration: exploration,
		Quality:     quality,
		Sources:     enriched,
	}
	markdown := RenderMarkdown(dossier)
	html := RenderHTML(dossier, "")
	history := domain.LearningHistoryEntry{
		Date:                  date,
		GeneratedAt:           now.UTC(),
		LessonType:            blueprint.LessonType,
		SourceTitles:          mapSourceTitles(enriched),
		SourceURLs:            mapSourceURLs(enriched),
		LessonSummary:         truncate(stripMarkdown(editorial.Lesson), 800),
		RecallQuestions:       extractQuestions(finalPractice),
		RetrievalPrompts:      slices.Clone(learning.Retrieval),
		ConceptStates:         slices.Clone(learning.Concepts),
		SuggestedNextConcepts: slices.Clone(learning.SuggestedNextConcepts),
		LearningObjective:     blueprint.LearningObjective,
		Concepts:              blueprintConcepts(blueprint),
	}
	return GenerateResult{
		Artifact: domain.DossierArtifact{
			Dossier:  dossier,
			Markdown: markdown,
			HTML:     html,
		},
		History:  history,
		Warnings: warnings,
	}, nil
}

func fitSectionsWithRequired(
	maximum int,
	optional []weightedSection,
	required weightedSection,
) string {
	requiredText := "# " + required.heading + "\n\n" + required.content
	remaining := maximum - len([]rune(requiredText)) - 2
	if remaining < 1000 {
		remaining = 1000
	}
	optionalText := fitSections(remaining, optional)
	if optionalText == "" {
		return requiredText
	}
	return optionalText + "\n\n" + requiredText
}

func decodeStructured(output string, value any) error {
	candidate := strings.TrimSpace(output)
	if strings.HasPrefix(candidate, "```") && strings.HasSuffix(candidate, "```") {
		candidate = strings.TrimSuffix(strings.TrimPrefix(candidate, "```"), "```")
		candidate = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(candidate), "json"))
	}
	if candidate == "" {
		return errors.New("structured output was empty")
	}
	decoder := json.NewDecoder(strings.NewReader(candidate))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("structured output was invalid JSON: %w", err)
	}
	return nil
}

type editorialOutput struct {
	Lesson       string   `json:"lesson"`
	Critique     string   `json:"critique"`
	Practice     string   `json:"practice"`
	Exploration  *string  `json:"exploration"`
	QualityNotes []string `json:"qualityNotes"`
}

func (e editorialOutput) validate() error {
	for name, value := range map[string]string{
		"lesson": e.Lesson, "critique": e.Critique, "practice": e.Practice,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("editorial %s must be non-empty", name)
		}
	}
	if e.Exploration != nil && strings.TrimSpace(*e.Exploration) != "" {
		return errors.New("editorial output must not include AI Exploration")
	}
	if len(e.Lesson) > 60_000 || len(e.Critique) > 30_000 || len(e.Practice) > 30_000 {
		return errors.New("editorial output exceeded its size contract")
	}
	if len(e.QualityNotes) > 10 {
		return errors.New("editorial output contains too many quality notes")
	}
	return nil
}

func validateCuration(value domain.Curation, itemCount int) error {
	if strings.TrimSpace(value.Theme) == "" || len(value.Theme) > 500 {
		return errors.New("curator theme is invalid")
	}
	if strings.TrimSpace(value.Rationale) == "" || len(value.Rationale) > 1000 {
		return errors.New("curator rationale is invalid")
	}
	minimum := min(3, itemCount)
	maximum := min(5, itemCount)
	if len(value.SelectedSourceIDs) < minimum || len(value.SelectedSourceIDs) > maximum {
		return fmt.Errorf("curator must select %d to %d Source Items", minimum, maximum)
	}
	seen := map[string]struct{}{}
	for _, id := range value.SelectedSourceIDs {
		index, err := strconv.Atoi(strings.TrimPrefix(id, "S"))
		if err != nil || fmt.Sprintf("S%d", index) != id || index < 1 || index > itemCount {
			return fmt.Errorf("curator selected unknown Source Item %s", id)
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("curator selected duplicate Source Item %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateBlueprint(value domain.LearningBlueprint) error {
	if value.LessonType != "" && !validLessonType(value.LessonType) {
		return errors.New("Blueprint lesson type is invalid")
	}
	fields := map[string]string{
		"learning objective":   value.LearningObjective,
		"central mechanism":    value.CentralMechanism,
		"worked example":       value.WorkedExample,
		"misconception":        value.Misconception,
		"practical experiment": value.PracticalExperiment,
		"continuity bridge":    value.ContinuityBridge,
	}
	for name, field := range fields {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("Blueprint %s must be non-empty", name)
		}
	}
	if len(value.Prerequisites) == 0 || len(value.Prerequisites) > 5 {
		return errors.New("Blueprint prerequisites must contain one to five items")
	}
	if err := validateShortList("concepts", value.Concepts, 1, 8); err != nil {
		return err
	}
	if err := validateShortList(
		"suggested next concepts",
		value.SuggestedNextConcepts,
		1,
		5,
	); err != nil {
		return err
	}
	return nil
}

func validLessonType(value domain.LessonType) bool {
	switch value {
	case domain.LessonFoundation,
		domain.LessonUpdate,
		domain.LessonDeepDive,
		domain.LessonSynthesis,
		domain.LessonApplication,
		domain.LessonReview:
		return true
	default:
		return false
	}
}

func resolveLessonType(value domain.LessonType, historyCount int) domain.LessonType {
	if validLessonType(value) {
		if historyCount == 0 {
			return domain.LessonFoundation
		}
		return value
	}
	if historyCount == 0 {
		return domain.LessonFoundation
	}
	return domain.LessonDeepDive
}

func validateShortList(name string, values []string, minimum, maximum int) error {
	if len(values) < minimum || len(values) > maximum {
		return fmt.Errorf(
			"Blueprint %s must contain %d to %d items",
			name,
			minimum,
			maximum,
		)
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" || len([]rune(normalized)) > 200 {
			return fmt.Errorf("Blueprint %s contains an invalid item", name)
		}
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("Blueprint %s contains a duplicate item", name)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

type weightedSection struct {
	heading string
	content string
	weight  int
}

func fitSections(maximum int, sections []weightedSection) string {
	totalWeight := 0
	headerLength := 0
	for _, section := range sections {
		totalWeight += section.weight
		headerLength += len([]rune("# " + section.heading + "\n\n"))
	}
	available := max(len(sections), maximum-headerLength-max(0, len(sections)-1)*2)
	var result []string
	allocated := 0
	for index, section := range sections {
		allocation := available * section.weight / totalWeight
		if index == len(sections)-1 {
			allocation = available - allocated
		}
		allocated += allocation
		result = append(result, "# "+section.heading+"\n\n"+truncate(section.content, allocation))
	}
	return strings.Join(result, "\n\n")
}

func formatSourceBundle(items []domain.SourceItem, maximum int) string {
	parts := make([]string, 0, len(items))
	for index, item := range items {
		sourceID := item.SourceID
		if sourceID == "" {
			sourceID = fmt.Sprintf("S%d", index+1)
		}
		published := "unknown"
		if item.PublishedAt != nil {
			published = item.PublishedAt.UTC().Format(time.RFC3339)
		}
		contentBasis := "feed summary"
		if item.ContentSource == "article" {
			contentBasis = "enriched article text"
		}
		lines := []string{
			fmt.Sprintf("## [%s] %s", sourceID, item.Title),
			"Source: " + item.Source,
			"Published: " + published,
			"URL: " + firstText(item.CanonicalURL, item.URL),
			"Content basis: " + contentBasis,
		}
		if item.Author != "" {
			lines = append(lines, "Author: "+item.Author)
		}
		lines = append(lines, "Source text: "+truncate(firstText(item.Summary, "No source text supplied."), maximum))
		parts = append(parts, strings.Join(lines, "\n"))
	}
	return strings.Join(parts, "\n\n")
}

func (g *Generator) learnerContext(
	newsletter domain.Newsletter,
	history []domain.LearningHistoryEntry,
	state domain.LearnerState,
) string {
	retained := history
	if g.cfg.HistoryEntries <= 0 {
		retained = nil
	} else if len(retained) > g.cfg.HistoryEntries {
		retained = retained[len(retained)-g.cfg.HistoryEntries:]
	}
	var prior []string
	for _, entry := range retained {
		recall := entry.RecallQuestions
		if len(recall) > 3 {
			recall = recall[:3]
		}
		concepts := entry.Concepts
		if len(concepts) > 5 {
			concepts = concepts[:5]
		}
		sourceTitles := entry.SourceTitles
		if len(sourceTitles) > 3 {
			sourceTitles = sourceTitles[:3]
		}
		prior = append(prior, fmt.Sprintf(
			"- %s\n  Objective: %s\n  Concepts: %s\n  Sources: %s\n  Summary: %s\n  Recall: %s",
			entry.Date,
			firstText(entry.LearningObjective, "not recorded"),
			firstText(strings.Join(concepts, " | "), "none recorded"),
			firstText(strings.Join(sourceTitles, " | "), "none recorded"),
			firstText(entry.LessonSummary, "not recorded"),
			firstText(strings.Join(recall, " | "), "none recorded"),
		))
	}
	if len(prior) == 0 {
		prior = []string{"- No previous lessons yet."}
	}
	var conceptState []string
	for _, concept := range state.Concepts {
		conceptState = append(conceptState, fmt.Sprintf(
			"- %s (%s): completed %d/%d exposures; confidence %d/100 from %d reviews",
			concept.Label,
			concept.Role,
			concept.CompletedCount,
			concept.ExposureCount,
			concept.ConfidenceScore,
			concept.ReviewAttemptCount,
		))
	}
	if len(conceptState) == 0 {
		conceptState = []string{"- No durable concept evidence yet."}
	}
	openQuestions := make([]string, 0, len(state.OpenQuestions))
	for _, question := range state.OpenQuestions {
		openQuestions = append(openQuestions, "- "+truncate(question, 500))
	}
	if len(openQuestions) == 0 {
		openQuestions = []string{"- No unresolved claim questions recorded."}
	}
	return strings.Join([]string{
		"# Learner",
		"Interests: " + newsletter.Topic,
		"Level: " + newsletter.LearnerLevel,
		"Goal: " + newsletter.LearnerGoal,
		fmt.Sprintf("Available time: %d minutes", newsletter.LessonMinutes),
		lessonWordBudgetFor(newsletter.LessonMinutes).promptLine(),
		"",
		"# Previous lessons",
		strings.Join(prior, "\n"),
		"",
		"# Current learner evidence",
		"Recent difficulty signal: " + firstText(state.Difficulty, "not recorded"),
		"Recent relevance signal: " + firstText(state.Relevance, "not recorded"),
		"Recent recall confidence: " + firstText(state.RecallConfidence, "not recorded"),
		strings.Join(conceptState, "\n"),
		"Open questions about prior claims:",
		strings.Join(openQuestions, "\n"),
		"",
		"Build deliberately on prior learning when it is relevant. Do not merely repeat it.",
		"Use learner evidence conservatively: reinforce low-confidence concepts, avoid repeating strong concepts without a new connection, and adjust depth when difficulty feedback is available.",
	}, "\n")
}

func stageInstructions() map[string]string {
	headings := make([]string, len(requiredLessonSections))
	for index, heading := range requiredLessonSections {
		headings[index] = fmt.Sprintf("%q", "## "+heading)
	}
	return map[string]string{
		"curator":     "Choose one coherent, high-value learning theme from the supplied Source Items. Select three to five complementary Source Item identifiers; use fewer only when fewer exist. Return strict JSON only: {\"theme\":\"...\",\"rationale\":\"...\",\"selectedSourceIds\":[\"S1\",\"S2\",\"S3\"]}.",
		"blueprint":   "Design one lesson before prose is written for the learner's level, goal, time, and previous lessons. Return strict JSON only with \"lessonType\" set to exactly one of \"foundation\", \"update\", \"deep_dive\", \"synthesis\", \"application\", or \"review\"; string fields \"learningObjective\", \"centralMechanism\", \"workedExample\", \"misconception\", \"practicalExperiment\", \"continuityBridge\"; non-empty string arrays \"prerequisites\" and \"concepts\"; and one to five strings in \"suggestedNextConcepts\" describing useful conceptual directions after this lesson. Use foundation for a learner's first lesson; use the other types only when the learner history supports that purpose.",
		"researcher":  "Write a compact research brief serving the Learning Blueprint. Explain claims, mechanisms, conditions, and implications using only supplied Source Items. Cite identifiers like [S1], distinguish facts from inference, and identify disagreement or missing evidence.",
		"skeptic":     "Audit the research brief against the enriched Source Items and Learning Blueprint. Identify weak evidence, missing context, alternatives, edge cases, and unsupported claims. Preserve valid Source Item identifiers and give exact constraints for a trustworthy lesson.",
		"teacher":     "Write only the source-grounded lesson. Use these exact Markdown headings once and in order: " + strings.Join(headings, ", ") + ". Make every section substantive, explain the central mechanism step by step, cite factual claims, honor the lesson-body word budget in the learner context, and end with the Takeaway.",
		"examiner":    "Create source-grounded retrieval practice. Use \"## Retrieval practice\" with at least three numbered short-answer questions ending in question marks, \"## Application challenge\" with one realistic transfer task, then <details>, <summary>Answer key</summary>, complete numbered answers, and </details>.",
		"exploration": "Create explicitly synthetic AI Exploration with one novel analogy, one cross-domain deduction, one hypothetical scenario, and one experiment idea. Label uncertainty. Do not use [S#] citations or rewrite the source-grounded lesson.",
		"editor":      "Rewrite the source-grounded lesson and practice for precision, depth, and the supplied learner. Preserve every required lesson heading exactly once and in order, citations, the practice contract, and collapsed answer key. Keep the lesson body within the exact word budget in the learner context; practice and critique do not count toward it. Return strict JSON only with string fields \"lesson\", \"critique\", \"practice\"; \"exploration\" must be null; \"qualityNotes\" is an array of short strings.",
	}
}

func localDate(value time.Time, zone string) (string, error) {
	location, err := time.LoadLocation(zone)
	if err != nil {
		return "", fmt.Errorf("invalid Newsletter timezone %q: %w", zone, err)
	}
	return value.In(location).Format(time.DateOnly), nil
}

func mapSourceTitles(items []domain.SourceItem) []string {
	result := make([]string, len(items))
	for index := range items {
		result[index] = items[index].Title
	}
	return result
}

func mapSourceURLs(items []domain.SourceItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(firstText(item.CanonicalURL, item.URL))
		if value != "" {
			result = append(result, value)
		}
	}
	return compactUniqueFold(result)
}

func extractQuestions(markdown string) []string {
	var questions []string
	for _, line := range strings.Split(markdown, "\n") {
		match := questionPattern.FindStringSubmatch(line)
		if len(match) > 0 {
			questions = append(questions, match[2])
			if len(questions) == 5 {
				break
			}
		}
	}
	return questions
}

func blueprintConcepts(value domain.LearningBlueprint) []string {
	values := append([]string{value.CentralMechanism}, value.Prerequisites...)
	values = append(values, value.Concepts...)
	values = compactUniqueFold(values)
	for index := range values {
		values[index] = truncate(values[index], 300)
	}
	return values
}

func compactUniqueFold(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
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

func repeatsRecentLearningSignal(
	blueprint domain.LearningBlueprint,
	sources []domain.SourceItem,
	history []domain.LearningHistoryEntry,
) bool {
	if len(history) == 0 {
		return false
	}
	candidateSources := make([]string, 0, len(sources))
	candidateSourceURLs := make([]string, 0, len(sources))
	for _, item := range sources {
		candidateSources = append(candidateSources, item.Title)
		candidateSourceURLs = append(
			candidateSourceURLs,
			strings.TrimSpace(firstText(item.CanonicalURL, item.URL)),
		)
	}
	start := max(0, len(history)-5)
	for _, prior := range history[start:] {
		if tokenSimilarity(blueprint.LearningObjective, prior.LearningObjective) < 0.70 {
			continue
		}
		if sliceSimilarity(blueprintConcepts(blueprint), prior.Concepts) < 0.66 {
			continue
		}
		evidenceSimilarity := sliceSimilarity(candidateSources, prior.SourceTitles)
		if len(prior.SourceURLs) > 0 && len(candidateSourceURLs) > 0 {
			evidenceSimilarity = exactSliceOverlap(candidateSourceURLs, prior.SourceURLs)
		}
		if evidenceSimilarity >= 0.66 {
			return true
		}
	}
	return false
}

func exactSliceOverlap(left, right []string) float64 {
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range left {
		if normalized := strings.ToLower(strings.TrimSpace(value)); normalized != "" {
			leftSet[normalized] = struct{}{}
		}
	}
	for _, value := range right {
		if normalized := strings.ToLower(strings.TrimSpace(value)); normalized != "" {
			rightSet[normalized] = struct{}{}
		}
	}
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0
	}
	intersection := 0
	for value := range leftSet {
		if _, exists := rightSet[value]; exists {
			intersection++
		}
	}
	return float64(intersection) / float64(min(len(leftSet), len(rightSet)))
}

func tokenSimilarity(left, right string) float64 {
	return setSimilarity(similarityTokens([]string{left}), similarityTokens([]string{right}))
}

func sliceSimilarity(left, right []string) float64 {
	return setSimilarity(similarityTokens(left), similarityTokens(right))
}

func similarityTokens(values []string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, value := range values {
		for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
			return (r < 'a' || r > 'z') && (r < '0' || r > '9')
		}) {
			if len(token) > 2 && token != "the" && token != "and" && token != "with" &&
				token != "from" && token != "into" && token != "using" {
				result[token] = struct{}{}
			}
		}
	}
	return result
}

func setSimilarity(left, right map[string]struct{}) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]struct{}, len(left)+len(right))
	for token := range left {
		union[token] = struct{}{}
		if _, exists := right[token]; exists {
			intersection++
		}
	}
	for token := range right {
		union[token] = struct{}{}
	}
	return float64(intersection) / float64(len(union))
}

func stripMarkdown(value string) string {
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(value, "$1")
	value = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(value, "$1")
	value = markupPattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
}

func truncate(value string, maximum int) string {
	runes := []rune(value)
	if maximum <= 0 || len(runes) <= maximum {
		return value
	}
	suffix := "\n[truncated]"
	limit := maximum - len([]rune(suffix))
	if limit < 0 {
		return string([]rune(suffix)[:maximum])
	}
	return strings.TrimRight(string(runes[:limit]), " \t\r\n") + suffix
}

func prettyJSON(value any) string {
	body, _ := json.MarshalIndent(value, "", "  ")
	return string(body)
}

func safeRepairReason(err error) string {
	return truncate(spacePattern.ReplaceAllString(err.Error(), " "), 500)
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
