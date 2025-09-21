#!/usr/bin/env python3
"""
Performance testing for graph queries and Neo4j operations
Tests graph traversal performance, complex queries, and scaling characteristics
"""

import asyncio
import pytest
import time
import statistics
import json
from datetime import datetime, timedelta
from typing import Dict, List, Any, Tuple
from dataclasses import dataclass
import uuid
import neo4j
from neo4j import GraphDatabase
import concurrent.futures
import psutil
import os

# Graph performance test configuration
@dataclass
class GraphTestConfig:
    """Configuration for graph performance testing"""
    neo4j_uri: str = "bolt://localhost:7687"
    neo4j_user: str = "neo4j"
    neo4j_password: str = "test_password"
    api_base_url: str = "http://localhost:8080"
    
    # Test parameters
    entity_count: int = 10000
    relationship_count: int = 50000
    concurrent_queries: int = 20
    test_duration: int = 300  # 5 minutes
    
    # Performance targets
    max_simple_query_ms: float = 100.0
    max_complex_query_ms: float = 1000.0
    max_traversal_query_ms: float = 2000.0
    max_error_rate: float = 0.01
    min_throughput_qps: float = 100.0

@dataclass
class GraphMetrics:
    """Performance metrics for graph operations"""
    query_type: str = ""
    total_queries: int = 0
    successful_queries: int = 0
    failed_queries: int = 0
    response_times: List[float] = None
    error_rate: float = 0.0
    avg_response_time: float = 0.0
    p50_response_time: float = 0.0
    p95_response_time: float = 0.0
    p99_response_time: float = 0.0
    throughput_qps: float = 0.0
    
    def __post_init__(self):
        if self.response_times is None:
            self.response_times = []

class GraphPerformanceTest:
    """Performance testing suite for graph operations"""
    
    def __init__(self, config: GraphTestConfig):
        self.config = config
        self.driver = None
        self.test_id = f"graph_perf_{int(time.time())}"
        self.test_entities = []
        self.test_relationships = []
        
    async def setup(self):
        """Setup test environment and data"""
        print("Setting up graph performance test environment...")
        
        # Connect to Neo4j
        self.driver = GraphDatabase.driver(
            self.config.neo4j_uri,
            auth=(self.config.neo4j_user, self.config.neo4j_password)
        )
        
        # Verify connection
        with self.driver.session() as session:
            result = session.run("RETURN 1")
            assert result.single()[0] == 1, "Neo4j connection failed"
            
        # Clear existing test data
        await self.cleanup_test_data()
        
        # Create test data
        await self.create_test_graph()
        
        print(f"✓ Test environment ready with {self.config.entity_count} entities and {self.config.relationship_count} relationships")
        
    async def cleanup(self):
        """Cleanup test environment"""
        await self.cleanup_test_data()
        if self.driver:
            self.driver.close()
            
    async def cleanup_test_data(self):
        """Remove test data from graph"""
        with self.driver.session() as session:
            session.run(f"MATCH (n) WHERE n.test_id = '{self.test_id}' DETACH DELETE n")
            
    async def create_test_graph(self):
        """Create test graph with various entity types and relationships"""
        print("Creating test graph data...")
        
        with self.driver.session() as session:
            # Create indexes for performance
            session.run("CREATE INDEX person_ssn IF NOT EXISTS FOR (p:Person) ON (p.ssn)")
            session.run("CREATE INDEX account_number IF NOT EXISTS FOR (a:Account) ON (a.account_number)")
            session.run("CREATE INDEX transaction_id IF NOT EXISTS FOR (t:Transaction) ON (t.transaction_id)")
            session.run("CREATE INDEX organization_tax_id IF NOT EXISTS FOR (o:Organization) ON (o.tax_id)")
            
            # Create person entities
            persons_data = []
            for i in range(self.config.entity_count // 4):
                person = {
                    "test_id": self.test_id,
                    "person_id": f"{self.test_id}_person_{i}",
                    "name": f"Person {i}",
                    "ssn": f"{100000000 + i:09d}",
                    "risk_score": (i % 100) / 100.0,
                    "created_at": datetime.now().isoformat()
                }
                persons_data.append(person)
                
            session.run(
                "UNWIND $persons as person "
                "CREATE (p:Person:Entity) SET p = person",
                persons=persons_data
            )
            
            # Create organization entities  
            orgs_data = []
            for i in range(self.config.entity_count // 4):
                org = {
                    "test_id": self.test_id,
                    "org_id": f"{self.test_id}_org_{i}",
                    "name": f"Organization {i}",
                    "tax_id": f"{10000000 + i:08d}",
                    "industry": ["finance", "tech", "retail", "manufacturing"][i % 4],
                    "risk_score": (i % 100) / 100.0,
                    "created_at": datetime.now().isoformat()
                }
                orgs_data.append(org)
                
            session.run(
                "UNWIND $orgs as org "
                "CREATE (o:Organization:Entity) SET o = org",
                orgs=orgs_data
            )
            
            # Create account entities
            accounts_data = []
            for i in range(self.config.entity_count // 4):
                account = {
                    "test_id": self.test_id,
                    "account_id": f"{self.test_id}_account_{i}",
                    "account_number": f"ACC{1000000 + i:07d}",
                    "bank": f"Bank {i % 10}",
                    "balance": float(1000 + (i * 100)),
                    "created_at": datetime.now().isoformat()
                }
                accounts_data.append(account)
                
            session.run(
                "UNWIND $accounts as account "
                "CREATE (a:Account:Entity) SET a = account",
                accounts=accounts_data
            )
            
            # Create transaction entities
            transactions_data = []
            for i in range(self.config.entity_count // 4):
                transaction = {
                    "test_id": self.test_id,
                    "transaction_id": f"{self.test_id}_txn_{i}",
                    "amount": float(100 + (i % 10000)),
                    "currency": "USD",
                    "timestamp": (datetime.now() - timedelta(days=i % 365)).isoformat(),
                    "status": ["completed", "pending", "failed"][i % 3]
                }
                transactions_data.append(transaction)
                
            session.run(
                "UNWIND $transactions as txn "
                "CREATE (t:Transaction:Entity) SET t = txn",
                transactions=transactions_data
            )
            
            # Create relationships
            print("Creating relationships...")
            
            # Person -> Account ownership
            session.run(f"""
                MATCH (p:Person {{test_id: '{self.test_id}'}})
                MATCH (a:Account {{test_id: '{self.test_id}'}})
                WHERE toInteger(split(p.person_id, '_')[3]) = toInteger(split(a.account_id, '_')[3])
                CREATE (p)-[:OWNS {{created_at: datetime()}}]->(a)
            """)
            
            # Organization -> Account ownership  
            session.run(f"""
                MATCH (o:Organization {{test_id: '{self.test_id}'}})
                MATCH (a:Account {{test_id: '{self.test_id}'}})
                WHERE toInteger(split(o.org_id, '_')[3]) % 2 = toInteger(split(a.account_id, '_')[3]) % 2
                AND toInteger(split(o.org_id, '_')[3]) < {self.config.entity_count // 8}
                CREATE (o)-[:OWNS {{created_at: datetime()}}]->(a)
            """)
            
            # Transaction -> Account relationships
            session.run(f"""
                MATCH (t:Transaction {{test_id: '{self.test_id}'}})
                MATCH (a1:Account {{test_id: '{self.test_id}'}})
                MATCH (a2:Account {{test_id: '{self.test_id}'}})
                WHERE toInteger(split(t.transaction_id, '_')[3]) = toInteger(split(a1.account_id, '_')[3])
                AND toInteger(split(a2.account_id, '_')[3]) = (toInteger(split(t.transaction_id, '_')[3]) + 1) % {self.config.entity_count // 4}
                CREATE (a1)-[:SENT {{transaction_id: t.transaction_id, amount: t.amount}}]->(t)
                CREATE (t)-[:RECEIVED {{transaction_id: t.transaction_id, amount: t.amount}}]->(a2)
            """)
            
            # Person relationships (family, business)
            session.run(f"""
                MATCH (p1:Person {{test_id: '{self.test_id}'}})
                MATCH (p2:Person {{test_id: '{self.test_id}'}})
                WHERE toInteger(split(p1.person_id, '_')[3]) < toInteger(split(p2.person_id, '_')[3])
                AND (toInteger(split(p1.person_id, '_')[3]) + toInteger(split(p2.person_id, '_')[3])) % 10 = 0
                CREATE (p1)-[:KNOWS {{relationship_type: 'business', strength: 0.7}}]->(p2)
            """)
            
        print("✓ Test graph created successfully")
        
    def execute_query_with_timing(self, session, query: str, parameters: Dict = None) -> Tuple[bool, float, Any]:
        """Execute a query and measure execution time"""
        start_time = time.time()
        
        try:
            result = session.run(query, parameters or {})
            records = list(result)  # Consume result
            
            end_time = time.time()
            execution_time = (end_time - start_time) * 1000  # Convert to milliseconds
            
            return True, execution_time, records
            
        except Exception as e:
            end_time = time.time()
            execution_time = (end_time - start_time) * 1000
            print(f"Query failed: {str(e)}")
            return False, execution_time, None
    
    async def test_simple_queries(self) -> GraphMetrics:
        """Test simple node and relationship queries"""
        print("Testing simple queries...")
        
        metrics = GraphMetrics(query_type="simple_queries")
        queries = [
            # Simple node lookups
            f"MATCH (p:Person {{test_id: '{self.test_id}'}}) WHERE p.risk_score > 0.5 RETURN count(p)",
            f"MATCH (a:Account {{test_id: '{self.test_id}'}}) WHERE a.balance > 5000 RETURN count(a)",
            f"MATCH (t:Transaction {{test_id: '{self.test_id}'}}) WHERE t.amount > 1000 RETURN count(t)",
            
            # Simple relationship queries
            f"MATCH (p:Person {{test_id: '{self.test_id}'}})-[:OWNS]->(a:Account) RETURN count(*)",
            f"MATCH (o:Organization {{test_id: '{self.test_id}'}})-[:OWNS]->(a:Account) RETURN count(*)",
            
            # Range queries
            f"MATCH (p:Person {{test_id: '{self.test_id}'}}) WHERE p.risk_score BETWEEN 0.3 AND 0.7 RETURN count(p)",
        ]
        
        with self.driver.session() as session:
            for i in range(100):  # Execute each query multiple times
                query = queries[i % len(queries)]
                
                success, execution_time, result = self.execute_query_with_timing(session, query)
                
                metrics.total_queries += 1
                metrics.response_times.append(execution_time)
                
                if success:
                    metrics.successful_queries += 1
                else:
                    metrics.failed_queries += 1
                    
        self._calculate_metrics(metrics)
        
        print(f"Simple queries - Avg: {metrics.avg_response_time:.2f}ms, P95: {metrics.p95_response_time:.2f}ms")
        return metrics
    
    async def test_complex_queries(self) -> GraphMetrics:
        """Test complex queries with aggregations and filtering"""
        print("Testing complex queries...")
        
        metrics = GraphMetrics(query_type="complex_queries")
        queries = [
            # Complex aggregations
            f"""
            MATCH (p:Person {{test_id: '{self.test_id}'}})-[:OWNS]->(a:Account)-[:SENT]->(t:Transaction)
            WHERE t.amount > 1000
            RETURN p.name, sum(t.amount) as total_amount, count(t) as transaction_count
            ORDER BY total_amount DESC
            LIMIT 10
            """,
            
            # Multi-hop patterns
            f"""
            MATCH (p1:Person {{test_id: '{self.test_id}'}})-[:OWNS]->(a1:Account)-[:SENT]->(t:Transaction)-[:RECEIVED]->(a2:Account)<-[:OWNS]-(p2:Person)
            WHERE p1 <> p2 AND t.amount > 5000
            RETURN p1.name, p2.name, count(t) as transactions, sum(t.amount) as total_amount
            ORDER BY total_amount DESC
            LIMIT 20
            """,
            
            # Risk analysis queries
            f"""
            MATCH (e:Entity {{test_id: '{self.test_id}'}})
            WHERE e.risk_score IS NOT NULL
            WITH e.risk_score as risk, count(e) as entity_count
            ORDER BY risk
            RETURN collect({{risk: risk, count: entity_count}}) as risk_distribution
            """,
            
            # Network analysis
            f"""
            MATCH (p:Person {{test_id: '{self.test_id}'}})-[:KNOWS*1..2]-(connected:Person)
            WHERE p.risk_score > 0.8
            RETURN p.name, count(DISTINCT connected) as network_size, avg(connected.risk_score) as avg_network_risk
            ORDER BY network_size DESC
            LIMIT 15
            """,
        ]
        
        with self.driver.session() as session:
            for i in range(50):  # Execute each query multiple times
                query = queries[i % len(queries)]
                
                success, execution_time, result = self.execute_query_with_timing(session, query)
                
                metrics.total_queries += 1
                metrics.response_times.append(execution_time)
                
                if success:
                    metrics.successful_queries += 1
                else:
                    metrics.failed_queries += 1
                    
        self._calculate_metrics(metrics)
        
        print(f"Complex queries - Avg: {metrics.avg_response_time:.2f}ms, P95: {metrics.p95_response_time:.2f}ms")
        return metrics
        
    async def test_graph_traversal(self) -> GraphMetrics:
        """Test graph traversal and pathfinding queries"""
        print("Testing graph traversal queries...")
        
        metrics = GraphMetrics(query_type="graph_traversal")
        
        # Get sample entity IDs for traversal
        with self.driver.session() as session:
            result = session.run(f"""
                MATCH (p:Person {{test_id: '{self.test_id}'}})
                RETURN p.person_id as id
                LIMIT 20
            """)
            person_ids = [record["id"] for record in result]
            
        queries = [
            # Variable length paths
            lambda pid: f"""
            MATCH (start:Person {{person_id: '{pid}'}})-[*1..3]-(connected)
            RETURN count(DISTINCT connected) as connected_entities
            """,
            
            # Shortest paths
            lambda pid1, pid2: f"""
            MATCH (start:Person {{person_id: '{pid1}'}}), (end:Person {{person_id: '{pid2}'}})
            MATCH path = shortestPath((start)-[*1..5]-(end))
            RETURN length(path) as path_length, nodes(path) as path_nodes
            """,
            
            # Transaction flow analysis
            lambda pid: f"""
            MATCH (p:Person {{person_id: '{pid}'}})-[:OWNS]->(a:Account)-[:SENT]->(t:Transaction)-[:RECEIVED]->(a2:Account)<-[:OWNS]-(p2:Person)
            WHERE t.amount > 1000
            RETURN p2.name, sum(t.amount) as total_flow, count(t) as transaction_count
            ORDER BY total_flow DESC
            LIMIT 10
            """,
            
            # Community detection (local clustering)
            lambda pid: f"""
            MATCH (center:Person {{person_id: '{pid}'}})-[:KNOWS]-(neighbor1:Person)-[:KNOWS]-(neighbor2:Person)
            WHERE neighbor1 <> neighbor2 AND (center)-[:KNOWS]-(neighbor2)
            RETURN count(DISTINCT neighbor1) as triangle_count
            """,
        ]
        
        with self.driver.session() as session:
            for i in range(100):
                # Select query type and parameters
                if i % 4 == 0:
                    # Variable length paths
                    pid = person_ids[i % len(person_ids)]
                    query = queries[0](pid)
                elif i % 4 == 1:
                    # Shortest paths
                    pid1 = person_ids[i % len(person_ids)]
                    pid2 = person_ids[(i + 5) % len(person_ids)]
                    query = queries[1](pid1, pid2)
                elif i % 4 == 2:
                    # Transaction flow
                    pid = person_ids[i % len(person_ids)]
                    query = queries[2](pid)
                else:
                    # Community detection
                    pid = person_ids[i % len(person_ids)]
                    query = queries[3](pid)
                
                success, execution_time, result = self.execute_query_with_timing(session, query)
                
                metrics.total_queries += 1
                metrics.response_times.append(execution_time)
                
                if success:
                    metrics.successful_queries += 1
                else:
                    metrics.failed_queries += 1
                    
        self._calculate_metrics(metrics)
        
        print(f"Graph traversal - Avg: {metrics.avg_response_time:.2f}ms, P95: {metrics.p95_response_time:.2f}ms")
        return metrics
    
    async def test_concurrent_queries(self) -> GraphMetrics:
        """Test concurrent query execution"""
        print(f"Testing concurrent queries with {self.config.concurrent_queries} threads...")
        
        metrics = GraphMetrics(query_type="concurrent_queries")
        
        # Mixed workload queries
        query_templates = [
            f"MATCH (p:Person {{test_id: '{self.test_id}'}}) WHERE p.risk_score > rand() RETURN count(p)",
            f"MATCH (a:Account {{test_id: '{self.test_id}'}}) WHERE a.balance > (rand() * 10000) RETURN count(a)",
            f"MATCH (p:Person {{test_id: '{self.test_id}'}})-[:OWNS]->(a:Account) RETURN p.name, a.balance ORDER BY a.balance DESC LIMIT 5",
            f"MATCH (t:Transaction {{test_id: '{self.test_id}'}}) WHERE t.amount > (rand() * 5000) RETURN avg(t.amount)",
        ]
        
        def worker_thread():
            """Worker function for concurrent execution"""
            thread_metrics = GraphMetrics()
            
            with self.driver.session() as session:
                for _ in range(25):  # Each thread executes 25 queries
                    query = query_templates[_ % len(query_templates)]
                    
                    success, execution_time, result = self.execute_query_with_timing(session, query)
                    
                    thread_metrics.total_queries += 1
                    thread_metrics.response_times.append(execution_time)
                    
                    if success:
                        thread_metrics.successful_queries += 1
                    else:
                        thread_metrics.failed_queries += 1
                        
            return thread_metrics
        
        # Execute concurrent queries
        start_time = time.time()
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=self.config.concurrent_queries) as executor:
            futures = [executor.submit(worker_thread) for _ in range(self.config.concurrent_queries)]
            thread_results = [future.result() for future in concurrent.futures.as_completed(futures)]
        
        end_time = time.time()
        
        # Aggregate results
        for thread_result in thread_results:
            metrics.total_queries += thread_result.total_queries
            metrics.successful_queries += thread_result.successful_queries
            metrics.failed_queries += thread_result.failed_queries
            metrics.response_times.extend(thread_result.response_times)
        
        metrics.throughput_qps = metrics.total_queries / (end_time - start_time)
        
        self._calculate_metrics(metrics)
        
        print(f"Concurrent queries - Throughput: {metrics.throughput_qps:.2f} QPS, Avg: {metrics.avg_response_time:.2f}ms")
        return metrics
    
    def _calculate_metrics(self, metrics: GraphMetrics):
        """Calculate statistical metrics from response times"""
        if metrics.response_times:
            metrics.avg_response_time = statistics.mean(metrics.response_times)
            metrics.p50_response_time = statistics.median(metrics.response_times)
            
            if len(metrics.response_times) >= 20:
                metrics.p95_response_time = statistics.quantiles(metrics.response_times, n=20)[18]
            if len(metrics.response_times) >= 100:
                metrics.p99_response_time = statistics.quantiles(metrics.response_times, n=100)[98]
                
        metrics.error_rate = metrics.failed_queries / max(metrics.total_queries, 1)
    
    def validate_performance_requirements(self, all_metrics: List[GraphMetrics]) -> bool:
        """Validate that performance requirements are met"""
        print("\n--- Graph Performance Requirements Validation ---")
        
        validation_passed = True
        
        for metrics in all_metrics:
            print(f"\n{metrics.query_type}:")
            
            # Check error rate
            if metrics.error_rate > self.config.max_error_rate:
                print(f"  ❌ Error rate too high: {metrics.error_rate:.2%} > {self.config.max_error_rate:.2%}")
                validation_passed = False
            else:
                print(f"  ✅ Error rate acceptable: {metrics.error_rate:.2%}")
            
            # Check response times based on query type
            if metrics.query_type == "simple_queries":
                max_allowed = self.config.max_simple_query_ms
            elif metrics.query_type == "complex_queries":
                max_allowed = self.config.max_complex_query_ms
            else:
                max_allowed = self.config.max_traversal_query_ms
                
            if metrics.p95_response_time > max_allowed:
                print(f"  ❌ 95th percentile latency too high: {metrics.p95_response_time:.2f}ms > {max_allowed}ms")
                validation_passed = False
            else:
                print(f"  ✅ 95th percentile latency acceptable: {metrics.p95_response_time:.2f}ms")
            
            # Check throughput for concurrent queries
            if metrics.query_type == "concurrent_queries":
                if metrics.throughput_qps < self.config.min_throughput_qps:
                    print(f"  ❌ Throughput too low: {metrics.throughput_qps:.2f} QPS < {self.config.min_throughput_qps} QPS")
                    validation_passed = False
                else:
                    print(f"  ✅ Throughput acceptable: {metrics.throughput_qps:.2f} QPS")
        
        return validation_passed
    
    async def run_full_graph_performance_test(self) -> Dict[str, Any]:
        """Execute complete graph performance test suite"""
        print("🚀 Starting Graph Performance Test")
        print("=" * 60)
        
        try:
            await self.setup()
            
            # Execute test phases
            simple_metrics = await self.test_simple_queries()
            complex_metrics = await self.test_complex_queries()
            traversal_metrics = await self.test_graph_traversal()
            concurrent_metrics = await self.test_concurrent_queries()
            
            all_metrics = [simple_metrics, complex_metrics, traversal_metrics, concurrent_metrics]
            
            # Validate requirements
            validation_passed = self.validate_performance_requirements(all_metrics)
            
            # Generate summary
            total_queries = sum(m.total_queries for m in all_metrics)
            total_successful = sum(m.successful_queries for m in all_metrics)
            avg_response_time = statistics.mean([m.avg_response_time for m in all_metrics])
            
            summary = {
                "test_id": self.test_id,
                "entity_count": self.config.entity_count,
                "relationship_count": self.config.relationship_count,
                "total_queries": total_queries,
                "total_successful": total_successful,
                "average_response_time": avg_response_time,
                "validation_passed": validation_passed,
                "metrics_by_type": {m.query_type: m for m in all_metrics}
            }
            
            print("\n" + "=" * 60)
            print("📊 GRAPH PERFORMANCE TEST SUMMARY")
            print("=" * 60)
            print(f"Test ID: {self.test_id}")
            print(f"Graph size: {self.config.entity_count:,} entities, {self.config.relationship_count:,} relationships")
            print(f"Total queries: {total_queries:,}")
            print(f"Successful queries: {total_successful:,}")
            print(f"Average response time: {avg_response_time:.2f}ms")
            print(f"Validation status: {'PASSED' if validation_passed else 'FAILED'}")
            
            if validation_passed:
                print("🎉 All graph performance requirements met!")
            else:
                print("⚠️  Some performance requirements not met - check details above")
                
            return summary
            
        finally:
            await self.cleanup()


# Test execution functions
@pytest.mark.asyncio
async def test_graph_simple_query_performance():
    """Test simple graph query performance"""
    config = GraphTestConfig(entity_count=1000, relationship_count=5000)
    test_suite = GraphPerformanceTest(config)
    
    await test_suite.setup()
    try:
        metrics = await test_suite.test_simple_queries()
        
        assert metrics.error_rate < 0.05, f"Error rate too high: {metrics.error_rate}"
        assert metrics.p95_response_time < 200, f"95th percentile latency too high: {metrics.p95_response_time}ms"
        
    finally:
        await test_suite.cleanup()

@pytest.mark.asyncio
async def test_graph_complex_query_performance():
    """Test complex graph query performance"""
    config = GraphTestConfig(entity_count=1000, relationship_count=5000)
    test_suite = GraphPerformanceTest(config)
    
    await test_suite.setup()
    try:
        metrics = await test_suite.test_complex_queries()
        
        assert metrics.error_rate < 0.05, f"Error rate too high: {metrics.error_rate}"
        assert metrics.p95_response_time < 1500, f"95th percentile latency too high: {metrics.p95_response_time}ms"
        
    finally:
        await test_suite.cleanup()

@pytest.mark.asyncio
async def test_graph_full_performance_suite():
    """Execute the complete graph performance test suite"""
    config = GraphTestConfig()
    test_suite = GraphPerformanceTest(config)
    
    summary = await test_suite.run_full_graph_performance_test()
    
    assert summary["validation_passed"], "Graph performance validation failed"
    assert summary["total_queries"] > 500, "Not enough queries executed during test"


if __name__ == "__main__":
    """Direct execution for debugging"""
    async def main():
        config = GraphTestConfig()
        test_suite = GraphPerformanceTest(config)
        await test_suite.run_full_graph_performance_test()
        
    asyncio.run(main())