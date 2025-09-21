package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/engine"
)

// HTTPHandlers contains HTTP request handlers
type HTTPHandlers struct {
	engine *engine.GraphEngine
	config config.Config
	logger *slog.Logger
}

// NewHTTPHandlers creates new HTTP handlers
func NewHTTPHandlers(
	engine *engine.GraphEngine,
	config config.Config,
	logger *slog.Logger,
) *HTTPHandlers {
	return &HTTPHandlers{
		engine: engine,
		config: config,
		logger: logger,
	}
}

// RegisterRoutes registers HTTP routes
func (h *HTTPHandlers) RegisterRoutes(router *mux.Router) {
	// Analysis endpoints
	router.HandleFunc("/api/v1/analysis/subgraph", h.analyzeSubGraph).Methods("POST")
	router.HandleFunc("/api/v1/analysis/paths", h.findPaths).Methods("POST")
	router.HandleFunc("/api/v1/analysis/metrics", h.calculateMetrics).Methods("POST")
	router.HandleFunc("/api/v1/analysis/jobs/{jobId}", h.getAnalysisJob).Methods("GET")
	router.HandleFunc("/api/v1/analysis/jobs", h.listAnalysisJobs).Methods("GET")

	// Investigation endpoints
	router.HandleFunc("/api/v1/investigations", h.createInvestigation).Methods("POST")
	router.HandleFunc("/api/v1/investigations/{id}", h.getInvestigation).Methods("GET")
	router.HandleFunc("/api/v1/investigations/{id}", h.updateInvestigation).Methods("PUT")
	router.HandleFunc("/api/v1/investigations", h.listInvestigations).Methods("GET")

	// Entity endpoints
	router.HandleFunc("/api/v1/entities/{id}/neighborhood", h.getEntityNeighborhood).Methods("GET")
	router.HandleFunc("/api/v1/entities/{id}/metrics", h.getEntityMetrics).Methods("GET")

	// Pattern endpoints
	router.HandleFunc("/api/v1/patterns", h.listPatterns).Methods("GET")
	router.HandleFunc("/api/v1/patterns/{id}", h.getPattern).Methods("GET")

	// Health check
	router.HandleFunc("/health", h.healthCheck).Methods("GET")
	router.HandleFunc("/ready", h.readinessCheck).Methods("GET")
}

// analyzeSubGraph handles subgraph analysis requests
func (h *HTTPHandlers) analyzeSubGraph(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeSubGraphRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if len(req.EntityIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "entity_ids is required", nil)
		return
	}

	if req.AnalysisType == "" {
		h.writeError(w, http.StatusBadRequest, "analysis_type is required", nil)
		return
	}

	// Convert to engine request
	analysisReq := &engine.AnalysisRequest{
		Type:      req.AnalysisType,
		EntityIDs: req.EntityIDs,
		Options: engine.AnalysisOptions{
			MaxDepth:           req.Options.MaxDepth,
			MaxPathLength:      req.Options.MaxPathLength,
			MinConfidence:      req.Options.MinConfidence,
			IncludePatterns:    req.Options.IncludePatterns,
			IncludeMetrics:     req.Options.IncludeMetrics,
			IncludeCommunities: req.Options.IncludeCommunities,
		},
		RequestedBy: req.RequestedBy,
		Parameters:  req.Parameters,
	}

	// Perform analysis
	result, err := h.engine.AnalyzeSubGraph(r.Context(), analysisReq)
	if err != nil {
		h.logger.Error("Failed to analyze subgraph", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to analyze subgraph", err)
		return
	}

	// Convert result
	response := &AnalyzeSubGraphResponse{
		JobID:       result.JobID,
		Status:      result.Status,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
		Metadata:    result.Metadata,
	}

	if result.SubGraph != nil {
		response.SubGraph = convertSubGraphFromEngine(result.SubGraph)
	}

	response.Paths = convertPathsFromEngine(result.Paths)
	response.Patterns = convertPatternsFromEngine(result.Patterns)
	response.Communities = convertCommunitiesFromEngine(result.Communities)
	response.Insights = convertInsightsFromEngine(result.Insights)

	h.writeJSON(w, http.StatusOK, response)
}

// findPaths handles path finding requests
func (h *HTTPHandlers) findPaths(w http.ResponseWriter, r *http.Request) {
	var req FindPathsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if len(req.SourceIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "source_ids is required", nil)
		return
	}
	if len(req.TargetIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "target_ids is required", nil)
		return
	}

	// Convert to engine request
	pathReq := &engine.PathRequest{
		SourceIDs:   req.SourceIDs,
		TargetIDs:   req.TargetIDs,
		MaxLength:   req.MaxLength,
		Algorithm:   req.Algorithm,
		WeightField: req.WeightField,
	}

	// Find paths
	paths, err := h.engine.FindPaths(r.Context(), pathReq)
	if err != nil {
		h.logger.Error("Failed to find paths", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to find paths", err)
		return
	}

	response := &FindPathsResponse{
		Paths: convertPathsFromEngine(paths),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// calculateMetrics handles network metrics calculation
func (h *HTTPHandlers) calculateMetrics(w http.ResponseWriter, r *http.Request) {
	var req CalculateMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.EntityIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "entity_ids is required", nil)
		return
	}

	// Calculate metrics
	metrics, err := h.engine.CalculateNetworkMetrics(r.Context(), req.EntityIDs)
	if err != nil {
		h.logger.Error("Failed to calculate metrics", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to calculate metrics", err)
		return
	}

	response := &CalculateMetricsResponse{
		Metrics: convertMetricsFromEngine(metrics),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// getAnalysisJob retrieves an analysis job
func (h *HTTPHandlers) getAnalysisJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	if jobID == "" {
		h.writeError(w, http.StatusBadRequest, "job_id is required", nil)
		return
	}

	job, err := h.engine.GetAnalysisJob(r.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to get analysis job", "job_id", jobID, "error", err)
		h.writeError(w, http.StatusNotFound, "Analysis job not found", err)
		return
	}

	response := convertAnalysisJobFromEngine(job)
	h.writeJSON(w, http.StatusOK, response)
}

// listAnalysisJobs lists analysis jobs with pagination
func (h *HTTPHandlers) listAnalysisJobs(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.getPaginationParams(r)
	status := r.URL.Query().Get("status")

	jobs, total, err := h.engine.ListAnalysisJobs(r.Context(), limit, offset, status)
	if err != nil {
		h.logger.Error("Failed to list analysis jobs", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to list analysis jobs", err)
		return
	}

	response := &ListAnalysisJobsResponse{
		Jobs:   convertAnalysisJobsFromEngine(jobs),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// createInvestigation creates a new investigation
func (h *HTTPHandlers) createInvestigation(w http.ResponseWriter, r *http.Request) {
	var req CreateInvestigationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "name is required", nil)
		return
	}
	if len(req.EntityIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "entity_ids is required", nil)
		return
	}

	// Convert to engine request
	investigationReq := &engine.InvestigationRequest{
		Name:        req.Name,
		Description: req.Description,
		EntityIDs:   req.EntityIDs,
		Priority:    req.Priority,
		CreatedBy:   req.CreatedBy,
		AssignedTo:  req.AssignedTo,
		Parameters:  req.Parameters,
	}

	// Create investigation
	investigation, err := h.engine.CreateInvestigation(r.Context(), investigationReq)
	if err != nil {
		h.logger.Error("Failed to create investigation", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to create investigation", err)
		return
	}

	response := convertInvestigationFromEngine(investigation)
	h.writeJSON(w, http.StatusCreated, response)
}

// getInvestigation retrieves an investigation
func (h *HTTPHandlers) getInvestigation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	investigationID := vars["id"]

	if investigationID == "" {
		h.writeError(w, http.StatusBadRequest, "investigation_id is required", nil)
		return
	}

	investigation, err := h.engine.GetInvestigation(r.Context(), investigationID)
	if err != nil {
		h.logger.Error("Failed to get investigation", "investigation_id", investigationID, "error", err)
		h.writeError(w, http.StatusNotFound, "Investigation not found", err)
		return
	}

	response := convertInvestigationFromEngine(investigation)
	h.writeJSON(w, http.StatusOK, response)
}

// updateInvestigation updates an investigation
func (h *HTTPHandlers) updateInvestigation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	investigationID := vars["id"]

	if investigationID == "" {
		h.writeError(w, http.StatusBadRequest, "investigation_id is required", nil)
		return
	}

	var req UpdateInvestigationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update investigation
	err := h.engine.UpdateInvestigation(r.Context(), investigationID, &engine.InvestigationUpdate{
		Status:      req.Status,
		Priority:    req.Priority,
		AssignedTo:  req.AssignedTo,
		Description: req.Description,
		UpdatedBy:   req.UpdatedBy,
	})
	if err != nil {
		h.logger.Error("Failed to update investigation", "investigation_id", investigationID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to update investigation", err)
		return
	}

	// Return updated investigation
	investigation, err := h.engine.GetInvestigation(r.Context(), investigationID)
	if err != nil {
		h.logger.Error("Failed to get updated investigation", "investigation_id", investigationID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to get updated investigation", err)
		return
	}

	response := convertInvestigationFromEngine(investigation)
	h.writeJSON(w, http.StatusOK, response)
}

// listInvestigations lists investigations with pagination
func (h *HTTPHandlers) listInvestigations(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.getPaginationParams(r)
	status := r.URL.Query().Get("status")
	assignedTo := r.URL.Query().Get("assigned_to")

	investigations, total, err := h.engine.ListInvestigations(r.Context(), limit, offset, status, assignedTo)
	if err != nil {
		h.logger.Error("Failed to list investigations", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to list investigations", err)
		return
	}

	response := &ListInvestigationsResponse{
		Investigations: convertInvestigationsFromEngine(investigations),
		Total:          total,
		Limit:          limit,
		Offset:         offset,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// getEntityNeighborhood gets entity neighborhood
func (h *HTTPHandlers) getEntityNeighborhood(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["id"]

	if entityID == "" {
		h.writeError(w, http.StatusBadRequest, "entity_id is required", nil)
		return
	}

	// Parse relationship types filter
	relationshipTypes := []string{}
	if types := r.URL.Query().Get("relationship_types"); types != "" {
		relationshipTypes = strings.Split(types, ",")
	}

	subGraph, err := h.engine.GetEntityNeighborhood(r.Context(), entityID, relationshipTypes)
	if err != nil {
		h.logger.Error("Failed to get entity neighborhood", "entity_id", entityID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to get entity neighborhood", err)
		return
	}

	response := &GetEntityNeighborhoodResponse{
		EntityID: entityID,
		SubGraph: convertSubGraphFromEngine(subGraph),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// getEntityMetrics gets entity metrics
func (h *HTTPHandlers) getEntityMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["id"]

	if entityID == "" {
		h.writeError(w, http.StatusBadRequest, "entity_id is required", nil)
		return
	}

	metrics, err := h.engine.CalculateNetworkMetrics(r.Context(), []string{entityID})
	if err != nil {
		h.logger.Error("Failed to get entity metrics", "entity_id", entityID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to get entity metrics", err)
		return
	}

	if len(metrics) == 0 {
		h.writeError(w, http.StatusNotFound, "Entity metrics not found", nil)
		return
	}

	response := convertNetworkMetricFromEngine(metrics[0])
	h.writeJSON(w, http.StatusOK, response)
}

// listPatterns lists detected patterns
func (h *HTTPHandlers) listPatterns(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.getPaginationParams(r)
	patternType := r.URL.Query().Get("pattern_type")
	minConfidence := parseFloat(r.URL.Query().Get("min_confidence"), 0.0)

	patterns, total, err := h.engine.ListPatterns(r.Context(), limit, offset, patternType, minConfidence)
	if err != nil {
		h.logger.Error("Failed to list patterns", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to list patterns", err)
		return
	}

	response := &ListPatternsResponse{
		Patterns: convertPatternsFromEngine(patterns),
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// getPattern retrieves a specific pattern
func (h *HTTPHandlers) getPattern(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patternID := vars["id"]

	if patternID == "" {
		h.writeError(w, http.StatusBadRequest, "pattern_id is required", nil)
		return
	}

	pattern, err := h.engine.GetPattern(r.Context(), patternID)
	if err != nil {
		h.logger.Error("Failed to get pattern", "pattern_id", patternID, "error", err)
		h.writeError(w, http.StatusNotFound, "Pattern not found", err)
		return
	}

	response := convertPatternFromEngine(pattern)
	h.writeJSON(w, http.StatusOK, response)
}

// healthCheck returns service health status
func (h *HTTPHandlers) healthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "graph-engine",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// readinessCheck returns service readiness status
func (h *HTTPHandlers) readinessCheck(w http.ResponseWriter, r *http.Request) {
	// Check if engine is ready
	if !h.engine.IsReady() {
		h.writeError(w, http.StatusServiceUnavailable, "Service not ready", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ready",
		"service": "graph-engine",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Helper methods

// getPaginationParams extracts pagination parameters from request
func (h *HTTPHandlers) getPaginationParams(r *http.Request) (limit, offset int) {
	limit = parseInt(r.URL.Query().Get("limit"), 50)  // Default limit
	offset = parseInt(r.URL.Query().Get("offset"), 0) // Default offset

	// Enforce maximum limit
	if limit > 1000 {
		limit = 1000
	}

	return limit, offset
}

// parseInt parses integer with default value
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultValue
}

// parseFloat parses float with default value
func parseFloat(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return defaultValue
}

// writeJSON writes JSON response
func (h *HTTPHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// writeError writes error response
func (h *HTTPHandlers) writeError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil && h.config.Debug {
		response["details"] = err.Error()
	}

	h.writeJSON(w, status, response)
}

// Conversion functions

// convertSubGraphFromEngine converts engine SubGraph to HTTP response format
func convertSubGraphFromEngine(sg *engine.SubGraph) *SubGraph {
	if sg == nil {
		return nil
	}

	return &SubGraph{
		Entities:      convertEntitiesFromEngine(sg.Entities),
		Relationships: convertRelationshipsFromEngine(sg.Relationships),
		Metadata:      sg.Metadata,
	}
}

// convertEntitiesFromEngine converts engine entities
func convertEntitiesFromEngine(entities []*engine.Entity) []*Entity {
	var result []*Entity
	for _, entity := range entities {
		result = append(result, &Entity{
			ID:         entity.ID,
			Type:       entity.Type,
			Properties: entity.Properties,
		})
	}
	return result
}

// convertRelationshipsFromEngine converts engine relationships
func convertRelationshipsFromEngine(relationships []*engine.Relationship) []*Relationship {
	var result []*Relationship
	for _, rel := range relationships {
		result = append(result, &Relationship{
			ID:         rel.ID,
			Type:       rel.Type,
			SourceID:   rel.SourceID,
			TargetID:   rel.TargetID,
			Properties: rel.Properties,
		})
	}
	return result
}

// convertPathsFromEngine converts engine paths
func convertPathsFromEngine(paths []*engine.Path) []*Path {
	var result []*Path
	for _, path := range paths {
		result = append(result, &Path{
			StartEntity:   convertEntityFromEngine(path.StartEntity),
			EndEntity:     convertEntityFromEngine(path.EndEntity),
			Entities:      convertEntitiesFromEngine(path.Entities),
			Relationships: convertRelationshipsFromEngine(path.Relationships),
			Length:        path.Length,
			Cost:          path.Cost,
		})
	}
	return result
}

// convertEntityFromEngine converts single entity
func convertEntityFromEngine(entity *engine.Entity) *Entity {
	if entity == nil {
		return nil
	}
	return &Entity{
		ID:         entity.ID,
		Type:       entity.Type,
		Properties: entity.Properties,
	}
}

// Additional conversion functions would continue here...
// (Implementing all conversion functions to maintain consistency)
