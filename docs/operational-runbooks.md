# AegisShield Operational Runbooks

## 🎯 Overview

This document contains detailed operational procedures for the day-to-day management of the AegisShield platform. These runbooks provide step-by-step instructions for common operational tasks, troubleshooting procedures, and emergency response protocols.

## 📚 Table of Contents

1. [Service Management](#service-management)
2. [Database Operations](#database-operations)
3. [Monitoring and Alerting](#monitoring-and-alerting)
4. [Backup and Recovery](#backup-and-recovery)
5. [Security Operations](#security-operations)
6. [Performance Optimization](#performance-optimization)
7. [Incident Response](#incident-response)
8. [Maintenance Procedures](#maintenance-procedures)

## 🔧 Service Management

### Starting Services

**Start All Services**
```bash
#!/bin/bash
# Start all AegisShield services in dependency order

echo "Starting AegisShield services..."

# 1. Start infrastructure services first
kubectl scale deployment postgresql --replicas=1 -n aegisshield-prod
kubectl scale deployment neo4j --replicas=1 -n aegisshield-prod
kubectl scale deployment vault --replicas=1 -n aegisshield-prod

# Wait for databases to be ready
kubectl wait --for=condition=ready pod -l app=postgresql -n aegisshield-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=neo4j -n aegisshield-prod --timeout=300s

# 2. Start core services
services=("data-ingestion" "entity-resolution" "graph-engine")
for service in "${services[@]}"; do
  kubectl scale deployment $service --replicas=3 -n aegisshield-prod
  kubectl wait --for=condition=ready pod -l app=$service -n aegisshield-prod --timeout=300s
done

# 3. Start API and frontend
kubectl scale deployment api-gateway --replicas=3 -n aegisshield-prod
kubectl scale deployment alert-engine --replicas=3 -n aegisshield-prod
kubectl scale deployment frontend --replicas=3 -n aegisshield-prod

# 4. Verify all services are running
kubectl get pods -n aegisshield-prod

echo "All services started successfully"
```

**Stop All Services**
```bash
#!/bin/bash
# Gracefully stop all AegisShield services

echo "Stopping AegisShield services..."

# 1. Stop frontend and API first
kubectl scale deployment frontend --replicas=0 -n aegisshield-prod
kubectl scale deployment api-gateway --replicas=0 -n aegisshield-prod
kubectl scale deployment alert-engine --replicas=0 -n aegisshield-prod

# 2. Stop core services
services=("graph-engine" "entity-resolution" "data-ingestion")
for service in "${services[@]}"; do
  kubectl scale deployment $service --replicas=0 -n aegisshield-prod
done

# 3. Stop infrastructure services last
kubectl scale deployment vault --replicas=0 -n aegisshield-prod
kubectl scale deployment neo4j --replicas=0 -n aegisshield-prod
kubectl scale deployment postgresql --replicas=0 -n aegisshield-prod

echo "All services stopped successfully"
```

### Service Health Checks

**Comprehensive Health Check**
```bash
#!/bin/bash
# Comprehensive service health check

echo "=== AegisShield Health Check Report ==="
echo "Timestamp: $(date)"
echo "Environment: Production"
echo

# Function to check service health
check_service_health() {
  local service=$1
  local namespace=${2:-aegisshield-prod}
  
  echo "Checking $service..."
  
  # Pod status
  pods=$(kubectl get pods -n $namespace -l app=$service --no-headers | wc -l)
  ready_pods=$(kubectl get pods -n $namespace -l app=$service --no-headers | grep Running | wc -l)
  
  echo "  Pods: $ready_pods/$pods ready"
  
  # Health endpoint check (if available)
  if kubectl get service $service -n $namespace >/dev/null 2>&1; then
    health_status=$(kubectl run health-check-$service --rm -i --restart=Never --image=curlimages/curl --quiet -- \
      curl -s -o /dev/null -w "%{http_code}" http://$service.$namespace.svc.cluster.local:8080/health 2>/dev/null || echo "N/A")
    echo "  Health endpoint: $health_status"
  fi
  
  # Recent restarts
  restarts=$(kubectl get pods -n $namespace -l app=$service -o jsonpath='{.items[*].status.containerStatuses[*].restartCount}' | awk '{sum+=$1} END {print sum+0}')
  echo "  Recent restarts: $restarts"
  
  echo
}

# Check all services
services=("postgresql" "neo4j" "vault" "data-ingestion" "entity-resolution" "graph-engine" "api-gateway" "alert-engine" "frontend")

for service in "${services[@]}"; do
  check_service_health $service
done

# Overall system status
echo "=== Overall System Status ==="
total_pods=$(kubectl get pods -n aegisshield-prod --no-headers | wc -l)
running_pods=$(kubectl get pods -n aegisshield-prod --no-headers | grep Running | wc -l)
echo "Total running pods: $running_pods/$total_pods"

# Resource utilization
echo "=== Resource Utilization ==="
kubectl top nodes --no-headers | awk '{print "Node " $1 ": CPU " $3 ", Memory " $5}'

echo "=== Health Check Complete ==="
```

## 🗄️ Database Operations

### PostgreSQL Operations

**Database Connection Management**
```bash
#!/bin/bash
# PostgreSQL connection management

echo "=== PostgreSQL Connection Management ==="

# Check current connections
echo "1. Current connections:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    SELECT 
      datname,
      count(*) as connections,
      max_conn,
      round((count(*)*100/max_conn)::numeric, 2) as percentage
    FROM pg_stat_activity, 
         (SELECT setting::int as max_conn FROM pg_settings WHERE name='max_connections') mc
    GROUP BY datname, max_conn
    ORDER BY connections DESC;
  "

# Check for idle connections
echo "2. Idle connections:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    SELECT count(*) as idle_connections 
    FROM pg_stat_activity 
    WHERE state = 'idle' AND state_change < NOW() - INTERVAL '5 minutes';
  "

# Terminate idle connections
echo "3. Terminating long-idle connections..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    SELECT pg_terminate_backend(pid) 
    FROM pg_stat_activity 
    WHERE state = 'idle' 
      AND state_change < NOW() - INTERVAL '30 minutes'
      AND datname = 'aegisshield';
  "
```

**Database Performance Monitoring**
```bash
#!/bin/bash
# PostgreSQL performance monitoring

echo "=== PostgreSQL Performance Report ==="

# Top queries by execution time
echo "1. Slowest queries (by mean time):"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    SELECT 
      substr(query, 1, 60) as query_snippet,
      calls,
      total_time,
      mean_time,
      rows
    FROM pg_stat_statements 
    ORDER BY mean_time DESC 
    LIMIT 10;
  "

# Database cache hit ratio
echo "2. Cache hit ratio:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    SELECT 
      'buffer_cache' as cache_type,
      round(
        (sum(heap_blks_hit) * 100.0 / (sum(heap_blks_hit) + sum(heap_blks_read)))::numeric, 
        2
      ) as hit_ratio_percent
    FROM pg_statio_user_tables
    UNION ALL
    SELECT 
      'index_cache' as cache_type,
      round(
        (sum(idx_blks_hit) * 100.0 / (sum(idx_blks_hit) + sum(idx_blks_read)))::numeric, 
        2
      ) as hit_ratio_percent
    FROM pg_statio_user_indexes;
  "

# Lock analysis
echo "3. Current locks:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "
    SELECT 
      mode,
      count(*) as count
    FROM pg_locks 
    WHERE granted = true
    GROUP BY mode
    ORDER BY count DESC;
  "
```

### Neo4j Operations

**Neo4j Graph Maintenance**
```bash
#!/bin/bash
# Neo4j graph maintenance

echo "=== Neo4j Graph Maintenance ==="

# Database statistics
echo "1. Database statistics:"
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p $(kubectl get secret neo4j-secret -n aegisshield-prod -o jsonpath='{.data.password}' | base64 -d) \
  "CALL db.stats.retrieve('GRAPH COUNTS') YIELD section, data RETURN section, data;"

# Index status
echo "2. Index status:"
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p $(kubectl get secret neo4j-secret -n aegisshield-prod -o jsonpath='{.data.password}' | base64 -d) \
  "SHOW INDEXES;"

# Query performance
echo "3. Recent query performance:"
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p $(kubectl get secret neo4j-secret -n aegisshield-prod -o jsonpath='{.data.password}' | base64 -d) \
  "CALL dbms.listQueries() YIELD query, elapsedTimeMillis WHERE elapsedTimeMillis > 1000 RETURN query, elapsedTimeMillis ORDER BY elapsedTimeMillis DESC LIMIT 10;"

# Memory usage
echo "4. Memory usage:"
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p $(kubectl get secret neo4j-secret -n aegisshield-prod -o jsonpath='{.data.password}' | base64 -d) \
  "CALL dbms.queryJmx('java.lang:type=Memory') YIELD attributes RETURN attributes.HeapMemoryUsage;"
```

## 📊 Monitoring and Alerting

### Alert Management

**Check Active Alerts**
```bash
#!/bin/bash
# Check active alerts in Alertmanager

echo "=== Active Alerts Report ==="

# Get current alerts from Alertmanager
ALERTMANAGER_URL="http://alertmanager.monitoring.svc.cluster.local:9093"

curl -s "$ALERTMANAGER_URL/api/v1/alerts" | jq -r '
  .data[] | 
  select(.status.state == "active") |
  "\(.labels.alertname) - \(.labels.severity) - \(.annotations.summary)"
' | sort

# Check silenced alerts
echo "=== Silenced Alerts ==="
curl -s "$ALERTMANAGER_URL/api/v1/silences" | jq -r '
  .data[] |
  select(.status.state == "active") |
  "\(.matchers[].value) - Expires: \(.endsAt)"
'
```

**Silence Alert**
```bash
#!/bin/bash
# Silence an alert in Alertmanager

ALERT_NAME=$1
DURATION=${2:-"1h"}
COMMENT=${3:-"Silenced during maintenance"}

if [ -z "$ALERT_NAME" ]; then
  echo "Usage: silence_alert.sh <alert_name> [duration] [comment]"
  exit 1
fi

ALERTMANAGER_URL="http://alertmanager.monitoring.svc.cluster.local:9093"

# Create silence
curl -X POST "$ALERTMANAGER_URL/api/v1/silences" \
  -H "Content-Type: application/json" \
  -d "{
    \"matchers\": [
      {
        \"name\": \"alertname\",
        \"value\": \"$ALERT_NAME\",
        \"isRegex\": false
      }
    ],
    \"startsAt\": \"$(date -u +%Y-%m-%dT%H:%M:%S.000Z)\",
    \"endsAt\": \"$(date -u -d \"+$DURATION\" +%Y-%m-%dT%H:%M:%S.000Z)\",
    \"comment\": \"$COMMENT\",
    \"createdBy\": \"ops-script\"
  }"

echo "Alert $ALERT_NAME silenced for $DURATION"
```

### Metrics Collection

**System Metrics Report**
```bash
#!/bin/bash
# Comprehensive system metrics report

echo "=== System Metrics Report ==="
echo "Timestamp: $(date)"

# Prometheus URL
PROMETHEUS_URL="http://prometheus.monitoring.svc.cluster.local:9090"

# Function to query Prometheus
query_prometheus() {
  local query=$1
  local label=$2
  
  echo "$label:"
  curl -s "$PROMETHEUS_URL/api/v1/query?query=$query" | \
    jq -r '.data.result[] | "\(.metric.instance // .metric.job): \(.value[1])"' | \
    head -10
  echo
}

# API response times
query_prometheus 'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))' "API Response Times (95th percentile, 5min)"

# Error rates
query_prometheus 'rate(http_requests_total{status=~"5.."}[5m])' "Error Rates (5xx errors, 5min rate)"

# CPU utilization
query_prometheus 'avg by (instance) (100 * (1 - rate(node_cpu_seconds_total{mode="idle"}[5m])))' "CPU Utilization by Node"

# Memory utilization
query_prometheus 'avg by (instance) (100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)))' "Memory Utilization by Node"

# Database connections
query_prometheus 'pg_stat_database_numbackends' "PostgreSQL Active Connections"

# Neo4j memory usage
query_prometheus 'neo4j_ids_in_use_relationship' "Neo4j Relationships in Use"
```

## 💾 Backup and Recovery

### Manual Backup Operations

**Create Manual Backup**
```bash
#!/bin/bash
# Create manual backup of all critical data

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/tmp/aegisshield-backup-$BACKUP_DATE"

echo "Creating manual backup: $BACKUP_DATE"

# 1. PostgreSQL backup
echo "1. Backing up PostgreSQL..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  pg_dump -U postgres aegisshield > "$BACKUP_DIR/postgresql-$BACKUP_DATE.sql"

# 2. Neo4j backup
echo "2. Backing up Neo4j..."
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  neo4j-admin backup --backup-dir=/backup --name=manual-$BACKUP_DATE

kubectl cp aegisshield-prod/neo4j-0:/backup/manual-$BACKUP_DATE "$BACKUP_DIR/neo4j-$BACKUP_DATE"

# 3. Vault backup
echo "3. Backing up Vault..."
kubectl exec -n aegisshield-prod deployment/vault -- \
  vault operator raft snapshot save /tmp/vault-$BACKUP_DATE.snap

kubectl cp aegisshield-prod/vault-0:/tmp/vault-$BACKUP_DATE.snap "$BACKUP_DIR/vault-$BACKUP_DATE.snap"

# 4. Configuration backup
echo "4. Backing up Kubernetes configurations..."
kubectl get all,configmap,secret,pvc -n aegisshield-prod -o yaml > "$BACKUP_DIR/k8s-resources-$BACKUP_DATE.yaml"

# 5. Compress and upload
echo "5. Compressing backup..."
tar -czf "aegisshield-backup-$BACKUP_DATE.tar.gz" -C /tmp "aegisshield-backup-$BACKUP_DATE"

# Upload to S3 (if configured)
if [ ! -z "$AWS_S3_BACKUP_BUCKET" ]; then
  aws s3 cp "aegisshield-backup-$BACKUP_DATE.tar.gz" "s3://$AWS_S3_BACKUP_BUCKET/manual-backups/"
  echo "Backup uploaded to S3"
fi

echo "Manual backup completed: aegisshield-backup-$BACKUP_DATE.tar.gz"
```

### Recovery Operations

**Database Recovery**
```bash
#!/bin/bash
# Database recovery from backup

BACKUP_FILE=$1
RECOVERY_TYPE=${2:-"full"}

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: recover_database.sh <backup_file> [full|partial]"
  exit 1
fi

echo "Starting database recovery from: $BACKUP_FILE"
echo "Recovery type: $RECOVERY_TYPE"

# 1. Scale down applications
echo "1. Scaling down applications..."
kubectl scale deployment api-gateway --replicas=0 -n aegisshield-prod
kubectl scale deployment data-ingestion --replicas=0 -n aegisshield-prod
kubectl scale deployment entity-resolution --replicas=0 -n aegisshield-prod

# 2. PostgreSQL recovery
if [[ "$RECOVERY_TYPE" == "full" ]]; then
  echo "2. Recovering PostgreSQL..."
  kubectl exec -n aegisshield-prod deployment/postgresql -- \
    psql -U postgres -c "DROP DATABASE IF EXISTS aegisshield;"
  kubectl exec -n aegisshield-prod deployment/postgresql -- \
    psql -U postgres -c "CREATE DATABASE aegisshield;"
  
  kubectl exec -i -n aegisshield-prod deployment/postgresql -- \
    psql -U postgres aegisshield < "$BACKUP_FILE/postgresql.sql"
fi

# 3. Neo4j recovery
echo "3. Recovering Neo4j..."
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  neo4j-admin restore --from=/backup/restore --database=neo4j --force

# 4. Restart services
echo "4. Restarting services..."
kubectl rollout restart deployment/postgresql -n aegisshield-prod
kubectl rollout restart deployment/neo4j -n aegisshield-prod

kubectl wait --for=condition=ready pod -l app=postgresql -n aegisshield-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=neo4j -n aegisshield-prod --timeout=300s

# 5. Scale up applications
echo "5. Scaling up applications..."
kubectl scale deployment api-gateway --replicas=3 -n aegisshield-prod
kubectl scale deployment data-ingestion --replicas=3 -n aegisshield-prod
kubectl scale deployment entity-resolution --replicas=2 -n aegisshield-prod

echo "Database recovery completed"
```

## 🔒 Security Operations

### Security Audit

**Daily Security Check**
```bash
#!/bin/bash
# Daily security audit check

echo "=== Daily Security Audit ==="
echo "Date: $(date)"

# 1. Check for failed login attempts
echo "1. Failed authentication attempts (last 24h):"
kubectl logs -n aegisshield-prod deployment/api-gateway --since=24h | \
  grep -i "authentication failed\|unauthorized\|forbidden" | wc -l

# 2. Check certificate expiration
echo "2. Certificate expiration status:"
kubectl get certificate -n aegisshield-prod -o custom-columns="NAME:.metadata.name,READY:.status.conditions[0].status,SECRET:.spec.secretName"

# 3. Check for security vulnerabilities in images
echo "3. Container security scan results:"
kubectl get vulnerabilityreports -n aegisshield-prod --no-headers | \
  awk '{print $1 ": " $3 " criticals, " $4 " highs"}'

# 4. Check network policies
echo "4. Network policy status:"
kubectl get networkpolicy -n aegisshield-prod --no-headers | wc -l
echo "Network policies in place"

# 5. Check RBAC permissions
echo "5. Service account permissions:"
kubectl get rolebindings,clusterrolebindings -n aegisshield-prod --no-headers | wc -l
echo "Role bindings configured"

# 6. Check for privileged containers
echo "6. Privileged container check:"
kubectl get pods -n aegisshield-prod -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.securityContext.privileged}{"\n"}{end}' | \
  grep -v "null\|false" || echo "No privileged containers found"

echo "=== Security Audit Complete ==="
```

### Access Management

**User Access Review**
```bash
#!/bin/bash
# User access review

echo "=== User Access Review ==="

# 1. List all service accounts
echo "1. Service Accounts:"
kubectl get serviceaccounts -n aegisshield-prod

# 2. List role bindings
echo "2. Role Bindings:"
kubectl get rolebindings -n aegisshield-prod -o custom-columns="NAME:.metadata.name,ROLE:.roleRef.name,SUBJECTS:.subjects[*].name"

# 3. Check cluster role bindings
echo "3. Cluster Role Bindings:"
kubectl get clusterrolebindings -o custom-columns="NAME:.metadata.name,ROLE:.roleRef.name,SUBJECTS:.subjects[*].name" | \
  grep aegisshield

# 4. API access logs
echo "4. Recent API access (last 1h):"
kubectl logs -n aegisshield-prod deployment/api-gateway --since=1h | \
  grep -E "POST|PUT|DELETE" | \
  awk '{print $1 " " $7 " " $9}' | \
  sort | uniq -c | sort -nr | head -10
```

## 🚨 Incident Response

### Emergency Procedures

**Service Outage Response**
```bash
#!/bin/bash
# Emergency service outage response

SERVICE=$1
SEVERITY=${2:-"high"}

if [ -z "$SERVICE" ]; then
  echo "Usage: emergency_response.sh <service> [severity]"
  exit 1
fi

echo "=== EMERGENCY RESPONSE: $SERVICE OUTAGE ==="
echo "Severity: $SEVERITY"
echo "Timestamp: $(date)"

# 1. Immediate assessment
echo "1. Service Status Assessment:"
kubectl describe deployment $SERVICE -n aegisshield-prod
kubectl get pods -n aegisshield-prod -l app=$SERVICE

# 2. Check dependencies
echo "2. Dependency Check:"
if [[ "$SERVICE" != "postgresql" && "$SERVICE" != "neo4j" ]]; then
  kubectl get pods -n aegisshield-prod -l app=postgresql
  kubectl get pods -n aegisshield-prod -l app=neo4j
fi

# 3. Emergency restart
echo "3. Emergency Restart:"
kubectl rollout restart deployment/$SERVICE -n aegisshield-prod

# 4. Monitor recovery
echo "4. Monitoring Recovery:"
kubectl rollout status deployment/$SERVICE -n aegisshield-prod --timeout=300s

# 5. Health verification
echo "5. Health Verification:"
sleep 30
kubectl run emergency-health-check --rm -i --restart=Never --image=curlimages/curl -- \
  curl -f http://$SERVICE.aegisshield-prod.svc.cluster.local:8080/health

# 6. Notification
echo "6. Incident Notification:"
echo "Service $SERVICE outage detected and recovery initiated at $(date)"

# 7. Log collection for analysis
echo "7. Collecting logs for analysis:"
kubectl logs -n aegisshield-prod deployment/$SERVICE --previous > /tmp/$SERVICE-failure-logs-$(date +%Y%m%d_%H%M%S).log

echo "=== Emergency response completed for $SERVICE ==="
```

**Database Emergency Recovery**
```bash
#!/bin/bash
# Database emergency recovery procedures

DB_TYPE=$1  # postgresql or neo4j
ACTION=${2:-"restart"}  # restart, recover, or failover

echo "=== DATABASE EMERGENCY RECOVERY ==="
echo "Database: $DB_TYPE"
echo "Action: $ACTION"

case $DB_TYPE in
  "postgresql")
    case $ACTION in
      "restart")
        echo "Restarting PostgreSQL..."
        kubectl rollout restart deployment/postgresql -n aegisshield-prod
        kubectl wait --for=condition=ready pod -l app=postgresql -n aegisshield-prod --timeout=300s
        ;;
      "recover")
        echo "Recovering PostgreSQL from backup..."
        ./restore-database.sh postgresql
        ;;
      "failover")
        echo "PostgreSQL failover not implemented (single instance)"
        ;;
    esac
    ;;
  "neo4j")
    case $ACTION in
      "restart")
        echo "Restarting Neo4j..."
        kubectl rollout restart deployment/neo4j -n aegisshield-prod
        kubectl wait --for=condition=ready pod -l app=neo4j -n aegisshield-prod --timeout=300s
        ;;
      "recover")
        echo "Recovering Neo4j from backup..."
        ./restore-database.sh neo4j
        ;;
    esac
    ;;
esac

echo "Database emergency recovery completed"
```

## 🔧 Maintenance Procedures

### Routine Maintenance

**Weekly Maintenance Tasks**
```bash
#!/bin/bash
# Weekly maintenance routine

echo "=== AegisShield Weekly Maintenance ==="
echo "Date: $(date)"

# 1. Update system packages
echo "1. Updating system packages..."
kubectl create job weekly-system-update --from=cronjob/system-update -n kube-system

# 2. Clean up old logs
echo "2. Cleaning up old logs..."
kubectl create job weekly-log-cleanup --from=cronjob/log-cleanup -n monitoring

# 3. Database maintenance
echo "3. Running database maintenance..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "VACUUM ANALYZE;"

kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p password "CALL db.stats.clear();"

# 4. Security scan
echo "4. Running security scan..."
./infrastructure/scripts/security-audit.sh --weekly

# 5. Backup verification
echo "5. Verifying recent backups..."
kubectl get cronjobs -n aegisshield-prod | grep backup

# 6. Certificate renewal check
echo "6. Checking certificate expiration..."
kubectl get certificate -n aegisshield-prod

# 7. Performance report
echo "7. Generating performance report..."
./generate-performance-report.sh --weekly

echo "=== Weekly maintenance completed ==="
```

### Configuration Updates

**Update Application Configuration**
```bash
#!/bin/bash
# Update application configuration

CONFIG_TYPE=$1  # configmap or secret
CONFIG_NAME=$2
NAMESPACE=${3:-"aegisshield-prod"}

echo "Updating $CONFIG_TYPE: $CONFIG_NAME in namespace: $NAMESPACE"

# 1. Backup current configuration
echo "1. Backing up current configuration..."
kubectl get $CONFIG_TYPE $CONFIG_NAME -n $NAMESPACE -o yaml > /tmp/$CONFIG_NAME-backup-$(date +%Y%m%d_%H%M%S).yaml

# 2. Apply new configuration
echo "2. Applying new configuration..."
kubectl apply -f $CONFIG_NAME.yaml -n $NAMESPACE

# 3. Restart affected deployments
echo "3. Restarting affected deployments..."
kubectl get deployments -n $NAMESPACE -o json | \
  jq -r ".items[] | select(.spec.template.spec.volumes[]?.configMap.name == \"$CONFIG_NAME\" or .spec.template.spec.containers[]?.envFrom[]?.configMapRef.name == \"$CONFIG_NAME\") | .metadata.name" | \
  xargs -I {} kubectl rollout restart deployment/{} -n $NAMESPACE

# 4. Verify rollout
echo "4. Verifying rollout..."
kubectl get deployments -n $NAMESPACE -o json | \
  jq -r ".items[] | select(.spec.template.spec.volumes[]?.configMap.name == \"$CONFIG_NAME\" or .spec.template.spec.containers[]?.envFrom[]?.configMapRef.name == \"$CONFIG_NAME\") | .metadata.name" | \
  xargs -I {} kubectl rollout status deployment/{} -n $NAMESPACE

echo "Configuration update completed"
```

This comprehensive operational runbook provides detailed procedures for managing the AegisShield platform in production environments. Each section includes specific commands and troubleshooting steps for common operational scenarios.