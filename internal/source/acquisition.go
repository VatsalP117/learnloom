package source

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

const userAgent = "learnloom/1.0 (+https://learnloom.blog)"

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
	client *http.Client
	cfg    Config
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
		cfg: cfg,
		client: &http.Client{
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
			if err != nil || len([]rune(article.Text)) < a.cfg.MinimumArticleChars {
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
	requestCtx, cancel := context.WithTimeout(ctx, a.cfg.FeedTimeout)
	defer cancel()
	body, contentType, finalURL, err := a.get(requestCtx, definition.URL, a.cfg.MaxFeedBytes,
		"application/atom+xml, application/rss+xml, application/xml, text/xml")
	if err != nil {
		return nil, err
	}
	if !isFeedContentType(contentType) {
		return nil, fmt.Errorf("unsupported feed content type %q", contentType)
	}
	items, err := ParseFeed(body, definition.Name)
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
		if resolved, resolveErr := finalURL.Parse(items[index].URL); resolveErr == nil {
			items[index].URL = resolved.String()
			items[index].CanonicalURL = items[index].URL
		}
	}
	return items, nil
}

type article struct {
	Text         string
	CanonicalURL string
	Author       string
}

func (a *Acquisition) fetchArticle(ctx context.Context, rawURL string) (article, error) {
	requestCtx, cancel := context.WithTimeout(ctx, a.cfg.ArticleTimeout)
	defer cancel()
	body, contentType, finalURL, err := a.get(requestCtx, rawURL, a.cfg.MaxArticleBytes,
		"text/html, text/plain;q=0.9")
	if err != nil {
		return article{}, err
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch mediaType {
	case "text/plain":
		return article{
			Text:         truncateRunes(string(body), a.cfg.MaxArticleCharacters),
			CanonicalURL: finalURL.String(),
		}, nil
	case "text/html", "application/xhtml+xml":
		text, canonical, author := extractArticle(string(body), finalURL)
		return article{
			Text:         truncateRunes(text, a.cfg.MaxArticleCharacters),
			CanonicalURL: canonical,
			Author:       author,
		}, nil
	default:
		return article{}, fmt.Errorf("unsupported article content type %q", contentType)
	}
}

func (a *Acquisition) get(
	ctx context.Context,
	rawURL string,
	maxBytes int64,
	accept string,
) ([]byte, string, *url.URL, error) {
	parsed, err := validateWebURL(rawURL)
	if err != nil {
		return nil, "", nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", nil, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", userAgent)
	response, err := a.client.Do(request)
	if err != nil {
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			return nil, "", nil, fmt.Errorf(
				"source request failed during %s: %w",
				requestErr.Op,
				requestErr.Err,
			)
		}
		return nil, "", nil, errors.New("source request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", nil, fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	body, err := readBounded(response.Body, maxBytes)
	if err != nil {
		return nil, "", nil, err
	}
	return body, response.Header.Get("Content-Type"), response.Request.URL, nil
}

func secureTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	resolver := net.DefaultResolver
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve source host: %w", err)
			}
			if len(addresses) == 0 {
				return nil, errors.New("source hostname did not resolve")
			}
			for _, address := range addresses {
				if !isPublicAddress(address) {
					return nil, errors.New("source URL resolves to a non-public address")
				}
			}
			selected := addresses[0].Unmap()
			return dialer.DialContext(ctx, network, net.JoinHostPort(selected.String(), port))
		},
	}
}

func redirectPolicy(maximum int) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, via []*http.Request) error {
		if len(via) > maximum {
			return errors.New("source redirected too many times")
		}
		_, err := validateWebURL(request.URL.String())
		return err
	}
}

func validateWebURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.New("source URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("source URL must use HTTP or HTTPS")
	}
	if parsed.Hostname() == "" || parsed.User != nil {
		return nil, errors.New("source URL must have a host and no credentials")
	}
	if strings.EqualFold(parsed.Hostname(), "localhost") ||
		strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".localhost") {
		return nil, errors.New("source URL resolves to a non-public address")
	}
	if address, err := netip.ParseAddr(parsed.Hostname()); err == nil && !isPublicAddress(address) {
		return nil, errors.New("source URL resolves to a non-public address")
	}
	return parsed, nil
}

var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
	"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
	"2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20",
)

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() ||
		address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() ||
		address.IsUnspecified() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func mustPrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}

func readBounded(reader io.Reader, maximum int64) ([]byte, error) {
	if maximum < 1 {
		return nil, errors.New("response size limit is invalid")
	}
	body, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maximum {
		return nil, errors.New("source exceeded size limit")
	}
	return body, nil
}

func isFeedContentType(value string) bool {
	mediaType, _, _ := mime.ParseMediaType(value)
	switch mediaType {
	case "application/atom+xml", "application/rss+xml", "application/xml",
		"text/xml", "text/plain", "":
		return true
	default:
		return false
	}
}

type rssDocument struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	Content     string `xml:"encoded"`
	Published   string `xml:"pubDate"`
	Date        string `xml:"date"`
}

type atomDocument struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	ID        string     `xml:"id"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Links     []atomLink `xml:"link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func ParseFeed(body []byte, sourceName string) ([]domain.SourceItem, error) {
	var rss rssDocument
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		items := make([]domain.SourceItem, 0, len(rss.Channel.Items))
		for _, input := range rss.Channel.Items {
			link := firstNonEmpty(input.Link, input.GUID)
			if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(link) == "" {
				continue
			}
			items = append(items, domain.SourceItem{
				Source:        sourceName,
				Title:         cleanText(input.Title),
				URL:           strings.TrimSpace(link),
				CanonicalURL:  strings.TrimSpace(link),
				Summary:       cleanText(firstNonEmpty(input.Content, input.Description)),
				PublishedAt:   parseDate(firstNonEmpty(input.Published, input.Date)),
				ContentSource: "feed-summary",
			})
		}
		return items, nil
	}
	var atom atomDocument
	if err := xml.Unmarshal(body, &atom); err != nil {
		return nil, fmt.Errorf("parse feed XML: %w", err)
	}
	items := make([]domain.SourceItem, 0, len(atom.Entries))
	for _, input := range atom.Entries {
		link := input.ID
		for _, candidate := range input.Links {
			if candidate.Rel == "" || candidate.Rel == "alternate" {
				link = candidate.Href
				break
			}
		}
		if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(link) == "" {
			continue
		}
		items = append(items, domain.SourceItem{
			Source:        sourceName,
			Title:         cleanText(input.Title),
			URL:           strings.TrimSpace(link),
			CanonicalURL:  strings.TrimSpace(link),
			Summary:       cleanText(firstNonEmpty(input.Summary, input.Content)),
			PublishedAt:   parseDate(firstNonEmpty(input.Published, input.Updated)),
			ContentSource: "feed-summary",
		})
	}
	if len(items) == 0 {
		return nil, errors.New("feed contains no usable Source Items")
	}
	return items, nil
}

var (
	removeActive = regexp.MustCompile(`(?is)<(?:script|style|noscript|svg|nav|header|footer|form)\b[^>]*>.*?</(?:script|style|noscript|svg|nav|header|footer|form)>`)
	removeTags   = regexp.MustCompile(`(?s)<[^>]+>`)
	whitespace   = regexp.MustCompile(`\s+`)
	articleBody  = regexp.MustCompile(`(?is)<article\b[^>]*>(.*?)</article>`)
	mainBody     = regexp.MustCompile(`(?is)<main\b[^>]*>(.*?)</main>`)
	htmlBody     = regexp.MustCompile(`(?is)<body\b[^>]*>(.*?)</body>`)
	canonicalA   = regexp.MustCompile(`(?is)<link\b[^>]*\brel\s*=\s*["'][^"']*\bcanonical\b[^"']*["'][^>]*\bhref\s*=\s*["']([^"']+)["']`)
	canonicalB   = regexp.MustCompile(`(?is)<link\b[^>]*\bhref\s*=\s*["']([^"']+)["'][^>]*\brel\s*=\s*["'][^"']*\bcanonical\b[^"']*["']`)
	authorA      = regexp.MustCompile(`(?is)<meta\b[^>]*(?:name|property)\s*=\s*["'](?:author|article:author)["'][^>]*content\s*=\s*["']([^"']+)["']`)
	authorB      = regexp.MustCompile(`(?is)<meta\b[^>]*content\s*=\s*["']([^"']+)["'][^>]*(?:name|property)\s*=\s*["'](?:author|article:author)["']`)
)

func extractArticle(raw string, pageURL *url.URL) (string, string, string) {
	primary := firstCapture(articleBody, raw)
	if primary == "" {
		primary = firstCapture(mainBody, raw)
	}
	if primary == "" {
		primary = firstCapture(htmlBody, raw)
	}
	if primary == "" {
		primary = raw
	}
	text := cleanText(removeActive.ReplaceAllString(primary, " "))
	canonical := pageURL.String()
	rawCanonical := firstNonEmpty(firstCapture(canonicalA, raw), firstCapture(canonicalB, raw))
	if rawCanonical != "" {
		if parsed, err := pageURL.Parse(html.UnescapeString(rawCanonical)); err == nil &&
			(parsed.Scheme == "http" || parsed.Scheme == "https") {
			canonical = parsed.String()
		}
	}
	author := cleanText(firstNonEmpty(firstCapture(authorA, raw), firstCapture(authorB, raw)))
	return text, canonical, truncateRunes(author, 300)
}

func cleanText(value string) string {
	return strings.TrimSpace(whitespace.ReplaceAllString(
		html.UnescapeString(removeTags.ReplaceAllString(value, " ")), " ",
	))
}

func firstCapture(expression *regexp.Regexp, value string) string {
	match := expression.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func parseDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339, time.RFC3339Nano, time.RFC1123Z, time.RFC1123,
		time.RFC822Z, time.RFC822, time.RFC850, time.ANSIC,
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateRunes(value string, maximum int) string {
	runes := []rune(value)
	if maximum <= 0 || len(runes) <= maximum {
		return value
	}
	suffix := "\n[truncated]"
	limit := maximum - len([]rune(suffix))
	if limit < 0 {
		return string([]rune(suffix)[:maximum])
	}
	return strings.TrimRight(string(runes[:limit]), " \t\r\n") + suffix
}

func safeError(err error) string {
	return truncateRunes(whitespace.ReplaceAllString(err.Error(), " "), 240)
}
