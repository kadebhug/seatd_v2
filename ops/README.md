# Operations

Delivery automation, operational tooling, and runbooks belong here.

## Incident Response

Start with [Incident Triage](runbooks/incident-triage.md) when the symptom is
unclear. Use the subsystem runbooks when an alert or report points to a
specific failure mode:

- [API availability and latency](runbooks/api-availability-latency.md)
- [Realtime delivery](runbooks/realtime-delivery.md)
- [Outbox health](runbooks/outbox-health.md)
- [Postgres health](runbooks/postgres-health.md)
- [Display fleet](runbooks/display-fleet.md)
- [QR assist](runbooks/qr-assist.md)
- [Integrations](runbooks/integrations.md)

## Prometheus

The API, worker, and realtime Go services expose Prometheus metrics on their
existing service ports at `GET /metrics`. Load `prometheus/seatd-alerts.yml`
into Prometheus as a rule file and scrape each backend service.

Expected local scrape targets:

- API: `localhost:8080/metrics`
- Worker: `localhost:8081/metrics`
- Realtime: `localhost:8082/metrics`

## Grafana

Provision or import the dashboards in `grafana/dashboards/` with a Prometheus
datasource variable named `DS_PROMETHEUS`.

- `seatd-api-health.json`: API availability, 5xx ratio, request rate, route latency, and scrape health.
- `seatd-realtime-health.json`: realtime active connections, connection churn, message drops, and backpressure closes.
- `seatd-data-plane-health.json`: outbox backlog, outbox processing, DB pool saturation, and canceled acquires.
- `seatd-operations-health.json`: display heartbeat freshness, guest QR denials, integration webhooks, and reconciliation discrepancies.

## Tracing

The API, worker, and realtime Go services initialize OpenTelemetry tracing at
startup. Local/test environments default to console spans. Staging and
production use no-op tracing unless `OTEL_TRACES_EXPORTER=otlp` and an OTLP
endpoint are configured.

Common tracing variables:

- `OTEL_TRACES_EXPORTER=console|otlp|none`
- `OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4318`
- `SEATD_TRACING_SAMPLE_RATIO=0.10`
- `SEATD_TRACING_DB_STATEMENTS=false`
