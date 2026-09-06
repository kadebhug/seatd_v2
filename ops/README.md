# Operations

Delivery automation, operational tooling, and runbooks belong here.

## Prometheus

The API, worker, and realtime Go services expose Prometheus metrics on their
existing service ports at `GET /metrics`. Load `prometheus/seatd-alerts.yml`
into Prometheus as a rule file and scrape each backend service.

Expected local scrape targets:

- API: `localhost:8080/metrics`
- Worker: `localhost:8081/metrics`
- Realtime: `localhost:8082/metrics`

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
