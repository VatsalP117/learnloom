package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SearXNGConfig struct {
	BaseURL          string
	Timeout          time.Duration
	MaxResponseBytes int64
	SafeSearch       int
}

type SearXNG struct {
	baseURL *url.URL
	client  *http.Client
	cfg     SearXNGConfig
}

func NewSearXNG(cfg SearXNGConfig) (*SearXNG, error) {
	baseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || baseURL.Host == "" || baseURL.User != nil ||
		(baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, errors.New("SEARXNG_BASE_URL must be an HTTP(S) origin without credentials")
	}
	if baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, errors.New("SEARXNG_BASE_URL must not contain a query or fragment")
	}
	if baseURL.Path != "" && baseURL.Path != "/" {
		return nil, errors.New("SEARXNG_BASE_URL must be an origin without a path")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = 2 << 20
	}
	if cfg.SafeSearch < 1 || cfg.SafeSearch > 2 {
		cfg.SafeSearch = 1
	}
	return &SearXNG{
		baseURL: baseURL,
		client:  &http.Client{Timeout: cfg.Timeout},
		cfg:     cfg,
	}, nil
}

func (client *SearXNG) Search(
	ctx context.Context,
	request SearchRequest,
) ([]SearchCandidate, error) {
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return nil, errors.New("SearXNG query is empty")
	}
	endpoint := client.baseURL.ResolveReference(&url.URL{Path: "search"})
	values := endpoint.Query()
	values.Set("q", query)
	values.Set("format", "json")
	values.Set("safesearch", strconv.Itoa(client.cfg.SafeSearch))
	if request.Language != "" {
		values.Set("language", request.Language)
	}
	if request.Category != "" {
		values.Set("categories", request.Category)
	}
	page := request.Page
	if page < 1 {
		page = 1
	}
	values.Set("pageno", strconv.Itoa(page))
	endpoint.RawQuery = values.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, errors.New("create SearXNG request")
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("User-Agent", userAgent)
	response, err := client.client.Do(httpRequest)
	if err != nil {
		return nil, errors.New("SearXNG request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusForbidden:
			return nil, errors.New("SearXNG JSON format is disabled")
		case http.StatusTooManyRequests:
			return nil, errors.New("SearXNG is rate limited")
		default:
			return nil, fmt.Errorf("SearXNG returned HTTP %d", response.StatusCode)
		}
	}
	body, err := readBounded(response.Body, client.cfg.MaxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("read SearXNG response: %w", err)
	}
	var payload struct {
		Results []struct {
			Title         string   `json:"title"`
			URL           string   `json:"url"`
			Content       string   `json:"content"`
			Engine        string   `json:"engine"`
			Engines       []string `json:"engines"`
			PublishedDate string   `json:"publishedDate"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, errors.New("SearXNG returned malformed JSON")
	}
	results := make([]SearchCandidate, 0, len(payload.Results))
	for index, item := range payload.Results {
		if _, err := validateWebURL(item.URL); err != nil {
			continue
		}
		engines := item.Engines
		if len(engines) == 0 && item.Engine != "" {
			engines = []string{item.Engine}
		}
		var publishedAt *time.Time
		if value, err := time.Parse(time.RFC3339, item.PublishedDate); err == nil {
			publishedAt = &value
		}
		results = append(results, SearchCandidate{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Snippet:     strings.TrimSpace(item.Content),
			Engines:     engines,
			Rank:        index + 1,
			PublishedAt: publishedAt,
		})
	}
	return results, nil
}
