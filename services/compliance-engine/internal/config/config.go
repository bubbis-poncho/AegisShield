package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Compliance ComplianceConfig `mapstructure:"compliance"`
	Reporting  ReportingConfig  `mapstructure:"reporting"`
	Audit      AuditConfig      `mapstructure:"audit"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Security   SecurityConfig   `mapstructure:"security"`
}

// ServerConfig contains HTTP/gRPC server configuration
type ServerConfig struct {
	HTTPPort int    `mapstructure:"http_port"`
	GRPCPort int    `mapstructure:"grpc_port"`
	Host     string `mapstructure:"host"`
	Timeout  int    `mapstructure:"timeout"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
	PoolSize int    `mapstructure:"pool_size"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"pool_size"`
}

// KafkaConfig contains Kafka configuration
type KafkaConfig struct {
	Brokers              string        `mapstructure:"brokers"`
	GroupID              string        `mapstructure:"group_id"`
	SecurityProtocol     string        `mapstructure:"security_protocol"`
	SASLMechanism        string        `mapstructure:"sasl_mechanism"`
	SASLUsername         string        `mapstructure:"sasl_username"`
	SASLPassword         string        `mapstructure:"sasl_password"`
	ProducerTimeout      int           `mapstructure:"producer_timeout"`
	ConsumerTimeout      int           `mapstructure:"consumer_timeout"`
	MaxRetries           int           `mapstructure:"max_retries"`
	RetryBackoff         time.Duration `mapstructure:"retry_backoff"`
	Topics               KafkaTopics   `mapstructure:"topics"`
}

// KafkaTopics defines all Kafka topic names
type KafkaTopics struct {
	ComplianceEvents    string `mapstructure:"compliance_events"`
	AuditLogs          string `mapstructure:"audit_logs"`
	RegulatoryReports  string `mapstructure:"regulatory_reports"`
	AlertEvents        string `mapstructure:"alert_events"`
	TransactionEvents  string `mapstructure:"transaction_events"`
	ViolationEvents    string `mapstructure:"violation_events"`
	ReportGeneration   string `mapstructure:"report_generation"`
}

// ComplianceConfig contains compliance engine settings
type ComplianceConfig struct {
	RulesEngine      RulesEngineConfig      `mapstructure:"rules_engine"`
	Monitoring       ComplianceMonitoring   `mapstructure:"monitoring"`
	Regulations      RegulationsConfig      `mapstructure:"regulations"`
	ViolationHandling ViolationHandlingConfig `mapstructure:"violation_handling"`
	DataRetention    DataRetentionConfig    `mapstructure:"data_retention"`
}

// RulesEngineConfig contains compliance rules engine settings
type RulesEngineConfig struct {
	EnableRealTimeMonitoring bool          `mapstructure:"enable_realtime_monitoring"`
	RuleEvaluationInterval   time.Duration `mapstructure:"rule_evaluation_interval"`
	MaxConcurrentRules       int           `mapstructure:"max_concurrent_rules"`
	RuleTimeout              time.Duration `mapstructure:"rule_timeout"`
	EnableRuleCaching        bool          `mapstructure:"enable_rule_caching"`
	CacheTTL                 time.Duration `mapstructure:"cache_ttl"`
}

// ComplianceMonitoring contains monitoring configuration
type ComplianceMonitoring struct {
	EnableAutoDetection   bool          `mapstructure:"enable_auto_detection"`
	ScanInterval          time.Duration `mapstructure:"scan_interval"`
	AlertThresholds       AlertThresholds `mapstructure:"alert_thresholds"`
	EnablePatternLearning bool          `mapstructure:"enable_pattern_learning"`
}

// AlertThresholds defines thresholds for various alert types
type AlertThresholds struct {
	HighRiskScore     float64 `mapstructure:"high_risk_score"`
	MediumRiskScore   float64 `mapstructure:"medium_risk_score"`
	ViolationCount    int     `mapstructure:"violation_count"`
	SuspiciousPattern int     `mapstructure:"suspicious_pattern"`
}

// RegulationsConfig contains regulatory framework settings
type RegulationsConfig struct {
	EnabledRegulations []string                    `mapstructure:"enabled_regulations"`
	RegionSettings     map[string]RegionConfig     `mapstructure:"region_settings"`
	UpdateFrequency    time.Duration               `mapstructure:"update_frequency"`
	ExternalSources    []ExternalRegulationSource  `mapstructure:"external_sources"`
}

// RegionConfig contains region-specific compliance settings
type RegionConfig struct {
	Jurisdiction    string   `mapstructure:"jurisdiction"`
	ApplicableLaws  []string `mapstructure:"applicable_laws"`
	ReportingFormat string   `mapstructure:"reporting_format"`
	Language        string   `mapstructure:"language"`
	Timezone        string   `mapstructure:"timezone"`
}

// ExternalRegulationSource defines external regulation data sources
type ExternalRegulationSource struct {
	Name     string `mapstructure:"name"`
	URL      string `mapstructure:"url"`
	APIKey   string `mapstructure:"api_key"`
	Format   string `mapstructure:"format"`
	Schedule string `mapstructure:"schedule"`
}

// ViolationHandlingConfig contains violation processing settings
type ViolationHandlingConfig struct {
	AutoEscalation      bool                    `mapstructure:"auto_escalation"`
	EscalationRules     []EscalationRule        `mapstructure:"escalation_rules"`
	NotificationChannels []NotificationChannel  `mapstructure:"notification_channels"`
	RemedialActions     []RemedialAction        `mapstructure:"remedial_actions"`
}

// EscalationRule defines violation escalation rules
type EscalationRule struct {
	Condition    string        `mapstructure:"condition"`
	Severity     string        `mapstructure:"severity"`
	Delay        time.Duration `mapstructure:"delay"`
	Recipients   []string      `mapstructure:"recipients"`
	RequireAck   bool          `mapstructure:"require_ack"`
}

// NotificationChannel defines notification delivery channels
type NotificationChannel struct {
	Type     string            `mapstructure:"type"`
	Config   map[string]string `mapstructure:"config"`
	Enabled  bool              `mapstructure:"enabled"`
	Priority string            `mapstructure:"priority"`
}

// RemedialAction defines automated remedial actions
type RemedialAction struct {
	Trigger     string `mapstructure:"trigger"`
	Action      string `mapstructure:"action"`
	Parameters  map[string]interface{} `mapstructure:"parameters"`
	AutoExecute bool   `mapstructure:"auto_execute"`
}

// DataRetentionConfig contains data retention policies
type DataRetentionConfig struct {
	AuditLogs         time.Duration `mapstructure:"audit_logs"`
	ComplianceReports time.Duration `mapstructure:"compliance_reports"`
	ViolationRecords  time.Duration `mapstructure:"violation_records"`
	ArchiveSettings   ArchiveConfig `mapstructure:"archive_settings"`
}

// ArchiveConfig contains data archiving settings
type ArchiveConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	StorageProvider  string `mapstructure:"storage_provider"`
	CompressionType  string `mapstructure:"compression_type"`
	EncryptionKey    string `mapstructure:"encryption_key"`
	ArchiveSchedule  string `mapstructure:"archive_schedule"`
}

// ReportingConfig contains reporting engine settings
type ReportingConfig struct {
	Templates        TemplatesConfig        `mapstructure:"templates"`
	Generation       ReportGenerationConfig `mapstructure:"generation"`
	Distribution     DistributionConfig     `mapstructure:"distribution"`
	Formats          FormatsConfig          `mapstructure:"formats"`
	Scheduling       SchedulingConfig       `mapstructure:"scheduling"`
}

// TemplatesConfig contains report template settings
type TemplatesConfig struct {
	TemplatesPath    string `mapstructure:"templates_path"`
	CustomTemplates  bool   `mapstructure:"custom_templates"`
	DefaultTemplate  string `mapstructure:"default_template"`
	TemplateCache    bool   `mapstructure:"template_cache"`
}

// ReportGenerationConfig contains report generation settings
type ReportGenerationConfig struct {
	MaxConcurrent    int           `mapstructure:"max_concurrent"`
	Timeout          time.Duration `mapstructure:"timeout"`
	ChunkSize        int           `mapstructure:"chunk_size"`
	EnableAsync      bool          `mapstructure:"enable_async"`
	QueueSize        int           `mapstructure:"queue_size"`
}

// DistributionConfig contains report distribution settings
type DistributionConfig struct {
	EmailSettings    EmailConfig        `mapstructure:"email"`
	SFTPSettings     SFTPConfig         `mapstructure:"sftp"`
	APIEndpoints     []APIEndpoint      `mapstructure:"api_endpoints"`
	StorageSettings  StorageConfig      `mapstructure:"storage"`
}

// EmailConfig contains email distribution settings
type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	FromAddress  string `mapstructure:"from_address"`
	UseTLS       bool   `mapstructure:"use_tls"`
}

// SFTPConfig contains SFTP distribution settings
type SFTPConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	PrivateKey string `mapstructure:"private_key"`
	RemotePath string `mapstructure:"remote_path"`
}

// APIEndpoint defines external API endpoints for report distribution
type APIEndpoint struct {
	Name     string            `mapstructure:"name"`
	URL      string            `mapstructure:"url"`
	Method   string            `mapstructure:"method"`
	Headers  map[string]string `mapstructure:"headers"`
	AuthType string            `mapstructure:"auth_type"`
	Timeout  time.Duration     `mapstructure:"timeout"`
}

// StorageConfig contains storage settings for reports
type StorageConfig struct {
	Provider   string `mapstructure:"provider"`
	BucketName string `mapstructure:"bucket_name"`
	Region     string `mapstructure:"region"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	Path       string `mapstructure:"path"`
}

// FormatsConfig contains supported report formats
type FormatsConfig struct {
	EnabledFormats []string          `mapstructure:"enabled_formats"`
	PDFSettings    PDFFormatConfig   `mapstructure:"pdf"`
	ExcelSettings  ExcelFormatConfig `mapstructure:"excel"`
	CSVSettings    CSVFormatConfig   `mapstructure:"csv"`
	XMLSettings    XMLFormatConfig   `mapstructure:"xml"`
}

// PDFFormatConfig contains PDF-specific settings
type PDFFormatConfig struct {
	FontFamily   string  `mapstructure:"font_family"`
	FontSize     int     `mapstructure:"font_size"`
	Margins      Margins `mapstructure:"margins"`
	Orientation  string  `mapstructure:"orientation"`
	WatermarkURL string  `mapstructure:"watermark_url"`
}

// ExcelFormatConfig contains Excel-specific settings
type ExcelFormatConfig struct {
	SheetName      string `mapstructure:"sheet_name"`
	IncludeCharts  bool   `mapstructure:"include_charts"`
	PasswordProtect bool  `mapstructure:"password_protect"`
	DefaultPassword string `mapstructure:"default_password"`
}

// CSVFormatConfig contains CSV-specific settings
type CSVFormatConfig struct {
	Delimiter    string `mapstructure:"delimiter"`
	QuoteChar    string `mapstructure:"quote_char"`
	EscapeChar   string `mapstructure:"escape_char"`
	IncludeHeader bool  `mapstructure:"include_header"`
}

// XMLFormatConfig contains XML-specific settings
type XMLFormatConfig struct {
	RootElement  string `mapstructure:"root_element"`
	Namespace    string `mapstructure:"namespace"`
	SchemaURL    string `mapstructure:"schema_url"`
	ValidateXSD  bool   `mapstructure:"validate_xsd"`
}

// Margins defines PDF margins
type Margins struct {
	Top    float64 `mapstructure:"top"`
	Bottom float64 `mapstructure:"bottom"`
	Left   float64 `mapstructure:"left"`
	Right  float64 `mapstructure:"right"`
}

// SchedulingConfig contains report scheduling settings
type SchedulingConfig struct {
	EnableScheduler    bool                `mapstructure:"enable_scheduler"`
	DefaultSchedule    string              `mapstructure:"default_schedule"`
	MaxScheduledReports int                `mapstructure:"max_scheduled_reports"`
	ScheduleFormats    []ScheduleFormat    `mapstructure:"schedule_formats"`
	Notifications      ScheduleNotifications `mapstructure:"notifications"`
}

// ScheduleFormat defines available schedule formats
type ScheduleFormat struct {
	Name        string `mapstructure:"name"`
	CronPattern string `mapstructure:"cron_pattern"`
	Description string `mapstructure:"description"`
}

// ScheduleNotifications contains schedule notification settings
type ScheduleNotifications struct {
	OnSuccess bool `mapstructure:"on_success"`
	OnFailure bool `mapstructure:"on_failure"`
	OnDelay   bool `mapstructure:"on_delay"`
}

// AuditConfig contains audit trail settings
type AuditConfig struct {
	EnableAuditLog    bool              `mapstructure:"enable_audit_log"`
	LogLevel          string            `mapstructure:"log_level"`
	RetentionPeriod   time.Duration     `mapstructure:"retention_period"`
	EncryptLogs       bool              `mapstructure:"encrypt_logs"`
	CompressLogs      bool              `mapstructure:"compress_logs"`
	AuditCategories   []AuditCategory   `mapstructure:"audit_categories"`
	ExternalForwarding ExternalForwarding `mapstructure:"external_forwarding"`
}

// AuditCategory defines audit log categories
type AuditCategory struct {
	Name        string   `mapstructure:"name"`
	Events      []string `mapstructure:"events"`
	Severity    string   `mapstructure:"severity"`
	Retention   time.Duration `mapstructure:"retention"`
}

// ExternalForwarding contains external audit log forwarding settings
type ExternalForwarding struct {
	Enabled     bool              `mapstructure:"enabled"`
	Endpoints   []ForwardEndpoint `mapstructure:"endpoints"`
	Format      string            `mapstructure:"format"`
	BatchSize   int               `mapstructure:"batch_size"`
	FlushInterval time.Duration   `mapstructure:"flush_interval"`
}

// ForwardEndpoint defines external audit forwarding endpoints
type ForwardEndpoint struct {
	Name      string            `mapstructure:"name"`
	URL       string            `mapstructure:"url"`
	AuthType  string            `mapstructure:"auth_type"`
	Headers   map[string]string `mapstructure:"headers"`
	Timeout   time.Duration     `mapstructure:"timeout"`
	RetryCount int              `mapstructure:"retry_count"`
}

// MonitoringConfig contains monitoring and metrics settings
type MonitoringConfig struct {
	EnableMetrics   bool              `mapstructure:"enable_metrics"`
	MetricsPort     int               `mapstructure:"metrics_port"`
	MetricsPath     string            `mapstructure:"metrics_path"`
	HealthCheck     HealthCheckConfig `mapstructure:"health_check"`
	AlertManager    AlertManagerConfig `mapstructure:"alert_manager"`
}

// HealthCheckConfig contains health check settings
type HealthCheckConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	Port             int           `mapstructure:"port"`
	Path             string        `mapstructure:"path"`
	Interval         time.Duration `mapstructure:"interval"`
	Timeout          time.Duration `mapstructure:"timeout"`
	UnhealthyThreshold int         `mapstructure:"unhealthy_threshold"`
}

// AlertManagerConfig contains alert manager settings
type AlertManagerConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	URL        string `mapstructure:"url"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	WebhookURL string `mapstructure:"webhook_url"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	JWTSecret       string           `mapstructure:"jwt_secret"`
	JWTExpiry       time.Duration    `mapstructure:"jwt_expiry"`
	EnableTLS       bool             `mapstructure:"enable_tls"`
	TLSCertFile     string           `mapstructure:"tls_cert_file"`
	TLSKeyFile      string           `mapstructure:"tls_key_file"`
	EnableRateLimit bool             `mapstructure:"enable_rate_limit"`
	RateLimit       RateLimitConfig  `mapstructure:"rate_limit"`
	Encryption      EncryptionConfig `mapstructure:"encryption"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int           `mapstructure:"requests_per_minute"`
	BurstSize         int           `mapstructure:"burst_size"`
	CleanupInterval   time.Duration `mapstructure:"cleanup_interval"`
}

// EncryptionConfig contains encryption settings
type EncryptionConfig struct {
	Algorithm   string `mapstructure:"algorithm"`
	KeySize     int    `mapstructure:"key_size"`
	KeyRotation time.Duration `mapstructure:"key_rotation"`
	MasterKey   string `mapstructure:"master_key"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	// Set default values
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.http_port", 8080)
	viper.SetDefault("server.grpc_port", 9090)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.timeout", 30)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.pool_size", 25)

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.pool_size", 10)

	// Kafka defaults
	viper.SetDefault("kafka.group_id", "compliance-engine")
	viper.SetDefault("kafka.producer_timeout", 30)
	viper.SetDefault("kafka.consumer_timeout", 30)
	viper.SetDefault("kafka.max_retries", 3)
	viper.SetDefault("kafka.retry_backoff", "1s")

	// Compliance defaults
	viper.SetDefault("compliance.rules_engine.enable_realtime_monitoring", true)
	viper.SetDefault("compliance.rules_engine.rule_evaluation_interval", "5m")
	viper.SetDefault("compliance.rules_engine.max_concurrent_rules", 10)
	viper.SetDefault("compliance.rules_engine.rule_timeout", "30s")
	viper.SetDefault("compliance.rules_engine.enable_rule_caching", true)
	viper.SetDefault("compliance.rules_engine.cache_ttl", "1h")

	// Monitoring defaults
	viper.SetDefault("monitoring.enable_metrics", true)
	viper.SetDefault("monitoring.metrics_port", 8081)
	viper.SetDefault("monitoring.metrics_path", "/metrics")
	viper.SetDefault("monitoring.health_check.enabled", true)
	viper.SetDefault("monitoring.health_check.port", 8082)
	viper.SetDefault("monitoring.health_check.path", "/health")

	// Security defaults
	viper.SetDefault("security.jwt_expiry", "24h")
	viper.SetDefault("security.enable_tls", false)
	viper.SetDefault("security.enable_rate_limit", true)
	viper.SetDefault("security.rate_limit.requests_per_minute", 60)
	viper.SetDefault("security.rate_limit.burst_size", 10)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Kafka.Brokers == "" {
		return fmt.Errorf("Kafka brokers are required")
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.Username,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// InitLogger initializes the logger based on configuration
func (c *Config) InitLogger() (*zap.Logger, error) {
	var config zap.Config

	if c.Server.Host == "0.0.0.0" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger, nil
}