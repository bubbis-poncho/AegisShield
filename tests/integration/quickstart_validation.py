#!/usr/bin/env python3
"""
Quickstart validation script for AegisShield platform
Validates complete end-to-end functionality according to quickstart.md
"""

import requests
import json
import time
import subprocess
import os
import sys
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
import uuid
import asyncio

@dataclass
class ValidationConfig:
    """Configuration for quickstart validation"""
    api_base_url: str = "http://localhost:8080"
    frontend_base_url: str = "http://localhost:3000"
    auth_token: str = ""
    test_timeout: int = 30
    
    # Test data
    test_person: Dict = None
    test_organization: Dict = None
    test_accounts: List[Dict] = None
    test_transactions: List[Dict] = None
    
    def __post_init__(self):
        if self.test_person is None:
            self.test_person = {
                "name": "John Smith",
                "ssn": "123-45-6789",
                "address": "123 Main St, New York, NY",
                "risk_profile": "medium"
            }
        
        if self.test_organization is None:
            self.test_organization = {
                "name": "ABC Trading Corp",
                "tax_id": "12-3456789",
                "address": "789 Business Blvd, Chicago, IL",
                "industry": "Import/Export"
            }
        
        if self.test_accounts is None:
            self.test_accounts = [
                {
                    "account_number": "1001234567",
                    "holder": "John Smith",
                    "bank": "First National Bank",
                    "type": "checking"
                },
                {
                    "account_number": "2009876543",
                    "holder": "ABC Trading Corp",
                    "bank": "Commerce Bank",
                    "type": "business"
                }
            ]
        
        if self.test_transactions is None:
            self.test_transactions = [
                {
                    "source_account": "1001234567",
                    "destination_account": "2009876543",
                    "amount": 9500.00,
                    "currency": "USD",
                    "transaction_date": "2025-01-15T10:30:00Z",
                    "description": "Business payment"
                },
                {
                    "source_account": "2009876543",
                    "destination_account": "1001234567",
                    "amount": 9000.00,
                    "currency": "USD",
                    "transaction_date": "2025-01-15T14:45:00Z",
                    "description": "Refund"
                }
            ]

@dataclass
class ValidationResult:
    """Result from validation test"""
    test_name: str
    passed: bool
    duration_seconds: float
    details: Dict[str, Any]
    error_message: Optional[str] = None

class QuickstartValidator:
    """Validates quickstart scenarios end-to-end"""
    
    def __init__(self, config: ValidationConfig):
        self.config = config
        self.session = requests.Session()
        self.results = []
        self.test_entities = {}
        
    def setup_authentication(self) -> bool:
        """Setup authentication for API calls"""
        try:
            # Try to authenticate with default credentials
            response = self.session.post(
                f"{self.config.api_base_url}/auth/login",
                json={
                    "email": "admin@aegisshield.com",
                    "password": "admin_password_123"
                },
                timeout=self.config.test_timeout
            )
            
            if response.status_code == 200:
                data = response.json()
                self.config.auth_token = data.get("access_token", "")
                self.session.headers.update({
                    "Authorization": f"Bearer {self.config.auth_token}"
                })
                return True
            else:
                print(f"Authentication failed: {response.status_code}")
                return False
                
        except Exception as e:
            print(f"Authentication setup failed: {e}")
            return False
    
    def check_infrastructure_health(self) -> ValidationResult:
        """Check that all infrastructure services are running"""
        print("Checking infrastructure health...")
        start_time = time.time()
        
        try:
            # Check API gateway
            api_response = self.session.get(
                f"{self.config.api_base_url}/health",
                timeout=self.config.test_timeout
            )
            
            # Check frontend
            frontend_response = requests.get(
                f"{self.config.frontend_base_url}/health",
                timeout=self.config.test_timeout
            )
            
            # Check Kubernetes services
            try:
                kubectl_result = subprocess.run(
                    ["kubectl", "get", "pods", "-n", "aegisshield"],
                    capture_output=True,
                    text=True,
                    timeout=10
                )
                pods_healthy = "Running" in kubectl_result.stdout
            except:
                pods_healthy = False
            
            duration = time.time() - start_time
            
            all_healthy = (
                api_response.status_code == 200 and
                frontend_response.status_code == 200 and
                pods_healthy
            )
            
            return ValidationResult(
                test_name="Infrastructure Health Check",
                passed=all_healthy,
                duration_seconds=duration,
                details={
                    "api_status": api_response.status_code,
                    "frontend_status": frontend_response.status_code,
                    "kubernetes_pods": pods_healthy
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Infrastructure Health Check",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def test_data_ingestion(self) -> ValidationResult:
        """Test Scenario 1: Data Ingestion (Expected: <30 seconds)"""
        print("Testing data ingestion...")
        start_time = time.time()
        
        try:
            # Ingest transaction data
            response = self.session.post(
                f"{self.config.api_base_url}/api/v1/data/transactions",
                json={"transactions": self.config.test_transactions},
                timeout=self.config.test_timeout
            )
            
            if response.status_code != 201:
                return ValidationResult(
                    test_name="Data Ingestion",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={"response_code": response.status_code},
                    error_message=f"Expected 201, got {response.status_code}"
                )
            
            # Check ingestion status
            status_response = self.session.get(
                f"{self.config.api_base_url}/api/v1/data/status",
                timeout=self.config.test_timeout
            )
            
            if status_response.status_code == 200:
                status_data = status_response.json()
                processed_count = status_data.get("processed_count", 0)
            else:
                processed_count = 0
            
            duration = time.time() - start_time
            success = duration < 30 and processed_count >= 2
            
            return ValidationResult(
                test_name="Data Ingestion",
                passed=success,
                duration_seconds=duration,
                details={
                    "processed_count": processed_count,
                    "expected_duration": "<30 seconds",
                    "actual_duration": f"{duration:.2f} seconds"
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Data Ingestion",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def test_entity_resolution(self) -> ValidationResult:
        """Test Scenario 2: Entity Resolution (Expected: <60 seconds)"""
        print("Testing entity resolution...")
        start_time = time.time()
        
        try:
            # Wait for entity resolution to process
            time.sleep(5)
            
            # Search for John Smith entity
            response = self.session.get(
                f"{self.config.api_base_url}/api/v1/entities/search",
                params={"q": "John Smith"},
                timeout=self.config.test_timeout
            )
            
            if response.status_code != 200:
                return ValidationResult(
                    test_name="Entity Resolution",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={"response_code": response.status_code},
                    error_message=f"Entity search failed: {response.status_code}"
                )
            
            entities = response.json().get("entities", [])
            john_smith_entity = None
            
            for entity in entities:
                if "John Smith" in entity.get("name", ""):
                    john_smith_entity = entity
                    break
            
            duration = time.time() - start_time
            success = (
                duration < 60 and
                john_smith_entity is not None and
                john_smith_entity.get("type") == "person"
            )
            
            if john_smith_entity:
                self.test_entities["john_smith_id"] = john_smith_entity.get("id")
            
            return ValidationResult(
                test_name="Entity Resolution",
                passed=success,
                duration_seconds=duration,
                details={
                    "entity_found": john_smith_entity is not None,
                    "entity_type": john_smith_entity.get("type") if john_smith_entity else None,
                    "expected_duration": "<60 seconds",
                    "actual_duration": f"{duration:.2f} seconds"
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Entity Resolution",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def test_alert_generation(self) -> ValidationResult:
        """Test Scenario 3: Alert Generation (Expected: <120 seconds)"""
        print("Testing alert generation...")
        start_time = time.time()
        
        try:
            # Wait for alert processing
            time.sleep(10)
            
            john_smith_id = self.test_entities.get("john_smith_id")
            if not john_smith_id:
                # Try to find the entity again
                search_response = self.session.get(
                    f"{self.config.api_base_url}/api/v1/entities/search",
                    params={"q": "John Smith"},
                    timeout=self.config.test_timeout
                )
                
                if search_response.status_code == 200:
                    entities = search_response.json().get("entities", [])
                    if entities:
                        john_smith_id = entities[0].get("id")
            
            if not john_smith_id:
                return ValidationResult(
                    test_name="Alert Generation",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={},
                    error_message="Could not find John Smith entity for alert testing"
                )
            
            # Check for alerts
            response = self.session.get(
                f"{self.config.api_base_url}/api/v1/alerts",
                params={"entity_id": john_smith_id},
                timeout=self.config.test_timeout
            )
            
            if response.status_code != 200:
                return ValidationResult(
                    test_name="Alert Generation",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={"response_code": response.status_code},
                    error_message=f"Alert query failed: {response.status_code}"
                )
            
            alerts = response.json().get("alerts", [])
            structuring_alert = None
            
            for alert in alerts:
                if "structuring" in alert.get("alert_type", "").lower():
                    structuring_alert = alert
                    break
            
            duration = time.time() - start_time
            success = (
                duration < 120 and
                structuring_alert is not None and
                structuring_alert.get("severity", "").lower() in ["high", "critical"] and
                float(structuring_alert.get("confidence_score", 0)) > 0.8
            )
            
            if structuring_alert:
                self.test_entities["alert_id"] = structuring_alert.get("id")
            
            return ValidationResult(
                test_name="Alert Generation",
                passed=success,
                duration_seconds=duration,
                details={
                    "alert_found": structuring_alert is not None,
                    "alert_type": structuring_alert.get("alert_type") if structuring_alert else None,
                    "severity": structuring_alert.get("severity") if structuring_alert else None,
                    "confidence_score": structuring_alert.get("confidence_score") if structuring_alert else None,
                    "expected_duration": "<120 seconds",
                    "actual_duration": f"{duration:.2f} seconds"
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Alert Generation",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def test_investigation_workflow(self) -> ValidationResult:
        """Test Scenario 4: Investigation Workflow (Expected: <180 seconds)"""
        print("Testing investigation workflow...")
        start_time = time.time()
        
        try:
            alert_id = self.test_entities.get("alert_id")
            if not alert_id:
                return ValidationResult(
                    test_name="Investigation Workflow",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={},
                    error_message="No alert ID available for investigation testing"
                )
            
            # Start investigation from alert
            response = self.session.post(
                f"{self.config.api_base_url}/api/v1/alerts/{alert_id}/investigate",
                json={
                    "investigation_title": "Potential Structuring - John Smith",
                    "investigation_notes": "Reviewing transactions below reporting threshold"
                },
                timeout=self.config.test_timeout
            )
            
            if response.status_code != 201:
                return ValidationResult(
                    test_name="Investigation Workflow",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={"response_code": response.status_code},
                    error_message=f"Investigation creation failed: {response.status_code}"
                )
            
            investigation_data = response.json()
            investigation_id = investigation_data.get("id")
            
            if investigation_id:
                self.test_entities["investigation_id"] = investigation_id
            
            duration = time.time() - start_time
            success = (
                duration < 180 and
                investigation_id is not None and
                investigation_data.get("status") in ["active", "open", "in_progress"]
            )
            
            return ValidationResult(
                test_name="Investigation Workflow",
                passed=success,
                duration_seconds=duration,
                details={
                    "investigation_created": investigation_id is not None,
                    "investigation_id": investigation_id,
                    "status": investigation_data.get("status"),
                    "expected_duration": "<180 seconds",
                    "actual_duration": f"{duration:.2f} seconds"
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Investigation Workflow",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def test_graph_exploration(self) -> ValidationResult:
        """Test Scenario 5: Graph Exploration (Expected: <5 seconds)"""
        print("Testing graph exploration...")
        start_time = time.time()
        
        try:
            john_smith_id = self.test_entities.get("john_smith_id")
            if not john_smith_id:
                return ValidationResult(
                    test_name="Graph Exploration",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={},
                    error_message="No John Smith entity ID available for graph testing"
                )
            
            # Explore entity relationships
            response = self.session.post(
                f"{self.config.api_base_url}/api/v1/graph/explore",
                json={
                    "entity_id": john_smith_id,
                    "depth": 2,
                    "min_strength": 0.3
                },
                timeout=self.config.test_timeout
            )
            
            if response.status_code != 200:
                return ValidationResult(
                    test_name="Graph Exploration",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={"response_code": response.status_code},
                    error_message=f"Graph exploration failed: {response.status_code}"
                )
            
            graph_data = response.json()
            nodes = graph_data.get("nodes", [])
            edges = graph_data.get("edges", [])
            
            # Check for expected entities
            has_person = any(node.get("type") == "Person" for node in nodes)
            has_account = any(node.get("type") == "Account" for node in nodes)
            has_relationships = len(edges) > 0
            
            duration = time.time() - start_time
            success = (
                duration < 5 and
                has_person and
                has_account and
                has_relationships
            )
            
            return ValidationResult(
                test_name="Graph Exploration",
                passed=success,
                duration_seconds=duration,
                details={
                    "nodes_count": len(nodes),
                    "edges_count": len(edges),
                    "has_person": has_person,
                    "has_account": has_account,
                    "expected_duration": "<5 seconds",
                    "actual_duration": f"{duration:.2f} seconds"
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Graph Exploration",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def test_sanctions_screening(self) -> ValidationResult:
        """Test Scenario 6: Sanctions Screening"""
        print("Testing sanctions screening...")
        start_time = time.time()
        
        try:
            # Upload sanctions list
            sanctions_response = self.session.post(
                f"{self.config.api_base_url}/api/v1/data/sanctions",
                json={
                    "list_type": "OFAC_SDN",
                    "records": [
                        {
                            "name": "Maria Garcia",
                            "aliases": ["M. Garcia", "Garcia, Maria"],
                            "program": "COUNTER-TERRORISM",
                            "risk_level": "HIGH"
                        }
                    ]
                },
                timeout=self.config.test_timeout
            )
            
            if sanctions_response.status_code != 201:
                return ValidationResult(
                    test_name="Sanctions Screening",
                    passed=False,
                    duration_seconds=time.time() - start_time,
                    details={"sanctions_upload_status": sanctions_response.status_code},
                    error_message=f"Sanctions upload failed: {sanctions_response.status_code}"
                )
            
            # Attempt transaction involving sanctioned entity
            transaction_response = self.session.post(
                f"{self.config.api_base_url}/api/v1/data/transactions",
                json={
                    "transactions": [
                        {
                            "source_account": "3001234567",
                            "destination_account": "4009876543",
                            "amount": 5000.00,
                            "currency": "USD",
                            "description": "Payment to Maria Garcia"
                        }
                    ]
                },
                timeout=self.config.test_timeout
            )
            
            # Check for sanctions alert
            time.sleep(2)  # Allow time for alert processing
            
            alerts_response = self.session.get(
                f"{self.config.api_base_url}/api/v1/alerts",
                params={"alert_type": "sanctions"},
                timeout=self.config.test_timeout
            )
            
            sanctions_alert = None
            if alerts_response.status_code == 200:
                alerts = alerts_response.json().get("alerts", [])
                for alert in alerts:
                    if "Maria Garcia" in alert.get("title", "") or alert.get("alert_type") == "Sanctions":
                        sanctions_alert = alert
                        break
            
            duration = time.time() - start_time
            success = (
                sanctions_alert is not None and
                sanctions_alert.get("severity", "").lower() in ["critical", "high"] and
                float(sanctions_alert.get("confidence_score", 0)) > 0.9
            )
            
            return ValidationResult(
                test_name="Sanctions Screening",
                passed=success,
                duration_seconds=duration,
                details={
                    "sanctions_uploaded": sanctions_response.status_code == 201,
                    "transaction_processed": transaction_response.status_code in [201, 202],
                    "alert_generated": sanctions_alert is not None,
                    "alert_severity": sanctions_alert.get("severity") if sanctions_alert else None,
                    "confidence_score": sanctions_alert.get("confidence_score") if sanctions_alert else None
                }
            )
            
        except Exception as e:
            return ValidationResult(
                test_name="Sanctions Screening",
                passed=False,
                duration_seconds=time.time() - start_time,
                details={},
                error_message=str(e)
            )
    
    def run_quickstart_validation(self) -> Dict[str, Any]:
        """Execute complete quickstart validation"""
        print("🚀 Starting Quickstart Validation")
        print("=" * 60)
        
        # Setup authentication
        if not self.setup_authentication():
            return {
                "error": "Failed to setup authentication",
                "total_tests": 0,
                "passed_tests": 0,
                "failed_tests": 0
            }
        
        # Execute validation tests
        self.results = [
            self.check_infrastructure_health(),
            self.test_data_ingestion(),
            self.test_entity_resolution(),
            self.test_alert_generation(),
            self.test_investigation_workflow(),
            self.test_graph_exploration(),
            self.test_sanctions_screening()
        ]
        
        # Calculate summary
        total_tests = len(self.results)
        passed_tests = sum(1 for r in self.results if r.passed)
        failed_tests = total_tests - passed_tests
        
        # Generate report
        summary = {
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "failed_tests": failed_tests,
            "success_rate": (passed_tests / total_tests) * 100 if total_tests > 0 else 0,
            "total_duration": sum(r.duration_seconds for r in self.results),
            "test_results": self.results
        }
        
        print("\n" + "=" * 60)
        print("📋 QUICKSTART VALIDATION SUMMARY")
        print("=" * 60)
        print(f"Total Tests: {total_tests}")
        print(f"Passed: {passed_tests}")
        print(f"Failed: {failed_tests}")
        print(f"Success Rate: {summary['success_rate']:.1f}%")
        print(f"Total Duration: {summary['total_duration']:.2f} seconds")
        
        print("\n--- Test Results ---")
        for result in self.results:
            status = "PASS" if result.passed else "FAIL"
            print(f"{result.test_name}: {status} ({result.duration_seconds:.2f}s)")
            if not result.passed and result.error_message:
                print(f"  Error: {result.error_message}")
        
        if summary['success_rate'] >= 100:
            print("\n🎉 All quickstart scenarios validated successfully!")
        elif summary['success_rate'] >= 80:
            print("\n✅ Most quickstart scenarios validated - minor issues detected")
        else:
            print("\n⚠️  Significant issues detected in quickstart validation")
        
        return summary


def main():
    """Main execution function"""
    config = ValidationConfig()
    
    # Override config from environment if available
    config.api_base_url = os.getenv("AEGIS_API_URL", config.api_base_url)
    config.frontend_base_url = os.getenv("AEGIS_FRONTEND_URL", config.frontend_base_url)
    
    validator = QuickstartValidator(config)
    summary = validator.run_quickstart_validation()
    
    # Exit with appropriate code
    if summary.get("success_rate", 0) >= 80:
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()