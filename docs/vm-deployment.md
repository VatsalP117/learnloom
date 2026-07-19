# VM deployment

Learnloom is a finite batch process. The host scheduler starts one container,
the Daily Run completes or fails, and the container exits. No inbound port is
required.

## 1. Prepare the VM

Install Git, Docker Engine, and the Docker Compose plugin. Clone the repository
to `/opt/learnloom`:

```sh
sudo git clone https://github.com/VatsalP117/learnloom.git /opt/learnloom
sudo chown -R "$USER":"$USER" /opt/learnloom
cd /opt/learnloom
```

## 2. Configure secrets and interests

```sh
cp .env.example .env
cp config.example.json config.json
chmod 600 .env config.json
```

Edit `.env` and set `DEEPSEEK_API_KEY`. Edit `config.json` to choose interests,
feeds, time zone, learner profile, and model.

To send email:

1. Verify a sending domain in Resend.
2. Set `RESEND_API_KEY` in `.env`.
3. Set a valid `from` address and a placeholder legacy `to` address in
   `config.json`.
4. Change the Resend delivery's `enabled` field to `true`.

Neither file is copied into the image or tracked by Git.

The dashboard stores each Newsletter's enabled state and recipients in SQLite;
it never stores the Resend key. The `to` address in `config.json` continues to
apply to the finite `learn run` command, while dashboard Newsletter recipients
apply to worker deliveries.

## 3. Build and diagnose

```sh
docker compose build
docker compose run --rm learnloom doctor --config /app/config.json
docker compose run --rm learnloom run --demo --config /app/config.json
```

The demo does not call a model or send configured deliveries. A live run is:

```sh
docker compose run --rm learnloom run --config /app/config.json
```

The named `learnloom-data` volume contains Dossiers, Learning History, Daily
Run records, Workspace SQLite data, Delivery Receipts, locks, and logs. Back up
that volume with the VM.

For the long-running multi-newsletter dashboard and worker:

```sh
docker compose up -d dashboard worker
docker compose logs -f dashboard worker
```

Reach the loopback-only dashboard over an SSH tunnel:

```sh
ssh -L 3000:127.0.0.1:3000 user@your-vm
```

Then open `http://127.0.0.1:3000`. Do not expose it publicly; authentication is
not implemented.

Hosted mode validates `LEARNLOOM_ROOT_DOMAIN`, an exact HTTPS
`LEARNLOOM_APP_ORIGIN`, and Clerk server keys. The frontend image must also be
built with the matching `VITE_CLERK_PUBLISHABLE_KEY`. Authenticated users are
mapped to account-scoped Workspace access and claim one private-by-default
site. Personal-site rendering, public ingress hardening, and untrusted feed URL
protection are still incomplete, so keep using the loopback deployment above
for real data until those launch blockers land.

Dossier filenames include an immutable generation identifier. The Daily Run
record points to the active generation, so an interrupted forced regeneration
cannot overwrite content associated with an earlier Delivery Receipt.

Learnloom never guesses that an existing lock is stale. If the host or
container is killed mid-run, first confirm no Learnloom process is active, then
remove the named `.lock` file under `/data/data/locks` before retrying.

## 4. Install the systemd timer

The supplied files assume the repository lives at `/opt/learnloom`:

```sh
sudo cp deploy/learnloom.service /etc/systemd/system/
sudo cp deploy/learnloom.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now learnloom.timer
systemctl list-timers learnloom.timer
```

Run immediately or inspect logs:

```sh
sudo systemctl start learnloom.service
journalctl -u learnloom.service -n 200 --no-pager
```

Edit `OnCalendar` in `learnloom.timer` to change the host-local schedule, then
run `sudo systemctl daemon-reload && sudo systemctl restart learnloom.timer`.
The oneshot service intentionally has no host-level start timeout. Each model
request remains bounded by `provider.timeoutSeconds` and `provider.retries`,
while the service allows a valid multi-stage run to finish.

## Updating

```sh
cd /opt/learnloom
git pull --ff-only
docker compose build --pull
sudo systemctl start learnloom.service
```

The mounted data volume survives image rebuilds.
