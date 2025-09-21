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

// EvidenceHandler handles HTTP requests for evidence
type EvidenceHandler struct {
	repo   *repository.EvidenceRepository
	logger *zap.Logger
}

// NewEvidenceHandler creates a new evidence handler
func NewEvidenceHandler(repo *repository.EvidenceRepository, logger *zap.Logger) *EvidenceHandler {
	return &EvidenceHandler{
		repo:   repo,
		logger: logger.Named("evidence_handler"),
	}
}

// CreateEvidence creates new evidence for an investigation
func (h *EvidenceHandler) CreateEvidence(c *gin.Context) {
	// Get investigation ID from URL
	investigationIDStr := c.Param("id")
	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	var req models.CreateEvidenceRequest
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

	evidence, err := h.repo.Create(c.Request.Context(), investigationID, &req, userID)
	if err != nil {
		h.logger.Error("Failed to create evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create evidence"})
		return
	}

	h.logger.Info("Evidence created", zap.String("id", evidence.ID.String()))
	c.JSON(http.StatusCreated, evidence)
}

// GetEvidence retrieves evidence by ID
func (h *EvidenceHandler) GetEvidence(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
		return
	}

	evidence, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to get evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get evidence"})
		return
	}

	c.JSON(http.StatusOK, evidence)
}

// ListEvidence lists evidence for an investigation
func (h *EvidenceHandler) ListEvidence(c *gin.Context) {
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

	// Parse filter parameters
	filter := &models.EvidenceFilter{}
	
	// Search term
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	// Evidence type filter
	if evidenceType := c.Query("type"); evidenceType != "" {
		// In a real implementation, you'd parse this properly
	}

	// Authentication filter
	if authStr := c.Query("authenticated"); authStr != "" {
		if auth, err := strconv.ParseBool(authStr); err == nil {
			filter.IsAuthenticated = &auth
		}
	}

	result, err := h.repo.GetByInvestigationID(c.Request.Context(), investigationID, filter, paginate)
	if err != nil {
		h.logger.Error("Failed to list evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list evidence"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateEvidenceFile updates file information for evidence
func (h *EvidenceHandler) UpdateEvidenceFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
		return
	}

	var req struct {
		FilePath string `json:"file_path" binding:"required"`
		FileHash string `json:"file_hash" binding:"required"`
		MimeType string `json:"mime_type" binding:"required"`
		FileSize int64  `json:"file_size" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.repo.UpdateFile(c.Request.Context(), id, req.FilePath, req.FileHash, req.MimeType, req.FileSize)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to update evidence file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update evidence file"})
		return
	}

	h.logger.Info("Evidence file updated", zap.String("id", id.String()))
	c.JSON(http.StatusOK, gin.H{"message": "Evidence file updated successfully"})
}

// AuthenticateEvidence marks evidence as authenticated
func (h *EvidenceHandler) AuthenticateEvidence(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
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

	var req struct {
		Method string `json:"method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.repo.Authenticate(c.Request.Context(), id, userID, req.Method)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to authenticate evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate evidence"})
		return
	}

	h.logger.Info("Evidence authenticated", zap.String("id", id.String()), zap.String("method", req.Method))
	c.JSON(http.StatusOK, gin.H{"message": "Evidence authenticated successfully"})
}

// UpdateEvidenceStatus updates the status of evidence
func (h *EvidenceHandler) UpdateEvidenceStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
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

	var req struct {
		Status models.EvidenceStatus `json:"status" binding:"required"`
		Reason string               `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.repo.UpdateStatus(c.Request.Context(), id, req.Status, userID, req.Reason)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to update evidence status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update evidence status"})
		return
	}

	h.logger.Info("Evidence status updated", zap.String("id", id.String()), zap.String("status", string(req.Status)))
	c.JSON(http.StatusOK, gin.H{"message": "Evidence status updated successfully"})
}

// DeleteEvidence deletes evidence
func (h *EvidenceHandler) DeleteEvidence(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
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

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req) // Optional reason

	err = h.repo.Delete(c.Request.Context(), id, userID, req.Reason)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to delete evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete evidence"})
		return
	}

	h.logger.Info("Evidence deleted", zap.String("id", id.String()))
	c.JSON(http.StatusNoContent, nil)
}

// GetChainOfCustody retrieves the chain of custody for evidence
func (h *EvidenceHandler) GetChainOfCustody(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
		return
	}

	evidence, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to get evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get evidence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chain_of_custody": evidence.ChainOfCustody})
}

// AddChainOfCustodyEntry adds an entry to the chain of custody
func (h *EvidenceHandler) AddChainOfCustodyEntry(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID"})
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

	var req struct {
		Action   string `json:"action" binding:"required"`
		Location string `json:"location"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.repo.UpdateChainOfCustody(c.Request.Context(), id, userID, req.Action, req.Location, req.Notes)
	if err != nil {
		if err.Error() == "evidence not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Evidence not found"})
			return
		}
		h.logger.Error("Failed to update chain of custody", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chain of custody"})
		return
	}

	h.logger.Info("Chain of custody updated", zap.String("id", id.String()), zap.String("action", req.Action))
	c.JSON(http.StatusOK, gin.H{"message": "Chain of custody updated successfully"})
}

// SearchEvidence performs search across evidence
func (h *EvidenceHandler) SearchEvidence(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	paginate := database.NewPaginate(limit, offset)

	// Parse search parameters
	filter := &models.EvidenceFilter{}
	if search := c.Query("q"); search != "" {
		filter.Search = &search
	}

	// Get investigation ID if provided
	investigationIDStr := c.Query("investigation_id")
	if investigationIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Investigation ID required for evidence search"})
		return
	}

	investigationID, err := uuid.Parse(investigationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID"})
		return
	}

	result, err := h.repo.GetByInvestigationID(c.Request.Context(), investigationID, filter, paginate)
	if err != nil {
		h.logger.Error("Failed to search evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search evidence"})
		return
	}

	c.JSON(http.StatusOK, result)
}