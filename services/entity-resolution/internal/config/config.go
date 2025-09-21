package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Kafka    KafkaConfig    `json:"kafka"`
	Neo4j    Neo4jConfig    `json:"neo4j"`
	Matching MatchingConfig `json:"matching"`
	Logging  LoggingConfig  `json:"logging"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	GRPCPort int `json:"grpc_port"`
	HTTPPort int `json:"http_port"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Database        string        `json:"database"`
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	SSLMode         string        `json:"ssl_mode"`
	MaxConnections  int           `json:"max_connections"`
	MaxIdleTime     time.Duration `json:"max_idle_time"`
	MaxLifetime     time.Duration `json:"max_lifetime"`
	ConnectTimeout  time.Duration `json:"connect_timeout"`
	MigrationsPath  string        `json:"migrations_path"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers                []string      `json:"brokers"`
	ConsumerGroup          string        `json:"consumer_group"`
	TransactionTopic       string        `json:"transaction_topic"`
	EntityResolutionTopic  string        `json:"entity_resolution_topic"`
	BatchSize              int           `json:"batch_size"`
	BatchTimeout           time.Duration `json:"batch_timeout"`
	RetryAttempts          int           `json:"retry_attempts"`
	RetryBackoff           time.Duration `json:"retry_backoff"`
	CompressionType        string        `json:"compression_type"`
	RequiredAcks           int           `json:"required_acks"`
	MaxMessageBytes        int           `json:"max_message_bytes"`
}

// Neo4jConfig holds Neo4j configuration
type Neo4jConfig struct {
	URI                string        `json:"uri"`
	Username           string        `json:"username"`
	Password           string        `json:"password"`
	Database           string        `json:"database"`
	MaxConnections     int           `json:"max_connections"`
	ConnectionTimeout  time.Duration `json:"connection_timeout"`
	MaxTransactionTime time.Duration `json:"max_transaction_time"`
}

// MatchingConfig holds entity matching configuration
type MatchingConfig struct {
	NameSimilarityThreshold    float64 `json:"name_similarity_threshold"`
	AddressSimilarityThreshold float64 `json:"address_similarity_threshold"`
	PhoneSimilarityThreshold   float64 `json:"phone_similarity_threshold"`
	EmailSimilarityThreshold   float64 `json:"email_similarity_threshold"`
	OverallSimilarityThreshold float64 `json:"overall_similarity_threshold"`
	MaxCandidates              int     `json:"max_candidates"`
	FuzzyMatchingEnabled       bool    `json:"fuzzy_matching_enabled"`
	PhoneticMatchingEnabled    bool    `json:"phonetic_matching_enabled"`
	BlockingEnabled            bool    `json:"blocking_enabled"`
	BlockingKeySize            int     `json:"blocking_key_size"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			GRPCPort: getEnvInt("GRPC_PORT", 50052),
			HTTPPort: getEnvInt("HTTP_PORT", 8082),
		},
		Database: DatabaseConfig{
			Host:            getEnvString("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			Database:        getEnvString("DB_NAME", "aegisshield_entity_resolution"),
			Username:        getEnvString("DB_USER", "postgres"),
			Password:        getEnvString("DB_PASSWORD", "password"),
			SSLMode:         getEnvString("DB_SSL_MODE", "disable"),
			MaxConnections:  getEnvInt("DB_MAX_CONNECTIONS", 25),
			MaxIdleTime:     getEnvDuration("DB_MAX_IDLE_TIME", 30*time.Minute),
			MaxLifetime:     getEnvDuration("DB_MAX_LIFETIME", 2*time.Hour),
			ConnectTimeout:  getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
			MigrationsPath:  getEnvString("DB_MIGRATIONS_PATH", "file://migrations"),
		},
		Kafka: KafkaConfig{
			Brokers:               getEnvStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			ConsumerGroup:         getEnvString("KAFKA_CONSUMER_GROUP", "entity-resolution-service"),
			TransactionTopic:      getEnvString("KAFKA_TRANSACTION_TOPIC", "transactions.processed"),
			EntityResolutionTopic: getEnvString("KAFKA_ENTITY_RESOLUTION_TOPIC", "entities.resolved"),
			BatchSize:             getEnvInt("KAFKA_BATCH_SIZE", 100),
			BatchTimeout:          getEnvDuration("KAFKA_BATCH_TIMEOUT", 5*time.Second),
			RetryAttempts:         getEnvInt("KAFKA_RETRY_ATTEMPTS", 3),
			RetryBackoff:          getEnvDuration("KAFKA_RETRY_BACKOFF", 1*time.Second),
			CompressionType:       getEnvString("KAFKA_COMPRESSION_TYPE", "snappy"),
			RequiredAcks:          getEnvInt("KAFKA_REQUIRED_ACKS", 1),
			MaxMessageBytes:       getEnvInt("KAFKA_MAX_MESSAGE_BYTES", 1000000),
		},
		Neo4j: Neo4jConfig{
			URI:                getEnvString("NEO4J_URI", "bolt://localhost:7687"),
			Username:           getEnvString("NEO4J_USERNAME", "neo4j"),
			Password:           getEnvString("NEO4J_PASSWORD", "password"),
			Database:           getEnvString("NEO4J_DATABASE", "neo4j"),
			MaxConnections:     getEnvInt("NEO4J_MAX_CONNECTIONS", 10),
			ConnectionTimeout:  getEnvDuration("NEO4J_CONNECTION_TIMEOUT", 30*time.Second),
			MaxTransactionTime: getEnvDuration("NEO4J_MAX_TRANSACTION_TIME", 30*time.Second),
		},
		Matching: MatchingConfig{
			NameSimilarityThreshold:    getEnvFloat("MATCHING_NAME_THRESHOLD", 0.8),
			AddressSimilarityThreshold: getEnvFloat("MATCHING_ADDRESS_THRESHOLD", 0.85),
			PhoneSimilarityThreshold:   getEnvFloat("MATCHING_PHONE_THRESHOLD", 0.9),
			EmailSimilarityThreshold:   getEnvFloat("MATCHING_EMAIL_THRESHOLD", 0.95),
			OverallSimilarityThreshold: getEnvFloat("MATCHING_OVERALL_THRESHOLD", 0.75),
			MaxCandidates:              getEnvInt("MATCHING_MAX_CANDIDATES", 100),
			FuzzyMatchingEnabled:       getEnvBool("MATCHING_FUZZY_ENABLED", true),
			PhoneticMatchingEnabled:    getEnvBool("MATCHING_PHONETIC_ENABLED", true),
			BlockingEnabled:            getEnvBool("MATCHING_BLOCKING_ENABLED", true),
			BlockingKeySize:            getEnvInt("MATCHING_BLOCKING_KEY_SIZE", 3),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
	}

	return config, config.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}

	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("Kafka brokers are required")
	}

	if c.Kafka.ConsumerGroup == "" {
		return fmt.Errorf("Kafka consumer group is required")
	}

	if c.Neo4j.URI == "" {
		return fmt.Errorf("Neo4j URI is required")
	}

	if c.Neo4j.Username == "" {
		return fmt.Errorf("Neo4j username is required")
	}

	if c.Matching.NameSimilarityThreshold < 0 || c.Matching.NameSimilarityThreshold > 1 {
		return fmt.Errorf("name similarity threshold must be between 0 and 1")
	}

	if c.Matching.OverallSimilarityThreshold < 0 || c.Matching.OverallSimilarityThreshold > 1 {
		return fmt.Errorf("overall similarity threshold must be between 0 and 1")
	}

	if c.Matching.MaxCandidates <= 0 {
		return fmt.Errorf("max candidates must be positive")
	}

	return nil
}

// DatabaseDSN returns the database connection string
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.Username,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}