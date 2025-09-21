package quality

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aegisshield/data-integration/internal/config"
	"go.uber.org/zap"
)

// Checker handles data quality assessment
type Checker struct {
	config config.QualityConfig
	logger *zap.Logger
}

// QualityReport represents a comprehensive data quality assessment
type QualityReport struct {
	OverallScore    float64                `json:"overall_score"`
	Dimensions      map[string]float64     `json:"dimensions"`
	RecordCount     int                    `json:"record_count"`
	AssessedAt      time.Time              `json:"assessed_at"`
	Issues          []QualityIssue         `json:"issues,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
	FieldScores     map[string]FieldScore  `json:"field_scores"`
}

// QualityIssue represents a data quality issue
type QualityIssue struct {
	Type        IssueType   `json:"type"`
	Severity    Severity    `json:"severity"`
	Field       string      `json:"field,omitempty"`
	Description string      `json:"description"`
	Count       int         `json:"count"`
	Percentage  float64     `json:"percentage"`
	Examples    []interface{} `json:"examples,omitempty"`
}

// FieldScore represents quality scores for a specific field
type FieldScore struct {
	Field           string  `json:"field"`
	CompletenessScore float64 `json:"completeness_score"`
	AccuracyScore    float64 `json:"accuracy_score"`
	ConsistencyScore float64 `json:"consistency_score"`
	ValidityScore    float64 `json:"validity_score"`
	OverallScore     float64 `json:"overall_score"`
}

// IssueType represents the type of quality issue
type IssueType string

const (
	IssueTypeCompleteness IssueType = "completeness"
	IssueTypeAccuracy     IssueType = "accuracy"
	IssueTypeConsistency  IssueType = "consistency"
	IssueTypeValidity     IssueType = "validity"
	IssueTypeFreshness    IssueType = "freshness"
	IssueTypeUniqueness   IssueType = "uniqueness"
)

// Severity represents the severity of a quality issue
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// QualityDimension represents a data quality dimension
type QualityDimension string

const (
	DimensionCompleteness QualityDimension = "completeness"
	DimensionAccuracy     QualityDimension = "accuracy"
	DimensionConsistency  QualityDimension = "consistency"
	DimensionValidity     QualityDimension = "validity"
	DimensionUniqueness   QualityDimension = "uniqueness"
	DimensionFreshness    QualityDimension = "freshness"
)

// NewChecker creates a new data quality checker
func NewChecker(config config.QualityConfig, logger *zap.Logger) *Checker {
	return &Checker{
		config: config,
		logger: logger,
	}
}

// CheckQuality performs comprehensive data quality assessment
func (c *Checker) CheckQuality(ctx context.Context, records []map[string]interface{}) (*QualityReport, error) {
	if !c.config.EnableQualityChecks {
		return &QualityReport{
			OverallScore: 1.0,
			RecordCount:  len(records),
			AssessedAt:   time.Now(),
		}, nil
	}

	c.logger.Info("Starting data quality assessment",
		zap.Int("record_count", len(records)))

	report := &QualityReport{
		Dimensions:  make(map[string]float64),
		RecordCount: len(records),
		AssessedAt:  time.Now(),
		Issues:      []QualityIssue{},
		FieldScores: make(map[string]FieldScore),
	}

	if len(records) == 0 {
		report.OverallScore = 1.0
		return report, nil
	}

	// Assess each dimension
	completenessScore := c.assessCompleteness(records, report)
	accuracyScore := c.assessAccuracy(records, report)
	consistencyScore := c.assessConsistency(records, report)
	validityScore := c.assessValidity(records, report)
	uniquenessScore := c.assessUniqueness(records, report)
	freshnessScore := c.assessFreshness(records, report)

	// Store dimension scores
	report.Dimensions[string(DimensionCompleteness)] = completenessScore
	report.Dimensions[string(DimensionAccuracy)] = accuracyScore
	report.Dimensions[string(DimensionConsistency)] = consistencyScore
	report.Dimensions[string(DimensionValidity)] = validityScore
	report.Dimensions[string(DimensionUniqueness)] = uniquenessScore
	report.Dimensions[string(DimensionFreshness)] = freshnessScore

	// Calculate overall score (weighted average)
	weights := map[string]float64{
		string(DimensionCompleteness): 0.25,
		string(DimensionAccuracy):     0.25,
		string(DimensionConsistency):  0.20,
		string(DimensionValidity):     0.15,
		string(DimensionUniqueness):   0.10,
		string(DimensionFreshness):    0.05,
	}

	overallScore := 0.0
	for dimension, score := range report.Dimensions {
		if weight, exists := weights[dimension]; exists {
			overallScore += score * weight
		}
	}
	report.OverallScore = overallScore

	// Assess field-level quality
	c.assessFieldQuality(records, report)

	// Generate recommendations
	c.generateRecommendations(report)

	c.logger.Info("Data quality assessment completed",
		zap.Float64("overall_score", report.OverallScore),
		zap.Int("issues_found", len(report.Issues)))

	return report, nil
}

// assessCompleteness assesses data completeness
func (c *Checker) assessCompleteness(records []map[string]interface{}, report *QualityReport) float64 {
	if len(records) == 0 {
		return 1.0
	}

	// Get all fields
	allFields := make(map[string]bool)
	for _, record := range records {
		for field := range record {
			allFields[field] = true
		}
	}

	totalFields := len(allFields)
	totalCells := len(records) * totalFields
	completeCells := 0

	fieldCompleteness := make(map[string]int)

	for _, record := range records {
		for field := range allFields {
			if value, exists := record[field]; exists && !c.isEmptyValue(value) {
				completeCells++
				fieldCompleteness[field]++
			}
		}
	}

	completenessScore := float64(completeCells) / float64(totalCells)

	// Identify fields with low completeness
	for field := range allFields {
		fieldCompleteness := float64(fieldCompleteness[field]) / float64(len(records))
		if fieldCompleteness < c.config.CompletenessThreshold {
			missingCount := len(records) - fieldCompleteness[field]
			issue := QualityIssue{
				Type:        IssueTypeCompleteness,
				Severity:    c.getSeverity(fieldCompleteness),
				Field:       field,
				Description: fmt.Sprintf("Field '%s' has low completeness", field),
				Count:       missingCount,
				Percentage:  (1 - fieldCompleteness) * 100,
			}
			report.Issues = append(report.Issues, issue)
		}
	}

	return completenessScore
}

// assessAccuracy assesses data accuracy
func (c *Checker) assessAccuracy(records []map[string]interface{}, report *QualityReport) float64 {
	if len(records) == 0 {
		return 1.0
	}

	totalChecks := 0
	accurateChecks := 0

	// Check for common accuracy issues
	for i, record := range records {
		for field, value := range record {
			if c.isEmptyValue(value) {
				continue
			}

			totalChecks++

			// Check for obvious accuracy issues
			if c.isAccurateValue(field, value) {
				accurateChecks++
			} else {
				issue := QualityIssue{
					Type:        IssueTypeAccuracy,
					Severity:    SeverityMedium,
					Field:       field,
					Description: fmt.Sprintf("Potential accuracy issue in field '%s'", field),
					Count:       1,
					Examples:    []interface{}{value},
				}
				
				// Check if we already have this issue for this field
				found := false
				for j, existingIssue := range report.Issues {
					if existingIssue.Type == IssueTypeAccuracy && existingIssue.Field == field {
						report.Issues[j].Count++
						if len(report.Issues[j].Examples) < 5 {
							report.Issues[j].Examples = append(report.Issues[j].Examples, value)
						}
						found = true
						break
					}
				}
				
				if !found {
					report.Issues = append(report.Issues, issue)
				}
			}
		}
		
		// Limit to first 1000 records for performance
		if i >= 1000 {
			break
		}
	}

	if totalChecks == 0 {
		return 1.0
	}

	return float64(accurateChecks) / float64(totalChecks)
}

// assessConsistency assesses data consistency
func (c *Checker) assessConsistency(records []map[string]interface{}, report *QualityReport) float64 {
	if len(records) == 0 {
		return 1.0
	}

	// Check format consistency for each field
	fieldFormats := make(map[string]map[string]int)
	fieldTypes := make(map[string]map[string]int)

	for _, record := range records {
		for field, value := range record {
			if c.isEmptyValue(value) {
				continue
			}

			// Track value formats
			if fieldFormats[field] == nil {
				fieldFormats[field] = make(map[string]int)
			}
			format := c.detectValueFormat(value)
			fieldFormats[field][format]++

			// Track data types
			if fieldTypes[field] == nil {
				fieldTypes[field] = make(map[string]int)
			}
			dataType := reflect.TypeOf(value).String()
			fieldTypes[field][dataType]++
		}
	}

	totalFields := len(fieldFormats)
	consistentFields := 0

	for field, formats := range fieldFormats {
		// Calculate format consistency
		maxFormatCount := 0
		totalFormatCount := 0
		for _, count := range formats {
			if count > maxFormatCount {
				maxFormatCount = count
			}
			totalFormatCount += count
		}

		formatConsistency := float64(maxFormatCount) / float64(totalFormatCount)
		
		// Calculate type consistency
		types := fieldTypes[field]
		maxTypeCount := 0
		totalTypeCount := 0
		for _, count := range types {
			if count > maxTypeCount {
				maxTypeCount = count
			}
			totalTypeCount += count
		}

		typeConsistency := float64(maxTypeCount) / float64(totalTypeCount)
		
		// Overall field consistency
		fieldConsistency := (formatConsistency + typeConsistency) / 2

		if fieldConsistency >= c.config.ConsistencyThreshold {
			consistentFields++
		} else {
			issue := QualityIssue{
				Type:        IssueTypeConsistency,
				Severity:    c.getSeverity(fieldConsistency),
				Field:       field,
				Description: fmt.Sprintf("Field '%s' has inconsistent formats or types", field),
				Percentage:  (1 - fieldConsistency) * 100,
			}
			report.Issues = append(report.Issues, issue)
		}
	}

	if totalFields == 0 {
		return 1.0
	}

	return float64(consistentFields) / float64(totalFields)
}

// assessValidity assesses data validity
func (c *Checker) assessValidity(records []map[string]interface{}, report *QualityReport) float64 {
	if len(records) == 0 {
		return 1.0
	}

	totalValues := 0
	validValues := 0

	for _, record := range records {
		for field, value := range record {
			if c.isEmptyValue(value) {
				continue
			}

			totalValues++

			if c.isValidValue(field, value) {
				validValues++
			} else {
				issue := QualityIssue{
					Type:        IssueTypeValidity,
					Severity:    SeverityMedium,
					Field:       field,
					Description: fmt.Sprintf("Invalid value in field '%s'", field),
					Count:       1,
					Examples:    []interface{}{value},
				}

				// Check if we already have this issue for this field
				found := false
				for j, existingIssue := range report.Issues {
					if existingIssue.Type == IssueTypeValidity && existingIssue.Field == field {
						report.Issues[j].Count++
						if len(report.Issues[j].Examples) < 5 {
							report.Issues[j].Examples = append(report.Issues[j].Examples, value)
						}
						found = true
						break
					}
				}

				if !found {
					report.Issues = append(report.Issues, issue)
				}
			}
		}
	}

	if totalValues == 0 {
		return 1.0
	}

	return float64(validValues) / float64(totalValues)
}

// assessUniqueness assesses data uniqueness
func (c *Checker) assessUniqueness(records []map[string]interface{}, report *QualityReport) float64 {
	if len(records) <= 1 {
		return 1.0
	}

	// Check for duplicate records
	recordHashes := make(map[string]int)
	duplicateCount := 0

	for _, record := range records {
		hash := c.calculateRecordHash(record)
		recordHashes[hash]++
		if recordHashes[hash] > 1 {
			duplicateCount++
		}
	}

	uniquenessScore := float64(len(records)-duplicateCount) / float64(len(records))

	if duplicateCount > 0 {
		issue := QualityIssue{
			Type:        IssueTypeUniqueness,
			Severity:    c.getSeverity(uniquenessScore),
			Description: "Duplicate records found",
			Count:       duplicateCount,
			Percentage:  float64(duplicateCount) / float64(len(records)) * 100,
		}
		report.Issues = append(report.Issues, issue)
	}

	return uniquenessScore
}

// assessFreshness assesses data freshness
func (c *Checker) assessFreshness(records []map[string]interface{}, report *QualityReport) float64 {
	if len(records) == 0 {
		return 1.0
	}

	// Look for timestamp fields
	timestampFields := []string{"timestamp", "created_at", "updated_at", "date", "time"}
	
	freshRecords := 0
	totalRecords := 0
	now := time.Now()
	threshold := now.Add(-c.config.FreshnessThreshold)

	for _, record := range records {
		hasTimestamp := false
		
		for _, field := range timestampFields {
			if value, exists := record[field]; exists {
				hasTimestamp = true
				totalRecords++
				
				if timestamp, err := c.parseTimestamp(value); err == nil {
					if timestamp.After(threshold) {
						freshRecords++
					}
				}
				break
			}
		}
		
		if !hasTimestamp {
			// If no timestamp field, assume fresh
			totalRecords++
			freshRecords++
		}
	}

	if totalRecords == 0 {
		return 1.0
	}

	freshnessScore := float64(freshRecords) / float64(totalRecords)

	if freshnessScore < 0.9 {
		staleCount := totalRecords - freshRecords
		issue := QualityIssue{
			Type:        IssueTypeFreshness,
			Severity:    c.getSeverity(freshnessScore),
			Description: "Data freshness below threshold",
			Count:       staleCount,
			Percentage:  float64(staleCount) / float64(totalRecords) * 100,
		}
		report.Issues = append(report.Issues, issue)
	}

	return freshnessScore
}

// assessFieldQuality assesses quality for individual fields
func (c *Checker) assessFieldQuality(records []map[string]interface{}, report *QualityReport) {
	// Get all fields
	allFields := make(map[string]bool)
	for _, record := range records {
		for field := range record {
			allFields[field] = true
		}
	}

	for field := range allFields {
		fieldScore := c.calculateFieldScore(field, records)
		report.FieldScores[field] = fieldScore
	}
}

// calculateFieldScore calculates quality score for a specific field
func (c *Checker) calculateFieldScore(field string, records []map[string]interface{}) FieldScore {
	score := FieldScore{
		Field: field,
	}

	totalCount := 0
	completeCount := 0
	validCount := 0
	accurateCount := 0

	values := make([]interface{}, 0)
	formats := make(map[string]int)

	for _, record := range records {
		if value, exists := record[field]; exists {
			totalCount++
			values = append(values, value)

			// Completeness
			if !c.isEmptyValue(value) {
				completeCount++

				// Validity
				if c.isValidValue(field, value) {
					validCount++
				}

				// Accuracy
				if c.isAccurateValue(field, value) {
					accurateCount++
				}

				// Format consistency
				format := c.detectValueFormat(value)
				formats[format]++
			}
		}
	}

	// Calculate scores
	if totalCount > 0 {
		score.CompletenessScore = float64(completeCount) / float64(totalCount)
		
		if completeCount > 0 {
			score.ValidityScore = float64(validCount) / float64(completeCount)
			score.AccuracyScore = float64(accurateCount) / float64(completeCount)

			// Consistency score based on format uniformity
			if len(formats) > 0 {
				maxFormatCount := 0
				for _, count := range formats {
					if count > maxFormatCount {
						maxFormatCount = count
					}
				}
				score.ConsistencyScore = float64(maxFormatCount) / float64(completeCount)
			} else {
				score.ConsistencyScore = 1.0
			}
		} else {
			score.ValidityScore = 1.0
			score.AccuracyScore = 1.0
			score.ConsistencyScore = 1.0
		}
	} else {
		score.CompletenessScore = 1.0
		score.ValidityScore = 1.0
		score.AccuracyScore = 1.0
		score.ConsistencyScore = 1.0
	}

	// Overall score (weighted average)
	score.OverallScore = (score.CompletenessScore*0.3 + 
						 score.ValidityScore*0.25 + 
						 score.AccuracyScore*0.25 + 
						 score.ConsistencyScore*0.2)

	return score
}

// Helper methods

func (c *Checker) isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str) == ""
	}
	
	return false
}

func (c *Checker) isValidValue(field string, value interface{}) bool {
	// Basic validation rules based on field names and value patterns
	fieldLower := strings.ToLower(field)
	
	switch {
	case strings.Contains(fieldLower, "email"):
		return c.isValidEmail(value)
	case strings.Contains(fieldLower, "phone"):
		return c.isValidPhone(value)
	case strings.Contains(fieldLower, "amount") || strings.Contains(fieldLower, "price"):
		return c.isValidAmount(value)
	case strings.Contains(fieldLower, "date") || strings.Contains(fieldLower, "time"):
		return c.isValidTimestamp(value)
	default:
		return true // Assume valid if no specific rules
	}
}

func (c *Checker) isAccurateValue(field string, value interface{}) bool {
	// Check for obvious accuracy issues
	if str, ok := value.(string); ok {
		// Check for suspicious patterns
		if strings.Contains(strings.ToLower(str), "test") ||
		   strings.Contains(strings.ToLower(str), "dummy") ||
		   strings.Contains(strings.ToLower(str), "example") {
			return false
		}
		
		// Check for repeated characters (like "aaaaa" or "11111")
		if len(str) > 3 && c.hasRepeatedChars(str) {
			return false
		}
	}
	
	// Check for unrealistic numeric values
	if num, err := c.convertToFloat64(value); err == nil {
		fieldLower := strings.ToLower(field)
		if strings.Contains(fieldLower, "age") && (num < 0 || num > 150) {
			return false
		}
		if strings.Contains(fieldLower, "amount") && num < 0 {
			return false
		}
	}
	
	return true
}

func (c *Checker) detectValueFormat(value interface{}) string {
	if value == nil {
		return "null"
	}
	
	switch v := value.(type) {
	case string:
		// Detect common string formats
		if c.isValidEmail(v) {
			return "email"
		}
		if c.isValidPhone(v) {
			return "phone"
		}
		if c.isValidTimestamp(v) {
			return "timestamp"
		}
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return "numeric_string"
		}
		return "string"
	case int, int8, int16, int32, int64:
		return "integer"
	case float32, float64:
		return "float"
	case bool:
		return "boolean"
	case time.Time:
		return "datetime"
	default:
		return reflect.TypeOf(value).String()
	}
}

func (c *Checker) isValidEmail(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}
	
	// Simple email validation
	return strings.Contains(str, "@") && strings.Contains(str, ".")
}

func (c *Checker) isValidPhone(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}
	
	// Remove common phone number formatting
	cleaned := strings.ReplaceAll(str, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	
	// Check if remaining characters are digits
	if len(cleaned) < 7 || len(cleaned) > 15 {
		return false
	}
	
	for _, char := range cleaned {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

func (c *Checker) isValidAmount(value interface{}) bool {
	_, err := c.convertToFloat64(value)
	return err == nil
}

func (c *Checker) isValidTimestamp(value interface{}) bool {
	_, err := c.parseTimestamp(value)
	return err == nil
}

func (c *Checker) parseTimestamp(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		// Try common timestamp formats
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02",
			"01/02/2006",
			"2006/01/02",
		}
		
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		
		return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", v)
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type: %T", value)
	}
}

func (c *Checker) convertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func (c *Checker) hasRepeatedChars(s string) bool {
	if len(s) < 4 {
		return false
	}
	
	charCount := make(map[rune]int)
	for _, char := range s {
		charCount[char]++
	}
	
	// If any character appears in more than 70% of the string, it's likely repeated
	threshold := int(math.Ceil(float64(len(s)) * 0.7))
	for _, count := range charCount {
		if count >= threshold {
			return true
		}
	}
	
	return false
}

func (c *Checker) calculateRecordHash(record map[string]interface{}) string {
	// Simple hash calculation - in production, use a proper hash function
	keys := make([]string, 0, len(record))
	for key := range record {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", key, record[key]))
	}
	
	return strings.Join(parts, "|")
}

func (c *Checker) getSeverity(score float64) Severity {
	switch {
	case score >= 0.9:
		return SeverityLow
	case score >= 0.7:
		return SeverityMedium
	case score >= 0.5:
		return SeverityHigh
	default:
		return SeverityCritical
	}
}

func (c *Checker) generateRecommendations(report *QualityReport) {
	recommendations := []string{}
	
	// Recommendations based on overall score
	if report.OverallScore < 0.8 {
		recommendations = append(recommendations, "Overall data quality is below acceptable threshold. Consider implementing data cleansing procedures.")
	}
	
	// Recommendations based on specific issues
	for _, issue := range report.Issues {
		switch issue.Type {
		case IssueTypeCompleteness:
			if issue.Severity >= SeverityHigh {
				recommendations = append(recommendations, fmt.Sprintf("Field '%s' has significant missing data. Consider making this field required or providing default values.", issue.Field))
			}
		case IssueTypeAccuracy:
			recommendations = append(recommendations, fmt.Sprintf("Accuracy issues detected in field '%s'. Review data source and validation rules.", issue.Field))
		case IssueTypeConsistency:
			recommendations = append(recommendations, fmt.Sprintf("Format inconsistencies in field '%s'. Implement standardization rules.", issue.Field))
		case IssueTypeValidity:
			recommendations = append(recommendations, fmt.Sprintf("Invalid values found in field '%s'. Add validation constraints.", issue.Field))
		case IssueTypeUniqueness:
			recommendations = append(recommendations, "Duplicate records detected. Implement deduplication process.")
		case IssueTypeFreshness:
			recommendations = append(recommendations, "Data freshness issues detected. Consider more frequent data updates.")
		}
	}
	
	// Limit recommendations
	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}
	
	report.Recommendations = recommendations
}