# AegisShield ML Pipeline Makefile

.PHONY: build run test clean deps lint docker-build docker-run

# Variables
APP_NAME=ml-pipeline
BINARY_NAME=ml-pipeline-server
GO_FILES=$(shell find . -type f -name '*.go')
CONFIG_FILE=config/config.yaml

# Default target
all: deps lint test build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(BINARY_NAME) ./cmd/server

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
	./bin/$(BINARY_NAME) -config $(CONFIG_FILE)

# Run in development mode
dev:
	@echo "Running in development mode..."
	go run ./cmd/server -config $(CONFIG_FILE)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./test/integration/...

# Run end-to-end tests
test-e2e:
	@echo "Running e2e tests..."
	go test -v -tags=e2e ./test/e2e/...

# Lint the code
lint:
	@echo "Linting code..."
	golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Database operations
db-migrate-up:
	@echo "Running database migrations..."
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/aegisshield_ml_pipeline?sslmode=disable" up

db-migrate-down:
	@echo "Rolling back database migrations..."
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/aegisshield_ml_pipeline?sslmode=disable" down

db-migrate-create:
	@echo "Creating new migration: $(name)"
	migrate create -ext sql -dir ./migrations -seq $(name)

# Docker operations
docker-build:
	@echo "Building Docker image..."
	docker build -t aegisshield/ml-pipeline:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 8081:8081 -p 8082:8082 \
		-v $(PWD)/config:/app/config \
		aegisshield/ml-pipeline:latest

docker-compose-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	docker-compose down

# Development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golang/mock/mockgen@latest

# Generate code
generate:
	@echo "Generating code..."
	go generate ./...

# Performance benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Security scan
security:
	@echo "Running security scan..."
	gosec ./...

# Documentation
docs:
	@echo "Generating documentation..."
	godoc -http=:6060

# Release build
release:
	@echo "Building release version..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/server

# Help
help:
	@echo "Available targets:"
	@echo "  build              Build the application"
	@echo "  run                Build and run the application"
	@echo "  dev                Run in development mode"
	@echo "  test               Run tests"
	@echo "  test-coverage      Run tests with coverage"
	@echo "  test-integration   Run integration tests"
	@echo "  test-e2e           Run end-to-end tests"
	@echo "  lint               Lint the code"
	@echo "  fmt                Format the code"
	@echo "  clean              Clean build artifacts"
	@echo "  deps               Install dependencies"
	@echo "  db-migrate-up      Run database migrations"
	@echo "  db-migrate-down    Rollback database migrations"
	@echo "  db-migrate-create  Create new migration"
	@echo "  docker-build       Build Docker image"
	@echo "  docker-run         Run Docker container"
	@echo "  docker-compose-up  Start services with Docker Compose"
	@echo "  docker-compose-down Stop services with Docker Compose"
	@echo "  install-tools      Install development tools"
	@echo "  generate           Generate code"
	@echo "  bench              Run benchmarks"
	@echo "  security           Run security scan"
	@echo "  docs               Generate documentation"
	@echo "  release            Build release version"
	@echo "  help               Show this help message"