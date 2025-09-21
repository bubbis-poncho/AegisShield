package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"investigation-toolkit/internal/models"
	"investigation-toolkit/internal/repository"
)

type CollaborationHandler struct {
	collaborationRepo repository.CollaborationRepository
	auditRepo        repository.AuditRepository
}

func NewCollaborationHandler(collaborationRepo repository.CollaborationRepository, auditRepo repository.AuditRepository) *CollaborationHandler {
	return &CollaborationHandler{
		collaborationRepo: collaborationRepo,
		auditRepo:        auditRepo,
	}
}

// Comments
func (h *CollaborationHandler) CreateComment(c *gin.Context) {
	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	comment := &models.Comment{
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		ParentID:    req.ParentID,
		Content:     req.Content,
		AuthorID:    req.AuthorID,
		Mentions:    req.Mentions,
		Attachments: req.Attachments,
	}

	if err := h.collaborationRepo.CreateComment(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.AuthorID,
		Action:      "create_comment",
		EntityType:  req.EntityType,
		EntityID:    &req.EntityID,
		Description: "Created comment on " + req.EntityType,
		NewValues:   map[string]interface{}{"comment": comment},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusCreated, comment)
}

func (h *CollaborationHandler) GetComment(c *gin.Context) {
	idParam := c.Param("id")
	commentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	comment, err := h.collaborationRepo.GetComment(c.Request.Context(), commentID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comment)
}

func (h *CollaborationHandler) UpdateComment(c *gin.Context) {
	idParam := c.Param("id")
	commentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	var req models.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get existing comment
	comment, err := h.collaborationRepo.GetComment(c.Request.Context(), commentID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comment", "details": err.Error()})
		return
	}

	oldContent := comment.Content
	comment.Content = req.Content
	comment.Mentions = req.Mentions
	comment.Attachments = req.Attachments

	if err := h.collaborationRepo.UpdateComment(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UpdatedBy,
		Action:      "update_comment",
		EntityType:  "comment",
		EntityID:    &commentID,
		Description: "Updated comment",
		OldValues:   map[string]interface{}{"content": oldContent},
		NewValues:   map[string]interface{}{"content": req.Content},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, comment)
}

func (h *CollaborationHandler) DeleteComment(c *gin.Context) {
	idParam := c.Param("id")
	commentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	userID := c.GetHeader("X-User-ID")
	userUUID, _ := uuid.Parse(userID)

	// Get comment for audit before deletion
	comment, err := h.collaborationRepo.GetComment(c.Request.Context(), commentID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comment", "details": err.Error()})
		return
	}

	if err := h.collaborationRepo.DeleteComment(c.Request.Context(), commentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &userUUID,
		Action:      "delete_comment",
		EntityType:  "comment",
		EntityID:    &commentID,
		Description: "Deleted comment",
		OldValues:   map[string]interface{}{"comment": comment},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

func (h *CollaborationHandler) GetCommentsByEntity(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityIDParam := c.Param("entity_id")
	entityID, err := uuid.Parse(entityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity ID format"})
		return
	}

	comments, err := h.collaborationRepo.GetCommentsByEntity(c.Request.Context(), entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comments", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

// Assignments
func (h *CollaborationHandler) CreateAssignment(c *gin.Context) {
	var req models.CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	assignment := &models.Assignment{
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		AssignedTo:  req.AssignedTo,
		AssignedBy:  req.AssignedBy,
		Role:        req.Role,
		Description: req.Description,
		DueDate:     req.DueDate,
	}

	if err := h.collaborationRepo.CreateAssignment(c.Request.Context(), assignment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.AssignedBy,
		Action:      "create_assignment",
		EntityType:  req.EntityType,
		EntityID:    &req.EntityID,
		Description: "Assigned " + req.Role + " to user",
		NewValues:   map[string]interface{}{"assignment": assignment},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusCreated, assignment)
}

func (h *CollaborationHandler) GetAssignment(c *gin.Context) {
	idParam := c.Param("id")
	assignmentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID format"})
		return
	}

	assignment, err := h.collaborationRepo.GetAssignment(c.Request.Context(), assignmentID)
	if err != nil {
		if err.Error() == "assignment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get assignment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assignment)
}

func (h *CollaborationHandler) UpdateAssignment(c *gin.Context) {
	idParam := c.Param("id")
	assignmentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID format"})
		return
	}

	var req models.UpdateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get existing assignment
	assignment, err := h.collaborationRepo.GetAssignment(c.Request.Context(), assignmentID)
	if err != nil {
		if err.Error() == "assignment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get assignment", "details": err.Error()})
		return
	}

	oldAssignedTo := assignment.AssignedTo
	assignment.AssignedTo = req.AssignedTo
	assignment.Role = req.Role
	assignment.Description = req.Description
	assignment.DueDate = req.DueDate

	if err := h.collaborationRepo.UpdateAssignment(c.Request.Context(), assignment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UpdatedBy,
		Action:      "update_assignment",
		EntityType:  "assignment",
		EntityID:    &assignmentID,
		Description: "Updated assignment",
		OldValues:   map[string]interface{}{"assigned_to": oldAssignedTo},
		NewValues:   map[string]interface{}{"assigned_to": req.AssignedTo, "role": req.Role},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, assignment)
}

func (h *CollaborationHandler) GetUserAssignments(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	assignments, err := h.collaborationRepo.GetAssignmentsByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user assignments", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"assignments": assignments})
}

// Teams
func (h *CollaborationHandler) CreateTeam(c *gin.Context) {
	var req models.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	team := &models.Team{
		Name:        req.Name,
		Description: req.Description,
		LeadID:      req.LeadID,
		CreatedBy:   req.CreatedBy,
	}

	if err := h.collaborationRepo.CreateTeam(c.Request.Context(), team); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team", "details": err.Error()})
		return
	}

	// Add team lead as a member
	if req.LeadID != nil {
		h.collaborationRepo.AddTeamMember(c.Request.Context(), team.ID, *req.LeadID, "lead")
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.CreatedBy,
		Action:      "create_team",
		EntityType:  "team",
		EntityID:    &team.ID,
		Description: "Created team: " + team.Name,
		NewValues:   map[string]interface{}{"team": team},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusCreated, team)
}

func (h *CollaborationHandler) GetTeam(c *gin.Context) {
	idParam := c.Param("id")
	teamID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID format"})
		return
	}

	team, err := h.collaborationRepo.GetTeam(c.Request.Context(), teamID)
	if err != nil {
		if err.Error() == "team not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get team", "details": err.Error()})
		return
	}

	// Get team members
	members, err := h.collaborationRepo.GetTeamMembers(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get team members", "details": err.Error()})
		return
	}

	response := models.TeamResponse{
		Team:    team,
		Members: members,
	}

	c.JSON(http.StatusOK, response)
}

func (h *CollaborationHandler) UpdateTeam(c *gin.Context) {
	idParam := c.Param("id")
	teamID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID format"})
		return
	}

	var req models.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get existing team
	team, err := h.collaborationRepo.GetTeam(c.Request.Context(), teamID)
	if err != nil {
		if err.Error() == "team not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get team", "details": err.Error()})
		return
	}

	team.Name = req.Name
	team.Description = req.Description
	team.LeadID = req.LeadID

	if err := h.collaborationRepo.UpdateTeam(c.Request.Context(), team); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UpdatedBy,
		Action:      "update_team",
		EntityType:  "team",
		EntityID:    &teamID,
		Description: "Updated team: " + team.Name,
		NewValues:   map[string]interface{}{"team": team},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, team)
}

func (h *CollaborationHandler) AddTeamMember(c *gin.Context) {
	teamIDParam := c.Param("team_id")
	teamID, err := uuid.Parse(teamIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID format"})
		return
	}

	var req models.AddTeamMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	if err := h.collaborationRepo.AddTeamMember(c.Request.Context(), teamID, req.UserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add team member", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.AddedBy,
		Action:      "add_team_member",
		EntityType:  "team",
		EntityID:    &teamID,
		Description: "Added team member with role: " + req.Role,
		NewValues:   map[string]interface{}{"user_id": req.UserID, "role": req.Role},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Team member added successfully"})
}

func (h *CollaborationHandler) RemoveTeamMember(c *gin.Context) {
	teamIDParam := c.Param("team_id")
	teamID, err := uuid.Parse(teamIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID format"})
		return
	}

	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	if err := h.collaborationRepo.RemoveTeamMember(c.Request.Context(), teamID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove team member", "details": err.Error()})
		return
	}

	removedByStr := c.GetHeader("X-User-ID")
	removedBy, _ := uuid.Parse(removedByStr)

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &removedBy,
		Action:      "remove_team_member",
		EntityType:  "team",
		EntityID:    &teamID,
		Description: "Removed team member",
		OldValues:   map[string]interface{}{"user_id": userID},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Team member removed successfully"})
}

func (h *CollaborationHandler) GetUserTeams(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	teams, err := h.collaborationRepo.GetUserTeams(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user teams", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// Notifications
func (h *CollaborationHandler) CreateNotification(c *gin.Context) {
	var req models.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	notification := &models.NotificationEvent{
		UserID:     req.UserID,
		Type:       req.Type,
		Title:      req.Title,
		Message:    req.Message,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Metadata:   req.Metadata,
		IsRead:     false,
	}

	if err := h.collaborationRepo.CreateNotification(c.Request.Context(), notification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *CollaborationHandler) GetUserNotifications(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	unreadOnlyStr := c.Query("unread_only")
	unreadOnly, _ := strconv.ParseBool(unreadOnlyStr)

	notifications, err := h.collaborationRepo.GetUserNotifications(c.Request.Context(), userID, unreadOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (h *CollaborationHandler) MarkNotificationAsRead(c *gin.Context) {
	notificationIDParam := c.Param("id")
	notificationID, err := uuid.Parse(notificationIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID format"})
		return
	}

	userIDStr := c.GetHeader("X-User-ID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID in header"})
		return
	}

	if err := h.collaborationRepo.MarkNotificationAsRead(c.Request.Context(), notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (h *CollaborationHandler) MarkAllNotificationsAsRead(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	if err := h.collaborationRepo.MarkAllNotificationsAsRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all notifications as read", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

// Activity and Statistics
func (h *CollaborationHandler) GetCollaborationStats(c *gin.Context) {
	var filter models.CollaborationStatsFilter

	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			filter.DateFrom = dateFrom
		}
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			filter.DateTo = dateTo
		}
	}

	stats, err := h.collaborationRepo.GetCollaborationStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collaboration stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CollaborationHandler) GetUserActivityStats(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")

	if dateFromStr == "" || dateToStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date_from and date_to parameters are required"})
		return
	}

	dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_from format, use RFC3339"})
		return
	}

	dateTo, err := time.Parse(time.RFC3339, dateToStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_to format, use RFC3339"})
		return
	}

	stats, err := h.collaborationRepo.GetUserActivityStats(c.Request.Context(), userID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user activity stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CollaborationHandler) GetTeamActivityStats(c *gin.Context) {
	teamIDParam := c.Param("team_id")
	teamID, err := uuid.Parse(teamIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID format"})
		return
	}

	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")

	if dateFromStr == "" || dateToStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date_from and date_to parameters are required"})
		return
	}

	dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_from format, use RFC3339"})
		return
	}

	dateTo, err := time.Parse(time.RFC3339, dateToStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_to format, use RFC3339"})
		return
	}

	stats, err := h.collaborationRepo.GetTeamActivityStats(c.Request.Context(), teamID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get team activity stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}