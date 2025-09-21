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

type CollaborationRepository interface {
	// Comments
	CreateComment(ctx context.Context, comment *models.Comment) error
	GetComment(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	UpdateComment(ctx context.Context, comment *models.Comment) error
	DeleteComment(ctx context.Context, id uuid.UUID) error
	ListComments(ctx context.Context, filter models.CommentFilter) ([]*models.Comment, int, error)
	GetCommentsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Comment, error)
	
	// Assignments
	CreateAssignment(ctx context.Context, assignment *models.Assignment) error
	GetAssignment(ctx context.Context, id uuid.UUID) (*models.Assignment, error)
	UpdateAssignment(ctx context.Context, assignment *models.Assignment) error
	DeleteAssignment(ctx context.Context, id uuid.UUID) error
	ListAssignments(ctx context.Context, filter models.AssignmentFilter) ([]*models.Assignment, int, error)
	GetAssignmentsByUser(ctx context.Context, userID uuid.UUID) ([]*models.Assignment, error)
	GetAssignmentsByInvestigation(ctx context.Context, investigationID uuid.UUID) ([]*models.Assignment, error)
	
	// Teams
	CreateTeam(ctx context.Context, team *models.Team) error
	GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error)
	UpdateTeam(ctx context.Context, team *models.Team) error
	DeleteTeam(ctx context.Context, id uuid.UUID) error
	ListTeams(ctx context.Context, filter models.TeamFilter) ([]*models.Team, int, error)
	
	// Team Members
	AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, role string) error
	RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error
	UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, role string) error
	GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*models.TeamMember, error)
	GetUserTeams(ctx context.Context, userID uuid.UUID) ([]*models.Team, error)
	
	// Notifications
	CreateNotification(ctx context.Context, notification *models.NotificationEvent) error
	GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationEvent, error)
	UpdateNotification(ctx context.Context, notification *models.NotificationEvent) error
	DeleteNotification(ctx context.Context, id uuid.UUID) error
	ListNotifications(ctx context.Context, filter models.NotificationFilter) ([]*models.NotificationEvent, int, error)
	GetUserNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]*models.NotificationEvent, error)
	MarkNotificationAsRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllNotificationsAsRead(ctx context.Context, userID uuid.UUID) error
	
	// Activity Tracking
	CreateActivity(ctx context.Context, activity *models.Activity) error
	GetActivity(ctx context.Context, id uuid.UUID) (*models.Activity, error)
	ListActivities(ctx context.Context, filter models.ActivityFilter) ([]*models.Activity, int, error)
	GetActivitiesByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Activity, error)
	GetActivitiesByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*models.Activity, error)
	
	// Collaboration Stats
	GetCollaborationStats(ctx context.Context, filter models.CollaborationStatsFilter) (*models.CollaborationStats, error)
	GetUserActivityStats(ctx context.Context, userID uuid.UUID, dateFrom, dateTo time.Time) (*models.UserActivityStats, error)
	GetTeamActivityStats(ctx context.Context, teamID uuid.UUID, dateFrom, dateTo time.Time) (*models.TeamActivityStats, error)
}

type collaborationRepository struct {
	db *sqlx.DB
}

func NewCollaborationRepository(db *sqlx.DB) CollaborationRepository {
	return &collaborationRepository{db: db}
}

// Comments
func (r *collaborationRepository) CreateComment(ctx context.Context, comment *models.Comment) error {
	query := `
		INSERT INTO comments (
			id, entity_type, entity_id, parent_id, content, author_id,
			mentions, attachments, created_at, updated_at
		) VALUES (
			:id, :entity_type, :entity_id, :parent_id, :content, :author_id,
			:mentions, :attachments, :created_at, :updated_at
		)`
	
	comment.ID = uuid.New()
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, comment)
	if err != nil {
		return errors.Wrap(err, "failed to create comment")
	}
	
	return nil
}

func (r *collaborationRepository) GetComment(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	var comment models.Comment
	query := `
		SELECT id, entity_type, entity_id, parent_id, content, author_id,
			   mentions, attachments, created_at, updated_at
		FROM comments
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &comment, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("comment not found")
		}
		return nil, errors.Wrap(err, "failed to get comment")
	}
	
	return &comment, nil
}

func (r *collaborationRepository) UpdateComment(ctx context.Context, comment *models.Comment) error {
	query := `
		UPDATE comments
		SET content = :content, mentions = :mentions, attachments = :attachments,
			updated_at = :updated_at
		WHERE id = :id`
	
	comment.UpdatedAt = time.Now()
	
	result, err := r.db.NamedExecContext(ctx, query, comment)
	if err != nil {
		return errors.Wrap(err, "failed to update comment")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("comment not found")
	}
	
	return nil
}

func (r *collaborationRepository) DeleteComment(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM comments WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete comment")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("comment not found")
	}
	
	return nil
}

func (r *collaborationRepository) ListComments(ctx context.Context, filter models.CommentFilter) ([]*models.Comment, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM comments
		WHERE 1=1`
	
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
	
	if filter.AuthorID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("author_id = $%d", argCount))
		args = append(args, *filter.AuthorID)
	}
	
	if filter.ParentID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argCount))
		args = append(args, *filter.ParentID)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count comments")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, entity_type, entity_id, parent_id, content, author_id,
			   mentions, attachments, created_at, updated_at ` +
		baseQuery + `
		ORDER BY created_at ASC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var comments []*models.Comment
	err = r.db.SelectContext(ctx, &comments, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list comments")
	}
	
	return comments, total, nil
}

func (r *collaborationRepository) GetCommentsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Comment, error) {
	query := `
		SELECT id, entity_type, entity_id, parent_id, content, author_id,
			   mentions, attachments, created_at, updated_at
		FROM comments
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at ASC`
	
	var comments []*models.Comment
	err := r.db.SelectContext(ctx, &comments, query, entityType, entityID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get comments by entity")
	}
	
	return comments, nil
}

// Assignments
func (r *collaborationRepository) CreateAssignment(ctx context.Context, assignment *models.Assignment) error {
	query := `
		INSERT INTO assignments (
			id, entity_type, entity_id, assigned_to, assigned_by, role,
			description, due_date, created_at, updated_at
		) VALUES (
			:id, :entity_type, :entity_id, :assigned_to, :assigned_by, :role,
			:description, :due_date, :created_at, :updated_at
		)`
	
	assignment.ID = uuid.New()
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, assignment)
	if err != nil {
		return errors.Wrap(err, "failed to create assignment")
	}
	
	return nil
}

func (r *collaborationRepository) GetAssignment(ctx context.Context, id uuid.UUID) (*models.Assignment, error) {
	var assignment models.Assignment
	query := `
		SELECT id, entity_type, entity_id, assigned_to, assigned_by, role,
			   description, due_date, created_at, updated_at
		FROM assignments
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &assignment, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("assignment not found")
		}
		return nil, errors.Wrap(err, "failed to get assignment")
	}
	
	return &assignment, nil
}

func (r *collaborationRepository) UpdateAssignment(ctx context.Context, assignment *models.Assignment) error {
	query := `
		UPDATE assignments
		SET assigned_to = :assigned_to, role = :role, description = :description,
			due_date = :due_date, updated_at = :updated_at
		WHERE id = :id`
	
	assignment.UpdatedAt = time.Now()
	
	result, err := r.db.NamedExecContext(ctx, query, assignment)
	if err != nil {
		return errors.Wrap(err, "failed to update assignment")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("assignment not found")
	}
	
	return nil
}

func (r *collaborationRepository) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignments WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete assignment")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("assignment not found")
	}
	
	return nil
}

func (r *collaborationRepository) ListAssignments(ctx context.Context, filter models.AssignmentFilter) ([]*models.Assignment, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM assignments
		WHERE 1=1`
	
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
	
	if filter.AssignedTo != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argCount))
		args = append(args, *filter.AssignedTo)
	}
	
	if filter.AssignedBy != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("assigned_by = $%d", argCount))
		args = append(args, *filter.AssignedBy)
	}
	
	if filter.Role != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("role = $%d", argCount))
		args = append(args, filter.Role)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count assignments")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, entity_type, entity_id, assigned_to, assigned_by, role,
			   description, due_date, created_at, updated_at ` +
		baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var assignments []*models.Assignment
	err = r.db.SelectContext(ctx, &assignments, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list assignments")
	}
	
	return assignments, total, nil
}

func (r *collaborationRepository) GetAssignmentsByUser(ctx context.Context, userID uuid.UUID) ([]*models.Assignment, error) {
	query := `
		SELECT id, entity_type, entity_id, assigned_to, assigned_by, role,
			   description, due_date, created_at, updated_at
		FROM assignments
		WHERE assigned_to = $1
		ORDER BY created_at DESC`
	
	var assignments []*models.Assignment
	err := r.db.SelectContext(ctx, &assignments, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get assignments by user")
	}
	
	return assignments, nil
}

func (r *collaborationRepository) GetAssignmentsByInvestigation(ctx context.Context, investigationID uuid.UUID) ([]*models.Assignment, error) {
	query := `
		SELECT id, entity_type, entity_id, assigned_to, assigned_by, role,
			   description, due_date, created_at, updated_at
		FROM assignments
		WHERE entity_type = 'investigation' AND entity_id = $1
		ORDER BY created_at DESC`
	
	var assignments []*models.Assignment
	err := r.db.SelectContext(ctx, &assignments, query, investigationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get assignments by investigation")
	}
	
	return assignments, nil
}

// Teams
func (r *collaborationRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	query := `
		INSERT INTO teams (
			id, name, description, lead_id, created_by, created_at, updated_at
		) VALUES (
			:id, :name, :description, :lead_id, :created_by, :created_at, :updated_at
		)`
	
	team.ID = uuid.New()
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, team)
	if err != nil {
		return errors.Wrap(err, "failed to create team")
	}
	
	return nil
}

func (r *collaborationRepository) GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	var team models.Team
	query := `
		SELECT id, name, description, lead_id, created_by, created_at, updated_at
		FROM teams
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &team, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("team not found")
		}
		return nil, errors.Wrap(err, "failed to get team")
	}
	
	return &team, nil
}

func (r *collaborationRepository) UpdateTeam(ctx context.Context, team *models.Team) error {
	query := `
		UPDATE teams
		SET name = :name, description = :description, lead_id = :lead_id,
			updated_at = :updated_at
		WHERE id = :id`
	
	team.UpdatedAt = time.Now()
	
	result, err := r.db.NamedExecContext(ctx, query, team)
	if err != nil {
		return errors.Wrap(err, "failed to update team")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("team not found")
	}
	
	return nil
}

func (r *collaborationRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	// First delete team members
	_, err := r.db.ExecContext(ctx, "DELETE FROM team_members WHERE team_id = $1", id)
	if err != nil {
		return errors.Wrap(err, "failed to delete team members")
	}
	
	// Then delete the team
	query := `DELETE FROM teams WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete team")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("team not found")
	}
	
	return nil
}

func (r *collaborationRepository) ListTeams(ctx context.Context, filter models.TeamFilter) ([]*models.Team, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM teams
		WHERE 1=1`
	
	if filter.LeadID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("lead_id = $%d", argCount))
		args = append(args, *filter.LeadID)
	}
	
	if filter.CreatedBy != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argCount))
		args = append(args, *filter.CreatedBy)
	}
	
	if filter.Name != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argCount))
		args = append(args, "%"+filter.Name+"%")
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count teams")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, name, description, lead_id, created_by, created_at, updated_at ` +
		baseQuery + `
		ORDER BY name ASC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var teams []*models.Team
	err = r.db.SelectContext(ctx, &teams, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list teams")
	}
	
	return teams, total, nil
}

// Team Members
func (r *collaborationRepository) AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, role string) error {
	query := `
		INSERT INTO team_members (team_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (team_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			joined_at = EXCLUDED.joined_at`
	
	_, err := r.db.ExecContext(ctx, query, teamID, userID, role, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to add team member")
	}
	
	return nil
}

func (r *collaborationRepository) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	query := `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`
	
	result, err := r.db.ExecContext(ctx, query, teamID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to remove team member")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("team member not found")
	}
	
	return nil
}

func (r *collaborationRepository) UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, role string) error {
	query := `UPDATE team_members SET role = $1 WHERE team_id = $2 AND user_id = $3`
	
	result, err := r.db.ExecContext(ctx, query, role, teamID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to update team member role")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("team member not found")
	}
	
	return nil
}

func (r *collaborationRepository) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*models.TeamMember, error) {
	query := `
		SELECT team_id, user_id, role, joined_at
		FROM team_members
		WHERE team_id = $1
		ORDER BY joined_at ASC`
	
	var members []*models.TeamMember
	err := r.db.SelectContext(ctx, &members, query, teamID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team members")
	}
	
	return members, nil
}

func (r *collaborationRepository) GetUserTeams(ctx context.Context, userID uuid.UUID) ([]*models.Team, error) {
	query := `
		SELECT t.id, t.name, t.description, t.lead_id, t.created_by, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = $1
		ORDER BY t.name ASC`
	
	var teams []*models.Team
	err := r.db.SelectContext(ctx, &teams, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user teams")
	}
	
	return teams, nil
}

// Notifications
func (r *collaborationRepository) CreateNotification(ctx context.Context, notification *models.NotificationEvent) error {
	query := `
		INSERT INTO notification_events (
			id, user_id, type, title, message, entity_type, entity_id,
			metadata, is_read, created_at
		) VALUES (
			:id, :user_id, :type, :title, :message, :entity_type, :entity_id,
			:metadata, :is_read, :created_at
		)`
	
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, notification)
	if err != nil {
		return errors.Wrap(err, "failed to create notification")
	}
	
	return nil
}

func (r *collaborationRepository) GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationEvent, error) {
	var notification models.NotificationEvent
	query := `
		SELECT id, user_id, type, title, message, entity_type, entity_id,
			   metadata, is_read, read_at, created_at
		FROM notification_events
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &notification, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("notification not found")
		}
		return nil, errors.Wrap(err, "failed to get notification")
	}
	
	return &notification, nil
}

func (r *collaborationRepository) UpdateNotification(ctx context.Context, notification *models.NotificationEvent) error {
	query := `
		UPDATE notification_events
		SET title = :title, message = :message, metadata = :metadata,
			is_read = :is_read, read_at = :read_at
		WHERE id = :id`
	
	result, err := r.db.NamedExecContext(ctx, query, notification)
	if err != nil {
		return errors.Wrap(err, "failed to update notification")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("notification not found")
	}
	
	return nil
}

func (r *collaborationRepository) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notification_events WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete notification")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("notification not found")
	}
	
	return nil
}

func (r *collaborationRepository) ListNotifications(ctx context.Context, filter models.NotificationFilter) ([]*models.NotificationEvent, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM notification_events
		WHERE 1=1`
	
	if filter.UserID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argCount))
		args = append(args, *filter.UserID)
	}
	
	if filter.Type != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("type = $%d", argCount))
		args = append(args, filter.Type)
	}
	
	if filter.IsRead != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("is_read = $%d", argCount))
		args = append(args, *filter.IsRead)
	}
	
	if filter.EntityType != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argCount))
		args = append(args, filter.EntityType)
	}
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count notifications")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, user_id, type, title, message, entity_type, entity_id,
			   metadata, is_read, read_at, created_at ` +
		baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var notifications []*models.NotificationEvent
	err = r.db.SelectContext(ctx, &notifications, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list notifications")
	}
	
	return notifications, total, nil
}

func (r *collaborationRepository) GetUserNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]*models.NotificationEvent, error) {
	query := `
		SELECT id, user_id, type, title, message, entity_type, entity_id,
			   metadata, is_read, read_at, created_at
		FROM notification_events
		WHERE user_id = $1`
	
	args := []interface{}{userID}
	
	if unreadOnly {
		query += " AND is_read = false"
	}
	
	query += " ORDER BY created_at DESC LIMIT 100"
	
	var notifications []*models.NotificationEvent
	err := r.db.SelectContext(ctx, &notifications, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user notifications")
	}
	
	return notifications, nil
}

func (r *collaborationRepository) MarkNotificationAsRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `
		UPDATE notification_events
		SET is_read = true, read_at = $1
		WHERE id = $2 AND user_id = $3`
	
	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		return errors.Wrap(err, "failed to mark notification as read")
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	
	if rowsAffected == 0 {
		return errors.New("notification not found or not owned by user")
	}
	
	return nil
}

func (r *collaborationRepository) MarkAllNotificationsAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE notification_events
		SET is_read = true, read_at = $1
		WHERE user_id = $2 AND is_read = false`
	
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return errors.Wrap(err, "failed to mark all notifications as read")
	}
	
	return nil
}

// Activity Tracking
func (r *collaborationRepository) CreateActivity(ctx context.Context, activity *models.Activity) error {
	query := `
		INSERT INTO activities (
			id, user_id, action, entity_type, entity_id, description,
			metadata, created_at
		) VALUES (
			:id, :user_id, :action, :entity_type, :entity_id, :description,
			:metadata, :created_at
		)`
	
	activity.ID = uuid.New()
	activity.CreatedAt = time.Now()
	
	_, err := r.db.NamedExecContext(ctx, query, activity)
	if err != nil {
		return errors.Wrap(err, "failed to create activity")
	}
	
	return nil
}

func (r *collaborationRepository) GetActivity(ctx context.Context, id uuid.UUID) (*models.Activity, error) {
	var activity models.Activity
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   metadata, created_at
		FROM activities
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, &activity, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("activity not found")
		}
		return nil, errors.Wrap(err, "failed to get activity")
	}
	
	return &activity, nil
}

func (r *collaborationRepository) ListActivities(ctx context.Context, filter models.ActivityFilter) ([]*models.Activity, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 0
	
	baseQuery := `
		FROM activities
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
	
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Count query
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count activities")
	}
	
	// Data query with pagination
	dataQuery := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   metadata, created_at ` +
		baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	
	args = append(args, filter.Limit, filter.Offset)
	
	var activities []*models.Activity
	err = r.db.SelectContext(ctx, &activities, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list activities")
	}
	
	return activities, total, nil
}

func (r *collaborationRepository) GetActivitiesByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Activity, error) {
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   metadata, created_at
		FROM activities
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT 100`
	
	var activities []*models.Activity
	err := r.db.SelectContext(ctx, &activities, query, entityType, entityID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get activities by entity")
	}
	
	return activities, nil
}

func (r *collaborationRepository) GetActivitiesByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*models.Activity, error) {
	if limit <= 0 {
		limit = 50
	}
	
	query := `
		SELECT id, user_id, action, entity_type, entity_id, description,
			   metadata, created_at
		FROM activities
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var activities []*models.Activity
	err := r.db.SelectContext(ctx, &activities, query, userID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get activities by user")
	}
	
	return activities, nil
}

// Collaboration Statistics
func (r *collaborationRepository) GetCollaborationStats(ctx context.Context, filter models.CollaborationStatsFilter) (*models.CollaborationStats, error) {
	var stats models.CollaborationStats
	
	baseQuery := `WHERE 1=1`
	var conditions []string
	var args []interface{}
	argCount := 0
	
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
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	// Get comment stats
	commentQuery := `
		SELECT 
			COUNT(*) as total_comments,
			COUNT(DISTINCT author_id) as active_commenters
		FROM comments ` + baseQuery
	
	err := r.db.GetContext(ctx, &stats, commentQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get comment stats")
	}
	
	// Get assignment stats
	assignmentQuery := `
		SELECT 
			COUNT(*) as total_assignments,
			COUNT(DISTINCT assigned_to) as users_with_assignments
		FROM assignments ` + baseQuery
	
	var assignmentStats struct {
		TotalAssignments      int `db:"total_assignments"`
		UsersWithAssignments int `db:"users_with_assignments"`
	}
	
	err = r.db.GetContext(ctx, &assignmentStats, assignmentQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get assignment stats")
	}
	
	stats.TotalAssignments = assignmentStats.TotalAssignments
	stats.UsersWithAssignments = assignmentStats.UsersWithAssignments
	
	// Get team stats
	teamQuery := `SELECT COUNT(*) as total_teams FROM teams ` + baseQuery
	err = r.db.GetContext(ctx, &stats.TotalTeams, teamQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team stats")
	}
	
	return &stats, nil
}

func (r *collaborationRepository) GetUserActivityStats(ctx context.Context, userID uuid.UUID, dateFrom, dateTo time.Time) (*models.UserActivityStats, error) {
	var stats models.UserActivityStats
	
	query := `
		SELECT 
			COUNT(*) as total_activities,
			COUNT(CASE WHEN action = 'comment' THEN 1 END) as comments_count,
			COUNT(CASE WHEN action = 'assignment' THEN 1 END) as assignments_count,
			COUNT(CASE WHEN action = 'investigation_update' THEN 1 END) as investigation_updates,
			COUNT(CASE WHEN action = 'evidence_added' THEN 1 END) as evidence_additions
		FROM activities
		WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3`
	
	err := r.db.GetContext(ctx, &stats, query, userID, dateFrom, dateTo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user activity stats")
	}
	
	stats.UserID = userID
	stats.DateFrom = dateFrom
	stats.DateTo = dateTo
	
	return &stats, nil
}

func (r *collaborationRepository) GetTeamActivityStats(ctx context.Context, teamID uuid.UUID, dateFrom, dateTo time.Time) (*models.TeamActivityStats, error) {
	var stats models.TeamActivityStats
	
	query := `
		SELECT 
			COUNT(*) as total_activities,
			COUNT(DISTINCT a.user_id) as active_members,
			COUNT(CASE WHEN a.action = 'comment' THEN 1 END) as total_comments,
			COUNT(CASE WHEN a.action = 'assignment' THEN 1 END) as total_assignments
		FROM activities a
		JOIN team_members tm ON a.user_id = tm.user_id
		WHERE tm.team_id = $1 AND a.created_at >= $2 AND a.created_at <= $3`
	
	err := r.db.GetContext(ctx, &stats, query, teamID, dateFrom, dateTo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team activity stats")
	}
	
	stats.TeamID = teamID
	stats.DateFrom = dateFrom
	stats.DateTo = dateTo
	
	return &stats, nil
}