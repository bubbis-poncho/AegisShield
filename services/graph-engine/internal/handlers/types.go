package handlers

import (
	"time"

	"github.com/aegisshield/graph-engine/internal/engine"
)

// Request types

// AnalyzeSubGraphRequest represents a subgraph analysis request
type AnalyzeSubGraphRequest struct {
	AnalysisType string                 `json:"analysis_type"`
	EntityIDs    []string               `json:"entity_ids"`
	Options      AnalysisOptions        `json:"options"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestedBy  string                 `json:"requested_by"`
}

// AnalysisOptions contains analysis configuration options
type AnalysisOptions struct {
	MaxDepth           int     `json:"max_depth,omitempty"`
	MaxPathLength      int     `json:"max_path_length,omitempty"`
	MinConfidence      float64 `json:"min_confidence,omitempty"`
	IncludePatterns    bool    `json:"include_patterns,omitempty"`
	IncludeMetrics     bool    `json:"include_metrics,omitempty"`
	IncludeCommunities bool    `json:"include_communities,omitempty"`
}

// FindPathsRequest represents a path finding request
type FindPathsRequest struct {
	SourceIDs   []string `json:"source_ids"`
	TargetIDs   []string `json:"target_ids"`
	MaxLength   int      `json:"max_length,omitempty"`
	Algorithm   string   `json:"algorithm,omitempty"`
	WeightField string   `json:"weight_field,omitempty"`
}

// CalculateMetricsRequest represents a metrics calculation request
type CalculateMetricsRequest struct {
	EntityIDs []string `json:"entity_ids"`
}

// CreateInvestigationRequest represents an investigation creation request
type CreateInvestigationRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	EntityIDs   []string               `json:"entity_ids"`
	Priority    string                 `json:"priority,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// UpdateInvestigationRequest represents an investigation update request
type UpdateInvestigationRequest struct {
	Status      string `json:"status,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssignedTo  string `json:"assigned_to,omitempty"`
	Description string `json:"description,omitempty"`
	UpdatedBy   string `json:"updated_by"`
}

// Response types

// AnalyzeSubGraphResponse represents a subgraph analysis response
type AnalyzeSubGraphResponse struct {
	JobID       string                 `json:"job_id"`
	Status      string                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	SubGraph    *SubGraph              `json:"subgraph,omitempty"`
	Paths       []*Path                `json:"paths,omitempty"`
	Patterns    []*PatternMatch        `json:"patterns,omitempty"`
	Communities []*Community           `json:"communities,omitempty"`
	Insights    []*AnalysisInsight     `json:"insights,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FindPathsResponse represents a path finding response
type FindPathsResponse struct {
	Paths []*Path `json:"paths"`
}

// CalculateMetricsResponse represents a metrics calculation response
type CalculateMetricsResponse struct {
	Metrics []*NetworkMetrics `json:"metrics"`
}

// ListAnalysisJobsResponse represents analysis jobs list response
type ListAnalysisJobsResponse struct {
	Jobs   []*AnalysisJob `json:"jobs"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// ListInvestigationsResponse represents investigations list response
type ListInvestigationsResponse struct {
	Investigations []*Investigation `json:"investigations"`
	Total          int              `json:"total"`
	Limit          int              `json:"limit"`
	Offset         int              `json:"offset"`
}

// GetEntityNeighborhoodResponse represents entity neighborhood response
type GetEntityNeighborhoodResponse struct {
	EntityID string    `json:"entity_id"`
	SubGraph *SubGraph `json:"subgraph"`
}

// ListPatternsResponse represents patterns list response
type ListPatternsResponse struct {
	Patterns []*PatternMatch `json:"patterns"`
	Total    int             `json:"total"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
}

// Data types

// SubGraph represents a graph substructure
type SubGraph struct {
	Entities      []*Entity              `json:"entities"`
	Relationships []*Relationship        `json:"relationships"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Entity represents a graph entity
type Entity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Relationship represents a graph relationship
type Relationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Path represents a path between entities
type Path struct {
	StartEntity   *Entity         `json:"start_entity"`
	EndEntity     *Entity         `json:"end_entity"`
	Entities      []*Entity       `json:"entities"`
	Relationships []*Relationship `json:"relationships"`
	Length        int             `json:"length"`
	Cost          float64         `json:"cost"`
}

// PatternMatch represents a detected pattern
type PatternMatch struct {
	ID            string                 `json:"id,omitempty"`
	PatternType   string                 `json:"pattern_type"`
	Entities      []*Entity              `json:"entities"`
	Relationships []*Relationship        `json:"relationships"`
	Confidence    float64                `json:"confidence"`
	Severity      string                 `json:"severity,omitempty"`
	Description   string                 `json:"description,omitempty"`
	DetectedAt    time.Time              `json:"detected_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Community represents a detected community
type Community struct {
	ID         string    `json:"id"`
	Entities   []string  `json:"entities"`
	Size       int       `json:"size"`
	Density    float64   `json:"density"`
	Modularity float64   `json:"modularity"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// AnalysisInsight represents an analysis insight
type AnalysisInsight struct {
	ID          string                 `json:"id,omitempty"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Severity    string                 `json:"severity"`
	EntityIDs   []string               `json:"entity_ids"`
	Evidence    map[string]interface{} `json:"evidence"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
}

// NetworkMetrics represents network analysis metrics
type NetworkMetrics struct {
	EntityID              string                 `json:"entity_id"`
	DegreeCentrality      float64                `json:"degree_centrality"`
	BetweennessCentrality float64                `json:"betweenness_centrality"`
	ClosenessCentrality   float64                `json:"closeness_centrality"`
	EigenvectorCentrality float64                `json:"eigenvector_centrality"`
	PageRank              float64                `json:"page_rank"`
	ClusteringCoefficient float64                `json:"clustering_coefficient"`
	CommunityID           string                 `json:"community_id,omitempty"`
	CalculatedAt          time.Time              `json:"calculated_at"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// AnalysisJob represents an analysis job
type AnalysisJob struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Progress    int                    `json:"progress"`
	Total       int                    `json:"total"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Results     map[string]interface{} `json:"results,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
}

// Investigation represents an investigation case
type Investigation struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Status      string                 `json:"status"`
	Priority    string                 `json:"priority"`
	Entities    []string               `json:"entities"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Conversion functions

// convertPatternsFromEngine converts engine patterns to HTTP response format
func convertPatternsFromEngine(patterns []*engine.PatternMatch) []*PatternMatch {
	var result []*PatternMatch
	for _, pattern := range patterns {
		result = append(result, convertPatternFromEngine(pattern))
	}
	return result
}

// convertPatternFromEngine converts single engine pattern
func convertPatternFromEngine(pattern *engine.PatternMatch) *PatternMatch {
	if pattern == nil {
		return nil
	}

	return &PatternMatch{
		ID:            pattern.ID,
		PatternType:   pattern.PatternType,
		Entities:      convertEntitiesFromEngine(pattern.Entities),
		Relationships: convertRelationshipsFromEngine(pattern.Relationships),
		Confidence:    pattern.Confidence,
		Severity:      pattern.Severity,
		Description:   pattern.Description,
		DetectedAt:    pattern.DetectedAt,
		Metadata:      pattern.Metadata,
	}
}

// convertCommunitiesFromEngine converts engine communities
func convertCommunitiesFromEngine(communities []*engine.Community) []*Community {
	var result []*Community
	for _, community := range communities {
		result = append(result, &Community{
			ID:         community.ID,
			Entities:   community.Entities,
			Size:       community.Size,
			Density:    community.Density,
			Modularity: community.Modularity,
			CreatedAt:  community.CreatedAt,
		})
	}
	return result
}

// convertInsightsFromEngine converts engine insights
func convertInsightsFromEngine(insights []*engine.AnalysisInsight) []*AnalysisInsight {
	var result []*AnalysisInsight
	for _, insight := range insights {
		result = append(result, &AnalysisInsight{
			ID:          insight.ID,
			Type:        insight.Type,
			Title:       insight.Title,
			Description: insight.Description,
			Confidence:  insight.Confidence,
			Severity:    insight.Severity,
			EntityIDs:   insight.EntityIDs,
			Evidence:    insight.Evidence,
			CreatedAt:   insight.CreatedAt,
		})
	}
	return result
}

// convertMetricsFromEngine converts engine metrics
func convertMetricsFromEngine(metrics []*engine.NetworkMetrics) []*NetworkMetrics {
	var result []*NetworkMetrics
	for _, metric := range metrics {
		result = append(result, convertNetworkMetricFromEngine(metric))
	}
	return result
}

// convertNetworkMetricFromEngine converts single network metric
func convertNetworkMetricFromEngine(metric *engine.NetworkMetrics) *NetworkMetrics {
	if metric == nil {
		return nil
	}

	return &NetworkMetrics{
		EntityID:              metric.EntityID,
		DegreeCentrality:      metric.DegreeCentrality,
		BetweennessCentrality: metric.BetweennessCentrality,
		ClosenessCentrality:   metric.ClosenessCentrality,
		EigenvectorCentrality: metric.EigenvectorCentrality,
		PageRank:              metric.PageRank,
		ClusteringCoefficient: metric.ClusteringCoeff,
		CommunityID:           metric.CommunityID,
		CalculatedAt:          metric.CalculatedAt,
		Metadata:              metric.Metadata,
	}
}

// convertAnalysisJobFromEngine converts engine analysis job
func convertAnalysisJobFromEngine(job *engine.AnalysisJob) *AnalysisJob {
	if job == nil {
		return nil
	}

	return &AnalysisJob{
		ID:          job.ID,
		Type:        job.Type,
		Status:      job.Status,
		Progress:    job.Progress,
		Total:       job.Total,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		Error:       job.Error,
		Parameters:  job.Parameters,
		Results:     job.Results,
		CreatedBy:   job.CreatedBy,
	}
}

// convertAnalysisJobsFromEngine converts multiple engine analysis jobs
func convertAnalysisJobsFromEngine(jobs []*engine.AnalysisJob) []*AnalysisJob {
	var result []*AnalysisJob
	for _, job := range jobs {
		result = append(result, convertAnalysisJobFromEngine(job))
	}
	return result
}

// convertInvestigationFromEngine converts engine investigation
func convertInvestigationFromEngine(investigation *engine.Investigation) *Investigation {
	if investigation == nil {
		return nil
	}

	return &Investigation{
		ID:          investigation.ID,
		Name:        investigation.Name,
		Description: investigation.Description,
		Status:      investigation.Status,
		Priority:    investigation.Priority,
		Entities:    investigation.Entities,
		CreatedAt:   investigation.CreatedAt,
		UpdatedAt:   investigation.UpdatedAt,
		CreatedBy:   investigation.CreatedBy,
		AssignedTo:  investigation.AssignedTo,
		Metadata:    investigation.Metadata,
	}
}

// convertInvestigationsFromEngine converts multiple engine investigations
func convertInvestigationsFromEngine(investigations []*engine.Investigation) []*Investigation {
	var result []*Investigation
	for _, investigation := range investigations {
		result = append(result, convertInvestigationFromEngine(investigation))
	}
	return result
}