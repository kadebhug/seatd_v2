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
