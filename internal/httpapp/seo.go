package httpapp

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
)

type seoPage struct {
	Path        string
	Eyebrow     string
	Title       string
	Description string
	Lead        string
	Problem     string
	Body        []string
	Steps       []seoStep
	Benefits    []seoBenefit
	For         []string
	Related     []string
}

type seoStep struct {
	Title string
	Copy  string
}

type seoBenefit struct {
	Title string
	Copy  string
}

var seoPages = []seoPage{
	{
		Path:        "/solutions",
		Eyebrow:     "Learning problems, solved deliberately",
		Title:       "Turn information into a learning practice",
		Description: "Give Learnloom a topic. It establishes a useful source environment, builds a progressive learning path, and helps each lesson build on the last.",
		Lead:        "Finding information is easy. Building durable understanding from it is the difficult part. Learnloom maintains the loop between discovery, explanation, recall, and a lasting archive.",
		Problem:     "A calmer answer to information overload",
		Body: []string{
			"Most learning tools solve only one step. Feeds find new material, read-later apps save it, summaries shorten it, and note-taking tools store fragments. The learner still has to decide what matters and connect it to everything they encountered before.",
			"Learnloom turns a topic and a trusted information environment into a continuing sequence of source-grounded lessons. Each lesson is designed to explain a mechanism, make it concrete, question its limits, and give the learner a way to retrieve the idea later.",
		},
		Steps: []seoStep{
			{Title: "Choose an intent", Copy: "Describe the subject, current level, and outcome you want to work toward."},
			{Title: "Establish the source environment", Copy: "Let Learnloom discover and validate useful sources, bring your own, or combine both approaches."},
			{Title: "Build continuity", Copy: "New Dossiers become part of a learning history instead of disappearing into another feed."},
		},
		Benefits: []seoBenefit{
			{Title: "Remember what you read", Copy: "Retrieval questions and continuity make reading an active learning session."},
			{Title: "Keep up without chasing everything", Copy: "A small, worthwhile lesson replaces an endless stream of updates."},
			{Title: "Own a lasting curriculum", Copy: "Every completed Dossier returns to a searchable personal learning home."},
		},
		For:     []string{"Curious professionals following fast-moving fields", "Independent learners building knowledge over months", "Readers who value evidence, nuance, and source visibility"},
		Related: []string{"/solutions/remember-what-you-read", "/solutions/keep-up-with-your-field", "/solutions/build-a-personal-learning-curriculum"},
	},
	{
		Path:        "/solutions/remember-what-you-read",
		Eyebrow:     "For readers tired of forgetting",
		Title:       "Remember what you read—and build on it",
		Description: "Turn articles and trusted sources into connected lessons with retrieval practice, examples, citations, and a lasting learning history.",
		Lead:        "Reading more is not the same as learning more. Learnloom helps turn worthwhile material into mental models you can retrieve, apply, and connect to what comes next.",
		Problem:     "Why useful reading fades so quickly",
		Body: []string{
			"Articles are usually consumed as isolated events. Even a clear explanation can disappear when there is no reason to retrieve it, no worked example to test it against, and no place for it in a larger structure.",
			"Learnloom prepares a Knowledge Dossier from current source material. A Dossier includes the central mechanism, necessary context, a concrete example, skeptical review, common misconceptions, and retrieval questions. The original sources remain attached so brevity does not come at the expense of evidence.",
		},
		Steps: []seoStep{
			{Title: "Explain the mechanism", Copy: "Identify the causal model or central idea worth remembering, not merely a list of facts."},
			{Title: "Make it concrete", Copy: "Use a worked example and limitations to show where the model helps—and where it does not."},
			{Title: "Retrieve and reconnect", Copy: "Practice recalling the idea and connect later Dossiers to concepts already covered."},
		},
		Benefits: []seoBenefit{
			{Title: "Less passive consumption", Copy: "Retrieval prompts reveal whether an explanation actually became usable knowledge."},
			{Title: "Fewer disconnected notes", Copy: "Lessons live together in a chronological, topic-based learning history."},
			{Title: "Evidence remains visible", Copy: "Citations and source links stay with the lesson for verification and deeper reading."},
		},
		For:     []string{"People who save more articles than they revisit", "Professionals who need to apply what they read", "Independent learners who want durable understanding"},
		Related: []string{"/product/ai-learning-assistant", "/solutions/turn-information-into-understanding", "/solutions/build-a-personal-learning-curriculum"},
	},
	{
		Path:        "/solutions/keep-up-with-your-field",
		Eyebrow:     "Stay current without living in a feed",
		Title:       "Keep up with your field without information overload",
		Description: "Give Learnloom a changing field to follow and turn meaningful developments into coherent lessons instead of another noisy feed or crowded inbox.",
		Lead:        "A changing field produces more updates than any thoughtful person can follow. Learnloom narrows the stream to what deserves understanding and turns it into a lesson that builds context over time.",
		Problem:     "The cost of staying current",
		Body: []string{
			"News feeds reward novelty and engagement. Newsletters compete for attention. Search answers one question at a time. None of them is responsible for deciding whether an update is well supported, genuinely new, or important to your longer-term learning goal.",
			"With Learnloom, you define the topic, level, and outcome. Learnloom can establish the information environment for you, follow only sources you choose, or fill gaps around them. It prepares a focused Dossier when there is something worthwhile to learn. Daily is a rhythm, not a demand to manufacture novelty.",
		},
		Steps: []seoStep{
			{Title: "Define the intent", Copy: "Choose the question you want to follow; let Learnloom establish the source environment or set your own boundary."},
			{Title: "Reduce repetition", Copy: "Evaluate new material in the context of what your learning stream has already covered."},
			{Title: "Turn updates into context", Copy: "Explain why a development matters, what mechanism drives it, and what remains uncertain."},
		},
		Benefits: []seoBenefit{
			{Title: "Signal over volume", Copy: "A focused lesson replaces a queue of tabs, newsletters, and repeated explainers."},
			{Title: "Current and cumulative", Copy: "New developments deepen a living curriculum rather than resetting the learner to the basics."},
			{Title: "Control over sources", Copy: "Your trusted publications and experts can remain the foundation of the information environment."},
		},
		For:     []string{"Researchers and operators in changing industries", "Leaders who need context, not headline monitoring", "Specialists following two to five important subjects"},
		Related: []string{"/product/trusted-source-learning", "/solutions/remember-what-you-read", "/compare/ai-summaries"},
	},
	{
		Path:        "/solutions/build-a-personal-learning-curriculum",
		Eyebrow:     "A curriculum that follows the changing world",
		Title:       "Build a personal learning curriculum over time",
		Description: "Start with a topic, current level, and learning goal. Learnloom establishes an evolving curriculum with each lesson building on the last.",
		Lead:        "A fixed course is coherent but can become dated. The open web is current but fragmented. Learnloom combines current sources with the continuity of a curriculum.",
		Problem:     "Move beyond isolated courses and disconnected notes",
		Body: []string{
			"Independent learning often alternates between broad courses and unstructured browsing. Courses provide sequence but may not follow new developments. Browsing provides recency but rarely establishes prerequisites, progression, or retrieval.",
			"A Learnloom learning stream starts with an intent: the subject, what you already understand, and what capability you want to develop. Each new Dossier contributes a bounded learning history that helps later lessons avoid unnecessary repetition, fill prerequisite gaps, and increase depth gradually.",
		},
		Steps: []seoStep{
			{Title: "Set a destination", Copy: "Describe what progress would look like instead of choosing only a broad topic label."},
			{Title: "Learn at the right depth", Copy: "Use your current level and lesson length to shape explanation and pacing."},
			{Title: "Grow a coherent archive", Copy: "Organize completed lessons by stream and connect new ideas to prior concepts."},
		},
		Benefits: []seoBenefit{
			{Title: "Personal progression", Copy: "The learning path reflects your goal and history rather than a generic syllabus."},
			{Title: "Responsive to new evidence", Copy: "A living curriculum can incorporate meaningful developments from current sources."},
			{Title: "A body of work", Copy: "Your personal learning home becomes a durable, optionally shareable record of inquiry."},
		},
		For:     []string{"Self-directed learners with a long-term subject", "Career changers building depth in a new domain", "Curious generalists maintaining several learning streams"},
		Related: []string{"/solutions", "/product/ai-learning-assistant", "/solutions/keep-up-with-your-field"},
	},
	{
		Path:        "/solutions/turn-information-into-understanding",
		Eyebrow:     "From collected information to usable models",
		Title:       "Turn information into durable understanding",
		Description: "Learnloom synthesizes trusted sources into cited lessons with mechanisms, examples, skepticism, retrieval, and continuity.",
		Lead:        "Information becomes useful when you can explain the mechanism, test it against an example, recognize its limits, and retrieve it when needed.",
		Problem:     "Summarization solves length, not understanding",
		Body: []string{
			"A short recap can tell you what a source said while removing the evidence, productive difficulty, and connections that make an idea memorable. It often produces familiarity—the feeling that something makes sense—without the ability to explain or apply it.",
			"Learnloom treats synthesis as instructional design. It selects a central learning objective, preserves citations, explains causal structure, surfaces uncertainty, and asks retrieval questions. The result is a compact lesson rather than compressed content.",
		},
		Steps: []seoStep{
			{Title: "Synthesize across sources", Copy: "Combine complementary evidence around one worthwhile learning objective."},
			{Title: "Teach, question, and verify", Copy: "Pair explanation with examples, skeptical review, misconceptions, and visible sources."},
			{Title: "Create continuity", Copy: "Use prior learning to make the next lesson feel like a next step, not another isolated answer."},
		},
		Benefits: []seoBenefit{
			{Title: "Mental models over bullet points", Copy: "The lesson prioritizes causal explanations that can guide reasoning."},
			{Title: "Nuance survives compression", Copy: "Limitations, competing explanations, and citations remain part of the reading experience."},
			{Title: "Learning has a next step", Copy: "Retrieval and continuity turn one useful reading session into a practice."},
		},
		For:     []string{"Readers dissatisfied with shallow AI summaries", "Teams learning from research and industry sources", "People who want to reason with information, not merely collect it"},
		Related: []string{"/compare/ai-summaries", "/solutions/remember-what-you-read", "/product/trusted-source-learning"},
	},
	{
		Path:        "/product/ai-learning-assistant",
		Eyebrow:     "An AI assistant designed around learning",
		Title:       "A personal AI learning assistant grounded in your sources",
		Description: "Give Learnloom a topic. It establishes the source environment and prepares lessons that build on your learning history—not disposable chat answers.",
		Lead:        "Learnloom is an AI learning assistant for maintaining a learning practice: it prepares focused lessons, preserves evidence, supports recall, and keeps the results in a lasting personal archive.",
		Problem:     "An assistant with continuity and an information boundary",
		Body: []string{
			"General AI assistants are useful for questions in the moment, but the learner must repeatedly provide context, judge sources, save useful answers, and decide what should come next. A conversation ends without becoming a curriculum.",
			"Learnloom begins with your intent and source policy. It uses those boundaries to prepare Knowledge Dossiers and records what each completed lesson covered. This makes future lessons more capable of continuing the thread instead of generating another generic introduction.",
		},
		Steps: []seoStep{
			{Title: "Tell it what you are learning", Copy: "Set a topic, current level, desired outcome, lesson length, and rhythm."},
			{Title: "Ground it in evidence", Copy: "Let Learnloom discover and validate useful sources, provide your own, or ask it to fill the gaps."},
			{Title: "Receive a prepared lesson", Copy: "Read a structured Dossier with citations, practice, and continuity in your learning home."},
		},
		Benefits: []seoBenefit{
			{Title: "Persistent context", Copy: "Learning History helps later Dossiers build from concepts already introduced."},
			{Title: "Source transparency", Copy: "Original source items remain visible so important claims can be checked."},
			{Title: "A product beyond chat", Copy: "Scheduling, delivery, archives, and public or private publishing maintain the full loop."},
		},
		For:     []string{"Independent learners seeking a guided practice", "Professionals following research and industry change", "People who want AI assistance without losing source control"},
		Related: []string{"/product/trusted-source-learning", "/solutions/build-a-personal-learning-curriculum", "/compare/ai-summaries"},
	},
	{
		Path:        "/product/trusted-source-learning",
		Eyebrow:     "Keep the evidence attached",
		Title:       "Learn from sources you already trust",
		Description: "Build a recurring learning stream from trusted publications, feeds, research organizations, and expert websites while keeping citations visible.",
		Lead:        "Personalization should include the information environment, not only the writing style. Learnloom lets the learner define which sources deserve attention.",
		Problem:     "A trustworthy lesson begins before generation",
		Body: []string{
			"An answer can sound clear while relying on weak, repetitive, or irrelevant material. Source quality and coverage therefore have to be part of the learning workflow rather than an invisible detail of generation.",
			"Learnloom accepts source URLs and feeds attached to a particular learning stream. It safely acquires current material, keeps canonical source links, and builds the Dossier from the selected evidence. The source index remains available for verification and deeper reading.",
		},
		Steps: []seoStep{
			{Title: "Supply the source", Copy: "Add a publication, article, RSS or Atom feed, research group, or expert website."},
			{Title: "Curate a useful set", Copy: "Select material that can support one coherent learning opportunity."},
			{Title: "Preserve provenance", Copy: "Carry titles, publishers, links, and citations into the finished Dossier."},
		},
		Benefits: []seoBenefit{
			{Title: "Control the information boundary", Copy: "Use a bounded source set when trust and relevance matter more than broad discovery."},
			{Title: "Verify important claims", Copy: "Return to the original material rather than treating generated prose as an authority."},
			{Title: "Build from diverse evidence", Copy: "A Dossier can connect complementary sources instead of summarizing only one page."},
		},
		For:     []string{"Learners with publications or experts they already follow", "Research-heavy professionals who need provenance", "People wary of opaque AI-generated answers"},
		Related: []string{"/product/ai-learning-assistant", "/solutions/keep-up-with-your-field", "/solutions/turn-information-into-understanding"},
	},
	{
		Path:        "/compare/ai-summaries",
		Eyebrow:     "Learning Dossiers versus one-off summaries",
		Title:       "Learnloom vs. AI summaries",
		Description: "Compare ordinary AI summaries with Learnloom's source-grounded Knowledge Dossiers, retrieval practice, learning history, and personal archive.",
		Lead:        "AI summaries are useful when the job is to make one document shorter. Learnloom is designed for a different job: turning changing source material into understanding that accumulates.",
		Problem:     "Choose based on the outcome you need",
		Body: []string{
			"Use a normal summary when you need a quick orientation to one document and do not need to retain much of it. It is fast, disposable, and appropriate for many everyday tasks.",
			"Use Learnloom when you are following a subject over time. A Knowledge Dossier can synthesize several sources, explain mechanisms, show examples and limitations, prompt retrieval, and connect the lesson to a continuing history. The tradeoff is intentional: it asks for more attention than a short recap because durable learning requires more than recognition.",
		},
		Steps: []seoStep{
			{Title: "Ordinary summary", Copy: "Optimizes for compression of a particular input and an immediate answer."},
			{Title: "Knowledge Dossier", Copy: "Optimizes for explanation, evidence, skepticism, practice, and later retrieval."},
			{Title: "Learning stream", Copy: "Adds continuity so a sequence of lessons can increase depth rather than repeat introductions."},
		},
		Benefits: []seoBenefit{
			{Title: "Use summaries for speed", Copy: "A concise recap remains the right tool when retention and progression are not important."},
			{Title: "Use Learnloom for depth", Copy: "Choose a Dossier when you need to understand and revisit the underlying idea."},
			{Title: "Keep sources in view", Copy: "Learnloom keeps the evidence index attached and labels generated exploration separately."},
		},
		For:     []string{"People comparing AI learning and summarization tools", "Readers who need more than shortened articles", "Learners building knowledge in a subject over time"},
		Related: []string{"/solutions/turn-information-into-understanding", "/product/ai-learning-assistant", "/solutions/remember-what-you-read"},
	},
}

func seoPageForPath(path string) (seoPage, bool) {
	for _, page := range seoPages {
		if page.Path == path {
			return page, true
		}
	}
	return seoPage{}, false
}

func (s *Server) renderApexRobots(
	response http.ResponseWriter,
	request *http.Request,
) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300")
	if request.Method != http.MethodHead {
		fmt.Fprintf(
			response,
			"User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n",
			strings.TrimRight(s.cfg.ApexOrigin, "/"),
		)
	}
}

func (s *Server) renderApexSitemap(
	response http.ResponseWriter,
	request *http.Request,
) {
	origin := strings.TrimRight(s.cfg.ApexOrigin, "/")
	locations := []string{origin + "/"}
	for _, page := range seoPages {
		locations = append(locations, origin+page.Path)
	}
	for _, page := range authorityPages {
		locations = append(locations, origin+page.Path)
	}
	if len(s.cfg.FeaturedSites) > 0 {
		examples, err := s.loadFeaturedExamples(request.Context())
		if err != nil {
			if s.logger != nil {
				s.logger.WarnContext(request.Context(), "load featured examples for sitemap", "error", err)
			}
		} else if len(examples) > 0 {
			locations = append(locations, origin+"/examples")
		}
	}
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	body.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, location := range locations {
		body.WriteString("<url><loc>" + html.EscapeString(location) + "</loc></url>")
	}
	body.WriteString("</urlset>")
	response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(body.String()))
	}
}

func (s *Server) renderSEOPage(
	response http.ResponseWriter,
	request *http.Request,
	page seoPage,
) {
	origin := strings.TrimRight(s.cfg.ApexOrigin, "/")
	canonical := origin + page.Path
	document := renderSEODocument(page, canonical, s.cfg.AppOrigin)
	s.applyAppCSP(response)
	response.Header().Set("X-Robots-Tag", "index, follow")
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=3600")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func renderSEODocument(page seoPage, canonical, appOrigin string) string {
	var body strings.Builder
	body.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	body.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	body.WriteString(`<link rel="icon" href="/favicon.svg" type="image/svg+xml">`)
	body.WriteString(renderSEOHead(
		page.Title+" | Learnloom",
		page.Description,
		canonical,
		strings.TrimSuffix(canonical, page.Path),
	))
	body.WriteString(`<style>` + seoCSS + `</style></head><body>`)
	body.WriteString(renderMarketingNav(appOrigin, page.Path))
	body.WriteString(`<main><section class="seo-hero"><p class="eyebrow">` + html.EscapeString(page.Eyebrow) + `</p>`)
	body.WriteString(`<h1>` + html.EscapeString(page.Title) + `</h1><p class="lead">` + html.EscapeString(page.Lead) + `</p>`)
	body.WriteString(`<div class="hero-actions"><a class="primary" href="` + html.EscapeString(strings.TrimRight(appOrigin, "/")+"/sign-up") + `">Build your learning path <span>↗</span></a><a href="/#how-it-works">See how Learnloom works</a></div></section>`)
	body.WriteString(`<section class="seo-section prose"><p class="eyebrow">The problem</p><h2>` + html.EscapeString(page.Problem) + `</h2>`)
	for _, paragraph := range page.Body {
		body.WriteString("<p>" + html.EscapeString(paragraph) + "</p>")
	}
	body.WriteString(`</section><section class="seo-section"><p class="eyebrow">How it works</p><h2>A learning loop, not another content queue</h2><div class="step-grid">`)
	for index, step := range page.Steps {
		fmt.Fprintf(
			&body,
			`<article><span>%02d</span><h3>%s</h3><p>%s</p></article>`,
			index+1,
			html.EscapeString(step.Title),
			html.EscapeString(step.Copy),
		)
	}
	body.WriteString(`</div></section><section class="seo-section benefit-section"><p class="eyebrow">What changes</p><h2>Designed for understanding that lasts</h2><div class="benefit-grid">`)
	for _, benefit := range page.Benefits {
		body.WriteString(`<article><h3>` + html.EscapeString(benefit.Title) + `</h3><p>` + html.EscapeString(benefit.Copy) + `</p></article>`)
	}
	body.WriteString(`</div></section><section class="seo-section for-section"><div><p class="eyebrow">Who it is for</p><h2>Built for sustained curiosity</h2></div><ul>`)
	for _, item := range page.For {
		body.WriteString("<li><span>✓</span>" + html.EscapeString(item) + "</li>")
	}
	body.WriteString(`</ul></section><section class="seo-section related"><p class="eyebrow">Continue exploring</p><h2>Related ways to use Learnloom</h2><div>`)
	for _, path := range page.Related {
		if related, ok := seoPageForPath(path); ok {
			body.WriteString(`<a href="` + html.EscapeString(related.Path) + `"><span>` + html.EscapeString(related.Title) + `</span><b>↗</b></a>`)
		}
	}
	body.WriteString(`</div></section><section class="seo-cta"><p class="eyebrow">Your learning path is waiting</p><h2>Give curiosity a direction.</h2><p>Start with a subject. Learnloom can establish the source environment and prepare the first Knowledge Dossier—or follow only the sources you choose.</p><a class="primary" href="`)
	body.WriteString(html.EscapeString(strings.TrimRight(appOrigin, "/") + "/sign-up"))
	body.WriteString(`">Build your learning path <span>↗</span></a></section></main>`)
	body.WriteString(renderMarketingFooter(appOrigin))
	body.WriteString(`</body></html>`)
	return body.String()
}

func renderSEOHead(title, description, canonical, apexOrigin string) string {
	schema := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{
				"@type": "Organization",
				"@id":   "https://learnloom.blog/#organization",
				"name":  "Learnloom",
				"url":   "https://learnloom.blog/",
			},
			map[string]any{
				"@type":               "SoftwareApplication",
				"@id":                 "https://learnloom.blog/#application",
				"name":                "Learnloom",
				"applicationCategory": "EducationalApplication",
				"operatingSystem":     "Web",
				"url":                 canonical,
				"description":         description,
				"publisher": map[string]any{
					"@id": "https://learnloom.blog/#organization",
				},
			},
		},
	}
	encoded, _ := json.Marshal(schema)
	return `<title>` + html.EscapeString(title) + `</title>` +
		`<meta name="description" content="` + html.EscapeString(description) + `">` +
		`<link rel="canonical" href="` + html.EscapeString(canonical) + `">` +
		`<meta property="og:type" content="website">` +
		`<meta property="og:site_name" content="Learnloom">` +
		`<meta property="og:title" content="` + html.EscapeString(title) + `">` +
		`<meta property="og:description" content="` + html.EscapeString(description) + `">` +
		`<meta property="og:url" content="` + html.EscapeString(canonical) + `">` +
		renderSocialImageMetadata(apexOrigin) +
		`<meta name="twitter:title" content="` + html.EscapeString(title) + `">` +
		`<meta name="twitter:description" content="` + html.EscapeString(description) + `">` +
		`<script type="application/ld+json">` + string(encoded) + `</script>`
}

func decorateMarketingIndex(body []byte, apexOrigin string) []byte {
	const title = "Learnloom | Give us a topic. We’ll build your learning home."
	const description = "Give Learnloom a topic. It builds a learning path and gives every lesson a lasting home at your own Learnloom address."
	canonical := strings.TrimRight(apexOrigin, "/") + "/"
	document := string(body)
	document = strings.Replace(
		document,
		"<title>Learnloom · Knowledge Dossiers</title>",
		"<title>"+html.EscapeString(title)+"</title>",
		1,
	)
	document = strings.Replace(
		document,
		`content="Give Learnloom a topic. It finds and evaluates useful sources, builds a progressive learning path, and helps you remember what matters."`,
		`content="`+html.EscapeString(description)+`"`,
		1,
	)
	head := renderSEOHead(title, description, canonical, apexOrigin)
	head = strings.Replace(head, "<title>"+html.EscapeString(title)+"</title>", "", 1)
	head = strings.Replace(head, `<meta name="description" content="`+html.EscapeString(description)+`">`, "", 1)
	document = strings.Replace(document, "</head>", head+"</head>", 1)
	return []byte(document)
}

const seoCSS = `
@font-face{font-family:Manrope;font-style:normal;font-weight:400;font-display:swap;src:url("/assets/fonts/manrope-400.woff2") format("woff2")}@font-face{font-family:Manrope;font-style:normal;font-weight:500;font-display:swap;src:url("/assets/fonts/manrope-500.woff2") format("woff2")}@font-face{font-family:Manrope;font-style:normal;font-weight:600;font-display:swap;src:url("/assets/fonts/manrope-600.woff2") format("woff2")}@font-face{font-family:Manrope;font-style:normal;font-weight:700;font-display:swap;src:url("/assets/fonts/manrope-700.woff2") format("woff2")}
:root{--ink:#17211b;--muted:#5f6b63;--paper:#f7f6f0;--green:#496b4c;--line:rgba(23,33,27,.12);color:var(--ink);background:var(--paper);font-family:Manrope,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-synthesis:none}
*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;background:var(--paper)}a{color:inherit}.seo-nav{position:absolute;z-index:10;inset:0 0 auto;height:84px;width:min(1440px,calc(100% - 72px));margin:auto;display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:34px;background:transparent}
.seo-brand{width:max-content;display:inline-flex;align-items:center;gap:10px;color:inherit;font-size:21px;font-weight:750;letter-spacing:-.045em;text-decoration:none}.seo-brand span{width:31px;height:31px;display:grid;place-items:center;border-radius:9px;color:#eef6ec;background:#1d2c22;box-shadow:inset 0 0 0 1px rgba(255,255,255,.14),0 8px 20px rgba(19,36,24,.13)}.seo-brand strong{font:inherit}
.seo-nav nav{display:flex;align-items:center;justify-content:center;gap:clamp(18px,2.2vw,34px)}.seo-nav nav a,.nav-sign-in,.nav-cta{text-decoration:none;font-size:13px;font-weight:600}.seo-nav nav a,.nav-sign-in{color:rgba(23,33,27,.72)}.seo-nav nav a:hover,.nav-sign-in:hover,.seo-nav nav a[aria-current=page]{color:var(--ink)}.seo-nav nav a[aria-current=page]{text-decoration:underline;text-decoration-color:#87ac57;text-decoration-thickness:2px;text-underline-offset:7px}.seo-nav-actions{display:flex;align-items:center;justify-content:flex-end;gap:22px}.nav-cta,.primary{display:inline-flex;align-items:center;justify-content:center;gap:10px;border-radius:999px;color:#f8fbf7!important;background:#17221a;box-shadow:0 12px 32px rgba(18,35,23,.16);text-decoration:none;font-weight:700;transition:transform .18s ease,background .18s ease,box-shadow .18s ease}.nav-cta{min-height:43px;padding:0 18px;font-size:12px}.nav-cta:hover,.primary:hover{transform:translateY(-2px);background:#243b2a;box-shadow:0 16px 38px rgba(18,35,23,.22)}
main{overflow:hidden}.seo-hero{position:relative;min-height:820px;display:flex;flex-direction:column;justify-content:center;padding:clamp(150px,13vw,190px) max(24px,calc((100vw - 1120px)/2)) 210px;background-image:linear-gradient(90deg,rgba(239,248,250,.88) 0%,rgba(239,248,250,.68) 42%,rgba(239,248,250,.06) 72%),linear-gradient(180deg,rgba(235,246,252,.14) 0%,rgba(17,36,24,.15) 100%),url("/assets/marketing/learnloom-landscape.avif");background-position:center bottom;background-size:cover;border-bottom:1px solid var(--line)}
.seo-hero:after{position:absolute;inset:auto 0 0;height:150px;background:linear-gradient(180deg,transparent,rgba(16,32,22,.16));content:""}.seo-hero>*{position:relative;z-index:1}
.eyebrow{margin:0 0 20px;color:#47644c;font-size:10px;font-weight:800;letter-spacing:.15em;text-transform:uppercase}.seo-hero h1{max-width:920px;margin:0;font-size:clamp(52px,6.5vw,92px);font-weight:520;line-height:.98;letter-spacing:-.06em}.lead{max-width:720px;margin:30px 0;color:#46564b;font-size:clamp(17px,1.7vw,20px);line-height:1.68}.hero-actions{display:flex;align-items:center;gap:25px;flex-wrap:wrap}.hero-actions a{font-size:13px;font-weight:700;text-decoration:none}.primary{min-height:50px;padding:0 22px}
.seo-section{max-width:1120px;margin:0 auto;padding:clamp(76px,9vw,124px) 24px;border-bottom:1px solid var(--line)}.seo-section h2,.seo-cta h2{max-width:800px;margin:0 0 34px;font-size:clamp(40px,5vw,64px);font-weight:520;line-height:1.04;letter-spacing:-.05em}.prose p:not(.eyebrow){max-width:780px;color:#385149;font-family:Georgia,"Times New Roman",serif;font-size:21px;line-height:1.78}.step-grid,.benefit-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:18px}.step-grid article,.benefit-grid article{padding:30px;border:1px solid var(--line);border-radius:20px;background:rgba(255,255,255,.72);box-shadow:0 18px 50px rgba(27,48,34,.05)}
.step-grid article>span{display:block;margin-bottom:58px;color:#527355;font-size:11px;font-weight:800}.seo-section h3{margin:0 0 12px;font-size:20px;letter-spacing:-.025em}.seo-section article p{margin:0;color:#526a64;line-height:1.68}.benefit-section{max-width:none;padding-left:max(24px,calc((100vw - 1120px)/2));padding-right:max(24px,calc((100vw - 1120px)/2));color:#f7fbf6;background:#17251c}.benefit-section .eyebrow{color:#abc477}.benefit-section article{color:var(--ink);background:#f7f6f0}
.for-section{display:grid;grid-template-columns:1fr 1fr;gap:64px;align-items:start}.for-section ul{list-style:none;margin:0;padding:0}.for-section li{display:flex;gap:14px;padding:18px 0;border-bottom:1px solid var(--line);font-size:17px;line-height:1.5}.for-section li span{color:#527355;font-weight:900}.related>div{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.related a{min-height:160px;display:flex;justify-content:space-between;gap:20px;padding:26px;border:1px solid var(--line);border-radius:18px;background:rgba(255,255,255,.74);text-decoration:none;font-weight:750;line-height:1.4;transition:transform .18s ease,box-shadow .18s ease}.related a:hover{transform:translateY(-3px);box-shadow:0 18px 44px rgba(27,48,34,.08)}.related b{color:#527355}
.seo-cta{text-align:center;padding:clamp(96px,12vw,160px) 24px;background:radial-gradient(circle at 50% 0,rgba(255,255,255,.75),transparent 35%),linear-gradient(160deg,#dcecf4,#e5eddc 58%,#efe2d3)}.seo-cta h2,.seo-cta p{margin-left:auto;margin-right:auto}.seo-cta>p:not(.eyebrow){max-width:650px;margin-bottom:30px;color:#48625b;font-size:18px;line-height:1.65}.seo-cta .primary{width:max-content;margin:auto}
.seo-footer{display:grid;grid-template-columns:minmax(230px,1fr) minmax(520px,1.5fr);gap:70px;padding:70px max(24px,calc((100vw - 1240px)/2)) 30px;color:#ecf4eb;background:#142119}.seo-footer-intro p{max-width:310px;margin:22px 0 0;color:#9eafa2;line-height:1.6}.seo-footer-links{display:grid;grid-template-columns:repeat(4,1fr);gap:34px}.seo-footer-links div{display:grid;align-content:start;gap:11px}.seo-footer-links strong{margin-bottom:6px;color:#f6faf5;font-size:11px;letter-spacing:.1em;text-transform:uppercase}.seo-footer-links a{color:#aebdb1;font-size:13px;line-height:1.45;text-decoration:none}.seo-footer-links a:hover{color:#fff}.seo-footer-bottom{grid-column:1/-1;display:flex;justify-content:space-between;margin-top:30px;padding-top:25px;border-top:1px solid rgba(255,255,255,.1);color:#778a7b;font-size:11px}
@media(max-width:1050px){.seo-nav{width:calc(100% - 40px)}.seo-nav nav a:not(:first-child){display:none}.seo-nav nav{justify-content:flex-end}.seo-nav-actions{gap:14px}.seo-footer{grid-template-columns:1fr}.seo-footer-links{grid-template-columns:repeat(4,1fr)}}
@media(max-width:780px){.seo-nav{height:72px;grid-template-columns:1fr auto auto;gap:14px;width:calc(100% - 32px)}.seo-nav nav{display:flex}.seo-nav nav a:first-child{display:block;font-size:12px}.nav-sign-in{display:none}.seo-brand{font-size:19px}.seo-hero{min-height:720px;padding:130px 20px 170px;background-image:linear-gradient(180deg,rgba(239,248,250,.72) 0%,rgba(239,248,250,.45) 48%,rgba(17,36,24,.18) 100%),url("/assets/marketing/learnloom-landscape-mobile.avif");background-position:center bottom}.seo-hero h1{font-size:clamp(46px,13vw,62px)}.step-grid,.benefit-grid,.related>div,.for-section{grid-template-columns:1fr}.step-grid article>span{margin-bottom:30px}.for-section{gap:20px}.seo-footer-links{grid-template-columns:1fr 1fr}.seo-footer-bottom{align-items:flex-start;flex-direction:column;gap:8px}}
@media(max-width:460px){.seo-footer-links{grid-template-columns:1fr}.nav-cta{padding:0 14px}.seo-hero h1{font-size:44px}}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}.nav-cta,.primary,.related a{transition:none}}
`
