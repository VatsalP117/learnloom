# Learnloom

Learnloom is a self-hosted personal learning engine. It retrieves fresh
RSS/Atom material, weaves it together with your Learning History, and produces
a source-indexed daily Dossier built through:

1. Coherent source curation and bounded full-text enrichment
2. An explicit Learning Blueprint and continuity bridge
3. Source-grounded research and skeptical evidence review
4. A mechanism-focused lesson, worked example, and misconception check
5. Retrieval questions, answer key, and an application challenge
6. A final editorial rewrite and deterministic quality gate

Run it locally, schedule it on macOS, or deploy the finite Docker job to a VM.
Use your own DeepSeek or OpenAI-compatible key, a Command Code subscription,
or the deterministic offline demo. Optionally receive each Dossier through
Resend email.

The v0.5 dashboard runs multiple topic-focused Newsletters. Each Newsletter has
its own schedule, timezone, recipients, Issue history, Learning History,
Dossier previews, durable email delivery status, and an optional AI Exploration
setting.

> **CLI deprecation:** the `learn` command is retained as a compatibility
> launcher for existing local, scheduled, and container deployments. New hosted
> administration and migration features must not be added to it; dedicated
> service entrypoints and authenticated operator workflows will replace it
> before removal.

## Five-minute local start

Requirements: Node.js 22.13 or newer. There are no runtime dependencies.

```sh
git clone https://github.com/VatsalP117/learnloom.git
cd learnloom
cp config.example.json config.json
cp .env.example .env
npm test
npm run demo
```

The demo uses no network, model credits, or email. It writes immutable
generation-versioned Markdown and JSON Dossiers under `output/`.

For a live direct DeepSeek run:

```sh
export DEEPSEEK_API_KEY="your-key"
npm run doctor
npm start
```

Secrets are read only from environment variables. Learnloom never reads a key
from `config.json`, writes one to a run record, or includes one in an error.

## How a Daily Run behaves

Each profile and local date has one deterministic Daily Run:

- The Dossier is persisted before external delivery.
- A same-day rerun reuses the saved Dossier.
- Successful destinations are skipped.
- Failed destinations retry without spending model tokens again.
- `learn run --force` explicitly regenerates the Dossier.
- An owner-token filesystem lock prevents overlapping runs for the same profile
  and date. Locks are never stolen automatically.

The application home defaults to the directory containing `config.json`. Set
`LEARNLOOM_HOME` to place all state under one durable directory.

## Content-quality pipeline

Learnloom first asks the model to select three to five complementary Source
Items. It then attempts bounded full-text extraction from those pages; an
unavailable, thin, blocked, or unsupported page falls back to its feed summary
without failing the Daily Run.

The selected material becomes a structured Learning Blueprint before research
or lesson prose is written. A final editor rewrites the lesson and practice,
then Learnloom rejects the Dossier if it lacks required teaching sections,
valid Source Item citations, retrieval questions, an application challenge, or
a collapsed answer key. Dossier JSON includes the deterministic quality score
and provenance for every selected source.

Direct Daily Runs can enable the optional synthetic section:

```json
{
  "content": {
    "aiExplorationEnabled": true,
    "maxArticleBytes": 524288,
    "maxArticleCharacters": 16000
  }
}
```

For Newsletters, use **Generation settings** in the dashboard. AI Exploration
is off by default. When enabled it is rendered in a separate labelled panel,
uses no source citation markers, and is excluded from core retrieval practice.

## Run the multi-newsletter dashboard

Run the dashboard and worker in two terminals:

```sh
npm run dashboard:demo
npm run worker:demo
```

Open `http://127.0.0.1:3000`. Create Newsletters, queue **Run now**, rerun the
one-shot demo worker, then refresh the Newsletter page to open generated Issue
previews.

Demo mode has no Resend configuration and does not send email. For live
generation and delivery, run both commands with `--config config.json`, enable
the installation-level Resend entry, and configure recipients on each
Newsletter page.

The dashboard is local-only and has no authentication. See
[the dashboard guide](docs/dashboard-test-phase.md).

## Hosted personal-subdomain foundation

Learnloom now has an explicit hostname-routing boundary for the hosted product:

- `learnloom.blog` is the apex surface.
- `app.learnloom.blog` is reserved for the authenticated control plane.
- `<username>.learnloom.blog` is reserved for a learner's reading site.
- local/self-hosted mode retains the existing exact Host allowlist.

The routing foundation can be configured with:

```sh
export LEARNLOOM_DEPLOYMENT_MODE=hosted
export LEARNLOOM_ROOT_DOMAIN=learnloom.blog
export LEARNLOOM_APP_ORIGIN=https://app.learnloom.blog
```

Hosted mode requires Clerk server keys and a matching
`VITE_CLERK_PUBLISHABLE_KEY` at frontend build time. Google-authenticated users
are provisioned into isolated local accounts, must claim one username, and can
access only their own Newsletters and Issues. Sites start private. Once
published, the claimed hostname serves a public home, per-topic archives,
canonical Dossier pages, `robots.txt`, and a sitemap. Newsletter streams and
individual Dossiers can be hidden independently, and hosted email includes the
canonical reading link when the content is public.

The hosted path is still an implementation preview: production DNS/TLS, Clerk
production-domain validation, feed-fetch hardening, deletion lifecycle, and
operational observability must be completed before public launch. See
[the public launch checklist](docs/public-launch-checklist.md) for the bounded
release gate and
[the hosted-subdomains implementation plan](docs/hosted-subdomains-implementation-plan.md)
for the full design.

## Model providers

### Bring your own OpenAI-compatible key

The public example uses DeepSeek:

```json
{
  "provider": {
    "kind": "openai-compatible",
    "baseUrl": "https://api.deepseek.com",
    "apiKeyEnv": "DEEPSEEK_API_KEY",
    "model": "deepseek-v4-pro",
    "maxTokens": 8192,
    "timeoutSeconds": 600,
    "retries": 2
  }
}
```

Change `baseUrl`, `apiKeyEnv`, and `model` for another compatible provider.
The configured base URL must use HTTPS and expose `/chat/completions` and
`/models`. For a local model server only, loopback HTTP can be enabled with
`"allowInsecureHttp": true`; remote plaintext HTTP is always rejected.

### Command Code subscription

Local Command Code usage remains supported:

```json
{
  "provider": {
    "kind": "commandcode",
    "executable": "cmd",
    "model": "deepseek-v4-pro",
    "timeoutSeconds": 600
  }
}
```

Install and authenticate the official CLI:

```sh
npm install -g command-code
cmd login
npm run doctor
```

Learnloom invokes documented non-interactive print mode with plan permissions,
one turn, and no model tools.

## Email through Resend

Verify a sending domain in Resend, set `RESEND_API_KEY`, and enable the example
delivery:

```json
{
  "deliveries": [
    {
      "id": "morning-email",
      "kind": "resend",
      "enabled": true,
      "apiKeyEnv": "RESEND_API_KEY",
      "from": "Learnloom <daily@updates.example.com>",
      "to": "you@example.com",
      "subjectPrefix": "Learnloom"
    }
  ]
}
```

Email HTML is rendered from the canonical Dossier with all model/source text
escaped. Only HTTP(S) source links become clickable. A deterministic Resend
idempotency key prevents duplicate sends during normal retries.

For dashboard Newsletters, the enabled Resend entry supplies the API-key
environment variable, sender, and subject prefix. Its `to` value remains the
recipient for legacy `learn run`; each Newsletter's recipients are managed in
the dashboard. Generation and email are separate queues: a failure is visible
on the Issue and **Retry email** reuses the saved Dossier without another model
call.

Disabling email cancels unsent queued/failed receipts; re-enabling applies only
to future Issues. If Resend may have accepted a request but its response was
lost, Learnloom marks the outcome **Unknown** and does not offer ordinary retry,
because sending again could duplicate the email.

## Deploy on a VM

```sh
cp config.example.json config.json
cp .env.example .env
docker compose build
docker compose run --rm learnloom doctor --config /app/config.json
docker compose run --rm learnloom run --config /app/config.json
```

The image runs as a non-root user, has a read-only root filesystem under
Compose, and stores durable state in `learnloom-data`. The legacy finite job
opens no port. Use the supplied host systemd timer for its 9:00 a.m. schedule.

For the multi-newsletter service, run the dashboard and worker roles:

```sh
docker compose up -d dashboard worker
```

The dashboard is bound to VM loopback at `127.0.0.1:3000`; use an SSH tunnel
and do not expose it publicly before authentication is implemented.

See [the complete VM guide](docs/vm-deployment.md).

## macOS scheduling

After a successful live run:

```sh
node bin/learn.mjs schedule install
node bin/learn.mjs schedule status
```

Choose another time with `--hour` and `--minute`; remove the job with
`node bin/learn.mjs schedule remove`.

## Commands

```text
learn init [--config path] [--force]
learn run [--config path] [--demo] [--force]
learn doctor [--config path]
learn serve [--config path] [--demo] [--host 127.0.0.1] [--port 3000]
learn worker [--config path] [--demo] [--once] [--interval 30]
learn schedule install [--config path] [--hour 9] [--minute 0]
learn schedule status
learn schedule remove
```

## Sources and safety

- One failed feed becomes a warning; all feeds failing aborts before model use.
- Feed text is untrusted reference material, never model instructions.
- Article enrichment validates HTTP(S), credentials, DNS addresses, redirects,
  response type, download size, and timeout. It does not execute page scripts
  or bypass paywalls.
- Model adapters expose no downstream tools.
- Source/model HTML is escaped before email rendering.
- Feed URLs remain operator-controlled. Article links discovered inside trusted
  feeds are screened against private and reserved network destinations, but
  Learnloom is not a general multi-tenant crawler.
- Important claims should be verified through the Dossier's source links.

## Development

```sh
npm test
npm run check
npm run demo
```

Architecture vocabulary lives in [CONTEXT.md](CONTEXT.md), and the current
design is documented in [docs/architecture.md](docs/architecture.md).

Learnloom is released under the [MIT License](LICENSE).
