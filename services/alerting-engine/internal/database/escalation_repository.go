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

// EscalationRepository handles escalation policy data operations
type EscalationRepository struct {
	BaseRepository
	logger *slog.Logger
}

// NewEscalationRepository creates a new escalation repository
func NewEscalationRepository(db *sqlx.DB, logger *slog.Logger) *EscalationRepository {
	return &EscalationRepository{
		BaseRepository: BaseRepository{db: db},
		logger:         logger,
	}
}

// Create creates a new escalation policy
func (e *EscalationRepository) Create(ctx context.Context, policy *EscalationPolicy) error {
	query := `
		INSERT INTO escalation_policies (
			id, name, description, enabled, rules, notification_channels,
			escalation_levels, conditions, tags, metadata, created_by,
			updated_by, version, created_at, updated_at
		) VALUES (
			:id, :name, :description, :enabled, :rules, :notification_channels,
			:escalation_levels, :conditions, :tags, :metadata, :created_by,
			:updated_by, :version, :created_at, :updated_at
		)`

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.Version = 1

	_, err := e.db.NamedExecContext(ctx, query, policy)
	if err != nil {
		e.logger.Error("Failed to create escalation policy", 
			"policy_id", policy.ID, 
			"name", policy.Name,
			"error", err)
		return fmt.Errorf("failed to create escalation policy: %w", err)
	}

	e.logger.Info("Escalation policy created", 
		"policy_id", policy.ID,
		"name", policy.Name)
	return nil
}

// GetByID retrieves an escalation policy by ID
func (e *EscalationRepository) GetByID(ctx context.Context, id string) (*EscalationPolicy, error) {
	query := `
		SELECT * FROM escalation_policies 
		WHERE id = $1 AND deleted_at IS NULL`

	var policy EscalationPolicy
	err := e.db.GetContext(ctx, &policy, query, id)
	if err != nil {
		e.logger.Error("Failed to get escalation policy by ID", "policy_id", id, "error", err)
		return nil, fmt.Errorf("failed to get escalation policy by ID: %w", err)
	}

	return &policy, nil
}

// GetByName retrieves an escalation policy by name
func (e *EscalationRepository) GetByName(ctx context.Context, name string) (*EscalationPolicy, error) {
	query := `
		SELECT * FROM escalation_policies 
		WHERE name = $1 AND deleted_at IS NULL`

	var policy EscalationPolicy
	err := e.db.GetContext(ctx, &policy, query, name)
	if err != nil {
		e.logger.Error("Failed to get escalation policy by name", "name", name, "error", err)
		return nil, fmt.Errorf("failed to get escalation policy by name: %w", err)
	}

	return &policy, nil
}

// Update updates an existing escalation policy
func (e *EscalationRepository) Update(ctx context.Context, policy *EscalationPolicy) error {
	// First, get the current version
	currentPolicy, err := e.GetByID(ctx, policy.ID)
	if err != nil {
		return fmt.Errorf("failed to get current policy: %w", err)
	}

	query := `
		UPDATE escalation_policies SET
			name = :name,
			description = :description,
			enabled = :enabled,
			rules = :rules,
			notification_channels = :notification_channels,
			escalation_levels = :escalation_levels,
			conditions = :conditions,
			tags = :tags,
			metadata = :metadata,
			updated_by = :updated_by,
			version = :version,
			updated_at = :updated_at
		WHERE id = :id AND version = :current_version AND deleted_at IS NULL`

	policy.Version = currentPolicy.Version + 1
	policy.UpdatedAt = time.Now()

	// Use a custom struct to include current version for optimistic locking
	updateData := struct {
		*EscalationPolicy
		CurrentVersion int `db:"current_version"`
	}{
		EscalationPolicy: policy,
		CurrentVersion:   currentPolicy.Version,
	}

	result, err := e.db.NamedExecContext(ctx, query, updateData)
	if err != nil {
		e.logger.Error("Failed to update escalation policy", "policy_id", policy.ID, "error", err)
		return fmt.Errorf("failed to update escalation policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("escalation policy not found or version conflict: %s", policy.ID)
	}

	e.logger.Info("Escalation policy updated", 
		"policy_id", policy.ID, 
		"new_version", policy.Version)
	return nil
}

// List retrieves escalation policies with filtering and pagination
func (e *EscalationRepository) List(ctx context.Context, filter Filter) ([]*EscalationPolicy, int, error) {
	whereClause, args, argIndex := e.buildWhereClause(filter)
	
	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM escalation_policies %s", whereClause)
	var total int
	err := e.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		e.logger.Error("Failed to count escalation policies", "error", err)
		return nil, 0, fmt.Errorf("failed to count escalation policies: %w", err)
	}

	// Data query
	orderClause := e.buildOrderClause(filter)
	limitClause := e.buildLimitClause(filter, &argIndex, &args)
	
	dataQuery := fmt.Sprintf(`
		SELECT * FROM escalation_policies %s %s %s`,
		whereClause, orderClause, limitClause)

	var policies []*EscalationPolicy
	err = e.db.SelectContext(ctx, &policies, dataQuery, args...)
	if err != nil {
		e.logger.Error("Failed to list escalation policies", "error", err)
		return nil, 0, fmt.Errorf("failed to list escalation policies: %w", err)
	}

	return policies, total, nil
}

// ListEnabled retrieves all enabled escalation policies
func (e *EscalationRepository) ListEnabled(ctx context.Context) ([]*EscalationPolicy, error) {
	query := `
		SELECT * FROM escalation_policies 
		WHERE enabled = true AND deleted_at IS NULL
		ORDER BY name ASC`

	var policies []*EscalationPolicy
	err := e.db.SelectContext(ctx, &policies, query)
	if err != nil {
		e.logger.Error("Failed to list enabled escalation policies", "error", err)
		return nil, fmt.Errorf("failed to list enabled escalation policies: %w", err)
	}

	return policies, nil
}

// ListByTags retrieves escalation policies that have any of the specified tags
func (e *EscalationRepository) ListByTags(ctx context.Context, tags []string) ([]*EscalationPolicy, error) {
	query := `
		SELECT * FROM escalation_policies 
		WHERE tags && $1 AND enabled = true AND deleted_at IS NULL
		ORDER BY name ASC`

	var policies []*EscalationPolicy
	err := e.db.SelectContext(ctx, &policies, query, pq.Array(tags))
	if err != nil {
		e.logger.Error("Failed to list escalation policies by tags", "tags", tags, "error", err)
		return nil, fmt.Errorf("failed to list escalation policies by tags: %w", err)
	}

	return policies, nil
}

// Enable enables an escalation policy
func (e *EscalationRepository) Enable(ctx context.Context, id, updatedBy string) error {
	query := `
		UPDATE escalation_policies SET
			enabled = true,
			updated_by = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := e.db.ExecContext(ctx, query, id, updatedBy)
	if err != nil {
		e.logger.Error("Failed to enable escalation policy", "policy_id", id, "error", err)
		return fmt.Errorf("failed to enable escalation policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("escalation policy not found: %s", id)
	}

	e.logger.Info("Escalation policy enabled", "policy_id", id, "updated_by", updatedBy)
	return nil
}

// Disable disables an escalation policy
func (e *EscalationRepository) Disable(ctx context.Context, id, updatedBy string) error {
	query := `
		UPDATE escalation_policies SET
			enabled = false,
			updated_by = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := e.db.ExecContext(ctx, query, id, updatedBy)
	if err != nil {
		e.logger.Error("Failed to disable escalation policy", "policy_id", id, "error", err)
		return fmt.Errorf("failed to disable escalation policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("escalation policy not found: %s", id)
	}

	e.logger.Info("Escalation policy disabled", "policy_id", id, "updated_by", updatedBy)
	return nil
}

// Delete soft deletes an escalation policy
func (e *EscalationRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE escalation_policies SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := e.db.ExecContext(ctx, query, id)
	if err != nil {
		e.logger.Error("Failed to delete escalation policy", "policy_id", id, "error", err)
		return fmt.Errorf("failed to delete escalation policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("escalation policy not found: %s", id)
	}

	e.logger.Info("Escalation policy deleted", "policy_id", id)
	return nil
}

// ValidateName checks if an escalation policy name is available
func (e *EscalationRepository) ValidateName(ctx context.Context, name, excludeID string) error {
	query := `
		SELECT COUNT(*) FROM escalation_policies 
		WHERE name = $1 AND id != $2 AND deleted_at IS NULL`

	var count int
	err := e.db.GetContext(ctx, &count, query, name, excludeID)
	if err != nil {
		return fmt.Errorf("failed to validate escalation policy name: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("escalation policy name already exists: %s", name)
	}

	return nil
}

// GetMatchingPolicies retrieves escalation policies that match given conditions
func (e *EscalationRepository) GetMatchingPolicies(ctx context.Context, alert *Alert) ([]*EscalationPolicy, error) {
	// This is a simplified implementation. In practice, you would have more sophisticated
	// matching logic based on alert properties, rule conditions, etc.
	query := `
		SELECT * FROM escalation_policies 
		WHERE enabled = true 
		AND deleted_at IS NULL
		AND (
			rules = '[]'::jsonb OR 
			rules @> $1::jsonb OR
			conditions @> $2::jsonb
		)
		ORDER BY name ASC`

	// Create simplified matching criteria
	ruleMatch := fmt.Sprintf(`["%s"]`, alert.RuleID)
	conditionMatch := fmt.Sprintf(`{"severity": "%s", "type": "%s"}`, alert.Severity, alert.Type)

	var policies []*EscalationPolicy
	err := e.db.SelectContext(ctx, &policies, query, ruleMatch, conditionMatch)
	if err != nil {
		e.logger.Error("Failed to get matching escalation policies", 
			"alert_id", alert.ID, 
			"error", err)
		return nil, fmt.Errorf("failed to get matching escalation policies: %w", err)
	}

	e.logger.Debug("Found matching escalation policies", 
		"alert_id", alert.ID,
		"policy_count", len(policies))
	return policies, nil
}

// GetPolicyUsageStats retrieves usage statistics for escalation policies
func (e *EscalationRepository) GetPolicyUsageStats(ctx context.Context, since time.Time) ([]*PolicyUsageStats, error) {
	query := `
		SELECT 
			ep.id as policy_id,
			ep.name as policy_name,
			COUNT(a.id) as alert_count,
			COUNT(CASE WHEN a.status = 'escalated' THEN 1 END) as escalated_count,
			COUNT(CASE WHEN a.escalation_level > 0 THEN 1 END) as escalation_triggered_count
		FROM escalation_policies ep
		LEFT JOIN alerts a ON a.escalation_policy_id = ep.id AND a.created_at >= $1
		WHERE ep.deleted_at IS NULL
		GROUP BY ep.id, ep.name
		ORDER BY alert_count DESC`

	var stats []*PolicyUsageStats
	err := e.db.SelectContext(ctx, &stats, query, since)
	if err != nil {
		e.logger.Error("Failed to get policy usage stats", "error", err)
		return nil, fmt.Errorf("failed to get policy usage stats: %w", err)
	}

	return stats, nil
}

// Duplicate creates a copy of an existing escalation policy
func (e *EscalationRepository) Duplicate(ctx context.Context, sourceID, newID, newName, createdBy string) (*EscalationPolicy, error) {
	sourcePolicy, err := e.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source policy: %w", err)
	}

	// Create new policy based on source
	newPolicy := *sourcePolicy
	newPolicy.ID = newID
	newPolicy.Name = newName
	newPolicy.CreatedBy = createdBy
	newPolicy.UpdatedBy = createdBy
	newPolicy.Version = 1
	newPolicy.Enabled = false // Start disabled
	newPolicy.CreatedAt = time.Time{}
	newPolicy.UpdatedAt = time.Time{}

	err = e.Create(ctx, &newPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicated policy: %w", err)
	}

	return &newPolicy, nil
}

// GetEscalationHistory retrieves escalation history for analysis
func (e *EscalationRepository) GetEscalationHistory(ctx context.Context, policyID string, since time.Time) ([]*EscalationEvent, error) {
	query := `
		SELECT 
			a.id as alert_id,
			a.title,
			a.severity,
			a.escalation_level,
			a.escalated_at,
			a.escalated_by,
			a.created_at
		FROM alerts a
		WHERE a.escalation_policy_id = $1 
		AND a.escalated_at >= $2
		AND a.escalation_level > 0
		ORDER BY a.escalated_at DESC`

	var events []*EscalationEvent
	err := e.db.SelectContext(ctx, &events, query, policyID, since)
	if err != nil {
		e.logger.Error("Failed to get escalation history", 
			"policy_id", policyID, 
			"error", err)
		return nil, fmt.Errorf("failed to get escalation history: %w", err)
	}

	return events, nil
}

// Helper methods

func (e *EscalationRepository) buildWhereClause(filter Filter) (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIndex := 0

	// Base condition
	conditions = append(conditions, "deleted_at IS NULL")

	// Enabled filter
	if enabled, ok := filter.Filters["enabled"].(bool); ok {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, enabled)
	}

	// Created by filter
	if createdBy, ok := filter.Filters["created_by"].(string); ok && createdBy != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIndex))
		args = append(args, createdBy)
	}

	// Tags filter
	if tags, ok := filter.Filters["tags"].([]string); ok && len(tags) > 0 {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIndex))
		args = append(args, pq.Array(tags))
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

	// Search filter
	if search, ok := filter.Filters["search"].(string); ok && search != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf(`
			(name ILIKE $%d OR description ILIKE $%d)`, 
			argIndex, argIndex))
		args = append(args, "%"+search+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args, argIndex
}

func (e *EscalationRepository) buildOrderClause(filter Filter) string {
	sortBy := "name"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortOrder := "ASC"
	if filter.SortOrder != "" {
		sortOrder = strings.ToUpper(filter.SortOrder)
	}

	return fmt.Sprintf("ORDER BY %s %s", sortBy, sortOrder)
}

func (e *EscalationRepository) buildLimitClause(filter Filter, argIndex *int, args *[]interface{}) string {
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

// Additional types for escalation analysis

type PolicyUsageStats struct {
	PolicyID                 string `db:"policy_id"`
	PolicyName               string `db:"policy_name"`
	AlertCount               int    `db:"alert_count"`
	EscalatedCount           int    `db:"escalated_count"`
	EscalationTriggeredCount int    `db:"escalation_triggered_count"`
}

type EscalationEvent struct {
	AlertID       string     `db:"alert_id"`
	Title         string     `db:"title"`
	Severity      string     `db:"severity"`
	EscalationLevel int      `db:"escalation_level"`
	EscalatedAt   *time.Time `db:"escalated_at"`
	EscalatedBy   *string    `db:"escalated_by"`
	CreatedAt     time.Time  `db:"created_at"`
}