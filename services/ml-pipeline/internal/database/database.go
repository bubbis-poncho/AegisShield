package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"../../internal/config"
	"../../internal/models"
)

// Database wraps the GORM database connection
type Database struct {
	*gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	// Configure GORM logger
	logLevel := logger.Silent
	if cfg.SSLMode == "disable" { // Development mode indicator
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return &Database{DB: db}, nil
}

// AutoMigrate runs automatic migration for all models
func (db *Database) AutoMigrate() error {
	return db.DB.AutoMigrate(
		&models.Model{},
		&models.TrainingJob{},
		&models.Deployment{},
		&models.Experiment{},
		&models.ABTest{},
		&models.ModelMetric{},
		&models.Feature{},
		&models.DataDrift{},
		&models.PredictionRequest{},
	)
}

// Close closes the database connection
func (db *Database) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks the database connection
func (db *Database) Health() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// ModelRepository provides database operations for models
type ModelRepository struct {
	db *Database
}

// NewModelRepository creates a new model repository
func NewModelRepository(db *Database) *ModelRepository {
	return &ModelRepository{db: db}
}

// Create creates a new model
func (r *ModelRepository) Create(model *models.Model) error {
	return r.db.Create(model).Error
}

// GetByID retrieves a model by ID
func (r *ModelRepository) GetByID(id string) (*models.Model, error) {
	var model models.Model
	err := r.db.Preload("TrainingJobs").
		Preload("Deployments").
		Preload("Experiments").
		Preload("ABTests").
		Preload("ModelMetrics").
		First(&model, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// GetByName retrieves a model by name and version
func (r *ModelRepository) GetByName(name, version string) (*models.Model, error) {
	var model models.Model
	err := r.db.Where("name = ? AND version = ?", name, version).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// List retrieves models with pagination and filtering
func (r *ModelRepository) List(filters map[string]interface{}, limit, offset int) ([]*models.Model, int64, error) {
	var models []*models.Model
	var total int64

	query := r.db.Model(&models.Model{})

	// Apply filters
	for key, value := range filters {
		switch key {
		case "type":
			query = query.Where("type = ?", value)
		case "status":
			query = query.Where("status = ?", value)
		case "algorithm":
			query = query.Where("algorithm = ?", value)
		case "is_active":
			query = query.Where("is_active = ?", value)
		case "created_by":
			query = query.Where("created_by = ?", value)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	return models, total, err
}

// Update updates a model
func (r *ModelRepository) Update(model *models.Model) error {
	return r.db.Save(model).Error
}

// Delete soft deletes a model
func (r *ModelRepository) Delete(id string) error {
	return r.db.Delete(&models.Model{}, "id = ?", id).Error
}

// GetActiveModels retrieves all active models
func (r *ModelRepository) GetActiveModels() ([]*models.Model, error) {
	var models []*models.Model
	err := r.db.Where("is_active = ? AND status = ?", true, models.ModelStatusDeployed).
		Find(&models).Error
	return models, err
}

// GetModelsByType retrieves models by type
func (r *ModelRepository) GetModelsByType(modelType models.ModelType) ([]*models.Model, error) {
	var models []*models.Model
	err := r.db.Where("type = ?", modelType).
		Order("created_at DESC").
		Find(&models).Error
	return models, err
}

// TrainingJobRepository provides database operations for training jobs
type TrainingJobRepository struct {
	db *Database
}

// NewTrainingJobRepository creates a new training job repository
func NewTrainingJobRepository(db *Database) *TrainingJobRepository {
	return &TrainingJobRepository{db: db}
}

// Create creates a new training job
func (r *TrainingJobRepository) Create(job *models.TrainingJob) error {
	return r.db.Create(job).Error
}

// GetByID retrieves a training job by ID
func (r *TrainingJobRepository) GetByID(id string) (*models.TrainingJob, error) {
	var job models.TrainingJob
	err := r.db.Preload("Model").
		Preload("Experiments").
		First(&job, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetByModelID retrieves training jobs for a model
func (r *TrainingJobRepository) GetByModelID(modelID string) ([]*models.TrainingJob, error) {
	var jobs []*models.TrainingJob
	err := r.db.Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&jobs).Error
	return jobs, err
}

// GetRunningJobs retrieves all currently running training jobs
func (r *TrainingJobRepository) GetRunningJobs() ([]*models.TrainingJob, error) {
	var jobs []*models.TrainingJob
	err := r.db.Where("status = ?", models.TrainingStatusRunning).
		Find(&jobs).Error
	return jobs, err
}

// Update updates a training job
func (r *TrainingJobRepository) Update(job *models.TrainingJob) error {
	return r.db.Save(job).Error
}

// DeploymentRepository provides database operations for deployments
type DeploymentRepository struct {
	db *Database
}

// NewDeploymentRepository creates a new deployment repository
func NewDeploymentRepository(db *Database) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

// Create creates a new deployment
func (r *DeploymentRepository) Create(deployment *models.Deployment) error {
	return r.db.Create(deployment).Error
}

// GetByID retrieves a deployment by ID
func (r *DeploymentRepository) GetByID(id string) (*models.Deployment, error) {
	var deployment models.Deployment
	err := r.db.Preload("Model").
		Preload("ABTests").
		First(&deployment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// GetByModelID retrieves deployments for a model
func (r *DeploymentRepository) GetByModelID(modelID string) ([]*models.Deployment, error) {
	var deployments []*models.Deployment
	err := r.db.Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&deployments).Error
	return deployments, err
}

// GetActiveDeployments retrieves all active deployments
func (r *DeploymentRepository) GetActiveDeployments() ([]*models.Deployment, error) {
	var deployments []*models.Deployment
	err := r.db.Where("status = ?", models.DeploymentStatusActive).
		Find(&deployments).Error
	return deployments, err
}

// GetByEnvironment retrieves deployments by environment
func (r *DeploymentRepository) GetByEnvironment(environment string) ([]*models.Deployment, error) {
	var deployments []*models.Deployment
	err := r.db.Where("environment = ?", environment).
		Order("created_at DESC").
		Find(&deployments).Error
	return deployments, err
}

// Update updates a deployment
func (r *DeploymentRepository) Update(deployment *models.Deployment) error {
	return r.db.Save(deployment).Error
}

// Delete soft deletes a deployment
func (r *DeploymentRepository) Delete(id string) error {
	return r.db.Delete(&models.Deployment{}, "id = ?", id).Error
}

// ExperimentRepository provides database operations for experiments
type ExperimentRepository struct {
	db *Database
}

// NewExperimentRepository creates a new experiment repository
func NewExperimentRepository(db *Database) *ExperimentRepository {
	return &ExperimentRepository{db: db}
}

// Create creates a new experiment
func (r *ExperimentRepository) Create(experiment *models.Experiment) error {
	return r.db.Create(experiment).Error
}

// GetByID retrieves an experiment by ID
func (r *ExperimentRepository) GetByID(id string) (*models.Experiment, error) {
	var experiment models.Experiment
	err := r.db.Preload("Model").
		Preload("TrainingJob").
		First(&experiment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &experiment, nil
}

// GetByModelID retrieves experiments for a model
func (r *ExperimentRepository) GetByModelID(modelID string) ([]*models.Experiment, error) {
	var experiments []*models.Experiment
	err := r.db.Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&experiments).Error
	return experiments, err
}

// GetRunningExperiments retrieves all currently running experiments
func (r *ExperimentRepository) GetRunningExperiments() ([]*models.Experiment, error) {
	var experiments []*models.Experiment
	err := r.db.Where("status = ?", models.ExperimentStatusRunning).
		Find(&experiments).Error
	return experiments, err
}

// Update updates an experiment
func (r *ExperimentRepository) Update(experiment *models.Experiment) error {
	return r.db.Save(experiment).Error
}

// ABTestRepository provides database operations for A/B tests
type ABTestRepository struct {
	db *Database
}

// NewABTestRepository creates a new A/B test repository
func NewABTestRepository(db *Database) *ABTestRepository {
	return &ABTestRepository{db: db}
}

// Create creates a new A/B test
func (r *ABTestRepository) Create(test *models.ABTest) error {
	return r.db.Create(test).Error
}

// GetByID retrieves an A/B test by ID
func (r *ABTestRepository) GetByID(id string) (*models.ABTest, error) {
	var test models.ABTest
	err := r.db.Preload("Model").
		Preload("Deployment").
		Preload("ChampionModel").
		Preload("ChallengerModel").
		Preload("WinnerModel").
		First(&test, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &test, nil
}

// GetRunningTests retrieves all currently running A/B tests
func (r *ABTestRepository) GetRunningTests() ([]*models.ABTest, error) {
	var tests []*models.ABTest
	err := r.db.Where("status = ?", models.ABTestStatusRunning).
		Preload("ChampionModel").
		Preload("ChallengerModel").
		Find(&tests).Error
	return tests, err
}

// Update updates an A/B test
func (r *ABTestRepository) Update(test *models.ABTest) error {
	return r.db.Save(test).Error
}

// ModelMetricRepository provides database operations for model metrics
type ModelMetricRepository struct {
	db *Database
}

// NewModelMetricRepository creates a new model metric repository
func NewModelMetricRepository(db *Database) *ModelMetricRepository {
	return &ModelMetricRepository{db: db}
}

// Create creates a new model metric
func (r *ModelMetricRepository) Create(metric *models.ModelMetric) error {
	return r.db.Create(metric).Error
}

// CreateBatch creates multiple model metrics in a batch
func (r *ModelMetricRepository) CreateBatch(metrics []*models.ModelMetric) error {
	return r.db.CreateInBatches(metrics, 1000).Error
}

// GetByModelID retrieves metrics for a model
func (r *ModelMetricRepository) GetByModelID(modelID string, limit int) ([]*models.ModelMetric, error) {
	var metrics []*models.ModelMetric
	err := r.db.Where("model_id = ?", modelID).
		Order("recorded_at DESC").
		Limit(limit).
		Find(&metrics).Error
	return metrics, err
}

// GetByModelIDAndMetric retrieves specific metrics for a model
func (r *ModelMetricRepository) GetByModelIDAndMetric(modelID, metricName string, since time.Time) ([]*models.ModelMetric, error) {
	var metrics []*models.ModelMetric
	err := r.db.Where("model_id = ? AND metric_name = ? AND recorded_at >= ?", modelID, metricName, since).
		Order("recorded_at ASC").
		Find(&metrics).Error
	return metrics, err
}

// GetLatestMetrics retrieves the latest metrics for all models
func (r *ModelMetricRepository) GetLatestMetrics() ([]*models.ModelMetric, error) {
	var metrics []*models.ModelMetric
	err := r.db.Raw(`
		SELECT DISTINCT ON (model_id, metric_name) *
		FROM model_metrics
		ORDER BY model_id, metric_name, recorded_at DESC
	`).Find(&metrics).Error
	return metrics, err
}

// FeatureRepository provides database operations for features
type FeatureRepository struct {
	db *Database
}

// NewFeatureRepository creates a new feature repository
func NewFeatureRepository(db *Database) *FeatureRepository {
	return &FeatureRepository{db: db}
}

// Create creates a new feature
func (r *FeatureRepository) Create(feature *models.Feature) error {
	return r.db.Create(feature).Error
}

// GetByID retrieves a feature by ID
func (r *FeatureRepository) GetByID(id string) (*models.Feature, error) {
	var feature models.Feature
	err := r.db.First(&feature, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// GetByName retrieves a feature by name
func (r *FeatureRepository) GetByName(name string) (*models.Feature, error) {
	var feature models.Feature
	err := r.db.Where("name = ?", name).First(&feature).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// GetActiveFeatures retrieves all active features
func (r *FeatureRepository) GetActiveFeatures() ([]*models.Feature, error) {
	var features []*models.Feature
	err := r.db.Where("is_active = ?", true).
		Order("name ASC").
		Find(&features).Error
	return features, err
}

// GetByCategory retrieves features by category
func (r *FeatureRepository) GetByCategory(category models.FeatureCategory) ([]*models.Feature, error) {
	var features []*models.Feature
	err := r.db.Where("category = ?", category).
		Order("name ASC").
		Find(&features).Error
	return features, err
}

// Update updates a feature
func (r *FeatureRepository) Update(feature *models.Feature) error {
	return r.db.Save(feature).Error
}

// Delete soft deletes a feature
func (r *FeatureRepository) Delete(id string) error {
	return r.db.Delete(&models.Feature{}, "id = ?", id).Error
}

// DataDriftRepository provides database operations for data drift detection
type DataDriftRepository struct {
	db *Database
}

// NewDataDriftRepository creates a new data drift repository
func NewDataDriftRepository(db *Database) *DataDriftRepository {
	return &DataDriftRepository{db: db}
}

// Create creates a new data drift record
func (r *DataDriftRepository) Create(drift *models.DataDrift) error {
	return r.db.Create(drift).Error
}

// CreateBatch creates multiple data drift records in a batch
func (r *DataDriftRepository) CreateBatch(drifts []*models.DataDrift) error {
	return r.db.CreateInBatches(drifts, 1000).Error
}

// GetByModelID retrieves data drift records for a model
func (r *DataDriftRepository) GetByModelID(modelID string, limit int) ([]*models.DataDrift, error) {
	var drifts []*models.DataDrift
	err := r.db.Where("model_id = ?", modelID).
		Order("detected_at DESC").
		Limit(limit).
		Find(&drifts).Error
	return drifts, err
}

// GetDriftAlerts retrieves drift records that exceed the threshold
func (r *DataDriftRepository) GetDriftAlerts(since time.Time) ([]*models.DataDrift, error) {
	var drifts []*models.DataDrift
	err := r.db.Where("is_drift = ? AND detected_at >= ?", true, since).
		Preload("Model").
		Order("detected_at DESC").
		Find(&drifts).Error
	return drifts, err
}

// PredictionRequestRepository provides database operations for prediction requests
type PredictionRequestRepository struct {
	db *Database
}

// NewPredictionRequestRepository creates a new prediction request repository
func NewPredictionRequestRepository(db *Database) *PredictionRequestRepository {
	return &PredictionRequestRepository{db: db}
}

// Create creates a new prediction request
func (r *PredictionRequestRepository) Create(request *models.PredictionRequest) error {
	return r.db.Create(request).Error
}

// GetByID retrieves a prediction request by ID
func (r *PredictionRequestRepository) GetByID(id string) (*models.PredictionRequest, error) {
	var request models.PredictionRequest
	err := r.db.Preload("Model").
		First(&request, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetByRequestID retrieves a prediction request by request ID
func (r *PredictionRequestRepository) GetByRequestID(requestID string) (*models.PredictionRequest, error) {
	var request models.PredictionRequest
	err := r.db.Where("request_id = ?", requestID).
		Preload("Model").
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetByModelID retrieves prediction requests for a model
func (r *PredictionRequestRepository) GetByModelID(modelID string, limit int) ([]*models.PredictionRequest, error) {
	var requests []*models.PredictionRequest
	err := r.db.Where("model_id = ?", modelID).
		Order("requested_at DESC").
		Limit(limit).
		Find(&requests).Error
	return requests, err
}

// GetRecentRequests retrieves recent prediction requests
func (r *PredictionRequestRepository) GetRecentRequests(since time.Time, limit int) ([]*models.PredictionRequest, error) {
	var requests []*models.PredictionRequest
	err := r.db.Where("requested_at >= ?", since).
		Order("requested_at DESC").
		Limit(limit).
		Find(&requests).Error
	return requests, err
}

// Update updates a prediction request
func (r *PredictionRequestRepository) Update(request *models.PredictionRequest) error {
	return r.db.Save(request).Error
}

// GetPerformanceStats retrieves performance statistics for a model
func (r *PredictionRequestRepository) GetPerformanceStats(modelID string, since time.Time) (map[string]interface{}, error) {
	var stats struct {
		TotalRequests   int64
		SuccessRequests int64
		FailedRequests  int64
		AvgProcessingTime float64
		AvgResponseTime   float64
	}

	err := r.db.Model(&models.PredictionRequest{}).
		Select(`
			COUNT(*) as total_requests,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as success_requests,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_requests,
			AVG(EXTRACT(EPOCH FROM processing_time)) as avg_processing_time,
			AVG(EXTRACT(EPOCH FROM response_time)) as avg_response_time
		`).
		Where("model_id = ? AND requested_at >= ?", modelID, since).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"total_requests":       stats.TotalRequests,
		"success_requests":     stats.SuccessRequests,
		"failed_requests":      stats.FailedRequests,
		"avg_processing_time":  stats.AvgProcessingTime,
		"avg_response_time":    stats.AvgResponseTime,
	}

	if stats.TotalRequests > 0 {
		result["success_rate"] = float64(stats.SuccessRequests) / float64(stats.TotalRequests)
		result["error_rate"] = float64(stats.FailedRequests) / float64(stats.TotalRequests)
	}

	return result, nil
}

// Repositories aggregates all repository instances
type Repositories struct {
	Model             *ModelRepository
	TrainingJob       *TrainingJobRepository
	Deployment        *DeploymentRepository
	Experiment        *ExperimentRepository
	ABTest            *ABTestRepository
	ModelMetric       *ModelMetricRepository
	Feature           *FeatureRepository
	DataDrift         *DataDriftRepository
	PredictionRequest *PredictionRequestRepository
}

// NewRepositories creates all repository instances
func NewRepositories(db *Database) *Repositories {
	return &Repositories{
		Model:             NewModelRepository(db),
		TrainingJob:       NewTrainingJobRepository(db),
		Deployment:        NewDeploymentRepository(db),
		Experiment:        NewExperimentRepository(db),
		ABTest:            NewABTestRepository(db),
		ModelMetric:       NewModelMetricRepository(db),
		Feature:           NewFeatureRepository(db),
		DataDrift:         NewDataDriftRepository(db),
		PredictionRequest: NewPredictionRequestRepository(db),
	}
}