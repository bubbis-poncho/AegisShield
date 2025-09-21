package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/aegisshield/alerting-engine/internal/config"
)

// Connect establishes a database connection
func Connect(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations runs database migrations
func RunMigrations(cfg config.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(cfg.MigrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Base repository struct with common functionality
type BaseRepository struct {
	db *sqlx.DB
}

// Transaction executes a function within a database transaction
func (r *BaseRepository) Transaction(fn func(*sqlx.Tx) error) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// Common audit fields
type AuditFields struct {
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// Alert represents an alert in the system
type Alert struct {
	ID               string                 `db:"id" json:"id"`
	RuleID           string                 `db:"rule_id" json:"rule_id"`
	RuleName         string                 `db:"rule_name" json:"rule_name"`
	Type             string                 `db:"type" json:"type"`
	Severity         string                 `db:"severity" json:"severity"`
	Priority         string                 `db:"priority" json:"priority"`
	Status           string                 `db:"status" json:"status"`
	Title            string                 `db:"title" json:"title"`
	Description      string                 `db:"description" json:"description"`
	Source           string                 `db:"source" json:"source"`
	SourceEvent      map[string]interface{} `db:"source_event" json:"source_event"`
	EntityIDs        []string               `db:"entity_ids" json:"entity_ids"`
	Tags             []string               `db:"tags" json:"tags"`
	Metadata         map[string]interface{} `db:"metadata" json:"metadata"`
	Fingerprint      string                 `db:"fingerprint" json:"fingerprint"`
	CorrelationID    *string                `db:"correlation_id" json:"correlation_id,omitempty"`
	ParentAlertID    *string                `db:"parent_alert_id" json:"parent_alert_id,omitempty"`
	EscalationLevel  int                    `db:"escalation_level" json:"escalation_level"`
	EscalatedAt      *time.Time             `db:"escalated_at" json:"escalated_at,omitempty"`
	AcknowledgedAt   *time.Time             `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
	AcknowledgedBy   *string                `db:"acknowledged_by" json:"acknowledged_by,omitempty"`
	ResolvedAt       *time.Time             `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy       *string                `db:"resolved_by" json:"resolved_by,omitempty"`
	ResolutionReason *string                `db:"resolution_reason" json:"resolution_reason,omitempty"`
	AssignedTo       *string                `db:"assigned_to" json:"assigned_to,omitempty"`
	ExpiresAt        *time.Time             `db:"expires_at" json:"expires_at,omitempty"`
	NotificationSent bool                   `db:"notification_sent" json:"notification_sent"`
	LastNotifiedAt   *time.Time             `db:"last_notified_at" json:"last_notified_at,omitempty"`
	AuditFields
}

// Rule represents an alerting rule
type Rule struct {
	ID               string                 `db:"id" json:"id"`
	Name             string                 `db:"name" json:"name"`
	Description      string                 `db:"description" json:"description"`
	Type             string                 `db:"type" json:"type"`
	Severity         string                 `db:"severity" json:"severity"`
	Priority         string                 `db:"priority" json:"priority"`
	Enabled          bool                   `db:"enabled" json:"enabled"`
	Conditions       map[string]interface{} `db:"conditions" json:"conditions"`
	Actions          map[string]interface{} `db:"actions" json:"actions"`
	Tags             []string               `db:"tags" json:"tags"`
	Metadata         map[string]interface{} `db:"metadata" json:"metadata"`
	ThrottleWindow   *time.Duration         `db:"throttle_window" json:"throttle_window,omitempty"`
	EvaluationWindow *time.Duration         `db:"evaluation_window" json:"evaluation_window,omitempty"`
	GroupBy          []string               `db:"group_by" json:"group_by"`
	NotificationChannels []string           `db:"notification_channels" json:"notification_channels"`
	EscalationPolicy *string                `db:"escalation_policy" json:"escalation_policy,omitempty"`
	CreatedBy        string                 `db:"created_by" json:"created_by"`
	UpdatedBy        string                 `db:"updated_by" json:"updated_by"`
	Version          int                    `db:"version" json:"version"`
	AuditFields
}

// Notification represents a sent notification
type Notification struct {
	ID           string                 `db:"id" json:"id"`
	AlertID      string                 `db:"alert_id" json:"alert_id"`
	Channel      string                 `db:"channel" json:"channel"`
	ChannelType  string                 `db:"channel_type" json:"channel_type"`
	Recipient    string                 `db:"recipient" json:"recipient"`
	Subject      *string                `db:"subject" json:"subject,omitempty"`
	Content      string                 `db:"content" json:"content"`
	Status       string                 `db:"status" json:"status"`
	SentAt       *time.Time             `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt  *time.Time             `db:"delivered_at" json:"delivered_at,omitempty"`
	FailedAt     *time.Time             `db:"failed_at" json:"failed_at,omitempty"`
	Error        *string                `db:"error" json:"error,omitempty"`
	Retries      int                    `db:"retries" json:"retries"`
	MaxRetries   int                    `db:"max_retries" json:"max_retries"`
	NextRetryAt  *time.Time             `db:"next_retry_at" json:"next_retry_at,omitempty"`
	Metadata     map[string]interface{} `db:"metadata" json:"metadata"`
	ExternalID   *string                `db:"external_id" json:"external_id,omitempty"`
	ExternalRef  *string                `db:"external_ref" json:"external_ref,omitempty"`
	AuditFields
}

// EscalationPolicy represents an escalation policy
type EscalationPolicy struct {
	ID          string                 `db:"id" json:"id"`
	Name        string                 `db:"name" json:"name"`
	Description string                 `db:"description" json:"description"`
	Enabled     bool                   `db:"enabled" json:"enabled"`
	Rules       []EscalationRule       `db:"rules" json:"rules"`
	Metadata    map[string]interface{} `db:"metadata" json:"metadata"`
	CreatedBy   string                 `db:"created_by" json:"created_by"`
	UpdatedBy   string                 `db:"updated_by" json:"updated_by"`
	AuditFields
}

// EscalationRule represents a single escalation rule
type EscalationRule struct {
	Level               int      `json:"level"`
	DelayMinutes        int      `json:"delay_minutes"`
	NotificationChannels []string `json:"notification_channels"`
	Recipients          []string `json:"recipients"`
	Conditions          map[string]interface{} `json:"conditions,omitempty"`
}

// AlertStats represents alert statistics
type AlertStats struct {
	Total        int `db:"total" json:"total"`
	Open         int `db:"open" json:"open"`
	Acknowledged int `db:"acknowledged" json:"acknowledged"`
	Resolved     int `db:"resolved" json:"resolved"`
	Critical     int `db:"critical" json:"critical"`
	High         int `db:"high" json:"high"`
	Medium       int `db:"medium" json:"medium"`
	Low          int `db:"low" json:"low"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	Total     int `db:"total" json:"total"`
	Pending   int `db:"pending" json:"pending"`
	Sent      int `db:"sent" json:"sent"`
	Delivered int `db:"delivered" json:"delivered"`
	Failed    int `db:"failed" json:"failed"`
}

// Filter represents common filtering options
type Filter struct {
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
	SortBy     string                 `json:"sort_by,omitempty"`
	SortOrder  string                 `json:"sort_order,omitempty"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	DateFrom   *time.Time             `json:"date_from,omitempty"`
	DateTo     *time.Time             `json:"date_to,omitempty"`
	IncludeDeleted bool                `json:"include_deleted,omitempty"`
}