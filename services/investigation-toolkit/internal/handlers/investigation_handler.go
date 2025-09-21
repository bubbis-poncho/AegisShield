package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/models"
	"investigation-toolkit/internal/repository"
)

// InvestigationHandler handles HTTP requests for investigations
type InvestigationHandler struct {
	repo   *repository.InvestigationRepository
	logger *zap.Logger
}

// NewInvestigationHandler creates a new investigation handler
func NewInvestigationHandler(repo *repository.InvestigationRepository, logger *zap.Logger) *InvestigationHandler {
	return &InvestigationHandler{
		repo:   repo,
		logger: logger.Named("investigation_handler"),
	}
}

// CreateInvestigation creates a new investigation
func (h *InvestigationHandler) CreateInvestigation(c *gin.Context) {
	var req models.CreateInvestigationRequest
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

	investigation, err := h.repo.Create(c.Request.Context(), &req, userID)
	if err != nil {
		h.logger.Error("Failed to create investigation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create investigation"})
		return
	}

	h.logger.Info("Investigation created", zap.String("id", investigation.ID.String()))
	c.JSON(http.StatusCreated, investigation)
}

// GetInvestigation retrieves an investigation by ID
func (h *InvestigationHandler) GetInvestigation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	investigation, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "investigation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Investigation not found"})
			return
		}
		h.logger.Error("Failed to get investigation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get investigation"})
		return
	}

	c.JSON(http.StatusOK, investigation)
}

// UpdateInvestigation updates an investigation
func (h *InvestigationHandler) UpdateInvestigation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	var req models.UpdateInvestigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	investigation, err := h.repo.Update(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "investigation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Investigation not found"})
			return
		}
		h.logger.Error("Failed to update investigation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update investigation"})
		return
	}

	h.logger.Info("Investigation updated", zap.String("id", id.String()))
	c.JSON(http.StatusOK, investigation)
}

// DeleteInvestigation deletes an investigation
func (h *InvestigationHandler) DeleteInvestigation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	err = h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "investigation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Investigation not found"})
			return
		}
		h.logger.Error("Failed to delete investigation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete investigation"})
		return
	}

	h.logger.Info("Investigation deleted", zap.String("id", id.String()))
	c.JSON(http.StatusNoContent, nil)
}

// ListInvestigations lists investigations with filtering and pagination
func (h *InvestigationHandler) ListInvestigations(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	// Parse filter parameters
	filter := &models.InvestigationFilter{}
	
	// Case types
	if caseTypesStr := c.Query("case_types"); caseTypesStr != "" {
		// In a real implementation, you'd parse comma-separated values
		// For now, simplified
	}

	// Search term
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	// Add other filter parsing as needed

	result, err := h.repo.List(c.Request.Context(), filter, paginate)
	if err != nil {
		h.logger.Error("Failed to list investigations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list investigations"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetInvestigationStats retrieves investigation statistics
func (h *InvestigationHandler) GetInvestigationStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	// First verify the investigation exists
	_, err = h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "investigation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Investigation not found"})
			return
		}
		h.logger.Error("Failed to get investigation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get investigation"})
		return
	}

	stats, err := h.repo.GetInvestigationStats(c.Request.Context(), nil)
	if err != nil {
		h.logger.Error("Failed to get investigation stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get investigation stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// UpdateAssignment updates the assignment of an investigation
func (h *InvestigationHandler) UpdateAssignment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	var req struct {
		AssignedTo *uuid.UUID `json:"assigned_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.repo.UpdateAssignment(c.Request.Context(), id, req.AssignedTo)
	if err != nil {
		if err.Error() == "investigation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Investigation not found"})
			return
		}
		h.logger.Error("Failed to update investigation assignment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment"})
		return
	}

	h.logger.Info("Investigation assignment updated", zap.String("id", id.String()))
	c.JSON(http.StatusOK, gin.H{"message": "Assignment updated successfully"})
}

// UpdateStatus updates the status of an investigation
func (h *InvestigationHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	var req struct {
		Status models.Status `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.repo.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if err.Error() == "investigation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Investigation not found"})
			return
		}
		h.logger.Error("Failed to update investigation status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	h.logger.Info("Investigation status updated", zap.String("id", id.String()), zap.String("status", string(req.Status)))
	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// GetAssignedInvestigations retrieves investigations assigned to the current user
func (h *InvestigationHandler) GetAssignedInvestigations(c *gin.Context) {
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

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	result, err := h.repo.GetAssignedInvestigations(c.Request.Context(), userID, paginate)
	if err != nil {
		h.logger.Error("Failed to get assigned investigations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get assigned investigations"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUserDashboard retrieves dashboard data for the current user
func (h *InvestigationHandler) GetUserDashboard(c *gin.Context) {
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

	stats, err := h.repo.GetInvestigationStats(c.Request.Context(), &userID)
	if err != nil {
		h.logger.Error("Failed to get user dashboard stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard data"})
		return
	}

	// Get recent assigned investigations
	paginate := database.NewPaginate(10, 0) // Latest 10
	recentInvestigations, err := h.repo.GetAssignedInvestigations(c.Request.Context(), userID, paginate)
	if err != nil {
		h.logger.Error("Failed to get recent investigations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard data"})
		return
	}

	dashboard := map[string]interface{}{
		"stats":                 stats,
		"recent_investigations": recentInvestigations,
	}

	c.JSON(http.StatusOK, dashboard)
}

// SearchInvestigations performs search across investigations
func (h *InvestigationHandler) SearchInvestigations(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	// Parse search parameters
	filter := &models.InvestigationFilter{}
	if search := c.Query("q"); search != "" {
		filter.Search = &search
	}

	// Add additional search filters as needed

	result, err := h.repo.List(c.Request.Context(), filter, paginate)
	if err != nil {
		h.logger.Error("Failed to search investigations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search investigations"})
		return
	}

	c.JSON(http.StatusOK, result)
}