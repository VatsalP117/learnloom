package source

import (
	"html"
	"mime"
	"net/url"
	"regexp"
	"strings"

	"github.com/VatsalP117/learnloom/internal/domain"
)

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
	feedAutoRSS  = regexp.MustCompile(`(?is)<link\b[^>]*\btype\s*=\s*["']application\/(?:rss|atom)\+xml["'][^>]*\bhref\s*=\s*["']([^"']+)["']`)
	feedAutoAtom = regexp.MustCompile(`(?is)<link\b[^>]*\bhref\s*=\s*["']([^"']+)["'][^>]*\btype\s*=\s*["']application\/(?:rss|atom)\+xml["']`)
	feedAutoJSON = regexp.MustCompile(`(?is)<link\b[^>]*\btype\s*=\s*["']application\/(?:feed\+)?json["'][^>]*\bhref\s*=\s*["']([^"']+)["']`)
)

type article struct {
	Text         string
	CanonicalURL string
	Author       string
	FeedURL      string
}

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

func findFeedAutoDiscovery(raw string, pageURL *url.URL) string {
	rawFeed := firstNonEmpty(
		firstCapture(feedAutoRSS, raw),
		firstCapture(feedAutoAtom, raw),
		firstCapture(feedAutoJSON, raw),
	)
	if rawFeed == "" {
		return ""
	}
	if parsed, err := pageURL.Parse(html.UnescapeString(rawFeed)); err == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") {
		return parsed.String()
	}
	return ""
}

func resolveCanonicalURL(raw string, pageURL *url.URL) string {
	rawCanonical := firstNonEmpty(firstCapture(canonicalA, raw), firstCapture(canonicalB, raw))
	if rawCanonical != "" {
		if parsed, err := pageURL.Parse(html.UnescapeString(rawCanonical)); err == nil &&
			(parsed.Scheme == "http" || parsed.Scheme == "https") {
			return parsed.String()
		}
	}
	return pageURL.String()
}

func detectKind(contentType string, body []byte) domain.SourceKind {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch mediaType {
	case "text/plain":
		return domain.SourceKindText
	case "application/pdf":
		return domain.SourceKindPDF
	case "application/atom+xml":
		return domain.SourceKindAtom
	case "application/rss+xml":
		return domain.SourceKindRSS
	case "application/feed+json":
		return domain.SourceKindJSONFeed
	}
	bodyStr := string(body)
	if strings.HasPrefix(strings.TrimSpace(bodyStr), "<?xml") ||
		strings.Contains(strings.ToLower(bodyStr[:min(len(bodyStr), 200)]), "<rss") {
		return domain.SourceKindRSS
	}
	if strings.Contains(strings.ToLower(bodyStr[:min(len(bodyStr), 200)]), "<feed") {
		return domain.SourceKindAtom
	}
	if strings.Contains(strings.TrimSpace(bodyStr[:min(len(bodyStr), 50)]), `"version"`) {
		return domain.SourceKindJSONFeed
	}
	return domain.SourceKindHTML
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
