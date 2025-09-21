package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"investigation-toolkit/internal/database"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db     *database.Database
	logger *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.Database, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger.Named("health_handler"),
	}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "investigation-toolkit",
		"version": "1.0.0",
	})
}

// Ready returns readiness status including database connectivity
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Check database connectivity
	if err := h.db.Health(ctx); err != nil {
		h.logger.Error("Database health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "connected",
	})
}

// Live returns liveness status
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}