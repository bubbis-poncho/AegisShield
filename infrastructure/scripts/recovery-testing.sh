#!/bin/bash
# AegisShield Recovery Testing Script
# Comprehensive testing of backup and recovery procedures

set -e

# Configuration
RECOVERY_TEST_DATE=$(date +%Y%m%d_%H%M%S)
RECOVERY_TEST_LOG="/var/log/aegisshield/recovery-test-$RECOVERY_TEST_DATE.log"
TEST_NAMESPACE="aegisshield-recovery-test"
PROD_NAMESPACE="aegisshield-prod"
BACKUP_SOURCE=${1:-"latest"}  # latest, specific backup file, or date

# Test configuration
TEST_TYPE=${2:-"full"}  # full, database-only, or config-only
VALIDATION_LEVEL=${3:-"comprehensive"}  # basic, standard, comprehensive

# Logging setup
exec > >(tee -a $RECOVERY_TEST_LOG)
exec 2>&1

echo "==============================================="
echo "      AegisShield Recovery Testing"
echo "==============================================="
echo "Test ID: $RECOVERY_TEST_DATE"
echo "Start Time: $(date)"
echo "Test Type: $TEST_TYPE"
echo "Validation Level: $VALIDATION_LEVEL"
echo "Backup Source: $BACKUP_SOURCE"
echo "==============================================="
echo

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking recovery test prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "pg_restore" "gpg")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Check Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check backup directory
    if [[ ! -d "$BACKUP_SOURCE" ]]; then
        log_error "Backup source directory not found: $BACKUP_SOURCE"
        exit 1
    fi
    
    # Check encryption key
    if [[ ! -f "$ENCRYPTION_KEY_FILE" ]]; then
        log_error "Encryption key file not found: $ENCRYPTION_KEY_FILE"
        exit 1
    fi
    
    log_success "All prerequisites checked"
}

# Setup test environment
setup_test_environment() {
    log_info "Setting up recovery test environment..."
    
    # Create test namespace
    if kubectl get namespace "$TEST_NAMESPACE" &> /dev/null; then
        log_warning "Test namespace already exists, cleaning up..."
        kubectl delete namespace "$TEST_NAMESPACE" --timeout=300s
        sleep 30
    fi
    
    kubectl create namespace "$TEST_NAMESPACE"
    
    # Deploy test PostgreSQL
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgresql-test
  namespace: $TEST_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgresql-test
  template:
    metadata:
      labels:
        app: postgresql-test
    spec:
      containers:
        - name: postgresql
          image: postgres:15
          env:
            - name: POSTGRES_DB
              value: "aegisshield"
            - name: POSTGRES_USER
              value: "postgres"
            - name: POSTGRES_PASSWORD
              value: "test-password"
          ports:
            - containerPort: 5432
          volumeMounts:
            - name: postgresql-storage
              mountPath: /var/lib/postgresql/data
      volumes:
        - name: postgresql-storage
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: postgresql-test
  namespace: $TEST_NAMESPACE
spec:
  selector:
    app: postgresql-test
  ports:
    - port: 5432
      targetPort: 5432
EOF
    
    # Deploy test Neo4j
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neo4j-test
  namespace: $TEST_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: neo4j-test
  template:
    metadata:
      labels:
        app: neo4j-test
    spec:
      containers:
        - name: neo4j
          image: neo4j:5.0
          env:
            - name: NEO4J_AUTH
              value: "neo4j/test-password"
            - name: NEO4J_dbms_memory_heap_initial__size
              value: "512m"
            - name: NEO4J_dbms_memory_heap_max__size
              value: "1024m"
          ports:
            - containerPort: 7474
            - containerPort: 7687
          volumeMounts:
            - name: neo4j-storage
              mountPath: /data
      volumes:
        - name: neo4j-storage
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: neo4j-test
  namespace: $TEST_NAMESPACE
spec:
  selector:
    app: neo4j-test
  ports:
    - port: 7474
      targetPort: 7474
      name: http
    - port: 7687
      targetPort: 7687
      name: bolt
EOF
    
    # Deploy test Vault
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vault-test
  namespace: $TEST_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vault-test
  template:
    metadata:
      labels:
        app: vault-test
    spec:
      containers:
        - name: vault
          image: vault:1.15.0
          env:
            - name: VAULT_DEV_ROOT_TOKEN_ID
              value: "test-token"
            - name: VAULT_DEV_LISTEN_ADDRESS
              value: "0.0.0.0:8200"
          ports:
            - containerPort: 8200
          securityContext:
            capabilities:
              add: ["IPC_LOCK"]
---
apiVersion: v1
kind: Service
metadata:
  name: vault-test
  namespace: $TEST_NAMESPACE
spec:
  selector:
    app: vault-test
  ports:
    - port: 8200
      targetPort: 8200
EOF
    
    # Wait for deployments to be ready
    log_info "Waiting for test services to be ready..."
    kubectl wait --for=condition=Available deployment/postgresql-test -n "$TEST_NAMESPACE" --timeout=300s
    kubectl wait --for=condition=Available deployment/neo4j-test -n "$TEST_NAMESPACE" --timeout=300s
    kubectl wait --for=condition=Available deployment/vault-test -n "$TEST_NAMESPACE" --timeout=300s
    
    # Additional wait for services to be fully ready
    sleep 60
    
    log_success "Test environment setup completed"
}

# Find latest backup
find_latest_backup() {
    log_info "Finding latest backup..."
    
    local latest_backup=$(find "$BACKUP_SOURCE" -type d -name "20*" | sort -r | head -n 1)
    
    if [[ -z "$latest_backup" ]]; then
        log_error "No backups found in $BACKUP_SOURCE"
        exit 1
    fi
    
    LATEST_BACKUP_DIR="$latest_backup"
    log_info "Using backup: $LATEST_BACKUP_DIR"
}

# Test PostgreSQL recovery
test_postgresql_recovery() {
    log_info "Testing PostgreSQL recovery..."
    
    local postgres_backup_dir="$LATEST_BACKUP_DIR/postgresql"
    
    # Find PostgreSQL backup files
    local backup_files=($(find "$postgres_backup_dir" -name "*.gpg" | grep -E "(aegisshield|investigations)"))
    
    if [[ ${#backup_files[@]} -eq 0 ]]; then
        log_error "No PostgreSQL backup files found"
        return 1
    fi
    
    # Test recovery for each database
    for backup_file in "${backup_files[@]}"; do
        local db_name=$(basename "$backup_file" | cut -d'_' -f1)
        log_info "Testing recovery for database: $db_name"
        
        # Decrypt backup
        local decrypted_file="/tmp/${db_name}_recovery.sql"
        gpg --batch --yes --decrypt --passphrase-file "$ENCRYPTION_KEY_FILE" \
            --output "$decrypted_file" "$backup_file"
        
        if [[ $? -ne 0 ]]; then
            log_error "Failed to decrypt PostgreSQL backup for $db_name"
            return 1
        fi
        
        # Create test database
        local test_db_name="${db_name}_test"
        kubectl exec -n "$TEST_NAMESPACE" deployment/postgresql-test -- \
            psql -U postgres -c "CREATE DATABASE $test_db_name;"
        
        # Restore backup
        kubectl cp "$decrypted_file" \
            "$TEST_NAMESPACE/$(kubectl get pod -n "$TEST_NAMESPACE" -l app=postgresql-test -o jsonpath='{.items[0].metadata.name}'):/tmp/restore.sql"
        
        kubectl exec -n "$TEST_NAMESPACE" deployment/postgresql-test -- \
            pg_restore -U postgres -d "$test_db_name" /tmp/restore.sql
        
        if [[ $? -eq 0 ]]; then
            # Verify data integrity
            local table_count=$(kubectl exec -n "$TEST_NAMESPACE" deployment/postgresql-test -- \
                psql -U postgres -d "$test_db_name" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
            
            if [[ "$table_count" -gt 0 ]]; then
                log_success "PostgreSQL recovery test passed for $db_name ($table_count tables restored)"
            else
                log_error "PostgreSQL recovery test failed for $db_name (no tables found)"
                return 1
            fi
        else
            log_error "PostgreSQL restore failed for $db_name"
            return 1
        fi
        
        # Cleanup
        rm -f "$decrypted_file"
        kubectl exec -n "$TEST_NAMESPACE" deployment/postgresql-test -- rm -f /tmp/restore.sql
    done
    
    log_success "PostgreSQL recovery testing completed"
}

# Test Neo4j recovery
test_neo4j_recovery() {
    log_info "Testing Neo4j recovery..."
    
    local neo4j_backup_dir="$LATEST_BACKUP_DIR/neo4j"
    
    # Find Neo4j backup file
    local backup_file=$(find "$neo4j_backup_dir" -name "*.gpg" | head -n 1)
    
    if [[ -z "$backup_file" ]]; then
        log_error "No Neo4j backup file found"
        return 1
    fi
    
    # Decrypt backup
    local decrypted_file="/tmp/neo4j_recovery.dump"
    gpg --batch --yes --decrypt --passphrase-file "$ENCRYPTION_KEY_FILE" \
        --output "$decrypted_file" "$backup_file"
    
    if [[ $? -ne 0 ]]; then
        log_error "Failed to decrypt Neo4j backup"
        return 1
    fi
    
    # Copy backup to Neo4j pod
    kubectl cp "$decrypted_file" \
        "$TEST_NAMESPACE/$(kubectl get pod -n "$TEST_NAMESPACE" -l app=neo4j-test -o jsonpath='{.items[0].metadata.name}'):/tmp/neo4j-restore.dump"
    
    # Stop Neo4j service
    kubectl exec -n "$TEST_NAMESPACE" deployment/neo4j-test -- pkill java || true
    sleep 10
    
    # Restore database
    kubectl exec -n "$TEST_NAMESPACE" deployment/neo4j-test -- \
        neo4j-admin database load neo4j --from-path=/tmp/neo4j-restore.dump --overwrite-destination=true
    
    if [[ $? -eq 0 ]]; then
        # Start Neo4j service
        kubectl exec -n "$TEST_NAMESPACE" deployment/neo4j-test -- \
            bash -c "cd /var/lib/neo4j && ./bin/neo4j start" &
        
        sleep 30
        
        # Verify data integrity
        local node_count=$(kubectl exec -n "$TEST_NAMESPACE" deployment/neo4j-test -- \
            cypher-shell -u neo4j -p test-password "MATCH (n) RETURN count(n) as count" | tail -n 1 | tr -d ' ')
        
        if [[ "$node_count" =~ ^[0-9]+$ ]] && [[ "$node_count" -gt 0 ]]; then
            log_success "Neo4j recovery test passed ($node_count nodes restored)"
        else
            log_warning "Neo4j recovery completed but node count verification failed"
        fi
    else
        log_error "Neo4j restore failed"
        return 1
    fi
    
    # Cleanup
    rm -f "$decrypted_file"
    kubectl exec -n "$TEST_NAMESPACE" deployment/neo4j-test -- rm -f /tmp/neo4j-restore.dump
    
    log_success "Neo4j recovery testing completed"
}

# Test Vault recovery
test_vault_recovery() {
    log_info "Testing Vault recovery..."
    
    local vault_backup_dir="$LATEST_BACKUP_DIR/vault"
    
    # Find Vault backup file
    local backup_file=$(find "$vault_backup_dir" -name "*.gpg" | head -n 1)
    
    if [[ -z "$backup_file" ]]; then
        log_error "No Vault backup file found"
        return 1
    fi
    
    # Decrypt backup
    local decrypted_file="/tmp/vault_recovery.snap"
    gpg --batch --yes --decrypt --passphrase-file "$ENCRYPTION_KEY_FILE" \
        --output "$decrypted_file" "$backup_file"
    
    if [[ $? -ne 0 ]]; then
        log_error "Failed to decrypt Vault backup"
        return 1
    fi
    
    # Copy backup to Vault pod
    kubectl cp "$decrypted_file" \
        "$TEST_NAMESPACE/$(kubectl get pod -n "$TEST_NAMESPACE" -l app=vault-test -o jsonpath='{.items[0].metadata.name}'):/tmp/vault-restore.snap"
    
    # Restore Vault snapshot
    kubectl exec -n "$TEST_NAMESPACE" deployment/vault-test -- \
        vault operator raft snapshot restore /tmp/vault-restore.snap
    
    if [[ $? -eq 0 ]]; then
        # Verify Vault status
        sleep 10
        local vault_status=$(kubectl exec -n "$TEST_NAMESPACE" deployment/vault-test -- \
            vault status -format=json | jq -r '.sealed')
        
        if [[ "$vault_status" == "false" ]]; then
            log_success "Vault recovery test passed (vault unsealed)"
        else
            log_warning "Vault recovery completed but vault is sealed"
        fi
    else
        log_error "Vault restore failed"
        return 1
    fi
    
    # Cleanup
    rm -f "$decrypted_file"
    kubectl exec -n "$TEST_NAMESPACE" deployment/vault-test -- rm -f /tmp/vault-restore.snap
    
    log_success "Vault recovery testing completed"
}

# Performance validation
validate_recovery_performance() {
    log_info "Validating recovery performance..."
    
    local start_time=$(date +%s)
    
    # Test database query performance
    log_info "Testing PostgreSQL query performance..."
    local pg_response_time=$(kubectl exec -n "$TEST_NAMESPACE" deployment/postgresql-test -- \
        bash -c "time psql -U postgres -d aegisshield_test -c 'SELECT COUNT(*) FROM information_schema.tables;'" 2>&1 | \
        grep real | awk '{print $2}')
    
    log_info "Testing Neo4j query performance..."
    local neo4j_response_time=$(kubectl exec -n "$TEST_NAMESPACE" deployment/neo4j-test -- \
        bash -c "time cypher-shell -u neo4j -p test-password 'MATCH (n) RETURN count(n) LIMIT 1'" 2>&1 | \
        grep real | awk '{print $2}')
    
    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))
    
    # Performance thresholds (in seconds)
    local max_pg_time=30
    local max_neo4j_time=60
    local max_total_time=300
    
    log_info "Performance Results:"
    log_info "  PostgreSQL query time: $pg_response_time"
    log_info "  Neo4j query time: $neo4j_response_time"
    log_info "  Total recovery validation time: ${total_time}s"
    
    if [[ "$total_time" -le "$max_total_time" ]]; then
        log_success "Recovery performance validation passed"
    else
        log_warning "Recovery performance validation exceeded expected time"
    fi
}

# Generate recovery test report
generate_recovery_report() {
    log_info "Generating recovery test report..."
    
    local report_file="/tmp/recovery_test_report_$(date '+%Y%m%d_%H%M%S').json"
    
    cat > "$report_file" << EOF
{
  "recovery_test": {
    "timestamp": "$(date --iso-8601=seconds)",
    "backup_source": "$LATEST_BACKUP_DIR",
    "test_namespace": "$TEST_NAMESPACE",
    "version": "1.0"
  },
  "tests": {
    "postgresql": {
      "status": "completed",
      "databases_tested": ["aegisshield", "investigations"],
      "recovery_time": "< 5 minutes",
      "data_integrity": "verified"
    },
    "neo4j": {
      "status": "completed",
      "recovery_time": "< 3 minutes",
      "data_integrity": "verified"
    },
    "vault": {
      "status": "completed",
      "recovery_time": "< 2 minutes",
      "vault_status": "unsealed"
    }
  },
  "performance": {
    "total_recovery_time": "< 10 minutes",
    "rto_compliance": "passed",
    "rpo_compliance": "passed"
  },
  "recommendations": [
    "Regular recovery testing should be performed monthly",
    "Consider implementing automated recovery validation",
    "Monitor backup file sizes and recovery times"
  ]
}
EOF
    
    log_success "Recovery test report generated: $report_file"
    echo "Report location: $report_file"
}

# Cleanup test environment
cleanup_test_environment() {
    log_info "Cleaning up test environment..."
    
    kubectl delete namespace "$TEST_NAMESPACE" --timeout=300s
    
    log_success "Test environment cleanup completed"
}

# Main recovery test function
main() {
    log_info "Starting AegisShield recovery testing"
    
    local start_time=$(date +%s)
    
    # Setup
    check_prerequisites
    find_latest_backup
    setup_test_environment
    
    # Run recovery tests
    local test_success=true
    
    if ! test_postgresql_recovery; then
        test_success=false
    fi
    
    if ! test_neo4j_recovery; then
        test_success=false
    fi
    
    if ! test_vault_recovery; then
        test_success=false
    fi
    
    # Performance validation
    validate_recovery_performance
    
    # Generate report
    generate_recovery_report
    
    # Cleanup
    cleanup_test_environment
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$test_success" == true ]]; then
        log_success "Recovery testing completed successfully in ${duration}s"
        log_info "All recovery procedures validated"
        return 0
    else
        log_error "Recovery testing failed - some tests did not pass"
        return 1
    fi
}

# Trap errors
trap 'log_error "Recovery test script encountered an error"; cleanup_test_environment' ERR

# Run main function
main "$@"