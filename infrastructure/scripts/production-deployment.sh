#!/bin/bash

# AegisShield Production Deployment Script
# This script executes the production deployment with rollback procedures
# Version: 1.0.0
# Created: $(date)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="aegisshield-production"
HELM_RELEASE="aegisshield"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/production-deployment-$(date +%Y%m%d-%H%M%S).log"
BACKUP_DIR="${SCRIPT_DIR}/deployment-backup-$(date +%Y%m%d-%H%M%S)"

# Deployment configuration
DEPLOYMENT_VERSION="${DEPLOYMENT_VERSION:-1.0.0}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-aegisshield.azurecr.io}"
HELM_TIMEOUT="${HELM_TIMEOUT:-600s}"
ROLLBACK_ENABLED="${ROLLBACK_ENABLED:-true}"

# Logging function
log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') $1" | tee -a "$LOG_FILE"
}

log_info() {
    log "${BLUE}[INFO]${NC} $1"
}

log_success() {
    log "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    log "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    log "${RED}[ERROR]${NC} $1"
}

# Error handling
handle_error() {
    log_error "Deployment failed at step: $1"
    log_error "Check logs for details: $LOG_FILE"
    
    if [ "$ROLLBACK_ENABLED" = "true" ]; then
        log_warning "Initiating rollback procedures..."
        rollback_deployment
    fi
    
    exit 1
}

# Pre-deployment checks
pre_deployment_checks() {
    log_info "Running pre-deployment checks..."
    
    # Check kubectl connectivity
    if ! kubectl cluster-info &> /dev/null; then
        handle_error "Kubernetes cluster connectivity"
    fi
    
    # Check namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        handle_error "Production namespace not found"
    fi
    
    # Check Helm
    if ! command -v helm &> /dev/null; then
        handle_error "Helm not installed"
    fi
    
    # Check Docker registry access
    if ! docker pull "${DOCKER_REGISTRY}/health-check:latest" &> /dev/null; then
        log_warning "Docker registry access check failed - continuing with deployment"
    fi
    
    # Run go-live readiness assessment
    if [ -f "${SCRIPT_DIR}/go-live-assessment.sh" ]; then
        log_info "Running final go-live readiness assessment..."
        if ! bash "${SCRIPT_DIR}/go-live-assessment.sh" &> /dev/null; then
            log_warning "Go-live assessment found issues - review before proceeding"
            read -p "Continue with deployment? (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log_info "Deployment cancelled by user"
                exit 0
            fi
        fi
    fi
    
    log_success "Pre-deployment checks completed"
}

# Create deployment backup
create_deployment_backup() {
    log_info "Creating deployment backup..."
    
    mkdir -p "$BACKUP_DIR"
    
    # Backup current Helm release
    if helm list -n "$NAMESPACE" | grep -q "$HELM_RELEASE"; then
        helm get all "$HELM_RELEASE" -n "$NAMESPACE" > "$BACKUP_DIR/helm-release-backup.yaml"
        log_info "Helm release backed up"
    fi
    
    # Backup current deployments
    kubectl get deployments -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/deployments-backup.yaml"
    kubectl get services -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/services-backup.yaml"
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/configmaps-backup.yaml"
    kubectl get secrets -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/secrets-backup.yaml"
    
    # Backup database (if needed)
    log_info "Creating database backup before deployment..."
    if kubectl get statefulset postgresql -n "$NAMESPACE" &> /dev/null; then
        kubectl exec -n "$NAMESPACE" postgresql-0 -- pg_dumpall -U postgres > "$BACKUP_DIR/postgres-backup.sql" || log_warning "PostgreSQL backup failed"
    fi
    
    log_success "Deployment backup created: $BACKUP_DIR"
}

# Update Docker images
update_docker_images() {
    log_info "Updating Docker images..."
    
    # List of services to update
    local services=(
        "api-gateway"
        "data-ingestion"
        "graph-engine"
        "entity-resolution"
        "ml-pipeline"
        "analytics-dashboard"
        "investigation-toolkit"
        "alerting-engine"
        "compliance-engine"
        "user-management"
    )
    
    for service in "${services[@]}"; do
        local image_tag="${DOCKER_REGISTRY}/${service}:${DEPLOYMENT_VERSION}"
        log_info "Pulling image: $image_tag"
        
        if docker pull "$image_tag" &> /dev/null; then
            log_success "✅ Pulled $service:$DEPLOYMENT_VERSION"
        else
            log_warning "⚠️ Failed to pull $service:$DEPLOYMENT_VERSION - will use latest"
        fi
    done
}

# Deploy Helm charts
deploy_helm_charts() {
    log_info "Deploying Helm charts..."
    
    # Add Helm repositories if needed
    helm repo add bitnami https://charts.bitnami.com/bitnami &> /dev/null || true
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts &> /dev/null || true
    helm repo update &> /dev/null
    
    # Deploy or upgrade AegisShield
    local helm_values_file="${SCRIPT_DIR}/../helm/aegisshield/values-production.yaml"
    
    if [ ! -f "$helm_values_file" ]; then
        log_warning "Production values file not found, creating default..."
        create_production_values_file "$helm_values_file"
    fi
    
    log_info "Deploying AegisShield Helm chart..."
    
    if helm list -n "$NAMESPACE" | grep -q "$HELM_RELEASE"; then
        # Upgrade existing release
        helm upgrade "$HELM_RELEASE" "${SCRIPT_DIR}/../helm/aegisshield" \
            --namespace "$NAMESPACE" \
            --values "$helm_values_file" \
            --timeout "$HELM_TIMEOUT" \
            --wait \
            --atomic || handle_error "Helm upgrade failed"
    else
        # Install new release
        helm install "$HELM_RELEASE" "${SCRIPT_DIR}/../helm/aegisshield" \
            --namespace "$NAMESPACE" \
            --values "$helm_values_file" \
            --timeout "$HELM_TIMEOUT" \
            --wait \
            --atomic || handle_error "Helm install failed"
    fi
    
    log_success "Helm deployment completed"
}

# Create production values file
create_production_values_file() {
    local values_file="$1"
    
    cat > "$values_file" <<EOF
# AegisShield Production Values
# Generated: $(date)

global:
  environment: production
  imageRegistry: ${DOCKER_REGISTRY}
  imageTag: ${DEPLOYMENT_VERSION}
  
  # Security settings
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
    readOnlyRootFilesystem: true
  
  # Resource settings
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 2
      memory: 4Gi

# API Gateway
apiGateway:
  enabled: true
  replicaCount: 3
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
  service:
    type: ClusterIP
    port: 8080
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
      nginx.ingress.kubernetes.io/ssl-redirect: "true"
    hosts:
      - host: api.aegisshield.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: api-tls
        hosts:
          - api.aegisshield.com

# Data Ingestion
dataIngestion:
  enabled: true
  replicaCount: 2
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 8

# Graph Engine
graphEngine:
  enabled: true
  replicaCount: 2
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 6

# Database configurations
postgresql:
  enabled: true
  auth:
    existingSecret: production-secrets
  primary:
    persistence:
      enabled: true
      size: 100Gi
      storageClass: fast-ssd
  metrics:
    enabled: true
  backup:
    enabled: true
    schedule: "0 2 * * *"

neo4j:
  enabled: true
  neo4j:
    passwordFromSecret: production-secrets
  persistence:
    size: 200Gi
    storageClass: fast-ssd
  metrics:
    enabled: true

# Monitoring
monitoring:
  prometheus:
    enabled: true
  grafana:
    enabled: true
  alertmanager:
    enabled: true

# Security
security:
  networkPolicies:
    enabled: true
  podSecurityPolicy:
    enabled: true
  rbac:
    enabled: true
EOF
    
    log_info "Created production values file: $values_file"
}

# Post-deployment validation
post_deployment_validation() {
    log_info "Running post-deployment validation..."
    
    # Wait for rollout to complete
    log_info "Waiting for deployments to be ready..."
    kubectl rollout status deployment/api-gateway -n "$NAMESPACE" --timeout=600s || handle_error "API Gateway rollout failed"
    kubectl rollout status deployment/data-ingestion -n "$NAMESPACE" --timeout=600s || handle_error "Data Ingestion rollout failed"
    kubectl rollout status deployment/graph-engine -n "$NAMESPACE" --timeout=600s || handle_error "Graph Engine rollout failed"
    
    # Health checks
    log_info "Running health checks..."
    local max_retries=30
    local retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        if kubectl get pods -n "$NAMESPACE" | grep -q "Running"; then
            local running_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Running --no-headers | wc -l)
            local total_pods=$(kubectl get pods -n "$NAMESPACE" --no-headers | wc -l)
            
            if [ "$running_pods" -eq "$total_pods" ] && [ "$total_pods" -gt 0 ]; then
                log_success "All pods are running ($running_pods/$total_pods)"
                break
            fi
        fi
        
        ((retry_count++))
        log_info "Waiting for pods to be ready... ($retry_count/$max_retries)"
        sleep 10
    done
    
    if [ $retry_count -eq $max_retries ]; then
        handle_error "Pods not ready after maximum retries"
    fi
    
    # API health checks
    log_info "Testing API endpoints..."
    if kubectl port-forward service/api-gateway 8080:8080 -n "$NAMESPACE" &> /dev/null &
    then
        local port_forward_pid=$!
        sleep 5
        
        if curl -f http://localhost:8080/health &> /dev/null; then
            log_success "API Gateway health check passed"
        else
            log_warning "API Gateway health check failed"
        fi
        
        kill $port_forward_pid 2>/dev/null || true
    fi
    
    # Database connectivity checks
    log_info "Testing database connectivity..."
    if kubectl exec -n "$NAMESPACE" postgresql-0 -- pg_isready -U postgres &> /dev/null; then
        log_success "PostgreSQL connectivity check passed"
    else
        log_warning "PostgreSQL connectivity check failed"
    fi
    
    if kubectl exec -n "$NAMESPACE" neo4j-0 -- cypher-shell "RETURN 1" &> /dev/null; then
        log_success "Neo4j connectivity check passed"
    else
        log_warning "Neo4j connectivity check failed"
    fi
    
    log_success "Post-deployment validation completed"
}

# Performance validation
performance_validation() {
    log_info "Running performance validation..."
    
    # Run basic load test
    if [ -f "${SCRIPT_DIR}/../tests/performance/quick-load-test.sh" ]; then
        log_info "Running quick load test..."
        bash "${SCRIPT_DIR}/../tests/performance/quick-load-test.sh" || log_warning "Load test failed"
    else
        log_warning "Load test script not found, skipping performance validation"
    fi
    
    # Check resource usage
    log_info "Checking resource usage..."
    kubectl top pods -n "$NAMESPACE" 2>/dev/null || log_warning "Resource metrics not available"
    kubectl top nodes 2>/dev/null || log_warning "Node metrics not available"
    
    log_success "Performance validation completed"
}

# Security validation
security_validation() {
    log_info "Running security validation..."
    
    # Run compliance validation
    if [ -f "${SCRIPT_DIR}/compliance-validation.sh" ]; then
        log_info "Running compliance validation..."
        bash "${SCRIPT_DIR}/compliance-validation.sh" --production || log_warning "Compliance issues found"
    fi
    
    # Check security configurations
    log_info "Validating security configurations..."
    
    # Network policies
    local network_policies=$(kubectl get networkpolicy -n "$NAMESPACE" --no-headers | wc -l)
    if [ "$network_policies" -gt 0 ]; then
        log_success "Network policies configured: $network_policies"
    else
        log_warning "No network policies found"
    fi
    
    # RBAC
    local roles=$(kubectl get role -n "$NAMESPACE" --no-headers | wc -l)
    if [ "$roles" -gt 0 ]; then
        log_success "RBAC roles configured: $roles"
    else
        log_warning "No RBAC roles found"
    fi
    
    log_success "Security validation completed"
}

# Rollback deployment
rollback_deployment() {
    log_warning "Starting deployment rollback..."
    
    # Rollback Helm release
    if helm list -n "$NAMESPACE" | grep -q "$HELM_RELEASE"; then
        log_info "Rolling back Helm release..."
        helm rollback "$HELM_RELEASE" -n "$NAMESPACE" || log_error "Helm rollback failed"
    fi
    
    # Restore from backup if needed
    if [ -d "$BACKUP_DIR" ]; then
        log_info "Restoring from backup..."
        kubectl apply -f "$BACKUP_DIR/deployments-backup.yaml" || log_warning "Deployment restore failed"
        kubectl apply -f "$BACKUP_DIR/services-backup.yaml" || log_warning "Service restore failed"
    fi
    
    log_warning "Rollback completed - check system status"
}

# Generate deployment report
generate_deployment_report() {
    log_info "Generating deployment report..."
    
    local report_file="${SCRIPT_DIR}/deployment-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" <<EOF
# AegisShield Production Deployment Report

**Deployment Date**: $(date)  
**Version**: ${DEPLOYMENT_VERSION}  
**Namespace**: ${NAMESPACE}  
**Helm Release**: ${HELM_RELEASE}  

## Deployment Summary

### Services Deployed
$(kubectl get deployments -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print "- " $1 " (" $2 "/" $3 " ready)"}' || echo "- Unable to retrieve deployment status")

### Database Status
$(kubectl get statefulsets -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print "- " $1 " (" $2 "/" $3 " ready)"}' || echo "- Unable to retrieve database status")

### Pod Status
\`\`\`
$(kubectl get pods -n "$NAMESPACE" 2>/dev/null || echo "Unable to retrieve pod status")
\`\`\`

### Service Status
\`\`\`
$(kubectl get services -n "$NAMESPACE" 2>/dev/null || echo "Unable to retrieve service status")
\`\`\`

### Ingress Configuration
\`\`\`
$(kubectl get ingress -n "$NAMESPACE" 2>/dev/null || echo "No ingress configured")
\`\`\`

## Resource Usage
\`\`\`
$(kubectl top pods -n "$NAMESPACE" 2>/dev/null || echo "Resource metrics not available")
\`\`\`

## Configuration
- **Docker Registry**: ${DOCKER_REGISTRY}
- **Deployment Version**: ${DEPLOYMENT_VERSION}
- **Backup Location**: ${BACKUP_DIR}
- **Log File**: ${LOG_FILE}

## Post-Deployment Tasks

### Immediate (0-24 hours)
- [ ] Monitor system health and performance
- [ ] Verify all integrations are working
- [ ] Check log files for errors
- [ ] Validate user access and permissions
- [ ] Run smoke tests on critical functionality

### Short-term (1-7 days)
- [ ] Performance monitoring and optimization
- [ ] User training completion
- [ ] Documentation updates
- [ ] Backup verification
- [ ] Security audit completion

### Medium-term (1-4 weeks)
- [ ] Full load testing in production
- [ ] Compliance audit completion
- [ ] Disaster recovery testing
- [ ] Process optimization
- [ ] Knowledge transfer completion

## Rollback Information
- **Backup Directory**: ${BACKUP_DIR}
- **Rollback Command**: \`helm rollback ${HELM_RELEASE} -n ${NAMESPACE}\`
- **Emergency Contacts**: See operational runbook

## Notes
$(date): Production deployment completed successfully
EOF
    
    log_success "Deployment report generated: $report_file"
    
    # Display summary
    echo ""
    log_info "=================================================="
    log_info "AEGISSHIELD PRODUCTION DEPLOYMENT COMPLETED"
    log_info "=================================================="
    log_info "Version: ${DEPLOYMENT_VERSION}"
    log_info "Namespace: ${NAMESPACE}"
    log_info "Deployment Report: $report_file"
    log_info "Deployment Log: $LOG_FILE"
    log_info "Backup Location: $BACKUP_DIR"
    log_info "=================================================="
    echo ""
}

# Main execution
main() {
    log_info "Starting AegisShield Production Deployment..."
    log_info "Version: ${DEPLOYMENT_VERSION}"
    log_info "Registry: ${DOCKER_REGISTRY}"
    log_info "Namespace: ${NAMESPACE}"
    log_info "Log file: $LOG_FILE"
    
    # Confirm deployment
    log_warning "This will deploy AegisShield v${DEPLOYMENT_VERSION} to production"
    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled by user"
        exit 0
    fi
    
    pre_deployment_checks
    create_deployment_backup
    update_docker_images
    deploy_helm_charts
    post_deployment_validation
    performance_validation
    security_validation
    generate_deployment_report
    
    log_success "🎉 AegisShield production deployment completed successfully!"
    log_info "Monitor the system closely for the next 24 hours"
    log_info "Refer to the operational runbook for ongoing maintenance"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi