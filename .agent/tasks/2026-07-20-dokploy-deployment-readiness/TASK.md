# Dokploy Deployment Readiness

Prepare Learnloom for deployment on an existing Dokploy-managed VM.

The production hostname contract is:

- `learnloom.blog`: public marketing site
- `www.learnloom.blog`: redirect to the marketing site
- `app.learnloom.blog`: authenticated application and API
- `<username>.learnloom.blog`: public personal learning site, for example
  `wutsell.learnloom.blog`

The deployment must preserve secure production defaults, keep stateful services
private, run migrations before the web and worker roles, and document the exact
Dokploy, DNS, wildcard TLS, Clerk, and verification steps.
