package httpapp

import (
	"html"
	"strings"
)

const (
	socialImagePath = "/social-preview.png"
	socialImageAlt  = "Learnloom — give us a topic, and we’ll build the learning path."
)

func socialImageURL(apexOrigin string) string {
	origin := strings.TrimRight(apexOrigin, "/")
	if origin == "" {
		origin = "https://learnloom.blog"
	}
	return origin + socialImagePath
}

func renderSocialImageMetadata(apexOrigin string) string {
	imageURL := html.EscapeString(socialImageURL(apexOrigin))
	alt := html.EscapeString(socialImageAlt)
	return `<meta property="og:image" content="` + imageURL + `">` +
		`<meta property="og:image:secure_url" content="` + imageURL + `">` +
		`<meta property="og:image:type" content="image/png">` +
		`<meta property="og:image:width" content="1200">` +
		`<meta property="og:image:height" content="630">` +
		`<meta property="og:image:alt" content="` + alt + `">` +
		`<meta name="twitter:card" content="summary_large_image">` +
		`<meta name="twitter:image" content="` + imageURL + `">` +
		`<meta name="twitter:image:alt" content="` + alt + `">`
}
