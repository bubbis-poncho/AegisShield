package engine

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/database"
	"github.com/aegisshield/graph-engine/internal/kafka"
	"github.com/aegisshield/graph-engine/internal/metrics"
	"github.com/aegisshield/graph-engine/internal/neo4j"
	"github.com/google/uuid"
)

// GraphEngine orchestrates graph analysis operations
type GraphEngine struct {
	db          *database.Repository
	neo4jClient *neo4j.Client
	producer    *kafka.Producer
	config      config.Config
	metrics     *metrics.Collector
	logger      *slog.Logger
	
	// Analysis management
	activeAnalyses sync.Map
	analysisSemaphore chan struct{}
}

// AnalysisRequest represents a graph analysis request
type AnalysisRequest struct {
	Type        string                 `json:"type"`
	EntityIDs   []string               `json:"entity_ids"`
	Parameters  map[string]interface{} `json:"parameters"`
	Options     AnalysisOptions        `json:"options"`
	RequestedBy string                 `json:"requested_by,omitempty"`
}

// AnalysisOptions provides options for analysis
type AnalysisOptions struct {
	MaxDepth         int     `json:"max_depth,omitempty"`
	MaxPathLength    int     `json:"max_path_length,omitempty"`
	MinConfidence    float64 `json:"min_confidence,omitempty"`
	IncludePatterns  bool    `json:"include_patterns,omitempty"`
	IncludeMetrics   bool    `json:"include_metrics,omitempty"`
	IncludeCommunities bool  `json:"include_communities,omitempty"`
}

// AnalysisResult represents the result of graph analysis
type AnalysisResult struct {
	JobID         string                  `json:"job_id"`
	Type          string                  `json:"type"`
	Status        string                  `json:"status"`
	SubGraph      *neo4j.SubGraph         `json:"subgraph,omitempty"`
	Paths         []*neo4j.Path           `json:"paths,omitempty"`
	Patterns      []*neo4j.PatternMatch   `json:"patterns,omitempty"`
	Communities   []*neo4j.Community      `json:"communities,omitempty"`
	Centrality    []*neo4j.CentralityMetrics `json:"centrality,omitempty"`
	NetworkMetrics []*database.NetworkMetrics `json:"network_metrics,omitempty"`
	Insights      []AnalysisInsight       `json:"insights,omitempty"`
	Metadata      map[string]interface{}  `json:"metadata"`
	StartedAt     time.Time               `json:"started_at"`
	CompletedAt   *time.Time              `json:"completed_at,omitempty"`
	Duration      time.Duration           `json:"duration,omitempty"`
}

// AnalysisInsight represents an insight discovered during analysis
type AnalysisInsight struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	EntityIDs   []string               `json:"entity_ids"`
	Evidence    map[string]interface{} `json:"evidence"`
	Severity    string                 `json:"severity"`
}

// InvestigationRequest represents an investigation request
type InvestigationRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	EntityIDs   []string               `json:"entity_ids"`
	Priority    string                 `json:"priority"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
}

// PathRequest represents a path finding request
type PathRequest struct {
	SourceIDs    []string `json:"source_ids"`
	TargetIDs    []string `json:"target_ids"`
	MaxLength    int      `json:"max_length,omitempty"`
	Algorithm    string   `json:"algorithm,omitempty"` // "shortest", "all_simple", "weighted"
	WeightField  string   `json:"weight_field,omitempty"`
}

// NewGraphEngine creates a new graph engine
func NewGraphEngine(
	db *database.Repository,
	neo4jClient *neo4j.Client,
	producer *kafka.Producer,
	config config.Config,
	metrics *metrics.Collector,
	logger *slog.Logger,
) *GraphEngine {
	return &GraphEngine{
		db:          db,
		neo4jClient: neo4jClient,
		producer:    producer,
		config:      config,
		metrics:     metrics,
		logger:      logger,
		analysisSemaphore: make(chan struct{}, config.GraphEngine.MaxConcurrentAnalyses),
	}
}

// AnalyzeSubGraph performs comprehensive subgraph analysis
func (e *GraphEngine) AnalyzeSubGraph(ctx context.Context, request *AnalysisRequest) (*AnalysisResult, error) {
	// Acquire analysis semaphore
	select {
	case e.analysisSemaphore <- struct{}{}:
		defer func() { <-e.analysisSemaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	jobID := uuid.New().String()
	startTime := time.Now()

	e.logger.Info("Starting subgraph analysis",
		"job_id", jobID,
		"type", request.Type,
		"entity_count", len(request.EntityIDs))

	// Create analysis job record
	job := &database.AnalysisJob{
		ID:         jobID,
		Type:       request.Type,
		Status:     "processing",
		Parameters: request.Parameters,
		StartedAt:  startTime,
		CreatedBy:  request.RequestedBy,
	}

	if err := e.db.CreateAnalysisJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create analysis job: %w", err)
	}

	// Track active analysis
	e.activeAnalyses.Store(jobID, job)
	defer e.activeAnalyses.Delete(jobID)

	result := &AnalysisResult{
		JobID:     jobID,
		Type:      request.Type,
		Status:    "processing",
		StartedAt: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// Set default options
	options := request.Options
	if options.MaxDepth == 0 {
		options.MaxDepth = e.config.GraphEngine.MaxTraversalDepth
	}
	if options.MaxPathLength == 0 {
		options.MaxPathLength = e.config.GraphEngine.MaxPathLength
	}
	if options.MinConfidence == 0 {
		options.MinConfidence = e.config.GraphEngine.MinPathConfidence
	}

	// Get subgraph
	subGraph, err := e.neo4jClient.GetSubGraph(ctx, request.EntityIDs, options.MaxDepth)
	if err != nil {
		e.updateJobStatus(ctx, jobID, "failed", fmt.Sprintf("Failed to get subgraph: %v", err))
		return nil, fmt.Errorf("failed to get subgraph: %w", err)
	}
	result.SubGraph = subGraph

	// Perform additional analyses based on options
	if options.IncludeMetrics {
		centrality, err := e.calculateCentralityMetrics(ctx, request.EntityIDs)
		if err != nil {
			e.logger.Warn("Failed to calculate centrality metrics", "error", err)
		} else {
			result.Centrality = centrality
		}
	}

	if options.IncludePatterns {
		patterns, err := e.detectPatterns(ctx, request.EntityIDs)
		if err != nil {
			e.logger.Warn("Failed to detect patterns", "error", err)
		} else {
			result.Patterns = patterns
		}
	}

	if options.IncludeCommunities {
		communities, err := e.detectCommunities(ctx, request.EntityIDs)
		if err != nil {
			e.logger.Warn("Failed to detect communities", "error", err)
		} else {
			result.Communities = communities
		}
	}

	// Generate insights
	insights := e.generateInsights(ctx, result)
	result.Insights = insights

	// Complete analysis
	completedAt := time.Now()
	result.CompletedAt = &completedAt
	result.Duration = completedAt.Sub(startTime)
	result.Status = "completed"

	// Update job status
	e.updateJobStatus(ctx, jobID, "completed", "")

	// Publish analysis event
	if err := e.producer.PublishAnalysisCompleted(ctx, result); err != nil {
		e.logger.Warn("Failed to publish analysis event", "error", err)
	}

	// Update metrics
	e.metrics.RecordAnalysisCompleted(request.Type, result.Duration, len(result.Insights))

	e.logger.Info("Subgraph analysis completed",
		"job_id", jobID,
		"duration_ms", result.Duration.Milliseconds(),
		"entities", len(result.SubGraph.Entities),
		"relationships", len(result.SubGraph.Relationships),
		"insights", len(result.Insights))

	return result, nil
}

// FindPaths finds paths between entities
func (e *GraphEngine) FindPaths(ctx context.Context, request *PathRequest) ([]*neo4j.Path, error) {
	e.logger.Info("Finding paths",
		"sources", len(request.SourceIDs),
		"targets", len(request.TargetIDs),
		"max_length", request.MaxLength)

	maxLength := request.MaxLength
	if maxLength == 0 {
		maxLength = e.config.GraphEngine.MaxPathLength
	}

	timer := e.metrics.NewTimer()
	defer func() {
		e.metrics.RecordPathFindingDuration(timer.Duration())
	}()

	paths, err := e.neo4jClient.FindShortestPaths(ctx, request.SourceIDs, request.TargetIDs, maxLength)
	if err != nil {
		e.metrics.RecordPathFindingError()
		return nil, fmt.Errorf("failed to find paths: %w", err)
	}

	// Filter paths by confidence if specified
	if e.config.GraphEngine.MinPathConfidence > 0 {
		var filteredPaths []*neo4j.Path
		for _, path := range paths {
			if e.calculatePathConfidence(path) >= e.config.GraphEngine.MinPathConfidence {
				filteredPaths = append(filteredPaths, path)
			}
		}
		paths = filteredPaths
	}

	e.logger.Info("Paths found",
		"count", len(paths),
		"duration_ms", timer.Duration().Milliseconds())

	return paths, nil
}

// CreateInvestigation creates a new investigation
func (e *GraphEngine) CreateInvestigation(ctx context.Context, request *InvestigationRequest) (*database.Investigation, error) {
	investigation := &database.Investigation{
		ID:          uuid.New().String(),
		Name:        request.Name,
		Description: request.Description,
		Status:      "active",
		Priority:    request.Priority,
		Entities:    request.EntityIDs,
		Metadata:    request.Parameters,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   request.CreatedBy,
		AssignedTo:  request.AssignedTo,
	}

	if err := e.db.CreateInvestigation(ctx, investigation); err != nil {
		return nil, fmt.Errorf("failed to create investigation: %w", err)
	}

	// Publish investigation event
	if err := e.producer.PublishInvestigationCreated(ctx, investigation); err != nil {
		e.logger.Warn("Failed to publish investigation event", "error", err)
	}

	e.logger.Info("Investigation created",
		"investigation_id", investigation.ID,
		"name", investigation.Name,
		"entity_count", len(investigation.Entities))

	return investigation, nil
}

// GetAnalysisJob retrieves an analysis job
func (e *GraphEngine) GetAnalysisJob(ctx context.Context, jobID string) (*database.AnalysisJob, error) {
	return e.db.GetAnalysisJob(ctx, jobID)
}

// GetInvestigation retrieves an investigation
func (e *GraphEngine) GetInvestigation(ctx context.Context, investigationID string) (*database.Investigation, error) {
	return e.db.GetInvestigation(ctx, investigationID)
}

// GetEntityNeighborhood gets the immediate neighborhood of an entity
func (e *GraphEngine) GetEntityNeighborhood(ctx context.Context, entityID string, relationshipTypes []string) (*neo4j.SubGraph, error) {
	return e.neo4jClient.GetEntityNeighborhood(ctx, entityID, relationshipTypes)
}

// CalculateNetworkMetrics calculates comprehensive network metrics
func (e *GraphEngine) CalculateNetworkMetrics(ctx context.Context, entityIDs []string) ([]*database.NetworkMetrics, error) {
	timer := e.metrics.NewTimer()
	defer func() {
		e.metrics.RecordMetricsCalculationDuration(timer.Duration())
	}()

	// Get centrality metrics from Neo4j
	centralityMetrics, err := e.neo4jClient.CalculateCentralityMetrics(ctx, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate centrality metrics: %w", err)
	}

	// Convert to database format and store
	var networkMetrics []*database.NetworkMetrics
	for _, metric := range centralityMetrics {
		dbMetric := &database.NetworkMetrics{
			ID:                    uuid.New().String(),
			EntityID:              metric.EntityID,
			DegreeCentrality:      metric.DegreeCentrality,
			BetweennessCentrality: metric.BetweennessCentrality,
			ClosenessCentrality:   metric.ClosenessCentrality,
			EigenvectorCentrality: metric.EigenvectorCentrality,
			PageRank:              metric.PageRank,
			ClusteringCoeff:       0.0, // Would calculate actual clustering coefficient
			CalculatedAt:          time.Now(),
			UpdatedAt:             time.Now(),
		}

		if err := e.db.CreateNetworkMetrics(ctx, dbMetric); err != nil {
			e.logger.Warn("Failed to store network metrics", "entity_id", metric.EntityID, "error", err)
		} else {
			networkMetrics = append(networkMetrics, dbMetric)
		}
	}

	return networkMetrics, nil
}

// Private helper methods

func (e *GraphEngine) calculateCentralityMetrics(ctx context.Context, entityIDs []string) ([]*neo4j.CentralityMetrics, error) {
	return e.neo4jClient.CalculateCentralityMetrics(ctx, entityIDs)
}

func (e *GraphEngine) detectPatterns(ctx context.Context, entityIDs []string) ([]*neo4j.PatternMatch, error) {
	var allPatterns []*neo4j.PatternMatch

	// Detect different pattern types
	patternTypes := []string{"triangle", "star", "chain"}
	
	for _, patternType := range patternTypes {
		patterns, err := e.neo4jClient.FindPatterns(ctx, patternType, entityIDs)
		if err != nil {
			e.logger.Warn("Failed to detect pattern", "type", patternType, "error", err)
			continue
		}
		allPatterns = append(allPatterns, patterns...)
	}

	return allPatterns, nil
}

func (e *GraphEngine) detectCommunities(ctx context.Context, entityIDs []string) ([]*neo4j.Community, error) {
	return e.neo4jClient.DetectCommunities(ctx, entityIDs)
}

func (e *GraphEngine) generateInsights(ctx context.Context, result *AnalysisResult) []AnalysisInsight {
	var insights []AnalysisInsight

	// Analyze centrality for high-influence entities
	if result.Centrality != nil {
		for _, metric := range result.Centrality {
			if metric.DegreeCentrality > e.config.GraphEngine.CentralityThreshold {
				insights = append(insights, AnalysisInsight{
					Type:        "high_centrality",
					Title:       "High Centrality Entity",
					Description: fmt.Sprintf("Entity %s has high degree centrality (%.2f)", metric.EntityID, metric.DegreeCentrality),
					Confidence:  0.9,
					EntityIDs:   []string{metric.EntityID},
					Evidence: map[string]interface{}{
						"degree_centrality": metric.DegreeCentrality,
					},
					Severity: "medium",
				})
			}
		}
	}

	// Analyze patterns for suspicious activity
	if result.Patterns != nil {
		for _, pattern := range result.Patterns {
			if pattern.Confidence > e.config.GraphEngine.AnomalyThreshold {
				var entityIDs []string
				for _, entity := range pattern.Entities {
					entityIDs = append(entityIDs, entity.ID)
				}

				insights = append(insights, AnalysisInsight{
					Type:        "suspicious_pattern",
					Title:       fmt.Sprintf("Suspicious %s Pattern", pattern.PatternType),
					Description: fmt.Sprintf("Detected %s pattern with high confidence", pattern.PatternType),
					Confidence:  pattern.Confidence,
					EntityIDs:   entityIDs,
					Evidence: map[string]interface{}{
						"pattern_type": pattern.PatternType,
						"confidence":   pattern.Confidence,
					},
					Severity: "high",
				})
			}
		}
	}

	// Analyze communities for clustering
	if result.Communities != nil {
		for _, community := range result.Communities {
			if community.Size > 5 && community.Density > e.config.GraphEngine.ClusteringThreshold {
				insights = append(insights, AnalysisInsight{
					Type:        "dense_cluster",
					Title:       "Dense Entity Cluster",
					Description: fmt.Sprintf("Found dense cluster with %d entities", community.Size),
					Confidence:  0.8,
					EntityIDs:   community.Entities,
					Evidence: map[string]interface{}{
						"cluster_size": community.Size,
						"density":      community.Density,
					},
					Severity: "medium",
				})
			}
		}
	}

	return insights
}

func (e *GraphEngine) calculatePathConfidence(path *neo4j.Path) float64 {
	if path.Length == 0 {
		return 1.0
	}

	// Simple confidence calculation based on path length and relationship weights
	confidence := 1.0 / math.Pow(float64(path.Length), 0.5)
	
	// Adjust based on relationship properties if available
	for _, rel := range path.Relationships {
		if conf, exists := rel.Properties["confidence"]; exists {
			if confFloat, ok := conf.(float64); ok {
				confidence *= confFloat
			}
		}
	}

	return confidence
}

func (e *GraphEngine) updateJobStatus(ctx context.Context, jobID, status, errorMsg string) {
	job := &database.AnalysisJob{
		ID:     jobID,
		Status: status,
		Error:  errorMsg,
	}

	if status == "completed" || status == "failed" {
		now := time.Now()
		job.CompletedAt = &now
	}

	if err := e.db.UpdateAnalysisJob(ctx, job); err != nil {
		e.logger.Warn("Failed to update job status", "job_id", jobID, "error", err)
	}
}