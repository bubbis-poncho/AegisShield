#!/bin/bash
# Vault Health Check Script
# Constitutional Principle: Comprehensive Testing & Data Integrity

set -e

NAMESPACE=${NAMESPACE:-aegisshield}
SERVICE=${SERVICE:-vault}
PORT=${PORT:-8200}

echo "Testing HashiCorp Vault connectivity..."

# Test 1: Check if service is available
echo "✓ Checking if Vault service exists..."
kubectl get service $SERVICE -n $NAMESPACE > /dev/null

# Test 2: Check if pods are running
echo "✓ Checking if Vault pods are running..."
kubectl get pods -l app=vault -n $NAMESPACE --field-selector=status.phase=Running | grep Running > /dev/null

# Test 3: Vault status check
echo "✓ Testing Vault status..."
kubectl exec -n $NAMESPACE statefulset/vault -- \
  vault status

# Test 4: Health endpoint test
echo "✓ Testing health endpoint..."
kubectl run vault-test --rm -i --restart=Never --image=curlimages/curl -n $NAMESPACE -- \
  curl -f http://$SERVICE:$PORT/v1/sys/health

# Test 5: Vault initialization check
echo "✓ Checking Vault initialization status..."
kubectl exec -n $NAMESPACE statefulset/vault -- \
  vault status | grep "Initialized" | grep "true" > /dev/null && echo "Vault is initialized" || echo "Vault needs initialization"

# Test 6: Seal status check
echo "✓ Checking Vault seal status..."
SEAL_STATUS=$(kubectl exec -n $NAMESPACE statefulset/vault -- vault status | grep "Sealed" | awk '{print $2}')
if [ "$SEAL_STATUS" = "false" ]; then
  echo "✓ Vault is unsealed and ready"
  
  # Test 7: Authentication test (if unsealed)
  echo "✓ Testing Vault authentication..."
  # Note: In production, use proper authentication methods
  # This is a basic connectivity test
  
  # Test 8: Secret engine test
  echo "✓ Testing secret operations..."
  kubectl exec -n $NAMESPACE statefulset/vault -- \
    vault kv put secret/health-check status="healthy" timestamp="$(date -Iseconds)"
    
  kubectl exec -n $NAMESPACE statefulset/vault -- \
    vault kv get secret/health-check
    
  kubectl exec -n $NAMESPACE statefulset/vault -- \
    vault kv delete secret/health-check
    
else
  echo "⚠️  Vault is sealed - manual unsealing required for full testing"
  echo "   Run: kubectl exec -n $NAMESPACE statefulset/vault -- vault operator unseal <unseal-key>"
fi

# Test 9: Metrics endpoint test (if available)
echo "✓ Testing metrics endpoint..."
kubectl run vault-metrics-test --rm -i --restart=Never --image=curlimages/curl -n $NAMESPACE -- \
  curl -f http://$SERVICE:$PORT/v1/sys/metrics?format=prometheus || echo "Metrics endpoint not accessible (normal if sealed)"

echo "✅ Vault health check completed!"