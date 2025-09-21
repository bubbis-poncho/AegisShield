package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector contains all metrics for entity resolution service
type Collector struct {
	// Entity resolution metrics
	EntitiesResolvedTotal   prometheus.Counter
	EntitiesCreatedTotal    prometheus.Counter
	EntitiesUpdatedTotal    prometheus.Counter
	EntityLinksCreatedTotal prometheus.Counter

	// Batch processing metrics
	BatchJobsTotal     prometheus.Counter
	BatchJobsCompleted prometheus.Counter
	BatchJobsFailed    prometheus.Counter
	BatchSizeHistogram prometheus.Histogram

	// Performance metrics
	ResolutionDuration    prometheus.Histogram
	MatchingDuration      prometheus.Histogram
	StandardizationDuration prometheus.Histogram
	DatabaseQueryDuration prometheus.Histogram
	Neo4jQueryDuration    prometheus.Histogram

	// Quality metrics
	ConfidenceScoreHistogram prometheus.Histogram
	MatchCandidatesHistogram prometheus.Histogram
	AutoMergeRate           prometheus.Gauge
	ManualReviewRate        prometheus.Gauge

	// System metrics
	ActiveResolutionJobs prometheus.Gauge
	KafkaMessagesProcessed prometheus.Counter
	KafkaMessagesPublished prometheus.Counter
	DatabaseConnections    prometheus.Gauge
	Neo4jConnections       prometheus.Gauge

	// Error metrics
	ResolutionErrors     prometheus.Counter
	DatabaseErrors       prometheus.Counter
	Neo4jErrors          prometheus.Counter
	KafkaErrors          prometheus.Counter
	StandardizationErrors prometheus.Counter
	MatchingErrors       prometheus.Counter
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		// Entity resolution metrics
		EntitiesResolvedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_entities_resolved_total",
			Help: "The total number of entities resolved",
		}),
		EntitiesCreatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_entities_created_total",
			Help: "The total number of new entities created",
		}),
		EntitiesUpdatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_entities_updated_total",
			Help: "The total number of entities updated",
		}),
		EntityLinksCreatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_links_created_total",
			Help: "The total number of entity links created",
		}),

		// Batch processing metrics
		BatchJobsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_batch_jobs_total",
			Help: "The total number of batch jobs started",
		}),
		BatchJobsCompleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_batch_jobs_completed_total",
			Help: "The total number of batch jobs completed successfully",
		}),
		BatchJobsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_batch_jobs_failed_total",
			Help: "The total number of batch jobs that failed",
		}),
		BatchSizeHistogram: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_batch_size",
			Help:    "The size of batch processing jobs",
			Buckets: prometheus.LinearBuckets(10, 10, 10), // 10, 20, 30, ..., 100
		}),

		// Performance metrics
		ResolutionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_duration_seconds",
			Help:    "The duration of entity resolution operations in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		MatchingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_matching_duration_seconds",
			Help:    "The duration of entity matching operations in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		StandardizationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_standardization_duration_seconds",
			Help:    "The duration of data standardization operations in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		DatabaseQueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_database_query_duration_seconds",
			Help:    "The duration of database queries in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		Neo4jQueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_neo4j_query_duration_seconds",
			Help:    "The duration of Neo4j queries in seconds",
			Buckets: prometheus.DefBuckets,
		}),

		// Quality metrics
		ConfidenceScoreHistogram: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_confidence_score",
			Help:    "The confidence scores of entity resolutions",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 10), // 0.1, 0.2, ..., 1.0
		}),
		MatchCandidatesHistogram: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entity_resolution_match_candidates",
			Help:    "The number of match candidates found per resolution",
			Buckets: prometheus.LinearBuckets(0, 5, 10), // 0, 5, 10, ..., 45
		}),
		AutoMergeRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "entity_resolution_auto_merge_rate",
			Help: "The rate of automatic entity merges",
		}),
		ManualReviewRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "entity_resolution_manual_review_rate",
			Help: "The rate of entities requiring manual review",
		}),

		// System metrics
		ActiveResolutionJobs: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "entity_resolution_active_jobs",
			Help: "The number of currently active resolution jobs",
		}),
		KafkaMessagesProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_kafka_messages_processed_total",
			Help: "The total number of Kafka messages processed",
		}),
		KafkaMessagesPublished: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_kafka_messages_published_total",
			Help: "The total number of Kafka messages published",
		}),
		DatabaseConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "entity_resolution_database_connections",
			Help: "The number of active database connections",
		}),
		Neo4jConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "entity_resolution_neo4j_connections",
			Help: "The number of active Neo4j connections",
		}),

		// Error metrics
		ResolutionErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_errors_total",
			Help: "The total number of entity resolution errors",
		}),
		DatabaseErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_database_errors_total",
			Help: "The total number of database errors",
		}),
		Neo4jErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_neo4j_errors_total",
			Help: "The total number of Neo4j errors",
		}),
		KafkaErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_kafka_errors_total",
			Help: "The total number of Kafka errors",
		}),
		StandardizationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_standardization_errors_total",
			Help: "The total number of data standardization errors",
		}),
		MatchingErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entity_resolution_matching_errors_total",
			Help: "The total number of entity matching errors",
		}),
	}
}

// RecordEntityResolved records a successful entity resolution
func (c *Collector) RecordEntityResolved(isNewEntity bool, confidenceScore float64, matchCandidates int, duration time.Duration) {
	c.EntitiesResolvedTotal.Inc()
	c.ResolutionDuration.Observe(duration.Seconds())
	c.ConfidenceScoreHistogram.Observe(confidenceScore)
	c.MatchCandidatesHistogram.Observe(float64(matchCandidates))

	if isNewEntity {
		c.EntitiesCreatedTotal.Inc()
	} else {
		c.EntitiesUpdatedTotal.Inc()
	}
}

// RecordEntityLinkCreated records a new entity link creation
func (c *Collector) RecordEntityLinkCreated() {
	c.EntityLinksCreatedTotal.Inc()
}

// RecordBatchJobStarted records the start of a batch job
func (c *Collector) RecordBatchJobStarted(batchSize int) {
	c.BatchJobsTotal.Inc()
	c.BatchSizeHistogram.Observe(float64(batchSize))
	c.ActiveResolutionJobs.Inc()
}

// RecordBatchJobCompleted records the completion of a batch job
func (c *Collector) RecordBatchJobCompleted(successful bool) {
	c.ActiveResolutionJobs.Dec()
	if successful {
		c.BatchJobsCompleted.Inc()
	} else {
		c.BatchJobsFailed.Inc()
	}
}

// RecordMatchingDuration records the duration of matching operations
func (c *Collector) RecordMatchingDuration(duration time.Duration) {
	c.MatchingDuration.Observe(duration.Seconds())
}

// RecordStandardizationDuration records the duration of standardization operations
func (c *Collector) RecordStandardizationDuration(duration time.Duration) {
	c.StandardizationDuration.Observe(duration.Seconds())
}

// RecordDatabaseQuery records database query metrics
func (c *Collector) RecordDatabaseQuery(duration time.Duration, err error) {
	c.DatabaseQueryDuration.Observe(duration.Seconds())
	if err != nil {
		c.DatabaseErrors.Inc()
	}
}

// RecordNeo4jQuery records Neo4j query metrics
func (c *Collector) RecordNeo4jQuery(duration time.Duration, err error) {
	c.Neo4jQueryDuration.Observe(duration.Seconds())
	if err != nil {
		c.Neo4jErrors.Inc()
	}
}

// RecordKafkaMessage records Kafka message processing
func (c *Collector) RecordKafkaMessage(processed bool, err error) {
	if processed {
		c.KafkaMessagesProcessed.Inc()
	} else {
		c.KafkaMessagesPublished.Inc()
	}
	
	if err != nil {
		c.KafkaErrors.Inc()
	}
}

// RecordResolutionError records a resolution error
func (c *Collector) RecordResolutionError() {
	c.ResolutionErrors.Inc()
}

// RecordStandardizationError records a standardization error
func (c *Collector) RecordStandardizationError() {
	c.StandardizationErrors.Inc()
}

// RecordMatchingError records a matching error
func (c *Collector) RecordMatchingError() {
	c.MatchingErrors.Inc()
}

// UpdateConnectionCounts updates connection count metrics
func (c *Collector) UpdateConnectionCounts(database, neo4j int) {
	c.DatabaseConnections.Set(float64(database))
	c.Neo4jConnections.Set(float64(neo4j))
}

// UpdateQualityMetrics updates quality-related metrics
func (c *Collector) UpdateQualityMetrics(autoMergeRate, manualReviewRate float64) {
	c.AutoMergeRate.Set(autoMergeRate)
	c.ManualReviewRate.Set(manualReviewRate)
}

// Timer is a helper for timing operations
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Duration returns the elapsed duration
func (t *Timer) Duration() time.Duration {
	return time.Since(t.start)
}

// ObserveDuration observes the duration on a histogram
func (t *Timer) ObserveDuration(histogram prometheus.Histogram) {
	histogram.Observe(t.Duration().Seconds())
}

// Helper functions for common metrics patterns

// TrackResolutionOperation tracks the duration and outcome of a resolution operation
func (c *Collector) TrackResolutionOperation(operation func() error) error {
	timer := NewTimer()
	err := operation()
	
	if err != nil {
		c.RecordResolutionError()
	}
	
	c.ResolutionDuration.Observe(timer.Duration().Seconds())
	return err
}

// TrackDatabaseOperation tracks the duration and outcome of a database operation
func (c *Collector) TrackDatabaseOperation(operation func() error) error {
	timer := NewTimer()
	err := operation()
	c.RecordDatabaseQuery(timer.Duration(), err)
	return err
}

// TrackNeo4jOperation tracks the duration and outcome of a Neo4j operation
func (c *Collector) TrackNeo4jOperation(operation func() error) error {
	timer := NewTimer()
	err := operation()
	c.RecordNeo4jQuery(timer.Duration(), err)
	return err
}

// TrackMatchingOperation tracks the duration and outcome of a matching operation
func (c *Collector) TrackMatchingOperation(operation func() error) error {
	timer := NewTimer()
	err := operation()
	
	if err != nil {
		c.RecordMatchingError()
	}
	
	c.RecordMatchingDuration(timer.Duration())
	return err
}

// TrackStandardizationOperation tracks the duration and outcome of a standardization operation
func (c *Collector) TrackStandardizationOperation(operation func() error) error {
	timer := NewTimer()
	err := operation()
	
	if err != nil {
		c.RecordStandardizationError()
	}
	
	c.RecordStandardizationDuration(timer.Duration())
	return err
}