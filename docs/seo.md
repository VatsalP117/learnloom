# Search discovery operations

Learnloom's search strategy is problem-led. The apex site should answer the
questions people ask before they know the product name, while public learner
sites provide carefully controlled, source-grounded examples of learning in
practice.

## Indexable surfaces

The apex sitemap contains the homepage and the server-rendered pages registered
in `internal/httpapp/seo.go`. Each page must have a distinct search intent,
substantive visible content, a canonical URL, a useful title and description,
internal links, and an honest path into the product.

The authenticated application is always served with `X-Robots-Tag: noindex,
nofollow`. Privacy and terms pages use `noindex, follow`. The legacy
`/marketing` URL permanently redirects to the homepage, and trailing-slash
variants of SEO routes permanently redirect to their canonical form.

Public learner subdomains own their own `robots.txt` and sitemap. A learner home
or topic archive is indexable only after it contains a published Dossier. A
published Dossier receives canonical, social, and `Article` metadata. Missing,
private, hidden, and empty pages must remain out of the index.

## Initial intent map

The current landing-page set covers:

- remembering what was read;
- keeping up with a changing field without information overload;
- building a personal learning curriculum;
- turning information into durable understanding;
- finding a source-grounded AI learning assistant;
- learning from trusted sources; and
- comparing Knowledge Dossiers with one-off AI summaries.

The authority layer adds substantive guides for recall, keeping up with a
changing field, and building a personal learning system. `/how-learnloom-works`
and `/editorial-principles` document product boundaries, source handling,
quality limitations, and publication controls. These pages should be updated
when the corresponding product behavior changes.

New pages should be added only after query evidence shows a distinct learner
need. Do not create pages for trivial keyword variations. Merge overlapping
intent into an existing page and improve that page instead.

## Deployment checklist

After deploying an SEO release:

1. Fetch `/`, `/robots.txt`, `/sitemap.xml`, one SEO page, the app homepage,
   one populated learner home, one empty learner home, and one public Dossier.
2. Confirm status, canonical URL, `X-Robots-Tag`, title, description, and
   structured data for each response.
3. Verify the domain property in Google Search Console.
4. Submit `https://learnloom.blog/sitemap.xml` in Google Search Console and
   Bing Webmaster Tools.
5. Inspect the homepage and representative SEO and Dossier URLs with Google's
   URL Inspection and Rich Results tools.
6. Confirm that the rendered homepage matches what a normal visitor sees.
7. Record the release date in the search-performance dashboard.

Search Console and Bing verification tokens belong in deployment configuration,
not in this repository.

## Measurement

Review performance monthly using:

- non-branded impressions and clicks;
- indexed canonical URLs versus submitted URLs;
- queries and pages in positions 5–20;
- click-through rate by landing page and query;
- landing-page sign-up rate;
- sign-up to first-stream and first-Dossier activation;
- crawl errors, duplicate canonicals, and excluded URLs; and
- pages whose impressions, engagement, or conversions are declining.

Pages with promising impressions should receive clearer answers, stronger
examples, and relevant internal links. Overlapping or persistently unhelpful
pages should be consolidated, redirected, or removed.

## Editorial and abuse guardrails

- Write for a learner's problem, not a search-engine variation.
- Keep feature claims aligned with the shipped product.
- Prefer original explanations, examples, source analysis, and product evidence
  over commodity summaries.
- Keep source provenance visible and distinguish synthetic exploration.
- Do not buy links, automate mass guest posts, or publish unreviewed bulk pages.
- Do not make private material indexable.
- Remove empty, failed, duplicate, hidden, or abusive public content from
  sitemaps and indexing.
