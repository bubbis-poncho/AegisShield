package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelStatus represents the status of a model
type ModelStatus string

const (
	ModelStatusTraining   ModelStatus = "training"
	ModelStatusTrained    ModelStatus = "trained"
	ModelStatusDeployed   ModelStatus = "deployed"
	ModelStatusFailed     ModelStatus = "failed"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusTesting    ModelStatus = "testing"
	ModelStatusCandidate  ModelStatus = "candidate"
	ModelStatusChampion   ModelStatus = "champion"
)

// ModelType represents the type of machine learning model
type ModelType string

const (
	ModelTypeFraudDetection       ModelType = "fraud_detection"
	ModelTypeRiskScoring         ModelType = "risk_scoring"
	ModelTypeAnomalyDetection    ModelType = "anomaly_detection"
	ModelTypeTransactionScoring  ModelType = "transaction_scoring"
	ModelTypeEntityResolution    ModelType = "entity_resolution"
	ModelTypePatternRecognition  ModelType = "pattern_recognition"
	ModelTypeTimeSeriesForecasting ModelType = "time_series_forecasting"
	ModelTypeClassification      ModelType = "classification"
	ModelTypeRegression          ModelType = "regression"
	ModelTypeClustering          ModelType = "clustering"
)

// AlgorithmType represents the machine learning algorithm
type AlgorithmType string

const (
	AlgorithmXGBoost          AlgorithmType = "xgboost"
	AlgorithmRandomForest     AlgorithmType = "random_forest"
	AlgorithmLogisticRegression AlgorithmType = "logistic_regression"
	AlgorithmNeuralNetwork    AlgorithmType = "neural_network"
	AlgorithmIsolationForest  AlgorithmType = "isolation_forest"
	AlgorithmLSTM             AlgorithmType = "lstm"
	AlgorithmSVM              AlgorithmType = "svm"
	AlgorithmKMeans           AlgorithmType = "kmeans"
	AlgorithmDBSCAN           AlgorithmType = "dbscan"
	AlgorithmGradientBoosting AlgorithmType = "gradient_boosting"
)

// Model represents a machine learning model
type Model struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	Name            string          `gorm:"not null;index" json:"name"`
	Description     string          `json:"description"`
	Type            ModelType       `gorm:"not null;index" json:"type"`
	Algorithm       AlgorithmType   `gorm:"not null" json:"algorithm"`
	Version         string          `gorm:"not null;index" json:"version"`
	Status          ModelStatus     `gorm:"not null;index" json:"status"`
	
	// Training information
	TrainingDataset string          `json:"training_dataset"`
	TrainingJobID   *uuid.UUID      `gorm:"type:uuid;index" json:"training_job_id,omitempty"`
	TrainingStarted *time.Time      `json:"training_started,omitempty"`
	TrainingCompleted *time.Time    `json:"training_completed,omitempty"`
	TrainingDuration *time.Duration `json:"training_duration,omitempty"`
	
	// Model artifacts
	ModelPath       string          `json:"model_path"`
	ArtifactsPath   string          `json:"artifacts_path"`
	ConfigPath      string          `json:"config_path"`
	
	// Performance metrics
	Metrics         JSON            `gorm:"type:jsonb" json:"metrics"`
	ValidationMetrics JSON          `gorm:"type:jsonb" json:"validation_metrics"`
	TestMetrics     JSON            `gorm:"type:jsonb" json:"test_metrics"`
	
	// Hyperparameters and configuration
	Hyperparameters JSON            `gorm:"type:jsonb" json:"hyperparameters"`
	Configuration   JSON            `gorm:"type:jsonb" json:"configuration"`
	FeatureConfig   JSON            `gorm:"type:jsonb" json:"feature_config"`
	
	// Deployment information
	DeploymentID    *uuid.UUID      `gorm:"type:uuid;index" json:"deployment_id,omitempty"`
	EndpointURL     string          `json:"endpoint_url,omitempty"`
	IsActive        bool            `gorm:"default:false;index" json:"is_active"`
	TrafficWeight   float64         `gorm:"default:0" json:"traffic_weight"`
	
	// Monitoring and drift detection
	LastMonitored   *time.Time      `json:"last_monitored,omitempty"`
	DriftScore      *float64        `json:"drift_score,omitempty"`
	PerformanceScore *float64       `json:"performance_score,omitempty"`
	
	// Metadata
	Tags            JSON            `gorm:"type:jsonb" json:"tags"`
	Metadata        JSON            `gorm:"type:jsonb" json:"metadata"`
	
	// Audit fields
	CreatedBy       string          `gorm:"not null" json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`
	
	// Relationships
	TrainingJobs    []TrainingJob   `gorm:"foreignKey:ModelID" json:"training_jobs,omitempty"`
	Deployments     []Deployment    `gorm:"foreignKey:ModelID" json:"deployments,omitempty"`
	Experiments     []Experiment    `gorm:"foreignKey:ModelID" json:"experiments,omitempty"`
	ABTests         []ABTest        `gorm:"foreignKey:ModelID" json:"ab_tests,omitempty"`
	ModelMetrics    []ModelMetric   `gorm:"foreignKey:ModelID" json:"model_metrics,omitempty"`
}

// TrainingJob represents a model training job
type TrainingJob struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"model_id"`
	Name            string          `gorm:"not null" json:"name"`
	Status          TrainingStatus  `gorm:"not null;index" json:"status"`
	
	// Training configuration
	Algorithm       AlgorithmType   `gorm:"not null" json:"algorithm"`
	Hyperparameters JSON            `gorm:"type:jsonb" json:"hyperparameters"`
	Configuration   JSON            `gorm:"type:jsonb" json:"configuration"`
	
	// Data configuration
	TrainingDataset string          `gorm:"not null" json:"training_dataset"`
	ValidationDataset string        `json:"validation_dataset"`
	TestDataset     string          `json:"test_dataset"`
	FeatureConfig   JSON            `gorm:"type:jsonb" json:"feature_config"`
	
	// Execution details
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Duration        *time.Duration  `json:"duration,omitempty"`
	ResourceUsage   JSON            `gorm:"type:jsonb" json:"resource_usage"`
	
	// Results
	Metrics         JSON            `gorm:"type:jsonb" json:"metrics"`
	ValidationMetrics JSON          `gorm:"type:jsonb" json:"validation_metrics"`
	TestMetrics     JSON            `gorm:"type:jsonb" json:"test_metrics"`
	ModelPath       string          `json:"model_path"`
	ArtifactsPath   string          `json:"artifacts_path"`
	
	// Error handling
	ErrorMessage    string          `json:"error_message,omitempty"`
	ErrorDetails    JSON            `gorm:"type:jsonb" json:"error_details"`
	RetryCount      int             `gorm:"default:0" json:"retry_count"`
	
	// Audit fields
	CreatedBy       string          `gorm:"not null" json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	
	// Relationships
	Model           Model           `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	Experiments     []Experiment    `gorm:"foreignKey:TrainingJobID" json:"experiments,omitempty"`
}

// TrainingStatus represents the status of a training job
type TrainingStatus string

const (
	TrainingStatusPending    TrainingStatus = "pending"
	TrainingStatusRunning    TrainingStatus = "running"
	TrainingStatusCompleted  TrainingStatus = "completed"
	TrainingStatusFailed     TrainingStatus = "failed"
	TrainingStatusCancelled  TrainingStatus = "cancelled"
	TrainingStatusRetrying   TrainingStatus = "retrying"
)

// Deployment represents a model deployment
type Deployment struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"model_id"`
	Name            string          `gorm:"not null" json:"name"`
	Status          DeploymentStatus `gorm:"not null;index" json:"status"`
	
	// Deployment configuration
	Environment     string          `gorm:"not null;index" json:"environment"`
	Strategy        DeploymentStrategy `gorm:"not null" json:"strategy"`
	TrafficWeight   float64         `gorm:"default:0" json:"traffic_weight"`
	TargetWeight    float64         `gorm:"default:100" json:"target_weight"`
	
	// Endpoint information
	EndpointURL     string          `json:"endpoint_url"`
	EndpointType    EndpointType    `gorm:"not null" json:"endpoint_type"`
	
	// Scaling configuration
	MinInstances    int             `gorm:"default:1" json:"min_instances"`
	MaxInstances    int             `gorm:"default:10" json:"max_instances"`
	CurrentInstances int            `gorm:"default:1" json:"current_instances"`
	
	// Resource configuration
	CPURequest      string          `json:"cpu_request"`
	MemoryRequest   string          `json:"memory_request"`
	CPULimit        string          `json:"cpu_limit"`
	MemoryLimit     string          `json:"memory_limit"`
	GPURequest      int             `gorm:"default:0" json:"gpu_request"`
	
	// Deployment timing
	DeployedAt      *time.Time      `json:"deployed_at,omitempty"`
	UndeployedAt    *time.Time      `json:"undeployed_at,omitempty"`
	LastHealthCheck *time.Time      `json:"last_health_check,omitempty"`
	
	// Configuration
	Configuration   JSON            `gorm:"type:jsonb" json:"configuration"`
	EnvironmentVars JSON            `gorm:"type:jsonb" json:"environment_vars"`
	
	// Monitoring
	HealthStatus    HealthStatus    `gorm:"default:'unknown'" json:"health_status"`
	LastError       string          `json:"last_error,omitempty"`
	
	// Audit fields
	CreatedBy       string          `gorm:"not null" json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	
	// Relationships
	Model           Model           `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	ABTests         []ABTest        `gorm:"foreignKey:DeploymentID" json:"ab_tests,omitempty"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending     DeploymentStatus = "pending"
	DeploymentStatusDeploying   DeploymentStatus = "deploying"
	DeploymentStatusActive      DeploymentStatus = "active"
	DeploymentStatusFailed      DeploymentStatus = "failed"
	DeploymentStatusUndeploying DeploymentStatus = "undeploying"
	DeploymentStatusInactive    DeploymentStatus = "inactive"
	DeploymentStatusRollingBack DeploymentStatus = "rolling_back"
)

// DeploymentStrategy represents the deployment strategy
type DeploymentStrategy string

const (
	DeploymentStrategyBlueGreen DeploymentStrategy = "blue_green"
	DeploymentStrategyCanary    DeploymentStrategy = "canary"
	DeploymentStrategyRolling   DeploymentStrategy = "rolling"
	DeploymentStrategyInstant   DeploymentStrategy = "instant"
)

// EndpointType represents the type of model endpoint
type EndpointType string

const (
	EndpointTypeREST    EndpointType = "rest"
	EndpointTypeGRPC    EndpointType = "grpc"
	EndpointTypeBatch   EndpointType = "batch"
	EndpointTypeStream  EndpointType = "stream"
)

// HealthStatus represents the health status of a deployment
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// Experiment represents a machine learning experiment
type Experiment struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID         *uuid.UUID      `gorm:"type:uuid;index" json:"model_id,omitempty"`
	TrainingJobID   *uuid.UUID      `gorm:"type:uuid;index" json:"training_job_id,omitempty"`
	Name            string          `gorm:"not null" json:"name"`
	Description     string          `json:"description"`
	Status          ExperimentStatus `gorm:"not null;index" json:"status"`
	
	// Experiment configuration
	Hypothesis      string          `json:"hypothesis"`
	Objective       string          `json:"objective"`
	Metrics         []string        `gorm:"type:text[]" json:"metrics"`
	
	// Parameters and results
	Parameters      JSON            `gorm:"type:jsonb" json:"parameters"`
	Results         JSON            `gorm:"type:jsonb" json:"results"`
	Artifacts       JSON            `gorm:"type:jsonb" json:"artifacts"`
	
	// Timing
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Duration        *time.Duration  `json:"duration,omitempty"`
	
	// Audit fields
	CreatedBy       string          `gorm:"not null" json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	
	// Relationships
	Model           *Model          `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	TrainingJob     *TrainingJob    `gorm:"foreignKey:TrainingJobID" json:"training_job,omitempty"`
}

// ExperimentStatus represents the status of an experiment
type ExperimentStatus string

const (
	ExperimentStatusPlanned   ExperimentStatus = "planned"
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusFailed    ExperimentStatus = "failed"
	ExperimentStatusCancelled ExperimentStatus = "cancelled"
)

// ABTest represents an A/B test for model comparison
type ABTest struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID         *uuid.UUID      `gorm:"type:uuid;index" json:"model_id,omitempty"`
	DeploymentID    *uuid.UUID      `gorm:"type:uuid;index" json:"deployment_id,omitempty"`
	Name            string          `gorm:"not null" json:"name"`
	Description     string          `json:"description"`
	Status          ABTestStatus    `gorm:"not null;index" json:"status"`
	
	// Test configuration
	ChampionModelID uuid.UUID       `gorm:"type:uuid;not null" json:"champion_model_id"`
	ChallengerModelID uuid.UUID     `gorm:"type:uuid;not null" json:"challenger_model_id"`
	TrafficSplit    float64         `gorm:"not null" json:"traffic_split"`
	
	// Test criteria
	SuccessMetric   string          `gorm:"not null" json:"success_metric"`
	MinimumSampleSize int           `gorm:"not null" json:"minimum_sample_size"`
	SignificanceLevel float64       `gorm:"not null" json:"significance_level"`
	MinimumEffect   float64         `json:"minimum_effect"`
	
	// Test duration
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	PlannedDuration time.Duration   `json:"planned_duration"`
	ActualDuration  *time.Duration  `json:"actual_duration,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	
	// Results
	ChampionMetrics JSON            `gorm:"type:jsonb" json:"champion_metrics"`
	ChallengerMetrics JSON          `gorm:"type:jsonb" json:"challenger_metrics"`
	StatisticalSignificance *bool   `json:"statistical_significance,omitempty"`
	WinnerModelID   *uuid.UUID      `gorm:"type:uuid" json:"winner_model_id,omitempty"`
	ConfidenceLevel *float64        `json:"confidence_level,omitempty"`
	
	// Auto-promotion settings
	AutoPromote     bool            `gorm:"default:false" json:"auto_promote"`
	PromotionThreshold float64      `json:"promotion_threshold"`
	PromotedAt      *time.Time      `json:"promoted_at,omitempty"`
	
	// Audit fields
	CreatedBy       string          `gorm:"not null" json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	
	// Relationships
	Model           *Model          `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	Deployment      *Deployment     `gorm:"foreignKey:DeploymentID" json:"deployment,omitempty"`
	ChampionModel   Model           `gorm:"foreignKey:ChampionModelID" json:"champion_model,omitempty"`
	ChallengerModel Model           `gorm:"foreignKey:ChallengerModelID" json:"challenger_model,omitempty"`
	WinnerModel     *Model          `gorm:"foreignKey:WinnerModelID" json:"winner_model,omitempty"`
}

// ABTestStatus represents the status of an A/B test
type ABTestStatus string

const (
	ABTestStatusPlanned   ABTestStatus = "planned"
	ABTestStatusRunning   ABTestStatus = "running"
	ABTestStatusCompleted ABTestStatus = "completed"
	ABTestStatusFailed    ABTestStatus = "failed"
	ABTestStatusCancelled ABTestStatus = "cancelled"
	ABTestStatusPromoted  ABTestStatus = "promoted"
)

// ModelMetric represents model performance metrics over time
type ModelMetric struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID     uuid.UUID       `gorm:"type:uuid;not null;index" json:"model_id"`
	MetricName  string          `gorm:"not null;index" json:"metric_name"`
	MetricValue float64         `gorm:"not null" json:"metric_value"`
	MetricType  MetricType      `gorm:"not null" json:"metric_type"`
	
	// Context
	Environment string          `gorm:"index" json:"environment"`
	DataWindow  time.Duration   `json:"data_window"`
	SampleSize  int             `json:"sample_size"`
	
	// Metadata
	Metadata    JSON            `gorm:"type:jsonb" json:"metadata"`
	Tags        JSON            `gorm:"type:jsonb" json:"tags"`
	
	// Timestamp
	RecordedAt  time.Time       `gorm:"not null;index" json:"recorded_at"`
	CreatedAt   time.Time       `json:"created_at"`
	
	// Relationships
	Model       Model           `gorm:"foreignKey:ModelID" json:"model,omitempty"`
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeAccuracy    MetricType = "accuracy"
	MetricTypePrecision   MetricType = "precision"
	MetricTypeRecall      MetricType = "recall"
	MetricTypeF1Score     MetricType = "f1_score"
	MetricTypeAUC         MetricType = "auc"
	MetricTypeRMSE        MetricType = "rmse"
	MetricTypeMAE         MetricType = "mae"
	MetricTypeLatency     MetricType = "latency"
	MetricTypeThroughput  MetricType = "throughput"
	MetricTypeErrorRate   MetricType = "error_rate"
	MetricTypeDriftScore  MetricType = "drift_score"
	MetricTypeCustom      MetricType = "custom"
)

// Feature represents a feature used in models
type Feature struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	Name            string          `gorm:"not null;unique;index" json:"name"`
	DisplayName     string          `gorm:"not null" json:"display_name"`
	Description     string          `json:"description"`
	DataType        FeatureDataType `gorm:"not null" json:"data_type"`
	Category        FeatureCategory `gorm:"not null;index" json:"category"`
	
	// Feature configuration
	IsActive        bool            `gorm:"default:true;index" json:"is_active"`
	Importance      *float64        `json:"importance,omitempty"`
	ComputeLogic    string          `json:"compute_logic"`
	Dependencies    []string        `gorm:"type:text[]" json:"dependencies"`
	
	// Data validation
	ValidationRules JSON            `gorm:"type:jsonb" json:"validation_rules"`
	DefaultValue    interface{}     `gorm:"type:jsonb" json:"default_value"`
	MinValue        *float64        `json:"min_value,omitempty"`
	MaxValue        *float64        `json:"max_value,omitempty"`
	AllowedValues   []string        `gorm:"type:text[]" json:"allowed_values"`
	
	// Monitoring
	LastComputed    *time.Time      `json:"last_computed,omitempty"`
	ComputeErrors   int             `gorm:"default:0" json:"compute_errors"`
	DriftScore      *float64        `json:"drift_score,omitempty"`
	
	// Metadata
	Tags            JSON            `gorm:"type:jsonb" json:"tags"`
	Metadata        JSON            `gorm:"type:jsonb" json:"metadata"`
	
	// Audit fields
	CreatedBy       string          `gorm:"not null" json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`
}

// FeatureDataType represents the data type of a feature
type FeatureDataType string

const (
	FeatureDataTypeNumeric     FeatureDataType = "numeric"
	FeatureDataTypeCategorical FeatureDataType = "categorical"
	FeatureDataTypeBoolean     FeatureDataType = "boolean"
	FeatureDataTypeText        FeatureDataType = "text"
	FeatureDataTypeTimestamp   FeatureDataType = "timestamp"
	FeatureDataTypeJSON        FeatureDataType = "json"
	FeatureDataTypeArray       FeatureDataType = "array"
)

// FeatureCategory represents the category of a feature
type FeatureCategory string

const (
	FeatureCategoryTransaction  FeatureCategory = "transaction"
	FeatureCategoryAccount      FeatureCategory = "account"
	FeatureCategoryEntity       FeatureCategory = "entity"
	FeatureCategoryTemporal     FeatureCategory = "temporal"
	FeatureCategoryNetwork      FeatureCategory = "network"
	FeatureCategoryBehavioral   FeatureCategory = "behavioral"
	FeatureCategoryDemographic  FeatureCategory = "demographic"
	FeatureCategoryAggregate    FeatureCategory = "aggregate"
	FeatureCategoryDerived      FeatureCategory = "derived"
)

// DataDrift represents data drift detection results
type DataDrift struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"model_id"`
	FeatureName     string          `gorm:"not null;index" json:"feature_name"`
	DriftMethod     string          `gorm:"not null" json:"drift_method"`
	DriftScore      float64         `gorm:"not null" json:"drift_score"`
	Threshold       float64         `gorm:"not null" json:"threshold"`
	IsDrift         bool            `gorm:"not null;index" json:"is_drift"`
	
	// Drift details
	ReferenceStart  time.Time       `gorm:"not null" json:"reference_start"`
	ReferenceEnd    time.Time       `gorm:"not null" json:"reference_end"`
	CurrentStart    time.Time       `gorm:"not null" json:"current_start"`
	CurrentEnd      time.Time       `gorm:"not null" json:"current_end"`
	
	// Statistics
	ReferenceStats  JSON            `gorm:"type:jsonb" json:"reference_stats"`
	CurrentStats    JSON            `gorm:"type:jsonb" json:"current_stats"`
	DriftDetails    JSON            `gorm:"type:jsonb" json:"drift_details"`
	
	// Detection metadata
	DetectedAt      time.Time       `gorm:"not null;index" json:"detected_at"`
	SampleSize      int             `json:"sample_size"`
	
	// Audit fields
	CreatedAt       time.Time       `json:"created_at"`
	
	// Relationships
	Model           Model           `gorm:"foreignKey:ModelID" json:"model,omitempty"`
}

// PredictionRequest represents a prediction request
type PredictionRequest struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	ModelID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"model_id"`
	RequestID       string          `gorm:"not null;unique;index" json:"request_id"`
	
	// Request data
	Features        JSON            `gorm:"type:jsonb;not null" json:"features"`
	RequestMetadata JSON            `gorm:"type:jsonb" json:"request_metadata"`
	
	// Prediction result
	Prediction      JSON            `gorm:"type:jsonb" json:"prediction"`
	Confidence      *float64        `json:"confidence,omitempty"`
	Probability     *float64        `json:"probability,omitempty"`
	
	// Performance metrics
	ProcessingTime  time.Duration   `json:"processing_time"`
	ResponseTime    time.Duration   `json:"response_time"`
	
	// Context
	Environment     string          `gorm:"index" json:"environment"`
	ClientID        string          `gorm:"index" json:"client_id"`
	UserID          string          `gorm:"index" json:"user_id"`
	
	// Status
	Status          RequestStatus   `gorm:"not null;index" json:"status"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	
	// Feedback
	GroundTruth     JSON            `gorm:"type:jsonb" json:"ground_truth"`
	FeedbackScore   *float64        `json:"feedback_score,omitempty"`
	FeedbackAt      *time.Time      `json:"feedback_at,omitempty"`
	
	// Timing
	RequestedAt     time.Time       `gorm:"not null;index" json:"requested_at"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	
	// Relationships
	Model           Model           `gorm:"foreignKey:ModelID" json:"model,omitempty"`
}

// RequestStatus represents the status of a prediction request
type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusProcessing RequestStatus = "processing"
	RequestStatusCompleted RequestStatus = "completed"
	RequestStatusFailed    RequestStatus = "failed"
	RequestStatusTimeout   RequestStatus = "timeout"
)

// JSON represents a JSON field for GORM
type JSON json.RawMessage

// Scan implements the Scanner interface for GORM
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	*j = JSON(bytes)
	return nil
}

// Value implements the driver.Valuer interface for GORM
func (j JSON) Value() (interface{}, error) {
	if j == nil {
		return nil, nil
	}
	return string(j), nil
}

// MarshalJSON implements the json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return nil
	}
	*j = JSON(data)
	return nil
}

// BeforeCreate hook for models
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (tj *TrainingJob) BeforeCreate(tx *gorm.DB) error {
	if tj.ID == uuid.Nil {
		tj.ID = uuid.New()
	}
	return nil
}

func (d *Deployment) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

func (e *Experiment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (a *ABTest) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (mm *ModelMetric) BeforeCreate(tx *gorm.DB) error {
	if mm.ID == uuid.Nil {
		mm.ID = uuid.New()
	}
	return nil
}

func (f *Feature) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

func (dd *DataDrift) BeforeCreate(tx *gorm.DB) error {
	if dd.ID == uuid.Nil {
		dd.ID = uuid.New()
	}
	return nil
}

func (pr *PredictionRequest) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}