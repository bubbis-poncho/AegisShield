package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"investigation-toolkit/internal/models"
)

type WorkflowRepository interface {
	// Workflow Templates
	CreateTemplate(ctx context.Context, template *models.WorkflowTemplate) error
	GetTemplate(ctx context.Context, id uuid.UUID) (*models.WorkflowTemplate, error)
	UpdateTemplate(ctx context.Context, template *models.WorkflowTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	ListTemplates(ctx context.Context, filter models.WorkflowTemplateFilter) ([]*models.WorkflowTemplate, int, error)
	
	// Workflow Instances
	CreateInstance(ctx context.Context, instance *models.WorkflowInstance) error
	GetInstance(ctx context.Context, id uuid.UUID) (*models.WorkflowInstance, error)
	UpdateInstance(ctx context.Context, instance *models.WorkflowInstance) error
	DeleteInstance(ctx context.Context, id uuid.UUID) error
	ListInstances(ctx context.Context, filter models.WorkflowInstanceFilter) ([]*models.WorkflowInstance, int, error)
	GetInstancesByInvestigation(ctx context.Context, investigationID uuid.UUID) ([]*models.WorkflowInstance, error)
	
	// Workflow Steps
	CreateStep(ctx context.Context, step *models.WorkflowStep) error
	GetStep(ctx context.Context, id uuid.UUID) (*models.WorkflowStep, error)
	UpdateStep(ctx context.Context, step *models.WorkflowStep) error
	DeleteStep(ctx context.Context, id uuid.UUID) error
	ListSteps(ctx context.Context, instanceID uuid.UUID) ([]*models.WorkflowStep, error)
	
	// Step Execution
	StartStep(ctx context.Context, stepID uuid.UUID, assignedTo uuid.UUID) error
	CompleteStep(ctx context.Context, stepID uuid.UUID, userID uuid.UUID, result string, outputs map[string]interface{}) error
	SkipStep(ctx context.Context, stepID uuid.UUID, userID uuid.UUID, reason string) error
	
	// Workflow Statistics
	GetWorkflowStats(ctx context.Context, filter models.WorkflowStatsFilter) (*models.WorkflowStats, error)
	GetStepStats(ctx context.Context, templateID uuid.UUID) ([]*models.StepStats, error)
	
	// Workflow Automation
	GetPendingSteps(ctx context.Context, userID *uuid.UUID) ([]*models.WorkflowStep, error)
	GetOverdueSteps(ctx context.Context) ([]*models.WorkflowStep, error)
	GetInstancesReadyForNextStep(ctx context.Context) ([]*models.WorkflowInstance, error)
}

type workflowRepository struct {
	db *sqlx.DB
}

func NewWorkflowRepository(db *sqlx.DB) WorkflowRepository {
	return &workflowRepository{db: db}
}

// Workflow Templates
func (r *workflowRepository) CreateTemplate(ctx context.Context, template *models.WorkflowTemplate) error {
	query := `
		INSERT INTO workflow_templates (
			id, name, description, version, category, is_active, steps, 
			created_by, created_at, updated_at
		) VALUES (
			:id, :name, :description, :version, :category, :is_active, :steps,
			:created_by, :created_at, :updated_at
		)`
	
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, template)
	if err != nil {
		return errors.Wrap(err, "failed to create workflow template")
	}
	
	return nil
}

func (r *workflowRepository) GetTemplate(ctx context.Context, id uuid.UUID) (*models.WorkflowTemplate, error) {
	var template models.WorkflowTemplate
	query := `
		SELECT id, name, description, version, category, is_active, steps,
			   created_by, created_at, updated_at
		FROM workflow_templates
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &template, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("workflow template not found")
		}
		return nil, errors.Wrap(err, "failed to get workflow template")
	}
	
	return &template, nil
}

func (r *workflowRepository) UpdateTemplate(ctx context.Context, template *models.WorkflowTemplate) error {
	query := `
		UPDATE workflow_templates
		SET name = :name, description = :description, version = :version,
			category = :category, is_active = :is_active, steps = :steps,
			updated_at = :updated_at
		WHERE id = :id`
	
	template.UpdatedAt = time.Now()
	
	result, err := r.db.NamedExecContext(ctx, query, template)
	if err != nil {
		return errors.Wrap(err, "failed to update workflow template")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow template not found")
	}
	
	return nil
}

func (r *workflowRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workflow_templates WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete workflow template")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow template not found")
	}
	
	return nil
}

func (r *workflowRepository) ListTemplates(ctx context.Context, filter models.WorkflowTemplateFilter) ([]*models.WorkflowTemplate, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM workflow_templates
		WHERE 1=1`
	
	if filter.Category != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("category = $%d", argCount))
		args = append(args, filter.Category)
	}
	
	if filter.IsActive != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argCount))
		args = append(args, *filter.IsActive)
	}
	
	if filter.CreatedBy != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argCount))
		args = append(args, *filter.CreatedBy)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count workflow templates")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, name, description, version, category, is_active, steps,
			   created_by, created_at, updated_at ` +
		baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var templates []*models.WorkflowTemplate
	err = r.db.SelectContext(ctx, &templates, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list workflow templates")
	}
	
	return templates, total, nil
}

// Workflow Instances
func (r *workflowRepository) CreateInstance(ctx context.Context, instance *models.WorkflowInstance) error {
	query := `
		INSERT INTO workflow_instances (
			id, template_id, investigation_id, name, status, current_step_index,
			context, created_by, started_at, created_at, updated_at
		) VALUES (
			:id, :template_id, :investigation_id, :name, :status, :current_step_index,
			:context, :created_by, :started_at, :created_at, :updated_at
		)`
	
	instance.ID = uuid.New()
	instance.CreatedAt = time.Now()
	instance.UpdatedAt = time.Now()
	if instance.StartedAt == nil && instance.Status == models.WorkflowStatusInProgress {
		now := time.Now()
		instance.StartedAt = &now
	}
	
	_, err := r.db.NamedExecContext(ctx, query, instance)
	if err != nil {
		return errors.Wrap(err, "failed to create workflow instance")
	}
	
	return nil
}

func (r *workflowRepository) GetInstance(ctx context.Context, id uuid.UUID) (*models.WorkflowInstance, error) {
	var instance models.WorkflowInstance
	query := `
		SELECT id, template_id, investigation_id, name, status, current_step_index,
			   context, created_by, started_at, completed_at, created_at, updated_at
		FROM workflow_instances
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &instance, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("workflow instance not found")
		}
		return nil, errors.Wrap(err, "failed to get workflow instance")
	}
	
	return &instance, nil
}

func (r *workflowRepository) UpdateInstance(ctx context.Context, instance *models.WorkflowInstance) error {
	query := `
		UPDATE workflow_instances
		SET status = :status, current_step_index = :current_step_index,
			context = :context, started_at = :started_at, completed_at = :completed_at,
			updated_at = :updated_at
		WHERE id = :id`
	
	instance.UpdatedAt = time.Now()
	
	result, err := r.db.NamedExecContext(ctx, query, instance)
	if err != nil {
		return errors.Wrap(err, "failed to update workflow instance")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow instance not found")
	}
	
	return nil
}

func (r *workflowRepository) DeleteInstance(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workflow_instances WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete workflow instance")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow instance not found")
	}
	
	return nil
}

func (r *workflowRepository) ListInstances(ctx context.Context, filter models.WorkflowInstanceFilter) ([]*models.WorkflowInstance, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM workflow_instances
		WHERE 1=1`
	
	if filter.TemplateID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("template_id = $%d", argCount))
		args = append(args, *filter.TemplateID)
	}
	
	if filter.InvestigationID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("investigation_id = $%d", argCount))
		args = append(args, *filter.InvestigationID)
	}
	
	if filter.Status != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, filter.Status)
	}
	
	if filter.CreatedBy != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argCount))
		args = append(args, *filter.CreatedBy)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count workflow instances")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, template_id, investigation_id, name, status, current_step_index,
			   context, created_by, started_at, completed_at, created_at, updated_at ` +
		baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var instances []*models.WorkflowInstance
	err = r.db.SelectContext(ctx, &instances, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list workflow instances")
	}
	
	return instances, total, nil
}

func (r *workflowRepository) GetInstancesByInvestigation(ctx context.Context, investigationID uuid.UUID) ([]*models.WorkflowInstance, error) {
	query := `
		SELECT id, template_id, investigation_id, name, status, current_step_index,
			   context, created_by, started_at, completed_at, created_at, updated_at
		FROM workflow_instances
		WHERE investigation_id = $1
		ORDER BY created_at ASC`
	
	var instances []*models.WorkflowInstance
	err := r.db.SelectContext(ctx, &instances, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow instances by investigation")
	}
	
	return instances, nil
}

// Workflow Steps
func (r *workflowRepository) CreateStep(ctx context.Context, step *models.WorkflowStep) error {
	query := `
		INSERT INTO workflow_steps (
			id, instance_id, name, description, step_index, status, step_type,
			config, inputs, outputs, assigned_to, due_date, created_at, updated_at
		) VALUES (
			:id, :instance_id, :name, :description, :step_index, :status, :step_type,
			:config, :inputs, :outputs, :assigned_to, :due_date, :created_at, :updated_at
		)`
	
	step.ID = uuid.New()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, step)
	if err != nil {
		return errors.Wrap(err, "failed to create workflow step")
	}
	
	return nil
}

func (r *workflowRepository) GetStep(ctx context.Context, id uuid.UUID) (*models.WorkflowStep, error) {
	var step models.WorkflowStep
	query := `
		SELECT id, instance_id, name, description, step_index, status, step_type,
			   config, inputs, outputs, assigned_to, due_date, started_at, completed_at,
			   created_at, updated_at
		FROM workflow_steps
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &step, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("workflow step not found")
		}
		return nil, errors.Wrap(err, "failed to get workflow step")
	}
	
	return &step, nil
}

func (r *workflowRepository) UpdateStep(ctx context.Context, step *models.WorkflowStep) error {
	query := `
		UPDATE workflow_steps
		SET status = :status, inputs = :inputs, outputs = :outputs,
			assigned_to = :assigned_to, due_date = :due_date,
			started_at = :started_at, completed_at = :completed_at,
			updated_at = :updated_at
		WHERE id = :id`
	
	step.UpdatedAt = time.Now()
	
	result, err := r.db.NamedExecContext(ctx, query, step)
	if err != nil {
		return errors.Wrap(err, "failed to update workflow step")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow step not found")
	}
	
	return nil
}

func (r *workflowRepository) DeleteStep(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workflow_steps WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete workflow step")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow step not found")
	}
	
	return nil
}

func (r *workflowRepository) ListSteps(ctx context.Context, instanceID uuid.UUID) ([]*models.WorkflowStep, error) {
	query := `
		SELECT id, instance_id, name, description, step_index, status, step_type,
			   config, inputs, outputs, assigned_to, due_date, started_at, completed_at,
			   created_at, updated_at
		FROM workflow_steps
		WHERE instance_id = $1
		ORDER BY step_index ASC`
	
	var steps []*models.WorkflowStep
	err := r.db.SelectContext(ctx, &steps, query, instanceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list workflow steps")
	}
	
	return steps, nil
}

// Step Execution
func (r *workflowRepository) StartStep(ctx context.Context, stepID uuid.UUID, assignedTo uuid.UUID) error {
	query := `
		UPDATE workflow_steps
		SET status = $1, assigned_to = $2, started_at = $3, updated_at = $4
		WHERE id = $5 AND status = $6`
	
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, models.StepStatusInProgress, assignedTo, now, now, stepID, models.StepStatusPending)
	if err != nil {
		return errors.Wrap(err, "failed to start workflow step")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("workflow step not found or not in pending status")
	}
	
	return nil
}

func (r *workflowRepository) CompleteStep(ctx context.Context, stepID uuid.UUID, userID uuid.UUID, result string, outputs map[string]interface{}) error {
	query := `
		UPDATE workflow_steps
		SET status = $1, outputs = $2, completed_at = $3, updated_at = $4
		WHERE id = $5 AND status = $6 AND (assigned_to = $7 OR assigned_to IS NULL)`
	
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, models.StepStatusCompleted, outputs, now, now, stepID, models.StepStatusInProgress, userID)
	if err != nil {
		return errors.Wrap(err, "failed to complete workflow step")
	}
	
	return nil
}

func (r *workflowRepository) SkipStep(ctx context.Context, stepID uuid.UUID, userID uuid.UUID, reason string) error {
	query := `
		UPDATE workflow_steps
		SET status = $1, outputs = $2, completed_at = $3, updated_at = $4
		WHERE id = $5 AND (assigned_to = $6 OR assigned_to IS NULL)`
	
	now := time.Now()
	outputs := map[string]interface{}{
		"skipped": true,
		"reason":  reason,
	}
	
	_, err := r.db.ExecContext(ctx, query, models.StepStatusSkipped, outputs, now, now, stepID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to skip workflow step")
	}
	
	return nil
}

// Workflow Statistics
func (r *workflowRepository) GetWorkflowStats(ctx context.Context, filter models.WorkflowStatsFilter) (*models.WorkflowStats, error) {
	var stats models.WorkflowStats
	
	baseQuery := `
		FROM workflow_instances wi
		JOIN workflow_templates wt ON wi.template_id = wt.id
		WHERE 1=1`
	
	var conditions []string
	var args []interface{}
	argCount := 0
	
	if filter.TemplateID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("wi.template_id = $%d", argCount))
		args = append(args, *filter.TemplateID)
	}
	
	if filter.CreatedBy != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("wi.created_by = $%d", argCount))
		args = append(args, *filter.CreatedBy)
	}
	
	if !filter.DateFrom.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("wi.created_at >= $%d", argCount))
		args = append(args, filter.DateFrom)
	}
	
	if !filter.DateTo.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("wi.created_at <= $%d", argCount))
		args = append(args, filter.DateTo)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Get basic counts
	query := `
		SELECT 
			COUNT(*) as total_instances,
			COUNT(CASE WHEN wi.status = 'completed' THEN 1 END) as completed_instances,
			COUNT(CASE WHEN wi.status = 'in_progress' THEN 1 END) as in_progress_instances,
			COUNT(CASE WHEN wi.status = 'failed' THEN 1 END) as failed_instances,
			AVG(CASE WHEN wi.status = 'completed' AND wi.completed_at IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (wi.completed_at - wi.started_at))/3600 END) as avg_completion_hours
		` + baseQuery
	
	err := r.db.GetContext(ctx, &stats, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow stats")
	}
	
	return &stats, nil
}

func (r *workflowRepository) GetStepStats(ctx context.Context, templateID uuid.UUID) ([]*models.StepStats, error) {
	query := `
		SELECT 
			ws.step_index,
			ws.name as step_name,
			COUNT(*) as total_executions,
			COUNT(CASE WHEN ws.status = 'completed' THEN 1 END) as completed_count,
			COUNT(CASE WHEN ws.status = 'failed' THEN 1 END) as failed_count,
			COUNT(CASE WHEN ws.status = 'skipped' THEN 1 END) as skipped_count,
			AVG(CASE WHEN ws.status = 'completed' AND ws.completed_at IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (ws.completed_at - ws.started_at))/3600 END) as avg_duration_hours
		FROM workflow_steps ws
		JOIN workflow_instances wi ON ws.instance_id = wi.id
		WHERE wi.template_id = $1
		GROUP BY ws.step_index, ws.name
		ORDER BY ws.step_index`
	
	var stats []*models.StepStats
	err := r.db.SelectContext(ctx, &stats, query, templateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get step stats")
	}
	
	return stats, nil
}

// Workflow Automation
func (r *workflowRepository) GetPendingSteps(ctx context.Context, userID *uuid.UUID) ([]*models.WorkflowStep, error) {
	var query string
	var args []interface{}
	
	if userID != nil {
		query = `
			SELECT ws.id, ws.instance_id, ws.name, ws.description, ws.step_index, 
				   ws.status, ws.step_type, ws.config, ws.inputs, ws.outputs, 
				   ws.assigned_to, ws.due_date, ws.started_at, ws.completed_at,
				   ws.created_at, ws.updated_at
			FROM workflow_steps ws
			JOIN workflow_instances wi ON ws.instance_id = wi.id
			WHERE ws.status = $1 AND (ws.assigned_to = $2 OR ws.assigned_to IS NULL)
				  AND wi.status = $3
			ORDER BY ws.due_date ASC NULLS LAST, ws.created_at ASC`
		args = []interface{}{models.StepStatusPending, *userID, models.WorkflowStatusInProgress}
	} else {
		query = `
			SELECT ws.id, ws.instance_id, ws.name, ws.description, ws.step_index, 
				   ws.status, ws.step_type, ws.config, ws.inputs, ws.outputs, 
				   ws.assigned_to, ws.due_date, ws.started_at, ws.completed_at,
				   ws.created_at, ws.updated_at
			FROM workflow_steps ws
			JOIN workflow_instances wi ON ws.instance_id = wi.id
			WHERE ws.status = $1 AND wi.status = $2
			ORDER BY ws.due_date ASC NULLS LAST, ws.created_at ASC`
		args = []interface{}{models.StepStatusPending, models.WorkflowStatusInProgress}
	}
	
	var steps []*models.WorkflowStep
	err := r.db.SelectContext(ctx, &steps, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pending steps")
	}
	
	return steps, nil
}

func (r *workflowRepository) GetOverdueSteps(ctx context.Context) ([]*models.WorkflowStep, error) {
	query := `
		SELECT ws.id, ws.instance_id, ws.name, ws.description, ws.step_index, 
			   ws.status, ws.step_type, ws.config, ws.inputs, ws.outputs, 
			   ws.assigned_to, ws.due_date, ws.started_at, ws.completed_at,
			   ws.created_at, ws.updated_at
		FROM workflow_steps ws
		JOIN workflow_instances wi ON ws.instance_id = wi.id
		WHERE ws.status IN ($1, $2) AND ws.due_date < $3 AND wi.status = $4
		ORDER BY ws.due_date ASC`
	
	var steps []*models.WorkflowStep
	err := r.db.SelectContext(ctx, &steps, query, 
		models.StepStatusPending, models.StepStatusInProgress, time.Now(), models.WorkflowStatusInProgress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get overdue steps")
	}
	
	return steps, nil
}

func (r *workflowRepository) GetInstancesReadyForNextStep(ctx context.Context) ([]*models.WorkflowInstance, error) {
	query := `
		SELECT DISTINCT wi.id, wi.template_id, wi.investigation_id, wi.name, wi.status, 
			   wi.current_step_index, wi.context, wi.created_by, wi.started_at, 
			   wi.completed_at, wi.created_at, wi.updated_at
		FROM workflow_instances wi
		WHERE wi.status = $1
		  AND NOT EXISTS (
			  SELECT 1 FROM workflow_steps ws 
			  WHERE ws.instance_id = wi.id 
			    AND ws.step_index = wi.current_step_index
			    AND ws.status IN ($2, $3)
		  )
		ORDER BY wi.created_at ASC`
	
	var instances []*models.WorkflowInstance
	err := r.db.SelectContext(ctx, &instances, query, 
		models.WorkflowStatusInProgress, models.StepStatusPending, models.StepStatusInProgress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instances ready for next step")
	}
	
	return instances, nil
}