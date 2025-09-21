package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/aegisshield/graph-engine/internal/config"
)

// MetricsCollector collects and exports metrics for the graph engine service
type MetricsCollector struct {
	config config.Config
	logger *slog.Logger

	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec

	// Analysis metrics
	analysisJobsTotal       *prometheus.CounterVec
	analysisJobDuration     *prometheus.HistogramVec
	analysisJobsActive      prometheus.Gauge
	analysisJobsQueued      prometheus.Gauge
	subgraphAnalysisTotal   *prometheus.CounterVec
	pathfindingTotal        *prometheus.CounterVec
	metricsCalculationTotal *prometheus.CounterVec

	// Graph metrics
	entitiesProcessed       *prometheus.CounterVec
	relationshipsProcessed  *prometheus.CounterVec
	subgraphSize           *prometheus.HistogramVec
	pathLength             *prometheus.HistogramVec
	centralityCalculations *prometheus.CounterVec

	// Pattern detection metrics
	patternsDetected      *prometheus.CounterVec
	patternConfidence     *prometheus.HistogramVec
	communitiesDetected   *prometheus.CounterVec
	communitySize         *prometheus.HistogramVec
	communityModularity   *prometheus.HistogramVec

	// Investigation metrics
	investigationsTotal     *prometheus.CounterVec
	investigationDuration   *prometheus.HistogramVec
	investigationsActive    prometheus.Gauge
	investigationEntities   *prometheus.HistogramVec

	// Database metrics
	dbConnections          prometheus.Gauge
	dbConnectionsActive    prometheus.Gauge
	dbQueryDuration        *prometheus.HistogramVec
	dbQueriesTotal         *prometheus.CounterVec
	dbConnectionErrors     *prometheus.CounterVec

	// Neo4j metrics
	neo4jConnections       prometheus.Gauge
	neo4jQueryDuration     *prometheus.HistogramVec
	neo4jQueriesTotal      *prometheus.CounterVec
	neo4jConnectionErrors  *prometheus.CounterVec
	neo4jSubgraphQueries   *prometheus.CounterVec
	neo4jPathQueries       *prometheus.CounterVec
	neo4jCentralityQueries *prometheus.CounterVec

	// Kafka metrics
	kafkaMessagesProduced *prometheus.CounterVec
	kafkaMessagesConsumed *prometheus.CounterVec
	kafkaProduceErrors    *prometheus.CounterVec
	kafkaConsumeErrors    *prometheus.CounterVec
	kafkaConsumerLag      *prometheus.GaugeVec

	// System metrics
	goroutinesActive prometheus.Gauge
	memoryUsage      prometheus.Gauge
	cpuUsage         prometheus.Gauge

	// Performance metrics
	analysisPerformance    *prometheus.HistogramVec
	networkComplexity      *prometheus.HistogramVec
	algorithmPerformance   *prometheus.HistogramVec
	cacheHitRate          *prometheus.GaugeVec
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config config.Config, logger *slog.Logger) *MetricsCollector {
	return &MetricsCollector{
		config: config,
		logger: logger,

		// Request metrics
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_requests_total",
				Help: "Total number of requests processed",
			},
			[]string{"method", "endpoint", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "graph_engine_requests_in_flight",
				Help: "Number of requests currently being processed",
			},
			[]string{"method", "endpoint"},
		),

		// Analysis metrics
		analysisJobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_analysis_jobs_total",
				Help: "Total number of analysis jobs",
			},
			[]string{"type", "status"},
		),
		analysisJobDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_analysis_job_duration_seconds",
				Help:    "Analysis job duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 300, 600, 1800, 3600},
			},
			[]string{"type"},
		),
		analysisJobsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_analysis_jobs_active",
				Help: "Number of active analysis jobs",
			},
		),
		analysisJobsQueued: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_analysis_jobs_queued",
				Help: "Number of queued analysis jobs",
			},
		),
		subgraphAnalysisTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_subgraph_analysis_total",
				Help: "Total number of subgraph analyses",
			},
			[]string{"type", "status"},
		),
		pathfindingTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_pathfinding_total",
				Help: "Total number of pathfinding operations",
			},
			[]string{"algorithm", "status"},
		),
		metricsCalculationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_metrics_calculation_total",
				Help: "Total number of metrics calculations",
			},
			[]string{"metric_type", "status"},
		),

		// Graph metrics
		entitiesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_entities_processed_total",
				Help: "Total number of entities processed",
			},
			[]string{"operation", "entity_type"},
		),
		relationshipsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_relationships_processed_total",
				Help: "Total number of relationships processed",
			},
			[]string{"operation", "relationship_type"},
		),
		subgraphSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_subgraph_size",
				Help:    "Size of extracted subgraphs",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
			},
			[]string{"analysis_type"},
		),
		pathLength: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_path_length",
				Help:    "Length of found paths",
				Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20},
			},
			[]string{"algorithm"},
		),
		centralityCalculations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_centrality_calculations_total",
				Help: "Total number of centrality calculations",
			},
			[]string{"centrality_type", "status"},
		),

		// Pattern detection metrics
		patternsDetected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_patterns_detected_total",
				Help: "Total number of patterns detected",
			},
			[]string{"pattern_type", "severity"},
		),
		patternConfidence: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_pattern_confidence",
				Help:    "Confidence scores of detected patterns",
				Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			},
			[]string{"pattern_type"},
		),
		communitiesDetected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_communities_detected_total",
				Help: "Total number of communities detected",
			},
			[]string{"algorithm", "size_category"},
		),
		communitySize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_community_size",
				Help:    "Size of detected communities",
				Buckets: []float64{2, 5, 10, 20, 50, 100, 200, 500, 1000},
			},
			[]string{"algorithm"},
		),
		communityModularity: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_community_modularity",
				Help:    "Modularity scores of detected communities",
				Buckets: []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			},
			[]string{"algorithm"},
		),

		// Investigation metrics
		investigationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_investigations_total",
				Help: "Total number of investigations",
			},
			[]string{"priority", "status"},
		),
		investigationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_investigation_duration_seconds",
				Help:    "Investigation duration in seconds",
				Buckets: []float64{3600, 7200, 14400, 28800, 86400, 172800, 604800, 1209600},
			},
			[]string{"priority"},
		),
		investigationsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_investigations_active",
				Help: "Number of active investigations",
			},
		),
		investigationEntities: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_investigation_entities",
				Help:    "Number of entities in investigations",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"priority"},
		),

		// Database metrics
		dbConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_db_connections",
				Help: "Number of database connections",
			},
		),
		dbConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_db_connections_active",
				Help: "Number of active database connections",
			},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			},
			[]string{"operation", "table"},
		),
		dbQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		dbConnectionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_db_connection_errors_total",
				Help: "Total number of database connection errors",
			},
			[]string{"error_type"},
		),

		// Neo4j metrics
		neo4jConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_neo4j_connections",
				Help: "Number of Neo4j connections",
			},
		),
		neo4jQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_neo4j_query_duration_seconds",
				Help:    "Neo4j query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30},
			},
			[]string{"operation"},
		),
		neo4jQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_neo4j_queries_total",
				Help: "Total number of Neo4j queries",
			},
			[]string{"operation", "status"},
		),
		neo4jConnectionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_neo4j_connection_errors_total",
				Help: "Total number of Neo4j connection errors",
			},
			[]string{"error_type"},
		),
		neo4jSubgraphQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_neo4j_subgraph_queries_total",
				Help: "Total number of Neo4j subgraph queries",
			},
			[]string{"depth", "status"},
		),
		neo4jPathQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_neo4j_path_queries_total",
				Help: "Total number of Neo4j path queries",
			},
			[]string{"algorithm", "status"},
		),
		neo4jCentralityQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_neo4j_centrality_queries_total",
				Help: "Total number of Neo4j centrality queries",
			},
			[]string{"centrality_type", "status"},
		),

		// Kafka metrics
		kafkaMessagesProduced: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_kafka_messages_produced_total",
				Help: "Total number of Kafka messages produced",
			},
			[]string{"topic"},
		),
		kafkaMessagesConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_kafka_messages_consumed_total",
				Help: "Total number of Kafka messages consumed",
			},
			[]string{"topic"},
		),
		kafkaProduceErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_kafka_produce_errors_total",
				Help: "Total number of Kafka produce errors",
			},
			[]string{"topic", "error_type"},
		),
		kafkaConsumeErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "graph_engine_kafka_consume_errors_total",
				Help: "Total number of Kafka consume errors",
			},
			[]string{"topic", "error_type"},
		),
		kafkaConsumerLag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "graph_engine_kafka_consumer_lag",
				Help: "Kafka consumer lag",
			},
			[]string{"topic", "partition"},
		),

		// System metrics
		goroutinesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_goroutines_active",
				Help: "Number of active goroutines",
			},
		),
		memoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),
		cpuUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "graph_engine_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),

		// Performance metrics
		analysisPerformance: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_analysis_performance_ratio",
				Help:    "Analysis performance ratio (entities per second)",
				Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000},
			},
			[]string{"analysis_type"},
		),
		networkComplexity: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_network_complexity",
				Help:    "Network complexity metrics",
				Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			},
			[]string{"metric_type"},
		),
		algorithmPerformance: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "graph_engine_algorithm_performance_seconds",
				Help:    "Algorithm performance in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10, 100, 1000},
			},
			[]string{"algorithm", "input_size_category"},
		),
		cacheHitRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "graph_engine_cache_hit_rate",
				Help: "Cache hit rate",
			},
			[]string{"cache_type"},
		),
	}
}

// Request tracking methods

// IncrementRequests increments request counter
func (m *MetricsCollector) IncrementRequests(method, endpoint, status string) {
	m.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// ObserveRequestDuration observes request duration
func (m *MetricsCollector) ObserveRequestDuration(method, endpoint string, duration time.Duration) {
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// SetRequestsInFlight sets in-flight request gauge
func (m *MetricsCollector) SetRequestsInFlight(method, endpoint string, count int) {
	m.requestsInFlight.WithLabelValues(method, endpoint).Set(float64(count))
}

// Analysis tracking methods

// IncrementAnalysisJobs increments analysis job counter
func (m *MetricsCollector) IncrementAnalysisJobs(jobType, status string) {
	m.analysisJobsTotal.WithLabelValues(jobType, status).Inc()
}

// ObserveAnalysisJobDuration observes analysis job duration
func (m *MetricsCollector) ObserveAnalysisJobDuration(jobType string, duration time.Duration) {
	m.analysisJobDuration.WithLabelValues(jobType).Observe(duration.Seconds())
}

// SetAnalysisJobsActive sets active analysis jobs gauge
func (m *MetricsCollector) SetAnalysisJobsActive(count int) {
	m.analysisJobsActive.Set(float64(count))
}

// SetAnalysisJobsQueued sets queued analysis jobs gauge
func (m *MetricsCollector) SetAnalysisJobsQueued(count int) {
	m.analysisJobsQueued.Set(float64(count))
}

// IncrementSubgraphAnalysis increments subgraph analysis counter
func (m *MetricsCollector) IncrementSubgraphAnalysis(analysisType, status string) {
	m.subgraphAnalysisTotal.WithLabelValues(analysisType, status).Inc()
}

// IncrementPathfinding increments pathfinding counter
func (m *MetricsCollector) IncrementPathfinding(algorithm, status string) {
	m.pathfindingTotal.WithLabelValues(algorithm, status).Inc()
}

// IncrementMetricsCalculation increments metrics calculation counter
func (m *MetricsCollector) IncrementMetricsCalculation(metricType, status string) {
	m.metricsCalculationTotal.WithLabelValues(metricType, status).Inc()
}

// Graph tracking methods

// IncrementEntitiesProcessed increments entities processed counter
func (m *MetricsCollector) IncrementEntitiesProcessed(operation, entityType string, count int) {
	m.entitiesProcessed.WithLabelValues(operation, entityType).Add(float64(count))
}

// IncrementRelationshipsProcessed increments relationships processed counter
func (m *MetricsCollector) IncrementRelationshipsProcessed(operation, relationshipType string, count int) {
	m.relationshipsProcessed.WithLabelValues(operation, relationshipType).Add(float64(count))
}

// ObserveSubgraphSize observes subgraph size
func (m *MetricsCollector) ObserveSubgraphSize(analysisType string, size int) {
	m.subgraphSize.WithLabelValues(analysisType).Observe(float64(size))
}

// ObservePathLength observes path length
func (m *MetricsCollector) ObservePathLength(algorithm string, length int) {
	m.pathLength.WithLabelValues(algorithm).Observe(float64(length))
}

// IncrementCentralityCalculations increments centrality calculation counter
func (m *MetricsCollector) IncrementCentralityCalculations(centralityType, status string) {
	m.centralityCalculations.WithLabelValues(centralityType, status).Inc()
}

// Pattern detection tracking methods

// IncrementPatternsDetected increments patterns detected counter
func (m *MetricsCollector) IncrementPatternsDetected(patternType, severity string) {
	m.patternsDetected.WithLabelValues(patternType, severity).Inc()
}

// ObservePatternConfidence observes pattern confidence
func (m *MetricsCollector) ObservePatternConfidence(patternType string, confidence float64) {
	m.patternConfidence.WithLabelValues(patternType).Observe(confidence)
}

// IncrementCommunitiesDetected increments communities detected counter
func (m *MetricsCollector) IncrementCommunitiesDetected(algorithm, sizeCategory string) {
	m.communitiesDetected.WithLabelValues(algorithm, sizeCategory).Inc()
}

// ObserveCommunitySize observes community size
func (m *MetricsCollector) ObserveCommunitySize(algorithm string, size int) {
	m.communitySize.WithLabelValues(algorithm).Observe(float64(size))
}

// ObserveCommunityModularity observes community modularity
func (m *MetricsCollector) ObserveCommunityModularity(algorithm string, modularity float64) {
	m.communityModularity.WithLabelValues(algorithm).Observe(modularity)
}

// Investigation tracking methods

// IncrementInvestigations increments investigations counter
func (m *MetricsCollector) IncrementInvestigations(priority, status string) {
	m.investigationsTotal.WithLabelValues(priority, status).Inc()
}

// ObserveInvestigationDuration observes investigation duration
func (m *MetricsCollector) ObserveInvestigationDuration(priority string, duration time.Duration) {
	m.investigationDuration.WithLabelValues(priority).Observe(duration.Seconds())
}

// SetInvestigationsActive sets active investigations gauge
func (m *MetricsCollector) SetInvestigationsActive(count int) {
	m.investigationsActive.Set(float64(count))
}

// ObserveInvestigationEntities observes investigation entity count
func (m *MetricsCollector) ObserveInvestigationEntities(priority string, count int) {
	m.investigationEntities.WithLabelValues(priority).Observe(float64(count))
}

// Database tracking methods

// SetDBConnections sets database connections gauge
func (m *MetricsCollector) SetDBConnections(count int) {
	m.dbConnections.Set(float64(count))
}

// SetDBConnectionsActive sets active database connections gauge
func (m *MetricsCollector) SetDBConnectionsActive(count int) {
	m.dbConnectionsActive.Set(float64(count))
}

// ObserveDBQueryDuration observes database query duration
func (m *MetricsCollector) ObserveDBQueryDuration(operation, table string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// IncrementDBQueries increments database queries counter
func (m *MetricsCollector) IncrementDBQueries(operation, table, status string) {
	m.dbQueriesTotal.WithLabelValues(operation, table, status).Inc()
}

// IncrementDBConnectionErrors increments database connection errors counter
func (m *MetricsCollector) IncrementDBConnectionErrors(errorType string) {
	m.dbConnectionErrors.WithLabelValues(errorType).Inc()
}

// Neo4j tracking methods

// SetNeo4jConnections sets Neo4j connections gauge
func (m *MetricsCollector) SetNeo4jConnections(count int) {
	m.neo4jConnections.Set(float64(count))
}

// ObserveNeo4jQueryDuration observes Neo4j query duration
func (m *MetricsCollector) ObserveNeo4jQueryDuration(operation string, duration time.Duration) {
	m.neo4jQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// IncrementNeo4jQueries increments Neo4j queries counter
func (m *MetricsCollector) IncrementNeo4jQueries(operation, status string) {
	m.neo4jQueriesTotal.WithLabelValues(operation, status).Inc()
}

// IncrementNeo4jConnectionErrors increments Neo4j connection errors counter
func (m *MetricsCollector) IncrementNeo4jConnectionErrors(errorType string) {
	m.neo4jConnectionErrors.WithLabelValues(errorType).Inc()
}

// IncrementNeo4jSubgraphQueries increments Neo4j subgraph queries counter
func (m *MetricsCollector) IncrementNeo4jSubgraphQueries(depth, status string) {
	m.neo4jSubgraphQueries.WithLabelValues(depth, status).Inc()
}

// IncrementNeo4jPathQueries increments Neo4j path queries counter
func (m *MetricsCollector) IncrementNeo4jPathQueries(algorithm, status string) {
	m.neo4jPathQueries.WithLabelValues(algorithm, status).Inc()
}

// IncrementNeo4jCentralityQueries increments Neo4j centrality queries counter
func (m *MetricsCollector) IncrementNeo4jCentralityQueries(centralityType, status string) {
	m.neo4jCentralityQueries.WithLabelValues(centralityType, status).Inc()
}

// Kafka tracking methods

// IncrementKafkaMessagesProduced increments Kafka messages produced counter
func (m *MetricsCollector) IncrementKafkaMessagesProduced(topic string) {
	m.kafkaMessagesProduced.WithLabelValues(topic).Inc()
}

// IncrementKafkaMessagesConsumed increments Kafka messages consumed counter
func (m *MetricsCollector) IncrementKafkaMessagesConsumed(topic string) {
	m.kafkaMessagesConsumed.WithLabelValues(topic).Inc()
}

// IncrementKafkaProduceErrors increments Kafka produce errors counter
func (m *MetricsCollector) IncrementKafkaProduceErrors(topic, errorType string) {
	m.kafkaProduceErrors.WithLabelValues(topic, errorType).Inc()
}

// IncrementKafkaConsumeErrors increments Kafka consume errors counter
func (m *MetricsCollector) IncrementKafkaConsumeErrors(topic, errorType string) {
	m.kafkaConsumeErrors.WithLabelValues(topic, errorType).Inc()
}

// SetKafkaConsumerLag sets Kafka consumer lag gauge
func (m *MetricsCollector) SetKafkaConsumerLag(topic, partition string, lag int64) {
	m.kafkaConsumerLag.WithLabelValues(topic, partition).Set(float64(lag))
}

// System tracking methods

// SetGoroutinesActive sets active goroutines gauge
func (m *MetricsCollector) SetGoroutinesActive(count int) {
	m.goroutinesActive.Set(float64(count))
}

// SetMemoryUsage sets memory usage gauge
func (m *MetricsCollector) SetMemoryUsage(bytes uint64) {
	m.memoryUsage.Set(float64(bytes))
}

// SetCPUUsage sets CPU usage gauge
func (m *MetricsCollector) SetCPUUsage(percent float64) {
	m.cpuUsage.Set(percent)
}

// Performance tracking methods

// ObserveAnalysisPerformance observes analysis performance
func (m *MetricsCollector) ObserveAnalysisPerformance(analysisType string, entitiesPerSecond float64) {
	m.analysisPerformance.WithLabelValues(analysisType).Observe(entitiesPerSecond)
}

// ObserveNetworkComplexity observes network complexity
func (m *MetricsCollector) ObserveNetworkComplexity(metricType string, complexity float64) {
	m.networkComplexity.WithLabelValues(metricType).Observe(complexity)
}

// ObserveAlgorithmPerformance observes algorithm performance
func (m *MetricsCollector) ObserveAlgorithmPerformance(algorithm, inputSizeCategory string, duration time.Duration) {
	m.algorithmPerformance.WithLabelValues(algorithm, inputSizeCategory).Observe(duration.Seconds())
}

// SetCacheHitRate sets cache hit rate gauge
func (m *MetricsCollector) SetCacheHitRate(cacheType string, hitRate float64) {
	m.cacheHitRate.WithLabelValues(cacheType).Set(hitRate)
}

// StartPeriodicCollection starts periodic collection of system metrics
func (m *MetricsCollector) StartPeriodicCollection(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (m *MetricsCollector) collectSystemMetrics() {
	// Implementation would collect actual system metrics
	// This is a placeholder for the basic structure
	m.logger.Debug("Collecting system metrics")
}

// GetSizeCategory returns size category for metrics
func GetSizeCategory(size int) string {
	switch {
	case size <= 10:
		return "small"
	case size <= 100:
		return "medium"
	case size <= 1000:
		return "large"
	default:
		return "xlarge"
	}
}