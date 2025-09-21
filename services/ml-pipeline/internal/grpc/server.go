package grpc

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"../config"
	"../database"
	"../monitoring"
	"../training"
	"../inference"
	pb "../../proto" // This would be the generated protobuf code
)

// Server implements the gRPC server for ML Pipeline
type Server struct {
	pb.UnimplementedMLPipelineServiceServer
	config     *config.Config
	logger     *zap.Logger
	repos      *database.Repositories
	monitor    *monitoring.ModelMonitor
	trainer    *training.TrainingEngine
	inferencer *inference.InferenceEngine
}

// NewServer creates a new gRPC server
func NewServer(
	cfg *config.Config,
	logger *zap.Logger,
	repos *database.Repositories,
	monitor *monitoring.ModelMonitor,
	trainer *training.TrainingEngine,
	inferencer *inference.InferenceEngine,
) *Server {
	return &Server{
		config:     cfg,
		logger:     logger,
		repos:      repos,
		monitor:    monitor,
		trainer:    trainer,
		inferencer: inferencer,
	}
}

// CreateModel creates a new ML model
func (s *Server) CreateModel(ctx context.Context, req *pb.CreateModelRequest) (*pb.CreateModelResponse, error) {
	s.logger.Info("Creating model via gRPC", zap.String("name", req.Name))

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "model name is required")
	}

	model := &database.Model{
		Name:        req.Name,
		Description: req.Description,
		Algorithm:   req.Algorithm,
		Status:      "created",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repos.ModelRepo.Create(ctx, model); err != nil {
		s.logger.Error("Failed to create model", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create model")
	}

	response := &pb.CreateModelResponse{
		Model: &pb.Model{
			Id:          model.ID,
			Name:        model.Name,
			Description: model.Description,
			Algorithm:   model.Algorithm,
			Status:      model.Status,
			Version:     model.Version,
			CreatedAt:   model.CreatedAt.Unix(),
			UpdatedAt:   model.UpdatedAt.Unix(),
		},
	}

	return response, nil
}

// GetModel retrieves a model by ID
func (s *Server) GetModel(ctx context.Context, req *pb.GetModelRequest) (*pb.GetModelResponse, error) {
	s.logger.Debug("Getting model via gRPC", zap.String("model_id", req.ModelId))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	model, err := s.repos.ModelRepo.GetByID(ctx, req.ModelId)
	if err != nil {
		s.logger.Error("Failed to get model", zap.Error(err))
		return nil, status.Error(codes.NotFound, "model not found")
	}

	response := &pb.GetModelResponse{
		Model: &pb.Model{
			Id:          model.ID,
			Name:        model.Name,
			Description: model.Description,
			Algorithm:   model.Algorithm,
			Status:      model.Status,
			Version:     model.Version,
			CreatedAt:   model.CreatedAt.Unix(),
			UpdatedAt:   model.UpdatedAt.Unix(),
		},
	}

	return response, nil
}

// ListModels lists all models
func (s *Server) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	s.logger.Debug("Listing models via gRPC")

	models, err := s.repos.ModelRepo.GetAll(ctx)
	if err != nil {
		s.logger.Error("Failed to list models", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list models")
	}

	var pbModels []*pb.Model
	for _, model := range models {
		pbModels = append(pbModels, &pb.Model{
			Id:          model.ID,
			Name:        model.Name,
			Description: model.Description,
			Algorithm:   model.Algorithm,
			Status:      model.Status,
			Version:     model.Version,
			CreatedAt:   model.CreatedAt.Unix(),
			UpdatedAt:   model.UpdatedAt.Unix(),
		})
	}

	response := &pb.ListModelsResponse{
		Models: pbModels,
	}

	return response, nil
}

// TrainModel starts training for a model
func (s *Server) TrainModel(ctx context.Context, req *pb.TrainModelRequest) (*pb.TrainModelResponse, error) {
	s.logger.Info("Starting model training via gRPC", zap.String("model_id", req.ModelId))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	if req.DatasetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "dataset path is required")
	}

	// Create training job
	job := &database.TrainingJob{
		ModelID:     req.ModelId,
		DatasetPath: req.DatasetPath,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.repos.TrainingJobRepo.Create(ctx, job); err != nil {
		s.logger.Error("Failed to create training job", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create training job")
	}

	// Submit training job
	if err := s.trainer.SubmitJob(ctx, job); err != nil {
		s.logger.Error("Failed to submit training job", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to submit training job")
	}

	response := &pb.TrainModelResponse{
		JobId:   job.ID,
		Status:  job.Status,
		Message: "Training job submitted successfully",
	}

	return response, nil
}

// GetTrainingStatus returns the status of a training job
func (s *Server) GetTrainingStatus(ctx context.Context, req *pb.GetTrainingStatusRequest) (*pb.GetTrainingStatusResponse, error) {
	s.logger.Debug("Getting training status via gRPC", zap.String("job_id", req.JobId))

	if req.JobId == "" {
		return nil, status.Error(codes.InvalidArgument, "job ID is required")
	}

	job, err := s.repos.TrainingJobRepo.GetByID(ctx, req.JobId)
	if err != nil {
		s.logger.Error("Failed to get training job", zap.Error(err))
		return nil, status.Error(codes.NotFound, "training job not found")
	}

	response := &pb.GetTrainingStatusResponse{
		JobId:     job.ID,
		ModelId:   job.ModelID,
		Status:    job.Status,
		Progress:  job.Progress,
		Message:   job.Message,
		CreatedAt: job.CreatedAt.Unix(),
		UpdatedAt: job.UpdatedAt.Unix(),
	}

	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Unix()
	}

	return response, nil
}

// DeployModel deploys a trained model
func (s *Server) DeployModel(ctx context.Context, req *pb.DeployModelRequest) (*pb.DeployModelResponse, error) {
	s.logger.Info("Deploying model via gRPC", 
		zap.String("model_id", req.ModelId),
		zap.String("version", req.Version))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	if req.Version == "" {
		return nil, status.Error(codes.InvalidArgument, "version is required")
	}

	// Create deployment record
	deployment := &database.ModelDeployment{
		ModelID:     req.ModelId,
		Version:     req.Version,
		Environment: req.Environment,
		Status:      "deploying",
		CreatedAt:   time.Now(),
	}

	if err := s.repos.DeploymentRepo.Create(ctx, deployment); err != nil {
		s.logger.Error("Failed to create deployment", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create deployment")
	}

	// Deploy model to inference engine
	config := make(map[string]interface{})
	if err := s.inferencer.DeployModel(ctx, req.ModelId, req.Version, config); err != nil {
		s.logger.Error("Failed to deploy model", zap.Error(err))
		deployment.Status = "failed"
		s.repos.DeploymentRepo.Update(ctx, deployment)
		return nil, status.Error(codes.Internal, "failed to deploy model")
	}

	deployment.Status = "deployed"
	if err := s.repos.DeploymentRepo.Update(ctx, deployment); err != nil {
		s.logger.Error("Failed to update deployment status", zap.Error(err))
	}

	response := &pb.DeployModelResponse{
		DeploymentId: deployment.ID,
		Status:       deployment.Status,
		Message:      "Model deployed successfully",
	}

	return response, nil
}

// Predict makes a prediction using a deployed model
func (s *Server) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictResponse, error) {
	s.logger.Debug("Making prediction via gRPC", zap.String("model_id", req.ModelId))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	if len(req.Features) == 0 {
		return nil, status.Error(codes.InvalidArgument, "features are required")
	}

	// Convert protobuf features to map
	features := make(map[string]interface{})
	for key, value := range req.Features {
		features[key] = value.GetStringValue() // Simplified - handle different types
	}

	// Make prediction
	result, err := s.inferencer.Predict(ctx, req.ModelId, features, req.Version)
	if err != nil {
		s.logger.Error("Prediction failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "prediction failed")
	}

	response := &pb.PredictResponse{
		PredictionId: result.ID,
		Result:       fmt.Sprintf("%v", result.Result),
		Confidence:   result.Confidence,
		ModelVersion: result.ModelVersion,
		Timestamp:    result.Timestamp.Unix(),
	}

	return response, nil
}

// BatchPredict makes batch predictions
func (s *Server) BatchPredict(ctx context.Context, req *pb.BatchPredictRequest) (*pb.BatchPredictResponse, error) {
	s.logger.Debug("Making batch predictions via gRPC", 
		zap.String("model_id", req.ModelId),
		zap.Int("batch_size", len(req.Features)))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	if len(req.Features) == 0 {
		return nil, status.Error(codes.InvalidArgument, "features are required")
	}

	// Convert protobuf features to slice of maps
	var featuresList []map[string]interface{}
	for _, featureMap := range req.Features {
		features := make(map[string]interface{})
		for key, value := range featureMap.Features {
			features[key] = value.GetStringValue() // Simplified
		}
		featuresList = append(featuresList, features)
	}

	// Make batch predictions
	results, err := s.inferencer.BatchPredict(ctx, req.ModelId, featuresList, req.Version)
	if err != nil {
		s.logger.Error("Batch prediction failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "batch prediction failed")
	}

	// Convert results to protobuf
	var pbResults []*pb.PredictionResult
	for _, result := range results {
		pbResults = append(pbResults, &pb.PredictionResult{
			PredictionId: result.ID,
			Result:       fmt.Sprintf("%v", result.Result),
			Confidence:   result.Confidence,
			ModelVersion: result.ModelVersion,
			Timestamp:    result.Timestamp.Unix(),
		})
	}

	response := &pb.BatchPredictResponse{
		Results: pbResults,
	}

	return response, nil
}

// GetModelMetrics returns monitoring metrics for a model
func (s *Server) GetModelMetrics(ctx context.Context, req *pb.GetModelMetricsRequest) (*pb.GetModelMetricsResponse, error) {
	s.logger.Debug("Getting model metrics via gRPC", zap.String("model_id", req.ModelId))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	metrics, err := s.monitor.GetMetrics(ctx, req.ModelId)
	if err != nil {
		s.logger.Error("Failed to get model metrics", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to retrieve metrics")
	}

	response := &pb.GetModelMetricsResponse{
		ModelId:     metrics.ModelID,
		ModelName:   metrics.ModelName,
		CollectedAt: metrics.CollectedAt.Unix(),
		Performance: &pb.PerformanceMetrics{
			Accuracy:   metrics.Performance.Accuracy,
			Precision:  metrics.Performance.Precision,
			Recall:     metrics.Performance.Recall,
			F1Score:    metrics.Performance.F1Score,
			Auc:        metrics.Performance.AUC,
			Latency:    metrics.Performance.Latency,
			Throughput: metrics.Performance.Throughput,
			ErrorRate:  metrics.Performance.ErrorRate,
		},
		Health: metrics.Health,
	}

	return response, nil
}

// GetModelHealth returns health status for a model
func (s *Server) GetModelHealth(ctx context.Context, req *pb.GetModelHealthRequest) (*pb.GetModelHealthResponse, error) {
	s.logger.Debug("Getting model health via gRPC", zap.String("model_id", req.ModelId))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	health, err := s.monitor.CheckHealth(ctx, req.ModelId)
	if err != nil {
		s.logger.Error("Failed to check model health", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to check model health")
	}

	response := &pb.GetModelHealthResponse{
		ModelId:     req.ModelId,
		Status:      health.Status,
		Score:       health.Score,
		CheckedAt:   health.CheckedAt.Unix(),
		Issues:      health.Issues,
	}

	return response, nil
}

// GetDriftStatus returns drift detection status for a model
func (s *Server) GetDriftStatus(ctx context.Context, req *pb.GetDriftStatusRequest) (*pb.GetDriftStatusResponse, error) {
	s.logger.Debug("Getting drift status via gRPC", zap.String("model_id", req.ModelId))

	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model ID is required")
	}

	driftStatus, err := s.monitor.GetDriftStatus(ctx, req.ModelId)
	if err != nil {
		s.logger.Error("Failed to get drift status", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to retrieve drift status")
	}

	response := &pb.GetDriftStatusResponse{
		ModelId:      req.ModelId,
		HasDrift:     driftStatus.HasDrift,
		DriftScore:   driftStatus.DriftScore,
		Threshold:    driftStatus.Threshold,
		LastChecked:  driftStatus.LastChecked.Unix(),
		AffectedFeatures: driftStatus.AffectedFeatures,
	}

	return response, nil
}