# Prometheus
`Prometheus` implements the prometheus exporter for gmqtt with an optional web UI dashboard.

**Metrics Endpoint**: http://127.0.0.1:8082/metrics
**Web Dashboard**: http://127.0.0.1:8082/ or http://127.0.0.1:8082/dashboard

# Configuration

```yaml
plugins:
  prometheus:
    path: "/metrics"              # URL path for Prometheus metrics endpoint
    listen_address: ":8082"       # Address to listen on
    enable_dashboard: true        # Enable/disable web UI dashboard (default: true)
```

# Web Dashboard

The plugin includes a built-in web dashboard that displays broker metrics in real-time:
- Auto-refreshes every 5 seconds
- Shows connections, messages, packets, and subscriptions
- Responsive design for desktop and mobile
- No external dependencies required

To disable the dashboard while keeping the metrics endpoint:
```yaml
plugins:
  prometheus:
    enable_dashboard: false
```

# Metrics

metric name | Type | Labels 
---|---|---
gmqtt_clients_connected_total | Counter | 
gmqtt_messages_dropped_total | Counter | qos:  qos of the dropped message
gmqtt_packets_received_bytes_total | Counter | type: type of the packet
gmqtt_packets_received_total | Counter |  type: type of the packet
gmqtt_packets_sent_bytes_total | Counter | type: type of the packet
gmqtt_packets_sent_total | Counter | type: type of the packet
gmqtt_sessions_created_total | Counter | 
gmqtt_sessions_terminated_total | Counter | reason: the reason of termination. (expired|taken_over|normal)
gmqtt_sessions_active_current | Gauge | 
gmqtt_sessions_expired_total | Counter |
gmqtt_sessions_inactive_current | Gauge |
gmqtt_subscriptions_current | Gauge |
gmqtt_subscriptions_total | Counter |
gmqtt_messages_queued_current | Gauge |
gmqtt_messages_received_total | Counter | qos: qos of the message
gmqtt_messages_sent_total | Counter | qos: qos of the message