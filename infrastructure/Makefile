# AegisShield Infrastructure Makefile
# Constitutional Principle: Modular Code & Comprehensive Testing

.PHONY: help dev-cluster-up dev-deploy staging-deploy prod-deploy clean

# Default target
help:
	@echo "AegisShield Infrastructure Commands"
	@echo "=================================="
	@echo "dev-cluster-up    - Create local development cluster"
	@echo "dev-deploy        - Deploy all services to development"
	@echo "staging-deploy    - Deploy to staging environment"
	@echo "prod-deploy       - Deploy to production environment"
	@echo "clean             - Clean up all resources"
	@echo "test-infra        - Test infrastructure components"

# Development cluster setup
dev-cluster-up:
	@echo "Creating development Kubernetes cluster..."
	kind create cluster --name aegisshield-dev --config k8s/kind-config.yaml
	kubectl cluster-info --context kind-aegisshield-dev

# Development deployment
dev-deploy: apply-namespace apply-databases apply-messaging
	@echo "Development deployment complete!"
	@echo "Access services:"
	@echo "  PostgreSQL: kubectl port-forward svc/postgresql 5432:5432"
	@echo "  Neo4j:      kubectl port-forward svc/neo4j 7474:7474"
	@echo "  Kafka:      kubectl port-forward svc/kafka 9092:9092"

# Apply core infrastructure components
apply-namespace:
	kubectl apply -f k8s/namespace.yaml

apply-databases:
	kubectl apply -f k8s/postgresql.yaml
	kubectl apply -f k8s/neo4j.yaml
	@echo "Waiting for databases to be ready..."
	kubectl wait --for=condition=ready pod -l app=postgresql --timeout=300s -n aegisshield
	kubectl wait --for=condition=ready pod -l app=neo4j --timeout=300s -n aegisshield

apply-messaging:
	kubectl apply -f k8s/kafka.yaml
	@echo "Waiting for messaging to be ready..."
	kubectl wait --for=condition=ready pod -l app=zookeeper --timeout=300s -n aegisshield
	kubectl wait --for=condition=ready pod -l app=kafka --timeout=300s -n aegisshield

apply-vault:
	kubectl apply -f k8s/vault.yaml
	@echo "Waiting for Vault to be ready..."
	kubectl wait --for=condition=ready pod -l app=vault --timeout=300s -n aegisshield

apply-monitoring:
	kubectl apply -f k8s/monitoring.yaml
	@echo "Waiting for monitoring to be ready..."
	kubectl wait --for=condition=ready pod -l app=prometheus --timeout=300s -n aegisshield

# Staging deployment
staging-deploy:
	@echo "Deploying to staging environment..."
	kubectl config use-context staging-cluster
	$(MAKE) apply-namespace apply-databases apply-messaging apply-vault apply-monitoring

# Production deployment  
prod-deploy:
	@echo "Deploying to production environment..."
	kubectl config use-context production-cluster
	$(MAKE) apply-namespace apply-databases apply-messaging apply-vault apply-monitoring

# Test infrastructure
test-infra:
	@echo "Testing infrastructure components..."
	./scripts/test-postgresql.sh
	./scripts/test-neo4j.sh
	./scripts/test-kafka.sh
	./scripts/test-vault.sh

# Clean up resources
clean:
	kubectl delete namespace aegisshield --ignore-not-found=true
	kubectl delete namespace aegisshield-system --ignore-not-found=true
	kind delete cluster --name aegisshield-dev

# Development helpers
port-forward-pg:
	kubectl port-forward svc/postgresql 5432:5432 -n aegisshield

port-forward-neo4j:
	kubectl port-forward svc/neo4j 7474:7474 -n aegisshield

port-forward-kafka:
	kubectl port-forward svc/kafka 9092:9092 -n aegisshield

logs-postgresql:
	kubectl logs -f statefulset/postgresql -n aegisshield

logs-neo4j:
	kubectl logs -f statefulset/neo4j -n aegisshield

logs-kafka:
	kubectl logs -f statefulset/kafka -n aegisshield