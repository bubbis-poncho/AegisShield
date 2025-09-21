package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for the investigation toolkit service
type Config struct {
	Environment string         `yaml:"environment"`
	Debug       bool           `yaml:"debug"`
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	Neo4j       Neo4jConfig    `yaml:"neo4j"`
	Kafka       KafkaConfig    `yaml:"kafka"`
	Redis       RedisConfig    `yaml:"redis"`
	Storage     StorageConfig  `yaml:"storage"`
	Search      SearchConfig   `yaml:"search"`
	Auth        AuthConfig     `yaml:"auth"`
	Workflow    WorkflowConfig `yaml:"workflow"`
	Audit       AuditConfig    `yaml:"audit"`
}

// ServerConfig contains HTTP and gRPC server settings
type ServerConfig struct {
	HTTPPort         int           `yaml:"http_port"`
	GRPCPort         int           `yaml:"grpc_port"`
	ReadTimeout      time.Duration `yaml:"read_timeout"`
	WriteTimeout     time.Duration `yaml:"write_timeout"`
	IdleTimeout      time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout  time.Duration `yaml:"shutdown_timeout"`
	MaxHeaderBytes   int           `yaml:"max_header_bytes"`
	EnableProfiling  bool          `yaml:"enable_profiling"`
	EnableReflection bool          `yaml:"enable_reflection"`
}

// DatabaseConfig contains PostgreSQL database settings
type DatabaseConfig struct {
	ConnectionString    string        `yaml:"connection_string"`
	MaxOpenConnections  int           `yaml:"max_open_connections"`
	MaxIdleConnections  int           `yaml:"max_idle_connections"`
	ConnectionLifetime  time.Duration `yaml:"connection_lifetime"`
	ConnectionTimeout   time.Duration `yaml:"connection_timeout"`
	QueryTimeout        time.Duration `yaml:"query_timeout"`
	MigrationPath       string        `yaml:"migration_path"`
	EnableQueryLogging  bool          `yaml:"enable_query_logging"`
	SlowQueryThreshold  time.Duration `yaml:"slow_query_threshold"`
}

// Neo4jConfig contains Neo4j graph database settings
type Neo4jConfig struct {
	URI                    string        `yaml:"uri"`
	Username               string        `yaml:"username"`
	Password               string        `yaml:"password"`
	MaxConnectionPoolSize  int           `yaml:"max_connection_pool_size"`
	MaxTransactionRetries  int           `yaml:"max_transaction_retries"`
	ConnectionTimeout      time.Duration `yaml:"connection_timeout"`
	MaxConnectionLifetime  time.Duration `yaml:"max_connection_lifetime"`
	EnableQueryLogging     bool          `yaml:"enable_query_logging"`
	Database               string        `yaml:"database"`
}

// KafkaConfig contains Kafka settings
type KafkaConfig struct {
	Brokers              []string      `yaml:"brokers"`
	SecurityProtocol     string        `yaml:"security_protocol"`
	SASLMechanism        string        `yaml:"sasl_mechanism"`
	SASLUsername         string        `yaml:"sasl_username"`
	SASLPassword         string        `yaml:"sasl_password"`
	EnableSSL            bool          `yaml:"enable_ssl"`
	SSLCALocation        string        `yaml:"ssl_ca_location"`
	ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
	RequestTimeout       time.Duration `yaml:"request_timeout"`
	RetryMax             int           `yaml:"retry_max"`
	RetryBackoff         time.Duration `yaml:"retry_backoff"`
	BatchSize            int           `yaml:"batch_size"`
	BatchTimeout         time.Duration `yaml:"batch_timeout"`
	CompressionType      string        `yaml:"compression_type"`
	EnableIdempotent     bool          `yaml:"enable_idempotent"`
	Topics               KafkaTopicsConfig `yaml:"topics"`
	Consumer             KafkaConsumerConfig `yaml:"consumer"`
	Producer             KafkaProducerConfig `yaml:"producer"`
}

// KafkaTopicsConfig contains topic names
type KafkaTopicsConfig struct {
	Investigations       string `yaml:"investigations"`
	Evidence             string `yaml:"evidence"`
	CaseUpdates          string `yaml:"case_updates"`
	CollaborationEvents  string `yaml:"collaboration_events"`
	WorkflowEvents       string `yaml:"workflow_events"`
	AuditEvents          string `yaml:"audit_events"`
}

// KafkaConsumerConfig contains consumer-specific settings
type KafkaConsumerConfig struct {
	GroupID                string        `yaml:"group_id"`
	AutoOffsetReset        string        `yaml:"auto_offset_reset"`
	EnableAutoCommit       bool          `yaml:"enable_auto_commit"`
	AutoCommitInterval     time.Duration `yaml:"auto_commit_interval"`
	SessionTimeout         time.Duration `yaml:"session_timeout"`
	HeartbeatInterval      time.Duration `yaml:"heartbeat_interval"`
	MaxPollRecords         int           `yaml:"max_poll_records"`
	MaxPollInterval        time.Duration `yaml:"max_poll_interval"`
	FetchMinBytes          int           `yaml:"fetch_min_bytes"`
	FetchMaxBytes          int           `yaml:"fetch_max_bytes"`
	FetchMaxWait           time.Duration `yaml:"fetch_max_wait"`
}

// KafkaProducerConfig contains producer-specific settings
type KafkaProducerConfig struct {
	Acks                 string        `yaml:"acks"`
	Retries              int           `yaml:"retries"`
	BatchSize            int           `yaml:"batch_size"`
	LingerMS             time.Duration `yaml:"linger_ms"`
	BufferMemory         int           `yaml:"buffer_memory"`
	MaxBlockMS           time.Duration `yaml:"max_block_ms"`
	RequestTimeoutMS     time.Duration `yaml:"request_timeout_ms"`
	DeliveryTimeoutMS    time.Duration `yaml:"delivery_timeout_ms"`
	EnableIdempotence    bool          `yaml:"enable_idempotence"`
	MaxInFlightRequests  int           `yaml:"max_in_flight_requests"`
}

// RedisConfig contains Redis cache settings
type RedisConfig struct {
	Addresses            []string      `yaml:"addresses"`
	Username             string        `yaml:"username"`
	Password             string        `yaml:"password"`
	Database             int           `yaml:"database"`
	MaxRetries           int           `yaml:"max_retries"`
	MinRetryBackoff      time.Duration `yaml:"min_retry_backoff"`
	MaxRetryBackoff      time.Duration `yaml:"max_retry_backoff"`
	DialTimeout          time.Duration `yaml:"dial_timeout"`
	ReadTimeout          time.Duration `yaml:"read_timeout"`
	WriteTimeout         time.Duration `yaml:"write_timeout"`
	PoolSize             int           `yaml:"pool_size"`
	MinIdleConnections   int           `yaml:"min_idle_connections"`
	MaxConnAge           time.Duration `yaml:"max_conn_age"`
	PoolTimeout          time.Duration `yaml:"pool_timeout"`
	IdleTimeout          time.Duration `yaml:"idle_timeout"`
	IdleCheckFrequency   time.Duration `yaml:"idle_check_frequency"`
	EnableTLS            bool          `yaml:"enable_tls"`
	TLSCertFile          string        `yaml:"tls_cert_file"`
	TLSKeyFile           string        `yaml:"tls_key_file"`
	TLSCAFile            string        `yaml:"tls_ca_file"`
	TLSSkipVerify        bool          `yaml:"tls_skip_verify"`
}

// StorageConfig contains file storage settings
type StorageConfig struct {
	Provider        string          `yaml:"provider"` // local, s3, gcs, azure
	LocalPath       string          `yaml:"local_path"`
	S3Config        S3Config        `yaml:"s3"`
	GCSConfig       GCSConfig       `yaml:"gcs"`
	AzureConfig     AzureConfig     `yaml:"azure"`
	MaxFileSize     int64           `yaml:"max_file_size"`
	AllowedTypes    []string        `yaml:"allowed_types"`
	RetentionPeriod time.Duration   `yaml:"retention_period"`
	EncryptionKey   string          `yaml:"encryption_key"`
	EnableVersioning bool           `yaml:"enable_versioning"`
}

// S3Config contains AWS S3 storage settings
type S3Config struct {
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SessionToken    string `yaml:"session_token"`
	Endpoint        string `yaml:"endpoint"`
	UseSSL          bool   `yaml:"use_ssl"`
	PathStyle       bool   `yaml:"path_style"`
}

// GCSConfig contains Google Cloud Storage settings
type GCSConfig struct {
	ProjectID           string `yaml:"project_id"`
	Bucket              string `yaml:"bucket"`
	CredentialsFile     string `yaml:"credentials_file"`
	CredentialsJSON     string `yaml:"credentials_json"`
}

// AzureConfig contains Azure Storage settings
type AzureConfig struct {
	StorageAccount   string `yaml:"storage_account"`
	StorageKey       string `yaml:"storage_key"`
	ContainerName    string `yaml:"container_name"`
	Endpoint         string `yaml:"endpoint"`
}

// SearchConfig contains Elasticsearch settings
type SearchConfig struct {
	Addresses            []string      `yaml:"addresses"`
	Username             string        `yaml:"username"`
	Password             string        `yaml:"password"`
	APIKey               string        `yaml:"api_key"`
	CloudID              string        `yaml:"cloud_id"`
	EnableSSL            bool          `yaml:"enable_ssl"`
	SSLCertificatePath   string        `yaml:"ssl_certificate_path"`
	SSLKeyPath           string        `yaml:"ssl_key_path"`
	SSLCAPath            string        `yaml:"ssl_ca_path"`
	SSLSkipVerify        bool          `yaml:"ssl_skip_verify"`
	MaxRetries           int           `yaml:"max_retries"`
	RequestTimeout       time.Duration `yaml:"request_timeout"`
	MaxIdleConnections   int           `yaml:"max_idle_connections"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout"`
	EnableGzip           bool          `yaml:"enable_gzip"`
	EnableMetrics        bool          `yaml:"enable_metrics"`
	IndexPrefix          string        `yaml:"index_prefix"`
	IndexSettings        map[string]interface{} `yaml:"index_settings"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	Provider         string        `yaml:"provider"` // jwt, oauth2, ldap
	JWTSecret        string        `yaml:"jwt_secret"`
	JWTExpiration    time.Duration `yaml:"jwt_expiration"`
	JWTRefreshExp    time.Duration `yaml:"jwt_refresh_expiration"`
	OAuth2Config     OAuth2Config  `yaml:"oauth2"`
	LDAPConfig       LDAPConfig    `yaml:"ldap"`
	EnableMFA        bool          `yaml:"enable_mfa"`
	MFAProvider      string        `yaml:"mfa_provider"`
	SessionTimeout   time.Duration `yaml:"session_timeout"`
	MaxSessions      int           `yaml:"max_sessions"`
	EnableAuditLog   bool          `yaml:"enable_audit_log"`
}

// OAuth2Config contains OAuth2 settings
type OAuth2Config struct {
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes"`
	AuthURL      string   `yaml:"auth_url"`
	TokenURL     string   `yaml:"token_url"`
	UserInfoURL  string   `yaml:"user_info_url"`
}

// LDAPConfig contains LDAP settings
type LDAPConfig struct {
	Host               string        `yaml:"host"`
	Port               int           `yaml:"port"`
	UseSSL             bool          `yaml:"use_ssl"`
	StartTLS           bool          `yaml:"start_tls"`
	SkipCertVerify     bool          `yaml:"skip_cert_verify"`
	BindDN             string        `yaml:"bind_dn"`
	BindPassword       string        `yaml:"bind_password"`
	UserSearchBase     string        `yaml:"user_search_base"`
	UserSearchFilter   string        `yaml:"user_search_filter"`
	GroupSearchBase    string        `yaml:"group_search_base"`
	GroupSearchFilter  string        `yaml:"group_search_filter"`
	UserAttrUsername   string        `yaml:"user_attr_username"`
	UserAttrEmail      string        `yaml:"user_attr_email"`
	UserAttrName       string        `yaml:"user_attr_name"`
	GroupAttrName      string        `yaml:"group_attr_name"`
	ConnectionTimeout  time.Duration `yaml:"connection_timeout"`
	RequestTimeout     time.Duration `yaml:"request_timeout"`
}

// WorkflowConfig contains investigation workflow settings
type WorkflowConfig struct {
	EnableAutomation     bool          `yaml:"enable_automation"`
	DefaultTemplate      string        `yaml:"default_template"`
	TemplatesPath        string        `yaml:"templates_path"`
	MaxStepsPerWorkflow  int           `yaml:"max_steps_per_workflow"`
	StepTimeout          time.Duration `yaml:"step_timeout"`
	WorkflowTimeout      time.Duration `yaml:"workflow_timeout"`
	EnableParallelSteps  bool          `yaml:"enable_parallel_steps"`
	MaxParallelSteps     int           `yaml:"max_parallel_steps"`
	RetryPolicy          RetryPolicy   `yaml:"retry_policy"`
	NotificationConfig   NotificationConfig `yaml:"notification"`
}

// RetryPolicy contains workflow retry settings
type RetryPolicy struct {
	MaxRetries      int           `yaml:"max_retries"`
	InitialDelay    time.Duration `yaml:"initial_delay"`
	MaxDelay        time.Duration `yaml:"max_delay"`
	Multiplier      float64       `yaml:"multiplier"`
	EnableJitter    bool          `yaml:"enable_jitter"`
}

// NotificationConfig contains workflow notification settings
type NotificationConfig struct {
	EnableEmail         bool     `yaml:"enable_email"`
	EnableSlack         bool     `yaml:"enable_slack"`
	EnableWebhooks      bool     `yaml:"enable_webhooks"`
	EmailTemplatesPath  string   `yaml:"email_templates_path"`
	SlackWebhookURL     string   `yaml:"slack_webhook_url"`
	SlackChannel        string   `yaml:"slack_channel"`
	WebhookEndpoints    []string `yaml:"webhook_endpoints"`
	NotifyOnStart       bool     `yaml:"notify_on_start"`
	NotifyOnComplete    bool     `yaml:"notify_on_complete"`
	NotifyOnFailure     bool     `yaml:"notify_on_failure"`
	NotifyOnAssignment  bool     `yaml:"notify_on_assignment"`
}

// AuditConfig contains audit logging settings
type AuditConfig struct {
	EnableAuditLog      bool          `yaml:"enable_audit_log"`
	AuditLevel          string        `yaml:"audit_level"` // basic, detailed, full
	LogRetentionPeriod  time.Duration `yaml:"log_retention_period"`
	EnableFileOutput    bool          `yaml:"enable_file_output"`
	AuditLogPath        string        `yaml:"audit_log_path"`
	EnableDBOutput      bool          `yaml:"enable_db_output"`
	EnableKafkaOutput   bool          `yaml:"enable_kafka_output"`
	KafkaAuditTopic     string        `yaml:"kafka_audit_topic"`
	SensitiveFields     []string      `yaml:"sensitive_fields"`
	ExcludedEndpoints   []string      `yaml:"excluded_endpoints"`
	IncludeRequestBody  bool          `yaml:"include_request_body"`
	IncludeResponseBody bool          `yaml:"include_response_body"`
	MaxPayloadSize      int           `yaml:"max_payload_size"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Debug:       getBoolEnv("DEBUG", false),

		Server: ServerConfig{
			HTTPPort:         getIntEnv("HTTP_PORT", 8080),
			GRPCPort:         getIntEnv("GRPC_PORT", 9090),
			ReadTimeout:      getDurationEnv("READ_TIMEOUT", 30*time.Second),
			WriteTimeout:     getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:      getDurationEnv("IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout:  getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
			MaxHeaderBytes:   getIntEnv("MAX_HEADER_BYTES", 1048576),
			EnableProfiling:  getBoolEnv("ENABLE_PROFILING", false),
			EnableReflection: getBoolEnv("ENABLE_REFLECTION", false),
		},

		Database: DatabaseConfig{
			ConnectionString:    getEnv("DATABASE_URL", "postgres://localhost:5432/investigation_toolkit?sslmode=disable"),
			MaxOpenConnections:  getIntEnv("DB_MAX_OPEN_CONNECTIONS", 25),
			MaxIdleConnections:  getIntEnv("DB_MAX_IDLE_CONNECTIONS", 5),
			ConnectionLifetime:  getDurationEnv("DB_CONNECTION_LIFETIME", 1*time.Hour),
			ConnectionTimeout:   getDurationEnv("DB_CONNECTION_TIMEOUT", 30*time.Second),
			QueryTimeout:        getDurationEnv("DB_QUERY_TIMEOUT", 30*time.Second),
			MigrationPath:       getEnv("DB_MIGRATION_PATH", "file://migrations"),
			EnableQueryLogging:  getBoolEnv("DB_ENABLE_QUERY_LOGGING", false),
			SlowQueryThreshold:  getDurationEnv("DB_SLOW_QUERY_THRESHOLD", 1*time.Second),
		},

		Neo4j: Neo4jConfig{
			URI:                   getEnv("NEO4J_URI", "bolt://localhost:7687"),
			Username:              getEnv("NEO4J_USERNAME", "neo4j"),
			Password:              getEnv("NEO4J_PASSWORD", "password"),
			MaxConnectionPoolSize: getIntEnv("NEO4J_MAX_POOL_SIZE", 50),
			MaxTransactionRetries: getIntEnv("NEO4J_MAX_TX_RETRIES", 3),
			ConnectionTimeout:     getDurationEnv("NEO4J_CONNECTION_TIMEOUT", 30*time.Second),
			MaxConnectionLifetime: getDurationEnv("NEO4J_MAX_CONNECTION_LIFETIME", 1*time.Hour),
			EnableQueryLogging:    getBoolEnv("NEO4J_ENABLE_QUERY_LOGGING", false),
			Database:              getEnv("NEO4J_DATABASE", "neo4j"),
		},

		Kafka: KafkaConfig{
			Brokers:           getStringSliceEnv("KAFKA_BROKERS", []string{"localhost:9092"}),
			SecurityProtocol:  getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
			ConnectionTimeout: getDurationEnv("KAFKA_CONNECTION_TIMEOUT", 30*time.Second),
			RequestTimeout:    getDurationEnv("KAFKA_REQUEST_TIMEOUT", 10*time.Second),
			RetryMax:          getIntEnv("KAFKA_RETRY_MAX", 3),
			RetryBackoff:      getDurationEnv("KAFKA_RETRY_BACKOFF", 100*time.Millisecond),
			BatchSize:         getIntEnv("KAFKA_BATCH_SIZE", 16384),
			BatchTimeout:      getDurationEnv("KAFKA_BATCH_TIMEOUT", 10*time.Millisecond),
			CompressionType:   getEnv("KAFKA_COMPRESSION_TYPE", "snappy"),
			EnableIdempotent:  getBoolEnv("KAFKA_ENABLE_IDEMPOTENT", true),

			Topics: KafkaTopicsConfig{
				Investigations:      getEnv("KAFKA_TOPIC_INVESTIGATIONS", "investigations"),
				Evidence:            getEnv("KAFKA_TOPIC_EVIDENCE", "evidence"),
				CaseUpdates:         getEnv("KAFKA_TOPIC_CASE_UPDATES", "case-updates"),
				CollaborationEvents: getEnv("KAFKA_TOPIC_COLLABORATION", "collaboration-events"),
				WorkflowEvents:      getEnv("KAFKA_TOPIC_WORKFLOW", "workflow-events"),
				AuditEvents:         getEnv("KAFKA_TOPIC_AUDIT", "audit-events"),
			},

			Consumer: KafkaConsumerConfig{
				GroupID:             getEnv("KAFKA_CONSUMER_GROUP_ID", "investigation-toolkit"),
				AutoOffsetReset:     getEnv("KAFKA_AUTO_OFFSET_RESET", "earliest"),
				EnableAutoCommit:    getBoolEnv("KAFKA_ENABLE_AUTO_COMMIT", true),
				AutoCommitInterval:  getDurationEnv("KAFKA_AUTO_COMMIT_INTERVAL", 1*time.Second),
				SessionTimeout:      getDurationEnv("KAFKA_SESSION_TIMEOUT", 30*time.Second),
				HeartbeatInterval:   getDurationEnv("KAFKA_HEARTBEAT_INTERVAL", 3*time.Second),
				MaxPollRecords:      getIntEnv("KAFKA_MAX_POLL_RECORDS", 500),
				MaxPollInterval:     getDurationEnv("KAFKA_MAX_POLL_INTERVAL", 5*time.Minute),
				FetchMinBytes:       getIntEnv("KAFKA_FETCH_MIN_BYTES", 1),
				FetchMaxBytes:       getIntEnv("KAFKA_FETCH_MAX_BYTES", 52428800),
				FetchMaxWait:        getDurationEnv("KAFKA_FETCH_MAX_WAIT", 500*time.Millisecond),
			},

			Producer: KafkaProducerConfig{
				Acks:                getEnv("KAFKA_PRODUCER_ACKS", "all"),
				Retries:             getIntEnv("KAFKA_PRODUCER_RETRIES", 3),
				BatchSize:           getIntEnv("KAFKA_PRODUCER_BATCH_SIZE", 16384),
				LingerMS:            getDurationEnv("KAFKA_PRODUCER_LINGER_MS", 5*time.Millisecond),
				BufferMemory:        getIntEnv("KAFKA_PRODUCER_BUFFER_MEMORY", 33554432),
				MaxBlockMS:          getDurationEnv("KAFKA_PRODUCER_MAX_BLOCK_MS", 60*time.Second),
				RequestTimeoutMS:    getDurationEnv("KAFKA_PRODUCER_REQUEST_TIMEOUT_MS", 30*time.Second),
				DeliveryTimeoutMS:   getDurationEnv("KAFKA_PRODUCER_DELIVERY_TIMEOUT_MS", 2*time.Minute),
				EnableIdempotence:   getBoolEnv("KAFKA_PRODUCER_ENABLE_IDEMPOTENCE", true),
				MaxInFlightRequests: getIntEnv("KAFKA_PRODUCER_MAX_IN_FLIGHT_REQUESTS", 5),
			},
		},

		Redis: RedisConfig{
			Addresses:          getStringSliceEnv("REDIS_ADDRESSES", []string{"localhost:6379"}),
			Password:           getEnv("REDIS_PASSWORD", ""),
			Database:           getIntEnv("REDIS_DATABASE", 0),
			MaxRetries:         getIntEnv("REDIS_MAX_RETRIES", 3),
			MinRetryBackoff:    getDurationEnv("REDIS_MIN_RETRY_BACKOFF", 8*time.Millisecond),
			MaxRetryBackoff:    getDurationEnv("REDIS_MAX_RETRY_BACKOFF", 512*time.Millisecond),
			DialTimeout:        getDurationEnv("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:        getDurationEnv("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:       getDurationEnv("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:           getIntEnv("REDIS_POOL_SIZE", 20),
			MinIdleConnections: getIntEnv("REDIS_MIN_IDLE_CONNECTIONS", 5),
			MaxConnAge:         getDurationEnv("REDIS_MAX_CONN_AGE", 30*time.Minute),
			PoolTimeout:        getDurationEnv("REDIS_POOL_TIMEOUT", 4*time.Second),
			IdleTimeout:        getDurationEnv("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			IdleCheckFrequency: getDurationEnv("REDIS_IDLE_CHECK_FREQUENCY", 1*time.Minute),
		},

		Storage: StorageConfig{
			Provider:         getEnv("STORAGE_PROVIDER", "local"),
			LocalPath:        getEnv("STORAGE_LOCAL_PATH", "./storage"),
			MaxFileSize:      getInt64Env("STORAGE_MAX_FILE_SIZE", 100*1024*1024), // 100MB
			AllowedTypes:     getStringSliceEnv("STORAGE_ALLOWED_TYPES", []string{"pdf", "doc", "docx", "xls", "xlsx", "txt", "jpg", "png", "zip"}),
			RetentionPeriod:  getDurationEnv("STORAGE_RETENTION_PERIOD", 365*24*time.Hour), // 1 year
			EnableVersioning: getBoolEnv("STORAGE_ENABLE_VERSIONING", true),
		},

		Search: SearchConfig{
			Addresses:            getStringSliceEnv("ELASTICSEARCH_ADDRESSES", []string{"http://localhost:9200"}),
			Username:             getEnv("ELASTICSEARCH_USERNAME", ""),
			Password:             getEnv("ELASTICSEARCH_PASSWORD", ""),
			MaxRetries:           getIntEnv("ELASTICSEARCH_MAX_RETRIES", 3),
			RequestTimeout:       getDurationEnv("ELASTICSEARCH_REQUEST_TIMEOUT", 30*time.Second),
			MaxIdleConnections:   getIntEnv("ELASTICSEARCH_MAX_IDLE_CONNECTIONS", 10),
			ResponseHeaderTimeout: getDurationEnv("ELASTICSEARCH_RESPONSE_HEADER_TIMEOUT", 10*time.Second),
			EnableGzip:           getBoolEnv("ELASTICSEARCH_ENABLE_GZIP", true),
			EnableMetrics:        getBoolEnv("ELASTICSEARCH_ENABLE_METRICS", true),
			IndexPrefix:          getEnv("ELASTICSEARCH_INDEX_PREFIX", "investigation-toolkit"),
		},

		Auth: AuthConfig{
			Provider:       getEnv("AUTH_PROVIDER", "jwt"),
			JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
			JWTExpiration:  getDurationEnv("JWT_EXPIRATION", 24*time.Hour),
			JWTRefreshExp:  getDurationEnv("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
			EnableMFA:      getBoolEnv("AUTH_ENABLE_MFA", false),
			SessionTimeout: getDurationEnv("AUTH_SESSION_TIMEOUT", 8*time.Hour),
			MaxSessions:    getIntEnv("AUTH_MAX_SESSIONS", 5),
			EnableAuditLog: getBoolEnv("AUTH_ENABLE_AUDIT_LOG", true),
		},

		Workflow: WorkflowConfig{
			EnableAutomation:    getBoolEnv("WORKFLOW_ENABLE_AUTOMATION", true),
			DefaultTemplate:     getEnv("WORKFLOW_DEFAULT_TEMPLATE", "standard-investigation"),
			TemplatesPath:       getEnv("WORKFLOW_TEMPLATES_PATH", "./templates/workflows"),
			MaxStepsPerWorkflow: getIntEnv("WORKFLOW_MAX_STEPS", 50),
			StepTimeout:         getDurationEnv("WORKFLOW_STEP_TIMEOUT", 1*time.Hour),
			WorkflowTimeout:     getDurationEnv("WORKFLOW_TIMEOUT", 24*time.Hour),
			EnableParallelSteps: getBoolEnv("WORKFLOW_ENABLE_PARALLEL_STEPS", true),
			MaxParallelSteps:    getIntEnv("WORKFLOW_MAX_PARALLEL_STEPS", 5),

			RetryPolicy: RetryPolicy{
				MaxRetries:   getIntEnv("WORKFLOW_RETRY_MAX_RETRIES", 3),
				InitialDelay: getDurationEnv("WORKFLOW_RETRY_INITIAL_DELAY", 1*time.Second),
				MaxDelay:     getDurationEnv("WORKFLOW_RETRY_MAX_DELAY", 5*time.Minute),
				Multiplier:   getFloatEnv("WORKFLOW_RETRY_MULTIPLIER", 2.0),
				EnableJitter: getBoolEnv("WORKFLOW_RETRY_ENABLE_JITTER", true),
			},

			NotificationConfig: NotificationConfig{
				EnableEmail:        getBoolEnv("WORKFLOW_NOTIFY_ENABLE_EMAIL", true),
				EnableSlack:        getBoolEnv("WORKFLOW_NOTIFY_ENABLE_SLACK", false),
				EnableWebhooks:     getBoolEnv("WORKFLOW_NOTIFY_ENABLE_WEBHOOKS", false),
				EmailTemplatesPath: getEnv("WORKFLOW_EMAIL_TEMPLATES_PATH", "./templates/email"),
				SlackWebhookURL:    getEnv("WORKFLOW_SLACK_WEBHOOK_URL", ""),
				SlackChannel:       getEnv("WORKFLOW_SLACK_CHANNEL", "#investigations"),
				WebhookEndpoints:   getStringSliceEnv("WORKFLOW_WEBHOOK_ENDPOINTS", []string{}),
				NotifyOnStart:      getBoolEnv("WORKFLOW_NOTIFY_ON_START", true),
				NotifyOnComplete:   getBoolEnv("WORKFLOW_NOTIFY_ON_COMPLETE", true),
				NotifyOnFailure:    getBoolEnv("WORKFLOW_NOTIFY_ON_FAILURE", true),
				NotifyOnAssignment: getBoolEnv("WORKFLOW_NOTIFY_ON_ASSIGNMENT", true),
			},
		},

		Audit: AuditConfig{
			EnableAuditLog:      getBoolEnv("AUDIT_ENABLE_LOG", true),
			AuditLevel:          getEnv("AUDIT_LEVEL", "detailed"),
			LogRetentionPeriod:  getDurationEnv("AUDIT_LOG_RETENTION_PERIOD", 90*24*time.Hour), // 90 days
			EnableFileOutput:    getBoolEnv("AUDIT_ENABLE_FILE_OUTPUT", true),
			AuditLogPath:        getEnv("AUDIT_LOG_PATH", "./logs/audit.log"),
			EnableDBOutput:      getBoolEnv("AUDIT_ENABLE_DB_OUTPUT", true),
			EnableKafkaOutput:   getBoolEnv("AUDIT_ENABLE_KAFKA_OUTPUT", false),
			KafkaAuditTopic:     getEnv("AUDIT_KAFKA_TOPIC", "audit-events"),
			SensitiveFields:     getStringSliceEnv("AUDIT_SENSITIVE_FIELDS", []string{"password", "token", "secret", "key"}),
			ExcludedEndpoints:   getStringSliceEnv("AUDIT_EXCLUDED_ENDPOINTS", []string{"/health", "/metrics"}),
			IncludeRequestBody:  getBoolEnv("AUDIT_INCLUDE_REQUEST_BODY", true),
			IncludeResponseBody: getBoolEnv("AUDIT_INCLUDE_RESPONSE_BODY", false),
			MaxPayloadSize:      getIntEnv("AUDIT_MAX_PAYLOAD_SIZE", 10240), // 10KB
		},
	}

	// Load S3 configuration if provider is s3
	if cfg.Storage.Provider == "s3" {
		cfg.Storage.S3Config = S3Config{
			Region:          getEnv("S3_REGION", "us-east-1"),
			Bucket:          getEnv("S3_BUCKET", ""),
			AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
			SessionToken:    getEnv("S3_SESSION_TOKEN", ""),
			Endpoint:        getEnv("S3_ENDPOINT", ""),
			UseSSL:          getBoolEnv("S3_USE_SSL", true),
			PathStyle:       getBoolEnv("S3_PATH_STYLE", false),
		}
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}

	if c.Database.ConnectionString == "" {
		return fmt.Errorf("database connection string is required")
	}

	if c.Neo4j.URI == "" {
		return fmt.Errorf("Neo4j URI is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker is required")
	}

	if c.Storage.Provider == "s3" && c.Storage.S3Config.Bucket == "" {
		return fmt.Errorf("S3 bucket is required when using S3 storage provider")
	}

	if c.Auth.JWTSecret == "change-me-in-production" && c.Environment == "production" {
		return fmt.Errorf("JWT secret must be changed in production")
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}