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
	"go.uber.org/zap"

	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/models"
)

// InvestigationRepository handles investigation-related database operations
type InvestigationRepository struct {
	*database.Repository
}

// NewInvestigationRepository creates a new investigation repository
func NewInvestigationRepository(db *database.Database, logger *zap.Logger) *InvestigationRepository {
	return &InvestigationRepository{
		Repository: database.NewRepository(db, logger),
	}
}

// Create creates a new investigation
func (r *InvestigationRepository) Create(ctx context.Context, req *models.CreateInvestigationRequest, createdBy uuid.UUID) (*models.Investigation, error) {
	investigation := &models.Investigation{
		ID:             uuid.New(),
		Title:          req.Title,
		Description:    req.Description,
		CaseType:       req.CaseType,
		Priority:       req.Priority,
		Status:         models.StatusOpen,
		AssignedTo:     req.AssignedTo,
		CreatedBy:      createdBy,
		ExternalCaseID: req.ExternalCaseID,
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		DueDate:        req.DueDate,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `
		INSERT INTO investigations (
			id, title, description, case_type, priority, status, assigned_to, 
			created_by, external_case_id, tags, metadata, due_date, created_at, updated_at
		) VALUES (
			:id, :title, :description, :case_type, :priority, :status, :assigned_to,
			:created_by, :external_case_id, :tags, :metadata, :due_date, :created_at, :updated_at
		) RETURNING id, created_at, updated_at`

	rows, err := r.DB().NamedQueryContext(ctx, query, investigation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create investigation")
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&investigation.ID, &investigation.CreatedAt, &investigation.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan created investigation")
		}
	}

	return investigation, nil
}

// GetByID retrieves an investigation by ID
func (r *InvestigationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Investigation, error) {
	var investigation models.Investigation
	
	query := `
		SELECT id, title, description, case_type, priority, status, assigned_to,
			   created_by, external_case_id, tags, metadata, created_at, updated_at,
			   due_date, closed_at, archived_at
		FROM investigations 
		WHERE id = $1`

	err := r.DB().GetContext(ctx, &investigation, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("investigation not found")
		}
		return nil, errors.Wrap(err, "failed to get investigation")
	}

	return &investigation, nil
}

// Update updates an investigation
func (r *InvestigationRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateInvestigationRequest) (*models.Investigation, error) {
	// Build dynamic update query
	setParts := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := map[string]interface{}{"id": id}
	argIndex := 1

	if req.Title != nil {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex+1))
		args["title"] = *req.Title
		argIndex++
	}
	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex+1))
		args["description"] = *req.Description
		argIndex++
	}
	if req.CaseType != nil {
		setParts = append(setParts, fmt.Sprintf("case_type = $%d", argIndex+1))
		args["case_type"] = *req.CaseType
		argIndex++
	}
	if req.Priority != nil {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex+1))
		args["priority"] = *req.Priority
		argIndex++
	}
	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex+1))
		args["status"] = *req.Status
		argIndex++
		
		// Handle status-specific logic
		if *req.Status == models.StatusClosed {
			setParts = append(setParts, "closed_at = CURRENT_TIMESTAMP")
		} else if *req.Status == models.StatusArchived {
			setParts = append(setParts, "archived_at = CURRENT_TIMESTAMP")
		}
	}
	if req.AssignedTo != nil {
		setParts = append(setParts, fmt.Sprintf("assigned_to = $%d", argIndex+1))
		args["assigned_to"] = *req.AssignedTo
		argIndex++
	}
	if req.ExternalCaseID != nil {
		setParts = append(setParts, fmt.Sprintf("external_case_id = $%d", argIndex+1))
		args["external_case_id"] = *req.ExternalCaseID
		argIndex++
	}
	if req.Tags != nil {
		setParts = append(setParts, fmt.Sprintf("tags = $%d", argIndex+1))
		args["tags"] = req.Tags
		argIndex++
	}
	if req.Metadata != nil {
		setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex+1))
		args["metadata"] = req.Metadata
		argIndex++
	}
	if req.DueDate != nil {
		setParts = append(setParts, fmt.Sprintf("due_date = $%d", argIndex+1))
		args["due_date"] = *req.DueDate
		argIndex++
	}

	if len(setParts) == 1 { // Only updated_at
		return r.GetByID(ctx, id)
	}

	query := fmt.Sprintf(`
		UPDATE investigations 
		SET %s
		WHERE id = $1
		RETURNING id, title, description, case_type, priority, status, assigned_to,
				  created_by, external_case_id, tags, metadata, created_at, updated_at,
				  due_date, closed_at, archived_at`,
		strings.Join(setParts, ", "))

	var investigation models.Investigation
	err := r.DB().GetContext(ctx, &investigation, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("investigation not found")
		}
		return nil, errors.Wrap(err, "failed to update investigation")
	}

	return &investigation, nil
}

// Delete soft deletes an investigation
func (r *InvestigationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE investigations 
		SET status = 'archived', archived_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	result, err := r.DB().ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete investigation")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("investigation not found")
	}

	return nil
}

// List retrieves investigations with filtering and pagination
func (r *InvestigationRepository) List(ctx context.Context, filter *models.InvestigationFilter, paginate *database.Paginate) (*database.PaginatedResult, error) {
	whereConditions := []string{"1=1"}
	args := make(map[string]interface{})
	argIndex := 0

	// Build where conditions
	if filter != nil {
		if len(filter.CaseTypes) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("case_type = ANY($%d)", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = filter.CaseTypes
		}
		if len(filter.Priorities) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("priority = ANY($%d)", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = filter.Priorities
		}
		if len(filter.Statuses) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("status = ANY($%d)", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = filter.Statuses
		}
		if filter.AssignedTo != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("assigned_to = $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = *filter.AssignedTo
		}
		if filter.CreatedBy != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("created_by = $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = *filter.CreatedBy
		}
		if filter.CreatedAfter != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = *filter.CreatedAfter
		}
		if filter.CreatedBefore != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("created_at <= $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = *filter.CreatedBefore
		}
		if filter.DueAfter != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("due_date >= $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = *filter.DueAfter
		}
		if filter.DueBefore != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("due_date <= $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = *filter.DueBefore
		}
		if len(filter.Tags) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("tags && $%d", argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = filter.Tags
		}
		if filter.Search != nil && *filter.Search != "" {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
			args[fmt.Sprintf("arg%d", argIndex)] = "%" + *filter.Search + "%"
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM investigations WHERE %s", whereClause)
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get investigations count")
	}

	// Get data
	dataQuery := fmt.Sprintf(`
		SELECT id, title, description, case_type, priority, status, assigned_to,
			   created_by, external_case_id, tags, metadata, created_at, updated_at,
			   due_date, closed_at, archived_at
		FROM investigations 
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex+1, argIndex+2)

	args[fmt.Sprintf("arg%d", argIndex+1)] = paginate.Limit
	args[fmt.Sprintf("arg%d", argIndex+2)] = paginate.Offset

	var investigations []models.Investigation
	err = r.DB().SelectContext(ctx, &investigations, dataQuery, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get investigations")
	}

	return database.NewPaginatedResult(investigations, total, paginate), nil
}

// GetByExternalCaseID retrieves an investigation by external case ID
func (r *InvestigationRepository) GetByExternalCaseID(ctx context.Context, externalCaseID string) (*models.Investigation, error) {
	var investigation models.Investigation
	
	query := `
		SELECT id, title, description, case_type, priority, status, assigned_to,
			   created_by, external_case_id, tags, metadata, created_at, updated_at,
			   due_date, closed_at, archived_at
		FROM investigations 
		WHERE external_case_id = $1`

	err := r.DB().GetContext(ctx, &investigation, query, externalCaseID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("investigation not found")
		}
		return nil, errors.Wrap(err, "failed to get investigation by external case ID")
	}

	return &investigation, nil
}

// GetAssignedInvestigations retrieves investigations assigned to a specific user
func (r *InvestigationRepository) GetAssignedInvestigations(ctx context.Context, userID uuid.UUID, paginate *database.Paginate) (*database.PaginatedResult, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM investigations WHERE assigned_to = $1 AND status NOT IN ('closed', 'archived')"
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get assigned investigations count")
	}

	// Get data
	dataQuery := `
		SELECT id, title, description, case_type, priority, status, assigned_to,
			   created_by, external_case_id, tags, metadata, created_at, updated_at,
			   due_date, closed_at, archived_at
		FROM investigations 
		WHERE assigned_to = $1 AND status NOT IN ('closed', 'archived')
		ORDER BY priority DESC, due_date ASC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3`

	var investigations []models.Investigation
	err = r.DB().SelectContext(ctx, &investigations, dataQuery, userID, paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get assigned investigations")
	}

	return database.NewPaginatedResult(investigations, total, paginate), nil
}

// GetInvestigationStats retrieves investigation statistics
func (r *InvestigationRepository) GetInvestigationStats(ctx context.Context, userID *uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Base query conditions
	baseConditions := "1=1"
	args := []interface{}{}
	argIndex := 0

	if userID != nil {
		argIndex++
		baseConditions += fmt.Sprintf(" AND assigned_to = $%d", argIndex)
		args = append(args, *userID)
	}

	// Total investigations
	query := fmt.Sprintf("SELECT COUNT(*) FROM investigations WHERE %s", baseConditions)
	var total int64
	err := r.DB().GetContext(ctx, &total, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total investigations")
	}
	stats["total"] = total

	// By status
	query = fmt.Sprintf(`
		SELECT status, COUNT(*) 
		FROM investigations 
		WHERE %s 
		GROUP BY status`, baseConditions)
	
	rows, err := r.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get investigations by status")
	}
	defer rows.Close()

	statusStats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, errors.Wrap(err, "failed to scan status stats")
		}
		statusStats[status] = count
	}
	stats["by_status"] = statusStats

	// By priority
	query = fmt.Sprintf(`
		SELECT priority, COUNT(*) 
		FROM investigations 
		WHERE %s AND status NOT IN ('closed', 'archived')
		GROUP BY priority`, baseConditions)
	
	rows, err = r.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get investigations by priority")
	}
	defer rows.Close()

	priorityStats := make(map[string]int64)
	for rows.Next() {
		var priority string
		var count int64
		if err := rows.Scan(&priority, &count); err != nil {
			return nil, errors.Wrap(err, "failed to scan priority stats")
		}
		priorityStats[priority] = count
	}
	stats["by_priority"] = priorityStats

	// Overdue investigations
	query = fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM investigations 
		WHERE %s AND due_date < CURRENT_TIMESTAMP AND status NOT IN ('closed', 'archived')`, baseConditions)
	
	var overdue int64
	err = r.DB().GetContext(ctx, &overdue, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get overdue investigations")
	}
	stats["overdue"] = overdue

	return stats, nil
}

// UpdateAssignment updates the assignment of an investigation
func (r *InvestigationRepository) UpdateAssignment(ctx context.Context, id uuid.UUID, assignedTo *uuid.UUID) error {
	query := `
		UPDATE investigations 
		SET assigned_to = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	result, err := r.DB().ExecContext(ctx, query, assignedTo, id)
	if err != nil {
		return errors.Wrap(err, "failed to update investigation assignment")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("investigation not found")
	}

	return nil
}

// UpdateStatus updates the status of an investigation
func (r *InvestigationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.Status) error {
	setParts := []string{"status = $1", "updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{status, id}

	// Handle status-specific logic
	if status == models.StatusClosed {
		setParts = append(setParts, "closed_at = CURRENT_TIMESTAMP")
	} else if status == models.StatusArchived {
		setParts = append(setParts, "archived_at = CURRENT_TIMESTAMP")
	}

	query := fmt.Sprintf(`
		UPDATE investigations 
		SET %s
		WHERE id = $2`,
		strings.Join(setParts, ", "))

	result, err := r.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to update investigation status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("investigation not found")
	}

	return nil
}