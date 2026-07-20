package source

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

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

type jsonFeed struct {
	Version string         `json:"version"`
	Items   []jsonFeedItem `json:"items"`
}

type jsonFeedItem struct {
	ID            string           `json:"id"`
	URL           string           `json:"url"`
	ExternalURL   string           `json:"external_url"`
	Title         string           `json:"title"`
	ContentHTML   string           `json:"content_html"`
	ContentText   string           `json:"content_text"`
	Summary       string           `json:"summary"`
	DatePublished string           `json:"date_published"`
	DateModified  string           `json:"date_modified"`
	Authors       []jsonFeedAuthor `json:"authors"`
}

type jsonFeedAuthor struct {
	Name string `json:"name"`
}

func ParseFeed(body []byte, sourceName string) ([]domain.SourceItem, error) {
	if items, err := parseRSS(body, sourceName); err == nil {
		return items, nil
	}
	if items, err := parseAtom(body, sourceName); err == nil {
		return items, nil
	}
	if items, err := parseJSONFeed(body, sourceName); err == nil {
		return items, nil
	}
	return nil, errors.New("could not parse feed as RSS, Atom, or JSON Feed")
}

func parseRSS(body []byte, sourceName string) ([]domain.SourceItem, error) {
	var rss rssDocument
	if err := xml.Unmarshal(body, &rss); err != nil || len(rss.Channel.Items) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("RSS has no items")
	}
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

func parseAtom(body []byte, sourceName string) ([]domain.SourceItem, error) {
	var atom atomDocument
	if err := xml.Unmarshal(body, &atom); err != nil {
		return nil, err
	}
	if len(atom.Entries) == 0 {
		return nil, errors.New("Atom feed has no entries")
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

func parseJSONFeed(body []byte, sourceName string) ([]domain.SourceItem, error) {
	var feed jsonFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	if feed.Version == "" || len(feed.Items) == 0 {
		return nil, errors.New("JSON Feed has no items")
	}
	items := make([]domain.SourceItem, 0, len(feed.Items))
	for _, input := range feed.Items {
		itemURL := firstNonEmpty(input.URL, input.ExternalURL, input.ID)
		if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(itemURL) == "" {
			continue
		}
		summary := firstNonEmpty(input.Summary, input.ContentText,
			cleanText(input.ContentHTML))
		published := firstNonEmpty(input.DatePublished, input.DateModified)
		author := ""
		if len(input.Authors) > 0 {
			author = input.Authors[0].Name
		}
		items = append(items, domain.SourceItem{
			Source:        sourceName,
			Title:         cleanText(input.Title),
			URL:           strings.TrimSpace(itemURL),
			CanonicalURL:  strings.TrimSpace(itemURL),
			Summary:       cleanText(summary),
			PublishedAt:   parseDate(published),
			ContentSource: "feed-summary",
			Author:        author,
		})
	}
	if len(items) == 0 {
		return nil, errors.New("JSON feed contains no usable Source Items")
	}
	return items, nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
