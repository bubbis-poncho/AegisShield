#!/usr/bin/env python3
"""
End-to-end test for sanctions screening workflow
Tests integration with sanctions lists and compliance procedures
"""

import pytest
import asyncio
import httpx
import json
from datetime import datetime
from typing import Dict, List, Any
import os
import uuid

# Test configuration
API_BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8080")
DATA_INGESTION_URL = os.getenv("DATA_INGESTION_URL", "http://localhost:8060")
SANCTIONS_API_URL = os.getenv("SANCTIONS_API_URL", "http://localhost:8064")

class TestSanctionsScreening:
    """End-to-end test suite for sanctions screening workflow"""
    
    @pytest.fixture(autouse=True)
    async def setup_test_data(self):
        """Setup test data and clean up after test"""
        self.test_id = str(uuid.uuid4())[:8]
        self.test_entities = []
        self.test_transactions = []
        self.test_alerts = []
        self.test_investigations = []
        
        # Setup authentication
        self.auth_token = await self._get_auth_token()
        self.headers = {"Authorization": f"Bearer {self.auth_token}"}
        
        # Setup test sanctions data
        await self._setup_test_sanctions_data()
        
        yield
        
        # Cleanup
        await self._cleanup_test_data()
    
    async def _get_auth_token(self) -> str:
        """Get authentication token for API calls"""
        return "test-token-sanctions-12345"
    
    async def _setup_test_sanctions_data(self):
        """Setup test sanctions data"""
        self.test_sanctions_entities = [
            {
                "name": "Ivan Petrov",
                "aliases": ["I. Petrov", "Ivan P.", "Иван Петров"],
                "sanctions_lists": ["OFAC_SDN", "EU_SANCTIONS"],
                "entity_type": "individual",
                "country": "RU",
                "date_of_birth": "1975-03-15",
                "sanctions_date": "2022-02-26",
                "sanctions_reason": "Actions undermining sovereignty of Ukraine"
            },
            {
                "name": "Blocked Corporation LLC",
                "aliases": ["Blocked Corp", "BC LLC"],
                "sanctions_lists": ["OFAC_SDN"],
                "entity_type": "organization",
                "country": "RU",
                "registration_number": "RU123456789",
                "sanctions_date": "2022-04-08",
                "sanctions_reason": "Owned by sanctioned individual"
            },
            {
                "name": "Suspicious Bank",
                "aliases": ["SusBank", "S-Bank"],
                "sanctions_lists": ["OFAC_SSI"],
                "entity_type": "financial_institution",
                "country": "IR",
                "swift_code": "SUSBIRTE",
                "sanctions_date": "2023-01-15",
                "sanctions_reason": "Facilitating illicit transactions"
            }
        ]
    
    async def _cleanup_test_data(self):
        """Clean up test data"""
        try:
            async with httpx.AsyncClient() as client:
                # Clean up investigations
                for investigation_id in self.test_investigations:
                    await client.delete(
                        f"{API_BASE_URL}/investigations/{investigation_id}",
                        headers=self.headers
                    )
                
                # Clean up test entities
                for entity_id in self.test_entities:
                    await client.delete(
                        f"{DATA_INGESTION_URL}/entities/{entity_id}",
                        headers=self.headers
                    )
        except Exception as e:
            print(f"Cleanup warning: {e}")
    
    @pytest.mark.asyncio
    async def test_direct_sanctions_match(self):
        """Test direct sanctions list matching"""
        
        # Create entity that matches sanctions list
        sanctioned_entity_data = {
            "name": "Ivan Petrov",
            "entity_type": "individual",
            "country": "RU",
            "date_of_birth": "1975-03-15",
            "metadata": {
                "test_entity": True,
                "test_id": self.test_id
            }
        }
        
        # Ingest entity - should trigger sanctions screening
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=sanctioned_entity_data,
                headers=self.headers
            )
            assert response.status_code == 201
            entity = response.json()
            self.test_entities.append(entity["id"])
        
        # Wait for sanctions screening to complete
        await asyncio.sleep(5)
        
        # Verify sanctions match detected
        sanctions_result = await self._get_sanctions_screening_result(entity["id"])
        assert sanctions_result is not None
        assert sanctions_result["match_found"] == True
        assert sanctions_result["match_confidence"] > 0.9
        assert "OFAC_SDN" in sanctions_result["matched_lists"]
        assert len(sanctions_result["matches"]) > 0
        
        # Verify high-priority alert generated
        alert = await self._wait_for_sanctions_alert(entity["id"])
        assert alert is not None
        assert alert["alert_type"] == "sanctions_match"
        assert alert["priority"] == "critical"
        assert alert["risk_score"] > 0.95
        
        # Verify automatic investigation creation
        investigation = await self._wait_for_auto_investigation(alert["id"])
        assert investigation is not None
        assert investigation["investigation_type"] == "sanctions_violation"
        assert investigation["priority"] == "critical"
        assert investigation["status"] == "open"
        self.test_investigations.append(investigation["id"])
    
    @pytest.mark.asyncio
    async def test_fuzzy_sanctions_matching(self):
        """Test fuzzy matching for sanctions with name variations"""
        
        # Create entity with slight name variation
        fuzzy_entity_data = {
            "name": "I. Petrov",  # Alias of sanctioned "Ivan Petrov"
            "entity_type": "individual",
            "country": "RU",
            "metadata": {
                "test_entity": True,
                "test_id": self.test_id
            }
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=fuzzy_entity_data,
                headers=self.headers
            )
            assert response.status_code == 201
            entity = response.json()
            self.test_entities.append(entity["id"])
        
        # Wait for sanctions screening
        await asyncio.sleep(5)
        
        # Verify fuzzy match detected
        sanctions_result = await self._get_sanctions_screening_result(entity["id"])
        assert sanctions_result is not None
        assert sanctions_result["match_found"] == True
        assert sanctions_result["match_confidence"] > 0.7  # Lower confidence for fuzzy match
        assert sanctions_result["match_type"] == "fuzzy"
        
        # Verify alert with appropriate confidence
        alert = await self._wait_for_sanctions_alert(entity["id"])
        assert alert is not None
        assert alert["priority"] == "high"  # High but not critical for fuzzy match
    
    @pytest.mark.asyncio
    async def test_sanctions_transaction_blocking(self):
        """Test transaction blocking when sanctioned entity involved"""
        
        # Create normal entity
        normal_entity_data = {
            "name": f"Normal Entity {self.test_id}",
            "entity_type": "individual",
            "country": "US",
            "metadata": {"test_entity": True, "test_id": self.test_id}
        }
        
        # Create sanctioned entity
        sanctioned_entity_data = {
            "name": "Blocked Corporation LLC",
            "entity_type": "organization",
            "country": "RU",
            "registration_number": "RU123456789",
            "metadata": {"test_entity": True, "test_id": self.test_id}
        }
        
        async with httpx.AsyncClient() as client:
            # Create normal entity
            response1 = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=normal_entity_data,
                headers=self.headers
            )
            normal_entity = response1.json()
            self.test_entities.append(normal_entity["id"])
            
            # Create sanctioned entity
            response2 = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=sanctioned_entity_data,
                headers=self.headers
            )
            sanctioned_entity = response2.json()
            self.test_entities.append(sanctioned_entity["id"])
        
        # Wait for sanctions screening
        await asyncio.sleep(5)
        
        # Attempt transaction with sanctioned entity
        transaction_data = {
            "transaction_id": f"BLOCKED_{self.test_id}",
            "sender_id": normal_entity["id"],
            "receiver_id": sanctioned_entity["id"],
            "amount": 50000.00,
            "currency": "USD",
            "timestamp": datetime.utcnow().isoformat(),
            "transaction_type": "wire_transfer",
            "source_system": "test_system"
        }
        
        # Transaction should be blocked or flagged
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/transactions",
                json=transaction_data,
                headers=self.headers
            )
            
            # Transaction may be blocked (400) or accepted but flagged (201)
            if response.status_code == 400:
                # Transaction blocked
                error_data = response.json()
                assert "sanctions" in error_data["reason"].lower()
            elif response.status_code == 201:
                # Transaction accepted but should be flagged
                transaction = response.json()
                self.test_transactions.append(transaction["transaction_id"])
                
                # Verify immediate critical alert
                alert = await self._wait_for_sanctions_alert_transaction(transaction["transaction_id"])
                assert alert is not None
                assert alert["alert_type"] == "sanctions_transaction"
                assert alert["priority"] == "critical"
            else:
                pytest.fail(f"Unexpected response: {response.status_code}")
    
    @pytest.mark.asyncio
    async def test_sanctions_list_updates(self):
        """Test handling of sanctions list updates"""
        
        # Create entity that's not initially sanctioned
        entity_data = {
            "name": f"Future Sanctioned Entity {self.test_id}",
            "entity_type": "individual",
            "country": "XX",
            "metadata": {"test_entity": True, "test_id": self.test_id}
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=entity_data,
                headers=self.headers
            )
            entity = response.json()
            self.test_entities.append(entity["id"])
        
        # Initial screening should show no match
        await asyncio.sleep(2)
        initial_result = await self._get_sanctions_screening_result(entity["id"])
        assert initial_result["match_found"] == False
        
        # Simulate sanctions list update
        new_sanction_entry = {
            "name": entity_data["name"],
            "entity_type": "individual",
            "country": "XX",
            "sanctions_lists": ["OFAC_SDN"],
            "sanctions_date": datetime.utcnow().isoformat(),
            "sanctions_reason": "Test sanctions addition"
        }
        
        # Add to sanctions list (simulate update)
        await self._add_to_sanctions_list(new_sanction_entry)
        
        # Trigger re-screening
        await self._trigger_sanctions_rescreening(entity["id"])
        
        # Wait for re-screening
        await asyncio.sleep(5)
        
        # Verify entity now matches sanctions
        updated_result = await self._get_sanctions_screening_result(entity["id"])
        assert updated_result["match_found"] == True
        assert "OFAC_SDN" in updated_result["matched_lists"]
        
        # Verify alert generated for newly sanctioned entity
        alert = await self._wait_for_sanctions_alert(entity["id"])
        assert alert is not None
        assert alert["alert_type"] == "new_sanctions_match"
    
    @pytest.mark.asyncio
    async def test_compliance_reporting_sanctions(self):
        """Test sanctions compliance reporting"""
        
        # Create sanctioned entity and transaction
        sanctioned_entity = await self._create_sanctioned_entity()
        normal_entity = await self._create_normal_entity()
        
        transaction_data = {
            "transaction_id": f"SAR_{self.test_id}",
            "sender_id": normal_entity["id"],
            "receiver_id": sanctioned_entity["id"],
            "amount": 75000.00,
            "currency": "USD",
            "timestamp": datetime.utcnow().isoformat(),
            "transaction_type": "wire_transfer",
            "source_system": "test_system"
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/transactions",
                json=transaction_data,
                headers=self.headers
            )
            if response.status_code == 201:
                self.test_transactions.append(transaction_data["transaction_id"])
        
        # Wait for alert and investigation
        await asyncio.sleep(10)
        
        # Generate sanctions compliance report
        report_request = {
            "report_type": "sanctions_compliance",
            "start_date": (datetime.utcnow().replace(hour=0, minute=0, second=0)).isoformat(),
            "end_date": datetime.utcnow().isoformat(),
            "format": "json",
            "include_details": True
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/reports/generate",
                json=report_request,
                headers=self.headers
            )
            assert response.status_code == 200
            report = response.json()
        
        # Verify report contents
        assert report["report_type"] == "sanctions_compliance"
        assert len(report["sanctions_matches"]) > 0
        assert len(report["blocked_transactions"]) >= 0
        assert report["compliance_status"] in ["NON_COMPLIANT", "UNDER_REVIEW"]
        
        # Verify required regulatory fields
        sanctions_match = report["sanctions_matches"][0]
        assert "entity_name" in sanctions_match
        assert "sanctions_lists" in sanctions_match
        assert "match_confidence" in sanctions_match
        assert "transaction_details" in sanctions_match
    
    # Helper methods
    
    async def _get_sanctions_screening_result(self, entity_id: str) -> Dict:
        """Get sanctions screening result for entity"""
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{SANCTIONS_API_URL}/screening/{entity_id}",
                headers=self.headers
            )
            if response.status_code == 200:
                return response.json()
            return {"match_found": False}
    
    async def _wait_for_sanctions_alert(self, entity_id: str, timeout: int = 30) -> Dict:
        """Wait for sanctions alert generation"""
        start_time = asyncio.get_event_loop().time()
        
        while asyncio.get_event_loop().time() - start_time < timeout:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{API_BASE_URL}/alerts",
                    params={"entity_id": entity_id, "alert_type": "sanctions_match"},
                    headers=self.headers
                )
                
                if response.status_code == 200:
                    alerts = response.json().get("alerts", [])
                    if alerts:
                        return alerts[0]
            
            await asyncio.sleep(2)
        
        return None
    
    async def _wait_for_sanctions_alert_transaction(self, transaction_id: str, timeout: int = 30) -> Dict:
        """Wait for sanctions alert for transaction"""
        start_time = asyncio.get_event_loop().time()
        
        while asyncio.get_event_loop().time() - start_time < timeout:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{API_BASE_URL}/alerts",
                    params={"transaction_id": transaction_id, "alert_type": "sanctions_transaction"},
                    headers=self.headers
                )
                
                if response.status_code == 200:
                    alerts = response.json().get("alerts", [])
                    if alerts:
                        return alerts[0]
            
            await asyncio.sleep(2)
        
        return None
    
    async def _wait_for_auto_investigation(self, alert_id: str, timeout: int = 30) -> Dict:
        """Wait for automatic investigation creation"""
        start_time = asyncio.get_event_loop().time()
        
        while asyncio.get_event_loop().time() - start_time < timeout:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{API_BASE_URL}/investigations",
                    params={"alert_id": alert_id},
                    headers=self.headers
                )
                
                if response.status_code == 200:
                    investigations = response.json().get("investigations", [])
                    if investigations:
                        return investigations[0]
            
            await asyncio.sleep(2)
        
        return None
    
    async def _add_to_sanctions_list(self, sanction_entry: Dict):
        """Add entry to sanctions list (test simulation)"""
        async with httpx.AsyncClient() as client:
            await client.post(
                f"{SANCTIONS_API_URL}/test/add_sanction",
                json=sanction_entry,
                headers=self.headers
            )
    
    async def _trigger_sanctions_rescreening(self, entity_id: str):
        """Trigger sanctions re-screening for entity"""
        async with httpx.AsyncClient() as client:
            await client.post(
                f"{SANCTIONS_API_URL}/rescreen/{entity_id}",
                headers=self.headers
            )
    
    async def _create_sanctioned_entity(self) -> Dict:
        """Create a sanctioned entity for testing"""
        entity_data = {
            "name": "Suspicious Bank",
            "entity_type": "financial_institution",
            "country": "IR",
            "swift_code": "SUSBIRTE",
            "metadata": {"test_entity": True, "test_id": self.test_id}
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=entity_data,
                headers=self.headers
            )
            entity = response.json()
            self.test_entities.append(entity["id"])
            return entity
    
    async def _create_normal_entity(self) -> Dict:
        """Create a normal entity for testing"""
        entity_data = {
            "name": f"Normal Bank {self.test_id}",
            "entity_type": "financial_institution",
            "country": "US",
            "metadata": {"test_entity": True, "test_id": self.test_id}
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=entity_data,
                headers=self.headers
            )
            entity = response.json()
            self.test_entities.append(entity["id"])
            return entity


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-s"])