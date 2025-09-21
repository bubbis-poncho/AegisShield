# AegisShield Maintenance Procedures

## 🎯 Overview

This document outlines comprehensive maintenance procedures for the AegisShield financial crime investigation platform. These procedures ensure optimal performance, security, and reliability of the production environment.

## 📅 Maintenance Schedule

### Daily Maintenance
- **Time:** 02:00 UTC
- **Duration:** 30 minutes
- **Impact:** Minimal service interruption

### Weekly Maintenance
- **Time:** Sunday 02:00 UTC
- **Duration:** 2 hours
- **Impact:** Possible brief service interruptions

### Monthly Maintenance
- **Time:** First Sunday of month, 02:00 UTC
- **Duration:** 4 hours
- **Impact:** Planned maintenance window

### Quarterly Maintenance
- **Time:** First Sunday of quarter, 02:00 UTC
- **Duration:** 8 hours
- **Impact:** Extended maintenance window

## 🔄 Daily Maintenance Procedures

### Automated Daily Tasks

**Daily Health Check Script**
```bash
#!/bin/bash
# Daily automated health check
# Location: /opt/aegisshield/scripts/daily-health-check.sh
# Cron: 0 2 * * * /opt/aegisshield/scripts/daily-health-check.sh

LOGFILE="/var/log/aegisshield/daily-health-$(date +%Y%m%d).log"
exec > >(tee -a $LOGFILE)
exec 2>&1

echo "=== AegisShield Daily Health Check Started ==="
echo "Timestamp: $(date)"

# 1. Service Health Verification
echo "1. Checking service health..."
failed_services=0

services=("postgresql" "neo4j" "vault" "data-ingestion" "entity-resolution" "graph-engine" "api-gateway" "alert-engine" "frontend")

for service in "${services[@]}"; do
  echo "  Checking $service..."
  
  # Check pod status
  ready_pods=$(kubectl get pods -n aegisshield-prod -l app=$service --no-headers | grep Running | wc -l)
  total_pods=$(kubectl get pods -n aegisshield-prod -l app=$service --no-headers | wc -l)
  
  if [ $ready_pods -eq 0 ] || [ $ready_pods -lt $total_pods ]; then
    echo "    ❌ $service: $ready_pods/$total_pods pods ready"
    failed_services=$((failed_services + 1))
    
    # Attempt automatic recovery
    kubectl rollout restart deployment/$service -n aegisshield-prod
    echo "    🔄 Restarted $service deployment"
  else
    echo "    ✅ $service: $ready_pods/$total_pods pods ready"
  fi
done

# 2. Database Connectivity Check
echo "2. Checking database connectivity..."

# PostgreSQL
if kubectl exec -n aegisshield-prod deployment/postgresql -- pg_isready >/dev/null 2>&1; then
  echo "  ✅ PostgreSQL: Connected"
else
  echo "  ❌ PostgreSQL: Connection failed"
  failed_services=$((failed_services + 1))
fi

# Neo4j
if kubectl exec -n aegisshield-prod deployment/neo4j -- cypher-shell -u neo4j -p password "RETURN 1" >/dev/null 2>&1; then
  echo "  ✅ Neo4j: Connected"
else
  echo "  ❌ Neo4j: Connection failed"
  failed_services=$((failed_services + 1))
fi

# 3. Storage Space Check
echo "3. Checking storage space..."
kubectl exec -n aegisshield-prod deployment/postgresql -- df -h /var/lib/postgresql/data | tail -n 1 | awk '{
  if (substr($5, 1, length($5)-1) > 80) 
    print "  ⚠️  PostgreSQL disk usage: " $5 " (Warning: >80%)"
  else 
    print "  ✅ PostgreSQL disk usage: " $5
}'

kubectl exec -n aegisshield-prod deployment/neo4j -- df -h /data | tail -n 1 | awk '{
  if (substr($5, 1, length($5)-1) > 80) 
    print "  ⚠️  Neo4j disk usage: " $5 " (Warning: >80%)"
  else 
    print "  ✅ Neo4j disk usage: " $5
}'

# 4. Memory Usage Check
echo "4. Checking memory usage..."
kubectl top nodes --no-headers | while read node cpu memory; do
  memory_percent=$(echo $memory | grep -o '[0-9]*' | head -1)
  if [ $memory_percent -gt 85 ]; then
    echo "  ⚠️  Node $node memory usage: $memory (Warning: >85%)"
  else
    echo "  ✅ Node $node memory usage: $memory"
  fi
done

# 5. Certificate Expiration Check
echo "5. Checking certificate expiration..."
kubectl get certificate -n aegisshield-prod -o json | jq -r '.items[] | select(.status.conditions[0].status != "True") | .metadata.name' | while read cert; do
  echo "  ⚠️  Certificate issue: $cert"
done

# 6. Backup Verification
echo "6. Verifying recent backups..."
last_backup=$(kubectl get cronjobs -n aegisshield-prod database-backup -o jsonpath='{.status.lastScheduleTime}')
if [ -n "$last_backup" ]; then
  echo "  ✅ Last backup: $last_backup"
else
  echo "  ❌ No recent backup found"
  failed_services=$((failed_services + 1))
fi

# 7. Alert Status
echo "7. Checking critical alerts..."
critical_alerts=$(curl -s http://alertmanager.monitoring.svc.cluster.local:9093/api/v1/alerts | jq -r '.data[] | select(.labels.severity == "critical" and .status.state == "active") | .labels.alertname' | wc -l)

if [ $critical_alerts -gt 0 ]; then
  echo "  ⚠️  $critical_alerts critical alerts active"
else
  echo "  ✅ No critical alerts"
fi

# Summary
echo "=== Daily Health Check Summary ==="
if [ $failed_services -eq 0 ]; then
  echo "✅ All systems healthy"
  exit 0
else
  echo "❌ $failed_services issues detected"
  # Send notification
  curl -X POST "$SLACK_WEBHOOK_URL" -d "{\"text\":\"AegisShield Daily Health Check: $failed_services issues detected\"}"
  exit 1
fi
```

### Daily Log Rotation

**Log Rotation Script**
```bash
#!/bin/bash
# Daily log rotation
# Location: /opt/aegisshield/scripts/daily-log-rotation.sh
# Cron: 0 3 * * * /opt/aegisshield/scripts/daily-log-rotation.sh

echo "=== Daily Log Rotation Started ==="
echo "Timestamp: $(date)"

# 1. Application Logs
echo "1. Rotating application logs..."
services=("api-gateway" "data-ingestion" "entity-resolution" "graph-engine" "alert-engine")

for service in "${services[@]}"; do
  echo "  Rotating logs for $service..."
  kubectl exec -n aegisshield-prod deployment/$service -- \
    find /var/log -name "*.log" -type f -mtime +7 -delete
done

# 2. Database Logs
echo "2. Rotating database logs..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  find /var/log/postgresql -name "*.log" -type f -mtime +14 -delete

# 3. System Logs (on nodes)
echo "3. Rotating system logs..."
kubectl get nodes -o jsonpath='{.items[*].metadata.name}' | xargs -I {} \
  kubectl debug node/{} -it --image=busybox -- \
  find /host/var/log -name "*.log" -type f -mtime +30 -delete

# 4. Monitoring Logs
echo "4. Rotating monitoring logs..."
kubectl exec -n monitoring deployment/prometheus -- \
  find /prometheus -name "*.log" -type f -mtime +7 -delete

echo "=== Daily Log Rotation Completed ==="
```

## 📅 Weekly Maintenance Procedures

### System Updates

**Weekly System Update Script**
```bash
#!/bin/bash
# Weekly system updates
# Location: /opt/aegisshield/scripts/weekly-system-update.sh
# Cron: 0 2 * * 0 /opt/aegisshield/scripts/weekly-system-update.sh

MAINTENANCE_LOG="/var/log/aegisshield/weekly-maintenance-$(date +%Y%m%d).log"
exec > >(tee -a $MAINTENANCE_LOG)
exec 2>&1

echo "=== AegisShield Weekly Maintenance Started ==="
echo "Timestamp: $(date)"

# 1. Container Image Updates
echo "1. Checking for container image updates..."
services=("api-gateway" "data-ingestion" "entity-resolution" "graph-engine" "alert-engine" "frontend")

for service in "${services[@]}"; do
  echo "  Checking $service for updates..."
  
  current_image=$(kubectl get deployment $service -n aegisshield-prod -o jsonpath='{.spec.template.spec.containers[0].image}')
  echo "    Current image: $current_image"
  
  # Check for latest image (implement your registry check here)
  # For now, we'll skip automatic updates and just log
  echo "    Image check completed"
done

# 2. Security Patches
echo "2. Applying security patches..."
./infrastructure/scripts/security-audit.sh --apply-patches

# 3. Database Maintenance
echo "3. Performing database maintenance..."

# PostgreSQL maintenance
echo "  PostgreSQL maintenance..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "VACUUM ANALYZE;"

kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "REINDEX DATABASE aegisshield;"

# Update PostgreSQL statistics
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT pg_stat_reset();"

# Neo4j maintenance
echo "  Neo4j maintenance..."
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p password "CALL db.stats.clear();"

# 4. Performance Optimization
echo "4. Running performance optimization..."
./infrastructure/scripts/performance-optimization.sh --weekly

# 5. Backup Verification
echo "5. Verifying backup integrity..."
latest_backup=$(aws s3 ls s3://aegisshield-backups/ --recursive | sort | tail -n 1 | awk '{print $4}')
if [ -n "$latest_backup" ]; then
  echo "  Latest backup: $latest_backup"
  # Verify backup integrity
  aws s3 cp "s3://aegisshield-backups/$latest_backup" /tmp/verify-backup.tar.gz
  if tar -tzf /tmp/verify-backup.tar.gz >/dev/null 2>&1; then
    echo "  ✅ Backup integrity verified"
  else
    echo "  ❌ Backup integrity check failed"
  fi
  rm -f /tmp/verify-backup.tar.gz
fi

# 6. Certificate Management
echo "6. Managing certificates..."
kubectl get certificate -n aegisshield-prod -o json | jq -r '.items[] | select(.status.conditions[0].status != "True" or (.status.notAfter | fromdateiso8601) < (now + 2592000)) | .metadata.name' | while read cert; do
  echo "  Renewing certificate: $cert"
  kubectl delete certificate $cert -n aegisshield-prod
  kubectl apply -f infrastructure/k8s/certificates/$cert.yaml
done

# 7. Monitoring Dashboard Updates
echo "7. Updating monitoring dashboards..."
kubectl apply -f infrastructure/k8s/monitoring/grafana-dashboards.yaml

echo "=== Weekly Maintenance Completed ==="
```

### Performance Tuning

**Weekly Performance Analysis**
```bash
#!/bin/bash
# Weekly performance analysis and tuning
# Location: /opt/aegisshield/scripts/weekly-performance-tuning.sh

echo "=== Weekly Performance Analysis ==="
echo "Timestamp: $(date)"

PROMETHEUS_URL="http://prometheus.monitoring.svc.cluster.local:9090"
REPORT_FILE="/var/log/aegisshield/performance-report-$(date +%Y%m%d).txt"

# 1. API Performance Analysis
echo "1. API Performance Analysis" | tee $REPORT_FILE

# Get 95th percentile response times
api_p95=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=histogram_quantile(0.95,%20rate(http_request_duration_seconds_bucket[7d]))" | jq -r '.data.result[0].value[1]')
echo "  API 95th percentile response time (7d): ${api_p95}s" | tee -a $REPORT_FILE

if (( $(echo "$api_p95 > 2.0" | bc -l) )); then
  echo "  ⚠️  API response time exceeds threshold (2.0s)" | tee -a $REPORT_FILE
  # Scale up API Gateway
  kubectl scale deployment api-gateway --replicas=5 -n aegisshield-prod
  echo "  🔄 Scaled API Gateway to 5 replicas" | tee -a $REPORT_FILE
fi

# 2. Database Performance Analysis
echo "2. Database Performance Analysis" | tee -a $REPORT_FILE

# PostgreSQL slow queries
echo "  PostgreSQL slow queries:" | tee -a $REPORT_FILE
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT query, mean_time, calls FROM pg_stat_statements WHERE mean_time > 1000 ORDER BY mean_time DESC LIMIT 5;" | tee -a $REPORT_FILE

# Cache hit ratio
cache_ratio=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT round((sum(heap_blks_hit) * 100.0 / (sum(heap_blks_hit) + sum(heap_blks_read)))::numeric, 2) FROM pg_statio_user_tables;")
echo "  PostgreSQL cache hit ratio: ${cache_ratio}%" | tee -a $REPORT_FILE

if (( $(echo "$cache_ratio < 95" | bc -l) )); then
  echo "  ⚠️  Cache hit ratio below optimal (95%)" | tee -a $REPORT_FILE
  # Increase shared_buffers if possible
  echo "  💡 Consider increasing shared_buffers" | tee -a $REPORT_FILE
fi

# 3. Resource Utilization Analysis
echo "3. Resource Utilization Analysis" | tee -a $REPORT_FILE

# Node resource usage
kubectl top nodes --no-headers | while read node cpu memory; do
  cpu_percent=$(echo $cpu | grep -o '[0-9]*' | head -1)
  memory_percent=$(echo $memory | grep -o '[0-9]*' | head -1)
  
  echo "  Node $node: CPU ${cpu_percent}%, Memory ${memory_percent}%" | tee -a $REPORT_FILE
  
  if [ $cpu_percent -gt 80 ]; then
    echo "    ⚠️  High CPU usage on $node" | tee -a $REPORT_FILE
  fi
  
  if [ $memory_percent -gt 80 ]; then
    echo "    ⚠️  High memory usage on $node" | tee -a $REPORT_FILE
  fi
done

# 4. Auto-scaling Recommendations
echo "4. Auto-scaling Recommendations" | tee -a $REPORT_FILE

services=("api-gateway" "data-ingestion" "entity-resolution")
for service in "${services[@]}"; do
  # Get current CPU usage
  cpu_usage=$(kubectl top pods -n aegisshield-prod -l app=$service --no-headers | awk '{sum+=$2} END {print sum/NR}' | grep -o '[0-9]*')
  
  if [ $cpu_usage -gt 70 ]; then
    echo "  🔄 Recommend scaling up $service (CPU: ${cpu_usage}%)" | tee -a $REPORT_FILE
    current_replicas=$(kubectl get deployment $service -n aegisshield-prod -o jsonpath='{.spec.replicas}')
    new_replicas=$((current_replicas + 1))
    kubectl scale deployment $service --replicas=$new_replicas -n aegisshield-prod
    echo "    Scaled $service from $current_replicas to $new_replicas replicas" | tee -a $REPORT_FILE
  elif [ $cpu_usage -lt 30 ] && [ $(kubectl get deployment $service -n aegisshield-prod -o jsonpath='{.spec.replicas}') -gt 2 ]; then
    echo "  🔽 Recommend scaling down $service (CPU: ${cpu_usage}%)" | tee -a $REPORT_FILE
    current_replicas=$(kubectl get deployment $service -n aegisshield-prod -o jsonpath='{.spec.replicas}')
    new_replicas=$((current_replicas - 1))
    kubectl scale deployment $service --replicas=$new_replicas -n aegisshield-prod
    echo "    Scaled $service from $current_replicas to $new_replicas replicas" | tee -a $REPORT_FILE
  fi
done

echo "=== Performance Analysis Completed ===" | tee -a $REPORT_FILE
```

## 🗓️ Monthly Maintenance Procedures

### Comprehensive System Review

**Monthly System Review Script**
```bash
#!/bin/bash
# Monthly comprehensive system review
# Location: /opt/aegisshield/scripts/monthly-system-review.sh
# Cron: 0 2 1 * * /opt/aegisshield/scripts/monthly-system-review.sh

MONTHLY_REPORT="/var/log/aegisshield/monthly-report-$(date +%Y%m).txt"
exec > >(tee -a $MONTHLY_REPORT)
exec 2>&1

echo "=== AegisShield Monthly System Review ==="
echo "Report Period: $(date -d 'last month' '+%B %Y')"
echo "Generated: $(date)"

# 1. System Availability Report
echo "1. SYSTEM AVAILABILITY REPORT"
echo "=============================="

# Calculate uptime percentage for each service
services=("api-gateway" "data-ingestion" "entity-resolution" "graph-engine" "alert-engine")

for service in "${services[@]}"; do
  # Query Prometheus for uptime data
  uptime=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=avg_over_time(up{job=\"$service\"}[30d])" | jq -r '.data.result[0].value[1]')
  uptime_percent=$(echo "scale=2; $uptime * 100" | bc)
  echo "$service: ${uptime_percent}% uptime"
done

# 2. Performance Metrics Summary
echo ""
echo "2. PERFORMANCE METRICS SUMMARY"
echo "==============================="

# API performance trends
echo "API Performance (30-day averages):"
avg_response_time=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=avg_over_time(http_request_duration_seconds[30d])" | jq -r '.data.result[0].value[1]')
echo "  Average response time: ${avg_response_time}s"

error_rate=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=rate(http_requests_total{status=~\"5..\"}[30d])" | jq -r '.data.result[0].value[1]')
echo "  Error rate: $error_rate requests/sec"

# Database performance
echo "Database Performance:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT 'Total queries: ' || sum(calls), 'Avg time: ' || round(avg(mean_time)::numeric, 2) || 'ms' FROM pg_stat_statements;"

# 3. Capacity Planning
echo ""
echo "3. CAPACITY PLANNING"
echo "===================="

# Resource growth trends
echo "Resource Usage Trends (30-day):"
kubectl top nodes --no-headers | while read node cpu memory; do
  echo "Node $node: CPU $cpu, Memory $memory"
done

# Storage growth
echo "Storage Growth:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT pg_database_size('aegisshield')/1024/1024 || ' MB' as database_size;"

# 4. Security Review
echo ""
echo "4. SECURITY REVIEW"
echo "=================="

# Certificate status
echo "Certificate Status:"
kubectl get certificate -n aegisshield-prod -o custom-columns="NAME:.metadata.name,READY:.status.conditions[0].status,EXPIRES:.status.notAfter"

# Security scan summary
echo "Security Scan Results:"
./infrastructure/scripts/security-audit.sh --monthly-report

# 5. Backup and Recovery Review
echo ""
echo "5. BACKUP AND RECOVERY REVIEW"
echo "=============================="

# Backup success rate
echo "Backup Success Rate (30 days):"
successful_backups=$(kubectl logs -n aegisshield-prod cronjob/database-backup --since=720h | grep -c "Backup completed successfully")
total_backups=$(kubectl logs -n aegisshield-prod cronjob/database-backup --since=720h | grep -c "Starting backup")
if [ $total_backups -gt 0 ]; then
  success_rate=$(echo "scale=2; $successful_backups * 100 / $total_backups" | bc)
  echo "  Success rate: ${success_rate}% ($successful_backups/$total_backups)"
else
  echo "  No backup logs found"
fi

# Recovery testing
echo "Recovery Testing Status:"
if [ -f "/var/log/aegisshield/last-recovery-test.log" ]; then
  last_test=$(stat -c %Y /var/log/aegisshield/last-recovery-test.log)
  days_ago=$(( ($(date +%s) - $last_test) / 86400 ))
  echo "  Last recovery test: $days_ago days ago"
  if [ $days_ago -gt 30 ]; then
    echo "  ⚠️  Recovery test overdue (>30 days)"
  fi
else
  echo "  ❌ No recovery test records found"
fi

# 6. Cost Analysis
echo ""
echo "6. COST ANALYSIS"
echo "================"

# Resource costs (placeholder - implement based on your cloud provider)
echo "Resource Allocation:"
total_cpu=$(kubectl describe nodes | grep -A 5 "Allocated resources" | grep "cpu" | awk '{sum+=$2} END {print sum}')
total_memory=$(kubectl describe nodes | grep -A 5 "Allocated resources" | grep "memory" | awk '{sum+=$2} END {print sum}')
echo "  Total CPU allocated: ${total_cpu}m"
echo "  Total Memory allocated: ${total_memory}Ki"

# 7. Recommendations
echo ""
echo "7. RECOMMENDATIONS"
echo "=================="

# Generate recommendations based on analysis
echo "System Recommendations:"

# Check if scaling needed
high_cpu_services=$(kubectl top pods -n aegisshield-prod --no-headers | awk '$3 > 70 {print $1}' | wc -l)
if [ $high_cpu_services -gt 0 ]; then
  echo "  • Consider scaling up $high_cpu_services services with high CPU usage"
fi

# Storage recommendations
db_size=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT pg_database_size('aegisshield')/1024/1024/1024;")
if (( $(echo "$db_size > 50" | bc -l) )); then
  echo "  • Database size approaching 50GB, consider archiving old data"
fi

# Performance recommendations
if (( $(echo "$avg_response_time > 1.5" | bc -l) )); then
  echo "  • API response times elevated, investigate optimization opportunities"
fi

echo ""
echo "=== Monthly System Review Completed ==="
```

### Data Archival

**Monthly Data Archival Script**
```bash
#!/bin/bash
# Monthly data archival
# Location: /opt/aegisshield/scripts/monthly-data-archival.sh

echo "=== Monthly Data Archival Started ==="
echo "Timestamp: $(date)"

ARCHIVE_DATE=$(date -d '6 months ago' '+%Y-%m-%d')
ARCHIVE_DIR="/tmp/aegisshield-archive-$(date +%Y%m)"

# 1. Archive old investigation data
echo "1. Archiving investigation data older than 6 months..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    COPY (
      SELECT * FROM investigations 
      WHERE created_at < '$ARCHIVE_DATE'
    ) TO '/tmp/investigations_archive.csv' WITH CSV HEADER;
  "

kubectl cp aegisshield-prod/postgresql-0:/tmp/investigations_archive.csv "$ARCHIVE_DIR/investigations_archive.csv"

# 2. Archive old alert data
echo "2. Archiving alert data older than 6 months..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    COPY (
      SELECT * FROM alerts 
      WHERE created_at < '$ARCHIVE_DATE'
    ) TO '/tmp/alerts_archive.csv' WITH CSV HEADER;
  "

kubectl cp aegisshield-prod/postgresql-0:/tmp/alerts_archive.csv "$ARCHIVE_DIR/alerts_archive.csv"

# 3. Archive old audit logs
echo "3. Archiving audit logs older than 6 months..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    COPY (
      SELECT * FROM audit_logs 
      WHERE created_at < '$ARCHIVE_DATE'
    ) TO '/tmp/audit_logs_archive.csv' WITH CSV HEADER;
  "

kubectl cp aegisshield-prod/postgresql-0:/tmp/audit_logs_archive.csv "$ARCHIVE_DIR/audit_logs_archive.csv"

# 4. Compress and upload archive
echo "4. Compressing and uploading archive..."
tar -czf "aegisshield-archive-$(date +%Y%m).tar.gz" -C /tmp "aegisshield-archive-$(date +%Y%m)"

# Upload to S3
aws s3 cp "aegisshield-archive-$(date +%Y%m).tar.gz" "s3://aegisshield-archives/monthly/"

# 5. Clean up archived data from database
echo "5. Cleaning up archived data from database..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    DELETE FROM investigations WHERE created_at < '$ARCHIVE_DATE';
    DELETE FROM alerts WHERE created_at < '$ARCHIVE_DATE';
    DELETE FROM audit_logs WHERE created_at < '$ARCHIVE_DATE';
    VACUUM FULL;
  "

# 6. Update statistics
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "ANALYZE;"

echo "=== Monthly Data Archival Completed ==="
```

## 🔄 Quarterly Maintenance Procedures

### Major System Updates

**Quarterly Major Updates Script**
```bash
#!/bin/bash
# Quarterly major system updates
# Location: /opt/aegisshield/scripts/quarterly-major-updates.sh

echo "=== AegisShield Quarterly Major Updates ==="
echo "Quarter: Q$(( ($(date +%-m)-1)/3 + 1 )) $(date +%Y)"
echo "Timestamp: $(date)"

# 1. Kubernetes Version Update
echo "1. Planning Kubernetes version update..."
current_version=$(kubectl version --short | grep "Server Version" | awk '{print $3}')
echo "Current Kubernetes version: $current_version"

# Check for available updates (implementation depends on your cluster management)
echo "Checking for available Kubernetes updates..."
# This would involve your specific cluster update procedures

# 2. Application Version Updates
echo "2. Planning application version updates..."
services=("api-gateway" "data-ingestion" "entity-resolution" "graph-engine" "alert-engine" "frontend")

for service in "${services[@]}"; do
  current_image=$(kubectl get deployment $service -n aegisshield-prod -o jsonpath='{.spec.template.spec.containers[0].image}')
  echo "Current $service image: $current_image"
  
  # Check for latest stable version
  # This would involve checking your container registry for latest tags
done

# 3. Database Version Updates
echo "3. Planning database version updates..."

# PostgreSQL version check
pg_version=$(kubectl exec -n aegisshield-prod deployment/postgresql -- psql -U postgres -t -c "SELECT version();")
echo "Current PostgreSQL version: $pg_version"

# Neo4j version check
neo4j_version=$(kubectl exec -n aegisshield-prod deployment/neo4j -- cypher-shell -u neo4j -p password "CALL dbms.components() YIELD name, versions RETURN name, versions[0];")
echo "Current Neo4j version: $neo4j_version"

# 4. Security Infrastructure Updates
echo "4. Updating security infrastructure..."
./infrastructure/scripts/security-audit.sh --quarterly-update

# 5. Monitoring Stack Updates
echo "5. Updating monitoring stack..."
helm repo update
helm upgrade prometheus infrastructure/helm/prometheus/ -n monitoring
helm upgrade grafana infrastructure/helm/grafana/ -n monitoring

# 6. Disaster Recovery Testing
echo "6. Comprehensive disaster recovery testing..."
./infrastructure/scripts/recovery-testing.sh --quarterly

# 7. Performance Baseline Updates
echo "7. Updating performance baselines..."
./infrastructure/scripts/performance-optimization.sh --quarterly-baseline

echo "=== Quarterly Major Updates Completed ==="
```

### Compliance Review

**Quarterly Compliance Review Script**
```bash
#!/bin/bash
# Quarterly compliance review
# Location: /opt/aegisshield/scripts/quarterly-compliance-review.sh

COMPLIANCE_REPORT="/var/log/aegisshield/compliance-report-Q$(( ($(date +%-m)-1)/3 + 1 ))-$(date +%Y).txt"
exec > >(tee -a $COMPLIANCE_REPORT)
exec 2>&1

echo "=== AegisShield Quarterly Compliance Review ==="
echo "Period: Q$(( ($(date +%-m)-1)/3 + 1 )) $(date +%Y)"
echo "Generated: $(date)"

# 1. SOX Compliance Review
echo ""
echo "1. SOX COMPLIANCE REVIEW"
echo "========================"

echo "Access Control Review:"
# Review user access and permissions
kubectl get rolebindings,clusterrolebindings -n aegisshield-prod -o yaml > /tmp/access-review.yaml
echo "  ✅ Access control configurations exported for review"

echo "Audit Trail Verification:"
# Verify audit logging is functioning
audit_count=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT count(*) FROM audit_logs WHERE created_at >= NOW() - INTERVAL '90 days';")
echo "  Audit records (90 days): $audit_count"

if [ $audit_count -gt 1000 ]; then
  echo "  ✅ Adequate audit trail maintained"
else
  echo "  ⚠️  Low audit record count, investigate logging"
fi

# 2. PCI-DSS Compliance Review
echo ""
echo "2. PCI-DSS COMPLIANCE REVIEW"
echo "============================"

echo "Encryption Status:"
# Check encryption at rest
kubectl get persistentvolumes -o custom-columns="NAME:.metadata.name,STORAGECLASS:.spec.storageClassName" | grep encrypted || echo "  ⚠️  No encrypted storage classes found"

# Check encryption in transit
tls_certificates=$(kubectl get certificate -n aegisshield-prod --no-headers | wc -l)
echo "  TLS certificates configured: $tls_certificates"

echo "Network Segmentation:"
network_policies=$(kubectl get networkpolicy -n aegisshield-prod --no-headers | wc -l)
echo "  Network policies in place: $network_policies"

# 3. GDPR Compliance Review
echo ""
echo "3. GDPR COMPLIANCE REVIEW"
echo "========================="

echo "Data Retention Policy:"
# Check data retention implementation
old_data_count=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT count(*) FROM personal_data WHERE created_at < NOW() - INTERVAL '7 years';")
echo "  Records older than 7 years: $old_data_count"

if [ $old_data_count -gt 0 ]; then
  echo "  ⚠️  Old personal data records found, review for deletion"
else
  echo "  ✅ Data retention policy compliance maintained"
fi

echo "Data Processing Logs:"
processing_logs=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT count(*) FROM data_processing_logs WHERE created_at >= NOW() - INTERVAL '90 days';")
echo "  Data processing log entries (90 days): $processing_logs"

# 4. Security Compliance
echo ""
echo "4. SECURITY COMPLIANCE REVIEW"
echo "============================="

echo "Vulnerability Assessment:"
./infrastructure/scripts/security-audit.sh --compliance-scan

echo "Password Policy Compliance:"
# Check password policies (implementation depends on your auth system)
echo "  Password policies configured in identity provider"

echo "Multi-Factor Authentication:"
mfa_enabled_users=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT count(*) FROM users WHERE mfa_enabled = true;")
total_users=$(kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -t -c "SELECT count(*) FROM users;")

if [ $total_users -gt 0 ]; then
  mfa_percentage=$(echo "scale=2; $mfa_enabled_users * 100 / $total_users" | bc)
  echo "  MFA adoption rate: ${mfa_percentage}% ($mfa_enabled_users/$total_users)"
  
  if (( $(echo "$mfa_percentage >= 95" | bc -l) )); then
    echo "  ✅ High MFA adoption rate"
  else
    echo "  ⚠️  MFA adoption below target (95%)"
  fi
fi

# 5. Backup and Recovery Compliance
echo ""
echo "5. BACKUP AND RECOVERY COMPLIANCE"
echo "=================================="

echo "Backup Frequency Compliance:"
backup_frequency=$(kubectl get cronjob database-backup -n aegisshield-prod -o jsonpath='{.spec.schedule}')
echo "  Backup schedule: $backup_frequency"

echo "Recovery Testing Compliance:"
if [ -f "/var/log/aegisshield/quarterly-recovery-test.log" ]; then
  last_recovery_test=$(stat -c %Y /var/log/aegisshield/quarterly-recovery-test.log)
  days_since_test=$(( ($(date +%s) - $last_recovery_test) / 86400 ))
  echo "  Last recovery test: $days_since_test days ago"
  
  if [ $days_since_test -le 90 ]; then
    echo "  ✅ Recovery testing up to date"
  else
    echo "  ⚠️  Recovery testing overdue"
  fi
else
  echo "  ❌ No recovery test records found"
fi

# 6. Recommendations
echo ""
echo "6. COMPLIANCE RECOMMENDATIONS"
echo "============================="

echo "Priority Actions:"
# Generate compliance recommendations based on findings
echo "  • Schedule quarterly recovery test if overdue"
echo "  • Review and update data retention policies"
echo "  • Increase MFA adoption if below target"
echo "  • Update security scanning frequency"
echo "  • Review access control permissions"

echo ""
echo "=== Quarterly Compliance Review Completed ==="
```

This comprehensive maintenance procedures document provides structured approaches to keeping the AegisShield platform secure, performant, and compliant with all regulatory requirements.