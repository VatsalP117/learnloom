# Learnloom

Learnloom is a self-hosted personal learning engine. It retrieves fresh
RSS/Atom material, weaves it together with your Learning History, and produces
a source-indexed daily Dossier with:

1. A researched theme
2. A skeptical evidence review
3. A focused lesson
4. Retrieval questions and an application exercise

Run it locally, schedule it on macOS, or deploy the finite Docker job to a VM.
Use your own DeepSeek or OpenAI-compatible key, a Command Code subscription,
or the deterministic offline demo. Optionally receive each Dossier through
Resend email.

The v0.3 test phase also includes a small single-user dashboard for running
multiple topic-focused Newsletters. Each Newsletter has its own schedule,
timezone, Issue history, Learning History, and Dossier previews.

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

## Test the multi-newsletter dashboard

Run the dashboard and worker in two terminals:

```sh
npm run dashboard:demo
npm run worker:demo
```

Open `http://127.0.0.1:3000`. Create Newsletters, queue **Run now**, rerun the
one-shot demo worker, then refresh the Newsletter page to open generated Issue
previews.

The dashboard is local-only and has no authentication. Live delivery is
forcibly disabled in the Newsletter worker during this phase, even if the base
configuration enables Resend.

See [the dashboard test guide](docs/dashboard-test-phase.md).

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

For the multi-newsletter test phase, run the dashboard and worker roles:

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
- Model adapters expose no downstream tools.
- Source/model HTML is escaped before email rendering.
- Feed URLs are operator-controlled. The current fetcher is not suitable for
  accepting untrusted multi-tenant URLs because it does not block private
  network destinations.
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
