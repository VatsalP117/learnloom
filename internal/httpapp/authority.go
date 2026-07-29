package httpapp

import (
	"encoding/json"
	"html"
	"net/http"
	"strings"
)

type authorityPage struct {
	Path        string
	Category    string
	SchemaType  string
	Title       string
	Description string
	Lead        string
	Sections    []authoritySection
	References  []authorityReference
	Takeaways   []string
	Related     []string
}

type authoritySection struct {
	Title      string
	Paragraphs []string
	Bullets    []string
}

type authorityReference struct {
	Title    string
	Citation string
	URL      string
}

var authorityPages = []authorityPage{
	{
		Path:        "/guides",
		Category:    "Learnloom guides",
		SchemaType:  "CollectionPage",
		Title:       "Practical guides for learning in an age of information abundance",
		Description: "Evidence-aware guides for remembering what you read, keeping up with changing fields, and building a personal learning system.",
		Lead:        "The difficult part of modern learning is not access. It is deciding what deserves attention, turning it into a useful model, and returning to it before it disappears.",
		Sections: []authoritySection{
			{
				Title: "Learn with a repeatable loop",
				Paragraphs: []string{
					"These guides treat learning as a system rather than a motivation problem. Each one begins with a real constraint: limited attention, changing information, weak recall, or disconnected tools. The goal is a workflow that can survive an ordinary week.",
					"The methods do not require reading everything or building a perfect knowledge base. They emphasize clear questions, credible sources, deliberate synthesis, retrieval, and a small archive that remains useful later.",
				},
			},
			{
				Title: "Start with the problem you feel most often",
				Bullets: []string{
					"If useful articles become vague memories, begin with the guide to remembering what you read.",
					"If your field changes faster than you can follow it, begin with the guide to staying current without information overload.",
					"If courses, notes, feeds, and bookmarks feel disconnected, begin with the personal learning-system guide.",
				},
			},
		},
		Takeaways: []string{
			"Depth comes from what you retrieve and connect, not how much you collect.",
			"A bounded information environment is often more useful than an infinite feed.",
			"A learning archive should help determine what comes next.",
		},
		Related: []string{"/guides/how-to-remember-what-you-read", "/guides/how-to-keep-up-with-your-field", "/guides/how-to-build-a-personal-learning-system"},
	},
	{
		Path:        "/guides/how-to-remember-what-you-read",
		Category:    "Guide · Durable learning",
		SchemaType:  "Article",
		Title:       "How to remember what you read",
		Description: "A practical reading workflow using clear intent, explanation, retrieval practice, spacing, application, and a connected learning archive.",
		Lead:        "Forgetting is not evidence that you are a bad reader. Most reading workflows produce recognition in the moment but never create the retrieval and connections that make an idea available later.",
		Sections: []authoritySection{
			{
				Title: "Begin with a question, not a queue",
				Paragraphs: []string{
					"Before opening an article or chapter, write the question you expect it to help answer. A useful question creates a selection rule: some passages become central, others become context, and many can be ignored.",
					"Broad intentions such as “learn about artificial intelligence” are difficult to act on. A question such as “why do language models hallucinate, and which interventions change the failure rate?” gives the reading a job.",
				},
				Bullets: []string{
					"What should become clearer after this reading?",
					"What decision, explanation, or capability should improve?",
					"What do I currently believe about the answer?",
				},
			},
			{
				Title: "Translate the source into a model",
				Paragraphs: []string{
					"Do not begin by highlighting sentences. First try to state the central mechanism in your own words: what changes, what causes the change, and under which conditions the explanation holds.",
					"Then create one worked example. If the mechanism cannot explain a concrete case, the explanation may still be verbal familiarity rather than understanding. Add one limitation or competing explanation to keep the model from becoming more certain than the evidence.",
				},
			},
			{
				Title: "Retrieve before you review",
				Paragraphs: []string{
					"Close the source and answer a small number of questions without looking. Retrieval is useful precisely because it feels less fluent than rereading. The gap between what seemed clear and what can be reconstructed shows where the model is weak.",
					"Questions should require explanation or application, not recognition. “What were the three sections?” is usually less useful than “What mechanism connects the evidence to the conclusion?”",
				},
				Bullets: []string{
					"Explain the central idea without using the source's phrasing.",
					"Apply it to a new example.",
					"Name one condition under which it could fail.",
				},
			},
			{
				Title: "Return through spacing and connection",
				Paragraphs: []string{
					"Revisit the idea after some forgetting has occurred. The exact schedule matters less than having several effortful returns. A short retrieval after a day, a week, and when a related idea appears is more valuable than repeatedly rereading the original note in one sitting.",
					"Store the learning around concepts and questions, not only source titles. When new material arrives, link it to the earlier mechanism: does it extend the model, contradict it, supply a better example, or reveal a missing prerequisite?",
				},
			},
			{
				Title: "Use the smallest workflow you can maintain",
				Paragraphs: []string{
					"A sustainable reading record can be compact: the question, a mechanism, an example, a limitation, three retrieval prompts, and a source link. The purpose is not to reproduce the article. It is to preserve enough structure to think with the idea again.",
					"Learnloom's Knowledge Dossiers follow this pattern by combining explanation, evidence, skepticism, practice, and continuity in one lesson. The same structure also works as a manual reading practice.",
				},
			},
		},
		References: []authorityReference{
			{
				Title:    "Test-Enhanced Learning: Taking Memory Tests Improves Long-Term Retention",
				Citation: "Roediger and Karpicke, Psychological Science, 2006.",
				URL:      "https://www.psychologicalscience.org/journals/psychological-science/j.1467-9280.2006.01693.x/",
			},
			{
				Title:    "Distributed practice in verbal recall tasks: A review and quantitative synthesis",
				Citation: "Cepeda, Pashler, Vul, Wixted, and Rohrer, Psychological Bulletin, 2006.",
				URL:      "https://pubmed.ncbi.nlm.nih.gov/16719566/",
			},
			{
				Title:    "Improving Students' Learning With Effective Learning Techniques",
				Citation: "Dunlosky, Rawson, Marsh, Nathan, and Willingham, Psychological Science in the Public Interest, 2013.",
				URL:      "https://doi.org/10.1177/1529100612453266",
			},
		},
		Takeaways: []string{
			"Set a question before reading.",
			"Explain a mechanism and test it with an example.",
			"Retrieve before rereading.",
			"Reconnect the idea when related material appears.",
		},
		Related: []string{"/solutions/remember-what-you-read", "/guides/how-to-build-a-personal-learning-system", "/product/ai-learning-assistant"},
	},
	{
		Path:        "/guides/how-to-keep-up-with-your-field",
		Category:    "Guide · Information overload",
		SchemaType:  "Article",
		Title:       "How to keep up with your field without reading everything",
		Description: "A sustainable system for following a changing field using bounded questions, a credible source portfolio, synthesis, and deliberate review.",
		Lead:        "Keeping up does not mean seeing every update. It means noticing the developments that change your model of the field while protecting enough attention to understand them.",
		Sections: []authoritySection{
			{
				Title: "Define the field as a set of questions",
				Paragraphs: []string{
					"A field label is too broad to guide attention. Convert it into three to five questions that matter to your work or curiosity. For climate technology, those questions might concern cost curves, deployment constraints, policy, financing, and evidence from operating projects.",
					"These questions become filters. An update earns attention when it changes an answer, supplies strong evidence, or exposes a missing prerequisite. Popularity alone is not a reason to add it to the queue.",
				},
			},
			{
				Title: "Build a source portfolio, not one perfect feed",
				Paragraphs: []string{
					"Different sources play different roles. Primary research and official data establish evidence. Specialist publications provide reporting and context. Skilled explainers make mechanisms legible. Skeptical or contrasting voices reveal uncertainty.",
					"Choose a small portfolio across those roles and review it periodically. A source that produces high volume but rarely changes your understanding is expensive even when every individual item is free.",
				},
				Bullets: []string{
					"Primary evidence: papers, datasets, standards, filings, or official releases.",
					"Domain reporting: publications close to the people and institutions doing the work.",
					"Interpretation: experts who explain mechanisms and disclose uncertainty.",
					"Counterweight: credible sources likely to challenge the dominant account.",
				},
			},
			{
				Title: "Separate monitoring from learning",
				Paragraphs: []string{
					"Monitoring should be fast and selective. Learning should be slower and focused. Combining them creates a feed in which every headline demands the same cognitive mode.",
					"Use a short monitoring window to identify candidates. Then choose at most one coherent learning opportunity: a mechanism to understand, a disagreement to compare, or a development that changes prior assumptions. Several related items can become one synthesis rather than several disconnected summaries.",
				},
			},
			{
				Title: "Maintain a change log for your mental model",
				Paragraphs: []string{
					"For each meaningful update, record what you believed before, what the new evidence suggests, how confident you are, and what would change the conclusion again. This makes progress visible without requiring an exhaustive archive.",
					"Connect the update to earlier entries. Repetition may indicate a stable foundation, while contradiction may reveal a genuine shift, different conditions, or weak sourcing. The connection is often more valuable than the update itself.",
				},
			},
			{
				Title: "Allow quiet days",
				Paragraphs: []string{
					"A healthy learning system does not manufacture novelty. Some days the evidence is repetitive, speculative, or irrelevant to the questions you chose. Doing nothing protects attention and makes the next selected lesson more trustworthy.",
					"Learnloom uses learning streams and trusted sources to support this bounded approach. Its goal is a worthwhile Dossier, not an obligation to amplify every new item.",
				},
			},
		},
		Takeaways: []string{
			"Follow questions rather than a field label.",
			"Assign clear roles to a small source portfolio.",
			"Monitor broadly but learn selectively.",
			"Record changes to your model, not every headline.",
		},
		Related: []string{"/solutions/keep-up-with-your-field", "/product/trusted-source-learning", "/guides/how-to-remember-what-you-read"},
	},
	{
		Path:        "/guides/how-to-build-a-personal-learning-system",
		Category:    "Guide · Personal curriculum",
		SchemaType:  "Article",
		Title:       "How to build a personal learning system that compounds",
		Description: "Design a maintainable personal learning system around intent, trusted inputs, focused synthesis, retrieval, and a useful archive.",
		Lead:        "A personal learning system should reduce decisions and deepen understanding. If maintaining the system consumes more energy than learning, it has become another collection hobby.",
		Sections: []authoritySection{
			{
				Title: "Choose a small number of learning streams",
				Paragraphs: []string{
					"A stream is a sustained question or capability, not an inbox folder. Two to five active streams are usually enough to create variety without fragmenting attention. Give each stream an outcome that can guide depth and selection.",
					"For example, “urban planning” is a category. “Understand how transport, land use, and municipal finance shape housing supply” is a direction. It makes sources and lessons easier to evaluate.",
				},
			},
			{
				Title: "Give every tool one responsibility",
				Paragraphs: []string{
					"Many systems fail because bookmarks, notes, feeds, task managers, and AI chats each contain overlapping fragments. Decide where discovery happens, where temporary candidates wait, where synthesis lives, and where retrieval is scheduled.",
					"Minimize copying between tools. A source link can remain a source link. Only promote material into the durable archive after it has contributed to an explanation, decision, example, or question worth revisiting.",
				},
				Bullets: []string{
					"Intent: what you are trying to understand or become able to do.",
					"Inputs: the sources and discovery channels allowed into the stream.",
					"Synthesis: a focused lesson or note that creates a usable model.",
					"Retrieval: prompts that test whether the model remains available.",
					"History: a compact record that informs the next learning step.",
				},
			},
			{
				Title: "Use a standard lesson shape",
				Paragraphs: []string{
					"A repeatable structure lowers the cost of turning source material into learning. Start with one objective, explain the central mechanism, add prerequisites only when necessary, work through an example, examine limitations, and finish with retrieval or application.",
					"Keep the sources attached. Synthesis should make evidence easier to reason about, not erase where it came from. Separate deductions or speculative extensions from claims supported by the source set.",
				},
			},
			{
				Title: "Let history influence what comes next",
				Paragraphs: []string{
					"An archive compounds only when it participates in future learning. Before preparing a new lesson, review what the stream has already covered. Avoid repeating the same introduction unless retrieval shows that reinforcement is needed.",
					"Use history to increase depth, fill prerequisites, compare changed evidence, and revisit important mechanisms with new examples. This is the difference between storing lessons and maintaining a curriculum.",
				},
			},
			{
				Title: "Review the system, not just the content",
				Paragraphs: []string{
					"Once a month, ask which streams produced useful changes in understanding, which sources created signal, and which routines caused friction. Pause streams that no longer serve a real question. Remove noisy inputs. Shorten the lesson format if it is not sustainable.",
					"Learnloom automates parts of this loop through recurring learning streams, Knowledge Dossiers, Learning History, retrieval practice, and a personal archive. The design principle remains useful even when the workflow is entirely manual: every component should help decide, understand, remember, or return.",
				},
			},
		},
		Takeaways: []string{
			"Organize around sustained questions and outcomes.",
			"Give discovery, synthesis, retrieval, and history distinct roles.",
			"Promote only understood material into the durable archive.",
			"Use prior learning to select and shape the next lesson.",
		},
		Related: []string{"/solutions/build-a-personal-learning-curriculum", "/guides/how-to-remember-what-you-read", "/how-learnloom-works"},
	},
	{
		Path:        "/how-learnloom-works",
		Category:    "Product transparency",
		SchemaType:  "Article",
		Title:       "How Learnloom turns current sources into a learning practice",
		Description: "A transparent explanation of Learnloom's learning streams, source handling, Knowledge Dossiers, quality checks, history, retrieval, and publishing controls.",
		Lead:        "Learnloom is designed around a simple responsibility: maintain the loop between what is changing in the world and what one learner is trying to understand over time.",
		Sections: []authoritySection{
			{
				Title: "1. The learner defines the intent",
				Paragraphs: []string{
					"A learning stream begins with a topic or question, current level, desired outcome, lesson length, schedule, and source policy. These are instructional constraints, not profile decoration. They determine what material is relevant and how deeply it should be explained.",
					"Learnloom currently supports learner-provided trusted sources. Broader source discovery is a planned capability and should not be assumed to be active unless it is visibly offered during stream creation.",
				},
			},
			{
				Title: "2. Sources remain an explicit boundary",
				Paragraphs: []string{
					"A stream can include articles, publications, websites, and RSS or Atom feeds. Learnloom safely fetches public source material, follows bounded redirects, rejects private network targets, and records canonical source information.",
					"Source provenance remains visible in the finished Dossier. Generated prose is not presented as a replacement for the underlying evidence, and important claims should be checked against the linked material.",
				},
			},
			{
				Title: "3. A Dossier is prepared as a lesson",
				Paragraphs: []string{
					"The system selects a coherent learning opportunity rather than producing an unrelated summary for every item. A Knowledge Dossier can include an objective, central mechanism, prerequisites, synthesis across sources, a worked example, skeptical review, misconception, retrieval questions, and practical application.",
					"Optional AI Exploration is labeled separately because analogies, deductions, and scenarios can extend beyond sourced claims. The distinction is intended to preserve epistemic clarity without preventing useful speculation.",
				},
			},
			{
				Title: "4. Quality and failure remain visible",
				Paragraphs: []string{
					"Preparation passes through explicit curation, research, teaching, skepticism, practice, and editorial validation stages. The service records generation attempts and classifies failures so work can be retried safely without pretending that every run succeeds.",
					"Model output can still be wrong. Quality gates, citations, and structured review reduce risk but do not make generated content authoritative. Learnloom is a learning aid, not a substitute for professional judgment or primary evidence.",
				},
			},
			{
				Title: "5. Completed learning becomes history",
				Paragraphs: []string{
					"A completed Dossier contributes a bounded record of concepts, objective, sources, summary, and recall prompts. Later lessons can use that history to avoid repetitive introductions, restore prerequisites, increase depth, or revisit an important idea.",
					"The product's unit of value is therefore not one generated answer. It is the sequence: a changing information environment becoming a more coherent personal curriculum.",
				},
			},
			{
				Title: "6. The learner controls delivery and visibility",
				Paragraphs: []string{
					"Dossiers live in the authenticated learning archive and can be delivered by email as a prompt. Learners may also claim a personal subdomain and choose whether the site, individual streams, and individual Dossiers are public.",
					"Private and hidden material is excluded from public reading routes. Empty public archives are kept out of search indexing until they contain a published Dossier.",
				},
			},
		},
		Takeaways: []string{
			"The learner controls the intent and source environment.",
			"Dossiers are structured lessons, not one-click summaries.",
			"Citations, limitations, and generated exploration remain distinguishable.",
			"Learning History makes later lessons part of a sequence.",
		},
		Related: []string{"/product/ai-learning-assistant", "/product/trusted-source-learning", "/editorial-principles"},
	},
	{
		Path:        "/editorial-principles",
		Category:    "Trust and editorial standards",
		SchemaType:  "Article",
		Title:       "Learnloom's editorial principles",
		Description: "The principles Learnloom uses for source provenance, useful synthesis, uncertainty, AI exploration, attention protection, and learner control.",
		Lead:        "A learning system earns trust by making its boundaries visible. These principles guide how Learnloom selects, explains, qualifies, and publishes generated lessons.",
		Sections: []authoritySection{
			{
				Title: "Preserve provenance",
				Paragraphs: []string{
					"Every Dossier should make its source set inspectable. Titles, publishers, and canonical links remain attached so a learner can verify important claims, understand the information boundary, and continue into the original material.",
				},
			},
			{
				Title: "Optimize for understanding, not compression",
				Paragraphs: []string{
					"Shorter is useful only when the mechanism, evidence, and uncertainty survive. A Dossier should create a usable explanation through context, examples, skepticism, and retrieval rather than merely restating source conclusions in fewer words.",
				},
			},
			{
				Title: "Do not manufacture novelty",
				Paragraphs: []string{
					"A schedule is a learning rhythm, not a quota for content. When available material is repetitive, weakly supported, or irrelevant to the learner's intent, protecting attention is more valuable than forcing another lesson.",
				},
			},
			{
				Title: "Make uncertainty legible",
				Paragraphs: []string{
					"Explanations should identify assumptions, limitations, and credible competing accounts. Confidence should follow the evidence. Clear prose must not be mistaken for settled knowledge.",
				},
			},
			{
				Title: "Separate evidence from exploration",
				Paragraphs: []string{
					"Analogies, deductions, scenarios, and experiments can support learning, but they may extend beyond cited sources. Learnloom labels optional AI Exploration so learners can distinguish sourced synthesis from synthetic extension.",
				},
			},
			{
				Title: "Respect learner control",
				Paragraphs: []string{
					"Learners choose their intent, source policy, delivery, AI Exploration setting, and publication visibility. Private material must remain private, and public visibility should not silently expose hidden streams or Dossiers.",
				},
			},
			{
				Title: "Treat generated work as fallible",
				Paragraphs: []string{
					"Models can omit context, misunderstand sources, and produce incorrect statements. Quality checks and source links reduce but do not remove that risk. Important claims should be verified, and product language should never imply infallibility.",
				},
			},
		},
		Takeaways: []string{
			"Sources remain visible.",
			"Attention is protected from forced output.",
			"Uncertainty and synthetic exploration are labeled.",
			"Learners retain control over privacy and publication.",
		},
		Related: []string{"/how-learnloom-works", "/product/trusted-source-learning", "/compare/ai-summaries"},
	},
}

func authorityPageForPath(path string) (authorityPage, bool) {
	for _, page := range authorityPages {
		if page.Path == path {
			return page, true
		}
	}
	return authorityPage{}, false
}

func (s *Server) renderAuthorityPage(
	response http.ResponseWriter,
	request *http.Request,
	page authorityPage,
) {
	origin := strings.TrimRight(s.cfg.ApexOrigin, "/")
	canonical := origin + page.Path
	document := renderAuthorityDocument(page, canonical, s.cfg.AppOrigin)
	s.applyAppCSP(response)
	response.Header().Set("X-Robots-Tag", "index, follow")
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=3600")
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(document))
	}
}

func renderAuthorityDocument(page authorityPage, canonical, appOrigin string) string {
	var body strings.Builder
	body.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	body.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	body.WriteString(`<link rel="icon" href="/favicon.svg" type="image/svg+xml">`)
	body.WriteString(renderAuthorityHead(page, canonical))
	body.WriteString(`<style>` + seoCSS + authorityCSS + `</style></head><body>`)
	body.WriteString(`<header class="seo-nav"><a class="seo-brand" href="/"><span>✣</span>Learnloom</a>`)
	body.WriteString(`<nav aria-label="Main navigation"><a href="/solutions">Solutions</a><a href="/product/ai-learning-assistant">Product</a><a href="/guides">Guides</a><a href="/how-learnloom-works">How it works</a></nav>`)
	body.WriteString(`<a class="nav-cta" href="` + html.EscapeString(strings.TrimRight(appOrigin, "/")+"/sign-up") + `">Start learning <span>↗</span></a></header>`)
	body.WriteString(`<main><article class="authority-article"><header class="authority-hero"><p class="eyebrow">` + html.EscapeString(page.Category) + `</p><h1>` + html.EscapeString(page.Title) + `</h1><p>` + html.EscapeString(page.Lead) + `</p></header>`)
	body.WriteString(`<div class="authority-layout"><div class="authority-body">`)
	for _, section := range page.Sections {
		body.WriteString(`<section><h2>` + html.EscapeString(section.Title) + `</h2>`)
		for _, paragraph := range section.Paragraphs {
			body.WriteString(`<p>` + html.EscapeString(paragraph) + `</p>`)
		}
		if len(section.Bullets) > 0 {
			body.WriteString(`<ul>`)
			for _, bullet := range section.Bullets {
				body.WriteString(`<li>` + html.EscapeString(bullet) + `</li>`)
			}
			body.WriteString(`</ul>`)
		}
		body.WriteString(`</section>`)
	}
	if len(page.References) > 0 {
		body.WriteString(`<section class="authority-sources"><h2>Research references</h2><ol>`)
		for _, reference := range page.References {
			body.WriteString(`<li><a href="` + html.EscapeString(reference.URL) +
				`" rel="external">` + html.EscapeString(reference.Title) +
				`</a><span>` + html.EscapeString(reference.Citation) + `</span></li>`)
		}
		body.WriteString(`</ol></section>`)
	}
	body.WriteString(`</div><aside><p class="eyebrow">Key takeaways</p><ul>`)
	for _, takeaway := range page.Takeaways {
		body.WriteString(`<li><span>✓</span>` + html.EscapeString(takeaway) + `</li>`)
	}
	body.WriteString(`</ul></aside></div></article><section class="seo-section related authority-related"><p class="eyebrow">Continue exploring</p><h2>Put the ideas into practice</h2><div>`)
	for _, path := range page.Related {
		title, ok := publicPageTitle(path)
		if ok {
			body.WriteString(`<a href="` + html.EscapeString(path) + `"><span>` + html.EscapeString(title) + `</span><b>↗</b></a>`)
		}
	}
	body.WriteString(`</div></section><section class="seo-cta"><p class="eyebrow">A learning system that maintains the loop</p><h2>Turn the method into a practice.</h2><p>Choose a subject, bring the sources you trust, and let each lesson build on the last.</p><a class="primary" href="`)
	body.WriteString(html.EscapeString(strings.TrimRight(appOrigin, "/") + "/sign-up"))
	body.WriteString(`">Create your learning stream <span>↗</span></a></section></main>`)
	body.WriteString(renderSEOFooter(appOrigin))
	body.WriteString(`</body></html>`)
	return body.String()
}

func renderAuthorityHead(page authorityPage, canonical string) string {
	citations := make([]string, 0, len(page.References))
	for _, reference := range page.References {
		citations = append(citations, reference.URL)
	}
	schema := map[string]any{
		"@context":         "https://schema.org",
		"@type":            page.SchemaType,
		"headline":         page.Title,
		"description":      page.Description,
		"mainEntityOfPage": canonical,
		"datePublished":    "2026-07-29",
		"dateModified":     "2026-07-29",
		"author": map[string]any{
			"@type": "Organization",
			"name":  "Learnloom",
			"url":   "https://learnloom.blog/",
		},
		"publisher": map[string]any{
			"@type": "Organization",
			"name":  "Learnloom",
			"url":   "https://learnloom.blog/",
		},
	}
	if len(citations) > 0 {
		schema["citation"] = citations
	}
	encoded, _ := json.Marshal(schema)
	title := page.Title + " | Learnloom"
	openGraphType := "article"
	if page.SchemaType == "CollectionPage" {
		openGraphType = "website"
	}
	return `<title>` + html.EscapeString(title) + `</title>` +
		`<meta name="description" content="` + html.EscapeString(page.Description) + `">` +
		`<link rel="canonical" href="` + html.EscapeString(canonical) + `">` +
		`<meta property="og:type" content="` + openGraphType + `">` +
		`<meta property="og:site_name" content="Learnloom">` +
		`<meta property="og:title" content="` + html.EscapeString(title) + `">` +
		`<meta property="og:description" content="` + html.EscapeString(page.Description) + `">` +
		`<meta property="og:url" content="` + html.EscapeString(canonical) + `">` +
		`<meta name="twitter:card" content="summary">` +
		`<meta name="twitter:title" content="` + html.EscapeString(title) + `">` +
		`<meta name="twitter:description" content="` + html.EscapeString(page.Description) + `">` +
		`<script type="application/ld+json">` + string(encoded) + `</script>`
}

func publicPageTitle(path string) (string, bool) {
	if page, ok := seoPageForPath(path); ok {
		return page.Title, true
	}
	if page, ok := authorityPageForPath(path); ok {
		return page.Title, true
	}
	return "", false
}

func renderSEOFooter(appOrigin string) string {
	return `<footer><a class="seo-brand" href="/"><span>✣</span>Learnloom</a>` +
		`<p>Current sources, woven into durable understanding.</p><nav>` +
		`<a href="/guides">Guides</a><a href="/editorial-principles">Editorial principles</a>` +
		`<a href="/privacy">Privacy</a><a href="/terms">Terms</a><a href="` +
		html.EscapeString(strings.TrimRight(appOrigin, "/")+"/sign-in") +
		`">Sign in</a></nav></footer>`
}

const authorityCSS = `
.authority-article{max-width:1120px;margin:0 auto;padding:clamp(75px,9vw,130px) 24px 40px}
.authority-hero{max-width:900px;padding-bottom:clamp(60px,8vw,100px);border-bottom:1px solid rgba(16,37,33,.14)}
.authority-hero h1{font-family:Georgia,serif;font-size:clamp(46px,7vw,82px);font-weight:500;line-height:1.04;letter-spacing:-.045em;margin:0 0 30px}
.authority-hero>p:not(.eyebrow){max-width:760px;color:#47605a;font-size:20px;line-height:1.7}
.authority-layout{display:grid;grid-template-columns:minmax(0,720px) minmax(230px,1fr);gap:clamp(50px,8vw,100px);align-items:start;padding-top:80px}
.authority-body section{padding:0 0 58px;margin:0 0 58px;border-bottom:1px solid rgba(16,37,33,.13)}
.authority-body h2{font-family:Georgia,serif;font-size:clamp(32px,4vw,46px);font-weight:500;line-height:1.15;letter-spacing:-.03em;margin:0 0 24px}
.authority-body p{font-family:Georgia,serif;font-size:19px;line-height:1.82;color:#354f48;margin:0 0 20px}
.authority-body ul{padding:8px 0 0 25px}.authority-body li{padding:5px 0;color:#354f48;font-size:16px;line-height:1.65}
.authority-sources ol{padding-left:24px}.authority-sources li{padding:9px 0 13px 6px}.authority-sources a{display:block;color:#1d5d4e;font-weight:750;line-height:1.45}.authority-sources span{display:block;color:#60736e;font-size:14px;line-height:1.5;margin-top:4px}
.authority-layout aside{position:sticky;top:30px;padding:27px;border:1px solid rgba(16,37,33,.14);border-radius:17px;background:#eef3eb}
.authority-layout aside ul{list-style:none;margin:0;padding:0}.authority-layout aside li{display:flex;gap:11px;padding:13px 0;border-bottom:1px solid rgba(16,37,33,.11);font-size:14px;line-height:1.5}.authority-layout aside li:last-child{border:0}.authority-layout aside span{color:#397064;font-weight:900}
.authority-related{padding-top:80px}
@media(max-width:780px){.authority-layout{grid-template-columns:1fr;padding-top:55px}.authority-layout aside{position:static;grid-row:1}.authority-body p{font-size:18px}.authority-hero h1{font-size:46px}}
`
