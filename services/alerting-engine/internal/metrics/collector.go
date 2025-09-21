package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/kafka"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
	"github.com/aegis-shield/services/alerting-engine/internal/scheduler"
)

// Collector manages Prometheus metrics for the alerting engine
type Collector struct {
	config    *config.Config
	logger    *slog.Logger
	
	// Component references for stats collection
	alertRepo        *database.AlertRepository
	ruleRepo         *database.RuleRepository
	notificationRepo *database.NotificationRepository
	escalationRepo   *database.EscalationRepository
	ruleEngine       *engine.RuleEngine
	notificationMgr  *notification.Manager
	eventProcessor   *kafka.EventProcessor
	scheduler        *scheduler.Scheduler

	// Prometheus metrics
	alertsTotal          *prometheus.CounterVec
	alertsActive         prometheus.Gauge
	alertsDuration       *prometheus.HistogramVec
	alertsEscalated      prometheus.Counter
	alertsResolved       prometheus.Counter
	alertsAcknowledged   prometheus.Counter

	rulesTotal           prometheus.Gauge
	rulesActive          prometheus.Gauge
	ruleEvaluationsTotal *prometheus.CounterVec
	ruleEvaluationDuration prometheus.Histogram
	ruleMatchesTotal     *prometheus.CounterVec

	notificationsTotal   *prometheus.CounterVec
	notificationDuration *prometheus.HistogramVec
	notificationErrors   *prometheus.CounterVec
	notificationRetries  *prometheus.CounterVec

	eventsProcessed      *prometheus.CounterVec
	eventProcessingTime  prometheus.Histogram
	kafkaLag             prometheus.Gauge
	kafkaOffset          prometheus.Gauge

	tasksTotal           prometheus.Gauge
	tasksExecuted        *prometheus.CounterVec
	taskDuration         *prometheus.HistogramVec
	taskErrors           *prometheus.CounterVec

	// System metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	grpcRequestsTotal    *prometheus.CounterVec
	grpcRequestDuration  *prometheus.HistogramVec

	// Database metrics
	dbConnectionsActive  prometheus.Gauge
	dbConnectionsIdle    prometheus.Gauge
	dbQueriesTotal       *prometheus.CounterVec
	dbQueryDuration      *prometheus.HistogramVec

	// Internal state
	mu                   sync.RWMutex
	lastCollectionTime   time.Time
	collectionInterval   time.Duration
}

// NewCollector creates a new metrics collector
func NewCollector(
	cfg *config.Config,
	logger *slog.Logger,
	alertRepo *database.AlertRepository,
	ruleRepo *database.RuleRepository,
	notificationRepo *database.NotificationRepository,
	escalationRepo *database.EscalationRepository,
	ruleEngine *engine.RuleEngine,
	notificationMgr *notification.Manager,
	eventProcessor *kafka.EventProcessor,
	scheduler *scheduler.Scheduler,
) *Collector {
	return &Collector{
		config:           cfg,
		logger:           logger,
		alertRepo:        alertRepo,
		ruleRepo:         ruleRepo,
		notificationRepo: notificationRepo,
		escalationRepo:   escalationRepo,
		ruleEngine:       ruleEngine,
		notificationMgr:  notificationMgr,
		eventProcessor:   eventProcessor,
		scheduler:        scheduler,
		collectionInterval: 30 * time.Second,
	}
}

// RegisterMetrics registers all Prometheus metrics
func (c *Collector) RegisterMetrics() {
	// Alert metrics
	c.alertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_alerts_total",
			Help: "Total number of alerts created",
		},
		[]string{"severity", "type", "source", "status"},
	)

	c.alertsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_alerts_active",
			Help: "Number of currently active alerts",
		},
	)

	c.alertsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_alert_duration_seconds",
			Help:    "Duration of alerts from creation to resolution",
			Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1 minute to 68 hours
		},
		[]string{"severity", "type"},
	)

	c.alertsEscalated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alerting_engine_alerts_escalated_total",
			Help: "Total number of alerts escalated",
		},
	)

	c.alertsResolved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alerting_engine_alerts_resolved_total",
			Help: "Total number of alerts resolved",
		},
	)

	c.alertsAcknowledged = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alerting_engine_alerts_acknowledged_total",
			Help: "Total number of alerts acknowledged",
		},
	)

	// Rule metrics
	c.rulesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_rules_total",
			Help: "Total number of rules configured",
		},
	)

	c.rulesActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_rules_active",
			Help: "Number of currently active rules",
		},
	)

	c.ruleEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_rule_evaluations_total",
			Help: "Total number of rule evaluations",
		},
		[]string{"rule_id", "result"},
	)

	c.ruleEvaluationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_rule_evaluation_duration_seconds",
			Help:    "Duration of rule evaluations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
		},
	)

	c.ruleMatchesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_rule_matches_total",
			Help: "Total number of rule matches",
		},
		[]string{"rule_id", "severity"},
	)

	// Notification metrics
	c.notificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_notifications_total",
			Help: "Total number of notifications sent",
		},
		[]string{"channel", "type", "status"},
	)

	c.notificationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_notification_duration_seconds",
			Help:    "Duration of notification delivery",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to 102s
		},
		[]string{"channel", "type"},
	)

	c.notificationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_notification_errors_total",
			Help: "Total number of notification errors",
		},
		[]string{"channel", "type", "error_type"},
	)

	c.notificationRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_notification_retries_total",
			Help: "Total number of notification retries",
		},
		[]string{"channel", "type"},
	)

	// Event processing metrics
	c.eventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"topic", "status"},
	)

	c.eventProcessingTime = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_event_processing_duration_seconds",
			Help:    "Duration of event processing",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 4s
		},
	)

	c.kafkaLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_kafka_lag",
			Help: "Current Kafka consumer lag",
		},
	)

	c.kafkaOffset = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_kafka_offset",
			Help: "Current Kafka consumer offset",
		},
	)

	// Scheduler metrics
	c.tasksTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_scheduled_tasks_total",
			Help: "Total number of scheduled tasks",
		},
	)

	c.tasksExecuted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_tasks_executed_total",
			Help: "Total number of tasks executed",
		},
		[]string{"task_id", "status"},
	)

	c.taskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_task_duration_seconds",
			Help:    "Duration of task execution",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to 512s
		},
		[]string{"task_id"},
	)

	c.taskErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_task_errors_total",
			Help: "Total number of task errors",
		},
		[]string{"task_id", "error_type"},
	)

	// HTTP metrics
	c.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	c.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// gRPC metrics
	c.grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	c.grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_grpc_request_duration_seconds",
			Help:    "Duration of gRPC requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Database metrics
	c.dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	c.dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alerting_engine_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	c.dbQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerting_engine_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "status"},
	)

	c.dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alerting_engine_db_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
		},
		[]string{"operation"},
	)
}

// Start begins the metrics collection process
func (c *Collector) Start(ctx context.Context) error {
	c.logger.Info("Starting metrics collector")

	c.RegisterMetrics()

	ticker := time.NewTicker(c.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping metrics collector")
			return ctx.Err()
		case <-ticker.C:
			c.collectMetrics(ctx)
		}
	}
}

// collectMetrics collects metrics from all components
func (c *Collector) collectMetrics(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastCollectionTime = time.Now()

	// Collect alert metrics
	c.collectAlertMetrics(ctx)

	// Collect rule metrics
	c.collectRuleMetrics(ctx)

	// Collect notification metrics
	c.collectNotificationMetrics(ctx)

	// Collect event processing metrics
	c.collectEventMetrics(ctx)

	// Collect scheduler metrics
	c.collectSchedulerMetrics(ctx)

	// Collect database metrics
	c.collectDatabaseMetrics(ctx)
}

func (c *Collector) collectAlertMetrics(ctx context.Context) {
	if c.alertRepo == nil {
		return
	}

	// Get alert statistics
	since := time.Now().Add(-24 * time.Hour)
	stats, err := c.alertRepo.GetStatsByTimeRange(ctx, since, time.Now())
	if err != nil {
		c.logger.Error("Failed to collect alert metrics", "error", err)
		return
	}

	// Update gauges
	if totalStats, ok := stats["total"].(map[string]interface{}); ok {
		if active, ok := totalStats["active"].(int64); ok {
			c.alertsActive.Set(float64(active))
		}
	}

	// The counters would be updated in real-time by the application
	// This is just for demonstration of how to collect from stats
}

func (c *Collector) collectRuleMetrics(ctx context.Context) {
	if c.ruleEngine == nil {
		return
	}

	stats := c.ruleEngine.GetRuleStats()
	
	if totalRules, ok := stats["total_rules"].(int); ok {
		c.rulesTotal.Set(float64(totalRules))
	}
	
	if activeRules, ok := stats["active_rules"].(int); ok {
		c.rulesActive.Set(float64(activeRules))
	}
}

func (c *Collector) collectNotificationMetrics(ctx context.Context) {
	if c.notificationMgr == nil {
		return
	}

	stats := c.notificationMgr.GetStats()
	
	// Update metrics based on notification manager stats
	// This would be implemented based on the actual stats structure
	c.logger.Debug("Collected notification metrics", "stats", stats)
}

func (c *Collector) collectEventMetrics(ctx context.Context) {
	if c.eventProcessor == nil {
		return
	}

	stats := c.eventProcessor.GetStats()
	
	// Update Kafka metrics
	if lagMap, ok := stats["consumer_lag"].(map[string]interface{}); ok {
		var totalLag float64
		for _, lag := range lagMap {
			if lagVal, ok := lag.(int64); ok {
				totalLag += float64(lagVal)
			}
		}
		c.kafkaLag.Set(totalLag)
	}

	if offsetMap, ok := stats["consumer_offset"].(map[string]interface{}); ok {
		var totalOffset float64
		for _, offset := range offsetMap {
			if offsetVal, ok := offset.(int64); ok {
				totalOffset += float64(offsetVal)
			}
		}
		c.kafkaOffset.Set(totalOffset)
	}
}

func (c *Collector) collectSchedulerMetrics(ctx context.Context) {
	if c.scheduler == nil {
		return
	}

	stats := c.scheduler.GetSchedulerStats()
	
	if totalTasks, ok := stats["total_tasks"].(int); ok {
		c.tasksTotal.Set(float64(totalTasks))
	}
}

func (c *Collector) collectDatabaseMetrics(ctx context.Context) {
	// This would collect database connection pool metrics
	// Implementation depends on the database driver being used
	c.logger.Debug("Collecting database metrics")
}

// RecordAlertCreated records an alert creation event
func (c *Collector) RecordAlertCreated(severity, alertType, source, status string) {
	c.alertsTotal.WithLabelValues(severity, alertType, source, status).Inc()
}

// RecordAlertEscalated records an alert escalation event
func (c *Collector) RecordAlertEscalated() {
	c.alertsEscalated.Inc()
}

// RecordAlertResolved records an alert resolution event
func (c *Collector) RecordAlertResolved() {
	c.alertsResolved.Inc()
}

// RecordAlertAcknowledged records an alert acknowledgment event
func (c *Collector) RecordAlertAcknowledged() {
	c.alertsAcknowledged.Inc()
}

// RecordAlertDuration records the duration of an alert
func (c *Collector) RecordAlertDuration(severity, alertType string, duration time.Duration) {
	c.alertsDuration.WithLabelValues(severity, alertType).Observe(duration.Seconds())
}

// RecordRuleEvaluation records a rule evaluation event
func (c *Collector) RecordRuleEvaluation(ruleID, result string, duration time.Duration) {
	c.ruleEvaluationsTotal.WithLabelValues(ruleID, result).Inc()
	c.ruleEvaluationDuration.Observe(duration.Seconds())
}

// RecordRuleMatch records a rule match event
func (c *Collector) RecordRuleMatch(ruleID, severity string) {
	c.ruleMatchesTotal.WithLabelValues(ruleID, severity).Inc()
}

// RecordNotification records a notification event
func (c *Collector) RecordNotification(channel, notificationType, status string, duration time.Duration) {
	c.notificationsTotal.WithLabelValues(channel, notificationType, status).Inc()
	c.notificationDuration.WithLabelValues(channel, notificationType).Observe(duration.Seconds())
}

// RecordNotificationError records a notification error
func (c *Collector) RecordNotificationError(channel, notificationType, errorType string) {
	c.notificationErrors.WithLabelValues(channel, notificationType, errorType).Inc()
}

// RecordNotificationRetry records a notification retry
func (c *Collector) RecordNotificationRetry(channel, notificationType string) {
	c.notificationRetries.WithLabelValues(channel, notificationType).Inc()
}

// RecordEventProcessed records an event processing event
func (c *Collector) RecordEventProcessed(topic, status string, duration time.Duration) {
	c.eventsProcessed.WithLabelValues(topic, status).Inc()
	c.eventProcessingTime.Observe(duration.Seconds())
}

// RecordTaskExecution records a task execution event
func (c *Collector) RecordTaskExecution(taskID, status string, duration time.Duration) {
	c.tasksExecuted.WithLabelValues(taskID, status).Inc()
	c.taskDuration.WithLabelValues(taskID).Observe(duration.Seconds())
}

// RecordTaskError records a task error
func (c *Collector) RecordTaskError(taskID, errorType string) {
	c.taskErrors.WithLabelValues(taskID, errorType).Inc()
}

// RecordHTTPRequest records an HTTP request event
func (c *Collector) RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	c.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	c.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request event
func (c *Collector) RecordGRPCRequest(method, status string, duration time.Duration) {
	c.grpcRequestsTotal.WithLabelValues(method, status).Inc()
	c.grpcRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordDatabaseQuery records a database query event
func (c *Collector) RecordDatabaseQuery(operation, status string, duration time.Duration) {
	c.dbQueriesTotal.WithLabelValues(operation, status).Inc()
	c.dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// GetStats returns current metrics statistics
func (c *Collector) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"last_collection_time": c.lastCollectionTime,
		"collection_interval":  c.collectionInterval.String(),
		"metrics_registered":   true,
	}
}