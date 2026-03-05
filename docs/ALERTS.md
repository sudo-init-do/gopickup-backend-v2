# Operational Alerts

This guide defines the critical alerts for the GoPickup production backend.

## 1. Alert Channels

- **PagerDuty/OpsGenie**: For critical (P0) incidents.
- **Slack/Email**: For warnings (P1/P2) and non-urgent notifications.

## 2. Recommended Thresholds

### A. Critical (P0) - Immediate Action Required

| Alert Name | Condition | Severity | Description |
|---|---|---|---|
| **High 5xx Rate** | 5xx Errors > 5% for 5 mins | P0 | Backend is returning server errors frequently. |
| **Database Down** | `GET /api/v1/ready` returns 503 | P0 | Database connection lost or overloaded. |
| **High Latency** | p95 Latency > 2s for 5 mins | P0 | App is extremely slow, affecting user experience. |
| **Disk Space** | Disk Usage > 90% | P0 | Server or DB running out of disk space. |

### B. Warning (P1) - Investigate

| Alert Name | Condition | Severity | Description |
|---|---|---|---|
| **High 4xx Rate** | 4xx Errors > 10% for 10 mins | P1 | Possible bad client release or attack. |
| **High Memory** | Memory Usage > 80% | P1 | Potential memory leak or need for scaling. |
| **CPU Load** | CPU Load > 80% for 15 mins | P1 | High processing load. |
| **WS Disconnects** | Drop > 20% in 5 mins | P1 | Websocket server restart or network issue. |

## 3. WebSocket Specific Alerts

- **Connection Spikes**: > 50% increase in 5 mins (possible DDoS).
- **Zero Connections**: If connections drop to 0 unexpectedly during peak hours.

## 4. Database Specific Alerts

- **Max Connections**: Active connections > 80% of `max_connections`.
- **Replication Lag**: > 1 minute (if using read replicas).

## 5. Deployment Alerts

- **New Release**: Alert on Slack when a new version is deployed.
- **Rollback**: Alert if deployment fails or rollback occurs.
