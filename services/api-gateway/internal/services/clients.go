package services

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"aegisshield/services/api-gateway/internal/config"
	dataIngestionPb "aegisshield/shared/proto"
	entityResolutionPb "aegisshield/shared/proto"
	alertingPb "aegisshield/shared/proto"
	graphPb "aegisshield/shared/proto"
)

type ServiceClients struct {
	DataIngestion   dataIngestionPb.DataIngestionServiceClient
	EntityResolution entityResolutionPb.EntityResolutionServiceClient
	AlertingEngine   alertingPb.AlertingEngineServiceClient
	GraphEngine      graphPb.GraphEngineServiceClient
	
	// gRPC connections
	dataIngestionConn   *grpc.ClientConn
	entityResolutionConn *grpc.ClientConn
	alertingEngineConn   *grpc.ClientConn
	graphEngineConn      *grpc.ClientConn
}

func NewServiceClients(cfg *config.Config) (*ServiceClients, error) {
	clients := &ServiceClients{}

	// Data Ingestion Service
	dataIngestionConn, err := grpc.Dial(
		cfg.Services.DataIngestionURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to data ingestion service: %w", err)
	}
	clients.dataIngestionConn = dataIngestionConn
	clients.DataIngestion = dataIngestionPb.NewDataIngestionServiceClient(dataIngestionConn)

	// Entity Resolution Service
	entityResolutionConn, err := grpc.Dial(
		cfg.Services.EntityResolutionURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to entity resolution service: %w", err)
	}
	clients.entityResolutionConn = entityResolutionConn
	clients.EntityResolution = entityResolutionPb.NewEntityResolutionServiceClient(entityResolutionConn)

	// Alerting Engine Service
	alertingEngineConn, err := grpc.Dial(
		cfg.Services.AlertingEngineURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to alerting engine service: %w", err)
	}
	clients.alertingEngineConn = alertingEngineConn
	clients.AlertingEngine = alertingPb.NewAlertingEngineServiceClient(alertingEngineConn)

	// Graph Engine Service
	graphEngineConn, err := grpc.Dial(
		cfg.Services.GraphEngineURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to graph engine service: %w", err)
	}
	clients.graphEngineConn = graphEngineConn
	clients.GraphEngine = graphPb.NewGraphEngineServiceClient(graphEngineConn)

	return clients, nil
}

func (s *ServiceClients) Close() {
	if s.dataIngestionConn != nil {
		s.dataIngestionConn.Close()
	}
	if s.entityResolutionConn != nil {
		s.entityResolutionConn.Close()
	}
	if s.alertingEngineConn != nil {
		s.alertingEngineConn.Close()
	}
	if s.graphEngineConn != nil {
		s.graphEngineConn.Close()
	}
}

func (s *ServiceClients) HealthCheck(ctx context.Context) error {
	// Check each service health
	connections := []struct {
		name string
		conn *grpc.ClientConn
	}{
		{"data-ingestion", s.dataIngestionConn},
		{"entity-resolution", s.entityResolutionConn},
		{"alerting-engine", s.alertingEngineConn},
		{"graph-engine", s.graphEngineConn},
	}

	for _, service := range connections {
		if service.conn == nil {
			return fmt.Errorf("connection to %s service is nil", service.name)
		}

		healthClient := grpc_health_v1.NewHealthClient(service.conn)
		
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := healthClient.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{
			Service: service.name,
		})
		cancel()

		if err != nil {
			return fmt.Errorf("health check failed for %s service: %w", service.name, err)
		}
	}

	return nil
}