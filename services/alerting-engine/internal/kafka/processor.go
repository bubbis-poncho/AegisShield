package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
)

// Consumer handles Kafka message consumption for event processing
type Consumer struct {
	config           *config.Config
	logger           *slog.Logger
	reader           *kafka.Reader
	ruleEngine       *engine.RuleEngine
	alertRepo        *database.AlertRepository
	notificationRepo *database.NotificationRepository
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
	messageCount     int64
	errorCount       int64
	lastProcessed    time.Time
}

// Producer handles Kafka message production for alert notifications
type Producer struct {
	config       *config.Config
	logger       *slog.Logger
	writer       *kafka.Writer
	shutdownChan chan struct{}
	wg           sync.WaitGroup
	messageCount int64
	errorCount   int64
}

// EventMessage represents an incoming event message
type EventMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertMessage represents an outgoing alert notification
type AlertMessage struct {
	AlertID     string                 `json:"alert_id"`
	RuleID      string                 `json:"rule_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Priority    string                 `json:"priority"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	cfg *config.Config,
	logger *slog.Logger,
	ruleEngine *engine.RuleEngine,
	alertRepo *database.AlertRepository,
	notificationRepo *database.NotificationRepository,
) (*Consumer, error) {
	// Configure Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		GroupID:        cfg.Kafka.Consumer.GroupID,
		Topic:          cfg.Kafka.Consumer.EventTopic,
		MinBytes:       cfg.Kafka.Consumer.MinBytes,
		MaxBytes:       cfg.Kafka.Consumer.MaxBytes,
		CommitInterval: time.Duration(cfg.Kafka.Consumer.CommitIntervalMs) * time.Millisecond,
		StartOffset:    kafka.LastOffset,
		Logger:         &KafkaLogger{logger: logger},
		ErrorLogger:    &KafkaErrorLogger{logger: logger},
	})

	consumer := &Consumer{
		config:           cfg,
		logger:           logger,
		reader:           reader,
		ruleEngine:       ruleEngine,
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
		shutdownChan:     make(chan struct{}),
	}

	return consumer, nil
}

// Start starts the Kafka consumer
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer",
		"topic", c.config.Kafka.Consumer.EventTopic,
		"group_id", c.config.Kafka.Consumer.GroupID)

	// Start consumer workers
	for i := 0; i < c.config.Kafka.Consumer.WorkerCount; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i)
	}

	// Start metrics reporter
	c.wg.Add(1)
	go c.metricsReporter(ctx)

	c.logger.Info("Kafka consumer started", "workers", c.config.Kafka.Consumer.WorkerCount)
	return nil
}

// Stop stops the Kafka consumer
func (c *Consumer) Stop() {
	c.logger.Info("Stopping Kafka consumer")
	close(c.shutdownChan)
	
	if c.reader != nil {
		c.reader.Close()
	}
	
	c.wg.Wait()
	c.logger.Info("Kafka consumer stopped")
}

// worker processes Kafka messages
func (c *Consumer) worker(ctx context.Context, workerID int) {
	defer c.wg.Done()

	c.logger.Debug("Starting Kafka consumer worker", "worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdownChan:
			return
		default:
			// Read message with timeout
			readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			message, err := c.reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					continue // Timeout is expected for graceful shutdown
				}
				c.logger.Error("Failed to read Kafka message",
					"worker_id", workerID,
					"error", err)
				c.errorCount++
				time.Sleep(1 * time.Second) // Brief pause on error
				continue
			}

			// Process message
			if err := c.processMessage(ctx, &message); err != nil {
				c.logger.Error("Failed to process Kafka message",
					"worker_id", workerID,
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset,
					"error", err)
				c.errorCount++
			} else {
				c.messageCount++
				c.lastProcessed = time.Now()
			}
		}
	}
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, message *kafka.Message) error {
	// Parse event message
	var eventMsg EventMessage
	if err := json.Unmarshal(message.Value, &eventMsg); err != nil {
		return fmt.Errorf("failed to unmarshal event message: %w", err)
	}

	c.logger.Debug("Processing event message",
		"event_id", eventMsg.ID,
		"event_type", eventMsg.Type,
		"source", eventMsg.Source)

	// Create evaluation context
	eventData := map[string]interface{}{
		"id":        eventMsg.ID,
		"type":      eventMsg.Type,
		"source":    eventMsg.Source,
		"timestamp": eventMsg.Timestamp,
	}

	// Add event data
	for k, v := range eventMsg.Data {
		eventData[k] = v
	}

	// Evaluate event against rules
	results, err := c.ruleEngine.EvaluateEvent(ctx, eventData)
	if err != nil {
		return fmt.Errorf("failed to evaluate event against rules: %w", err)
	}

	// Process matched rules
	for _, result := range results {
		if result.Matched {
			c.logger.Info("Rule matched for event",
				"event_id", eventMsg.ID,
				"rule_id", result.RuleID,
				"rule_name", result.RuleName,
				"actions", result.Actions)

			// Execute rule actions would be handled by the rule engine
			// For now, we log the match
		}
	}

	return nil
}

// metricsReporter reports consumer metrics
func (c *Consumer) metricsReporter(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdownChan:
			return
		case <-ticker.C:
			c.logger.Debug("Kafka consumer metrics",
				"messages_processed", c.messageCount,
				"errors", c.errorCount,
				"last_processed", c.lastProcessed)
		}
	}
}

// GetStats returns consumer statistics
func (c *Consumer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"messages_processed": c.messageCount,
		"errors":            c.errorCount,
		"last_processed":    c.lastProcessed,
		"is_running":        c.reader != nil,
	}
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.Config, logger *slog.Logger) (*Producer, error) {
	// Configure Kafka writer
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Kafka.Brokers...),
		Topic:                  cfg.Kafka.Producer.AlertTopic,
		Balancer:               &kafka.LeastBytes{},
		BatchSize:              cfg.Kafka.Producer.BatchSize,
		BatchTimeout:           time.Duration(cfg.Kafka.Producer.BatchTimeoutMs) * time.Millisecond,
		ReadTimeout:            time.Duration(cfg.Kafka.Producer.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout:           time.Duration(cfg.Kafka.Producer.WriteTimeoutMs) * time.Millisecond,
		RequiredAcks:           kafka.RequiredAcks(cfg.Kafka.Producer.RequiredAcks),
		Async:                  cfg.Kafka.Producer.Async,
		CompressionCodec:       compress.Snappy,
		Logger:                 &KafkaLogger{logger: logger},
		ErrorLogger:            &KafkaErrorLogger{logger: logger},
	}

	producer := &Producer{
		config:       cfg,
		logger:       logger,
		writer:       writer,
		shutdownChan: make(chan struct{}),
	}

	return producer, nil
}

// Start starts the Kafka producer
func (p *Producer) Start(ctx context.Context) error {
	p.logger.Info("Starting Kafka producer", "topic", p.config.Kafka.Producer.AlertTopic)

	// Start metrics reporter
	p.wg.Add(1)
	go p.metricsReporter(ctx)

	p.logger.Info("Kafka producer started")
	return nil
}

// Stop stops the Kafka producer
func (p *Producer) Stop() {
	p.logger.Info("Stopping Kafka producer")
	close(p.shutdownChan)
	
	if p.writer != nil {
		p.writer.Close()
	}
	
	p.wg.Wait()
	p.logger.Info("Kafka producer stopped")
}

// PublishAlert publishes an alert to Kafka
func (p *Producer) PublishAlert(ctx context.Context, alert *database.Alert) error {
	// Create alert message
	alertMsg := AlertMessage{
		AlertID:     alert.ID,
		RuleID:      alert.RuleID,
		Type:        alert.Type,
		Severity:    alert.Severity,
		Priority:    alert.Priority,
		Title:       alert.Title,
		Description: alert.Description,
		Source:      alert.Source,
		Timestamp:   alert.CreatedAt,
	}

	// Add metadata if available
	if alert.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(alert.Metadata, &metadata); err == nil {
			alertMsg.Metadata = metadata
		}
	}

	// Add event data if available
	if alert.EventData != nil {
		var eventData map[string]interface{}
		if err := json.Unmarshal(alert.EventData, &eventData); err == nil {
			alertMsg.Data = eventData
		}
	}

	// Serialize message
	messageBytes, err := json.Marshal(alertMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal alert message: %w", err)
	}

	// Create Kafka message
	kafkaMsg := kafka.Message{
		Key:   []byte(alert.ID),
		Value: messageBytes,
		Headers: []kafka.Header{
			{Key: "alert_id", Value: []byte(alert.ID)},
			{Key: "rule_id", Value: []byte(alert.RuleID)},
			{Key: "severity", Value: []byte(alert.Severity)},
			{Key: "type", Value: []byte(alert.Type)},
		},
	}

	// Write message
	if err := p.writer.WriteMessages(ctx, kafkaMsg); err != nil {
		p.errorCount++
		return fmt.Errorf("failed to write alert message to Kafka: %w", err)
	}

	p.messageCount++
	p.logger.Debug("Alert published to Kafka",
		"alert_id", alert.ID,
		"rule_id", alert.RuleID,
		"severity", alert.Severity)

	return nil
}

// PublishNotification publishes a notification event to Kafka
func (p *Producer) PublishNotification(ctx context.Context, notification *database.Notification) error {
	// Create notification message
	notificationMsg := map[string]interface{}{
		"notification_id": notification.ID,
		"alert_id":       notification.AlertID,
		"rule_id":        notification.RuleID,
		"channel":        notification.Channel,
		"recipient":      notification.Recipient,
		"subject":        notification.Subject,
		"message":        notification.Message,
		"status":         notification.Status,
		"priority":       notification.Priority,
		"created_at":     notification.CreatedAt,
	}

	// Add metadata if available
	if notification.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(notification.Metadata, &metadata); err == nil {
			notificationMsg["metadata"] = metadata
		}
	}

	// Serialize message
	messageBytes, err := json.Marshal(notificationMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal notification message: %w", err)
	}

	// Create Kafka message
	kafkaMsg := kafka.Message{
		Key:   []byte(notification.ID),
		Value: messageBytes,
		Headers: []kafka.Header{
			{Key: "notification_id", Value: []byte(notification.ID)},
			{Key: "alert_id", Value: []byte(notification.AlertID)},
			{Key: "channel", Value: []byte(notification.Channel)},
			{Key: "status", Value: []byte(notification.Status)},
		},
	}

	// Write message to notifications topic
	writer := &kafka.Writer{
		Addr:         kafka.TCP(p.config.Kafka.Brokers...),
		Topic:        p.config.Kafka.Producer.NotificationTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    p.config.Kafka.Producer.BatchSize,
		BatchTimeout: time.Duration(p.config.Kafka.Producer.BatchTimeoutMs) * time.Millisecond,
	}
	defer writer.Close()

	if err := writer.WriteMessages(ctx, kafkaMsg); err != nil {
		p.errorCount++
		return fmt.Errorf("failed to write notification message to Kafka: %w", err)
	}

	p.messageCount++
	p.logger.Debug("Notification published to Kafka",
		"notification_id", notification.ID,
		"alert_id", notification.AlertID,
		"channel", notification.Channel)

	return nil
}

// metricsReporter reports producer metrics
func (p *Producer) metricsReporter(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdownChan:
			return
		case <-ticker.C:
			p.logger.Debug("Kafka producer metrics",
				"messages_published", p.messageCount,
				"errors", p.errorCount)
		}
	}
}

// GetStats returns producer statistics
func (p *Producer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"messages_published": p.messageCount,
		"errors":            p.errorCount,
		"is_running":        p.writer != nil,
	}
}

// EventProcessor coordinates event processing workflow
type EventProcessor struct {
	config           *config.Config
	logger           *slog.Logger
	consumer         *Consumer
	producer         *Producer
	ruleEngine       *engine.RuleEngine
	alertRepo        *database.AlertRepository
	notificationRepo *database.NotificationRepository
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(
	cfg *config.Config,
	logger *slog.Logger,
	consumer *Consumer,
	producer *Producer,
	ruleEngine *engine.RuleEngine,
	alertRepo *database.AlertRepository,
	notificationRepo *database.NotificationRepository,
) *EventProcessor {
	return &EventProcessor{
		config:           cfg,
		logger:           logger,
		consumer:         consumer,
		producer:         producer,
		ruleEngine:       ruleEngine,
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
		shutdownChan:     make(chan struct{}),
	}
}

// Start starts the event processor
func (e *EventProcessor) Start(ctx context.Context) error {
	e.logger.Info("Starting event processor")

	// Start consumer
	if err := e.consumer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Start producer
	if err := e.producer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start producer: %w", err)
	}

	e.logger.Info("Event processor started")
	return nil
}

// Stop stops the event processor
func (e *EventProcessor) Stop() {
	e.logger.Info("Stopping event processor")
	close(e.shutdownChan)
	
	e.consumer.Stop()
	e.producer.Stop()
	
	e.wg.Wait()
	e.logger.Info("Event processor stopped")
}

// GetStats returns event processor statistics
func (e *EventProcessor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"consumer": e.consumer.GetStats(),
		"producer": e.producer.GetStats(),
	}
}

// Kafka logging adapters

type KafkaLogger struct {
	logger *slog.Logger
}

func (l *KafkaLogger) Printf(format string, v ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

type KafkaErrorLogger struct {
	logger *slog.Logger
}

func (l *KafkaErrorLogger) Printf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...))
}