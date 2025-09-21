#!/bin/bash

# AegisShield Monitoring Stack Deployment Script
# Deploys Prometheus, Grafana, and Alertmanager for comprehensive monitoring

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_success "kubectl is available and connected to cluster"
}

# Create namespace if it doesn't exist
create_namespace() {
    log_info "Creating monitoring namespace..."
    
    if kubectl get namespace monitoring &> /dev/null; then
        log_warning "Monitoring namespace already exists"
    else
        kubectl create namespace monitoring
        log_success "Created monitoring namespace"
    fi
}

# Deploy Prometheus
deploy_prometheus() {
    log_info "Deploying Prometheus..."
    
    # Apply ConfigMaps first
    log_info "Applying Prometheus configuration..."
    kubectl apply -f prometheus-config.yaml
    kubectl apply -f prometheus-rules.yaml
    
    # Wait for ConfigMaps to be ready
    kubectl wait --for=condition=Ready configmap/prometheus-config -n monitoring --timeout=30s
    kubectl wait --for=condition=Ready configmap/prometheus-rules -n monitoring --timeout=30s
    
    # Deploy Prometheus
    log_info "Deploying Prometheus server..."
    kubectl apply -f prometheus-deployment.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=Available deployment/prometheus -n monitoring --timeout=300s
    
    log_success "Prometheus deployed successfully"
}

# Deploy Grafana
deploy_grafana() {
    log_info "Deploying Grafana..."
    
    # Apply Grafana dashboards ConfigMap first
    log_info "Applying Grafana dashboards..."
    kubectl apply -f grafana-dashboards.yaml
    
    # Wait for ConfigMap to be ready
    kubectl wait --for=condition=Ready configmap/grafana-dashboards -n monitoring --timeout=30s
    
    # Deploy Grafana
    log_info "Deploying Grafana server..."
    kubectl apply -f grafana-deployment.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=Available deployment/grafana -n monitoring --timeout=300s
    
    log_success "Grafana deployed successfully"
}

# Deploy Alertmanager
deploy_alertmanager() {
    log_info "Deploying Alertmanager..."
    
    # Apply Alertmanager configuration first
    log_info "Applying Alertmanager configuration..."
    kubectl apply -f alertmanager-config.yaml
    
    # Wait for ConfigMap to be ready
    kubectl wait --for=condition=Ready configmap/alertmanager-config -n monitoring --timeout=30s
    
    # Deploy Alertmanager
    log_info "Deploying Alertmanager server..."
    kubectl apply -f alertmanager-deployment.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=Available deployment/alertmanager -n monitoring --timeout=300s
    
    log_success "Alertmanager deployed successfully"
}

# Verify deployments
verify_deployments() {
    log_info "Verifying monitoring stack deployment..."
    
    # Check pod status
    log_info "Checking pod status..."
    kubectl get pods -n monitoring
    
    # Check service status
    log_info "Checking service status..."
    kubectl get services -n monitoring
    
    # Check if all deployments are ready
    local deployments=("prometheus" "grafana" "alertmanager")
    for deployment in "${deployments[@]}"; do
        if kubectl get deployment "$deployment" -n monitoring &> /dev/null; then
            ready=$(kubectl get deployment "$deployment" -n monitoring -o jsonpath='{.status.readyReplicas}')
            desired=$(kubectl get deployment "$deployment" -n monitoring -o jsonpath='{.spec.replicas}')
            
            if [ "$ready" == "$desired" ]; then
                log_success "$deployment is ready ($ready/$desired replicas)"
            else
                log_error "$deployment is not ready ($ready/$desired replicas)"
            fi
        else
            log_error "$deployment not found"
        fi
    done
}

# Get access information
get_access_info() {
    log_info "Getting access information..."
    
    echo
    echo "===== MONITORING STACK ACCESS INFORMATION ====="
    echo
    
    # Prometheus
    echo "📊 PROMETHEUS:"
    prometheus_port=$(kubectl get svc prometheus -n monitoring -o jsonpath='{.spec.ports[0].port}')
    echo "   Internal URL: http://prometheus.monitoring.svc.cluster.local:$prometheus_port"
    echo "   Port Forward: kubectl port-forward svc/prometheus -n monitoring $prometheus_port:$prometheus_port"
    echo "   Then access: http://localhost:$prometheus_port"
    echo
    
    # Grafana
    echo "📈 GRAFANA:"
    grafana_port=$(kubectl get svc grafana -n monitoring -o jsonpath='{.spec.ports[0].port}')
    echo "   Internal URL: http://grafana.monitoring.svc.cluster.local:$grafana_port"
    echo "   Port Forward: kubectl port-forward svc/grafana -n monitoring $grafana_port:$grafana_port"
    echo "   Then access: http://localhost:$grafana_port"
    echo "   Default Credentials: admin / AegisShield2025!"
    echo
    
    # Alertmanager
    echo "🚨 ALERTMANAGER:"
    alertmanager_port=$(kubectl get svc alertmanager -n monitoring -o jsonpath='{.spec.ports[0].port}')
    echo "   Internal URL: http://alertmanager.monitoring.svc.cluster.local:$alertmanager_port"
    echo "   Port Forward: kubectl port-forward svc/alertmanager -n monitoring $alertmanager_port:$alertmanager_port"
    echo "   Then access: http://localhost:$alertmanager_port"
    echo
    
    echo "================================================="
    echo
}

# Test monitoring endpoints
test_endpoints() {
    log_info "Testing monitoring endpoints..."
    
    # Test if services are responding
    local services=("prometheus:9090" "grafana:3000" "alertmanager:9093")
    
    for service in "${services[@]}"; do
        service_name=$(echo "$service" | cut -d':' -f1)
        service_port=$(echo "$service" | cut -d':' -f2)
        
        log_info "Testing $service_name endpoint..."
        
        # Use kubectl to test the endpoint
        if kubectl run test-curl-$service_name --rm -i --restart=Never --image=curlimages/curl:latest -n monitoring -- curl -s -o /dev/null -w "%{http_code}" "http://$service_name:$service_port" &> /dev/null; then
            log_success "$service_name is responding"
        else
            log_warning "$service_name may not be ready yet (this is normal during initial startup)"
        fi
    done
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test pods..."
    kubectl delete pod --field-selector=status.phase=Succeeded -n monitoring &> /dev/null || true
}

# Main deployment function
main() {
    log_info "Starting AegisShield Monitoring Stack Deployment"
    echo
    
    # Pre-deployment checks
    check_kubectl
    
    # Create namespace
    create_namespace
    
    # Deploy components in order
    deploy_prometheus
    echo
    deploy_grafana
    echo
    deploy_alertmanager
    echo
    
    # Verify deployment
    verify_deployments
    echo
    
    # Test endpoints
    test_endpoints
    echo
    
    # Get access information
    get_access_info
    
    # Cleanup
    cleanup
    
    log_success "AegisShield Monitoring Stack deployment completed!"
    echo
    log_info "Next steps:"
    echo "  1. Port forward to access services locally"
    echo "  2. Configure notification channels in Alertmanager"
    echo "  3. Import additional dashboards in Grafana"
    echo "  4. Set up backup procedures for monitoring data"
}

# Run main function
main "$@"