package patterns

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/neo4j"
	"github.com/google/uuid"
)

// PatternDetector identifies suspicious patterns in the graph
type PatternDetector struct {
	neo4jClient *neo4j.Client
	config      config.GraphEngineConfig
	logger      *slog.Logger
}

// PatternType represents different types of patterns to detect
type PatternType string

const (
	PatternTypeSmurfing          PatternType = "smurfing"
	PatternTypeLayering          PatternType = "layering"
	PatternTypeStructuring       PatternType = "structuring"
	PatternTypeCircularFlow      PatternType = "circular_flow"
	PatternTypeRapidMovement     PatternType = "rapid_movement"
	PatternTypeHighRiskGeography PatternType = "high_risk_geography"
	PatternTypeUnusualVolume     PatternType = "unusual_volume"
	PatternTypeShellCompany      PatternType = "shell_company"
	PatternTypeMuleAccount       PatternType = "mule_account"
	PatternTypeKitingScheme      PatternType = "kiting_scheme"
)

// Pattern represents a detected suspicious pattern
type Pattern struct {
	ID               string                 `json:"id"`
	Type             PatternType            `json:"type"`
	Entities         []*neo4j.Entity        `json:"entities"`
	Relationships    []*neo4j.Relationship  `json:"relationships"`
	Confidence       float64                `json:"confidence"`
	RiskScore        float64                `json:"risk_score"`
	DetectedAt       time.Time              `json:"detected_at"`
	Description      string                 `json:"description"`
	Indicators       []string               `json:"indicators"`
	Metadata         map[string]interface{} `json:"metadata"`
	InvestigationID  string                 `json:"investigation_id,omitempty"`
}

// DetectionRequest represents a pattern detection request
type DetectionRequest struct {
	Types           []PatternType          `json:"types"`
	EntityIDs       []string               `json:"entity_ids,omitempty"`
	TimeWindow      time.Duration          `json:"time_window,omitempty"`
	MinConfidence   float64                `json:"min_confidence"`
	MaxDepth        int                    `json:"max_depth"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	InvestigationID string                 `json:"investigation_id,omitempty"`
}

// DetectionResult contains the results of pattern detection
type DetectionResult struct {
	RequestID       string     `json:"request_id"`
	Patterns        []*Pattern `json:"patterns"`
	ProcessingTime  time.Duration `json:"processing_time"`
	EntitiesAnalyzed int       `json:"entities_analyzed"`
	PatternsFound   int        `json:"patterns_found"`
	HighRiskPatterns int       `json:"high_risk_patterns"`
}

// SmurfingIndicators represents indicators for smurfing detection
type SmurfingIndicators struct {
	MultipleSmallTransactions bool    `json:"multiple_small_transactions"`
	JustBelowThreshold        bool    `json:"just_below_threshold"`
	FrequencyScore            float64 `json:"frequency_score"`
	AmountVariation           float64 `json:"amount_variation"`
	NumberOfAccounts          int     `json:"number_of_accounts"`
}

// LayeringIndicators represents indicators for layering detection
type LayeringIndicators struct {
	ComplexityScore     float64 `json:"complexity_score"`
	NumberOfLayers      int     `json:"number_of_layers"`
	GeographicDiversity int     `json:"geographic_diversity"`
	InstitutionDiversity int    `json:"institution_diversity"`
	TimeSpread          time.Duration `json:"time_spread"`
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(client *neo4j.Client, config config.GraphEngineConfig, logger *slog.Logger) *PatternDetector {
	return &PatternDetector{
		neo4jClient: client,
		config:      config,
		logger:      logger,
	}
}

// DetectPatterns performs comprehensive pattern detection
func (pd *PatternDetector) DetectPatterns(ctx context.Context, req *DetectionRequest) (*DetectionResult, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	pd.logger.Info("Starting pattern detection",
		"request_id", requestID,
		"types", req.Types,
		"entity_count", len(req.EntityIDs))

	result := &DetectionResult{
		RequestID: requestID,
		Patterns:  make([]*Pattern, 0),
	}

	// Detect each requested pattern type
	for _, patternType := range req.Types {
		patterns, err := pd.detectPatternType(ctx, patternType, req)
		if err != nil {
			pd.logger.Error("Failed to detect pattern type",
				"type", patternType,
				"error", err)
			continue
		}

		result.Patterns = append(result.Patterns, patterns...)
	}

	// Calculate result statistics
	result.ProcessingTime = time.Since(startTime)
	result.PatternsFound = len(result.Patterns)

	// Count high-risk patterns (confidence > 0.8)
	for _, pattern := range result.Patterns {
		if pattern.Confidence > 0.8 {
			result.HighRiskPatterns++
		}
	}

	pd.logger.Info("Pattern detection completed",
		"request_id", requestID,
		"patterns_found", result.PatternsFound,
		"high_risk_patterns", result.HighRiskPatterns,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// detectPatternType detects a specific pattern type
func (pd *PatternDetector) detectPatternType(ctx context.Context, patternType PatternType, req *DetectionRequest) ([]*Pattern, error) {
	switch patternType {
	case PatternTypeSmurfing:
		return pd.detectSmurfingPattern(ctx, req)
	case PatternTypeLayering:
		return pd.detectLayeringPattern(ctx, req)
	case PatternTypeStructuring:
		return pd.detectStructuringPattern(ctx, req)
	case PatternTypeCircularFlow:
		return pd.detectCircularFlowPattern(ctx, req)
	case PatternTypeRapidMovement:
		return pd.detectRapidMovementPattern(ctx, req)
	case PatternTypeHighRiskGeography:
		return pd.detectHighRiskGeographyPattern(ctx, req)
	case PatternTypeUnusualVolume:
		return pd.detectUnusualVolumePattern(ctx, req)
	case PatternTypeShellCompany:
		return pd.detectShellCompanyPattern(ctx, req)
	case PatternTypeMuleAccount:
		return pd.detectMuleAccountPattern(ctx, req)
	case PatternTypeKitingScheme:
		return pd.detectKitingSchemePattern(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported pattern type: %s", patternType)
	}
}

// detectSmurfingPattern detects smurfing (structuring) patterns
func (pd *PatternDetector) detectSmurfingPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Cypher query to find potential smurfing patterns
	query := `
		MATCH (source:Account)-[t:TRANSACTION]->(dest:Account)
		WHERE t.amount < $threshold
		AND t.timestamp >= datetime() - duration($timeWindow)
		WITH source, dest, COUNT(t) as txCount, 
			 AVG(t.amount) as avgAmount,
			 STDEV(t.amount) as amountStdev,
			 COLLECT(t) as transactions
		WHERE txCount >= $minTransactions
		AND avgAmount < $avgThreshold
		RETURN source, dest, txCount, avgAmount, amountStdev, transactions
		ORDER BY txCount DESC
		LIMIT 100
	`

	threshold := 10000.0 // $10,000 threshold
	if val, ok := req.Parameters["threshold"]; ok {
		if t, ok := val.(float64); ok {
			threshold = t
		}
	}

	minTransactions := 5
	if val, ok := req.Parameters["min_transactions"]; ok {
		if mt, ok := val.(int); ok {
			minTransactions = mt
		}
	}

	timeWindow := req.TimeWindow
	if timeWindow == 0 {
		timeWindow = 30 * 24 * time.Hour // 30 days default
	}

	params := map[string]interface{}{
		"threshold":       threshold,
		"avgThreshold":    threshold * 0.8,
		"minTransactions": minTransactions,
		"timeWindow":      timeWindow.String(),
	}

	records, err := pd.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute smurfing detection query: %w", err)
	}

	patterns := make([]*Pattern, 0)
	for _, record := range records {
		pattern := pd.buildSmurfingPattern(record, req)
		if pattern != nil && pattern.Confidence >= req.MinConfidence {
			patterns = append(patterns, pattern)
		}
	}

	return patterns, nil
}

// buildSmurfingPattern builds a smurfing pattern from query results
func (pd *PatternDetector) buildSmurfingPattern(record map[string]interface{}, req *DetectionRequest) *Pattern {
	txCount, ok := record["txCount"].(int64)
	if !ok {
		return nil
	}

	avgAmount, ok := record["avgAmount"].(float64)
	if !ok {
		return nil
	}

	amountStdev, ok := record["amountStdev"].(float64)
	if !ok {
		amountStdev = 0
	}

	// Calculate confidence based on various factors
	confidence := pd.calculateSmurfingConfidence(int(txCount), avgAmount, amountStdev)
	
	// Calculate risk score
	riskScore := pd.calculateRiskScore(confidence, PatternTypeSmurfing)

	indicators := []string{
		fmt.Sprintf("Multiple small transactions: %d", txCount),
		fmt.Sprintf("Average amount: $%.2f", avgAmount),
		fmt.Sprintf("Amount variation: %.2f", amountStdev),
	}

	if avgAmount < 10000 {
		indicators = append(indicators, "Amounts consistently below reporting threshold")
	}

	if amountStdev < avgAmount*0.1 {
		indicators = append(indicators, "Unusually consistent transaction amounts")
	}

	smurfingIndicators := &SmurfingIndicators{
		MultipleSmallTransactions: txCount >= 5,
		JustBelowThreshold:        avgAmount >= 9000 && avgAmount < 10000,
		FrequencyScore:            float64(txCount) / 30.0, // transactions per day
		AmountVariation:           amountStdev / avgAmount,
		NumberOfAccounts:          2, // source and dest
	}

	pattern := &Pattern{
		ID:          uuid.New().String(),
		Type:        PatternTypeSmurfing,
		Confidence:  confidence,
		RiskScore:   riskScore,
		DetectedAt:  time.Now(),
		Description: fmt.Sprintf("Potential smurfing pattern with %d transactions averaging $%.2f", txCount, avgAmount),
		Indicators:  indicators,
		Metadata: map[string]interface{}{
			"transaction_count":      txCount,
			"average_amount":         avgAmount,
			"amount_standard_dev":    amountStdev,
			"smurfing_indicators":    smurfingIndicators,
		},
		InvestigationID: req.InvestigationID,
	}

	// Extract entities and relationships from record
	// This would be implemented based on the actual Neo4j record structure

	return pattern
}

// detectLayeringPattern detects layering patterns in money laundering
func (pd *PatternDetector) detectLayeringPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Complex query to identify layering patterns
	query := `
		MATCH path = (start:Account)-[:TRANSACTION*2..10]->(end:Account)
		WHERE start <> end
		AND ALL(r IN relationships(path) WHERE r.timestamp >= datetime() - duration($timeWindow))
		WITH path, length(path) as pathLength,
			 [n IN nodes(path) | n.country] as countries,
			 [n IN nodes(path) | n.institution] as institutions,
			 [r IN relationships(path) | r.amount] as amounts
		WHERE pathLength >= $minLayers
		AND SIZE(apoc.coll.toSet(countries)) >= $minCountries
		RETURN path, pathLength, countries, institutions, amounts,
			   SIZE(apoc.coll.toSet(countries)) as countryDiversity,
			   SIZE(apoc.coll.toSet(institutions)) as institutionDiversity
		ORDER BY pathLength DESC, countryDiversity DESC
		LIMIT 50
	`

	minLayers := 3
	if val, ok := req.Parameters["min_layers"]; ok {
		if ml, ok := val.(int); ok {
			minLayers = ml
		}
	}

	minCountries := 2
	if val, ok := req.Parameters["min_countries"]; ok {
		if mc, ok := val.(int); ok {
			minCountries = mc
		}
	}

	timeWindow := req.TimeWindow
	if timeWindow == 0 {
		timeWindow = 90 * 24 * time.Hour // 90 days default
	}

	params := map[string]interface{}{
		"minLayers":    minLayers,
		"minCountries": minCountries,
		"timeWindow":   timeWindow.String(),
	}

	records, err := pd.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute layering detection query: %w", err)
	}

	patterns := make([]*Pattern, 0)
	for _, record := range records {
		pattern := pd.buildLayeringPattern(record, req)
		if pattern != nil && pattern.Confidence >= req.MinConfidence {
			patterns = append(patterns, pattern)
		}
	}

	return patterns, nil
}

// buildLayeringPattern builds a layering pattern from query results
func (pd *PatternDetector) buildLayeringPattern(record map[string]interface{}, req *DetectionRequest) *Pattern {
	pathLength, ok := record["pathLength"].(int64)
	if !ok {
		return nil
	}

	countryDiversity, ok := record["countryDiversity"].(int64)
	if !ok {
		return nil
	}

	institutionDiversity, ok := record["institutionDiversity"].(int64)
	if !ok {
		return nil
	}

	// Calculate confidence based on complexity factors
	confidence := pd.calculateLayeringConfidence(int(pathLength), int(countryDiversity), int(institutionDiversity))
	riskScore := pd.calculateRiskScore(confidence, PatternTypeLayering)

	indicators := []string{
		fmt.Sprintf("Transaction chain length: %d", pathLength),
		fmt.Sprintf("Geographic diversity: %d countries", countryDiversity),
		fmt.Sprintf("Institution diversity: %d institutions", institutionDiversity),
	}

	if pathLength > 5 {
		indicators = append(indicators, "Unusually long transaction chain")
	}

	if countryDiversity >= 3 {
		indicators = append(indicators, "High geographic diversity")
	}

	layeringIndicators := &LayeringIndicators{
		ComplexityScore:      float64(pathLength) * float64(countryDiversity),
		NumberOfLayers:       int(pathLength),
		GeographicDiversity:  int(countryDiversity),
		InstitutionDiversity: int(institutionDiversity),
	}

	pattern := &Pattern{
		ID:          uuid.New().String(),
		Type:        PatternTypeLayering,
		Confidence:  confidence,
		RiskScore:   riskScore,
		DetectedAt:  time.Now(),
		Description: fmt.Sprintf("Complex layering pattern with %d layers across %d countries", pathLength, countryDiversity),
		Indicators:  indicators,
		Metadata: map[string]interface{}{
			"path_length":            pathLength,
			"country_diversity":      countryDiversity,
			"institution_diversity":  institutionDiversity,
			"layering_indicators":    layeringIndicators,
		},
		InvestigationID: req.InvestigationID,
	}

	return pattern
}

// detectCircularFlowPattern detects circular money flow patterns
func (pd *PatternDetector) detectCircularFlowPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	query := `
		MATCH path = (start:Account)-[:TRANSACTION*3..8]->(start)
		WHERE ALL(r IN relationships(path) WHERE r.timestamp >= datetime() - duration($timeWindow))
		WITH path, 
			 length(path) as circleLength,
			 [r IN relationships(path) | r.amount] as amounts,
			 [r IN relationships(path) | r.timestamp] as timestamps
		WHERE circleLength >= $minCircleLength
		RETURN path, circleLength, amounts, timestamps,
			   reduce(total = 0, amount IN amounts | total + amount) as totalAmount,
			   max(timestamps) - min(timestamps) as timeSpan
		ORDER BY circleLength DESC, totalAmount DESC
		LIMIT 30
	`

	minCircleLength := 3
	if val, ok := req.Parameters["min_circle_length"]; ok {
		if mcl, ok := val.(int); ok {
			minCircleLength = mcl
		}
	}

	timeWindow := req.TimeWindow
	if timeWindow == 0 {
		timeWindow = 60 * 24 * time.Hour // 60 days default
	}

	params := map[string]interface{}{
		"minCircleLength": minCircleLength,
		"timeWindow":      timeWindow.String(),
	}

	records, err := pd.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute circular flow detection query: %w", err)
	}

	patterns := make([]*Pattern, 0)
	for _, record := range records {
		pattern := pd.buildCircularFlowPattern(record, req)
		if pattern != nil && pattern.Confidence >= req.MinConfidence {
			patterns = append(patterns, pattern)
		}
	}

	return patterns, nil
}

// buildCircularFlowPattern builds a circular flow pattern
func (pd *PatternDetector) buildCircularFlowPattern(record map[string]interface{}, req *DetectionRequest) *Pattern {
	circleLength, ok := record["circleLength"].(int64)
	if !ok {
		return nil
	}

	totalAmount, ok := record["totalAmount"].(float64)
	if !ok {
		return nil
	}

	// Calculate confidence based on circle characteristics
	confidence := pd.calculateCircularFlowConfidence(int(circleLength), totalAmount)
	riskScore := pd.calculateRiskScore(confidence, PatternTypeCircularFlow)

	indicators := []string{
		fmt.Sprintf("Circular transaction flow with %d steps", circleLength),
		fmt.Sprintf("Total amount circulated: $%.2f", totalAmount),
	}

	if circleLength <= 4 && totalAmount > 50000 {
		indicators = append(indicators, "Short circle with high amount")
	}

	if circleLength >= 6 {
		indicators = append(indicators, "Complex circular structure")
	}

	pattern := &Pattern{
		ID:          uuid.New().String(),
		Type:        PatternTypeCircularFlow,
		Confidence:  confidence,
		RiskScore:   riskScore,
		DetectedAt:  time.Now(),
		Description: fmt.Sprintf("Circular money flow with %d steps totaling $%.2f", circleLength, totalAmount),
		Indicators:  indicators,
		Metadata: map[string]interface{}{
			"circle_length": circleLength,
			"total_amount":  totalAmount,
		},
		InvestigationID: req.InvestigationID,
	}

	return pattern
}

// Additional pattern detection methods would be implemented here...
// detectStructuringPattern, detectRapidMovementPattern, etc.

// calculateSmurfingConfidence calculates confidence for smurfing patterns
func (pd *PatternDetector) calculateSmurfingConfidence(txCount int, avgAmount, amountStdev float64) float64 {
	confidence := 0.0

	// Transaction count factor
	if txCount >= 10 {
		confidence += 0.3
	} else if txCount >= 5 {
		confidence += 0.2
	}

	// Amount threshold factor
	if avgAmount >= 9000 && avgAmount < 10000 {
		confidence += 0.4 // Just below threshold
	} else if avgAmount < 5000 {
		confidence += 0.2
	}

	// Consistency factor
	if amountStdev > 0 {
		coefficient := amountStdev / avgAmount
		if coefficient < 0.1 {
			confidence += 0.3 // Very consistent amounts
		} else if coefficient < 0.2 {
			confidence += 0.2
		}
	}

	return math.Min(confidence, 1.0)
}

// calculateLayeringConfidence calculates confidence for layering patterns
func (pd *PatternDetector) calculateLayeringConfidence(pathLength, countryDiversity, institutionDiversity int) float64 {
	confidence := 0.0

	// Path length factor
	if pathLength >= 6 {
		confidence += 0.3
	} else if pathLength >= 4 {
		confidence += 0.2
	}

	// Geographic diversity factor
	if countryDiversity >= 4 {
		confidence += 0.3
	} else if countryDiversity >= 3 {
		confidence += 0.2
	} else if countryDiversity >= 2 {
		confidence += 0.1
	}

	// Institution diversity factor
	if institutionDiversity >= 4 {
		confidence += 0.3
	} else if institutionDiversity >= 3 {
		confidence += 0.2
	}

	// Complexity bonus
	complexityScore := float64(pathLength) * float64(countryDiversity)
	if complexityScore > 15 {
		confidence += 0.1
	}

	return math.Min(confidence, 1.0)
}

// calculateCircularFlowConfidence calculates confidence for circular flow patterns
func (pd *PatternDetector) calculateCircularFlowConfidence(circleLength int, totalAmount float64) float64 {
	confidence := 0.3 // Base confidence for any circular flow

	// Length factor
	if circleLength <= 4 {
		confidence += 0.2 // Simple circles are more suspicious
	} else if circleLength >= 7 {
		confidence += 0.3 // Complex circles are also suspicious
	}

	// Amount factor
	if totalAmount > 100000 {
		confidence += 0.3
	} else if totalAmount > 50000 {
		confidence += 0.2
	}

	return math.Min(confidence, 1.0)
}

// calculateRiskScore calculates overall risk score for a pattern
func (pd *PatternDetector) calculateRiskScore(confidence float64, patternType PatternType) float64 {
	// Base risk score from confidence
	riskScore := confidence * 100

	// Pattern type multipliers
	multipliers := map[PatternType]float64{
		PatternTypeSmurfing:          1.0,
		PatternTypeLayering:          1.2,
		PatternTypeStructuring:       1.1,
		PatternTypeCircularFlow:      1.3,
		PatternTypeRapidMovement:     1.1,
		PatternTypeHighRiskGeography: 1.4,
		PatternTypeUnusualVolume:     1.0,
		PatternTypeShellCompany:      1.5,
		PatternTypeMuleAccount:       1.2,
		PatternTypeKitingScheme:      1.3,
	}

	if multiplier, exists := multipliers[patternType]; exists {
		riskScore *= multiplier
	}

	return math.Min(riskScore, 100.0)
}

// GetPatternStatistics returns statistics about detected patterns
func (pd *PatternDetector) GetPatternStatistics(ctx context.Context, timeWindow time.Duration) (map[string]interface{}, error) {
	query := `
		MATCH (p:Pattern)
		WHERE p.detected_at >= datetime() - duration($timeWindow)
		RETURN p.type as pattern_type, 
			   COUNT(p) as count,
			   AVG(p.confidence) as avg_confidence,
			   AVG(p.risk_score) as avg_risk_score,
			   MAX(p.risk_score) as max_risk_score
		ORDER BY count DESC
	`

	params := map[string]interface{}{
		"timeWindow": timeWindow.String(),
	}

	records, err := pd.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern statistics: %w", err)
	}

	statistics := make(map[string]interface{})
	patternStats := make([]map[string]interface{}, 0)

	for _, record := range records {
		stat := map[string]interface{}{
			"pattern_type":    record["pattern_type"],
			"count":           record["count"],
			"avg_confidence":  record["avg_confidence"],
			"avg_risk_score":  record["avg_risk_score"],
			"max_risk_score":  record["max_risk_score"],
		}
		patternStats = append(patternStats, stat)
	}

	statistics["patterns"] = patternStats
	statistics["total_patterns"] = len(patternStats)
	statistics["time_window"] = timeWindow.String()

	return statistics, nil
}

// Placeholder methods for other pattern types
func (pd *PatternDetector) detectStructuringPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting structuring patterns
	return []*Pattern{}, nil
}

func (pd *PatternDetector) detectRapidMovementPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting rapid movement patterns
	return []*Pattern{}, nil
}

func (pd *PatternDetector) detectHighRiskGeographyPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting high-risk geography patterns
	return []*Pattern{}, nil
}

func (pd *PatternDetector) detectUnusualVolumePattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting unusual volume patterns
	return []*Pattern{}, nil
}

func (pd *PatternDetector) detectShellCompanyPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting shell company patterns
	return []*Pattern{}, nil
}

func (pd *PatternDetector) detectMuleAccountPattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting mule account patterns
	return []*Pattern{}, nil
}

func (pd *PatternDetector) detectKitingSchemePattern(ctx context.Context, req *DetectionRequest) ([]*Pattern, error) {
	// Implementation for detecting kiting scheme patterns
	return []*Pattern{}, nil
}