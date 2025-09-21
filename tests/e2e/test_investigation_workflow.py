#!/usr/bin/env python3
"""
End-to-end test for suspicious transaction workflow
Tests complete investigation workflow from transaction ingestion to case resolution
"""

import pytest
import asyncio
import httpx
import json
from datetime import datetime, timedelta
from typing import Dict, List, Any
import os
import uuid

# Test configuration
API_BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8080")
DATA_INGESTION_URL = os.getenv("DATA_INGESTION_URL", "http://localhost:8060")
TEST_TIMEOUT = 60  # seconds

class TestInvestigationWorkflow:
    """End-to-end test suite for suspicious transaction investigation workflow"""
    
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
        
        yield
        
        # Cleanup test data
        await self._cleanup_test_data()
    
    async def _get_auth_token(self) -> str:
        """Get authentication token for API calls"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/auth/login",
                json={
                    "username": "test_investigator",
                    "password": "test_password"
                }
            )
            if response.status_code == 200:
                return response.json()["token"]
            else:
                # For testing, return a mock token
                return "test-token-12345"
    
    async def _cleanup_test_data(self):
        """Clean up test data created during the test"""
        try:
            async with httpx.AsyncClient() as client:
                # Clean up investigations
                for investigation_id in self.test_investigations:
                    await client.delete(
                        f"{API_BASE_URL}/graphql",
                        json={
                            "query": """
                                mutation DeleteInvestigation($id: ID!) {
                                    deleteInvestigation(id: $id) {
                                        success
                                    }
                                }
                            """,
                            "variables": {"id": investigation_id}
                        },
                        headers=self.headers
                    )
                
                # Clean up test entities and transactions
                for entity_id in self.test_entities:
                    await client.delete(
                        f"{DATA_INGESTION_URL}/entities/{entity_id}",
                        headers=self.headers
                    )
        except Exception as e:
            print(f"Cleanup warning: {e}")
    
    @pytest.mark.asyncio
    async def test_complete_investigation_workflow(self):
        """
        Test complete suspicious transaction investigation workflow:
        1. Ingest suspicious transaction
        2. Verify alert generation
        3. Create investigation
        4. Add evidence and analysis
        5. Update investigation status
        6. Generate compliance report
        """
        
        # Step 1: Create test entities
        sender_entity = await self._create_test_entity("sender")
        receiver_entity = await self._create_test_entity("receiver")
        
        # Step 2: Ingest suspicious transaction
        transaction_data = {
            "transaction_id": f"TXN_{self.test_id}",
            "sender_id": sender_entity["id"],
            "receiver_id": receiver_entity["id"],
            "amount": 250000.00,  # Large amount to trigger alert
            "currency": "USD",
            "timestamp": datetime.utcnow().isoformat(),
            "transaction_type": "wire_transfer",
            "source_system": "test_system",
            "metadata": {
                "test_transaction": True,
                "test_id": self.test_id,
                "country_sender": "US",
                "country_receiver": "KY",  # High-risk jurisdiction
                "purpose": "Investment"
            }
        }
        
        transaction_response = await self._ingest_transaction(transaction_data)
        assert transaction_response["status"] == "success"
        self.test_transactions.append(transaction_response["transaction_id"])
        
        # Step 3: Wait for alert generation and verify
        alert = await self._wait_for_alert(transaction_data["transaction_id"])
        assert alert is not None
        assert alert["risk_score"] > 0.7  # Should be high-risk
        assert "high_amount" in alert["alert_types"] or "cross_border" in alert["alert_types"]
        self.test_alerts.append(alert["id"])
        
        # Step 4: Create investigation from alert
        investigation_data = {
            "title": f"Investigation for Transaction {transaction_data['transaction_id']}",
            "description": "E2E test investigation for suspicious cross-border transaction",
            "alert_ids": [alert["id"]],
            "priority": "high",
            "assigned_to": "test_investigator",
            "investigation_type": "money_laundering"
        }
        
        investigation = await self._create_investigation(investigation_data)
        assert investigation["status"] == "open"
        assert investigation["priority"] == "high"
        self.test_investigations.append(investigation["id"])
        
        # Step 5: Add evidence to investigation
        evidence_data = {
            "investigation_id": investigation["id"],
            "evidence_type": "transaction_analysis",
            "description": "Analysis shows pattern consistent with layering",
            "source": "automated_analysis",
            "metadata": {
                "pattern_type": "layering",
                "confidence": 0.85,
                "supporting_transactions": [transaction_data["transaction_id"]]
            }
        }
        
        evidence = await self._add_evidence(evidence_data)
        assert evidence["status"] == "accepted"
        
        # Step 6: Update investigation with analysis
        analysis_update = {
            "investigation_id": investigation["id"],
            "findings": [
                {
                    "type": "suspicious_pattern",
                    "description": "Large cross-border transfer to high-risk jurisdiction",
                    "severity": "high",
                    "supporting_evidence": [evidence["id"]]
                }
            ],
            "risk_assessment": {
                "overall_risk": "high",
                "money_laundering_risk": 0.9,
                "terrorist_financing_risk": 0.3,
                "sanctions_risk": 0.2
            }
        }
        
        updated_investigation = await self._update_investigation_analysis(analysis_update)
        assert updated_investigation["risk_assessment"]["overall_risk"] == "high"
        
        # Step 7: Progress investigation status
        status_updates = ["under_review", "escalated", "completed"]
        for status in status_updates:
            updated_investigation = await self._update_investigation_status(
                investigation["id"], 
                status
            )
            assert updated_investigation["status"] == status
        
        # Step 8: Generate compliance report
        report_request = {
            "investigation_id": investigation["id"],
            "report_type": "suspicious_activity_report",
            "format": "json",
            "include_evidence": True,
            "include_analysis": True
        }
        
        report = await self._generate_compliance_report(report_request)
        assert report["report_type"] == "suspicious_activity_report"
        assert len(report["suspicious_activities"]) > 0
        assert report["investigation_summary"]["status"] == "completed"
        
        # Step 9: Verify audit trail
        audit_trail = await self._get_audit_trail(investigation["id"])
        assert len(audit_trail) >= 4  # Creation, evidence, analysis, status updates
        
        # Verify all critical actions are logged
        action_types = [entry["action"] for entry in audit_trail]
        expected_actions = ["create_investigation", "add_evidence", "update_analysis", "update_status"]
        for action in expected_actions:
            assert action in action_types
    
    @pytest.mark.asyncio
    async def test_sanctions_screening_workflow(self):
        """Test sanctions screening integration in investigation workflow"""
        
        # Create entity with sanctions match
        sanctioned_entity = await self._create_test_entity("sanctioned", {
            "name": "Test Sanctioned Entity",
            "entity_type": "individual",
            "metadata": {
                "known_sanctions_match": True,
                "sanctions_lists": ["OFAC_SDN"]
            }
        })
        
        normal_entity = await self._create_test_entity("normal")
        
        # Create transaction involving sanctioned entity
        transaction_data = {
            "transaction_id": f"SANC_{self.test_id}",
            "sender_id": normal_entity["id"],
            "receiver_id": sanctioned_entity["id"],
            "amount": 50000.00,
            "currency": "USD",
            "timestamp": datetime.utcnow().isoformat(),
            "transaction_type": "wire_transfer",
            "source_system": "test_system"
        }
        
        transaction_response = await self._ingest_transaction(transaction_data)
        self.test_transactions.append(transaction_response["transaction_id"])
        
        # Verify sanctions alert generation
        alert = await self._wait_for_alert(transaction_data["transaction_id"], timeout=30)
        assert alert is not None
        assert alert["risk_score"] > 0.9  # Sanctions should be highest risk
        assert "sanctions_match" in alert["alert_types"]
        
        # Verify investigation auto-creation for sanctions
        investigations = await self._get_investigations_for_alert(alert["id"])
        assert len(investigations) > 0
        sanctions_investigation = investigations[0]
        assert sanctions_investigation["priority"] == "critical"
        assert sanctions_investigation["investigation_type"] == "sanctions_violation"
        
        self.test_investigations.append(sanctions_investigation["id"])
    
    @pytest.mark.asyncio
    async def test_network_analysis_workflow(self):
        """Test network analysis and relationship discovery in investigations"""
        
        # Create network of connected entities
        entities = []
        for i in range(5):
            entity = await self._create_test_entity(f"network_{i}")
            entities.append(entity)
        
        # Create circular transaction pattern
        transactions = []
        for i in range(len(entities)):
            sender = entities[i]
            receiver = entities[(i + 1) % len(entities)]
            
            transaction_data = {
                "transaction_id": f"NET_{self.test_id}_{i}",
                "sender_id": sender["id"],
                "receiver_id": receiver["id"],
                "amount": 100000.00,
                "currency": "USD",
                "timestamp": (datetime.utcnow() + timedelta(minutes=i)).isoformat(),
                "transaction_type": "wire_transfer",
                "source_system": "test_system"
            }
            
            response = await self._ingest_transaction(transaction_data)
            transactions.append(response["transaction_id"])
            self.test_transactions.append(response["transaction_id"])
        
        # Wait for pattern detection
        await asyncio.sleep(10)  # Allow time for pattern analysis
        
        # Verify network alert generation
        network_alerts = await self._get_alerts_by_pattern("circular_transactions")
        assert len(network_alerts) > 0
        
        network_alert = network_alerts[0]
        assert network_alert["risk_score"] > 0.8
        
        # Create investigation for network analysis
        investigation_data = {
            "title": f"Network Analysis Investigation {self.test_id}",
            "description": "Circular transaction pattern investigation",
            "alert_ids": [network_alert["id"]],
            "priority": "high",
            "investigation_type": "network_analysis"
        }
        
        investigation = await self._create_investigation(investigation_data)
        self.test_investigations.append(investigation["id"])
        
        # Verify network analysis results
        network_analysis = await self._get_network_analysis(investigation["id"])
        assert network_analysis is not None
        assert len(network_analysis["entities"]) == 5
        assert network_analysis["pattern_type"] == "circular"
        assert network_analysis["risk_score"] > 0.8
    
    # Helper methods for API interactions
    
    async def _create_test_entity(self, entity_type: str, custom_data: Dict = None) -> Dict:
        """Create a test entity"""
        entity_data = {
            "name": f"Test {entity_type.title()} Entity {self.test_id}",
            "entity_type": "individual" if entity_type != "organization" else "organization",
            "country": "US",
            "metadata": {
                "test_entity": True,
                "test_id": self.test_id,
                **(custom_data or {})
            }
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/entities",
                json=entity_data,
                headers=self.headers
            )
            assert response.status_code == 201
            entity = response.json()
            self.test_entities.append(entity["id"])
            return entity
    
    async def _ingest_transaction(self, transaction_data: Dict) -> Dict:
        """Ingest a transaction"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{DATA_INGESTION_URL}/transactions",
                json=transaction_data,
                headers=self.headers
            )
            assert response.status_code == 201
            return response.json()
    
    async def _wait_for_alert(self, transaction_id: str, timeout: int = 30) -> Dict:
        """Wait for alert generation"""
        start_time = asyncio.get_event_loop().time()
        
        while asyncio.get_event_loop().time() - start_time < timeout:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{API_BASE_URL}/graphql",
                    params={
                        "query": """
                            query GetAlerts($transactionId: String!) {
                                alerts(transactionId: $transactionId) {
                                    id
                                    risk_score
                                    alert_types
                                    status
                                }
                            }
                        """,
                        "variables": json.dumps({"transactionId": transaction_id})
                    },
                    headers=self.headers
                )
                
                if response.status_code == 200:
                    data = response.json()
                    alerts = data.get("data", {}).get("alerts", [])
                    if alerts:
                        return alerts[0]
            
            await asyncio.sleep(2)
        
        return None
    
    async def _create_investigation(self, investigation_data: Dict) -> Dict:
        """Create an investigation"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/graphql",
                json={
                    "query": """
                        mutation CreateInvestigation($input: CreateInvestigationInput!) {
                            createInvestigation(input: $input) {
                                id
                                title
                                status
                                priority
                                investigation_type
                            }
                        }
                    """,
                    "variables": {"input": investigation_data}
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["createInvestigation"]
    
    async def _add_evidence(self, evidence_data: Dict) -> Dict:
        """Add evidence to investigation"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/graphql",
                json={
                    "query": """
                        mutation AddEvidence($input: AddEvidenceInput!) {
                            addEvidence(input: $input) {
                                id
                                status
                                evidence_type
                            }
                        }
                    """,
                    "variables": {"input": evidence_data}
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["addEvidence"]
    
    async def _update_investigation_analysis(self, analysis_data: Dict) -> Dict:
        """Update investigation analysis"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/graphql",
                json={
                    "query": """
                        mutation UpdateInvestigationAnalysis($input: UpdateAnalysisInput!) {
                            updateInvestigationAnalysis(input: $input) {
                                id
                                risk_assessment
                                findings
                            }
                        }
                    """,
                    "variables": {"input": analysis_data}
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["updateInvestigationAnalysis"]
    
    async def _update_investigation_status(self, investigation_id: str, status: str) -> Dict:
        """Update investigation status"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/graphql",
                json={
                    "query": """
                        mutation UpdateInvestigationStatus($id: ID!, $status: String!) {
                            updateInvestigationStatus(id: $id, status: $status) {
                                id
                                status
                                updated_at
                            }
                        }
                    """,
                    "variables": {"id": investigation_id, "status": status}
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["updateInvestigationStatus"]
    
    async def _generate_compliance_report(self, report_request: Dict) -> Dict:
        """Generate compliance report"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{API_BASE_URL}/reports/generate",
                json=report_request,
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()
    
    async def _get_audit_trail(self, investigation_id: str) -> List[Dict]:
        """Get audit trail for investigation"""
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{API_BASE_URL}/graphql",
                params={
                    "query": """
                        query GetAuditTrail($investigationId: ID!) {
                            auditTrail(investigationId: $investigationId) {
                                action
                                timestamp
                                user
                                details
                            }
                        }
                    """,
                    "variables": json.dumps({"investigationId": investigation_id})
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["auditTrail"]
    
    async def _get_alerts_by_pattern(self, pattern_type: str) -> List[Dict]:
        """Get alerts by pattern type"""
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{API_BASE_URL}/graphql",
                params={
                    "query": """
                        query GetAlertsByPattern($patternType: String!) {
                            alerts(patternType: $patternType) {
                                id
                                risk_score
                                alert_types
                                pattern_details
                            }
                        }
                    """,
                    "variables": json.dumps({"patternType": pattern_type})
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["alerts"]
    
    async def _get_investigations_for_alert(self, alert_id: str) -> List[Dict]:
        """Get investigations for alert"""
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{API_BASE_URL}/graphql",
                params={
                    "query": """
                        query GetInvestigationsForAlert($alertId: ID!) {
                            investigations(alertId: $alertId) {
                                id
                                priority
                                investigation_type
                                status
                            }
                        }
                    """,
                    "variables": json.dumps({"alertId": alert_id})
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["investigations"]
    
    async def _get_network_analysis(self, investigation_id: str) -> Dict:
        """Get network analysis for investigation"""
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{API_BASE_URL}/graphql",
                params={
                    "query": """
                        query GetNetworkAnalysis($investigationId: ID!) {
                            networkAnalysis(investigationId: $investigationId) {
                                entities
                                relationships
                                pattern_type
                                risk_score
                                insights
                            }
                        }
                    """,
                    "variables": json.dumps({"investigationId": investigation_id})
                },
                headers=self.headers
            )
            assert response.status_code == 200
            return response.json()["data"]["networkAnalysis"]


if __name__ == "__main__":
    # Run tests with pytest
    pytest.main([__file__, "-v", "-s"])