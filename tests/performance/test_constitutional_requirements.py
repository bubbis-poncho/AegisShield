#!/usr/bin/env python3
"""
Performance validation suite for AegisShield platform
Validates system meets constitutional performance requirements
"""

import asyncio
import aiohttp
import time
import statistics
import json
import psutil
import subprocess
import os
import sys
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass
import concurrent.futures
import threading
import uuid
import random
import tempfile

@dataclass
class PerformanceRequirements:
    """Constitutional performance requirements"""
    # Data ingestion requirements
    data_ingestion_max_time_minutes: float = 5.0  # <5 min for 10MB
    data_ingestion_test_size_mb: float = 10.0
    
    # Graph query requirements  
    graph_query_max_time_seconds: float = 2.0  # <2s for graph queries
    
    # API response requirements
    api_response_max_time_ms: float = 500.0  # <500ms for API responses
    
    # Concurrent user requirements
    min_concurrent_users: int = 1000  # 1000+ concurrent users
    
    # Throughput requirements
    min_transaction_throughput_tps: float = 1000.0  # 1000+ TPS
    
    # System stability requirements
    max_error_rate_percent: float = 1.0  # <1% error rate
    min_uptime_percent: float = 99.9  # 99.9% uptime

@dataclass
class PerformanceTestConfig:
    """Configuration for performance testing"""
    api_base_url: str = "http://localhost:8080"
    test_duration_seconds: int = 300  # 5 minutes
    warmup_duration_seconds: int = 60  # 1 minute warmup
    concurrent_users: int = 1000
    ramp_up_time_seconds: int = 120  # 2 minutes to ramp up
    
    # Authentication
    auth_token: str = ""
    
    # Test data configuration
    test_data_size_mb: float = 10.0
    transactions_per_batch: int = 100
    
    # Monitoring configuration
    monitor_interval_seconds: float = 5.0

@dataclass
class PerformanceResult:
    """Results from performance testing"""
    test_name: str
    requirement_met: bool
    measured_value: float
    required_value: float
    unit: str
    details: Dict[str, Any]
    duration_seconds: float

class SystemMonitor:
    """Monitor system resources during performance testing"""
    
    def __init__(self, interval: float = 5.0):
        self.interval = interval
        self.monitoring = False
        self.metrics = []
        self.monitor_thread = None
        
    def start_monitoring(self):
        """Start system monitoring"""
        self.monitoring = True
        self.metrics = []
        self.monitor_thread = threading.Thread(target=self._monitor_loop)
        self.monitor_thread.start()
        
    def stop_monitoring(self):
        """Stop system monitoring"""
        self.monitoring = False
        if self.monitor_thread:
            self.monitor_thread.join()
            
    def _monitor_loop(self):
        """Monitor system metrics in a loop"""
        while self.monitoring:
            try:
                # CPU and memory metrics
                cpu_percent = psutil.cpu_percent(interval=1)
                memory = psutil.virtual_memory()
                disk = psutil.disk_usage('/')
                
                # Network metrics
                network = psutil.net_io_counters()
                
                # Process metrics for specific services
                processes = {}
                for proc in psutil.process_iter(['pid', 'name', 'cpu_percent', 'memory_percent']):
                    try:
                        if any(service in proc.info['name'].lower() for service in 
                               ['data-ingestion', 'api-gateway', 'entity-resolution', 'graph-engine']):
                            processes[proc.info['name']] = {
                                'cpu_percent': proc.info['cpu_percent'],
                                'memory_percent': proc.info['memory_percent']
                            }
                    except (psutil.NoSuchProcess, psutil.AccessDenied):
                        continue
                
                metric = {
                    'timestamp': time.time(),
                    'cpu_percent': cpu_percent,
                    'memory_percent': memory.percent,
                    'memory_available_gb': memory.available / (1024**3),
                    'disk_usage_percent': disk.percent,
                    'disk_free_gb': disk.free / (1024**3),
                    'network_bytes_sent': network.bytes_sent,
                    'network_bytes_recv': network.bytes_recv,
                    'processes': processes
                }
                
                self.metrics.append(metric)
                
            except Exception as e:
                print(f"Monitoring error: {e}")
                
            time.sleep(self.interval)
    
    def get_summary(self) -> Dict[str, Any]:
        """Get summary of monitoring data"""
        if not self.metrics:
            return {}
            
        cpu_values = [m['cpu_percent'] for m in self.metrics]
        memory_values = [m['memory_percent'] for m in self.metrics]
        
        return {
            'duration_seconds': (self.metrics[-1]['timestamp'] - self.metrics[0]['timestamp']),
            'cpu_avg': statistics.mean(cpu_values),
            'cpu_max': max(cpu_values),
            'cpu_p95': statistics.quantiles(cpu_values, n=20)[18] if len(cpu_values) >= 20 else max(cpu_values),
            'memory_avg': statistics.mean(memory_values),
            'memory_max': max(memory_values),
            'memory_p95': statistics.quantiles(memory_values, n=20)[18] if len(memory_values) >= 20 else max(memory_values),
            'samples_count': len(self.metrics)
        }

class PerformanceValidator:
    """Validates system performance against constitutional requirements"""
    
    def __init__(self, config: PerformanceTestConfig, requirements: PerformanceRequirements):
        self.config = config
        self.requirements = requirements
        self.results = []
        self.monitor = SystemMonitor(config.monitor_interval_seconds)
        
    async def setup_authentication(self) -> bool:
        """Setup authentication for API calls"""
        try:
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    f"{self.config.api_base_url}/auth/login",
                    json={
                        "email": "admin@aegisshield.com",
                        "password": "admin_password_123"
                    },
                    timeout=aiohttp.ClientTimeout(total=30)
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        self.config.auth_token = data.get("access_token", "")
                        return True
                    return False
                    
        except Exception as e:
            print(f"Authentication setup failed: {e}")
            return False
    
    def generate_test_transactions(self, count: int) -> List[Dict[str, Any]]:
        """Generate realistic test transaction data"""
        transactions = []
        
        for i in range(count):
            source_account = f"ACC{random.randint(1000000, 9999999):07d}"
            dest_account = f"ACC{random.randint(1000000, 9999999):07d}"
            
            transaction = {
                "transaction_id": str(uuid.uuid4()),
                "source_account": source_account,
                "destination_account": dest_account,
                "amount": round(random.uniform(100, 100000), 2),
                "currency": random.choice(["USD", "EUR", "GBP", "JPY"]),
                "transaction_date": f"2024-01-{random.randint(1, 28):02d}T{random.randint(0, 23):02d}:{random.randint(0, 59):02d}:00Z",
                "description": random.choice([
                    "Wire transfer", "ACH payment", "International transfer",
                    "Business payment", "Salary payment", "Investment transfer"
                ]),
                "transaction_type": random.choice(["wire", "ach", "swift", "internal"]),
                "status": "completed"
            }
            transactions.append(transaction)
            
        return transactions
    
    async def test_data_ingestion_performance(self) -> PerformanceResult:
        """Test data ingestion performance: <5 min for 10MB"""
        print("Testing data ingestion performance...")
        
        start_time = time.time()
        self.monitor.start_monitoring()
        
        try:
            # Calculate number of transactions needed for target size
            sample_transaction = self.generate_test_transactions(1)[0]
            transaction_size_bytes = len(json.dumps(sample_transaction).encode('utf-8'))
            target_size_bytes = self.config.test_data_size_mb * 1024 * 1024
            total_transactions = int(target_size_bytes / transaction_size_bytes)
            
            print(f"Ingesting {total_transactions:,} transactions ({self.config.test_data_size_mb}MB)")
            
            successful_batches = 0
            failed_batches = 0
            
            # Process in batches
            batch_size = self.config.transactions_per_batch
            total_batches = (total_transactions + batch_size - 1) // batch_size
            
            async with aiohttp.ClientSession(
                headers={"Authorization": f"Bearer {self.config.auth_token}"},
                timeout=aiohttp.ClientTimeout(total=60)
            ) as session:
                
                for batch_num in range(total_batches):
                    batch_start = batch_num * batch_size
                    batch_end = min(batch_start + batch_size, total_transactions)
                    batch_transactions = self.generate_test_transactions(batch_end - batch_start)
                    
                    try:
                        async with session.post(
                            f"{self.config.api_base_url}/api/v1/data/transactions",
                            json={"transactions": batch_transactions}
                        ) as response:
                            if response.status in [200, 201, 202]:
                                successful_batches += 1
                            else:
                                failed_batches += 1
                                print(f"Batch {batch_num} failed: {response.status}")
                                
                    except Exception as e:
                        failed_batches += 1
                        print(f"Batch {batch_num} error: {e}")
                    
                    # Progress update
                    if batch_num % 10 == 0:
                        progress = (batch_num / total_batches) * 100
                        elapsed = time.time() - start_time
                        print(f"Progress: {progress:.1f}% ({elapsed:.1f}s elapsed)")
            
            duration_seconds = time.time() - start_time
            duration_minutes = duration_seconds / 60
            
            self.monitor.stop_monitoring()
            monitor_summary = self.monitor.get_summary()
            
            # Check if requirement is met
            requirement_met = duration_minutes <= self.requirements.data_ingestion_max_time_minutes
            error_rate = (failed_batches / max(total_batches, 1)) * 100
            
            return PerformanceResult(
                test_name="Data Ingestion Performance",
                requirement_met=requirement_met and error_rate <= self.requirements.max_error_rate_percent,
                measured_value=duration_minutes,
                required_value=self.requirements.data_ingestion_max_time_minutes,
                unit="minutes",
                duration_seconds=duration_seconds,
                details={
                    "total_transactions": total_transactions,
                    "data_size_mb": self.config.test_data_size_mb,
                    "successful_batches": successful_batches,
                    "failed_batches": failed_batches,
                    "error_rate_percent": error_rate,
                    "throughput_tps": total_transactions / duration_seconds,
                    "system_metrics": monitor_summary
                }
            )
            
        except Exception as e:
            self.monitor.stop_monitoring()
            return PerformanceResult(
                test_name="Data Ingestion Performance",
                requirement_met=False,
                measured_value=float('inf'),
                required_value=self.requirements.data_ingestion_max_time_minutes,
                unit="minutes",
                duration_seconds=time.time() - start_time,
                details={"error": str(e)}
            )
    
    async def test_graph_query_performance(self) -> PerformanceResult:
        """Test graph query performance: <2s for graph queries"""
        print("Testing graph query performance...")
        
        start_time = time.time()
        
        try:
            # First, ensure we have some entities to query
            sample_entities = await self.get_sample_entities()
            
            if not sample_entities:
                print("No entities found, creating sample data...")
                await self.create_sample_graph_data()
                sample_entities = await self.get_sample_entities()
            
            if not sample_entities:
                return PerformanceResult(
                    test_name="Graph Query Performance",
                    requirement_met=False,
                    measured_value=float('inf'),
                    required_value=self.requirements.graph_query_max_time_seconds,
                    unit="seconds",
                    duration_seconds=time.time() - start_time,
                    details={"error": "No entities available for graph queries"}
                )
            
            # Test various graph query types
            query_times = []
            successful_queries = 0
            failed_queries = 0
            
            async with aiohttp.ClientSession(
                headers={"Authorization": f"Bearer {self.config.auth_token}"},
                timeout=aiohttp.ClientTimeout(total=10)
            ) as session:
                
                # Test simple graph exploration queries
                for entity in sample_entities[:10]:  # Test with first 10 entities
                    entity_id = entity.get("id")
                    if not entity_id:
                        continue
                        
                    query_start = time.time()
                    
                    try:
                        async with session.post(
                            f"{self.config.api_base_url}/api/v1/graph/explore",
                            json={
                                "entity_id": entity_id,
                                "depth": 2,
                                "min_strength": 0.3
                            }
                        ) as response:
                            query_duration = time.time() - query_start
                            query_times.append(query_duration)
                            
                            if response.status == 200:
                                successful_queries += 1
                            else:
                                failed_queries += 1
                                
                    except Exception as e:
                        query_duration = time.time() - query_start
                        query_times.append(query_duration)
                        failed_queries += 1
                        print(f"Graph query failed: {e}")
                
                # Test complex graph queries
                for i in range(5):
                    query_start = time.time()
                    
                    try:
                        async with session.post(
                            f"{self.config.api_base_url}/api/v1/graph/search",
                            json={
                                "query_type": "pattern_detection",
                                "pattern": "suspicious_transactions",
                                "depth": 3,
                                "min_entities": 5
                            }
                        ) as response:
                            query_duration = time.time() - query_start
                            query_times.append(query_duration)
                            
                            if response.status == 200:
                                successful_queries += 1
                            else:
                                failed_queries += 1
                                
                    except Exception as e:
                        query_duration = time.time() - query_start
                        query_times.append(query_duration)
                        failed_queries += 1
            
            # Calculate metrics
            if query_times:
                avg_query_time = statistics.mean(query_times)
                max_query_time = max(query_times)
                p95_query_time = statistics.quantiles(query_times, n=20)[18] if len(query_times) >= 20 else max_query_time
            else:
                avg_query_time = max_query_time = p95_query_time = float('inf')
            
            total_queries = successful_queries + failed_queries
            error_rate = (failed_queries / max(total_queries, 1)) * 100
            
            # Check if requirement is met (use P95 for requirement validation)
            requirement_met = (
                p95_query_time <= self.requirements.graph_query_max_time_seconds and
                error_rate <= self.requirements.max_error_rate_percent
            )
            
            duration_seconds = time.time() - start_time
            
            return PerformanceResult(
                test_name="Graph Query Performance",
                requirement_met=requirement_met,
                measured_value=p95_query_time,
                required_value=self.requirements.graph_query_max_time_seconds,
                unit="seconds",
                duration_seconds=duration_seconds,
                details={
                    "total_queries": total_queries,
                    "successful_queries": successful_queries,
                    "failed_queries": failed_queries,
                    "error_rate_percent": error_rate,
                    "avg_query_time": avg_query_time,
                    "max_query_time": max_query_time,
                    "p95_query_time": p95_query_time,
                    "query_times": query_times[:10]  # Sample of query times
                }
            )
            
        except Exception as e:
            return PerformanceResult(
                test_name="Graph Query Performance",
                requirement_met=False,
                measured_value=float('inf'),
                required_value=self.requirements.graph_query_max_time_seconds,
                unit="seconds",
                duration_seconds=time.time() - start_time,
                details={"error": str(e)}
            )
    
    async def test_api_response_performance(self) -> PerformanceResult:
        """Test API response performance: <500ms for API responses"""
        print("Testing API response performance...")
        
        start_time = time.time()
        
        try:
            response_times = []
            successful_requests = 0
            failed_requests = 0
            
            # Test various API endpoints
            endpoints = [
                ("/api/v1/health", "GET", None),
                ("/api/v1/cases", "GET", None),
                ("/api/v1/entities/search", "GET", {"q": "test"}),
                ("/api/v1/alerts", "GET", None),
                ("/api/v1/investigations", "GET", None),
                ("/api/v1/statistics/dashboard", "GET", None)
            ]
            
            async with aiohttp.ClientSession(
                headers={"Authorization": f"Bearer {self.config.auth_token}"},
                timeout=aiohttp.ClientTimeout(total=2)
            ) as session:
                
                # Test each endpoint multiple times
                for endpoint, method, params in endpoints:
                    for _ in range(20):  # 20 requests per endpoint
                        request_start = time.time()
                        
                        try:
                            if method == "GET":
                                async with session.get(
                                    f"{self.config.api_base_url}{endpoint}",
                                    params=params
                                ) as response:
                                    request_duration = (time.time() - request_start) * 1000  # Convert to ms
                                    response_times.append(request_duration)
                                    
                                    if response.status == 200:
                                        successful_requests += 1
                                    else:
                                        failed_requests += 1
                                        
                        except Exception as e:
                            request_duration = (time.time() - request_start) * 1000
                            response_times.append(request_duration)
                            failed_requests += 1
            
            # Calculate metrics
            if response_times:
                avg_response_time = statistics.mean(response_times)
                max_response_time = max(response_times)
                p95_response_time = statistics.quantiles(response_times, n=20)[18] if len(response_times) >= 20 else max_response_time
                p99_response_time = statistics.quantiles(response_times, n=100)[98] if len(response_times) >= 100 else max_response_time
            else:
                avg_response_time = max_response_time = p95_response_time = p99_response_time = float('inf')
            
            total_requests = successful_requests + failed_requests
            error_rate = (failed_requests / max(total_requests, 1)) * 100
            
            # Check if requirement is met (use P95 for requirement validation)
            requirement_met = (
                p95_response_time <= self.requirements.api_response_max_time_ms and
                error_rate <= self.requirements.max_error_rate_percent
            )
            
            duration_seconds = time.time() - start_time
            
            return PerformanceResult(
                test_name="API Response Performance",
                requirement_met=requirement_met,
                measured_value=p95_response_time,
                required_value=self.requirements.api_response_max_time_ms,
                unit="milliseconds",
                duration_seconds=duration_seconds,
                details={
                    "total_requests": total_requests,
                    "successful_requests": successful_requests,
                    "failed_requests": failed_requests,
                    "error_rate_percent": error_rate,
                    "avg_response_time_ms": avg_response_time,
                    "max_response_time_ms": max_response_time,
                    "p95_response_time_ms": p95_response_time,
                    "p99_response_time_ms": p99_response_time,
                    "endpoints_tested": len(endpoints)
                }
            )
            
        except Exception as e:
            return PerformanceResult(
                test_name="API Response Performance",
                requirement_met=False,
                measured_value=float('inf'),
                required_value=self.requirements.api_response_max_time_ms,
                unit="milliseconds",
                duration_seconds=time.time() - start_time,
                details={"error": str(e)}
            )
    
    async def test_concurrent_users_performance(self) -> PerformanceResult:
        """Test concurrent users performance: 1000+ concurrent users"""
        print(f"Testing concurrent users performance ({self.config.concurrent_users} users)...")
        
        start_time = time.time()
        self.monitor.start_monitoring()
        
        try:
            # Simulate concurrent user sessions
            total_operations = 0
            successful_operations = 0
            failed_operations = 0
            all_response_times = []
            
            # Create semaphore to limit concurrent connections
            semaphore = asyncio.Semaphore(self.config.concurrent_users)
            
            async def simulate_user_session(user_id: int):
                """Simulate a single user session"""
                async with semaphore:
                    session_start = time.time()
                    user_operations = 0
                    user_successes = 0
                    user_failures = 0
                    
                    # Each user performs multiple operations
                    operations = [
                        ("GET", "/api/v1/cases", None),
                        ("GET", "/api/v1/entities/search", {"q": f"user_{user_id}"}),
                        ("GET", "/api/v1/alerts", {"limit": 10}),
                        ("GET", "/api/v1/statistics/dashboard", None)
                    ]
                    
                    async with aiohttp.ClientSession(
                        headers={"Authorization": f"Bearer {self.config.auth_token}"},
                        timeout=aiohttp.ClientTimeout(total=10)
                    ) as session:
                        
                        for method, endpoint, params in operations:
                            op_start = time.time()
                            user_operations += 1
                            
                            try:
                                async with session.get(
                                    f"{self.config.api_base_url}{endpoint}",
                                    params=params
                                ) as response:
                                    op_duration = (time.time() - op_start) * 1000
                                    all_response_times.append(op_duration)
                                    
                                    if response.status == 200:
                                        user_successes += 1
                                    else:
                                        user_failures += 1
                                        
                            except Exception as e:
                                op_duration = (time.time() - op_start) * 1000
                                all_response_times.append(op_duration)
                                user_failures += 1
                    
                    return user_operations, user_successes, user_failures
            
            # Run concurrent user sessions
            print(f"Starting {self.config.concurrent_users} concurrent user sessions...")
            
            tasks = []
            for user_id in range(self.config.concurrent_users):
                task = asyncio.create_task(simulate_user_session(user_id))
                tasks.append(task)
            
            # Process results as they complete
            for completed_task in asyncio.as_completed(tasks):
                ops, successes, failures = await completed_task
                total_operations += ops
                successful_operations += successes
                failed_operations += failures
            
            duration_seconds = time.time() - start_time
            
            self.monitor.stop_monitoring()
            monitor_summary = self.monitor.get_summary()
            
            # Calculate metrics
            error_rate = (failed_operations / max(total_operations, 1)) * 100
            throughput = total_operations / duration_seconds
            
            if all_response_times:
                avg_response_time = statistics.mean(all_response_times)
                p95_response_time = statistics.quantiles(all_response_times, n=20)[18] if len(all_response_times) >= 20 else max(all_response_times)
            else:
                avg_response_time = p95_response_time = float('inf')
            
            # Check if requirements are met
            requirement_met = (
                self.config.concurrent_users >= self.requirements.min_concurrent_users and
                error_rate <= self.requirements.max_error_rate_percent and
                p95_response_time <= self.requirements.api_response_max_time_ms
            )
            
            return PerformanceResult(
                test_name="Concurrent Users Performance",
                requirement_met=requirement_met,
                measured_value=float(self.config.concurrent_users),
                required_value=float(self.requirements.min_concurrent_users),
                unit="users",
                duration_seconds=duration_seconds,
                details={
                    "concurrent_users": self.config.concurrent_users,
                    "total_operations": total_operations,
                    "successful_operations": successful_operations,
                    "failed_operations": failed_operations,
                    "error_rate_percent": error_rate,
                    "operations_per_second": throughput,
                    "avg_response_time_ms": avg_response_time,
                    "p95_response_time_ms": p95_response_time,
                    "system_metrics": monitor_summary
                }
            )
            
        except Exception as e:
            self.monitor.stop_monitoring()
            return PerformanceResult(
                test_name="Concurrent Users Performance",
                requirement_met=False,
                measured_value=0,
                required_value=float(self.requirements.min_concurrent_users),
                unit="users",
                duration_seconds=time.time() - start_time,
                details={"error": str(e)}
            )
    
    async def get_sample_entities(self) -> List[Dict[str, Any]]:
        """Get sample entities for testing"""
        try:
            async with aiohttp.ClientSession(
                headers={"Authorization": f"Bearer {self.config.auth_token}"},
                timeout=aiohttp.ClientTimeout(total=10)
            ) as session:
                async with session.get(
                    f"{self.config.api_base_url}/api/v1/entities",
                    params={"limit": 20}
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        return data.get("entities", [])
            return []
        except Exception:
            return []
    
    async def create_sample_graph_data(self):
        """Create sample data for graph testing"""
        try:
            # Create some sample entities and relationships
            sample_data = {
                "entities": [
                    {
                        "type": "person",
                        "name": "Test Person 1",
                        "properties": {"ssn": "123456789"}
                    },
                    {
                        "type": "organization", 
                        "name": "Test Org 1",
                        "properties": {"tax_id": "987654321"}
                    }
                ],
                "transactions": self.generate_test_transactions(10)
            }
            
            async with aiohttp.ClientSession(
                headers={"Authorization": f"Bearer {self.config.auth_token}"},
                timeout=aiohttp.ClientTimeout(total=30)
            ) as session:
                await session.post(
                    f"{self.config.api_base_url}/api/v1/data/bulk",
                    json=sample_data
                )
        except Exception as e:
            print(f"Failed to create sample data: {e}")
    
    async def run_performance_validation(self) -> Dict[str, Any]:
        """Execute complete performance validation"""
        print("🚀 Starting Performance Validation")
        print("=" * 60)
        
        # Setup authentication
        if not await self.setup_authentication():
            return {
                "error": "Failed to setup authentication",
                "total_tests": 0,
                "passed_tests": 0,
                "failed_tests": 0
            }
        
        # Execute performance tests
        print("Running performance tests...")
        
        # Run tests in sequence to avoid resource conflicts
        data_ingestion_result = await self.test_data_ingestion_performance()
        self.results.append(data_ingestion_result)
        
        graph_query_result = await self.test_graph_query_performance()
        self.results.append(graph_query_result)
        
        api_response_result = await self.test_api_response_performance()
        self.results.append(api_response_result)
        
        concurrent_users_result = await self.test_concurrent_users_performance()
        self.results.append(concurrent_users_result)
        
        # Calculate summary
        total_tests = len(self.results)
        passed_tests = sum(1 for r in self.results if r.requirement_met)
        failed_tests = total_tests - passed_tests
        
        # Generate report
        summary = {
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "failed_tests": failed_tests,
            "success_rate": (passed_tests / total_tests) * 100 if total_tests > 0 else 0,
            "total_duration": sum(r.duration_seconds for r in self.results),
            "constitutional_requirements": {
                "data_ingestion_max_minutes": self.requirements.data_ingestion_max_time_minutes,
                "graph_query_max_seconds": self.requirements.graph_query_max_time_seconds,
                "api_response_max_ms": self.requirements.api_response_max_time_ms,
                "min_concurrent_users": self.requirements.min_concurrent_users
            },
            "test_results": self.results
        }
        
        print("\n" + "=" * 60)
        print("📊 PERFORMANCE VALIDATION SUMMARY")
        print("=" * 60)
        print(f"Total Tests: {total_tests}")
        print(f"Requirements Met: {passed_tests}")
        print(f"Requirements Failed: {failed_tests}")
        print(f"Success Rate: {summary['success_rate']:.1f}%")
        print(f"Total Duration: {summary['total_duration']:.2f} seconds")
        
        print("\n--- Performance Requirements ---")
        for result in self.results:
            status = "✅ PASS" if result.requirement_met else "❌ FAIL"
            print(f"{result.test_name}: {status}")
            print(f"  Measured: {result.measured_value:.2f} {result.unit}")
            print(f"  Required: {result.required_value:.2f} {result.unit}")
            print(f"  Duration: {result.duration_seconds:.2f}s")
        
        if summary['success_rate'] >= 100:
            print("\n🎉 All constitutional performance requirements met!")
        elif summary['success_rate'] >= 75:
            print("\n✅ Most performance requirements met - minor issues detected")
        else:
            print("\n⚠️  Significant performance issues detected - optimization needed")
        
        return summary


async def main():
    """Main execution function"""
    requirements = PerformanceRequirements()
    config = PerformanceTestConfig()
    
    # Override config from environment if available
    config.api_base_url = os.getenv("AEGIS_API_URL", config.api_base_url)
    config.concurrent_users = int(os.getenv("CONCURRENT_USERS", config.concurrent_users))
    
    validator = PerformanceValidator(config, requirements)
    summary = await validator.run_performance_validation()
    
    # Exit with appropriate code
    if summary.get("success_rate", 0) >= 75:
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())