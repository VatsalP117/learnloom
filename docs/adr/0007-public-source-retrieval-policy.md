# ADR-0007: Public source retrieval policy

Status: accepted
Date: 2026-08-12

## Context

Learnloom can build a learning path from only a topic. That promise requires
retrieving useful public web sources without asking the learner to assemble a
reading list first. Many legitimate public sites do not publish a
`robots.txt` file, and the absence of that file does not communicate a denial
of access or grant a license to republish the site's content.

The native source client already limits response size and time, identifies
itself, pins DNS resolution, rejects credentials in URLs, rejects non-public
network destinations, and revalidates every redirect. The product still needs
an explicit policy separating public retrieval from access-control bypass and
from permission to republish source text.

## Decision

- A missing `robots.txt` file does not block discovery or retrieval of a
  publicly accessible HTTP(S) source.
- Retrieval may use public pages discovered by Learnloom, supplied by a
  learner, or present in an existing source catalog.
- Learnloom does not bypass authentication, paywalls, CAPTCHA challenges,
  explicit access denials, credential requirements, or other technical access
  controls. Browser automation and authenticated scraping remain out of scope.
- Source requests continue through the bounded, DNS-pinned fetch path. Private,
  loopback, link-local, credentialed, and unsafe redirect targets remain
  prohibited regardless of product demand.
- Retrieval does not imply ownership, endorsement, or a license to republish a
  source. Learnloom uses source material to create an attributed instructional
  synthesis and keeps citations so learners can inspect the original.
- Public Dossiers must not be used as a substitute for publishing substantial
  portions of a source. They retain source attribution, correction/reporting,
  moderation, and removal controls.
- A credible source-owner or rights-holder complaint is reviewed through
  `support@learnloom.blog`. Learnloom may block future retrieval and remove,
  correct, unpublish, or de-index affected public material while preserving the
  minimum evidence needed to handle the report.
- Source snapshots follow the account and learning-content lifecycle. This
  decision does not invent a shorter automatic retention period that the
  system does not currently enforce.

`robots.txt` is neither treated as a copyright license nor used as the sole
legal conclusion about a source. This product decision should be reviewed with
counsel before broad commercial scale, but source discovery has no runtime
approval flag and missing `robots.txt` is not a launch blocker.

## Consequences

- Topic-only onboarding can search the public web without failing merely
  because a site omitted `robots.txt`.
- Security and access-control boundaries remain fail closed.
- Citations, bounded synthesis, complaints, and public-content removal are part
  of the operating model rather than optional presentation details.
- Legal review remains a launch-evidence task; it is not represented as a
  technical crawler permission check.
