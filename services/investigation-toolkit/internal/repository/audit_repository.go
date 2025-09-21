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

type AuditRepository interface {
	// Audit Log Management
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	GetAuditLog(ctx context.Context, id uuid.UUID) (*models.AuditLog, error)
	ListAuditLogs(ctx context.Context, filter models.AuditLogFilter) ([]*models.AuditLog, int, error)
	GetAuditLogsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.AuditLog, error)
	GetAuditLogsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error)
	GetAuditLogsByAction(ctx context.Context, action string, limit int) ([]*models.AuditLog, error)
	
	// Compliance and Chain of Custody
	CreateChainOfCustodyEntry(ctx context.Context, entry *models.ChainOfCustodyEntry) error
	GetChainOfCustody(ctx context.Context, evidenceID uuid.UUID) ([]*models.ChainOfCustodyEntry, error)
	VerifyChainOfCustody(ctx context.Context, evidenceID uuid.UUID) (*models.ChainOfCustodyVerification, error)
	
	// Access Control Audit
	LogUserAccess(ctx context.Context, userID uuid.UUID, resource string, action string, metadata map[string]interface{}) error
	GetUserAccessLogs(ctx context.Context, userID uuid.UUID, dateFrom, dateTo time.Time) ([]*models.UserAccessLog, error)
	GetResourceAccessLogs(ctx context.Context, resource string, dateFrom, dateTo time.Time) ([]*models.ResourceAccessLog, error)
	
	// Data Integrity
	CreateDataIntegrityCheck(ctx context.Context, check *models.DataIntegrityCheck) error
	GetDataIntegrityChecks(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DataIntegrityCheck, error)
	VerifyDataIntegrity(ctx context.Context, entityType string, entityID uuid.UUID) (*models.DataIntegrityResult, error)
	
	// Compliance Reports
	GenerateComplianceReport(ctx context.Context, filter models.ComplianceReportFilter) (*models.ComplianceReport, error)
	GetAuditSummary(ctx context.Context, filter models.AuditSummaryFilter) (*models.AuditSummary, error)
	GetUserActivitySummary(ctx context.Context, userID uuid.UUID, dateFrom, dateTo time.Time) (*models.UserActivitySummary, error)
	
	// Retention and Archival
	ArchiveOldAuditLogs(ctx context.Context, retentionPeriod time.Duration) (int64, error)
	GetAuditLogRetentionStats(ctx context.Context) (*models.AuditLogRetentionStats, error)
	PurgeArchivedLogs(ctx context.Context, archivalDate time.Time) (int64, error)
}

type auditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) AuditRepository {
	return &auditRepository{db: db}
}

// Audit Log Management
func (r *auditRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, user_id, action, entity_type, entity_id, description,
			old_values, new_values, metadata, ip_address, user_agent,
			session_id, created_at
		) VALUES (
			:id, :user_id, :action, :entity_type, :entity_id, :description,
			:old_values, :new_values, :metadata, :ip_address, :user_agent,
			:session_id, :created_at
		)`
	
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, log)
	if err != nil {
		return errors.Wrap(err, "failed to create audit log")
	}
	
	return nil
}

func (r *auditRepository) GetAuditLog(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var log models.AuditLog
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   old_values, new_values, metadata, ip_address, user_agent,
			   session_id, created_at
		FROM audit_logs
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &log, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("audit log not found")
		}
		return nil, errors.Wrap(err, "failed to get audit log")
	}
	
	return &log, nil
}

func (r *auditRepository) ListAuditLogs(ctx context.Context, filter models.AuditLogFilter) ([]*models.AuditLog, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM audit_logs
		WHERE 1=1`
	
	if filter.UserID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argCount))
		args = append(args, *filter.UserID)
	}
	
	if filter.Action != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("action = $%d", argCount))
		args = append(args, filter.Action)
	}
	
	if filter.EntityType != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argCount))
		args = append(args, filter.EntityType)
	}
	
	if filter.EntityID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argCount))
		args = append(args, *filter.EntityID)
	}
	
	if !filter.DateFrom.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argCount))
		args = append(args, filter.DateFrom)
	}
	
	if !filter.DateTo.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argCount))
		args = append(args, filter.DateTo)
	}
	
	if filter.IPAddress != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("ip_address = $%d", argCount))
		args = append(args, filter.IPAddress)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count audit logs")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   old_values, new_values, metadata, ip_address, user_agent,
			   session_id, created_at ` +
		baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var logs []*models.AuditLog
	err = r.db.SelectContext(ctx, &logs, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list audit logs")
	}
	
	return logs, total, nil
}

func (r *auditRepository) GetAuditLogsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.AuditLog, error) {
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   old_values, new_values, metadata, ip_address, user_agent,
			   session_id, created_at
		FROM audit_logs
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT 1000`
	
	var logs []*models.AuditLog
	err := r.db.SelectContext(ctx, &logs, query, entityType, entityID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit logs by entity")
	}
	
	return logs, nil
}

func (r *auditRepository) GetAuditLogsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   old_values, new_values, metadata, ip_address, user_agent,
			   session_id, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var logs []*models.AuditLog
	err := r.db.SelectContext(ctx, &logs, query, userID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit logs by user")
	}
	
	return logs, nil
}

func (r *auditRepository) GetAuditLogsByAction(ctx context.Context, action string, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   old_values, new_values, metadata, ip_address, user_agent,
			   session_id, created_at
		FROM audit_logs
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var logs []*models.AuditLog
	err := r.db.SelectContext(ctx, &logs, query, action, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit logs by action")
	}
	
	return logs, nil
}

// Chain of Custody
func (r *auditRepository) CreateChainOfCustodyEntry(ctx context.Context, entry *models.ChainOfCustodyEntry) error {
	query := `
		INSERT INTO chain_of_custody (
			id, evidence_id, user_id, action, location, description,
			hash_before, hash_after, metadata, created_at
		) VALUES (
			:id, :evidence_id, :user_id, :action, :location, :description,
			:hash_before, :hash_after, :metadata, :created_at
		)`
	
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, entry)
	if err != nil {
		return errors.Wrap(err, "failed to create chain of custody entry")
	}
	
	return nil
}

func (r *auditRepository) GetChainOfCustody(ctx context.Context, evidenceID uuid.UUID) ([]*models.ChainOfCustodyEntry, error) {
	query := `
		SELECT id, evidence_id, user_id, action, location, description,
			   hash_before, hash_after, metadata, created_at
		FROM chain_of_custody
		WHERE evidence_id = $1
		ORDER BY created_at ASC`
	
	var entries []*models.ChainOfCustodyEntry
	err := r.db.SelectContext(ctx, &entries, query, evidenceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chain of custody")
	}
	
	return entries, nil
}

func (r *auditRepository) VerifyChainOfCustody(ctx context.Context, evidenceID uuid.UUID) (*models.ChainOfCustodyVerification, error) {
	entries, err := r.GetChainOfCustody(ctx, evidenceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chain of custody for verification")
	}
	
	verification := &models.ChainOfCustodyVerification{
		EvidenceID:      evidenceID,
		IsValid:         true,
		TotalEntries:    len(entries),
		VerifiedAt:      time.Now(),
		ValidationErrors: []string{},
	}
	
	// Verify hash chain integrity
	for i, entry := range entries {
		if i == 0 {
			// First entry should have a hash_before
			if entry.HashBefore == "" {
				verification.IsValid = false
				verification.ValidationErrors = append(verification.ValidationErrors, 
					fmt.Sprintf("First entry missing initial hash at %s", entry.CreatedAt.Format(time.RFC3339)))
			}
		} else {
			// Subsequent entries should have matching hashes
			prevEntry := entries[i-1]
			if entry.HashBefore != prevEntry.HashAfter {
				verification.IsValid = false
				verification.ValidationErrors = append(verification.ValidationErrors, 
					fmt.Sprintf("Hash mismatch between entries at %s and %s", 
						prevEntry.CreatedAt.Format(time.RFC3339), 
						entry.CreatedAt.Format(time.RFC3339)))
			}
		}
		
		// Check for required fields
		if entry.UserID == uuid.Nil {
			verification.IsValid = false
			verification.ValidationErrors = append(verification.ValidationErrors, 
				fmt.Sprintf("Missing user ID in entry at %s", entry.CreatedAt.Format(time.RFC3339)))
		}
		
		if entry.Action == "" {
			verification.IsValid = false
			verification.ValidationErrors = append(verification.ValidationErrors, 
				fmt.Sprintf("Missing action in entry at %s", entry.CreatedAt.Format(time.RFC3339)))
		}
	}
	
	return verification, nil
}

// Access Control Audit
func (r *auditRepository) LogUserAccess(ctx context.Context, userID uuid.UUID, resource string, action string, metadata map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:      &userID,
		Action:      fmt.Sprintf("access_%s", action),
		EntityType:  "access_control",
		Description: fmt.Sprintf("User accessed %s with action %s", resource, action),
		Metadata:    metadata,
	}
	
	if metadata != nil {
		if ip, exists := metadata["ip_address"]; exists {
			if ipStr, ok := ip.(string); ok {
				log.IPAddress = &ipStr
			}
		}
		if ua, exists := metadata["user_agent"]; exists {
			if uaStr, ok := ua.(string); ok {
				log.UserAgent = &uaStr
			}
		}
		if sid, exists := metadata["session_id"]; exists {
			if sidStr, ok := sid.(string); ok {
				log.SessionID = &sidStr
			}
		}
	}
	
	return r.CreateAuditLog(ctx, log)
}

func (r *auditRepository) GetUserAccessLogs(ctx context.Context, userID uuid.UUID, dateFrom, dateTo time.Time) ([]*models.UserAccessLog, error) {
	query := `
		SELECT 
			id,
			user_id,
			action,
			description,
			ip_address,
			user_agent,
			session_id,
			metadata,
			created_at
		FROM audit_logs
		WHERE user_id = $1 
		  AND action LIKE 'access_%'
		  AND created_at >= $2 
		  AND created_at <= $3
		ORDER BY created_at DESC`
	
	var logs []*models.UserAccessLog
	err := r.db.SelectContext(ctx, &logs, query, userID, dateFrom, dateTo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user access logs")
	}
	
	return logs, nil
}

func (r *auditRepository) GetResourceAccessLogs(ctx context.Context, resource string, dateFrom, dateTo time.Time) ([]*models.ResourceAccessLog, error) {
	query := `
		SELECT 
			id,
			user_id,
			action,
			description,
			ip_address,
			user_agent,
			metadata,
			created_at
		FROM audit_logs
		WHERE action LIKE 'access_%'
		  AND description LIKE $1
		  AND created_at >= $2 
		  AND created_at <= $3
		ORDER BY created_at DESC`
	
	var logs []*models.ResourceAccessLog
	err := r.db.SelectContext(ctx, &logs, query, "%"+resource+"%", dateFrom, dateTo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resource access logs")
	}
	
	return logs, nil
}

// Data Integrity
func (r *auditRepository) CreateDataIntegrityCheck(ctx context.Context, check *models.DataIntegrityCheck) error {
	query := `
		INSERT INTO data_integrity_checks (
			id, entity_type, entity_id, check_type, hash_algorithm,
			expected_hash, actual_hash, is_valid, metadata, checked_at
		) VALUES (
			:id, :entity_type, :entity_id, :check_type, :hash_algorithm,
			:expected_hash, :actual_hash, :is_valid, :metadata, :checked_at
		)`
	
	check.ID = uuid.New()
	check.CheckedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, check)
	if err != nil {
		return errors.Wrap(err, "failed to create data integrity check")
	}
	
	return nil
}

func (r *auditRepository) GetDataIntegrityChecks(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DataIntegrityCheck, error) {
	query := `
		SELECT id, entity_type, entity_id, check_type, hash_algorithm,
			   expected_hash, actual_hash, is_valid, metadata, checked_at
		FROM data_integrity_checks
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY checked_at DESC`
	
	var checks []*models.DataIntegrityCheck
	err := r.db.SelectContext(ctx, &checks, query, entityType, entityID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get data integrity checks")
	}
	
	return checks, nil
}

func (r *auditRepository) VerifyDataIntegrity(ctx context.Context, entityType string, entityID uuid.UUID) (*models.DataIntegrityResult, error) {
	checks, err := r.GetDataIntegrityChecks(ctx, entityType, entityID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get data integrity checks for verification")
	}
	
	result := &models.DataIntegrityResult{
		EntityType:        entityType,
		EntityID:          entityID,
		IsValid:           true,
		TotalChecks:       len(checks),
		FailedChecks:      0,
		LastVerifiedAt:    time.Now(),
		ValidationErrors:  []string{},
	}
	
	if len(checks) == 0 {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, "No integrity checks found")
		return result, nil
	}
	
	// Check the most recent verification of each check type
	checkTypes := make(map[string]*models.DataIntegrityCheck)
	for _, check := range checks {
		if existing, exists := checkTypes[check.CheckType]; !exists || check.CheckedAt.After(existing.CheckedAt) {
			checkTypes[check.CheckType] = check
		}
	}
	
	for checkType, check := range checkTypes {
		if !check.IsValid {
			result.IsValid = false
			result.FailedChecks++
			result.ValidationErrors = append(result.ValidationErrors, 
				fmt.Sprintf("%s integrity check failed: expected %s, got %s", 
					checkType, check.ExpectedHash, check.ActualHash))
		}
	}
	
	return result, nil
}

// Compliance Reports
func (r *auditRepository) GenerateComplianceReport(ctx context.Context, filter models.ComplianceReportFilter) (*models.ComplianceReport, error) {
	report := &models.ComplianceReport{
		GeneratedAt: time.Now(),
		DateFrom:    filter.DateFrom,
		DateTo:      filter.DateTo,
		Sections:    make(map[string]interface{}),
	}
	
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseCondition := "WHERE 1=1"
	
	if !filter.DateFrom.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argCount))
		args = append(args, filter.DateFrom)
	}
	
	if !filter.DateTo.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argCount))
		args = append(args, filter.DateTo)
	}
	
	if len(conditions) > 0 {
		baseCondition += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Audit Activity Summary
	auditQuery := `
		SELECT 
			COUNT(*) as total_logs,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT entity_type) as entity_types,
			COUNT(CASE WHEN action LIKE 'access_%' THEN 1 END) as access_logs,
			COUNT(CASE WHEN action LIKE 'create_%' THEN 1 END) as create_logs,
			COUNT(CASE WHEN action LIKE 'update_%' THEN 1 END) as update_logs,
			COUNT(CASE WHEN action LIKE 'delete_%' THEN 1 END) as delete_logs
		FROM audit_logs ` + baseCondition
	
	var auditSummary struct {
		TotalLogs    int `db:"total_logs"`
		UniqueUsers  int `db:"unique_users"`
		EntityTypes  int `db:"entity_types"`
		AccessLogs   int `db:"access_logs"`
		CreateLogs   int `db:"create_logs"`
		UpdateLogs   int `db:"update_logs"`
		DeleteLogs   int `db:"delete_logs"`
	}
	
	err := r.db.GetContext(ctx, &auditSummary, auditQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit summary for compliance report")
	}
	
	report.Sections["audit_summary"] = auditSummary
	
	// Data Integrity Summary
	integrityQuery := `
		SELECT 
			COUNT(*) as total_checks,
			COUNT(CASE WHEN is_valid = true THEN 1 END) as valid_checks,
			COUNT(CASE WHEN is_valid = false THEN 1 END) as failed_checks,
			COUNT(DISTINCT entity_type) as checked_entity_types
		FROM data_integrity_checks ` + baseCondition
	
	var integritySummary struct {
		TotalChecks        int `db:"total_checks"`
		ValidChecks        int `db:"valid_checks"`
		FailedChecks       int `db:"failed_checks"`
		CheckedEntityTypes int `db:"checked_entity_types"`
	}
	
	err = r.db.GetContext(ctx, &integritySummary, integrityQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get integrity summary for compliance report")
	}
	
	report.Sections["integrity_summary"] = integritySummary
	
	// Chain of Custody Summary
	custodyQuery := `
		SELECT 
			COUNT(DISTINCT evidence_id) as evidence_items_tracked,
			COUNT(*) as total_custody_entries,
			COUNT(DISTINCT user_id) as users_handling_evidence
		FROM chain_of_custody ` + baseCondition
	
	var custodySummary struct {
		EvidenceItemsTracked   int `db:"evidence_items_tracked"`
		TotalCustodyEntries    int `db:"total_custody_entries"`
		UsersHandlingEvidence int `db:"users_handling_evidence"`
	}
	
	err = r.db.GetContext(ctx, &custodySummary, custodyQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custody summary for compliance report")
	}
	
	report.Sections["custody_summary"] = custodySummary
	
	return report, nil
}

func (r *auditRepository) GetAuditSummary(ctx context.Context, filter models.AuditSummaryFilter) (*models.AuditSummary, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseCondition := "WHERE 1=1"
	
	if !filter.DateFrom.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argCount))
		args = append(args, filter.DateFrom)
	}
	
	if !filter.DateTo.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argCount))
		args = append(args, filter.DateTo)
	}
	
	if filter.UserID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argCount))
		args = append(args, *filter.UserID)
	}
	
	if filter.EntityType != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argCount))
		args = append(args, filter.EntityType)
	}
	
	if len(conditions) > 0 {
		baseCondition += " AND " + strings.Join(conditions, " AND ")
	}
	
	query := `
		SELECT 
			COUNT(*) as total_entries,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT entity_type) as entity_types,
			COUNT(DISTINCT DATE(created_at)) as active_days,
			COUNT(CASE WHEN action LIKE 'create_%' THEN 1 END) as create_actions,
			COUNT(CASE WHEN action LIKE 'update_%' THEN 1 END) as update_actions,
			COUNT(CASE WHEN action LIKE 'delete_%' THEN 1 END) as delete_actions,
			COUNT(CASE WHEN action LIKE 'access_%' THEN 1 END) as access_actions
		FROM audit_logs ` + baseCondition
	
	var summary models.AuditSummary
	err := r.db.GetContext(ctx, &summary, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit summary")
	}
	
	summary.DateFrom = filter.DateFrom
	summary.DateTo = filter.DateTo
	summary.GeneratedAt = time.Now()
	
	return &summary, nil
}

func (r *auditRepository) GetUserActivitySummary(ctx context.Context, userID uuid.UUID, dateFrom, dateTo time.Time) (*models.UserActivitySummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_actions,
			COUNT(CASE WHEN action LIKE 'create_%' THEN 1 END) as create_actions,
			COUNT(CASE WHEN action LIKE 'update_%' THEN 1 END) as update_actions,
			COUNT(CASE WHEN action LIKE 'delete_%' THEN 1 END) as delete_actions,
			COUNT(CASE WHEN action LIKE 'access_%' THEN 1 END) as access_actions,
			COUNT(DISTINCT entity_type) as entity_types_accessed,
			COUNT(DISTINCT DATE(created_at)) as active_days,
			COUNT(DISTINCT ip_address) as unique_ip_addresses
		FROM audit_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3`
	
	var summary models.UserActivitySummary
	err := r.db.GetContext(ctx, &summary, query, userID, dateFrom, dateTo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user activity summary")
	}
	
	summary.UserID = userID
	summary.DateFrom = dateFrom
	summary.DateTo = dateTo
	summary.GeneratedAt = time.Now()
	
	// Get most recent activity
	recentQuery := `
		SELECT created_at, action, entity_type, description
		FROM audit_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
		LIMIT 10`
	
	var recentActivities []struct {
		CreatedAt   time.Time `db:"created_at"`
		Action      string    `db:"action"`
		EntityType  string    `db:"entity_type"`
		Description string    `db:"description"`
	}
	
	err = r.db.SelectContext(ctx, &recentActivities, recentQuery, userID, dateFrom, dateTo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get recent activities")
	}
	
	for _, activity := range recentActivities {
		summary.RecentActivities = append(summary.RecentActivities, models.RecentActivity{
			Timestamp:   activity.CreatedAt,
			Action:      activity.Action,
			EntityType:  activity.EntityType,
			Description: activity.Description,
		})
	}
	
	return &summary, nil
}

// Retention and Archival
func (r *auditRepository) ArchiveOldAuditLogs(ctx context.Context, retentionPeriod time.Duration) (int64, error) {
	cutoffDate := time.Now().Add(-retentionPeriod)
	
	// Move old logs to archive table
	query := `
		INSERT INTO audit_logs_archive 
		SELECT * FROM audit_logs 
		WHERE created_at < $1`
	
	result, err := r.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return 0, errors.Wrap(err, "failed to archive audit logs")
	}
	
	archivedCount, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get archived rows count")
	}
	
	// Delete from main table
	deleteQuery := `DELETE FROM audit_logs WHERE created_at < $1`
	_, err = r.db.ExecContext(ctx, deleteQuery, cutoffDate)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete archived audit logs from main table")
	}
	
	return archivedCount, nil
}

func (r *auditRepository) GetAuditLogRetentionStats(ctx context.Context) (*models.AuditLogRetentionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_logs,
			MIN(created_at) as oldest_log,
			MAX(created_at) as newest_log,
			COUNT(CASE WHEN created_at < NOW() - INTERVAL '1 year' THEN 1 END) as logs_older_than_year,
			COUNT(CASE WHEN created_at < NOW() - INTERVAL '7 years' THEN 1 END) as logs_older_than_seven_years,
			pg_size_pretty(pg_total_relation_size('audit_logs')) as table_size
		FROM audit_logs`
	
	var stats models.AuditLogRetentionStats
	err := r.db.GetContext(ctx, &stats, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit log retention stats")
	}
	
	// Get archive stats if archive table exists
	archiveQuery := `
		SELECT 
			COUNT(*) as archived_logs,
			MIN(created_at) as oldest_archived_log,
			MAX(created_at) as newest_archived_log,
			pg_size_pretty(pg_total_relation_size('audit_logs_archive')) as archive_table_size
		FROM audit_logs_archive`
	
	var archiveStats struct {
		ArchivedLogs        int        `db:"archived_logs"`
		OldestArchivedLog   *time.Time `db:"oldest_archived_log"`
		NewestArchivedLog   *time.Time `db:"newest_archived_log"`
		ArchiveTableSize    string     `db:"archive_table_size"`
	}
	
	err = r.db.GetContext(ctx, &archiveStats, archiveQuery)
	if err == nil {
		stats.ArchivedLogs = archiveStats.ArchivedLogs
		stats.OldestArchivedLog = archiveStats.OldestArchivedLog
		stats.NewestArchivedLog = archiveStats.NewestArchivedLog
		stats.ArchiveTableSize = archiveStats.ArchiveTableSize
	}
	
	stats.GeneratedAt = time.Now()
	
	return &stats, nil
}

func (r *auditRepository) PurgeArchivedLogs(ctx context.Context, archivalDate time.Time) (int64, error) {
	query := `DELETE FROM audit_logs_archive WHERE created_at < $1`
	
	result, err := r.db.ExecContext(ctx, query, archivalDate)
	if err != nil {
		return 0, errors.Wrap(err, "failed to purge archived audit logs")
	}
	
	purgedCount, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get purged rows count")
	}
	
	return purgedCount, nil
}