# AegisShield Production Deployment Guide

## 📋 Overview

This document provides comprehensive guidance for deploying the AegisShield financial crime investigation platform to production environments. It covers deployment procedures, operational runbooks, maintenance schedules, and troubleshooting guides.

## 🎯 Production Environment Requirements

### Infrastructure Prerequisites

**Kubernetes Cluster:**
- Kubernetes v1.24+ with RBAC enabled
- Minimum 8 nodes with 16GB RAM, 8 vCPU each
- Storage class with SSD-backed persistent volumes
- Load balancer support (AWS ALB, GCP LB, or similar)
- Network policies support

**External Dependencies:**
- Domain name with SSL certificate
- SMTP server for notifications
- S3-compatible storage for backups
- Monitoring infrastructure (if external)

### Resource Requirements

| Component | CPU | Memory | Storage | Replicas |
|-----------|-----|--------|---------|----------|
| API Gateway | 2000m | 4Gi | 10Gi | 3 |
| Data Ingestion | 1000m | 2Gi | 5Gi | 3 |
| Entity Resolution | 2000m | 8Gi | 10Gi | 2 |
| Alert Engine | 1000m | 2Gi | 5Gi | 3 |
| Graph Engine | 1500m | 4Gi | 10Gi | 2 |
| PostgreSQL | 4000m | 8Gi | 100Gi | 1 |
| Neo4j | 2000m | 4Gi | 50Gi | 1 |
| Vault | 500m | 1Gi | 10Gi | 1 |
| Frontend | 500m | 1Gi | 1Gi | 3 |

**Total Cluster Requirements:**
- **CPU:** 60+ vCPUs
- **Memory:** 150+ GB RAM
- **Storage:** 500+ GB

## 🚀 Deployment Procedures

### Phase 1: Pre-Deployment Preparation

**1.1 Environment Setup**
```bash
# Create production namespace
kubectl create namespace aegisshield-prod

# Label namespace for monitoring
kubectl label namespace aegisshield-prod environment=production

# Create service accounts
kubectl apply -f infrastructure/k8s/rbac/
```

**1.2 Secrets Management**
```bash
# Create encryption keys
openssl rand -base64 32 > encryption.key

# Create database credentials
kubectl create secret generic postgresql-secret \
  --from-literal=username=postgres \
  --from-literal=password=$(openssl rand -base64 32) \
  -n aegisshield-prod

# Create JWT signing key
kubectl create secret generic jwt-secret \
  --from-literal=signing-key=$(openssl rand -base64 64) \
  -n aegisshield-prod

# Create TLS certificates
kubectl create secret tls aegisshield-tls \
  --cert=path/to/certificate.crt \
  --key=path/to/private.key \
  -n aegisshield-prod
```

**1.3 Storage Provisioning**
```bash
# Apply storage classes
kubectl apply -f infrastructure/k8s/storage/

# Create persistent volume claims
kubectl apply -f infrastructure/k8s/storage/pvcs/
```

### Phase 2: Database Deployment

**2.1 PostgreSQL Deployment**
```bash
# Deploy PostgreSQL with HA configuration
helm install postgresql infrastructure/helm/postgresql/ \
  --namespace aegisshield-prod \
  --values infrastructure/helm/postgresql/values-production.yaml \
  --wait --timeout=600s

# Verify deployment
kubectl get pods -n aegisshield-prod -l app=postgresql
kubectl logs -n aegisshield-prod deployment/postgresql
```

**2.2 Neo4j Deployment**
```bash
# Deploy Neo4j cluster
helm install neo4j infrastructure/helm/neo4j/ \
  --namespace aegisshield-prod \
  --values infrastructure/helm/neo4j/values-production.yaml \
  --wait --timeout=600s

# Verify cluster formation
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p $(kubectl get secret neo4j-secret -o jsonpath='{.data.password}' | base64 -d) \
  "SHOW DATABASES"
```

**2.3 Vault Deployment**
```bash
# Deploy HashiCorp Vault
helm install vault infrastructure/helm/vault/ \
  --namespace aegisshield-prod \
  --values infrastructure/helm/vault/values-production.yaml \
  --wait --timeout=600s

# Initialize and unseal Vault
kubectl exec -n aegisshield-prod vault-0 -- vault operator init
kubectl exec -n aegisshield-prod vault-0 -- vault operator unseal <key1>
kubectl exec -n aegisshield-prod vault-0 -- vault operator unseal <key2>
kubectl exec -n aegisshield-prod vault-0 -- vault operator unseal <key3>
```

### Phase 3: Application Services Deployment

**3.1 Backend Services**
```bash
# Deploy in dependency order
services=("data-ingestion" "entity-resolution" "graph-engine" "alert-engine" "api-gateway")

for service in "${services[@]}"; do
  echo "Deploying $service..."
  
  helm install "$service" "infrastructure/helm/$service/" \
    --namespace aegisshield-prod \
    --values "infrastructure/helm/$service/values-production.yaml" \
    --wait --timeout=600s
  
  # Verify deployment
  kubectl rollout status deployment/"$service" -n aegisshield-prod
done
```

**3.2 Frontend Deployment**
```bash
# Deploy frontend application
helm install frontend infrastructure/helm/frontend/ \
  --namespace aegisshield-prod \
  --values infrastructure/helm/frontend/values-production.yaml \
  --wait --timeout=600s

# Verify deployment
kubectl get pods -n aegisshield-prod -l app=frontend
```

### Phase 4: Monitoring and Observability

**4.1 Monitoring Stack**
```bash
# Deploy monitoring namespace
kubectl create namespace monitoring

# Deploy Prometheus, Grafana, Alertmanager
./infrastructure/scripts/deploy-monitoring.sh

# Import production dashboards
kubectl apply -f infrastructure/k8s/monitoring/grafana-dashboards.yaml
```

**4.2 Logging Configuration**
```bash
# Deploy logging stack (if not using external)
helm install fluentd infrastructure/helm/fluentd/ \
  --namespace monitoring \
  --values infrastructure/helm/fluentd/values-production.yaml
```

### Phase 5: External Integrations

**5.1 Load Balancer Configuration**
```bash
# Deploy ingress with SSL termination
kubectl apply -f infrastructure/k8s/ingress/production-ingress.yaml

# Verify SSL certificate
kubectl get certificate -n aegisshield-prod
```

**5.2 DNS Configuration**
```bash
# Get load balancer external IP
EXTERNAL_IP=$(kubectl get service nginx-ingress-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Configure DNS records
# A record: aegisshield.yourdomain.com -> $EXTERNAL_IP
# CNAME record: api.aegisshield.yourdomain.com -> aegisshield.yourdomain.com
```

## 📚 Operational Runbooks

### 📖 Daily Operations

**Daily Health Check Procedure**
```bash
#!/bin/bash
# Daily health check script

echo "=== AegisShield Daily Health Check ==="
echo "Date: $(date)"
echo

# Check pod status
echo "1. Pod Status:"
kubectl get pods -n aegisshield-prod | grep -v Running

# Check service health
echo "2. Service Health:"
services=("api-gateway" "data-ingestion" "entity-resolution" "alert-engine")
for service in "${services[@]}"; do
  status=$(curl -s -o /dev/null -w "%{http_code}" http://$service.aegisshield-prod.svc.cluster.local:8080/health)
  echo "$service: $status"
done

# Check database connectivity
echo "3. Database Connectivity:"
kubectl exec -n aegisshield-prod deployment/postgresql -- pg_isready
kubectl exec -n aegisshield-prod deployment/neo4j -- cypher-shell -u neo4j -p password "RETURN 1"

# Check disk usage
echo "4. Storage Usage:"
kubectl exec -n aegisshield-prod deployment/postgresql -- df -h /var/lib/postgresql/data
kubectl exec -n aegisshield-prod deployment/neo4j -- df -h /data

# Check recent alerts
echo "5. Recent Critical Alerts:"
kubectl logs -n monitoring deployment/alertmanager --since=24h | grep CRITICAL

echo "=== Health Check Complete ==="
```

### 🔄 Weekly Maintenance

**Weekly Maintenance Checklist**
- [ ] Review monitoring dashboards and alerts
- [ ] Check backup completion status
- [ ] Review system resource utilization
- [ ] Update security patches (if available)
- [ ] Review application logs for errors
- [ ] Validate SSL certificate expiration
- [ ] Check database statistics and performance
- [ ] Review user access and permissions

**Weekly Performance Review**
```bash
#!/bin/bash
# Weekly performance review script

echo "=== AegisShield Weekly Performance Review ==="

# API response times
echo "1. API Performance (last 7 days):"
kubectl exec -n monitoring deployment/prometheus -- \
  promtool query instant 'avg_over_time(http_request_duration_seconds[7d])'

# Database performance
echo "2. Database Performance:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT query, calls, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# Resource utilization
echo "3. Resource Utilization:"
kubectl top nodes
kubectl top pods -n aegisshield-prod

# Error rates
echo "4. Error Rates (last 7 days):"
kubectl exec -n monitoring deployment/prometheus -- \
  promtool query instant 'rate(http_requests_total{status=~"5.."}[7d])'
```

### 🚨 Incident Response

**Critical Service Down**
```bash
#!/bin/bash
# Service down incident response

SERVICE_NAME=$1

echo "=== INCIDENT: $SERVICE_NAME Down ==="

# 1. Check pod status
echo "1. Checking pod status..."
kubectl describe pod -n aegisshield-prod -l app=$SERVICE_NAME

# 2. Check recent logs
echo "2. Recent logs (last 100 lines)..."
kubectl logs -n aegisshield-prod deployment/$SERVICE_NAME --tail=100

# 3. Check resource constraints
echo "3. Resource usage..."
kubectl top pod -n aegisshield-prod -l app=$SERVICE_NAME

# 4. Check dependencies
echo "4. Checking dependencies..."
kubectl get pods -n aegisshield-prod -l tier=database

# 5. Restart service if needed
echo "5. Restarting service..."
kubectl rollout restart deployment/$SERVICE_NAME -n aegisshield-prod

# 6. Monitor recovery
echo "6. Monitoring recovery..."
kubectl rollout status deployment/$SERVICE_NAME -n aegisshield-prod
```

**Database Performance Issues**
```bash
#!/bin/bash
# Database performance incident response

echo "=== INCIDENT: Database Performance Issues ==="

# 1. Check active connections
echo "1. Active connections:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';"

# 2. Check long-running queries
echo "2. Long-running queries:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT query, query_start, state FROM pg_stat_activity WHERE state != 'idle' AND query_start < NOW() - INTERVAL '5 minutes';"

# 3. Check locks
echo "3. Lock information:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT * FROM pg_locks WHERE NOT granted;"

# 4. Database cache hit ratio
echo "4. Cache hit ratio:"
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) AS cache_hit_ratio FROM pg_statio_user_tables;"
```

## 🔧 Maintenance Procedures

### Security Updates

**Security Patch Deployment**
```bash
#!/bin/bash
# Security patch deployment procedure

echo "=== Security Patch Deployment ==="

# 1. Check current versions
echo "1. Current component versions:"
kubectl get pods -n aegisshield-prod -o custom-columns="NAME:.metadata.name,IMAGE:.spec.containers[0].image"

# 2. Rolling update with new images
services=("api-gateway" "data-ingestion" "entity-resolution" "alert-engine")

for service in "${services[@]}"; do
  echo "Updating $service..."
  kubectl set image deployment/$service -n aegisshield-prod $service=aegisshield/$service:latest-secure
  kubectl rollout status deployment/$service -n aegisshield-prod
done

# 3. Verify security fixes
echo "3. Running security validation..."
./infrastructure/scripts/security-audit.sh --quick

# 4. Monitor for issues
echo "4. Monitoring for 30 minutes..."
sleep 1800
```

### Database Maintenance

**Database Optimization**
```bash
#!/bin/bash
# Database maintenance procedure

echo "=== Database Maintenance ==="

# 1. PostgreSQL maintenance
echo "1. PostgreSQL maintenance..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "VACUUM ANALYZE;"

kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "REINDEX DATABASE aegisshield;"

# 2. Neo4j maintenance
echo "2. Neo4j maintenance..."
kubectl exec -n aegisshield-prod deployment/neo4j -- \
  cypher-shell -u neo4j -p password "CALL db.stats.clear();"

# 3. Update statistics
echo "3. Updating database statistics..."
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT pg_stat_reset();"
```

### Certificate Renewal

**SSL Certificate Renewal**
```bash
#!/bin/bash
# SSL certificate renewal procedure

echo "=== SSL Certificate Renewal ==="

# 1. Check certificate expiration
echo "1. Checking certificate expiration..."
kubectl get certificate -n aegisshield-prod

# 2. Renew certificate (cert-manager)
echo "2. Renewing certificate..."
kubectl delete certificate aegisshield-tls -n aegisshield-prod
kubectl apply -f infrastructure/k8s/certificates/

# 3. Verify renewal
echo "3. Verifying certificate renewal..."
kubectl describe certificate aegisshield-tls -n aegisshield-prod

# 4. Test SSL
echo "4. Testing SSL connection..."
curl -I https://aegisshield.yourdomain.com
```

## 🚨 Troubleshooting Guide

### Common Issues

**1. Pod Startup Failures**
```bash
# Check pod events
kubectl describe pod <pod-name> -n aegisshield-prod

# Check resource constraints
kubectl top node

# Check PVC binding
kubectl get pvc -n aegisshield-prod

# Common fixes:
kubectl delete pod <pod-name> -n aegisshield-prod  # Restart pod
kubectl scale deployment <deployment> --replicas=0 -n aegisshield-prod  # Scale down
kubectl scale deployment <deployment> --replicas=3 -n aegisshield-prod  # Scale up
```

**2. Database Connection Issues**
```bash
# Test PostgreSQL connectivity
kubectl exec -n aegisshield-prod deployment/postgresql -- pg_isready

# Check connection pool
kubectl logs -n aegisshield-prod deployment/api-gateway | grep "connection pool"

# Reset connections
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'aegisshield' AND state = 'idle';"
```

**3. High Memory Usage**
```bash
# Check memory usage
kubectl top pods -n aegisshield-prod

# Check for memory leaks
kubectl exec -n aegisshield-prod <pod-name> -- ps aux

# Restart high-memory pods
kubectl delete pod <pod-name> -n aegisshield-prod
```

**4. Network Connectivity Issues**
```bash
# Test service connectivity
kubectl run test-pod --rm -i --restart=Never --image=busybox -- \
  wget -qO- http://api-gateway.aegisshield-prod.svc.cluster.local:8080/health

# Check network policies
kubectl get networkpolicy -n aegisshield-prod

# Check DNS resolution
kubectl run test-pod --rm -i --restart=Never --image=busybox -- \
  nslookup api-gateway.aegisshield-prod.svc.cluster.local
```

### Performance Issues

**High API Response Times**
```bash
# Check API metrics
kubectl exec -n monitoring deployment/prometheus -- \
  promtool query instant 'histogram_quantile(0.95, http_request_duration_seconds_bucket)'

# Check database performance
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "SELECT query, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# Scale API Gateway
kubectl scale deployment api-gateway --replicas=5 -n aegisshield-prod
```

**Database Slow Queries**
```bash
# Enable slow query logging
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "ALTER SYSTEM SET log_min_duration_statement = 1000;"

# Check slow queries
kubectl logs -n aegisshield-prod deployment/postgresql | grep "slow query"

# Analyze query plans
kubectl exec -n aegisshield-prod deployment/postgresql -- \
  psql -U postgres -c "EXPLAIN ANALYZE <slow-query>;"
```

## 📊 Monitoring and Alerting

### Key Metrics to Monitor

**Application Metrics:**
- API response times (p95 < 2 seconds)
- Error rates (< 1% for 4xx, < 0.1% for 5xx)
- Throughput (requests per second)
- Active user sessions
- Investigation processing times

**Infrastructure Metrics:**
- CPU utilization (< 70% average)
- Memory utilization (< 80% average)
- Disk usage (< 80% on all volumes)
- Network I/O and latency
- Pod restart frequency

**Database Metrics:**
- Connection pool utilization
- Query response times
- Active connections
- Lock wait times
- Cache hit ratios

### Alert Escalation Matrix

| Severity | Response Time | Escalation Path |
|----------|---------------|-----------------|
| Critical | 15 minutes | On-call → Team Lead → Manager |
| High | 1 hour | On-call → Team Lead |
| Medium | 4 hours | During business hours |
| Low | 24 hours | Next business day |

### Dashboard URLs

**Production Dashboards:**
- System Overview: `https://grafana.monitoring.local/d/aegisshield-overview`
- Application Performance: `https://grafana.monitoring.local/d/aegisshield-performance`
- Database Metrics: `https://grafana.monitoring.local/d/aegisshield-database`
- Business Metrics: `https://grafana.monitoring.local/d/aegisshield-business`

## 📞 Support and Escalation

### Contact Information

**Primary Contacts:**
- **On-Call Engineer:** +1-800-AEGIS-OPS
- **Development Team:** dev-team@aegisshield.com
- **DevOps Team:** devops@aegisshield.com
- **Security Team:** security@aegisshield.com

**Escalation Contacts:**
- **Engineering Manager:** engineering-manager@aegisshield.com
- **VP Engineering:** vp-engineering@aegisshield.com
- **CTO:** cto@aegisshield.com

### Communication Channels

**Slack Channels:**
- `#aegisshield-production` - Production issues and deployments
- `#aegisshield-alerts` - Automated alerts and monitoring
- `#aegisshield-oncall` - On-call coordination

**Emergency Procedures:**
1. **P0 (Critical):** Page on-call immediately, post in `#aegisshield-production`
2. **P1 (High):** Contact on-call within 1 hour, create incident ticket
3. **P2 (Medium):** Create ticket, handle during business hours
4. **P3 (Low):** Create ticket for next business day

## 📝 Change Management

### Deployment Windows

**Scheduled Maintenance:**
- **Major releases:** Second Saturday of each month, 2 AM UTC
- **Security patches:** As needed, with 24-hour notice
- **Hot fixes:** Emergency deployment, on-call approval required

**Change Approval Process:**
1. Change request submission
2. Security and performance review
3. Change advisory board approval
4. Deployment window scheduling
5. Rollback plan validation

### Rollback Procedures

**Application Rollback:**
```bash
# Rollback to previous version
kubectl rollout undo deployment/<service-name> -n aegisshield-prod

# Verify rollback
kubectl rollout status deployment/<service-name> -n aegisshield-prod

# Monitor for issues
kubectl logs -n aegisshield-prod deployment/<service-name> --tail=100
```

**Database Rollback:**
```bash
# Restore from backup (if data changes occurred)
./infrastructure/scripts/restore-database.sh --backup-date=<pre-change-backup>

# Verify data integrity
./infrastructure/scripts/validate-data-integrity.sh
```

This comprehensive production deployment guide provides all necessary procedures for successfully deploying, operating, and maintaining the AegisShield platform in production environments.