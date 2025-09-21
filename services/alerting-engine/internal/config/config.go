package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds the complete configuration for the alerting engine service
type Config struct {
	Environment string       `mapstructure:"environment"`
	Debug       bool         `mapstructure:"debug"`
	Server      ServerConfig `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Redis       RedisConfig    `mapstructure:"redis"`
	Kafka       KafkaConfig    `mapstructure:"kafka"`
	Alerting    AlertingConfig `mapstructure:"alerting"`
	Notifications NotificationsConfig `mapstructure:"notifications"`
	Rules       RulesConfig    `mapstructure:"rules"`
	Scheduler   SchedulerConfig `mapstructure:"scheduler"`
	Security    SecurityConfig `mapstructure:"security"`
	Logging     LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	HTTPPort int `mapstructure:"http_port"`
	GRPCPort int `mapstructure:"grpc_port"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	MigrationsPath  string `mapstructure:"migrations_path"`
}

// RedisConfig contains Redis configuration for caching
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// KafkaConfig contains Kafka configuration
type KafkaConfig struct {
	Brokers []string    `mapstructure:"brokers"`
	GroupID string      `mapstructure:"group_id"`
	Topics  TopicsConfig `mapstructure:"topics"`
	SASL    SASLConfig   `mapstructure:"sasl"`
}

// TopicsConfig contains Kafka topic configuration
type TopicsConfig struct {
	// Input topics (events to monitor)
	PatternDetected          string `mapstructure:"pattern_detected"`
	AnomalyDetected         string `mapstructure:"anomaly_detected"`
	InvestigationCreated    string `mapstructure:"investigation_created"`
	InvestigationUpdated    string `mapstructure:"investigation_updated"`
	AnalysisCompleted       string `mapstructure:"analysis_completed"`
	DataQualityIssues       string `mapstructure:"data_quality_issues"`
	SystemErrors            string `mapstructure:"system_errors"`
	ThresholdViolations     string `mapstructure:"threshold_violations"`
	
	// Output topics (alerts and notifications)
	AlertGenerated          string `mapstructure:"alert_generated"`
	AlertEscalated          string `mapstructure:"alert_escalated"`
	AlertResolved           string `mapstructure:"alert_resolved"`
	NotificationSent        string `mapstructure:"notification_sent"`
	NotificationFailed      string `mapstructure:"notification_failed"`
}

// SASLConfig contains SASL authentication configuration
type SASLConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// AlertingConfig contains alerting engine configuration
type AlertingConfig struct {
	ProcessingInterval    time.Duration `mapstructure:"processing_interval"`
	BatchSize            int           `mapstructure:"batch_size"`
	MaxRetries           int           `mapstructure:"max_retries"`
	RetryDelay           time.Duration `mapstructure:"retry_delay"`
	CorrelationWindow    time.Duration `mapstructure:"correlation_window"`
	DeduplicationWindow  time.Duration `mapstructure:"deduplication_window"`
	AlertTTL             time.Duration `mapstructure:"alert_ttl"`
	EscalationInterval   time.Duration `mapstructure:"escalation_interval"`
	MaxEscalationLevel   int           `mapstructure:"max_escalation_level"`
	HealthCheckInterval  time.Duration `mapstructure:"health_check_interval"`
	MetricsInterval      time.Duration `mapstructure:"metrics_interval"`
}

// NotificationsConfig contains notification configuration
type NotificationsConfig struct {
	Email     EmailConfig     `mapstructure:"email"`
	SMS       SMSConfig       `mapstructure:"sms"`
	Slack     SlackConfig     `mapstructure:"slack"`
	Teams     TeamsConfig     `mapstructure:"teams"`
	Webhook   WebhookConfig   `mapstructure:"webhook"`
	PagerDuty PagerDutyConfig `mapstructure:"pagerduty"`
	Templates TemplatesConfig `mapstructure:"templates"`
}

// EmailConfig contains email notification configuration
type EmailConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Provider        string        `mapstructure:"provider"` // sendgrid, smtp
	SendGridAPIKey  string        `mapstructure:"sendgrid_api_key"`
	SMTPHost        string        `mapstructure:"smtp_host"`
	SMTPPort        int           `mapstructure:"smtp_port"`
	SMTPUsername    string        `mapstructure:"smtp_username"`
	SMTPPassword    string        `mapstructure:"smtp_password"`
	FromAddress     string        `mapstructure:"from_address"`
	FromName        string        `mapstructure:"from_name"`
	ReplyTo         string        `mapstructure:"reply_to"`
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RateLimitPerMin int           `mapstructure:"rate_limit_per_min"`
}

// SMSConfig contains SMS notification configuration
type SMSConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Provider        string        `mapstructure:"provider"` // twilio
	TwilioSID       string        `mapstructure:"twilio_sid"`
	TwilioToken     string        `mapstructure:"twilio_token"`
	FromNumber      string        `mapstructure:"from_number"`
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RateLimitPerMin int           `mapstructure:"rate_limit_per_min"`
}

// SlackConfig contains Slack notification configuration
type SlackConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	WebhookURL      string        `mapstructure:"webhook_url"`
	BotToken        string        `mapstructure:"bot_token"`
	DefaultChannel  string        `mapstructure:"default_channel"`
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RateLimitPerMin int           `mapstructure:"rate_limit_per_min"`
}

// TeamsConfig contains Microsoft Teams notification configuration
type TeamsConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	WebhookURL      string        `mapstructure:"webhook_url"`
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RateLimitPerMin int           `mapstructure:"rate_limit_per_min"`
}

// WebhookConfig contains webhook notification configuration
type WebhookConfig struct {
	Enabled         bool            `mapstructure:"enabled"`
	DefaultURL      string          `mapstructure:"default_url"`
	Headers         map[string]string `mapstructure:"headers"`
	Timeout         time.Duration   `mapstructure:"timeout"`
	MaxRetries      int             `mapstructure:"max_retries"`
	RetryDelay      time.Duration   `mapstructure:"retry_delay"`
	RateLimitPerMin int             `mapstructure:"rate_limit_per_min"`
	SigningSecret   string          `mapstructure:"signing_secret"`
}

// PagerDutyConfig contains PagerDuty notification configuration
type PagerDutyConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	IntegrationKey  string        `mapstructure:"integration_key"`
	ServiceKey      string        `mapstructure:"service_key"`
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RateLimitPerMin int           `mapstructure:"rate_limit_per_min"`
}

// TemplatesConfig contains template configuration
type TemplatesConfig struct {
	Directory       string `mapstructure:"directory"`
	EmailTemplate   string `mapstructure:"email_template"`
	SMSTemplate     string `mapstructure:"sms_template"`
	SlackTemplate   string `mapstructure:"slack_template"`
	TeamsTemplate   string `mapstructure:"teams_template"`
	WebhookTemplate string `mapstructure:"webhook_template"`
}

// RulesConfig contains rule engine configuration
type RulesConfig struct {
	Directory           string        `mapstructure:"directory"`
	ReloadInterval      time.Duration `mapstructure:"reload_interval"`
	MaxRulesPerAlert    int           `mapstructure:"max_rules_per_alert"`
	EvaluationTimeout   time.Duration `mapstructure:"evaluation_timeout"`
	ParallelEvaluation  bool          `mapstructure:"parallel_evaluation"`
	CacheEnabled        bool          `mapstructure:"cache_enabled"`
	CacheTTL            time.Duration `mapstructure:"cache_ttl"`
	DefaultSeverity     string        `mapstructure:"default_severity"`
	DefaultPriority     string        `mapstructure:"default_priority"`
}

// SchedulerConfig contains scheduler configuration
type SchedulerConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	HealthCheckInterval    time.Duration `mapstructure:"health_check_interval"`
	CleanupInterval        time.Duration `mapstructure:"cleanup_interval"`
	EscalationCheckInterval time.Duration `mapstructure:"escalation_check_interval"`
	MetricsInterval        time.Duration `mapstructure:"metrics_interval"`
	AlertRetentionDays     int           `mapstructure:"alert_retention_days"`
	NotificationRetentionDays int        `mapstructure:"notification_retention_days"`
	RuleReloadInterval     time.Duration `mapstructure:"rule_reload_interval"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	EnableTLS           bool   `mapstructure:"enable_tls"`
	TLSCertPath         string `mapstructure:"tls_cert_path"`
	TLSKeyPath          string `mapstructure:"tls_key_path"`
	EnableAuthentication bool   `mapstructure:"enable_authentication"`
	JWTSecret           string `mapstructure:"jwt_secret"`
	APIKeyHeader        string `mapstructure:"api_key_header"`
	EncryptionKey       string `mapstructure:"encryption_key"`
	HashSalt           string `mapstructure:"hash_salt"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level           string `mapstructure:"level"`
	Format          string `mapstructure:"format"` // json, text
	Output          string `mapstructure:"output"` // stdout, file
	FilePath        string `mapstructure:"file_path"`
	MaxSize         int    `mapstructure:"max_size"`
	MaxBackups      int    `mapstructure:"max_backups"`
	MaxAge          int    `mapstructure:"max_age"`
	Compress        bool   `mapstructure:"compress"`
	IncludeSource   bool   `mapstructure:"include_source"`
}

// Load loads configuration from environment variables and config files
func Load() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/alerting-engine")

	// Set default values
	setDefaults()

	// Enable environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ALERTING_ENGINE")

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// General
	viper.SetDefault("environment", "development")
	viper.SetDefault("debug", false)

	// Server
	viper.SetDefault("server.http_port", 8084)
	viper.SetDefault("server.grpc_port", 9084)

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "aegisshield_alerting")
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("database.migrations_path", "file://migrations")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)

	// Kafka
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.group_id", "alerting-engine")
	viper.SetDefault("kafka.sasl.enabled", false)

	// Kafka Topics
	viper.SetDefault("kafka.topics.pattern_detected", "pattern-detected")
	viper.SetDefault("kafka.topics.anomaly_detected", "anomaly-detected")
	viper.SetDefault("kafka.topics.investigation_created", "investigation-created")
	viper.SetDefault("kafka.topics.investigation_updated", "investigation-updated")
	viper.SetDefault("kafka.topics.analysis_completed", "analysis-completed")
	viper.SetDefault("kafka.topics.data_quality_issues", "data-quality-issues")
	viper.SetDefault("kafka.topics.system_errors", "system-errors")
	viper.SetDefault("kafka.topics.threshold_violations", "threshold-violations")
	viper.SetDefault("kafka.topics.alert_generated", "alert-generated")
	viper.SetDefault("kafka.topics.alert_escalated", "alert-escalated")
	viper.SetDefault("kafka.topics.alert_resolved", "alert-resolved")
	viper.SetDefault("kafka.topics.notification_sent", "notification-sent")
	viper.SetDefault("kafka.topics.notification_failed", "notification-failed")

	// Alerting
	viper.SetDefault("alerting.processing_interval", "10s")
	viper.SetDefault("alerting.batch_size", 100)
	viper.SetDefault("alerting.max_retries", 3)
	viper.SetDefault("alerting.retry_delay", "5s")
	viper.SetDefault("alerting.correlation_window", "5m")
	viper.SetDefault("alerting.deduplication_window", "1h")
	viper.SetDefault("alerting.alert_ttl", "24h")
	viper.SetDefault("alerting.escalation_interval", "30m")
	viper.SetDefault("alerting.max_escalation_level", 3)
	viper.SetDefault("alerting.health_check_interval", "30s")
	viper.SetDefault("alerting.metrics_interval", "1m")

	// Notifications
	viper.SetDefault("notifications.email.enabled", false)
	viper.SetDefault("notifications.email.provider", "sendgrid")
	viper.SetDefault("notifications.email.max_retries", 3)
	viper.SetDefault("notifications.email.retry_delay", "10s")
	viper.SetDefault("notifications.email.timeout", "30s")
	viper.SetDefault("notifications.email.rate_limit_per_min", 60)

	viper.SetDefault("notifications.sms.enabled", false)
	viper.SetDefault("notifications.sms.provider", "twilio")
	viper.SetDefault("notifications.sms.max_retries", 3)
	viper.SetDefault("notifications.sms.retry_delay", "10s")
	viper.SetDefault("notifications.sms.timeout", "30s")
	viper.SetDefault("notifications.sms.rate_limit_per_min", 10)

	viper.SetDefault("notifications.slack.enabled", false)
	viper.SetDefault("notifications.slack.max_retries", 3)
	viper.SetDefault("notifications.slack.retry_delay", "5s")
	viper.SetDefault("notifications.slack.timeout", "15s")
	viper.SetDefault("notifications.slack.rate_limit_per_min", 60)

	viper.SetDefault("notifications.teams.enabled", false)
	viper.SetDefault("notifications.teams.max_retries", 3)
	viper.SetDefault("notifications.teams.retry_delay", "5s")
	viper.SetDefault("notifications.teams.timeout", "15s")
	viper.SetDefault("notifications.teams.rate_limit_per_min", 60)

	viper.SetDefault("notifications.webhook.enabled", false)
	viper.SetDefault("notifications.webhook.timeout", "30s")
	viper.SetDefault("notifications.webhook.max_retries", 3)
	viper.SetDefault("notifications.webhook.retry_delay", "10s")
	viper.SetDefault("notifications.webhook.rate_limit_per_min", 120)

	viper.SetDefault("notifications.pagerduty.enabled", false)
	viper.SetDefault("notifications.pagerduty.max_retries", 3)
	viper.SetDefault("notifications.pagerduty.retry_delay", "10s")
	viper.SetDefault("notifications.pagerduty.timeout", "30s")
	viper.SetDefault("notifications.pagerduty.rate_limit_per_min", 60)

	viper.SetDefault("notifications.templates.directory", "./templates")
	viper.SetDefault("notifications.templates.email_template", "email.html")
	viper.SetDefault("notifications.templates.sms_template", "sms.txt")
	viper.SetDefault("notifications.templates.slack_template", "slack.json")
	viper.SetDefault("notifications.templates.teams_template", "teams.json")
	viper.SetDefault("notifications.templates.webhook_template", "webhook.json")

	// Rules
	viper.SetDefault("rules.directory", "./rules")
	viper.SetDefault("rules.reload_interval", "5m")
	viper.SetDefault("rules.max_rules_per_alert", 10)
	viper.SetDefault("rules.evaluation_timeout", "10s")
	viper.SetDefault("rules.parallel_evaluation", true)
	viper.SetDefault("rules.cache_enabled", true)
	viper.SetDefault("rules.cache_ttl", "1h")
	viper.SetDefault("rules.default_severity", "medium")
	viper.SetDefault("rules.default_priority", "normal")

	// Scheduler
	viper.SetDefault("scheduler.enabled", true)
	viper.SetDefault("scheduler.health_check_interval", "1m")
	viper.SetDefault("scheduler.cleanup_interval", "1h")
	viper.SetDefault("scheduler.escalation_check_interval", "5m")
	viper.SetDefault("scheduler.metrics_interval", "30s")
	viper.SetDefault("scheduler.alert_retention_days", 30)
	viper.SetDefault("scheduler.notification_retention_days", 7)
	viper.SetDefault("scheduler.rule_reload_interval", "5m")

	// Security
	viper.SetDefault("security.enable_tls", false)
	viper.SetDefault("security.enable_authentication", false)
	viper.SetDefault("security.api_key_header", "X-API-Key")

	// Logging
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("logging.compress", true)
	viper.SetDefault("logging.include_source", false)
}