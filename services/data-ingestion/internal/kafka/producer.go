package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"aegisshield/services/data-ingestion/internal/config"
)

// Producer defines the Kafka producer interface
type Producer interface {
	Publish(topic, key string, message interface{}) error
	PublishBatch(topic string, messages []Message) error
	Close() error
}

// Message represents a Kafka message
type Message struct {
	Key   string
	Value interface{}
}

// KafkaProducer implements the Producer interface
type KafkaProducer struct {
	writers map[string]*kafka.Writer
	config  config.KafkaConfig
	logger  *logrus.Logger
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg config.KafkaConfig) (*KafkaProducer, error) {
	producer := &KafkaProducer{
		writers: make(map[string]*kafka.Writer),
		config:  cfg,
		logger:  logrus.New(),
	}

	// Create writers for each topic
	topics := []string{
		cfg.Topics.FileUpload,
		cfg.Topics.DataProcessing,
		cfg.Topics.DataValidation,
		cfg.Topics.TransactionFlow,
		cfg.Topics.ErrorEvents,
	}

	for _, topic := range topics {
		writer := &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchSize:    cfg.ProducerBatchSize,
			BatchTimeout: cfg.ProducerFlushTimeout,
			WriteTimeout: cfg.ProducerTimeout,
			RequiredAcks: kafka.RequireAll,
			Async:        false,
		}

		// Configure TLS/SASL if needed
		if cfg.SecurityProtocol != "PLAINTEXT" {
			// TODO: Implement TLS/SASL configuration
			producer.logger.Warn("TLS/SASL configuration not implemented yet")
		}

		producer.writers[topic] = writer
	}

	return producer, nil
}

// Publish sends a single message to the specified topic
func (p *KafkaProducer) Publish(topic, key string, message interface{}) error {
	writer, exists := p.writers[topic]
	if !exists {
		return fmt.Errorf("no writer configured for topic: %s", topic)
	}

	// Serialize message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Create Kafka message
	kafkaMessage := kafka.Message{
		Key:   []byte(key),
		Value: messageBytes,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{
				Key:   "content-type",
				Value: []byte("application/json"),
			},
			{
				Key:   "source-service",
				Value: []byte("data-ingestion"),
			},
		},
	}

	// Send message
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ProducerTimeout)
	defer cancel()

	if err := writer.WriteMessages(ctx, kafkaMessage); err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"topic": topic,
			"key":   key,
		}).Error("Failed to publish message")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"topic": topic,
		"key":   key,
	}).Debug("Message published successfully")

	return nil
}

// PublishBatch sends multiple messages to the specified topic
func (p *KafkaProducer) PublishBatch(topic string, messages []Message) error {
	writer, exists := p.writers[topic]
	if !exists {
		return fmt.Errorf("no writer configured for topic: %s", topic)
	}

	// Convert messages to Kafka messages
	kafkaMessages := make([]kafka.Message, len(messages))
	for i, msg := range messages {
		messageBytes, err := json.Marshal(msg.Value)
		if err != nil {
			return fmt.Errorf("failed to serialize message %d: %w", i, err)
		}

		kafkaMessages[i] = kafka.Message{
			Key:   []byte(msg.Key),
			Value: messageBytes,
			Time:  time.Now(),
			Headers: []kafka.Header{
				{
					Key:   "content-type",
					Value: []byte("application/json"),
				},
				{
					Key:   "source-service",
					Value: []byte("data-ingestion"),
				},
			},
		}
	}

	// Send batch
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ProducerTimeout)
	defer cancel()

	if err := writer.WriteMessages(ctx, kafkaMessages...); err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"topic":        topic,
			"message_count": len(messages),
		}).Error("Failed to publish batch")
		return fmt.Errorf("failed to publish batch: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"topic":        topic,
		"message_count": len(messages),
	}).Debug("Batch published successfully")

	return nil
}

// Close closes all Kafka writers
func (p *KafkaProducer) Close() error {
	var lastErr error
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			p.logger.WithError(err).WithField("topic", topic).Error("Failed to close writer")
			lastErr = err
		}
	}
	return lastErr
}

// PublishFileUploadEvent publishes a file upload event
func (p *KafkaProducer) PublishFileUploadEvent(fileID, fileName, fileType string, fileSize int64, uploadedBy string, metadata map[string]string) error {
	event := map[string]interface{}{
		"event_id":    fmt.Sprintf("file-upload-%s", fileID),
		"event_type":  "file_upload",
		"file_id":     fileID,
		"file_name":   fileName,
		"file_type":   fileType,
		"file_size":   fileSize,
		"uploaded_by": uploadedBy,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"metadata":    metadata,
	}

	return p.Publish(p.config.Topics.FileUpload, fileID, event)
}

// PublishDataProcessingEvent publishes a data processing event
func (p *KafkaProducer) PublishDataProcessingEvent(jobID, fileID, status string, recordsProcessed, recordsFailed int, processingTime float64) error {
	event := map[string]interface{}{
		"event_id":          fmt.Sprintf("data-processing-%s", jobID),
		"event_type":        "data_processing",
		"job_id":            jobID,
		"file_id":           fileID,
		"status":            status,
		"records_processed": recordsProcessed,
		"records_failed":    recordsFailed,
		"processing_time":   processingTime,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}

	return p.Publish(p.config.Topics.DataProcessing, jobID, event)
}

// PublishTransactionEvent publishes a transaction ingestion event
func (p *KafkaProducer) PublishTransactionEvent(transactionID, fromEntity, toEntity string, amount float64, currency, riskLevel string, riskScore float64) error {
	event := map[string]interface{}{
		"event_id":       fmt.Sprintf("transaction-%s", transactionID),
		"event_type":     "transaction_ingested",
		"transaction_id": transactionID,
		"from_entity":    fromEntity,
		"to_entity":      toEntity,
		"amount":         amount,
		"currency":       currency,
		"risk_level":     riskLevel,
		"risk_score":     riskScore,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	return p.Publish(p.config.Topics.TransactionFlow, transactionID, event)
}

// PublishValidationEvent publishes a data validation event
func (p *KafkaProducer) PublishValidationEvent(jobID string, isValid bool, errorCount int, validationErrors []map[string]interface{}) error {
	event := map[string]interface{}{
		"event_id":          fmt.Sprintf("validation-%s", jobID),
		"event_type":        "data_validation",
		"job_id":            jobID,
		"is_valid":          isValid,
		"error_count":       errorCount,
		"validation_errors": validationErrors,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}

	return p.Publish(p.config.Topics.DataValidation, jobID, event)
}

// PublishErrorEvent publishes an error event
func (p *KafkaProducer) PublishErrorEvent(component, operation, errorCode, errorMessage string, context map[string]interface{}) error {
	event := map[string]interface{}{
		"event_id":      fmt.Sprintf("error-%d", time.Now().UnixNano()),
		"event_type":    "error",
		"component":     component,
		"operation":     operation,
		"error_code":    errorCode,
		"error_message": errorMessage,
		"context":       context,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	return p.Publish(p.config.Topics.ErrorEvents, fmt.Sprintf("%s-%s", component, operation), event)
}