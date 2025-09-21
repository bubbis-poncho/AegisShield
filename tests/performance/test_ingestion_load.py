#!/usr/bin/env python3
"""
Performance testing for data ingestion service
Tests throughput, latency, and resource utilization under various load conditions
"""

import asyncio
import pytest
import time
import statistics
import json
import httpx
import psutil
import os
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timedelta
from typing import Dict, List, Any, Tuple
from dataclasses import dataclass
import uuid

# Performance test configuration
@dataclass
class LoadTestConfig:
    """Configuration for load testing"""
    api_base_url: str = "http://localhost:8080"
    data_ingestion_url: str = "http://localhost:8060" 
    concurrent_users: int = 50
    test_duration: int = 300  # 5 minutes
    ramp_up_time: int = 60   # 1 minute
    target_tps: int = 1000   # transactions per second
    max_latency_ms: float = 500.0
    max_error_rate: float = 0.01  # 1%
    max_cpu_usage: float = 0.80   # 80%
    max_memory_usage: float = 0.85 # 85%

@dataclass
class PerformanceMetrics:
    """Performance metrics collected during testing"""
    total_requests: int = 0
    successful_requests: int = 0
    failed_requests: int = 0
    response_times: List[float] = None
    throughput_per_second: List[int] = None
    error_rate: float = 0.0
    avg_response_time: float = 0.0
    p95_response_time: float = 0.0
    p99_response_time: float = 0.0
    max_cpu_usage: float = 0.0
    max_memory_usage: float = 0.0
    
    def __post_init__(self):
        if self.response_times is None:
            self.response_times = []
        if self.throughput_per_second is None:
            self.throughput_per_second = []

class DataIngestionLoadTest:
    """Load testing suite for data ingestion service"""
    
    def __init__(self, config: LoadTestConfig):
        self.config = config
        self.metrics = PerformanceMetrics()
        self.test_start_time = None
        self.test_id = f"load_test_{int(time.time())}"
        self.monitoring_task = None
        self.request_counts = []
        
    def generate_transaction_batch(self, batch_size: int = 100) -> Dict[str, Any]:
        """Generate a batch of synthetic transaction data"""
        transactions = []
        
        for i in range(batch_size):
            transaction = {
                "id": f"{self.test_id}_txn_{uuid.uuid4().hex[:8]}",
                "source_account": f"ACC{1000000 + (i % 10000)}",
                "destination_account": f"ACC{2000000 + ((i + 1000) % 10000)}",
                "amount": round(1.0 + (i % 50000), 2),
                "currency": "USD",
                "transaction_date": (datetime.now() - timedelta(seconds=i % 3600)).isoformat(),
                "description": f"Test transaction {i}",
                "channel": "online" if i % 2 == 0 else "wire",
                "country_code": "US",
                "merchant_category": f"MCC_{4000 + (i % 100)}",
                "reference_id": f"REF_{uuid.uuid4().hex[:12]}"
            }
            transactions.append(transaction)
            
        return {
            "transactions": transactions,
            "batch_id": f"{self.test_id}_batch_{uuid.uuid4().hex[:8]}",
            "source": "load_test",
            "timestamp": datetime.now().isoformat()
        }
    
    def generate_entity_data(self, entity_count: int = 50) -> Dict[str, Any]:
        """Generate synthetic entity data for testing"""
        persons = []
        organizations = []
        accounts = []
        
        for i in range(entity_count):
            # Generate person
            person = {
                "id": f"{self.test_id}_person_{i}",
                "name": f"Test Person {i}",
                "ssn": f"{100000000 + i:09d}",
                "address": f"{i} Test Street, City {i % 50}, State {i % 50}",
                "risk_profile": ["low", "medium", "high"][i % 3],
                "customer_since": (datetime.now() - timedelta(days=i % 1000)).isoformat(),
                "phone": f"+1555{1000000 + i:07d}",
                "email": f"person{i}@test.com"
            }
            persons.append(person)
            
            # Generate organization
            org = {
                "id": f"{self.test_id}_org_{i}",
                "name": f"Test Corp {i}",
                "tax_id": f"{10000000 + i:08d}",
                "address": f"{i} Business Blvd, City {i % 50}, State {i % 50}",
                "industry": ["technology", "finance", "retail", "manufacturing"][i % 4],
                "incorporation_date": (datetime.now() - timedelta(days=i % 2000)).isoformat(),
                "employee_count": 10 + (i % 1000)
            }
            organizations.append(org)
            
            # Generate accounts
            for j in range(2):  # 2 accounts per entity
                account = {
                    "id": f"{self.test_id}_acc_{i}_{j}",
                    "account_number": f"ACC{1000000 + i * 10 + j}",
                    "holder_id": person["id"] if j == 0 else org["id"],
                    "bank": f"Test Bank {i % 10}",
                    "type": "checking" if j == 0 else "business",
                    "opened_date": (datetime.now() - timedelta(days=i % 500)).isoformat(),
                    "balance": round(1000.0 + (i * 1000), 2),
                    "currency": "USD"
                }
                accounts.append(account)
        
        return {
            "persons": persons,
            "organizations": organizations, 
            "accounts": accounts
        }
    
    async def monitor_system_resources(self):
        """Monitor CPU and memory usage during testing"""
        while self.test_start_time and (time.time() - self.test_start_time) < self.config.test_duration:
            cpu_percent = psutil.cpu_percent(interval=1)
            memory_percent = psutil.virtual_memory().percent / 100.0
            
            self.metrics.max_cpu_usage = max(self.metrics.max_cpu_usage, cpu_percent / 100.0)
            self.metrics.max_memory_usage = max(self.metrics.max_memory_usage, memory_percent)
            
            await asyncio.sleep(5)
    
    async def send_transaction_batch(self, session: httpx.AsyncClient, batch_data: Dict) -> Tuple[bool, float]:
        """Send a single transaction batch and measure performance"""
        start_time = time.time()
        
        try:
            response = await session.post(
                f"{self.config.data_ingestion_url}/api/v1/transactions",
                json=batch_data,
                timeout=30.0
            )
            
            end_time = time.time()
            response_time = (end_time - start_time) * 1000  # Convert to milliseconds
            
            if response.status_code == 201:
                return True, response_time
            else:
                print(f"Error response: {response.status_code} - {response.text}")
                return False, response_time
                
        except Exception as e:
            end_time = time.time() 
            response_time = (end_time - start_time) * 1000
            print(f"Request failed: {str(e)}")
            return False, response_time
    
    async def send_entity_data(self, session: httpx.AsyncClient, entity_type: str, entity_data: List[Dict]) -> Tuple[int, int]:
        """Send entity data in batches"""
        successful = 0
        failed = 0
        
        # Send entities in smaller batches
        batch_size = 10
        for i in range(0, len(entity_data), batch_size):
            batch = entity_data[i:i + batch_size]
            
            for entity in batch:
                try:
                    response = await session.post(
                        f"{self.config.api_base_url}/api/v1/entities/{entity_type}",
                        json=entity,
                        timeout=10.0
                    )
                    
                    if response.status_code in [200, 201]:
                        successful += 1
                    else:
                        failed += 1
                        
                except Exception:
                    failed += 1
                    
        return successful, failed
    
    async def run_load_test_phase(self, phase_name: str, concurrent_requests: int, duration: int) -> Dict[str, Any]:
        """Run a single load test phase"""
        print(f"\n--- {phase_name} ---")
        print(f"Concurrent requests: {concurrent_requests}, Duration: {duration}s")
        
        phase_metrics = PerformanceMetrics()
        start_time = time.time()
        
        async with httpx.AsyncClient() as session:
            # Start system monitoring
            monitor_task = asyncio.create_task(self.monitor_system_resources())
            
            # Create semaphore to limit concurrent requests
            semaphore = asyncio.Semaphore(concurrent_requests)
            
            async def worker():
                """Worker function for sending requests"""
                while (time.time() - start_time) < duration:
                    async with semaphore:
                        batch_data = self.generate_transaction_batch(100)
                        success, response_time = await self.send_transaction_batch(session, batch_data)
                        
                        phase_metrics.total_requests += 1
                        phase_metrics.response_times.append(response_time)
                        
                        if success:
                            phase_metrics.successful_requests += 1
                        else:
                            phase_metrics.failed_requests += 1
                            
                        # Small delay to prevent overwhelming the system
                        await asyncio.sleep(0.1)
            
            # Start worker tasks
            workers = [asyncio.create_task(worker()) for _ in range(concurrent_requests)]
            
            # Wait for test duration
            await asyncio.sleep(duration)
            
            # Cancel workers
            for worker in workers:
                worker.cancel()
                
            # Wait for workers to finish
            await asyncio.gather(*workers, return_exceptions=True)
            
            # Stop monitoring
            monitor_task.cancel()
            
        # Calculate metrics
        if phase_metrics.response_times:
            phase_metrics.avg_response_time = statistics.mean(phase_metrics.response_times)
            phase_metrics.p95_response_time = statistics.quantiles(phase_metrics.response_times, n=20)[18]  # 95th percentile
            phase_metrics.p99_response_time = statistics.quantiles(phase_metrics.response_times, n=100)[98]  # 99th percentile
            
        phase_metrics.error_rate = phase_metrics.failed_requests / max(phase_metrics.total_requests, 1)
        
        actual_duration = time.time() - start_time
        throughput = phase_metrics.successful_requests / actual_duration
        
        print(f"Results:")
        print(f"  Total requests: {phase_metrics.total_requests}")
        print(f"  Successful: {phase_metrics.successful_requests}")
        print(f"  Failed: {phase_metrics.failed_requests}")
        print(f"  Error rate: {phase_metrics.error_rate:.2%}")
        print(f"  Throughput: {throughput:.2f} requests/second")
        print(f"  Avg response time: {phase_metrics.avg_response_time:.2f}ms")
        print(f"  95th percentile: {phase_metrics.p95_response_time:.2f}ms")
        print(f"  99th percentile: {phase_metrics.p99_response_time:.2f}ms")
        print(f"  Max CPU usage: {phase_metrics.max_cpu_usage:.1%}")
        print(f"  Max memory usage: {phase_metrics.max_memory_usage:.1%}")
        
        return {
            "phase": phase_name,
            "metrics": phase_metrics,
            "throughput": throughput,
            "duration": actual_duration
        }
    
    async def run_baseline_test(self) -> Dict[str, Any]:
        """Run baseline performance test with entities and small transaction load"""
        print("Setting up baseline test data...")
        
        async with httpx.AsyncClient() as session:
            # Setup test entities first
            entity_data = self.generate_entity_data(100)
            
            print("Ingesting test entities...")
            person_success, person_failed = await self.send_entity_data(session, "persons", entity_data["persons"])
            org_success, org_failed = await self.send_entity_data(session, "organizations", entity_data["organizations"])
            acc_success, acc_failed = await self.send_entity_data(session, "accounts", entity_data["accounts"])
            
            total_entities = person_success + org_success + acc_success
            print(f"Entities ingested: {total_entities}")
            
            # Wait for entity processing
            await asyncio.sleep(10)
            
        # Run low-volume transaction test
        return await self.run_load_test_phase("Baseline Load", 5, 60)
    
    async def run_stress_test(self) -> Dict[str, Any]:
        """Run stress test with high concurrent load"""
        return await self.run_load_test_phase("Stress Test", self.config.concurrent_users, 300)
    
    async def run_spike_test(self) -> Dict[str, Any]:
        """Run spike test with sudden load increase"""
        # Start with low load
        baseline_result = await self.run_load_test_phase("Spike Baseline", 5, 60)
        
        # Sudden spike to high load
        spike_result = await self.run_load_test_phase("Spike Peak", self.config.concurrent_users * 2, 120)
        
        # Return to baseline
        recovery_result = await self.run_load_test_phase("Spike Recovery", 5, 60)
        
        return {
            "baseline": baseline_result,
            "spike": spike_result,
            "recovery": recovery_result
        }
    
    def validate_performance_requirements(self, results: List[Dict[str, Any]]) -> bool:
        """Validate that performance requirements are met"""
        print("\n--- Performance Requirements Validation ---")
        
        validation_passed = True
        
        for result in results:
            phase = result["phase"]
            metrics = result["metrics"]
            throughput = result["throughput"]
            
            print(f"\n{phase}:")
            
            # Check error rate
            if metrics.error_rate > self.config.max_error_rate:
                print(f"  ❌ Error rate too high: {metrics.error_rate:.2%} > {self.config.max_error_rate:.2%}")
                validation_passed = False
            else:
                print(f"  ✅ Error rate acceptable: {metrics.error_rate:.2%}")
            
            # Check response time
            if metrics.p95_response_time > self.config.max_latency_ms:
                print(f"  ❌ 95th percentile latency too high: {metrics.p95_response_time:.2f}ms > {self.config.max_latency_ms}ms")
                validation_passed = False
            else:
                print(f"  ✅ 95th percentile latency acceptable: {metrics.p95_response_time:.2f}ms")
            
            # Check resource usage
            if metrics.max_cpu_usage > self.config.max_cpu_usage:
                print(f"  ❌ CPU usage too high: {metrics.max_cpu_usage:.1%} > {self.config.max_cpu_usage:.1%}")
                validation_passed = False
            else:
                print(f"  ✅ CPU usage acceptable: {metrics.max_cpu_usage:.1%}")
                
            if metrics.max_memory_usage > self.config.max_memory_usage:
                print(f"  ❌ Memory usage too high: {metrics.max_memory_usage:.1%} > {self.config.max_memory_usage:.1%}")
                validation_passed = False
            else:
                print(f"  ✅ Memory usage acceptable: {metrics.max_memory_usage:.1%}")
        
        return validation_passed
    
    async def run_full_load_test(self) -> Dict[str, Any]:
        """Execute complete load testing suite"""
        print("🚀 Starting Data Ingestion Load Test")
        print("=" * 60)
        
        self.test_start_time = time.time()
        
        try:
            # Test phases
            baseline_result = await self.run_baseline_test()
            stress_result = await self.run_stress_test()
            spike_results = await self.run_spike_test()
            
            all_results = [baseline_result, stress_result]
            if isinstance(spike_results, dict) and "baseline" in spike_results:
                all_results.extend([spike_results["baseline"], spike_results["spike"], spike_results["recovery"]])
            
            # Validate requirements
            validation_passed = self.validate_performance_requirements(all_results)
            
            # Generate summary report
            total_requests = sum(r["metrics"].total_requests for r in all_results)
            total_successful = sum(r["metrics"].successful_requests for r in all_results)
            avg_throughput = statistics.mean([r["throughput"] for r in all_results])
            
            summary = {
                "test_id": self.test_id,
                "total_test_duration": time.time() - self.test_start_time,
                "total_requests": total_requests,
                "total_successful": total_successful,
                "average_throughput": avg_throughput,
                "validation_passed": validation_passed,
                "phase_results": all_results
            }
            
            print("\n" + "=" * 60)
            print("📊 LOAD TEST SUMMARY")
            print("=" * 60)
            print(f"Test ID: {self.test_id}")
            print(f"Total requests: {total_requests:,}")
            print(f"Successful requests: {total_successful:,}")
            print(f"Average throughput: {avg_throughput:.2f} req/sec")
            print(f"Validation status: {'PASSED' if validation_passed else 'FAILED'}")
            
            if validation_passed:
                print("🎉 All performance requirements met!")
            else:
                print("⚠️  Some performance requirements not met - check details above")
                
            return summary
            
        except Exception as e:
            print(f"❌ Load test failed: {str(e)}")
            import traceback
            traceback.print_exc()
            raise


# Test execution functions
@pytest.mark.asyncio
async def test_data_ingestion_baseline_performance():
    """Test baseline performance under normal load"""
    config = LoadTestConfig(concurrent_users=10, test_duration=120)
    test_suite = DataIngestionLoadTest(config)
    
    result = await test_suite.run_baseline_test()
    
    # Assert performance requirements
    assert result["metrics"].error_rate < 0.05, f"Error rate too high: {result['metrics'].error_rate}"
    assert result["metrics"].p95_response_time < 1000, f"95th percentile latency too high: {result['metrics'].p95_response_time}ms"
    assert result["throughput"] > 10, f"Throughput too low: {result['throughput']} req/sec"

@pytest.mark.asyncio 
async def test_data_ingestion_stress_performance():
    """Test performance under stress conditions"""
    config = LoadTestConfig(concurrent_users=50, test_duration=180)
    test_suite = DataIngestionLoadTest(config)
    
    result = await test_suite.run_stress_test()
    
    # Assert stress performance requirements
    assert result["metrics"].error_rate < 0.10, f"Error rate too high under stress: {result['metrics'].error_rate}"
    assert result["metrics"].p95_response_time < 2000, f"95th percentile latency too high under stress: {result['metrics'].p95_response_time}ms"

@pytest.mark.asyncio
async def test_data_ingestion_full_load_test():
    """Execute the complete load test suite"""
    config = LoadTestConfig()
    test_suite = DataIngestionLoadTest(config)
    
    summary = await test_suite.run_full_load_test()
    
    assert summary["validation_passed"], "Load test validation failed"
    assert summary["total_requests"] > 1000, "Not enough requests processed during test"
    assert summary["average_throughput"] > 50, "Average throughput too low"


if __name__ == "__main__":
    """Direct execution for debugging"""
    async def main():
        config = LoadTestConfig()
        test_suite = DataIngestionLoadTest(config)
        await test_suite.run_full_load_test()
        
    asyncio.run(main())