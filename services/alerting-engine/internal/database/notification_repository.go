package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// NotificationRepository handles notification data operations
type NotificationRepository struct {
	BaseRepository
	logger *slog.Logger
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sqlx.DB, logger *slog.Logger) *NotificationRepository {
	return &NotificationRepository{
		BaseRepository: BaseRepository{db: db},
		logger:         logger,
	}
}

// Create creates a new notification
func (n *NotificationRepository) Create(ctx context.Context, notification *Notification) error {
	query := `
		INSERT INTO notifications (
			id, alert_id, rule_id, channel, recipient, subject, message,
			template_id, template_data, status, delivery_method, priority,
			scheduled_at, sent_at, delivered_at, failed_at, retry_count,
			max_retries, error_message, metadata, external_id, tracking_id,
			created_at, updated_at
		) VALUES (
			:id, :alert_id, :rule_id, :channel, :recipient, :subject, :message,
			:template_id, :template_data, :status, :delivery_method, :priority,
			:scheduled_at, :sent_at, :delivered_at, :failed_at, :retry_count,
			:max_retries, :error_message, :metadata, :external_id, :tracking_id,
			:created_at, :updated_at
		)`

	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	_, err := n.db.NamedExecContext(ctx, query, notification)
	if err != nil {
		n.logger.Error("Failed to create notification", 
			"notification_id", notification.ID, 
			"alert_id", notification.AlertID,
			"error", err)
		return fmt.Errorf("failed to create notification: %w", err)
	}

	n.logger.Info("Notification created", 
		"notification_id", notification.ID,
		"alert_id", notification.AlertID,
		"channel", notification.Channel,
		"recipient", notification.Recipient)
	return nil
}

// GetByID retrieves a notification by ID
func (n *NotificationRepository) GetByID(ctx context.Context, id string) (*Notification, error) {
	query := `SELECT * FROM notifications WHERE id = $1`

	var notification Notification
	err := n.db.GetContext(ctx, &notification, query, id)
	if err != nil {
		n.logger.Error("Failed to get notification by ID", "notification_id", id, "error", err)
		return nil, fmt.Errorf("failed to get notification by ID: %w", err)
	}

	return &notification, nil
}

// GetByAlertID retrieves all notifications for an alert
func (n *NotificationRepository) GetByAlertID(ctx context.Context, alertID string) ([]*Notification, error) {
	query := `
		SELECT * FROM notifications 
		WHERE alert_id = $1 
		ORDER BY created_at DESC`

	var notifications []*Notification
	err := n.db.SelectContext(ctx, &notifications, query, alertID)
	if err != nil {
		n.logger.Error("Failed to get notifications by alert ID", "alert_id", alertID, "error", err)
		return nil, fmt.Errorf("failed to get notifications by alert ID: %w", err)
	}

	return notifications, nil
}

// GetByRuleID retrieves all notifications for a rule
func (n *NotificationRepository) GetByRuleID(ctx context.Context, ruleID string) ([]*Notification, error) {
	query := `
		SELECT * FROM notifications 
		WHERE rule_id = $1 
		ORDER BY created_at DESC`

	var notifications []*Notification
	err := n.db.SelectContext(ctx, &notifications, query, ruleID)
	if err != nil {
		n.logger.Error("Failed to get notifications by rule ID", "rule_id", ruleID, "error", err)
		return nil, fmt.Errorf("failed to get notifications by rule ID: %w", err)
	}

	return notifications, nil
}

// Update updates an existing notification
func (n *NotificationRepository) Update(ctx context.Context, notification *Notification) error {
	query := `
		UPDATE notifications SET
			status = :status,
			sent_at = :sent_at,
			delivered_at = :delivered_at,
			failed_at = :failed_at,
			retry_count = :retry_count,
			error_message = :error_message,
			metadata = :metadata,
			external_id = :external_id,
			tracking_id = :tracking_id,
			updated_at = :updated_at
		WHERE id = :id`

	notification.UpdatedAt = time.Now()

	result, err := n.db.NamedExecContext(ctx, query, notification)
	if err != nil {
		n.logger.Error("Failed to update notification", "notification_id", notification.ID, "error", err)
		return fmt.Errorf("failed to update notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", notification.ID)
	}

	n.logger.Debug("Notification updated", 
		"notification_id", notification.ID,
		"status", notification.Status)
	return nil
}

// UpdateStatus updates the status of a notification
func (n *NotificationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	var updateQuery string
	var args []interface{}

	now := time.Now()
	
	switch status {
	case "sent":
		updateQuery = `
			UPDATE notifications SET
				status = $2,
				sent_at = $3,
				updated_at = $3
			WHERE id = $1`
		args = []interface{}{id, status, now}
	case "delivered":
		updateQuery = `
			UPDATE notifications SET
				status = $2,
				delivered_at = $3,
				updated_at = $3
			WHERE id = $1`
		args = []interface{}{id, status, now}
	case "failed":
		updateQuery = `
			UPDATE notifications SET
				status = $2,
				failed_at = $3,
				updated_at = $3
			WHERE id = $1`
		args = []interface{}{id, status, now}
	default:
		updateQuery = `
			UPDATE notifications SET
				status = $2,
				updated_at = $3
			WHERE id = $1`
		args = []interface{}{id, status, now}
	}

	result, err := n.db.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		n.logger.Error("Failed to update notification status", 
			"notification_id", id, 
			"status", status, 
			"error", err)
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", id)
	}

	n.logger.Debug("Notification status updated", 
		"notification_id", id,
		"status", status)
	return nil
}

// IncrementRetryCount increments the retry count for a notification
func (n *NotificationRepository) IncrementRetryCount(ctx context.Context, id, errorMessage string) error {
	query := `
		UPDATE notifications SET
			retry_count = retry_count + 1,
			error_message = $2,
			status = CASE 
				WHEN retry_count + 1 >= max_retries THEN 'failed'
				ELSE 'pending'
			END,
			failed_at = CASE 
				WHEN retry_count + 1 >= max_retries THEN NOW()
				ELSE failed_at
			END,
			updated_at = NOW()
		WHERE id = $1`

	result, err := n.db.ExecContext(ctx, query, id, errorMessage)
	if err != nil {
		n.logger.Error("Failed to increment retry count", 
			"notification_id", id, 
			"error", err)
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", id)
	}

	n.logger.Debug("Notification retry count incremented", 
		"notification_id", id)
	return nil
}

// List retrieves notifications with filtering and pagination
func (n *NotificationRepository) List(ctx context.Context, filter Filter) ([]*Notification, int, error) {
	whereClause, args, argIndex := n.buildWhereClause(filter)
	
	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", whereClause)
	var total int
	err := n.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		n.logger.Error("Failed to count notifications", "error", err)
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Data query
	orderClause := n.buildOrderClause(filter)
	limitClause := n.buildLimitClause(filter, &argIndex, &args)
	
	dataQuery := fmt.Sprintf(`
		SELECT * FROM notifications %s %s %s`,
		whereClause, orderClause, limitClause)

	var notifications []*Notification
	err = n.db.SelectContext(ctx, &notifications, dataQuery, args...)
	if err != nil {
		n.logger.Error("Failed to list notifications", "error", err)
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}

	return notifications, total, nil
}

// GetPendingNotifications retrieves notifications that need to be sent
func (n *NotificationRepository) GetPendingNotifications(ctx context.Context, limit int) ([]*Notification, error) {
	query := `
		SELECT * FROM notifications 
		WHERE status IN ('pending', 'retry') 
		AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		AND retry_count < max_retries
		ORDER BY priority DESC, created_at ASC
		LIMIT $1`

	var notifications []*Notification
	err := n.db.SelectContext(ctx, &notifications, query, limit)
	if err != nil {
		n.logger.Error("Failed to get pending notifications", "error", err)
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}

	return notifications, nil
}

// GetFailedNotifications retrieves failed notifications for cleanup or analysis
func (n *NotificationRepository) GetFailedNotifications(ctx context.Context, since time.Time, limit int) ([]*Notification, error) {
	query := `
		SELECT * FROM notifications 
		WHERE status = 'failed' 
		AND failed_at >= $1
		ORDER BY failed_at DESC
		LIMIT $2`

	var notifications []*Notification
	err := n.db.SelectContext(ctx, &notifications, query, since, limit)
	if err != nil {
		n.logger.Error("Failed to get failed notifications", "error", err)
		return nil, fmt.Errorf("failed to get failed notifications: %w", err)
	}

	return notifications, nil
}

// GetStatsByChannel retrieves delivery statistics by channel
func (n *NotificationRepository) GetStatsByChannel(ctx context.Context, since time.Time) ([]*NotificationStats, error) {
	query := `
		SELECT 
			channel,
			COUNT(*) as total_count,
			COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered_count,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
			COUNT(CASE WHEN status IN ('pending', 'retry') THEN 1 END) as pending_count,
			AVG(CASE WHEN delivered_at IS NOT NULL AND sent_at IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (delivered_at - sent_at)) END) as avg_delivery_time_seconds
		FROM notifications 
		WHERE created_at >= $1
		GROUP BY channel
		ORDER BY total_count DESC`

	var stats []*NotificationStats
	err := n.db.SelectContext(ctx, &stats, query, since)
	if err != nil {
		n.logger.Error("Failed to get notification stats by channel", "error", err)
		return nil, fmt.Errorf("failed to get notification stats by channel: %w", err)
	}

	return stats, nil
}

// GetStatsByRule retrieves delivery statistics by rule
func (n *NotificationRepository) GetStatsByRule(ctx context.Context, since time.Time) ([]*RuleNotificationStats, error) {
	query := `
		SELECT 
			rule_id,
			r.name as rule_name,
			COUNT(*) as total_count,
			COUNT(CASE WHEN n.status = 'delivered' THEN 1 END) as delivered_count,
			COUNT(CASE WHEN n.status = 'failed' THEN 1 END) as failed_count,
			COUNT(CASE WHEN n.status IN ('pending', 'retry') THEN 1 END) as pending_count
		FROM notifications n
		LEFT JOIN rules r ON n.rule_id = r.id
		WHERE n.created_at >= $1
		GROUP BY rule_id, r.name
		ORDER BY total_count DESC`

	var stats []*RuleNotificationStats
	err := n.db.SelectContext(ctx, &stats, query, since)
	if err != nil {
		n.logger.Error("Failed to get notification stats by rule", "error", err)
		return nil, fmt.Errorf("failed to get notification stats by rule: %w", err)
	}

	return stats, nil
}

// CleanupOldNotifications removes old notifications based on retention policy
func (n *NotificationRepository) CleanupOldNotifications(ctx context.Context, retentionDays int) (int, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	query := `
		DELETE FROM notifications 
		WHERE created_at < $1 
		AND status IN ('delivered', 'failed')`

	result, err := n.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		n.logger.Error("Failed to cleanup old notifications", "error", err)
		return 0, fmt.Errorf("failed to cleanup old notifications: %w", err)
	}

	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	n.logger.Info("Old notifications cleaned up", 
		"deleted_count", deletedCount,
		"retention_days", retentionDays)
	return int(deletedCount), nil
}

// GetNotificationsByTrackingID retrieves notifications by external tracking ID
func (n *NotificationRepository) GetNotificationsByTrackingID(ctx context.Context, trackingID string) ([]*Notification, error) {
	query := `
		SELECT * FROM notifications 
		WHERE tracking_id = $1 
		ORDER BY created_at DESC`

	var notifications []*Notification
	err := n.db.SelectContext(ctx, &notifications, query, trackingID)
	if err != nil {
		n.logger.Error("Failed to get notifications by tracking ID", 
			"tracking_id", trackingID, 
			"error", err)
		return nil, fmt.Errorf("failed to get notifications by tracking ID: %w", err)
	}

	return notifications, nil
}

// GetDeliveryRate calculates delivery rate for a time period
func (n *NotificationRepository) GetDeliveryRate(ctx context.Context, since time.Time, channel string) (*DeliveryRate, error) {
	var query string
	var args []interface{}

	if channel != "" {
		query = `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered,
				COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
			FROM notifications 
			WHERE created_at >= $1 AND channel = $2`
		args = []interface{}{since, channel}
	} else {
		query = `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered,
				COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
			FROM notifications 
			WHERE created_at >= $1`
		args = []interface{}{since}
	}

	var rate DeliveryRate
	err := n.db.GetContext(ctx, &rate, query, args...)
	if err != nil {
		n.logger.Error("Failed to get delivery rate", "error", err)
		return nil, fmt.Errorf("failed to get delivery rate: %w", err)
	}

	// Calculate percentage
	if rate.Total > 0 {
		rate.DeliveryRate = float64(rate.Delivered) / float64(rate.Total) * 100
		rate.FailureRate = float64(rate.Failed) / float64(rate.Total) * 100
	}

	return &rate, nil
}

// Helper methods

func (n *NotificationRepository) buildWhereClause(filter Filter) (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIndex := 0

	// Alert ID filter
	if alertID, ok := filter.Filters["alert_id"].(string); ok && alertID != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("alert_id = $%d", argIndex))
		args = append(args, alertID)
	}

	// Rule ID filter
	if ruleID, ok := filter.Filters["rule_id"].(string); ok && ruleID != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("rule_id = $%d", argIndex))
		args = append(args, ruleID)
	}

	// Channel filter
	if channel, ok := filter.Filters["channel"].(string); ok && channel != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("channel = $%d", argIndex))
		args = append(args, channel)
	}

	// Status filter
	if status, ok := filter.Filters["status"].(string); ok && status != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
	}

	// Status list filter
	if statuses, ok := filter.Filters["statuses"].([]string); ok && len(statuses) > 0 {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statuses))
	}

	// Recipient filter
	if recipient, ok := filter.Filters["recipient"].(string); ok && recipient != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("recipient ILIKE $%d", argIndex))
		args = append(args, "%"+recipient+"%")
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

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args, argIndex
}

func (n *NotificationRepository) buildOrderClause(filter Filter) string {
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

func (n *NotificationRepository) buildLimitClause(filter Filter, argIndex *int, args *[]interface{}) string {
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

// Additional types for statistics

type NotificationStats struct {
	Channel                  string  `db:"channel"`
	TotalCount              int     `db:"total_count"`
	DeliveredCount          int     `db:"delivered_count"`
	FailedCount             int     `db:"failed_count"`
	PendingCount            int     `db:"pending_count"`
	AvgDeliveryTimeSeconds  *float64 `db:"avg_delivery_time_seconds"`
}

type RuleNotificationStats struct {
	RuleID         string `db:"rule_id"`
	RuleName       string `db:"rule_name"`
	TotalCount     int    `db:"total_count"`
	DeliveredCount int    `db:"delivered_count"`
	FailedCount    int    `db:"failed_count"`
	PendingCount   int    `db:"pending_count"`
}

type DeliveryRate struct {
	Total        int     `db:"total"`
	Delivered    int     `db:"delivered"`
	Failed       int     `db:"failed"`
	DeliveryRate float64 `json:"delivery_rate"`
	FailureRate  float64 `json:"failure_rate"`
}