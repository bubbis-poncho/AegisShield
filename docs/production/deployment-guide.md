# AegisShield Production Deployment Guide

This document provides step-by-step instructions for executing the AegisShield production deployment with rollback procedures and post-deployment validation.

## Deployment Overview

**Project**: AegisShield Financial Crime Investigation Platform  
**Version**: 1.0.0  
**Target Environment**: Production Kubernetes Cluster  
**Deployment Method**: Helm Charts with Blue-Green Strategy  
**Estimated Duration**: 2-4 hours  

## Pre-Deployment Checklist

### ✅ Prerequisites Validation
- [ ] Go-live readiness assessment completed and approved
- [ ] All stakeholder sign-offs obtained
- [ ] Production environment configured and hardened
- [ ] Security validation completed
- [ ] Compliance requirements validated
- [ ] Performance benchmarks established
- [ ] Disaster recovery procedures tested
- [ ] Rollback procedures documented and tested

### ✅ Technical Prerequisites
- [ ] Kubernetes cluster v1.24+ operational
- [ ] Helm v3.10+ installed and configured
- [ ] Docker registry access verified
- [ ] Database clusters operational (PostgreSQL, Neo4j)
- [ ] Monitoring stack deployed (Prometheus, Grafana)
- [ ] Backup systems operational
- [ ] Network security policies applied
- [ ] SSL certificates provisioned

### ✅ Team Readiness
- [ ] Deployment team on standby
- [ ] Operations team ready for monitoring
- [ ] Security team ready for validation
- [ ] Business stakeholders notified
- [ ] Emergency contacts confirmed
- [ ] Communication channels established

## Deployment Phases

### Phase 1: Pre-Deployment Preparation (30 minutes)

#### 1.1 Final System Validation
```bash
# Run go-live readiness assessment
./infrastructure/scripts/go-live-assessment.sh

# Verify cluster status
kubectl cluster-info
kubectl get nodes
kubectl top nodes

# Check namespace readiness
kubectl get all -n aegisshield-production
```

#### 1.2 Backup Current State
```bash
# Create deployment backup
mkdir -p deployment-backup-$(date +%Y%m%d-%H%M%S)

# Backup Helm releases
helm list -A > deployment-backup/helm-releases.txt

# Backup Kubernetes resources
kubectl get all -n aegisshield-production -o yaml > deployment-backup/k8s-resources.yaml

# Backup databases
kubectl exec -n aegisshield-production postgresql-0 -- pg_dumpall > deployment-backup/postgres-backup.sql
```

#### 1.3 Environment Configuration
```bash
# Set deployment environment variables
export DEPLOYMENT_VERSION="1.0.0"
export DOCKER_REGISTRY="aegisshield.azurecr.io"
export NAMESPACE="aegisshield-production"
export HELM_RELEASE="aegisshield"

# Verify environment
echo "Deploying version $DEPLOYMENT_VERSION to $NAMESPACE"
```

### Phase 2: Infrastructure Deployment (45 minutes)

#### 2.1 Update Production Configuration
```bash
# Apply production Kubernetes configuration
kubectl apply -f infrastructure/k8s/production/production-config.yaml

# Verify network policies
kubectl get networkpolicy -n aegisshield-production

# Check resource quotas
kubectl describe resourcequota -n aegisshield-production
```

#### 2.2 Deploy Database Layer
```bash
# Deploy PostgreSQL
helm upgrade --install postgresql bitnami/postgresql \
  --namespace aegisshield-production \
  --values infrastructure/helm/postgresql/values-production.yaml \
  --wait

# Deploy Neo4j
helm upgrade --install neo4j neo4j/neo4j \
  --namespace aegisshield-production \
  --values infrastructure/helm/neo4j/values-production.yaml \
  --wait

# Verify database deployment
kubectl get statefulsets -n aegisshield-production
kubectl get pvc -n aegisshield-production
```

#### 2.3 Deploy Supporting Services
```bash
# Deploy Redis cache
helm upgrade --install redis bitnami/redis \
  --namespace aegisshield-production \
  --values infrastructure/helm/redis/values-production.yaml \
  --wait

# Deploy HashiCorp Vault
helm upgrade --install vault hashicorp/vault \
  --namespace aegisshield-production \
  --values infrastructure/helm/vault/values-production.yaml \
  --wait

# Verify supporting services
kubectl get pods -n aegisshield-production
```

### Phase 3: Application Deployment (60 minutes)

#### 3.1 Pull and Verify Images
```bash
# Pull all application images
services=("api-gateway" "data-ingestion" "graph-engine" "entity-resolution" "ml-pipeline" "analytics-dashboard" "investigation-toolkit" "alerting-engine" "compliance-engine" "user-management")

for service in "${services[@]}"; do
  docker pull ${DOCKER_REGISTRY}/${service}:${DEPLOYMENT_VERSION}
  echo "✅ Pulled ${service}:${DEPLOYMENT_VERSION}"
done
```

#### 3.2 Deploy AegisShield Application
```bash
# Execute production deployment script
chmod +x infrastructure/scripts/production-deployment.sh
./infrastructure/scripts/production-deployment.sh

# Monitor deployment progress
watch kubectl get pods -n aegisshield-production
```

#### 3.3 Verify Application Deployment
```bash
# Check deployment status
kubectl rollout status deployment/api-gateway -n aegisshield-production
kubectl rollout status deployment/data-ingestion -n aegisshield-production
kubectl rollout status deployment/graph-engine -n aegisshield-production

# Verify all pods are running
kubectl get pods -n aegisshield-production | grep Running
```

### Phase 4: Service Integration (30 minutes)

#### 4.1 Configure Load Balancer
```bash
# Deploy ingress configuration
kubectl apply -f infrastructure/k8s/production/ingress-production.yaml

# Verify ingress
kubectl get ingress -n aegisshield-production
kubectl describe ingress aegisshield-ingress -n aegisshield-production
```

#### 4.2 SSL Certificate Setup
```bash
# Verify certificate issuer
kubectl get clusterissuer

# Check certificate status
kubectl get certificate -n aegisshield-production
kubectl describe certificate api-tls -n aegisshield-production
```

#### 4.3 DNS Configuration
```bash
# Verify DNS records
nslookup api.aegisshield.com
nslookup app.aegisshield.com

# Test external connectivity
curl -I https://api.aegisshield.com/health
```

### Phase 5: Post-Deployment Validation (45 minutes)

#### 5.1 Health Checks
```bash
# API Gateway health check
curl -f https://api.aegisshield.com/health

# Service-specific health checks
curl -f https://api.aegisshield.com/api/v1/data-ingestion/health
curl -f https://api.aegisshield.com/api/v1/graph-engine/health
curl -f https://api.aegisshield.com/api/v1/entity-resolution/health
```

#### 5.2 Database Connectivity
```bash
# PostgreSQL connectivity
kubectl exec -n aegisshield-production postgresql-0 -- pg_isready -U postgres

# Neo4j connectivity
kubectl exec -n aegisshield-production neo4j-0 -- cypher-shell "RETURN 1"

# Redis connectivity
kubectl exec -n aegisshield-production redis-0 -- redis-cli ping
```

#### 5.3 Performance Validation
```bash
# Run performance tests
./tests/performance/production-load-test.sh

# Monitor resource usage
kubectl top pods -n aegisshield-production
kubectl top nodes
```

#### 5.4 Security Validation
```bash
# Run security validation
./infrastructure/scripts/compliance-validation.sh --production

# Check security policies
kubectl get networkpolicy -n aegisshield-production
kubectl get psp
kubectl auth can-i create pods --as=system:serviceaccount:aegisshield-production:default
```

#### 5.5 Functional Testing
```bash
# Run smoke tests
./tests/e2e/production-smoke-tests.sh

# Verify user authentication
curl -X POST https://api.aegisshield.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"test123"}'

# Test data ingestion
curl -X POST https://api.aegisshield.com/api/v1/data-ingestion/transaction \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"transaction_id":"test123","amount":1000}'
```

## Monitoring and Alerting

### Deployment Monitoring
```bash
# Monitor deployment metrics
kubectl get events -n aegisshield-production --sort-by='.lastTimestamp'

# Check application logs
kubectl logs -f deployment/api-gateway -n aegisshield-production
kubectl logs -f deployment/data-ingestion -n aegisshield-production

# Monitor resource usage
watch kubectl top pods -n aegisshield-production
```

### Alert Configuration
- **Critical Alerts**: Page operations team immediately
- **Warning Alerts**: Send Slack notifications
- **Info Alerts**: Log to monitoring dashboard

### Key Metrics to Monitor
- **Availability**: Service uptime > 99.9%
- **Performance**: API response time < 2 seconds
- **Error Rate**: < 0.1% error rate
- **Resource Usage**: CPU < 80%, Memory < 85%
- **Database Performance**: Query time < 500ms

## Rollback Procedures

### Automatic Rollback Triggers
- Pod failure rate > 50%
- API error rate > 5%
- Database connectivity loss
- Critical security vulnerability
- Performance degradation > 50%

### Manual Rollback Steps

#### 1. Immediate Rollback (5 minutes)
```bash
# Rollback Helm release
helm rollback aegisshield -n aegisshield-production

# Verify rollback
kubectl get pods -n aegisshield-production
kubectl rollout status deployment/api-gateway -n aegisshield-production
```

#### 2. Database Rollback (15 minutes)
```bash
# Restore database backup if needed
kubectl exec -n aegisshield-production postgresql-0 -- psql -U postgres < deployment-backup/postgres-backup.sql

# Verify database integrity
kubectl exec -n aegisshield-production postgresql-0 -- psql -U postgres -c "SELECT COUNT(*) FROM transactions;"
```

#### 3. Complete Environment Rollback (30 minutes)
```bash
# Restore Kubernetes resources
kubectl apply -f deployment-backup/k8s-resources.yaml

# Restart all services
kubectl rollout restart deployment -n aegisshield-production

# Verify system health
./infrastructure/scripts/go-live-assessment.sh
```

## Post-Deployment Tasks

### Immediate (0-4 hours)
- [ ] Monitor system health continuously
- [ ] Verify all integrations working
- [ ] Check error logs and resolve issues
- [ ] Validate user access and authentication
- [ ] Run comprehensive smoke tests
- [ ] Update monitoring dashboards
- [ ] Send deployment completion notification

### Short-term (4-24 hours)
- [ ] Monitor performance metrics
- [ ] Review application logs
- [ ] Validate backup procedures
- [ ] Test disaster recovery capabilities
- [ ] User acceptance testing
- [ ] Documentation updates
- [ ] Team knowledge transfer

### Medium-term (1-7 days)
- [ ] Performance optimization based on usage
- [ ] Security audit completion
- [ ] Compliance validation
- [ ] User training completion
- [ ] Process refinement
- [ ] Capacity planning review
- [ ] Operational handover

## Emergency Procedures

### War Room Setup
- **Location**: Conference Room A / Virtual Meeting
- **Duration**: First 24 hours post-deployment
- **Participants**: 
  - Deployment Manager
  - Lead Developer
  - DevOps Engineer
  - Security Engineer
  - Database Administrator
  - Business Stakeholder

### Emergency Contacts
- **Deployment Manager**: +1-555-DEPLOY-1
- **CTO**: +1-555-CTO-HELP
- **Security Lead**: +1-555-SEC-HELP
- **Database Admin**: +1-555-DBA-HELP
- **24/7 Support**: +1-555-SUPPORT

### Escalation Matrix
1. **Level 1**: Operations team (0-15 minutes)
2. **Level 2**: Engineering team (15-30 minutes)
3. **Level 3**: Management team (30-60 minutes)
4. **Level 4**: Executive team (60+ minutes)

## Success Criteria

### Technical Success
- [ ] All services deployed and running
- [ ] Health checks passing
- [ ] Performance benchmarks met
- [ ] Security validations passed
- [ ] Monitoring active and alerting
- [ ] Backup procedures verified

### Business Success
- [ ] User authentication working
- [ ] Core functionality operational
- [ ] Data ingestion processing
- [ ] Investigation workflows functional
- [ ] Reporting capabilities active
- [ ] Compliance requirements met

### Operational Success
- [ ] Support team trained and ready
- [ ] Documentation completed
- [ ] Monitoring procedures established
- [ ] Incident response tested
- [ ] Change management process active
- [ ] Continuous improvement plan defined

## Documentation Links

- [Production Environment Configuration](./environment-configuration.md)
- [Go-Live Readiness Assessment](./go-live-readiness-assessment.md)
- [Operational Runbook](./operational-runbook.md)
- [Disaster Recovery Plan](./disaster-recovery-plan.md)
- [Security Audit Report](./security-audit-report.md)
- [Compliance Validation Report](./compliance-validation-report.md)

## Conclusion

The AegisShield production deployment represents the culmination of comprehensive development, testing, and validation efforts. Following this deployment guide ensures a systematic, secure, and reliable transition to production operations.

**Remember**: Production deployment is not the end goal, but the beginning of operational excellence. Continuous monitoring, optimization, and improvement are essential for long-term success.

---

**Deployment Team Lead**: _______________  
**Date**: _______________  
**Approval**: _______________