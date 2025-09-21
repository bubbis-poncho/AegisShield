package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"../../internal/config"
	"../../internal/database"
	"../../internal/models"
)

// InferenceEngine manages model inference operations
type InferenceEngine struct {
	config       *config.Config
	db           *database.Database
	repos        *database.Repositories
	logger       *zap.Logger
	modelCache   *ModelCache
	predictors   map[string]Predictor
	loadBalancer *LoadBalancer
	circuitBreaker *CircuitBreaker
	rateLimiter  *RateLimiter
	mu           sync.RWMutex
}

// PredictionRequest represents a prediction request
type PredictionRequest struct {
	RequestID       string                 `json:"request_id"`
	ModelID         string                 `json:"model_id"`
	ModelVersion    string                 `json:"model_version,omitempty"`
	Features        map[string]interface{} `json:"features"`
	RequestMetadata map[string]interface{} `json:"request_metadata,omitempty"`
	Timeout         time.Duration          `json:"timeout,omitempty"`
	Priority        int                    `json:"priority,omitempty"`
}

// PredictionResponse represents a prediction response
type PredictionResponse struct {
	RequestID       string                 `json:"request_id"`
	ModelID         string                 `json:"model_id"`
	ModelVersion    string                 `json:"model_version"`
	Prediction      interface{}            `json:"prediction"`
	Confidence      *float64               `json:"confidence,omitempty"`
	Probability     *float64               `json:"probability,omitempty"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	ResponseTime    time.Duration          `json:"response_time"`
	Features        map[string]interface{} `json:"features"`
	Status          string                 `json:"status"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// BatchPredictionRequest represents a batch prediction request
type BatchPredictionRequest struct {
	RequestID       string                   `json:"request_id"`
	ModelID         string                   `json:"model_id"`
	ModelVersion    string                   `json:"model_version,omitempty"`
	Features        []map[string]interface{} `json:"features"`
	RequestMetadata map[string]interface{}   `json:"request_metadata,omitempty"`
	BatchSize       int                      `json:"batch_size,omitempty"`
	Timeout         time.Duration            `json:"timeout,omitempty"`
}

// BatchPredictionResponse represents a batch prediction response
type BatchPredictionResponse struct {
	RequestID      string                `json:"request_id"`
	ModelID        string                `json:"model_id"`
	ModelVersion   string                `json:"model_version"`
	Predictions    []PredictionResponse  `json:"predictions"`
	ProcessingTime time.Duration         `json:"processing_time"`
	ResponseTime   time.Duration         `json:"response_time"`
	Status         string                `json:"status"`
	ErrorMessage   string                `json:"error_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Predictor interface for different model types
type Predictor interface {
	Predict(ctx context.Context, features map[string]interface{}) (*PredictionResult, error)
	PredictBatch(ctx context.Context, features []map[string]interface{}) ([]*PredictionResult, error)
	GetModelInfo() *ModelInfo
	IsHealthy() bool
	Warmup(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// PredictionResult represents the result of a single prediction
type PredictionResult struct {
	Prediction   interface{} `json:"prediction"`
	Confidence   *float64    `json:"confidence,omitempty"`
	Probability  *float64    `json:"probability,omitempty"`
	FeatureUsed  map[string]interface{} `json:"features_used"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ModelInfo holds information about a loaded model
type ModelInfo struct {
	ModelID      string                 `json:"model_id"`
	Version      string                 `json:"version"`
	Type         models.ModelType       `json:"type"`
	Algorithm    models.AlgorithmType   `json:"algorithm"`
	LoadedAt     time.Time              `json:"loaded_at"`
	LastUsed     time.Time              `json:"last_used"`
	PredictionCount int64               `json:"prediction_count"`
	Configuration map[string]interface{} `json:"configuration"`
}

// NewInferenceEngine creates a new inference engine
func NewInferenceEngine(cfg *config.Config, db *database.Database, repos *database.Repositories, logger *zap.Logger) *InferenceEngine {
	engine := &InferenceEngine{
		config:     cfg,
		db:         db,
		repos:      repos,
		logger:     logger,
		modelCache: NewModelCache(cfg.ML.Inference.CacheEnabled, cfg.ML.Inference.CacheTTL),
		predictors: make(map[string]Predictor),
	}

	// Initialize components
	engine.loadBalancer = NewLoadBalancer(cfg.ML.Inference.LoadBalancing)
	
	if cfg.ML.Inference.CircuitBreaker.Enabled {
		engine.circuitBreaker = NewCircuitBreaker(
			cfg.ML.Inference.CircuitBreaker.FailureThreshold,
			cfg.ML.Inference.CircuitBreaker.RecoveryTimeout,
			cfg.ML.Inference.CircuitBreaker.SuccessThreshold,
		)
	}

	if cfg.ML.Inference.RateLimiting.Enabled {
		engine.rateLimiter = NewRateLimiter(
			cfg.ML.Inference.RateLimiting.RequestsPerSecond,
			cfg.ML.Inference.RateLimiting.BurstSize,
		)
	}

	return engine
}

// LoadModel loads a model for inference
func (e *InferenceEngine) LoadModel(ctx context.Context, modelID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	logger := e.logger.With(zap.String("model_id", modelID))
	logger.Info("Loading model for inference")

	// Get model from database
	model, err := e.repos.Model.GetByID(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	if model.Status != models.ModelStatusDeployed && model.Status != models.ModelStatusTrained {
		return fmt.Errorf("model is not in deployable state: %s", model.Status)
	}

	// Check if model is already loaded
	if _, exists := e.predictors[modelID]; exists {
		logger.Info("Model already loaded")
		return nil
	}

	// Create predictor based on algorithm
	predictor, err := e.createPredictor(model)
	if err != nil {
		return fmt.Errorf("failed to create predictor: %w", err)
	}

	// Warm up the model if enabled
	if e.config.ML.Inference.ModelWarmup {
		if err := predictor.Warmup(ctx); err != nil {
			logger.Warn("Model warmup failed", zap.Error(err))
		}
	}

	e.predictors[modelID] = predictor
	logger.Info("Model loaded successfully")
	return nil
}

// UnloadModel unloads a model from memory
func (e *InferenceEngine) UnloadModel(ctx context.Context, modelID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	logger := e.logger.With(zap.String("model_id", modelID))

	predictor, exists := e.predictors[modelID]
	if !exists {
		return fmt.Errorf("model not loaded: %s", modelID)
	}

	// Shutdown the predictor
	if err := predictor.Shutdown(ctx); err != nil {
		logger.Warn("Error shutting down predictor", zap.Error(err))
	}

	delete(e.predictors, modelID)
	logger.Info("Model unloaded successfully")
	return nil
}

// Predict performs single prediction
func (e *InferenceEngine) Predict(ctx context.Context, request *PredictionRequest) (*PredictionResponse, error) {
	startTime := time.Now()
	logger := e.logger.With(
		zap.String("request_id", request.RequestID),
		zap.String("model_id", request.ModelID),
	)

	// Apply rate limiting if enabled
	if e.rateLimiter != nil {
		if !e.rateLimiter.Allow() {
			return &PredictionResponse{
				RequestID:    request.RequestID,
				ModelID:      request.ModelID,
				Status:       "error",
				ErrorMessage: "rate limit exceeded",
				ResponseTime: time.Since(startTime),
			}, fmt.Errorf("rate limit exceeded")
		}
	}

	// Set timeout if specified
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	} else if e.config.ML.Inference.MaxLatency > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.ML.Inference.MaxLatency)
		defer cancel()
	}

	// Get predictor
	predictor, err := e.getPredictor(request.ModelID)
	if err != nil {
		return &PredictionResponse{
			RequestID:    request.RequestID,
			ModelID:      request.ModelID,
			Status:       "error",
			ErrorMessage: err.Error(),
			ResponseTime: time.Since(startTime),
		}, err
	}

	processingStart := time.Now()

	// Perform prediction
	var result *PredictionResult
	if e.circuitBreaker != nil {
		result, err = e.circuitBreaker.Execute(func() (*PredictionResult, error) {
			return predictor.Predict(ctx, request.Features)
		})
	} else {
		result, err = predictor.Predict(ctx, request.Features)
	}

	processingTime := time.Since(processingStart)

	if err != nil {
		logger.Error("Prediction failed", zap.Error(err))
		return &PredictionResponse{
			RequestID:      request.RequestID,
			ModelID:        request.ModelID,
			Status:         "error",
			ErrorMessage:   err.Error(),
			ProcessingTime: processingTime,
			ResponseTime:   time.Since(startTime),
		}, err
	}

	modelInfo := predictor.GetModelInfo()
	response := &PredictionResponse{
		RequestID:      request.RequestID,
		ModelID:        request.ModelID,
		ModelVersion:   modelInfo.Version,
		Prediction:     result.Prediction,
		Confidence:     result.Confidence,
		Probability:    result.Probability,
		ProcessingTime: processingTime,
		ResponseTime:   time.Since(startTime),
		Features:       result.FeatureUsed,
		Status:         "success",
		Metadata:       result.Metadata,
	}

	// Store prediction request asynchronously
	go e.storePredictionRequest(request, response)

	logger.Info("Prediction completed",
		zap.Duration("processing_time", processingTime),
		zap.Duration("response_time", response.ResponseTime))

	return response, nil
}

// PredictBatch performs batch prediction
func (e *InferenceEngine) PredictBatch(ctx context.Context, request *BatchPredictionRequest) (*BatchPredictionResponse, error) {
	startTime := time.Now()
	logger := e.logger.With(
		zap.String("request_id", request.RequestID),
		zap.String("model_id", request.ModelID),
		zap.Int("batch_size", len(request.Features)),
	)

	// Set timeout
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	// Get predictor
	predictor, err := e.getPredictor(request.ModelID)
	if err != nil {
		return &BatchPredictionResponse{
			RequestID:    request.RequestID,
			ModelID:      request.ModelID,
			Status:       "error",
			ErrorMessage: err.Error(),
			ResponseTime: time.Since(startTime),
		}, err
	}

	processingStart := time.Now()

	// Determine batch size
	batchSize := request.BatchSize
	if batchSize <= 0 {
		batchSize = e.config.ML.Inference.BatchSize
	}

	var allResults []*PredictionResult
	var allResponses []PredictionResponse

	// Process in batches
	for i := 0; i < len(request.Features); i += batchSize {
		end := i + batchSize
		if end > len(request.Features) {
			end = len(request.Features)
		}

		batch := request.Features[i:end]
		results, err := predictor.PredictBatch(ctx, batch)
		if err != nil {
			logger.Error("Batch prediction failed", zap.Error(err), zap.Int("batch_start", i))
			// Continue with partial results
			break
		}

		allResults = append(allResults, results...)

		// Convert to responses
		for j, result := range results {
			response := PredictionResponse{
				RequestID:    fmt.Sprintf("%s_%d", request.RequestID, i+j),
				ModelID:      request.ModelID,
				Prediction:   result.Prediction,
				Confidence:   result.Confidence,
				Probability:  result.Probability,
				Features:     result.FeatureUsed,
				Status:       "success",
				Metadata:     result.Metadata,
			}
			allResponses = append(allResponses, response)
		}
	}

	processingTime := time.Since(processingStart)
	modelInfo := predictor.GetModelInfo()

	response := &BatchPredictionResponse{
		RequestID:      request.RequestID,
		ModelID:        request.ModelID,
		ModelVersion:   modelInfo.Version,
		Predictions:    allResponses,
		ProcessingTime: processingTime,
		ResponseTime:   time.Since(startTime),
		Status:         "success",
	}

	logger.Info("Batch prediction completed",
		zap.Int("total_predictions", len(allResults)),
		zap.Duration("processing_time", processingTime))

	return response, nil
}

// getPredictor gets a predictor for the specified model
func (e *InferenceEngine) getPredictor(modelID string) (Predictor, error) {
	e.mu.RLock()
	predictor, exists := e.predictors[modelID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model not loaded: %s", modelID)
	}

	if !predictor.IsHealthy() {
		return nil, fmt.Errorf("model is unhealthy: %s", modelID)
	}

	return predictor, nil
}

// createPredictor creates a predictor for the given model
func (e *InferenceEngine) createPredictor(model *models.Model) (Predictor, error) {
	switch model.Algorithm {
	case models.AlgorithmXGBoost:
		return NewXGBoostPredictor(model, e.config, e.logger)
	case models.AlgorithmRandomForest:
		return NewRandomForestPredictor(model, e.config, e.logger)
	case models.AlgorithmLogisticRegression:
		return NewLogisticRegressionPredictor(model, e.config, e.logger)
	case models.AlgorithmNeuralNetwork:
		return NewNeuralNetworkPredictor(model, e.config, e.logger)
	case models.AlgorithmIsolationForest:
		return NewIsolationForestPredictor(model, e.config, e.logger)
	case models.AlgorithmLSTM:
		return NewLSTMPredictor(model, e.config, e.logger)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", model.Algorithm)
	}
}

// storePredictionRequest stores prediction request for monitoring and analysis
func (e *InferenceEngine) storePredictionRequest(request *PredictionRequest, response *PredictionResponse) {
	// Convert to database model
	predictionRequest := &models.PredictionRequest{
		RequestID:      request.RequestID,
		ModelID:        uuid.MustParse(request.ModelID),
		RequestedAt:    time.Now().UTC(),
		ProcessedAt:    &time.Time{},
		ProcessingTime: response.ProcessingTime,
		ResponseTime:   response.ResponseTime,
		Status:         models.RequestStatus(response.Status),
	}

	// Set processed time
	processedAt := time.Now().UTC()
	predictionRequest.ProcessedAt = &processedAt

	// Serialize features
	if featuresJSON, err := json.Marshal(request.Features); err == nil {
		predictionRequest.Features = models.JSON(featuresJSON)
	}

	// Serialize prediction
	if predictionJSON, err := json.Marshal(response.Prediction); err == nil {
		predictionRequest.Prediction = models.JSON(predictionJSON)
	}

	// Serialize metadata
	if request.RequestMetadata != nil {
		if metadataJSON, err := json.Marshal(request.RequestMetadata); err == nil {
			predictionRequest.RequestMetadata = models.JSON(metadataJSON)
		}
	}

	// Set confidence and probability
	predictionRequest.Confidence = response.Confidence
	predictionRequest.Probability = response.Probability

	// Store in database
	if err := e.repos.PredictionRequest.Create(predictionRequest); err != nil {
		e.logger.Error("Failed to store prediction request", zap.Error(err))
	}
}

// GetModelInfo returns information about loaded models
func (e *InferenceEngine) GetModelInfo() map[string]*ModelInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	info := make(map[string]*ModelInfo)
	for modelID, predictor := range e.predictors {
		info[modelID] = predictor.GetModelInfo()
	}
	return info
}

// GetLoadedModels returns a list of loaded model IDs
func (e *InferenceEngine) GetLoadedModels() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var models []string
	for modelID := range e.predictors {
		models = append(models, modelID)
	}
	return models
}

// HealthCheck performs health check on all loaded models
func (e *InferenceEngine) HealthCheck() map[string]bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	health := make(map[string]bool)
	for modelID, predictor := range e.predictors {
		health[modelID] = predictor.IsHealthy()
	}
	return health
}

// GetMetrics returns inference metrics
func (e *InferenceEngine) GetMetrics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	metrics := map[string]interface{}{
		"loaded_models": len(e.predictors),
		"cache_stats":   e.modelCache.GetStats(),
	}

	if e.circuitBreaker != nil {
		metrics["circuit_breaker"] = e.circuitBreaker.GetStats()
	}

	if e.rateLimiter != nil {
		metrics["rate_limiter"] = e.rateLimiter.GetStats()
	}

	// Add model-specific metrics
	modelMetrics := make(map[string]interface{})
	for modelID, predictor := range e.predictors {
		modelInfo := predictor.GetModelInfo()
		modelMetrics[modelID] = map[string]interface{}{
			"prediction_count": modelInfo.PredictionCount,
			"last_used":       modelInfo.LastUsed,
			"is_healthy":      predictor.IsHealthy(),
		}
	}
	metrics["models"] = modelMetrics

	return metrics
}

// Shutdown gracefully shuts down the inference engine
func (e *InferenceEngine) Shutdown(ctx context.Context) error {
	e.logger.Info("Shutting down inference engine")

	e.mu.Lock()
	defer e.mu.Unlock()

	// Shutdown all predictors
	for modelID, predictor := range e.predictors {
		if err := predictor.Shutdown(ctx); err != nil {
			e.logger.Warn("Error shutting down predictor",
				zap.String("model_id", modelID),
				zap.Error(err))
		}
	}

	// Clear predictors map
	e.predictors = make(map[string]Predictor)

	e.logger.Info("Inference engine shutdown complete")
	return nil
}