# AegisShield Monitoring & Alerting Setup

This directory contains the complete monitoring and alerting infrastructure for the AegisShield platform, providing comprehensive observability, alerting, and visualization capabilities.

## 📊 Overview

The monitoring stack includes:

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and dashboarding
- **Alertmanager** - Alert routing and notifications
- **Custom Rules** - AegisShield-specific alerting rules
- **Dashboards** - Pre-configured business and technical dashboards

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured and connected
- StorageClass `fast-ssd` available
- At least 4GB RAM and 2 CPU cores available

### Deployment

1. **Deploy the monitoring stack:**
   ```bash
   cd infrastructure/scripts
   chmod +x deploy-monitoring.sh
   ./deploy-monitoring.sh
   ```

2. **Access the services:**
   ```bash
   # Prometheus
   kubectl port-forward svc/prometheus -n monitoring 9090:9090
   # Access: http://localhost:9090
   
   # Grafana
   kubectl port-forward svc/grafana -n monitoring 3000:3000
   # Access: http://localhost:3000
   # Credentials: admin / AegisShield2025!
   
   # Alertmanager
   kubectl port-forward svc/alertmanager -n monitoring 9093:9093
   # Access: http://localhost:9093
   ```

## 📁 File Structure

```
monitoring/
├── prometheus-config.yaml          # Prometheus configuration with service discovery
├── prometheus-rules.yaml           # Alerting rules for AegisShield services
├── prometheus-deployment.yaml      # Prometheus Kubernetes deployment
├── grafana-dashboards.yaml        # Pre-configured Grafana dashboards
├── grafana-deployment.yaml        # Grafana Kubernetes deployment
├── alertmanager-config.yaml       # Alertmanager routing configuration
├── alertmanager-deployment.yaml   # Alertmanager Kubernetes deployment
└── README.md                       # This file
```

## 🔧 Configuration Details

### Prometheus Configuration

**Service Discovery:**
- Kubernetes services auto-discovery
- AegisShield microservices monitoring
- Database and infrastructure monitoring
- Custom metric scraping intervals

**Retention:** 30 days of metrics data
**Storage:** 50GB persistent volume

### Grafana Dashboards

**System Overview Dashboard:**
- Service health status
- Request rates and latencies
- Error rates and availability
- Resource utilization

**Performance Dashboard:**
- Database performance metrics
- API endpoint performance
- Transaction processing times
- Queue depths and processing rates

**Business Metrics Dashboard:**
- Investigation case metrics
- Alert generation rates
- Compliance status tracking
- User activity analytics

### Alerting Rules

**System Health Alerts:**
- Service down detection (5-minute threshold)
- High error rates (>5% for 10 minutes)
- Database connectivity issues
- Resource exhaustion warnings

**Performance Alerts:**
- High response times (>2s for 15 minutes)
- Queue depth warnings (>1000 items)
- Memory usage alerts (>80% for 10 minutes)
- CPU usage alerts (>90% for 15 minutes)

**Security Alerts:**
- Authentication failures spike
- Unauthorized access attempts
- Data access pattern anomalies
- Compliance violation detection

**Business Logic Alerts:**
- Investigation processing delays
- Alert rule failures
- Data ingestion interruptions
- Compliance check failures

### Alertmanager Routing

**Notification Channels:**
- **Email:** General alerts and summaries
- **Slack:** Real-time notifications for critical issues
- **PagerDuty:** Critical system failures requiring immediate response

**Routing Rules:**
- **Critical alerts:** Immediate PagerDuty notification
- **Business hours:** Slack notifications
- **After hours:** Email summaries only (except critical)
- **Security alerts:** Dedicated security team notifications

## 📈 Metrics Reference

### AegisShield Custom Metrics

**Investigation Metrics:**
```
aegis_investigations_total                    # Total investigations created
aegis_investigations_duration_seconds        # Investigation completion time
aegis_investigation_status                    # Current investigation status
aegis_investigation_queue_depth               # Pending investigations
```

**Alert Metrics:**
```
aegis_alerts_generated_total                 # Total alerts generated
aegis_alerts_resolved_total                  # Total alerts resolved
aegis_alert_resolution_time_seconds          # Time to resolve alerts
aegis_alert_false_positive_rate              # False positive rate
```

**Data Processing Metrics:**
```
aegis_data_ingestion_rate                    # Records ingested per second
aegis_data_processing_latency_seconds        # Data processing latency
aegis_entity_resolution_operations_total     # Entity resolution operations
aegis_graph_queries_duration_seconds         # Graph query performance
```

**Compliance Metrics:**
```
aegis_compliance_checks_total                # Compliance checks performed
aegis_compliance_violations_detected         # Violations detected
aegis_regulatory_reports_generated           # Reports generated
aegis_audit_trail_entries_total              # Audit log entries
```

### Infrastructure Metrics

**Service Health:**
```
up                                           # Service availability
http_requests_total                          # HTTP request count
http_request_duration_seconds                # Request latency
http_requests_errors_total                   # Error count
```

**Database Metrics:**
```
postgres_up                                  # PostgreSQL availability
postgres_active_connections                  # Active connections
postgres_query_duration_seconds              # Query performance
neo4j_up                                     # Neo4j availability
neo4j_query_execution_time                   # Cypher query performance
```

**System Resources:**
```
container_memory_usage_bytes                 # Memory usage
container_cpu_usage_seconds_total            # CPU usage
container_fs_usage_bytes                     # Disk usage
container_network_receive_bytes_total        # Network I/O
```

## 🚨 Alert Runbooks

### Critical Service Down

**Alert:** `AegisShieldServiceDown`
**Severity:** Critical
**Investigation Steps:**
1. Check service logs: `kubectl logs -n aegisshield deployment/<service>`
2. Verify service configuration
3. Check dependent services (database, vault)
4. Review resource limits and usage
5. Escalate to on-call engineer if unresolved in 15 minutes

### High Error Rate

**Alert:** `AegisShieldHighErrorRate`
**Severity:** Warning
**Investigation Steps:**
1. Identify error patterns in logs
2. Check recent deployments
3. Verify external dependencies
4. Review API gateway logs
5. Consider rollback if error rate continues to increase

### Database Performance Degradation

**Alert:** `DatabaseSlowQueries`
**Severity:** Warning
**Investigation Steps:**
1. Check active database connections
2. Review slow query logs
3. Monitor database resource usage
4. Check for lock contention
5. Consider query optimization or scaling

### Security Alert: Authentication Failures

**Alert:** `HighAuthenticationFailures`
**Severity:** Critical
**Investigation Steps:**
1. Review authentication logs immediately
2. Check for IP address patterns
3. Verify user account status
4. Check for brute force attacks
5. Consider IP blocking if malicious activity detected
6. Notify security team immediately

## 🔧 Maintenance

### Regular Tasks

**Daily:**
- Review dashboard metrics
- Check alert status and trends
- Verify backup completion

**Weekly:**
- Review alert rule effectiveness
- Update dashboards based on feedback
- Check monitoring system health

**Monthly:**
- Review retention policies
- Update alert thresholds based on trends
- Conduct alert runbook drills

### Backup Procedures

**Prometheus Data:**
```bash
# Create snapshot
kubectl exec -n monitoring deployment/prometheus -- promtool tsdb create-blocks-from snapshots

# Backup to external storage
kubectl exec -n monitoring deployment/prometheus -- tar czf /tmp/prometheus-backup.tar.gz /prometheus
```

**Grafana Configuration:**
```bash
# Export dashboards
kubectl exec -n monitoring deployment/grafana -- grafana-cli admin export-dashboard

# Backup Grafana database
kubectl exec -n monitoring deployment/grafana -- sqlite3 /var/lib/grafana/grafana.db ".backup /tmp/grafana-backup.db"
```

### Scaling Considerations

**Prometheus:**
- Monitor ingestion rate and adjust retention
- Consider federation for multi-cluster setups
- Scale storage based on metric volume

**Grafana:**
- Use external database for HA setup
- Configure load balancing for multiple instances
- Implement dashboard version control

**Alertmanager:**
- Configure clustering for high availability
- Implement external notification redundancy
- Monitor alert processing latency

## 🔍 Troubleshooting

### Common Issues

**Prometheus not scraping targets:**
1. Check service discovery configuration
2. Verify network policies
3. Confirm target endpoints are accessible
4. Review Prometheus logs for errors

**Grafana dashboards not loading:**
1. Verify Prometheus data source configuration
2. Check dashboard JSON syntax
3. Confirm metric names and labels
4. Review Grafana logs for errors

**Alerts not firing:**
1. Check alert rule syntax
2. Verify metric availability
3. Confirm Alertmanager configuration
4. Test notification channels

**Missing metrics:**
1. Verify service instrumentation
2. Check metric endpoint accessibility
3. Review service discovery labels
4. Confirm scrape configuration

### Performance Optimization

**Reduce metric cardinality:**
- Review label usage
- Implement recording rules for complex queries
- Use appropriate metric types

**Optimize query performance:**
- Use recording rules for dashboard queries
- Implement proper time ranges
- Consider metric aggregation

**Storage optimization:**
- Adjust retention policies
- Compress old data
- Monitor disk usage trends

## 📞 Support

For monitoring-related issues:

**Critical Issues (P0):**
- Page on-call engineer immediately
- Post in #aegisshield-alerts Slack channel
- Create incident in incident management system

**Non-Critical Issues (P1-P3):**
- Create ticket in monitoring project
- Post in #aegisshield-monitoring Slack channel
- Schedule review in weekly monitoring meeting

**Contact Information:**
- On-call Engineer: [PagerDuty escalation]
- Monitoring Team: monitoring@aegisshield.com
- Documentation: https://docs.aegisshield.com/monitoring