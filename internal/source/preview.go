package source

import (
	"context"
	"errors"
	"sort"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type PortfolioPreviewItem struct {
	Title             string            `json:"title"`
	URL               string            `json:"url"`
	RegistrableDomain string            `json:"registrableDomain"`
	Role              domain.SourceRole `json:"role"`
	SelectionReason   string            `json:"selectionReason"`
}

type PortfolioPreview struct {
	RankingVersion string                 `json:"rankingVersion"`
	Items          []PortfolioPreviewItem `json:"items"`
	MissingRoles   []domain.SourceRole    `json:"missingRoles"`
	Warnings       int                    `json:"warnings"`
}

func PreviewPortfolio(
	ctx context.Context,
	searcher Searcher,
	topic string,
	maxQueries, maxCandidates, maxActive int,
) (PortfolioPreview, error) {
	if searcher == nil {
		return PortfolioPreview{}, errors.New("source portfolio preview is unavailable")
	}
	queries := discoveryQueries(topic)
	if maxQueries < 1 || maxQueries > len(queries) {
		maxQueries = len(queries)
	}
	queries = queries[:maxQueries]
	if maxCandidates < 1 {
		maxCandidates = 30
	}
	if maxActive < 1 {
		maxActive = 8
	}
	outcomes := parallelMapOrdered(
		ctx,
		queries,
		min(5, len(queries)),
		func(ctx context.Context, query discoveryQuery) ([]SearchCandidate, error) {
			return searcher.Search(ctx, SearchRequest{
				Query: query.Query, Language: "all", Category: "general", Page: 1,
			})
		},
	)
	warnings := 0
	for _, outcome := range outcomes {
		if outcome.err != nil {
			warnings++
		}
	}
	raw := interleaveDiscoveryCandidates(queries, outcomes, maxCandidates)
	if len(raw) == 0 && warnings > 0 {
		return PortfolioPreview{}, errors.New("source discovery is temporarily unavailable")
	}
	selected, _ := rankDiscoveryCandidates(topic, raw, nil, maxCandidates, maxActive)
	preview := PortfolioPreview{
		RankingVersion: sourceRankingVersion,
		Warnings:       warnings,
		Items:          make([]PortfolioPreviewItem, 0, len(selected)),
	}
	covered := make(map[domain.SourceRole]bool)
	for _, candidate := range selected {
		covered[candidate.Role] = true
		preview.Items = append(preview.Items, PortfolioPreviewItem{
			Title: candidate.Title, URL: candidate.URL,
			RegistrableDomain: candidate.Domain, Role: candidate.Role,
			SelectionReason: candidate.Reason,
		})
	}
	for _, role := range []domain.SourceRole{
		domain.SourceRoleOfficialPrimary,
		domain.SourceRoleResearch,
		domain.SourceRolePractitioner,
		domain.SourceRoleCounterweight,
	} {
		if !covered[role] {
			preview.MissingRoles = append(preview.MissingRoles, role)
		}
	}
	sort.SliceStable(preview.Items, func(i, j int) bool {
		return roleOrder(preview.Items[i].Role) < roleOrder(preview.Items[j].Role)
	})
	return preview, nil
}

func roleOrder(role domain.SourceRole) int {
	switch role {
	case domain.SourceRoleOfficialPrimary:
		return 0
	case domain.SourceRoleResearch:
		return 1
	case domain.SourceRolePractitioner:
		return 2
	case domain.SourceRoleCounterweight:
		return 3
	case domain.SourceRoleReporting:
		return 4
	default:
		return 5
	}
}
