package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/entity-resolution/internal/resolver"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

// Producer wraps Kafka producer for entity resolution events
type Producer struct {
	producer sarama.SyncProducer
	config   config.KafkaConfig
	logger   *slog.Logger
}

// Consumer wraps Kafka consumer for processing entity resolution requests
type Consumer struct {
	consumer sarama.ConsumerGroup
	resolver *resolver.EntityResolver
	config   config.KafkaConfig
	logger   *slog.Logger
}

// EntityResolutionEvent represents an entity resolution event
type EntityResolutionEvent struct {
	EventID         string                 `json:"event_id"`
	EventType       string                 `json:"event_type"`
	EntityID        string                 `json:"entity_id"`
	EntityType      string                 `json:"entity_type"`
	Name            string                 `json:"name,omitempty"`
	Identifiers     map[string]interface{} `json:"identifiers,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	ConfidenceScore float64                `json:"confidence_score"`
	IsNewEntity     bool                   `json:"is_new_entity,omitempty"`
	MatchedEntities []string               `json:"matched_entities,omitempty"`
	SourceID        string                 `json:"source_id,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EntityLinkEvent represents an entity link creation event
type EntityLinkEvent struct {
	EventID         string                 `json:"event_id"`
	LinkID          string                 `json:"link_id"`
	SourceEntityID  string                 `json:"source_entity_id"`
	TargetEntityID  string                 `json:"target_entity_id"`
	LinkType        string                 `json:"link_type"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	ConfidenceScore float64                `json:"confidence_score"`
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// TransactionEvent represents a transaction for entity resolution
type TransactionEvent struct {
	TransactionID   string                 `json:"transaction_id"`
	EntityType      string                 `json:"entity_type"`
	Name            string                 `json:"name,omitempty"`
	Identifiers     map[string]interface{} `json:"identifiers,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	SourceID        string                 `json:"source_id,omitempty"`
	ProcessingMode  string                 `json:"processing_mode"` // "realtime", "batch"
	Priority        int                    `json:"priority"`        // 1-10, higher = more urgent
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// NewProducer creates a new Kafka producer
func NewProducer(config config.KafkaConfig, logger *slog.Logger) (*Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Retry.Max = 3
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Compression = sarama.CompressionSnappy

	brokers := strings.Split(config.Brokers, ",")
	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		config:   config,
		logger:   logger,
	}, nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// PublishEntityResolved publishes an entity resolution event
func (p *Producer) PublishEntityResolved(ctx context.Context, result *resolver.ResolutionResult, request *resolver.ResolutionRequest) error {
	event := &EntityResolutionEvent{
		EventID:         uuid.New().String(),
		EventType:       "entity.resolved",
		EntityID:        result.EntityID,
		EntityType:      request.EntityType,
		Name:            request.Name,
		Identifiers:     request.Identifiers,
		Attributes:      request.Attributes,
		ConfidenceScore: result.ConfidenceScore,
		IsNewEntity:     result.IsNewEntity,
		SourceID:        request.SourceID,
		Timestamp:       time.Now(),
	}

	// Add matched entity IDs
	for _, match := range result.MatchedEntities {
		event.MatchedEntities = append(event.MatchedEntities, match.EntityID)
	}

	return p.publishEvent(ctx, p.config.EntityResolvedTopic, event.EventID, event)
}

// PublishEntityCreated publishes an entity creation event
func (p *Producer) PublishEntityCreated(ctx context.Context, entityID, entityType, name string, identifiers, attributes map[string]interface{}) error {
	event := &EntityResolutionEvent{
		EventID:     uuid.New().String(),
		EventType:   "entity.created",
		EntityID:    entityID,
		EntityType:  entityType,
		Name:        name,
		Identifiers: identifiers,
		Attributes:  attributes,
		IsNewEntity: true,
		Timestamp:   time.Now(),
	}

	return p.publishEvent(ctx, p.config.EntityResolvedTopic, event.EventID, event)
}

// PublishEntityUpdated publishes an entity update event
func (p *Producer) PublishEntityUpdated(ctx context.Context, entityID, entityType, name string, identifiers, attributes map[string]interface{}) error {
	event := &EntityResolutionEvent{
		EventID:     uuid.New().String(),
		EventType:   "entity.updated",
		EntityID:    entityID,
		EntityType:  entityType,
		Name:        name,
		Identifiers: identifiers,
		Attributes:  attributes,
		IsNewEntity: false,
		Timestamp:   time.Now(),
	}

	return p.publishEvent(ctx, p.config.EntityResolvedTopic, event.EventID, event)
}

// PublishEntityLinkCreated publishes an entity link creation event
func (p *Producer) PublishEntityLinkCreated(ctx context.Context, linkID, sourceID, targetID, linkType string, properties map[string]interface{}, confidence float64) error {
	event := &EntityLinkEvent{
		EventID:         uuid.New().String(),
		LinkID:          linkID,
		SourceEntityID:  sourceID,
		TargetEntityID:  targetID,
		LinkType:        linkType,
		Properties:      properties,
		ConfidenceScore: confidence,
		Timestamp:       time.Now(),
	}

	return p.publishEvent(ctx, p.config.EntityLinkTopic, event.EventID, event)
}

// PublishBatchJobStatus publishes batch job status updates
func (p *Producer) PublishBatchJobStatus(ctx context.Context, job *resolver.BatchResolutionJob) error {
	event := map[string]interface{}{
		"event_id":     uuid.New().String(),
		"event_type":   "batch.job.status",
		"job_id":       job.JobID,
		"status":       job.Status,
		"progress":     job.Progress,
		"total":        job.Total,
		"started_at":   job.StartedAt,
		"completed_at": job.CompletedAt,
		"errors":       job.Errors,
		"timestamp":    time.Now(),
	}

	return p.publishEvent(ctx, p.config.BatchJobTopic, job.JobID, event)
}

// publishEvent publishes an event to the specified topic
func (p *Producer) publishEvent(ctx context.Context, topic, key string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("content-type"),
				Value: []byte("application/json"),
			},
			{
				Key:   []byte("event-time"),
				Value: []byte(time.Now().Format(time.RFC3339)),
			},
		},
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		p.logger.Error("Failed to publish event",
			"topic", topic,
			"key", key,
			"error", err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Info("Event published",
		"topic", topic,
		"key", key,
		"partition", partition,
		"offset", offset)

	return nil
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config config.KafkaConfig, resolver *resolver.EntityResolver, logger *slog.Logger) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Group.Session.Timeout = 10 * time.Second
	saramaConfig.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	brokers := strings.Split(config.Brokers, ",")
	consumer, err := sarama.NewConsumerGroup(brokers, config.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		resolver: resolver,
		config:   config,
		logger:   logger,
	}, nil
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	return c.consumer.Close()
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	topics := []string{c.config.TransactionTopic}
	
	handler := &consumerGroupHandler{
		consumer: c,
		logger:   c.logger,
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Kafka consumer context cancelled")
			return ctx.Err()
		default:
			if err := c.consumer.Consume(ctx, topics, handler); err != nil {
				c.logger.Error("Kafka consumer error", "error", err)
				return err
			}
		}
	}
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	consumer *Consumer
	logger   *slog.Logger
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group setup")
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group cleanup")
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			if err := h.processMessage(session.Context(), message); err != nil {
				h.logger.Error("Failed to process message",
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset,
					"error", err)
			} else {
				session.MarkMessage(message, "")
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *consumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	h.logger.Info("Processing message",
		"topic", message.Topic,
		"partition", message.Partition,
		"offset", message.Offset)

	switch message.Topic {
	case h.consumer.config.TransactionTopic:
		return h.processTransactionEvent(ctx, message)
	default:
		h.logger.Warn("Unknown topic", "topic", message.Topic)
		return nil
	}
}

func (h *consumerGroupHandler) processTransactionEvent(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event TransactionEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal transaction event: %w", err)
	}

	h.logger.Info("Processing transaction event",
		"transaction_id", event.TransactionID,
		"entity_type", event.EntityType,
		"processing_mode", event.ProcessingMode)

	// Convert to resolution request
	request := &resolver.ResolutionRequest{
		EntityType:  event.EntityType,
		Name:        event.Name,
		Identifiers: event.Identifiers,
		Attributes:  event.Attributes,
		SourceID:    event.SourceID,
	}

	// Process based on mode
	switch event.ProcessingMode {
	case "realtime":
		return h.processRealtimeTransaction(ctx, &event, request)
	case "batch":
		return h.processBatchTransaction(ctx, &event, request)
	default:
		return h.processRealtimeTransaction(ctx, &event, request)
	}
}

func (h *consumerGroupHandler) processRealtimeTransaction(ctx context.Context, event *TransactionEvent, request *resolver.ResolutionRequest) error {
	// Resolve entity
	result, err := h.consumer.resolver.ResolveEntity(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to resolve entity: %w", err)
	}

	h.logger.Info("Entity resolved",
		"transaction_id", event.TransactionID,
		"entity_id", result.EntityID,
		"is_new_entity", result.IsNewEntity,
		"confidence_score", result.ConfidenceScore)

	return nil
}

func (h *consumerGroupHandler) processBatchTransaction(ctx context.Context, event *TransactionEvent, request *resolver.ResolutionRequest) error {
	// For batch processing, we could collect multiple requests and process them together
	// For now, process individually but mark as batch
	result, err := h.consumer.resolver.ResolveEntity(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to resolve entity in batch: %w", err)
	}

	h.logger.Info("Entity resolved in batch",
		"transaction_id", event.TransactionID,
		"entity_id", result.EntityID,
		"is_new_entity", result.IsNewEntity,
		"confidence_score", result.ConfidenceScore)

	return nil
}

// Helper functions for external systems to publish events

// PublishTransactionForResolution publishes a transaction for entity resolution
func PublishTransactionForResolution(producer *Producer, entityType, name string, identifiers, attributes map[string]interface{}, mode string, priority int) error {
	event := &TransactionEvent{
		TransactionID:  uuid.New().String(),
		EntityType:     entityType,
		Name:           name,
		Identifiers:    identifiers,
		Attributes:     attributes,
		ProcessingMode: mode,
		Priority:       priority,
		Timestamp:      time.Now(),
	}

	return producer.publishEvent(context.Background(), producer.config.TransactionTopic, event.TransactionID, event)
}

// PublishTransactionBatch publishes multiple transactions for batch processing
func PublishTransactionBatch(producer *Producer, transactions []*TransactionEvent) error {
	for _, transaction := range transactions {
		if err := producer.publishEvent(context.Background(), producer.config.TransactionTopic, transaction.TransactionID, transaction); err != nil {
			return err
		}
	}
	return nil
}