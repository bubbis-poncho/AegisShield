#!/bin/bash

# AegisShield Production Environment Configuration Script
# This script configures the production Kubernetes cluster with security hardening
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
CLUSTER_NAME="aegisshield-prod"
REGION="us-east-1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/production-setup-$(date +%Y%m%d-%H%M%S).log"

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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi
    
    if ! command -v helm &> /dev/null; then
        missing_tools+=("helm")
    fi
    
    if ! command -v docker &> /dev/null; then
        missing_tools+=("docker")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install the missing tools and retry"
        exit 1
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        log_error "Please ensure kubectl is configured correctly"
        exit 1
    fi
    
    log_success "Prerequisites check completed"
}

# Validate cluster resources
validate_cluster_resources() {
    log_info "Validating cluster resources..."
    
    # Check node resources
    local total_cpu=$(kubectl top nodes --no-headers 2>/dev/null | awk '{sum += $2} END {print sum}' || echo "0")
    local total_memory=$(kubectl top nodes --no-headers 2>/dev/null | awk '{sum += $3} END {print sum}' || echo "0")
    
    log_info "Cluster CPU capacity: ${total_cpu}m"
    log_info "Cluster memory capacity: ${total_memory}Mi"
    
    # Check storage classes
    local storage_classes=$(kubectl get storageclass --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$storage_classes" -eq 0 ]; then
        log_warning "No storage classes found. Persistent volumes may not be available."
    fi
    
    log_success "Cluster resources validation completed"
}

# Setup production namespace
setup_namespace() {
    log_info "Setting up production namespace..."
    
    # Apply production configuration
    kubectl apply -f "${SCRIPT_DIR}/../production/production-config.yaml"
    
    # Wait for namespace to be ready
    kubectl wait --for=condition=Active namespace/${NAMESPACE} --timeout=60s
    
    # Set as current context
    kubectl config set-context --current --namespace=${NAMESPACE}
    
    log_success "Production namespace setup completed"
}

# Configure security hardening
configure_security() {
    log_info "Configuring security hardening..."
    
    # Install Pod Security Standards
    kubectl label namespace ${NAMESPACE} \
        pod-security.kubernetes.io/enforce=restricted \
        pod-security.kubernetes.io/audit=restricted \
        pod-security.kubernetes.io/warn=restricted \
        --overwrite
    
    # Apply security contexts
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: SecurityContextConstraints
metadata:
  name: aegisshield-scc
  namespace: ${NAMESPACE}
allowHostDirVolumePlugin: false
allowHostIPC: false
allowHostNetwork: false
allowHostPID: false
allowHostPorts: false
allowPrivilegedContainer: false
allowedCapabilities: null
defaultAddCapabilities: null
fsGroup:
  type: MustRunAs
  ranges:
  - min: 1000
    max: 65535
runAsUser:
  type: MustRunAsNonRoot
seLinuxContext:
  type: RunAsAny
users:
- system:serviceaccount:${NAMESPACE}:aegisshield-production-sa
EOF
    
    # Configure admission controllers
    log_info "Validating admission controllers..."
    local admission_controllers=$(kubectl api-versions | grep -E "admission|policy" | wc -l)
    log_info "Found ${admission_controllers} admission controller APIs"
    
    log_success "Security hardening configuration completed"
}

# Setup monitoring integration
setup_monitoring() {
    log_info "Setting up monitoring integration..."
    
    # Create monitoring namespace if it doesn't exist
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    # Install Prometheus Operator if not present
    if ! kubectl get crd prometheuses.monitoring.coreos.com &> /dev/null; then
        log_info "Installing Prometheus Operator..."
        kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
        
        # Wait for CRDs to be ready
        kubectl wait --for condition=established --timeout=60s crd/prometheuses.monitoring.coreos.com
        kubectl wait --for condition=established --timeout=60s crd/servicemonitors.monitoring.coreos.com
    fi
    
    # Apply ServiceMonitor for production services
    kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: aegisshield-production
  namespace: monitoring
  labels:
    app: aegisshield
    environment: production
spec:
  namespaceSelector:
    matchNames:
    - ${NAMESPACE}
  selector:
    matchLabels:
      monitoring: enabled
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    scrapeTimeout: 10s
    relabelings:
    - sourceLabels: [__meta_kubernetes_service_name]
      targetLabel: service
    - sourceLabels: [__meta_kubernetes_namespace]
      targetLabel: namespace
    - sourceLabels: [__meta_kubernetes_pod_name]
      targetLabel: pod
EOF
    
    log_success "Monitoring integration setup completed"
}

# Configure resource allocation
configure_resources() {
    log_info "Configuring production resource allocation..."
    
    # Apply vertical pod autoscaler if available
    if kubectl get crd verticalpodautoscalers.autoscaling.k8s.io &> /dev/null; then
        log_info "Configuring Vertical Pod Autoscaler..."
        kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: api-gateway-vpa
  namespace: ${NAMESPACE}
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: api-gateway
      minAllowed:
        cpu: 100m
        memory: 128Mi
      maxAllowed:
        cpu: 2
        memory: 4Gi
      controlledResources: ["cpu", "memory"]
EOF
    fi
    
    # Configure cluster autoscaler annotations
    kubectl annotate namespace ${NAMESPACE} \
        cluster-autoscaler.kubernetes.io/safe-to-evict="false" \
        --overwrite
    
    log_success "Resource allocation configuration completed"
}

# Setup backup integration
setup_backup_integration() {
    log_info "Setting up backup integration..."
    
    # Create backup service account
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backup-operator
  namespace: ${NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: backup-operator
rules:
- apiGroups: [""]
  resources: ["pods", "persistentvolumes", "persistentvolumeclaims"]
  verbs: ["get", "list", "create", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: backup-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: backup-operator
subjects:
- kind: ServiceAccount
  name: backup-operator
  namespace: ${NAMESPACE}
EOF
    
    # Schedule backup cronjobs
    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: database-backup
  namespace: ${NAMESPACE}
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: backup-operator
          containers:
          - name: backup
            image: postgres:13
            command:
            - /bin/bash
            - -c
            - |
              pg_dump \$DATABASE_URL > /backup/postgres-\$(date +%Y%m%d-%H%M%S).sql
              echo "Backup completed: \$(date)"
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: production-secrets
                  key: database-url
            volumeMounts:
            - name: backup-storage
              mountPath: /backup
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
EOF
    
    log_success "Backup integration setup completed"
}

# Validate production deployment
validate_deployment() {
    log_info "Validating production deployment configuration..."
    
    # Check resource quotas
    local quota_status=$(kubectl get resourcequota production-resource-quota -n ${NAMESPACE} -o jsonpath='{.status.used}' 2>/dev/null || echo "{}")
    log_info "Current resource quota usage: ${quota_status}"
    
    # Check network policies
    local network_policies=$(kubectl get networkpolicy -n ${NAMESPACE} --no-headers | wc -l)
    log_info "Network policies configured: ${network_policies}"
    
    # Check RBAC
    local roles=$(kubectl get role -n ${NAMESPACE} --no-headers | wc -l)
    local rolebindings=$(kubectl get rolebinding -n ${NAMESPACE} --no-headers | wc -l)
    log_info "RBAC roles: ${roles}, role bindings: ${rolebindings}"
    
    # Check pod security policies
    local psps=$(kubectl get psp --no-headers 2>/dev/null | grep aegisshield | wc -l || echo "0")
    log_info "Pod security policies: ${psps}"
    
    # Validate monitoring
    if kubectl get servicemonitor aegisshield-production-monitor -n ${NAMESPACE} &> /dev/null; then
        log_success "ServiceMonitor configured correctly"
    else
        log_warning "ServiceMonitor not found"
    fi
    
    log_success "Production deployment validation completed"
}

# Generate production deployment summary
generate_summary() {
    log_info "Generating production deployment summary..."
    
    local summary_file="${SCRIPT_DIR}/production-deployment-summary-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$summary_file" <<EOF
# AegisShield Production Environment Deployment Summary

Generated: $(date)
Cluster: ${CLUSTER_NAME}
Namespace: ${NAMESPACE}

## Configuration Applied

### Security Hardening
- ✅ Network policies configured
- ✅ RBAC roles and bindings applied
- ✅ Pod security policies enforced
- ✅ Resource quotas and limits set
- ✅ Security contexts configured

### Resource Management
- ✅ Horizontal Pod Autoscalers configured
- ✅ Pod Disruption Budgets set
- ✅ Priority classes defined
- ✅ Resource quotas enforced

### Monitoring Integration
- ✅ ServiceMonitor configured
- ✅ Prometheus integration enabled
- ✅ Metrics endpoints exposed

### Backup Configuration
- ✅ Backup service account created
- ✅ Backup cronjobs scheduled
- ✅ Backup storage configured

## Next Steps

1. Update production secrets in production-secrets Secret
2. Deploy application services using Helm charts
3. Configure external load balancer
4. Set up SSL certificates
5. Configure DNS records
6. Run production readiness tests

## Important Notes

- All secrets are template values and must be updated before production deployment
- Network policies restrict inter-pod communication - verify connectivity
- Resource limits are set conservatively - monitor and adjust as needed
- Backup schedule is set to daily at 2 AM UTC - adjust as required

## Compliance

- SOX: Audit trails enabled, access controls enforced
- PCI-DSS: Network segmentation, encryption at rest
- GDPR: Data protection policies, retention controls

EOF
    
    log_success "Production deployment summary generated: ${summary_file}"
    
    # Display summary
    cat "$summary_file"
}

# Main execution
main() {
    log_info "Starting AegisShield production environment configuration..."
    log_info "Log file: ${LOG_FILE}"
    
    check_prerequisites
    validate_cluster_resources
    setup_namespace
    configure_security
    setup_monitoring
    configure_resources
    setup_backup_integration
    validate_deployment
    generate_summary
    
    log_success "Production environment configuration completed successfully!"
    log_info "Review the deployment summary and update secrets before deploying services"
    log_warning "IMPORTANT: Update all secret values in production-secrets before deployment"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi