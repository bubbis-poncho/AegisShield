package resolution

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/neo4j"
	"github.com/google/uuid"
)

// EntityResolver performs entity resolution and relationship inference
type EntityResolver struct {
	neo4jClient *neo4j.Client
	config      config.GraphEngineConfig
	logger      *slog.Logger
}

// ResolutionRequest represents an entity resolution request
type ResolutionRequest struct {
	Entities           []*CandidateEntity     `json:"entities"`
	ResolutionStrategy ResolutionStrategy     `json:"resolution_strategy"`
	SimilarityThreshold float64               `json:"similarity_threshold"`
	MaxCandidates      int                    `json:"max_candidates"`
	FieldWeights       map[string]float64     `json:"field_weights,omitempty"`
	Parameters         map[string]interface{} `json:"parameters,omitempty"`
}

// CandidateEntity represents an entity candidate for resolution
type CandidateEntity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
	Source     string                 `json:"source,omitempty"`
}

// ResolutionStrategy represents different entity resolution strategies
type ResolutionStrategy string

const (
	StrategyExactMatch     ResolutionStrategy = "exact_match"
	StrategyFuzzyMatch     ResolutionStrategy = "fuzzy_match"
	StrategyMLSimilarity   ResolutionStrategy = "ml_similarity"
	StrategyHybrid         ResolutionStrategy = "hybrid"
	StrategyBehavioral     ResolutionStrategy = "behavioral"
)

// ResolutionResult contains entity resolution results
type ResolutionResult struct {
	RequestID      string                  `json:"request_id"`
	Matches        []*EntityMatch          `json:"matches"`
	NewEntities    []*ResolvedEntity       `json:"new_entities"`
	MergedEntities []*MergedEntity         `json:"merged_entities"`
	Statistics     *ResolutionStatistics   `json:"statistics"`
	ProcessingTime time.Duration           `json:"processing_time"`
}

// EntityMatch represents a potential entity match
type EntityMatch struct {
	CandidateID     string                 `json:"candidate_id"`
	MatchedEntityID string                 `json:"matched_entity_id"`
	Confidence      float64                `json:"confidence"`
	SimilarityScore float64                `json:"similarity_score"`
	MatchType       MatchType              `json:"match_type"`
	MatchingFields  []FieldMatch           `json:"matching_fields"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// MatchType represents the type of match found
type MatchType string

const (
	MatchTypeExact      MatchType = "exact"
	MatchTypeFuzzy      MatchType = "fuzzy"
	MatchTypeProbable   MatchType = "probable"
	MatchTypePossible   MatchType = "possible"
	MatchTypeBehavioral MatchType = "behavioral"
)

// FieldMatch represents a field-level match
type FieldMatch struct {
	FieldName      string  `json:"field_name"`
	CandidateValue string  `json:"candidate_value"`
	MatchedValue   string  `json:"matched_value"`
	Similarity     float64 `json:"similarity"`
	Weight         float64 `json:"weight"`
}

// ResolvedEntity represents a resolved entity
type ResolvedEntity struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Attributes      map[string]interface{} `json:"attributes"`
	Sources         []string               `json:"sources"`
	Confidence      float64                `json:"confidence"`
	CreatedAt       time.Time              `json:"created_at"`
}

// MergedEntity represents entities that were merged
type MergedEntity struct {
	ResultEntityID  string    `json:"result_entity_id"`
	MergedEntityIDs []string  `json:"merged_entity_ids"`
	MergeReason     string    `json:"merge_reason"`
	MergedAt        time.Time `json:"merged_at"`
}

// ResolutionStatistics contains statistics about the resolution process
type ResolutionStatistics struct {
	TotalCandidates    int     `json:"total_candidates"`
	ExactMatches       int     `json:"exact_matches"`
	FuzzyMatches       int     `json:"fuzzy_matches"`
	NewEntities        int     `json:"new_entities"`
	MergedEntities     int     `json:"merged_entities"`
	AverageConfidence  float64 `json:"average_confidence"`
	ProcessingTimeMs   int64   `json:"processing_time_ms"`
}

// RelationshipInferenceRequest represents a relationship inference request
type RelationshipInferenceRequest struct {
	EntityIDs          []string               `json:"entity_ids"`
	InferenceStrategy  InferenceStrategy      `json:"inference_strategy"`
	MinConfidence      float64                `json:"min_confidence"`
	MaxDepth           int                    `json:"max_depth"`
	RelationshipTypes  []string               `json:"relationship_types,omitempty"`
	Parameters         map[string]interface{} `json:"parameters,omitempty"`
}

// InferenceStrategy represents different relationship inference strategies
type InferenceStrategy string

const (
	InferenceStrategyTransactional InferenceStrategy = "transactional"
	InferenceStrategyTemporal      InferenceStrategy = "temporal"
	InferenceStrategyBehavioral    InferenceStrategy = "behavioral"
	InferenceStrategyNetwork       InferenceStrategy = "network"
	InferenceStrategyHybrid        InferenceStrategy = "hybrid"
)

// RelationshipInferenceResult contains relationship inference results
type RelationshipInferenceResult struct {
	InferredRelationships []*InferredRelationship `json:"inferred_relationships"`
	Statistics            *InferenceStatistics    `json:"statistics"`
	ProcessingTime        time.Duration           `json:"processing_time"`
}

// InferredRelationship represents an inferred relationship
type InferredRelationship struct {
	ID             string                 `json:"id"`
	SourceEntityID string                 `json:"source_entity_id"`
	TargetEntityID string                 `json:"target_entity_id"`
	Type           string                 `json:"type"`
	Confidence     float64                `json:"confidence"`
	Evidence       []RelationshipEvidence `json:"evidence"`
	InferredAt     time.Time              `json:"inferred_at"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// RelationshipEvidence represents evidence for an inferred relationship
type RelationshipEvidence struct {
	EvidenceType string                 `json:"evidence_type"`
	Description  string                 `json:"description"`
	Strength     float64                `json:"strength"`
	Source       string                 `json:"source"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// InferenceStatistics contains statistics about relationship inference
type InferenceStatistics struct {
	EntitiesAnalyzed       int     `json:"entities_analyzed"`
	RelationshipsInferred  int     `json:"relationships_inferred"`
	HighConfidenceInferred int     `json:"high_confidence_inferred"`
	AverageConfidence      float64 `json:"average_confidence"`
}

// NewEntityResolver creates a new entity resolver
func NewEntityResolver(client *neo4j.Client, config config.GraphEngineConfig, logger *slog.Logger) *EntityResolver {
	return &EntityResolver{
		neo4jClient: client,
		config:      config,
		logger:      logger,
	}
}

// ResolveEntities performs entity resolution on candidate entities
func (er *EntityResolver) ResolveEntities(ctx context.Context, req *ResolutionRequest) (*ResolutionResult, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	er.logger.Info("Starting entity resolution",
		"request_id", requestID,
		"candidates", len(req.Entities),
		"strategy", req.ResolutionStrategy)

	result := &ResolutionResult{
		RequestID:      requestID,
		Matches:        make([]*EntityMatch, 0),
		NewEntities:    make([]*ResolvedEntity, 0),
		MergedEntities: make([]*MergedEntity, 0),
		Statistics:     &ResolutionStatistics{TotalCandidates: len(req.Entities)},
	}

	// Process each candidate entity
	for _, candidate := range req.Entities {
		matches, err := er.findMatches(ctx, candidate, req)
		if err != nil {
			er.logger.Error("Failed to find matches for candidate",
				"candidate_id", candidate.ID,
				"error", err)
			continue
		}

		if len(matches) == 0 {
			// No matches found, create new entity
			newEntity := er.createNewEntity(candidate)
			result.NewEntities = append(result.NewEntities, newEntity)
			result.Statistics.NewEntities++
		} else {
			// Process matches
			bestMatch := er.selectBestMatch(matches, req.SimilarityThreshold)
			if bestMatch != nil {
				result.Matches = append(result.Matches, bestMatch)
				
				switch bestMatch.MatchType {
				case MatchTypeExact:
					result.Statistics.ExactMatches++
				case MatchTypeFuzzy, MatchTypeProbable:
					result.Statistics.FuzzyMatches++
				}
			} else {
				// No match above threshold, create new entity
				newEntity := er.createNewEntity(candidate)
				result.NewEntities = append(result.NewEntities, newEntity)
				result.Statistics.NewEntities++
			}
		}
	}

	// Post-process for entity merging
	mergedEntities := er.identifyMerges(ctx, result.Matches, req)
	result.MergedEntities = mergedEntities
	result.Statistics.MergedEntities = len(mergedEntities)

	// Calculate statistics
	result.ProcessingTime = time.Since(startTime)
	result.Statistics.ProcessingTimeMs = result.ProcessingTime.Milliseconds()
	
	if len(result.Matches) > 0 {
		totalConfidence := 0.0
		for _, match := range result.Matches {
			totalConfidence += match.Confidence
		}
		result.Statistics.AverageConfidence = totalConfidence / float64(len(result.Matches))
	}

	er.logger.Info("Entity resolution completed",
		"request_id", requestID,
		"matches", len(result.Matches),
		"new_entities", len(result.NewEntities),
		"merged_entities", len(result.MergedEntities),
		"processing_time", result.ProcessingTime)

	return result, nil
}

// findMatches finds potential matches for a candidate entity
func (er *EntityResolver) findMatches(ctx context.Context, candidate *CandidateEntity, req *ResolutionRequest) ([]*EntityMatch, error) {
	var matches []*EntityMatch
	var err error

	switch req.ResolutionStrategy {
	case StrategyExactMatch:
		matches, err = er.findExactMatches(ctx, candidate, req)
	case StrategyFuzzyMatch:
		matches, err = er.findFuzzyMatches(ctx, candidate, req)
	case StrategyMLSimilarity:
		matches, err = er.findMLSimilarityMatches(ctx, candidate, req)
	case StrategyHybrid:
		matches, err = er.findHybridMatches(ctx, candidate, req)
	case StrategyBehavioral:
		matches, err = er.findBehavioralMatches(ctx, candidate, req)
	default:
		return nil, fmt.Errorf("unsupported resolution strategy: %s", req.ResolutionStrategy)
	}

	if err != nil {
		return nil, err
	}

	// Sort matches by confidence
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	// Limit results
	if req.MaxCandidates > 0 && len(matches) > req.MaxCandidates {
		matches = matches[:req.MaxCandidates]
	}

	return matches, nil
}

// findExactMatches finds exact matches based on key attributes
func (er *EntityResolver) findExactMatches(ctx context.Context, candidate *CandidateEntity, req *ResolutionRequest) ([]*EntityMatch, error) {
	// Build query based on entity type
	var query string
	var params map[string]interface{}

	switch candidate.Type {
	case "Person":
		query, params = er.buildPersonExactMatchQuery(candidate)
	case "Account":
		query, params = er.buildAccountExactMatchQuery(candidate)
	case "Company":
		query, params = er.buildCompanyExactMatchQuery(candidate)
	default:
		query, params = er.buildGenericExactMatchQuery(candidate)
	}

	records, err := er.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute exact match query: %w", err)
	}

	matches := make([]*EntityMatch, 0)
	for _, record := range records {
		match := er.buildExactMatch(candidate, record)
		if match != nil {
			matches = append(matches, match)
		}
	}

	return matches, nil
}

// findFuzzyMatches finds fuzzy matches using string similarity
func (er *EntityResolver) findFuzzyMatches(ctx context.Context, candidate *CandidateEntity, req *ResolutionRequest) ([]*EntityMatch, error) {
	// Build fuzzy match query with string similarity functions
	query := `
		MATCH (e:` + candidate.Type + `)
		WHERE e.name IS NOT NULL
		WITH e, 
			 apoc.text.levenshteinSimilarity(COALESCE(e.name, ''), $candidateName) as nameSimilarity,
			 apoc.text.jaroWinklerDistance(COALESCE(e.name, ''), $candidateName) as nameJaro
		WHERE nameSimilarity >= $minSimilarity OR nameJaro >= $minSimilarity
		RETURN e.id as entityId, 
			   e.name as entityName,
			   nameSimilarity,
			   nameJaro,
			   e as entity
		ORDER BY GREATEST(nameSimilarity, nameJaro) DESC
		LIMIT $maxResults
	`

	candidateName := ""
	if name, ok := candidate.Attributes["name"].(string); ok {
		candidateName = name
	}

	params := map[string]interface{}{
		"candidateName": candidateName,
		"minSimilarity": 0.7,
		"maxResults":    req.MaxCandidates,
	}

	records, err := er.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute fuzzy match query: %w", err)
	}

	matches := make([]*EntityMatch, 0)
	for _, record := range records {
		match := er.buildFuzzyMatch(candidate, record)
		if match != nil && match.Confidence >= req.SimilarityThreshold {
			matches = append(matches, match)
		}
	}

	return matches, nil
}

// findMLSimilarityMatches uses machine learning for similarity matching
func (er *EntityResolver) findMLSimilarityMatches(ctx context.Context, candidate *CandidateEntity, req *ResolutionRequest) ([]*EntityMatch, error) {
	// This would integrate with ML models for semantic similarity
	// For now, implement a simplified version using attribute similarity
	
	matches := make([]*EntityMatch, 0)
	
	// Get potential candidates based on type
	query := `
		MATCH (e:` + candidate.Type + `)
		RETURN e.id as entityId, e as entity
		LIMIT 1000
	`

	records, err := er.neo4jClient.ExecuteQuery(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get ML similarity candidates: %w", err)
	}

	for _, record := range records {
		similarity := er.calculateMLSimilarity(candidate, record)
		if similarity >= req.SimilarityThreshold {
			match := &EntityMatch{
				CandidateID:     candidate.ID,
				MatchedEntityID: record["entityId"].(string),
				Confidence:      similarity,
				SimilarityScore: similarity,
				MatchType:       MatchTypeProbable,
			}
			matches = append(matches, match)
		}
	}

	return matches, nil
}

// findHybridMatches combines multiple matching strategies
func (er *EntityResolver) findHybridMatches(ctx context.Context, candidate *CandidateEntity, req *ResolutionRequest) ([]*EntityMatch, error) {
	allMatches := make(map[string]*EntityMatch)

	// Try exact matches first
	exactMatches, err := er.findExactMatches(ctx, candidate, req)
	if err == nil {
		for _, match := range exactMatches {
			match.Confidence *= 1.0 // Full weight for exact matches
			allMatches[match.MatchedEntityID] = match
		}
	}

	// Try fuzzy matches
	fuzzyMatches, err := er.findFuzzyMatches(ctx, candidate, req)
	if err == nil {
		for _, match := range fuzzyMatches {
			existing, exists := allMatches[match.MatchedEntityID]
			if exists {
				// Combine scores
				existing.Confidence = math.Max(existing.Confidence, match.Confidence*0.8)
			} else {
				match.Confidence *= 0.8 // Reduced weight for fuzzy matches
				allMatches[match.MatchedEntityID] = match
			}
		}
	}

	// Convert map to slice
	matches := make([]*EntityMatch, 0, len(allMatches))
	for _, match := range allMatches {
		matches = append(matches, match)
	}

	return matches, nil
}

// findBehavioralMatches finds matches based on behavioral patterns
func (er *EntityResolver) findBehavioralMatches(ctx context.Context, candidate *CandidateEntity, req *ResolutionRequest) ([]*EntityMatch, error) {
	// Analyze behavioral patterns like transaction patterns, network connections, etc.
	query := `
		MATCH (candidate:` + candidate.Type + ` {id: $candidateId})
		MATCH (e:` + candidate.Type + `)
		WHERE e.id <> $candidateId
		OPTIONAL MATCH (candidate)-[r1:TRANSACTION]->()
		OPTIONAL MATCH (e)-[r2:TRANSACTION]->()
		WITH candidate, e,
			 COUNT(DISTINCT r1) as candidateTxCount,
			 COUNT(DISTINCT r2) as entityTxCount,
			 AVG(r1.amount) as candidateAvgAmount,
			 AVG(r2.amount) as entityAvgAmount
		WHERE ABS(candidateTxCount - entityTxCount) <= $txCountTolerance
		AND ABS(candidateAvgAmount - entityAvgAmount) <= $amountTolerance
		RETURN e.id as entityId,
			   candidateTxCount,
			   entityTxCount,
			   candidateAvgAmount,
			   entityAvgAmount,
			   ABS(candidateTxCount - entityTxCount) as txCountDiff,
			   ABS(candidateAvgAmount - entityAvgAmount) as amountDiff
		ORDER BY txCountDiff + amountDiff
		LIMIT $maxResults
	`

	params := map[string]interface{}{
		"candidateId":       candidate.ID,
		"txCountTolerance":  10,
		"amountTolerance":   1000.0,
		"maxResults":        req.MaxCandidates,
	}

	records, err := er.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute behavioral match query: %w", err)
	}

	matches := make([]*EntityMatch, 0)
	for _, record := range records {
		match := er.buildBehavioralMatch(candidate, record)
		if match != nil && match.Confidence >= req.SimilarityThreshold {
			matches = append(matches, match)
		}
	}

	return matches, nil
}

// InferRelationships infers relationships between entities
func (er *EntityResolver) InferRelationships(ctx context.Context, req *RelationshipInferenceRequest) (*RelationshipInferenceResult, error) {
	startTime := time.Now()

	er.logger.Info("Starting relationship inference",
		"entities", len(req.EntityIDs),
		"strategy", req.InferenceStrategy,
		"max_depth", req.MaxDepth)

	result := &RelationshipInferenceResult{
		InferredRelationships: make([]*InferredRelationship, 0),
		Statistics: &InferenceStatistics{
			EntitiesAnalyzed: len(req.EntityIDs),
		},
	}

	// Infer relationships based on strategy
	switch req.InferenceStrategy {
	case InferenceStrategyTransactional:
		relationships, err := er.inferTransactionalRelationships(ctx, req)
		if err != nil {
			return nil, err
		}
		result.InferredRelationships = append(result.InferredRelationships, relationships...)
		
	case InferenceStrategyTemporal:
		relationships, err := er.inferTemporalRelationships(ctx, req)
		if err != nil {
			return nil, err
		}
		result.InferredRelationships = append(result.InferredRelationships, relationships...)
		
	case InferenceStrategyBehavioral:
		relationships, err := er.inferBehavioralRelationships(ctx, req)
		if err != nil {
			return nil, err
		}
		result.InferredRelationships = append(result.InferredRelationships, relationships...)
		
	case InferenceStrategyNetwork:
		relationships, err := er.inferNetworkRelationships(ctx, req)
		if err != nil {
			return nil, err
		}
		result.InferredRelationships = append(result.InferredRelationships, relationships...)
		
	case InferenceStrategyHybrid:
		// Combine multiple strategies
		strategies := []InferenceStrategy{
			InferenceStrategyTransactional,
			InferenceStrategyTemporal,
			InferenceStrategyBehavioral,
		}
		
		for _, strategy := range strategies {
			subReq := *req
			subReq.InferenceStrategy = strategy
			relationships, err := er.InferRelationships(ctx, &subReq)
			if err == nil {
				result.InferredRelationships = append(result.InferredRelationships, relationships.InferredRelationships...)
			}
		}
	}

	// Filter by confidence threshold
	filteredRelationships := make([]*InferredRelationship, 0)
	totalConfidence := 0.0
	highConfidenceCount := 0

	for _, rel := range result.InferredRelationships {
		if rel.Confidence >= req.MinConfidence {
			filteredRelationships = append(filteredRelationships, rel)
			totalConfidence += rel.Confidence
			
			if rel.Confidence > 0.8 {
				highConfidenceCount++
			}
		}
	}

	result.InferredRelationships = filteredRelationships
	result.Statistics.RelationshipsInferred = len(filteredRelationships)
	result.Statistics.HighConfidenceInferred = highConfidenceCount

	if len(filteredRelationships) > 0 {
		result.Statistics.AverageConfidence = totalConfidence / float64(len(filteredRelationships))
	}

	result.ProcessingTime = time.Since(startTime)

	er.logger.Info("Relationship inference completed",
		"relationships_inferred", len(result.InferredRelationships),
		"high_confidence", result.Statistics.HighConfidenceInferred,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// Helper methods for building queries and processing results

func (er *EntityResolver) buildPersonExactMatchQuery(candidate *CandidateEntity) (string, map[string]interface{}) {
	query := `
		MATCH (p:Person)
		WHERE ($firstName IS NULL OR p.firstName = $firstName)
		AND ($lastName IS NULL OR p.lastName = $lastName)
		AND ($dateOfBirth IS NULL OR p.dateOfBirth = $dateOfBirth)
		AND ($ssn IS NULL OR p.ssn = $ssn)
		RETURN p.id as entityId, p as entity
		LIMIT 10
	`

	params := map[string]interface{}{
		"firstName":   candidate.Attributes["first_name"],
		"lastName":    candidate.Attributes["last_name"],
		"dateOfBirth": candidate.Attributes["date_of_birth"],
		"ssn":         candidate.Attributes["ssn"],
	}

	return query, params
}

func (er *EntityResolver) buildAccountExactMatchQuery(candidate *CandidateEntity) (string, map[string]interface{}) {
	query := `
		MATCH (a:Account)
		WHERE ($accountNumber IS NULL OR a.accountNumber = $accountNumber)
		AND ($routingNumber IS NULL OR a.routingNumber = $routingNumber)
		AND ($iban IS NULL OR a.iban = $iban)
		RETURN a.id as entityId, a as entity
		LIMIT 10
	`

	params := map[string]interface{}{
		"accountNumber": candidate.Attributes["account_number"],
		"routingNumber": candidate.Attributes["routing_number"],
		"iban":          candidate.Attributes["iban"],
	}

	return query, params
}

func (er *EntityResolver) buildCompanyExactMatchQuery(candidate *CandidateEntity) (string, map[string]interface{}) {
	query := `
		MATCH (c:Company)
		WHERE ($name IS NULL OR c.name = $name)
		AND ($registrationNumber IS NULL OR c.registrationNumber = $registrationNumber)
		AND ($taxId IS NULL OR c.taxId = $taxId)
		RETURN c.id as entityId, c as entity
		LIMIT 10
	`

	params := map[string]interface{}{
		"name":               candidate.Attributes["name"],
		"registrationNumber": candidate.Attributes["registration_number"],
		"taxId":              candidate.Attributes["tax_id"],
	}

	return query, params
}

func (er *EntityResolver) buildGenericExactMatchQuery(candidate *CandidateEntity) (string, map[string]interface{}) {
	query := fmt.Sprintf(`
		MATCH (e:%s)
		WHERE e.id = $candidateId OR e.name = $candidateName
		RETURN e.id as entityId, e as entity
		LIMIT 10
	`, candidate.Type)

	params := map[string]interface{}{
		"candidateId":   candidate.ID,
		"candidateName": candidate.Attributes["name"],
	}

	return query, params
}

// Additional helper methods...

func (er *EntityResolver) buildExactMatch(candidate *CandidateEntity, record map[string]interface{}) *EntityMatch {
	entityID, ok := record["entityId"].(string)
	if !ok {
		return nil
	}

	return &EntityMatch{
		CandidateID:     candidate.ID,
		MatchedEntityID: entityID,
		Confidence:      1.0,
		SimilarityScore: 1.0,
		MatchType:       MatchTypeExact,
		MatchingFields:  []FieldMatch{},
	}
}

func (er *EntityResolver) buildFuzzyMatch(candidate *CandidateEntity, record map[string]interface{}) *EntityMatch {
	entityID, ok := record["entityId"].(string)
	if !ok {
		return nil
	}

	nameSimilarity := getFloat64(record, "nameSimilarity")
	nameJaro := getFloat64(record, "nameJaro")
	
	confidence := math.Max(nameSimilarity, nameJaro)

	return &EntityMatch{
		CandidateID:     candidate.ID,
		MatchedEntityID: entityID,
		Confidence:      confidence,
		SimilarityScore: confidence,
		MatchType:       MatchTypeFuzzy,
		MatchingFields: []FieldMatch{
			{
				FieldName:  "name",
				Similarity: confidence,
				Weight:     1.0,
			},
		},
	}
}

func (er *EntityResolver) buildBehavioralMatch(candidate *CandidateEntity, record map[string]interface{}) *EntityMatch {
	entityID, ok := record["entityId"].(string)
	if !ok {
		return nil
	}

	// Calculate behavioral similarity based on transaction patterns
	txCountDiff := getFloat64(record, "txCountDiff")
	amountDiff := getFloat64(record, "amountDiff")
	
	// Normalize differences to similarity score
	similarity := 1.0 / (1.0 + (txCountDiff + amountDiff)/100.0)

	return &EntityMatch{
		CandidateID:     candidate.ID,
		MatchedEntityID: entityID,
		Confidence:      similarity,
		SimilarityScore: similarity,
		MatchType:       MatchTypeBehavioral,
	}
}

func (er *EntityResolver) calculateMLSimilarity(candidate *CandidateEntity, record map[string]interface{}) float64 {
	// Simplified ML similarity calculation
	// In a real implementation, this would use trained ML models
	
	totalSimilarity := 0.0
	totalWeight := 0.0

	// Compare key attributes
	for key, candidateValue := range candidate.Attributes {
		if candidateStr, ok := candidateValue.(string); ok {
			if entityValue, exists := record[key]; exists {
				if entityStr, ok := entityValue.(string); ok {
					similarity := er.calculateStringSimilarity(candidateStr, entityStr)
					weight := 1.0
					
					totalSimilarity += similarity * weight
					totalWeight += weight
				}
			}
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalSimilarity / totalWeight
}

func (er *EntityResolver) calculateStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	// Simple Jaccard similarity for demonstration
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}

	for _, word := range words2 {
		set2[word] = true
	}

	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (er *EntityResolver) createNewEntity(candidate *CandidateEntity) *ResolvedEntity {
	return &ResolvedEntity{
		ID:         uuid.New().String(),
		Type:       candidate.Type,
		Attributes: candidate.Attributes,
		Sources:    []string{candidate.Source},
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
}

func (er *EntityResolver) selectBestMatch(matches []*EntityMatch, threshold float64) *EntityMatch {
	if len(matches) == 0 {
		return nil
	}

	// Sort by confidence
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	bestMatch := matches[0]
	if bestMatch.Confidence >= threshold {
		return bestMatch
	}

	return nil
}

func (er *EntityResolver) identifyMerges(ctx context.Context, matches []*EntityMatch, req *ResolutionRequest) []*MergedEntity {
	// Identify entities that should be merged based on multiple high-confidence matches
	mergedEntities := make([]*MergedEntity, 0)

	// Group matches by matched entity ID
	entityMatches := make(map[string][]*EntityMatch)
	for _, match := range matches {
		entityMatches[match.MatchedEntityID] = append(entityMatches[match.MatchedEntityID], match)
	}

	// Look for entities with multiple high-confidence matches
	for entityID, matchList := range entityMatches {
		if len(matchList) > 1 {
			highConfidenceCount := 0
			candidateIDs := make([]string, 0)

			for _, match := range matchList {
				if match.Confidence > 0.9 {
					highConfidenceCount++
					candidateIDs = append(candidateIDs, match.CandidateID)
				}
			}

			if highConfidenceCount > 1 {
				mergedEntity := &MergedEntity{
					ResultEntityID:  entityID,
					MergedEntityIDs: candidateIDs,
					MergeReason:     "Multiple high-confidence matches",
					MergedAt:        time.Now(),
				}
				mergedEntities = append(mergedEntities, mergedEntity)
			}
		}
	}

	return mergedEntities
}

// Placeholder methods for relationship inference strategies

func (er *EntityResolver) inferTransactionalRelationships(ctx context.Context, req *RelationshipInferenceRequest) ([]*InferredRelationship, error) {
	// Implementation for transactional relationship inference
	return []*InferredRelationship{}, nil
}

func (er *EntityResolver) inferTemporalRelationships(ctx context.Context, req *RelationshipInferenceRequest) ([]*InferredRelationship, error) {
	// Implementation for temporal relationship inference
	return []*InferredRelationship{}, nil
}

func (er *EntityResolver) inferBehavioralRelationships(ctx context.Context, req *RelationshipInferenceRequest) ([]*InferredRelationship, error) {
	// Implementation for behavioral relationship inference
	return []*InferredRelationship{}, nil
}

func (er *EntityResolver) inferNetworkRelationships(ctx context.Context, req *RelationshipInferenceRequest) ([]*InferredRelationship, error) {
	// Implementation for network-based relationship inference
	return []*InferredRelationship{}, nil
}

// getFloat64 safely extracts a float64 value from a record
func getFloat64(record map[string]interface{}, key string) float64 {
	if val, ok := record[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0.0
}