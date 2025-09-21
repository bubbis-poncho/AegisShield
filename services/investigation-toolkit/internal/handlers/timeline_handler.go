package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/models"
	"investigation-toolkit/internal/repository"
)

// TimelineHandler handles HTTP requests for timeline events
type TimelineHandler struct {
	repo   *repository.TimelineRepository
	logger *zap.Logger
}

// NewTimelineHandler creates a new timeline handler
func NewTimelineHandler(repo *repository.TimelineRepository, logger *zap.Logger) *TimelineHandler {
	return &TimelineHandler{
		repo:   repo,
		logger: logger.Named("timeline_handler"),
	}
}

// CreateTimelineEvent creates a new timeline event for an investigation
func (h *TimelineHandler) CreateTimelineEvent(c *gin.Context) {
	// Get investigation ID from URL
	investigationIDStr := c.Param("id")
	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	var req models.CreateTimelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Get user ID from context (would come from auth middleware)
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	timeline, err := h.repo.Create(c.Request.Context(), investigationID, &req, userID)
	if err != nil {
		h.logger.Error("Failed to create timeline event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create timeline event"})
		return
	}

	h.logger.Info("Timeline event created", zap.String("id", timeline.ID.String()))
	c.JSON(http.StatusCreated, timeline)
}

// GetTimelineEvent retrieves a timeline event by ID
func (h *TimelineHandler) GetTimelineEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeline event ID"})
		return
	}

	timeline, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "timeline event not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Timeline event not found"})
			return
		}
		h.logger.Error("Failed to get timeline event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline event"})
		return
	}

	c.JSON(http.StatusOK, timeline)
}

// UpdateTimelineEvent updates a timeline event
func (h *TimelineHandler) UpdateTimelineEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeline event ID"})
		return
	}

	var req models.CreateTimelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	timeline, err := h.repo.Update(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "timeline event not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Timeline event not found"})
			return
		}
		h.logger.Error("Failed to update timeline event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update timeline event"})
		return
	}

	h.logger.Info("Timeline event updated", zap.String("id", id.String()))
	c.JSON(http.StatusOK, timeline)
}

// DeleteTimelineEvent deletes a timeline event
func (h *TimelineHandler) DeleteTimelineEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeline event ID"})
		return
	}

	err = h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "timeline event not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Timeline event not found"})
			return
		}
		h.logger.Error("Failed to delete timeline event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete timeline event"})
		return
	}

	h.logger.Info("Timeline event deleted", zap.String("id", id.String()))
	c.JSON(http.StatusNoContent, nil)
}

// ListTimelineEvents lists timeline events for an investigation
func (h *TimelineHandler) ListTimelineEvents(c *gin.Context) {
	// Get investigation ID from URL
	investigationIDStr := c.Param("id")
	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	// Check for specific query types
	switch c.Query("type") {
	case "date_range":
		h.getTimelineByDateRange(c, investigationID, paginate)
		return
	case "event_type":
		h.getTimelineByEventType(c, investigationID, paginate)
		return
	case "participant":
		h.getTimelineByParticipant(c, investigationID, paginate)
		return
	default:
		// Default: get all timeline events for investigation
		result, err := h.repo.GetByInvestigationID(c.Request.Context(), investigationID, paginate)
		if err != nil {
			h.logger.Error("Failed to list timeline events", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list timeline events"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// getTimelineByDateRange retrieves timeline events within a date range
func (h *TimelineHandler) getTimelineByDateRange(c *gin.Context, investigationID uuid.UUID, paginate *database.Paginate) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required for date range query"})
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format, use RFC3339"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format, use RFC3339"})
		return
	}

	result, err := h.repo.GetByDateRange(c.Request.Context(), investigationID, startDate, endDate, paginate)
	if err != nil {
		h.logger.Error("Failed to get timeline events by date range", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline events by date range"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getTimelineByEventType retrieves timeline events by event type
func (h *TimelineHandler) getTimelineByEventType(c *gin.Context, investigationID uuid.UUID, paginate *database.Paginate) {
	eventTypeStr := c.Query("event_type")
	if eventTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_type is required"})
		return
	}

	eventType := models.EventType(eventTypeStr)
	result, err := h.repo.GetByEventType(c.Request.Context(), investigationID, eventType, paginate)
	if err != nil {
		h.logger.Error("Failed to get timeline events by type", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline events by type"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getTimelineByParticipant retrieves timeline events by participant
func (h *TimelineHandler) getTimelineByParticipant(c *gin.Context, investigationID uuid.UUID, paginate *database.Paginate) {
	participant := c.Query("participant")
	if participant == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "participant is required"})
		return
	}

	result, err := h.repo.GetByParticipant(c.Request.Context(), investigationID, participant, paginate)
	if err != nil {
		h.logger.Error("Failed to get timeline events by participant", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline events by participant"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTimelineStats retrieves timeline statistics for an investigation
func (h *TimelineHandler) GetTimelineStats(c *gin.Context) {
	// Get investigation ID from URL
	investigationIDStr := c.Param("id")
	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	stats, err := h.repo.GetTimelineStats(c.Request.Context(), investigationID)
	if err != nil {
		h.logger.Error("Failed to get timeline stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// SearchTimelineEvents performs search across timeline events
func (h *TimelineHandler) SearchTimelineEvents(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	// Get investigation ID if provided
	investigationIDStr := c.Query("investigation_id")
	if investigationIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Investigation ID required for timeline search"})
		return
	}

	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	// Parse search parameters
	searchTerm := c.Query("q")
	
	// Parse event types filter
	var eventTypes []models.EventType
	if eventTypesStr := c.Query("event_types"); eventTypesStr != "" {
		// In a real implementation, you'd parse comma-separated values
		// For now, simplified
	}

	// Parse date range
	var startDate, endDate *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = &parsed
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = &parsed
		}
	}

	result, err := h.repo.Search(c.Request.Context(), investigationID, searchTerm, eventTypes, startDate, endDate, paginate)
	if err != nil {
		h.logger.Error("Failed to search timeline events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search timeline events"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTimelineByEvidence retrieves timeline events related to specific evidence
func (h *TimelineHandler) GetTimelineByEvidence(c *gin.Context) {
	evidenceIDStr := c.Query("evidence_id")
	if evidenceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Evidence ID required"})
		return
	}

	evidenceID, err := uuid.Parse(evidenceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	result, err := h.repo.GetByEvidenceID(c.Request.Context(), evidenceID, paginate)
	if err != nil {
		h.logger.Error("Failed to get timeline events by evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline events by evidence"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BulkCreateTimelineEvents creates multiple timeline events
func (h *TimelineHandler) BulkCreateTimelineEvents(c *gin.Context) {
	// Get investigation ID from URL
	investigationIDStr := c.Param("id")
	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	var req struct {
		Events []models.CreateTimelineRequest `json:"events" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Get user ID from context (would come from auth middleware)
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Convert to pointers
	requests := make([]*models.CreateTimelineRequest, len(req.Events))
	for i := range req.Events {
		requests[i] = &req.Events[i]
	}

	timelines, err := h.repo.BulkCreate(c.Request.Context(), investigationID, requests, userID)
	if err != nil {
		h.logger.Error("Failed to bulk create timeline events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk create timeline events"})
		return
	}

	h.logger.Info("Timeline events bulk created", 
		zap.String("investigation_id", investigationID.String()),
		zap.Int("count", len(timelines)))
	c.JSON(http.StatusCreated, gin.H{
		"message": "Timeline events created successfully",
		"count":   len(timelines),
		"events":  timelines,
	})
}