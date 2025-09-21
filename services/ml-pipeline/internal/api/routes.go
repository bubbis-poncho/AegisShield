package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"../config"
	"../database"
	"../monitoring"
	"../training"
	"../inference"
)

// Router sets up the API routes
func SetupRouter(
	cfg *config.Config,
	logger *zap.Logger,
	repos *database.Repositories,
	monitor *monitoring.ModelMonitor,
	trainer *training.TrainingEngine,
	inferencer *inference.InferenceEngine,
) *gin.Engine {
	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware(logger))

	// Create handler
	handler := NewHandler(cfg, logger, repos, monitor, trainer, inferencer)

	// Health check
	router.GET("/health", handler.Health)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Model routes
		models := v1.Group("/models")
		{
			models.GET("", handler.GetModels)
			models.POST("", handler.CreateModel)
			models.GET("/:id", handler.GetModel)
			
			// Training routes
			models.POST("/:id/train", handler.TrainModel)
			models.GET("/:id/training-jobs", handler.GetTrainingJobs)
			models.GET("/:id/training-jobs/:job_id", handler.GetTrainingJob)
			
			// Deployment routes
			models.POST("/:id/deploy", handler.DeployModel)
			models.GET("/:id/deployments", handler.GetDeployments)
			
			// Prediction routes
			models.POST("/:id/predict", handler.Predict)
			models.POST("/:id/batch-predict", handler.BatchPredict)
			
			// Monitoring routes
			models.GET("/:id/metrics", handler.GetModelMetrics)
			models.GET("/:id/health", handler.GetModelHealth)
			models.GET("/:id/metrics/history", handler.GetMetricsHistory)
			models.GET("/:id/drift", handler.GetDriftStatus)
			models.POST("/:id/drift/trigger", handler.TriggerDriftDetection)
			models.GET("/:id/alerts", handler.GetAlerts)
		}

		// Training job routes
		training := v1.Group("/training")
		{
			training.GET("/:job_id", handler.GetTrainingJob)
		}

		// System-wide monitoring routes
		monitoring := v1.Group("/monitoring")
		{
			monitoring.GET("/metrics", handler.GetSystemMetrics)
			monitoring.GET("/alerts", handler.GetSystemAlerts)
			monitoring.GET("/health", handler.GetSystemHealth)
		}
	}

	return router
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a simple request ID (in production, use a proper UUID library)
			requestID = generateRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Log request
		latency := time.Since(start)
		requestID, _ := c.Get("request_id")
		
		logger.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("request_id", requestID.(string)),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	// In production, use a proper UUID library like github.com/google/uuid
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// Additional system-wide handlers

// GetSystemMetrics returns system-wide metrics
func (h *Handler) GetSystemMetrics(c *gin.Context) {
	// Get metrics for all models
	models, err := h.repos.ModelRepo.GetAll(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get models for system metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve system metrics"})
		return
	}

	var modelIDs []string
	for _, model := range models {
		modelIDs = append(modelIDs, model.ID)
	}

	aggregated, err := h.monitor.GetAggregatedMetrics(c.Request.Context(), modelIDs)
	if err != nil {
		h.logger.Error("Failed to get aggregated metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve aggregated metrics"})
		return
	}

	c.JSON(http.StatusOK, aggregated)
}

// GetSystemAlerts returns system-wide alerts
func (h *Handler) GetSystemAlerts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	alerts, err := h.monitor.GetSystemAlerts(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get system alerts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve system alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// GetSystemHealth returns overall system health
func (h *Handler) GetSystemHealth(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"services": map[string]interface{}{
			"database": h.checkDatabaseHealth(),
			"trainer":  h.checkTrainerHealth(),
			"inference": h.checkInferenceHealth(),
			"monitor":  h.checkMonitorHealth(),
		},
	}

	// Determine overall status
	allHealthy := true
	for _, service := range health["services"].(map[string]interface{}) {
		if service.(map[string]interface{})["status"] != "healthy" {
			allHealthy = false
			break
		}
	}

	if !allHealthy {
		health["status"] = "degraded"
	}

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// Helper methods for health checks

func (h *Handler) checkDatabaseHealth() map[string]interface{} {
	// Simple database health check
	ctx := context.Background()
	_, err := h.repos.ModelRepo.GetAll(ctx)
	
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	}
	
	return map[string]interface{}{
		"status": "healthy",
	}
}

func (h *Handler) checkTrainerHealth() map[string]interface{} {
	// Check trainer health
	healthy := h.trainer.IsHealthy()
	
	if !healthy {
		return map[string]interface{}{
			"status": "unhealthy",
		}
	}
	
	return map[string]interface{}{
		"status": "healthy",
	}
}

func (h *Handler) checkInferenceHealth() map[string]interface{} {
	// Check inference engine health
	healthy := h.inferencer.IsHealthy()
	
	if !healthy {
		return map[string]interface{}{
			"status": "unhealthy",
		}
	}
	
	return map[string]interface{}{
		"status": "healthy",
	}
}

func (h *Handler) checkMonitorHealth() map[string]interface{} {
	// Check monitor health
	healthy := h.monitor.IsHealthy()
	
	if !healthy {
		return map[string]interface{}{
			"status": "unhealthy",
		}
	}
	
	return map[string]interface{}{
		"status": "healthy",
	}
}