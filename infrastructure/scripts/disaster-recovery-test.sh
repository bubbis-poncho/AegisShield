#!/bin/bash
# AegisShield Disaster Recovery Testing Script
# This script tests comprehensive disaster recovery procedures
# to validate RTO (Recovery Time Objective) and RPO (Recovery Point Objective)

set -e

# Configuration
DR_TEST_DATE=$(date +%Y%m%d_%H%M%S)
DR_TEST_LOG="/var/log/aegisshield/dr-test-$DR_TEST_DATE.log"
DR_BACKUP_DIR="/tmp/dr-test-backup-$DR_TEST_DATE"
NAMESPACE="aegisshield-prod"
DR_NAMESPACE="aegisshield-dr-test"
PROMETHEUS_URL="http://prometheus.monitoring.svc.cluster.local:9090"

# RTO/RPO Targets
RTO_TARGET_MINUTES=30  # 30 minutes maximum downtime
RPO_TARGET_MINUTES=15  # 15 minutes maximum data loss

# Logging setup
exec > >(tee -a $DR_TEST_LOG)
exec 2>&1

echo "==============================================="
echo "    AegisShield Disaster Recovery Test"
echo "==============================================="
echo "Test ID: $DR_TEST_DATE"
echo "Start Time: $(date)"
echo "RTO Target: $RTO_TARGET_MINUTES minutes"
echo "RPO Target: $RPO_TARGET_MINUTES minutes"
echo "==============================================="
echo

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to measure time
start_timer() {
    echo $(date +%s)
}

end_timer() {
    local start_time=$1
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    echo $duration
}

# Function to check service health
check_service_health() {
    local service=$1
    local namespace=$2
    local timeout=${3:-60}
    
    log "Checking health of $service in namespace $namespace"
    
    # Wait for pods to be ready
    if kubectl wait --for=condition=ready pod -l app=$service -n $namespace --timeout=${timeout}s; then
        log "✅ $service is healthy"
        return 0
    else
        log "❌ $service failed health check"
        return 1
    fi
}

# Function to verify data integrity
verify_data_integrity() {
    local namespace=$1
    
    log "Verifying data integrity in namespace $namespace"
    
    # PostgreSQL data verification
    local pg_count=$(kubectl exec -n $namespace deployment/postgresql -- \
        psql -U postgres -t -c "SELECT count(*) FROM investigations;" 2>/dev/null || echo "0")
    log "PostgreSQL investigation records: $pg_count"
    
    # Neo4j data verification
    local neo4j_count=$(kubectl exec -n $namespace deployment/neo4j -- \
        cypher-shell -u neo4j -p password "MATCH (n) RETURN count(n);" 2>/dev/null | tail -n 1 || echo "0")
    log "Neo4j node count: $neo4j_count"
    
    # Return success if we have data
    if [ "$pg_count" -gt 0 ] && [ "$neo4j_count" -gt 0 ]; then
        log "✅ Data integrity verified"
        return 0
    else
        log "❌ Data integrity check failed"
        return 1
    fi
}

# Phase 1: Pre-Test Setup and Baseline
echo "==============================================="
echo "PHASE 1: PRE-TEST SETUP AND BASELINE"
echo "==============================================="

log "Creating DR test namespace..."
kubectl create namespace $DR_NAMESPACE || log "Namespace already exists"
kubectl label namespace $DR_NAMESPACE test=disaster-recovery

# Capture baseline metrics
log "Capturing baseline metrics..."
BASELINE_START=$(start_timer)

# Get current data counts for verification
BASELINE_PG_COUNT=$(kubectl exec -n $NAMESPACE deployment/postgresql -- \
    psql -U postgres -t -c "SELECT count(*) FROM investigations;" 2>/dev/null || echo "0")
BASELINE_NEO4J_COUNT=$(kubectl exec -n $NAMESPACE deployment/neo4j -- \
    cypher-shell -u neo4j -p password "MATCH (n) RETURN count(n);" 2>/dev/null | tail -n 1 || echo "0")

log "Baseline PostgreSQL records: $BASELINE_PG_COUNT"
log "Baseline Neo4j nodes: $BASELINE_NEO4J_COUNT"

# Test current system responsiveness
log "Testing baseline system performance..."
if kubectl run baseline-test --rm -i --restart=Never --image=curlimages/curl -- \
    curl -f -s -o /dev/null http://api-gateway.$NAMESPACE.svc.cluster.local:8080/health; then
    log "✅ Baseline system is responsive"
else
    log "⚠️  Baseline system is not responding optimally"
fi

BASELINE_END=$(end_timer $BASELINE_START)
log "Baseline capture completed in $BASELINE_END seconds"

# Phase 2: Create Fresh Backup
echo
echo "==============================================="
echo "PHASE 2: CREATE DISASTER RECOVERY BACKUP"
echo "==============================================="

BACKUP_START=$(start_timer)
log "Creating disaster recovery backup..."

mkdir -p $DR_BACKUP_DIR

# PostgreSQL backup
log "Creating PostgreSQL backup..."
kubectl exec -n $NAMESPACE deployment/postgresql -- \
    pg_dump -U postgres -Fc aegisshield > $DR_BACKUP_DIR/postgresql-dr.backup

if [ $? -eq 0 ]; then
    log "✅ PostgreSQL backup created successfully"
else
    log "❌ PostgreSQL backup failed"
    exit 1
fi

# Neo4j backup
log "Creating Neo4j backup..."
kubectl exec -n $NAMESPACE deployment/neo4j -- \
    neo4j-admin backup --backup-dir=/backup --name=dr-test-$DR_TEST_DATE

kubectl cp $NAMESPACE/neo4j-0:/backup/dr-test-$DR_TEST_DATE $DR_BACKUP_DIR/neo4j-dr-backup

if [ $? -eq 0 ]; then
    log "✅ Neo4j backup created successfully"
else
    log "❌ Neo4j backup failed"
    exit 1
fi

# Vault backup
log "Creating Vault backup..."
kubectl exec -n $NAMESPACE deployment/vault -- \
    vault operator raft snapshot save /tmp/vault-dr.snap

kubectl cp $NAMESPACE/vault-0:/tmp/vault-dr.snap $DR_BACKUP_DIR/vault-dr.snap

if [ $? -eq 0 ]; then
    log "✅ Vault backup created successfully"
else
    log "❌ Vault backup failed"
    exit 1
fi

# Configuration backup
log "Creating configuration backup..."
kubectl get all,configmap,secret,pvc -n $NAMESPACE -o yaml > $DR_BACKUP_DIR/k8s-resources.yaml

BACKUP_END=$(end_timer $BACKUP_START)
log "Backup creation completed in $BACKUP_END seconds"

# Calculate RPO compliance
if [ $BACKUP_END -le $((RPO_TARGET_MINUTES * 60)) ]; then
    log "✅ RPO target met: Backup completed in $BACKUP_END seconds (target: $((RPO_TARGET_MINUTES * 60)) seconds)"
else
    log "❌ RPO target missed: Backup took $BACKUP_END seconds (target: $((RPO_TARGET_MINUTES * 60)) seconds)"
fi

# Phase 3: Simulate Disaster Scenario
echo
echo "==============================================="
echo "PHASE 3: SIMULATE DISASTER SCENARIO"
echo "==============================================="

DISASTER_START=$(start_timer)
log "Simulating disaster scenario - scaling down all services..."

# Scale down all services to simulate total failure
services=("frontend" "api-gateway" "alert-engine" "graph-engine" "entity-resolution" "data-ingestion" "vault" "neo4j" "postgresql")

for service in "${services[@]}"; do
    log "Scaling down $service..."
    kubectl scale deployment $service --replicas=0 -n $NAMESPACE
done

# Wait for all pods to terminate
log "Waiting for all services to terminate..."
kubectl wait --for=delete pod --all -n $NAMESPACE --timeout=300s

# Verify disaster state
log "Verifying disaster state..."
running_pods=$(kubectl get pods -n $NAMESPACE --no-headers 2>/dev/null | grep Running | wc -l)
if [ $running_pods -eq 0 ]; then
    log "✅ Disaster simulation successful - all services down"
else
    log "⚠️  $running_pods pods still running after disaster simulation"
fi

DISASTER_END=$(end_timer $DISASTER_START)
log "Disaster simulation completed in $DISASTER_END seconds"

# Phase 4: Execute Disaster Recovery
echo
echo "==============================================="
echo "PHASE 4: EXECUTE DISASTER RECOVERY"
echo "==============================================="

RECOVERY_START=$(start_timer)
log "Starting disaster recovery process..."

# Deploy infrastructure services in DR namespace first
log "Deploying infrastructure services to DR namespace..."

# PostgreSQL
log "Deploying PostgreSQL..."
helm install postgresql-dr infrastructure/helm/postgresql/ \
    --namespace $DR_NAMESPACE \
    --values infrastructure/helm/postgresql/values-production.yaml \
    --set nameOverride=postgresql \
    --wait --timeout=600s

check_service_health postgresql $DR_NAMESPACE 300

# Neo4j
log "Deploying Neo4j..."
helm install neo4j-dr infrastructure/helm/neo4j/ \
    --namespace $DR_NAMESPACE \
    --values infrastructure/helm/neo4j/values-production.yaml \
    --set nameOverride=neo4j \
    --wait --timeout=600s

check_service_health neo4j $DR_NAMESPACE 300

# Vault
log "Deploying Vault..."
helm install vault-dr infrastructure/helm/vault/ \
    --namespace $DR_NAMESPACE \
    --values infrastructure/helm/vault/values-production.yaml \
    --set nameOverride=vault \
    --wait --timeout=600s

check_service_health vault $DR_NAMESPACE 300

# Phase 5: Restore Data
echo
echo "==============================================="
echo "PHASE 5: RESTORE DATA FROM BACKUP"
echo "==============================================="

RESTORE_START=$(start_timer)
log "Restoring data from backup..."

# Restore PostgreSQL
log "Restoring PostgreSQL data..."
kubectl cp $DR_BACKUP_DIR/postgresql-dr.backup $DR_NAMESPACE/postgresql-0:/tmp/restore.backup
kubectl exec -n $DR_NAMESPACE deployment/postgresql -- \
    pg_restore -U postgres -d aegisshield --clean --if-exists /tmp/restore.backup

if [ $? -eq 0 ]; then
    log "✅ PostgreSQL data restored successfully"
else
    log "❌ PostgreSQL data restore failed"
fi

# Restore Neo4j
log "Restoring Neo4j data..."
kubectl cp $DR_BACKUP_DIR/neo4j-dr-backup $DR_NAMESPACE/neo4j-0:/backup/restore
kubectl exec -n $DR_NAMESPACE deployment/neo4j -- \
    neo4j-admin restore --from=/backup/restore --database=neo4j --force

kubectl rollout restart deployment/neo4j -n $DR_NAMESPACE
check_service_health neo4j $DR_NAMESPACE 300

if [ $? -eq 0 ]; then
    log "✅ Neo4j data restored successfully"
else
    log "❌ Neo4j data restore failed"
fi

# Restore Vault
log "Restoring Vault data..."
kubectl cp $DR_BACKUP_DIR/vault-dr.snap $DR_NAMESPACE/vault-0:/tmp/restore.snap
kubectl exec -n $DR_NAMESPACE deployment/vault -- \
    vault operator raft snapshot restore /tmp/restore.snap

if [ $? -eq 0 ]; then
    log "✅ Vault data restored successfully"
else
    log "❌ Vault data restore failed"
fi

RESTORE_END=$(end_timer $RESTORE_START)
log "Data restoration completed in $RESTORE_END seconds"

# Phase 6: Deploy Application Services
echo
echo "==============================================="
echo "PHASE 6: DEPLOY APPLICATION SERVICES"
echo "==============================================="

APP_DEPLOY_START=$(start_timer)
log "Deploying application services..."

# Deploy core services
app_services=("data-ingestion" "entity-resolution" "graph-engine" "alert-engine" "api-gateway" "frontend")

for service in "${app_services[@]}"; do
    log "Deploying $service..."
    helm install "$service-dr" "infrastructure/helm/$service/" \
        --namespace $DR_NAMESPACE \
        --values "infrastructure/helm/$service/values-production.yaml" \
        --set nameOverride=$service \
        --wait --timeout=600s
    
    check_service_health $service $DR_NAMESPACE 300
done

APP_DEPLOY_END=$(end_timer $APP_DEPLOY_START)
log "Application services deployment completed in $APP_DEPLOY_END seconds"

# Phase 7: Verify Recovery
echo
echo "==============================================="
echo "PHASE 7: VERIFY DISASTER RECOVERY"
echo "==============================================="

VERIFICATION_START=$(start_timer)
log "Verifying disaster recovery..."

# Data integrity verification
verify_data_integrity $DR_NAMESPACE

# Service functionality verification
log "Testing service functionality..."

# Test API Gateway health
if kubectl run dr-test --rm -i --restart=Never --image=curlimages/curl -- \
    curl -f -s -o /dev/null http://api-gateway.$DR_NAMESPACE.svc.cluster.local:8080/health; then
    log "✅ API Gateway is responding"
else
    log "❌ API Gateway health check failed"
fi

# Test database connectivity
if kubectl exec -n $DR_NAMESPACE deployment/postgresql -- pg_isready; then
    log "✅ PostgreSQL is accepting connections"
else
    log "❌ PostgreSQL connection failed"
fi

if kubectl exec -n $DR_NAMESPACE deployment/neo4j -- \
    cypher-shell -u neo4j -p password "RETURN 1;" >/dev/null 2>&1; then
    log "✅ Neo4j is accepting connections"
else
    log "❌ Neo4j connection failed"
fi

# Verify data consistency
RECOVERED_PG_COUNT=$(kubectl exec -n $DR_NAMESPACE deployment/postgresql -- \
    psql -U postgres -t -c "SELECT count(*) FROM investigations;" 2>/dev/null || echo "0")
RECOVERED_NEO4J_COUNT=$(kubectl exec -n $DR_NAMESPACE deployment/neo4j -- \
    cypher-shell -u neo4j -p password "MATCH (n) RETURN count(n);" 2>/dev/null | tail -n 1 || echo "0")

log "Recovered PostgreSQL records: $RECOVERED_PG_COUNT (baseline: $BASELINE_PG_COUNT)"
log "Recovered Neo4j nodes: $RECOVERED_NEO4J_COUNT (baseline: $BASELINE_NEO4J_COUNT)"

# Calculate data consistency
if [ "$RECOVERED_PG_COUNT" -eq "$BASELINE_PG_COUNT" ] && [ "$RECOVERED_NEO4J_COUNT" -eq "$BASELINE_NEO4J_COUNT" ]; then
    log "✅ Complete data consistency achieved"
    DATA_LOSS=0
else
    log "⚠️  Data inconsistency detected"
    PG_LOSS=$((BASELINE_PG_COUNT - RECOVERED_PG_COUNT))
    NEO4J_LOSS=$((BASELINE_NEO4J_COUNT - RECOVERED_NEO4J_COUNT))
    log "PostgreSQL records lost: $PG_LOSS"
    log "Neo4j nodes lost: $NEO4J_LOSS"
    DATA_LOSS=1
fi

VERIFICATION_END=$(end_timer $VERIFICATION_START)
log "Verification completed in $VERIFICATION_END seconds"

# Calculate total recovery time
TOTAL_RECOVERY_TIME=$(end_timer $RECOVERY_START)
TOTAL_RECOVERY_MINUTES=$((TOTAL_RECOVERY_TIME / 60))

# Phase 8: Generate Recovery Report
echo
echo "==============================================="
echo "PHASE 8: DISASTER RECOVERY TEST REPORT"
echo "==============================================="

log "Generating disaster recovery test report..."

# RTO Analysis
if [ $TOTAL_RECOVERY_MINUTES -le $RTO_TARGET_MINUTES ]; then
    RTO_STATUS="✅ PASSED"
    log "RTO Target: $RTO_TARGET_MINUTES minutes"
    log "Actual Recovery Time: $TOTAL_RECOVERY_MINUTES minutes"
    log "RTO Status: PASSED"
else
    RTO_STATUS="❌ FAILED"
    log "RTO Target: $RTO_TARGET_MINUTES minutes"
    log "Actual Recovery Time: $TOTAL_RECOVERY_MINUTES minutes"
    log "RTO Status: FAILED - Exceeded target by $((TOTAL_RECOVERY_MINUTES - RTO_TARGET_MINUTES)) minutes"
fi

# RPO Analysis
BACKUP_MINUTES=$((BACKUP_END / 60))
if [ $BACKUP_MINUTES -le $RPO_TARGET_MINUTES ]; then
    RPO_STATUS="✅ PASSED"
    log "RPO Target: $RPO_TARGET_MINUTES minutes"
    log "Actual Backup Time: $BACKUP_MINUTES minutes"
    log "RPO Status: PASSED"
else
    RPO_STATUS="❌ FAILED"
    log "RPO Target: $RPO_TARGET_MINUTES minutes"
    log "Actual Backup Time: $BACKUP_MINUTES minutes"
    log "RPO Status: FAILED - Exceeded target by $((BACKUP_MINUTES - RPO_TARGET_MINUTES)) minutes"
fi

# Data Integrity Analysis
if [ $DATA_LOSS -eq 0 ]; then
    DATA_STATUS="✅ PASSED"
    log "Data Integrity: PASSED - No data loss detected"
else
    DATA_STATUS="❌ FAILED"
    log "Data Integrity: FAILED - Data inconsistency detected"
fi

# Overall Test Result
if [[ "$RTO_STATUS" == *"PASSED"* ]] && [[ "$RPO_STATUS" == *"PASSED"* ]] && [[ "$DATA_STATUS" == *"PASSED"* ]]; then
    OVERALL_STATUS="✅ PASSED"
    log "Overall DR Test Result: PASSED"
else
    OVERALL_STATUS="❌ FAILED"
    log "Overall DR Test Result: FAILED"
fi

# Phase 9: Cleanup
echo
echo "==============================================="
echo "PHASE 9: CLEANUP AND RESTORATION"
echo "==============================================="

log "Starting cleanup and restoration..."

# Option to keep DR environment for further testing
read -p "Keep DR environment for further testing? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log "Cleaning up DR test environment..."
    kubectl delete namespace $DR_NAMESPACE
    log "DR test namespace deleted"
else
    log "DR test environment preserved in namespace: $DR_NAMESPACE"
fi

# Restore original environment
log "Restoring original production environment..."
for service in "${services[@]}"; do
    original_replicas=$(helm get values $service -n $NAMESPACE -a | yq .replicaCount // 1)
    kubectl scale deployment $service --replicas=$original_replicas -n $NAMESPACE
done

# Wait for services to be ready
log "Waiting for production services to recover..."
for service in "${app_services[@]}"; do
    check_service_health $service $NAMESPACE 300
done

log "Production environment restored"

# Cleanup temporary files
rm -rf $DR_BACKUP_DIR
log "Temporary backup files cleaned up"

# Generate final summary
echo
echo "==============================================="
echo "    DISASTER RECOVERY TEST SUMMARY"
echo "==============================================="
echo "Test ID: $DR_TEST_DATE"
echo "End Time: $(date)"
echo "Total Test Duration: $(end_timer $BASELINE_START) seconds"
echo
echo "RTO Assessment: $RTO_STATUS"
echo "  Target: $RTO_TARGET_MINUTES minutes"
echo "  Actual: $TOTAL_RECOVERY_MINUTES minutes"
echo
echo "RPO Assessment: $RPO_STATUS"
echo "  Target: $RPO_TARGET_MINUTES minutes"
echo "  Actual: $BACKUP_MINUTES minutes"
echo
echo "Data Integrity: $DATA_STATUS"
echo
echo "Overall Result: $OVERALL_STATUS"
echo
echo "Phase Breakdown:"
echo "  Baseline Capture: $BASELINE_END seconds"
echo "  Backup Creation: $BACKUP_END seconds"
echo "  Disaster Simulation: $DISASTER_END seconds"
echo "  Infrastructure Recovery: $((RESTORE_END - RECOVERY_START)) seconds"
echo "  Data Restoration: $((RESTORE_END - RESTORE_START)) seconds"
echo "  Application Deployment: $APP_DEPLOY_END seconds"
echo "  Verification: $VERIFICATION_END seconds"
echo
echo "Log file: $DR_TEST_LOG"
echo "==============================================="

# Exit with appropriate code
if [[ "$OVERALL_STATUS" == *"PASSED"* ]]; then
    exit 0
else
    exit 1
fi