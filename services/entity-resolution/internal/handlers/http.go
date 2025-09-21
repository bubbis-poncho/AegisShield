package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/entity-resolution/internal/resolver"
	"github.com/gorilla/mux"
)

// HTTPHandler handles HTTP requests for entity resolution
type HTTPHandler struct {
	resolver *resolver.EntityResolver
	config   config.Config
	logger   *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(
	resolver *resolver.EntityResolver,
	config config.Config,
	logger *slog.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		resolver: resolver,
		config:   config,
		logger:   logger,
	}
}

// RegisterRoutes registers HTTP routes
func (h *HTTPHandler) RegisterRoutes(router *mux.Router) {
	// Entity resolution endpoints
	router.HandleFunc("/api/v1/entities/resolve", h.ResolveEntity).Methods("POST")
	router.HandleFunc("/api/v1/entities/resolve/batch", h.ResolveBatch).Methods("POST")
	router.HandleFunc("/api/v1/entities/{id}/similar", h.FindSimilarEntities).Methods("GET")
	
	// Entity link endpoints
	router.HandleFunc("/api/v1/entities/links", h.CreateEntityLink).Methods("POST")
	
	// Job management endpoints
	router.HandleFunc("/api/v1/jobs/{id}", h.GetResolutionJob).Methods("GET")
	
	// Health and status endpoints
	router.HandleFunc("/api/v1/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/api/v1/status", h.GetServiceStatus).Methods("GET")
	
	// Metrics endpoint (if needed)
	router.HandleFunc("/api/v1/metrics", h.GetMetrics).Methods("GET")
}

// ResolveEntity handles single entity resolution
func (h *HTTPHandler) ResolveEntity(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received ResolveEntity request", "remote_addr", r.RemoteAddr)

	var request resolver.ResolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if request.EntityType == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "entity_type is required", nil)
		return
	}

	// Resolve entity
	result, err := h.resolver.ResolveEntity(r.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to resolve entity", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to resolve entity", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, result)
	
	h.logger.Info("Entity resolved successfully",
		"entity_id", result.EntityID,
		"is_new_entity", result.IsNewEntity,
		"confidence_score", result.ConfidenceScore)
}

// ResolveBatch handles batch entity resolution
func (h *HTTPHandler) ResolveBatch(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received ResolveBatch request", "remote_addr", r.RemoteAddr)

	var requests []*resolver.ResolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate batch size
	if len(requests) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one entity is required", nil)
		return
	}

	if len(requests) > h.config.EntityResolution.MaxBatchSize {
		h.writeErrorResponse(w, http.StatusBadRequest, 
			fmt.Sprintf("Batch size exceeds maximum of %d", h.config.EntityResolution.MaxBatchSize), nil)
		return
	}

	// Validate each request
	for i, req := range requests {
		if req.EntityType == "" {
			h.writeErrorResponse(w, http.StatusBadRequest, 
				fmt.Sprintf("entity_type is required for request %d", i), nil)
			return
		}
	}

	// Process batch
	job, err := h.resolver.ResolveBatch(r.Context(), requests)
	if err != nil {
		h.logger.Error("Failed to process batch", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to process batch", err)
		return
	}

	h.writeJSONResponse(w, http.StatusAccepted, job)
	
	h.logger.Info("Batch processing initiated",
		"job_id", job.JobID,
		"total", job.Total)
}

// FindSimilarEntities finds entities similar to the given entity
func (h *HTTPHandler) FindSimilarEntities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["id"]
	
	h.logger.Info("Received FindSimilarEntities request",
		"entity_id", entityID,
		"remote_addr", r.RemoteAddr)

	if entityID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "entity_id is required", nil)
		return
	}

	// Get threshold from query parameters
	threshold := h.config.EntityResolution.NameSimilarityThreshold
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil && parsed > 0 && parsed <= 1 {
			threshold = parsed
		}
	}

	// Find similar entities
	matches, err := h.resolver.FindSimilarEntities(r.Context(), entityID, threshold)
	if err != nil {
		h.logger.Error("Failed to find similar entities", "entity_id", entityID, "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to find similar entities", err)
		return
	}

	response := map[string]interface{}{
		"entity_id":        entityID,
		"threshold":        threshold,
		"similar_entities": matches,
		"count":           len(matches),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
	
	h.logger.Info("Found similar entities",
		"entity_id", entityID,
		"count", len(matches),
		"threshold", threshold)
}

// CreateEntityLink creates a link between two entities
func (h *HTTPHandler) CreateEntityLink(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received CreateEntityLink request", "remote_addr", r.RemoteAddr)

	var request struct {
		SourceEntityID string                 `json:"source_entity_id"`
		TargetEntityID string                 `json:"target_entity_id"`
		LinkType       string                 `json:"link_type"`
		Properties     map[string]interface{} `json:"properties,omitempty"`
		Confidence     float64                `json:"confidence,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if request.SourceEntityID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "source_entity_id is required", nil)
		return
	}
	if request.TargetEntityID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "target_entity_id is required", nil)
		return
	}
	if request.LinkType == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "link_type is required", nil)
		return
	}

	// Set default confidence if not provided
	if request.Confidence <= 0 {
		request.Confidence = 1.0
	}

	// Create entity link
	err := h.resolver.CreateEntityLink(
		r.Context(),
		request.SourceEntityID,
		request.TargetEntityID,
		request.LinkType,
		request.Properties,
		request.Confidence,
	)

	if err != nil {
		h.logger.Error("Failed to create entity link", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create entity link", err)
		return
	}

	response := map[string]interface{}{
		"success":          true,
		"source_entity_id": request.SourceEntityID,
		"target_entity_id": request.TargetEntityID,
		"link_type":        request.LinkType,
		"confidence":       request.Confidence,
	}

	h.writeJSONResponse(w, http.StatusCreated, response)
	
	h.logger.Info("Entity link created successfully",
		"source_entity_id", request.SourceEntityID,
		"target_entity_id", request.TargetEntityID,
		"link_type", request.LinkType)
}

// GetResolutionJob retrieves the status of a resolution job
func (h *HTTPHandler) GetResolutionJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	h.logger.Info("Received GetResolutionJob request",
		"job_id", jobID,
		"remote_addr", r.RemoteAddr)

	if jobID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "job_id is required", nil)
		return
	}

	// Get job from resolver
	job, err := h.resolver.GetResolutionJob(r.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to get resolution job", "job_id", jobID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, "Resolution job not found", err)
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get resolution job", err)
		}
		return
	}

	h.writeJSONResponse(w, http.StatusOK, job)
}

// HealthCheck performs a health check
func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "entity-resolution",
		"version":   "1.0.0",
		"timestamp": r.Context().Value("timestamp"),
	}

	h.writeJSONResponse(w, http.StatusOK, health)
}

// GetServiceStatus returns detailed service status
func (h *HTTPHandler) GetServiceStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service": "entity-resolution",
		"version": "1.0.0",
		"status":  "running",
		"config": map[string]interface{}{
			"batch_size":                h.config.EntityResolution.BatchSize,
			"max_batch_size":           h.config.EntityResolution.MaxBatchSize,
			"name_similarity_threshold": h.config.EntityResolution.NameSimilarityThreshold,
			"auto_merge_threshold":     h.config.EntityResolution.AutoMergeThreshold,
		},
		"components": map[string]interface{}{
			"database":    "connected",
			"neo4j":       "connected",
			"kafka":       "connected",
			"standardizer": "active",
			"matcher":     "active",
		},
	}

	h.writeJSONResponse(w, http.StatusOK, status)
}

// GetMetrics returns service metrics
func (h *HTTPHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// This would typically integrate with Prometheus metrics
	metrics := map[string]interface{}{
		"entities_resolved_total": 0,
		"batch_jobs_total":       0,
		"entity_links_created":   0,
		"average_confidence":     0.0,
		"processing_time_avg":    0.0,
	}

	h.writeJSONResponse(w, http.StatusOK, metrics)
}

// Helper methods

func (h *HTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *HTTPHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := map[string]interface{}{
		"error":   message,
		"status":  statusCode,
	}

	if err != nil && h.config.Server.Debug {
		response["details"] = err.Error()
	}

	h.writeJSONResponse(w, statusCode, response)
}

// Middleware

// LoggingMiddleware logs HTTP requests
func (h *HTTPHandler) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.Context().Value("start_time")
		
		h.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())

		next.ServeHTTP(w, r)

		if start != nil {
			h.logger.Info("HTTP request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", start)
		}
	})
}

// CORSMiddleware handles CORS headers
func (h *HTTPHandler) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates authentication (placeholder)
func (h *HTTPHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks
		if r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		// TODO: Implement actual authentication
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware implements rate limiting (placeholder)
func (h *HTTPHandler) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement rate limiting
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}