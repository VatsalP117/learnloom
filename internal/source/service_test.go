package source

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type memorySourceRepository struct {
	mu             sync.Mutex
	specs          []domain.SourceSpec
	endpoints      map[string]domain.SourceEndpoint
	snapshots      map[string][]domain.SourceSnapshot
	issueLinks     map[string][]domain.IssueSource
	failEndpoint   error
	failSnapshot   error
	failIssueLinks error
	discoveryRuns  []domain.DiscoveryRun
}

func newMemorySourceRepository(specs ...domain.SourceSpec) *memorySourceRepository {
	return &memorySourceRepository{
		specs:      specs,
		endpoints:  make(map[string]domain.SourceEndpoint),
		snapshots:  make(map[string][]domain.SourceSnapshot),
		issueLinks: make(map[string][]domain.IssueSource),
	}
}

func (repo *memorySourceRepository) ListActiveSourceSpecs(context.Context, string) ([]domain.SourceSpec, error) {
	return append([]domain.SourceSpec(nil), repo.specs...), nil
}

func (repo *memorySourceRepository) UpsertSourceEndpoint(_ context.Context, endpoint domain.SourceEndpoint) (domain.SourceEndpoint, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.failEndpoint != nil {
		return domain.SourceEndpoint{}, repo.failEndpoint
	}
	repo.endpoints[endpoint.SourceSpecID] = endpoint
	return endpoint, nil
}

func (repo *memorySourceRepository) GetSourceEndpoint(_ context.Context, specID string) (domain.SourceEndpoint, bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	endpoint, ok := repo.endpoints[specID]
	return endpoint, ok, nil
}

func (repo *memorySourceRepository) InsertSourceSnapshot(_ context.Context, snapshot domain.SourceSnapshot) (string, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.failSnapshot != nil {
		return "", repo.failSnapshot
	}
	for _, existing := range repo.snapshots[snapshot.SourceEndpointID] {
		if existing.ItemKey == snapshot.ItemKey && existing.ContentSHA256 == snapshot.ContentSHA256 {
			return existing.ID, nil
		}
	}
	repo.snapshots[snapshot.SourceEndpointID] = append(repo.snapshots[snapshot.SourceEndpointID], snapshot)
	return snapshot.ID, nil
}

func (repo *memorySourceRepository) GetSourceSnapshots(_ context.Context, endpointID string, limit int) ([]domain.SourceSnapshot, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	snapshots := append([]domain.SourceSnapshot(nil), repo.snapshots[endpointID]...)
	if len(snapshots) > limit {
		snapshots = snapshots[:limit]
	}
	return snapshots, nil
}

func (repo *memorySourceRepository) HasIssueSources(_ context.Context, issueID string) (bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return len(repo.issueLinks[issueID]) > 0, nil
}

func (repo *memorySourceRepository) GetIssueSources(_ context.Context, issueID string) ([]domain.SourceSnapshot, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	links := repo.issueLinks[issueID]
	result := make([]domain.SourceSnapshot, 0, len(links))
	for _, link := range links {
		for _, snapshots := range repo.snapshots {
			for _, snapshot := range snapshots {
				if snapshot.ID == link.SourceSnapshotID {
					result = append(result, snapshot)
				}
			}
		}
	}
	return result, nil
}

func (repo *memorySourceRepository) InsertIssueSources(_ context.Context, issueID string, links []domain.IssueSource) (bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.failIssueLinks != nil {
		return false, repo.failIssueLinks
	}
	if len(repo.issueLinks[issueID]) > 0 {
		return false, nil
	}
	repo.issueLinks[issueID] = append([]domain.IssueSource(nil), links...)
	return true, nil
}

func (repo *memorySourceRepository) UpsertDiscoveredSourceSpec(_ context.Context, spec domain.SourceSpec) (domain.SourceSpec, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, existing := range repo.specs {
		if existing.CanonicalURL == spec.CanonicalURL {
			return existing, nil
		}
	}
	repo.specs = append(repo.specs, spec)
	return spec, nil
}

func (repo *memorySourceRepository) SetSourceSpecState(_ context.Context, specID string, state domain.SourceState, kind domain.SourceKind) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for index := range repo.specs {
		if repo.specs[index].ID == specID {
			repo.specs[index].State = state
			if kind != "" {
				repo.specs[index].Kind = kind
			}
			return nil
		}
	}
	return errors.New("source spec not found")
}

func (repo *memorySourceRepository) CreateDiscoveryRun(_ context.Context, run domain.DiscoveryRun) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.discoveryRuns = append(repo.discoveryRuns, run)
	return nil
}

func (repo *memorySourceRepository) CompleteDiscoveryRun(_ context.Context, run domain.DiscoveryRun) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for index := range repo.discoveryRuns {
		if repo.discoveryRuns[index].ID == run.ID {
			repo.discoveryRuns[index] = run
			return nil
		}
	}
	return errors.New("discovery run not found")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type fakeSearcher struct {
	mu      sync.Mutex
	calls   int
	results []SearchCandidate
	err     error
}

func (searcher *fakeSearcher) Search(context.Context, SearchRequest) ([]SearchCandidate, error) {
	searcher.mu.Lock()
	defer searcher.mu.Unlock()
	searcher.calls++
	if searcher.err != nil {
		return nil, searcher.err
	}
	return append([]SearchCandidate(nil), searcher.results...), nil
}

func sourceTestService(repo Repository, transport http.RoundTripper, cfg ServiceConfig) *Service {
	acquisition := New(Config{
		FeedTimeout:          time.Second,
		ArticleTimeout:       time.Second,
		MaxFeedBytes:         1 << 20,
		MaxArticleBytes:      1 << 20,
		MaxArticleCharacters: 10_000,
	})
	acquisition.Client = &http.Client{Transport: transport}
	return NewService(repo, acquisition, cfg)
}

func sourceTestSpec(kind domain.SourceKind) domain.SourceSpec {
	return domain.SourceSpec{
		ID:           "spec-1",
		NewsletterID: "newsletter-1",
		Origin:       domain.SourceOriginProvided,
		State:        domain.SourceStateActive,
		DisplayName:  "Reference",
		InputURL:     "https://example.com/source",
		CanonicalURL: "https://example.com/source",
		Scope:        domain.SourceScopeExact,
		Kind:         kind,
		ItemLimit:    8,
	}
}

func httpFixture(request *http.Request, status int, contentType, body string, headers map[string]string) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", contentType)
	for key, value := range headers {
		header.Set(key, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

func TestPrepareIssueFreezesExactEvidenceAndRetryMakesNoRequest(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindHTML)
	repo := newMemorySourceRepository(spec)
	requests := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		body := "<html><head><title>Reference</title></head><body><main><p>" +
			strings.Repeat("grounded explanation ", 50) +
			"</p></main></body></html>"
		return httpFixture(request, http.StatusOK, "text/html", body, nil), nil
	})
	service := sourceTestService(repo, transport, ServiceConfig{
		MinUsableItems:        4,
		SubstantialCharacters: 400,
	})
	newsletter := domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided}

	first, err := service.PrepareIssue(context.Background(), newsletter, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.PrepareIssue(context.Background(), newsletter, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests=%d, want 1", requests)
	}
	if second.HadPrior != true || !reflect.DeepEqual(first.Items, second.Items) {
		t.Fatalf("retry mismatch\nfirst=%#v\nsecond=%#v", first.Items, second.Items)
	}
	if first.Items[0].Source != spec.DisplayName {
		t.Fatalf("source provenance was not preserved: %#v", first.Items[0])
	}
}

func TestPrepareIssueFreezesOnlyFinalFeedSelection(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindRSS)
	spec.Scope = domain.SourceScopeFeed
	repo := newMemorySourceRepository(spec)
	summaryA := strings.Repeat("newest useful explanation ", 6)
	summaryB := strings.Repeat("second useful explanation ", 6)
	feed := `<?xml version="1.0"?><rss version="2.0"><channel><title>Feed</title>` +
		`<item><title>Older duplicate</title><link>https://example.com/a</link><description>` + summaryA + `</description><pubDate>Mon, 01 Jun 2026 10:00:00 GMT</pubDate></item>` +
		`<item><title>Newest</title><link>https://example.com/b</link><description>` + summaryB + `</description><pubDate>Mon, 20 Jul 2026 10:00:00 GMT</pubDate></item>` +
		`<item><title>Older duplicate</title><link>https://example.com/a</link><description>` + summaryA + `</description><pubDate>Mon, 01 Jun 2026 10:00:00 GMT</pubDate></item>` +
		`</channel></rss>`
	requests := 0
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return httpFixture(request, http.StatusOK, "application/rss+xml", feed, nil), nil
	}), ServiceConfig{MinUsableItems: 2, MaxItems: 2})
	newsletter := domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided}

	first, err := service.PrepareIssue(context.Background(), newsletter, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].Title != "Newest" {
		t.Fatalf("unexpected selection: %#v", first.Items)
	}
	if links := repo.issueLinks["issue-1"]; len(links) != 2 {
		t.Fatalf("frozen links=%d, want 2", len(links))
	}
	retry, err := service.PrepareIssue(context.Background(), newsletter, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || !reflect.DeepEqual(first.Items, retry.Items) {
		t.Fatalf("retry changed evidence: requests=%d first=%#v retry=%#v", requests, first.Items, retry.Items)
	}
}

func TestPrepareIssueRequiresDurableFreeze(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindHTML)
	repo := newMemorySourceRepository(spec)
	repo.failIssueLinks = errors.New("database unavailable")
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := "<html><body><main>" + strings.Repeat("substantial evidence ", 50) + "</main></body></html>"
		return httpFixture(request, http.StatusOK, "text/html", body, nil), nil
	}), ServiceConfig{SubstantialCharacters: 400})

	_, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided},
		"issue-1",
	)
	if err == nil || !strings.Contains(err.Error(), "freeze Issue evidence") {
		t.Fatalf("err=%v, want durable freeze failure", err)
	}
}

func TestPrepareIssueStopsOnEvidencePersistenceFailures(t *testing.T) {
	tests := []struct {
		name          string
		endpointError error
		snapshotError error
	}{
		{name: "endpoint", endpointError: errors.New("endpoint write failed")},
		{name: "snapshot", snapshotError: errors.New("snapshot write failed")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := sourceTestSpec(domain.SourceKindHTML)
			repo := newMemorySourceRepository(spec)
			repo.failEndpoint = test.endpointError
			repo.failSnapshot = test.snapshotError
			service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
				body := "<html><body><main>" + strings.Repeat("substantial evidence ", 50) + "</main></body></html>"
				return httpFixture(request, http.StatusOK, "text/html", body, nil), nil
			}), ServiceConfig{SubstantialCharacters: 400})
			_, err := service.PrepareIssue(
				context.Background(),
				domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided},
				"issue-1",
			)
			if err == nil {
				t.Fatal("persistence failure did not stop preparation")
			}
			if len(repo.issueLinks["issue-1"]) != 0 {
				t.Fatal("Issue evidence was linked after a persistence failure")
			}
		})
	}
}

func TestPrepareIssuePreservesValidatorsAcross304(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindRSS)
	spec.Scope = domain.SourceScopeFeed
	repo := newMemorySourceRepository(spec)
	requests := 0
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			feed := `<?xml version="1.0"?><rss version="2.0"><channel>` +
				`<item><title>One</title><link>https://example.com/one</link><description>` + strings.Repeat("useful evidence ", 8) + `</description></item>` +
				`</channel></rss>`
			return httpFixture(request, http.StatusOK, "application/rss+xml", feed, map[string]string{"ETag": `"v1"`}), nil
		}
		if got := request.Header.Get("If-None-Match"); got != `"v1"` {
			t.Fatalf("If-None-Match=%q", got)
		}
		return httpFixture(request, http.StatusNotModified, "", "", nil), nil
	}), ServiceConfig{MinUsableItems: 1, RefreshInterval: time.Nanosecond})
	newsletter := domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided}

	if _, err := service.PrepareIssue(context.Background(), newsletter, "issue-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PrepareIssue(context.Background(), newsletter, "issue-2"); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || repo.endpoints[spec.ID].ETag != `"v1"` {
		t.Fatalf("requests=%d endpoint=%#v", requests, repo.endpoints[spec.ID])
	}
}

func TestPrepareIssueRejectsStaleEvidenceOn304(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindRSS)
	spec.Scope = domain.SourceScopeFeed
	repo := newMemorySourceRepository(spec)
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return httpFixture(request, http.StatusNotModified, "", "", nil), nil
	}), ServiceConfig{
		MinUsableItems:     1,
		RefreshInterval:    time.Nanosecond,
		DefaultMaxStaleAge: time.Hour,
	})
	old := time.Now().Add(-2 * time.Hour)
	repo.endpoints[spec.ID] = domain.SourceEndpoint{
		ID:           "endpoint-1",
		SourceSpecID: spec.ID,
		EndpointURL:  spec.InputURL,
		Kind:         domain.SourceKindRSS,
		ETag:         `"old"`,
		LastCheckedAt: func() *time.Time {
			value := old
			return &value
		}(),
	}
	repo.snapshots["endpoint-1"] = []domain.SourceSnapshot{{
		ID:               "snapshot-1",
		SourceEndpointID: "endpoint-1",
		ItemKey:          "old",
		Title:            "Old",
		CanonicalURL:     "https://example.com/old",
		Content:          strings.Repeat("old evidence ", 10),
		ContentSource:    "feed-summary",
		FetchedAt:        old,
	}}
	_, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided},
		"issue-stale",
	)
	if err == nil || !strings.Contains(err.Error(), "too old") {
		t.Fatalf("err=%v, want stale evidence failure", err)
	}
}

func TestPrepareIssueRejectsEmptyFeed(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindRSS)
	repo := newMemorySourceRepository(spec)
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return httpFixture(
			request,
			http.StatusOK,
			"application/rss+xml",
			`<rss version="2.0"><channel><title>Empty</title>`+
				`<item><title>No content</title><link>https://example.com/empty</link></item>`+
				`</channel></rss>`,
			nil,
		), nil
	}), ServiceConfig{MinUsableItems: 1})
	_, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided},
		"issue-empty",
	)
	if err == nil || !strings.Contains(err.Error(), "feed contained no usable entries") {
		t.Fatalf("err=%v, want empty feed failure", err)
	}
}

func TestDiscoveryFlagDoesNotWaiveEvidenceMinimum(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindRSS)
	repo := newMemorySourceRepository(spec)
	feed := `<rss version="2.0"><channel><item><title>One</title><link>https://example.com/one</link><description>` +
		strings.Repeat("useful evidence ", 8) + `</description></item></channel></rss>`
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return httpFixture(request, http.StatusOK, "application/rss+xml", feed, nil), nil
	}), ServiceConfig{DiscoveryEnabled: true, MinUsableItems: 4})

	_, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeDiscovered},
		"issue-1",
	)
	if err == nil || !strings.Contains(err.Error(), "insufficient evidence") {
		t.Fatalf("err=%v, want evidence minimum failure", err)
	}
}

func TestProvidedModeNeverSearches(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindHTML)
	repo := newMemorySourceRepository(spec)
	searcher := &fakeSearcher{err: errors.New("must not be called")}
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := "<html><body><main>" + strings.Repeat("substantial provided evidence ", 40) + "</main></body></html>"
		return httpFixture(request, http.StatusOK, "text/html", body, nil), nil
	}), ServiceConfig{
		DiscoveryEnabled:      true,
		MinUsableItems:        4,
		SubstantialCharacters: 400,
	}).WithSearcher(searcher)

	if _, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeProvided},
		"issue-1",
	); err != nil {
		t.Fatal(err)
	}
	if searcher.calls != 0 {
		t.Fatalf("provided mode searched %d times", searcher.calls)
	}
}

func TestHybridWithSubstantialProvidedEvidenceDoesNotSearch(t *testing.T) {
	spec := sourceTestSpec(domain.SourceKindHTML)
	repo := newMemorySourceRepository(spec)
	searcher := &fakeSearcher{}
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := "<html><body><main>" + strings.Repeat("substantial provided evidence ", 40) + "</main></body></html>"
		return httpFixture(request, http.StatusOK, "text/html", body, nil), nil
	}), ServiceConfig{
		DiscoveryEnabled:      true,
		MinUsableItems:        4,
		SubstantialCharacters: 400,
	}).WithSearcher(searcher)

	if _, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{ID: spec.NewsletterID, SourceMode: domain.SourceModeHybrid},
		"issue-1",
	); err != nil {
		t.Fatal(err)
	}
	if searcher.calls != 0 {
		t.Fatalf("sufficient hybrid mode searched %d times", searcher.calls)
	}
}

func TestDiscoveredModeActivatesAndFreezesSearchResults(t *testing.T) {
	repo := newMemorySourceRepository()
	searcher := &fakeSearcher{results: []SearchCandidate{
		{Title: "Official inference docs", URL: "https://docs.example.com/inference", Snippet: "official inference documentation", Rank: 1},
		{Title: "Inference tutorial", URL: "https://learn.example.org/tutorial", Snippet: "inference tutorial examples", Rank: 2},
		{Title: "Inference research", URL: "https://papers.example.net/research", Snippet: "inference research paper", Rank: 3},
		{Title: "Inference practice", URL: "https://practice.example.edu/guide", Snippet: "inference practical guide", Rank: 4},
	}}
	requests := 0
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		body := "<html><body><main>" + strings.Repeat("grounded discovered evidence ", 35) + "</main></body></html>"
		return httpFixture(request, http.StatusOK, "text/html", body, nil), nil
	}), ServiceConfig{
		DiscoveryEnabled:       true,
		MinUsableItems:         4,
		TargetUsableItems:      4,
		DiscoveryMaxQueries:    3,
		DiscoveryMaxCandidates: 20,
		DiscoveryMaxActive:     4,
		SubstantialCharacters:  400,
	}).WithSearcher(searcher)
	newsletter := domain.Newsletter{
		ID: "newsletter-1", Topic: "LLM inference",
		SourceMode: domain.SourceModeDiscovered,
	}

	result, err := service.PrepareIssue(context.Background(), newsletter, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 4 || len(repo.issueLinks["issue-1"]) != 4 ||
		requests != 4 || searcher.calls != 3 {
		t.Fatalf(
			"items=%d links=%d requests=%d searches=%d",
			len(result.Items), len(repo.issueLinks["issue-1"]), requests, searcher.calls,
		)
	}
	if len(repo.discoveryRuns) != 1 || repo.discoveryRuns[0].State != "completed" ||
		repo.discoveryRuns[0].ActivatedCandidates != 4 {
		t.Fatalf("discovery run=%#v", repo.discoveryRuns)
	}
}

func TestHybridDiscoveryFailureCannotMaskInsufficientEvidence(t *testing.T) {
	repo := newMemorySourceRepository()
	searcher := &fakeSearcher{err: errors.New("search unavailable")}
	service := sourceTestService(repo, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, errors.New("unexpected source request")
	}), ServiceConfig{
		DiscoveryEnabled: true,
		MinUsableItems:   4,
	}).WithSearcher(searcher)

	_, err := service.PrepareIssue(
		context.Background(),
		domain.Newsletter{
			ID: "newsletter-1", Topic: "inference",
			SourceMode: domain.SourceModeHybrid,
		},
		"issue-1",
	)
	if err == nil || !strings.Contains(err.Error(), "could not provide enough") {
		t.Fatalf("err=%v, want total insufficiency", err)
	}
	if searcher.calls == 0 {
		t.Fatal("hybrid gap did not invoke discovery")
	}
}
