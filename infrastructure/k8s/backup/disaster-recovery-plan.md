# AegisShield Backup & Recovery Documentation

This document provides comprehensive guidance for backup and recovery procedures for the AegisShield platform.

## 📋 Overview

AegisShield implements a robust backup and recovery strategy to ensure business continuity and data protection:

- **Automated Daily Backups** - Full system backups at 2 AM UTC
- **Incremental Backups** - Every 6 hours for critical data
- **Monthly Recovery Testing** - Automated validation of recovery procedures
- **Multi-Layer Security** - Encrypted backups with secure key management
- **Geographic Distribution** - Backups stored in multiple AWS regions

## 🎯 Recovery Objectives

| Component | RTO (Recovery Time Objective) | RPO (Recovery Point Objective) |
|-----------|-------------------------------|--------------------------------|
| Critical Services | 1 hour | 15 minutes |
| PostgreSQL | 30 minutes | 6 hours |
| Neo4j | 45 minutes | 6 hours |
| Vault | 15 minutes | 6 hours |
| Application Services | 2 hours | 6 hours |

## 🗂️ Backup Strategy

### Backup Types

**1. Full Backup (Daily)**
- Complete system state backup
- All databases and configurations
- Scheduled at 2 AM UTC daily
- 30-day retention period

**2. Incremental Backup (6-hourly)**
- Transaction logs and changes
- PostgreSQL WAL files
- Vault transaction logs
- 7-day retention period

**3. Configuration Backup (Weekly)**
- Kubernetes configurations
- Application settings
- Security policies
- 90-day retention period

### Backup Components

**PostgreSQL Databases:**
- Main application database (`aegisshield`)
- Investigation data (`investigations`)
- Compliance records (`compliance`)
- Analytics data (`analytics`)

**Neo4j Graph Database:**
- Entity relationship graphs
- Investigation networks
- Risk analysis data

**HashiCorp Vault:**
- Encryption keys
- Secrets and certificates
- Security policies

**Application Configuration:**
- Kubernetes manifests
- ConfigMaps and Secrets
- Service configurations

## 🚀 Quick Start

### Prerequisites

```bash
# Install required tools
kubectl get nodes  # Verify cluster access
aws configure      # Configure AWS credentials
gpg --version     # Verify GPG for encryption
```

### Manual Backup

```bash
# Run immediate full backup
cd /infrastructure/scripts
chmod +x backup-automation.sh
./backup-automation.sh

# Check backup status
kubectl get jobs -n aegisshield | grep backup
```

### Manual Recovery Test

```bash
# Run recovery validation
cd /infrastructure/scripts
chmod +x recovery-testing.sh
./recovery-testing.sh

# Review test results
cat /tmp/recovery_test_report_*.json
```

## 📁 File Structure

```
backup/
├── backup-cronjobs.yaml           # Kubernetes CronJobs for automated backups
├── backup-automation.sh           # Main backup script
├── recovery-testing.sh            # Recovery validation script
├── disaster-recovery-plan.md      # This documentation
└── runbooks/
    ├── backup-procedures.md        # Detailed backup procedures
    ├── recovery-procedures.md      # Step-by-step recovery guide
    └── troubleshooting.md         # Common issues and solutions
```

## ⚙️ Configuration

### Backup Schedule Configuration

```yaml
schedules:
  full_backup:
    cron: "0 2 * * *"        # Daily at 2 AM UTC
    retention_days: 30
    
  incremental_backup:
    cron: "0 */6 * * *"      # Every 6 hours
    retention_days: 7
    
  recovery_test:
    cron: "0 3 1 * *"        # Monthly on 1st
    notification: required
```

### Storage Configuration

**Local Storage:**
- Path: `/var/backups/aegisshield`
- Size: 500GB persistent volume
- Encryption: AES-256 with GPG

**Cloud Storage:**
- S3 Bucket: `aegisshield-backups`
- Encryption: Server-side AES-256
- Storage Class: Standard-IA
- Cross-region replication enabled

### Encryption

**Backup Encryption:**
- Algorithm: AES-256 with GPG
- Key Management: HashiCorp Vault
- Key Rotation: Quarterly
- Passphrase: 32-character random string

## 🔄 Automated Procedures

### Daily Full Backup

**Process:**
1. Database snapshots created
2. Configuration exported
3. Files encrypted with GPG
4. Upload to S3 with versioning
5. Local cleanup (30-day retention)
6. Backup verification
7. Monitoring alerts sent

**Monitoring:**
- Backup completion status
- File size validation
- Upload verification
- Error alerting

### Incremental Backup

**PostgreSQL WAL Backup:**
```bash
# WAL archiving configuration
archive_mode = on
archive_command = '/opt/scripts/archive-wal.sh %p %f'
archive_timeout = 300  # 5 minutes
```

**Vault Transaction Log:**
```bash
# Continuous backup of Vault operations
vault audit enable file file_path=/var/log/vault/audit.log
```

### Recovery Testing

**Monthly Validation:**
1. Latest backup retrieved
2. Test environment provisioned
3. Recovery procedures executed
4. Data integrity validated
5. Performance benchmarks run
6. Test environment cleaned up
7. Results reported

## 🚨 Disaster Recovery Procedures

### Complete System Recovery

**Phase 1: Infrastructure Preparation (15 minutes)**
```bash
# 1. Provision new Kubernetes cluster
kubectl create namespace aegisshield-recovery

# 2. Setup storage classes and volumes
kubectl apply -f infrastructure/k8s/storage/

# 3. Deploy secrets and configmaps
kubectl apply -f infrastructure/k8s/secrets/
```

**Phase 2: Database Recovery (45 minutes)**
```bash
# 1. Deploy PostgreSQL
kubectl apply -f infrastructure/k8s/services/postgresql.yaml

# 2. Restore PostgreSQL databases
./scripts/restore-postgresql.sh --backup-date=20241201_020000

# 3. Deploy Neo4j
kubectl apply -f infrastructure/k8s/services/neo4j.yaml

# 4. Restore Neo4j database
./scripts/restore-neo4j.sh --backup-date=20241201_020000

# 5. Deploy and restore Vault
kubectl apply -f infrastructure/k8s/services/vault.yaml
./scripts/restore-vault.sh --backup-date=20241201_020000
```

**Phase 3: Application Recovery (60 minutes)**
```bash
# 1. Deploy application services
kubectl apply -f infrastructure/k8s/services/

# 2. Verify service health
kubectl get pods -n aegisshield

# 3. Run post-recovery validation
./scripts/post-recovery-validation.sh

# 4. Enable monitoring and alerting
kubectl apply -f infrastructure/k8s/monitoring/
```

### Partial Recovery Scenarios

**PostgreSQL Only Recovery:**
```bash
# Stop application services
kubectl scale deployment --replicas=0 -n aegisshield --all

# Restore database
./scripts/restore-postgresql.sh --database=aegisshield

# Restart services
kubectl scale deployment --replicas=1 -n aegisshield --all
```

**Configuration Recovery:**
```bash
# Restore Kubernetes configurations
kubectl apply -f /var/backups/aegisshield/latest/configs/

# Restart affected services
kubectl rollout restart deployment -n aegisshield
```

## 📊 Monitoring & Alerting

### Backup Monitoring

**Metrics Tracked:**
- Backup completion status
- Backup duration and size
- Upload success/failure rates
- Storage utilization
- Recovery test results

**Alerts Configured:**
- Backup failure (immediate)
- Backup size anomaly (warning)
- Storage space low (warning)
- Recovery test failure (critical)

### Prometheus Metrics

```yaml
# Backup metrics
aegis_backup_duration_seconds          # Backup completion time
aegis_backup_size_bytes                # Backup file size
aegis_backup_success_total             # Successful backups
aegis_backup_failure_total             # Failed backups
aegis_recovery_test_duration_seconds   # Recovery test time
aegis_recovery_test_success            # Recovery test status
```

### Grafana Dashboards

**Backup Operations Dashboard:**
- Backup success/failure trends
- Backup duration over time
- Storage utilization
- Recovery test results

## 🔧 Maintenance Procedures

### Weekly Tasks

**Backup Validation:**
```bash
# Verify recent backups
aws s3 ls s3://aegisshield-backups/$(date '+%Y/%m/%d')/

# Check backup integrity
./scripts/verify-backup-integrity.sh

# Review backup logs
kubectl logs -n aegisshield -l app=aegisshield-backup --tail=100
```

### Monthly Tasks

**Recovery Testing:**
```bash
# Full recovery test
./scripts/recovery-testing.sh --full-test

# Performance validation
./scripts/validate-recovery-performance.sh

# Update recovery documentation
git add docs/recovery-test-$(date '+%Y%m').md
```

### Quarterly Tasks

**Key Rotation:**
```bash
# Generate new encryption key
./scripts/rotate-backup-encryption-key.sh

# Update backup procedures with new key
kubectl create secret generic backup-encryption-key \
  --from-file=key=/etc/aegisshield/new-backup-key

# Restart backup jobs with new key
kubectl rollout restart cronjob -n aegisshield
```

## 🔍 Troubleshooting

### Common Issues

**Backup Failure - Disk Space:**
```bash
# Check available space
df -h /var/backups/aegisshield

# Cleanup old backups
./scripts/cleanup-old-backups.sh --days=20

# Verify backup can proceed
./scripts/backup-automation.sh --dry-run
```

**S3 Upload Failure:**
```bash
# Check AWS credentials
aws sts get-caller-identity

# Test S3 connectivity
aws s3 ls s3://aegisshield-backups/

# Check IAM permissions
aws iam simulate-principal-policy \
  --policy-source-arn arn:aws:iam::ACCOUNT:role/backup-role \
  --action-names s3:PutObject s3:GetObject \
  --resource-arns arn:aws:s3:::aegisshield-backups/*
```

**Database Restore Failure:**
```bash
# Check backup file integrity
gpg --decrypt backup.sql.gpg | head -10

# Verify database connectivity
kubectl exec -it deployment/postgresql -n aegisshield -- \
  psql -U postgres -c "SELECT version();"

# Check available disk space
kubectl exec deployment/postgresql -n aegisshield -- df -h
```

**Recovery Test Timeout:**
```bash
# Check test environment resources
kubectl get pods -n aegisshield-recovery-test
kubectl describe pod -n aegisshield-recovery-test

# Monitor test progress
kubectl logs -f -n aegisshield-recovery-test deployment/recovery-test

# Increase test timeout
export RECOVERY_TEST_TIMEOUT=3600  # 1 hour
```

### Performance Optimization

**Backup Performance:**
```bash
# Parallel database dumps
export BACKUP_PARALLEL_JOBS=4

# Compression optimization
export BACKUP_COMPRESSION_LEVEL=6

# Network bandwidth limit
export BACKUP_BANDWIDTH_LIMIT=100M
```

**Recovery Performance:**
```bash
# Parallel restore operations
export RESTORE_PARALLEL_JOBS=2

# Memory allocation for large restores
export RESTORE_MEMORY_LIMIT=8Gi

# Temporary disk space for restore
export RESTORE_TEMP_SPACE=100Gi
```

## 📞 Emergency Contacts

### Escalation Matrix

**P0 - Critical System Down:**
1. On-call Engineer (PagerDuty)
2. System Architect
3. CTO

**P1 - Backup Failure:**
1. DevOps Team Lead
2. Database Administrator
3. Security Officer

**P2 - Recovery Test Failure:**
1. Platform Engineer
2. QA Lead
3. Operations Manager

### Contact Information

- **Emergency Hotline:** +1-800-AEGIS-911
- **DevOps Team:** devops@aegisshield.com
- **Security Team:** security@aegisshield.com
- **Documentation:** https://docs.aegisshield.com/backup

## 📚 References

- [AWS S3 Backup Best Practices](https://docs.aws.amazon.com/s3/latest/userguide/backup-best-practices.html)
- [PostgreSQL Backup and Recovery](https://www.postgresql.org/docs/current/backup.html)
- [Neo4j Backup Procedures](https://neo4j.com/docs/operations-manual/current/backup-restore/)
- [HashiCorp Vault Backup](https://www.vaultproject.io/docs/concepts/integrated-storage/autopilot#automated-backups)
- [Kubernetes Backup Strategies](https://kubernetes.io/docs/concepts/cluster-administration/backup/)