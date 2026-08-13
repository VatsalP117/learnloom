package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/failure"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

type discoveryCandidate struct {
	SearchCandidate
	Query       string
	PlannedRole domain.SourceRole
	Role        domain.SourceRole
	Score       int
	Components  domain.SourceScoreComponents
	Reason      string
	URL         string
	Domain      string
}

type discoveryQuery struct {
	Query string            `json:"query"`
	Role  domain.SourceRole `json:"role"`
}

const sourceRankingVersion = "source-rank-v2"

var errDiscoveryUnavailable = errors.New("source discovery is temporarily unavailable")

func (svc *Service) shouldDiscover(mode domain.SourceMode, selected []preparedEvidence) bool {
	if mode == domain.SourceModeProvided {
		return false
	}
	if mode == domain.SourceModeHybrid {
		return !svc.hasHardMinimum(selected)
	}
	return len(selected) < svc.cfg.TargetUsableItems
}

func (svc *Service) discover(
	ctx context.Context,
	newsletter domain.Newsletter,
	issueID string,
	existingSpecs []domain.SourceSpec,
) ([]preparedEvidence, []string, error) {
	if !svc.cfg.DiscoveryEnabled || svc.searcher == nil {
		return nil, nil, errors.New("source discovery is disabled or unavailable")
	}
	queries := discoveryQueries(newsletter.Topic)
	if len(queries) > svc.cfg.DiscoveryMaxQueries {
		queries = queries[:svc.cfg.DiscoveryMaxQueries]
	}
	queryJSON, _ := json.Marshal(queries)
	started := time.Now().UTC()
	run := domain.DiscoveryRun{
		ID:           uuid.NewString(),
		NewsletterID: newsletter.ID,
		IssueID:      issueID,
		Reason:       discoveryReason(existingSpecs),
		State:        "running",
		QueryBundle:  string(queryJSON),
		StartedAt:    &started,
	}
	if err := svc.repo.CreateDiscoveryRun(ctx, run); err != nil {
		return nil, nil, fmt.Errorf("record discovery run: %w", err)
	}

	searches := parallelMapOrdered(
		ctx,
		queries,
		svc.cfg.MaxConcurrency,
		func(ctx context.Context, query discoveryQuery) ([]SearchCandidate, error) {
			return svc.searcher.Search(ctx, SearchRequest{
				Query: query.Query, Language: "all", Category: "general", Page: 1,
			})
		},
	)
	var warnings []string
	for _, outcome := range searches {
		if outcome.err != nil {
			warnings = append(warnings, safeError(outcome.err))
		}
	}
	raw := interleaveDiscoveryCandidates(queries, searches, svc.cfg.DiscoveryMaxCandidates)
	run.ReturnedCandidates = len(raw)
	ranked, rejected := rankDiscoveryCandidates(
		newsletter.Topic,
		raw,
		existingSpecs,
		svc.cfg.DiscoveryMaxCandidates,
		svc.cfg.DiscoveryMaxActive,
	)
	run.RejectedCandidates = rejected

	var evidence []preparedEvidence
	for _, candidate := range ranked {
		spec, err := svc.repo.UpsertDiscoveredSourceSpec(ctx, domain.SourceSpec{
			ID:              uuid.NewString(),
			NewsletterID:    newsletter.ID,
			Origin:          domain.SourceOriginDiscovered,
			State:           domain.SourceStateCandidate,
			DisplayName:     candidate.Title,
			InputURL:        candidate.URL,
			CanonicalURL:    candidate.URL,
			Scope:           domain.SourceScopeExact,
			ItemLimit:       8,
			DiscoveryReason: candidate.Reason,
			DiscoveryQuery:  candidate.Query,
			RankScore:       candidate.Score,
			Role:            candidate.Role,
			RankingVersion:  sourceRankingVersion,
			ScoreComponents: candidate.Components,
		})
		if err != nil {
			warnings = append(warnings, safeError(err))
			continue
		}
		if spec.Origin != domain.SourceOriginDiscovered {
			continue
		}
		resolved, err := svc.resolveAndSnapshot(ctx, spec)
		if err != nil {
			_ = svc.repo.SetSourceSpecState(
				ctx,
				spec.ID,
				domain.SourceStateUnhealthy,
				spec.Kind,
			)
			warnings = append(warnings, fmt.Sprintf("%s: %s", sourceName(spec), safeError(err)))
			continue
		}
		run.ResolvedCandidates++
		resolvedKind := spec.Kind
		if endpoint, ok, endpointErr := svc.repo.GetSourceEndpoint(ctx, spec.ID); endpointErr == nil && ok {
			resolvedKind = endpoint.Kind
		}
		if err := svc.repo.SetSourceSpecState(
			ctx,
			spec.ID,
			domain.SourceStateActive,
			resolvedKind,
		); err != nil {
			return nil, warnings, svc.finishDiscoveryRun(
				ctx,
				run,
				"failed",
				fmt.Errorf("activate discovered source: %w", err),
			)
		}
		run.ActivatedCandidates++
		evidence = append(evidence, resolved...)
		if run.ActivatedCandidates >= svc.cfg.DiscoveryMaxActive {
			break
		}
	}

	state := "completed"
	var runErr error
	if len(warnings) > 0 {
		state = "degraded"
	}
	if run.ActivatedCandidates == 0 && len(raw) == 0 && len(warnings) > 0 {
		state = "failed"
		runErr = failure.New(
			failure.CodeSourceDiscoveryUnavailable,
			failure.CategoryInfrastructure,
			"source_discovery",
			true,
			failure.PublicDelayed,
			fmt.Errorf("%w: %s", errDiscoveryUnavailable, warningsToErr(warnings)),
		)
	}
	if err := svc.finishDiscoveryRun(ctx, run, state, runErr); err != nil {
		return nil, warnings, err
	}
	return evidence, warnings, runErr
}

func interleaveDiscoveryCandidates(
	queries []discoveryQuery,
	outcomes []parallelOutcome[[]SearchCandidate],
	limit int,
) []discoveryCandidate {
	if limit < 1 || len(queries) == 0 || len(outcomes) == 0 {
		return nil
	}
	result := make([]discoveryCandidate, 0, limit)
	for offset := 0; len(result) < limit; offset++ {
		appended := false
		for index, outcome := range outcomes {
			if index >= len(queries) || outcome.err != nil || offset >= len(outcome.value) {
				continue
			}
			candidate := outcome.value[offset]
			result = append(result, discoveryCandidate{
				SearchCandidate: candidate,
				Query:           queries[index].Query,
				PlannedRole:     queries[index].Role,
			})
			appended = true
			if len(result) >= limit {
				break
			}
		}
		if !appended {
			break
		}
	}
	return result
}

func (svc *Service) finishDiscoveryRun(
	ctx context.Context,
	run domain.DiscoveryRun,
	state string,
	runErr error,
) error {
	completed := time.Now().UTC()
	run.State = state
	run.CompletedAt = &completed
	if runErr != nil {
		run.Error = safeError(runErr)
	}
	if err := svc.repo.CompleteDiscoveryRun(ctx, run); err != nil {
		return fmt.Errorf("complete discovery run: %w", err)
	}
	return runErr
}

func discoveryQueries(topic string) []discoveryQuery {
	topic = strings.TrimSpace(topic)
	return []discoveryQuery{
		{Query: topic + " official documentation specification reference", Role: domain.SourceRoleOfficialPrimary},
		{Query: topic + " research paper systematic review", Role: domain.SourceRoleResearch},
		{Query: topic + " implementation guide case study", Role: domain.SourceRolePractitioner},
		{Query: topic + " limitations risks failure modes critique", Role: domain.SourceRoleCounterweight},
		{Query: topic + " industry analysis reporting context", Role: domain.SourceRoleReporting},
	}
}

func discoveryReason(specs []domain.SourceSpec) string {
	if len(specs) == 0 {
		return "initial"
	}
	return "insufficient_items"
}

func rankDiscoveryCandidates(
	topic string,
	raw []discoveryCandidate,
	existingSpecs []domain.SourceSpec,
	maxCandidates, maxActive int,
) ([]discoveryCandidate, int) {
	existing := make(map[string]struct{}, len(existingSpecs))
	for _, spec := range existingSpecs {
		existing[normalizeCandidateURL(spec.CanonicalURL)] = struct{}{}
		existing[normalizeCandidateURL(spec.InputURL)] = struct{}{}
	}
	topicTokens := tokenSet(topic)
	seen := make(map[string]struct{})
	var candidates []discoveryCandidate
	rejected := 0
	for _, candidate := range raw {
		normalized, domainName, err := gateCandidateURL(candidate.SearchCandidate.URL)
		if err != nil {
			rejected++
			continue
		}
		if _, ok := existing[normalized]; ok {
			rejected++
			continue
		}
		if _, ok := seen[normalized]; ok {
			rejected++
			continue
		}
		seen[normalized] = struct{}{}
		candidate.URL = normalized
		candidate.Domain = domainName
		candidate.Role = classifyCandidateRole(candidate)
		candidate.Components = scoreCandidate(candidate, topicTokens, time.Now().UTC())
		candidate.Score = candidate.Components.Total()
		candidate.Reason = candidateReason(candidate)
		candidates = append(candidates, candidate)
		if len(candidates) >= maxCandidates {
			break
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].URL < candidates[j].URL
		}
		return candidates[i].Score > candidates[j].Score
	})
	perDomain := make(map[string]int)
	selected := make([]discoveryCandidate, 0, min(maxActive, len(candidates)))
	domainLimit := max(1, maxActive/4)
	selectedURLs := make(map[string]struct{})
	for _, role := range []domain.SourceRole{
		domain.SourceRoleOfficialPrimary,
		domain.SourceRoleResearch,
		domain.SourceRolePractitioner,
		domain.SourceRoleCounterweight,
	} {
		for _, candidate := range candidates {
			if candidate.Role != role || !candidateEligible(candidate) || perDomain[candidate.Domain] >= domainLimit {
				continue
			}
			if perDomain[candidate.Domain] == 0 {
				candidate.Components.Independence = 10
			} else {
				candidate.Components.Independence = 2
			}
			candidate.Score = candidate.Components.Total()
			selected = append(selected, candidate)
			selectedURLs[candidate.URL] = struct{}{}
			perDomain[candidate.Domain]++
			break
		}
	}
	for _, candidate := range candidates {
		if len(selected) >= maxActive {
			break
		}
		if _, exists := selectedURLs[candidate.URL]; exists {
			continue
		}
		if !candidateEligible(candidate) || perDomain[candidate.Domain] >= domainLimit {
			rejected++
			continue
		}
		if perDomain[candidate.Domain] == 0 {
			candidate.Components.Independence = 10
		} else {
			candidate.Components.Independence = 2
		}
		candidate.Score = candidate.Components.Total()
		perDomain[candidate.Domain]++
		selected = append(selected, candidate)
	}
	return selected, rejected
}

func candidateEligible(candidate discoveryCandidate) bool {
	if candidate.Score <= 0 || candidate.Components.Relevance < 10 ||
		candidate.Components.Accessibility < 5 {
		return false
	}
	switch candidate.Role {
	case domain.SourceRoleOfficialPrimary:
		return candidate.Components.Authority >= 18 && candidate.Components.Primaryness >= 14
	case domain.SourceRoleResearch:
		return candidate.Components.Authority >= 6
	case domain.SourceRolePractitioner:
		return candidate.Components.Usefulness >= 9
	case domain.SourceRoleCounterweight:
		return candidate.Components.Counterweight >= 12
	case domain.SourceRoleReporting:
		return candidate.Components.Usefulness >= 9
	default:
		return false
	}
}

func gateCandidateURL(raw string) (string, string, error) {
	parsed, err := validateWebURL(raw)
	if err != nil {
		return "", "", err
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "t.co", "bit.ly", "tinyurl.com", "goo.gl", "facebook.com",
		"www.facebook.com", "instagram.com", "www.instagram.com":
		return "", "", errors.New("candidate host is not accepted")
	}
	path := strings.ToLower(parsed.EscapedPath())
	if strings.Contains(path, "/search") || strings.Contains(path, "/results") {
		return "", "", errors.New("search result pages are not accepted")
	}
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()
	registrable, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		registrable = host
	}
	return parsed.String(), registrable, nil
}

func normalizeCandidateURL(raw string) string {
	normalized, _, err := gateCandidateURL(raw)
	if err != nil {
		return ""
	}
	return normalized
}

func scoreCandidate(
	candidate discoveryCandidate,
	topicTokens map[string]struct{},
	now time.Time,
) domain.SourceScoreComponents {
	components := domain.SourceScoreComponents{
		SearchRank: max(0, 11-candidate.Rank),
	}
	searchable := tokenSet(candidate.Title + " " + candidate.Snippet)
	overlap := 0
	for token := range topicTokens {
		if _, ok := searchable[token]; ok {
			overlap++
		}
	}
	components.Relevance = min(30, overlap*10)
	if overlap == 0 {
		components.Negative -= 25
	}
	lower := strings.ToLower(candidate.Title + " " + candidate.Snippet + " " + candidate.URL)
	switch candidate.Role {
	case domain.SourceRoleOfficialPrimary:
		if containsAny(lower, "official", "documentation", "reference", "specification", "standard") {
			components.Authority += 7
			components.Primaryness += 6
		}
		if officialHostSignal(candidate.URL) {
			components.Authority += 18
			components.Primaryness += 14
		}
	case domain.SourceRoleResearch:
		if researchHostSignal(candidate.URL) {
			components.Authority += 22
			components.Primaryness += 10
		}
		if containsAny(lower, "paper", "study", "systematic review", "proceedings", "journal") {
			components.Authority += 6
		}
	case domain.SourceRolePractitioner:
		if containsAny(lower, "guide", "implementation", "case study", "example", "tutorial") {
			components.Usefulness += 14
		}
		if strings.Contains(lower, "github.com/") {
			components.Primaryness += 8
		}
	case domain.SourceRoleReporting:
		if containsAny(lower, "analysis", "report", "context", "interview") {
			components.Usefulness += 9
		}
	case domain.SourceRoleCounterweight:
		if containsAny(lower, "limitation", "risk", "failure", "critique", "trade-off", "tradeoff") {
			components.Counterweight += 18
		}
	}
	if candidate.PublishedAt != nil {
		age := now.Sub(candidate.PublishedAt.UTC())
		switch {
		case age < 0:
			components.Negative -= 5
		case age <= 365*24*time.Hour:
			components.Recency += 10
		case age <= 3*365*24*time.Hour:
			components.Recency += 5
		case candidate.Role != domain.SourceRoleOfficialPrimary &&
			candidate.Role != domain.SourceRoleResearch:
			components.Negative -= 8
		}
	}
	if strings.HasPrefix(candidate.URL, "https://") {
		components.Accessibility += 5
	}
	if parsed, err := url.Parse(candidate.URL); err == nil && parsed.Path != "" && parsed.Path != "/" {
		components.Accessibility += 3
	}
	if len([]rune(strings.TrimSpace(candidate.Snippet))) < 40 {
		components.Negative -= 8
	}
	if containsAny(lower, "top 10", "best tools", "ultimate guide", "you won't believe") {
		components.Negative -= 18
	}
	if parsed, err := url.Parse(candidate.URL); err == nil &&
		containsAny(strings.ToLower(parsed.Path), "/tag/", "/category/", "/author/", "/archive/") {
		components.Negative -= 10
	}
	return components
}

func candidateReason(candidate discoveryCandidate) string {
	switch candidate.Role {
	case domain.SourceRoleOfficialPrimary:
		return "Primary reference for definitions, specifications, or maintained documentation"
	case domain.SourceRoleResearch:
		return "Research evidence or a review of the available evidence"
	case domain.SourceRolePractitioner:
		return "Practical explanation with implementation detail or worked examples"
	case domain.SourceRoleReporting:
		return "Independent reporting or context around the topic"
	case domain.SourceRoleCounterweight:
		return "Counterweight covering limitations, risks, or failure modes"
	default:
		return "Adds relevant independent coverage"
	}
}

func classifyCandidateRole(candidate discoveryCandidate) domain.SourceRole {
	lower := strings.ToLower(candidate.Title + " " + candidate.Snippet + " " + candidate.URL)
	switch {
	case researchHostSignal(candidate.URL) || containsAny(lower, "systematic review", "research paper", "journal", "proceedings"):
		return domain.SourceRoleResearch
	case containsAny(lower, "limitation", "risks", "failure modes", "critique", "trade-off", "tradeoff"):
		return domain.SourceRoleCounterweight
	case officialHostSignal(candidate.URL) && containsAny(lower, "documentation", "reference", "specification", "standard", "official"):
		return domain.SourceRoleOfficialPrimary
	case containsAny(lower, "implementation guide", "case study", "tutorial", "worked example"):
		return domain.SourceRolePractitioner
	case candidate.PlannedRole != "":
		return candidate.PlannedRole
	default:
		return domain.SourceRoleReporting
	}
}

func officialHostSignal(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(parsed.EscapedPath())
	return strings.HasPrefix(host, "docs.") || strings.HasPrefix(host, "developer.") ||
		strings.HasPrefix(host, "api.") || strings.HasSuffix(host, ".gov") ||
		strings.HasSuffix(host, ".gov.uk") || containsAny(
		path,
		"/docs/",
		"/documentation/",
		"/reference/",
		"/specification/",
		"/standards/",
	)
}

func researchHostSignal(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "doi.org" || host == "arxiv.org" || host == "openreview.net" ||
		strings.HasSuffix(host, ".ncbi.nlm.nih.gov") || host == "dl.acm.org" ||
		host == "ieeexplore.ieee.org" || strings.HasSuffix(host, ".edu")
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func tokenSet(value string) map[string]struct{} {
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if len([]rune(field)) > 1 {
			result[field] = struct{}{}
		}
	}
	return result
}
