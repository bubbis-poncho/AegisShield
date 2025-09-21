package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/aegisshield/data-ingestion/internal/database"
	"github.com/aegisshield/data-ingestion/internal/kafka"
	"github.com/aegisshield/data-ingestion/internal/metrics"
	"github.com/aegisshield/shared/models"
	pb "github.com/aegisshield/shared/proto"
	"github.com/google/uuid"
)

// DataValidator handles data validation operations
type DataValidator struct {
	repository    *database.Repository
	kafkaProducer *kafka.Producer
	metrics       *metrics.Collector
	logger        *slog.Logger
}

// ValidationResult represents the result of data validation
type ValidationResult struct {
	IsValid       bool                   `json:"is_valid"`
	Errors        []ValidationError      `json:"errors"`
	Warnings      []ValidationWarning    `json:"warnings"`
	QualityScore  float64               `json:"quality_score"`
	ValidatedAt   time.Time             `json:"validated_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field       string `json:"field"`
	Message     string `json:"message"`
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Value       string `json:"value,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Value    string `json:"value,omitempty"`
}

// NewDataValidator creates a new data validator
func NewDataValidator(
	repository *database.Repository,
	kafkaProducer *kafka.Producer,
	metrics *metrics.Collector,
	logger *slog.Logger,
) *DataValidator {
	return &DataValidator{
		repository:    repository,
		kafkaProducer: kafkaProducer,
		metrics:       metrics,
		logger:        logger,
	}
}

// ValidateTransaction validates transaction data
func (v *DataValidator) ValidateTransaction(ctx context.Context, transaction *pb.Transaction) (*ValidationResult, error) {
	start := time.Now()
	defer func() {
		v.metrics.RecordHistogram("validate_data_duration_seconds", time.Since(start).Seconds())
	}()

	v.metrics.IncrementCounter("validate_data_requests_total")

	result := &ValidationResult{
		IsValid:      true,
		Errors:       []ValidationError{},
		Warnings:     []ValidationWarning{},
		QualityScore: 1.0,
		ValidatedAt:  time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Basic validation
	v.validateBasicFields(transaction, result)

	// Schema validation
	v.validateSchema(transaction, result)

	// Business rule validation
	v.validateBusinessRules(transaction, result)

	// Data quality checks
	v.validateDataQuality(transaction, result)

	// Calculate final quality score
	result.QualityScore = v.calculateQualityScore(result)

	// Determine if valid based on errors
	result.IsValid = len(result.Errors) == 0

	// Store validation result
	if err := v.storeValidationResult(ctx, transaction, result); err != nil {
		v.logger.Error("failed to store validation result",
			"transaction_id", transaction.Id,
			"error", err)
		// Don't fail validation for storage errors
	}

	// Publish validation event
	if err := v.publishValidationEvent(ctx, transaction, result); err != nil {
		v.logger.Error("failed to publish validation event",
			"transaction_id", transaction.Id,
			"error", err)
		// Don't fail validation for event publishing errors
	}

	if !result.IsValid {
		v.metrics.IncrementCounter("validate_data_errors_total")
		v.logger.Warn("transaction validation failed",
			"transaction_id", transaction.Id,
			"error_count", len(result.Errors),
			"warning_count", len(result.Warnings))
	} else {
		v.logger.Info("transaction validation passed",
			"transaction_id", transaction.Id,
			"quality_score", result.QualityScore,
			"warning_count", len(result.Warnings))
	}

	return result, nil
}

// ValidateTransactionBatch validates multiple transactions
func (v *DataValidator) ValidateTransactionBatch(ctx context.Context, transactions []*pb.Transaction) ([]*ValidationResult, error) {
	v.logger.Info("validating transaction batch", "count", len(transactions))

	results := make([]*ValidationResult, len(transactions))
	var validCount, invalidCount int

	for i, transaction := range transactions {
		result, err := v.ValidateTransaction(ctx, transaction)
		if err != nil {
			v.logger.Error("failed to validate transaction in batch",
				"transaction_id", transaction.Id,
				"error", err)
			// Create a failure result
			results[i] = &ValidationResult{
				IsValid:      false,
				Errors:       []ValidationError{{Field: "general", Message: err.Error(), Code: "VALIDATION_ERROR", Severity: "ERROR"}},
				Warnings:     []ValidationWarning{},
				QualityScore: 0.0,
				ValidatedAt:  time.Now(),
				Metadata:     make(map[string]interface{}),
			}
			invalidCount++
		} else {
			results[i] = result
			if result.IsValid {
				validCount++
			} else {
				invalidCount++
			}
		}
	}

	v.logger.Info("batch validation completed",
		"total", len(transactions),
		"valid", validCount,
		"invalid", invalidCount)

	return results, nil
}

// validateBasicFields validates basic required fields
func (v *DataValidator) validateBasicFields(transaction *pb.Transaction, result *ValidationResult) {
	// Transaction ID validation
	if transaction.Id == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "id",
			Message:  "Transaction ID is required",
			Code:     "REQUIRED_FIELD",
			Severity: "ERROR",
		})
	} else {
		if _, err := uuid.Parse(transaction.Id); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:    "id",
				Message:  "Transaction ID must be a valid UUID",
				Code:     "INVALID_FORMAT",
				Severity: "ERROR",
				Value:    transaction.Id,
			})
		}
	}

	// Amount validation
	if transaction.Amount <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "amount",
			Message:  "Transaction amount must be positive",
			Code:     "INVALID_VALUE",
			Severity: "ERROR",
			Value:    fmt.Sprintf("%.2f", transaction.Amount),
		})
	}

	// Currency validation
	if transaction.Currency == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "currency",
			Message:  "Currency is required",
			Code:     "REQUIRED_FIELD",
			Severity: "ERROR",
		})
	} else if !v.isValidCurrency(transaction.Currency) {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "currency",
			Message:  "Invalid currency code",
			Code:     "INVALID_FORMAT",
			Severity: "ERROR",
			Value:    transaction.Currency,
		})
	}

	// Transaction type validation
	if transaction.Type == pb.TransactionType_UNKNOWN {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "type",
			Message:  "Transaction type is required",
			Code:     "REQUIRED_FIELD",
			Severity: "ERROR",
		})
	}

	// Timestamp validation
	if transaction.Timestamp == nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "timestamp",
			Message:  "Timestamp is required",
			Code:     "REQUIRED_FIELD",
			Severity: "ERROR",
		})
	} else {
		transactionTime := transaction.Timestamp.AsTime()
		now := time.Now()

		// Check if timestamp is in the future
		if transactionTime.After(now.Add(5 * time.Minute)) {
			result.Errors = append(result.Errors, ValidationError{
				Field:    "timestamp",
				Message:  "Transaction timestamp cannot be in the future",
				Code:     "INVALID_TIMESTAMP",
				Severity: "ERROR",
				Value:    transactionTime.Format(time.RFC3339),
			})
		}

		// Check if timestamp is too old (more than 7 days)
		if transactionTime.Before(now.AddDate(0, 0, -7)) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "timestamp",
				Message: "Transaction timestamp is older than 7 days",
				Code:    "OLD_TIMESTAMP",
				Value:   transactionTime.Format(time.RFC3339),
			})
		}
	}

	// Source account validation
	if transaction.SourceAccountId == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "source_account_id",
			Message:  "Source account ID is required",
			Code:     "REQUIRED_FIELD",
			Severity: "ERROR",
		})
	} else if !v.isValidAccountID(transaction.SourceAccountId) {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "source_account_id",
			Message:  "Invalid source account ID format",
			Code:     "INVALID_FORMAT",
			Severity: "ERROR",
			Value:    transaction.SourceAccountId,
		})
	}

	// Destination account validation (optional for some transaction types)
	if transaction.DestinationAccountId != "" && !v.isValidAccountID(transaction.DestinationAccountId) {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "destination_account_id",
			Message:  "Invalid destination account ID format",
			Code:     "INVALID_FORMAT",
			Severity: "ERROR",
			Value:    transaction.DestinationAccountId,
		})
	}

	// Same account validation
	if transaction.SourceAccountId != "" && transaction.DestinationAccountId != "" &&
		transaction.SourceAccountId == transaction.DestinationAccountId {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "accounts",
			Message: "Source and destination accounts are the same",
			Code:    "SAME_ACCOUNT",
		})
	}
}

// validateSchema validates data against predefined schemas
func (v *DataValidator) validateSchema(transaction *pb.Transaction, result *ValidationResult) {
	// Validate transaction type specific requirements
	switch transaction.Type {
	case pb.TransactionType_WIRE_TRANSFER:
		if transaction.DestinationAccountId == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    "destination_account_id",
				Message:  "Destination account is required for wire transfers",
				Code:     "REQUIRED_FIELD_FOR_TYPE",
				Severity: "ERROR",
			})
		}

	case pb.TransactionType_CASH_WITHDRAWAL, pb.TransactionType_CASH_DEPOSIT:
		if transaction.DestinationAccountId != "" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "destination_account_id",
				Message: "Destination account not typically used for cash transactions",
				Code:    "UNEXPECTED_FIELD",
			})
		}

	case pb.TransactionType_DIRECT_DEBIT, pb.TransactionType_DIRECT_CREDIT:
		if transaction.DestinationAccountId == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    "destination_account_id",
				Message:  "Destination account is required for direct debit/credit",
				Code:     "REQUIRED_FIELD_FOR_TYPE",
				Severity: "ERROR",
			})
		}
	}

	// Validate amount precision (max 2 decimal places for most currencies)
	if v.hasExcessivePrecision(transaction.Amount) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "amount",
			Message: "Amount has more than 2 decimal places",
			Code:    "EXCESSIVE_PRECISION",
			Value:   fmt.Sprintf("%.6f", transaction.Amount),
		})
	}
}

// validateBusinessRules validates business-specific rules
func (v *DataValidator) validateBusinessRules(transaction *pb.Transaction, result *ValidationResult) {
	// Large transaction validation
	if transaction.Amount > 100000 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "amount",
			Message: "Large transaction amount requires additional review",
			Code:    "LARGE_AMOUNT",
			Value:   fmt.Sprintf("%.2f", transaction.Amount),
		})
	}

	// Very large transaction validation
	if transaction.Amount > 1000000 {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "amount",
			Message:  "Transaction amount exceeds maximum allowed limit",
			Code:     "AMOUNT_LIMIT_EXCEEDED",
			Severity: "ERROR",
			Value:    fmt.Sprintf("%.2f", transaction.Amount),
		})
	}

	// Micro transaction validation
	if transaction.Amount < 0.01 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "amount",
			Message: "Very small transaction amount",
			Code:    "MICRO_TRANSACTION",
			Value:   fmt.Sprintf("%.6f", transaction.Amount),
		})
	}

	// Round amount pattern detection
	if transaction.Amount == float64(int64(transaction.Amount)) && int64(transaction.Amount)%1000 == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "amount",
			Message: "Round amount pattern detected",
			Code:    "ROUND_AMOUNT_PATTERN",
			Value:   fmt.Sprintf("%.0f", transaction.Amount),
		})
	}

	// Off-hours transaction validation
	if transaction.Timestamp != nil {
		transactionTime := transaction.Timestamp.AsTime()
		hour := transactionTime.Hour()
		if hour < 6 || hour > 22 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "timestamp",
				Message: "Off-hours transaction",
				Code:    "OFF_HOURS_TRANSACTION",
				Value:   transactionTime.Format("15:04"),
			})
		}
	}

	// Weekend transaction validation
	if transaction.Timestamp != nil {
		transactionTime := transaction.Timestamp.AsTime()
		if transactionTime.Weekday() == time.Saturday || transactionTime.Weekday() == time.Sunday {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "timestamp",
				Message: "Weekend transaction",
				Code:    "WEEKEND_TRANSACTION",
				Value:   transactionTime.Weekday().String(),
			})
		}
	}
}

// validateDataQuality performs data quality checks
func (v *DataValidator) validateDataQuality(transaction *pb.Transaction, result *ValidationResult) {
	// Description quality check
	if transaction.Description == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "description",
			Message: "Missing transaction description",
			Code:    "MISSING_DESCRIPTION",
		})
	} else {
		// Check for generic descriptions
		genericDescriptions := []string{"transaction", "payment", "transfer", "deposit", "withdrawal"}
		lowerDesc := strings.ToLower(transaction.Description)
		for _, generic := range genericDescriptions {
			if lowerDesc == generic {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Field:   "description",
					Message: "Generic transaction description",
					Code:    "GENERIC_DESCRIPTION",
					Value:   transaction.Description,
				})
				break
			}
		}

		// Check description length
		if len(transaction.Description) < 5 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "description",
				Message: "Very short transaction description",
				Code:    "SHORT_DESCRIPTION",
				Value:   transaction.Description,
			})
		}
	}

	// External ID quality check
	if transaction.ExternalId == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "external_id",
			Message: "Missing external transaction ID",
			Code:    "MISSING_EXTERNAL_ID",
		})
	}

	// Check for suspicious patterns
	if v.hasSuspiciousPatterns(transaction) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "general",
			Message: "Transaction contains suspicious patterns",
			Code:    "SUSPICIOUS_PATTERN",
		})
	}
}

// calculateQualityScore calculates overall data quality score
func (v *DataValidator) calculateQualityScore(result *ValidationResult) float64 {
	score := 1.0

	// Deduct for errors (more severe penalty)
	for _, err := range result.Errors {
		switch err.Severity {
		case "ERROR":
			score -= 0.2
		case "CRITICAL":
			score -= 0.4
		}
	}

	// Deduct for warnings (less severe penalty)
	for range result.Warnings {
		score -= 0.05
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score
}

// isValidCurrency checks if the currency code is valid (ISO 4217)
func (v *DataValidator) isValidCurrency(currency string) bool {
	validCurrencies := []string{
		"USD", "EUR", "GBP", "JPY", "AUD", "CAD", "CHF", "CNY", "SEK", "NZD",
		"MXN", "SGD", "HKD", "NOK", "TRY", "ZAR", "BRL", "INR", "KRW", "RUB",
	}

	for _, valid := range validCurrencies {
		if currency == valid {
			return true
		}
	}
	return false
}

// isValidAccountID checks if the account ID format is valid
func (v *DataValidator) isValidAccountID(accountID string) bool {
	// Basic format validation - adjust based on your account ID format
	// This example assumes alphanumeric IDs between 8-20 characters
	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{8,20}$`, accountID)
	return matched
}

// hasExcessivePrecision checks if amount has more than 2 decimal places
func (v *DataValidator) hasExcessivePrecision(amount float64) bool {
	// Convert to string and check decimal places
	str := fmt.Sprintf("%.6f", amount)
	parts := strings.Split(str, ".")
	if len(parts) != 2 {
		return false
	}
	
	// Remove trailing zeros
	decimal := strings.TrimRight(parts[1], "0")
	return len(decimal) > 2
}

// hasSuspiciousPatterns checks for suspicious transaction patterns
func (v *DataValidator) hasSuspiciousPatterns(transaction *pb.Transaction) bool {
	// Check for structuring patterns (amounts just under reporting thresholds)
	if transaction.Amount >= 9000 && transaction.Amount < 10000 {
		return true
	}

	// Check for suspicious descriptions
	suspiciousKeywords := []string{"cash", "bearer", "anonymous", "untraceable"}
	lowerDesc := strings.ToLower(transaction.Description)
	for _, keyword := range suspiciousKeywords {
		if strings.Contains(lowerDesc, keyword) {
			return true
		}
	}

	return false
}

// storeValidationResult stores the validation result in the database
func (v *DataValidator) storeValidationResult(ctx context.Context, transaction *pb.Transaction, result *ValidationResult) error {
	// Convert errors to JSON
	errorsJSON, err := json.Marshal(result.Errors)
	if err != nil {
		return fmt.Errorf("failed to marshal errors: %w", err)
	}

	// Convert warnings to JSON
	warningsJSON, err := json.Marshal(result.Warnings)
	if err != nil {
		return fmt.Errorf("failed to marshal warnings: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(result.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	validationError := &models.ValidationError{
		ID:            uuid.New(),
		TransactionID: uuid.MustParse(transaction.Id),
		ErrorType:     "VALIDATION",
		ErrorMessage:  fmt.Sprintf("Validation result: %d errors, %d warnings", len(result.Errors), len(result.Warnings)),
		Severity:      v.getSeverity(result),
		Field:         "transaction",
		ValidatedAt:   result.ValidatedAt,
		Errors:        errorsJSON,
		Warnings:      warningsJSON,
		QualityScore:  result.QualityScore,
		IsValid:       result.IsValid,
		Metadata:      metadataJSON,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return v.repository.CreateValidationError(ctx, validationError)
}

// getSeverity determines the overall severity based on validation results
func (v *DataValidator) getSeverity(result *ValidationResult) string {
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			if err.Severity == "CRITICAL" {
				return "CRITICAL"
			}
		}
		return "ERROR"
	}

	if len(result.Warnings) > 0 {
		return "WARNING"
	}

	return "INFO"
}

// publishValidationEvent publishes a validation event to Kafka
func (v *DataValidator) publishValidationEvent(ctx context.Context, transaction *pb.Transaction, result *ValidationResult) error {
	event := &pb.DataValidationEvent{
		TransactionId:  transaction.Id,
		IsValid:        result.IsValid,
		ErrorCount:     int32(len(result.Errors)),
		WarningCount:   int32(len(result.Warnings)),
		QualityScore:   result.QualityScore,
		ValidatedAt:    result.ValidatedAt.Unix(),
		ValidatorId:    "data-ingestion-service",
	}

	return v.kafkaProducer.PublishDataValidationEvent(ctx, event)
}