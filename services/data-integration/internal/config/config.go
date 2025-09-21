package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Environment string         `mapstructure:"environment"`
	Server      ServerConfig   `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Kafka       KafkaConfig    `mapstructure:"kafka"`
	ETL         ETLConfig      `mapstructure:"etl"`
	Storage     StorageConfig  `mapstructure:"storage"`
	Monitoring  MonitoringConfig `mapstructure:"monitoring"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	HTTPPort     int `mapstructure:"http_port"`
	GRPCPort     int `mapstructure:"grpc_port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
	IdleTimeout  int `mapstructure:"idle_timeout"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL             string `mapstructure:"url"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// KafkaConfig represents Kafka configuration
type KafkaConfig struct {
	Brokers          []string `mapstructure:"brokers"`
	GroupID          string   `mapstructure:"group_id"`
	Topics           TopicsConfig `mapstructure:"topics"`
	ConsumerTimeout  int      `mapstructure:"consumer_timeout"`
	ProducerTimeout  int      `mapstructure:"producer_timeout"`
	RetryInterval    int      `mapstructure:"retry_interval"`
	MaxRetries       int      `mapstructure:"max_retries"`
}

// TopicsConfig represents Kafka topics configuration
type TopicsConfig struct {
	RawData        string `mapstructure:"raw_data"`
	ProcessedData  string `mapstructure:"processed_data"`
	ValidationErrors string `mapstructure:"validation_errors"`
	DataLineage    string `mapstructure:"data_lineage"`
	SchemaChanges  string `mapstructure:"schema_changes"`
	QualityMetrics string `mapstructure:"quality_metrics"`
}

// ETLConfig represents ETL pipeline configuration
type ETLConfig struct {
	BatchSize           int           `mapstructure:"batch_size"`
	ProcessingInterval  time.Duration `mapstructure:"processing_interval"`
	RetentionPeriod     time.Duration `mapstructure:"retention_period"`
	MaxConcurrentJobs   int           `mapstructure:"max_concurrent_jobs"`
	ValidationRules     ValidationConfig `mapstructure:"validation"`
	DataQuality         QualityConfig    `mapstructure:"quality"`
}

// ValidationConfig represents data validation configuration
type ValidationConfig struct {
	EnableSchemaValidation bool     `mapstructure:"enable_schema_validation"`
	EnableDataProfiling    bool     `mapstructure:"enable_data_profiling"`
	RequiredFields         []string `mapstructure:"required_fields"`
	DataTypes              map[string]string `mapstructure:"data_types"`
	BusinessRules          []BusinessRule    `mapstructure:"business_rules"`
}

// BusinessRule represents a business validation rule
type BusinessRule struct {
	Name        string      `mapstructure:"name"`
	Description string      `mapstructure:"description"`
	Field       string      `mapstructure:"field"`
	Rule        string      `mapstructure:"rule"`
	Parameters  interface{} `mapstructure:"parameters"`
}

// QualityConfig represents data quality configuration
type QualityConfig struct {
	EnableQualityChecks    bool    `mapstructure:"enable_quality_checks"`
	CompletenessThreshold  float64 `mapstructure:"completeness_threshold"`
	AccuracyThreshold      float64 `mapstructure:"accuracy_threshold"`
	ConsistencyThreshold   float64 `mapstructure:"consistency_threshold"`
	FreshnessThreshold     time.Duration `mapstructure:"freshness_threshold"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type        string `mapstructure:"type"`
	Endpoint    string `mapstructure:"endpoint"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	Region      string `mapstructure:"region"`
	Bucket      string `mapstructure:"bucket"`
	Prefix      string `mapstructure:"prefix"`
	Encryption  bool   `mapstructure:"encryption"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	MetricsEnabled  bool   `mapstructure:"metrics_enabled"`
	MetricsPort     int    `mapstructure:"metrics_port"`
	HealthCheckPath string `mapstructure:"health_check_path"`
	LogLevel        string `mapstructure:"log_level"`
	TracingEnabled  bool   `mapstructure:"tracing_enabled"`
	TracingEndpoint string `mapstructure:"tracing_endpoint"`
}

// Load loads configuration from various sources
func Load() (Config, error) {
	var config Config

	// Set default values
	viper.SetDefault("environment", "development")
	viper.SetDefault("server.http_port", 8080)
	viper.SetDefault("server.grpc_port", 9090)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 30)

	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.conn_max_lifetime", 300)

	viper.SetDefault("kafka.group_id", "data-integration")
	viper.SetDefault("kafka.consumer_timeout", 30)
	viper.SetDefault("kafka.producer_timeout", 10)
	viper.SetDefault("kafka.retry_interval", 5)
	viper.SetDefault("kafka.max_retries", 3)

	viper.SetDefault("kafka.topics.raw_data", "raw-data")
	viper.SetDefault("kafka.topics.processed_data", "processed-data")
	viper.SetDefault("kafka.topics.validation_errors", "validation-errors")
	viper.SetDefault("kafka.topics.data_lineage", "data-lineage")
	viper.SetDefault("kafka.topics.schema_changes", "schema-changes")
	viper.SetDefault("kafka.topics.quality_metrics", "quality-metrics")

	viper.SetDefault("etl.batch_size", 1000)
	viper.SetDefault("etl.processing_interval", "30s")
	viper.SetDefault("etl.retention_period", "720h") // 30 days
	viper.SetDefault("etl.max_concurrent_jobs", 10)

	viper.SetDefault("etl.validation.enable_schema_validation", true)
	viper.SetDefault("etl.validation.enable_data_profiling", true)

	viper.SetDefault("etl.quality.enable_quality_checks", true)
	viper.SetDefault("etl.quality.completeness_threshold", 0.95)
	viper.SetDefault("etl.quality.accuracy_threshold", 0.99)
	viper.SetDefault("etl.quality.consistency_threshold", 0.98)
	viper.SetDefault("etl.quality.freshness_threshold", "1h")

	viper.SetDefault("storage.type", "s3")
	viper.SetDefault("storage.encryption", true)

	viper.SetDefault("monitoring.metrics_enabled", true)
	viper.SetDefault("monitoring.metrics_port", 9090)
	viper.SetDefault("monitoring.health_check_path", "/health")
	viper.SetDefault("monitoring.log_level", "info")
	viper.SetDefault("monitoring.tracing_enabled", true)

	// Set configuration sources
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/data-integration")

	// Enable environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("DATA_INTEGRATION")

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal configuration
	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return config, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// validateConfig validates the loaded configuration
func validateConfig(config Config) error {
	// Validate server configuration
	if config.Server.HTTPPort <= 0 || config.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", config.Server.HTTPPort)
	}

	if config.Server.GRPCPort <= 0 || config.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", config.Server.GRPCPort)
	}

	// Validate database configuration
	if config.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Validate Kafka configuration
	if len(config.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker is required")
	}

	// Validate ETL configuration
	if config.ETL.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	if config.ETL.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("max concurrent jobs must be positive")
	}

	// Validate quality thresholds
	if config.ETL.DataQuality.CompletenessThreshold < 0 || config.ETL.DataQuality.CompletenessThreshold > 1 {
		return fmt.Errorf("completeness threshold must be between 0 and 1")
	}

	if config.ETL.DataQuality.AccuracyThreshold < 0 || config.ETL.DataQuality.AccuracyThreshold > 1 {
		return fmt.Errorf("accuracy threshold must be between 0 and 1")
	}

	if config.ETL.DataQuality.ConsistencyThreshold < 0 || config.ETL.DataQuality.ConsistencyThreshold > 1 {
		return fmt.Errorf("consistency threshold must be between 0 and 1")
	}

	return nil
}

// GetDatabaseURL returns the database connection URL with environment variable substitution
func (c *Config) GetDatabaseURL() string {
	return os.ExpandEnv(c.Database.URL)
}

// GetStorageCredentials returns storage credentials with environment variable substitution
func (c *Config) GetStorageCredentials() (string, string) {
	accessKey := os.ExpandEnv(c.Storage.AccessKey)
	secretKey := os.ExpandEnv(c.Storage.SecretKey)
	return accessKey, secretKey
}