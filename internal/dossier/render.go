package dossier

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/VatsalP117/learnloom/internal/domain"
)

var (
	strongPattern = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	codePattern   = regexp.MustCompile("`([^`]+)`")
)

func RenderMarkdown(dossier domain.Dossier) string {
	var output strings.Builder
	fmt.Fprintf(&output, "# Learning Dossier — %s\n\n", dossier.Date)
	fmt.Fprintf(
		&output,
		"> Generated from %d Source Items through curation, source enrichment, Learning Blueprint, research, skepticism, teaching, practice, and editorial validation.\n\n",
		len(dossier.Sources),
	)
	for _, section := range []string{dossier.Lesson, dossier.Critique, dossier.Practice} {
		output.WriteString(demoteHeading(section))
		output.WriteString("\n\n")
	}
	if dossier.Exploration != nil {
		output.WriteString("## AI Exploration\n\n")
		output.WriteString("> Opt-in synthetic exploration. These analogies, deductions, and scenarios extend beyond the cited sources and may be speculative.\n\n")
		output.WriteString(demoteHeading(*dossier.Exploration))
		output.WriteString("\n\n")
	}
	output.WriteString("## Source Index\n\n")
	for index, item := range dossier.Sources {
		sourceID := firstText(item.SourceID, fmt.Sprintf("S%d", index+1))
		fmt.Fprintf(
			&output,
			"%d. **[%s] %s** — %s  \n   %s\n",
			index+1,
			sourceID,
			escapeMarkdown(item.Title),
			item.Source,
			firstText(item.CanonicalURL, item.URL),
		)
	}
	fmt.Fprintf(
		&output,
		"\nQuality gate: %d/100 · %d enriched sources · %d retrieval questions\n",
		dossier.Quality.Score,
		dossier.Quality.Metrics["enrichedSources"],
		dossier.Quality.Metrics["retrievalQuestions"],
	)
	output.WriteString("\n---\n\n")
	fmt.Fprintf(
		&output,
		"Generated at %s · Model output can be wrong; verify important claims at the linked sources.\n",
		dossier.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
	)
	return output.String()
}

func RenderHTML(dossier domain.Dossier, webURL string) string {
	sections := []string{
		renderMarkdownFragment(dossier.Lesson),
		renderMarkdownFragment(dossier.Critique),
		renderMarkdownFragment(dossier.Practice),
	}
	var exploration string
	if dossier.Exploration != nil {
		exploration = `<section style="margin:32px 0 0;padding:24px;border:1px solid #f0c36a;border-radius:12px;background:#fff8e8">
<p style="margin:0 0 8px;color:#9a5b13;font-size:11px;font-weight:800;letter-spacing:.12em;text-transform:uppercase">AI Exploration · Opt-in</p>
<p style="margin:0 0 18px;color:#7c5a2d;font-size:13px;line-height:1.5">Synthetic analogies, deductions, and scenarios that extend beyond cited sources. They may be speculative.</p>` +
			renderMarkdownFragment(*dossier.Exploration) + `</section>`
	}
	var sources strings.Builder
	for index, item := range dossier.Sources {
		sourceID := firstText(item.SourceID, fmt.Sprintf("S%d", index+1))
		link := safeHTTPURL(firstText(item.CanonicalURL, item.URL))
		fmt.Fprintf(
			&sources,
			`<li style="margin:0 0 10px"><strong>[%s] %s</strong> — %s`,
			html.EscapeString(sourceID),
			html.EscapeString(item.Title),
			html.EscapeString(item.Source),
		)
		if link != "" {
			fmt.Fprintf(
				&sources,
				`<br><a href="%s" style="color:#047857">%s</a>`,
				html.EscapeString(link),
				html.EscapeString(link),
			)
		}
		sources.WriteString("</li>")
	}
	var webLink string
	if link := safeHTTPURL(webURL); link != "" {
		webLink = fmt.Sprintf(
			`<p style="margin:0 0 28px"><a href="%s" style="display:inline-block;padding:11px 17px;border-radius:999px;background:#047857;color:#fff;font-weight:700;text-decoration:none">Read on the web</a></p>`,
			html.EscapeString(link),
		)
	}
	return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` +
		html.EscapeString(dossier.Title) + `</title></head>
<body style="margin:0;background:#f5f5f4;color:#0f172a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">
<main style="max-width:720px;margin:0 auto;padding:32px 20px">
<div style="background:#fff;border:1px solid #e7e5e4;border-radius:16px;padding:32px">
<p style="margin:0 0 8px;color:#047857;font-size:12px;font-weight:700;letter-spacing:.12em;text-transform:uppercase">Learnloom · ` +
		html.EscapeString(dossier.Date) + `</p>
<h1 style="margin:0 0 28px;font-size:30px;line-height:1.2">` +
		html.EscapeString(dossier.Title) + `</h1>` +
		webLink +
		strings.Join(sections, `<hr style="border:0;border-top:1px solid #e7e5e4;margin:32px 0">`) +
		exploration +
		`<hr style="border:0;border-top:1px solid #e7e5e4;margin:32px 0">
<h2 style="font-size:20px">Sources</h2><ol style="padding-left:22px">` +
		sources.String() + `</ol>
<p style="margin-top:28px;color:#78716c;font-size:12px">Model output can be wrong. Verify important claims at linked sources.</p>
</div></main></body></html>`
}

func renderMarkdownFragment(markdown string) string {
	var output strings.Builder
	listType := ""
	closeList := func() {
		if listType != "" {
			fmt.Fprintf(&output, "</%s>", listType)
			listType = ""
		}
	}
	inDetails := false
	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			closeList()
			continue
		}
		if strings.EqualFold(line, "<details>") {
			closeList()
			inDetails = true
			output.WriteString(`<details style="margin:20px 0">`)
			continue
		}
		if strings.EqualFold(line, "</details>") {
			closeList()
			if inDetails {
				output.WriteString("</details>")
				inDetails = false
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "<summary>") &&
			strings.HasSuffix(strings.ToLower(line), "</summary>") {
			closeList()
			value := line[len("<summary>") : len(line)-len("</summary>")]
			output.WriteString(`<summary style="cursor:pointer;font-weight:700">` + formatInline(value) + `</summary>`)
			continue
		}
		if match := headingPattern.FindStringSubmatch(line); len(match) > 0 {
			closeList()
			level := min(len(match[1])+1, 4)
			fmt.Fprintf(
				&output,
				`<h%d style="margin:24px 0 10px">%s</h%d>`,
				level,
				formatInline(match[2]),
				level,
			)
			continue
		}
		unordered := strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
		ordered := answerPattern.MatchString(line)
		if unordered || ordered {
			wanted := "ul"
			value := strings.TrimSpace(line[2:])
			if ordered {
				wanted = "ol"
				value = answerPattern.FindStringSubmatch(line)[2]
			}
			if listType != wanted {
				closeList()
				listType = wanted
				fmt.Fprintf(&output, `<%s style="padding-left:24px">`, wanted)
			}
			output.WriteString(`<li style="margin:0 0 8px">` + formatInline(value) + "</li>")
			continue
		}
		closeList()
		output.WriteString(`<p style="margin:0 0 14px;line-height:1.65">` + formatInline(line) + "</p>")
	}
	closeList()
	if inDetails {
		output.WriteString("</details>")
	}
	return output.String()
}

func formatInline(value string) string {
	value = html.EscapeString(value)
	value = strongPattern.ReplaceAllString(value, "<strong>$1</strong>")
	return codePattern.ReplaceAllString(
		value,
		`<code style="background:#f5f5f4;padding:1px 4px;border-radius:4px">$1</code>`,
	)
}

func safeHTTPURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return parsed.String()
}

func demoteHeading(value string) string {
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		if strings.HasPrefix(line, "# ") {
			line = "#" + line
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func escapeMarkdown(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_",
	)
	return replacer.Replace(value)
}
