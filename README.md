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

## Five-minute local start

Requirements: Node.js 22 or newer. There are no runtime dependencies.

```sh
git clone https://github.com/VatsalP117/learnloom.git
cd learnloom
cp config.example.json config.json
cp .env.example .env
npm test
npm run demo
```

The demo uses no network, model credits, or email. It writes a Markdown and
JSON Dossier under `output/`.

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
- A filesystem lock prevents overlapping runs for the same profile and date.

The application home defaults to the directory containing `config.json`. Set
`LEARNLOOM_HOME` to place all state under one durable directory.

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
The configured base URL must expose `/chat/completions` and `/models`.

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
Compose, opens no inbound ports, and stores durable state in `learnloom-data`.
Use the supplied host systemd timer for the 9:00 a.m. schedule.

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
