package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"../config"
	"../database"
	"../monitoring"
	"../training"
	"../inference"
)

// Handler contains all API handlers
type Handler struct {
	config       *config.Config
	logger       *zap.Logger
	repos        *database.Repositories
	monitor      *monitoring.ModelMonitor
	trainer      *training.TrainingEngine
	inferencer   *inference.InferenceEngine
}

// NewHandler creates a new API handler
func NewHandler(
	cfg *config.Config,
	logger *zap.Logger,
	repos *database.Repositories,
	monitor *monitoring.ModelMonitor,
	trainer *training.TrainingEngine,
	inferencer *inference.InferenceEngine,
) *Handler {
	return &Handler{
		config:     cfg,
		logger:     logger,
		repos:      repos,
		monitor:    monitor,
		trainer:    trainer,
		inferencer: inferencer,
	}
}

// Health returns service health status
func (h *Handler) Health(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services": map[string]string{
			"database":  "healthy",
			"trainer":   "healthy",
			"inference": "healthy",
			"monitor":   "healthy",
		},
	}

	c.JSON(http.StatusOK, health)
}

// GetModels returns list of all models
func (h *Handler) GetModels(c *gin.Context) {
	models, err := h.repos.ModelRepo.GetAll(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get models", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

// GetModel returns a specific model by ID
func (h *Handler) GetModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	model, err := h.repos.ModelRepo.GetByID(c.Request.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to get model", zap.String("model_id", modelID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	c.JSON(http.StatusOK, model)
}

// CreateModel creates a new model
func (h *Handler) CreateModel(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Algorithm   string `json:"algorithm" binding:"required"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model := &database.Model{
		Name:        req.Name,
		Description: req.Description,
		Algorithm:   req.Algorithm,
		Parameters:  req.Parameters,
		Status:      "created",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repos.ModelRepo.Create(c.Request.Context(), model); err != nil {
		h.logger.Error("Failed to create model", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create model"})
		return
	}

	h.logger.Info("Model created", zap.String("model_id", model.ID), zap.String("name", model.Name))
	c.JSON(http.StatusCreated, model)
}

// TrainModel starts training for a model
func (h *Handler) TrainModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	var req struct {
		DatasetPath string                 `json:"dataset_path" binding:"required"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create training job
	job := &database.TrainingJob{
		ModelID:     modelID,
		DatasetPath: req.DatasetPath,
		Parameters:  req.Parameters,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := h.repos.TrainingJobRepo.Create(c.Request.Context(), job); err != nil {
		h.logger.Error("Failed to create training job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create training job"})
		return
	}

	// Submit training job
	if err := h.trainer.SubmitJob(c.Request.Context(), job); err != nil {
		h.logger.Error("Failed to submit training job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit training job"})
		return
	}

	h.logger.Info("Training job submitted", zap.String("job_id", job.ID), zap.String("model_id", modelID))
	c.JSON(http.StatusAccepted, job)
}

// GetTrainingJobs returns training jobs for a model
func (h *Handler) GetTrainingJobs(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	jobs, err := h.repos.TrainingJobRepo.GetByModelID(c.Request.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to get training jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve training jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// GetTrainingJob returns a specific training job
func (h *Handler) GetTrainingJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	job, err := h.repos.TrainingJobRepo.GetByID(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to get training job", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Training job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// DeployModel deploys a trained model
func (h *Handler) DeployModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	var req struct {
		Version     string                 `json:"version" binding:"required"`
		Environment string                 `json:"environment" binding:"required"`
		Config      map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deployment := &database.ModelDeployment{
		ModelID:     modelID,
		Version:     req.Version,
		Environment: req.Environment,
		Config:      req.Config,
		Status:      "deploying",
		CreatedAt:   time.Now(),
	}

	if err := h.repos.DeploymentRepo.Create(c.Request.Context(), deployment); err != nil {
		h.logger.Error("Failed to create deployment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment"})
		return
	}

	// Deploy model to inference engine
	if err := h.inferencer.DeployModel(c.Request.Context(), modelID, req.Version, req.Config); err != nil {
		h.logger.Error("Failed to deploy model", zap.Error(err))
		deployment.Status = "failed"
		h.repos.DeploymentRepo.Update(c.Request.Context(), deployment)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deploy model"})
		return
	}

	deployment.Status = "deployed"
	if err := h.repos.DeploymentRepo.Update(c.Request.Context(), deployment); err != nil {
		h.logger.Error("Failed to update deployment status", zap.Error(err))
	}

	h.logger.Info("Model deployed", zap.String("model_id", modelID), zap.String("version", req.Version))
	c.JSON(http.StatusOK, deployment)
}

// GetDeployments returns deployments for a model
func (h *Handler) GetDeployments(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	deployments, err := h.repos.DeploymentRepo.GetByModelID(c.Request.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to get deployments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve deployments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deployments": deployments})
}

// Predict makes a prediction using a deployed model
func (h *Handler) Predict(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	var req struct {
		Features map[string]interface{} `json:"features" binding:"required"`
		Version  string                 `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Make prediction
	result, err := h.inferencer.Predict(c.Request.Context(), modelID, req.Features, req.Version)
	if err != nil {
		h.logger.Error("Prediction failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Prediction failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BatchPredict makes batch predictions
func (h *Handler) BatchPredict(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	var req struct {
		Features []map[string]interface{} `json:"features" binding:"required"`
		Version  string                   `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Make batch predictions
	results, err := h.inferencer.BatchPredict(c.Request.Context(), modelID, req.Features, req.Version)
	if err != nil {
		h.logger.Error("Batch prediction failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Batch prediction failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"predictions": results})
}

// GetModelMetrics returns monitoring metrics for a model
func (h *Handler) GetModelMetrics(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	metrics, err := h.monitor.GetMetrics(c.Request.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to get model metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetModelHealth returns health status for a model
func (h *Handler) GetModelHealth(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	health, err := h.monitor.CheckHealth(c.Request.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to check model health", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check model health"})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetMetricsHistory returns historical metrics for a model
func (h *Handler) GetMetricsHistory(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hours parameter"})
		return
	}

	history, err := h.monitor.GetMetricsHistory(c.Request.Context(), modelID, hours)
	if err != nil {
		h.logger.Error("Failed to get metrics history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// GetDriftStatus returns drift detection status for a model
func (h *Handler) GetDriftStatus(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	driftStatus, err := h.monitor.GetDriftStatus(c.Request.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to get drift status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve drift status"})
		return
	}

	c.JSON(http.StatusOK, driftStatus)
}

// TriggerDriftDetection manually triggers drift detection for a model
func (h *Handler) TriggerDriftDetection(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	if err := h.monitor.TriggerDriftDetection(c.Request.Context(), modelID); err != nil {
		h.logger.Error("Failed to trigger drift detection", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger drift detection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Drift detection triggered successfully"})
}

// GetAlerts returns recent alerts for a model
func (h *Handler) GetAlerts(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	alerts, err := h.monitor.GetRecentAlerts(c.Request.Context(), modelID, 100)
	if err != nil {
		h.logger.Error("Failed to get alerts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}