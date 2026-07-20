package source

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type Config struct {
	FeedTimeout          time.Duration
	ArticleTimeout       time.Duration
	MaxFeedBytes         int64
	MaxArticleBytes      int64
	MaxArticleCharacters int
	MaxRedirects         int
	MinimumArticleChars  int
}

type Acquisition struct {
	Client *http.Client
	Cfg    Config
}

func New(cfg Config) *Acquisition {
	if cfg.FeedTimeout == 0 {
		cfg.FeedTimeout = 20 * time.Second
	}
	if cfg.ArticleTimeout == 0 {
		cfg.ArticleTimeout = 15 * time.Second
	}
	if cfg.MaxFeedBytes == 0 {
		cfg.MaxFeedBytes = 2 << 20
	}
	if cfg.MaxArticleBytes == 0 {
		cfg.MaxArticleBytes = 512 << 10
	}
	if cfg.MaxArticleCharacters == 0 {
		cfg.MaxArticleCharacters = 16_000
	}
	if cfg.MaxRedirects == 0 {
		cfg.MaxRedirects = 3
	}
	if cfg.MinimumArticleChars == 0 {
		cfg.MinimumArticleChars = 400
	}
	return &Acquisition{
		Cfg: cfg,
		Client: &http.Client{
			Transport:     secureTransport(),
			CheckRedirect: redirectPolicy(cfg.MaxRedirects),
		},
	}
}

func (a *Acquisition) Fetch(
	ctx context.Context,
	definitions []domain.SourceDefinition,
	maxItems int,
) ([]domain.SourceItem, []string, error) {
	if len(definitions) == 0 {
		return nil, nil, errors.New("at least one source definition is required")
	}
	type outcome struct {
		index int
		items []domain.SourceItem
		err   error
	}
	results := make(chan outcome, len(definitions))
	var wg sync.WaitGroup
	for index, definition := range definitions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := a.fetchFeed(ctx, definition)
			results <- outcome{index: index, items: items, err: err}
		}()
	}
	wg.Wait()
	close(results)

	ordered := make([]outcome, len(definitions))
	for result := range results {
		ordered[result.index] = result
	}
	var items []domain.SourceItem
	var warnings []string
	for index, result := range ordered {
		if result.err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", definitions[index].Name, result.err))
			continue
		}
		items = append(items, result.items...)
	}
	if len(items) == 0 {
		return nil, warnings, fmt.Errorf("no Source Items could be loaded: %s", strings.Join(warnings, "; "))
	}
	items = deduplicate(items)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].PublishedAt == nil {
			return false
		}
		if items[j].PublishedAt == nil {
			return true
		}
		return items[i].PublishedAt.After(*items[j].PublishedAt)
	})
	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}
	for index := range items {
		items[index].SourceID = fmt.Sprintf("S%d", index+1)
	}
	return items, warnings, nil
}

func (a *Acquisition) Enrich(
	ctx context.Context,
	items []domain.SourceItem,
) ([]domain.SourceItem, error) {
	if len(items) == 0 {
		return nil, errors.New("Source Item enrichment requires at least one item")
	}
	enriched := make([]domain.SourceItem, len(items))
	var wg sync.WaitGroup
	for index := range items {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item := items[index]
			article, err := a.fetchArticle(ctx, item.URL)
			if err != nil || len([]rune(article.Text)) < a.Cfg.MinimumArticleChars {
				item.ContentSource = "feed-summary"
				item.CanonicalURL = item.URL
				if err != nil {
					item.EnrichmentError = safeError(err)
				} else {
					item.EnrichmentError = "article text was too short"
				}
				enriched[index] = item
				return
			}
			item.Summary = article.Text
			item.ContentSource = "article"
			item.CanonicalURL = article.CanonicalURL
			item.Author = article.Author
			item.EnrichmentError = ""
			enriched[index] = item
		}()
	}
	wg.Wait()
	return enriched, nil
}

func (a *Acquisition) fetchFeed(
	ctx context.Context,
	definition domain.SourceDefinition,
) ([]domain.SourceItem, error) {
	requestCtx, cancel := context.WithTimeout(ctx, a.Cfg.FeedTimeout)
	defer cancel()
	result, err := doHTTP(requestCtx, a.Client, definition.URL, a.Cfg.MaxFeedBytes,
		"application/atom+xml, application/rss+xml, application/xml, text/xml, application/feed+json, application/json", "", "")
	if err != nil {
		return nil, err
	}
	if result.StatusCode == http.StatusNotModified {
		return nil, errors.New("source returned 304 Not Modified without stored state")
	}
	if !isFeedContentType(result.ContentType) {
		return nil, fmt.Errorf("unsupported content type %q", result.ContentType)
	}
	items, err := ParseFeed(result.Body, definition.Name)
	if err != nil {
		return nil, err
	}
	limit := definition.Limit
	if limit <= 0 {
		limit = 10
	}
	if len(items) > limit {
		items = items[:limit]
	}
	for index := range items {
		if resolved, resolveErr := result.FinalURL.Parse(items[index].URL); resolveErr == nil {
			items[index].URL = resolved.String()
			items[index].CanonicalURL = items[index].URL
		}
	}
	return items, nil
}

func (a *Acquisition) fetchArticle(ctx context.Context, rawURL string) (article, error) {
	requestCtx, cancel := context.WithTimeout(ctx, a.Cfg.ArticleTimeout)
	defer cancel()
	result, err := doHTTP(requestCtx, a.Client, rawURL, a.Cfg.MaxArticleBytes,
		"text/html, text/plain;q=0.9", "", "")
	if err != nil {
		return article{}, err
	}
	if result.StatusCode == http.StatusNotModified {
		return article{}, errors.New("source returned 304 Not Modified without stored state")
	}
	mediaType, _, _ := mime.ParseMediaType(result.ContentType)
	switch mediaType {
	case "text/plain":
		return article{
			Text:         truncateRunes(string(result.Body), a.Cfg.MaxArticleCharacters),
			CanonicalURL: result.FinalURL.String(),
		}, nil
	case "text/html", "application/xhtml+xml":
		text, canonical, author := extractArticle(string(result.Body), result.FinalURL)
		return article{
			Text:         truncateRunes(text, a.Cfg.MaxArticleCharacters),
			CanonicalURL: canonical,
			Author:       author,
		}, nil
	default:
		return article{}, fmt.Errorf("unsupported article content type %q", result.ContentType)
	}
}

func deduplicate(items []domain.SourceItem) []domain.SourceItem {
	seen := make(map[string]struct{}, len(items))
	result := make([]domain.SourceItem, 0, len(items))
	for _, item := range items {
		parsed, _ := url.Parse(item.URL)
		if parsed != nil {
			parsed.RawQuery = ""
			parsed.Fragment = ""
		}
		key := strings.ToLower(item.Title) + "\n"
		if parsed != nil {
			key += parsed.String()
		} else {
			key += item.URL
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func (a *Acquisition) resolveKind(ctx context.Context, rawURL string) domain.SourceKind {
	result, err := doHTTP(ctx, a.Client, rawURL, 4096,
		"application/atom+xml, application/rss+xml, application/xml, text/xml, application/feed+json, application/json, text/html, text/plain;q=0.9",
		"", "")
	if err != nil {
		return domain.SourceKindHTML
	}
	return detectKind(result.ContentType, result.Body)
}

func (a *Acquisition) findAutoFeed(ctx context.Context, rawURL string) (string, error) {
	result, err := doHTTP(ctx, a.Client, rawURL, 65536,
		"text/html;q=1.0", "", "")
	if err != nil {
		return "", err
	}
	return findFeedAutoDiscovery(string(result.Body), result.FinalURL), nil
}
