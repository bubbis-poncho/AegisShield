package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aegisshield/data-integration/internal/config"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.uber.org/zap"
)

// Producer handles Kafka message production
type Producer struct {
	producer *kafka.Producer
	config   config.KafkaConfig
	logger   *zap.Logger
}

// Consumer handles Kafka message consumption
type Consumer struct {
	consumer   *kafka.Consumer
	config     config.KafkaConfig
	logger     *zap.Logger
	processor  MessageProcessor
}

// MessageProcessor defines the interface for processing messages
type MessageProcessor interface {
	ProcessMessage(ctx context.Context, message *Message) error
}

// Message represents a Kafka message
type Message struct {
	Topic     string                 `json:"topic"`
	Key       string                 `json:"key,omitempty"`
	Value     interface{}            `json:"value"`
	Headers   map[string]string      `json:"headers,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewProducer creates a new Kafka producer
func NewProducer(config config.KafkaConfig, logger *zap.Logger) (*Producer, error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": config.Brokers,
		"client.id":         "data-integration-producer",
		"acks":             "all",
		"retries":          config.MaxRetries,
		"batch.size":       16384,
		"linger.ms":        10,
		"compression.type": "snappy",
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		config:   config,
		logger:   logger,
	}, nil
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config config.KafkaConfig, processor MessageProcessor, logger *zap.Logger) (*Consumer, error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": config.Brokers,
		"group.id":          config.GroupID,
		"client.id":         "data-integration-consumer",
		"auto.offset.reset": "earliest",
		"enable.auto.commit": false,
	}

	consumer, err := kafka.NewConsumer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &Consumer{
		consumer:  consumer,
		config:    config,
		logger:    logger,
		processor: processor,
	}, nil
}

// SendMessage sends a message to Kafka
func (p *Producer) SendMessage(ctx context.Context, message *Message) error {
	// Convert value to JSON
	valueBytes, err := json.Marshal(message.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal message value: %w", err)
	}

	// Create Kafka message
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &message.Topic,
			Partition: kafka.PartitionAny,
		},
		Value: valueBytes,
		Headers: p.convertHeaders(message.Headers),
		Timestamp: message.Timestamp,
	}

	if message.Key != "" {
		kafkaMsg.Key = []byte(message.Key)
	}

	// Send message
	deliveryChan := make(chan kafka.Event)
	defer close(deliveryChan)

	err = p.producer.Produce(kafkaMsg, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery confirmation
	select {
	case e := <-deliveryChan:
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				return fmt.Errorf("message delivery failed: %w", ev.TopicPartition.Error)
			}
			p.logger.Debug("Message delivered",
				zap.String("topic", *ev.TopicPartition.Topic),
				zap.Int32("partition", ev.TopicPartition.Partition),
				zap.Int64("offset", int64(ev.TopicPartition.Offset)))
		default:
			return fmt.Errorf("unexpected event type: %T", e)
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(p.config.ProducerTimeout) * time.Second):
		return fmt.Errorf("message delivery timeout")
	}

	return nil
}

// SendDataEvent sends a data processing event
func (p *Producer) SendDataEvent(ctx context.Context, eventType string, data interface{}, metadata map[string]interface{}) error {
	topic := p.getTopicForEventType(eventType)
	
	message := &Message{
		Topic:     topic,
		Key:       fmt.Sprintf("%s_%d", eventType, time.Now().Unix()),
		Value:     data,
		Headers:   map[string]string{"event_type": eventType},
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	return p.SendMessage(ctx, message)
}

// SendProcessedData sends processed data event
func (p *Producer) SendProcessedData(ctx context.Context, data interface{}, jobID string) error {
	return p.SendDataEvent(ctx, "processed_data", data, map[string]interface{}{
		"job_id": jobID,
	})
}

// SendValidationError sends validation error event
func (p *Producer) SendValidationError(ctx context.Context, errors []interface{}, jobID string) error {
	return p.SendDataEvent(ctx, "validation_error", errors, map[string]interface{}{
		"job_id": jobID,
	})
}

// SendQualityMetrics sends data quality metrics event
func (p *Producer) SendQualityMetrics(ctx context.Context, metrics interface{}, jobID string) error {
	return p.SendDataEvent(ctx, "quality_metrics", metrics, map[string]interface{}{
		"job_id": jobID,
	})
}

// SendLineageEvent sends lineage tracking event
func (p *Producer) SendLineageEvent(ctx context.Context, lineage interface{}, jobID string) error {
	return p.SendDataEvent(ctx, "lineage", lineage, map[string]interface{}{
		"job_id": jobID,
	})
}

// SendSchemaChange sends schema change event
func (p *Producer) SendSchemaChange(ctx context.Context, schemaChange interface{}, source string) error {
	return p.SendDataEvent(ctx, "schema_change", schemaChange, map[string]interface{}{
		"source": source,
	})
}

// Start starts the consumer
func (c *Consumer) Start(ctx context.Context) error {
	// Subscribe to topics
	topics := c.getSubscriptionTopics()
	
	c.logger.Info("Subscribing to topics", zap.Strings("topics", topics))
	
	err := c.consumer.SubscribeTopics(topics, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	// Start consuming messages
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer context cancelled")
			return ctx.Err()
		default:
			msg, err := c.consumer.ReadMessage(time.Duration(c.config.ConsumerTimeout) * time.Second)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				c.logger.Error("Error reading message", zap.Error(err))
				continue
			}

			// Process message
			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("Error processing message", zap.Error(err))
				continue
			}

			// Commit offset
			if _, err := c.consumer.CommitMessage(msg); err != nil {
				c.logger.Error("Error committing message", zap.Error(err))
			}
		}
	}
}

// Close closes the producer
func (p *Producer) Close() {
	if p.producer != nil {
		p.producer.Close()
	}
}

// Close closes the consumer
func (c *Consumer) Close() {
	if c.consumer != nil {
		c.consumer.Close()
	}
}

// Helper methods

func (p *Producer) convertHeaders(headers map[string]string) []kafka.Header {
	var kafkaHeaders []kafka.Header
	for key, value := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}
	return kafkaHeaders
}

func (p *Producer) getTopicForEventType(eventType string) string {
	switch eventType {
	case "processed_data":
		return p.config.Topics.ProcessedData
	case "validation_error":
		return p.config.Topics.ValidationErrors
	case "quality_metrics":
		return p.config.Topics.QualityMetrics
	case "lineage":
		return p.config.Topics.DataLineage
	case "schema_change":
		return p.config.Topics.SchemaChanges
	default:
		return p.config.Topics.RawData
	}
}

func (c *Consumer) getSubscriptionTopics() []string {
	return []string{
		c.config.Topics.RawData,
		c.config.Topics.ValidationErrors,
		c.config.Topics.SchemaChanges,
	}
}

func (c *Consumer) processMessage(ctx context.Context, kafkaMsg *kafka.Message) error {
	// Convert Kafka message to internal message format
	message := &Message{
		Topic:     *kafkaMsg.TopicPartition.Topic,
		Timestamp: kafkaMsg.Timestamp,
		Headers:   c.convertKafkaHeaders(kafkaMsg.Headers),
	}

	if kafkaMsg.Key != nil {
		message.Key = string(kafkaMsg.Key)
	}

	// Unmarshal value
	if err := json.Unmarshal(kafkaMsg.Value, &message.Value); err != nil {
		return fmt.Errorf("failed to unmarshal message value: %w", err)
	}

	c.logger.Debug("Processing message",
		zap.String("topic", message.Topic),
		zap.String("key", message.Key))

	// Process message using the processor
	return c.processor.ProcessMessage(ctx, message)
}

func (c *Consumer) convertKafkaHeaders(headers []kafka.Header) map[string]string {
	result := make(map[string]string)
	for _, header := range headers {
		result[header.Key] = string(header.Value)
	}
	return result
}

// ETL Pipeline message processor implementation
type ETLMessageProcessor struct {
	pipeline interface{} // ETL pipeline interface
	logger   *zap.Logger
}

// NewETLMessageProcessor creates a new ETL message processor
func NewETLMessageProcessor(pipeline interface{}, logger *zap.Logger) *ETLMessageProcessor {
	return &ETLMessageProcessor{
		pipeline: pipeline,
		logger:   logger,
	}
}

// ProcessMessage processes a Kafka message through the ETL pipeline
func (p *ETLMessageProcessor) ProcessMessage(ctx context.Context, message *Message) error {
	p.logger.Info("Processing ETL message",
		zap.String("topic", message.Topic),
		zap.String("key", message.Key))

	// Route message based on topic
	switch message.Topic {
	case "raw-data":
		return p.processRawData(ctx, message)
	case "validation-errors":
		return p.processValidationErrors(ctx, message)
	case "schema-changes":
		return p.processSchemaChanges(ctx, message)
	default:
		p.logger.Warn("Unknown topic", zap.String("topic", message.Topic))
		return nil
	}
}

func (p *ETLMessageProcessor) processRawData(ctx context.Context, message *Message) error {
	// This would process raw data through the ETL pipeline
	p.logger.Info("Processing raw data", zap.String("key", message.Key))
	
	// Extract data and submit to ETL pipeline
	// pipeline.ProcessData(ctx, message.Value, options)
	
	return nil
}

func (p *ETLMessageProcessor) processValidationErrors(ctx context.Context, message *Message) error {
	// This would handle validation errors
	p.logger.Info("Processing validation errors", zap.String("key", message.Key))
	return nil
}

func (p *ETLMessageProcessor) processSchemaChanges(ctx context.Context, message *Message) error {
	// This would handle schema evolution
	p.logger.Info("Processing schema changes", zap.String("key", message.Key))
	return nil
}