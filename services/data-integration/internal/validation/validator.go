package validation

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aegisshield/data-integration/internal/config"
	"go.uber.org/zap"
)

// Validator handles data validation
type Validator struct {
	config config.ValidationConfig
	logger *zap.Logger
	rules  map[string]*ValidationRule
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Field       string                 `json:"field"`
	Type        ValidationType         `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Required    bool                   `json:"required"`
}

// ValidationType represents the type of validation
type ValidationType string

const (
	ValidationTypeRequired    ValidationType = "required"
	ValidationTypeDataType    ValidationType = "data_type"
	ValidationTypeRange       ValidationType = "range"
	ValidationTypePattern     ValidationType = "pattern"
	ValidationTypeLength      ValidationType = "length"
	ValidationTypeCustom      ValidationType = "custom"
	ValidationTypeBusinessRule ValidationType = "business_rule"
)

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid        bool                   `json:"valid"`
	Errors       []ValidationError      `json:"errors,omitempty"`
	Warnings     []ValidationWarning    `json:"warnings,omitempty"`
	FieldResults map[string]FieldResult `json:"field_results"`
	RecordCount  int                    `json:"record_count"`
	ValidCount   int                    `json:"valid_count"`
	InvalidCount int                    `json:"invalid_count"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field       string `json:"field"`
	Rule        string `json:"rule"`
	Message     string `json:"message"`
	Value       interface{} `json:"value,omitempty"`
	RecordIndex int    `json:"record_index,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field       string `json:"field"`
	Rule        string `json:"rule"`
	Message     string `json:"message"`
	Value       interface{} `json:"value,omitempty"`
	RecordIndex int    `json:"record_index,omitempty"`
}

// FieldResult represents validation result for a specific field
type FieldResult struct {
	Field        string  `json:"field"`
	ValidCount   int     `json:"valid_count"`
	InvalidCount int     `json:"invalid_count"`
	ErrorRate    float64 `json:"error_rate"`
	CommonErrors []string `json:"common_errors,omitempty"`
}

// NewValidator creates a new validator
func NewValidator(config config.ValidationConfig, logger *zap.Logger) *Validator {
	validator := &Validator{
		config: config,
		logger: logger,
		rules:  make(map[string]*ValidationRule),
	}

	// Load default validation rules
	validator.loadDefaultRules()

	return validator
}

// ValidateRecords validates a slice of records
func (v *Validator) ValidateRecords(ctx context.Context, records []map[string]interface{}) ([]map[string]interface{}, []map[string]interface{}, error) {
	if !v.config.EnableSchemaValidation {
		return records, nil, nil
	}

	var validRecords []map[string]interface{}
	var invalidRecords []map[string]interface{}

	v.logger.Info("Validating records",
		zap.Int("total_records", len(records)))

	for i, record := range records {
		result := v.ValidateRecord(ctx, record, i)
		
		if result.Valid {
			validRecords = append(validRecords, record)
		} else {
			// Add validation errors to the record for debugging
			record["_validation_errors"] = result.Errors
			invalidRecords = append(invalidRecords, record)
		}
	}

	v.logger.Info("Validation completed",
		zap.Int("valid_records", len(validRecords)),
		zap.Int("invalid_records", len(invalidRecords)))

	return validRecords, invalidRecords, nil
}

// ValidateRecord validates a single record
func (v *Validator) ValidateRecord(ctx context.Context, record map[string]interface{}, recordIndex int) *ValidationResult {
	result := &ValidationResult{
		Valid:        true,
		Errors:       []ValidationError{},
		Warnings:     []ValidationWarning{},
		FieldResults: make(map[string]FieldResult),
		RecordCount:  1,
	}

	// Validate required fields
	for _, field := range v.config.RequiredFields {
		if _, exists := record[field]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:       field,
				Rule:        "required",
				Message:     fmt.Sprintf("Required field '%s' is missing", field),
				RecordIndex: recordIndex,
			})
		}
	}

	// Validate data types
	for field, expectedType := range v.config.DataTypes {
		if value, exists := record[field]; exists {
			if !v.validateDataType(value, expectedType) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:       field,
					Rule:        "data_type",
					Message:     fmt.Sprintf("Field '%s' has invalid type, expected %s", field, expectedType),
					Value:       value,
					RecordIndex: recordIndex,
				})
			}
		}
	}

	// Validate business rules
	for _, businessRule := range v.config.BusinessRules {
		if err := v.validateBusinessRule(record, businessRule, recordIndex); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:       businessRule.Field,
				Rule:        businessRule.Name,
				Message:     err.Error(),
				RecordIndex: recordIndex,
			})
		}
	}

	// Validate custom rules
	for _, rule := range v.rules {
		if err := v.validateCustomRule(record, rule, recordIndex); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:       rule.Field,
				Rule:        rule.Name,
				Message:     err.Error(),
				RecordIndex: recordIndex,
			})
		}
	}

	// Update counts
	if result.Valid {
		result.ValidCount = 1
	} else {
		result.InvalidCount = 1
	}

	return result
}

// validateDataType validates the data type of a value
func (v *Validator) validateDataType(value interface{}, expectedType string) bool {
	if value == nil {
		return true // Allow null values, use required validation for mandatory fields
	}

	switch strings.ToLower(expectedType) {
	case "string", "text":
		_, ok := value.(string)
		return ok
	case "int", "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			return true
		case float64:
			// JSON numbers are parsed as float64, check if it's a whole number
			f := value.(float64)
			return f == float64(int64(f))
		case string:
			_, err := strconv.Atoi(value.(string))
			return err == nil
		}
		return false
	case "float", "decimal", "number":
		switch value.(type) {
		case int, int8, int16, int32, int64, float32, float64:
			return true
		case string:
			_, err := strconv.ParseFloat(value.(string), 64)
			return err == nil
		}
		return false
	case "bool", "boolean":
		switch value.(type) {
		case bool:
			return true
		case string:
			s := strings.ToLower(value.(string))
			return s == "true" || s == "false" || s == "1" || s == "0"
		}
		return false
	case "datetime", "timestamp":
		switch v := value.(type) {
		case time.Time:
			return true
		case string:
			// Try to parse common datetime formats
			formats := []string{
				time.RFC3339,
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05Z",
				"2006-01-02",
			}
			for _, format := range formats {
				if _, err := time.Parse(format, v); err == nil {
					return true
				}
			}
			return false
		}
		return false
	case "array", "list":
		return reflect.TypeOf(value).Kind() == reflect.Slice
	case "object", "map":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		// Unknown type, assume valid
		return true
	}
}

// validateBusinessRule validates a business rule
func (v *Validator) validateBusinessRule(record map[string]interface{}, rule config.BusinessRule, recordIndex int) error {
	value, exists := record[rule.Field]
	if !exists {
		return nil // Field doesn't exist, skip validation
	}

	switch rule.Rule {
	case "min_value":
		if params, ok := rule.Parameters.(map[string]interface{}); ok {
			if minVal, ok := params["min"].(float64); ok {
				if numVal, err := v.convertToFloat64(value); err == nil {
					if numVal < minVal {
						return fmt.Errorf("value %v is less than minimum %v", value, minVal)
					}
				}
			}
		}

	case "max_value":
		if params, ok := rule.Parameters.(map[string]interface{}); ok {
			if maxVal, ok := params["max"].(float64); ok {
				if numVal, err := v.convertToFloat64(value); err == nil {
					if numVal > maxVal {
						return fmt.Errorf("value %v is greater than maximum %v", value, maxVal)
					}
				}
			}
		}

	case "pattern":
		if params, ok := rule.Parameters.(map[string]interface{}); ok {
			if pattern, ok := params["pattern"].(string); ok {
				if strVal, ok := value.(string); ok {
					if matched, _ := regexp.MatchString(pattern, strVal); !matched {
						return fmt.Errorf("value '%s' does not match pattern '%s'", strVal, pattern)
					}
				}
			}
		}

	case "length":
		if params, ok := rule.Parameters.(map[string]interface{}); ok {
			if strVal, ok := value.(string); ok {
				length := len(strVal)
				if minLen, ok := params["min"].(float64); ok && length < int(minLen) {
					return fmt.Errorf("string length %d is less than minimum %d", length, int(minLen))
				}
				if maxLen, ok := params["max"].(float64); ok && length > int(maxLen) {
					return fmt.Errorf("string length %d is greater than maximum %d", length, int(maxLen))
				}
			}
		}

	case "in_list":
		if params, ok := rule.Parameters.(map[string]interface{}); ok {
			if allowedValues, ok := params["values"].([]interface{}); ok {
				found := false
				for _, allowed := range allowedValues {
					if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", allowed) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("value '%v' is not in allowed list", value)
				}
			}
		}

	case "not_null":
		if value == nil || value == "" {
			return fmt.Errorf("field cannot be null or empty")
		}

	case "unique":
		// This would require checking against existing data
		// For now, just log that uniqueness check is needed
		v.logger.Debug("Uniqueness check required", zap.String("field", rule.Field))

	default:
		v.logger.Warn("Unknown business rule", zap.String("rule", rule.Rule))
	}

	return nil
}

// validateCustomRule validates a custom rule
func (v *Validator) validateCustomRule(record map[string]interface{}, rule *ValidationRule, recordIndex int) error {
	value, exists := record[rule.Field]
	if !exists && !rule.Required {
		return nil // Field doesn't exist and it's not required
	}

	if !exists && rule.Required {
		return fmt.Errorf("required field '%s' is missing", rule.Field)
	}

	switch rule.Type {
	case ValidationTypeRange:
		return v.validateRange(value, rule.Parameters)
	case ValidationTypePattern:
		return v.validatePattern(value, rule.Parameters)
	case ValidationTypeLength:
		return v.validateLength(value, rule.Parameters)
	default:
		return nil
	}
}

// validateRange validates numeric range
func (v *Validator) validateRange(value interface{}, params map[string]interface{}) error {
	numVal, err := v.convertToFloat64(value)
	if err != nil {
		return fmt.Errorf("cannot convert value to number for range validation: %w", err)
	}

	if minVal, ok := params["min"].(float64); ok && numVal < minVal {
		return fmt.Errorf("value %v is less than minimum %v", numVal, minVal)
	}

	if maxVal, ok := params["max"].(float64); ok && numVal > maxVal {
		return fmt.Errorf("value %v is greater than maximum %v", numVal, maxVal)
	}

	return nil
}

// validatePattern validates string pattern
func (v *Validator) validatePattern(value interface{}, params map[string]interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be string for pattern validation")
	}

	pattern, ok := params["pattern"].(string)
	if !ok {
		return fmt.Errorf("pattern parameter is required")
	}

	matched, err := regexp.MatchString(pattern, strVal)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if !matched {
		return fmt.Errorf("value '%s' does not match pattern '%s'", strVal, pattern)
	}

	return nil
}

// validateLength validates string length
func (v *Validator) validateLength(value interface{}, params map[string]interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be string for length validation")
	}

	length := len(strVal)

	if minLen, ok := params["min"].(float64); ok && length < int(minLen) {
		return fmt.Errorf("string length %d is less than minimum %d", length, int(minLen))
	}

	if maxLen, ok := params["max"].(float64); ok && length > int(maxLen) {
		return fmt.Errorf("string length %d is greater than maximum %d", length, int(maxLen))
	}

	return nil
}

// convertToFloat64 converts various numeric types to float64
func (v *Validator) convertToFloat64(value interface{}) (float64, error) {
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

// loadDefaultRules loads default validation rules
func (v *Validator) loadDefaultRules() {
	// Email validation rule
	v.rules["email"] = &ValidationRule{
		Name:        "email",
		Description: "Validates email address format",
		Type:        ValidationTypePattern,
		Parameters: map[string]interface{}{
			"pattern": `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
	}

	// Phone number validation rule
	v.rules["phone"] = &ValidationRule{
		Name:        "phone",
		Description: "Validates phone number format",
		Type:        ValidationTypePattern,
		Parameters: map[string]interface{}{
			"pattern": `^\+?[1-9]\d{1,14}$`, // E.164 format
		},
	}

	// Currency amount validation rule
	v.rules["currency"] = &ValidationRule{
		Name:        "currency",
		Description: "Validates currency amount (up to 2 decimal places)",
		Type:        ValidationTypePattern,
		Parameters: map[string]interface{}{
			"pattern": `^\d+(\.\d{1,2})?$`,
		},
	}

	// ISO date validation rule
	v.rules["iso_date"] = &ValidationRule{
		Name:        "iso_date",
		Description: "Validates ISO date format (YYYY-MM-DD)",
		Type:        ValidationTypePattern,
		Parameters: map[string]interface{}{
			"pattern": `^\d{4}-\d{2}-\d{2}$`,
		},
	}

	// Account number validation rule
	v.rules["account_number"] = &ValidationRule{
		Name:        "account_number",
		Description: "Validates account number format",
		Type:        ValidationTypeLength,
		Parameters: map[string]interface{}{
			"min": 8.0,
			"max": 20.0,
		},
	}
}

// AddRule adds a custom validation rule
func (v *Validator) AddRule(rule *ValidationRule) {
	v.rules[rule.Name] = rule
	v.logger.Info("Added validation rule",
		zap.String("rule_name", rule.Name),
		zap.String("description", rule.Description))
}

// RemoveRule removes a validation rule
func (v *Validator) RemoveRule(ruleName string) {
	delete(v.rules, ruleName)
	v.logger.Info("Removed validation rule", zap.String("rule_name", ruleName))
}

// GetRules returns all validation rules
func (v *Validator) GetRules() map[string]*ValidationRule {
	rules := make(map[string]*ValidationRule)
	for k, v := range v.rules {
		rules[k] = v
	}
	return rules
}

// ProfileData performs data profiling to discover patterns and quality issues
func (v *Validator) ProfileData(ctx context.Context, records []map[string]interface{}) (*DataProfile, error) {
	if !v.config.EnableDataProfiling {
		return nil, nil
	}

	profile := &DataProfile{
		RecordCount: len(records),
		Fields:      make(map[string]*FieldProfile),
		ProfiledAt:  time.Now(),
	}

	// Analyze each field
	fieldStats := make(map[string]*fieldStatistics)
	
	for _, record := range records {
		for field, value := range record {
			if _, exists := fieldStats[field]; !exists {
				fieldStats[field] = &fieldStatistics{
					values:    make(map[interface{}]int),
					nullCount: 0,
					typeCount: make(map[string]int),
				}
			}

			stats := fieldStats[field]
			stats.totalCount++

			if value == nil || value == "" {
				stats.nullCount++
			} else {
				stats.values[value]++
				stats.typeCount[reflect.TypeOf(value).String()]++

				// Update min/max for numeric values
				if numVal, err := v.convertToFloat64(value); err == nil {
					if stats.minValue == nil || numVal < *stats.minValue {
						stats.minValue = &numVal
					}
					if stats.maxValue == nil || numVal > *stats.maxValue {
						stats.maxValue = &numVal
					}
				}

				// Update string length stats
				if strVal, ok := value.(string); ok {
					length := len(strVal)
					if stats.minLength == nil || length < *stats.minLength {
						stats.minLength = &length
					}
					if stats.maxLength == nil || length > *stats.maxLength {
						stats.maxLength = &length
					}
				}
			}
		}
	}

	// Convert statistics to field profiles
	for field, stats := range fieldStats {
		fieldProfile := &FieldProfile{
			Name:         field,
			TotalCount:   stats.totalCount,
			NullCount:    stats.nullCount,
			UniqueCount:  len(stats.values),
			Completeness: float64(stats.totalCount-stats.nullCount) / float64(stats.totalCount),
			DataTypes:    stats.typeCount,
		}

		if stats.minValue != nil && stats.maxValue != nil {
			fieldProfile.MinValue = stats.minValue
			fieldProfile.MaxValue = stats.maxValue
		}

		if stats.minLength != nil && stats.maxLength != nil {
			fieldProfile.MinLength = stats.minLength
			fieldProfile.MaxLength = stats.maxLength
		}

		// Find most common values
		fieldProfile.TopValues = v.getTopValues(stats.values, 10)

		profile.Fields[field] = fieldProfile
	}

	v.logger.Info("Data profiling completed",
		zap.Int("record_count", profile.RecordCount),
		zap.Int("field_count", len(profile.Fields)))

	return profile, nil
}

// DataProfile represents data profiling results
type DataProfile struct {
	RecordCount int                       `json:"record_count"`
	Fields      map[string]*FieldProfile  `json:"fields"`
	ProfiledAt  time.Time                 `json:"profiled_at"`
}

// FieldProfile represents profiling results for a field
type FieldProfile struct {
	Name         string                 `json:"name"`
	TotalCount   int                    `json:"total_count"`
	NullCount    int                    `json:"null_count"`
	UniqueCount  int                    `json:"unique_count"`
	Completeness float64                `json:"completeness"`
	DataTypes    map[string]int         `json:"data_types"`
	MinValue     *float64               `json:"min_value,omitempty"`
	MaxValue     *float64               `json:"max_value,omitempty"`
	MinLength    *int                   `json:"min_length,omitempty"`
	MaxLength    *int                   `json:"max_length,omitempty"`
	TopValues    []ValueFrequency       `json:"top_values,omitempty"`
}

// ValueFrequency represents a value and its frequency
type ValueFrequency struct {
	Value     interface{} `json:"value"`
	Frequency int         `json:"frequency"`
	Percentage float64    `json:"percentage"`
}

type fieldStatistics struct {
	totalCount int
	nullCount  int
	values     map[interface{}]int
	typeCount  map[string]int
	minValue   *float64
	maxValue   *float64
	minLength  *int
	maxLength  *int
}

// getTopValues returns the most common values for a field
func (v *Validator) getTopValues(values map[interface{}]int, limit int) []ValueFrequency {
	type valueFreq struct {
		value interface{}
		count int
	}

	var freqs []valueFreq
	totalCount := 0

	for value, count := range values {
		freqs = append(freqs, valueFreq{value: value, count: count})
		totalCount += count
	}

	// Sort by frequency (descending)
	for i := 0; i < len(freqs)-1; i++ {
		for j := i + 1; j < len(freqs); j++ {
			if freqs[j].count > freqs[i].count {
				freqs[i], freqs[j] = freqs[j], freqs[i]
			}
		}
	}

	// Take top values
	if len(freqs) > limit {
		freqs = freqs[:limit]
	}

	// Convert to ValueFrequency
	result := make([]ValueFrequency, len(freqs))
	for i, freq := range freqs {
		result[i] = ValueFrequency{
			Value:      freq.value,
			Frequency:  freq.count,
			Percentage: float64(freq.count) / float64(totalCount) * 100,
		}
	}

	return result
}