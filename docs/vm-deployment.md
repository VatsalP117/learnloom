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
3. Set valid `from` and `to` addresses in `config.json`.
4. Change the Resend delivery's `enabled` field to `true`.

Neither file is copied into the image or tracked by Git.

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
Run records, locks, and logs. Back up that volume with the VM.

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

## Updating

```sh
cd /opt/learnloom
git pull --ff-only
docker compose build --pull
sudo systemctl start learnloom.service
```

The mounted data volume survives image rebuilds.

