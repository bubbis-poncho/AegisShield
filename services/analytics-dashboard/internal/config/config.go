package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Environment string         `mapstructure:"environment"`
	Server      ServerConfig   `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Redis       RedisConfig    `mapstructure:"redis"`
	Kafka       KafkaConfig    `mapstructure:"kafka"`
	Analytics   AnalyticsConfig `mapstructure:"analytics"`
	Logging     LoggingConfig  `mapstructure:"logging"`
	Metrics     MetricsConfig  `mapstructure:"metrics"`
	Security    SecurityConfig `mapstructure:"security"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	HTTP      HTTPConfig      `mapstructure:"http"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
}

// HTTPConfig contains HTTP server settings
type HTTPConfig struct {
	Port           int `mapstructure:"port"`
	ReadTimeout    int `mapstructure:"read_timeout"`
	WriteTimeout   int `mapstructure:"write_timeout"`
	IdleTimeout    int `mapstructure:"idle_timeout"`
	MaxHeaderBytes int `mapstructure:"max_header_bytes"`
}

// WebSocketConfig contains WebSocket server settings
type WebSocketConfig struct {
	Port               int `mapstructure:"port"`
	ReadBufferSize     int `mapstructure:"read_buffer_size"`
	WriteBufferSize    int `mapstructure:"write_buffer_size"`
	CheckOrigin        bool `mapstructure:"check_origin"`
	EnableCompression  bool `mapstructure:"enable_compression"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	Name               string `mapstructure:"name"`
	Username           string `mapstructure:"username"`
	Password           string `mapstructure:"password"`
	SSLMode            string `mapstructure:"ssl_mode"`
	MaxOpenConnections int    `mapstructure:"max_open_connections"`
	MaxIdleConnections int    `mapstructure:"max_idle_connections"`
	ConnMaxLifetime    int    `mapstructure:"connection_max_lifetime"`
	MigrationsPath     string `mapstructure:"migrations_path"`
}

// RedisConfig contains Redis connection settings
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	Database     int    `mapstructure:"database"`
	MaxRetries   int    `mapstructure:"max_retries"`
	DialTimeout  int    `mapstructure:"dial_timeout"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	PoolSize     int    `mapstructure:"pool_size"`
	PoolTimeout  int    `mapstructure:"pool_timeout"`
}

// KafkaConfig contains Kafka configuration
type KafkaConfig struct {
	Brokers       []string           `mapstructure:"brokers"`
	ConsumerGroup string             `mapstructure:"consumer_group"`
	Topics        KafkaTopicsConfig  `mapstructure:"topics"`
	Producer      KafkaProducerConfig `mapstructure:"producer"`
	Consumer      KafkaConsumerConfig `mapstructure:"consumer"`
}

// KafkaTopicsConfig contains Kafka topic names
type KafkaTopicsConfig struct {
	MetricsEvents    string `mapstructure:"metrics_events"`
	AlertEvents      string `mapstructure:"alert_events"`
	TransactionEvents string `mapstructure:"transaction_events"`
	UserEvents       string `mapstructure:"user_events"`
}

// KafkaProducerConfig contains Kafka producer settings
type KafkaProducerConfig struct {
	MaxMessageBytes int `mapstructure:"max_message_bytes"`
	RequiredAcks    int `mapstructure:"required_acks"`
	Timeout         int `mapstructure:"timeout"`
}

// KafkaConsumerConfig contains Kafka consumer settings
type KafkaConsumerConfig struct {
	SessionTimeout     int `mapstructure:"session_timeout"`
	HeartbeatInterval  int `mapstructure:"heartbeat_interval"`
	MaxProcessingTime  int `mapstructure:"max_processing_time"`
}

// AnalyticsConfig contains analytics-specific configuration
type AnalyticsConfig struct {
	Dashboard     DashboardConfig     `mapstructure:"dashboard"`
	DataSources   DataSourcesConfig   `mapstructure:"data_sources"`
	RealTime      RealTimeConfig      `mapstructure:"real_time"`
	Visualization VisualizationConfig `mapstructure:"visualization"`
	KPI           KPIConfig           `mapstructure:"kpi"`
	Export        ExportConfig        `mapstructure:"export"`
}

// DashboardConfig contains dashboard settings
type DashboardConfig struct {
	DefaultRefreshInterval int      `mapstructure:"default_refresh_interval"`
	MaxWidgetsPerDashboard int      `mapstructure:"max_widgets_per_dashboard"`
	DefaultTimeRange       string   `mapstructure:"default_time_range"`
	SupportedTimeRanges    []string `mapstructure:"supported_time_ranges"`
	MaxDataPoints          int      `mapstructure:"max_data_points"`
}

// DataSourcesConfig contains data source settings
type DataSourcesConfig struct {
	TransactionDB    DataSourceConfig `mapstructure:"transaction_db"`
	GraphDB          DataSourceConfig `mapstructure:"graph_db"`
	AlertEngine      DataSourceConfig `mapstructure:"alert_engine"`
	MLPipeline       DataSourceConfig `mapstructure:"ml_pipeline"`
	ComplianceEngine DataSourceConfig `mapstructure:"compliance_engine"`
}

// DataSourceConfig contains individual data source settings
type DataSourceConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	URL        string `mapstructure:"url"`
	Timeout    int    `mapstructure:"timeout"`
	MaxRetries int    `mapstructure:"max_retries"`
}

// RealTimeConfig contains real-time processing settings
type RealTimeConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	UpdateInterval    int  `mapstructure:"update_interval"`
	MaxConnections    int  `mapstructure:"max_connections"`
	BufferSize        int  `mapstructure:"buffer_size"`
	CompressionLevel  int  `mapstructure:"compression_level"`
}

// VisualizationConfig contains chart and visualization settings
type VisualizationConfig struct {
	Charts ChartConfig `mapstructure:"charts"`
	Maps   MapConfig   `mapstructure:"maps"`
	Tables TableConfig `mapstructure:"tables"`
}

// ChartConfig contains chart-specific settings
type ChartConfig struct {
	DefaultType      string `mapstructure:"default_type"`
	MaxSeriesCount   int    `mapstructure:"max_series_count"`
	AnimationEnabled bool   `mapstructure:"animation_enabled"`
	TooltipEnabled   bool   `mapstructure:"tooltip_enabled"`
}

// MapConfig contains map visualization settings
type MapConfig struct {
	DefaultProvider string `mapstructure:"default_provider"`
	MaxMarkers      int    `mapstructure:"max_markers"`
	ClusteringEnabled bool `mapstructure:"clustering_enabled"`
}

// TableConfig contains table visualization settings
type TableConfig struct {
	DefaultPageSize int  `mapstructure:"default_page_size"`
	MaxPageSize     int  `mapstructure:"max_page_size"`
	SortingEnabled  bool `mapstructure:"sorting_enabled"`
	FilteringEnabled bool `mapstructure:"filtering_enabled"`
}

// KPIConfig contains KPI monitoring settings
type KPIConfig struct {
	RefreshInterval  int      `mapstructure:"refresh_interval"`
	AlertThresholds  map[string]float64 `mapstructure:"alert_thresholds"`
	TrendAnalysis    TrendConfig `mapstructure:"trend_analysis"`
	Benchmarking     BenchmarkConfig `mapstructure:"benchmarking"`
}

// TrendConfig contains trend analysis settings
type TrendConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	LookbackPeriod int  `mapstructure:"lookback_period"`
	MinDataPoints  int  `mapstructure:"min_data_points"`
}

// BenchmarkConfig contains benchmarking settings
type BenchmarkConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	ComparisonPeriods []string `mapstructure:"comparison_periods"`
}

// ExportConfig contains data export settings
type ExportConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	MaxExportSize  int      `mapstructure:"max_export_size"`
	SupportedFormats []string `mapstructure:"supported_formats"`
	RetentionDays  int      `mapstructure:"retention_days"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// MetricsConfig contains metrics and monitoring configuration
type MetricsConfig struct {
	Enabled    bool           `mapstructure:"enabled"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
}

// PrometheusConfig contains Prometheus metrics configuration
type PrometheusConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
	Port     int    `mapstructure:"port"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	APIAuth      APIAuthConfig `mapstructure:"api_auth"`
	TLS          TLSConfig     `mapstructure:"tls"`
	RateLimiting RateLimitConfig `mapstructure:"rate_limiting"`
}

// APIAuthConfig contains API authentication settings
type APIAuthConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Type      string `mapstructure:"type"`
	JWTSecret string `mapstructure:"jwt_secret"`
	JWTExpiry int    `mapstructure:"jwt_expiry"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
	BurstSize         int  `mapstructure:"burst_size"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set environment variable prefix
	viper.SetEnvPrefix("ANALYTICS_DASHBOARD")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Override with environment variables
	overrideWithEnvVars()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.http.port", 8080)
	viper.SetDefault("server.http.read_timeout", 30)
	viper.SetDefault("server.http.write_timeout", 30)
	viper.SetDefault("server.http.idle_timeout", 120)
	viper.SetDefault("server.http.max_header_bytes", 1048576)
	viper.SetDefault("server.websocket.port", 8081)
	viper.SetDefault("server.websocket.read_buffer_size", 1024)
	viper.SetDefault("server.websocket.write_buffer_size", 1024)
	viper.SetDefault("server.websocket.check_origin", false)
	viper.SetDefault("server.websocket.enable_compression", true)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_connections", 25)
	viper.SetDefault("database.max_idle_connections", 25)
	viper.SetDefault("database.connection_max_lifetime", 300)

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.dial_timeout", 5)
	viper.SetDefault("redis.read_timeout", 3)
	viper.SetDefault("redis.write_timeout", 3)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.pool_timeout", 30)

	// Analytics defaults
	viper.SetDefault("analytics.dashboard.default_refresh_interval", 30)
	viper.SetDefault("analytics.dashboard.max_widgets_per_dashboard", 20)
	viper.SetDefault("analytics.dashboard.default_time_range", "24h")
	viper.SetDefault("analytics.dashboard.max_data_points", 1000)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
}

// overrideWithEnvVars overrides configuration with environment variables
func overrideWithEnvVars() {
	// Database environment variables
	if host := os.Getenv("DATABASE_HOST"); host != "" {
		viper.Set("database.host", host)
	}
	if port := os.Getenv("DATABASE_PORT"); port != "" {
		viper.Set("database.port", port)
	}
	if name := os.Getenv("DATABASE_NAME"); name != "" {
		viper.Set("database.name", name)
	}
	if username := os.Getenv("DATABASE_USERNAME"); username != "" {
		viper.Set("database.username", username)
	}
	if password := os.Getenv("DATABASE_PASSWORD"); password != "" {
		viper.Set("database.password", password)
	}

	// Redis environment variables
	if host := os.Getenv("REDIS_HOST"); host != "" {
		viper.Set("redis.host", host)
	}
	if port := os.Getenv("REDIS_PORT"); port != "" {
		viper.Set("redis.port", port)
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		viper.Set("redis.password", password)
	}

	// Kafka environment variables
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		viper.Set("kafka.brokers", strings.Split(brokers, ","))
	}

	// Security environment variables
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		viper.Set("security.api_auth.jwt_secret", jwtSecret)
	}
}