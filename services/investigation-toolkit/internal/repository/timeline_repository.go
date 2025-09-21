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

// TimelineRepository handles timeline-related database operations
type TimelineRepository struct {
	*database.Repository
}

// NewTimelineRepository creates a new timeline repository
func NewTimelineRepository(db *database.Database, logger *zap.Logger) *TimelineRepository {
	return &TimelineRepository{
		Repository: database.NewRepository(db, logger),
	}
}

// Create creates a new timeline event
func (r *TimelineRepository) Create(ctx context.Context, investigationID uuid.UUID, req *models.CreateTimelineRequest, createdBy uuid.UUID) (*models.Timeline, error) {
	timeline := &models.Timeline{
		ID:                 uuid.New(),
		InvestigationID:    investigationID,
		Title:              req.Title,
		Description:        req.Description,
		EventType:          req.EventType,
		EventDate:          req.EventDate,
		DurationMinutes:    req.DurationMinutes,
		Location:           req.Location,
		Participants:       req.Participants,
		RelatedEvidenceIDs: req.RelatedEvidenceIDs,
		ExternalReferences: req.ExternalReferences,
		Metadata:           req.Metadata,
		Tags:               req.Tags,
		CreatedBy:          createdBy,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	query := `
		INSERT INTO timelines (
			id, investigation_id, title, description, event_type, event_date, duration_minutes,
			location, participants, related_evidence_ids, external_references, metadata, tags,
			created_by, created_at, updated_at
		) VALUES (
			:id, :investigation_id, :title, :description, :event_type, :event_date, :duration_minutes,
			:location, :participants, :related_evidence_ids, :external_references, :metadata, :tags,
			:created_by, :created_at, :updated_at
		) RETURNING id, created_at, updated_at`

	rows, err := r.DB().NamedQueryContext(ctx, query, timeline)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create timeline event")
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&timeline.ID, &timeline.CreatedAt, &timeline.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan created timeline event")
		}
	}

	return timeline, nil
}

// GetByID retrieves a timeline event by ID
func (r *TimelineRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Timeline, error) {
	var timeline models.Timeline
	
	query := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE id = $1`

	err := r.DB().GetContext(ctx, &timeline, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("timeline event not found")
		}
		return nil, errors.Wrap(err, "failed to get timeline event")
	}

	return &timeline, nil
}

// GetByInvestigationID retrieves all timeline events for an investigation
func (r *TimelineRepository) GetByInvestigationID(ctx context.Context, investigationID uuid.UUID, paginate *database.Paginate) (*database.PaginatedResult, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM timelines WHERE investigation_id = $1"
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events count")
	}

	// Get data
	dataQuery := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE investigation_id = $1
		ORDER BY event_date DESC
		LIMIT $2 OFFSET $3`

	var timelines []models.Timeline
	err = r.DB().SelectContext(ctx, &timelines, dataQuery, investigationID, paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events")
	}

	return database.NewPaginatedResult(timelines, total, paginate), nil
}

// GetByDateRange retrieves timeline events within a date range
func (r *TimelineRepository) GetByDateRange(ctx context.Context, investigationID uuid.UUID, startDate, endDate time.Time, paginate *database.Paginate) (*database.PaginatedResult, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM timelines 
		WHERE investigation_id = $1 AND event_date BETWEEN $2 AND $3`
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, investigationID, startDate, endDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events count for date range")
	}

	// Get data
	dataQuery := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE investigation_id = $1 AND event_date BETWEEN $2 AND $3
		ORDER BY event_date ASC
		LIMIT $4 OFFSET $5`

	var timelines []models.Timeline
	err = r.DB().SelectContext(ctx, &timelines, dataQuery, investigationID, startDate, endDate, paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events for date range")
	}

	return database.NewPaginatedResult(timelines, total, paginate), nil
}

// GetByEventType retrieves timeline events by event type
func (r *TimelineRepository) GetByEventType(ctx context.Context, investigationID uuid.UUID, eventType models.EventType, paginate *database.Paginate) (*database.PaginatedResult, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM timelines 
		WHERE investigation_id = $1 AND event_type = $2`
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, investigationID, eventType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events count by type")
	}

	// Get data
	dataQuery := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE investigation_id = $1 AND event_type = $2
		ORDER BY event_date DESC
		LIMIT $3 OFFSET $4`

	var timelines []models.Timeline
	err = r.DB().SelectContext(ctx, &timelines, dataQuery, investigationID, eventType, paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events by type")
	}

	return database.NewPaginatedResult(timelines, total, paginate), nil
}

// Update updates a timeline event
func (r *TimelineRepository) Update(ctx context.Context, id uuid.UUID, req *models.CreateTimelineRequest) (*models.Timeline, error) {
	query := `
		UPDATE timelines 
		SET title = $1, description = $2, event_type = $3, event_date = $4, duration_minutes = $5,
			location = $6, participants = $7, related_evidence_ids = $8, external_references = $9,
			metadata = $10, tags = $11, updated_at = CURRENT_TIMESTAMP
		WHERE id = $12
		RETURNING id, investigation_id, title, description, event_type, event_date, duration_minutes,
				  location, participants, related_evidence_ids, external_references, metadata, tags,
				  created_by, created_at, updated_at`

	var timeline models.Timeline
	err := r.DB().GetContext(ctx, &timeline, query,
		req.Title, req.Description, req.EventType, req.EventDate, req.DurationMinutes,
		req.Location, req.Participants, req.RelatedEvidenceIDs, req.ExternalReferences,
		req.Metadata, req.Tags, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("timeline event not found")
		}
		return nil, errors.Wrap(err, "failed to update timeline event")
	}

	return &timeline, nil
}

// Delete deletes a timeline event
func (r *TimelineRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM timelines WHERE id = $1"

	result, err := r.DB().ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete timeline event")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("timeline event not found")
	}

	return nil
}

// GetByParticipant retrieves timeline events where a specific participant is involved
func (r *TimelineRepository) GetByParticipant(ctx context.Context, investigationID uuid.UUID, participant string, paginate *database.Paginate) (*database.PaginatedResult, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM timelines 
		WHERE investigation_id = $1 AND $2 = ANY(participants)`
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, investigationID, participant)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events count by participant")
	}

	// Get data
	dataQuery := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE investigation_id = $1 AND $2 = ANY(participants)
		ORDER BY event_date DESC
		LIMIT $3 OFFSET $4`

	var timelines []models.Timeline
	err = r.DB().SelectContext(ctx, &timelines, dataQuery, investigationID, participant, paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events by participant")
	}

	return database.NewPaginatedResult(timelines, total, paginate), nil
}

// GetByEvidenceID retrieves timeline events related to specific evidence
func (r *TimelineRepository) GetByEvidenceID(ctx context.Context, evidenceID uuid.UUID, paginate *database.Paginate) (*database.PaginatedResult, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM timelines 
		WHERE $1 = ANY(related_evidence_ids)`
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, evidenceID.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events count by evidence")
	}

	// Get data
	dataQuery := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE $1 = ANY(related_evidence_ids)
		ORDER BY event_date DESC
		LIMIT $2 OFFSET $3`

	var timelines []models.Timeline
	err = r.DB().SelectContext(ctx, &timelines, dataQuery, evidenceID.String(), paginate.Limit, paginate.Offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events by evidence")
	}

	return database.NewPaginatedResult(timelines, total, paginate), nil
}

// Search performs a full-text search on timeline events
func (r *TimelineRepository) Search(ctx context.Context, investigationID uuid.UUID, searchTerm string, eventTypes []models.EventType, startDate, endDate *time.Time, paginate *database.Paginate) (*database.PaginatedResult, error) {
	whereConditions := []string{"investigation_id = $1"}
	args := []interface{}{investigationID}
	argIndex := 1

	// Add search term
	if searchTerm != "" {
		argIndex++
		whereConditions = append(whereConditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+searchTerm+"%")
	}

	// Add event types filter
	if len(eventTypes) > 0 {
		argIndex++
		whereConditions = append(whereConditions, fmt.Sprintf("event_type = ANY($%d)", argIndex))
		args = append(args, eventTypes)
	}

	// Add date range filter
	if startDate != nil {
		argIndex++
		whereConditions = append(whereConditions, fmt.Sprintf("event_date >= $%d", argIndex))
		args = append(args, *startDate)
	}
	if endDate != nil {
		argIndex++
		whereConditions = append(whereConditions, fmt.Sprintf("event_date <= $%d", argIndex))
		args = append(args, *endDate)
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM timelines WHERE %s", whereClause)
	var total int64
	err := r.DB().GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events search count")
	}

	// Get data
	dataQuery := fmt.Sprintf(`
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE %s
		ORDER BY event_date DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex+1, argIndex+2)

	args = append(args, paginate.Limit, paginate.Offset)

	var timelines []models.Timeline
	err = r.DB().SelectContext(ctx, &timelines, dataQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search timeline events")
	}

	return database.NewPaginatedResult(timelines, total, paginate), nil
}

// GetTimelineStats retrieves timeline statistics for an investigation
func (r *TimelineRepository) GetTimelineStats(ctx context.Context, investigationID uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total events
	query := "SELECT COUNT(*) FROM timelines WHERE investigation_id = $1"
	var total int64
	err := r.DB().GetContext(ctx, &total, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total timeline events")
	}
	stats["total"] = total

	// By event type
	query = `
		SELECT event_type, COUNT(*) 
		FROM timelines 
		WHERE investigation_id = $1
		GROUP BY event_type`
	
	rows, err := r.DB().QueryContext(ctx, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events by type")
	}
	defer rows.Close()

	typeStats := make(map[string]int64)
	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, errors.Wrap(err, "failed to scan type stats")
		}
		typeStats[eventType] = count
	}
	stats["by_type"] = typeStats

	// Date range
	query = `
		SELECT MIN(event_date), MAX(event_date) 
		FROM timelines 
		WHERE investigation_id = $1`
	
	var minDate, maxDate *time.Time
	err = r.DB().QueryRowContext(ctx, query, investigationID).Scan(&minDate, &maxDate)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "failed to get timeline date range")
	}
	
	dateRange := map[string]interface{}{
		"start": minDate,
		"end":   maxDate,
	}
	stats["date_range"] = dateRange

	// Duration statistics (for events with duration)
	query = `
		SELECT COUNT(*), AVG(duration_minutes), MIN(duration_minutes), MAX(duration_minutes)
		FROM timelines 
		WHERE investigation_id = $1 AND duration_minutes IS NOT NULL`
	
	var countWithDuration int64
	var avgDuration, minDuration, maxDuration *float64
	err = r.DB().QueryRowContext(ctx, query, investigationID).Scan(&countWithDuration, &avgDuration, &minDuration, &maxDuration)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "failed to get duration statistics")
	}
	
	durationStats := map[string]interface{}{
		"events_with_duration": countWithDuration,
		"average_duration":     avgDuration,
		"min_duration":         minDuration,
		"max_duration":         maxDuration,
	}
	stats["duration"] = durationStats

	// Most active participants
	query = `
		SELECT participant, COUNT(*) as event_count
		FROM (
			SELECT UNNEST(participants) as participant
			FROM timelines 
			WHERE investigation_id = $1
		) as participant_events
		GROUP BY participant
		ORDER BY event_count DESC
		LIMIT 10`
	
	rows, err = r.DB().QueryContext(ctx, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get participant statistics")
	}
	defer rows.Close()

	participantStats := make([]map[string]interface{}, 0)
	for rows.Next() {
		var participant string
		var count int64
		if err := rows.Scan(&participant, &count); err != nil {
			return nil, errors.Wrap(err, "failed to scan participant stats")
		}
		participantStats = append(participantStats, map[string]interface{}{
			"participant": participant,
			"event_count": count,
		})
	}
	stats["top_participants"] = participantStats

	return stats, nil
}

// GetEventsForEvidence retrieves timeline events that reference specific evidence
func (r *TimelineRepository) GetEventsForEvidence(ctx context.Context, evidenceIDs []uuid.UUID) ([]models.Timeline, error) {
	if len(evidenceIDs) == 0 {
		return []models.Timeline{}, nil
	}

	// Convert UUIDs to strings for the array query
	evidenceIDStrings := make([]string, len(evidenceIDs))
	for i, id := range evidenceIDs {
		evidenceIDStrings[i] = id.String()
	}

	query := `
		SELECT id, investigation_id, title, description, event_type, event_date, duration_minutes,
			   location, participants, related_evidence_ids, external_references, metadata, tags,
			   created_by, created_at, updated_at
		FROM timelines 
		WHERE related_evidence_ids && $1
		ORDER BY event_date DESC`

	var timelines []models.Timeline
	err := r.DB().SelectContext(ctx, &timelines, query, evidenceIDStrings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get timeline events for evidence")
	}

	return timelines, nil
}

// BulkCreate creates multiple timeline events in a single transaction
func (r *TimelineRepository) BulkCreate(ctx context.Context, investigationID uuid.UUID, requests []*models.CreateTimelineRequest, createdBy uuid.UUID) ([]models.Timeline, error) {
	timelines := make([]models.Timeline, 0, len(requests))

	err := r.WithTx(ctx, func(tx *sqlx.Tx) error {
		for _, req := range requests {
			timeline := models.Timeline{
				ID:                 uuid.New(),
				InvestigationID:    investigationID,
				Title:              req.Title,
				Description:        req.Description,
				EventType:          req.EventType,
				EventDate:          req.EventDate,
				DurationMinutes:    req.DurationMinutes,
				Location:           req.Location,
				Participants:       req.Participants,
				RelatedEvidenceIDs: req.RelatedEvidenceIDs,
				ExternalReferences: req.ExternalReferences,
				Metadata:           req.Metadata,
				Tags:               req.Tags,
				CreatedBy:          createdBy,
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}

			query := `
				INSERT INTO timelines (
					id, investigation_id, title, description, event_type, event_date, duration_minutes,
					location, participants, related_evidence_ids, external_references, metadata, tags,
					created_by, created_at, updated_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
				) RETURNING created_at, updated_at`

			err := tx.QueryRowContext(ctx, query,
				timeline.ID, timeline.InvestigationID, timeline.Title, timeline.Description,
				timeline.EventType, timeline.EventDate, timeline.DurationMinutes, timeline.Location,
				timeline.Participants, timeline.RelatedEvidenceIDs, timeline.ExternalReferences,
				timeline.Metadata, timeline.Tags, timeline.CreatedBy, timeline.CreatedAt, timeline.UpdatedAt,
			).Scan(&timeline.CreatedAt, &timeline.UpdatedAt)
			if err != nil {
				return errors.Wrap(err, "failed to create timeline event in bulk")
			}

			timelines = append(timelines, timeline)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return timelines, nil
}