# Monitoring & Observability

This guide describes how to monitor the GoPickup backend in production.

## 1. Logging Strategy

Logs are output to `stdout` in JSON format when `GIN_MODE=release`.

**Log Fields:**
- `level`: (INFO, WARN, ERROR)
- `msg`: The log message
- `ts`: Timestamp (ISO8601)
- `request_id`: Unique trace ID for correlation
- `latency`: Request duration (ms)
- `status`: HTTP status code (200, 400, 500)
- `path`: Request path

**Where logs go:**
- **Docker**: `docker logs -f gopickup-app`
- **Aggregation**: Use a log driver (fluentd, logstash) or a cloud logging service (AWS CloudWatch, GCP Logging, Datadog) to collect these logs.

## 2. Key Metrics to Monitor

Set up dashboards for these critical signals:

### A. API Health
- **Error Rate (5xx)**: Should be < 1%. Spike indicates system failure.
- **Latency (p95)**: Should be < 500ms for most endpoints. Monitor slow queries.
- **Throughput**: Requests per second (RPS). Sudden drop = outage; spike = DDoS or viral traffic.

### B. Database Health
- **Connection Count**: Ensure it stays below `max_connections` (default 100 usually).
- **Ready Check**: Monitor `/api/v1/ready`. If it returns 503, DB is down or unreachable.

### C. Realtime (WebSocket)
- **Active Connections**: Number of open WS connections.
- **Disconnect Rate**: Sudden drop in connections indicates a restart or network issue.

## 3. Metrics Endpoint (Prometheus)

We provide an optional metrics endpoint at `/api/v1/metrics`.

**Enable**: Set `ENABLE_METRICS=true` in `.env`.

**Output**: JSON format with basic runtime stats (uptime, memory usage, goroutines).

**Integration**:
To integrate with Prometheus/Grafana:
1.  Add `prometheus-client` middleware (future enhancement).
2.  Scrape `/api/v1/metrics` (requires custom adapter if JSON, or use standard exporter).

## 4. Error Tracking (Sentry)

We recommend integrating Sentry for real-time error tracking.

1.  **Sign up** for Sentry.
2.  **Add DSN** to `.env`: `SENTRY_DSN=https://...`
3.  **Update Code**: Initialize Sentry SDK in `main.go`. (Not currently implemented in base code, but recommended for production).

## 5. Health Checks

Configure your load balancer or uptime monitor (Pingdom, UptimeRobot) to check:

- **Liveness**: `GET /api/v1/health` (Checks if server process is running)
- **Readiness**: `GET /api/v1/ready` (Checks if DB is connected)
