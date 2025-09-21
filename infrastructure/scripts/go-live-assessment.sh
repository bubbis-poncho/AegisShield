#!/bin/bash

# AegisShield Go-Live Readiness Assessment Script
# This script automates the validation of go-live readiness criteria
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
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/go-live-assessment-$(date +%Y%m%d-%H%M%S).log"
RESULTS_FILE="${SCRIPT_DIR}/go-live-results-$(date +%Y%m%d-%H%M%S).json"

# Assessment results
declare -A RESULTS
TOTAL_SCORE=0
MAX_SCORE=0

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

# Record assessment result
record_result() {
    local category="$1"
    local test="$2"
    local status="$3"
    local score="$4"
    local max_score="$5"
    local notes="${6:-}"
    
    RESULTS["${category}_${test}"]="$status"
    TOTAL_SCORE=$((TOTAL_SCORE + score))
    MAX_SCORE=$((MAX_SCORE + max_score))
    
    log_info "[$category] $test: $status (Score: $score/$max_score) $notes"
}

# Check Kubernetes cluster connectivity
check_cluster_connectivity() {
    log_info "Checking Kubernetes cluster connectivity..."
    
    if kubectl cluster-info &> /dev/null; then
        record_result "INFRASTRUCTURE" "Cluster_Connectivity" "PASS" 10 10 "✅"
    else
        record_result "INFRASTRUCTURE" "Cluster_Connectivity" "FAIL" 0 10 "❌ Cannot connect to cluster"
        return 1
    fi
    
    # Check cluster version
    local k8s_version=$(kubectl version --short 2>/dev/null | grep "Server Version" | awk '{print $3}' || echo "unknown")
    if [[ "$k8s_version" =~ v1\.(2[4-9]|[3-9][0-9]) ]]; then
        record_result "INFRASTRUCTURE" "Kubernetes_Version" "PASS" 5 5 "✅ Version: $k8s_version"
    else
        record_result "INFRASTRUCTURE" "Kubernetes_Version" "FAIL" 0 5 "❌ Unsupported version: $k8s_version"
    fi
}

# Check production namespace readiness
check_namespace_readiness() {
    log_info "Checking production namespace readiness..."
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        record_result "INFRASTRUCTURE" "Production_Namespace" "PASS" 5 5 "✅"
        
        # Check namespace labels
        local labels=$(kubectl get namespace "$NAMESPACE" -o jsonpath='{.metadata.labels}' 2>/dev/null || echo "{}")
        if echo "$labels" | grep -q "security.level.*high"; then
            record_result "SECURITY" "Namespace_Security_Labels" "PASS" 5 5 "✅"
        else
            record_result "SECURITY" "Namespace_Security_Labels" "FAIL" 0 5 "❌ Missing security labels"
        fi
    else
        record_result "INFRASTRUCTURE" "Production_Namespace" "FAIL" 0 5 "❌ Namespace not found"
        return 1
    fi
}

# Check application services health
check_application_services() {
    log_info "Checking application services health..."
    
    local services=("api-gateway" "data-ingestion" "graph-engine" "entity-resolution" "ml-pipeline" "analytics-dashboard" "investigation-toolkit" "alerting-engine" "compliance-engine" "user-management")
    local healthy_services=0
    
    for service in "${services[@]}"; do
        if kubectl get deployment "$service" -n "$NAMESPACE" &> /dev/null; then
            local ready_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
            local desired_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
            
            if [ "$ready_replicas" -eq "$desired_replicas" ] && [ "$ready_replicas" -gt 0 ]; then
                record_result "APPLICATIONS" "${service}_Health" "PASS" 5 5 "✅ $ready_replicas/$desired_replicas ready"
                ((healthy_services++))
            else
                record_result "APPLICATIONS" "${service}_Health" "FAIL" 0 5 "❌ $ready_replicas/$desired_replicas ready"
            fi
        else
            record_result "APPLICATIONS" "${service}_Health" "FAIL" 0 5 "❌ Deployment not found"
        fi
    done
    
    # Overall service health score
    local service_health_score=$((healthy_services * 10 / ${#services[@]}))
    record_result "APPLICATIONS" "Overall_Service_Health" "INFO" "$service_health_score" 10 "($healthy_services/${#services[@]} services healthy)"
}

# Check database connectivity
check_database_connectivity() {
    log_info "Checking database connectivity..."
    
    # Check PostgreSQL
    if kubectl get statefulset postgresql -n "$NAMESPACE" &> /dev/null; then
        local pg_ready=$(kubectl get statefulset postgresql -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "$pg_ready" -gt 0 ]; then
            record_result "DATABASES" "PostgreSQL_Health" "PASS" 10 10 "✅"
        else
            record_result "DATABASES" "PostgreSQL_Health" "FAIL" 0 10 "❌ Not ready"
        fi
    else
        record_result "DATABASES" "PostgreSQL_Health" "FAIL" 0 10 "❌ Not deployed"
    fi
    
    # Check Neo4j
    if kubectl get statefulset neo4j -n "$NAMESPACE" &> /dev/null; then
        local neo4j_ready=$(kubectl get statefulset neo4j -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "$neo4j_ready" -gt 0 ]; then
            record_result "DATABASES" "Neo4j_Health" "PASS" 10 10 "✅"
        else
            record_result "DATABASES" "Neo4j_Health" "FAIL" 0 10 "❌ Not ready"
        fi
    else
        record_result "DATABASES" "Neo4j_Health" "FAIL" 0 10 "❌ Not deployed"
    fi
    
    # Check Redis
    if kubectl get deployment redis -n "$NAMESPACE" &> /dev/null; then
        local redis_ready=$(kubectl get deployment redis -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "$redis_ready" -gt 0 ]; then
            record_result "DATABASES" "Redis_Health" "PASS" 5 5 "✅"
        else
            record_result "DATABASES" "Redis_Health" "FAIL" 0 5 "❌ Not ready"
        fi
    else
        record_result "DATABASES" "Redis_Health" "FAIL" 0 5 "❌ Not deployed"
    fi
}

# Check security configurations
check_security_configurations() {
    log_info "Checking security configurations..."
    
    # Check network policies
    local network_policies=$(kubectl get networkpolicy -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$network_policies" -ge 3 ]; then
        record_result "SECURITY" "Network_Policies" "PASS" 10 10 "✅ $network_policies policies"
    else
        record_result "SECURITY" "Network_Policies" "FAIL" 0 10 "❌ Insufficient policies: $network_policies"
    fi
    
    # Check RBAC
    local roles=$(kubectl get role -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    local rolebindings=$(kubectl get rolebinding -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$roles" -gt 0 ] && [ "$rolebindings" -gt 0 ]; then
        record_result "SECURITY" "RBAC_Configuration" "PASS" 10 10 "✅ $roles roles, $rolebindings bindings"
    else
        record_result "SECURITY" "RBAC_Configuration" "FAIL" 0 10 "❌ Missing RBAC: $roles roles, $rolebindings bindings"
    fi
    
    # Check pod security policies
    local psps=$(kubectl get psp --no-headers 2>/dev/null | grep -c aegisshield || echo "0")
    if [ "$psps" -gt 0 ]; then
        record_result "SECURITY" "Pod_Security_Policies" "PASS" 10 10 "✅ $psps policies"
    else
        record_result "SECURITY" "Pod_Security_Policies" "WARN" 5 10 "⚠️ No custom PSPs found"
    fi
    
    # Check secrets
    if kubectl get secret production-secrets -n "$NAMESPACE" &> /dev/null; then
        record_result "SECURITY" "Production_Secrets" "PASS" 10 10 "✅"
    else
        record_result "SECURITY" "Production_Secrets" "FAIL" 0 10 "❌ Missing production secrets"
    fi
}

# Check monitoring setup
check_monitoring_setup() {
    log_info "Checking monitoring setup..."
    
    # Check Prometheus
    if kubectl get pod -l app=prometheus -n monitoring &> /dev/null; then
        record_result "MONITORING" "Prometheus" "PASS" 10 10 "✅"
    else
        record_result "MONITORING" "Prometheus" "FAIL" 0 10 "❌ Not deployed"
    fi
    
    # Check Grafana
    if kubectl get pod -l app=grafana -n monitoring &> /dev/null; then
        record_result "MONITORING" "Grafana" "PASS" 10 10 "✅"
    else
        record_result "MONITORING" "Grafana" "FAIL" 0 10 "❌ Not deployed"
    fi
    
    # Check ServiceMonitors
    local servicemonitors=$(kubectl get servicemonitor -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$servicemonitors" -gt 0 ]; then
        record_result "MONITORING" "ServiceMonitors" "PASS" 5 5 "✅ $servicemonitors monitors"
    else
        record_result "MONITORING" "ServiceMonitors" "FAIL" 0 5 "❌ No ServiceMonitors found"
    fi
    
    # Check alerting rules
    local alerting_rules=$(kubectl get prometheusrule -n monitoring --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$alerting_rules" -gt 0 ]; then
        record_result "MONITORING" "Alerting_Rules" "PASS" 5 5 "✅ $alerting_rules rules"
    else
        record_result "MONITORING" "Alerting_Rules" "WARN" 2 5 "⚠️ No alerting rules found"
    fi
}

# Check resource allocation
check_resource_allocation() {
    log_info "Checking resource allocation..."
    
    # Check resource quotas
    if kubectl get resourcequota production-resource-quota -n "$NAMESPACE" &> /dev/null; then
        record_result "RESOURCES" "Resource_Quotas" "PASS" 5 5 "✅"
        
        # Check quota usage
        local cpu_used=$(kubectl get resourcequota production-resource-quota -n "$NAMESPACE" -o jsonpath='{.status.used.requests\.cpu}' 2>/dev/null || echo "0")
        local memory_used=$(kubectl get resourcequota production-resource-quota -n "$NAMESPACE" -o jsonpath='{.status.used.requests\.memory}' 2>/dev/null || echo "0")
        record_result "RESOURCES" "Resource_Usage" "INFO" 0 0 "CPU: $cpu_used, Memory: $memory_used"
    else
        record_result "RESOURCES" "Resource_Quotas" "FAIL" 0 5 "❌ No resource quotas"
    fi
    
    # Check HPA
    local hpas=$(kubectl get hpa -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$hpas" -ge 2 ]; then
        record_result "RESOURCES" "Horizontal_Pod_Autoscalers" "PASS" 5 5 "✅ $hpas HPAs"
    else
        record_result "RESOURCES" "Horizontal_Pod_Autoscalers" "WARN" 2 5 "⚠️ Only $hpas HPAs found"
    fi
    
    # Check PDBs
    local pdbs=$(kubectl get pdb -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$pdbs" -ge 2 ]; then
        record_result "RESOURCES" "Pod_Disruption_Budgets" "PASS" 5 5 "✅ $pdbs PDBs"
    else
        record_result "RESOURCES" "Pod_Disruption_Budgets" "WARN" 2 5 "⚠️ Only $pdbs PDBs found"
    fi
}

# Check backup configuration
check_backup_configuration() {
    log_info "Checking backup configuration..."
    
    # Check backup cronjobs
    local backup_jobs=$(kubectl get cronjob -n "$NAMESPACE" --no-headers 2>/dev/null | grep -c backup || echo "0")
    if [ "$backup_jobs" -ge 1 ]; then
        record_result "BACKUP" "Backup_Jobs" "PASS" 10 10 "✅ $backup_jobs jobs"
    else
        record_result "BACKUP" "Backup_Jobs" "FAIL" 0 10 "❌ No backup jobs found"
    fi
    
    # Check backup storage
    local backup_pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep -c backup || echo "0")
    if [ "$backup_pvcs" -gt 0 ]; then
        record_result "BACKUP" "Backup_Storage" "PASS" 5 5 "✅ $backup_pvcs PVCs"
    else
        record_result "BACKUP" "Backup_Storage" "WARN" 2 5 "⚠️ No backup PVCs found"
    fi
}

# Performance validation
validate_performance() {
    log_info "Validating performance requirements..."
    
    # This would typically run actual performance tests
    # For now, we'll check if performance testing infrastructure is in place
    
    if [ -f "${SCRIPT_DIR}/../tests/performance/load-test.sh" ]; then
        record_result "PERFORMANCE" "Load_Test_Scripts" "PASS" 5 5 "✅ Scripts available"
    else
        record_result "PERFORMANCE" "Load_Test_Scripts" "FAIL" 0 5 "❌ No load test scripts"
    fi
    
    # Check if monitoring can track performance metrics
    if kubectl get servicemonitor aegisshield-production-monitor -n "$NAMESPACE" &> /dev/null; then
        record_result "PERFORMANCE" "Performance_Monitoring" "PASS" 10 10 "✅"
    else
        record_result "PERFORMANCE" "Performance_Monitoring" "FAIL" 0 10 "❌ No performance monitoring"
    fi
}

# Compliance validation
validate_compliance() {
    log_info "Validating compliance requirements..."
    
    # Check if compliance validation scripts exist
    if [ -f "${SCRIPT_DIR}/compliance-validation.sh" ]; then
        record_result "COMPLIANCE" "Compliance_Scripts" "PASS" 10 10 "✅"
        
        # Run basic compliance checks
        if bash "${SCRIPT_DIR}/compliance-validation.sh" --dry-run &> /dev/null; then
            record_result "COMPLIANCE" "Compliance_Validation" "PASS" 10 10 "✅"
        else
            record_result "COMPLIANCE" "Compliance_Validation" "WARN" 5 10 "⚠️ Validation issues found"
        fi
    else
        record_result "COMPLIANCE" "Compliance_Scripts" "FAIL" 0 10 "❌ No compliance scripts"
    fi
    
    # Check audit logging
    local audit_logs=$(kubectl get pods -n "$NAMESPACE" -l app=falco --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$audit_logs" -gt 0 ]; then
        record_result "COMPLIANCE" "Audit_Logging" "PASS" 10 10 "✅ Falco deployed"
    else
        record_result "COMPLIANCE" "Audit_Logging" "WARN" 5 10 "⚠️ Limited audit logging"
    fi
}

# Generate assessment report
generate_assessment_report() {
    log_info "Generating assessment report..."
    
    local final_score_percentage=$((TOTAL_SCORE * 100 / MAX_SCORE))
    local assessment_status
    
    if [ "$final_score_percentage" -ge 85 ]; then
        assessment_status="GO"
    elif [ "$final_score_percentage" -ge 70 ]; then
        assessment_status="GO_WITH_CONDITIONS"
    else
        assessment_status="NO_GO"
    fi
    
    # Generate JSON report
    cat > "$RESULTS_FILE" <<EOF
{
  "assessment_date": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "total_score": $TOTAL_SCORE,
  "max_score": $MAX_SCORE,
  "percentage": $final_score_percentage,
  "status": "$assessment_status",
  "results": {
EOF
    
    local first=true
    for key in "${!RESULTS[@]}"; do
        if [ "$first" = true ]; then
            first=false
        else
            echo "," >> "$RESULTS_FILE"
        fi
        echo "    \"$key\": \"${RESULTS[$key]}\"" >> "$RESULTS_FILE"
    done
    
    cat >> "$RESULTS_FILE" <<EOF
  },
  "recommendations": [
EOF
    
    # Add recommendations based on results
    local recommendations=()
    
    if [[ "${RESULTS[SECURITY_Production_Secrets]:-}" == "FAIL" ]]; then
        recommendations+=("Update production secrets before deployment")
    fi
    
    if [[ "${RESULTS[MONITORING_Prometheus]:-}" == "FAIL" ]]; then
        recommendations+=("Deploy Prometheus monitoring stack")
    fi
    
    if [[ "${RESULTS[BACKUP_Backup_Jobs]:-}" == "FAIL" ]]; then
        recommendations+=("Configure automated backup procedures")
    fi
    
    if [ ${#recommendations[@]} -gt 0 ]; then
        for i in "${!recommendations[@]}"; do
            if [ $i -gt 0 ]; then
                echo "," >> "$RESULTS_FILE"
            fi
            echo "    \"${recommendations[$i]}\"" >> "$RESULTS_FILE"
        done
    fi
    
    cat >> "$RESULTS_FILE" <<EOF
  ]
}
EOF
    
    # Display summary
    log_info "=================================================="
    log_info "GO-LIVE READINESS ASSESSMENT SUMMARY"
    log_info "=================================================="
    log_info "Total Score: $TOTAL_SCORE / $MAX_SCORE ($final_score_percentage%)"
    log_info "Assessment Status: $assessment_status"
    log_info "=================================================="
    
    case "$assessment_status" in
        "GO")
            log_success "✅ READY FOR PRODUCTION DEPLOYMENT"
            ;;
        "GO_WITH_CONDITIONS")
            log_warning "⚠️ READY WITH CONDITIONS - Address warnings before deployment"
            ;;
        "NO_GO")
            log_error "❌ NOT READY - Critical issues must be resolved"
            ;;
    esac
    
    log_info "Detailed results saved to: $RESULTS_FILE"
    log_info "Assessment log saved to: $LOG_FILE"
}

# Main execution
main() {
    log_info "Starting AegisShield Go-Live Readiness Assessment..."
    log_info "Assessment log: $LOG_FILE"
    log_info "Results file: $RESULTS_FILE"
    
    check_cluster_connectivity
    check_namespace_readiness
    check_application_services
    check_database_connectivity
    check_security_configurations
    check_monitoring_setup
    check_resource_allocation
    check_backup_configuration
    validate_performance
    validate_compliance
    generate_assessment_report
    
    log_info "Go-Live Readiness Assessment completed!"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi