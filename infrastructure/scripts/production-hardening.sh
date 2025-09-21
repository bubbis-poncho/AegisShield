#!/bin/bash

# AegisShield Production Hardening Script
# This script applies additional security hardening measures for production deployment
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
LOG_FILE="${SCRIPT_DIR}/production-hardening-$(date +%Y%m%d-%H%M%S).log"

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

# Apply additional security hardening
apply_security_hardening() {
    log_info "Applying additional security hardening..."
    
    # Configure admission controllers
    kubectl apply -f - <<EOF
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionWebhook
metadata:
  name: aegisshield-security-policy
rules:
- operations: ["CREATE", "UPDATE"]
  apiGroups: [""]
  apiVersions: ["v1"]
  resources: ["pods"]
  namespaceSelector:
    matchLabels:
      name: ${NAMESPACE}
clientConfig:
  service:
    name: security-policy-webhook
    namespace: ${NAMESPACE}
    path: "/validate"
admissionReviewVersions: ["v1", "v1beta1"]
sideEffects: None
failurePolicy: Fail
EOF
    
    # Configure OPA Gatekeeper policies
    kubectl apply -f - <<EOF
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata:
  name: k8srequiredsecuritycontext
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredSecurityContext
      validation:
        type: object
        properties:
          runAsNonRoot:
            type: boolean
          runAsUser:
            type: integer
            minimum: 1000
  targets:
  - target: admission.k8s.gatekeeper.sh
    rego: |
      package k8srequiredsecuritycontext
      
      violation[{"msg": msg}] {
        container := input.review.object.spec.containers[_]
        not container.securityContext.runAsNonRoot
        msg := "Container must run as non-root user"
      }
      
      violation[{"msg": msg}] {
        container := input.review.object.spec.containers[_]
        container.securityContext.runAsUser < 1000
        msg := "Container must run with user ID >= 1000"
      }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredSecurityContext
metadata:
  name: must-run-as-nonroot
spec:
  match:
    kinds:
    - apiGroups: [""]
      kinds: ["Pod"]
    namespaces: [${NAMESPACE}]
  parameters:
    runAsNonRoot: true
    runAsUser: 1000
EOF
    
    log_success "Security hardening applied"
}

# Configure TLS and encryption
configure_tls_encryption() {
    log_info "Configuring TLS and encryption..."
    
    # Generate TLS certificates for internal communication
    kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: aegisshield-ca-issuer
  namespace: ${NAMESPACE}
spec:
  ca:
    secretName: aegisshield-ca-secret
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: api-gateway-tls
  namespace: ${NAMESPACE}
spec:
  secretName: api-gateway-tls-secret
  issuerRef:
    name: aegisshield-ca-issuer
    kind: Issuer
  commonName: api-gateway.${NAMESPACE}.svc.cluster.local
  dnsNames:
  - api-gateway.${NAMESPACE}.svc.cluster.local
  - api-gateway
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: data-ingestion-tls
  namespace: ${NAMESPACE}
spec:
  secretName: data-ingestion-tls-secret
  issuerRef:
    name: aegisshield-ca-issuer
    kind: Issuer
  commonName: data-ingestion.${NAMESPACE}.svc.cluster.local
  dnsNames:
  - data-ingestion.${NAMESPACE}.svc.cluster.local
  - data-ingestion
EOF
    
    # Configure service mesh for mTLS
    kubectl apply -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ${NAMESPACE}
spec:
  mtls:
    mode: STRICT
---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: aegisshield-authz
  namespace: ${NAMESPACE}
spec:
  rules:
  - from:
    - source:
        namespaces: ["${NAMESPACE}"]
    to:
    - operation:
        methods: ["GET", "POST", "PUT", "DELETE"]
EOF
    
    log_success "TLS and encryption configuration completed"
}

# Configure audit logging
configure_audit_logging() {
    log_info "Configuring audit logging..."
    
    # Configure Falco for runtime security monitoring
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: falco-config
  namespace: ${NAMESPACE}
data:
  falco.yaml: |
    rules_file:
      - /etc/falco/falco_rules.yaml
      - /etc/falco/falco_rules.local.yaml
      - /etc/falco/k8s_audit_rules.yaml
    
    json_output: true
    json_include_output_property: true
    
    log_stderr: true
    log_syslog: true
    log_level: info
    
    outputs:
      rate: 1
      max_burst: 1000
    
    grpc:
      enabled: true
      bind_address: "0.0.0.0:5060"
      threadiness: 8
    
    grpc_output:
      enabled: true
    
    webserver:
      enabled: true
      listen_port: 8765
      k8s_audit_endpoint: /k8s-audit
      ssl_enabled: false
  
  falco_rules.local.yaml: |
    - rule: AegisShield Sensitive File Access
      desc: Detect access to sensitive AegisShield files
      condition: >
        open_read and
        fd.name contains "/etc/aegisshield" or
        fd.name contains "/var/lib/aegisshield" or
        fd.name contains "/opt/aegisshield/config"
      output: >
        Sensitive file accessed (user=%user.name command=%proc.cmdline
        file=%fd.name container=%container.name image=%container.image.repository)
      priority: WARNING
      tags: [filesystem, aegisshield]
    
    - rule: AegisShield Unexpected Network Connection
      desc: Detect unexpected network connections from AegisShield services
      condition: >
        outbound and
        container.name contains "aegisshield" and
        not fd.sip in (postgresql_ips, neo4j_ips, vault_ips)
      output: >
        Unexpected network connection (user=%user.name command=%proc.cmdline
        connection=%fd.name container=%container.name image=%container.image.repository)
      priority: WARNING
      tags: [network, aegisshield]
EOF
    
    # Deploy Falco DaemonSet
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: falco
  namespace: ${NAMESPACE}
  labels:
    app: falco
spec:
  selector:
    matchLabels:
      app: falco
  template:
    metadata:
      labels:
        app: falco
    spec:
      serviceAccount: falco
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
      containers:
      - name: falco
        image: falcosecurity/falco:0.32.2
        securityContext:
          privileged: true
        args:
        - /usr/bin/falco
        - --cri
        - /host/run/containerd/containerd.sock
        - --k8s-api
        - --k8s-api-cert
        - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        - --k8s-api-token
        - /var/run/secrets/kubernetes.io/serviceaccount/token
        volumeMounts:
        - mountPath: /host/var/run/docker.sock
          name: docker-socket
        - mountPath: /host/run/containerd/containerd.sock
          name: containerd-socket
        - mountPath: /host/dev
          name: dev-fs
        - mountPath: /host/proc
          name: proc-fs
          readOnly: true
        - mountPath: /host/boot
          name: boot-fs
          readOnly: true
        - mountPath: /host/lib/modules
          name: lib-modules
          readOnly: true
        - mountPath: /host/usr
          name: usr-fs
          readOnly: true
        - mountPath: /etc/falco
          name: falco-config
      volumes:
      - name: docker-socket
        hostPath:
          path: /var/run/docker.sock
      - name: containerd-socket
        hostPath:
          path: /run/containerd/containerd.sock
      - name: dev-fs
        hostPath:
          path: /dev
      - name: proc-fs
        hostPath:
          path: /proc
      - name: boot-fs
        hostPath:
          path: /boot
      - name: lib-modules
        hostPath:
          path: /lib/modules
      - name: usr-fs
        hostPath:
          path: /usr
      - name: falco-config
        configMap:
          name: falco-config
EOF
    
    log_success "Audit logging configuration completed"
}

# Configure compliance monitoring
configure_compliance_monitoring() {
    log_info "Configuring compliance monitoring..."
    
    # Deploy compliance monitoring stack
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: compliance-rules
  namespace: ${NAMESPACE}
data:
  sox-rules.yaml: |
    groups:
    - name: sox_compliance
      rules:
      - alert: SOXAuditTrailMissing
        expr: increase(audit_log_entries_total[5m]) == 0
        for: 5m
        labels:
          severity: critical
          compliance: sox
        annotations:
          summary: "SOX audit trail missing"
          description: "No audit log entries detected for 5 minutes"
      
      - alert: SOXUnauthorizedAccess
        expr: rate(failed_login_attempts_total[5m]) > 5
        for: 2m
        labels:
          severity: warning
          compliance: sox
        annotations:
          summary: "SOX unauthorized access detected"
          description: "High rate of failed login attempts"
  
  pci-rules.yaml: |
    groups:
    - name: pci_compliance
      rules:
      - alert: PCIUnencryptedData
        expr: unencrypted_data_transmission_total > 0
        for: 0m
        labels:
          severity: critical
          compliance: pci
        annotations:
          summary: "PCI unencrypted data transmission"
          description: "Unencrypted sensitive data detected"
      
      - alert: PCINetworkViolation
        expr: network_policy_violations_total > 0
        for: 0m
        labels:
          severity: high
          compliance: pci
        annotations:
          summary: "PCI network policy violation"
          description: "Network segmentation violation detected"
  
  gdpr-rules.yaml: |
    groups:
    - name: gdpr_compliance
      rules:
      - alert: GDPRDataRetentionViolation
        expr: data_retention_days > 365
        for: 0m
        labels:
          severity: high
          compliance: gdpr
        annotations:
          summary: "GDPR data retention violation"
          description: "Data retained beyond GDPR limits"
      
      - alert: GDPRConsentMissing
        expr: consent_missing_total > 0
        for: 0m
        labels:
          severity: warning
          compliance: gdpr
        annotations:
          summary: "GDPR consent missing"
          description: "Processing without valid consent detected"
EOF
    
    # Configure compliance dashboard
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: compliance-dashboard
  namespace: ${NAMESPACE}
data:
  dashboard.json: |
    {
      "dashboard": {
        "title": "AegisShield Compliance Dashboard",
        "panels": [
          {
            "title": "SOX Compliance Score",
            "type": "stat",
            "targets": [
              {
                "expr": "sox_compliance_score",
                "legendFormat": "SOX Score"
              }
            ]
          },
          {
            "title": "PCI-DSS Compliance Score",
            "type": "stat",
            "targets": [
              {
                "expr": "pci_compliance_score",
                "legendFormat": "PCI Score"
              }
            ]
          },
          {
            "title": "GDPR Compliance Score",
            "type": "stat",
            "targets": [
              {
                "expr": "gdpr_compliance_score",
                "legendFormat": "GDPR Score"
              }
            ]
          }
        ]
      }
    }
EOF
    
    log_success "Compliance monitoring configuration completed"
}

# Configure backup encryption
configure_backup_encryption() {
    log_info "Configuring backup encryption..."
    
    # Create encryption keys for backups
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: backup-encryption-key
  namespace: ${NAMESPACE}
type: Opaque
data:
  encryption.key: $(openssl rand -base64 32 | base64 -w 0)
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: encrypted-backup
  namespace: ${NAMESPACE}
spec:
  schedule: "0 */4 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: aegisshield/backup-tool:latest
            command:
            - /bin/bash
            - -c
            - |
              # Backup with encryption
              pg_dump $DATABASE_URL | \
              openssl enc -aes-256-cbc -salt -k $ENCRYPTION_KEY | \
              aws s3 cp - s3://aegisshield-backups/encrypted/postgres-$(date +%Y%m%d-%H%M%S).sql.enc
              
              # Backup Neo4j with encryption
              neo4j-admin backup --backup-dir=/tmp/neo4j-backup --name=graph
              tar -czf - /tmp/neo4j-backup | \
              openssl enc -aes-256-cbc -salt -k $ENCRYPTION_KEY | \
              aws s3 cp - s3://aegisshield-backups/encrypted/neo4j-$(date +%Y%m%d-%H%M%S).tar.gz.enc
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: production-secrets
                  key: database-url
            - name: ENCRYPTION_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-encryption-key
                  key: encryption.key
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: access-key-id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: secret-access-key
          restartPolicy: OnFailure
EOF
    
    log_success "Backup encryption configuration completed"
}

# Main execution
main() {
    log_info "Starting AegisShield production hardening..."
    log_info "Log file: ${LOG_FILE}"
    
    apply_security_hardening
    configure_tls_encryption
    configure_audit_logging
    configure_compliance_monitoring
    configure_backup_encryption
    
    log_success "Production hardening completed successfully!"
    log_warning "Review all configurations and update secrets before production deployment"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi