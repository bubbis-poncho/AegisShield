# AegisShield Production Deployment Guide

This guide provides comprehensive instructions for deploying AegisShield to production environments.

## Overview

AegisShield is deployed as a microservices architecture on Kubernetes with the following components:
- **API Gateway**: GraphQL API and authentication
- **Data Ingestion**: Transaction processing and normalization
- **Entity Resolution**: Entity matching and deduplication
- **Graph Engine**: Graph database operations and analytics
- **Alerting Engine**: Real-time alert generation and routing
- **Investigation Toolkit**: Case management and workflow
- **Analytics Dashboard**: Reporting and visualization
- **Frontend**: Next.js web application

## Prerequisites

### Infrastructure Requirements

- **Kubernetes Cluster**: v1.24+ with at least 6 nodes
- **Node Specifications**: 4 vCPU, 16GB RAM minimum per node
- **Storage**: SSD-based storage class with at least 500GB available
- **Network**: Load balancer with SSL termination capability
- **DNS**: Ability to configure DNS records for the domain

### Resource Requirements

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit | Storage |
|-----------|-------------|----------------|-----------|--------------|---------|
| PostgreSQL | 1 core | 2GB | 2 cores | 4GB | 100GB |
| Neo4j | 2 cores | 4GB | 4 cores | 8GB | 200GB |
| Kafka | 0.5 core | 1GB | 1 core | 2GB | 50GB |
| API Gateway | 0.5 core | 512MB | 1 core | 1GB | - |
| Data Ingestion | 1 core | 1GB | 2 cores | 2GB | - |
| Entity Resolution | 1 core | 2GB | 2 cores | 4GB | - |
| Graph Engine | 1 core | 1GB | 2 cores | 2GB | - |
| Alerting Engine | 0.5 core | 512MB | 1 core | 1GB | - |
| Frontend | 0.25 core | 256MB | 0.5 core | 512MB | - |
| **Total** | **8.75 cores** | **12.3GB** | **17 cores** | **25.5GB** | **350GB** |

### Required Tools

- `kubectl` v1.24+
- `helm` v3.8+
- `docker` v20.10+
- `terraform` v1.0+ (for infrastructure)
- `aws` CLI (if using AWS)

## Pre-Deployment Configuration

### 1. Environment Configuration

Create production environment configuration:

```bash
# Set environment variables
export DEPLOYMENT_ENV=production
export NAMESPACE=aegisshield-prod
export REGION=us-west-2
export CLUSTER_NAME=aegisshield-production
export DOMAIN_NAME=aegisshield.company.com
export IMAGE_REGISTRY=your-registry.com/aegisshield
export IMAGE_TAG=v1.0.0
```

### 2. Secrets Management

Configure required secrets:

```bash
# Database passwords
kubectl create secret generic database-secrets \
  --from-literal=postgres-password="$(openssl rand -base64 32)" \
  --from-literal=neo4j-password="$(openssl rand -base64 32)" \
  -n $NAMESPACE

# JWT signing key
kubectl create secret generic jwt-secret \
  --from-literal=jwt-signing-key="$(openssl rand -base64 64)" \
  -n $NAMESPACE

# External service credentials
kubectl create secret generic external-services \
  --from-literal=slack-webhook-url="$SLACK_WEBHOOK_URL" \
  --from-literal=pagerduty-key="$PAGERDUTY_KEY" \
  --from-literal=aws-access-key="$AWS_ACCESS_KEY" \
  --from-literal=aws-secret-key="$AWS_SECRET_KEY" \
  -n $NAMESPACE
```

### 3. TLS Certificates

Configure SSL certificates using cert-manager:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: aegisshield-tls
  namespace: aegisshield-prod
spec:
  secretName: aegisshield-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - aegisshield.company.com
  - api.aegisshield.company.com
  - grafana.aegisshield.company.com
```

## Deployment Process

### Step 1: Infrastructure Deployment

Execute the automated deployment script:

```bash
# Full production deployment
./scripts/production-deployment.sh deploy
```

Or deploy components manually:

```bash
# 1. Create namespace
kubectl create namespace $NAMESPACE

# 2. Deploy databases
helm upgrade --install postgresql bitnami/postgresql \
  --namespace $NAMESPACE \
  --values infrastructure/helm/values/postgresql-prod.yaml

kubectl apply -f infrastructure/k8s/neo4j.yaml -n $NAMESPACE

# 3. Deploy message queue
helm upgrade --install kafka bitnami/kafka \
  --namespace $NAMESPACE \
  --values infrastructure/helm/values/kafka-prod.yaml

# 4. Deploy monitoring
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  --namespace $NAMESPACE \
  --values infrastructure/helm/values/monitoring-prod.yaml
```

### Step 2: Application Deployment

```bash
# Deploy AegisShield services
helm upgrade --install aegisshield infrastructure/helm/aegisshield \
  --namespace $NAMESPACE \
  --values infrastructure/helm/values/production.yaml \
  --set global.imageTag=$IMAGE_TAG \
  --set global.domain=$DOMAIN_NAME \
  --wait --timeout=10m
```

### Step 3: Database Initialization

```bash
# Run database migrations
kubectl exec -n $NAMESPACE deployment/aegisshield-data-ingestion -- \
  ./migrate -path ./migrations -database "$POSTGRES_URL" up

# Initialize Neo4j schema
kubectl exec -n $NAMESPACE deployment/aegisshield-graph-engine -- \
  ./init-schema
```

### Step 4: Configuration Validation

```bash
# Validate deployment
./scripts/production-deployment.sh validate

# Check service health
kubectl get pods -n $NAMESPACE
kubectl get services -n $NAMESPACE
kubectl get ingress -n $NAMESPACE
```

## Post-Deployment Configuration

### 1. DNS Configuration

Configure DNS records to point to the load balancer:

```bash
# Get load balancer IP
LB_IP=$(kubectl get ingress aegisshield-ingress -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Configure DNS records
# aegisshield.company.com -> $LB_IP
# api.aegisshield.company.com -> $LB_IP
# grafana.aegisshield.company.com -> $LB_IP
```

### 2. Monitoring Setup

Access monitoring dashboards:

- **Grafana**: https://grafana.aegisshield.company.com
  - Default admin credentials in `grafana-admin` secret
  - Import AegisShield dashboards from `infrastructure/k8s/monitoring/dashboards/`

- **Prometheus**: https://prometheus.aegisshield.company.com
  - Configure alert rules and notification channels

- **Jaeger**: https://jaeger.aegisshield.company.com
  - Distributed tracing for performance monitoring

### 3. Backup Configuration

Configure automated backups:

```bash
# Database backups
kubectl apply -f infrastructure/k8s/backup/backup-cronjobs.yaml -n $NAMESPACE

# Volume snapshots
kubectl apply -f infrastructure/k8s/backup/volume-snapshots.yaml -n $NAMESPACE
```

### 4. Security Hardening

Apply security configurations:

```bash
# Network policies
kubectl apply -f infrastructure/k8s/security/network-policies.yaml -n $NAMESPACE

# Pod security policies
kubectl apply -f infrastructure/k8s/security/pod-security-policies.yaml -n $NAMESPACE

# RBAC
kubectl apply -f infrastructure/k8s/security/rbac.yaml -n $NAMESPACE
```

## Validation and Testing

### 1. Smoke Tests

Run basic functionality tests:

```bash
# API health check
curl -f https://api.aegisshield.company.com/health

# Authentication test
curl -X POST https://api.aegisshield.company.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# Data ingestion test
curl -X POST https://api.aegisshield.company.com/api/v1/transactions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @test-data/sample-transaction.json
```

### 2. Performance Testing

Execute performance validation:

```bash
# Run performance tests
python tests/performance/test_constitutional_requirements.py

# Load testing
kubectl apply -f tests/performance/load-test-job.yaml -n $NAMESPACE
```

### 3. Security Audit

Execute security audit:

```bash
# Comprehensive security audit
python tests/security/security_audit.py
```

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Application Metrics**:
   - Request rate and latency
   - Error rates
   - Queue depths
   - Processing throughput

2. **Infrastructure Metrics**:
   - CPU and memory utilization
   - Disk I/O and space
   - Network traffic
   - Pod restart counts

3. **Business Metrics**:
   - Transaction processing rate
   - Alert generation rate
   - Investigation resolution time
   - User activity

### Alert Configuration

Critical alerts to configure:

```yaml
groups:
- name: aegisshield.critical
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: High error rate detected

  - alert: DatabaseConnectionFailure
    expr: up{job="postgresql"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: Database connection failure

  - alert: HighMemoryUsage
    expr: container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.9
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High memory usage detected
```

## Maintenance Procedures

### 1. Regular Updates

```bash
# Update application
helm upgrade aegisshield infrastructure/helm/aegisshield \
  --namespace $NAMESPACE \
  --set global.imageTag=$NEW_VERSION

# Update dependencies
helm upgrade postgresql bitnami/postgresql --namespace $NAMESPACE
helm upgrade kafka bitnami/kafka --namespace $NAMESPACE
```

### 2. Scaling Operations

```bash
# Scale services
kubectl scale deployment aegisshield-api-gateway --replicas=5 -n $NAMESPACE
kubectl scale deployment aegisshield-data-ingestion --replicas=3 -n $NAMESPACE

# Auto-scaling configuration
kubectl apply -f infrastructure/k8s/hpa.yaml -n $NAMESPACE
```

### 3. Backup and Recovery

```bash
# Manual backup
kubectl exec -n $NAMESPACE deployment/postgresql -- \
  pg_dump -U aegisshield aegisshield > backup-$(date +%Y%m%d).sql

# Recovery
kubectl exec -i -n $NAMESPACE deployment/postgresql -- \
  psql -U aegisshield aegisshield < backup-20231201.sql
```

## Troubleshooting

### Common Issues

1. **Pod Startup Failures**:
   ```bash
   kubectl describe pod <pod-name> -n $NAMESPACE
   kubectl logs <pod-name> -n $NAMESPACE
   ```

2. **Database Connection Issues**:
   ```bash
   kubectl exec -it deployment/postgresql -n $NAMESPACE -- psql -U aegisshield
   ```

3. **Performance Issues**:
   ```bash
   kubectl top pods -n $NAMESPACE
   kubectl get hpa -n $NAMESPACE
   ```

### Support Contacts

- **Infrastructure Issues**: infrastructure-team@company.com
- **Application Issues**: dev-team@company.com
- **Security Issues**: security-team@company.com
- **Emergency**: Call escalation procedures

## Rollback Procedures

### Emergency Rollback

```bash
# Rollback to previous version
./scripts/production-deployment.sh rollback 1

# Or manual rollback
helm rollback aegisshield 1 --namespace $NAMESPACE
```

### Planned Rollback

1. Stop incoming traffic (maintenance mode)
2. Backup current state
3. Execute rollback
4. Validate functionality
5. Resume traffic

## Compliance and Auditing

### Required Documentation

- Deployment logs and timestamps
- Configuration changes
- Security scan results
- Performance test results
- Access logs and audit trails

### Compliance Checks

- SOC 2 Type II requirements
- PCI DSS compliance (if handling card data)
- GDPR compliance for EU data
- Local financial regulations

## Success Criteria

Deployment is considered successful when:

- [ ] All pods are running and healthy
- [ ] All health checks pass
- [ ] API endpoints respond correctly
- [ ] Database connections established
- [ ] Monitoring dashboards show green status
- [ ] Security scans pass
- [ ] Performance tests meet SLA requirements
- [ ] Backup procedures working
- [ ] DNS resolution working
- [ ] SSL certificates valid

## Next Steps

After successful deployment:

1. Conduct user acceptance testing
2. Schedule performance optimization review
3. Plan disaster recovery testing
4. Update runbooks and documentation
5. Schedule security audit
6. Train operations team
7. Implement monitoring alerting
8. Plan capacity scaling strategy

---

**Deployment Checklist**: Use this guide alongside the automated deployment script for a comprehensive production deployment of AegisShield.