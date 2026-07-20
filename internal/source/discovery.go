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
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

type discoveryCandidate struct {
	SearchCandidate
	Query  string
	Score  int
	Reason string
	URL    string
	Domain string
}

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

	var raw []discoveryCandidate
	var warnings []string
	for _, query := range queries {
		results, err := svc.searcher.Search(ctx, SearchRequest{
			Query: query, Language: "all", Category: "general", Page: 1,
		})
		if err != nil {
			warnings = append(warnings, safeError(err))
			continue
		}
		for _, result := range results {
			raw = append(raw, discoveryCandidate{
				SearchCandidate: result,
				Query:           query,
			})
			if len(raw) >= svc.cfg.DiscoveryMaxCandidates {
				break
			}
		}
		if len(raw) >= svc.cfg.DiscoveryMaxCandidates {
			break
		}
	}
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
	if run.ActivatedCandidates == 0 {
		state = "failed"
		runErr = errors.New("discovery returned no usable sources")
	}
	if err := svc.finishDiscoveryRun(ctx, run, state, runErr); err != nil {
		return nil, warnings, err
	}
	return evidence, warnings, runErr
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

func discoveryQueries(topic string) []string {
	topic = strings.TrimSpace(topic)
	return []string{
		topic + " official documentation",
		topic + " tutorial guide examples",
		topic + " research paper review",
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
		candidate.Score = scoreCandidate(candidate, topicTokens)
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
	for _, candidate := range candidates {
		if perDomain[candidate.Domain] >= 2 {
			rejected++
			continue
		}
		perDomain[candidate.Domain]++
		selected = append(selected, candidate)
		if len(selected) >= maxActive {
			break
		}
	}
	return selected, rejected
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

func scoreCandidate(candidate discoveryCandidate, topicTokens map[string]struct{}) int {
	score := max(0, 30-candidate.Rank*2)
	searchable := tokenSet(candidate.Title + " " + candidate.Snippet)
	for token := range topicTokens {
		if _, ok := searchable[token]; ok {
			score += 8
		}
	}
	lowerQuery := strings.ToLower(candidate.Query)
	lowerTitle := strings.ToLower(candidate.Title + " " + candidate.Snippet)
	if strings.Contains(lowerQuery, "official") &&
		(strings.Contains(lowerTitle, "official") || strings.Contains(lowerTitle, "documentation")) {
		score += 15
	}
	if strings.Contains(lowerQuery, "tutorial") &&
		(strings.Contains(lowerTitle, "tutorial") || strings.Contains(lowerTitle, "example")) {
		score += 10
	}
	if strings.Contains(lowerQuery, "research") &&
		(strings.Contains(lowerTitle, "paper") || strings.Contains(lowerTitle, "research")) {
		score += 10
	}
	if strings.HasPrefix(candidate.URL, "https://") {
		score += 3
	}
	if parsed, err := url.Parse(candidate.URL); err == nil && parsed.Path != "" && parsed.Path != "/" {
		score += 3
	}
	return score
}

func candidateReason(candidate discoveryCandidate) string {
	query := strings.ToLower(candidate.Query)
	switch {
	case strings.Contains(query, "official"):
		return "official reference"
	case strings.Contains(query, "tutorial"):
		return "adds practical examples"
	case strings.Contains(query, "research"):
		return "adds research context"
	default:
		return "adds topical coverage"
	}
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
