package dossier

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/VatsalP117/learnloom/internal/domain"
)

var requiredLessonSections = []string{
	"Learning objective",
	"Two-minute recall",
	"Why this matters",
	"Mental model",
	"How it works",
	"Worked example",
	"Common misconception",
	"Practical experiment",
	"Takeaway",
}

var (
	headingPattern  = regexp.MustCompile(`^(#{1,4})\s+(.+?)\s*$`)
	citationPattern = regexp.MustCompile(`\[S([1-9][0-9]*)\]`)
	questionPattern = regexp.MustCompile(`^\s*([0-9]+)\.\s+(.+\?)\s*$`)
	answerPattern   = regexp.MustCompile(`^\s*([0-9]+)\.\s+(.+?)\s*$`)
	htmlTagPattern  = regexp.MustCompile(`<[^>]+>`)
	markupPattern   = regexp.MustCompile("[`*_>#-]")
	spacePattern    = regexp.MustCompile(`\s+`)
)

func evaluateQuality(
	lesson, critique, practice string,
	exploration *string,
	sources []domain.SourceItem,
	blueprint domain.LearningBlueprint,
	historyCount int,
	wordBudget lessonWordBudget,
) (domain.QualityReport, error) {
	sections, order, counts := markdownSections(lesson)
	var missing []string
	var shallow []string
	for _, required := range requiredLessonSections {
		body, exists := sections[strings.ToLower(required)]
		if !exists {
			missing = append(missing, required)
			continue
		}
		if len([]rune(plainText(body))) < 30 || len(words(body)) < 5 {
			shallow = append(shallow, required)
		}
		if counts[strings.ToLower(required)] != 1 {
			return domain.QualityReport{}, fmt.Errorf(
				"editorial lesson must contain %q exactly once", required,
			)
		}
	}
	if len(missing) > 0 {
		return domain.QualityReport{}, fmt.Errorf(
			"editorial lesson is missing required sections: %s",
			strings.Join(missing, ", "),
		)
	}
	if !equalFoldSlice(order, requiredLessonSections) {
		return domain.QualityReport{}, errors.New(
			"editorial lesson must contain every required heading exactly once and in order",
		)
	}
	if len(shallow) > 0 {
		return domain.QualityReport{}, fmt.Errorf(
			"editorial lesson sections need substantive content: %s",
			strings.Join(shallow, ", "),
		)
	}
	lessonWords := markdownBodyWordCount(lesson)
	if lessonWords < wordBudget.minimum || lessonWords > wordBudget.maximum {
		return domain.QualityReport{}, fmt.Errorf(
			"editorial lesson has %d words; lesson body must contain %d to %d words to fit the learner's available time",
			lessonWords,
			wordBudget.minimum,
			wordBudget.maximum,
		)
	}

	known := make(map[string]struct{}, len(sources))
	for index, item := range sources {
		id := item.SourceID
		if id == "" {
			id = fmt.Sprintf("S%d", index+1)
		}
		known[id] = struct{}{}
	}
	cited := sourceIDs(lesson + "\n" + critique + "\n" + practice)
	lessonCited := sourceIDs(lesson)
	for _, id := range cited {
		if _, exists := known[id]; !exists {
			return domain.QualityReport{}, fmt.Errorf(
				"editorial output cites unknown Source Item %s", id,
			)
		}
	}
	requiredCitations := min(2, len(sources))
	if len(lessonCited) < requiredCitations {
		return domain.QualityReport{}, fmt.Errorf(
			"editorial lesson must cite at least %d Source Items", requiredCitations,
		)
	}

	retrieval := sectionBody(practice, "Retrieval practice")
	if retrieval == "" {
		return domain.QualityReport{}, errors.New(
			"editorial practice is missing a Retrieval practice section",
		)
	}
	type numberedText struct {
		number int
		text   string
	}
	var questions []numberedText
	seenQuestions := map[string]struct{}{}
	for _, line := range strings.Split(retrieval, "\n") {
		match := questionPattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		normalized := strings.ToLower(plainText(match[2]))
		if _, duplicate := seenQuestions[normalized]; duplicate {
			return domain.QualityReport{}, errors.New(
				"retrieval questions must be distinct",
			)
		}
		seenQuestions[normalized] = struct{}{}
		questions = append(questions, numberedText{number: number, text: match[2]})
	}
	if len(questions) < 3 {
		return domain.QualityReport{}, errors.New(
			"editorial practice must contain at least three retrieval questions",
		)
	}
	for index, question := range questions {
		if question.number != index+1 || len(words(question.text)) < 4 {
			return domain.QualityReport{}, errors.New(
				"retrieval questions must be substantive and sequentially numbered",
			)
		}
	}
	challenge := sectionBody(practice, "Application challenge")
	if len([]rune(plainText(challenge))) < 40 || len(words(challenge)) < 7 {
		return domain.QualityReport{}, errors.New(
			"editorial Application challenge must be substantive",
		)
	}
	answerKey := betweenFold(practice, "<summary>Answer key</summary>", "</details>")
	if answerKey == "" {
		return domain.QualityReport{}, errors.New(
			"editorial practice is missing a collapsed answer key",
		)
	}
	answers := map[int]string{}
	for _, line := range strings.Split(answerKey, "\n") {
		match := answerPattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		answers[number] = match[2]
	}
	if len(answers) != len(questions) {
		return domain.QualityReport{}, errors.New(
			"answer key must answer every retrieval question",
		)
	}
	for _, question := range questions {
		answer := plainText(answers[question.number])
		if len([]rune(answer)) < 15 || len(words(answer)) < 3 ||
			strings.HasSuffix(answer, "?") {
			return domain.QualityReport{}, errors.New(
				"answer key must provide substantive numbered answers",
			)
		}
	}
	if exploration != nil && citationPattern.MatchString(*exploration) {
		return domain.QualityReport{}, errors.New(
			"AI Exploration must not use Source Item citation markers",
		)
	}

	enriched := 0
	for _, item := range sources {
		if item.ContentSource == "article" {
			enriched++
		}
	}
	coverage := 0.0
	if len(known) > 0 {
		coverage = float64(len(lessonCited)) / float64(len(known))
	}
	checks := map[string]bool{
		"requiredLessonSections":    true,
		"substantiveLessonSections": true,
		"lessonTimeFit":             true,
		"sourceGrounding":           len(lessonCited) >= requiredCitations,
		"validCitationIdentifiers":  true,
		"retrievalPractice":         len(questions) >= 3,
		"applicationChallenge":      true,
		"collapsedAnswerKey":        true,
		"continuity":                historyCount == 0 || strings.TrimSpace(blueprint.ContinuityBridge) != "",
		"explorationBoundary":       exploration == nil || !citationPattern.MatchString(*exploration),
	}
	score := 20 + min(25, int(coverage*25)) + 5 + 15 + 10 + 10
	if checks["continuity"] {
		score += 5
	}
	if checks["explorationBoundary"] {
		score += 5
	}
	if enriched > 0 {
		score += 5
	} else {
		score += 2
	}
	return domain.QualityReport{
		Version: 1,
		Score:   min(score, 100),
		Checks:  checks,
		Metrics: map[string]int{
			"selectedSources":    len(sources),
			"enrichedSources":    enriched,
			"citedSources":       len(lessonCited),
			"lessonWords":        lessonWords,
			"lessonWordsMinimum": wordBudget.minimum,
			"lessonWordsMaximum": wordBudget.maximum,
			"retrievalQuestions": len(questions),
			"answeredQuestions":  len(answers),
		},
	}, nil
}

type lessonWordBudget struct {
	minimum int
	maximum int
}

func lessonWordBudgetFor(minutes int) lessonWordBudget {
	if minutes <= 0 {
		minutes = 20
	}
	return lessonWordBudget{
		minimum: min(max(minutes*30, 300), 1800),
		maximum: min(max(minutes*90, 700), 3200),
	}
}

func (b lessonWordBudget) promptLine() string {
	return fmt.Sprintf(
		"Lesson body budget: %d-%d words (practice, skeptical review, and sources are excluded).",
		b.minimum,
		b.maximum,
	)
}

func markdownBodyWordCount(markdown string) int {
	var body strings.Builder
	for _, line := range strings.Split(markdown, "\n") {
		if headingPattern.MatchString(line) {
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	return len(words(body.String()))
}

func markdownSections(markdown string) (map[string]string, []string, map[string]int) {
	sections := map[string]string{}
	counts := map[string]int{}
	var order []string
	var heading string
	var body strings.Builder
	flush := func() {
		if heading == "" {
			return
		}
		sections[strings.ToLower(heading)] = strings.TrimSpace(body.String())
		body.Reset()
	}
	for _, line := range strings.Split(markdown, "\n") {
		match := headingPattern.FindStringSubmatch(line)
		if len(match) > 0 {
			flush()
			heading = strings.TrimSpace(match[2])
			lower := strings.ToLower(heading)
			counts[lower]++
			if containsFold(requiredLessonSections, heading) {
				order = append(order, heading)
			}
			continue
		}
		if heading != "" {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	flush()
	return sections, order, counts
}

func sectionBody(markdown, wanted string) string {
	sections, _, _ := markdownSections(markdown)
	return sections[strings.ToLower(wanted)]
}

func sourceIDs(value string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, match := range citationPattern.FindAllStringSubmatch(value, -1) {
		id := "S" + match[1]
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func plainText(value string) string {
	value = citationPattern.ReplaceAllString(value, "")
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = markupPattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
}

func words(value string) []string {
	return strings.FieldsFunc(plainText(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\'' && r != '’' && r != '-'
	})
}

func containsFold(values []string, wanted string) bool {
	return slices.ContainsFunc(values, func(value string) bool {
		return strings.EqualFold(value, wanted)
	})
}

func equalFoldSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !strings.EqualFold(left[index], right[index]) {
			return false
		}
	}
	return true
}

func betweenFold(value, start, end string) string {
	lower := strings.ToLower(value)
	startIndex := strings.Index(lower, strings.ToLower(start))
	if startIndex < 0 {
		return ""
	}
	startIndex += len(start)
	endIndex := strings.Index(lower[startIndex:], strings.ToLower(end))
	if endIndex < 0 {
		return ""
	}
	return strings.TrimSpace(value[startIndex : startIndex+endIndex])
}
