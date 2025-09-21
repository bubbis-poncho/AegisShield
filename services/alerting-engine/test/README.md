# Alerting Engine Test Suite

This directory contains comprehensive tests for the Alerting Engine service.

## Test Structure

### Unit Tests (`unit_test.go`)
- **Alert Repository**: Validation logic, state transitions
- **Rule Engine**: Expression evaluation, error handling
- **Notification Manager**: Message creation, rate limiting
- **Scheduler**: Task management, cron validation
- **Event Processor**: Message validation, alert creation
- **Database Migrations**: Schema validation, index strategy

### Integration Tests (`integration_test.go`)
- **Database Integration**: Full CRUD operations with PostgreSQL
- **Repository Testing**: Real database operations with Testcontainers
- **Alert Lifecycle**: Create, acknowledge, resolve, escalate workflows
- **Rule Management**: Enable/disable, validation, filtering

### API Tests (`api_test.go`)
- **HTTP Endpoints**: REST API testing with mock dependencies
- **gRPC Services**: Protocol buffer API testing with bufconn
- **Request Validation**: Input validation and error handling
- **Response Format**: JSON/protobuf response verification

### End-to-End Tests (`e2e_test.go`)
- **Event Processing Workflow**: Full pipeline testing (mock infrastructure)
- **Performance Testing**: Rule evaluation under load
- **Notification Delivery**: Multi-channel delivery testing
- **Scheduler Integration**: Task execution and management

## Running Tests

### Prerequisites
- Go 1.21+
- Docker (for Testcontainers integration tests)
- PostgreSQL 15+ (for integration tests)

### Run All Tests
```bash
go test ./test/... -v
```

### Run Unit Tests Only
```bash
go test ./test/ -run TestUnit -v
```

### Run Integration Tests
```bash
go test ./test/ -run TestIntegration -v
```

### Run with Coverage
```bash
go test ./test/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Performance Tests
```bash
go test ./test/ -run TestPerformance -v -timeout=30s
```

### Skip E2E Tests (for faster CI)
```bash
go test ./test/... -short -v
```

## Test Configuration

### Environment Variables
- `DATABASE_URL`: PostgreSQL connection string for integration tests
- `KAFKA_BROKERS`: Kafka brokers for event processing tests
- `TEST_TIMEOUT`: Test timeout duration (default: 30s)

### Mock Configuration
Tests use mock implementations for external dependencies:
- Mock repositories for unit tests
- Testcontainers for integration tests
- In-memory stores for performance tests

## Test Data

### Sample Alerts
- High severity transaction alerts
- Acknowledgment and resolution workflows
- Escalation scenarios

### Sample Rules
- Amount threshold rules
- Velocity detection rules
- Cross-border transaction rules
- Invalid expression handling

### Sample Events
- Transaction events
- Entity resolution events
- Alert trigger events

## Coverage Targets

- **Unit Tests**: 90%+ coverage
- **Integration Tests**: Critical paths covered
- **API Tests**: All endpoints tested
- **E2E Tests**: Major workflows validated

## Continuous Integration

### GitHub Actions Workflow
```yaml
name: Alerting Engine Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - run: go test ./test/... -v -cover
```

## Performance Benchmarks

### Rule Evaluation
- Target: < 5ms per event evaluation
- Load: 1000+ concurrent evaluations
- Memory: < 100MB for rule engine

### Database Operations
- Alert creation: < 10ms
- Alert queries: < 50ms
- Bulk operations: < 100ms per 1000 records

### API Response Times
- HTTP endpoints: < 200ms
- gRPC calls: < 100ms
- Health checks: < 10ms

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Ensure PostgreSQL is running
   - Check connection string format
   - Verify database permissions

2. **Testcontainers Failures**
   - Ensure Docker is running
   - Check available ports
   - Verify container images

3. **Performance Test Failures**
   - Increase timeout values
   - Check system resources
   - Reduce test load

### Debug Mode
Enable debug logging in tests:
```bash
export LOG_LEVEL=debug
go test ./test/... -v
```

## Best Practices

### Writing Tests
1. Use descriptive test names
2. Follow Arrange-Act-Assert pattern
3. Use table-driven tests for multiple scenarios
4. Mock external dependencies
5. Test both success and failure cases

### Test Organization
1. Group related tests in subtests
2. Use setup/teardown functions
3. Keep tests independent
4. Use meaningful assertions

### Performance Testing
1. Set realistic benchmarks
2. Test under load
3. Monitor resource usage
4. Profile slow tests

## Contributing

When adding new features:
1. Write tests first (TDD)
2. Ensure good test coverage
3. Add integration tests for new repositories
4. Update API tests for new endpoints
5. Document test scenarios