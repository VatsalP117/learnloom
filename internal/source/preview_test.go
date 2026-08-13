package source

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type previewSearcher struct{ fail bool }

func (searcher previewSearcher) Search(
	_ context.Context,
	request SearchRequest,
) ([]SearchCandidate, error) {
	if searcher.fail {
		return nil, errors.New("search unavailable")
	}
	query := strings.ToLower(request.Query)
	switch {
	case strings.Contains(query, "official"):
		return []SearchCandidate{{
			Title:   "Official inference documentation",
			URL:     "https://docs.vendor.example/reference/inference",
			Snippet: "Maintained official documentation and reference for inference systems.", Rank: 3,
		}}, nil
	case strings.Contains(query, "research"):
		return []SearchCandidate{{
			Title:   "Inference systems research paper",
			URL:     "https://arxiv.org/abs/1234.5678",
			Snippet: "Research paper with experiments on inference systems and measured outcomes.", Rank: 4,
		}}, nil
	case strings.Contains(query, "implementation"):
		return []SearchCandidate{{
			Title:   "Inference implementation guide",
			URL:     "https://engineering.example.net/inference-guide",
			Snippet: "Implementation guide and production case study with worked examples.", Rank: 2,
		}}, nil
	case strings.Contains(query, "limitations"):
		return []SearchCandidate{{
			Title:   "Inference limitations and failure modes",
			URL:     "https://safety.example.org/inference-risks",
			Snippet: "Independent critique of limitations, risks, and failure modes in inference systems.", Rank: 5,
		}}, nil
	default:
		return []SearchCandidate{{
			Title:   "Inference industry analysis",
			URL:     "https://reporting.example.com/inference-analysis",
			Snippet: "Independent reporting and analysis providing industry context for inference systems.", Rank: 1,
		}}, nil
	}
}

func TestPreviewPortfolioReturnsExplainableRoleCoverage(t *testing.T) {
	t.Parallel()
	preview, err := PreviewPortfolio(
		context.Background(), previewSearcher{}, "inference systems", 5, 30, 8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if preview.RankingVersion != sourceRankingVersion || len(preview.Items) != 5 ||
		len(preview.MissingRoles) != 0 || preview.Warnings != 0 {
		t.Fatalf("preview=%#v", preview)
	}
	for _, item := range preview.Items {
		if item.Role == "" || item.SelectionReason == "" || item.URL == "" ||
			item.RegistrableDomain == "" {
			t.Fatalf("unexplained preview item=%#v", item)
		}
	}
}

func TestPreviewPortfolioFailsSafelyWhenEveryQueryFails(t *testing.T) {
	t.Parallel()
	_, err := PreviewPortfolio(
		context.Background(), previewSearcher{fail: true}, "inference", 5, 30, 8,
	)
	if err == nil || strings.Contains(err.Error(), "search unavailable") {
		t.Fatalf("unsafe or missing preview error: %v", err)
	}
}
