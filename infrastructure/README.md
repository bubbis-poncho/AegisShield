# AegisShield Infrastructure

This directory contains all infrastructure-as-code configurations for the AegisShield platform.

## Components

### Kubernetes (`k8s/`)
- Core application deployments and services
- Database stateful sets (PostgreSQL, Neo4j, Apache Doris)
- Message queue (Apache Kafka) cluster
- Security infrastructure (HashiCorp Vault)
- Monitoring stack (Prometheus, Grafana, Jaeger)
- Istio service mesh configuration

### Terraform (`terraform/`)
- Cloud provider infrastructure (AWS/GCP)
- Network configuration and security groups
- Managed services setup and configuration
- Environment-specific variable files
- State management and remote backends

## Deployment Environments

### Development
- Single-node Kubernetes cluster (local)
- Minimal resource allocations
- Hot-reloading enabled for rapid development

### Staging  
- Multi-node cluster with HA databases
- Production-like configuration at reduced scale
- Full monitoring and observability stack

### Production
- Multi-zone deployment for high availability
- Auto-scaling enabled for all services
- Full security hardening and compliance monitoring

## Quick Start

```bash
# Development environment
make dev-cluster-up
make dev-deploy

# Staging environment  
make staging-cluster-up
make staging-deploy

# Production environment
make prod-cluster-up
make prod-deploy
```

## Security

All infrastructure follows constitutional security principles:
- Least privilege access controls
- Network segmentation via Istio policies
- Secrets management through HashiCorp Vault
- Regular security scanning and compliance checks
- Audit logging for all infrastructure changes

## Monitoring

Comprehensive observability stack:
- **Metrics**: Prometheus + Grafana dashboards
- **Logs**: Centralized logging with structured formats
- **Traces**: Distributed tracing via Jaeger
- **Alerts**: PagerDuty integration for critical issues
- **SLIs/SLOs**: Performance and reliability targets