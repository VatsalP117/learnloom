package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
)

type Searcher interface {
	Search(context.Context, SearchRequest) ([]SearchCandidate, error)
}

type BrowserExtractor interface {
	Extract(context.Context, string) (ExtractedPage, error)
}

type DocumentExtractor interface {
	Extract(context.Context, DocumentInput) ([]ExtractedDocumentPart, error)
}

type SearchRequest struct {
	Query    string
	Language string
	Category string
	Page     int
}

type SearchCandidate struct {
	Title       string
	URL         string
	Snippet     string
	Engines     []string
	Rank        int
	PublishedAt *time.Time
}

type ExtractedPage struct {
	Text         string
	CanonicalURL string
	Title        string
	Error        string
}

type DocumentInput struct {
	Bytes    []byte
	FileName string
}

type ExtractedDocumentPart struct {
	Text     string
	Page     int
	Section  string
	Metadata map[string]string
}

type Repository interface {
	ListActiveSourceSpecs(ctx context.Context, newsletterID string) ([]domain.SourceSpec, error)
	UpsertSourceEndpoint(ctx context.Context, endpoint domain.SourceEndpoint) (domain.SourceEndpoint, error)
	GetSourceEndpoint(ctx context.Context, specID string) (domain.SourceEndpoint, error)
	InsertSourceSnapshot(ctx context.Context, snapshot domain.SourceSnapshot) (string, error)
	GetSourceSnapshots(ctx context.Context, endpointID string, limit int) ([]domain.SourceSnapshot, error)
	HasIssueSources(ctx context.Context, issueID string) (bool, error)
	GetIssueSources(ctx context.Context, issueID string) ([]domain.SourceSnapshot, error)
	InsertIssueSources(ctx context.Context, issueID string, links []domain.IssueSource) error
}

type Service struct {
	repo        Repository
	acquisition *Acquisition
	cfg         ServiceConfig
}

type ServiceConfig struct {
	DiscoveryEnabled   bool
	MinUsableItems     int
	DefaultMaxStaleAge time.Duration
}

func NewService(repo Repository, acquisition *Acquisition, cfg ServiceConfig) *Service {
	if cfg.MinUsableItems == 0 {
		cfg.MinUsableItems = 4
	}
	if cfg.DefaultMaxStaleAge == 0 {
		cfg.DefaultMaxStaleAge = 30 * 24 * time.Hour
	}
	return &Service{repo: repo, acquisition: acquisition, cfg: cfg}
}

type PrepareIssueResult struct {
	Items    []domain.SourceItem
	Warnings []string
	HadPrior bool
}

func (svc *Service) PrepareIssue(
	ctx context.Context,
	newsletter domain.Newsletter,
	issueID string,
) (PrepareIssueResult, error) {
	if hasSources, err := svc.repo.HasIssueSources(ctx, issueID); err != nil {
		return PrepareIssueResult{}, fmt.Errorf("check issue sources: %w", err)
	} else if hasSources {
		snapshots, err := svc.repo.GetIssueSources(ctx, issueID)
		if err != nil {
			return PrepareIssueResult{}, fmt.Errorf("load frozen issue sources: %w", err)
		}
		items := snapshotsToSourceItems(snapshots)
		return PrepareIssueResult{Items: items, HadPrior: true}, nil
	}

	specs, err := svc.repo.ListActiveSourceSpecs(ctx, newsletter.ID)
	if err != nil {
		return PrepareIssueResult{}, fmt.Errorf("list source specs: %w", err)
	}

	var allItems []domain.SourceItem
	var warnings []string
	var snapshotLinks []domain.IssueSource
	position := 0

	for _, spec := range specs {
		items, specWarnings, links, err := svc.resolveAndFreeze(ctx, spec, &position)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", spec.DisplayName, err))
		}
		warnings = append(warnings, specWarnings...)
		allItems = append(allItems, items...)
		snapshotLinks = append(snapshotLinks, links...)
	}

	if len(allItems) == 0 {
		return PrepareIssueResult{}, svc.noEvidenceError(newsletter.SourceMode, warnings)
	}

	deduped := deduplicateSourceItems(allItems)
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].PublishedAt == nil {
			return false
		}
		if deduped[j].PublishedAt == nil {
			return true
		}
		return deduped[i].PublishedAt.After(*deduped[j].PublishedAt)
	})
	if len(deduped) > 50 {
		deduped = deduped[:50]
	}

	for index := range deduped {
		deduped[index].SourceID = fmt.Sprintf("S%d", index+1)
	}

	if len(deduped) < svc.cfg.MinUsableItems {
		hasSubstantial := false
		for _, item := range deduped {
			if len([]rune(item.Summary)) >= 200 {
				hasSubstantial = true
				break
			}
		}
		if newsletter.SourceMode == domain.SourceModeProvided && hasSubstantial {
		} else if !svc.cfg.DiscoveryEnabled {
			return PrepareIssueResult{}, fmt.Errorf("insufficient evidence: %d items found, minimum is %d", len(deduped), svc.cfg.MinUsableItems)
		}
	}

	if len(snapshotLinks) > 0 {
		if err := svc.repo.InsertIssueSources(ctx, issueID, snapshotLinks); err != nil {
			warnings = append(warnings, "failed to freeze issue evidence links")
		}
	}

	return PrepareIssueResult{Items: deduped, Warnings: warnings}, nil
}

func (svc *Service) resolveAndFreeze(
	ctx context.Context,
	spec domain.SourceSpec,
	position *int,
) ([]domain.SourceItem, []string, []domain.IssueSource, error) {
	if spec.Kind == "" {
		resolveKind := svc.acquisition.resolveKind(ctx, spec.InputURL)
		spec.Kind = resolveKind
	}

	if spec.Kind == domain.SourceKindHTML {
		autoFeed, err := svc.acquisition.findAutoFeed(ctx, spec.InputURL)
		if err == nil && autoFeed != "" {
			return svc.freezeFeedFromURL(ctx, spec, autoFeed, position)
		}
	}

	switch spec.Kind {
	case domain.SourceKindRSS, domain.SourceKindAtom, domain.SourceKindJSONFeed:
		return svc.freezeFeed(ctx, spec, position)
	case domain.SourceKindHTML, domain.SourceKindText:
		return svc.freezeExact(ctx, spec, position)
	default:
		return svc.freezeExact(ctx, spec, position)
	}
}

func (svc *Service) freezeFeed(
	ctx context.Context,
	spec domain.SourceSpec,
	position *int,
) ([]domain.SourceItem, []string, []domain.IssueSource, error) {
	var endpoint domain.SourceEndpoint
	existing, err := svc.repo.GetSourceEndpoint(ctx, spec.ID)
	if err == nil {
		endpoint = existing
	} else {
		endpoint = domain.SourceEndpoint{
			ID:           uuid.NewString(),
			SourceSpecID: spec.ID,
			EndpointURL:  spec.InputURL,
			CanonicalURL: spec.InputURL,
			Kind:         spec.Kind,
			Health:       "unknown",
			UpdatedAt:    time.Now().UTC(),
		}
	}

	now := time.Now().UTC()
	result, fetchErr := doHTTP(ctx, svc.acquisition.Client, endpoint.EndpointURL,
		svc.acquisition.Cfg.MaxFeedBytes,
		"application/atom+xml, application/rss+xml, application/xml, text/xml, application/feed+json, application/json",
		endpoint.ETag, endpoint.LastModified)

	if fetchErr != nil {
		endpoint.ConsecutiveFailures++
		endpoint.LastCheckedAt = &now
		endpoint.LastError = safeError(fetchErr)
		if endpoint.ConsecutiveFailures >= 5 {
			endpoint.Health = "failing"
		}
		svc.repo.UpsertSourceEndpoint(ctx, endpoint)
		return nil, nil, nil, fetchErr
	}

	endpoint.LastCheckedAt = &now
	endpoint.LastHTTPStatus = result.StatusCode
	endpoint.ConsecutiveFailures = 0

	if result.StatusCode == http.StatusNotModified {
		endpoint.ETag = result.ETag
		endpoint.LastModified = result.LastModified
		endpoint.Health = "healthy"
		svc.repo.UpsertSourceEndpoint(ctx, endpoint)
		snapshots, err := svc.repo.GetSourceSnapshots(ctx, endpoint.ID, spec.ItemLimit)
		if err != nil {
			return nil, nil, nil, err
		}
		items := snapshotsToSourceItems(snapshots)
		links := make([]domain.IssueSource, len(snapshots))
		for i := range snapshots {
			links[i] = domain.IssueSource{
				IssueID:          "will-be-filled",
				SourceSnapshotID: snapshots[i].ID,
				Position:         *position + i,
				CreatedAt:        now,
			}
		}
		*position += len(snapshots)
		return items, nil, links, nil
	}

	endpoint.Kind = detectKind(result.ContentType, result.Body)
	endpoint.EndpointURL = result.FinalURL.String()
	endpoint.CanonicalURL = result.FinalURL.String()
	endpoint.ETag = result.ETag
	endpoint.LastModified = result.LastModified
	endpoint.Health = "healthy"
	endpoint.LastSuccessAt = &now

	if endpoint.EndpointURL != endpoint.CanonicalURL {
		endpoint.CanonicalURL = endpoint.EndpointURL
	}

	_, _ = svc.repo.UpsertSourceEndpoint(ctx, endpoint)

	items, err := ParseFeed(result.Body, spec.DisplayName)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(items) > spec.ItemLimit {
		items = items[:spec.ItemLimit]
	}

	var sourceItems []domain.SourceItem
	var links []domain.IssueSource

	for i, item := range items {
		if resolved, resolveErr := result.FinalURL.Parse(item.URL); resolveErr == nil {
			item.URL = resolved.String()
			item.CanonicalURL = item.URL
		}
		title := item.Title
		if title == "" {
			title = spec.DisplayName
		}
		content := item.Summary
		if content == "" {
			content = item.Title
		}
		contentHash := hashContent(content)
		snapshot := domain.SourceSnapshot{
			ID:               uuid.NewString(),
			SourceEndpointID: endpoint.ID,
			ItemKey:          fmt.Sprintf("%s:%d", endpoint.ID, i),
			Title:            title,
			CanonicalURL:     item.CanonicalURL,
			Author:           item.Author,
			PublishedAt:      item.PublishedAt,
			Content:          content,
			ContentSource:    "feed-summary",
			ContentSHA256:    contentHash,
			Metadata:         "{}",
			FetchedAt:        now,
		}
		snapshotID, snapErr := svc.repo.InsertSourceSnapshot(ctx, snapshot)
		if snapErr != nil {
			continue
		}
		links = append(links, domain.IssueSource{
			IssueID:          "will-be-filled",
			SourceSnapshotID: snapshotID,
			Position:         *position + len(links),
			CreatedAt:        now,
		})
		sourceItems = append(sourceItems, domain.SourceItem{
			Source:        spec.DisplayName,
			Title:         title,
			URL:           item.CanonicalURL,
			CanonicalURL:  item.CanonicalURL,
			Summary:       content,
			PublishedAt:   item.PublishedAt,
			ContentSource: "feed-summary",
			Author:        item.Author,
		})
	}
	*position += len(links)
	return sourceItems, nil, links, nil
}

func (svc *Service) freezeExact(
	ctx context.Context,
	spec domain.SourceSpec,
	position *int,
) ([]domain.SourceItem, []string, []domain.IssueSource, error) {
	var endpoint domain.SourceEndpoint
	existing, err := svc.repo.GetSourceEndpoint(ctx, spec.ID)
	if err == nil {
		endpoint = existing
	} else {
		endpoint = domain.SourceEndpoint{
			ID:           uuid.NewString(),
			SourceSpecID: spec.ID,
			EndpointURL:  spec.InputURL,
			CanonicalURL: spec.InputURL,
			Kind:         domain.SourceKindHTML,
			Health:       "unknown",
			UpdatedAt:    time.Now().UTC(),
		}
	}

	now := time.Now().UTC()
	result, fetchErr := doHTTP(ctx, svc.acquisition.Client, endpoint.EndpointURL,
		svc.acquisition.Cfg.MaxArticleBytes,
		"text/html, text/plain;q=0.9",
		endpoint.ETag, endpoint.LastModified)

	if fetchErr != nil {
		endpoint.ConsecutiveFailures++
		endpoint.LastCheckedAt = &now
		endpoint.LastError = safeError(fetchErr)
		if endpoint.ConsecutiveFailures >= 5 {
			endpoint.Health = "failing"
		}
		svc.repo.UpsertSourceEndpoint(ctx, endpoint)
		return nil, nil, nil, fetchErr
	}

	endpoint.LastCheckedAt = &now
	endpoint.LastHTTPStatus = result.StatusCode
	endpoint.ConsecutiveFailures = 0
	endpoint.Kind = detectKind(result.ContentType, result.Body)
	endpoint.EndpointURL = result.FinalURL.String()
	endpoint.CanonicalURL = result.FinalURL.String()
	endpoint.ETag = result.ETag
	endpoint.LastModified = result.LastModified

	if result.StatusCode == http.StatusNotModified {
		endpoint.Health = "healthy"
		svc.repo.UpsertSourceEndpoint(ctx, endpoint)
		snapshots, err := svc.repo.GetSourceSnapshots(ctx, endpoint.ID, 1)
		if err != nil || len(snapshots) == 0 {
			return nil, nil, nil, fmt.Errorf("304 without stored exact snapshots")
		}
		items := snapshotsToSourceItems(snapshots)
		links := []domain.IssueSource{{
			IssueID:          "will-be-filled",
			SourceSnapshotID: snapshots[0].ID,
			Position:         *position,
			CreatedAt:        now,
		}}
		*position++
		return items, nil, links, nil
	}

	endpoint.Health = "healthy"
	endpoint.LastSuccessAt = &now
	_, _ = svc.repo.UpsertSourceEndpoint(ctx, endpoint)

	text, canonicalURL, _ := extractArticle(string(result.Body), result.FinalURL)
	if len([]rune(text)) < 100 {
		return nil, nil, nil, fmt.Errorf("article content too short")
	}

	contentHash := hashContent(text)
	snapshot := domain.SourceSnapshot{
		ID:               uuid.NewString(),
		SourceEndpointID: endpoint.ID,
		ItemKey:          canonicalURL,
		Title:            spec.DisplayName,
		CanonicalURL:     canonicalURL,
		Content:          truncateRunes(text, svc.acquisition.Cfg.MaxArticleCharacters),
		ContentSource:    "article",
		ContentSHA256:    contentHash,
		Metadata:         "{}",
		FetchedAt:        now,
	}
	snapshotID, snapErr := svc.repo.InsertSourceSnapshot(ctx, snapshot)
	if snapErr != nil {
		return nil, nil, nil, snapErr
	}

	item := domain.SourceItem{
		Source:        spec.DisplayName,
		Title:         spec.DisplayName,
		URL:           canonicalURL,
		CanonicalURL:  canonicalURL,
		Summary:       snapshot.Content,
		ContentSource: "article",
	}
	links := []domain.IssueSource{{
		IssueID:          "will-be-filled",
		SourceSnapshotID: snapshotID,
		Position:         *position,
		CreatedAt:        now,
	}}
	*position++
	return []domain.SourceItem{item}, nil, links, nil
}

func (svc *Service) freezeFeedFromURL(
	ctx context.Context,
	spec domain.SourceSpec,
	feedURL string,
	position *int,
) ([]domain.SourceItem, []string, []domain.IssueSource, error) {
	feedSpec := spec
	feedSpec.Kind = domain.SourceKindRSS
	feedSpec.InputURL = feedURL
	return svc.freezeFeed(ctx, feedSpec, position)
}

func (svc *Service) noEvidenceError(mode domain.SourceMode, warnings []string) error {
	switch mode {
	case domain.SourceModeProvided:
		return fmt.Errorf("the supplied sources could not provide enough readable evidence: %v", warningsToErr(warnings))
	case domain.SourceModeHybrid:
		return fmt.Errorf("no usable sources were found and discovery is unavailable: %v", warningsToErr(warnings))
	case domain.SourceModeDiscovered:
		return fmt.Errorf("discovery is disabled; cannot generate without evidence")
	default:
		return fmt.Errorf("no usable sources found: %v", warningsToErr(warnings))
	}
}

func snapshotsToSourceItems(snapshots []domain.SourceSnapshot) []domain.SourceItem {
	items := make([]domain.SourceItem, 0, len(snapshots))
	for index, snapshot := range snapshots {
		items = append(items, domain.SourceItem{
			SourceID:      fmt.Sprintf("S%d", index+1),
			Source:        "",
			Title:         snapshot.Title,
			URL:           snapshot.CanonicalURL,
			CanonicalURL:  snapshot.CanonicalURL,
			Summary:       snapshot.Content,
			PublishedAt:   snapshot.PublishedAt,
			ContentSource: snapshot.ContentSource,
			Author:        snapshot.Author,
		})
	}
	return items
}

func deduplicateSourceItems(items []domain.SourceItem) []domain.SourceItem {
	seen := map[string]struct{}{}
	var result []domain.SourceItem
	for _, item := range items {
		key := strings.ToLower(item.Title) + "|" + item.CanonicalURL
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

func warningsToErr(warnings []string) string {
	if len(warnings) == 0 {
		return "unknown error"
	}
	result := warnings[0]
	for _, w := range warnings[1:] {
		result += "; " + w
	}
	return result
}
