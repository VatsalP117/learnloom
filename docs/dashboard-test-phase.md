# Multi-newsletter dashboard

The dashboard is deliberately local and single-user. It manages Newsletter
configuration, scheduling, Issue generation, history, previews, recipients,
and Resend delivery receipts.

## Run the offline demo

Use two terminals from the repository.

Terminal one starts the dashboard:

```sh
npm run dashboard:demo
```

Open `http://127.0.0.1:3000`, create one or more Newsletters, and select **Run
now** on each.

Terminal two processes all queued Issues:

```sh
npm run worker:demo
```

Refresh a Newsletter detail page to open its generated Dossier preview. Demo
mode uses deterministic local Source Items and never calls a model or delivery
adapter.

The SQLite Workspace is stored under `data/workspace.sqlite`; generated
Dossiers and Learning History remain in the existing profile-scoped
directories. Each Newsletter ID maps internally to one profile so histories do
not mix.

## Run with the configured model

```sh
node bin/learn.mjs serve --config config.json
node bin/learn.mjs worker --config config.json
```

The worker generates queued Issues, then drains pending email deliveries. An
enabled Resend entry in `config.json` supplies the sender, subject prefix, and
API-key environment variable. Newsletter recipient lists are configured in the
dashboard and override that entry's legacy `to` field.

Generation completion and the pending email receipt are committed together in
SQLite. A failed email remains failed until **Retry email** is selected; retry
loads the immutable Dossier files and does not invoke the model again.
Disabling email durably cancels pending and failed receipts without reviving
them when email is later re-enabled. A receipt already being delivered remains
visible as in flight.

If the worker transmits a request but receives no reliable Resend response, the
receipt becomes **Unknown** rather than retryable. This avoids presenting an
unsafe retry when the first email may already have been accepted.

The worker checks schedules every 30 seconds. A deterministic scheduled Issue
is created once per Newsletter and local date. **Run now** creates a separate
manual Issue and returns immediately; model work never runs inside the HTTP
request.

## Docker Compose

```sh
docker compose up -d dashboard worker
docker compose logs -f dashboard worker
```

The dashboard roles use a Compose profile so an existing plain
`docker compose up` does not unexpectedly replace the legacy finite-job
behavior. Naming `dashboard worker` explicitly activates only these two roles.

The dashboard is published only at `127.0.0.1:3000` on the VM. To reach it from
your computer without exposing it publicly:

```sh
ssh -L 3000:127.0.0.1:3000 user@your-vm
```

Then open `http://127.0.0.1:3000` locally.

## Security limit

The dashboard has CSRF protection and strict browser security headers, but it
does not have authentication. Do not publish port 3000 to a public interface.
The CLI refuses a non-loopback listener unless `--allow-remote` is supplied;
Compose uses that explicit opt-in inside the container while binding the host
port to loopback.

Requests also require a loopback `Host` header to prevent localhost DNS
rebinding. If a trusted reverse proxy uses a different hostname, add it
explicitly with `--allowed-host dashboard.internal`. The option can be repeated;
do not add an untrusted public hostname before authentication exists.

## Crash recovery

If generation is interrupted, the Issue can remain `generating` and the Daily
Run can leave its owner lock. Learnloom does not guess that either is stale.
Confirm that no worker is active before manually removing a stale lock.

A worker interrupted while a receipt is `delivering` also requires operator
inspection; automatic stale-claim recovery is later operational hardening.
