#!/bin/bash
# PostgreSQL Health Check Script
# Constitutional Principle: Comprehensive Testing

set -e

NAMESPACE=${NAMESPACE:-aegisshield}
SERVICE=${SERVICE:-postgresql}
PORT=${PORT:-5432}

echo "Testing PostgreSQL connectivity..."

# Test 1: Check if service is available
echo "✓ Checking if PostgreSQL service exists..."
kubectl get service $SERVICE -n $NAMESPACE > /dev/null

# Test 2: Check if pods are running
echo "✓ Checking if PostgreSQL pods are running..."
kubectl get pods -l app=postgresql -n $NAMESPACE --field-selector=status.phase=Running | grep Running > /dev/null

# Test 3: Port connectivity test
echo "✓ Testing port connectivity..."
kubectl run pg-test --rm -i --restart=Never --image=postgres:15.4-alpine -n $NAMESPACE -- \
  sh -c "pg_isready -h $SERVICE -p $PORT"

# Test 4: Database connection test
echo "✓ Testing database connection..."
kubectl exec -n $NAMESPACE statefulset/postgresql -- \
  psql -h localhost -U aegisshield_user -d aegisshield -c "SELECT version();"

# Test 5: Performance test - simple query
echo "✓ Testing basic query performance..."
kubectl exec -n $NAMESPACE statefulset/postgresql -- \
  psql -h localhost -U aegisshield_user -d aegisshield -c "
    SELECT 
      current_timestamp as test_time,
      'PostgreSQL health check passed' as status;
  "

echo "✅ PostgreSQL health check completed successfully!"