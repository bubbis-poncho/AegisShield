#!/bin/bash

# AegisShield Automated Backup Script
# Provides automated backup procedures for PostgreSQL, Neo4j, and Vault

set -e

# Configuration
BACKUP_BASE_DIR="/var/backups/aegisshield"
RETENTION_DAYS=30
S3_BUCKET="aegisshield-backups"
ENCRYPTION_KEY_FILE="/etc/aegisshield/backup-encryption.key"

# Service configurations
POSTGRES_HOST="postgresql.aegisshield.svc.cluster.local"
POSTGRES_PORT="5432"
POSTGRES_USER="postgres"
POSTGRES_DATABASES=("aegisshield" "investigations" "compliance" "analytics")

NEO4J_HOST="neo4j.aegisshield.svc.cluster.local"
NEO4J_PORT="7687"
NEO4J_USER="neo4j"

VAULT_ADDR="https://vault.aegisshield.svc.cluster.local:8200"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Create backup directory structure
setup_backup_directories() {
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    BACKUP_DATE_DIR="$BACKUP_BASE_DIR/$timestamp"
    
    mkdir -p "$BACKUP_DATE_DIR"/{postgresql,neo4j,vault,logs}
    
    log_info "Created backup directory: $BACKUP_DATE_DIR"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking backup prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "pg_dump" "aws" "gpg")
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
    
    # Check encryption key
    if [[ ! -f "$ENCRYPTION_KEY_FILE" ]]; then
        log_error "Encryption key file not found: $ENCRYPTION_KEY_FILE"
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS credentials not configured"
        exit 1
    fi
    
    log_success "All prerequisites checked"
}

# Get database credentials from Kubernetes secrets
get_database_credentials() {
    log_info "Retrieving database credentials..."
    
    # PostgreSQL password
    POSTGRES_PASSWORD=$(kubectl get secret postgresql-secret -n aegisshield -o jsonpath='{.data.password}' | base64 -d)
    
    # Neo4j password
    NEO4J_PASSWORD=$(kubectl get secret neo4j-secret -n aegisshield -o jsonpath='{.data.password}' | base64 -d)
    
    # Vault token
    VAULT_TOKEN=$(kubectl get secret vault-secret -n aegisshield -o jsonpath='{.data.token}' | base64 -d)
    
    log_success "Database credentials retrieved"
}

# Backup PostgreSQL databases
backup_postgresql() {
    log_info "Starting PostgreSQL backup..."
    
    local postgres_backup_dir="$BACKUP_DATE_DIR/postgresql"
    
    for db in "${POSTGRES_DATABASES[@]}"; do
        log_info "Backing up PostgreSQL database: $db"
        
        local backup_file="$postgres_backup_dir/${db}_$(date '+%Y%m%d_%H%M%S').sql"
        
        # Create database dump
        PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
            -h "$POSTGRES_HOST" \
            -p "$POSTGRES_PORT" \
            -U "$POSTGRES_USER" \
            -d "$db" \
            --verbose \
            --no-password \
            --format=custom \
            --compress=9 \
            --file="$backup_file" \
            2>> "$BACKUP_DATE_DIR/logs/postgresql_backup.log"
        
        if [[ $? -eq 0 ]]; then
            log_success "PostgreSQL database '$db' backup completed"
            
            # Encrypt backup
            gpg --cipher-algo AES256 --compress-algo 1 --s2k-mode 3 \
                --s2k-digest-algo SHA512 --s2k-count 65536 \
                --symmetric --output "${backup_file}.gpg" \
                --passphrase-file "$ENCRYPTION_KEY_FILE" \
                "$backup_file"
            
            # Remove unencrypted backup
            rm "$backup_file"
            
            log_success "PostgreSQL database '$db' backup encrypted"
        else
            log_error "PostgreSQL database '$db' backup failed"
            return 1
        fi
    done
    
    # Backup PostgreSQL configuration
    log_info "Backing up PostgreSQL configuration..."
    kubectl get configmap postgresql-config -n aegisshield -o yaml > "$postgres_backup_dir/postgresql-config.yaml"
    kubectl get secret postgresql-secret -n aegisshield -o yaml > "$postgres_backup_dir/postgresql-secret.yaml"
    
    log_success "PostgreSQL backup completed"
}

# Backup Neo4j database
backup_neo4j() {
    log_info "Starting Neo4j backup..."
    
    local neo4j_backup_dir="$BACKUP_DATE_DIR/neo4j"
    local backup_file="$neo4j_backup_dir/neo4j_$(date '+%Y%m%d_%H%M%S').dump"
    
    # Create Neo4j backup using neo4j-admin
    kubectl exec -n aegisshield deployment/neo4j -- \
        neo4j-admin database dump neo4j \
        --to-path=/tmp/neo4j-backup.dump \
        2>> "$BACKUP_DATE_DIR/logs/neo4j_backup.log"
    
    if [[ $? -eq 0 ]]; then
        # Copy backup from pod
        kubectl cp aegisshield/$(kubectl get pod -n aegisshield -l app=neo4j -o jsonpath='{.items[0].metadata.name}'):/tmp/neo4j-backup.dump "$backup_file"
        
        # Cleanup backup from pod
        kubectl exec -n aegisshield deployment/neo4j -- rm -f /tmp/neo4j-backup.dump
        
        log_success "Neo4j database backup completed"
        
        # Encrypt backup
        gpg --cipher-algo AES256 --compress-algo 1 --s2k-mode 3 \
            --s2k-digest-algo SHA512 --s2k-count 65536 \
            --symmetric --output "${backup_file}.gpg" \
            --passphrase-file "$ENCRYPTION_KEY_FILE" \
            "$backup_file"
        
        # Remove unencrypted backup
        rm "$backup_file"
        
        log_success "Neo4j database backup encrypted"
    else
        log_error "Neo4j database backup failed"
        return 1
    fi
    
    # Backup Neo4j configuration
    log_info "Backing up Neo4j configuration..."
    kubectl get configmap neo4j-config -n aegisshield -o yaml > "$neo4j_backup_dir/neo4j-config.yaml"
    kubectl get secret neo4j-secret -n aegisshield -o yaml > "$neo4j_backup_dir/neo4j-secret.yaml"
    
    log_success "Neo4j backup completed"
}

# Backup Vault data
backup_vault() {
    log_info "Starting Vault backup..."
    
    local vault_backup_dir="$BACKUP_DATE_DIR/vault"
    local backup_file="$vault_backup_dir/vault_$(date '+%Y%m%d_%H%M%S').json"
    
    # Set Vault environment
    export VAULT_ADDR="$VAULT_ADDR"
    export VAULT_TOKEN="$VAULT_TOKEN"
    
    # Create Vault snapshot
    kubectl exec -n aegisshield deployment/vault -- \
        vault operator raft snapshot save /tmp/vault-snapshot.snap \
        2>> "$BACKUP_DATE_DIR/logs/vault_backup.log"
    
    if [[ $? -eq 0 ]]; then
        # Copy snapshot from pod
        kubectl cp aegisshield/$(kubectl get pod -n aegisshield -l app=vault -o jsonpath='{.items[0].metadata.name}'):/tmp/vault-snapshot.snap "$backup_file"
        
        # Cleanup snapshot from pod
        kubectl exec -n aegisshield deployment/vault -- rm -f /tmp/vault-snapshot.snap
        
        log_success "Vault snapshot backup completed"
        
        # Encrypt backup
        gpg --cipher-algo AES256 --compress-algo 1 --s2k-mode 3 \
            --s2k-digest-algo SHA512 --s2k-count 65536 \
            --symmetric --output "${backup_file}.gpg" \
            --passphrase-file "$ENCRYPTION_KEY_FILE" \
            "$backup_file"
        
        # Remove unencrypted backup
        rm "$backup_file"
        
        log_success "Vault backup encrypted"
    else
        log_error "Vault backup failed"
        return 1
    fi
    
    # Backup Vault configuration
    log_info "Backing up Vault configuration..."
    kubectl get configmap vault-config -n aegisshield -o yaml > "$vault_backup_dir/vault-config.yaml"
    kubectl get secret vault-secret -n aegisshield -o yaml > "$vault_backup_dir/vault-secret.yaml"
    
    log_success "Vault backup completed"
}

# Upload backups to S3
upload_to_s3() {
    log_info "Uploading backups to S3..."
    
    local backup_archive="$BACKUP_BASE_DIR/aegisshield_backup_$(date '+%Y%m%d_%H%M%S').tar.gz"
    
    # Create compressed archive
    tar -czf "$backup_archive" -C "$BACKUP_BASE_DIR" "$(basename "$BACKUP_DATE_DIR")"
    
    # Upload to S3 with server-side encryption
    aws s3 cp "$backup_archive" "s3://$S3_BUCKET/$(date '+%Y/%m/%d')/" \
        --server-side-encryption AES256 \
        --storage-class STANDARD_IA \
        --metadata "backup-date=$(date '+%Y-%m-%d'),backup-type=full"
    
    if [[ $? -eq 0 ]]; then
        log_success "Backup uploaded to S3: s3://$S3_BUCKET/$(date '+%Y/%m/%d')/$(basename "$backup_archive")"
        
        # Remove local archive after successful upload
        rm "$backup_archive"
    else
        log_error "Failed to upload backup to S3"
        return 1
    fi
}

# Cleanup old backups
cleanup_old_backups() {
    log_info "Cleaning up old backups..."
    
    # Remove local backups older than retention period
    find "$BACKUP_BASE_DIR" -type d -name "20*" -mtime +$RETENTION_DAYS -exec rm -rf {} \; 2>/dev/null || true
    
    # Remove old S3 backups
    aws s3api list-objects-v2 --bucket "$S3_BUCKET" --query "Contents[?LastModified<='$(date -d "$RETENTION_DAYS days ago" --iso-8601)'].Key" --output text | \
    while read -r key; do
        if [[ -n "$key" && "$key" != "None" ]]; then
            aws s3 rm "s3://$S3_BUCKET/$key"
            log_info "Removed old backup: $key"
        fi
    done
    
    log_success "Old backup cleanup completed"
}

# Generate backup report
generate_backup_report() {
    log_info "Generating backup report..."
    
    local report_file="$BACKUP_DATE_DIR/backup_report.json"
    
    cat > "$report_file" << EOF
{
  "backup_info": {
    "timestamp": "$(date --iso-8601=seconds)",
    "backup_id": "$(basename "$BACKUP_DATE_DIR")",
    "version": "1.0",
    "retention_days": $RETENTION_DAYS
  },
  "services": {
    "postgresql": {
      "status": "completed",
      "databases": $(printf '%s\n' "${POSTGRES_DATABASES[@]}" | jq -R . | jq -s .),
      "backup_size": "$(du -sh "$BACKUP_DATE_DIR/postgresql" | cut -f1)"
    },
    "neo4j": {
      "status": "completed",
      "backup_size": "$(du -sh "$BACKUP_DATE_DIR/neo4j" | cut -f1)"
    },
    "vault": {
      "status": "completed",
      "backup_size": "$(du -sh "$BACKUP_DATE_DIR/vault" | cut -f1)"
    }
  },
  "storage": {
    "local_path": "$BACKUP_DATE_DIR",
    "s3_bucket": "$S3_BUCKET",
    "encryption": "AES256",
    "compression": "gzip"
  },
  "verification": {
    "checksums": {
      "postgresql": "$(find "$BACKUP_DATE_DIR/postgresql" -name "*.gpg" -exec sha256sum {} \; | sha256sum | cut -d' ' -f1)",
      "neo4j": "$(find "$BACKUP_DATE_DIR/neo4j" -name "*.gpg" -exec sha256sum {} \; | sha256sum | cut -d' ' -f1)",
      "vault": "$(find "$BACKUP_DATE_DIR/vault" -name "*.gpg" -exec sha256sum {} \; | sha256sum | cut -d' ' -f1)"
    }
  }
}
EOF
    
    log_success "Backup report generated: $report_file"
}

# Send backup notification
send_notification() {
    local status="$1"
    local message="$2"
    
    # Send to monitoring system
    if command -v curl &> /dev/null; then
        curl -X POST "http://alertmanager.monitoring.svc.cluster.local:9093/api/v1/alerts" \
            -H "Content-Type: application/json" \
            -d "[{
                \"labels\": {
                    \"alertname\": \"BackupStatus\",
                    \"service\": \"backup\",
                    \"severity\": \"$([[ "$status" == "success" ]] && echo "info" || echo "critical")\"
                },
                \"annotations\": {
                    \"summary\": \"AegisShield Backup $status\",
                    \"description\": \"$message\"
                }
            }]" &> /dev/null
    fi
    
    # Log notification
    if [[ "$status" == "success" ]]; then
        log_success "$message"
    else
        log_error "$message"
    fi
}

# Main backup function
main() {
    log_info "Starting AegisShield automated backup"
    
    local start_time=$(date +%s)
    
    # Setup
    setup_backup_directories
    check_prerequisites
    get_database_credentials
    
    # Perform backups
    if backup_postgresql && backup_neo4j && backup_vault; then
        # Post-backup tasks
        generate_backup_report
        upload_to_s3
        cleanup_old_backups
        
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        send_notification "success" "AegisShield backup completed successfully in ${duration}s"
        
        log_success "AegisShield backup completed successfully"
        log_info "Backup location: $BACKUP_DATE_DIR"
        log_info "Duration: ${duration} seconds"
        
        return 0
    else
        send_notification "failed" "AegisShield backup failed - check logs for details"
        log_error "AegisShield backup failed"
        return 1
    fi
}

# Trap errors and send notifications
trap 'send_notification "failed" "AegisShield backup script encountered an error"' ERR

# Run main function
main "$@"