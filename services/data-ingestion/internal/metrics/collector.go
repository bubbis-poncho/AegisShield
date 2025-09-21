package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all metrics for the data ingestion service
type Collector struct {
	// Request counters
	uploadFileRequests       prometheus.Counter
	uploadFileStreamRequests prometheus.Counter
	processTransactionRequests prometheus.Counter
	validateDataRequests     prometheus.Counter

	// Error counters
	uploadFileErrors       prometheus.Counter
	uploadFileStreamErrors prometheus.Counter
	processTransactionErrors prometheus.Counter
	validateDataErrors     prometheus.Counter

	// Processing histograms
	uploadFileDuration       prometheus.Histogram
	uploadFileStreamDuration prometheus.Histogram
	processTransactionDuration prometheus.Histogram
	validateDataDuration     prometheus.Histogram

	// File metrics
	uploadedFileSize       prometheus.Histogram
	uploadedFileStreamSize prometheus.Histogram

	// Transaction metrics
	processedTransactions prometheus.Gauge
	failedTransactions    prometheus.Gauge
	transactionBatchSize  prometheus.Histogram

	// Job metrics
	activeJobs     prometheus.Gauge
	completedJobs  prometheus.Counter
	failedJobs     prometheus.Counter
	jobDuration    prometheus.Histogram

	// Database metrics
	dbConnections    prometheus.Gauge
	dbQueries        prometheus.Counter
	dbQueryDuration  prometheus.Histogram
	dbErrors         prometheus.Counter

	// Kafka metrics
	kafkaMessages       prometheus.Counter
	kafkaMessageErrors  prometheus.Counter
	kafkaPublishDuration prometheus.Histogram

	// Storage metrics
	storageOperations       prometheus.Counter
	storageErrors           prometheus.Counter
	storageOperationDuration prometheus.Histogram
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		// Request counters
		uploadFileRequests: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "upload_file_requests_total",
			Help:      "Total number of file upload requests",
		}),
		uploadFileStreamRequests: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "upload_file_stream_requests_total",
			Help:      "Total number of file stream upload requests",
		}),
		processTransactionRequests: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "process_transaction_requests_total",
			Help:      "Total number of transaction processing requests",
		}),
		validateDataRequests: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "validate_data_requests_total",
			Help:      "Total number of data validation requests",
		}),

		// Error counters
		uploadFileErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "upload_file_errors_total",
			Help:      "Total number of file upload errors",
		}),
		uploadFileStreamErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "upload_file_stream_errors_total",
			Help:      "Total number of file stream upload errors",
		}),
		processTransactionErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "process_transaction_errors_total",
			Help:      "Total number of transaction processing errors",
		}),
		validateDataErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "validate_data_errors_total",
			Help:      "Total number of data validation errors",
		}),

		// Processing histograms
		uploadFileDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "upload_file_duration_seconds",
			Help:      "Duration of file upload operations",
			Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		}),
		uploadFileStreamDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "upload_file_stream_duration_seconds",
			Help:      "Duration of file stream upload operations",
			Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		}),
		processTransactionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "process_transaction_duration_seconds",
			Help:      "Duration of transaction processing operations",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		}),
		validateDataDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "validate_data_duration_seconds",
			Help:      "Duration of data validation operations",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		}),

		// File metrics
		uploadedFileSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "uploaded_file_size_bytes",
			Help:      "Size of uploaded files in bytes",
			Buckets:   []float64{1024, 10240, 102400, 1048576, 10485760, 104857600, 1073741824}, // 1KB to 1GB
		}),
		uploadedFileStreamSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "uploaded_file_stream_size_bytes",
			Help:      "Size of uploaded stream files in bytes",
			Buckets:   []float64{1024, 10240, 102400, 1048576, 10485760, 104857600, 1073741824}, // 1KB to 1GB
		}),

		// Transaction metrics
		processedTransactions: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "processed_transactions_total",
			Help:      "Total number of successfully processed transactions",
		}),
		failedTransactions: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "failed_transactions_total",
			Help:      "Total number of failed transactions",
		}),
		transactionBatchSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "transaction_batch_size",
			Help:      "Size of transaction batches",
			Buckets:   []float64{1, 10, 50, 100, 500, 1000, 5000, 10000},
		}),

		// Job metrics
		activeJobs: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "active_jobs",
			Help:      "Number of currently active processing jobs",
		}),
		completedJobs: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "completed_jobs_total",
			Help:      "Total number of completed jobs",
		}),
		failedJobs: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "failed_jobs_total",
			Help:      "Total number of failed jobs",
		}),
		jobDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "job_duration_seconds",
			Help:      "Duration of processing jobs",
			Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600}, // 1s to 1h
		}),

		// Database metrics
		dbConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "db_connections",
			Help:      "Number of active database connections",
		}),
		dbQueries: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "db_queries_total",
			Help:      "Total number of database queries",
		}),
		dbQueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "db_query_duration_seconds",
			Help:      "Duration of database queries",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		}),
		dbErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "db_errors_total",
			Help:      "Total number of database errors",
		}),

		// Kafka metrics
		kafkaMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "kafka_messages_total",
			Help:      "Total number of Kafka messages published",
		}),
		kafkaMessageErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "kafka_message_errors_total",
			Help:      "Total number of Kafka message publishing errors",
		}),
		kafkaPublishDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "kafka_publish_duration_seconds",
			Help:      "Duration of Kafka message publishing",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		}),

		// Storage metrics
		storageOperations: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "storage_operations_total",
			Help:      "Total number of storage operations",
		}),
		storageErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "storage_errors_total",
			Help:      "Total number of storage errors",
		}),
		storageOperationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "aegisshield",
			Subsystem: "data_ingestion",
			Name:      "storage_operation_duration_seconds",
			Help:      "Duration of storage operations",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		}),
	}
}

// Register registers all metrics with Prometheus (auto-registered via promauto)
func (c *Collector) Register() {
	// Metrics are auto-registered via promauto
}

// Increment methods
func (c *Collector) IncrementCounter(name string) {
	switch name {
	case "upload_file_requests_total":
		c.uploadFileRequests.Inc()
	case "upload_file_stream_requests_total":
		c.uploadFileStreamRequests.Inc()
	case "process_transaction_stream_requests_total":
		c.processTransactionRequests.Inc()
	case "validate_data_requests_total":
		c.validateDataRequests.Inc()
	case "upload_file_errors_total":
		c.uploadFileErrors.Inc()
	case "upload_file_stream_errors_total":
		c.uploadFileStreamErrors.Inc()
	case "process_transaction_stream_errors_total":
		c.processTransactionErrors.Inc()
	case "validate_data_errors_total":
		c.validateDataErrors.Inc()
	case "completed_jobs_total":
		c.completedJobs.Inc()
	case "failed_jobs_total":
		c.failedJobs.Inc()
	case "db_queries_total":
		c.dbQueries.Inc()
	case "db_errors_total":
		c.dbErrors.Inc()
	case "kafka_messages_total":
		c.kafkaMessages.Inc()
	case "kafka_message_errors_total":
		c.kafkaMessageErrors.Inc()
	case "storage_operations_total":
		c.storageOperations.Inc()
	case "storage_errors_total":
		c.storageErrors.Inc()
	}
}

// Record histogram values
func (c *Collector) RecordHistogram(name string, value float64) {
	switch name {
	case "upload_file_duration_seconds":
		c.uploadFileDuration.Observe(value)
	case "upload_file_stream_duration_seconds":
		c.uploadFileStreamDuration.Observe(value)
	case "process_transaction_stream_duration_seconds":
		c.processTransactionDuration.Observe(value)
	case "validate_data_duration_seconds":
		c.validateDataDuration.Observe(value)
	case "uploaded_file_size_bytes":
		c.uploadedFileSize.Observe(value)
	case "uploaded_file_stream_size_bytes":
		c.uploadedFileStreamSize.Observe(value)
	case "transaction_batch_size":
		c.transactionBatchSize.Observe(value)
	case "job_duration_seconds":
		c.jobDuration.Observe(value)
	case "db_query_duration_seconds":
		c.dbQueryDuration.Observe(value)
	case "kafka_publish_duration_seconds":
		c.kafkaPublishDuration.Observe(value)
	case "storage_operation_duration_seconds":
		c.storageOperationDuration.Observe(value)
	}
}

// Set gauge values
func (c *Collector) RecordGauge(name string, value float64) {
	switch name {
	case "processed_transactions_total":
		c.processedTransactions.Set(value)
	case "failed_transactions_total":
		c.failedTransactions.Set(value)
	case "active_jobs":
		c.activeJobs.Set(value)
	case "db_connections":
		c.dbConnections.Set(value)
	}
}

// Add gauge values (for cumulative metrics)
func (c *Collector) AddGauge(name string, value float64) {
	switch name {
	case "processed_transactions_total":
		c.processedTransactions.Add(value)
	case "failed_transactions_total":
		c.failedTransactions.Add(value)
	case "active_jobs":
		c.activeJobs.Add(value)
	case "db_connections":
		c.dbConnections.Add(value)
	}
}