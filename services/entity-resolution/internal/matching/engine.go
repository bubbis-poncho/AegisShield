package matching

import (
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/entity-resolution/internal/standardization"
	"github.com/agnivade/levenshtein"
	"github.com/armon/go-radix"
)

// Engine handles fuzzy matching for entity resolution
type Engine struct {
	config       config.MatchingConfig
	standardizer *standardization.Engine
	logger       *slog.Logger
	nameIndex    *radix.Tree
	phoneIndex   map[string][]string
	emailIndex   map[string][]string
}

// MatchCandidate represents a potential entity match
type MatchCandidate struct {
	EntityID          string                 `json:"entity_id"`
	OverallScore      float64                `json:"overall_score"`
	NameScore         float64                `json:"name_score"`
	AddressScore      float64                `json:"address_score"`
	PhoneScore        float64                `json:"phone_score"`
	EmailScore        float64                `json:"email_score"`
	IdentifierMatches map[string]float64     `json:"identifier_matches"`
	Evidence          map[string]interface{} `json:"evidence"`
}

// MatchInput represents input data for matching
type MatchInput struct {
	Name       string            `json:"name"`
	Address    string            `json:"address"`
	Phone      string            `json:"phone"`
	Email      string            `json:"email"`
	Identifiers map[string]string `json:"identifiers"`
}

// MatchResult represents the result of a matching operation
type MatchResult struct {
	Query          *MatchInput        `json:"query"`
	Candidates     []*MatchCandidate  `json:"candidates"`
	BestMatch      *MatchCandidate    `json:"best_match,omitempty"`
	IsMatch        bool               `json:"is_match"`
	MatchConfidence float64           `json:"match_confidence"`
	ProcessingTime  int64             `json:"processing_time_ms"`
}

// NewEngine creates a new matching engine
func NewEngine(config config.MatchingConfig, standardizer *standardization.Engine, logger *slog.Logger) *Engine {
	return &Engine{
		config:       config,
		standardizer: standardizer,
		logger:       logger,
		nameIndex:    radix.New(),
		phoneIndex:   make(map[string][]string),
		emailIndex:   make(map[string][]string),
	}
}

// FindMatches finds potential matches for the given input
func (e *Engine) FindMatches(input *MatchInput, candidateEntities []CandidateEntity) (*MatchResult, error) {
	result := &MatchResult{
		Query:      input,
		Candidates: []*MatchCandidate{},
		IsMatch:    false,
	}

	// Apply blocking if enabled to reduce candidate set
	if e.config.BlockingEnabled {
		candidateEntities = e.applyBlocking(input, candidateEntities)
	}

	// Score each candidate
	for _, candidate := range candidateEntities {
		score := e.calculateMatchScore(input, &candidate)
		
		if score.OverallScore >= e.config.OverallSimilarityThreshold {
			result.Candidates = append(result.Candidates, score)
		}
	}

	// Sort candidates by overall score
	sort.Slice(result.Candidates, func(i, j int) bool {
		return result.Candidates[i].OverallScore > result.Candidates[j].OverallScore
	})

	// Limit to max candidates
	if len(result.Candidates) > e.config.MaxCandidates {
		result.Candidates = result.Candidates[:e.config.MaxCandidates]
	}

	// Determine best match
	if len(result.Candidates) > 0 {
		result.BestMatch = result.Candidates[0]
		result.IsMatch = result.BestMatch.OverallScore >= e.config.OverallSimilarityThreshold
		result.MatchConfidence = result.BestMatch.OverallScore
	}

	return result, nil
}

// CandidateEntity represents an entity that could be a match
type CandidateEntity struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Address     string            `json:"address"`
	Phone       string            `json:"phone"`
	Email       string            `json:"email"`
	Identifiers map[string]string `json:"identifiers"`
}

// calculateMatchScore calculates the overall match score between input and candidate
func (e *Engine) calculateMatchScore(input *MatchInput, candidate *CandidateEntity) *MatchCandidate {
	matchCandidate := &MatchCandidate{
		EntityID:          candidate.ID,
		IdentifierMatches: make(map[string]float64),
		Evidence:          make(map[string]interface{}),
	}

	// Calculate individual scores
	matchCandidate.NameScore = e.calculateNameSimilarity(input.Name, candidate.Name)
	matchCandidate.AddressScore = e.calculateAddressSimilarity(input.Address, candidate.Address)
	matchCandidate.PhoneScore = e.calculatePhoneSimilarity(input.Phone, candidate.Phone)
	matchCandidate.EmailScore = e.calculateEmailSimilarity(input.Email, candidate.Email)

	// Calculate identifier matches
	for key, inputValue := range input.Identifiers {
		if candidateValue, exists := candidate.Identifiers[key]; exists {
			matchCandidate.IdentifierMatches[key] = e.calculateExactMatch(inputValue, candidateValue)
		}
	}

	// Calculate weighted overall score
	matchCandidate.OverallScore = e.calculateWeightedScore(matchCandidate)

	// Store evidence
	matchCandidate.Evidence["name_comparison"] = map[string]interface{}{
		"input_name":      input.Name,
		"candidate_name":  candidate.Name,
		"similarity":      matchCandidate.NameScore,
	}

	if input.Address != "" && candidate.Address != "" {
		matchCandidate.Evidence["address_comparison"] = map[string]interface{}{
			"input_address":      input.Address,
			"candidate_address":  candidate.Address,
			"similarity":         matchCandidate.AddressScore,
		}
	}

	return matchCandidate
}

// Name similarity calculation
func (e *Engine) calculateNameSimilarity(name1, name2 string) float64 {
	if name1 == "" || name2 == "" {
		return 0.0
	}

	// Standardize names
	std1 := e.standardizer.StandardizeName(name1)
	std2 := e.standardizer.StandardizeName(name2)

	var maxScore float64

	// Exact match on standardized names
	if std1.Standardized == std2.Standardized {
		maxScore = 1.0
	}

	// Fuzzy matching if enabled
	if e.config.FuzzyMatchingEnabled {
		// Levenshtein similarity on standardized names
		levenScore := e.calculateLevenshteinSimilarity(std1.Standardized, std2.Standardized)
		maxScore = math.Max(maxScore, levenScore)

		// Token-based similarity
		tokenScore := e.calculateTokenSimilarity(std1.Tokens, std2.Tokens)
		maxScore = math.Max(maxScore, tokenScore)
	}

	// Phonetic matching if enabled
	if e.config.PhoneticMatchingEnabled {
		// Phonetic similarity
		if std1.Phonetic == std2.Phonetic && std1.Phonetic != "" {
			maxScore = math.Max(maxScore, 0.8)
		}

		// Metaphone similarity
		if std1.Metaphone == std2.Metaphone && std1.Metaphone != "" {
			maxScore = math.Max(maxScore, 0.75)
		}
	}

	return maxScore
}

// Address similarity calculation
func (e *Engine) calculateAddressSimilarity(addr1, addr2 string) float64 {
	if addr1 == "" || addr2 == "" {
		return 0.0
	}

	// Standardize addresses
	std1 := e.standardizer.StandardizeAddress(addr1)
	std2 := e.standardizer.StandardizeAddress(addr2)

	// Component-wise comparison
	var componentScores []float64

	// Street number comparison (exact match)
	if std1.StreetNumber != "" && std2.StreetNumber != "" {
		if std1.StreetNumber == std2.StreetNumber {
			componentScores = append(componentScores, 1.0)
		} else {
			componentScores = append(componentScores, 0.0)
		}
	}

	// Street name comparison (fuzzy)
	if std1.StreetName != "" && std2.StreetName != "" {
		streetScore := e.calculateLevenshteinSimilarity(std1.StreetName, std2.StreetName)
		componentScores = append(componentScores, streetScore*0.8) // Weight street name highly
	}

	// City comparison (fuzzy)
	if std1.City != "" && std2.City != "" {
		cityScore := e.calculateLevenshteinSimilarity(std1.City, std2.City)
		componentScores = append(componentScores, cityScore*0.6)
	}

	// State comparison (exact)
	if std1.State != "" && std2.State != "" {
		if std1.State == std2.State {
			componentScores = append(componentScores, 0.4)
		}
	}

	// Postal code comparison (exact)
	if std1.PostalCode != "" && std2.PostalCode != "" {
		if std1.PostalCode == std2.PostalCode {
			componentScores = append(componentScores, 0.7)
		}
	}

	// Calculate average of component scores
	if len(componentScores) == 0 {
		return 0.0
	}

	var sum float64
	for _, score := range componentScores {
		sum += score
	}

	return math.Min(1.0, sum/float64(len(componentScores)))
}

// Phone similarity calculation
func (e *Engine) calculatePhoneSimilarity(phone1, phone2 string) float64 {
	if phone1 == "" || phone2 == "" {
		return 0.0
	}

	// Standardize phone numbers
	std1 := e.standardizer.StandardizePhone(phone1)
	std2 := e.standardizer.StandardizePhone(phone2)

	// Exact match on standardized numbers
	if std1.Standardized == std2.Standardized {
		return 1.0
	}

	// Compare just the number part (without country code/area code)
	if std1.Number != "" && std2.Number != "" && std1.Number == std2.Number {
		return 0.9
	}

	// Fuzzy match on the full standardized number
	return e.calculateLevenshteinSimilarity(std1.Standardized, std2.Standardized)
}

// Email similarity calculation
func (e *Engine) calculateEmailSimilarity(email1, email2 string) float64 {
	if email1 == "" || email2 == "" {
		return 0.0
	}

	// Standardize emails
	std1 := e.standardizer.StandardizeEmail(email1)
	std2 := e.standardizer.StandardizeEmail(email2)

	// Exact match
	if std1 == std2 {
		return 1.0
	}

	// Compare local parts (before @)
	parts1 := strings.Split(std1, "@")
	parts2 := strings.Split(std2, "@")

	if len(parts1) == 2 && len(parts2) == 2 {
		// Same domain, compare local parts
		if parts1[1] == parts2[1] {
			localSim := e.calculateLevenshteinSimilarity(parts1[0], parts2[0])
			return localSim * 0.9 // High weight for same domain
		}

		// Different domains, lower overall similarity
		localSim := e.calculateLevenshteinSimilarity(parts1[0], parts2[0])
		domainSim := e.calculateLevenshteinSimilarity(parts1[1], parts2[1])
		return (localSim + domainSim) / 2 * 0.7
	}

	// Fallback to general string similarity
	return e.calculateLevenshteinSimilarity(std1, std2)
}

// Exact match calculation
func (e *Engine) calculateExactMatch(value1, value2 string) float64 {
	if value1 == "" || value2 == "" {
		return 0.0
	}

	// Normalize and compare
	norm1 := strings.ToLower(strings.TrimSpace(value1))
	norm2 := strings.ToLower(strings.TrimSpace(value2))

	if norm1 == norm2 {
		return 1.0
	}

	return 0.0
}

// Levenshtein similarity calculation
func (e *Engine) calculateLevenshteinSimilarity(s1, s2 string) float64 {
	if s1 == "" && s2 == "" {
		return 1.0
	}

	if s1 == "" || s2 == "" {
		return 0.0
	}

	distance := levenshtein.ComputeDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - (float64(distance) / maxLen)
}

// Token-based similarity calculation
func (e *Engine) calculateTokenSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Create sets of tokens
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, token := range tokens1 {
		set1[strings.ToLower(token)] = true
	}

	for _, token := range tokens2 {
		set2[strings.ToLower(token)] = true
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := len(set1)

	for token := range set2 {
		if set1[token] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// Weighted score calculation
func (e *Engine) calculateWeightedScore(candidate *MatchCandidate) float64 {
	var score float64
	var totalWeight float64

	// Name weight (highest)
	nameWeight := 0.4
	if candidate.NameScore > 0 {
		score += candidate.NameScore * nameWeight
		totalWeight += nameWeight
	}

	// Address weight
	addressWeight := 0.25
	if candidate.AddressScore > 0 {
		score += candidate.AddressScore * addressWeight
		totalWeight += addressWeight
	}

	// Phone weight
	phoneWeight := 0.15
	if candidate.PhoneScore > 0 {
		score += candidate.PhoneScore * phoneWeight
		totalWeight += phoneWeight
	}

	// Email weight
	emailWeight := 0.1
	if candidate.EmailScore > 0 {
		score += candidate.EmailScore * emailWeight
		totalWeight += emailWeight
	}

	// Identifier weights (very high for exact matches)
	identifierWeight := 0.1
	for _, identifierScore := range candidate.IdentifierMatches {
		if identifierScore > 0 {
			score += identifierScore * identifierWeight
			totalWeight += identifierWeight
		}
	}

	// Normalize by total weight
	if totalWeight > 0 {
		return score / totalWeight
	}

	return 0.0
}

// Blocking operations
func (e *Engine) applyBlocking(input *MatchInput, candidates []CandidateEntity) []CandidateEntity {
	if input.Name == "" {
		return candidates
	}

	// Generate blocking key from name
	blockingKey := e.generateBlockingKey(input.Name)
	if blockingKey == "" {
		return candidates
	}

	// Filter candidates that share the blocking key
	var filtered []CandidateEntity
	for _, candidate := range candidates {
		candidateKey := e.generateBlockingKey(candidate.Name)
		if candidateKey != "" && e.shareBlockingKey(blockingKey, candidateKey) {
			filtered = append(filtered, candidate)
		}
	}

	// If blocking filtered too many candidates, return original set
	if len(filtered) < len(candidates)/10 { // Less than 10% remaining
		return candidates
	}

	return filtered
}

func (e *Engine) generateBlockingKey(name string) string {
	if name == "" {
		return ""
	}

	// Standardize name first
	std := e.standardizer.StandardizeName(name)
	
	// Use first N characters of standardized name
	key := std.Standardized
	if len(key) > e.config.BlockingKeySize {
		key = key[:e.config.BlockingKeySize]
	}

	return strings.ToLower(key)
}

func (e *Engine) shareBlockingKey(key1, key2 string) bool {
	if key1 == "" || key2 == "" {
		return false
	}

	// Exact match
	if key1 == key2 {
		return true
	}

	// Allow for small differences in blocking key
	if len(key1) == len(key2) {
		differences := 0
		for i := 0; i < len(key1); i++ {
			if key1[i] != key2[i] {
				differences++
			}
		}
		// Allow 1 character difference for blocking keys
		return differences <= 1
	}

	return false
}

// Index management (for performance optimization)
func (e *Engine) IndexEntity(entityID string, name string, phone string, email string) {
	// Index name
	if name != "" {
		std := e.standardizer.StandardizeName(name)
		blockingKey := e.generateBlockingKey(name)
		if blockingKey != "" {
			if existing, ok := e.nameIndex.Get(blockingKey); ok {
				entities := existing.([]string)
				entities = append(entities, entityID)
				e.nameIndex.Insert(blockingKey, entities)
			} else {
				e.nameIndex.Insert(blockingKey, []string{entityID})
			}
		}
	}

	// Index phone
	if phone != "" {
		std := e.standardizer.StandardizePhone(phone)
		if std.Standardized != "" {
			e.phoneIndex[std.Standardized] = append(e.phoneIndex[std.Standardized], entityID)
		}
	}

	// Index email
	if email != "" {
		std := e.standardizer.StandardizeEmail(email)
		if std != "" {
			e.emailIndex[std] = append(e.emailIndex[std], entityID)
		}
	}
}

func (e *Engine) GetCandidatesByIndex(input *MatchInput) []string {
	candidateSet := make(map[string]bool)

	// Get candidates by name
	if input.Name != "" {
		blockingKey := e.generateBlockingKey(input.Name)
		if blockingKey != "" {
			if entities, ok := e.nameIndex.Get(blockingKey); ok {
				for _, entityID := range entities.([]string) {
					candidateSet[entityID] = true
				}
			}
		}
	}

	// Get candidates by phone
	if input.Phone != "" {
		std := e.standardizer.StandardizePhone(input.Phone)
		if entities, ok := e.phoneIndex[std.Standardized]; ok {
			for _, entityID := range entities {
				candidateSet[entityID] = true
			}
		}
	}

	// Get candidates by email
	if input.Email != "" {
		std := e.standardizer.StandardizeEmail(input.Email)
		if entities, ok := e.emailIndex[std]; ok {
			for _, entityID := range entities {
				candidateSet[entityID] = true
			}
		}
	}

	// Convert set to slice
	var candidates []string
	for entityID := range candidateSet {
		candidates = append(candidates, entityID)
	}

	return candidates
}