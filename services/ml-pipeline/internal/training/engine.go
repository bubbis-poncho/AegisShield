package training

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"../../internal/config"
	"../../internal/database"
	"../../internal/models"
)

// TrainingEngine manages model training operations
type TrainingEngine struct {
	config      *config.Config
	db          *database.Database
	repos       *database.Repositories
	logger      *zap.Logger
	trainers    map[models.AlgorithmType]Trainer
	jobQueue    chan *TrainingJobRequest
	workers     []*TrainingWorker
}

// TrainingJobRequest represents a training job request
type TrainingJobRequest struct {
	JobID           uuid.UUID
	ModelID         uuid.UUID
	Algorithm       models.AlgorithmType
	Hyperparameters map[string]interface{}
	Configuration   map[string]interface{}
	DataConfig      *DataConfiguration
	Context         context.Context
}

// DataConfiguration holds training data configuration
type DataConfiguration struct {
	TrainingDataset   string                 `json:"training_dataset"`
	ValidationDataset string                 `json:"validation_dataset"`
	TestDataset       string                 `json:"test_dataset"`
	FeatureConfig     map[string]interface{} `json:"feature_config"`
	DataWindow        time.Duration          `json:"data_window"`
	ValidationSplit   float64                `json:"validation_split"`
	TestSplit         float64                `json:"test_split"`
	Preprocessing     *PreprocessingConfig   `json:"preprocessing"`
}

// PreprocessingConfig holds data preprocessing configuration
type PreprocessingConfig struct {
	NormalizeFeatures   bool                   `json:"normalize_features"`
	HandleMissingValues string                 `json:"handle_missing_values"` // drop, impute, forward_fill
	ImputationStrategy  string                 `json:"imputation_strategy"`   // mean, median, mode, constant
	FeatureSelection    *FeatureSelectionConfig `json:"feature_selection"`
	OutlierDetection    *OutlierDetectionConfig `json:"outlier_detection"`
	DataBalancing       *DataBalancingConfig   `json:"data_balancing"`
}

// FeatureSelectionConfig holds feature selection configuration
type FeatureSelectionConfig struct {
	Enabled    bool     `json:"enabled"`
	Method     string   `json:"method"`     // variance, correlation, mutual_info, recursive
	MaxFeatures int     `json:"max_features"`
	Threshold   float64  `json:"threshold"`
}

// OutlierDetectionConfig holds outlier detection configuration
type OutlierDetectionConfig struct {
	Enabled   bool    `json:"enabled"`
	Method    string  `json:"method"`    // iqr, zscore, isolation_forest
	Threshold float64 `json:"threshold"`
	Action    string  `json:"action"`    // remove, cap, log
}

// DataBalancingConfig holds data balancing configuration
type DataBalancingConfig struct {
	Enabled   bool   `json:"enabled"`
	Method    string `json:"method"`    // oversample, undersample, smote
	Ratio     float64 `json:"ratio"`
}

// TrainingResult holds the results of model training
type TrainingResult struct {
	Success         bool                   `json:"success"`
	ModelPath       string                 `json:"model_path"`
	ArtifactsPath   string                 `json:"artifacts_path"`
	Metrics         map[string]float64     `json:"metrics"`
	ValidationMetrics map[string]float64   `json:"validation_metrics"`
	TestMetrics     map[string]float64     `json:"test_metrics"`
	FeatureImportance map[string]float64   `json:"feature_importance"`
	TrainingDuration time.Duration         `json:"training_duration"`
	ResourceUsage   map[string]interface{} `json:"resource_usage"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Artifacts       map[string]string      `json:"artifacts"`
}

// Trainer interface for different ML algorithms
type Trainer interface {
	Train(ctx context.Context, request *TrainingJobRequest) (*TrainingResult, error)
	ValidateConfig(config map[string]interface{}) error
	GetDefaultConfig() map[string]interface{}
	GetSupportedMetrics() []string
}

// TrainingWorker processes training jobs
type TrainingWorker struct {
	id      int
	engine  *TrainingEngine
	logger  *zap.Logger
	stop    chan bool
	stopped chan bool
}

// NewTrainingEngine creates a new training engine
func NewTrainingEngine(cfg *config.Config, db *database.Database, repos *database.Repositories, logger *zap.Logger) *TrainingEngine {
	engine := &TrainingEngine{
		config:   cfg,
		db:       db,
		repos:    repos,
		logger:   logger,
		trainers: make(map[models.AlgorithmType]Trainer),
		jobQueue: make(chan *TrainingJobRequest, 100),
	}

	// Register algorithm trainers
	engine.registerTrainers()

	// Start worker goroutines
	engine.startWorkers()

	return engine
}

// registerTrainers registers all available training algorithms
func (e *TrainingEngine) registerTrainers() {
	e.trainers[models.AlgorithmXGBoost] = NewXGBoostTrainer(e.config, e.logger)
	e.trainers[models.AlgorithmRandomForest] = NewRandomForestTrainer(e.config, e.logger)
	e.trainers[models.AlgorithmLogisticRegression] = NewLogisticRegressionTrainer(e.config, e.logger)
	e.trainers[models.AlgorithmNeuralNetwork] = NewNeuralNetworkTrainer(e.config, e.logger)
	e.trainers[models.AlgorithmIsolationForest] = NewIsolationForestTrainer(e.config, e.logger)
	e.trainers[models.AlgorithmLSTM] = NewLSTMTrainer(e.config, e.logger)
}

// startWorkers starts the training worker goroutines
func (e *TrainingEngine) startWorkers() {
	maxWorkers := e.config.ML.Training.MaxConcurrentJobs
	e.workers = make([]*TrainingWorker, maxWorkers)

	for i := 0; i < maxWorkers; i++ {
		worker := &TrainingWorker{
			id:      i,
			engine:  e,
			logger:  e.logger.With(zap.Int("worker_id", i)),
			stop:    make(chan bool, 1),
			stopped: make(chan bool, 1),
		}
		e.workers[i] = worker
		go worker.start()
	}

	e.logger.Info("Started training workers", zap.Int("worker_count", maxWorkers))
}

// SubmitTrainingJob submits a new training job
func (e *TrainingEngine) SubmitTrainingJob(ctx context.Context, job *models.TrainingJob) error {
	// Validate the training job
	if err := e.validateTrainingJob(job); err != nil {
		return fmt.Errorf("invalid training job: %w", err)
	}

	// Create training job request
	request := &TrainingJobRequest{
		JobID:           job.ID,
		ModelID:         job.ModelID,
		Algorithm:       job.Algorithm,
		Hyperparameters: make(map[string]interface{}),
		Configuration:   make(map[string]interface{}),
		Context:         ctx,
	}

	// Parse hyperparameters
	if job.Hyperparameters != nil {
		if err := json.Unmarshal(job.Hyperparameters, &request.Hyperparameters); err != nil {
			return fmt.Errorf("failed to parse hyperparameters: %w", err)
		}
	}

	// Parse configuration
	if job.Configuration != nil {
		if err := json.Unmarshal(job.Configuration, &request.Configuration); err != nil {
			return fmt.Errorf("failed to parse configuration: %w", err)
		}
	}

	// Parse data configuration
	if job.FeatureConfig != nil {
		dataConfig := &DataConfiguration{}
		if err := json.Unmarshal(job.FeatureConfig, dataConfig); err != nil {
			return fmt.Errorf("failed to parse data configuration: %w", err)
		}
		request.DataConfig = dataConfig
	}

	// Set default data configuration if not provided
	if request.DataConfig == nil {
		request.DataConfig = &DataConfiguration{
			TrainingDataset:   job.TrainingDataset,
			ValidationDataset: job.ValidationDataset,
			TestDataset:       job.TestDataset,
			ValidationSplit:   e.config.ML.Training.ValidationSplit,
		}
	}

	// Update job status to pending
	job.Status = models.TrainingStatusPending
	if err := e.repos.TrainingJob.Update(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Submit to job queue
	select {
	case e.jobQueue <- request:
		e.logger.Info("Training job submitted", zap.String("job_id", job.ID.String()))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("training queue is full")
	}
}

// validateTrainingJob validates a training job
func (e *TrainingEngine) validateTrainingJob(job *models.TrainingJob) error {
	if job.ModelID == uuid.Nil {
		return fmt.Errorf("model ID is required")
	}

	if job.Algorithm == "" {
		return fmt.Errorf("algorithm is required")
	}

	if _, exists := e.trainers[job.Algorithm]; !exists {
		return fmt.Errorf("unsupported algorithm: %s", job.Algorithm)
	}

	if job.TrainingDataset == "" {
		return fmt.Errorf("training dataset is required")
	}

	return nil
}

// GetJobStatus returns the status of a training job
func (e *TrainingEngine) GetJobStatus(jobID uuid.UUID) (*models.TrainingJob, error) {
	return e.repos.TrainingJob.GetByID(jobID.String())
}

// CancelJob cancels a training job
func (e *TrainingEngine) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := e.repos.TrainingJob.GetByID(jobID.String())
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status != models.TrainingStatusPending && job.Status != models.TrainingStatusRunning {
		return fmt.Errorf("cannot cancel job in status: %s", job.Status)
	}

	job.Status = models.TrainingStatusCancelled
	job.UpdatedBy = "system" // TODO: Get from context
	return e.repos.TrainingJob.Update(job)
}

// GetRunningJobs returns all currently running training jobs
func (e *TrainingEngine) GetRunningJobs() ([]*models.TrainingJob, error) {
	return e.repos.TrainingJob.GetRunningJobs()
}

// GetQueuedJobs returns the number of queued training jobs
func (e *TrainingEngine) GetQueuedJobs() int {
	return len(e.jobQueue)
}

// Shutdown gracefully shuts down the training engine
func (e *TrainingEngine) Shutdown(ctx context.Context) error {
	e.logger.Info("Shutting down training engine")

	// Stop all workers
	for _, worker := range e.workers {
		worker.stop <- true
	}

	// Wait for workers to stop
	for _, worker := range e.workers {
		select {
		case <-worker.stopped:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	e.logger.Info("Training engine shutdown complete")
	return nil
}

// start starts the training worker
func (w *TrainingWorker) start() {
	w.logger.Info("Training worker started")
	defer func() {
		w.stopped <- true
		w.logger.Info("Training worker stopped")
	}()

	for {
		select {
		case <-w.stop:
			return
		case request := <-w.engine.jobQueue:
			w.processTrainingJob(request)
		}
	}
}

// processTrainingJob processes a single training job
func (w *TrainingWorker) processTrainingJob(request *TrainingJobRequest) {
	ctx := request.Context
	logger := w.logger.With(zap.String("job_id", request.JobID.String()))

	logger.Info("Processing training job", zap.String("algorithm", string(request.Algorithm)))

	// Get the training job from database
	job, err := w.engine.repos.TrainingJob.GetByID(request.JobID.String())
	if err != nil {
		logger.Error("Failed to get training job", zap.Error(err))
		return
	}

	// Check if job was cancelled
	if job.Status == models.TrainingStatusCancelled {
		logger.Info("Training job was cancelled")
		return
	}

	// Update job status to running
	startTime := time.Now()
	job.Status = models.TrainingStatusRunning
	job.StartedAt = &startTime
	if err := w.engine.repos.TrainingJob.Update(job); err != nil {
		logger.Error("Failed to update job status", zap.Error(err))
		return
	}

	// Get the trainer for this algorithm
	trainer, exists := w.engine.trainers[request.Algorithm]
	if !exists {
		w.failJob(job, fmt.Sprintf("unsupported algorithm: %s", request.Algorithm))
		return
	}

	// Train the model
	result, err := trainer.Train(ctx, request)
	if err != nil {
		w.failJob(job, fmt.Sprintf("training failed: %v", err))
		return
	}

	// Update job with results
	w.completeJob(job, result)
}

// completeJob marks a training job as completed
func (w *TrainingWorker) completeJob(job *models.TrainingJob, result *TrainingResult) {
	logger := w.logger.With(zap.String("job_id", job.ID.String()))

	completedAt := time.Now()
	duration := completedAt.Sub(*job.StartedAt)

	job.Status = models.TrainingStatusCompleted
	job.CompletedAt = &completedAt
	job.Duration = &duration
	job.ModelPath = result.ModelPath
	job.ArtifactsPath = result.ArtifactsPath

	// Serialize metrics
	if result.Metrics != nil {
		metricsJSON, _ := json.Marshal(result.Metrics)
		job.Metrics = models.JSON(metricsJSON)
	}

	if result.ValidationMetrics != nil {
		validationJSON, _ := json.Marshal(result.ValidationMetrics)
		job.ValidationMetrics = models.JSON(validationJSON)
	}

	if result.TestMetrics != nil {
		testJSON, _ := json.Marshal(result.TestMetrics)
		job.TestMetrics = models.JSON(testJSON)
	}

	if result.ResourceUsage != nil {
		resourceJSON, _ := json.Marshal(result.ResourceUsage)
		job.ResourceUsage = models.JSON(resourceJSON)
	}

	job.UpdatedBy = "system" // TODO: Get from context

	if err := w.engine.repos.TrainingJob.Update(job); err != nil {
		logger.Error("Failed to update completed job", zap.Error(err))
		return
	}

	// Update the associated model
	w.updateModelFromTrainingResult(job, result)

	logger.Info("Training job completed successfully",
		zap.Duration("duration", duration),
		zap.String("model_path", result.ModelPath))
}

// failJob marks a training job as failed
func (w *TrainingWorker) failJob(job *models.TrainingJob, errorMessage string) {
	logger := w.logger.With(zap.String("job_id", job.ID.String()))

	completedAt := time.Now()
	var duration time.Duration
	if job.StartedAt != nil {
		duration = completedAt.Sub(*job.StartedAt)
	}

	job.Status = models.TrainingStatusFailed
	job.CompletedAt = &completedAt
	job.Duration = &duration
	job.ErrorMessage = errorMessage
	job.UpdatedBy = "system" // TODO: Get from context

	if err := w.engine.repos.TrainingJob.Update(job); err != nil {
		logger.Error("Failed to update failed job", zap.Error(err))
		return
	}

	// Update the associated model status
	if model, err := w.engine.repos.Model.GetByID(job.ModelID.String()); err == nil {
		model.Status = models.ModelStatusFailed
		model.UpdatedBy = "system"
		w.engine.repos.Model.Update(model)
	}

	logger.Error("Training job failed", zap.String("error", errorMessage))
}

// updateModelFromTrainingResult updates the model with training results
func (w *TrainingWorker) updateModelFromTrainingResult(job *models.TrainingJob, result *TrainingResult) {
	logger := w.logger.With(zap.String("job_id", job.ID.String()))

	model, err := w.engine.repos.Model.GetByID(job.ModelID.String())
	if err != nil {
		logger.Error("Failed to get model", zap.Error(err))
		return
	}

	// Update model with training results
	model.Status = models.ModelStatusTrained
	model.TrainingJobID = &job.ID
	model.TrainingStarted = job.StartedAt
	model.TrainingCompleted = job.CompletedAt
	model.TrainingDuration = job.Duration
	model.ModelPath = result.ModelPath
	model.ArtifactsPath = result.ArtifactsPath
	model.Metrics = job.Metrics
	model.ValidationMetrics = job.ValidationMetrics
	model.TestMetrics = job.TestMetrics
	model.UpdatedBy = "system" // TODO: Get from context

	if err := w.engine.repos.Model.Update(model); err != nil {
		logger.Error("Failed to update model", zap.Error(err))
		return
	}

	logger.Info("Model updated with training results", zap.String("model_id", model.ID.String()))
}

// RetryFailedJob retries a failed training job
func (e *TrainingEngine) RetryFailedJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := e.repos.TrainingJob.GetByID(jobID.String())
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status != models.TrainingStatusFailed {
		return fmt.Errorf("can only retry failed jobs, current status: %s", job.Status)
	}

	// Reset job status and counters
	job.Status = models.TrainingStatusPending
	job.RetryCount++
	job.StartedAt = nil
	job.CompletedAt = nil
	job.Duration = nil
	job.ErrorMessage = ""
	job.UpdatedBy = "system" // TODO: Get from context

	if err := e.repos.TrainingJob.Update(job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Resubmit the job
	return e.SubmitTrainingJob(ctx, job)
}

// CreateModelArtifactsDir creates the directory structure for model artifacts
func CreateModelArtifactsDir(basePath, modelID, version string) (string, error) {
	artifactsPath := filepath.Join(basePath, "models", modelID, version)
	if err := os.MkdirAll(artifactsPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create artifacts directory: %w", err)
	}
	return artifactsPath, nil
}

// SaveModelMetadata saves model metadata to a JSON file
func SaveModelMetadata(artifactsPath string, metadata map[string]interface{}) error {
	metadataPath := filepath.Join(artifactsPath, "metadata.json")
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// LoadModelMetadata loads model metadata from a JSON file
func LoadModelMetadata(artifactsPath string) (map[string]interface{}, error) {
	metadataPath := filepath.Join(artifactsPath, "metadata.json")
	metadataJSON, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}