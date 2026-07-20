package source

import (
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestRankDiscoveryCandidatesDeduplicatesAndDiversifies(t *testing.T) {
	raw := []discoveryCandidate{
		{SearchCandidate: SearchCandidate{Title: "Official inference docs", URL: "https://docs.example.com/a?utm_source=test", Snippet: "official inference documentation", Rank: 1}, Query: "inference official documentation"},
		{SearchCandidate: SearchCandidate{Title: "Duplicate", URL: "https://docs.example.com/a", Snippet: "inference", Rank: 2}, Query: "inference tutorial guide examples"},
		{SearchCandidate: SearchCandidate{Title: "Second same domain", URL: "https://blog.example.com/b", Snippet: "inference tutorial", Rank: 3}, Query: "inference tutorial guide examples"},
		{SearchCandidate: SearchCandidate{Title: "Third same domain", URL: "https://learn.example.com/c", Snippet: "inference examples", Rank: 4}, Query: "inference tutorial guide examples"},
		{SearchCandidate: SearchCandidate{Title: "Research", URL: "https://papers.example.org/paper", Snippet: "inference research paper", Rank: 5}, Query: "inference research paper review"},
		{SearchCandidate: SearchCandidate{Title: "Private", URL: "http://127.0.0.1/secret", Snippet: "inference", Rank: 6}, Query: "inference research paper review"},
	}
	selected, rejected := rankDiscoveryCandidates(
		"inference",
		raw,
		[]domain.SourceSpec{{InputURL: "https://already.example.net/source"}},
		20,
		5,
	)
	if len(selected) != 3 {
		t.Fatalf("selected=%#v", selected)
	}
	perDomain := map[string]int{}
	for _, candidate := range selected {
		perDomain[candidate.Domain]++
		if perDomain[candidate.Domain] > 2 {
			t.Fatalf("domain diversity was not enforced: %#v", selected)
		}
	}
	if rejected < 3 {
		t.Fatalf("rejected=%d, want duplicate, private, and domain overflow", rejected)
	}
}
