# AegisShield Testing Strategy

This directory contains all testing infrastructure for the AegisShield platform, following our constitutional principle of **Comprehensive Testing**.

## Testing Pyramid

### Unit Tests (Individual Services)
- `services/*/tests/` - Service-specific unit tests
- **Go**: Standard `go test` with table-driven tests and mocks
- **Python**: `pytest` with fixtures and parametrized tests
- **TypeScript**: `Jest` with comprehensive mocking

### Integration Tests (`tests/integration/`)
- gRPC service integration tests
- Database integration tests
- Message queue integration tests
- API contract validation tests

### End-to-End Tests (`tests/e2e/`)
- Complete user workflow tests
- Cross-service communication tests
- Performance and load testing
- Security and compliance validation

## Test-Driven Development (TDD) Approach

### Constitutional Principle: Write Failing Tests First

1. **Red**: Write failing tests that describe desired behavior
2. **Green**: Implement minimal code to make tests pass
3. **Refactor**: Improve code while keeping tests green

### Test Categories

#### gRPC Contract Tests
- **Data Ingestion Service**: File upload, streaming validation
- **Entity Resolution Service**: Identity matching, graph linking
- **Alerting Engine**: Rule evaluation, notification dispatch
- **Graph Engine**: Query processing, relationship analysis

#### Integration Workflow Tests
- **Data Pipeline**: Ingestion → Resolution → Graph → Alerts
- **User Workflow**: Authentication → Query → Visualization
- **Compliance**: Audit logging, data retention, privacy controls

## Testing Tools and Frameworks

### Backend Testing (Go/Python)
```bash
# Go services
go test ./... -v -race -cover
go test ./... -bench=. -benchmem

# Python services  
pytest --cov=src --cov-report=html
pytest --benchmark-only
```

### Frontend Testing (TypeScript)
```bash
# Unit and integration tests
npm test -- --coverage
npm run test:watch

# E2E tests with Playwright
npm run test:e2e
npm run test:e2e:ui
```

### Infrastructure Testing
```bash
# Database integration tests
make test-infra
./scripts/test-postgresql.sh
./scripts/test-neo4j.sh
./scripts/test-kafka.sh
```

## Test Data Management

### Test Fixtures
- **Synthetic Data**: Generated test datasets for development
- **Anonymized Data**: Production-like data with PII removed
- **Edge Cases**: Boundary conditions and error scenarios

### Test Environments
- **Local**: Individual developer testing with minimal data
- **CI/CD**: Automated testing with full synthetic datasets
- **Staging**: Integration testing with production-scale data

## Performance Testing

### Load Testing Targets
- **Data Ingestion**: 10,000+ TPS sustained throughput
- **Graph Queries**: Sub-200ms response time for complex queries
- **Real-time Alerts**: Sub-100ms rule evaluation latency
- **API Gateway**: 99.9% uptime under normal load

### Tools
- **Go**: Built-in benchmarking + `pprof` profiling
- **Artillery.js**: HTTP load testing for APIs
- **K6**: Kubernetes-native performance testing
- **Neo4j Browser**: Graph query performance analysis

## Security Testing

### Automated Security Scans
- **SAST**: Static application security testing in CI/CD
- **DAST**: Dynamic application security testing
- **Container Scanning**: Vulnerability assessment of Docker images
- **Dependency Scanning**: Third-party library vulnerability checks

### Compliance Testing
- **Data Privacy**: GDPR/CCPA compliance validation
- **Financial Regulations**: PCI DSS, SOX, AML compliance
- **Audit Trails**: Complete request/response logging verification

## Continuous Testing

### CI/CD Integration
- **Pre-commit**: Linting, formatting, basic unit tests
- **Pull Request**: Full test suite, security scans, performance regression
- **Deployment**: Integration tests, smoke tests, rollback validation
- **Production**: Synthetic monitoring, real user monitoring

### Test Coverage Requirements
- **Unit Tests**: 90%+ code coverage for all services
- **Integration Tests**: 100% gRPC contract coverage
- **E2E Tests**: All critical user journeys covered
- **Performance**: All SLA targets validated under load

## Test Organization

```
tests/
├── unit/                   # Service-specific unit tests
│   ├── data-ingestion/
│   ├── entity-resolution/
│   ├── alerting-engine/
│   └── graph-engine/
├── integration/            # Cross-service integration tests
│   ├── grpc-contracts/
│   ├── database-integration/
│   ├── messaging-integration/
│   └── api-contracts/
├── e2e/                    # End-to-end workflow tests
│   ├── user-workflows/
│   ├── compliance-scenarios/
│   └── performance-tests/
├── fixtures/               # Test data and mocks
│   ├── synthetic-data/
│   ├── mock-responses/
│   └── test-configurations/
└── utils/                  # Testing utilities and helpers
    ├── test-clients/
    ├── data-generators/
    └── assertion-helpers/
```

This comprehensive testing approach ensures that every component meets our constitutional requirements for reliability, performance, and security before deployment.