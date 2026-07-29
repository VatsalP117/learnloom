# Learnloom SEO next steps

This document is the working roadmap for search discovery after the initial SEO
implementation. It separates work that belongs in the repository from external
account, deployment, and distribution work.

External setup can remain deferred until the repository work is ready to
deploy. When beginning the external checklist, complete it in the order shown
so search engines never inspect a partially configured release.

## Current implementation

The repository already provides:

- a server-rendered, problem-led landing-page architecture;
- long-form learning guides, product transparency, and editorial-principles
  pages;
- canonical URLs, descriptions, Open Graph metadata, and relevant structured
  data;
- apex and learner-site `robots.txt` and XML sitemaps;
- permanent canonical redirects;
- `noindex` protection for the authenticated app, legal pages, missing pages,
  private material, empty archives, and unqualified content;
- owner-controlled search indexing that is separate from public visibility and
  defaults off;
- a curated `/examples` gallery requiring both operator allowlisting and owner
  indexing consent; and
- automated coverage for metadata, sitemap, canonical, indexing, configuration,
  and publishing behavior.

## Next repository phases

### Phase 1: Search measurement adapter

Goal: connect search traffic to meaningful product activation without coupling
the product to one analytics provider.

- [ ] Define a small first-party event contract for:
  - SEO landing-page view;
  - sign-up CTA selection;
  - sign-up completion;
  - first learning-stream creation;
  - first generated Dossier;
  - first completed Dossier; and
  - search-indexing opt-in.
- [ ] Preserve the original landing path and referrer through sign-up.
- [ ] Add campaign parameters only where Learnloom controls the outbound link;
  do not rewrite normal organic-search URLs.
- [ ] Choose an analytics adapter with a disabled-by-default configuration.
- [ ] Update privacy disclosure and consent behavior if the selected provider
  uses non-essential cookies or cross-site identifiers.
- [ ] Add tests proving that analytics failure never blocks learning or
  publishing flows.

Acceptance criteria:

- Organic landing page to first-Dossier activation can be measured.
- No secret or analytics identifier is required to run local development.
- No private Dossier content, source content, email address, or account ID is
  sent to analytics.

### Phase 2: Social preview system

Goal: make shared Learnloom pages recognizable and useful before a visitor
clicks.

- [ ] Create a stable 1200×630 default social image for the apex site.
- [ ] Add route-specific social images for product pages, guides, and examples.
- [ ] Generate safe Dossier preview images from title, stream, learner display
  name, and Learnloom branding.
- [ ] Never place private content, email addresses, source excerpts, or
  unpublished titles in preview images.
- [ ] Add image width, height, MIME type, and alt metadata where supported.
- [ ] Add automated metadata tests and inspect representative cards with social
  preview tools after deployment.

Acceptance criteria:

- The homepage, one guide, `/examples`, one learner home, and one Dossier all
  produce a valid preview.
- A preview remains correct if the page is shared without JavaScript.

### Phase 3: Index-quality and moderation controls

Goal: allow useful public learning to be discovered without turning learner
subdomains into an uncontrolled scaled-content surface.

- [ ] Define an index-eligibility policy for individual Dossiers, including:
  - successful generation and explicit publication;
  - owner search-indexing consent;
  - a non-empty objective and substantive lesson;
  - visible source provenance;
  - no unresolved generation or artifact failure; and
  - no abuse, impersonation, or unsafe publication flag.
- [ ] Record the eligibility decision and reason so it is inspectable.
- [ ] Exclude ineligible Dossiers from sitemaps and return `noindex`.
- [ ] Add an operator action to remove a site or Dossier from discovery without
  deleting the learner's private record.
- [ ] Add a report/correction path for public pages.
- [ ] Add a documented response process for copyright, privacy, and inaccurate
  content reports.
- [ ] Replace the deployment allowlist with an authenticated curation workflow
  only when the number of featured sites makes environment configuration
  cumbersome.

Acceptance criteria:

- Public does not automatically mean indexable.
- Every indexed learner page has an explainable eligibility decision.
- Removal from discovery does not require destructive account deletion.

### Phase 4: Search-led content iteration

Goal: expand only where real query evidence shows a distinct learner need.

- [ ] Review queries and pages in positions 5–20.
- [ ] Improve titles and snippets where impressions are healthy but
  click-through rate is weak.
- [ ] Strengthen pages that earn relevant impressions but do not drive
  activation.
- [ ] Consolidate pages that compete for the same intent.
- [ ] Create a new page only when its primary search intent cannot be served by
  an existing page.
- [ ] Add comparison pages only when the comparison is factually supportable
  and useful without disparaging another product.
- [ ] Refresh guides when product behavior, cited evidence, or learner needs
  change.

Initial query clusters to monitor:

- remember what you read;
- keep up with research or a changing field;
- personal learning system;
- personal curriculum;
- AI learning assistant;
- source-grounded AI learning;
- daily personalized lessons;
- alternatives to AI summaries, newsletters, RSS readers, and read-later
  tools; and
- searchable personal learning archive.

Acceptance criteria:

- Every new content page has a query hypothesis, intended visitor, conversion
  path, and review date.
- No page exists only for a trivial keyword variation.

### Phase 5: Original evidence and authority

Goal: earn citations and links through material that cannot be reproduced by a
generic summary.

- [ ] Publish worked examples showing how a source set becomes a Dossier.
- [ ] Develop privacy-safe aggregate insights only after the dataset is large
  enough and the aggregation policy has been reviewed.
- [ ] Publish transparent methodology with any original dataset or benchmark.
- [ ] Create reusable learning-stream templates for meaningful subjects and
  outcomes.
- [ ] Invite qualified educators, researchers, and practitioners to review
  relevant guides.
- [ ] Record corrections and substantive updates publicly.

Acceptance criteria:

- Original research cannot reveal an individual learner or private source set.
- Claims remain reproducible from the published methodology and evidence.

## External setup checklist

Perform this section after the SEO release is deployed to the production
domain.

### 1. Deploy and verify the release

- [ ] Deploy the current `main` image.
- [ ] Run the database migration role before the web role. Confirm schema
  migration `005_site_search_indexing.sql` is applied.
- [ ] Confirm apex, `www`, `app`, and wildcard learner domains have valid TLS.
- [ ] Confirm `www` permanently redirects to the apex domain.
- [ ] Fetch and inspect:
  - `https://learnloom.blog/`;
  - `https://learnloom.blog/robots.txt`;
  - `https://learnloom.blog/sitemap.xml`;
  - `https://learnloom.blog/guides`;
  - `https://learnloom.blog/examples`;
  - `https://app.learnloom.blog/`;
  - one public but indexing-disabled learner site;
  - one opted-in learner site; and
  - one opted-in published Dossier.
- [ ] Confirm the app returns `X-Robots-Tag: noindex, nofollow`.
- [ ] Confirm indexing-disabled learner pages return `noindex, follow`.
- [ ] Confirm opted-in, populated learner pages return `index, follow`.

### 2. Curate the initial examples

- [ ] Ask each candidate learner to review public visibility and enable search
  discovery from the Publishing screen.
- [ ] Review the learner home, sources, and recent Dossiers for quality, safety,
  and privacy.
- [ ] Configure the production web role:

  ```sh
  FEATURED_SITE_USERNAMES=maya,ada
  ```

- [ ] Redeploy the web role.
- [ ] Confirm only reviewed, public, opted-in, populated sites appear at
  `https://learnloom.blog/examples`.
- [ ] Keep an internal record of why each site was selected and its next review
  date.

### 3. Configure Google Search Console

- [ ] Create a **Domain property** for `learnloom.blog` in
  [Google Search Console](https://search.google.com/search-console/about).
  A Domain property covers the apex domain and all subdomains.
- [ ] Add the DNS TXT verification record supplied by Google.
- [ ] Wait for DNS propagation and complete verification.
- [ ] Submit `https://learnloom.blog/sitemap.xml`.
- [ ] Inspect the homepage, one problem-led page, one guide, `/examples`, one
  opted-in learner home, and one Dossier with
  [URL Inspection](https://support.google.com/webmasters/answer/9012289).
- [ ] Confirm the indexing-disabled public learner page is excluded because of
  its `noindex` directive.
- [ ] Validate representative structured data with the
  [Rich Results Test](https://search.google.com/test/rich-results).
- [ ] Do not request indexing for empty, private, hidden, or
  indexing-disabled pages.

Learner subdomains advertise their own sitemap from `robots.txt`. Search
Console's Domain property provides visibility across those subdomains. Submit
individual learner sitemaps manually only when diagnosing discovery for an
important opted-in site; do not build a bulk submission process before it is
needed.

### 4. Configure Bing Webmaster Tools

- [ ] Add `learnloom.blog` to
  [Bing Webmaster Tools](https://www.bing.com/webmasters/about).
- [ ] Import the verified property from Search Console or complete Bing's
  ownership verification.
- [ ] Submit `https://learnloom.blog/sitemap.xml`.
- [ ] Inspect crawling and indexing for the same representative URLs used in
  Search Console.
- [ ] Review crawl errors, blocked URLs, duplicate URLs, and structured-data
  reports.
- [ ] Consider implementing the
  [IndexNow protocol](https://www.indexnow.org/documentation) during a later
  repository phase if Dossier publication volume makes faster discovery
  valuable.

### 5. Configure analytics

- [ ] Select the analytics provider and document why it fits Learnloom's
  privacy requirements.
- [ ] Create separate production and staging properties/sites.
- [ ] Store identifiers and secrets in deployment configuration, never in the
  repository.
- [ ] Enable the repository's analytics adapter when it exists.
- [ ] Verify the complete anonymous funnel:
  - organic landing;
  - sign-up CTA;
  - sign-up;
  - first learning stream;
  - first Dossier; and
  - first completed Dossier.
- [ ] Exclude team, staging, uptime-monitor, and automated-test traffic.
- [ ] Set a retention period appropriate for product analysis.
- [ ] Update the privacy policy and consent UI before enabling any tracking that
  requires consent.

### 6. Validate performance and rendering

- [ ] Run [PageSpeed Insights](https://pagespeed.web.dev/) for the homepage,
  one landing page, one guide, `/examples`, and one Dossier on mobile and
  desktop.
- [ ] Use Search Console's Core Web Vitals report after enough field data is
  available.
- [ ] Verify that important text, links, canonical metadata, and structured
  data exist in the server response without requiring client-side rendering.
- [ ] Test at least one low-bandwidth mobile session and one JavaScript-disabled
  fetch.
- [ ] Fix regressions before increasing content volume.

### 7. Establish distribution and authority

- [ ] Update the GitHub repository homepage and description to point to the
  canonical product domain.
- [ ] Create accurate product profiles only on reputable directories relevant
  to learning, productivity, and knowledge tools.
- [ ] Prepare a launch page with one clear product explanation and real
  examples.
- [ ] Share the strongest guides with communities that would genuinely benefit;
  do not mass-submit links.
- [ ] Pursue partnerships with educators, researchers, newsletter authors, and
  domain experts where Learnloom can supply a useful learning artifact.
- [ ] Never purchase links, automate guest-post networks, or exchange links at
  scale.

### 8. Create the monthly review

- [ ] Record a monthly owner and review date.
- [ ] Export or dashboard:
  - non-branded impressions and clicks;
  - indexed versus submitted canonical URLs;
  - queries and pages in positions 5–20;
  - click-through rate;
  - organic sign-up and first-Dossier activation;
  - crawl errors and excluded URLs;
  - Core Web Vitals;
  - example-gallery traffic; and
  - indexed learner pages.
- [ ] Record every title, content, sitemap, or indexing-policy change with its
  date so performance changes can be interpreted.
- [ ] Review featured examples for continued consent, safety, and quality.
- [ ] Remove or consolidate content that is redundant, stale, or persistently
  unhelpful.

## First external success criteria

The initial external SEO setup is complete when:

- Google and Bing verify ownership of `learnloom.blog`;
- the apex sitemap is accepted without material errors;
- representative server-rendered pages are indexed;
- the authenticated app and indexing-disabled learner pages remain excluded;
- `/examples` contains only reviewed and opted-in sites;
- organic landing-to-first-Dossier activation can be measured;
- no critical mobile performance regression remains; and
- a named owner and monthly review date exist.

Search engines do not guarantee indexing or ranking. These steps establish
eligibility, clarity, measurement, and a sustainable improvement loop; they do
not justify publishing pages that are not useful to learners.
