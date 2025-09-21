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

// RuleRepository handles rule data operations
type RuleRepository struct {
	BaseRepository
	logger *slog.Logger
}

// NewRuleRepository creates a new rule repository
func NewRuleRepository(db *sqlx.DB, logger *slog.Logger) *RuleRepository {
	return &RuleRepository{
		BaseRepository: BaseRepository{db: db},
		logger:         logger,
	}
}

// Create creates a new rule
func (r *RuleRepository) Create(ctx context.Context, rule *Rule) error {
	query := `
		INSERT INTO rules (
			id, name, description, type, severity, priority, enabled,
			conditions, actions, tags, metadata, throttle_window,
			evaluation_window, group_by, notification_channels,
			escalation_policy, created_by, updated_by, version,
			created_at, updated_at
		) VALUES (
			:id, :name, :description, :type, :severity, :priority, :enabled,
			:conditions, :actions, :tags, :metadata, :throttle_window,
			:evaluation_window, :group_by, :notification_channels,
			:escalation_policy, :created_by, :updated_by, :version,
			:created_at, :updated_at
		)`

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.Version = 1

	_, err := r.db.NamedExecContext(ctx, query, rule)
	if err != nil {
		r.logger.Error("Failed to create rule", "rule_id", rule.ID, "error", err)
		return fmt.Errorf("failed to create rule: %w", err)
	}

	r.logger.Info("Rule created", "rule_id", rule.ID, "name", rule.Name)
	return nil
}

// GetByID retrieves a rule by ID
func (r *RuleRepository) GetByID(ctx context.Context, id string) (*Rule, error) {
	query := `
		SELECT * FROM rules 
		WHERE id = $1 AND deleted_at IS NULL`

	var rule Rule
	err := r.db.GetContext(ctx, &rule, query, id)
	if err != nil {
		r.logger.Error("Failed to get rule by ID", "rule_id", id, "error", err)
		return nil, fmt.Errorf("failed to get rule by ID: %w", err)
	}

	return &rule, nil
}

// GetByName retrieves a rule by name
func (r *RuleRepository) GetByName(ctx context.Context, name string) (*Rule, error) {
	query := `
		SELECT * FROM rules 
		WHERE name = $1 AND deleted_at IS NULL`

	var rule Rule
	err := r.db.GetContext(ctx, &rule, query, name)
	if err != nil {
		r.logger.Error("Failed to get rule by name", "name", name, "error", err)
		return nil, fmt.Errorf("failed to get rule by name: %w", err)
	}

	return &rule, nil
}

// Update updates an existing rule
func (r *RuleRepository) Update(ctx context.Context, rule *Rule) error {
	// First, get the current version
	currentRule, err := r.GetByID(ctx, rule.ID)
	if err != nil {
		return fmt.Errorf("failed to get current rule: %w", err)
	}

	query := `
		UPDATE rules SET
			name = :name,
			description = :description,
			type = :type,
			severity = :severity,
			priority = :priority,
			enabled = :enabled,
			conditions = :conditions,
			actions = :actions,
			tags = :tags,
			metadata = :metadata,
			throttle_window = :throttle_window,
			evaluation_window = :evaluation_window,
			group_by = :group_by,
			notification_channels = :notification_channels,
			escalation_policy = :escalation_policy,
			updated_by = :updated_by,
			version = :version,
			updated_at = :updated_at
		WHERE id = :id AND version = :current_version AND deleted_at IS NULL`

	rule.Version = currentRule.Version + 1
	rule.UpdatedAt = time.Now()

	// Use a custom struct to include current version for optimistic locking
	updateData := struct {
		*Rule
		CurrentVersion int `db:"current_version"`
	}{
		Rule:           rule,
		CurrentVersion: currentRule.Version,
	}

	result, err := r.db.NamedExecContext(ctx, query, updateData)
	if err != nil {
		r.logger.Error("Failed to update rule", "rule_id", rule.ID, "error", err)
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found or version conflict: %s", rule.ID)
	}

	r.logger.Info("Rule updated", "rule_id", rule.ID, "new_version", rule.Version)
	return nil
}

// List retrieves rules with filtering and pagination
func (r *RuleRepository) List(ctx context.Context, filter Filter) ([]*Rule, int, error) {
	whereClause, args, argIndex := r.buildWhereClause(filter)
	
	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM rules %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		r.logger.Error("Failed to count rules", "error", err)
		return nil, 0, fmt.Errorf("failed to count rules: %w", err)
	}

	// Data query
	orderClause := r.buildOrderClause(filter)
	limitClause := r.buildLimitClause(filter, &argIndex, &args)
	
	dataQuery := fmt.Sprintf(`
		SELECT * FROM rules %s %s %s`,
		whereClause, orderClause, limitClause)

	var rules []*Rule
	err = r.db.SelectContext(ctx, &rules, dataQuery, args...)
	if err != nil {
		r.logger.Error("Failed to list rules", "error", err)
		return nil, 0, fmt.Errorf("failed to list rules: %w", err)
	}

	return rules, total, nil
}

// ListEnabled retrieves all enabled rules
func (r *RuleRepository) ListEnabled(ctx context.Context) ([]*Rule, error) {
	query := `
		SELECT * FROM rules 
		WHERE enabled = true AND deleted_at IS NULL
		ORDER BY priority DESC, name ASC`

	var rules []*Rule
	err := r.db.SelectContext(ctx, &rules, query)
	if err != nil {
		r.logger.Error("Failed to list enabled rules", "error", err)
		return nil, fmt.Errorf("failed to list enabled rules: %w", err)
	}

	return rules, nil
}

// ListByType retrieves rules by type
func (r *RuleRepository) ListByType(ctx context.Context, ruleType string) ([]*Rule, error) {
	query := `
		SELECT * FROM rules 
		WHERE type = $1 AND enabled = true AND deleted_at IS NULL
		ORDER BY priority DESC, name ASC`

	var rules []*Rule
	err := r.db.SelectContext(ctx, &rules, query, ruleType)
	if err != nil {
		r.logger.Error("Failed to list rules by type", "type", ruleType, "error", err)
		return nil, fmt.Errorf("failed to list rules by type: %w", err)
	}

	return rules, nil
}

// ListByTags retrieves rules that have any of the specified tags
func (r *RuleRepository) ListByTags(ctx context.Context, tags []string) ([]*Rule, error) {
	query := `
		SELECT * FROM rules 
		WHERE tags && $1 AND enabled = true AND deleted_at IS NULL
		ORDER BY priority DESC, name ASC`

	var rules []*Rule
	err := r.db.SelectContext(ctx, &rules, query, pq.Array(tags))
	if err != nil {
		r.logger.Error("Failed to list rules by tags", "tags", tags, "error", err)
		return nil, fmt.Errorf("failed to list rules by tags: %w", err)
	}

	return rules, nil
}

// Enable enables a rule
func (r *RuleRepository) Enable(ctx context.Context, id, updatedBy string) error {
	query := `
		UPDATE rules SET
			enabled = true,
			updated_by = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, updatedBy)
	if err != nil {
		r.logger.Error("Failed to enable rule", "rule_id", id, "error", err)
		return fmt.Errorf("failed to enable rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	r.logger.Info("Rule enabled", "rule_id", id, "updated_by", updatedBy)
	return nil
}

// Disable disables a rule
func (r *RuleRepository) Disable(ctx context.Context, id, updatedBy string) error {
	query := `
		UPDATE rules SET
			enabled = false,
			updated_by = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, updatedBy)
	if err != nil {
		r.logger.Error("Failed to disable rule", "rule_id", id, "error", err)
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	r.logger.Info("Rule disabled", "rule_id", id, "updated_by", updatedBy)
	return nil
}

// Delete soft deletes a rule
func (r *RuleRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE rules SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete rule", "rule_id", id, "error", err)
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	r.logger.Info("Rule deleted", "rule_id", id)
	return nil
}

// GetVersion retrieves a specific version of a rule
func (r *RuleRepository) GetVersion(ctx context.Context, id string, version int) (*Rule, error) {
	// In a full implementation, this would require a rule_versions table
	// For now, we'll just return the current version if it matches
	rule, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if rule.Version != version {
		return nil, fmt.Errorf("rule version %d not found", version)
	}

	return rule, nil
}

// GetVersionHistory retrieves version history for a rule
func (r *RuleRepository) GetVersionHistory(ctx context.Context, id string) ([]*Rule, error) {
	// In a full implementation, this would query a rule_versions table
	// For now, we'll just return the current version
	rule, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return []*Rule{rule}, nil
}

// Duplicate creates a copy of an existing rule
func (r *RuleRepository) Duplicate(ctx context.Context, sourceID, newID, newName, createdBy string) (*Rule, error) {
	sourceRule, err := r.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source rule: %w", err)
	}

	// Create new rule based on source
	newRule := *sourceRule
	newRule.ID = newID
	newRule.Name = newName
	newRule.CreatedBy = createdBy
	newRule.UpdatedBy = createdBy
	newRule.Version = 1
	newRule.Enabled = false // Start disabled
	newRule.CreatedAt = time.Time{}
	newRule.UpdatedAt = time.Time{}

	err = r.Create(ctx, &newRule)
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicated rule: %w", err)
	}

	return &newRule, nil
}

// ValidateName checks if a rule name is available
func (r *RuleRepository) ValidateName(ctx context.Context, name, excludeID string) error {
	query := `
		SELECT COUNT(*) FROM rules 
		WHERE name = $1 AND id != $2 AND deleted_at IS NULL`

	var count int
	err := r.db.GetContext(ctx, &count, query, name, excludeID)
	if err != nil {
		return fmt.Errorf("failed to validate rule name: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("rule name already exists: %s", name)
	}

	return nil
}

// Helper methods

func (r *RuleRepository) buildWhereClause(filter Filter) (string, []interface{}, int) {
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

	// Type filter
	if ruleType, ok := filter.Filters["type"].(string); ok && ruleType != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, ruleType)
	}

	// Severity filter
	if severity, ok := filter.Filters["severity"].(string); ok && severity != "" {
		argIndex++
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIndex))
		args = append(args, severity)
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

func (r *RuleRepository) buildOrderClause(filter Filter) string {
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

func (r *RuleRepository) buildLimitClause(filter Filter, argIndex *int, args *[]interface{}) string {
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