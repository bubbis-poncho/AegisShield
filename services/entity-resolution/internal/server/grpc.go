package server

import (
	"context"
	"log/slog"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/entity-resolution/internal/resolver"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/aegisshield/shared/proto"
)

// GRPCServer implements the entity resolution gRPC service
type GRPCServer struct {
	pb.UnimplementedEntityResolutionServer
	resolver *resolver.EntityResolver
	config   config.Config
	logger   *slog.Logger
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(
	resolver *resolver.EntityResolver,
	config config.Config,
	logger *slog.Logger,
) *GRPCServer {
	return &GRPCServer{
		resolver: resolver,
		config:   config,
		logger:   logger,
	}
}

// ResolveEntity resolves a single entity
func (s *GRPCServer) ResolveEntity(ctx context.Context, req *pb.ResolveEntityRequest) (*pb.ResolveEntityResponse, error) {
	s.logger.Info("Received ResolveEntity request",
		"entity_type", req.EntityType,
		"name", req.Name)

	// Validate request
	if req.EntityType == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}

	// Convert protobuf request to internal request
	resolverReq := &resolver.ResolutionRequest{
		EntityType:  req.EntityType,
		Name:        req.Name,
		Identifiers: protoMapToGoMap(req.Identifiers),
		Attributes:  protoMapToGoMap(req.Attributes),
		SourceID:    req.SourceId,
	}

	// Resolve entity
	result, err := s.resolver.ResolveEntity(ctx, resolverReq)
	if err != nil {
		s.logger.Error("Failed to resolve entity", "error", err)
		return nil, status.Error(codes.Internal, "failed to resolve entity")
	}

	// Convert result to protobuf response
	response := &pb.ResolveEntityResponse{
		EntityId:         result.EntityID,
		IsNewEntity:      result.IsNewEntity,
		ConfidenceScore:  result.ConfidenceScore,
		StandardizedData: goMapToProtoMap(result.StandardizedData),
		CreatedLinks:     result.CreatedLinks,
	}

	// Convert matched entities
	for _, match := range result.MatchedEntities {
		pbMatch := &pb.MatchCandidate{
			EntityId:       match.EntityID,
			MatchScore:     match.MatchScore,
			MatchedFields:  match.MatchedFields,
			ConflictFields: match.ConflictFields,
			RecommendMerge: match.RecommendMerge,
		}
		response.MatchedEntities = append(response.MatchedEntities, pbMatch)
	}

	s.logger.Info("Entity resolution completed",
		"entity_id", result.EntityID,
		"is_new_entity", result.IsNewEntity,
		"confidence_score", result.ConfidenceScore)

	return response, nil
}

// ResolveBatch processes multiple entities in batch
func (s *GRPCServer) ResolveBatch(ctx context.Context, req *pb.ResolveBatchRequest) (*pb.ResolveBatchResponse, error) {
	s.logger.Info("Received ResolveBatch request", "count", len(req.Entities))

	if len(req.Entities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one entity is required")
	}

	if len(req.Entities) > s.config.EntityResolution.MaxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, "batch size exceeds maximum of %d", s.config.EntityResolution.MaxBatchSize)
	}

	// Convert protobuf requests to internal requests
	var resolverReqs []*resolver.ResolutionRequest
	for _, entity := range req.Entities {
		resolverReq := &resolver.ResolutionRequest{
			EntityType:  entity.EntityType,
			Name:        entity.Name,
			Identifiers: protoMapToGoMap(entity.Identifiers),
			Attributes:  protoMapToGoMap(entity.Attributes),
			SourceID:    entity.SourceId,
		}
		resolverReqs = append(resolverReqs, resolverReq)
	}

	// Process batch
	job, err := s.resolver.ResolveBatch(ctx, resolverReqs)
	if err != nil {
		s.logger.Error("Failed to process batch", "error", err)
		return nil, status.Error(codes.Internal, "failed to process batch")
	}

	// Convert job to protobuf response
	response := &pb.ResolveBatchResponse{
		JobId:     job.JobID,
		Status:    job.Status,
		StartedAt: timestamppb.New(job.StartedAt),
		Progress:  int32(job.Progress),
		Total:     int32(job.Total),
		Errors:    job.Errors,
	}

	if job.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	// Convert results
	for _, result := range job.Results {
		pbResult := &pb.ResolveEntityResponse{
			EntityId:         result.EntityID,
			IsNewEntity:      result.IsNewEntity,
			ConfidenceScore:  result.ConfidenceScore,
			StandardizedData: goMapToProtoMap(result.StandardizedData),
			CreatedLinks:     result.CreatedLinks,
		}

		for _, match := range result.MatchedEntities {
			pbMatch := &pb.MatchCandidate{
				EntityId:       match.EntityID,
				MatchScore:     match.MatchScore,
				MatchedFields:  match.MatchedFields,
				ConflictFields: match.ConflictFields,
				RecommendMerge: match.RecommendMerge,
			}
			pbResult.MatchedEntities = append(pbResult.MatchedEntities, pbMatch)
		}

		response.Results = append(response.Results, pbResult)
	}

	s.logger.Info("Batch resolution initiated",
		"job_id", job.JobID,
		"total", job.Total)

	return response, nil
}

// GetResolutionJob retrieves the status of a resolution job
func (s *GRPCServer) GetResolutionJob(ctx context.Context, req *pb.GetResolutionJobRequest) (*pb.GetResolutionJobResponse, error) {
	s.logger.Info("Received GetResolutionJob request", "job_id", req.JobId)

	if req.JobId == "" {
		return nil, status.Error(codes.InvalidArgument, "job_id is required")
	}

	// Get job from resolver
	job, err := s.resolver.GetResolutionJob(ctx, req.JobId)
	if err != nil {
		s.logger.Error("Failed to get resolution job", "job_id", req.JobId, "error", err)
		return nil, status.Error(codes.NotFound, "resolution job not found")
	}

	// Convert to protobuf response
	response := &pb.GetResolutionJobResponse{
		JobId:     job.JobID,
		Status:    job.Status,
		StartedAt: timestamppb.New(job.StartedAt),
		Progress:  int32(job.Progress),
		Total:     int32(job.Total),
		Errors:    job.Errors,
	}

	if job.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	return response, nil
}

// FindSimilarEntities finds entities similar to the given entity
func (s *GRPCServer) FindSimilarEntities(ctx context.Context, req *pb.FindSimilarEntitiesRequest) (*pb.FindSimilarEntitiesResponse, error) {
	s.logger.Info("Received FindSimilarEntities request",
		"entity_id", req.EntityId,
		"threshold", req.Threshold)

	if req.EntityId == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_id is required")
	}

	threshold := req.Threshold
	if threshold <= 0 {
		threshold = s.config.EntityResolution.NameSimilarityThreshold
	}

	// Find similar entities
	matches, err := s.resolver.FindSimilarEntities(ctx, req.EntityId, threshold)
	if err != nil {
		s.logger.Error("Failed to find similar entities", "entity_id", req.EntityId, "error", err)
		return nil, status.Error(codes.Internal, "failed to find similar entities")
	}

	// Convert to protobuf response
	response := &pb.FindSimilarEntitiesResponse{}
	for _, match := range matches {
		pbMatch := &pb.MatchCandidate{
			EntityId:       match.EntityID,
			MatchScore:     match.MatchScore,
			MatchedFields:  match.MatchedFields,
			ConflictFields: match.ConflictFields,
			RecommendMerge: match.RecommendMerge,
		}
		response.SimilarEntities = append(response.SimilarEntities, pbMatch)
	}

	s.logger.Info("Found similar entities",
		"entity_id", req.EntityId,
		"count", len(matches))

	return response, nil
}

// CreateEntityLink creates a link between two entities
func (s *GRPCServer) CreateEntityLink(ctx context.Context, req *pb.CreateEntityLinkRequest) (*pb.CreateEntityLinkResponse, error) {
	s.logger.Info("Received CreateEntityLink request",
		"source_id", req.SourceEntityId,
		"target_id", req.TargetEntityId,
		"link_type", req.LinkType)

	// Validate request
	if req.SourceEntityId == "" {
		return nil, status.Error(codes.InvalidArgument, "source_entity_id is required")
	}
	if req.TargetEntityId == "" {
		return nil, status.Error(codes.InvalidArgument, "target_entity_id is required")
	}
	if req.LinkType == "" {
		return nil, status.Error(codes.InvalidArgument, "link_type is required")
	}

	confidence := req.Confidence
	if confidence <= 0 {
		confidence = 1.0
	}

	// Create entity link
	err := s.resolver.CreateEntityLink(
		ctx,
		req.SourceEntityId,
		req.TargetEntityId,
		req.LinkType,
		protoMapToGoMap(req.Properties),
		confidence,
	)

	if err != nil {
		s.logger.Error("Failed to create entity link", "error", err)
		return nil, status.Error(codes.Internal, "failed to create entity link")
	}

	linkID := uuid.New().String()
	response := &pb.CreateEntityLinkResponse{
		LinkId:  linkID,
		Success: true,
	}

	s.logger.Info("Entity link created",
		"link_id", linkID,
		"source_id", req.SourceEntityId,
		"target_id", req.TargetEntityId)

	return response, nil
}

// StreamResolution streams entity resolution results
func (s *GRPCServer) StreamResolution(stream pb.EntityResolution_StreamResolutionServer) error {
	s.logger.Info("Started StreamResolution session")

	for {
		req, err := stream.Recv()
		if err != nil {
			s.logger.Info("StreamResolution session ended", "error", err)
			return err
		}

		// Convert and process request
		resolverReq := &resolver.ResolutionRequest{
			EntityType:  req.EntityType,
			Name:        req.Name,
			Identifiers: protoMapToGoMap(req.Identifiers),
			Attributes:  protoMapToGoMap(req.Attributes),
			SourceID:    req.SourceId,
		}

		result, err := s.resolver.ResolveEntity(stream.Context(), resolverReq)
		if err != nil {
			s.logger.Error("Failed to resolve entity in stream", "error", err)
			continue
		}

		// Convert and send response
		response := &pb.ResolveEntityResponse{
			EntityId:         result.EntityID,
			IsNewEntity:      result.IsNewEntity,
			ConfidenceScore:  result.ConfidenceScore,
			StandardizedData: goMapToProtoMap(result.StandardizedData),
			CreatedLinks:     result.CreatedLinks,
		}

		for _, match := range result.MatchedEntities {
			pbMatch := &pb.MatchCandidate{
				EntityId:       match.EntityID,
				MatchScore:     match.MatchScore,
				MatchedFields:  match.MatchedFields,
				ConflictFields: match.ConflictFields,
				RecommendMerge: match.RecommendMerge,
			}
			response.MatchedEntities = append(response.MatchedEntities, pbMatch)
		}

		if err := stream.Send(response); err != nil {
			s.logger.Error("Failed to send stream response", "error", err)
			return err
		}
	}
}

// HealthCheck performs a health check
func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:  "SERVING",
		Message: "Entity Resolution Service is healthy",
	}, nil
}

// Helper functions

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