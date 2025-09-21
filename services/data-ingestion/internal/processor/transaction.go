package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/aegisshield/data-ingestion/internal/database"
	"github.com/aegisshield/data-ingestion/internal/kafka"
	"github.com/aegisshield/data-ingestion/internal/metrics"
	"github.com/aegisshield/shared/models"
	pb "github.com/aegisshield/shared/proto"
	"github.com/google/uuid"
)

// TransactionProcessor handles transaction data processing
type TransactionProcessor struct {
	repository      *database.Repository
	kafkaProducer   *kafka.Producer
	metrics         *metrics.Collector
	logger          *slog.Logger
}

// NewTransactionProcessor creates a new transaction processor
func NewTransactionProcessor(
	repository *database.Repository,
	kafkaProducer *kafka.Producer,
	metrics *metrics.Collector,
	logger *slog.Logger,
) *TransactionProcessor {
	return &TransactionProcessor{
		repository:    repository,
		kafkaProducer: kafkaProducer,
		metrics:       metrics,
		logger:        logger,
	}
}

// ProcessTransaction processes a single transaction
func (p *TransactionProcessor) ProcessTransaction(ctx context.Context, transaction *pb.Transaction) error {
	start := time.Now()
	defer func() {
		p.metrics.RecordHistogram("process_transaction_stream_duration_seconds", time.Since(start).Seconds())
	}()

	p.metrics.IncrementCounter("process_transaction_stream_requests_total")

	// Validate transaction
	if err := p.validateTransaction(transaction); err != nil {
		p.metrics.IncrementCounter("process_transaction_stream_errors_total")
		p.logger.Error("transaction validation failed",
			"transaction_id", transaction.Id,
			"error", err)
		return fmt.Errorf("validation failed: %w", err)
	}

	// Enrich transaction data
	enrichedTransaction, err := p.enrichTransaction(ctx, transaction)
	if err != nil {
		p.metrics.IncrementCounter("process_transaction_stream_errors_total")
		p.logger.Error("transaction enrichment failed",
			"transaction_id", transaction.Id,
			"error", err)
		return fmt.Errorf("enrichment failed: %w", err)
	}

	// Calculate risk score
	riskScore, err := p.calculateRiskScore(enrichedTransaction)
	if err != nil {
		p.metrics.IncrementCounter("process_transaction_stream_errors_total")
		p.logger.Error("risk score calculation failed",
			"transaction_id", transaction.Id,
			"error", err)
		return fmt.Errorf("risk score calculation failed: %w", err)
	}

	// Apply business rules
	alertTriggered, businessRuleResults, err := p.applyBusinessRules(enrichedTransaction, riskScore)
	if err != nil {
		p.metrics.IncrementCounter("process_transaction_stream_errors_total")
		p.logger.Error("business rules application failed",
			"transaction_id", transaction.Id,
			"error", err)
		return fmt.Errorf("business rules application failed: %w", err)
	}

	// Store transaction
	dbTransaction := &models.Transaction{
		ID:                    uuid.MustParse(enrichedTransaction.Id),
		ExternalID:           enrichedTransaction.ExternalId,
		Amount:                enrichedTransaction.Amount,
		Currency:              enrichedTransaction.Currency,
		TransactionType:       enrichedTransaction.Type.String(),
		Timestamp:             enrichedTransaction.Timestamp.AsTime(),
		SourceAccountID:       enrichedTransaction.SourceAccountId,
		DestinationAccountID:  enrichedTransaction.DestinationAccountId,
		Description:           enrichedTransaction.Description,
		RiskScore:             riskScore,
		Status:                "processed",
		AlertTriggered:        alertTriggered,
		ProcessedAt:           time.Now(),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Store enriched data as JSON
	if enrichedData, err := json.Marshal(enrichedTransaction.EnrichedData); err == nil {
		dbTransaction.EnrichedData = enrichedData
	}

	// Store business rule results as JSON
	if businessRuleData, err := json.Marshal(businessRuleResults); err == nil {
		dbTransaction.BusinessRuleResults = businessRuleData
	}

	if err := p.repository.CreateTransaction(ctx, dbTransaction); err != nil {
		p.metrics.IncrementCounter("process_transaction_stream_errors_total")
		p.logger.Error("failed to store transaction",
			"transaction_id", transaction.Id,
			"error", err)
		return fmt.Errorf("failed to store transaction: %w", err)
	}

	// Publish event
	if err := p.publishTransactionProcessedEvent(ctx, enrichedTransaction, riskScore, alertTriggered); err != nil {
		p.logger.Error("failed to publish transaction processed event",
			"transaction_id", transaction.Id,
			"error", err)
		// Don't fail the transaction processing for event publishing failures
	}

	p.metrics.AddGauge("processed_transactions_total", 1)
	p.logger.Info("transaction processed successfully",
		"transaction_id", transaction.Id,
		"risk_score", riskScore,
		"alert_triggered", alertTriggered)

	return nil
}

// ProcessTransactionBatch processes multiple transactions in a batch
func (p *TransactionProcessor) ProcessTransactionBatch(ctx context.Context, transactions []*pb.Transaction) error {
	start := time.Now()
	defer func() {
		p.metrics.RecordHistogram("transaction_batch_size", float64(len(transactions)))
		p.metrics.RecordHistogram("process_transaction_stream_duration_seconds", time.Since(start).Seconds())
	}()

	p.logger.Info("processing transaction batch", "count", len(transactions))

	var processedCount, failedCount int

	for _, transaction := range transactions {
		if err := p.ProcessTransaction(ctx, transaction); err != nil {
			failedCount++
			p.logger.Error("failed to process transaction in batch",
				"transaction_id", transaction.Id,
				"error", err)
		} else {
			processedCount++
		}
	}

	p.logger.Info("batch processing completed",
		"total", len(transactions),
		"processed", processedCount,
		"failed", failedCount)

	return nil
}

// validateTransaction validates transaction data
func (p *TransactionProcessor) validateTransaction(transaction *pb.Transaction) error {
	if transaction == nil {
		return fmt.Errorf("transaction is nil")
	}

	if transaction.Id == "" {
		return fmt.Errorf("transaction ID is required")
	}

	if _, err := uuid.Parse(transaction.Id); err != nil {
		return fmt.Errorf("invalid transaction ID format: %w", err)
	}

	if transaction.Amount <= 0 {
		return fmt.Errorf("transaction amount must be positive")
	}

	if transaction.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if transaction.Type == pb.TransactionType_UNKNOWN {
		return fmt.Errorf("transaction type is required")
	}

	if transaction.Timestamp == nil {
		return fmt.Errorf("timestamp is required")
	}

	if transaction.SourceAccountId == "" {
		return fmt.Errorf("source account ID is required")
	}

	return nil
}

// enrichTransaction enriches transaction with additional data
func (p *TransactionProcessor) enrichTransaction(ctx context.Context, transaction *pb.Transaction) (*pb.Transaction, error) {
	// Create a copy of the transaction
	enriched := &pb.Transaction{
		Id:                   transaction.Id,
		ExternalId:           transaction.ExternalId,
		Amount:               transaction.Amount,
		Currency:             transaction.Currency,
		Type:                 transaction.Type,
		Timestamp:            transaction.Timestamp,
		SourceAccountId:      transaction.SourceAccountId,
		DestinationAccountId: transaction.DestinationAccountId,
		Description:          transaction.Description,
		EnrichedData:         make(map[string]string),
	}

	// Copy existing enriched data
	for k, v := range transaction.EnrichedData {
		enriched.EnrichedData[k] = v
	}

	// Add enrichment data
	enriched.EnrichedData["processing_timestamp"] = time.Now().Format(time.RFC3339)
	enriched.EnrichedData["processor_id"] = "data-ingestion-service"

	// Enrich with transaction patterns
	enriched.EnrichedData["transaction_pattern"] = p.detectTransactionPattern(transaction)

	// Enrich with geographic data (if available)
	if geoData := p.enrichGeographicData(transaction); geoData != "" {
		enriched.EnrichedData["geographic_info"] = geoData
	}

	// Enrich with time-based patterns
	enriched.EnrichedData["time_pattern"] = p.analyzeTimePattern(transaction)

	// Enrich with amount patterns
	enriched.EnrichedData["amount_pattern"] = p.analyzeAmountPattern(transaction)

	return enriched, nil
}

// calculateRiskScore calculates risk score for the transaction
func (p *TransactionProcessor) calculateRiskScore(transaction *pb.Transaction) (float64, error) {
	var riskScore float64

	// Base risk score
	riskScore = 0.1

	// Amount-based risk
	if transaction.Amount > 10000 {
		riskScore += 0.3
	} else if transaction.Amount > 5000 {
		riskScore += 0.2
	} else if transaction.Amount > 1000 {
		riskScore += 0.1
	}

	// Time-based risk (off-hours transactions are riskier)
	transactionTime := transaction.Timestamp.AsTime()
	hour := transactionTime.Hour()
	if hour < 6 || hour > 22 {
		riskScore += 0.2
	}

	// Weekend transactions
	if transactionTime.Weekday() == time.Saturday || transactionTime.Weekday() == time.Sunday {
		riskScore += 0.1
	}

	// Transaction type risk
	switch transaction.Type {
	case pb.TransactionType_WIRE_TRANSFER:
		riskScore += 0.3
	case pb.TransactionType_CASH_WITHDRAWAL:
		riskScore += 0.2
	case pb.TransactionType_ONLINE_PURCHASE:
		riskScore += 0.1
	}

	// Cross-border transactions (if destination account is different country)
	if strings.Contains(transaction.Description, "international") || 
	   strings.Contains(strings.ToLower(transaction.Description), "foreign") {
		riskScore += 0.4
	}

	// Round trips (same day reverse transactions)
	// This would require database lookup in a real implementation
	if p.isRoundTripTransaction(transaction) {
		riskScore += 0.5
	}

	// Normalize risk score to 0-1 range
	if riskScore > 1.0 {
		riskScore = 1.0
	}

	return riskScore, nil
}

// applyBusinessRules applies business rules to the transaction
func (p *TransactionProcessor) applyBusinessRules(transaction *pb.Transaction, riskScore float64) (bool, map[string]interface{}, error) {
	businessRuleResults := make(map[string]interface{})
	alertTriggered := false

	// Rule 1: High-value transaction rule
	if transaction.Amount > 10000 {
		businessRuleResults["high_value_transaction"] = true
		businessRuleResults["high_value_threshold"] = 10000
		alertTriggered = true
	}

	// Rule 2: High risk score rule
	if riskScore > 0.7 {
		businessRuleResults["high_risk_score"] = true
		businessRuleResults["risk_score"] = riskScore
		businessRuleResults["risk_threshold"] = 0.7
		alertTriggered = true
	}

	// Rule 3: Structuring detection (multiple transactions just under reporting threshold)
	if transaction.Amount >= 9000 && transaction.Amount < 10000 {
		businessRuleResults["potential_structuring"] = true
		businessRuleResults["structuring_threshold"] = 10000
		alertTriggered = true
	}

	// Rule 4: Off-hours high-value transaction
	transactionTime := transaction.Timestamp.AsTime()
	hour := transactionTime.Hour()
	if (hour < 6 || hour > 22) && transaction.Amount > 5000 {
		businessRuleResults["off_hours_high_value"] = true
		businessRuleResults["transaction_hour"] = hour
		alertTriggered = true
	}

	// Rule 5: International wire transfer
	if transaction.Type == pb.TransactionType_WIRE_TRANSFER && 
	   (strings.Contains(transaction.Description, "international") || 
		strings.Contains(strings.ToLower(transaction.Description), "foreign")) {
		businessRuleResults["international_wire"] = true
		alertTriggered = true
	}

	// Rule 6: Cash-intensive business patterns
	if transaction.Type == pb.TransactionType_CASH_DEPOSIT && transaction.Amount > 5000 {
		businessRuleResults["large_cash_deposit"] = true
		alertTriggered = true
	}

	businessRuleResults["alert_triggered"] = alertTriggered
	businessRuleResults["rules_evaluated"] = 6
	businessRuleResults["evaluation_timestamp"] = time.Now().Format(time.RFC3339)

	return alertTriggered, businessRuleResults, nil
}

// detectTransactionPattern detects patterns in the transaction
func (p *TransactionProcessor) detectTransactionPattern(transaction *pb.Transaction) string {
	amount := transaction.Amount
	transactionTime := transaction.Timestamp.AsTime()
	hour := transactionTime.Hour()

	// Round amounts pattern
	if amount == float64(int64(amount)) && int64(amount)%1000 == 0 {
		return "round_amount"
	}

	// Business hours pattern
	if hour >= 9 && hour <= 17 {
		return "business_hours"
	}

	// Off-hours pattern
	if hour < 6 || hour > 22 {
		return "off_hours"
	}

	// Weekend pattern
	if transactionTime.Weekday() == time.Saturday || transactionTime.Weekday() == time.Sunday {
		return "weekend"
	}

	return "normal"
}

// enrichGeographicData enriches with geographic information
func (p *TransactionProcessor) enrichGeographicData(transaction *pb.Transaction) string {
	// In a real implementation, this would lookup geographic data
	// based on account information or transaction details
	if strings.Contains(strings.ToLower(transaction.Description), "atm") {
		return "atm_transaction"
	}
	
	if strings.Contains(strings.ToLower(transaction.Description), "online") {
		return "online_transaction"
	}

	return "unknown_location"
}

// analyzeTimePattern analyzes time-based patterns
func (p *TransactionProcessor) analyzeTimePattern(transaction *pb.Transaction) string {
	transactionTime := transaction.Timestamp.AsTime()
	hour := transactionTime.Hour()

	if hour >= 6 && hour < 12 {
		return "morning"
	} else if hour >= 12 && hour < 18 {
		return "afternoon"
	} else if hour >= 18 && hour < 22 {
		return "evening"
	} else {
		return "late_night"
	}
}

// analyzeAmountPattern analyzes amount-based patterns
func (p *TransactionProcessor) analyzeAmountPattern(transaction *pb.Transaction) string {
	amount := transaction.Amount

	if amount < 100 {
		return "micro"
	} else if amount < 1000 {
		return "small"
	} else if amount < 10000 {
		return "medium"
	} else if amount < 100000 {
		return "large"
	} else {
		return "very_large"
	}
}

// isRoundTripTransaction checks if this is a round trip transaction
func (p *TransactionProcessor) isRoundTripTransaction(transaction *pb.Transaction) bool {
	// In a real implementation, this would query the database for
	// reverse transactions within a time window
	// For now, return false as placeholder
	return false
}

// publishTransactionProcessedEvent publishes a transaction processed event
func (p *TransactionProcessor) publishTransactionProcessedEvent(ctx context.Context, transaction *pb.Transaction, riskScore float64, alertTriggered bool) error {
	event := &pb.TransactionProcessedEvent{
		TransactionId:  transaction.Id,
		Amount:         transaction.Amount,
		Currency:       transaction.Currency,
		Type:           transaction.Type,
		RiskScore:      riskScore,
		AlertTriggered: alertTriggered,
		ProcessedAt:    time.Now().Unix(),
		ProcessorId:    "data-ingestion-service",
		EnrichedData:   transaction.EnrichedData,
	}

	return p.kafkaProducer.PublishTransactionProcessedEvent(ctx, event)
}