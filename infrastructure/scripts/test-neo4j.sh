#!/bin/bash
# Neo4j Health Check Script
# Constitutional Principle: Comprehensive Testing

set -e

NAMESPACE=${NAMESPACE:-aegisshield}
SERVICE=${SERVICE:-neo4j}
HTTP_PORT=${HTTP_PORT:-7474}
BOLT_PORT=${BOLT_PORT:-7687}

echo "Testing Neo4j connectivity..."

# Test 1: Check if service is available
echo "✓ Checking if Neo4j service exists..."
kubectl get service $SERVICE -n $NAMESPACE > /dev/null

# Test 2: Check if pods are running
echo "✓ Checking if Neo4j pods are running..."
kubectl get pods -l app=neo4j -n $NAMESPACE --field-selector=status.phase=Running | grep Running > /dev/null

# Test 3: HTTP endpoint test
echo "✓ Testing HTTP endpoint..."
kubectl run neo4j-test --rm -i --restart=Never --image=curlimages/curl -n $NAMESPACE -- \
  curl -f http://$SERVICE:$HTTP_PORT/

# Test 4: Database connection test via HTTP API
echo "✓ Testing database connection via HTTP API..."
kubectl exec -n $NAMESPACE statefulset/neo4j -- \
  curl -H "Content-Type: application/json" \
  -d '{"query":"RETURN datetime() as current_time"}' \
  http://localhost:$HTTP_PORT/db/data/transaction/commit

# Test 5: Cypher query test
echo "✓ Testing basic Cypher query..."
kubectl exec -n $NAMESPACE statefulset/neo4j -- \
  cypher-shell -u neo4j -p CHANGE_ME_IN_PRODUCTION \
  "RETURN 'Neo4j health check passed' as status, datetime() as test_time;"

# Test 6: Performance test - create and delete test node
echo "✓ Testing basic graph operations..."
kubectl exec -n $NAMESPACE statefulset/neo4j -- \
  cypher-shell -u neo4j -p CHANGE_ME_IN_PRODUCTION \
  "CREATE (test:HealthCheck {timestamp: datetime()}) RETURN test.timestamp; 
   MATCH (test:HealthCheck) DELETE test;"

echo "✅ Neo4j health check completed successfully!"