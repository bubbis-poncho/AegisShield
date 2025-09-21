# AegisShield ML Pipeline Service

A comprehensive machine learning pipeline service for the AegisShield financial crime detection platform. This service provides enterprise-grade ML lifecycle management, model training, inference, monitoring, and automated retraining capabilities.

## Features

### 🤖 Model Training & Management
- **Multi-Algorithm Support**: XGBoost, Random Forest, Neural Networks, LSTM
- **Asynchronous Training**: Worker pool-based training with job queuing
- **Hyperparameter Tuning**: Automated parameter optimization
- **Model Versioning**: Complete model lifecycle management
- **Training Pipeline**: End-to-end training workflow management

### 🚀 Model Inference
- **High-Performance Serving**: Optimized model serving with caching
- **Batch & Real-time Predictions**: Support for both prediction modes
- **Load Balancing**: Multiple serving strategies (round-robin, least-connections)
- **Circuit Breakers**: Fault tolerance and resilience patterns
- **Rate Limiting**: Traffic control and resource protection

### 📊 Model Monitoring & Observability
- **Performance Tracking**: Accuracy, precision, recall, F1-score, AUC metrics
- **Data Drift Detection**: Statistical methods (KS test, PSI, Jensen-Shannon)
- **Model Health Monitoring**: Automated health checks and alerts
- **Metrics Collection**: Time-series metrics storage and analysis
- **Alert Management**: Multi-channel alerting (Email, Slack, Webhook)

### 🧪 A/B Testing Framework
- **Traffic Splitting**: Controlled traffic distribution for model comparison
- **Statistical Testing**: Significance testing for model performance
- **Automated Promotion**: Winner selection and automated model promotion
- **Experiment Tracking**: Complete A/B test lifecycle management

### 🏪 Feature Store
- **Feature Management**: Centralized feature storage and serving
- **Real-time Serving**: Low-latency feature retrieval
- **Feature Validation**: Schema and data quality validation
- **Feature Engineering**: Automated feature transformation pipeline

### 🔧 Production-Ready Features
- **Scalable Architecture**: Microservices-based design
- **High Availability**: Fault-tolerant service design
- **Security**: JWT authentication, TLS support, rate limiting
- **Observability**: Prometheus metrics, structured logging
- **Configuration Management**: Environment-based configuration

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    AegisShield ML Pipeline                  │
├─────────────────────────────────────────────────────────────┤
│  REST API (Port 8080)  │  gRPC API (Port 8081)            │
├─────────────────────────────────────────────────────────────┤
│  Training Engine       │  Inference Engine                 │
│  • XGBoost            │  • Model Serving                  │
│  • Random Forest      │  • Batch Prediction               │
│  • Neural Networks    │  • Load Balancing                 │
│  • LSTM               │  • Caching                        │
├─────────────────────────────────────────────────────────────┤
│  Model Monitor         │  Feature Store                    │
│  • Performance        │  • Real-time Serving              │
│  • Drift Detection    │  • Feature Engineering            │
│  • Health Checks      │  • Validation                     │
│  • Alerting           │  • Versioning                     │
├─────────────────────────────────────────────────────────────┤
│  Data Layer: PostgreSQL │ Cache: Redis │ Events: Kafka    │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 13+
- Redis 6+
- Kafka 2.8+ (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/aegisshield/ml-pipeline.git
   cd ml-pipeline
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up database**
   ```bash
   # Create PostgreSQL database
   createdb aegisshield_ml_pipeline
   
   # Run migrations
   migrate -path ./migrations -database "postgres://user:pass@localhost/aegisshield_ml_pipeline?sslmode=disable" up
   ```

4. **Configure the service**
   ```bash
   # Copy configuration template
   cp config/config.yaml.example config/config.yaml
   
   # Edit configuration
   nano config/config.yaml
   ```

5. **Start the service**
   ```bash
   go run cmd/server/main.go -config config/config.yaml
   ```

### Docker Deployment

```bash
# Build the image
docker build -t aegisshield/ml-pipeline:latest .

# Run with docker-compose
docker-compose up -d
```

## API Documentation

### REST API Endpoints

#### Model Management
- `GET /api/v1/models` - List all models
- `POST /api/v1/models` - Create a new model
- `GET /api/v1/models/{id}` - Get model details
- `PUT /api/v1/models/{id}` - Update model
- `DELETE /api/v1/models/{id}` - Delete model

#### Training
- `POST /api/v1/models/{id}/train` - Start model training
- `GET /api/v1/models/{id}/training-jobs` - List training jobs
- `GET /api/v1/models/{id}/training-jobs/{job_id}` - Get training job status

#### Deployment
- `POST /api/v1/models/{id}/deploy` - Deploy model
- `GET /api/v1/models/{id}/deployments` - List deployments

#### Prediction
- `POST /api/v1/models/{id}/predict` - Make single prediction
- `POST /api/v1/models/{id}/batch-predict` - Make batch predictions

#### Monitoring
- `GET /api/v1/models/{id}/metrics` - Get model metrics
- `GET /api/v1/models/{id}/health` - Check model health
- `GET /api/v1/models/{id}/drift` - Get drift status
- `GET /api/v1/models/{id}/alerts` - Get model alerts

### Example Usage

#### Create a Model
```bash
curl -X POST http://localhost:8080/api/v1/models \
  -H "Content-Type: application/json" \
  -d '{
    "name": "fraud_detection_v1",
    "description": "Credit card fraud detection model",
    "algorithm": "xgboost",
    "parameters": {
      "max_depth": 6,
      "learning_rate": 0.1,
      "n_estimators": 100
    }
  }'
```

#### Start Training
```bash
curl -X POST http://localhost:8080/api/v1/models/{model_id}/train \
  -H "Content-Type: application/json" \
  -d '{
    "dataset_path": "/data/fraud_training_data.csv",
    "parameters": {
      "validation_split": 0.2,
      "test_split": 0.1
    }
  }'
```

#### Make Prediction
```bash
curl -X POST http://localhost:8080/api/v1/models/{model_id}/predict \
  -H "Content-Type: application/json" \
  -d '{
    "features": {
      "amount": 150.50,
      "merchant_category": "grocery",
      "time_of_day": "evening",
      "location": "domestic"
    }
  }'
```

## Configuration

The service is configured via YAML configuration file. Key configuration sections:

### Database Configuration
```yaml
database:
  host: localhost
  port: 5432
  name: aegisshield_ml_pipeline
  username: postgres
  password: postgres
  ssl_mode: disable
```

### ML Configuration
```yaml
ml:
  training:
    max_workers: 4
    job_timeout: 3600
    algorithms:
      xgboost:
        enabled: true
        default_params:
          max_depth: 6
          learning_rate: 0.1
```

### Monitoring Configuration
```yaml
ml:
  model_monitoring:
    drift_detection:
      enabled: true
      drift_method: ks
      drift_threshold: 0.05
    alerting:
      enabled: true
      channels: [email, slack, webhook]
```

## Development

### Project Structure
```
ml-pipeline/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── api/            # REST API handlers
│   ├── config/         # Configuration management
│   ├── database/       # Database layer and repositories
│   ├── grpc/          # gRPC server implementation
│   ├── inference/     # Model inference engine
│   ├── monitoring/    # Model monitoring and alerting
│   ├── server/        # Server initialization and management
│   └── training/      # Model training engine
├── migrations/         # Database migrations
├── config/            # Configuration files
└── docs/              # Documentation
```

### Running Tests
```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./test/integration/...

# End-to-end tests
go test -tags=e2e ./test/e2e/...
```

### Development Guidelines

1. **Code Quality**: Follow Go best practices and use `gofmt`, `golint`, `go vet`
2. **Testing**: Maintain >80% test coverage
3. **Documentation**: Keep README and API docs updated
4. **Logging**: Use structured logging with appropriate log levels
5. **Error Handling**: Implement proper error handling and recovery

## Monitoring & Operations

### Health Checks
- **Service Health**: `GET /health`
- **Component Health**: `GET /api/v1/monitoring/health`
- **Database Health**: Included in overall health check

### Metrics
The service exposes Prometheus metrics on port 8082:
- **Model Performance**: Accuracy, latency, throughput
- **System Metrics**: CPU, memory, request counts
- **Business Metrics**: Prediction volumes, error rates

### Logging
Structured JSON logging with configurable levels:
```bash
# Set log level
export LOG_LEVEL=debug

# View logs in development
tail -f /var/log/ml-pipeline.log | jq '.'
```

### Alerting
Configured alerting channels:
- **Email**: SMTP-based email alerts
- **Slack**: Webhook-based Slack notifications
- **Webhook**: Custom webhook integrations

## Security

### Authentication
- JWT-based API authentication
- Configurable token expiry
- Role-based access control (planned)

### Network Security
- TLS support for HTTPS/gRPC
- Rate limiting and DDoS protection
- Input validation and sanitization

### Data Security
- Encrypted database connections
- Secure credential management
- Audit logging for sensitive operations

## Performance

### Optimization Features
- **Connection Pooling**: Database and Redis connection pooling
- **Caching**: Multi-level caching strategy
- **Async Processing**: Non-blocking training and inference
- **Load Balancing**: Multiple serving strategies

### Benchmarks
- **Prediction Latency**: <50ms p95
- **Throughput**: >1000 predictions/second
- **Training Speed**: Optimized for large datasets
- **Memory Usage**: Efficient memory management

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup
```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/golang/mock/mockgen@latest

# Run linting
golangci-lint run

# Generate mocks
go generate ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/aegisshield/ml-pipeline/issues)
- **Discussions**: [GitHub Discussions](https://github.com/aegisshield/ml-pipeline/discussions)
- **Email**: ml-team@aegisshield.com

## Roadmap

### Q1 2024
- [ ] Enhanced A/B testing framework
- [ ] Advanced drift detection algorithms
- [ ] Model explainability features
- [ ] Real-time feature engineering

### Q2 2024
- [ ] Multi-model ensemble support
- [ ] Advanced security features
- [ ] Performance optimizations
- [ ] Cloud provider integrations

### Q3 2024
- [ ] AutoML capabilities
- [ ] Advanced monitoring dashboards
- [ ] Model governance framework
- [ ] Edge deployment support

---

**AegisShield ML Pipeline** - Enterprise-grade machine learning for financial crime detection.