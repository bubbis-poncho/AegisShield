package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the data ingestion service
type Config struct {
	Environment string         `json:"environment"`
	Server      ServerConfig   `json:"server"`
	Database    DatabaseConfig `json:"database"`
	Storage     StorageConfig  `json:"storage"`
	Kafka       KafkaConfig    `json:"kafka"`
	Tracing     TracingConfig  `json:"tracing"`
	Metrics     MetricsConfig  `json:"metrics"`
}

type ServerConfig struct {
	GRPCPort        int           `json:"grpc_port"`
	HTTPPort        int           `json:"http_port"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	MaxFileSize     int64         `json:"max_file_size"`
	UploadTimeout   time.Duration `json:"upload_timeout"`
}

type DatabaseConfig struct {
	URL             string        `json:"url"`
	Driver          string        `json:"driver"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
	MigrationsPath  string        `json:"migrations_path"`
}

type StorageConfig struct {
	Type            string `json:"type"` // "s3", "local", "gcs"
	BucketName      string `json:"bucket_name"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"` // For S3-compatible storage
	LocalPath       string `json:"local_path"`
	MaxRetries      int    `json:"max_retries"`
}

type KafkaConfig struct {
	Brokers              []string      `json:"brokers"`
	SecurityProtocol     string        `json:"security_protocol"`
	SASLMechanism        string        `json:"sasl_mechanism"`
	SASLUsername         string        `json:"sasl_username"`
	SASLPassword         string        `json:"sasl_password"`
	SSLCALocation        string        `json:"ssl_ca_location"`
	ProducerTimeout      time.Duration `json:"producer_timeout"`
	ProducerRetries      int           `json:"producer_retries"`
	ProducerBatchSize    int           `json:"producer_batch_size"`
	ProducerFlushTimeout time.Duration `json:"producer_flush_timeout"`
	
	// Topic configurations
	Topics struct {
		FileUpload      string `json:"file_upload"`
		DataProcessing  string `json:"data_processing"`
		DataValidation  string `json:"data_validation"`
		TransactionFlow string `json:"transaction_flow"`
		ErrorEvents     string `json:"error_events"`
	} `json:"topics"`
}

type TracingConfig struct {
	Enabled     bool    `json:"enabled"`
	ServiceName string  `json:"service_name"`
	Environment string  `json:"environment"`
	Endpoint    string  `json:"endpoint"`
	SampleRate  float64 `json:"sample_rate"`
}

type MetricsConfig struct {
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	Namespace  string `json:"namespace"`
	Subsystem  string `json:"subsystem"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			GRPCPort:        getEnvAsInt("GRPC_PORT", 50051),
			HTTPPort:        getEnvAsInt("HTTP_PORT", 8080),
			ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", "30s"),
			MaxFileSize:     getEnvAsInt64("MAX_FILE_SIZE", 100*1024*1024), // 100MB
			UploadTimeout:   getEnvAsDuration("UPLOAD_TIMEOUT", "5m"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://user:password@localhost/aegisshield?sslmode=disable"),
			Driver:          getEnv("DATABASE_DRIVER", "postgres"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", "5m"),
			MigrationsPath:  getEnv("DB_MIGRATIONS_PATH", "file://migrations"),
		},
		Storage: StorageConfig{
			Type:            getEnv("STORAGE_TYPE", "local"),
			BucketName:      getEnv("STORAGE_BUCKET_NAME", "aegisshield-data"),
			Region:          getEnv("STORAGE_REGION", "us-east-1"),
			AccessKeyID:     getEnv("STORAGE_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("STORAGE_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("STORAGE_ENDPOINT", ""),
			LocalPath:       getEnv("STORAGE_LOCAL_PATH", "./uploads"),
			MaxRetries:      getEnvAsInt("STORAGE_MAX_RETRIES", 3),
		},
		Kafka: KafkaConfig{
			Brokers:              getEnvAsStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			SecurityProtocol:     getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
			SASLMechanism:        getEnv("KAFKA_SASL_MECHANISM", ""),
			SASLUsername:         getEnv("KAFKA_SASL_USERNAME", ""),
			SASLPassword:         getEnv("KAFKA_SASL_PASSWORD", ""),
			SSLCALocation:        getEnv("KAFKA_SSL_CA_LOCATION", ""),
			ProducerTimeout:      getEnvAsDuration("KAFKA_PRODUCER_TIMEOUT", "10s"),
			ProducerRetries:      getEnvAsInt("KAFKA_PRODUCER_RETRIES", 3),
			ProducerBatchSize:    getEnvAsInt("KAFKA_PRODUCER_BATCH_SIZE", 16384),
			ProducerFlushTimeout: getEnvAsDuration("KAFKA_PRODUCER_FLUSH_TIMEOUT", "5s"),
		},
		Tracing: TracingConfig{
			Enabled:     getEnvAsBool("TRACING_ENABLED", true),
			ServiceName: getEnv("TRACING_SERVICE_NAME", "data-ingestion-service"),
			Environment: getEnv("TRACING_ENVIRONMENT", "development"),
			Endpoint:    getEnv("TRACING_ENDPOINT", "http://localhost:14268/api/traces"),
			SampleRate:  getEnvAsFloat64("TRACING_SAMPLE_RATE", 0.1),
		},
		Metrics: MetricsConfig{
			Enabled:   getEnvAsBool("METRICS_ENABLED", true),
			Port:      getEnvAsInt("METRICS_PORT", 2112),
			Path:      getEnv("METRICS_PATH", "/metrics"),
			Namespace: getEnv("METRICS_NAMESPACE", "aegisshield"),
			Subsystem: getEnv("METRICS_SUBSYSTEM", "data_ingestion"),
		},
	}

	// Set Kafka topics
	cfg.Kafka.Topics.FileUpload = getEnv("KAFKA_TOPIC_FILE_UPLOAD", "aegis.data.file-upload")
	cfg.Kafka.Topics.DataProcessing = getEnv("KAFKA_TOPIC_DATA_PROCESSING", "aegis.data.processing")
	cfg.Kafka.Topics.DataValidation = getEnv("KAFKA_TOPIC_DATA_VALIDATION", "aegis.data.validation")
	cfg.Kafka.Topics.TransactionFlow = getEnv("KAFKA_TOPIC_TRANSACTION_FLOW", "aegis.data.transaction-flow")
	cfg.Kafka.Topics.ErrorEvents = getEnv("KAFKA_TOPIC_ERROR_EVENTS", "aegis.data.errors")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker is required")
	}

	if c.Storage.Type == "s3" || c.Storage.Type == "gcs" {
		if c.Storage.BucketName == "" {
			return fmt.Errorf("bucket name is required for cloud storage")
		}
	}

	if c.Storage.Type == "local" && c.Storage.LocalPath == "" {
		return fmt.Errorf("local path is required for local storage")
	}

	if c.Server.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}

	return nil
}

// Utility functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	if parsed, err := time.ParseDuration(defaultValue); err == nil {
		return parsed
	}
	return 0
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		return strings.Split(value, ",")
	}
	return defaultValue
}