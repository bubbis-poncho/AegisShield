package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/engine"
)

// Consumer handles Kafka message consumption
type Consumer struct {
	consumer sarama.ConsumerGroup
	engine   *engine.GraphEngine
	config   config.Config
	logger   *slog.Logger
	topics   []string
	ctx      context.Context
	cancel   context.CancelFunc
}

// Producer handles Kafka message production
type Producer struct {
	producer sarama.SyncProducer
	config   config.Config
	logger   *slog.Logger
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	engine *engine.GraphEngine,
	config config.Config,
	logger *slog.Logger,
) (*Consumer, error) {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	kafkaConfig.Consumer.Group.Session.Timeout = 10 * time.Second
	kafkaConfig.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	kafkaConfig.Consumer.Return.Errors = true

	// Configure authentication if enabled
	if config.Kafka.SASL.Enabled {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		kafkaConfig.Net.SASL.User = config.Kafka.SASL.Username
		kafkaConfig.Net.SASL.Password = config.Kafka.SASL.Password
	}

	consumer, err := sarama.NewConsumerGroup(config.Kafka.Brokers, config.Kafka.GroupID, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	topics := []string{
		config.Kafka.Topics.EntityResolved,
		config.Kafka.Topics.EntityLinked,
		config.Kafka.Topics.DataProcessed,
		config.Kafka.Topics.AnalysisRequested,
	}

	return &Consumer{
		consumer: consumer,
		engine:   engine,
		config:   config,
		logger:   logger,
		topics:   topics,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// NewProducer creates a new Kafka producer
func NewProducer(config config.Config, logger *slog.Logger) (*Producer, error) {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = 5
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Partitioner = sarama.NewRandomPartitioner

	// Configure authentication if enabled
	if config.Kafka.SASL.Enabled {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		kafkaConfig.Net.SASL.User = config.Kafka.SASL.Username
		kafkaConfig.Net.SASL.Password = config.Kafka.SASL.Password
	}

	producer, err := sarama.NewSyncProducer(config.Kafka.Brokers, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{
		producer: producer,
		config:   config,
		logger:   logger,
	}, nil
}

// Start begins consuming messages
func (c *Consumer) Start() error {
	c.logger.Info("Starting Kafka consumer",
		"topics", c.topics,
		"group_id", c.config.Kafka.GroupID)

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				if err := c.consumer.Consume(c.ctx, c.topics, c); err != nil {
					c.logger.Error("Error consuming from Kafka", "error", err)
					time.Sleep(5 * time.Second) // Wait before retrying
				}
			}
		}
	}()

	// Monitor consumer errors
	go func() {
		for {
			select {
			case err := <-c.consumer.Errors():
				c.logger.Error("Kafka consumer error", "error", err)
			case <-c.ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop() error {
	c.logger.Info("Stopping Kafka consumer")
	c.cancel()
	return c.consumer.Close()
}

// Setup implements sarama.ConsumerGroupHandler
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	c.logger.Info("Kafka consumer group session setup")
	return nil
}

// Cleanup implements sarama.ConsumerGroupHandler
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	c.logger.Info("Kafka consumer group session cleanup")
	return nil
}

// ConsumeClaim implements sarama.ConsumerGroupHandler
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			if err := c.handleMessage(message); err != nil {
				c.logger.Error("Failed to handle message",
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

// handleMessage processes incoming Kafka messages
func (c *Consumer) handleMessage(message *sarama.ConsumerMessage) error {
	c.logger.Debug("Received Kafka message",
		"topic", message.Topic,
		"partition", message.Partition,
		"offset", message.Offset)

	switch message.Topic {
	case c.config.Kafka.Topics.EntityResolved:
		return c.handleEntityResolvedEvent(message)
	case c.config.Kafka.Topics.EntityLinked:
		return c.handleEntityLinkedEvent(message)
	case c.config.Kafka.Topics.DataProcessed:
		return c.handleDataProcessedEvent(message)
	case c.config.Kafka.Topics.AnalysisRequested:
		return c.handleAnalysisRequestedEvent(message)
	default:
		c.logger.Warn("Unknown topic", "topic", message.Topic)
		return nil
	}
}

// handleEntityResolvedEvent processes entity resolution events
func (c *Consumer) handleEntityResolvedEvent(message *sarama.ConsumerMessage) error {
	var event EntityResolvedEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal entity resolved event: %w", err)
	}

	c.logger.Info("Processing entity resolved event",
		"entity_id", event.EntityID,
		"entity_type", event.EntityType)

	// Check if entity should trigger any automated analysis
	ctx := context.Background()
	if err := c.engine.ProcessEntityResolvedEvent(ctx, &event); err != nil {
		return fmt.Errorf("failed to process entity resolved event: %w", err)
	}

	return nil
}

// handleEntityLinkedEvent processes entity linking events
func (c *Consumer) handleEntityLinkedEvent(message *sarama.ConsumerMessage) error {
	var event EntityLinkedEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal entity linked event: %w", err)
	}

	c.logger.Info("Processing entity linked event",
		"source_id", event.SourceEntityID,
		"target_id", event.TargetEntityID,
		"link_type", event.LinkType)

	// Update graph with new relationship
	ctx := context.Background()
	if err := c.engine.ProcessEntityLinkedEvent(ctx, &event); err != nil {
		return fmt.Errorf("failed to process entity linked event: %w", err)
	}

	return nil
}

// handleDataProcessedEvent processes data ingestion completion events
func (c *Consumer) handleDataProcessedEvent(message *sarama.ConsumerMessage) error {
	var event DataProcessedEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal data processed event: %w", err)
	}

	c.logger.Info("Processing data processed event",
		"job_id", event.JobID,
		"entity_count", event.EntityCount)

	// Trigger automated analysis if configured
	ctx := context.Background()
	if err := c.engine.ProcessDataProcessedEvent(ctx, &event); err != nil {
		return fmt.Errorf("failed to process data processed event: %w", err)
	}

	return nil
}

// handleAnalysisRequestedEvent processes analysis request events
func (c *Consumer) handleAnalysisRequestedEvent(message *sarama.ConsumerMessage) error {
	var event AnalysisRequestedEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal analysis requested event: %w", err)
	}

	c.logger.Info("Processing analysis requested event",
		"analysis_type", event.AnalysisType,
		"entity_count", len(event.EntityIDs))

	// Execute requested analysis
	ctx := context.Background()
	if err := c.engine.ProcessAnalysisRequestedEvent(ctx, &event); err != nil {
		return fmt.Errorf("failed to process analysis requested event: %w", err)
	}

	return nil
}

// PublishAnalysisCompleted publishes analysis completion event
func (p *Producer) PublishAnalysisCompleted(ctx context.Context, event *AnalysisCompletedEvent) error {
	return p.publishEvent(ctx, p.config.Kafka.Topics.AnalysisCompleted, event)
}

// PublishInvestigationCreated publishes investigation creation event
func (p *Producer) PublishInvestigationCreated(ctx context.Context, event *InvestigationCreatedEvent) error {
	return p.publishEvent(ctx, p.config.Kafka.Topics.InvestigationCreated, event)
}

// PublishInvestigationUpdated publishes investigation update event
func (p *Producer) PublishInvestigationUpdated(ctx context.Context, event *InvestigationUpdatedEvent) error {
	return p.publishEvent(ctx, p.config.Kafka.Topics.InvestigationUpdated, event)
}

// PublishPatternDetected publishes pattern detection event
func (p *Producer) PublishPatternDetected(ctx context.Context, event *PatternDetectedEvent) error {
	return p.publishEvent(ctx, p.config.Kafka.Topics.PatternDetected, event)
}

// PublishAnomalyDetected publishes anomaly detection event
func (p *Producer) PublishAnomalyDetected(ctx context.Context, event *AnomalyDetectedEvent) error {
	return p.publishEvent(ctx, p.config.Kafka.Topics.AnomalyDetected, event)
}

// PublishNetworkMetricsCalculated publishes network metrics calculation event
func (p *Producer) PublishNetworkMetricsCalculated(ctx context.Context, event *NetworkMetricsCalculatedEvent) error {
	return p.publishEvent(ctx, p.config.Kafka.Topics.NetworkMetricsCalculated, event)
}

// publishEvent publishes an event to Kafka
func (p *Producer) publishEvent(ctx context.Context, topic string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("content-type"),
				Value: []byte("application/json"),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(fmt.Sprintf("%d", time.Now().Unix())),
			},
		},
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}

	p.logger.Debug("Published event to Kafka",
		"topic", topic,
		"partition", partition,
		"offset", offset)

	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// Event types

// EntityResolvedEvent represents an entity resolution completion
type EntityResolvedEvent struct {
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Properties    map[string]interface{} `json:"properties"`
	Confidence    float64                `json:"confidence"`
	ResolvedAt    time.Time              `json:"resolved_at"`
	ResolvedBy    string                 `json:"resolved_by"`
	SourceDataID  string                 `json:"source_data_id"`
	ProcessingID  string                 `json:"processing_id"`
}

// EntityLinkedEvent represents entity linking completion
type EntityLinkedEvent struct {
	SourceEntityID string                 `json:"source_entity_id"`
	TargetEntityID string                 `json:"target_entity_id"`
	LinkType       string                 `json:"link_type"`
	Confidence     float64                `json:"confidence"`
	Properties     map[string]interface{} `json:"properties"`
	LinkedAt       time.Time              `json:"linked_at"`
	LinkedBy       string                 `json:"linked_by"`
}

// DataProcessedEvent represents data processing completion
type DataProcessedEvent struct {
	JobID         string    `json:"job_id"`
	DataType      string    `json:"data_type"`
	EntityCount   int       `json:"entity_count"`
	ProcessedAt   time.Time `json:"processed_at"`
	ProcessedBy   string    `json:"processed_by"`
	AutoAnalyze   bool      `json:"auto_analyze"`
}

// AnalysisRequestedEvent represents an analysis request
type AnalysisRequestedEvent struct {
	RequestID     string                 `json:"request_id"`
	AnalysisType  string                 `json:"analysis_type"`
	EntityIDs     []string               `json:"entity_ids"`
	Parameters    map[string]interface{} `json:"parameters"`
	RequestedAt   time.Time              `json:"requested_at"`
	RequestedBy   string                 `json:"requested_by"`
	Priority      string                 `json:"priority"`
}

// AnalysisCompletedEvent represents analysis completion
type AnalysisCompletedEvent struct {
	JobID         string                 `json:"job_id"`
	AnalysisType  string                 `json:"analysis_type"`
	Status        string                 `json:"status"`
	EntityCount   int                    `json:"entity_count"`
	PatternCount  int                    `json:"pattern_count"`
	InsightCount  int                    `json:"insight_count"`
	CompletedAt   time.Time              `json:"completed_at"`
	Duration      time.Duration          `json:"duration"`
	Error         string                 `json:"error,omitempty"`
	Results       map[string]interface{} `json:"results"`
}

// InvestigationCreatedEvent represents investigation creation
type InvestigationCreatedEvent struct {
	InvestigationID string    `json:"investigation_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Priority        string    `json:"priority"`
	EntityCount     int       `json:"entity_count"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	AssignedTo      string    `json:"assigned_to"`
}

// InvestigationUpdatedEvent represents investigation updates
type InvestigationUpdatedEvent struct {
	InvestigationID string                 `json:"investigation_id"`
	Status          string                 `json:"status"`
	UpdateType      string                 `json:"update_type"`
	UpdatedAt       time.Time              `json:"updated_at"`
	UpdatedBy       string                 `json:"updated_by"`
	Changes         map[string]interface{} `json:"changes"`
}

// PatternDetectedEvent represents pattern detection
type PatternDetectedEvent struct {
	PatternID    string                 `json:"pattern_id"`
	PatternType  string                 `json:"pattern_type"`
	EntityIDs    []string               `json:"entity_ids"`
	Confidence   float64                `json:"confidence"`
	Severity     string                 `json:"severity"`
	DetectedAt   time.Time              `json:"detected_at"`
	Evidence     map[string]interface{} `json:"evidence"`
	Description  string                 `json:"description"`
}

// AnomalyDetectedEvent represents anomaly detection
type AnomalyDetectedEvent struct {
	AnomalyID    string                 `json:"anomaly_id"`
	AnomalyType  string                 `json:"anomaly_type"`
	EntityIDs    []string               `json:"entity_ids"`
	Severity     string                 `json:"severity"`
	Confidence   float64                `json:"confidence"`
	DetectedAt   time.Time              `json:"detected_at"`
	Description  string                 `json:"description"`
	Evidence     map[string]interface{} `json:"evidence"`
	BaselineData map[string]interface{} `json:"baseline_data"`
}

// NetworkMetricsCalculatedEvent represents network metrics calculation
type NetworkMetricsCalculatedEvent struct {
	JobID           string    `json:"job_id"`
	EntityCount     int       `json:"entity_count"`
	MetricsCount    int       `json:"metrics_count"`
	CalculatedAt    time.Time `json:"calculated_at"`
	GraphDensity    float64   `json:"graph_density"`
	ComponentCount  int       `json:"component_count"`
	CommunityCount  int       `json:"community_count"`
	AvgPathLength   float64   `json:"avg_path_length"`
	ClusteringCoeff float64   `json:"clustering_coefficient"`
}