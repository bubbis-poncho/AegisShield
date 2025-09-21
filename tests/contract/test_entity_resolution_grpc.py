import pytest
import asyncio
import grpc
from grpc import aio
from datetime import datetime, timezone
import sys
import os

# Add the shared proto path to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), '../../shared/proto'))

import entity_resolution_pb2 as pb
import entity_resolution_pb2_grpc as pb_grpc


class TestEntityResolutionService:
    """
    gRPC contract tests for entity-resolution service
    These tests MUST FAIL initially (TDD principle)
    They define the expected contract for the entity resolution service
    """
    
    @pytest.fixture
    async def grpc_channel(self):
        """Setup gRPC channel for testing"""
        channel = aio.insecure_channel('localhost:50052')
        yield channel
        await channel.close()
    
    @pytest.fixture
    async def client(self, grpc_channel):
        """Create entity resolution service client"""
        return pb_grpc.EntityResolutionServiceStub(grpc_channel)
    
    @pytest.mark.asyncio
    async def test_resolve_person_valid_input_success(self, client):
        """Test successful person entity resolution"""
        request = pb.ResolvePersonRequest(
            person_data=pb.PersonData(
                names=["John Doe", "Jonathan Doe"],
                birth_date="1985-03-15",
                phone_numbers=["+1-555-0123", "+1-555-0124"],
                email_addresses=["john.doe@email.com", "j.doe@company.com"],
                addresses=[
                    pb.Address(
                        street="123 Main St",
                        city="New York",
                        state="NY",
                        zip_code="10001",
                        country="USA"
                    )
                ]
            ),
            confidence_threshold=0.8
        )
        
        response = await client.ResolvePerson(request)
        
        assert response is not None
        assert response.status == pb.ResolutionStatus.SUCCESS
        assert response.entity_id != ""
        assert response.confidence_score >= 0.8
        assert len(response.matched_attributes) > 0
        
    @pytest.mark.asyncio
    async def test_resolve_person_low_confidence_partial_match(self, client):
        """Test person resolution with low confidence threshold"""
        request = pb.ResolvePersonRequest(
            person_data=pb.PersonData(
                names=["Jane Smith"],
                phone_numbers=["+1-555-9999"]
            ),
            confidence_threshold=0.3
        )
        
        response = await client.ResolvePerson(request)
        
        assert response is not None
        assert response.status in [pb.ResolutionStatus.PARTIAL_MATCH, pb.ResolutionStatus.NO_MATCH]
        if response.status == pb.ResolutionStatus.PARTIAL_MATCH:
            assert 0.3 <= response.confidence_score < 0.8
            assert len(response.candidate_entities) > 0
    
    @pytest.mark.asyncio
    async def test_resolve_organization_valid_input_success(self, client):
        """Test successful organization entity resolution"""
        request = pb.ResolveOrganizationRequest(
            organization_data=pb.OrganizationData(
                legal_names=["Acme Corporation", "Acme Corp"],
                registration_numbers=["12-3456789", "987654321"],
                addresses=[
                    pb.Address(
                        street="456 Business Ave",
                        city="San Francisco",
                        state="CA",
                        zip_code="94105",
                        country="USA"
                    )
                ],
                industry_codes=["541511", "541512"]
            ),
            confidence_threshold=0.7
        )
        
        response = await client.ResolveOrganization(request)
        
        assert response is not None
        assert response.status == pb.ResolutionStatus.SUCCESS
        assert response.entity_id != ""
        assert response.confidence_score >= 0.7
        
    @pytest.mark.asyncio
    async def test_find_similar_entities_returns_candidates(self, client):
        """Test finding similar entities functionality"""
        request = pb.FindSimilarEntitiesRequest(
            entity_id="person_12345",
            entity_type=pb.EntityType.PERSON,
            similarity_threshold=0.6,
            max_results=10
        )
        
        response = await client.FindSimilarEntities(request)
        
        assert response is not None
        assert len(response.similar_entities) <= 10
        for entity in response.similar_entities:
            assert entity.similarity_score >= 0.6
            assert entity.entity_id != ""
            assert entity.entity_type == pb.EntityType.PERSON
    
    @pytest.mark.asyncio
    async def test_link_entities_creates_relationship(self, client):
        """Test linking entities to create relationships"""
        request = pb.LinkEntitiesRequest(
            source_entity_id="person_12345",
            target_entity_id="org_67890",
            relationship_type=pb.RelationshipType.EMPLOYMENT,
            confidence_score=0.9,
            evidence=[
                pb.Evidence(
                    type=pb.EvidenceType.DOCUMENT,
                    description="Employment contract",
                    confidence=0.95
                )
            ]
        )
        
        response = await client.LinkEntities(request)
        
        assert response is not None
        assert response.status == pb.LinkStatus.SUCCESS
        assert response.relationship_id != ""
        assert response.created_at is not None
        
    @pytest.mark.asyncio
    async def test_get_entity_graph_returns_relationships(self, client):
        """Test retrieving entity relationship graph"""
        request = pb.GetEntityGraphRequest(
            entity_id="person_12345",
            max_depth=2,
            relationship_types=[
                pb.RelationshipType.FAMILY,
                pb.RelationshipType.BUSINESS,
                pb.RelationshipType.EMPLOYMENT
            ]
        )
        
        response = await client.GetEntityGraph(request)
        
        assert response is not None
        assert response.center_entity.entity_id == "person_12345"
        assert len(response.nodes) >= 1  # At least the center entity
        assert all(rel.confidence >= 0.0 for rel in response.relationships)
        
    @pytest.mark.asyncio
    async def test_resolve_batch_entities_processes_multiple(self, client):
        """Test batch entity resolution"""
        persons = [
            pb.PersonData(
                names=["Alice Johnson"],
                email_addresses=["alice@example.com"]
            ),
            pb.PersonData(
                names=["Bob Wilson"],
                phone_numbers=["+1-555-1111"]
            )
        ]
        
        request = pb.BatchResolveRequest(
            person_entities=persons,
            confidence_threshold=0.7
        )
        
        response = await client.BatchResolve(request)
        
        assert response is not None
        assert len(response.results) == 2
        assert response.processed_count >= 0
        assert response.success_count >= 0
        assert response.error_count >= 0
        
    @pytest.mark.asyncio
    async def test_health_check_service_available(self, client):
        """Test service health check"""
        request = pb.HealthCheckRequest(
            service="entity-resolution"
        )
        
        response = await client.HealthCheck(request)
        
        assert response is not None
        assert response.status == pb.HealthStatus.SERVING
        assert response.message != ""
        
    @pytest.mark.asyncio
    async def test_get_resolution_metrics_returns_stats(self, client):
        """Test getting resolution performance metrics"""
        request = pb.GetMetricsRequest(
            time_range_hours=24
        )
        
        response = await client.GetResolutionMetrics(request)
        
        assert response is not None
        assert response.total_resolutions >= 0
        assert response.average_confidence >= 0.0
        assert response.processing_time_ms >= 0
        assert len(response.entity_type_counts) >= 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])