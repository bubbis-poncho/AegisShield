package resolver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/entity-resolution/internal/database"
	"github.com/aegisshield/entity-resolution/internal/matching"
	"github.com/aegisshield/entity-resolution/internal/neo4j"
	"github.com/aegisshield/entity-resolution/internal/standardization"
	"github.com/google/uuid"
)

// EntityResolver orchestrates entity resolution operations
type EntityResolver struct {
	db             *database.Repository
	neo4jClient    *neo4j.Client
	matcher        *matching.Engine
	standardizer   *standardization.Engine
	config         config.Config
	logger         *slog.Logger
}

// ResolutionRequest represents a request to resolve entities
type ResolutionRequest struct {
	EntityType  string                 `json:"entity_type"`
	Name        string                 `json:"name,omitempty"`
	Identifiers map[string]interface{} `json:"identifiers,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	SourceID    string                 `json:"source_id,omitempty"`
}

// ResolutionResult represents the result of entity resolution
type ResolutionResult struct {
	EntityID        string                 `json:"entity_id"`
	IsNewEntity     bool                   `json:"is_new_entity"`
	MatchedEntities []*MatchCandidate      `json:"matched_entities,omitempty"`
	ConfidenceScore float64                `json:"confidence_score"`
	StandardizedData map[string]interface{} `json:"standardized_data"`
	CreatedLinks    []string               `json:"created_links,omitempty"`
}

// MatchCandidate represents a potential entity match
type MatchCandidate struct {
	EntityID        string  `json:"entity_id"`
	MatchScore      float64 `json:"match_score"`
	MatchedFields   []string `json:"matched_fields"`
	ConflictFields  []string `json:"conflict_fields,omitempty"`
	RecommendMerge  bool    `json:"recommend_merge"`
}

// BatchResolutionJob represents a batch processing job
type BatchResolutionJob struct {
	JobID       string              `json:"job_id"`
	Status      string              `json:"status"`
	StartedAt   time.Time           `json:"started_at"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
	Progress    int                 `json:"progress"`
	Total       int                 `json:"total"`
	Results     []*ResolutionResult `json:"results,omitempty"`
	Errors      []string            `json:"errors,omitempty"`
}

// NewEntityResolver creates a new entity resolver
func NewEntityResolver(
	db *database.Repository,
	neo4jClient *neo4j.Client,
	matcher *matching.Engine,
	standardizer *standardization.Engine,
	config config.Config,
	logger *slog.Logger,
) *EntityResolver {
	return &EntityResolver{
		db:           db,
		neo4jClient:  neo4jClient,
		matcher:      matcher,
		standardizer: standardizer,
		config:       config,
		logger:       logger,
	}
}

// ResolveEntity resolves a single entity
func (r *EntityResolver) ResolveEntity(ctx context.Context, request *ResolutionRequest) (*ResolutionResult, error) {
	startTime := time.Now()
	
	r.logger.Info("Starting entity resolution",
		"entity_type", request.EntityType,
		"name", request.Name)

	// Step 1: Standardize the input data
	standardizedData, err := r.standardizeData(request)
	if err != nil {
		return nil, fmt.Errorf("failed to standardize data: %w", err)
	}

	// Step 2: Find potential matches
	candidates, err := r.findMatchCandidates(ctx, request, standardizedData)
	if err != nil {
		return nil, fmt.Errorf("failed to find match candidates: %w", err)
	}

	// Step 3: Evaluate matches and determine resolution
	result, err := r.evaluateMatches(ctx, request, standardizedData, candidates)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate matches: %w", err)
	}

	// Step 4: Create or update entity
	if err := r.persistResolution(ctx, request, result); err != nil {
		return nil, fmt.Errorf("failed to persist resolution: %w", err)
	}

	r.logger.Info("Entity resolution completed",
		"entity_id", result.EntityID,
		"is_new_entity", result.IsNewEntity,
		"confidence_score", result.ConfidenceScore,
		"duration_ms", time.Since(startTime).Milliseconds())

	return result, nil
}

// ResolveBatch processes multiple entities in batch
func (r *EntityResolver) ResolveBatch(ctx context.Context, requests []*ResolutionRequest) (*BatchResolutionJob, error) {
	jobID := uuid.New().String()
	
	job := &BatchResolutionJob{
		JobID:     jobID,
		Status:    "processing",
		StartedAt: time.Now(),
		Total:     len(requests),
		Progress:  0,
		Results:   make([]*ResolutionResult, 0, len(requests)),
		Errors:    make([]string, 0),
	}

	// Store job in database
	dbJob := &database.ResolutionJob{
		ID:          jobID,
		Status:      "processing",
		StartedAt:   job.StartedAt,
		Total:       job.Total,
		Progress:    0,
	}

	if err := r.db.CreateResolutionJob(ctx, dbJob); err != nil {
		return nil, fmt.Errorf("failed to create resolution job: %w", err)
	}

	// Process entities in batches
	batchSize := r.config.EntityResolution.BatchSize
	for i := 0; i < len(requests); i += batchSize {
		end := i + batchSize
		if end > len(requests) {
			end = len(requests)
		}

		batch := requests[i:end]
		batchResults, batchErrors := r.processBatch(ctx, batch)

		job.Results = append(job.Results, batchResults...)
		for _, err := range batchErrors {
			job.Errors = append(job.Errors, err.Error())
		}

		job.Progress = end
		
		// Update job progress
		dbJob.Progress = job.Progress
		if err := r.db.UpdateResolutionJob(ctx, dbJob); err != nil {
			r.logger.Warn("Failed to update job progress", "job_id", jobID, "error", err)
		}
	}

	// Complete job
	now := time.Now()
	job.CompletedAt = &now
	job.Status = "completed"

	dbJob.CompletedAt = &now
	dbJob.Status = "completed"
	if err := r.db.UpdateResolutionJob(ctx, dbJob); err != nil {
		r.logger.Warn("Failed to complete job", "job_id", jobID, "error", err)
	}

	r.logger.Info("Batch resolution completed",
		"job_id", jobID,
		"total", job.Total,
		"successful", len(job.Results),
		"errors", len(job.Errors))

	return job, nil
}

// GetResolutionJob retrieves a resolution job by ID
func (r *EntityResolver) GetResolutionJob(ctx context.Context, jobID string) (*BatchResolutionJob, error) {
	dbJob, err := r.db.GetResolutionJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolution job: %w", err)
	}

	job := &BatchResolutionJob{
		JobID:       dbJob.ID,
		Status:      dbJob.Status,
		StartedAt:   dbJob.StartedAt,
		CompletedAt: dbJob.CompletedAt,
		Progress:    dbJob.Progress,
		Total:       dbJob.Total,
	}

	return job, nil
}

// FindSimilarEntities finds entities similar to the given entity
func (r *EntityResolver) FindSimilarEntities(ctx context.Context, entityID string, threshold float64) ([]*MatchCandidate, error) {
	// Get entity from database
	entity, err := r.db.GetEntity(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Create request from entity
	request := &ResolutionRequest{
		EntityType:  entity.EntityType,
		Name:        entity.Name,
		Identifiers: entity.Identifiers,
		Attributes:  entity.Attributes,
	}

	// Standardize data
	standardizedData, err := r.standardizeData(request)
	if err != nil {
		return nil, fmt.Errorf("failed to standardize data: %w", err)
	}

	// Find candidates
	candidates, err := r.findMatchCandidates(ctx, request, standardizedData)
	if err != nil {
		return nil, fmt.Errorf("failed to find candidates: %w", err)
	}

	// Filter by threshold and exclude original entity
	var matches []*MatchCandidate
	for _, candidate := range candidates {
		if candidate.EntityID != entityID && candidate.MatchScore >= threshold {
			matches = append(matches, candidate)
		}
	}

	return matches, nil
}

// CreateEntityLink creates a link between two entities
func (r *EntityResolver) CreateEntityLink(ctx context.Context, sourceID, targetID, linkType string, properties map[string]interface{}, confidence float64) error {
	// Create database link
	link := &database.EntityLink{
		ID:              uuid.New().String(),
		SourceEntityID:  sourceID,
		TargetEntityID:  targetID,
		LinkType:        linkType,
		Properties:      properties,
		ConfidenceScore: confidence,
		CreatedAt:       time.Now(),
	}

	if err := r.db.CreateEntityLink(ctx, link); err != nil {
		return fmt.Errorf("failed to create entity link: %w", err)
	}

	// Create Neo4j relationship
	relationship := &neo4j.RelationshipEdge{
		ID:              link.ID,
		Type:            linkType,
		SourceEntityID:  sourceID,
		TargetEntityID:  targetID,
		Properties:      properties,
		ConfidenceScore: confidence,
		CreatedAt:       link.CreatedAt,
		UpdatedAt:       link.CreatedAt,
	}

	if err := r.neo4jClient.CreateRelationship(ctx, relationship); err != nil {
		r.logger.Warn("Failed to create Neo4j relationship", "error", err)
	}

	r.logger.Info("Entity link created",
		"link_id", link.ID,
		"source", sourceID,
		"target", targetID,
		"type", linkType)

	return nil
}

// standardizeData standardizes the input data
func (r *EntityResolver) standardizeData(request *ResolutionRequest) (map[string]interface{}, error) {
	standardized := make(map[string]interface{})

	// Standardize name
	if request.Name != "" {
		if standardizedName, err := r.standardizer.StandardizeName(request.Name); err == nil {
			standardized["name"] = standardizedName
		}
	}

	// Standardize identifiers and attributes
	for key, value := range request.Identifiers {
		if str, ok := value.(string); ok {
			switch strings.ToLower(key) {
			case "email":
				if standardizedEmail, err := r.standardizer.StandardizeEmail(str); err == nil {
					standardized[key] = standardizedEmail
				}
			case "phone":
				if standardizedPhone, err := r.standardizer.StandardizePhone(str); err == nil {
					standardized[key] = standardizedPhone
				}
			case "address":
				if standardizedAddress, err := r.standardizer.StandardizeAddress(str); err == nil {
					standardized[key] = standardizedAddress
				}
			default:
				standardized[key] = str
			}
		} else {
			standardized[key] = value
		}
	}

	// Copy other attributes
	for key, value := range request.Attributes {
		if _, exists := standardized[key]; !exists {
			standardized[key] = value
		}
	}

	return standardized, nil
}

// findMatchCandidates finds potential entity matches
func (r *EntityResolver) findMatchCandidates(ctx context.Context, request *ResolutionRequest, standardizedData map[string]interface{}) ([]*MatchCandidate, error) {
	var allCandidates []*MatchCandidate

	// Find exact matches by identifiers
	exactMatches, err := r.findExactMatches(ctx, request)
	if err != nil {
		r.logger.Warn("Failed to find exact matches", "error", err)
	} else {
		allCandidates = append(allCandidates, exactMatches...)
	}

	// Find fuzzy matches by name
	if standardizedName, ok := standardizedData["name"].(string); ok && standardizedName != "" {
		fuzzyMatches, err := r.findFuzzyMatches(ctx, request.EntityType, standardizedName)
		if err != nil {
			r.logger.Warn("Failed to find fuzzy matches", "error", err)
		} else {
			allCandidates = append(allCandidates, fuzzyMatches...)
		}
	}

	// Deduplicate and score candidates
	candidateMap := make(map[string]*MatchCandidate)
	for _, candidate := range allCandidates {
		if existing, exists := candidateMap[candidate.EntityID]; exists {
			// Take the higher score
			if candidate.MatchScore > existing.MatchScore {
				candidateMap[candidate.EntityID] = candidate
			}
		} else {
			candidateMap[candidate.EntityID] = candidate
		}
	}

	// Convert back to slice and sort by score
	var candidates []*MatchCandidate
	for _, candidate := range candidateMap {
		candidates = append(candidates, candidate)
	}

	// Sort by match score descending
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].MatchScore < candidates[j].MatchScore {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	return candidates, nil
}

// findExactMatches finds entities with exact identifier matches
func (r *EntityResolver) findExactMatches(ctx context.Context, request *ResolutionRequest) ([]*MatchCandidate, error) {
	var candidates []*MatchCandidate

	for key, value := range request.Identifiers {
		if str, ok := value.(string); ok && str != "" {
			entities, err := r.db.FindEntitiesByIdentifier(ctx, request.EntityType, key, str)
			if err != nil {
				continue
			}

			for _, entity := range entities {
				candidate := &MatchCandidate{
					EntityID:       entity.ID,
					MatchScore:     1.0, // Exact match
					MatchedFields:  []string{key},
					RecommendMerge: true,
				}
				candidates = append(candidates, candidate)
			}
		}
	}

	return candidates, nil
}

// findFuzzyMatches finds entities with fuzzy name matches
func (r *EntityResolver) findFuzzyMatches(ctx context.Context, entityType, standardizedName string) ([]*MatchCandidate, error) {
	entities, err := r.db.FindEntitiesByFuzzyName(ctx, entityType, standardizedName, r.config.EntityResolution.NameSimilarityThreshold)
	if err != nil {
		return nil, err
	}

	var candidates []*MatchCandidate
	for _, entity := range entities {
		// Calculate detailed match score
		matchResult := r.matcher.CalculateEntitySimilarity(
			map[string]interface{}{"name": standardizedName},
			map[string]interface{}{"name": entity.StandardizedName},
		)

		if matchResult.OverallScore >= r.config.EntityResolution.NameSimilarityThreshold {
			candidate := &MatchCandidate{
				EntityID:       entity.ID,
				MatchScore:     matchResult.OverallScore,
				MatchedFields:  []string{"name"},
				RecommendMerge: matchResult.OverallScore >= r.config.EntityResolution.AutoMergeThreshold,
			}
			candidates = append(candidates, candidate)
		}
	}

	return candidates, nil
}

// evaluateMatches evaluates match candidates and determines resolution
func (r *EntityResolver) evaluateMatches(ctx context.Context, request *ResolutionRequest, standardizedData map[string]interface{}, candidates []*MatchCandidate) (*ResolutionResult, error) {
	result := &ResolutionResult{
		MatchedEntities:  candidates,
		StandardizedData: standardizedData,
	}

	// If no strong matches, create new entity
	if len(candidates) == 0 || (len(candidates) > 0 && candidates[0].MatchScore < r.config.EntityResolution.AutoMergeThreshold) {
		result.EntityID = uuid.New().String()
		result.IsNewEntity = true
		result.ConfidenceScore = 1.0
		return result, nil
	}

	// If strong match exists, use existing entity
	bestMatch := candidates[0]
	if bestMatch.MatchScore >= r.config.EntityResolution.AutoMergeThreshold {
		result.EntityID = bestMatch.EntityID
		result.IsNewEntity = false
		result.ConfidenceScore = bestMatch.MatchScore
		return result, nil
	}

	// Ambiguous case - for now, create new entity with low confidence
	result.EntityID = uuid.New().String()
	result.IsNewEntity = true
	result.ConfidenceScore = 0.5
	return result, nil
}

// persistResolution persists the resolution result
func (r *EntityResolver) persistResolution(ctx context.Context, request *ResolutionRequest, result *ResolutionResult) error {
	now := time.Now()

	if result.IsNewEntity {
		// Create new entity
		entity := &database.Entity{
			ID:               result.EntityID,
			EntityType:       request.EntityType,
			Name:             request.Name,
			StandardizedName: getStringFromMap(result.StandardizedData, "name"),
			Identifiers:      request.Identifiers,
			Attributes:       request.Attributes,
			ConfidenceScore:  result.ConfidenceScore,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		if err := r.db.CreateEntity(ctx, entity); err != nil {
			return fmt.Errorf("failed to create entity: %w", err)
		}

		// Create Neo4j node
		neo4jEntity := &neo4j.EntityNode{
			ID:               entity.ID,
			EntityType:       entity.EntityType,
			Name:             entity.Name,
			StandardizedName: entity.StandardizedName,
			Identifiers:      entity.Identifiers,
			Attributes:       entity.Attributes,
			ConfidenceScore:  entity.ConfidenceScore,
			CreatedAt:        entity.CreatedAt,
			UpdatedAt:        entity.UpdatedAt,
		}

		if err := r.neo4jClient.CreateEntity(ctx, neo4jEntity); err != nil {
			r.logger.Warn("Failed to create Neo4j entity", "error", err)
		}
	} else {
		// Update existing entity with new data
		entity, err := r.db.GetEntity(ctx, result.EntityID)
		if err != nil {
			return fmt.Errorf("failed to get existing entity: %w", err)
		}

		// Merge data
		mergedIdentifiers := mergeMap(entity.Identifiers, request.Identifiers)
		mergedAttributes := mergeMap(entity.Attributes, request.Attributes)

		entity.Identifiers = mergedIdentifiers
		entity.Attributes = mergedAttributes
		entity.UpdatedAt = now

		if err := r.db.UpdateEntity(ctx, entity); err != nil {
			return fmt.Errorf("failed to update entity: %w", err)
		}

		// Update Neo4j node
		neo4jEntity := &neo4j.EntityNode{
			ID:               entity.ID,
			EntityType:       entity.EntityType,
			Name:             entity.Name,
			StandardizedName: entity.StandardizedName,
			Identifiers:      entity.Identifiers,
			Attributes:       entity.Attributes,
			ConfidenceScore:  entity.ConfidenceScore,
			CreatedAt:        entity.CreatedAt,
			UpdatedAt:        entity.UpdatedAt,
		}

		if err := r.neo4jClient.UpdateEntity(ctx, neo4jEntity); err != nil {
			r.logger.Warn("Failed to update Neo4j entity", "error", err)
		}
	}

	return nil
}

// processBatch processes a batch of resolution requests
func (r *EntityResolver) processBatch(ctx context.Context, requests []*ResolutionRequest) ([]*ResolutionResult, []error) {
	var results []*ResolutionResult
	var errors []error

	for _, request := range requests {
		result, err := r.ResolveEntity(ctx, request)
		if err != nil {
			errors = append(errors, err)
		} else {
			results = append(results, result)
		}
	}

	return results, errors
}

// Helper functions

func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func mergeMap(existing, new map[string]interface{}) map[string]interface{} {
	if existing == nil {
		existing = make(map[string]interface{})
	}
	
	for key, value := range new {
		existing[key] = value
	}
	
	return existing
}