# Deploy Learnloom on Dokploy

This guide assumes the VM already runs Dokploy and its Traefik ingress. It uses
[`compose.dokploy.yaml`](../compose.dokploy.yaml) to deploy Learnloom, Postgres,
MinIO, the worker, and private SearXNG discovery as one Compose service.

The hostname contract is:

| Host | Purpose |
| --- | --- |
| `learnloom.blog` | Public marketing site |
| `www.learnloom.blog` | Redirects to the marketing site |
| `app.learnloom.blog` | Clerk sign-in, dashboard, API, health, and webhooks |
| `*.learnloom.blog` | Public Personal Sites such as `wutsell.learnloom.blog` |

The application already enforces this contract. A request for any other host,
a nested subdomain, an invalid username, or a reserved username is rejected.

## 1. Before deploying

Have these accounts and values ready:

- The Git repository and branch containing this deployment configuration.
- A Clerk production instance for `learnloom.blog`.
- An OpenAI-compatible model API key. The checked-in default is DeepSeek.
- A verified Resend domain and sender, such as
  `Learnloom <dossiers@learnloom.blog>`.
- Access to the DNS provider for `learnloom.blog`.

Generate URL-safe secrets locally. Do not paste the output into chat, source
control, or deployment logs:

```sh
openssl rand -hex 24 # POSTGRES_PASSWORD
openssl rand -hex 32 # CSRF_SECRET
openssl rand -hex 16 # S3_ACCESS_KEY_ID
openssl rand -hex 32 # S3_SECRET_ACCESS_KEY
openssl rand -hex 32 # SEARXNG_SECRET
```

Start from [`.env.dokploy.example`](../.env.dokploy.example). `POSTGRES_PASSWORD`
must remain URL-safe because the Compose file inserts it into `DATABASE_URL`.

## 2. Configure DNS

Create these records before asking Dokploy to issue or use certificates.
Replace `VM_IP` with the public address of the Dokploy VM.

| Type | Name | Value |
| --- | --- | --- |
| `A` | `@` | `VM_IP` |
| `A` | `app` | `VM_IP` |
| `A` | `*` | `VM_IP` |
| `CNAME` | `www` | `learnloom.blog` |

An `A` record for `www` pointing to `VM_IP` is also valid. Add equivalent
`AAAA` records only if the VM, firewall, and Dokploy ingress accept IPv6.

The wildcard record makes a future username work without creating another DNS
record. It does not provide TLS by itself.

### Wildcard TLS

A normal Let's Encrypt HTTP challenge cannot issue `*.learnloom.blog`. Use one
of these approaches:

1. If DNS is on Cloudflare, create a Cloudflare Origin CA certificate covering
   both `learnloom.blog` and `*.learnloom.blog`, set Cloudflare SSL/TLS mode to
   **Full (strict)**, and upload the certificate and private key in Dokploy's
   **Certificates** page.
2. With another DNS provider, issue a certificate for those same two names
   using a DNS-01 ACME client, then upload the full chain and private key to
   Dokploy.

Never commit the private key. Confirm the certificate covers both the apex and
wildcard; a wildcard certificate does not cover the apex automatically.

## 3. Configure Clerk production

In Clerk:

1. Create or select the **production** instance and set its application/root
   domain to `learnloom.blog`.
2. Complete every DNS record Clerk displays. If Cloudflare is used, keep
   Clerk's verification/FAPI CNAME records **DNS only** until Clerk finishes
   verification and certificate deployment.
3. Copy the production `pk_live_...` value into both
   `VITE_CLERK_PUBLISHABLE_KEY` and `CLERK_PUBLISHABLE_KEY`.
4. Copy the production secret key, PEM JWT public key, and exact Frontend API
   HTTPS origin into `CLERK_SECRET_KEY`, `CLERK_JWT_KEY`, and
   `CLERK_FRONTEND_ORIGIN`.
5. Restrict Clerk's allowed subdomains/origins to `app.learnloom.blog`. Public
   Personal Sites do not run authenticated application code.
6. Add a webhook endpoint at
   `https://app.learnloom.blog/webhooks/clerk`, subscribe to `user.created`,
   `user.updated`, and `user.deleted`, and store its `whsec_...` signing secret
   as `CLERK_WEBHOOK_SECRET`.
7. If social sign-in is enabled, add the provider's production OAuth
   credentials and test its callback.

The Clerk publishable key and root domain are compiled into the frontend. Any
change to either requires a fresh image build, not only a container restart.

## 4. Create the Dokploy Compose service

1. In Dokploy, create or select the Learnloom project and production
   environment.
2. Add a **Docker Compose** service from the Git provider.
3. Select the repository and release branch.
4. Set the Compose path to `./compose.dokploy.yaml`.
5. Leave **Isolated Deployments** off for this stateful service so named volume
   identities remain stable across deployments.
6. Paste the completed `.env.dokploy.example` values into the service's
   Environment page. Do not upload the example file with real secrets.
7. Do not add host port mappings. Only Dokploy/Traefik should reach the web
   container on port `3000`.

The Compose deployment runs a one-shot migration after Postgres becomes
healthy. The web and worker roles start only after migrations and bucket
creation succeed. Postgres, MinIO, SearXNG, Valkey, and worker metrics have no
published ports.

`ALLOW_INSECURE_PRIVATE_SERVICES=true` is intentionally fixed in this Compose
file. It permits non-TLS connections only to local/private service hosts. It
does not permit an unencrypted public Postgres or S3 endpoint. If Postgres or
object storage is moved outside the private Compose network, use TLS and remove
that override.

## 5. Route the Compose service

The checked-in Compose file configures Traefik labels for the apex, `www`,
`app`, and valid first-level Personal Site subdomains. Do not add the same
hosts in Dokploy's **Domains** tab: some Dokploy installations deploy Compose
services through Docker Swarm, which reads `deploy.labels` rather than the
ordinary labels injected by that UI.

Click **Preview Compose** before deployment. Verify that Dokploy attaches only
`web` to its ingress network, preserves the `learnloom-*` Traefik labels, and
does not publish Postgres, MinIO, SearXNG, Valkey, or port `9090`.

The application performs the stricter final username and reserved-name checks.
The custom wildcard certificate must already be loaded into Traefik.

## 6. Deploy

Click **Deploy** and watch the logs in this order:

1. `postgres` and `minio` start.
2. `create-bucket` exits successfully.
3. `migrate` logs no error and exits with code `0`.
4. `web` logs `web listening` on `:3000`.
5. `worker` logs its metrics listener and begins polling.
6. `searxng` and `searxng-valkey` remain healthy/running.

MinIO uses the official pinned image `minio/minio:RELEASE.2025-09-07T16-13-09Z`.
Do not replace it with an unpinned `latest` image. Set `MINIO_IMAGE` only when
using an approved private mirror of that exact release.

## 7. Verify the release

Run these checks from outside the VM:

```sh
curl -fsS https://app.learnloom.blog/healthz
curl -fsS https://app.learnloom.blog/readyz
curl -I https://learnloom.blog/
curl -I https://www.learnloom.blog/
curl -I https://wutsell.learnloom.blog/
```

Expected results:

- `/healthz` returns `{"status":"ok"}`.
- `/readyz` returns `{"status":"ready"}`.
- The apex serves the marketing site.
- `www` returns a permanent redirect to the apex.
- An unclaimed/private username returns `404`; after claiming `wutsell` and
  publishing the site, the same URL serves the Personal Site.

Then complete one browser flow:

1. Sign up at `https://app.learnloom.blog/sign-up`.
2. Claim `wutsell` (or the desired available lowercase username).
3. Create a Dossier and confirm its first Issue queues.
4. Watch the worker logs until generation finishes.
5. Publish the Personal Site and open
   `https://wutsell.learnloom.blog`.
6. Confirm the Resend delivery and Clerk webhook attempts succeed.

## 8. Backups and routine updates

Before public signup:

- Configure scheduled Postgres backups to storage outside this VM.
- Back up the `object-data` volume or migrate artifacts to a versioned,
  encrypted S3 provider.
- Test a restore in an isolated environment.
- Keep `postgres-data`, `object-data`, and Valkey/SearXNG volume deletion
  protection enabled in Dokploy.

For an update, deploy a commit/image that has passed tests. Migrations are
forward-only: roll back web and worker to the previous image if necessary, but
do not reverse a production migration in place.

Finish every item in
[`docs/public-launch-checklist.md`](public-launch-checklist.md) before enabling
unrestricted public signup.

## Troubleshooting

- **`421 Misdirected Request`**: the request host is missing from the hostname
  contract or Traefik is forwarding an unexpected `Host` header.
- **Wildcard site has a certificate warning**: DNS works, but the served
  certificate does not cover `*.learnloom.blog`.
- **Web exits with `DATABASE_URL must require TLS` or `S3_ENDPOINT must use
  HTTPS`**: the private-service opt-in is missing, or the endpoint hostname is
  not local/private.
- **Web exits with a Clerk production error**: paste the PEM public key with
  its real line breaks and use the exact HTTPS Frontend API origin.
- **Discovery gets `403` from SearXNG**: confirm JSON remains enabled in
  `infra/searxng/settings.yml`.
- **`readyz` returns `503`**: inspect web logs for the failing Postgres or MinIO
  readiness dependency before restarting containers.
