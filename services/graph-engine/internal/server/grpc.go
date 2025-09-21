package server

import (
	"context"
	"log/slog"

	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/aegisshield/shared/proto"
)

// GRPCServer implements the graph engine gRPC service
type GRPCServer struct {
	pb.UnimplementedGraphEngineServer
	engine *engine.GraphEngine
	config config.Config
	logger *slog.Logger
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(
	engine *engine.GraphEngine,
	config config.Config,
	logger *slog.Logger,
) *GRPCServer {
	return &GRPCServer{
		engine: engine,
		config: config,
		logger: logger,
	}
}

// AnalyzeSubGraph performs subgraph analysis
func (s *GRPCServer) AnalyzeSubGraph(ctx context.Context, req *pb.AnalyzeSubGraphRequest) (*pb.AnalyzeSubGraphResponse, error) {
	s.logger.Info("Received AnalyzeSubGraph request",
		"analysis_type", req.AnalysisType,
		"entity_count", len(req.EntityIds))

	// Validate request
	if len(req.EntityIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "entity_ids is required")
	}

	if req.AnalysisType == "" {
		return nil, status.Error(codes.InvalidArgument, "analysis_type is required")
	}

	// Convert request to internal format
	analysisReq := &engine.AnalysisRequest{
		Type:      req.AnalysisType,
		EntityIDs: req.EntityIds,
		Options: engine.AnalysisOptions{
			MaxDepth:           int(req.Options.MaxDepth),
			MaxPathLength:      int(req.Options.MaxPathLength),
			MinConfidence:      req.Options.MinConfidence,
			IncludePatterns:    req.Options.IncludePatterns,
			IncludeMetrics:     req.Options.IncludeMetrics,
			IncludeCommunities: req.Options.IncludeCommunities,
		},
		RequestedBy: req.RequestedBy,
	}

	// Convert parameters
	if req.Parameters != nil {
		analysisReq.Parameters = protoMapToGoMap(req.Parameters)
	}

	// Perform analysis
	result, err := s.engine.AnalyzeSubGraph(ctx, analysisReq)
	if err != nil {
		s.logger.Error("Failed to analyze subgraph", "error", err)
		return nil, status.Error(codes.Internal, "failed to analyze subgraph")
	}

	// Convert result to protobuf response
	response := &pb.AnalyzeSubGraphResponse{
		JobId:     result.JobID,
		Status:    result.Status,
		StartedAt: timestamppb.New(result.StartedAt),
		Metadata:  goMapToProtoMap(result.Metadata),
	}

	if result.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*result.CompletedAt)
	}

	// Convert subgraph
	if result.SubGraph != nil {
		response.Subgraph = &pb.SubGraph{
			Entities:      convertEntitiesToProto(result.SubGraph.Entities),
			Relationships: convertRelationshipsToProto(result.SubGraph.Relationships),
			Metadata:      goMapToProtoMap(result.SubGraph.Metadata),
		}
	}

	// Convert paths
	for _, path := range result.Paths {
		pbPath := &pb.Path{
			StartEntity:   convertEntityToProto(path.StartEntity),
			EndEntity:     convertEntityToProto(path.EndEntity),
			Entities:      convertEntitiesToProto(path.Entities),
			Relationships: convertRelationshipsToProto(path.Relationships),
			Length:        int32(path.Length),
			Cost:          path.Cost,
		}
		response.Paths = append(response.Paths, pbPath)
	}

	// Convert patterns
	for _, pattern := range result.Patterns {
		pbPattern := &pb.PatternMatch{
			PatternType:   pattern.PatternType,
			Entities:      convertEntitiesToProto(pattern.Entities),
			Relationships: convertRelationshipsToProto(pattern.Relationships),
			Confidence:    pattern.Confidence,
			Metadata:      goMapToProtoMap(pattern.Metadata),
		}
		response.Patterns = append(response.Patterns, pbPattern)
	}

	// Convert communities
	for _, community := range result.Communities {
		pbCommunity := &pb.Community{
			Id:         community.ID,
			EntityIds:  community.Entities,
			Size:       int32(community.Size),
			Density:    community.Density,
			Modularity: community.Modularity,
		}
		response.Communities = append(response.Communities, pbCommunity)
	}

	// Convert insights
	for _, insight := range result.Insights {
		pbInsight := &pb.AnalysisInsight{
			Type:        insight.Type,
			Title:       insight.Title,
			Description: insight.Description,
			Confidence:  insight.Confidence,
			EntityIds:   insight.EntityIDs,
			Evidence:    goMapToProtoMap(insight.Evidence),
			Severity:    insight.Severity,
		}
		response.Insights = append(response.Insights, pbInsight)
	}

	s.logger.Info("Subgraph analysis completed",
		"job_id", result.JobID,
		"entity_count", len(response.Subgraph.Entities),
		"insight_count", len(response.Insights))

	return response, nil
}

// FindPaths finds paths between entities
func (s *GRPCServer) FindPaths(ctx context.Context, req *pb.FindPathsRequest) (*pb.FindPathsResponse, error) {
	s.logger.Info("Received FindPaths request",
		"source_count", len(req.SourceIds),
		"target_count", len(req.TargetIds))

	// Validate request
	if len(req.SourceIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "source_ids is required")
	}
	if len(req.TargetIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target_ids is required")
	}

	// Convert request
	pathReq := &engine.PathRequest{
		SourceIDs:   req.SourceIds,
		TargetIDs:   req.TargetIds,
		MaxLength:   int(req.MaxLength),
		Algorithm:   req.Algorithm,
		WeightField: req.WeightField,
	}

	// Find paths
	paths, err := s.engine.FindPaths(ctx, pathReq)
	if err != nil {
		s.logger.Error("Failed to find paths", "error", err)
		return nil, status.Error(codes.Internal, "failed to find paths")
	}

	// Convert response
	response := &pb.FindPathsResponse{}
	for _, path := range paths {
		pbPath := &pb.Path{
			StartEntity:   convertEntityToProto(path.StartEntity),
			EndEntity:     convertEntityToProto(path.EndEntity),
			Entities:      convertEntitiesToProto(path.Entities),
			Relationships: convertRelationshipsToProto(path.Relationships),
			Length:        int32(path.Length),
			Cost:          path.Cost,
		}
		response.Paths = append(response.Paths, pbPath)
	}

	s.logger.Info("Paths found", "count", len(paths))
	return response, nil
}

// CreateInvestigation creates a new investigation
func (s *GRPCServer) CreateInvestigation(ctx context.Context, req *pb.CreateInvestigationRequest) (*pb.CreateInvestigationResponse, error) {
	s.logger.Info("Received CreateInvestigation request", "name", req.Name)

	// Validate request
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if len(req.EntityIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "entity_ids is required")
	}

	// Convert request
	investigationReq := &engine.InvestigationRequest{
		Name:        req.Name,
		Description: req.Description,
		EntityIDs:   req.EntityIds,
		Priority:    req.Priority,
		CreatedBy:   req.CreatedBy,
		AssignedTo:  req.AssignedTo,
	}

	if req.Parameters != nil {
		investigationReq.Parameters = protoMapToGoMap(req.Parameters)
	}

	// Create investigation
	investigation, err := s.engine.CreateInvestigation(ctx, investigationReq)
	if err != nil {
		s.logger.Error("Failed to create investigation", "error", err)
		return nil, status.Error(codes.Internal, "failed to create investigation")
	}

	// Convert response
	response := &pb.CreateInvestigationResponse{
		InvestigationId: investigation.ID,
		Name:            investigation.Name,
		Status:          investigation.Status,
		CreatedAt:       timestamppb.New(investigation.CreatedAt),
	}

	s.logger.Info("Investigation created", "investigation_id", investigation.ID)
	return response, nil
}

// GetAnalysisJob retrieves an analysis job
func (s *GRPCServer) GetAnalysisJob(ctx context.Context, req *pb.GetAnalysisJobRequest) (*pb.GetAnalysisJobResponse, error) {
	s.logger.Info("Received GetAnalysisJob request", "job_id", req.JobId)

	if req.JobId == "" {
		return nil, status.Error(codes.InvalidArgument, "job_id is required")
	}

	// Get job
	job, err := s.engine.GetAnalysisJob(ctx, req.JobId)
	if err != nil {
		s.logger.Error("Failed to get analysis job", "job_id", req.JobId, "error", err)
		return nil, status.Error(codes.NotFound, "analysis job not found")
	}

	// Convert response
	response := &pb.GetAnalysisJobResponse{
		JobId:     job.ID,
		Type:      job.Type,
		Status:    job.Status,
		Progress:  int32(job.Progress),
		Total:     int32(job.Total),
		StartedAt: timestamppb.New(job.StartedAt),
		Error:     job.Error,
	}

	if job.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	if job.Parameters != nil {
		response.Parameters = goMapToProtoMap(job.Parameters)
	}

	if job.Results != nil {
		response.Results = goMapToProtoMap(job.Results)
	}

	return response, nil
}

// GetInvestigation retrieves an investigation
func (s *GRPCServer) GetInvestigation(ctx context.Context, req *pb.GetInvestigationRequest) (*pb.GetInvestigationResponse, error) {
	s.logger.Info("Received GetInvestigation request", "investigation_id", req.InvestigationId)

	if req.InvestigationId == "" {
		return nil, status.Error(codes.InvalidArgument, "investigation_id is required")
	}

	// Get investigation
	investigation, err := s.engine.GetInvestigation(ctx, req.InvestigationId)
	if err != nil {
		s.logger.Error("Failed to get investigation", "investigation_id", req.InvestigationId, "error", err)
		return nil, status.Error(codes.NotFound, "investigation not found")
	}

	// Convert response
	response := &pb.GetInvestigationResponse{
		InvestigationId: investigation.ID,
		Name:            investigation.Name,
		Description:     investigation.Description,
		Status:          investigation.Status,
		Priority:        investigation.Priority,
		EntityIds:       investigation.Entities,
		CreatedAt:       timestamppb.New(investigation.CreatedAt),
		UpdatedAt:       timestamppb.New(investigation.UpdatedAt),
		CreatedBy:       investigation.CreatedBy,
		AssignedTo:      investigation.AssignedTo,
	}

	if investigation.Metadata != nil {
		response.Metadata = goMapToProtoMap(investigation.Metadata)
	}

	return response, nil
}

// GetEntityNeighborhood gets entity neighborhood
func (s *GRPCServer) GetEntityNeighborhood(ctx context.Context, req *pb.GetEntityNeighborhoodRequest) (*pb.GetEntityNeighborhoodResponse, error) {
	s.logger.Info("Received GetEntityNeighborhood request", "entity_id", req.EntityId)

	if req.EntityId == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_id is required")
	}

	// Get neighborhood
	subGraph, err := s.engine.GetEntityNeighborhood(ctx, req.EntityId, req.RelationshipTypes)
	if err != nil {
		s.logger.Error("Failed to get entity neighborhood", "entity_id", req.EntityId, "error", err)
		return nil, status.Error(codes.Internal, "failed to get entity neighborhood")
	}

	// Convert response
	response := &pb.GetEntityNeighborhoodResponse{
		EntityId: req.EntityId,
		Subgraph: &pb.SubGraph{
			Entities:      convertEntitiesToProto(subGraph.Entities),
			Relationships: convertRelationshipsToProto(subGraph.Relationships),
			Metadata:      goMapToProtoMap(subGraph.Metadata),
		},
	}

	s.logger.Info("Entity neighborhood retrieved",
		"entity_id", req.EntityId,
		"neighbor_count", len(response.Subgraph.Entities)-1)

	return response, nil
}

// CalculateNetworkMetrics calculates network metrics
func (s *GRPCServer) CalculateNetworkMetrics(ctx context.Context, req *pb.CalculateNetworkMetricsRequest) (*pb.CalculateNetworkMetricsResponse, error) {
	s.logger.Info("Received CalculateNetworkMetrics request", "entity_count", len(req.EntityIds))

	if len(req.EntityIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "entity_ids is required")
	}

	// Calculate metrics
	metrics, err := s.engine.CalculateNetworkMetrics(ctx, req.EntityIds)
	if err != nil {
		s.logger.Error("Failed to calculate network metrics", "error", err)
		return nil, status.Error(codes.Internal, "failed to calculate network metrics")
	}

	// Convert response
	response := &pb.CalculateNetworkMetricsResponse{}
	for _, metric := range metrics {
		pbMetric := &pb.NetworkMetrics{
			EntityId:              metric.EntityID,
			DegreeCentrality:      metric.DegreeCentrality,
			BetweennessCentrality: metric.BetweennessCentrality,
			ClosenessCentrality:   metric.ClosenessCentrality,
			EigenvectorCentrality: metric.EigenvectorCentrality,
			PageRank:              metric.PageRank,
			ClusteringCoefficient: metric.ClusteringCoeff,
			CommunityId:           metric.CommunityID,
			CalculatedAt:          timestamppb.New(metric.CalculatedAt),
		}

		if metric.Metadata != nil {
			pbMetric.Metadata = goMapToProtoMap(metric.Metadata)
		}

		response.Metrics = append(response.Metrics, pbMetric)
	}

	s.logger.Info("Network metrics calculated", "metric_count", len(metrics))
	return response, nil
}

// HealthCheck performs a health check
func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:  "SERVING",
		Message: "Graph Engine Service is healthy",
	}, nil
}

// Helper functions

func convertEntityToProto(entity *engine.Entity) *pb.Entity {
	if entity == nil {
		return nil
	}

	return &pb.Entity{
		Id:         entity.ID,
		Type:       entity.Type,
		Properties: goMapToProtoMap(entity.Properties),
	}
}

func convertEntitiesToProto(entities []*engine.Entity) []*pb.Entity {
	var pbEntities []*pb.Entity
	for _, entity := range entities {
		pbEntities = append(pbEntities, convertEntityToProto(entity))
	}
	return pbEntities
}

func convertRelationshipToProto(rel *engine.Relationship) *pb.Relationship {
	if rel == nil {
		return nil
	}

	return &pb.Relationship{
		Id:         rel.ID,
		Type:       rel.Type,
		SourceId:   rel.SourceID,
		TargetId:   rel.TargetID,
		Properties: goMapToProtoMap(rel.Properties),
	}
}

func convertRelationshipsToProto(relationships []*engine.Relationship) []*pb.Relationship {
	var pbRelationships []*pb.Relationship
	for _, rel := range relationships {
		pbRelationships = append(pbRelationships, convertRelationshipToProto(rel))
	}
	return pbRelationships
}

func protoMapToGoMap(protoMap map[string]*pb.Value) map[string]interface{} {
	if protoMap == nil {
		return nil
	}

	goMap := make(map[string]interface{})
	for key, value := range protoMap {
		goMap[key] = protoValueToInterface(value)
	}
	return goMap
}

func protoValueToInterface(value *pb.Value) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.Kind.(type) {
	case *pb.Value_StringValue:
		return v.StringValue
	case *pb.Value_NumberValue:
		return v.NumberValue
	case *pb.Value_BoolValue:
		return v.BoolValue
	case *pb.Value_ListValue:
		var list []interface{}
		for _, item := range v.ListValue.Values {
			list = append(list, protoValueToInterface(item))
		}
		return list
	case *pb.Value_StructValue:
		return protoMapToGoMap(v.StructValue.Fields)
	default:
		return nil
	}
}

func goMapToProtoMap(goMap map[string]interface{}) map[string]*pb.Value {
	if goMap == nil {
		return nil
	}

	protoMap := make(map[string]*pb.Value)
	for key, value := range goMap {
		protoMap[key] = interfaceToProtoValue(value)
	}
	return protoMap
}

func interfaceToProtoValue(value interface{}) *pb.Value {
	if value == nil {
		return &pb.Value{Kind: &pb.Value_NullValue{}}
	}

	switch v := value.(type) {
	case string:
		return &pb.Value{Kind: &pb.Value_StringValue{StringValue: v}}
	case float64:
		return &pb.Value{Kind: &pb.Value_NumberValue{NumberValue: v}}
	case float32:
		return &pb.Value{Kind: &pb.Value_NumberValue{NumberValue: float64(v)}}
	case int:
		return &pb.Value{Kind: &pb.Value_NumberValue{NumberValue: float64(v)}}
	case int32:
		return &pb.Value{Kind: &pb.Value_NumberValue{NumberValue: float64(v)}}
	case int64:
		return &pb.Value{Kind: &pb.Value_NumberValue{NumberValue: float64(v)}}
	case bool:
		return &pb.Value{Kind: &pb.Value_BoolValue{BoolValue: v}}
	case []interface{}:
		var protoList []*pb.Value
		for _, item := range v {
			protoList = append(protoList, interfaceToProtoValue(item))
		}
		return &pb.Value{Kind: &pb.Value_ListValue{ListValue: &pb.ListValue{Values: protoList}}}
	case map[string]interface{}:
		return &pb.Value{Kind: &pb.Value_StructValue{StructValue: &pb.Struct{Fields: goMapToProtoMap(v)}}}
	default:
		return &pb.Value{Kind: &pb.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
}