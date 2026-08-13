package source

import (
	"fmt"
	"testing"
	"time"

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
	if len(selected) != 2 {
		t.Fatalf("selected=%#v", selected)
	}
	perDomain := map[string]int{}
	for _, candidate := range selected {
		perDomain[candidate.Domain]++
		if perDomain[candidate.Domain] > 2 {
			t.Fatalf("domain diversity was not enforced: %#v", selected)
		}
	}
	if rejected < 4 {
		t.Fatalf("rejected=%d, want duplicate, private, and domain overflow", rejected)
	}
}

func TestDiscoveryQueriesPlanForEvidenceRoles(t *testing.T) {
	t.Parallel()
	queries := discoveryQueries("LLM inference")
	if len(queries) != 5 {
		t.Fatalf("queries=%#v", queries)
	}
	roles := make(map[domain.SourceRole]bool)
	for _, query := range queries {
		roles[query.Role] = true
		if query.Query == "" {
			t.Fatal("query text is empty")
		}
	}
	for _, role := range []domain.SourceRole{
		domain.SourceRoleOfficialPrimary,
		domain.SourceRoleResearch,
		domain.SourceRolePractitioner,
		domain.SourceRoleReporting,
		domain.SourceRoleCounterweight,
	} {
		if !roles[role] {
			t.Fatalf("missing role %q in %#v", role, queries)
		}
	}
}

func TestDiscoveryCandidateIntakeCannotStarveLaterRoles(t *testing.T) {
	t.Parallel()
	queries := discoveryQueries("inference")
	outcomes := make([]parallelOutcome[[]SearchCandidate], len(queries))
	for index := range queries {
		for rank := 0; rank < 10; rank++ {
			outcomes[index].value = append(outcomes[index].value, SearchCandidate{
				Title: string(queries[index].Role),
				URL:   "https://example.com/" + string(queries[index].Role) + "/" + fmt.Sprint(rank),
				Rank:  rank + 1,
			})
		}
	}
	candidates := interleaveDiscoveryCandidates(queries, outcomes, 5)
	if len(candidates) != 5 {
		t.Fatalf("candidates=%#v", candidates)
	}
	for index, candidate := range candidates {
		if candidate.PlannedRole != queries[index].Role {
			t.Fatalf("candidate %d role=%q, want %q", index, candidate.PlannedRole, queries[index].Role)
		}
	}
}

func TestAuthorityAndRoleOutweighSearchPosition(t *testing.T) {
	t.Parallel()
	topic := tokenSet("Kubernetes admission control")
	spam := discoveryCandidate{
		SearchCandidate: SearchCandidate{
			Title:   "Top 10 best Kubernetes tools",
			URL:     "https://content.example.com/top-10",
			Snippet: "Kubernetes admission control tools in an ultimate guide for every team.",
			Rank:    1,
		},
		Role: domain.SourceRoleReporting,
	}
	official := discoveryCandidate{
		SearchCandidate: SearchCandidate{
			Title:   "Official admission control reference",
			URL:     "https://docs.kubernetes.io/reference/admission-control",
			Snippet: "Maintained documentation for Kubernetes admission control configuration and behavior.",
			Rank:    9,
		},
		Role: domain.SourceRoleOfficialPrimary,
	}
	spamScore := scoreCandidate(spam, topic, time.Now().UTC()).Total()
	officialScore := scoreCandidate(official, topic, time.Now().UTC()).Total()
	if officialScore <= spamScore {
		t.Fatalf("official=%d spam=%d; search position still dominates trust", officialScore, spamScore)
	}
}

func TestPortfolioSelectionCoversRolesAndCapsDomainShare(t *testing.T) {
	t.Parallel()
	raw := []discoveryCandidate{
		{SearchCandidate: SearchCandidate{Title: "Official inference documentation", URL: "https://docs.vendor.example/reference", Snippet: "Official inference documentation and maintained specification for model serving.", Rank: 4}, Query: "inference official documentation", PlannedRole: domain.SourceRoleOfficialPrimary},
		{SearchCandidate: SearchCandidate{Title: "Inference research paper", URL: "https://arxiv.org/abs/1234.5678", Snippet: "Research paper studying inference systems with experiments and limitations.", Rank: 5}, Query: "inference research paper", PlannedRole: domain.SourceRoleResearch},
		{SearchCandidate: SearchCandidate{Title: "Inference implementation guide", URL: "https://engineering.example.net/inference-guide", Snippet: "Implementation guide with a production case study and worked examples.", Rank: 2}, Query: "inference implementation guide", PlannedRole: domain.SourceRolePractitioner},
		{SearchCandidate: SearchCandidate{Title: "Inference failure modes and risks", URL: "https://safety.example.org/inference-risks", Snippet: "Independent critique covering limitations, risks, and common failure modes.", Rank: 6}, Query: "inference limitations risks", PlannedRole: domain.SourceRoleCounterweight},
		{SearchCandidate: SearchCandidate{Title: "Inference industry analysis", URL: "https://news.example.com/inference-analysis", Snippet: "Independent reporting and context on inference infrastructure adoption.", Rank: 1}, Query: "inference reporting", PlannedRole: domain.SourceRoleReporting},
	}
	selected, _ := rankDiscoveryCandidates("inference", raw, nil, 20, 8)
	roles := make(map[domain.SourceRole]bool)
	domains := make(map[string]int)
	for _, candidate := range selected {
		roles[candidate.Role] = true
		domains[candidate.Domain]++
		if domains[candidate.Domain] > 2 {
			t.Fatalf("domain cap exceeded: %#v", selected)
		}
	}
	for _, role := range []domain.SourceRole{
		domain.SourceRoleOfficialPrimary,
		domain.SourceRoleResearch,
		domain.SourceRolePractitioner,
		domain.SourceRoleCounterweight,
	} {
		if !roles[role] {
			t.Fatalf("portfolio missing %q: %#v", role, selected)
		}
	}
}
