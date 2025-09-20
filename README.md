# Data Intelligence and Investigation Platform

[![CI/CD Pipeline](https://github.com/aegisshield/platform/workflows/CI/badge.svg)](https://github.com/aegisshield/platform/actions)
[![Test Coverage](https://codecov.io/gh/aegisshield/platform/branch/main/graph/badge.svg)](https://codecov.io/gh/aegisshield/platform)
[![Security Scan](https://snyk.io/test/github/aegisshield/platform/badge.svg)](https://snyk.io/test/github/aegisshield/platform)

A comprehensive financial crimes investigation platform for detecting, analyzing, and investigating complex activities like money laundering, fraud, and other financial crimes. The platform ingests vast amounts of data from disconnected sources and uses advanced entity resolution and pattern detection to identify suspicious activities and provide powerful investigation tools.

## Architecture

**Three-Tier Microservices Architecture**:
- **Frontend**: Next.js 14+ with Shadcn UI, Zustand, TanStack Query, Cytoscape.js
- **API Gateway**: GraphQL aggregation layer for frontend data access  
- **Backend Services**: 6 containerized microservices in Go and Python
- **Infrastructure**: Kubernetes orchestration with Istio service mesh

**Technology Stack**:
- **Languages**: Go 1.21+, Python 3.11+, TypeScript
- **Databases**: PostgreSQL 15+, Neo4j 5+, Apache Doris, Apache Kafka
- **Storage**: AWS S3/GCP with Apache Iceberg table format
- **Performance**: 10,000+ TPS ingestion, <200ms query response, 99.9% uptime

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Kubernetes cluster (local or cloud)
- Helm 3.x
- Node.js 18+
- Go 1.21+
- Python 3.11+

### Local Development
```bash
# Clone the repository
git clone https://github.com/aegisshield/platform.git
cd platform

# Start infrastructure services
docker-compose -f docker-compose.dev.yml up -d

# Start backend services
cd backend
make dev

# Start frontend
cd frontend
npm install
npm run dev
```

### Production Deployment
```bash
# Deploy to Kubernetes
helm install aegisshield ./infrastructure/helm/aegisshield

# Verify deployment
kubectl get pods -n aegisshield
```

## Project Structure

```
├── backend/
│   ├── api-gateway/         # GraphQL API aggregation layer
│   ├── services/
│   │   ├── data-ingestion/  # Go service for real-time data processing
│   │   ├── entity-resolution/ # Python service for ML-based entity matching
│   │   ├── alert-engine/    # Go service for pattern detection
│   │   ├── investigation/   # Go service for graph queries
│   │   ├── user-management/ # Go service for auth/authz
│   │   └── reporting/       # Python service for compliance reports
│   ├── shared/
│   │   ├── proto/           # gRPC service definitions
│   │   ├── events/          # Kafka event schemas
│   │   └── models/          # Shared data models
│   └── infrastructure/
│       ├── k8s/             # Kubernetes manifests
│       ├── docker-compose/  # Local development
│       └── scripts/         # Deployment and migration scripts
├── frontend/
│   ├── src/
│   │   ├── components/      # React components with Shadcn UI
│   │   ├── pages/          # Next.js pages and API routes
│   │   ├── stores/         # Zustand state management
│   │   ├── lib/            # Graph visualization (Cytoscape)
│   │   └── types/          # TypeScript definitions
│   └── tests/              # Frontend tests
└── specs/                  # Feature specifications and documentation
```

## Core Features

### Data Ingestion & Integration
- Multi-source data ingestion (transaction logs, sanctions lists, customer data)
- Real-time streaming and batch processing
- Data validation, cleansing, and audit trails
- Support for various formats (JSON, CSV, XML, AVRO)

### Entity Resolution & Modeling
- Automated entity resolution using ML models
- Unified views of individuals, organizations, and concepts
- Declarative data modeling with multiple logical views
- Relationship mapping with confidence scoring

### Automated Alerting
- Real-time pattern detection for suspicious activities
- Configurable rule engine with risk scoring
- High-fidelity alerts with supporting evidence
- Integration with compliance workflows

### Interactive Investigation
- Graph-based visualization for entity exploration
- Visual relationship navigation and drill-down
- Hypothesis testing through interactive exploration
- Case management and audit trails

## Constitutional Principles

The platform is built on five core principles:

1. **Data Integrity is Paramount**: Transactional, auditable processes with error handling
2. **Scalability and Performance**: Massive scale, real-time processing, low-latency responses
3. **Modular and Maintainable Code**: Loosely coupled services with well-defined APIs
4. **Comprehensive Testing**: Robust test coverage with CI/CD automation
5. **Consistent and Intuitive User Experience**: Analyst-focused, efficient workflows

## Performance Targets

- **Ingestion**: 10,000+ transactions per second
- **Query Response**: <200ms for investigation queries  
- **Alert Generation**: <500ms for real-time patterns
- **Uptime**: 99.9% availability
- **Scale**: 100+ million entities, 1B+ relationships

## Compliance & Security

- **Regulatory**: GDPR, PCI-DSS, AML compliance frameworks
- **Security**: Encryption at rest and in transit, role-based access controls
- **Audit**: Comprehensive logging and audit trails for all activities
- **Privacy**: Data minimization and right to erasure capabilities

## Development

### Branch Strategy
- `main`: Production-ready code
- `develop`: Integration branch for features
- `feature/*`: Feature development branches
- `hotfix/*`: Critical production fixes

### Testing Strategy
- **Unit Tests**: 80%+ coverage per service
- **Integration Tests**: Cross-service workflow validation
- **End-to-End Tests**: Complete user journey validation
- **Performance Tests**: Load and stress testing

### CI/CD Pipeline
1. **Code Quality**: Linting, formatting, security scanning
2. **Testing**: Unit, integration, and E2E test execution
3. **Building**: Docker image creation and registry push
4. **Deployment**: Automated deployment to staging/production
5. **Monitoring**: Health checks and performance validation

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to this project.

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file for details.

## Support

For support and questions, please contact the AegisShield Platform Team.