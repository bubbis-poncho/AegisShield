package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"../../internal/config"
	"../../internal/database"
)

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(cfg *config.Config, repos *database.Repositories, logger *zap.Logger) *MetricsCollector {
	return &MetricsCollector{
		config: cfg,
		repos:  repos,
		logger: logger,
	}
}

// CollectModelMetrics collects comprehensive metrics for a model
func (mc *MetricsCollector) CollectModelMetrics(ctx context.Context, modelID string) (*ModelMetrics, error) {
	mc.logger.Debug("Collecting model metrics", zap.String("model_id", modelID))

	// Get model information
	model, err := mc.repos.ModelRepo.GetByID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Collect performance metrics
	perfMetrics, err := mc.collectPerformanceMetrics(ctx, modelID)
	if err != nil {
		mc.logger.Error("Failed to collect performance metrics", zap.Error(err))
		perfMetrics = &PerformanceMetrics{} // Use empty metrics on error
	}

	// Collect prediction metrics
	predMetrics, err := mc.collectPredictionMetrics(ctx, modelID)
	if err != nil {
		mc.logger.Error("Failed to collect prediction metrics", zap.Error(err))
		predMetrics = &PredictionMetrics{} // Use empty metrics on error
	}

	// Collect resource metrics
	resourceMetrics, err := mc.collectResourceMetrics(ctx, modelID)
	if err != nil {
		mc.logger.Error("Failed to collect resource metrics", zap.Error(err))
		resourceMetrics = &ResourceMetrics{} // Use empty metrics on error
	}

	// Collect business metrics
	businessMetrics, err := mc.collectBusinessMetrics(ctx, modelID)
	if err != nil {
		mc.logger.Error("Failed to collect business metrics", zap.Error(err))
		businessMetrics = &BusinessMetrics{} // Use empty metrics on error
	}

	metrics := &ModelMetrics{
		ModelID:           modelID,
		ModelName:         model.Name,
		ModelVersion:      model.Version,
		CollectedAt:       time.Now(),
		Performance:       *perfMetrics,
		Predictions:       *predMetrics,
		Resources:         *resourceMetrics,
		Business:          *businessMetrics,
		Health:            mc.calculateHealthScore(perfMetrics, predMetrics, resourceMetrics),
	}

	// Store metrics snapshot
	if err := mc.storeMetricsSnapshot(ctx, metrics); err != nil {
		mc.logger.Error("Failed to store metrics snapshot", zap.Error(err))
	}

	return metrics, nil
}

// collectPerformanceMetrics collects model performance metrics
func (mc *MetricsCollector) collectPerformanceMetrics(ctx context.Context, modelID string) (*PerformanceMetrics, error) {
	// Get recent prediction results for accuracy calculation
	// This is a simplified implementation - in production you'd have more sophisticated metrics
	
	metrics := &PerformanceMetrics{
		Accuracy:    0.0,
		Precision:   0.0,
		Recall:      0.0,
		F1Score:     0.0,
		AUC:         0.0,
		Latency:     0.0,
		Throughput:  0.0,
		ErrorRate:   0.0,
	}

	// Get recent metrics from database
	recentMetrics, err := mc.repos.MetricsRepo.GetRecentByModelID(ctx, modelID, 100)
	if err != nil {
		return metrics, err
	}

	if len(recentMetrics) == 0 {
		return metrics, nil
	}

	// Calculate aggregated metrics
	var totalLatency, totalAccuracy, totalPrecision, totalRecall, totalF1, totalAUC float64
	var totalRequests, totalErrors int
	
	for _, metric := range recentMetrics {
		if metric.Latency > 0 {
			totalLatency += metric.Latency
		}
		if metric.Accuracy > 0 {
			totalAccuracy += metric.Accuracy
		}
		if metric.Precision > 0 {
			totalPrecision += metric.Precision
		}
		if metric.Recall > 0 {
			totalRecall += metric.Recall
		}
		if metric.F1Score > 0 {
			totalF1 += metric.F1Score
		}
		if metric.AUC > 0 {
			totalAUC += metric.AUC
		}
		totalRequests++
		if metric.ErrorRate > 0 {
			totalErrors++
		}
	}

	count := float64(len(recentMetrics))
	if count > 0 {
		metrics.Accuracy = totalAccuracy / count
		metrics.Precision = totalPrecision / count
		metrics.Recall = totalRecall / count
		metrics.F1Score = totalF1 / count
		metrics.AUC = totalAUC / count
		metrics.Latency = totalLatency / count
		metrics.ErrorRate = float64(totalErrors) / count
		
		// Calculate throughput (requests per second)
		if len(recentMetrics) > 1 {
			timeSpan := recentMetrics[0].CreatedAt.Sub(recentMetrics[len(recentMetrics)-1].CreatedAt).Seconds()
			if timeSpan > 0 {
				metrics.Throughput = float64(totalRequests) / timeSpan
			}
		}
	}

	return metrics, nil
}

// collectPredictionMetrics collects prediction-related metrics
func (mc *MetricsCollector) collectPredictionMetrics(ctx context.Context, modelID string) (*PredictionMetrics, error) {
	metrics := &PredictionMetrics{
		TotalPredictions:   0,
		SuccessfulPredictions: 0,
		FailedPredictions:  0,
		AverageConfidence:  0.0,
		PredictionRate:     0.0,
	}

	// Get prediction counts from database
	predictionCounts, err := mc.repos.PredictionRepo.GetRecentCounts(ctx, modelID, 24*time.Hour)
	if err != nil {
		return metrics, err
	}

	metrics.TotalPredictions = predictionCounts.Total
	metrics.SuccessfulPredictions = predictionCounts.Successful
	metrics.FailedPredictions = predictionCounts.Failed

	// Calculate prediction rate (predictions per hour)
	if predictionCounts.Total > 0 {
		metrics.PredictionRate = float64(predictionCounts.Total) / 24.0
	}

	// Get average confidence from recent predictions
	avgConfidence, err := mc.repos.PredictionRepo.GetAverageConfidence(ctx, modelID, 1000)
	if err != nil {
		mc.logger.Warn("Failed to get average confidence", zap.Error(err))
	} else {
		metrics.AverageConfidence = avgConfidence
	}

	return metrics, nil
}

// collectResourceMetrics collects resource usage metrics
func (mc *MetricsCollector) collectResourceMetrics(ctx context.Context, modelID string) (*ResourceMetrics, error) {
	metrics := &ResourceMetrics{
		CPUUsage:    0.0,
		MemoryUsage: 0.0,
		DiskUsage:   0.0,
		NetworkIO:   0.0,
		GPUUsage:    0.0,
	}

	// In a real implementation, you would collect these from monitoring systems
	// like Prometheus, CloudWatch, or system monitors
	// For now, we'll simulate some values
	
	metrics.CPUUsage = 25.5    // 25.5% CPU usage
	metrics.MemoryUsage = 1024 // 1GB memory usage
	metrics.DiskUsage = 500    // 500MB disk usage
	metrics.NetworkIO = 100    // 100 MB/s network I/O
	metrics.GPUUsage = 0.0     // No GPU usage

	return metrics, nil
}

// collectBusinessMetrics collects business-related metrics
func (mc *MetricsCollector) collectBusinessMetrics(ctx context.Context, modelID string) (*BusinessMetrics, error) {
	metrics := &BusinessMetrics{
		FalsePositiveRate: 0.0,
		FalseNegativeRate: 0.0,
		CostPerPrediction: 0.0,
		BusinessValue:     0.0,
		SLACompliance:     0.0,
	}

	// Get business metrics from database or external systems
	// This would typically involve querying business outcome data
	
	// Simulate some business metrics
	metrics.FalsePositiveRate = 0.05 // 5% false positive rate
	metrics.FalseNegativeRate = 0.02 // 2% false negative rate
	metrics.CostPerPrediction = 0.001 // $0.001 per prediction
	metrics.BusinessValue = 1000.0    // $1000 business value generated
	metrics.SLACompliance = 0.995     // 99.5% SLA compliance

	return metrics, nil
}

// calculateHealthScore calculates overall model health score
func (mc *MetricsCollector) calculateHealthScore(perf *PerformanceMetrics, pred *PredictionMetrics, res *ResourceMetrics) float64 {
	// Simple health score calculation based on multiple factors
	score := 0.0
	factors := 0

	// Performance factors
	if perf.Accuracy > 0 {
		score += perf.Accuracy * 0.3
		factors++
	}
	if perf.ErrorRate >= 0 {
		score += (1.0 - perf.ErrorRate) * 0.2
		factors++
	}

	// Prediction factors
	if pred.TotalPredictions > 0 {
		successRate := float64(pred.SuccessfulPredictions) / float64(pred.TotalPredictions)
		score += successRate * 0.2
		factors++
	}

	// Resource factors
	if res.CPUUsage > 0 && res.CPUUsage < 80 {
		score += (1.0 - res.CPUUsage/100.0) * 0.15
		factors++
	}
	if res.MemoryUsage > 0 {
		// Assume healthy memory usage is below 2GB
		memoryScore := 1.0 - (res.MemoryUsage / 2048.0)
		if memoryScore < 0 {
			memoryScore = 0
		}
		score += memoryScore * 0.15
		factors++
	}

	if factors > 0 {
		return score / float64(factors)
	}

	return 0.5 // Default neutral score
}

// storeMetricsSnapshot stores a snapshot of metrics for historical analysis
func (mc *MetricsCollector) storeMetricsSnapshot(ctx context.Context, metrics *ModelMetrics) error {
	// Convert metrics to JSON for storage
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	snapshot := &database.MetricsSnapshot{
		ModelID:     metrics.ModelID,
		MetricsData: string(metricsJSON),
		CollectedAt: metrics.CollectedAt,
		HealthScore: metrics.Health,
	}

	return mc.repos.MetricsRepo.CreateSnapshot(ctx, snapshot)
}

// GetMetricsHistory returns historical metrics for a model
func (mc *MetricsCollector) GetMetricsHistory(ctx context.Context, modelID string, hours int) ([]*ModelMetrics, error) {
	since := time.Now().Add(time.Duration(-hours) * time.Hour)
	
	snapshots, err := mc.repos.MetricsRepo.GetSnapshotsSince(ctx, modelID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics snapshots: %w", err)
	}

	var metrics []*ModelMetrics
	for _, snapshot := range snapshots {
		var metric ModelMetrics
		if err := json.Unmarshal([]byte(snapshot.MetricsData), &metric); err != nil {
			mc.logger.Error("Failed to unmarshal metrics snapshot", zap.Error(err))
			continue
		}
		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

// GetAggregatedMetrics returns aggregated metrics across multiple models
func (mc *MetricsCollector) GetAggregatedMetrics(ctx context.Context, modelIDs []string) (*AggregatedMetrics, error) {
	aggregated := &AggregatedMetrics{
		ModelCount:       len(modelIDs),
		TotalPredictions: 0,
		AverageAccuracy:  0.0,
		AverageLatency:   0.0,
		OverallHealth:    0.0,
		CollectedAt:      time.Now(),
	}

	if len(modelIDs) == 0 {
		return aggregated, nil
	}

	var totalAccuracy, totalLatency, totalHealth float64
	var validModels int

	for _, modelID := range modelIDs {
		metrics, err := mc.CollectModelMetrics(ctx, modelID)
		if err != nil {
			mc.logger.Error("Failed to collect metrics for model", 
				zap.String("model_id", modelID), zap.Error(err))
			continue
		}

		aggregated.TotalPredictions += metrics.Predictions.TotalPredictions
		
		if metrics.Performance.Accuracy > 0 {
			totalAccuracy += metrics.Performance.Accuracy
		}
		if metrics.Performance.Latency > 0 {
			totalLatency += metrics.Performance.Latency
		}
		totalHealth += metrics.Health
		validModels++
	}

	if validModels > 0 {
		aggregated.AverageAccuracy = totalAccuracy / float64(validModels)
		aggregated.AverageLatency = totalLatency / float64(validModels)
		aggregated.OverallHealth = totalHealth / float64(validModels)
	}

	return aggregated, nil
}