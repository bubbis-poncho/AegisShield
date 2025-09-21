#!/usr/bin/env python3
"""
Load testing and stress testing for AegisShield platform
Tests system performance under high transaction volumes and concurrent users
"""

import asyncio
import aiohttp
import time
import json
import uuid
import statistics
from datetime import datetime, timedelta
from typing import Dict, List, Tuple
import os
import sys
from concurrent.futures import ThreadPoolExecutor
import threading
import signal

# Configuration
LOAD_TEST_CONFIG = {
    "api_base_url": os.getenv("API_BASE_URL", "http://localhost:8080"),
    "data_ingestion_url": os.getenv("DATA_INGESTION_URL", "http://localhost:8060"),
    "graph_engine_url": os.getenv("GRAPH_ENGINE_URL", "http://localhost:8065"),
    "concurrent_users": [10, 25, 50, 100, 200, 500],
    "test_duration_seconds": 300,  # 5 minutes per test
    "ramp_up_time": 60,  # 1 minute ramp up
    "think_time": 1.0,  # 1 second between requests
    "max_requests_per_second": 1000,
    "stress_test_multiplier": 2.0  # Stress test at 2x normal load
}


class LoadTestRunner:
    """Main load testing orchestrator"""
    
    def __init__(self):
        self.test_id = str(uuid.uuid4())[:8]
        self.results = {}
        self.stop_event = threading.Event()
        self.auth_token = "load-test-token-12345"
        self.headers = {"Authorization": f"Bearer {self.auth_token}"}
        
        # Setup signal handler for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully"""
        print(f"\n🛑 Received signal {signum}, shutting down gracefully...")
        self.stop_event.set()
    
    async def run_comprehensive_load_tests(self):
        """Run comprehensive load and stress testing suite"""
        print("🚀 Starting AegisShield Load & Stress Testing Suite")
        print(f"Test ID: {self.test_id}")
        print("=" * 80)
        
        try:
            # Phase 1: Baseline Performance Testing
            await self._run_baseline_performance_tests()
            
            # Phase 2: Load Testing (Normal Traffic Patterns)
            await self._run_load_testing_scenarios()
            
            # Phase 3: Stress Testing (Beyond Normal Capacity)
            await self._run_stress_testing_scenarios()
            
            # Phase 4: Endurance Testing (Sustained Load)
            await self._run_endurance_testing()
            
            # Phase 5: Spike Testing (Sudden Load Increases)
            await self._run_spike_testing()
            
            # Phase 6: Volume Testing (Large Data Sets)
            await self._run_volume_testing()
            
            # Generate comprehensive report
            await self._generate_load_test_report()
            
        except KeyboardInterrupt:
            print("\n⚠️ Load testing interrupted by user")
        except Exception as e:
            print(f"\n❌ Load testing failed: {str(e)}")
        finally:
            print("\n🏁 Load testing completed")
    
    async def _run_baseline_performance_tests(self):
        """Establish baseline performance metrics"""
        print("\n📊 Phase 1: Baseline Performance Testing")
        
        # Single user performance baseline
        baseline_tests = [
            ("transaction_ingestion", self._test_transaction_ingestion_single),
            ("entity_creation", self._test_entity_creation_single),
            ("graph_query", self._test_graph_query_single),
            ("alert_retrieval", self._test_alert_retrieval_single),
            ("investigation_workflow", self._test_investigation_workflow_single)
        ]
        
        baseline_results = {}
        
        for test_name, test_func in baseline_tests:
            print(f"  Running baseline: {test_name}")
            
            # Run test 10 times for average
            times = []
            for i in range(10):
                if self.stop_event.is_set():
                    break
                
                start_time = time.time()
                success = await test_func()
                end_time = time.time()
                
                if success:
                    times.append(end_time - start_time)
                
                await asyncio.sleep(0.5)  # Brief pause between iterations
            
            if times:
                baseline_results[test_name] = {
                    "average_time": statistics.mean(times),
                    "min_time": min(times),
                    "max_time": max(times),
                    "std_dev": statistics.stdev(times) if len(times) > 1 else 0,
                    "iterations": len(times)
                }
                
                avg_time = baseline_results[test_name]["average_time"]
                print(f"    ✅ Baseline {test_name}: {avg_time:.3f}s average")
            else:
                print(f"    ❌ Baseline {test_name}: No successful iterations")
        
        self.results["baseline_performance"] = baseline_results
    
    async def _run_load_testing_scenarios(self):
        """Run load testing with increasing concurrent users"""
        print("\n👥 Phase 2: Load Testing (Concurrent Users)")
        
        load_test_results = {}
        
        for user_count in LOAD_TEST_CONFIG["concurrent_users"]:
            if self.stop_event.is_set():
                break
                
            print(f"\n  Testing with {user_count} concurrent users")
            
            # Create user sessions
            sessions = []
            for i in range(user_count):
                session = UserSession(
                    user_id=f"load_user_{self.test_id}_{i}",
                    base_url=LOAD_TEST_CONFIG["api_base_url"],
                    headers=self.headers,
                    think_time=LOAD_TEST_CONFIG["think_time"]
                )
                sessions.append(session)
            
            # Run concurrent load test
            start_time = time.time()
            
            tasks = []
            for session in sessions:
                task = asyncio.create_task(
                    session.run_user_scenario(
                        duration=LOAD_TEST_CONFIG["test_duration_seconds"],
                        ramp_up=LOAD_TEST_CONFIG["ramp_up_time"] / user_count,
                        stop_event=self.stop_event
                    )
                )
                tasks.append(task)
            
            # Wait for all sessions to complete
            session_results = await asyncio.gather(*tasks, return_exceptions=True)
            
            end_time = time.time()
            total_duration = end_time - start_time
            
            # Aggregate results
            successful_sessions = [r for r in session_results if isinstance(r, dict)]
            failed_sessions = len(session_results) - len(successful_sessions)
            
            if successful_sessions:
                total_requests = sum(s["total_requests"] for s in successful_sessions)
                total_errors = sum(s["total_errors"] for s in successful_sessions)
                all_response_times = []
                
                for s in successful_sessions:
                    all_response_times.extend(s["response_times"])
                
                load_test_results[user_count] = {
                    "duration": total_duration,
                    "successful_sessions": len(successful_sessions),
                    "failed_sessions": failed_sessions,
                    "total_requests": total_requests,
                    "total_errors": total_errors,
                    "requests_per_second": total_requests / total_duration if total_duration > 0 else 0,
                    "error_rate": (total_errors / total_requests) * 100 if total_requests > 0 else 0,
                    "average_response_time": statistics.mean(all_response_times) if all_response_times else 0,
                    "median_response_time": statistics.median(all_response_times) if all_response_times else 0,
                    "p95_response_time": self._percentile(all_response_times, 95) if all_response_times else 0,
                    "p99_response_time": self._percentile(all_response_times, 99) if all_response_times else 0
                }
                
                rps = load_test_results[user_count]["requests_per_second"]
                error_rate = load_test_results[user_count]["error_rate"]
                avg_time = load_test_results[user_count]["average_response_time"]
                
                print(f"    ✅ {user_count} users: {rps:.1f} RPS, {error_rate:.1f}% errors, {avg_time:.3f}s avg")
            else:
                print(f"    ❌ {user_count} users: All sessions failed")
                load_test_results[user_count] = {"status": "failed"}
        
        self.results["load_testing"] = load_test_results
    
    async def _run_stress_testing_scenarios(self):
        """Run stress testing beyond normal capacity"""
        print("\n🔥 Phase 3: Stress Testing (Beyond Capacity)")
        
        # Find the maximum successful load from load testing
        max_successful_users = 0
        if "load_testing" in self.results:
            for users, result in self.results["load_testing"].items():
                if isinstance(result, dict) and result.get("error_rate", 100) < 10:  # Less than 10% error rate
                    max_successful_users = max(max_successful_users, users)
        
        if max_successful_users == 0:
            max_successful_users = 100  # Default fallback
        
        # Stress test at increasing multiples
        stress_multipliers = [1.5, 2.0, 3.0, 5.0]
        stress_results = {}
        
        for multiplier in stress_multipliers:
            if self.stop_event.is_set():
                break
                
            stress_users = int(max_successful_users * multiplier)
            print(f"\n  Stress testing with {stress_users} users ({multiplier}x normal)")
            
            # Run shorter stress test
            stress_duration = 120  # 2 minutes for stress test
            
            sessions = []
            for i in range(stress_users):
                session = UserSession(
                    user_id=f"stress_user_{self.test_id}_{i}",
                    base_url=LOAD_TEST_CONFIG["api_base_url"],
                    headers=self.headers,
                    think_time=0.5  # Faster requests for stress test
                )
                sessions.append(session)
            
            start_time = time.time()
            
            tasks = []
            for session in sessions:
                task = asyncio.create_task(
                    session.run_user_scenario(
                        duration=stress_duration,
                        ramp_up=10,  # Quick ramp up for stress test
                        stop_event=self.stop_event
                    )
                )
                tasks.append(task)
            
            session_results = await asyncio.gather(*tasks, return_exceptions=True)
            end_time = time.time()
            
            # Analyze stress test results
            successful_sessions = [r for r in session_results if isinstance(r, dict)]
            
            if successful_sessions:
                total_requests = sum(s["total_requests"] for s in successful_sessions)
                total_errors = sum(s["total_errors"] for s in successful_sessions)
                
                stress_results[multiplier] = {
                    "target_users": stress_users,
                    "successful_sessions": len(successful_sessions),
                    "total_requests": total_requests,
                    "total_errors": total_errors,
                    "error_rate": (total_errors / total_requests) * 100 if total_requests > 0 else 0,
                    "requests_per_second": total_requests / (end_time - start_time),
                    "system_breaking_point": total_errors > total_requests * 0.5  # 50% error rate = breaking point
                }
                
                error_rate = stress_results[multiplier]["error_rate"]
                rps = stress_results[multiplier]["requests_per_second"]
                
                if error_rate > 50:
                    print(f"    🔥 {multiplier}x: BREAKING POINT - {error_rate:.1f}% errors")
                    break
                else:
                    print(f"    ✅ {multiplier}x: {rps:.1f} RPS, {error_rate:.1f}% errors")
        
        self.results["stress_testing"] = stress_results
    
    async def _run_endurance_testing(self):
        """Run endurance testing with sustained load"""
        print("\n⏰ Phase 4: Endurance Testing (Sustained Load)")
        
        if self.stop_event.is_set():
            return
        
        # Use 50% of max successful load for endurance test
        endurance_users = 50
        endurance_duration = 600  # 10 minutes
        
        print(f"  Running endurance test: {endurance_users} users for {endurance_duration//60} minutes")
        
        sessions = []
        for i in range(endurance_users):
            session = UserSession(
                user_id=f"endurance_user_{self.test_id}_{i}",
                base_url=LOAD_TEST_CONFIG["api_base_url"],
                headers=self.headers,
                think_time=2.0  # Slower pace for endurance
            )
            sessions.append(session)
        
        start_time = time.time()
        
        tasks = []
        for session in sessions:
            task = asyncio.create_task(
                session.run_user_scenario(
                    duration=endurance_duration,
                    ramp_up=30,  # 30 second ramp up
                    stop_event=self.stop_event
                )
            )
            tasks.append(task)
        
        # Monitor performance over time
        monitoring_task = asyncio.create_task(
            self._monitor_endurance_performance(endurance_duration, start_time)
        )
        
        session_results = await asyncio.gather(*tasks, return_exceptions=True)
        await monitoring_task
        
        end_time = time.time()
        
        # Analyze endurance results
        successful_sessions = [r for r in session_results if isinstance(r, dict)]
        
        if successful_sessions:
            total_requests = sum(s["total_requests"] for s in successful_sessions)
            total_errors = sum(s["total_errors"] for s in successful_sessions)
            
            self.results["endurance_testing"] = {
                "duration": end_time - start_time,
                "target_users": endurance_users,
                "successful_sessions": len(successful_sessions),
                "total_requests": total_requests,
                "total_errors": total_errors,
                "error_rate": (total_errors / total_requests) * 100 if total_requests > 0 else 0,
                "requests_per_second": total_requests / (end_time - start_time),
                "stability_achieved": total_errors < total_requests * 0.1  # Less than 10% errors
            }
            
            error_rate = self.results["endurance_testing"]["error_rate"]
            print(f"    ✅ Endurance test: {error_rate:.1f}% error rate over {endurance_duration//60} minutes")
    
    async def _run_spike_testing(self):
        """Run spike testing with sudden load increases"""
        print("\n📈 Phase 5: Spike Testing (Sudden Load Increases)")
        
        if self.stop_event.is_set():
            return
        
        # Simulate sudden spikes in traffic
        base_users = 20
        spike_users = 200
        spike_duration = 60  # 1 minute spike
        
        print(f"  Simulating traffic spike: {base_users} → {spike_users} users")
        
        # Start with base load
        base_sessions = []
        for i in range(base_users):
            session = UserSession(
                user_id=f"base_user_{self.test_id}_{i}",
                base_url=LOAD_TEST_CONFIG["api_base_url"],
                headers=self.headers,
                think_time=2.0
            )
            base_sessions.append(session)
        
        # Start base load
        base_tasks = []
        for session in base_sessions:
            task = asyncio.create_task(
                session.run_user_scenario(
                    duration=180,  # 3 minutes total
                    ramp_up=0,
                    stop_event=self.stop_event
                )
            )
            base_tasks.append(task)
        
        # Wait 60 seconds, then add spike
        await asyncio.sleep(60)
        
        if not self.stop_event.is_set():
            print("    🚀 Triggering traffic spike...")
            
            spike_sessions = []
            for i in range(spike_users - base_users):
                session = UserSession(
                    user_id=f"spike_user_{self.test_id}_{i}",
                    base_url=LOAD_TEST_CONFIG["api_base_url"],
                    headers=self.headers,
                    think_time=0.5  # Aggressive spike traffic
                )
                spike_sessions.append(session)
            
            spike_tasks = []
            for session in spike_sessions:
                task = asyncio.create_task(
                    session.run_user_scenario(
                        duration=spike_duration,
                        ramp_up=5,  # Very quick ramp up
                        stop_event=self.stop_event
                    )
                )
                spike_tasks.append(task)
            
            # Wait for spike to complete
            spike_results = await asyncio.gather(*spike_tasks, return_exceptions=True)
            base_results = await asyncio.gather(*base_tasks, return_exceptions=True)
            
            # Analyze spike impact
            successful_spike = [r for r in spike_results if isinstance(r, dict)]
            successful_base = [r for r in base_results if isinstance(r, dict)]
            
            if successful_spike and successful_base:
                spike_errors = sum(s["total_errors"] for s in successful_spike)
                spike_requests = sum(s["total_requests"] for s in successful_spike)
                
                self.results["spike_testing"] = {
                    "base_users": base_users,
                    "spike_users": spike_users,
                    "spike_duration": spike_duration,
                    "spike_error_rate": (spike_errors / spike_requests) * 100 if spike_requests > 0 else 0,
                    "spike_handled": spike_errors < spike_requests * 0.2  # Less than 20% errors during spike
                }
                
                spike_error_rate = self.results["spike_testing"]["spike_error_rate"]
                print(f"    ✅ Spike test: {spike_error_rate:.1f}% error rate during spike")
    
    async def _run_volume_testing(self):
        """Run volume testing with large data sets"""
        print("\n💾 Phase 6: Volume Testing (Large Data Sets)")
        
        if self.stop_event.is_set():
            return
        
        # Test bulk data ingestion
        volume_sizes = [1000, 5000, 10000, 25000]
        volume_results = {}
        
        for volume_size in volume_sizes:
            if self.stop_event.is_set():
                break
                
            print(f"  Testing bulk ingestion: {volume_size} transactions")
            
            # Generate large transaction dataset
            transactions = []
            for i in range(volume_size):
                transactions.append({
                    "transaction_id": f"VOLUME_{self.test_id}_{i:06d}",
                    "sender_id": f"sender_{i % 1000}",
                    "receiver_id": f"receiver_{(i + 500) % 1000}",
                    "amount": float(100 + (i % 50000)),
                    "currency": "USD",
                    "timestamp": (datetime.utcnow() - timedelta(seconds=i)).isoformat(),
                    "transaction_type": "wire_transfer",
                    "source_system": "volume_test"
                })
            
            start_time = time.time()
            
            try:
                async with aiohttp.ClientSession() as session:
                    response = await session.post(
                        f"{LOAD_TEST_CONFIG['data_ingestion_url']}/transactions/bulk",
                        json={"transactions": transactions},
                        headers=self.headers,
                        timeout=aiohttp.ClientTimeout(total=300)  # 5 minute timeout
                    )
                    
                    end_time = time.time()
                    duration = end_time - start_time
                    
                    if response.status in [200, 201]:
                        throughput = volume_size / duration
                        
                        volume_results[volume_size] = {
                            "duration": duration,
                            "throughput": throughput,
                            "status": "success"
                        }
                        
                        print(f"    ✅ {volume_size} transactions: {throughput:.1f} TPS in {duration:.1f}s")
                    else:
                        print(f"    ❌ {volume_size} transactions: Failed with status {response.status}")
                        volume_results[volume_size] = {"status": "failed"}
                        
            except Exception as e:
                print(f"    ❌ {volume_size} transactions: Error - {str(e)}")
                volume_results[volume_size] = {"status": "error", "error": str(e)}
        
        self.results["volume_testing"] = volume_results
    
    async def _monitor_endurance_performance(self, duration: int, start_time: float):
        """Monitor performance during endurance test"""
        monitoring_interval = 30  # Monitor every 30 seconds
        performance_samples = []
        
        while time.time() - start_time < duration and not self.stop_event.is_set():
            try:
                # Sample system performance
                async with aiohttp.ClientSession() as session:
                    response = await session.get(
                        f"{LOAD_TEST_CONFIG['api_base_url']}/health",
                        headers=self.headers,
                        timeout=aiohttp.ClientTimeout(total=5)
                    )
                    
                    sample_time = time.time() - start_time
                    if response.status == 200:
                        performance_samples.append({
                            "time": sample_time,
                            "status": "healthy",
                            "response_time": response.headers.get("response-time", "unknown")
                        })
                    else:
                        performance_samples.append({
                            "time": sample_time,
                            "status": "degraded"
                        })
                        
            except Exception as e:
                performance_samples.append({
                    "time": time.time() - start_time,
                    "status": "error",
                    "error": str(e)
                })
            
            await asyncio.sleep(monitoring_interval)
        
        self.results["endurance_monitoring"] = performance_samples
    
    # Helper methods
    
    async def _test_transaction_ingestion_single(self) -> bool:
        """Single transaction ingestion test"""
        transaction = {
            "transaction_id": f"BASELINE_{uuid.uuid4().hex[:8]}",
            "sender_id": "test_sender",
            "receiver_id": "test_receiver",
            "amount": 1000.00,
            "currency": "USD",
            "timestamp": datetime.utcnow().isoformat(),
            "transaction_type": "wire_transfer"
        }
        
        try:
            async with aiohttp.ClientSession() as session:
                response = await session.post(
                    f"{LOAD_TEST_CONFIG['data_ingestion_url']}/transactions",
                    json=transaction,
                    headers=self.headers,
                    timeout=aiohttp.ClientTimeout(total=10)
                )
                return response.status in [200, 201]
        except Exception:
            return False
    
    async def _test_entity_creation_single(self) -> bool:
        """Single entity creation test"""
        entity = {
            "name": f"Test Entity {uuid.uuid4().hex[:8]}",
            "entity_type": "individual",
            "country": "US"
        }
        
        try:
            async with aiohttp.ClientSession() as session:
                response = await session.post(
                    f"{LOAD_TEST_CONFIG['data_ingestion_url']}/entities",
                    json=entity,
                    headers=self.headers,
                    timeout=aiohttp.ClientTimeout(total=10)
                )
                return response.status in [200, 201]
        except Exception:
            return False
    
    async def _test_graph_query_single(self) -> bool:
        """Single graph query test"""
        try:
            async with aiohttp.ClientSession() as session:
                response = await session.post(
                    f"{LOAD_TEST_CONFIG['graph_engine_url']}/traverse",
                    json={
                        "start_entity": "test_entity",
                        "max_depth": 2,
                        "limit": 100
                    },
                    headers=self.headers,
                    timeout=aiohttp.ClientTimeout(total=10)
                )
                return response.status == 200
        except Exception:
            return False
    
    async def _test_alert_retrieval_single(self) -> bool:
        """Single alert retrieval test"""
        try:
            async with aiohttp.ClientSession() as session:
                response = await session.get(
                    f"{LOAD_TEST_CONFIG['api_base_url']}/alerts",
                    headers=self.headers,
                    timeout=aiohttp.ClientTimeout(total=10)
                )
                return response.status == 200
        except Exception:
            return False
    
    async def _test_investigation_workflow_single(self) -> bool:
        """Single investigation workflow test"""
        investigation = {
            "title": f"Load Test Investigation {uuid.uuid4().hex[:8]}",
            "description": "Load testing investigation",
            "investigation_type": "aml_investigation",
            "priority": "medium"
        }
        
        try:
            async with aiohttp.ClientSession() as session:
                response = await session.post(
                    f"{LOAD_TEST_CONFIG['api_base_url']}/investigations",
                    json=investigation,
                    headers=self.headers,
                    timeout=aiohttp.ClientTimeout(total=10)
                )
                return response.status in [200, 201]
        except Exception:
            return False
    
    def _percentile(self, data: List[float], percentile: float) -> float:
        """Calculate percentile of data"""
        if not data:
            return 0
        
        sorted_data = sorted(data)
        index = (percentile / 100) * (len(sorted_data) - 1)
        
        if index.is_integer():
            return sorted_data[int(index)]
        else:
            lower = sorted_data[int(index)]
            upper = sorted_data[int(index) + 1]
            return lower + (upper - lower) * (index - int(index))
    
    async def _generate_load_test_report(self):
        """Generate comprehensive load test report"""
        print("\n📋 Generating Load Test Report")
        
        report = {
            "test_suite": "AegisShield Load & Stress Testing",
            "test_id": self.test_id,
            "timestamp": datetime.utcnow().isoformat(),
            "configuration": LOAD_TEST_CONFIG,
            "results": self.results,
            "analysis": self._analyze_load_test_results(),
            "recommendations": self._generate_performance_recommendations()
        }
        
        # Save detailed report
        report_filename = f"load_test_report_{self.test_id}.json"
        with open(report_filename, 'w') as f:
            json.dump(report, f, indent=2)
        
        # Print summary
        print("=" * 80)
        print("LOAD & STRESS TEST SUMMARY")
        print("=" * 80)
        
        # Baseline Performance
        if "baseline_performance" in self.results:
            print(f"\n📊 Baseline Performance:")
            for test, metrics in self.results["baseline_performance"].items():
                print(f"  {test}: {metrics['average_time']:.3f}s average")
        
        # Load Testing Results
        if "load_testing" in self.results:
            print(f"\n👥 Load Testing Results:")
            for users, metrics in self.results["load_testing"].items():
                if isinstance(metrics, dict) and "requests_per_second" in metrics:
                    rps = metrics["requests_per_second"]
                    error_rate = metrics["error_rate"]
                    print(f"  {users} users: {rps:.1f} RPS, {error_rate:.1f}% errors")
        
        # Stress Testing Results
        if "stress_testing" in self.results:
            print(f"\n🔥 Stress Testing Results:")
            for multiplier, metrics in self.results["stress_testing"].items():
                if "error_rate" in metrics:
                    error_rate = metrics["error_rate"]
                    breaking_point = metrics.get("system_breaking_point", False)
                    status = "BREAKING POINT" if breaking_point else "STABLE"
                    print(f"  {multiplier}x load: {error_rate:.1f}% errors ({status})")
        
        # Performance Recommendations
        recommendations = report["recommendations"]
        if recommendations:
            print(f"\n💡 Performance Recommendations:")
            for i, rec in enumerate(recommendations, 1):
                print(f"  {i}. {rec}")
        
        print(f"\n✅ Load test report saved to: {report_filename}")
    
    def _analyze_load_test_results(self) -> Dict:
        """Analyze load test results for patterns and insights"""
        analysis = {
            "performance_trends": {},
            "capacity_limits": {},
            "stability_assessment": {}
        }
        
        # Analyze load testing trends
        if "load_testing" in self.results:
            load_data = self.results["load_testing"]
            
            successful_configs = {}
            for users, metrics in load_data.items():
                if isinstance(metrics, dict) and metrics.get("error_rate", 100) < 10:
                    successful_configs[users] = metrics["requests_per_second"]
            
            if successful_configs:
                max_users = max(successful_configs.keys())
                max_rps = max(successful_configs.values())
                
                analysis["capacity_limits"] = {
                    "max_concurrent_users": max_users,
                    "max_requests_per_second": max_rps,
                    "estimated_capacity": max_rps * 1.2  # 20% buffer
                }
        
        # Analyze stress testing
        if "stress_testing" in self.results:
            stress_data = self.results["stress_testing"]
            
            breaking_point_found = False
            for multiplier, metrics in stress_data.items():
                if metrics.get("system_breaking_point", False):
                    analysis["capacity_limits"]["breaking_point_multiplier"] = multiplier
                    breaking_point_found = True
                    break
            
            if not breaking_point_found:
                analysis["capacity_limits"]["breaking_point_multiplier"] = "Not reached"
        
        # Analyze stability
        if "endurance_testing" in self.results:
            endurance_data = self.results["endurance_testing"]
            analysis["stability_assessment"] = {
                "stable_under_load": endurance_data.get("stability_achieved", False),
                "endurance_error_rate": endurance_data.get("error_rate", 0)
            }
        
        return analysis
    
    def _generate_performance_recommendations(self) -> List[str]:
        """Generate performance optimization recommendations"""
        recommendations = []
        
        # Check baseline performance
        if "baseline_performance" in self.results:
            baseline = self.results["baseline_performance"]
            
            for test, metrics in baseline.items():
                if metrics["average_time"] > 2.0:  # Slow baseline
                    recommendations.append(
                        f"Optimize {test} - baseline response time is {metrics['average_time']:.3f}s"
                    )
        
        # Check load testing results
        if "load_testing" in self.results:
            load_data = self.results["load_testing"]
            
            high_error_rates = []
            for users, metrics in load_data.items():
                if isinstance(metrics, dict) and metrics.get("error_rate", 0) > 5:
                    high_error_rates.append(users)
            
            if high_error_rates:
                recommendations.append(
                    f"Address error rates above 5% starting at {min(high_error_rates)} concurrent users"
                )
        
        # Check stress testing
        if "stress_testing" in self.results:
            stress_data = self.results["stress_testing"]
            
            if any(m.get("system_breaking_point", False) for m in stress_data.values()):
                recommendations.append(
                    "System breaking point identified - consider horizontal scaling or resource optimization"
                )
        
        # Check endurance testing
        if "endurance_testing" in self.results:
            endurance = self.results["endurance_testing"]
            
            if not endurance.get("stability_achieved", False):
                recommendations.append(
                    "System not stable under sustained load - investigate memory leaks or resource contention"
                )
        
        # General recommendations
        if not recommendations:
            recommendations.append("System performed well under all tested conditions")
        else:
            recommendations.append("Consider implementing auto-scaling policies for production deployment")
            recommendations.append("Monitor system resources (CPU, memory, I/O) during peak loads")
        
        return recommendations


class UserSession:
    """Simulates a user session with realistic behavior patterns"""
    
    def __init__(self, user_id: str, base_url: str, headers: Dict, think_time: float = 1.0):
        self.user_id = user_id
        self.base_url = base_url
        self.headers = headers
        self.think_time = think_time
        self.session_stats = {
            "total_requests": 0,
            "total_errors": 0,
            "response_times": []
        }
    
    async def run_user_scenario(self, duration: int, ramp_up: float, stop_event: threading.Event) -> Dict:
        """Run realistic user scenario for specified duration"""
        
        # Stagger user start time (ramp up)
        await asyncio.sleep(ramp_up)
        
        start_time = time.time()
        end_time = start_time + duration
        
        async with aiohttp.ClientSession() as session:
            while time.time() < end_time and not stop_event.is_set():
                
                # Simulate user workflow
                await self._simulate_user_workflow(session)
                
                # Think time between actions
                await asyncio.sleep(self.think_time)
        
        return self.session_stats
    
    async def _simulate_user_workflow(self, session: aiohttp.ClientSession):
        """Simulate realistic user workflow"""
        
        # User workflow scenarios with different probabilities
        scenarios = [
            (0.3, self._view_dashboard),           # 30% - View dashboard
            (0.2, self._search_entities),          # 20% - Search entities
            (0.15, self._view_alerts),             # 15% - View alerts
            (0.15, self._investigate_transaction), # 15% - Investigate transaction
            (0.1, self._create_investigation),     # 10% - Create investigation
            (0.05, self._view_reports),            # 5% - View reports
            (0.05, self._admin_actions)            # 5% - Admin actions
        ]
        
        # Select scenario based on probability
        import random
        rand = random.random()
        cumulative = 0
        
        for probability, scenario_func in scenarios:
            cumulative += probability
            if rand <= cumulative:
                await scenario_func(session)
                break
    
    async def _make_request(self, session: aiohttp.ClientSession, method: str, url: str, **kwargs) -> bool:
        """Make HTTP request and track statistics"""
        start_time = time.time()
        
        try:
            async with session.request(method, url, headers=self.headers, 
                                     timeout=aiohttp.ClientTimeout(total=30), **kwargs) as response:
                end_time = time.time()
                response_time = end_time - start_time
                
                self.session_stats["total_requests"] += 1
                self.session_stats["response_times"].append(response_time)
                
                if response.status >= 400:
                    self.session_stats["total_errors"] += 1
                    return False
                
                return True
                
        except Exception:
            self.session_stats["total_requests"] += 1
            self.session_stats["total_errors"] += 1
            self.session_stats["response_times"].append(30.0)  # Timeout
            return False
    
    # User scenario implementations
    
    async def _view_dashboard(self, session: aiohttp.ClientSession):
        """Simulate viewing dashboard"""
        await self._make_request(session, "GET", f"{self.base_url}/dashboard")
        await self._make_request(session, "GET", f"{self.base_url}/analytics/summary")
    
    async def _search_entities(self, session: aiohttp.ClientSession):
        """Simulate entity search"""
        search_terms = ["john", "smith", "bank", "corp", "international"]
        import random
        term = random.choice(search_terms)
        
        await self._make_request(session, "GET", f"{self.base_url}/search", 
                               params={"query": term, "type": "entity"})
    
    async def _view_alerts(self, session: aiohttp.ClientSession):
        """Simulate viewing alerts"""
        await self._make_request(session, "GET", f"{self.base_url}/alerts")
        await self._make_request(session, "GET", f"{self.base_url}/alerts", 
                               params={"priority": "high"})
    
    async def _investigate_transaction(self, session: aiohttp.ClientSession):
        """Simulate transaction investigation"""
        # Search for transactions
        await self._make_request(session, "GET", f"{self.base_url}/transactions", 
                               params={"limit": 50})
        
        # View transaction details (simulate selecting one)
        transaction_id = f"sample_transaction_{uuid.uuid4().hex[:8]}"
        await self._make_request(session, "GET", f"{self.base_url}/transactions/{transaction_id}")
        
        # View related entities
        await self._make_request(session, "GET", f"{self.base_url}/graph/traverse", 
                               json={"start_entity": transaction_id, "max_depth": 2})
    
    async def _create_investigation(self, session: aiohttp.ClientSession):
        """Simulate creating investigation"""
        investigation_data = {
            "title": f"Load Test Investigation {uuid.uuid4().hex[:8]}",
            "description": "Investigation created during load testing",
            "investigation_type": "aml_investigation",
            "priority": "medium"
        }
        
        await self._make_request(session, "POST", f"{self.base_url}/investigations", 
                               json=investigation_data)
    
    async def _view_reports(self, session: aiohttp.ClientSession):
        """Simulate viewing reports"""
        await self._make_request(session, "GET", f"{self.base_url}/reports")
        await self._make_request(session, "GET", f"{self.base_url}/reports/compliance")
    
    async def _admin_actions(self, session: aiohttp.ClientSession):
        """Simulate admin actions"""
        await self._make_request(session, "GET", f"{self.base_url}/admin/users")
        await self._make_request(session, "GET", f"{self.base_url}/admin/system-status")


async def main():
    """Run load testing suite"""
    try:
        load_tester = LoadTestRunner()
        await load_tester.run_comprehensive_load_tests()
        return 0
    except KeyboardInterrupt:
        print("\n⚠️ Load testing interrupted")
        return 1
    except Exception as e:
        print(f"\n❌ Load testing failed: {e}")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())