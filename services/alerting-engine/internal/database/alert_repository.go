package database

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// AlertRepository handles alert data operations
type AlertRepository struct {
	BaseRepository
	logger *slog.Logger
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *sqlx.DB, logger *slog.Logger) *AlertRepository {
	return &AlertRepository{
		BaseRepository: BaseRepository{db: db},
		logger:         logger,
	}
}

// Create creates a new alert
func (r *AlertRepository) Create(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO alerts (
			id, rule_id, rule_name, type, severity, priority, status,
			title, description, source, source_event, entity_ids, tags,
			metadata, fingerprint, correlation_id, parent_alert_id,
			escalation_level, assigned_to, expires_at, notification_sent,
			created_at, updated_at
		) VALUES (
			:id, :rule_id, :rule_name, :type, :severity, :priority, :status,
			:title, :description, :source, :source_event, :entity_ids, :tags,
			:metadata, :fingerprint, :correlation_id, :parent_alert_id,
			:escalation_level, :assigned_to, :expires_at, :notification_sent,
			:created_at, :updated_at
		)`

	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, alert)
	if err != nil {
		r.logger.Error("Failed to create alert", "alert_id", alert.ID, "error", err)
		return fmt.Errorf("failed to create alert: %w", err)
	}

	r.logger.Info("Alert created", "alert_id", alert.ID, "rule_id", alert.RuleID)
	return nil
}

// GetByID retrieves an alert by ID
func (r *AlertRepository) GetByID(ctx context.Context, id string) (*Alert, error) {
	query := `
		SELECT * FROM alerts 
		WHERE id = $1 AND deleted_at IS NULL`

	var alert Alert
	err := r.db.GetContext(ctx, &alert, query, id)
	if err != nil {
		r.logger.Error("Failed to get alert by ID", "alert_id", id, "error", err)
		return nil, fmt.Errorf("failed to get alert by ID: %w", err)
	}

	return &alert, nil
}

// Update updates an existing alert
func (r *AlertRepository) Update(ctx context.Context, alert *Alert) error {
	query := `
		UPDATE alerts SET
			status = :status,
			escalation_level = :escalation_level,
			escalated_at = :escalated_at,
			acknowledged_at = :acknowledged_at,
			acknowledged_by = :acknowledged_by,
			resolved_at = :resolved_at,
			resolved_by = :resolved_by,
			resolution_reason = :resolution_reason,
			assigned_to = :assigned_to,
			notification_sent = :notification_sent,
			last_notified_at = :last_notified_at,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL`

	alert.UpdatedAt = time.Now()

	result, err := r.db.NamedExecContext(ctx, query, alert)
	if err != nil {
		r.logger.Error("Failed to update alert", "alert_id", alert.ID, "error", err)
		return fmt.Errorf("failed to update alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found: %s", alert.ID)
	}

	r.logger.Info("Alert updated", "alert_id", alert.ID)
	return nil
}

// List retrieves alerts with filtering and pagination
func (r *AlertRepository) List(ctx context.Context, filter Filter) ([]*Alert, int, error) {
	whereClause, args, argIndex := r.buildWhereClause(filter)
	
	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alerts %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		r.logger.Error("Failed to count alerts", "error", err)
		return nil, 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Data query
	orderClause := r.buildOrderClause(filter)
	limitClause := r.buildLimitClause(filter, &argIndex, &args)
	
	dataQuery := fmt.Sprintf(`
		SELECT * FROM alerts %s %s %s`,
		whereClause, orderClause, limitClause)

	var alerts []*Alert
	err = r.db.SelectContext(ctx, &alerts, dataQuery, args...)
	if err != nil {
		r.logger.Error("Failed to list alerts", "error", err)
		return nil, 0, fmt.Errorf("failed to list alerts: %w", err)
	}

	return alerts, total, nil
}

// ListByStatus retrieves alerts by status
func (r *AlertRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*Alert, error) {
	query := `
		SELECT * FROM alerts 
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2`

	var alerts []*Alert
	err := r.db.SelectContext(ctx, &alerts, query, status, limit)
	if err != nil {
		r.logger.Error("Failed to list alerts by status", "status", status, "error", err)
		return nil, fmt.Errorf("failed to list alerts by status: %w", err)
	}

	return alerts, nil
}

// ListByCorrelationID retrieves alerts by correlation ID
func (r *AlertRepository) ListByCorrelationID(ctx context.Context, correlationID string) ([]*Alert, error) {
	query := `
		SELECT * FROM alerts 
		WHERE correlation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	var alerts []*Alert
	err := r.db.SelectContext(ctx, &alerts, query, correlationID)
	if err != nil {
		r.logger.Error("Failed to list alerts by correlation ID", "correlation_id", correlationID, "error", err)
		return nil, fmt.Errorf("failed to list alerts by correlation ID: %w", err)
	}

	return alerts, nil
}

// ListExpired retrieves expired alerts
func (r *AlertRepository) ListExpired(ctx context.Context, limit int) ([]*Alert, error) {
	query := `
		SELECT * FROM alerts 
		WHERE expires_at < NOW() 
		AND status NOT IN ('resolved', 'expired')
		AND deleted_at IS NULL
		ORDER BY expires_at ASC
		LIMIT $1`

	var alerts []*Alert
	err := r.db.SelectContext(ctx, &alerts, query, limit)
	if err != nil {
		r.logger.Error("Failed to list expired alerts", "error", err)
		return nil, fmt.Errorf("failed to list expired alerts: %w", err)
	}

	return alerts, nil
}

// ListForEscalation retrieves alerts that need escalation
func (r *AlertRepository) ListForEscalation(ctx context.Context, interval time.Duration, limit int) ([]*Alert, error) {
	query := `
		SELECT * FROM alerts 
		WHERE status = 'open'
		AND escalation_level < 3
		AND (escalated_at IS NULL OR escalated_at < NOW() - INTERVAL '%d minutes')
		AND created_at < NOW() - INTERVAL '%d minutes'
		AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1`

	queryFormatted := fmt.Sprintf(query, int(interval.Minutes()), int(interval.Minutes()))
	
	var alerts []*Alert
	err := r.db.SelectContext(ctx, &alerts, queryFormatted, limit)
	if err != nil {
		r.logger.Error("Failed to list alerts for escalation", "error", err)
		return nil, fmt.Errorf("failed to list alerts for escalation: %w", err)
	}

	return alerts, nil
}

// ListByFingerprint retrieves alerts by fingerprint for deduplication
func (r *AlertRepository) ListByFingerprint(ctx context.Context, fingerprint string, window time.Duration) ([]*Alert, error) {
	query := `
		SELECT * FROM alerts 
		WHERE fingerprint = $1 
		AND created_at > NOW() - INTERVAL '%d minutes'
		AND deleted_at IS NULL
		ORDER BY created_at DESC`

	queryFormatted := fmt.Sprintf(query, int(window.Minutes()))
	
	var alerts []*Alert
	err := r.db.SelectContext(ctx, &alerts, queryFormatted, fingerprint)
	if err != nil {
		r.logger.Error("Failed to list alerts by fingerprint", "fingerprint", fingerprint, "error", err)
		return nil, fmt.Errorf("failed to list alerts by fingerprint: %w", err)
	}

	return alerts, nil
}

// GetStats retrieves alert statistics
func (r *AlertRepository) GetStats(ctx context.Context, timeRange time.Duration) (*AlertStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'open' THEN 1 END) as open,
			COUNT(CASE WHEN status = 'acknowledged' THEN 1 END) as acknowledged,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved,
			COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical,
			COUNT(CASE WHEN severity = 'high' THEN 1 END) as high,
			COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium,
			COUNT(CASE WHEN severity = 'low' THEN 1 END) as low
		FROM alerts 
		WHERE created_at > NOW() - INTERVAL '%d hours'
		AND deleted_at IS NULL`

	queryFormatted := fmt.Sprintf(query, int(timeRange.Hours()))

	var stats AlertStats
	err := r.db.GetContext(ctx, &stats, queryFormatted)
	if err != nil {
		r.logger.Error("Failed to get alert stats", "error", err)
		return nil, fmt.Errorf("failed to get alert stats: %w", err)
	}

	return &stats, nil
}

// Acknowledge acknowledges an alert
func (r *AlertRepository) Acknowledge(ctx context.Context, alertID, acknowledgedBy string) error {
	query := `
		UPDATE alerts SET
			status = 'acknowledged',
			acknowledged_at = NOW(),
			acknowledged_by = $2,
			updated_at = NOW()
		WHERE id = $1 AND status = 'open' AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, alertID, acknowledgedBy)
	if err != nil {
		r.logger.Error("Failed to acknowledge alert", "alert_id", alertID, "error", err)
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or already acknowledged: %s", alertID)
	}

	r.logger.Info("Alert acknowledged", "alert_id", alertID, "acknowledged_by", acknowledgedBy)
	return nil
}

// Resolve resolves an alert
func (r *AlertRepository) Resolve(ctx context.Context, alertID, resolvedBy, reason string) error {
	query := `
		UPDATE alerts SET
			status = 'resolved',
			resolved_at = NOW(),
			resolved_by = $2,
			resolution_reason = $3,
			updated_at = NOW()
		WHERE id = $1 AND status IN ('open', 'acknowledged') AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, alertID, resolvedBy, reason)
	if err != nil {
		r.logger.Error("Failed to resolve alert", "alert_id", alertID, "error", err)
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or already resolved: %s", alertID)
	}

	r.logger.Info("Alert resolved", "alert_id", alertID, "resolved_by", resolvedBy, "reason", reason)
	return nil
}

// Escalate escalates an alert to the next level
func (r *AlertRepository) Escalate(ctx context.Context, alertID string) error {
	query := `
		UPDATE alerts SET
			escalation_level = escalation_level + 1,
			escalated_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND status = 'open' AND escalation_level < 3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, alertID)
	if err != nil {
		r.logger.Error("Failed to escalate alert", "alert_id", alertID, "error", err)
		return fmt.Errorf("failed to escalate alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or cannot be escalated: %s", alertID)
	}

	r.logger.Info("Alert escalated", "alert_id", alertID)
	return nil
}

// Delete soft deletes an alert
func (r *AlertRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE alerts SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete alert", "alert_id", id, "error", err)
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found: %s", id)
	}

	r.logger.Info("Alert deleted", "alert_id", id)
	return nil
}

// Cleanup deletes old alerts beyond retention period
func (r *AlertRepository) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	query := `
		DELETE FROM alerts 
		WHERE created_at < NOW() - INTERVAL '%d days'
		OR (deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '7 days')`

	queryFormatted := fmt.Sprintf(query, retentionDays)

	result, err := r.db.ExecContext(ctx, queryFormatted)
	if err != nil {
		r.logger.Error("Failed to cleanup alerts", "error", err)
		return 0, fmt.Errorf("failed to cleanup alerts: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Alerts cleaned up", "deleted_count", rowsAffected)
	return int(rowsAffected), nil
}

// Helper methods

func (r *AlertRepository) buildWhereClause(filter Filter) (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIndex := 0

	// Base condition
	conditions = append(conditions, "deleted_at IS NULL")

	// Status filter
	if status, ok := filter.Filters["status"].(string); ok && status != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
	}

	// Severity filter
	if severity, ok := filter.Filters["severity"].(string); ok && severity != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIndex))
		args = append(args, severity)
	}

	// Rule ID filter
	if ruleID, ok := filter.Filters["rule_id"].(string); ok && ruleID != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("rule_id = $%d", argIndex))
		args = append(args, ruleID)
	}

	// Assigned to filter
	if assignedTo, ok := filter.Filters["assigned_to"].(string); ok && assignedTo != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, assignedTo)
	}

	// Date range filters
	if filter.DateFrom != nil {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
	}

	if filter.DateTo != nil {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
	}

	// Entity IDs filter
	if entityIDs, ok := filter.Filters["entity_ids"].([]string); ok && len(entityIDs) > 0 {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("entity_ids && $%d", argIndex))
		args = append(args, pq.Array(entityIDs))
	}

	// Search filter
	if search, ok := filter.Filters["search"].(string); ok && search != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf(`
			(title ILIKE $%d OR description ILIKE $%d OR rule_name ILIKE $%d)`, 
			argIndex, argIndex, argIndex))
		args = append(args, "%"+search+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args, argIndex
}

func (r *AlertRepository) buildOrderClause(filter Filter) string {
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = strings.ToUpper(filter.SortOrder)
	}

	return fmt.Sprintf("ORDER BY %s %s", sortBy, sortOrder)
}

func (r *AlertRepository) buildLimitClause(filter Filter, argIndex *int, args *[]interface{}) string {
	if filter.Limit <= 0 {
		return ""
	}

	*argIndex++
	limitClause := fmt.Sprintf("LIMIT $%d", *argIndex)
	*args = append(*args, filter.Limit)

	if filter.Offset > 0 {
		*argIndex++
		limitClause += fmt.Sprintf(" OFFSET $%d", *argIndex)
		*args = append(*args, filter.Offset)
	}

	return limitClause
}

// Custom types for database storage

// JSONB implements database/sql/driver.Valuer and sql.Scanner for JSON fields
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
}