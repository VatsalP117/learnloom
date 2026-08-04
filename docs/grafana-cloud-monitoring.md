# Grafana Cloud monitoring

Learnloom sends application metrics, worker metrics, host metrics, container
metrics, and Docker logs to Grafana Cloud through one Grafana Alloy collector.
The collector runs as a separate Dokploy Compose service on the same VM. Public
availability is checked independently by Grafana Cloud Synthetic Monitoring.

```text
Learnloom web :9091 ─┐
Learnloom worker :9090 ─┬─ Grafana Alloy ── HTTPS ── Grafana Cloud
Linux host metrics ─────┤                     ├─ Prometheus metrics
Docker/cAdvisor metrics ┤                     └─ Loki logs
Learnloom Docker logs ──┘

Grafana public probe ── HTTPS ── app.learnloom.blog/readyz
```

The production app Compose project gives its private network the stable Docker
name `learnloom-backend`. `compose.monitoring.yaml`
attaches Alloy to that existing network and does not publish the Alloy UI or any
application metrics port.

## What is collected

Application metrics include:

- request count and errors;
- bounded request-duration histograms labeled by method, route template, and
  status class;
- queue depth and oldest-item age;
- Postgres pool use and acquire wait time;
- model tokens, retries, estimated spend, and learning outcomes;
- generation and delivery counters;
- active issues and deliveries;
- bounded worker-duration histograms for model stages, issue phases, total
  generation, and delivery outcomes.

Infrastructure collection includes a constrained node exporter, cAdvisor
metrics for the `learnloom` Compose project, Alloy self-monitoring, and Docker
logs for Learnloom containers. Logs are labeled only with stable values such as
host, environment, project, service, and container. Request IDs, account IDs,
paths, and messages remain in the log body and are not Loki labels.

## Repository assets

- `compose.monitoring.yaml`: separate Dokploy service for Alloy.
- `.env.monitoring.example`: required collector connection values.
- `infra/monitoring/config.alloy`: collection, filtering, and forwarding.
- `infra/monitoring/learnloom-dashboard.json`: versioned operations dashboard.
- `infra/monitoring/learnloom-rules.yaml`: recording and alert rules for
  Grafana Cloud Metrics.

## Manual setup: Grafana Cloud

These steps require the Grafana Cloud account owner and cannot be performed from
the repository alone.

1. Create or select a Grafana Cloud stack.
2. In **Connections**, open the Prometheus and Loki connection details. Record:
   - Prometheus remote-write URL and numeric username;
   - Loki push URL and numeric username.
3. Create a stack-scoped access policy named `learnloom-alloy` with only:
   - `metrics:write`;
   - `logs:write`.
4. Generate its token and store it in the password manager. Grafana displays
   the token only when it is created.
5. Do not give the long-lived Alloy token dashboard, user, read, or rule
   permissions.

Grafana Cloud access-policy tokens send telemetry. They do not authorize the
Grafana dashboard HTTP API; dashboard automation would require a separate
Grafana service-account token.

## Deploy Alloy in Dokploy

Deploy the main `compose.dokploy.yaml` service first. The `learnloom-backend`
network must exist before the separate monitoring service starts.

1. In the same Dokploy production environment, create another **Docker
   Compose** service from this repository.
2. Name it `learnloom-monitoring` and use `./compose.monitoring.yaml` as the
   Compose path.
3. Keep isolated deployments off so the service attaches to the VM's existing
   `learnloom-backend` network and retains `learnloom-alloy-data`.
4. Copy `.env.monitoring.example` into the service Environment page and replace
   every placeholder:

   ```dotenv
   GRAFANA_CLOUD_METRICS_URL=https://prometheus-prod-...grafana.net/api/prom/push
   GRAFANA_CLOUD_METRICS_USER=123456
   GRAFANA_CLOUD_LOGS_URL=https://logs-prod-...grafana.net/loki/api/v1/push
   GRAFANA_CLOUD_LOGS_USER=123456
   GRAFANA_CLOUD_TOKEN=glc_...
   MONITORING_HOST=learnloom-production-1
   ```

5. Preview the Compose definition. Confirm it publishes no host ports and joins
   only `learnloom-backend`.
6. Deploy and inspect the Alloy logs. Configuration errors, authentication
   failures, failed scrapes, or rejected remote writes must be resolved before
   proceeding.

The collector is deliberately privileged because embedded cAdvisor and host
exporters need the Docker socket, host cgroups, `/proc`, `/sys`, and filesystem
metadata. Access to the Docker socket is effectively root-equivalent even when
mounted read-only. Keep this service limited to the pinned official Alloy
image, do not expose its UI, and restrict who can edit the monitoring Compose
service. If that trust level is unacceptable, install Alloy as a locked-down
host service and use separate exporters instead.

## Verify ingestion

In **Grafana → Explore → Metrics**, run:

```promql
up{project="learnloom"}
```

Expected targets include:

- `job="learnloom", service="web"`;
- `job="learnloom", service="worker"`;
- `job="node"`;
- `job="docker"` series after containers are discovered;
- `job="alloy", service="alloy"`.

Then verify the new histograms:

```promql
learnloom_http_request_duration_seconds_count
learnloom_worker_operation_duration_seconds_count
```

Generate a few requests and allow at least one worker cycle before concluding
that an empty duration series is broken. Histograms do not appear until their
first observation.

In **Explore → Logs**, run:

```logql
{project="learnloom"}
```

Confirm logs exist for the expected Compose services and that no secret appears
in sampled entries.

## Install the dashboard

1. Open **Dashboards → New → Import**.
2. Upload `infra/monitoring/learnloom-dashboard.json`.
3. Select the Grafana Cloud Prometheus and Loki data sources for the two
   dashboard variables.
4. Open the dashboard and check that the web and worker target panels are `UP`.

The dashboard covers availability, request rate, 5xx ratio, route p95 latency,
worker phase latency/outcomes, queues, active work, Postgres saturation, model
spend, VM health, container resources, and logs.

## Upload recording and alert rules

New Grafana Cloud stacks store these as Grafana-managed rules. Use a separate,
short-lived Grafana service account; never add rule or dashboard permissions to
the Alloy ingestion token.

1. Create a service account with these fixed roles:
   - Alerting: Rules Reader, Rules Writer, Set provisioning status;
   - Data sources: Reader;
   - Folders: Creator, Reader, Writer.
2. Generate a short-lived token for that service account.
3. Use Grafana's Prometheus-rule conversion endpoint with `mimirtool`:

   ```sh
   export MIMIR_ADDRESS=https://your-stack.grafana.net/api/convert/
   export MIMIR_API_KEY=glsa_short_lived_service_account_token
   export MIMIR_TENANT_ID=1
   mimirtool \
     --extra-headers='X-Grafana-Alerting-Datasource-UID=grafanacloud-prom,X-Grafana-Alerting-Datasource-Name=grafanacloud-prom,X-Disable-Provenance=true' \
     rules load infra/monitoring/learnloom-rules.yaml
   ```

4. Confirm the `learnloom` folder contains all four rule groups in **Alerting
   → Alert rules**, then delete the temporary service account.

The rule file includes collector/target availability, worker liveness, 5xx
ratio, p95 latency, queue age, generation and delivery failures, claim
recovery, Postgres saturation, model budget, disk, and memory alerts.

## Configure notification delivery

1. Open **Alerting → Contact points** and create at least one actively monitored
   destination, such as Telegram, Slack, or email plus a push channel.
2. Use the contact point's **Test** action and confirm receipt.
3. In **Notification policies**, route `severity="critical"` immediately and
   `severity="warning"` with an appropriate grouping interval.
4. Confirm the Grafana-managed rules in the `learnloom` folder use that
   notification path.

Alert rules without a tested contact point do not constitute monitoring.

## Configure external checks

Alloy is on the application VM, so it cannot report a total VM or network
failure. Add an external Grafana Synthetic Monitoring HTTP check:

- target: `https://app.learnloom.blog/readyz`;
- method: `GET`;
- valid status: `200`;
- body expression: `"status"\s*:\s*"ready"`;
- frequency: `60s` from Mumbai and Frankfurt public probes;
- enable failed-check and TLS-expiration alerts.

Optionally add `https://learnloom.blog/` as a second availability check. Route
Synthetic Monitoring alerts by matching `namespace="synthetic_monitoring"` in
the notification policy.

## Safe rollout and rollback

Roll out in this order:

1. deploy application instrumentation;
2. deploy Alloy and verify ingestion;
3. import the dashboard;
4. upload rules;
5. configure and test notifications;
6. enable the synthetic check.

If Alloy causes unacceptable VM load, stop only the separate
`learnloom-monitoring` service. Learnloom continues serving because the
collector is not in its request or worker execution path. Preserve the Alloy
data volume when restarting so Docker log positions are retained.

Never remove the public synthetic check during a collector incident; it is the
independent signal that the service is reachable.
