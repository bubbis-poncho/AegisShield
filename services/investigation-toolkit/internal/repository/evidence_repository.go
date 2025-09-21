package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/models"
)

// EvidenceRepository handles evidence-related database operations
type EvidenceRepository struct {
	*database.Repository
}

// NewEvidenceRepository creates a new evidence repository
func NewEvidenceRepository(db *database.Database, logger *zap.Logger) *EvidenceRepository {
	return &EvidenceRepository{
		Repository: database.NewRepository(db, logger),
	}
}

// Create creates new evidence
func (r *EvidenceRepository) Create(ctx context.Context, investigationID uuid.UUID, req *models.CreateEvidenceRequest, collectedBy uuid.UUID) (*models.Evidence, error) {
	evidence := &models.Evidence{
		ID:                   uuid.New(),
		InvestigationID:      investigationID,
		Name:                 req.Name,
		Description:          req.Description,
		EvidenceType:         req.EvidenceType,
		Source:               req.Source,
		CollectionMethod:     req.CollectionMethod,
		CollectedBy:          collectedBy,
		CollectedAt:          time.Now(),
		ChainOfCustody:       models.JSONB{},
		Metadata:             req.Metadata,
		Tags:                 req.Tags,
		IsAuthenticated:      false,
		AuthenticationMethod: req.AuthenticationMethod,
		RetentionDate:        req.RetentionDate,
		Status:               models.EvidenceStatusActive,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	// Initialize chain of custody
	custodyEntry := map[string]interface{}{
		"action":      "collected",
		"user_id":     collectedBy,
		"timestamp":   evidence.CollectedAt,
		"location":    nil,
		"notes":       "Evidence collected and added to investigation",
	}
	evidence.ChainOfCustody = models.JSONB{
		"entries": []interface{}{custodyEntry},
	}

	query := `
		INSERT INTO evidence (
			id, investigation_id, name, description, evidence_type, source, collection_method,
			collected_by, collected_at, chain_of_custody, metadata, tags, is_authenticated,
			authentication_method, retention_date, status, created_at, updated_at
		) VALUES (
			:id, :investigation_id, :name, :description, :evidence_type, :source, :collection_method,
			:collected_by, :collected_at, :chain_of_custody, :metadata, :tags, :is_authenticated,
			:authentication_method, :retention_date, :status, :created_at, :updated_at
		) RETURNING id, created_at, updated_at`

	rows, err := r.DB().NamedQueryContext(ctx, query, evidence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create evidence")
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&evidence.ID, &evidence.CreatedAt, &evidence.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan created evidence")
		}
	}

	return evidence, nil
}

// GetByID retrieves evidence by ID
func (r *EvidenceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Evidence, error) {
	var evidence models.Evidence
	
	query := `
		SELECT id, investigation_id, name, description, evidence_type, source, collection_method,
			   file_path, file_size, file_hash, mime_type, collected_by, collected_at,
			   chain_of_custody, metadata, tags, is_authenticated, authentication_method,
			   authentication_date, authentication_by, retention_date, status, created_at, updated_at
		FROM evidence 
		WHERE id = $1`

	err := r.DB().GetContext(ctx, &evidence, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("evidence not found")
		}
		return nil, errors.Wrap(err, "failed to get evidence")
	}

	return &evidence, nil
}

// GetByInvestigationID retrieves all evidence for an investigation
func (r *EvidenceRepository) GetByInvestigationID(ctx context.Context, investigationID uuid.UUID, filter *models.EvidenceFilter, paginate *database.Paginate) (*database.PaginatedResult, error) {
	whereConditions := []string{"investigation_id = $1"}
	args := []interface{}{investigationID}
	argIndex := 1

	// Build where conditions
	if filter != nil {
		if len(filter.EvidenceTypes) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("evidence_type = ANY($%d)", argIndex))
			args = append(args, filter.EvidenceTypes)
		}
		if len(filter.Statuses) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("status = ANY($%d)", argIndex))
			args = append(args, filter.Statuses)
		}
		if filter.CollectedBy != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("collected_by = $%d", argIndex))
			args = append(args, *filter.CollectedBy)
		}
		if filter.CollectedAfter != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("collected_at >= $%d", argIndex))
			args = append(args, *filter.CollectedAfter)
		}
		if filter.CollectedBefore != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("collected_at <= $%d", argIndex))
			args = append(args, *filter.CollectedBefore)
		}
		if filter.IsAuthenticated != nil {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("is_authenticated = $%d", argIndex))
			args = append(args, *filter.IsAuthenticated)
		}
		if len(filter.Tags) > 0 {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("tags && $%d", argIndex))
			args = append(args, filter.Tags)
		}
		if filter.Search != nil && *filter.Search != "" {
			argIndex++
			whereConditions = append(whereConditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
			args = append(args, "%"+*filter.Search+"%")
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM evidence WHERE %s", whereClause)
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get evidence count")
	}

	// Get data
	dataQuery := fmt.Sprintf(`
		SELECT id, investigation_id, name, description, evidence_type, source, collection_method,
			   file_path, file_size, file_hash, mime_type, collected_by, collected_at,
			   chain_of_custody, metadata, tags, is_authenticated, authentication_method,
			   authentication_date, authentication_by, retention_date, status, created_at, updated_at
		FROM evidence 
		WHERE %s
		ORDER BY collected_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex+1, argIndex+2)

	args = append(args, paginate.Limit, paginate.Offset)

	var evidenceList []models.Evidence
	err = r.DB().SelectContext(ctx, &evidenceList, dataQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get evidence")
	}

	return database.NewPaginatedResult(evidenceList, total, paginate), nil
}

// UpdateFile updates file information for evidence
func (r *EvidenceRepository) UpdateFile(ctx context.Context, id uuid.UUID, filePath, fileHash, mimeType string, fileSize int64) error {
	query := `
		UPDATE evidence 
		SET file_path = $1, file_hash = $2, mime_type = $3, file_size = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5`

	result, err := r.DB().ExecContext(ctx, query, filePath, fileHash, mimeType, fileSize, id)
	if err != nil {
		return errors.Wrap(err, "failed to update evidence file")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("evidence not found")
	}

	return nil
}

// UpdateChainOfCustody adds an entry to the chain of custody
func (r *EvidenceRepository) UpdateChainOfCustody(ctx context.Context, id uuid.UUID, userID uuid.UUID, action, location, notes string) error {
	// First get the current chain of custody
	var currentChain models.JSONB
	query := "SELECT chain_of_custody FROM evidence WHERE id = $1"
	err := r.DB().GetContext(ctx, &currentChain, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to get current chain of custody")
	}

	// Add new entry
	newEntry := map[string]interface{}{
		"action":    action,
		"user_id":   userID,
		"timestamp": time.Now(),
		"location":  location,
		"notes":     notes,
	}

	entries, ok := currentChain["entries"].([]interface{})
	if !ok {
		entries = []interface{}{}
	}
	entries = append(entries, newEntry)
	currentChain["entries"] = entries

	// Update the chain of custody
	updateQuery := `
		UPDATE evidence 
		SET chain_of_custody = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	result, err := r.DB().ExecContext(ctx, updateQuery, currentChain, id)
	if err != nil {
		return errors.Wrap(err, "failed to update chain of custody")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("evidence not found")
	}

	return nil
}

// Authenticate marks evidence as authenticated
func (r *EvidenceRepository) Authenticate(ctx context.Context, id uuid.UUID, authenticatedBy uuid.UUID, method string) error {
	query := `
		UPDATE evidence 
		SET is_authenticated = true, authentication_method = $1, authentication_date = CURRENT_TIMESTAMP,
			authentication_by = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3`

	result, err := r.DB().ExecContext(ctx, query, method, authenticatedBy, id)
	if err != nil {
		return errors.Wrap(err, "failed to authenticate evidence")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("evidence not found")
	}

	// Add chain of custody entry
	return r.UpdateChainOfCustody(ctx, id, authenticatedBy, "authenticated", "", fmt.Sprintf("Evidence authenticated using method: %s", method))
}

// UpdateStatus updates the status of evidence
func (r *EvidenceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.EvidenceStatus, userID uuid.UUID, reason string) error {
	query := `
		UPDATE evidence 
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	result, err := r.DB().ExecContext(ctx, query, status, id)
	if err != nil {
		return errors.Wrap(err, "failed to update evidence status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("evidence not found")
	}

	// Add chain of custody entry
	action := fmt.Sprintf("status_changed_to_%s", status)
	return r.UpdateChainOfCustody(ctx, id, userID, action, "", reason)
}

// Delete soft deletes evidence
func (r *EvidenceRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, reason string) error {
	err := r.UpdateStatus(ctx, id, models.EvidenceStatusArchived, userID, reason)
	if err != nil {
		return errors.Wrap(err, "failed to delete evidence")
	}

	return nil
}

// GetByFileHash retrieves evidence by file hash to check for duplicates
func (r *EvidenceRepository) GetByFileHash(ctx context.Context, fileHash string) ([]models.Evidence, error) {
	var evidenceList []models.Evidence
	
	query := `
		SELECT id, investigation_id, name, description, evidence_type, source, collection_method,
			   file_path, file_size, file_hash, mime_type, collected_by, collected_at,
			   chain_of_custody, metadata, tags, is_authenticated, authentication_method,
			   authentication_date, authentication_by, retention_date, status, created_at, updated_at
		FROM evidence 
		WHERE file_hash = $1 AND status != 'archived'`

	err := r.DB().SelectContext(ctx, &evidenceList, query, fileHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get evidence by file hash")
	}

	return evidenceList, nil
}

// GetEvidenceStats retrieves evidence statistics for an investigation
func (r *EvidenceRepository) GetEvidenceStats(ctx context.Context, investigationID uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total evidence
	query := "SELECT COUNT(*) FROM evidence WHERE investigation_id = $1 AND status != 'archived'"
	var total int64
	err := r.DB().GetContext(ctx, &total, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total evidence")
	}
	stats["total"] = total

	// By type
	query = `
		SELECT evidence_type, COUNT(*) 
		FROM evidence 
		WHERE investigation_id = $1 AND status != 'archived'
		GROUP BY evidence_type`
	
	rows, err := r.DB().QueryContext(ctx, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get evidence by type")
	}
	defer rows.Close()

	typeStats := make(map[string]int64)
	for rows.Next() {
		var evidenceType string
		var count int64
		if err := rows.Scan(&evidenceType, &count); err != nil {
			return nil, errors.Wrap(err, "failed to scan type stats")
		}
		typeStats[evidenceType] = count
	}
	stats["by_type"] = typeStats

	// By status
	query = `
		SELECT status, COUNT(*) 
		FROM evidence 
		WHERE investigation_id = $1
		GROUP BY status`
	
	rows, err = r.DB().QueryContext(ctx, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get evidence by status")
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

	// Authentication stats
	query = "SELECT COUNT(*) FROM evidence WHERE investigation_id = $1 AND is_authenticated = true AND status != 'archived'"
	var authenticated int64
	err = r.DB().GetContext(ctx, &authenticated, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get authenticated evidence count")
	}
	stats["authenticated"] = authenticated

	// File size stats
	query = "SELECT COALESCE(SUM(file_size), 0) FROM evidence WHERE investigation_id = $1 AND file_size IS NOT NULL AND status != 'archived'"
	var totalSize int64
	err = r.DB().GetContext(ctx, &totalSize, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total file size")
	}
	stats["total_file_size"] = totalSize

	return stats, nil
}

// GetExpiredEvidence retrieves evidence that has passed its retention date
func (r *EvidenceRepository) GetExpiredEvidence(ctx context.Context, paginate *database.Paginate) (*database.PaginatedResult, error) {
	whereClause := "retention_date < CURRENT_TIMESTAMP AND status != 'archived'"

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM evidence WHERE %s", whereClause)
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get expired evidence count")
	}

	// Get data
	dataQuery := fmt.Sprintf(`
		SELECT id, investigation_id, name, description, evidence_type, source, collection_method,
			   file_path, file_size, file_hash, mime_type, collected_by, collected_at,
			   chain_of_custody, metadata, tags, is_authenticated, authentication_method,
			   authentication_date, authentication_by, retention_date, status, created_at, updated_at
		FROM evidence 
		WHERE %s
		ORDER BY retention_date ASC
		LIMIT $1 OFFSET $2`,
		whereClause)

	var evidenceList []models.Evidence
	err = r.DB().SelectContext(ctx, &evidenceList, dataQuery, paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get expired evidence")
	}

	return database.NewPaginatedResult(evidenceList, total, paginate), nil
}

// BulkUpdateStatus updates the status of multiple evidence items
func (r *EvidenceRepository) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status models.EvidenceStatus, userID uuid.UUID, reason string) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		// Update status
		query := `
			UPDATE evidence 
			SET status = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = ANY($2)`

		_, err := tx.ExecContext(ctx, query, status, ids)
		if err != nil {
			return errors.Wrap(err, "failed to bulk update evidence status")
		}

		// Add chain of custody entries
		for _, id := range ids {
			err = r.UpdateChainOfCustody(ctx, id, userID, fmt.Sprintf("status_changed_to_%s", status), "", reason)
			if err != nil {
				return errors.Wrapf(err, "failed to update chain of custody for evidence %s", id)
			}
		}

		return nil
	})
}