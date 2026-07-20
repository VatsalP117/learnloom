package source

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	GetSourceEndpoint(ctx context.Context, specID string) (domain.SourceEndpoint, bool, error)
	InsertSourceSnapshot(ctx context.Context, snapshot domain.SourceSnapshot) (string, error)
	GetSourceSnapshots(ctx context.Context, endpointID string, limit int) ([]domain.SourceSnapshot, error)
	HasIssueSources(ctx context.Context, issueID string) (bool, error)
	GetIssueSources(ctx context.Context, issueID string) ([]domain.SourceSnapshot, error)
	InsertIssueSources(ctx context.Context, issueID string, links []domain.IssueSource) (bool, error)
	UpsertDiscoveredSourceSpec(ctx context.Context, spec domain.SourceSpec) (domain.SourceSpec, error)
	SetSourceSpecState(ctx context.Context, specID string, state domain.SourceState, kind domain.SourceKind) error
	CreateDiscoveryRun(ctx context.Context, run domain.DiscoveryRun) error
	CompleteDiscoveryRun(ctx context.Context, run domain.DiscoveryRun) error
}

type Service struct {
	repo        Repository
	acquisition *Acquisition
	searcher    Searcher
	cfg         ServiceConfig
}

type ServiceConfig struct {
	DiscoveryEnabled       bool
	TargetUsableItems      int
	DiscoveryMaxQueries    int
	DiscoveryMaxCandidates int
	DiscoveryMaxActive     int
	MinUsableItems         int
	MaxItems               int
	MinItemCharacters      int
	SubstantialCharacters  int
	MaxItemCharacters      int
	RefreshInterval        time.Duration
	DefaultMaxStaleAge     time.Duration
}

func NewService(repo Repository, acquisition *Acquisition, cfg ServiceConfig) *Service {
	if cfg.MinUsableItems < 1 {
		cfg.MinUsableItems = 4
	}
	if cfg.TargetUsableItems < cfg.MinUsableItems {
		cfg.TargetUsableItems = 8
	}
	if cfg.DiscoveryMaxQueries < 1 {
		cfg.DiscoveryMaxQueries = 4
	}
	if cfg.DiscoveryMaxCandidates < 1 {
		cfg.DiscoveryMaxCandidates = 30
	}
	if cfg.DiscoveryMaxActive < 1 {
		cfg.DiscoveryMaxActive = 8
	}
	if cfg.MaxItems < 1 {
		cfg.MaxItems = 18
	}
	if cfg.MinItemCharacters < 1 {
		cfg.MinItemCharacters = 80
	}
	if cfg.SubstantialCharacters < 1 {
		cfg.SubstantialCharacters = 600
	}
	if cfg.MaxItemCharacters < 1 {
		cfg.MaxItemCharacters = 1800
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 12 * time.Hour
	}
	if cfg.DefaultMaxStaleAge <= 0 {
		cfg.DefaultMaxStaleAge = 30 * 24 * time.Hour
	}
	return &Service{repo: repo, acquisition: acquisition, cfg: cfg}
}

func (svc *Service) WithSearcher(searcher Searcher) *Service {
	svc.searcher = searcher
	return svc
}

type PrepareIssueResult struct {
	Items    []domain.SourceItem
	Warnings []string
	HadPrior bool
}

type preparedEvidence struct {
	Item       domain.SourceItem
	SnapshotID string
	CreatedAt  time.Time
}

type snapshotMetadata struct {
	Source string              `json:"source"`
	Origin domain.SourceOrigin `json:"origin"`
}

func (svc *Service) PrepareIssue(
	ctx context.Context,
	newsletter domain.Newsletter,
	issueID string,
) (PrepareIssueResult, error) {
	hasSources, err := svc.repo.HasIssueSources(ctx, issueID)
	if err != nil {
		return PrepareIssueResult{}, fmt.Errorf("check issue sources: %w", err)
	}
	if hasSources {
		snapshots, err := svc.repo.GetIssueSources(ctx, issueID)
		if err != nil {
			return PrepareIssueResult{}, fmt.Errorf("load frozen issue sources: %w", err)
		}
		items := snapshotsToSourceItems(snapshots)
		if len(items) == 0 {
			return PrepareIssueResult{}, errors.New("frozen Issue evidence is empty")
		}
		return PrepareIssueResult{Items: items, HadPrior: true}, nil
	}

	specs, err := svc.repo.ListActiveSourceSpecs(ctx, newsletter.ID)
	if err != nil {
		return PrepareIssueResult{}, fmt.Errorf("list source specs: %w", err)
	}

	var candidates []preparedEvidence
	var warnings []string
	for _, spec := range specs {
		evidence, err := svc.resolveAndSnapshot(ctx, spec)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", sourceName(spec), err))
			continue
		}
		candidates = append(candidates, evidence...)
	}

	selected := svc.selectEvidence(candidates)
	if svc.shouldDiscover(newsletter.SourceMode, selected) {
		discovered, discoveryWarnings, err := svc.discover(
			ctx,
			newsletter,
			issueID,
			specs,
		)
		warnings = append(warnings, discoveryWarnings...)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			candidates = append(candidates, discovered...)
			selected = svc.selectEvidence(candidates)
		}
	}
	if err := svc.validateSufficiency(newsletter.SourceMode, selected, warnings); err != nil {
		return PrepareIssueResult{}, err
	}

	links := make([]domain.IssueSource, len(selected))
	items := make([]domain.SourceItem, len(selected))
	for index := range selected {
		selected[index].Item.SourceID = fmt.Sprintf("S%d", index+1)
		items[index] = selected[index].Item
		links[index] = domain.IssueSource{
			IssueID:          issueID,
			SourceSnapshotID: selected[index].SnapshotID,
			Position:         index,
			CreatedAt:        selected[index].CreatedAt,
		}
	}
	inserted, err := svc.repo.InsertIssueSources(ctx, issueID, links)
	if err != nil {
		return PrepareIssueResult{}, fmt.Errorf("freeze Issue evidence: %w", err)
	}
	if !inserted {
		snapshots, err := svc.repo.GetIssueSources(ctx, issueID)
		if err != nil {
			return PrepareIssueResult{}, fmt.Errorf("load concurrently frozen Issue evidence: %w", err)
		}
		return PrepareIssueResult{
			Items:    snapshotsToSourceItems(snapshots),
			Warnings: warnings,
			HadPrior: true,
		}, nil
	}
	return PrepareIssueResult{Items: items, Warnings: warnings}, nil
}

func (svc *Service) selectEvidence(candidates []preparedEvidence) []preparedEvidence {
	seen := make(map[string]struct{}, len(candidates))
	selected := make([]preparedEvidence, 0, len(candidates))
	for _, candidate := range candidates {
		if len([]rune(strings.TrimSpace(candidate.Item.Summary))) < svc.cfg.MinItemCharacters {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(candidate.Item.Title)) + "|" +
			normalizeEvidenceURL(candidate.Item.CanonicalURL)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, candidate)
	}
	sort.SliceStable(selected, func(i, j int) bool {
		left, right := selected[i].Item.PublishedAt, selected[j].Item.PublishedAt
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return left.After(*right)
	})
	if len(selected) > svc.cfg.MaxItems {
		selected = selected[:svc.cfg.MaxItems]
	}
	return selected
}

func (svc *Service) validateSufficiency(
	mode domain.SourceMode,
	selected []preparedEvidence,
	warnings []string,
) error {
	if svc.hasHardMinimum(selected) {
		return nil
	}
	if len(selected) == 0 {
		return svc.noEvidenceError(mode, warnings)
	}
	return fmt.Errorf(
		"insufficient evidence: %d usable items found, minimum is %d or one substantial exact page",
		len(selected),
		svc.cfg.MinUsableItems,
	)
}

func (svc *Service) hasHardMinimum(selected []preparedEvidence) bool {
	if len(selected) >= svc.cfg.MinUsableItems {
		return true
	}
	for _, evidence := range selected {
		if evidence.Item.ContentSource == "article" &&
			len([]rune(evidence.Item.Summary)) >= svc.cfg.SubstantialCharacters {
			return true
		}
	}
	return false
}

func (svc *Service) resolveAndSnapshot(
	ctx context.Context,
	spec domain.SourceSpec,
) ([]preparedEvidence, error) {
	endpoint, exists, err := svc.repo.GetSourceEndpoint(ctx, spec.ID)
	if err != nil {
		return nil, fmt.Errorf("load source endpoint: %w", err)
	}
	if !exists {
		kind := spec.Kind
		if kind == "" {
			kind = domain.SourceKindHTML
		}
		now := time.Now().UTC()
		endpoint = domain.SourceEndpoint{
			ID:           uuid.NewString(),
			SourceSpecID: spec.ID,
			EndpointURL:  spec.InputURL,
			CanonicalURL: spec.InputURL,
			Kind:         kind,
			Health:       "unknown",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}
	if exists && endpoint.LastCheckedAt != nil &&
		time.Since(*endpoint.LastCheckedAt) < svc.cfg.RefreshInterval {
		snapshots, err := svc.repo.GetSourceSnapshots(ctx, endpoint.ID, itemLimit(spec))
		if err != nil {
			return nil, fmt.Errorf("load fresh source snapshots: %w", err)
		}
		if svc.snapshotsAreUsable(snapshots, time.Now().UTC()) {
			return snapshotsToEvidence(snapshots), nil
		}
	}
	return svc.fetchEndpoint(ctx, spec, endpoint, exists, true)
}

func (svc *Service) fetchEndpoint(
	ctx context.Context,
	spec domain.SourceSpec,
	endpoint domain.SourceEndpoint,
	persisted bool,
	allowAutoFeed bool,
) ([]preparedEvidence, error) {
	maxBytes := max(svc.acquisition.Cfg.MaxFeedBytes, svc.acquisition.Cfg.MaxArticleBytes)
	timeout := max(svc.acquisition.Cfg.FeedTimeout, svc.acquisition.Cfg.ArticleTimeout)
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, fetchErr := doHTTP(
		requestCtx,
		svc.acquisition.Client,
		endpoint.EndpointURL,
		maxBytes,
		"application/atom+xml, application/rss+xml, application/xml, text/xml, application/feed+json, application/json, text/html, text/plain;q=0.9",
		endpoint.ETag,
		endpoint.LastModified,
	)
	now := time.Now().UTC()
	if fetchErr != nil {
		endpoint.ConsecutiveFailures++
		endpoint.LastCheckedAt = &now
		endpoint.LastError = safeError(fetchErr)
		if endpoint.ConsecutiveFailures >= 5 {
			endpoint.Health = "failing"
		}
		if endpoint.Kind == "" {
			endpoint.Kind = domain.SourceKindHTML
		}
		endpoint.UpdatedAt = now
		if _, err := svc.repo.UpsertSourceEndpoint(ctx, endpoint); err != nil {
			return nil, fmt.Errorf("fetch source: %v; persist endpoint failure: %w", fetchErr, err)
		}
		return nil, fetchErr
	}

	endpoint.LastCheckedAt = &now
	endpoint.LastHTTPStatus = result.StatusCode
	endpoint.ConsecutiveFailures = 0
	endpoint.Health = "healthy"
	endpoint.LastError = ""
	endpoint.UpdatedAt = now

	if result.StatusCode == http.StatusNotModified {
		if !persisted {
			return nil, errors.New("source returned 304 without stored endpoint state")
		}
		if result.ETag != "" {
			endpoint.ETag = result.ETag
		}
		if result.LastModified != "" {
			endpoint.LastModified = result.LastModified
		}
		var err error
		endpoint, err = svc.repo.UpsertSourceEndpoint(ctx, endpoint)
		if err != nil {
			return nil, fmt.Errorf("persist source cache hit: %w", err)
		}
		snapshots, err := svc.repo.GetSourceSnapshots(ctx, endpoint.ID, itemLimit(spec))
		if err != nil {
			return nil, fmt.Errorf("load cached source snapshots: %w", err)
		}
		if len(snapshots) == 0 {
			return nil, errors.New("source returned 304 without stored snapshots")
		}
		if !svc.snapshotsAreUsable(snapshots, now) {
			return nil, errors.New("source returned 304 but its stored evidence is too old")
		}
		return snapshotsToEvidence(snapshots), nil
	}

	kind := detectKind(result.ContentType, result.Body)
	if allowAutoFeed && kind == domain.SourceKindHTML {
		if feedURL := findFeedAutoDiscovery(string(result.Body), result.FinalURL); feedURL != "" {
			feedEndpoint := endpoint
			feedEndpoint.EndpointURL = feedURL
			feedEndpoint.CanonicalURL = feedURL
			feedEndpoint.Kind = domain.SourceKindRSS
			feedEndpoint.ETag = ""
			feedEndpoint.LastModified = ""
			return svc.fetchEndpoint(ctx, spec, feedEndpoint, false, false)
		}
	}

	endpoint.Kind = kind
	endpoint.EndpointURL = result.FinalURL.String()
	endpoint.CanonicalURL = result.FinalURL.String()
	endpoint.ETag = result.ETag
	endpoint.LastModified = result.LastModified
	endpoint.LastSuccessAt = &now
	endpoint.LastChangedAt = &now
	var err error
	endpoint, err = svc.repo.UpsertSourceEndpoint(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("persist source endpoint: %w", err)
	}

	switch kind {
	case domain.SourceKindRSS, domain.SourceKindAtom, domain.SourceKindJSONFeed:
		return svc.snapshotFeed(ctx, spec, endpoint, result.Body, result.FinalURL, now)
	case domain.SourceKindHTML, domain.SourceKindText:
		return svc.snapshotExact(ctx, spec, endpoint, result.Body, result.FinalURL, now)
	default:
		return nil, fmt.Errorf("unsupported source kind %q", kind)
	}
}

func (svc *Service) snapshotFeed(
	ctx context.Context,
	spec domain.SourceSpec,
	endpoint domain.SourceEndpoint,
	body []byte,
	finalURL *url.URL,
	now time.Time,
) ([]preparedEvidence, error) {
	items, err := ParseFeed(body, sourceName(spec))
	if err != nil {
		return nil, err
	}
	if len(items) > itemLimit(spec) {
		items = items[:itemLimit(spec)]
	}
	metadata := encodeSnapshotMetadata(spec)
	evidence := make([]preparedEvidence, 0, len(items))
	for _, item := range items {
		if resolved, resolveErr := finalURL.Parse(item.URL); resolveErr == nil {
			item.URL = resolved.String()
			item.CanonicalURL = item.URL
		}
		canonicalURL := item.CanonicalURL
		if canonicalURL == "" {
			canonicalURL = endpoint.CanonicalURL
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = sourceName(spec)
		}
		content := truncateRunes(strings.TrimSpace(item.Summary), svc.cfg.MaxItemCharacters)
		if content == "" {
			continue
		}
		itemKey := normalizeEvidenceURL(canonicalURL)
		if itemKey == "" || itemKey == normalizeEvidenceURL(endpoint.CanonicalURL) {
			itemKey = hashContent(title + "|" + publishedKey(item.PublishedAt))
		}
		snapshot := domain.SourceSnapshot{
			ID:               uuid.NewString(),
			SourceEndpointID: endpoint.ID,
			ItemKey:          itemKey,
			Title:            title,
			CanonicalURL:     canonicalURL,
			Author:           item.Author,
			PublishedAt:      item.PublishedAt,
			Content:          content,
			ContentSource:    "feed-summary",
			ContentSHA256:    hashContent(content),
			Metadata:         metadata,
			FetchedAt:        now,
		}
		snapshotID, err := svc.repo.InsertSourceSnapshot(ctx, snapshot)
		if err != nil {
			return nil, fmt.Errorf("persist source snapshot: %w", err)
		}
		evidence = append(evidence, preparedEvidence{
			Item: domain.SourceItem{
				Source:        sourceName(spec),
				Title:         title,
				URL:           canonicalURL,
				CanonicalURL:  canonicalURL,
				Summary:       content,
				PublishedAt:   item.PublishedAt,
				ContentSource: "feed-summary",
				Author:        item.Author,
			},
			SnapshotID: snapshotID,
			CreatedAt:  now,
		})
	}
	if len(evidence) == 0 {
		return nil, errors.New("feed contained no usable entries")
	}
	return evidence, nil
}

func (svc *Service) snapshotsAreUsable(
	snapshots []domain.SourceSnapshot,
	now time.Time,
) bool {
	if len(snapshots) == 0 {
		return false
	}
	newest := snapshots[0].FetchedAt
	for _, snapshot := range snapshots[1:] {
		if snapshot.FetchedAt.After(newest) {
			newest = snapshot.FetchedAt
		}
	}
	return !newest.IsZero() && now.Sub(newest) <= svc.cfg.DefaultMaxStaleAge
}

func (svc *Service) snapshotExact(
	ctx context.Context,
	spec domain.SourceSpec,
	endpoint domain.SourceEndpoint,
	body []byte,
	finalURL *url.URL,
	now time.Time,
) ([]preparedEvidence, error) {
	text := string(body)
	canonicalURL := finalURL.String()
	contentSource := "article"
	if endpoint.Kind == domain.SourceKindHTML {
		text, canonicalURL, _ = extractArticle(text, finalURL)
	}
	if parsed, err := validateWebURL(canonicalURL); err == nil {
		canonicalURL = parsed.String()
	} else {
		canonicalURL = finalURL.String()
	}
	content := truncateRunes(strings.TrimSpace(text), svc.acquisition.Cfg.MaxArticleCharacters)
	if len([]rune(content)) < svc.cfg.MinItemCharacters {
		return nil, errors.New("exact source content is too short")
	}
	title := sourceName(spec)
	snapshot := domain.SourceSnapshot{
		ID:               uuid.NewString(),
		SourceEndpointID: endpoint.ID,
		ItemKey:          canonicalURL,
		Title:            title,
		CanonicalURL:     canonicalURL,
		Content:          content,
		ContentSource:    contentSource,
		ContentSHA256:    hashContent(content),
		Metadata:         encodeSnapshotMetadata(spec),
		FetchedAt:        now,
	}
	snapshotID, err := svc.repo.InsertSourceSnapshot(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("persist source snapshot: %w", err)
	}
	return []preparedEvidence{{
		Item: domain.SourceItem{
			Source:        title,
			Title:         title,
			URL:           canonicalURL,
			CanonicalURL:  canonicalURL,
			Summary:       content,
			ContentSource: contentSource,
		},
		SnapshotID: snapshotID,
		CreatedAt:  now,
	}}, nil
}

func snapshotsToEvidence(snapshots []domain.SourceSnapshot) []preparedEvidence {
	items := snapshotsToSourceItems(snapshots)
	evidence := make([]preparedEvidence, len(snapshots))
	for index := range snapshots {
		evidence[index] = preparedEvidence{
			Item:       items[index],
			SnapshotID: snapshots[index].ID,
			CreatedAt:  snapshots[index].FetchedAt,
		}
	}
	return evidence
}

func snapshotsToSourceItems(snapshots []domain.SourceSnapshot) []domain.SourceItem {
	items := make([]domain.SourceItem, 0, len(snapshots))
	for index, snapshot := range snapshots {
		var metadata snapshotMetadata
		_ = json.Unmarshal([]byte(snapshot.Metadata), &metadata)
		items = append(items, domain.SourceItem{
			SourceID:      fmt.Sprintf("S%d", index+1),
			Source:        metadata.Source,
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

func (svc *Service) noEvidenceError(mode domain.SourceMode, warnings []string) error {
	switch mode {
	case domain.SourceModeProvided:
		return fmt.Errorf("the supplied sources could not provide enough readable evidence: %s", warningsToErr(warnings))
	case domain.SourceModeHybrid:
		return fmt.Errorf("provided and discovered sources could not provide enough readable evidence: %s", warningsToErr(warnings))
	case domain.SourceModeDiscovered:
		if !svc.cfg.DiscoveryEnabled {
			return errors.New("source discovery is disabled; cannot generate without grounded evidence")
		}
		return fmt.Errorf("source discovery could not provide enough readable evidence: %s", warningsToErr(warnings))
	default:
		return fmt.Errorf("no usable sources found: %s", warningsToErr(warnings))
	}
}

func sourceName(spec domain.SourceSpec) string {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		return value
	}
	return spec.InputURL
}

func itemLimit(spec domain.SourceSpec) int {
	if spec.ItemLimit < 1 {
		return 8
	}
	return spec.ItemLimit
}

func encodeSnapshotMetadata(spec domain.SourceSpec) string {
	value, _ := json.Marshal(snapshotMetadata{Source: sourceName(spec), Origin: spec.Origin})
	return string(value)
}

func normalizeEvidenceURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(value))
	}
	parsed.Fragment = ""
	return strings.ToLower(parsed.String())
}

func publishedKey(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

func warningsToErr(warnings []string) string {
	if len(warnings) == 0 {
		return "unknown error"
	}
	return strings.Join(warnings, "; ")
}
