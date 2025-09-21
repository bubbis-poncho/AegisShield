package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Environment string        `mapstructure:"environment"`
	Server      ServerConfig  `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Neo4j       Neo4jConfig   `mapstructure:"neo4j"`
	Kafka       KafkaConfig   `mapstructure:"kafka"`
	GraphEngine GraphEngineConfig `mapstructure:"graph_engine"`
	Logging     LoggingConfig `mapstructure:"logging"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	GRPCPort     int  `mapstructure:"grpc_port"`
	HTTPPort     int  `mapstructure:"http_port"`
	ReadTimeout  int  `mapstructure:"read_timeout"`
	WriteTimeout int  `mapstructure:"write_timeout"`
	IdleTimeout  int  `mapstructure:"idle_timeout"`
	Debug        bool `mapstructure:"debug"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MaxIdleTime     time.Duration `mapstructure:"max_idle_time"`
	MaxLifetime     time.Duration `mapstructure:"max_lifetime"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	MigrationsPath  string        `mapstructure:"migrations_path"`
}

// Neo4jConfig holds Neo4j configuration
type Neo4jConfig struct {
	URI               string        `mapstructure:"uri"`
	Username          string        `mapstructure:"username"`
	Password          string        `mapstructure:"password"`
	Database          string        `mapstructure:"database"`
	MaxConnections    int           `mapstructure:"max_connections"`
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers                string `mapstructure:"brokers"`
	ConsumerGroup          string `mapstructure:"consumer_group"`
	GraphAnalysisTopic     string `mapstructure:"graph_analysis_topic"`
	NetworkEventsTopic     string `mapstructure:"network_events_topic"`
	InvestigationTopic     string `mapstructure:"investigation_topic"`
	PatternDetectionTopic  string `mapstructure:"pattern_detection_topic"`
	EntityResolvedTopic    string `mapstructure:"entity_resolved_topic"`
}

// GraphEngineConfig holds graph engine specific configuration
type GraphEngineConfig struct {
	MaxTraversalDepth      int     `mapstructure:"max_traversal_depth"`
	MaxPathLength          int     `mapstructure:"max_path_length"`
	MinPathConfidence      float64 `mapstructure:"min_path_confidence"`
	MaxConcurrentAnalyses  int     `mapstructure:"max_concurrent_analyses"`
	AnalysisTimeout        time.Duration `mapstructure:"analysis_timeout"`
	PatternCacheSize       int     `mapstructure:"pattern_cache_size"`
	CentralityThreshold    float64 `mapstructure:"centrality_threshold"`
	ClusteringThreshold    float64 `mapstructure:"clustering_threshold"`
	AnomalyThreshold       float64 `mapstructure:"anomaly_threshold"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/graph-engine")

	// Set default values
	setDefaults()

	// Enable environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("GRAPH_ENGINE")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Environment
	viper.SetDefault("environment", "development")

	// Server defaults
	viper.SetDefault("server.grpc_port", 50053)
	viper.SetDefault("server.http_port", 8083)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 120)
	viper.SetDefault("server.debug", false)

	// Database defaults
	viper.SetDefault("database.url", "postgres://postgres:password@localhost:5432/aegisshield?sslmode=disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_time", "30m")
	viper.SetDefault("database.max_lifetime", "1h")
	viper.SetDefault("database.connect_timeout", "10s")
	viper.SetDefault("database.migrations_path", "file://migrations")

	// Neo4j defaults
	viper.SetDefault("neo4j.uri", "bolt://localhost:7687")
	viper.SetDefault("neo4j.username", "neo4j")
	viper.SetDefault("neo4j.password", "password")
	viper.SetDefault("neo4j.database", "neo4j")
	viper.SetDefault("neo4j.max_connections", 10)
	viper.SetDefault("neo4j.connection_timeout", "30s")

	// Kafka defaults
	viper.SetDefault("kafka.brokers", "localhost:9092")
	viper.SetDefault("kafka.consumer_group", "graph-engine")
	viper.SetDefault("kafka.graph_analysis_topic", "graph.analysis")
	viper.SetDefault("kafka.network_events_topic", "network.events")
	viper.SetDefault("kafka.investigation_topic", "investigations")
	viper.SetDefault("kafka.pattern_detection_topic", "patterns.detected")
	viper.SetDefault("kafka.entity_resolved_topic", "entities.resolved")

	// Graph engine defaults
	viper.SetDefault("graph_engine.max_traversal_depth", 10)
	viper.SetDefault("graph_engine.max_path_length", 15)
	viper.SetDefault("graph_engine.min_path_confidence", 0.5)
	viper.SetDefault("graph_engine.max_concurrent_analyses", 5)
	viper.SetDefault("graph_engine.analysis_timeout", "5m")
	viper.SetDefault("graph_engine.pattern_cache_size", 1000)
	viper.SetDefault("graph_engine.centrality_threshold", 0.7)
	viper.SetDefault("graph_engine.clustering_threshold", 0.6)
	viper.SetDefault("graph_engine.anomaly_threshold", 0.8)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
}

func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.GRPCPort <= 0 || config.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", config.Server.GRPCPort)
	}

	if config.Server.HTTPPort <= 0 || config.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", config.Server.HTTPPort)
	}

	// Validate database configuration
	if config.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if config.Database.MaxConnections <= 0 {
		return fmt.Errorf("database max_connections must be positive")
	}

	// Validate Neo4j configuration
	if config.Neo4j.URI == "" {
		return fmt.Errorf("Neo4j URI is required")
	}

	if config.Neo4j.Username == "" {
		return fmt.Errorf("Neo4j username is required")
	}

	if config.Neo4j.Password == "" {
		return fmt.Errorf("Neo4j password is required")
	}

	// Validate Kafka configuration
	if config.Kafka.Brokers == "" {
		return fmt.Errorf("Kafka brokers are required")
	}

	if config.Kafka.ConsumerGroup == "" {
		return fmt.Errorf("Kafka consumer group is required")
	}

	// Validate graph engine configuration
	if config.GraphEngine.MaxTraversalDepth <= 0 {
		return fmt.Errorf("max_traversal_depth must be positive")
	}

	if config.GraphEngine.MaxPathLength <= 0 {
		return fmt.Errorf("max_path_length must be positive")
	}

	if config.GraphEngine.MinPathConfidence < 0 || config.GraphEngine.MinPathConfidence > 1 {
		return fmt.Errorf("min_path_confidence must be between 0 and 1")
	}

	if config.GraphEngine.MaxConcurrentAnalyses <= 0 {
		return fmt.Errorf("max_concurrent_analyses must be positive")
	}

	if config.GraphEngine.CentralityThreshold < 0 || config.GraphEngine.CentralityThreshold > 1 {
		return fmt.Errorf("centrality_threshold must be between 0 and 1")
	}

	if config.GraphEngine.ClusteringThreshold < 0 || config.GraphEngine.ClusteringThreshold > 1 {
		return fmt.Errorf("clustering_threshold must be between 0 and 1")
	}

	if config.GraphEngine.AnomalyThreshold < 0 || config.GraphEngine.AnomalyThreshold > 1 {
		return fmt.Errorf("anomaly_threshold must be between 0 and 1")
	}

	return nil
}