package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"../../internal/config"
	"../../internal/database"
	"../../internal/models"
)

// ModelMonitor manages model performance monitoring and drift detection
type ModelMonitor struct {
	config      *config.Config
	db          *database.Database
	repos       *database.Repositories
	logger      *zap.Logger
	monitors    map[string]*ModelPerformanceMonitor
	driftDetector *DriftDetector
	alertManager *AlertManager
	mu          sync.RWMutex
	stopChan    chan struct{}
	stopped     chan struct{}
}

// ModelPerformanceMonitor tracks performance metrics for a specific model
type ModelPerformanceMonitor struct {
	ModelID         string
	ModelVersion    string
	StartTime       time.Time
	LastUpdate      time.Time
	Metrics         *PerformanceMetrics
	ThresholdAlerts map[string]*AlertThreshold
	History         []*MetricsSnapshot
	mu              sync.RWMutex
}

// PerformanceMetrics holds current performance metrics
type PerformanceMetrics struct {
	RequestCount       int64             `json:"request_count"`
	SuccessCount       int64             `json:"success_count"`
	ErrorCount         int64             `json:"error_count"`
	SuccessRate        float64           `json:"success_rate"`
	ErrorRate          float64           `json:"error_rate"`
	AvgLatency         time.Duration     `json:"avg_latency"`
	P50Latency         time.Duration     `json:"p50_latency"`
	P95Latency         time.Duration     `json:"p95_latency"`
	P99Latency         time.Duration     `json:"p99_latency"`
	ThroughputRPS      float64           `json:"throughput_rps"`
	AccuracyScore      *float64          `json:"accuracy_score,omitempty"`
	PrecisionScore     *float64          `json:"precision_score,omitempty"`
	RecallScore        *float64          `json:"recall_score,omitempty"`
	F1Score            *float64          `json:"f1_score,omitempty"`
	AUCScore           *float64          `json:"auc_score,omitempty"`
	DriftScore         *float64          `json:"drift_score,omitempty"`
	LastUpdated        time.Time         `json:"last_updated"`
	CustomMetrics      map[string]float64 `json:"custom_metrics"`
}

// MetricsSnapshot represents a point-in-time metrics snapshot
type MetricsSnapshot struct {
	Timestamp time.Time           `json:"timestamp"`
	Metrics   *PerformanceMetrics `json:"metrics"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// AlertThreshold defines thresholds for triggering alerts
type AlertThreshold struct {
	MetricName     string    `json:"metric_name"`
	ThresholdType  string    `json:"threshold_type"` // min, max, change_rate
	Value          float64   `json:"value"`
	WindowDuration time.Duration `json:"window_duration"`
	Severity       string    `json:"severity"` // low, medium, high, critical
	Enabled        bool      `json:"enabled"`
	LastTriggered  *time.Time `json:"last_triggered,omitempty"`
}

// DriftDetector detects data and model drift
type DriftDetector struct {
	config         *config.Config
	logger         *zap.Logger
	repos          *database.Repositories
	methods        map[string]DriftDetectionMethod
	mu             sync.RWMutex
}

// DriftDetectionMethod interface for different drift detection algorithms
type DriftDetectionMethod interface {
	DetectDrift(ctx context.Context, reference, current []float64) (*DriftResult, error)
	GetMethodName() string
	GetThreshold() float64
}

// DriftResult represents the result of drift detection
type DriftResult struct {
	IsDrift        bool                   `json:"is_drift"`
	DriftScore     float64                `json:"drift_score"`
	Threshold      float64                `json:"threshold"`
	Method         string                 `json:"method"`
	PValue         *float64               `json:"p_value,omitempty"`
	StatisticValue *float64               `json:"statistic_value,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
	DetectedAt     time.Time              `json:"detected_at"`
}

// AlertManager manages alerts for model monitoring
type AlertManager struct {
	config    *config.Config
	logger    *zap.Logger
	repos     *database.Repositories
	channels  map[string]AlertChannel
	mu        sync.RWMutex
}

// AlertChannel interface for different alert channels
type AlertChannel interface {
	SendAlert(ctx context.Context, alert *Alert) error
	GetChannelType() string
}

// Alert represents a monitoring alert
type Alert struct {
	ID          uuid.UUID              `json:"id"`
	ModelID     string                 `json:"model_id"`
	AlertType   string                 `json:"alert_type"` // performance, drift, error
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// NewModelMonitor creates a new model monitor
func NewModelMonitor(cfg *config.Config, db *database.Database, repos *database.Repositories, logger *zap.Logger) *ModelMonitor {
	monitor := &ModelMonitor{
		config:      cfg,
		db:          db,
		repos:       repos,
		logger:      logger,
		monitors:    make(map[string]*ModelPerformanceMonitor),
		stopChan:    make(chan struct{}),
		stopped:     make(chan struct{}),
	}

	// Initialize drift detector
	monitor.driftDetector = NewDriftDetector(cfg, repos, logger)

	// Initialize alert manager
	monitor.alertManager = NewAlertManager(cfg, repos, logger)

	// Start monitoring loop
	go monitor.startMonitoring()

	return monitor
}

// RegisterModel registers a model for monitoring
func (m *ModelMonitor) RegisterModel(modelID, version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.monitors[modelID]; exists {
		return fmt.Errorf("model already registered: %s", modelID)
	}

	monitor := &ModelPerformanceMonitor{
		ModelID:      modelID,
		ModelVersion: version,
		StartTime:    time.Now(),
		LastUpdate:   time.Now(),
		Metrics: &PerformanceMetrics{
			LastUpdated:   time.Now(),
			CustomMetrics: make(map[string]float64),
		},
		ThresholdAlerts: make(map[string]*AlertThreshold),
		History:         make([]*MetricsSnapshot, 0),
	}

	// Set default thresholds
	monitor.setDefaultThresholds(m.config)

	m.monitors[modelID] = monitor
	m.logger.Info("Model registered for monitoring", zap.String("model_id", modelID))

	return nil
}

// UnregisterModel unregisters a model from monitoring
func (m *ModelMonitor) UnregisterModel(modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.monitors[modelID]; !exists {
		return fmt.Errorf("model not registered: %s", modelID)
	}

	delete(m.monitors, modelID)
	m.logger.Info("Model unregistered from monitoring", zap.String("model_id", modelID))

	return nil
}

// RecordPredictionMetrics records metrics for a prediction request
func (m *ModelMonitor) RecordPredictionMetrics(modelID string, latency time.Duration, success bool, metadata map[string]interface{}) {
	m.mu.RLock()
	monitor, exists := m.monitors[modelID]
	m.mu.RUnlock()

	if !exists {
		m.logger.Warn("Metrics recorded for unregistered model", zap.String("model_id", modelID))
		return
	}

	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	metrics := monitor.Metrics
	metrics.RequestCount++

	if success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}

	// Update rates
	metrics.SuccessRate = float64(metrics.SuccessCount) / float64(metrics.RequestCount)
	metrics.ErrorRate = float64(metrics.ErrorCount) / float64(metrics.RequestCount)

	// Update latency metrics (simplified - in production use histogram)
	if metrics.RequestCount == 1 {
		metrics.AvgLatency = latency
		metrics.P50Latency = latency
		metrics.P95Latency = latency
		metrics.P99Latency = latency
	} else {
		// Simple moving average for demo
		alpha := 0.1
		metrics.AvgLatency = time.Duration(float64(metrics.AvgLatency)*(1-alpha) + float64(latency)*alpha)
	}

	// Calculate throughput (requests per second over last minute)
	now := time.Now()
	if now.Sub(monitor.LastUpdate) >= time.Minute {
		duration := now.Sub(monitor.LastUpdate)
		requestsInWindow := float64(metrics.RequestCount - monitor.getRequestCountAtTime(now.Add(-duration)))
		metrics.ThroughputRPS = requestsInWindow / duration.Seconds()
		monitor.LastUpdate = now
	}

	metrics.LastUpdated = now

	// Store custom metrics from metadata
	if metadata != nil {
		for key, value := range metadata {
			if numValue, ok := value.(float64); ok {
				metrics.CustomMetrics[key] = numValue
			}
		}
	}

	// Check thresholds and trigger alerts if needed
	go m.checkThresholds(modelID, monitor)
}

// RecordModelPerformance records model performance metrics (accuracy, precision, etc.)
func (m *ModelMonitor) RecordModelPerformance(modelID string, performanceMetrics map[string]float64) {
	m.mu.RLock()
	monitor, exists := m.monitors[modelID]
	m.mu.RUnlock()

	if !exists {
		m.logger.Warn("Performance metrics recorded for unregistered model", zap.String("model_id", modelID))
		return
	}

	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	metrics := monitor.Metrics

	// Update performance metrics
	if accuracy, ok := performanceMetrics["accuracy"]; ok {
		metrics.AccuracyScore = &accuracy
	}
	if precision, ok := performanceMetrics["precision"]; ok {
		metrics.PrecisionScore = &precision
	}
	if recall, ok := performanceMetrics["recall"]; ok {
		metrics.RecallScore = &recall
	}
	if f1, ok := performanceMetrics["f1_score"]; ok {
		metrics.F1Score = &f1
	}
	if auc, ok := performanceMetrics["auc"]; ok {
		metrics.AUCScore = &auc
	}

	// Store in custom metrics as well
	for key, value := range performanceMetrics {
		metrics.CustomMetrics[key] = value
	}

	metrics.LastUpdated = time.Now()

	// Store metrics in database
	go m.storeMetricsSnapshot(modelID, monitor)
}

// DetectDrift performs drift detection for a model
func (m *ModelMonitor) DetectDrift(ctx context.Context, modelID, featureName string, currentData []float64) (*DriftResult, error) {
	// Get reference data (from training or previous window)
	referenceData, err := m.getReferenceData(modelID, featureName)
	if err != nil {
		return nil, fmt.Errorf("failed to get reference data: %w", err)
	}

	// Perform drift detection
	result, err := m.driftDetector.DetectDrift(ctx, featureName, referenceData, currentData)
	if err != nil {
		return nil, fmt.Errorf("drift detection failed: %w", err)
	}

	// Store drift result
	driftRecord := &models.DataDrift{
		ModelID:        uuid.MustParse(modelID),
		FeatureName:    featureName,
		DriftMethod:    result.Method,
		DriftScore:     result.DriftScore,
		Threshold:      result.Threshold,
		IsDrift:        result.IsDrift,
		DetectedAt:     result.DetectedAt,
		ReferenceStart: time.Now().Add(-24 * time.Hour), // Simplified
		ReferenceEnd:   time.Now().Add(-1 * time.Hour),
		CurrentStart:   time.Now().Add(-1 * time.Hour),
		CurrentEnd:     time.Now(),
	}

	// Serialize statistics
	if result.Metadata != nil {
		metadata, _ := json.Marshal(result.Metadata)
		driftRecord.DriftDetails = models.JSON(metadata)
	}

	if err := m.repos.DataDrift.Create(driftRecord); err != nil {
		m.logger.Error("Failed to store drift record", zap.Error(err))
	}

	// Update model metrics
	m.mu.RLock()
	monitor, exists := m.monitors[modelID]
	m.mu.RUnlock()

	if exists {
		monitor.mu.Lock()
		monitor.Metrics.DriftScore = &result.DriftScore
		monitor.mu.Unlock()
	}

	// Trigger alert if drift detected
	if result.IsDrift {
		alert := &Alert{
			ID:        uuid.New(),
			ModelID:   modelID,
			AlertType: "drift",
			Severity:  "medium",
			Title:     fmt.Sprintf("Data drift detected for feature %s", featureName),
			Description: fmt.Sprintf("Drift score: %.4f (threshold: %.4f)", result.DriftScore, result.Threshold),
			Metrics: map[string]interface{}{
				"feature_name": featureName,
				"drift_score":  result.DriftScore,
				"threshold":    result.Threshold,
				"method":       result.Method,
			},
			CreatedAt: time.Now(),
		}

		go m.alertManager.SendAlert(ctx, alert)
	}

	return result, nil
}

// GetModelMetrics returns current metrics for a model
func (m *ModelMonitor) GetModelMetrics(modelID string) (*PerformanceMetrics, error) {
	m.mu.RLock()
	monitor, exists := m.monitors[modelID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model not registered: %s", modelID)
	}

	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	metrics := *monitor.Metrics
	return &metrics, nil
}

// GetModelHistory returns historical metrics for a model
func (m *ModelMonitor) GetModelHistory(modelID string, since time.Time) ([]*MetricsSnapshot, error) {
	m.mu.RLock()
	monitor, exists := m.monitors[modelID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model not registered: %s", modelID)
	}

	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	var history []*MetricsSnapshot
	for _, snapshot := range monitor.History {
		if snapshot.Timestamp.After(since) {
			history = append(history, snapshot)
		}
	}

	return history, nil
}

// startMonitoring starts the monitoring loop
func (m *ModelMonitor) startMonitoring() {
	defer close(m.stopped)

	ticker := time.NewTicker(m.config.ML.ModelMonitoring.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.performPeriodicMonitoring()
		}
	}
}

// performPeriodicMonitoring performs periodic monitoring tasks
func (m *ModelMonitor) performPeriodicMonitoring() {
	m.mu.RLock()
	monitors := make(map[string]*ModelPerformanceMonitor)
	for k, v := range m.monitors {
		monitors[k] = v
	}
	m.mu.RUnlock()

	for modelID, monitor := range monitors {
		// Create snapshot
		monitor.mu.RLock()
		snapshot := &MetricsSnapshot{
			Timestamp: time.Now(),
			Metrics:   monitor.Metrics,
			Metadata: map[string]interface{}{
				"model_version": monitor.ModelVersion,
			},
		}
		monitor.mu.RUnlock()

		// Store snapshot
		monitor.mu.Lock()
		monitor.History = append(monitor.History, snapshot)
		
		// Keep only last 1000 snapshots
		if len(monitor.History) > 1000 {
			monitor.History = monitor.History[len(monitor.History)-1000:]
		}
		monitor.mu.Unlock()

		// Trigger periodic drift detection if enabled
		if m.config.ML.ModelMonitoring.DriftDetection.EnableDriftDetection {
			// This would trigger drift detection for all features
			// Implementation depends on feature store integration
		}
	}
}

// checkThresholds checks if any thresholds are exceeded
func (m *ModelMonitor) checkThresholds(modelID string, monitor *ModelPerformanceMonitor) {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	metrics := monitor.Metrics
	now := time.Now()

	for _, threshold := range monitor.ThresholdAlerts {
		if !threshold.Enabled {
			continue
		}

		// Check if threshold was recently triggered
		if threshold.LastTriggered != nil && now.Sub(*threshold.LastTriggered) < threshold.WindowDuration {
			continue
		}

		var currentValue float64
		var violated bool

		// Get current metric value
		switch threshold.MetricName {
		case "error_rate":
			currentValue = metrics.ErrorRate
		case "success_rate":
			currentValue = metrics.SuccessRate
		case "avg_latency":
			currentValue = float64(metrics.AvgLatency.Milliseconds())
		case "throughput_rps":
			currentValue = metrics.ThroughputRPS
		case "accuracy":
			if metrics.AccuracyScore != nil {
				currentValue = *metrics.AccuracyScore
			}
		case "drift_score":
			if metrics.DriftScore != nil {
				currentValue = *metrics.DriftScore
			}
		default:
			if value, exists := metrics.CustomMetrics[threshold.MetricName]; exists {
				currentValue = value
			} else {
				continue
			}
		}

		// Check threshold violation
		switch threshold.ThresholdType {
		case "min":
			violated = currentValue < threshold.Value
		case "max":
			violated = currentValue > threshold.Value
		}

		if violated {
			alert := &Alert{
				ID:        uuid.New(),
				ModelID:   modelID,
				AlertType: "performance",
				Severity:  threshold.Severity,
				Title:     fmt.Sprintf("%s threshold exceeded", threshold.MetricName),
				Description: fmt.Sprintf("Current value: %.4f, Threshold: %.4f", currentValue, threshold.Value),
				Metrics: map[string]interface{}{
					"metric_name":    threshold.MetricName,
					"current_value":  currentValue,
					"threshold":      threshold.Value,
					"threshold_type": threshold.ThresholdType,
				},
				CreatedAt: time.Now(),
			}

			// Update last triggered time
			threshold.LastTriggered = &now

			// Send alert
			go m.alertManager.SendAlert(context.Background(), alert)
		}
	}
}

// Helper methods

func (m *ModelPerformanceMonitor) setDefaultThresholds(cfg *config.Config) {
	thresholds := map[string]*AlertThreshold{
		"error_rate": {
			MetricName:     "error_rate",
			ThresholdType:  "max",
			Value:          cfg.ML.ModelMonitoring.AlertThresholds.ErrorRateLimit,
			WindowDuration: 5 * time.Minute,
			Severity:       "medium",
			Enabled:        true,
		},
		"avg_latency": {
			MetricName:     "avg_latency",
			ThresholdType:  "max",
			Value:          float64(cfg.ML.ModelMonitoring.PerformanceMonitoring.LatencyThreshold.Milliseconds()),
			WindowDuration: 5 * time.Minute,
			Severity:       "medium",
			Enabled:        true,
		},
		"accuracy": {
			MetricName:     "accuracy",
			ThresholdType:  "min",
			Value:          cfg.ML.ModelMonitoring.PerformanceMonitoring.AccuracyThreshold,
			WindowDuration: 15 * time.Minute,
			Severity:       "high",
			Enabled:        true,
		},
	}

	for name, threshold := range thresholds {
		m.ThresholdAlerts[name] = threshold
	}
}

func (m *ModelPerformanceMonitor) getRequestCountAtTime(t time.Time) int64 {
	// Simplified implementation - in production, use time-series data
	return 0
}

func (m *ModelMonitor) getReferenceData(modelID, featureName string) ([]float64, error) {
	// This would fetch reference data from feature store or training data
	// For now, return simulated reference data
	referenceData := make([]float64, 1000)
	for i := range referenceData {
		referenceData[i] = math.Sin(float64(i)*0.1) + 0.1*float64(i%10)
	}
	return referenceData, nil
}

func (m *ModelMonitor) storeMetricsSnapshot(modelID string, monitor *ModelPerformanceMonitor) {
	monitor.mu.RLock()
	metrics := monitor.Metrics
	monitor.mu.RUnlock()

	// Store key metrics in database
	metricRecords := []*models.ModelMetric{
		{
			ModelID:     uuid.MustParse(modelID),
			MetricName:  "success_rate",
			MetricValue: metrics.SuccessRate,
			MetricType:  models.MetricTypeCustom,
			RecordedAt:  time.Now(),
		},
		{
			ModelID:     uuid.MustParse(modelID),
			MetricName:  "error_rate",
			MetricValue: metrics.ErrorRate,
			MetricType:  models.MetricTypeErrorRate,
			RecordedAt:  time.Now(),
		},
		{
			ModelID:     uuid.MustParse(modelID),
			MetricName:  "avg_latency",
			MetricValue: float64(metrics.AvgLatency.Milliseconds()),
			MetricType:  models.MetricTypeLatency,
			RecordedAt:  time.Now(),
		},
		{
			ModelID:     uuid.MustParse(modelID),
			MetricName:  "throughput_rps",
			MetricValue: metrics.ThroughputRPS,
			MetricType:  models.MetricTypeThroughput,
			RecordedAt:  time.Now(),
		},
	}

	// Add performance metrics if available
	if metrics.AccuracyScore != nil {
		metricRecords = append(metricRecords, &models.ModelMetric{
			ModelID:     uuid.MustParse(modelID),
			MetricName:  "accuracy",
			MetricValue: *metrics.AccuracyScore,
			MetricType:  models.MetricTypeAccuracy,
			RecordedAt:  time.Now(),
		})
	}

	if err := m.repos.ModelMetric.CreateBatch(metricRecords); err != nil {
		m.logger.Error("Failed to store metrics snapshot", zap.Error(err))
	}
}

// Shutdown gracefully shuts down the model monitor
func (m *ModelMonitor) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down model monitor")
	close(m.stopChan)
	
	select {
	case <-m.stopped:
		m.logger.Info("Model monitor shutdown complete")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}