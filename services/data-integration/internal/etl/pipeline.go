package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aegisshield/data-integration/internal/config"
	"github.com/aegisshield/data-integration/internal/lineage"
	"github.com/aegisshield/data-integration/internal/quality"
	"github.com/aegisshield/data-integration/internal/storage"
	"github.com/aegisshield/data-integration/internal/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Pipeline represents an ETL pipeline
type Pipeline struct {
	config          config.Config
	validator       *validation.Validator
	qualityChecker  *quality.Checker
	lineageTracker  *lineage.Tracker
	storageManager  *storage.Manager
	logger          *zap.Logger
	jobQueue        chan *Job
	workerPool      sync.WaitGroup
	shutdown        chan struct{}
	metrics         *PipelineMetrics
}

// Job represents an ETL job
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Data        interface{}            `json:"data"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      JobStatus              `json:"status"`
	Error       string                 `json:"error,omitempty"`
	Metrics     *JobMetrics            `json:"metrics,omitempty"`
}

// JobStatus represents the status of an ETL job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// JobMetrics represents metrics for an ETL job
type JobMetrics struct {
	RecordsProcessed int           `json:"records_processed"`
	RecordsValid     int           `json:"records_valid"`
	RecordsInvalid   int           `json:"records_invalid"`
	ProcessingTime   time.Duration `json:"processing_time"`
	ValidationTime   time.Duration `json:"validation_time"`
	QualityScore     float64       `json:"quality_score"`
}

// PipelineMetrics represents metrics for the ETL pipeline
type PipelineMetrics struct {
	JobsTotal       int64         `json:"jobs_total"`
	JobsCompleted   int64         `json:"jobs_completed"`
	JobsFailed      int64         `json:"jobs_failed"`
	RecordsTotal    int64         `json:"records_total"`
	RecordsValid    int64         `json:"records_valid"`
	RecordsInvalid  int64         `json:"records_invalid"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	mu              sync.RWMutex
}

// ProcessingOptions represents options for data processing
type ProcessingOptions struct {
	SkipValidation     bool                   `json:"skip_validation"`
	SkipQualityChecks  bool                   `json:"skip_quality_checks"`
	SkipLineageTracking bool                  `json:"skip_lineage_tracking"`
	CustomTransforms   []TransformFunction    `json:"-"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// TransformFunction represents a data transformation function
type TransformFunction func(context.Context, interface{}) (interface{}, error)

// NewPipeline creates a new ETL pipeline
func NewPipeline(
	config config.Config,
	validator *validation.Validator,
	qualityChecker *quality.Checker,
	lineageTracker *lineage.Tracker,
	storageManager *storage.Manager,
	logger *zap.Logger,
) *Pipeline {
	return &Pipeline{
		config:         config,
		validator:      validator,
		qualityChecker: qualityChecker,
		lineageTracker: lineageTracker,
		storageManager: storageManager,
		logger:         logger,
		jobQueue:       make(chan *Job, config.ETL.MaxConcurrentJobs*2),
		shutdown:       make(chan struct{}),
		metrics:        &PipelineMetrics{},
	}
}

// Start starts the ETL pipeline workers
func (p *Pipeline) Start(ctx context.Context) error {
	p.logger.Info("Starting ETL pipeline",
		zap.Int("max_concurrent_jobs", p.config.ETL.MaxConcurrentJobs))

	// Start worker pool
	for i := 0; i < p.config.ETL.MaxConcurrentJobs; i++ {
		p.workerPool.Add(1)
		go p.worker(ctx, i)
	}

	// Start metrics reporter
	go p.metricsReporter(ctx)

	return nil
}

// Stop stops the ETL pipeline
func (p *Pipeline) Stop() error {
	p.logger.Info("Stopping ETL pipeline")

	close(p.shutdown)
	close(p.jobQueue)
	
	// Wait for all workers to finish
	done := make(chan struct{})
	go func() {
		p.workerPool.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		p.logger.Info("ETL pipeline stopped gracefully")
	case <-time.After(30 * time.Second):
		p.logger.Warn("ETL pipeline stop timeout")
	}

	return nil
}

// SubmitJob submits a new job to the pipeline
func (p *Pipeline) SubmitJob(job *Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}

	job.Status = JobStatusPending

	select {
	case p.jobQueue <- job:
		p.logger.Info("Job submitted",
			zap.String("job_id", job.ID),
			zap.String("type", job.Type),
			zap.String("source", job.Source))
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

// ProcessData processes data through the ETL pipeline
func (p *Pipeline) ProcessData(ctx context.Context, data interface{}, options *ProcessingOptions) (*JobMetrics, error) {
	if options == nil {
		options = &ProcessingOptions{}
	}

	job := &Job{
		ID:        uuid.New().String(),
		Type:      "batch_processing",
		Data:      data,
		CreatedAt: time.Now(),
		Status:    JobStatusRunning,
		Metrics:   &JobMetrics{},
	}

	startTime := time.Now()
	job.StartedAt = &startTime

	p.logger.Info("Processing data",
		zap.String("job_id", job.ID),
		zap.Bool("skip_validation", options.SkipValidation),
		zap.Bool("skip_quality_checks", options.SkipQualityChecks))

	// Process the data
	result, err := p.processJobData(ctx, job, options)
	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
		p.updateMetrics(job)
		return job.Metrics, err
	}

	completedTime := time.Now()
	job.CompletedAt = &completedTime
	job.Status = JobStatusCompleted
	job.Metrics.ProcessingTime = completedTime.Sub(*job.StartedAt)

	// Update metrics
	p.updateMetrics(job)

	p.logger.Info("Data processing completed",
		zap.String("job_id", job.ID),
		zap.Duration("processing_time", job.Metrics.ProcessingTime),
		zap.Int("records_processed", job.Metrics.RecordsProcessed))

	// Store result if needed
	if options.Metadata != nil {
		if err := p.storageManager.Store(ctx, job.ID, result, options.Metadata); err != nil {
			p.logger.Error("Failed to store processing result",
				zap.String("job_id", job.ID),
				zap.Error(err))
		}
	}

	return job.Metrics, nil
}

// GetJobStatus returns the status of a job
func (p *Pipeline) GetJobStatus(jobID string) (*Job, error) {
	// In a real implementation, this would query a job storage/database
	// For now, return a placeholder
	return &Job{
		ID:     jobID,
		Status: JobStatusCompleted,
	}, nil
}

// GetMetrics returns pipeline metrics
func (p *Pipeline) GetMetrics() *PipelineMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &PipelineMetrics{
		JobsTotal:         p.metrics.JobsTotal,
		JobsCompleted:     p.metrics.JobsCompleted,
		JobsFailed:        p.metrics.JobsFailed,
		RecordsTotal:      p.metrics.RecordsTotal,
		RecordsValid:      p.metrics.RecordsValid,
		RecordsInvalid:    p.metrics.RecordsInvalid,
		AvgProcessingTime: p.metrics.AvgProcessingTime,
	}
}

// worker processes jobs from the queue
func (p *Pipeline) worker(ctx context.Context, workerID int) {
	defer p.workerPool.Done()

	p.logger.Info("Starting ETL worker", zap.Int("worker_id", workerID))

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				p.logger.Info("Job queue closed, stopping worker", zap.Int("worker_id", workerID))
				return
			}

			p.processJob(ctx, job, workerID)

		case <-p.shutdown:
			p.logger.Info("Shutdown signal received, stopping worker", zap.Int("worker_id", workerID))
			return

		case <-ctx.Done():
			p.logger.Info("Context cancelled, stopping worker", zap.Int("worker_id", workerID))
			return
		}
	}
}

// processJob processes a single job
func (p *Pipeline) processJob(ctx context.Context, job *Job, workerID int) {
	startTime := time.Now()
	job.StartedAt = &startTime
	job.Status = JobStatusRunning

	p.logger.Info("Processing job",
		zap.String("job_id", job.ID),
		zap.String("type", job.Type),
		zap.Int("worker_id", workerID))

	// Process the job data
	options := &ProcessingOptions{
		Metadata: map[string]interface{}{
			"worker_id": workerID,
			"job_id":    job.ID,
		},
	}

	_, err := p.processJobData(ctx, job, options)
	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
		p.logger.Error("Job processing failed",
			zap.String("job_id", job.ID),
			zap.Error(err))
	} else {
		job.Status = JobStatusCompleted
		completedTime := time.Now()
		job.CompletedAt = &completedTime
		job.Metrics.ProcessingTime = completedTime.Sub(*job.StartedAt)

		p.logger.Info("Job processing completed",
			zap.String("job_id", job.ID),
			zap.Duration("processing_time", job.Metrics.ProcessingTime))
	}

	// Update metrics
	p.updateMetrics(job)
}

// processJobData processes the actual job data
func (p *Pipeline) processJobData(ctx context.Context, job *Job, options *ProcessingOptions) (interface{}, error) {
	job.Metrics = &JobMetrics{}

	// Extract records from data
	records, err := p.extractRecords(job.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract records: %w", err)
	}

	job.Metrics.RecordsProcessed = len(records)

	// Validate data if enabled
	if !options.SkipValidation && p.validator != nil {
		validationStart := time.Now()
		
		validRecords, invalidRecords, err := p.validator.ValidateRecords(ctx, records)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}

		job.Metrics.RecordsValid = len(validRecords)
		job.Metrics.RecordsInvalid = len(invalidRecords)
		job.Metrics.ValidationTime = time.Since(validationStart)
		records = validRecords

		p.logger.Info("Data validation completed",
			zap.String("job_id", job.ID),
			zap.Int("valid_records", len(validRecords)),
			zap.Int("invalid_records", len(invalidRecords)))
	}

	// Apply custom transforms
	for _, transform := range options.CustomTransforms {
		transformedData, err := transform(ctx, records)
		if err != nil {
			return nil, fmt.Errorf("transform failed: %w", err)
		}
		
		transformedRecords, err := p.extractRecords(transformedData)
		if err != nil {
			return nil, fmt.Errorf("failed to extract transformed records: %w", err)
		}
		records = transformedRecords
	}

	// Check data quality if enabled
	if !options.SkipQualityChecks && p.qualityChecker != nil {
		qualityReport, err := p.qualityChecker.CheckQuality(ctx, records)
		if err != nil {
			p.logger.Warn("Quality check failed",
				zap.String("job_id", job.ID),
				zap.Error(err))
		} else {
			job.Metrics.QualityScore = qualityReport.OverallScore
			
			p.logger.Info("Data quality check completed",
				zap.String("job_id", job.ID),
				zap.Float64("quality_score", qualityReport.OverallScore))
		}
	}

	// Track lineage if enabled
	if !options.SkipLineageTracking && p.lineageTracker != nil {
		lineageInfo := &lineage.LineageInfo{
			JobID:       job.ID,
			Source:      job.Source,
			Target:      job.Target,
			RecordCount: len(records),
			ProcessedAt: time.Now(),
			Metadata:    options.Metadata,
		}

		if err := p.lineageTracker.Track(ctx, lineageInfo); err != nil {
			p.logger.Warn("Failed to track lineage",
				zap.String("job_id", job.ID),
				zap.Error(err))
		}
	}

	return records, nil
}

// extractRecords extracts records from various data formats
func (p *Pipeline) extractRecords(data interface{}) ([]map[string]interface{}, error) {
	switch v := data.(type) {
	case []map[string]interface{}:
		return v, nil
	case map[string]interface{}:
		return []map[string]interface{}{v}, nil
	case []interface{}:
		records := make([]map[string]interface{}, len(v))
		for i, item := range v {
			if record, ok := item.(map[string]interface{}); ok {
				records[i] = record
			} else {
				return nil, fmt.Errorf("invalid record format at index %d", i)
			}
		}
		return records, nil
	case string:
		// Try to parse as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err != nil {
			return nil, fmt.Errorf("failed to parse JSON data: %w", err)
		}
		return p.extractRecords(jsonData)
	default:
		return nil, fmt.Errorf("unsupported data format: %T", data)
	}
}

// updateMetrics updates pipeline metrics
func (p *Pipeline) updateMetrics(job *Job) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.JobsTotal++

	if job.Status == JobStatusCompleted {
		p.metrics.JobsCompleted++
		if job.Metrics != nil {
			p.metrics.RecordsTotal += int64(job.Metrics.RecordsProcessed)
			p.metrics.RecordsValid += int64(job.Metrics.RecordsValid)
			p.metrics.RecordsInvalid += int64(job.Metrics.RecordsInvalid)

			// Update average processing time
			if p.metrics.JobsCompleted > 0 {
				totalTime := p.metrics.AvgProcessingTime * time.Duration(p.metrics.JobsCompleted-1)
				totalTime += job.Metrics.ProcessingTime
				p.metrics.AvgProcessingTime = totalTime / time.Duration(p.metrics.JobsCompleted)
			}
		}
	} else if job.Status == JobStatusFailed {
		p.metrics.JobsFailed++
	}
}

// metricsReporter periodically reports pipeline metrics
func (p *Pipeline) metricsReporter(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := p.GetMetrics()
			p.logger.Info("Pipeline metrics",
				zap.Int64("jobs_total", metrics.JobsTotal),
				zap.Int64("jobs_completed", metrics.JobsCompleted),
				zap.Int64("jobs_failed", metrics.JobsFailed),
				zap.Int64("records_total", metrics.RecordsTotal),
				zap.Duration("avg_processing_time", metrics.AvgProcessingTime))

		case <-ctx.Done():
			return
		}
	}
}

// Schema evolution methods

// UpdateSchema updates the schema for a data source
func (p *Pipeline) UpdateSchema(ctx context.Context, source string, schema map[string]interface{}) error {
	p.logger.Info("Updating schema",
		zap.String("source", source))

	// In a real implementation, this would:
	// 1. Validate the new schema
	// 2. Check compatibility with existing data
	// 3. Update schema registry
	// 4. Notify downstream consumers

	return nil
}

// GetSchema returns the current schema for a data source
func (p *Pipeline) GetSchema(source string) (map[string]interface{}, error) {
	// In a real implementation, this would query the schema registry
	return map[string]interface{}{
		"version": "1.0",
		"fields": map[string]interface{}{
			"id":         "string",
			"timestamp": "datetime",
			"amount":    "decimal",
		},
	}, nil
}

// Batch processing methods

// ProcessBatch processes a batch of data
func (p *Pipeline) ProcessBatch(ctx context.Context, batchData []interface{}, options *ProcessingOptions) (*JobMetrics, error) {
	// Split large batches into smaller chunks
	batchSize := p.config.ETL.BatchSize
	if len(batchData) <= batchSize {
		return p.ProcessData(ctx, batchData, options)
	}

	// Process in chunks
	totalMetrics := &JobMetrics{}
	for i := 0; i < len(batchData); i += batchSize {
		end := i + batchSize
		if end > len(batchData) {
			end = len(batchData)
		}

		chunk := batchData[i:end]
		chunkMetrics, err := p.ProcessData(ctx, chunk, options)
		if err != nil {
			return totalMetrics, err
		}

		// Aggregate metrics
		totalMetrics.RecordsProcessed += chunkMetrics.RecordsProcessed
		totalMetrics.RecordsValid += chunkMetrics.RecordsValid
		totalMetrics.RecordsInvalid += chunkMetrics.RecordsInvalid
		totalMetrics.ProcessingTime += chunkMetrics.ProcessingTime
		totalMetrics.ValidationTime += chunkMetrics.ValidationTime
	}

	return totalMetrics, nil
}

// Stream processing methods

// ProcessStream processes streaming data
func (p *Pipeline) ProcessStream(ctx context.Context, dataStream <-chan interface{}, options *ProcessingOptions) error {
	buffer := make([]interface{}, 0, p.config.ETL.BatchSize)
	ticker := time.NewTicker(p.config.ETL.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-dataStream:
			if !ok {
				// Stream closed, process remaining buffer
				if len(buffer) > 0 {
					_, err := p.ProcessData(ctx, buffer, options)
					return err
				}
				return nil
			}

			buffer = append(buffer, data)

			// Process when buffer is full
			if len(buffer) >= p.config.ETL.BatchSize {
				if _, err := p.ProcessData(ctx, buffer, options); err != nil {
					return err
				}
				buffer = buffer[:0] // Reset buffer
			}

		case <-ticker.C:
			// Process buffer on interval
			if len(buffer) > 0 {
				if _, err := p.ProcessData(ctx, buffer, options); err != nil {
					return err
				}
				buffer = buffer[:0] // Reset buffer
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}