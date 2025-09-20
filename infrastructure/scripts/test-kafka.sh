#!/bin/bash
# Kafka Health Check Script  
# Constitutional Principle: Comprehensive Testing

set -e

NAMESPACE=${NAMESPACE:-aegisshield}
KAFKA_SERVICE=${KAFKA_SERVICE:-kafka}
ZOOKEEPER_SERVICE=${ZOOKEEPER_SERVICE:-zookeeper}
KAFKA_PORT=${KAFKA_PORT:-9092}
ZK_PORT=${ZK_PORT:-2181}

echo "Testing Kafka cluster connectivity..."

# Test 1: Check if services are available
echo "✓ Checking if Kafka services exist..."
kubectl get service $KAFKA_SERVICE -n $NAMESPACE > /dev/null
kubectl get service $ZOOKEEPER_SERVICE -n $NAMESPACE > /dev/null

# Test 2: Check if pods are running
echo "✓ Checking if Kafka pods are running..."
kubectl get pods -l app=kafka -n $NAMESPACE --field-selector=status.phase=Running | grep Running > /dev/null
kubectl get pods -l app=zookeeper -n $NAMESPACE --field-selector=status.phase=Running | grep Running > /dev/null

# Test 3: ZooKeeper connectivity test
echo "✓ Testing ZooKeeper connectivity..."
kubectl run zk-test --rm -i --restart=Never --image=confluentinc/cp-zookeeper:7.4.0 -n $NAMESPACE -- \
  sh -c "echo ruok | nc $ZOOKEEPER_SERVICE $ZK_PORT"

# Test 4: Kafka broker connectivity
echo "✓ Testing Kafka broker connectivity..."
kubectl exec -n $NAMESPACE statefulset/kafka -- \
  kafka-broker-api-versions --bootstrap-server localhost:$KAFKA_PORT

# Test 5: Topic creation test
echo "✓ Testing topic creation..."
TEST_TOPIC="health-check-$(date +%s)"
kubectl exec -n $NAMESPACE statefulset/kafka -- \
  kafka-topics --create --topic $TEST_TOPIC --bootstrap-server localhost:$KAFKA_PORT \
  --partitions 3 --replication-factor 3

# Test 6: Message production test
echo "✓ Testing message production..."
kubectl exec -n $NAMESPACE statefulset/kafka -- \
  sh -c "echo 'Kafka health check message' | kafka-console-producer --topic $TEST_TOPIC --bootstrap-server localhost:$KAFKA_PORT"

# Test 7: Message consumption test
echo "✓ Testing message consumption..."
kubectl exec -n $NAMESPACE statefulset/kafka -- \
  timeout 10s kafka-console-consumer --topic $TEST_TOPIC --bootstrap-server localhost:$KAFKA_PORT \
  --from-beginning --max-messages 1

# Test 8: Cleanup test topic
echo "✓ Cleaning up test topic..."
kubectl exec -n $NAMESPACE statefulset/kafka -- \
  kafka-topics --delete --topic $TEST_TOPIC --bootstrap-server localhost:$KAFKA_PORT

# Test 9: Cluster metadata test
echo "✓ Testing cluster metadata..."
kubectl exec -n $NAMESPACE statefulset/kafka -- \
  kafka-metadata-shell --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --print

echo "✅ Kafka health check completed successfully!"