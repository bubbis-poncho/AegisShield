#!/usr/bin/env python3
"""
Performance testing for AegisShield platform
Tests data ingestion throughput and graph query performance
"""

import asyncio
import time
import statistics
import httpx
import json
from datetime import datetime, timedelta
from typing import List, Dict, Any
import os
import uuid
from concurrent.futures import ThreadPoolExecutor
import psutil
import aiofiles

# Test configuration
API_BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8080")
DATA_INGESTION_URL = os.getenv("DATA_INGESTION_URL", "http://localhost:8060")
GRAPH_ENGINE_URL = os.getenv("GRAPH_ENGINE_URL", "http://localhost:8065")


class PerformanceTestRunner:
    """Main performance test runner"""
    
    def __init__(self):
        self.test_id = str(uuid.uuid4())[:8]
        self.results = {}
        self.auth_token = "perf-test-token-12345"
        self.headers = {"Authorization": f"Bearer {self.auth_token}"}
        
    async def run_all_tests(self):
        """Run all performance tests"""
        print("🚀 Starting AegisShield Performance Tests")
        print(f"Test ID: {self.test_id}")
        print("=" * 60)
        
        # Data ingestion performance tests
        await self._test_transaction_ingestion_throughput()
        await self._test_entity_ingestion_throughput()
        await self._test_bulk_data_ingestion()
        
        # Graph query performance tests
        await self._test_graph_traversal_performance()
        await self._test_pattern_matching_performance()
        await self._test_complex_analytics_performance()
        
        # System resource tests
        await self._test_concurrent_user_load()
        await self._test_memory_usage_patterns()
        
        # Generate performance report
        await self._generate_performance_report()
        
        print("\n✅ Performance testing completed")
        return self.results
    
    async def _test_transaction_ingestion_throughput(self):
        """Test transaction ingestion throughput"""
        print("\n📊 Testing Transaction Ingestion Throughput")
        
        batch_sizes = [10, 50, 100, 500, 1000]
        results = {}
        
        for batch_size in batch_sizes:
            print(f"Testing batch size: {batch_size}")
            
            # Generate test transactions
            transactions = self._generate_test_transactions(batch_size)
            
            # Time the ingestion
            start_time = time.time()
            
            async with httpx.AsyncClient(timeout=300.0) as client:
                tasks = []
                for transaction in transactions:
                    task = client.post(
                        f"{DATA_INGESTION_URL}/transactions",
                        json=transaction,
                        headers=self.headers
                    )
                    tasks.append(task)
                
                responses = await asyncio.gather(*tasks, return_exceptions=True)
            
            end_time = time.time()
            duration = end_time - start_time
            
            # Calculate metrics
            successful_requests = sum(1 for r in responses if not isinstance(r, Exception) and r.status_code in [200, 201])
            throughput = successful_requests / duration
            
            results[batch_size] = {
                "duration": duration,
                "successful_requests": successful_requests,
                "throughput": throughput,
                "requests_per_second": throughput
            }
            
            print(f"  ✓ {successful_requests}/{batch_size} transactions ingested in {duration:.2f}s")
            print(f"  ✓ Throughput: {throughput:.2f} transactions/second")
        
        self.results["transaction_ingestion"] = results
    
    async def _test_entity_ingestion_throughput(self):
        """Test entity ingestion throughput"""
        print("\n🏢 Testing Entity Ingestion Throughput")
        
        batch_sizes = [10, 50, 100, 200]
        results = {}
        
        for batch_size in batch_sizes:
            print(f"Testing batch size: {batch_size}")
            
            # Generate test entities
            entities = self._generate_test_entities(batch_size)
            
            start_time = time.time()
            
            async with httpx.AsyncClient(timeout=300.0) as client:
                tasks = []
                for entity in entities:
                    task = client.post(
                        f"{DATA_INGESTION_URL}/entities",
                        json=entity,
                        headers=self.headers
                    )
                    tasks.append(task)
                
                responses = await asyncio.gather(*tasks, return_exceptions=True)
            
            end_time = time.time()
            duration = end_time - start_time
            
            successful_requests = sum(1 for r in responses if not isinstance(r, Exception) and r.status_code in [200, 201])
            throughput = successful_requests / duration
            
            results[batch_size] = {
                "duration": duration,
                "successful_requests": successful_requests,
                "throughput": throughput
            }
            
            print(f"  ✓ {successful_requests}/{batch_size} entities ingested in {duration:.2f}s")
            print(f"  ✓ Throughput: {throughput:.2f} entities/second")
        
        self.results["entity_ingestion"] = results
    
    async def _test_bulk_data_ingestion(self):
        """Test bulk data ingestion performance"""
        print("\n📦 Testing Bulk Data Ingestion")
        
        # Test large batch ingestion
        bulk_size = 5000
        print(f"Ingesting {bulk_size} transactions in bulk")
        
        transactions = self._generate_test_transactions(bulk_size)
        
        start_time = time.time()
        
        async with httpx.AsyncClient(timeout=600.0) as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/transactions/bulk",
                json={"transactions": transactions},
                headers=self.headers
            )
        
        end_time = time.time()
        duration = end_time - start_time
        
        if response.status_code in [200, 201]:
            result_data = response.json()
            successful_count = result_data.get("successful_count", 0)
            throughput = successful_count / duration
            
            print(f"  ✓ {successful_count}/{bulk_size} transactions bulk ingested in {duration:.2f}s")
            print(f"  ✓ Bulk throughput: {throughput:.2f} transactions/second")
            
            self.results["bulk_ingestion"] = {
                "bulk_size": bulk_size,
                "duration": duration,
                "successful_count": successful_count,
                "throughput": throughput
            }
        else:
            print(f"  ❌ Bulk ingestion failed: {response.status_code}")
    
    async def _test_graph_traversal_performance(self):
        """Test graph traversal performance"""
        print("\n🕸️ Testing Graph Traversal Performance")
        
        # Test different traversal depths
        depths = [1, 2, 3, 4, 5]
        results = {}
        
        for depth in depths:
            print(f"Testing traversal depth: {depth}")
            
            # Perform multiple traversals and measure time
            traversal_times = []
            
            for _ in range(10):  # 10 iterations per depth
                start_time = time.time()
                
                async with httpx.AsyncClient(timeout=60.0) as client:
                    response = await client.post(
                        f"{GRAPH_ENGINE_URL}/traverse",
                        json={
                            "start_entity": f"test_entity_{self.test_id}",
                            "max_depth": depth,
                            "relationship_types": ["TRANSACTED_WITH", "OWNS", "CONTROLS"],
                            "limit": 1000
                        },
                        headers=self.headers
                    )
                
                end_time = time.time()
                
                if response.status_code == 200:
                    traversal_times.append(end_time - start_time)
            
            if traversal_times:
                avg_time = statistics.mean(traversal_times)
                min_time = min(traversal_times)
                max_time = max(traversal_times)
                
                results[depth] = {
                    "average_time": avg_time,
                    "min_time": min_time,
                    "max_time": max_time,
                    "iterations": len(traversal_times)
                }
                
                print(f"  ✓ Depth {depth}: avg {avg_time:.3f}s, min {min_time:.3f}s, max {max_time:.3f}s")
        
        self.results["graph_traversal"] = results
    
    async def _test_pattern_matching_performance(self):
        """Test pattern matching performance"""
        print("\n🔍 Testing Pattern Matching Performance")
        
        patterns = [
            {
                "name": "simple_transfer",
                "description": "Simple A->B transfer",
                "max_results": 100
            },
            {
                "name": "circular_transfer",
                "description": "A->B->C->A circular pattern",
                "max_results": 50
            },
            {
                "name": "fan_out",
                "description": "A->B1,B2,B3... fan out pattern",
                "max_results": 25
            },
            {
                "name": "layered_transfer",
                "description": "Multi-layered transfer pattern",
                "max_results": 10
            }
        ]
        
        results = {}
        
        for pattern in patterns:
            print(f"Testing pattern: {pattern['name']}")
            
            pattern_times = []
            
            for _ in range(5):  # 5 iterations per pattern
                start_time = time.time()
                
                async with httpx.AsyncClient(timeout=120.0) as client:
                    response = await client.post(
                        f"{GRAPH_ENGINE_URL}/patterns/search",
                        json={
                            "pattern_type": pattern["name"],
                            "time_window": "7d",
                            "min_amount": 1000,
                            "max_results": pattern["max_results"]
                        },
                        headers=self.headers
                    )
                
                end_time = time.time()
                
                if response.status_code == 200:
                    pattern_times.append(end_time - start_time)
                    result_data = response.json()
                    pattern["result_count"] = len(result_data.get("matches", []))
            
            if pattern_times:
                avg_time = statistics.mean(pattern_times)
                results[pattern["name"]] = {
                    "average_time": avg_time,
                    "result_count": pattern.get("result_count", 0),
                    "iterations": len(pattern_times)
                }
                
                print(f"  ✓ {pattern['name']}: avg {avg_time:.3f}s, found {pattern.get('result_count', 0)} matches")
        
        self.results["pattern_matching"] = results
    
    async def _test_complex_analytics_performance(self):
        """Test complex analytics performance"""
        print("\n📈 Testing Complex Analytics Performance")
        
        analytics_queries = [
            {
                "name": "risk_scoring",
                "endpoint": "/analytics/risk-score",
                "params": {"entity_id": f"test_entity_{self.test_id}", "window": "30d"}
            },
            {
                "name": "network_analysis",
                "endpoint": "/analytics/network",
                "params": {"center_entity": f"test_entity_{self.test_id}", "radius": 3}
            },
            {
                "name": "anomaly_detection",
                "endpoint": "/analytics/anomalies",
                "params": {"time_window": "7d", "threshold": 0.8}
            },
            {
                "name": "clustering_analysis",
                "endpoint": "/analytics/clusters",
                "params": {"algorithm": "kmeans", "num_clusters": 5}
            }
        ]
        
        results = {}
        
        for query in analytics_queries:
            print(f"Testing analytics: {query['name']}")
            
            query_times = []
            
            for _ in range(3):  # 3 iterations per query
                start_time = time.time()
                
                async with httpx.AsyncClient(timeout=300.0) as client:
                    response = await client.get(
                        f"{API_BASE_URL}{query['endpoint']}",
                        params=query["params"],
                        headers=self.headers
                    )
                
                end_time = time.time()
                
                if response.status_code == 200:
                    query_times.append(end_time - start_time)
            
            if query_times:
                avg_time = statistics.mean(query_times)
                results[query["name"]] = {
                    "average_time": avg_time,
                    "iterations": len(query_times)
                }
                
                print(f"  ✓ {query['name']}: avg {avg_time:.3f}s")
        
        self.results["complex_analytics"] = results
    
    async def _test_concurrent_user_load(self):
        """Test concurrent user load"""
        print("\n👥 Testing Concurrent User Load")
        
        concurrent_users = [5, 10, 20, 50]
        results = {}
        
        for user_count in concurrent_users:
            print(f"Testing {user_count} concurrent users")
            
            async def simulate_user_session():
                """Simulate a user session with multiple API calls"""
                session_times = []
                
                async with httpx.AsyncClient(timeout=60.0) as client:
                    # Login
                    start = time.time()
                    await client.post(f"{API_BASE_URL}/auth/login", 
                                    json={"username": f"testuser_{uuid.uuid4()}", "password": "testpass"})
                    session_times.append(time.time() - start)
                    
                    # Dashboard data
                    start = time.time()
                    await client.get(f"{API_BASE_URL}/dashboard", headers=self.headers)
                    session_times.append(time.time() - start)
                    
                    # Search
                    start = time.time()
                    await client.get(f"{API_BASE_URL}/search", 
                                   params={"query": "test"}, headers=self.headers)
                    session_times.append(time.time() - start)
                    
                    # Investigation list
                    start = time.time()
                    await client.get(f"{API_BASE_URL}/investigations", headers=self.headers)
                    session_times.append(time.time() - start)
                
                return session_times
            
            start_time = time.time()
            
            # Run concurrent user sessions
            tasks = [simulate_user_session() for _ in range(user_count)]
            session_results = await asyncio.gather(*tasks, return_exceptions=True)
            
            end_time = time.time()
            total_duration = end_time - start_time
            
            successful_sessions = [r for r in session_results if not isinstance(r, Exception)]
            
            if successful_sessions:
                all_times = [time for session in successful_sessions for time in session]
                avg_response_time = statistics.mean(all_times)
                
                results[user_count] = {
                    "total_duration": total_duration,
                    "successful_sessions": len(successful_sessions),
                    "average_response_time": avg_response_time,
                    "concurrent_users": user_count
                }
                
                print(f"  ✓ {len(successful_sessions)}/{user_count} sessions completed")
                print(f"  ✓ Average response time: {avg_response_time:.3f}s")
        
        self.results["concurrent_load"] = results
    
    async def _test_memory_usage_patterns(self):
        """Test memory usage patterns"""
        print("\n💾 Testing Memory Usage Patterns")
        
        # Monitor memory usage during data ingestion
        initial_memory = psutil.virtual_memory().percent
        
        # Ingest large dataset
        large_dataset = self._generate_test_transactions(10000)
        
        memory_samples = []
        
        async def monitor_memory():
            for _ in range(60):  # Monitor for 60 seconds
                memory_samples.append(psutil.virtual_memory().percent)
                await asyncio.sleep(1)
        
        async def ingest_data():
            async with httpx.AsyncClient(timeout=600.0) as client:
                for i in range(0, len(large_dataset), 100):
                    batch = large_dataset[i:i+100]
                    await client.post(
                        f"{DATA_INGESTION_URL}/transactions/bulk",
                        json={"transactions": batch},
                        headers=self.headers
                    )
                    await asyncio.sleep(0.1)  # Small delay between batches
        
        # Run memory monitoring and data ingestion concurrently
        await asyncio.gather(monitor_memory(), ingest_data())
        
        final_memory = psutil.virtual_memory().percent
        max_memory = max(memory_samples) if memory_samples else initial_memory
        
        self.results["memory_usage"] = {
            "initial_memory_percent": initial_memory,
            "final_memory_percent": final_memory,
            "max_memory_percent": max_memory,
            "memory_increase": final_memory - initial_memory,
            "peak_memory_increase": max_memory - initial_memory
        }
        
        print(f"  ✓ Initial memory: {initial_memory:.1f}%")
        print(f"  ✓ Final memory: {final_memory:.1f}%")
        print(f"  ✓ Peak memory: {max_memory:.1f}%")
    
    async def _generate_performance_report(self):
        """Generate comprehensive performance report"""
        print("\n📋 Generating Performance Report")
        
        report = {
            "test_id": self.test_id,
            "timestamp": datetime.utcnow().isoformat(),
            "test_results": self.results,
            "summary": {
                "status": "COMPLETED",
                "total_tests": len(self.results),
                "recommendations": []
            }
        }
        
        # Add performance recommendations
        recommendations = []
        
        # Check transaction ingestion performance
        if "transaction_ingestion" in self.results:
            max_throughput = max(r["throughput"] for r in self.results["transaction_ingestion"].values())
            if max_throughput < 100:
                recommendations.append("Consider optimizing transaction ingestion pipeline for higher throughput")
        
        # Check graph traversal performance
        if "graph_traversal" in self.results:
            depth_5_time = self.results["graph_traversal"].get(5, {}).get("average_time", 0)
            if depth_5_time > 5.0:
                recommendations.append("Graph traversal at depth 5 is slow, consider query optimization")
        
        # Check memory usage
        if "memory_usage" in self.results:
            memory_increase = self.results["memory_usage"]["memory_increase"]
            if memory_increase > 20:
                recommendations.append("High memory usage increase detected, review memory management")
        
        report["summary"]["recommendations"] = recommendations
        
        # Save report to file
        report_filename = f"performance_report_{self.test_id}.json"
        async with aiofiles.open(report_filename, 'w') as f:
            await f.write(json.dumps(report, indent=2))
        
        print(f"  ✓ Performance report saved to: {report_filename}")
        
        # Print summary
        print("\n" + "="*60)
        print("PERFORMANCE TEST SUMMARY")
        print("="*60)
        
        for test_name, test_results in self.results.items():
            print(f"\n{test_name.upper()}:")
            if isinstance(test_results, dict):
                if "throughput" in str(test_results):
                    print(f"  Max throughput: {max(r.get('throughput', 0) for r in test_results.values() if isinstance(r, dict)):.2f} ops/sec")
                if "average_time" in str(test_results):
                    avg_times = [r.get('average_time', 0) for r in test_results.values() if isinstance(r, dict) and 'average_time' in r]
                    if avg_times:
                        print(f"  Average response time: {statistics.mean(avg_times):.3f}s")
        
        if recommendations:
            print(f"\nRECOMMENDATIONS:")
            for i, rec in enumerate(recommendations, 1):
                print(f"  {i}. {rec}")
        
        print("\n✅ Performance testing completed successfully!")
    
    def _generate_test_transactions(self, count: int) -> List[Dict]:
        """Generate test transactions"""
        transactions = []
        
        for i in range(count):
            transactions.append({
                "transaction_id": f"PERF_{self.test_id}_{i:06d}",
                "sender_id": f"sender_{i % 100}",
                "receiver_id": f"receiver_{(i + 50) % 100}",
                "amount": float(1000 + (i % 50000)),
                "currency": "USD",
                "timestamp": (datetime.utcnow() - timedelta(days=i % 30)).isoformat(),
                "transaction_type": ["wire_transfer", "ach", "check", "cash"][i % 4],
                "source_system": "performance_test",
                "metadata": {
                    "test_transaction": True,
                    "test_id": self.test_id,
                    "batch_index": i
                }
            })
        
        return transactions
    
    def _generate_test_entities(self, count: int) -> List[Dict]:
        """Generate test entities"""
        entities = []
        
        entity_types = ["individual", "organization", "financial_institution"]
        countries = ["US", "UK", "CA", "AU", "DE", "FR", "JP"]
        
        for i in range(count):
            entities.append({
                "name": f"Test Entity {self.test_id}_{i:06d}",
                "entity_type": entity_types[i % len(entity_types)],
                "country": countries[i % len(countries)],
                "metadata": {
                    "test_entity": True,
                    "test_id": self.test_id,
                    "batch_index": i
                }
            })
        
        return entities


async def main():
    """Run performance tests"""
    runner = PerformanceTestRunner()
    await runner.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())