package dossier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
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
	Newsletter domain.Newsletter
	History    []domain.LearningHistoryEntry
	Now        time.Time
	OnStage    func(string)
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
	if len(request.Newsletter.Sources) == 0 {
		return GenerateResult{}, errors.New("Newsletter requires at least one Source Item definition")
	}
	now := request.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	items, warnings, err := g.sources.Fetch(
		ctx,
		request.Newsletter.Sources,
		g.cfg.MaxItems,
	)
	if err != nil {
		return GenerateResult{}, err
	}
	learnerContext := g.learnerContext(request.Newsletter, request.History)
	candidateCharacters := max(300, min(
		g.cfg.MaxItemCharacters,
		int(float64(g.cfg.MaxIntermediateCharacters)*0.7)/len(items),
	))
	candidateBundle := formatSourceBundle(items, candidateCharacters)

	curation, err := runStructured(
		ctx,
		g.model,
		"curator",
		stageInstructions()["curator"],
		learnerContext+"\n\n# Candidate sources\n\n"+candidateBundle,
		request.OnStage,
		func(value domain.Curation) error { return validateCuration(value, len(items)) },
	)
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
	enriched, err := g.sources.Enrich(ctx, curated)
	if err != nil {
		return GenerateResult{}, err
	}
	sourceCharacters := max(1000, min(
		g.cfg.MaxArticleCharacters,
		int(float64(g.cfg.MaxIntermediateCharacters)*0.45)/len(enriched),
	))
	sourceBundle := formatSourceBundle(enriched, sourceCharacters)

	blueprint, err := runStructured(
		ctx,
		g.model,
		"blueprint",
		stageInstructions()["blueprint"],
		fitSections(g.cfg.MaxIntermediateCharacters, []weightedSection{
			{"Learner context", learnerContext, 2},
			{"Curated theme", prettyJSON(curation), 1},
			{"Enriched sources", sourceBundle, 5},
		}),
		request.OnStage,
		validateBlueprint,
	)
	if err != nil {
		return GenerateResult{}, err
	}
	blueprintText := prettyJSON(blueprint)
	research, err := g.runStage(ctx, "researcher", fitSections(
		g.cfg.MaxIntermediateCharacters,
		[]weightedSection{
			{"Learner context", learnerContext, 1},
			{"Learning blueprint", blueprintText, 2},
			{"Enriched sources", sourceBundle, 6},
		},
	), request.OnStage)
	if err != nil {
		return GenerateResult{}, err
	}
	critique, err := g.runStage(ctx, "skeptic", fitSections(
		g.cfg.MaxIntermediateCharacters,
		[]weightedSection{
			{"Learning blueprint", blueprintText, 1},
			{"Enriched sources", sourceBundle, 5},
			{"Research brief", research, 3},
		},
	), request.OnStage)
	if err != nil {
		return GenerateResult{}, err
	}
	lesson, err := g.runStage(ctx, "teacher", fitSections(
		g.cfg.MaxIntermediateCharacters,
		[]weightedSection{
			{"Learner context", learnerContext, 1},
			{"Learning blueprint", blueprintText, 2},
			{"Enriched sources", sourceBundle, 3},
			{"Research brief", research, 3},
			{"Skeptical review", critique, 2},
		},
	), request.OnStage)
	if err != nil {
		return GenerateResult{}, err
	}
	practice, err := g.runStage(ctx, "examiner", fitSections(
		g.cfg.MaxIntermediateCharacters,
		[]weightedSection{
			{"Learner context", learnerContext, 1},
			{"Learning blueprint", blueprintText, 2},
			{"Source-grounded lesson", lesson, 6},
		},
	), request.OnStage)
	if err != nil {
		return GenerateResult{}, err
	}
	var exploration *string
	if request.Newsletter.AIExplorationEnabled {
		value, err := g.runStage(ctx, "exploration", fitSections(
			g.cfg.MaxIntermediateCharacters,
			[]weightedSection{
				{"Learner context", learnerContext, 1},
				{"Learning blueprint", blueprintText, 2},
				{"Source-grounded lesson", lesson, 5},
				{"Skeptical review", critique, 2},
			},
		), request.OnStage)
		if err != nil {
			return GenerateResult{}, err
		}
		exploration = &value
	}

	editorial, err := runStructured(
		ctx,
		g.model,
		"editor",
		stageInstructions()["editor"],
		fitSections(g.cfg.MaxIntermediateCharacters, []weightedSection{
			{"Learning blueprint", blueprintText, 2},
			{"Enriched sources", sourceBundle, 3},
			{"Draft lesson", lesson, 5},
			{"Skeptical review", critique, 2},
			{"Draft practice", practice, 3},
		}),
		request.OnStage,
		func(value editorialOutput) error {
			if err := value.validate(); err != nil {
				return err
			}
			_, err := evaluateQuality(
				value.Lesson,
				value.Critique,
				value.Practice,
				nil,
				enriched,
				blueprint,
				len(request.History),
			)
			return err
		},
	)
	if err != nil {
		return GenerateResult{}, err
	}
	quality, err := evaluateQuality(
		editorial.Lesson,
		editorial.Critique,
		editorial.Practice,
		exploration,
		enriched,
		blueprint,
		len(request.History),
	)
	if err != nil {
		return GenerateResult{}, err
	}
	quality.EditorNotes = slices.Clone(editorial.QualityNotes)
	date, err := localDate(now, request.Newsletter.TimeZone)
	if err != nil {
		return GenerateResult{}, err
	}
	dossier := domain.Dossier{
		Version:     2,
		ProfileID:   request.Newsletter.ID,
		Date:        date,
		Title:       curation.Theme,
		GeneratedAt: now.UTC(),
		Model:       g.cfg.ModelName,
		Curation:    curation,
		Blueprint:   blueprint,
		Lesson:      editorial.Lesson,
		Critique:    editorial.Critique,
		Practice:    editorial.Practice,
		Exploration: exploration,
		Quality:     quality,
		Sources:     enriched,
	}
	markdown := RenderMarkdown(dossier)
	html := RenderHTML(dossier, "")
	history := domain.LearningHistoryEntry{
		Date:              date,
		GeneratedAt:       now.UTC(),
		SourceTitles:      mapSourceTitles(enriched),
		LessonSummary:     truncate(stripMarkdown(editorial.Lesson), 800),
		RecallQuestions:   extractQuestions(editorial.Practice),
		LearningObjective: blueprint.LearningObjective,
		Concepts:          blueprintConcepts(blueprint),
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

func (g *Generator) runStage(
	ctx context.Context,
	stage, input string,
	onStage func(string),
) (string, error) {
	if onStage != nil {
		onStage(stage)
	}
	output, err := g.model.Complete(ctx, CompletionRequest{
		Stage:       stage,
		Instruction: stageInstructions()[stage],
		Input:       input,
	})
	if err != nil {
		return "", fmt.Errorf("%s stage: %w", stage, err)
	}
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("%s stage returned empty output", stage)
	}
	return strings.TrimSpace(output), nil
}

func runStructured[T any](
	ctx context.Context,
	model Completer,
	stage, instruction, input string,
	onStage func(string),
	validate func(T) error,
) (T, error) {
	var zero T
	var repairReason string
	for attempt := 0; attempt < 2; attempt++ {
		stageInput := input
		if repairReason != "" {
			stageInput += "\n\n# Contract repair\n\nYour previous response was rejected: " +
				repairReason +
				"\nReturn a corrected response in the exact requested format."
		}
		if onStage != nil {
			onStage(stage)
		}
		output, err := model.Complete(ctx, CompletionRequest{
			Stage: stage, Instruction: instruction, Input: stageInput,
		})
		if err != nil {
			return zero, fmt.Errorf("%s stage: %w", stage, err)
		}
		var value T
		if err := decodeStructured(output, &value); err == nil {
			if err := validate(value); err == nil {
				return value, nil
			} else {
				repairReason = safeRepairReason(err)
				continue
			}
		} else {
			repairReason = safeRepairReason(err)
		}
	}
	return zero, fmt.Errorf("%s stage could not satisfy its output contract: %s", stage, repairReason)
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
		prior = append(prior, fmt.Sprintf(
			"- %s: %s\n  Recall: %s",
			entry.Date,
			entry.LessonSummary,
			firstText(strings.Join(recall, " | "), "none recorded"),
		))
	}
	if len(prior) == 0 {
		prior = []string{"- No previous lessons yet."}
	}
	return strings.Join([]string{
		"# Learner",
		"Interests: " + newsletter.Topic,
		"Level: " + newsletter.LearnerLevel,
		"Goal: " + newsletter.LearnerGoal,
		fmt.Sprintf("Available time: %d minutes", newsletter.LessonMinutes),
		"",
		"# Previous lessons",
		strings.Join(prior, "\n"),
		"",
		"Build deliberately on prior learning when it is relevant. Do not merely repeat it.",
	}, "\n")
}

func stageInstructions() map[string]string {
	headings := make([]string, len(requiredLessonSections))
	for index, heading := range requiredLessonSections {
		headings[index] = fmt.Sprintf("%q", "## "+heading)
	}
	return map[string]string{
		"curator":     "Choose one coherent, high-value learning theme from the supplied Source Items. Select three to five complementary Source Item identifiers; use fewer only when fewer exist. Return strict JSON only: {\"theme\":\"...\",\"rationale\":\"...\",\"selectedSourceIds\":[\"S1\",\"S2\",\"S3\"]}.",
		"blueprint":   "Design one lesson before prose is written for the learner's level, goal, time, and previous lessons. Return strict JSON only with string fields \"learningObjective\", \"centralMechanism\", \"workedExample\", \"misconception\", \"practicalExperiment\", \"continuityBridge\", plus a non-empty string array \"prerequisites\".",
		"researcher":  "Write a compact research brief serving the Learning Blueprint. Explain claims, mechanisms, conditions, and implications using only supplied Source Items. Cite identifiers like [S1], distinguish facts from inference, and identify disagreement or missing evidence.",
		"skeptic":     "Audit the research brief against the enriched Source Items and Learning Blueprint. Identify weak evidence, missing context, alternatives, edge cases, and unsupported claims. Preserve valid Source Item identifiers and give exact constraints for a trustworthy lesson.",
		"teacher":     "Write only the source-grounded lesson. Use these exact Markdown headings once and in order: " + strings.Join(headings, ", ") + ". Make every section substantive, explain the central mechanism step by step, cite factual claims, and end with the Takeaway.",
		"examiner":    "Create source-grounded retrieval practice. Use \"## Retrieval practice\" with at least three numbered short-answer questions ending in question marks, \"## Application challenge\" with one realistic transfer task, then <details>, <summary>Answer key</summary>, complete numbered answers, and </details>.",
		"exploration": "Create explicitly synthetic AI Exploration with one novel analogy, one cross-domain deduction, one hypothetical scenario, and one experiment idea. Label uncertainty. Do not use [S#] citations or rewrite the source-grounded lesson.",
		"editor":      "Rewrite the source-grounded lesson and practice for precision and depth. Preserve every required lesson heading exactly once and in order, citations, the practice contract, and collapsed answer key. Return strict JSON only with string fields \"lesson\", \"critique\", \"practice\"; \"exploration\" must be null; \"qualityNotes\" is an array of short strings.",
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
	for index := range values {
		values[index] = truncate(values[index], 300)
	}
	return values
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
