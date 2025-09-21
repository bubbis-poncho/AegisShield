package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the ML pipeline configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	ML         MLConfig         `mapstructure:"ml"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Security   SecurityConfig   `mapstructure:"security"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Features   FeaturesConfig   `mapstructure:"features"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	GRPCPort     int           `mapstructure:"grpc_port"`
	EnableCORS   bool          `mapstructure:"enable_cors"`
	EnableTLS    bool          `mapstructure:"enable_tls"`
	TLSCertFile  string        `mapstructure:"tls_cert_file"`
	TLSKeyFile   string        `mapstructure:"tls_key_file"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	Database     int           `mapstructure:"database"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers          []string      `mapstructure:"brokers"`
	SecurityProtocol string        `mapstructure:"security_protocol"`
	SASLUsername     string        `mapstructure:"sasl_username"`
	SASLPassword     string        `mapstructure:"sasl_password"`
	SASLMechanism    string        `mapstructure:"sasl_mechanism"`
	ConsumerGroup    string        `mapstructure:"consumer_group"`
	AutoOffsetReset  string        `mapstructure:"auto_offset_reset"`
	SessionTimeout   time.Duration `mapstructure:"session_timeout"`
	EnableAutoCommit bool          `mapstructure:"enable_auto_commit"`
	BatchSize        int           `mapstructure:"batch_size"`
	BatchTimeout     time.Duration `mapstructure:"batch_timeout"`
	RetryBackoff     time.Duration `mapstructure:"retry_backoff"`
	MaxRetries       int           `mapstructure:"max_retries"`
	
	// Topic configurations
	Topics TopicsConfig `mapstructure:"topics"`
}

// TopicsConfig holds Kafka topic configurations
type TopicsConfig struct {
	TrainingData     string `mapstructure:"training_data"`
	ModelUpdates     string `mapstructure:"model_updates"`
	Predictions      string `mapstructure:"predictions"`
	FeatureUpdates   string `mapstructure:"feature_updates"`
	ModelMetrics     string `mapstructure:"model_metrics"`
	ABTestResults    string `mapstructure:"ab_test_results"`
	ModelDeployment  string `mapstructure:"model_deployment"`
	DataDrift        string `mapstructure:"data_drift"`
}

// MLConfig holds machine learning configuration
type MLConfig struct {
	ModelStore        ModelStoreConfig   `mapstructure:"model_store"`
	Training          TrainingConfig     `mapstructure:"training"`
	Inference         InferenceConfig    `mapstructure:"inference"`
	FeatureStore      FeatureStoreConfig `mapstructure:"feature_store"`
	ABTesting         ABTestingConfig    `mapstructure:"ab_testing"`
	ModelMonitoring   ModelMonitoringConfig `mapstructure:"model_monitoring"`
	AutoRetraining    AutoRetrainingConfig `mapstructure:"auto_retraining"`
	DataValidation    DataValidationConfig `mapstructure:"data_validation"`
}

// ModelStoreConfig holds model storage configuration
type ModelStoreConfig struct {
	Type            string `mapstructure:"type"` // filesystem, s3, gcs, mlflow
	BasePath        string `mapstructure:"base_path"`
	S3Bucket        string `mapstructure:"s3_bucket"`
	S3Region        string `mapstructure:"s3_region"`
	S3AccessKey     string `mapstructure:"s3_access_key"`
	S3SecretKey     string `mapstructure:"s3_secret_key"`
	GCSBucket       string `mapstructure:"gcs_bucket"`
	GCSCredentials  string `mapstructure:"gcs_credentials"`
	MLFlowURL       string `mapstructure:"mlflow_url"`
	VersionRetention int   `mapstructure:"version_retention"`
	EnableEncryption bool  `mapstructure:"enable_encryption"`
}

// TrainingConfig holds model training configuration
type TrainingConfig struct {
	DefaultAlgorithm    string        `mapstructure:"default_algorithm"`
	MaxTrainingTime     time.Duration `mapstructure:"max_training_time"`
	ValidationSplit     float64       `mapstructure:"validation_split"`
	CrossValidationFolds int          `mapstructure:"cross_validation_folds"`
	EarlyStoppingPatience int         `mapstructure:"early_stopping_patience"`
	MaxConcurrentJobs   int           `mapstructure:"max_concurrent_jobs"`
	ResourceLimits      ResourceLimitsConfig `mapstructure:"resource_limits"`
	HyperparameterTuning HyperparameterConfig `mapstructure:"hyperparameter_tuning"`
	
	// Algorithm-specific configurations
	Algorithms AlgorithmConfigs `mapstructure:"algorithms"`
}

// InferenceConfig holds model inference configuration
type InferenceConfig struct {
	BatchSize           int           `mapstructure:"batch_size"`
	MaxLatency          time.Duration `mapstructure:"max_latency"`
	CacheEnabled        bool          `mapstructure:"cache_enabled"`
	CacheTTL            time.Duration `mapstructure:"cache_ttl"`
	LoadBalancing       string        `mapstructure:"load_balancing"` // round_robin, least_connections, weighted
	CircuitBreaker      CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	RateLimiting        RateLimitingConfig `mapstructure:"rate_limiting"`
	ModelWarmup         bool          `mapstructure:"model_warmup"`
	PredictionThreshold float64       `mapstructure:"prediction_threshold"`
}

// FeatureStoreConfig holds feature store configuration
type FeatureStoreConfig struct {
	Type              string        `mapstructure:"type"` // redis, postgres, feast
	RefreshInterval   time.Duration `mapstructure:"refresh_interval"`
	FeatureTTL        time.Duration `mapstructure:"feature_ttl"`
	MaxFeatures       int           `mapstructure:"max_features"`
	EnableVersioning  bool          `mapstructure:"enable_versioning"`
	ComputeEngine     ComputeEngineConfig `mapstructure:"compute_engine"`
	FeatureValidation FeatureValidationConfig `mapstructure:"feature_validation"`
}

// ABTestingConfig holds A/B testing configuration
type ABTestingConfig struct {
	EnableABTesting     bool          `mapstructure:"enable_ab_testing"`
	DefaultTrafficSplit float64       `mapstructure:"default_traffic_split"`
	MinimumSampleSize   int           `mapstructure:"minimum_sample_size"`
	SignificanceLevel   float64       `mapstructure:"significance_level"`
	TestDuration        time.Duration `mapstructure:"test_duration"`
	AutoPromote         bool          `mapstructure:"auto_promote"`
	PromotionThreshold  float64       `mapstructure:"promotion_threshold"`
	MetricsCollection   MetricsCollectionConfig `mapstructure:"metrics_collection"`
}

// ModelMonitoringConfig holds model monitoring configuration
type ModelMonitoringConfig struct {
	EnableMonitoring    bool          `mapstructure:"enable_monitoring"`
	MetricsInterval     time.Duration `mapstructure:"metrics_interval"`
	DriftDetection      DriftDetectionConfig `mapstructure:"drift_detection"`
	PerformanceMonitoring PerformanceMonitoringConfig `mapstructure:"performance_monitoring"`
	AlertThresholds     AlertThresholdsConfig `mapstructure:"alert_thresholds"`
	DataQualityChecks   DataQualityConfig `mapstructure:"data_quality_checks"`
}

// AutoRetrainingConfig holds automatic retraining configuration
type AutoRetrainingConfig struct {
	EnableAutoRetraining bool          `mapstructure:"enable_auto_retraining"`
	RetrainingSchedule   string        `mapstructure:"retraining_schedule"` // cron expression
	TriggerConditions    TriggerConditionsConfig `mapstructure:"trigger_conditions"`
	DataWindow           time.Duration `mapstructure:"data_window"`
	MinDataThreshold     int           `mapstructure:"min_data_threshold"`
	RetrainingCooldown   time.Duration `mapstructure:"retraining_cooldown"`
	AutoDeployment       AutoDeploymentConfig `mapstructure:"auto_deployment"`
}

// DataValidationConfig holds data validation configuration
type DataValidationConfig struct {
	EnableValidation    bool                    `mapstructure:"enable_validation"`
	SchemaValidation    SchemaValidationConfig  `mapstructure:"schema_validation"`
	QualityChecks       QualityChecksConfig     `mapstructure:"quality_checks"`
	AnomalyDetection    AnomalyDetectionConfig  `mapstructure:"anomaly_detection"`
	DataProfiling       DataProfilingConfig     `mapstructure:"data_profiling"`
}

// Supporting configuration structs
type ResourceLimitsConfig struct {
	CPULimit    string `mapstructure:"cpu_limit"`
	MemoryLimit string `mapstructure:"memory_limit"`
	GPULimit    int    `mapstructure:"gpu_limit"`
}

type HyperparameterConfig struct {
	EnableTuning   bool   `mapstructure:"enable_tuning"`
	TuningMethod   string `mapstructure:"tuning_method"` // grid, random, bayesian
	MaxTrials      int    `mapstructure:"max_trials"`
	ParallelTrials int    `mapstructure:"parallel_trials"`
}

type AlgorithmConfigs struct {
	XGBoost       map[string]interface{} `mapstructure:"xgboost"`
	RandomForest  map[string]interface{} `mapstructure:"random_forest"`
	LogisticRegression map[string]interface{} `mapstructure:"logistic_regression"`
	NeuralNetwork map[string]interface{} `mapstructure:"neural_network"`
	IsolationForest map[string]interface{} `mapstructure:"isolation_forest"`
	LSTM          map[string]interface{} `mapstructure:"lstm"`
}

type CircuitBreakerConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	FailureThreshold  int           `mapstructure:"failure_threshold"`
	RecoveryTimeout   time.Duration `mapstructure:"recovery_timeout"`
	SuccessThreshold  int           `mapstructure:"success_threshold"`
}

type RateLimitingConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	RequestsPerSecond int        `mapstructure:"requests_per_second"`
	BurstSize      int           `mapstructure:"burst_size"`
	WindowDuration time.Duration `mapstructure:"window_duration"`
}

type ComputeEngineConfig struct {
	Type        string            `mapstructure:"type"` // local, spark, ray
	SparkConfig map[string]string `mapstructure:"spark_config"`
	RayConfig   map[string]string `mapstructure:"ray_config"`
}

type FeatureValidationConfig struct {
	EnableValidation bool                   `mapstructure:"enable_validation"`
	Rules           []FeatureValidationRule `mapstructure:"rules"`
}

type FeatureValidationRule struct {
	FeatureName string      `mapstructure:"feature_name"`
	DataType    string      `mapstructure:"data_type"`
	MinValue    *float64    `mapstructure:"min_value"`
	MaxValue    *float64    `mapstructure:"max_value"`
	AllowNull   bool        `mapstructure:"allow_null"`
	Enum        []string    `mapstructure:"enum"`
}

type MetricsCollectionConfig struct {
	BusinessMetrics []string `mapstructure:"business_metrics"`
	TechnicalMetrics []string `mapstructure:"technical_metrics"`
}

type DriftDetectionConfig struct {
	EnableDriftDetection bool          `mapstructure:"enable_drift_detection"`
	DriftThreshold       float64       `mapstructure:"drift_threshold"`
	DriftMethod          string        `mapstructure:"drift_method"` // ks, psi, jensen_shannon
	WindowSize           int           `mapstructure:"window_size"`
	CheckInterval        time.Duration `mapstructure:"check_interval"`
}

type PerformanceMonitoringConfig struct {
	AccuracyThreshold float64 `mapstructure:"accuracy_threshold"`
	LatencyThreshold  time.Duration `mapstructure:"latency_threshold"`
	ThroughputThreshold int `mapstructure:"throughput_threshold"`
}

type AlertThresholdsConfig struct {
	AccuracyDrop    float64 `mapstructure:"accuracy_drop"`
	LatencyIncrease float64 `mapstructure:"latency_increase"`
	ErrorRateLimit  float64 `mapstructure:"error_rate_limit"`
	DataDriftLimit  float64 `mapstructure:"data_drift_limit"`
}

type DataQualityConfig struct {
	CompletenessThreshold float64 `mapstructure:"completeness_threshold"`
	ValidityThreshold     float64 `mapstructure:"validity_threshold"`
	UniquenessThreshold   float64 `mapstructure:"uniqueness_threshold"`
}

type TriggerConditionsConfig struct {
	AccuracyDropThreshold float64 `mapstructure:"accuracy_drop_threshold"`
	DataDriftThreshold    float64 `mapstructure:"data_drift_threshold"`
	ErrorRateThreshold    float64 `mapstructure:"error_rate_threshold"`
	DataVolumeThreshold   int     `mapstructure:"data_volume_threshold"`
}

type AutoDeploymentConfig struct {
	EnableAutoDeployment bool    `mapstructure:"enable_auto_deployment"`
	ValidationThreshold  float64 `mapstructure:"validation_threshold"`
	RolloutStrategy      string  `mapstructure:"rollout_strategy"` // blue_green, canary, rolling
	CanaryPercentage     float64 `mapstructure:"canary_percentage"`
}

type SchemaValidationConfig struct {
	EnableSchemaValidation bool   `mapstructure:"enable_schema_validation"`
	SchemaRegistry        string `mapstructure:"schema_registry"`
	StrictMode            bool   `mapstructure:"strict_mode"`
}

type QualityChecksConfig struct {
	EnableQualityChecks bool                 `mapstructure:"enable_quality_checks"`
	Checks             []DataQualityCheck   `mapstructure:"checks"`
}

type DataQualityCheck struct {
	Name        string                 `mapstructure:"name"`
	Type        string                 `mapstructure:"type"` // completeness, uniqueness, validity, consistency
	Threshold   float64                `mapstructure:"threshold"`
	Parameters  map[string]interface{} `mapstructure:"parameters"`
}

type AnomalyDetectionConfig struct {
	EnableAnomalyDetection bool    `mapstructure:"enable_anomaly_detection"`
	Method                string  `mapstructure:"method"` // isolation_forest, one_class_svm, lof
	Threshold             float64 `mapstructure:"threshold"`
	WindowSize            int     `mapstructure:"window_size"`
}

type DataProfilingConfig struct {
	EnableProfiling    bool          `mapstructure:"enable_profiling"`
	ProfilingInterval  time.Duration `mapstructure:"profiling_interval"`
	StoreProfiles      bool          `mapstructure:"store_profiles"`
	ProfileRetention   time.Duration `mapstructure:"profile_retention"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MetricsPath     string        `mapstructure:"metrics_path"`
	MetricsInterval time.Duration `mapstructure:"metrics_interval"`
	Jaeger          JaegerConfig  `mapstructure:"jaeger"`
}

// JaegerConfig holds Jaeger tracing configuration
type JaegerConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	ServiceName string `mapstructure:"service_name"`
	AgentHost   string `mapstructure:"agent_host"`
	AgentPort   int    `mapstructure:"agent_port"`
	SamplerType string `mapstructure:"sampler_type"`
	SamplerParam float64 `mapstructure:"sampler_param"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	EnableAuth       bool   `mapstructure:"enable_auth"`
	JWTSecret        string `mapstructure:"jwt_secret"`
	JWTExpiration    time.Duration `mapstructure:"jwt_expiration"`
	EnableRBAC       bool   `mapstructure:"enable_rbac"`
	EnableEncryption bool   `mapstructure:"enable_encryption"`
	EncryptionKey    string `mapstructure:"encryption_key"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type              string `mapstructure:"type"` // local, s3, gcs
	LocalPath         string `mapstructure:"local_path"`
	S3Bucket          string `mapstructure:"s3_bucket"`
	S3Region          string `mapstructure:"s3_region"`
	S3AccessKey       string `mapstructure:"s3_access_key"`
	S3SecretKey       string `mapstructure:"s3_secret_key"`
	GCSBucket         string `mapstructure:"gcs_bucket"`
	GCSCredentialsPath string `mapstructure:"gcs_credentials_path"`
}

// FeaturesConfig holds feature flags configuration
type FeaturesConfig struct {
	EnableAsyncTraining     bool `mapstructure:"enable_async_training"`
	EnableModelVersioning   bool `mapstructure:"enable_model_versioning"`
	EnableExperimentTracking bool `mapstructure:"enable_experiment_tracking"`
	EnableAutoScaling       bool `mapstructure:"enable_auto_scaling"`
	EnableGPUAcceleration   bool `mapstructure:"enable_gpu_acceleration"`
	EnableDistributedTraining bool `mapstructure:"enable_distributed_training"`
	EnableOnlineFeatures    bool `mapstructure:"enable_online_features"`
	EnableBatchFeatures     bool `mapstructure:"enable_batch_features"`
}

// LoadConfig loads configuration from environment and config files
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Set default values
	setDefaults()

	// Configure viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/ml-pipeline")

	// Enable reading environment variables
	viper.AutomaticEnv()

	// Read configuration file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.grpc_port", 9090)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.enable_cors", true)
	viper.SetDefault("server.enable_tls", false)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "ml_pipeline")
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 100)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.conn_max_lifetime", "1h")
	viper.SetDefault("database.conn_max_idle_time", "30m")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "5s")
	viper.SetDefault("redis.write_timeout", "5s")

	// Kafka defaults
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.consumer_group", "ml-pipeline")
	viper.SetDefault("kafka.auto_offset_reset", "earliest")
	viper.SetDefault("kafka.session_timeout", "30s")
	viper.SetDefault("kafka.enable_auto_commit", true)
	viper.SetDefault("kafka.batch_size", 100)
	viper.SetDefault("kafka.batch_timeout", "5s")
	viper.SetDefault("kafka.retry_backoff", "1s")
	viper.SetDefault("kafka.max_retries", 3)

	// Kafka topics
	viper.SetDefault("kafka.topics.training_data", "ml.training.data")
	viper.SetDefault("kafka.topics.model_updates", "ml.model.updates")
	viper.SetDefault("kafka.topics.predictions", "ml.predictions")
	viper.SetDefault("kafka.topics.feature_updates", "ml.features.updates")
	viper.SetDefault("kafka.topics.model_metrics", "ml.model.metrics")
	viper.SetDefault("kafka.topics.ab_test_results", "ml.ab.test.results")
	viper.SetDefault("kafka.topics.model_deployment", "ml.model.deployment")
	viper.SetDefault("kafka.topics.data_drift", "ml.data.drift")

	// ML configuration defaults
	viper.SetDefault("ml.model_store.type", "filesystem")
	viper.SetDefault("ml.model_store.base_path", "./models")
	viper.SetDefault("ml.model_store.version_retention", 10)
	viper.SetDefault("ml.model_store.enable_encryption", false)

	viper.SetDefault("ml.training.default_algorithm", "xgboost")
	viper.SetDefault("ml.training.max_training_time", "4h")
	viper.SetDefault("ml.training.validation_split", 0.2)
	viper.SetDefault("ml.training.cross_validation_folds", 5)
	viper.SetDefault("ml.training.early_stopping_patience", 10)
	viper.SetDefault("ml.training.max_concurrent_jobs", 3)

	viper.SetDefault("ml.inference.batch_size", 1000)
	viper.SetDefault("ml.inference.max_latency", "100ms")
	viper.SetDefault("ml.inference.cache_enabled", true)
	viper.SetDefault("ml.inference.cache_ttl", "1h")
	viper.SetDefault("ml.inference.load_balancing", "round_robin")
	viper.SetDefault("ml.inference.model_warmup", true)
	viper.SetDefault("ml.inference.prediction_threshold", 0.5)

	viper.SetDefault("ml.feature_store.type", "redis")
	viper.SetDefault("ml.feature_store.refresh_interval", "5m")
	viper.SetDefault("ml.feature_store.feature_ttl", "24h")
	viper.SetDefault("ml.feature_store.max_features", 10000)
	viper.SetDefault("ml.feature_store.enable_versioning", true)

	viper.SetDefault("ml.ab_testing.enable_ab_testing", true)
	viper.SetDefault("ml.ab_testing.default_traffic_split", 0.1)
	viper.SetDefault("ml.ab_testing.minimum_sample_size", 1000)
	viper.SetDefault("ml.ab_testing.significance_level", 0.05)
	viper.SetDefault("ml.ab_testing.test_duration", "7d")
	viper.SetDefault("ml.ab_testing.auto_promote", false)
	viper.SetDefault("ml.ab_testing.promotion_threshold", 0.95)

	viper.SetDefault("ml.model_monitoring.enable_monitoring", true)
	viper.SetDefault("ml.model_monitoring.metrics_interval", "1m")
	viper.SetDefault("ml.model_monitoring.drift_detection.enable_drift_detection", true)
	viper.SetDefault("ml.model_monitoring.drift_detection.drift_threshold", 0.1)
	viper.SetDefault("ml.model_monitoring.drift_detection.drift_method", "ks")
	viper.SetDefault("ml.model_monitoring.drift_detection.window_size", 1000)
	viper.SetDefault("ml.model_monitoring.drift_detection.check_interval", "1h")

	viper.SetDefault("ml.auto_retraining.enable_auto_retraining", true)
	viper.SetDefault("ml.auto_retraining.retraining_schedule", "0 2 * * 0") // Weekly at 2 AM Sunday
	viper.SetDefault("ml.auto_retraining.data_window", "30d")
	viper.SetDefault("ml.auto_retraining.min_data_threshold", 10000)
	viper.SetDefault("ml.auto_retraining.retraining_cooldown", "24h")

	// Monitoring defaults
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.metrics_path", "/metrics")
	viper.SetDefault("monitoring.metrics_interval", "15s")
	viper.SetDefault("monitoring.jaeger.enabled", true)
	viper.SetDefault("monitoring.jaeger.service_name", "ml-pipeline")
	viper.SetDefault("monitoring.jaeger.agent_host", "localhost")
	viper.SetDefault("monitoring.jaeger.agent_port", 6831)
	viper.SetDefault("monitoring.jaeger.sampler_type", "const")
	viper.SetDefault("monitoring.jaeger.sampler_param", 1.0)

	// Security defaults
	viper.SetDefault("security.enable_auth", true)
	viper.SetDefault("security.jwt_expiration", "24h")
	viper.SetDefault("security.enable_rbac", true)
	viper.SetDefault("security.enable_encryption", true)

	// Storage defaults
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.local_path", "./storage")

	// Features defaults
	viper.SetDefault("features.enable_async_training", true)
	viper.SetDefault("features.enable_model_versioning", true)
	viper.SetDefault("features.enable_experiment_tracking", true)
	viper.SetDefault("features.enable_auto_scaling", true)
	viper.SetDefault("features.enable_gpu_acceleration", false)
	viper.SetDefault("features.enable_distributed_training", false)
	viper.SetDefault("features.enable_online_features", true)
	viper.SetDefault("features.enable_batch_features", true)
}