#!/usr/bin/env bash
"""
Production Deployment Procedures for AegisShield Platform
Comprehensive deployment automation and validation
"""

set -euo pipefail

# Configuration
DEPLOYMENT_ENV="${DEPLOYMENT_ENV:-production}"
NAMESPACE="${NAMESPACE:-aegisshield-prod}"
REGION="${REGION:-us-west-2}"
CLUSTER_NAME="${CLUSTER_NAME:-aegisshield-production}"
DOMAIN_NAME="${DOMAIN_NAME:-aegisshield.company.com}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
}

# Prerequisites check
check_prerequisites() {
    log "Checking deployment prerequisites..."
    
    local missing_tools=()
    
    # Check required tools
    for tool in kubectl helm docker terraform aws; do
        if ! command -v $tool &> /dev/null; then
            missing_tools+=($tool)
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    # Check Kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check Helm repositories
    if ! helm repo list | grep -q bitnami; then
        warning "Adding required Helm repositories..."
        helm repo add bitnami https://charts.bitnami.com/bitnami
        helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
        helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
        helm repo update
    fi
    
    success "Prerequisites check completed"
}

# Infrastructure deployment
deploy_infrastructure() {
    log "Deploying infrastructure components..."
    
    # Create namespace
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    # Deploy PostgreSQL
    log "Deploying PostgreSQL..."
    helm upgrade --install postgresql bitnami/postgresql \
        --namespace $NAMESPACE \
        --set auth.postgresPassword="$(openssl rand -base64 32)" \
        --set auth.database=aegisshield \
        --set primary.persistence.size=100Gi \
        --set primary.resources.requests.memory=2Gi \
        --set primary.resources.requests.cpu=1000m \
        --set primary.resources.limits.memory=4Gi \
        --set primary.resources.limits.cpu=2000m \
        --set metrics.enabled=true \
        --wait
    
    # Deploy Neo4j
    log "Deploying Neo4j..."
    kubectl apply -f infrastructure/k8s/neo4j.yaml -n $NAMESPACE
    
    # Deploy Kafka
    log "Deploying Kafka..."
    helm upgrade --install kafka bitnami/kafka \
        --namespace $NAMESPACE \
        --set replicaCount=3 \
        --set persistence.size=50Gi \
        --set resources.requests.memory=1Gi \
        --set resources.requests.cpu=500m \
        --set resources.limits.memory=2Gi \
        --set resources.limits.cpu=1000m \
        --set metrics.kafka.enabled=true \
        --set metrics.jmx.enabled=true \
        --wait
    
    # Deploy HashiCorp Vault
    log "Deploying Vault..."
    kubectl apply -f infrastructure/k8s/vault.yaml -n $NAMESPACE
    
    # Deploy monitoring stack
    log "Deploying monitoring stack..."
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace $NAMESPACE \
        --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi \
        --set grafana.persistence.enabled=true \
        --set grafana.persistence.size=10Gi \
        --set alertmanager.alertmanagerSpec.storage.volumeClaimTemplate.spec.resources.requests.storage=10Gi \
        --wait
    
    helm upgrade --install jaeger jaegertracing/jaeger \
        --namespace $NAMESPACE \
        --set storage.type=elasticsearch \
        --wait
    
    success "Infrastructure deployment completed"
}

# Application services deployment
deploy_services() {
    log "Deploying application services..."
    
    # Build and push container images (if not already done)
    if [ "${SKIP_BUILD:-false}" != "true" ]; then
        log "Building and pushing container images..."
        make build-all
        make push-all
    fi
    
    # Deploy services using Helm chart
    log "Deploying AegisShield services..."
    helm upgrade --install aegisshield infrastructure/helm/aegisshield \
        --namespace $NAMESPACE \
        --set global.environment=production \
        --set global.imageRegistry=${IMAGE_REGISTRY:-docker.io/aegisshield} \
        --set global.imageTag=${IMAGE_TAG:-latest} \
        --set global.domain=$DOMAIN_NAME \
        --set ingress.enabled=true \
        --set ingress.className=nginx \
        --set ingress.tls.enabled=true \
        --set autoscaling.enabled=true \
        --set resources.production=true \
        --wait --timeout=10m
    
    success "Services deployment completed"
}

# Database initialization
initialize_databases() {
    log "Initializing databases..."
    
    # Wait for databases to be ready
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql --timeout=300s -n $NAMESPACE
    kubectl wait --for=condition=ready pod -l app=neo4j --timeout=300s -n $NAMESPACE
    
    # Run database migrations
    log "Running database migrations..."
    
    # PostgreSQL migrations
    for service in data-ingestion entity-resolution alerting-engine investigation-toolkit; do
        log "Running migrations for $service..."
        kubectl exec -n $NAMESPACE deployment/aegisshield-$service -- \
            ./migrate -path ./migrations -database "postgres://aegisshield:$(kubectl get secret postgresql -n $NAMESPACE -o jsonpath='{.data.postgres-password}' | base64 -d)@postgresql:5432/aegisshield?sslmode=disable" up
    done
    
    # Neo4j initialization
    log "Initializing Neo4j schema..."
    kubectl exec -n $NAMESPACE deployment/aegisshield-graph-engine -- \
        ./init-schema
    
    success "Database initialization completed"
}

# Security configuration
configure_security() {
    log "Configuring security settings..."
    
    # Configure network policies
    kubectl apply -f infrastructure/k8s/network-policies.yaml -n $NAMESPACE
    
    # Configure RBAC
    kubectl apply -f infrastructure/k8s/rbac.yaml -n $NAMESPACE
    
    # Configure Pod Security Standards
    kubectl label namespace $NAMESPACE \
        pod-security.kubernetes.io/enforce=restricted \
        pod-security.kubernetes.io/audit=restricted \
        pod-security.kubernetes.io/warn=restricted
    
    # Configure secrets
    log "Configuring secrets..."
    
    # Generate JWT signing key
    JWT_SECRET=$(openssl rand -base64 64)
    kubectl create secret generic jwt-secret \
        --from-literal=jwt-signing-key="$JWT_SECRET" \
        -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    # Configure TLS certificates
    if [ "${SETUP_TLS:-true}" = "true" ]; then
        log "Setting up TLS certificates..."
        kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: aegisshield-tls
  namespace: $NAMESPACE
spec:
  secretName: aegisshield-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - $DOMAIN_NAME
  - api.$DOMAIN_NAME
EOF
    fi
    
    success "Security configuration completed"
}

# Performance optimization
optimize_performance() {
    log "Applying performance optimizations..."
    
    # Configure resource quotas
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ResourceQuota
metadata:
  name: aegisshield-quota
  namespace: $NAMESPACE
spec:
  hard:
    requests.cpu: "20"
    requests.memory: 40Gi
    limits.cpu: "40"
    limits.memory: 80Gi
    persistentvolumeclaims: "20"
EOF
    
    # Configure horizontal pod autoscaling
    kubectl apply -f infrastructure/k8s/hpa.yaml -n $NAMESPACE
    
    # Configure pod disruption budgets
    kubectl apply -f infrastructure/k8s/pdb.yaml -n $NAMESPACE
    
    success "Performance optimization completed"
}

# Health checks and validation
validate_deployment() {
    log "Validating deployment..."
    
    # Check pod status
    log "Checking pod status..."
    if ! kubectl get pods -n $NAMESPACE | grep -v "Running\|Completed"; then
        success "All pods are running successfully"
    else
        error "Some pods are not in running state"
        kubectl get pods -n $NAMESPACE
        return 1
    fi
    
    # Check service endpoints
    log "Checking service endpoints..."
    services=(
        "aegisshield-api-gateway"
        "aegisshield-data-ingestion"
        "aegisshield-entity-resolution"
        "aegisshield-graph-engine"
        "aegisshield-alerting-engine"
    )
    
    for service in "${services[@]}"; do
        if kubectl get endpoints $service -n $NAMESPACE -o jsonpath='{.subsets[0].addresses}' | grep -q "ip"; then
            success "$service endpoint is ready"
        else
            error "$service endpoint is not ready"
            return 1
        fi
    done
    
    # Health check API endpoints
    log "Performing API health checks..."
    
    # Get ingress IP
    INGRESS_IP=$(kubectl get ingress aegisshield-ingress -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    
    if [ -z "$INGRESS_IP" ]; then
        warning "Ingress IP not yet available, using port-forward for testing"
        kubectl port-forward service/aegisshield-api-gateway 8080:80 -n $NAMESPACE &
        PF_PID=$!
        sleep 5
        API_URL="http://localhost:8080"
    else
        API_URL="https://$DOMAIN_NAME"
    fi
    
    # Test API endpoints
    if curl -f "$API_URL/health" > /dev/null 2>&1; then
        success "API health check passed"
    else
        error "API health check failed"
        return 1
    fi
    
    if curl -f "$API_URL/api/v1/status" > /dev/null 2>&1; then
        success "API status check passed"
    else
        error "API status check failed"
        return 1
    fi
    
    # Clean up port-forward if used
    if [ ! -z "${PF_PID:-}" ]; then
        kill $PF_PID
    fi
    
    success "Deployment validation completed"
}

# Monitoring setup
setup_monitoring() {
    log "Setting up monitoring and alerting..."
    
    # Configure Grafana dashboards
    kubectl apply -f infrastructure/k8s/monitoring/grafana-dashboards.yaml -n $NAMESPACE
    
    # Configure alerting rules
    kubectl apply -f infrastructure/k8s/monitoring/alert-rules.yaml -n $NAMESPACE
    
    # Configure notification channels
    if [ ! -z "${SLACK_WEBHOOK_URL:-}" ]; then
        kubectl create secret generic slack-webhook \
            --from-literal=url="$SLACK_WEBHOOK_URL" \
            -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    if [ ! -z "${PAGERDUTY_KEY:-}" ]; then
        kubectl create secret generic pagerduty-key \
            --from-literal=key="$PAGERDUTY_KEY" \
            -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    success "Monitoring setup completed"
}

# Backup configuration
setup_backup() {
    log "Setting up backup procedures..."
    
    # Configure database backups
    kubectl apply -f infrastructure/k8s/backup/backup-cronjobs.yaml -n $NAMESPACE
    
    # Configure volume snapshots
    kubectl apply -f infrastructure/k8s/backup/volume-snapshots.yaml -n $NAMESPACE
    
    success "Backup configuration completed"
}

# Generate deployment report
generate_deployment_report() {
    log "Generating deployment report..."
    
    REPORT_FILE="deployment-report-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > $REPORT_FILE <<EOF
AegisShield Production Deployment Report
========================================
Deployment Date: $(date)
Environment: $DEPLOYMENT_ENV
Namespace: $NAMESPACE
Cluster: $CLUSTER_NAME
Domain: $DOMAIN_NAME

Services Deployed:
------------------
$(kubectl get deployments -n $NAMESPACE --no-headers | awk '{print $1 " - " $2 "/" $3 " replicas ready"}')

Ingress Status:
---------------
$(kubectl get ingress -n $NAMESPACE)

Resource Usage:
---------------
$(kubectl top nodes)

Pod Status:
-----------
$(kubectl get pods -n $NAMESPACE)

Service Endpoints:
------------------
$(kubectl get endpoints -n $NAMESPACE)

Persistent Volumes:
-------------------
$(kubectl get pv | grep $NAMESPACE)

Monitoring URLs:
----------------
Grafana: https://grafana.$DOMAIN_NAME
Prometheus: https://prometheus.$DOMAIN_NAME
Jaeger: https://jaeger.$DOMAIN_NAME

Next Steps:
-----------
1. Configure DNS to point $DOMAIN_NAME to $(kubectl get ingress aegisshield-ingress -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
2. Run smoke tests to validate functionality
3. Configure monitoring alerts
4. Set up automated backups
5. Conduct security audit
6. Update documentation

Deployment Completed Successfully!
EOF

    success "Deployment report generated: $REPORT_FILE"
}

# Rollback function
rollback_deployment() {
    local version=${1:-}
    
    if [ -z "$version" ]; then
        error "Please specify version to rollback to"
        exit 1
    fi
    
    warning "Rolling back to version: $version"
    
    helm rollback aegisshield $version --namespace $NAMESPACE
    
    success "Rollback completed"
}

# Main deployment orchestration
main() {
    local action=${1:-deploy}
    
    case $action in
        "deploy")
            log "Starting AegisShield production deployment..."
            check_prerequisites
            deploy_infrastructure
            deploy_services
            initialize_databases
            configure_security
            optimize_performance
            setup_monitoring
            setup_backup
            validate_deployment
            generate_deployment_report
            success "🚀 AegisShield production deployment completed successfully!"
            ;;
        "validate")
            log "Validating existing deployment..."
            validate_deployment
            ;;
        "rollback")
            rollback_deployment $2
            ;;
        "status")
            kubectl get all -n $NAMESPACE
            ;;
        "logs")
            kubectl logs -f deployment/aegisshield-api-gateway -n $NAMESPACE
            ;;
        *)
            echo "Usage: $0 {deploy|validate|rollback|status|logs}"
            echo "  deploy   - Full production deployment"
            echo "  validate - Validate existing deployment"
            echo "  rollback - Rollback to previous version"
            echo "  status   - Show deployment status"
            echo "  logs     - Show API gateway logs"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"