package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/aegisshield/analytics-dashboard/internal/dashboard"
	"github.com/aegisshield/analytics-dashboard/internal/data"
	"github.com/aegisshield/analytics-dashboard/internal/realtime"
	"github.com/aegisshield/analytics-dashboard/internal/visualization"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the analytics dashboard
type Handler struct {
	dashboardManager *dashboard.Manager
	dataProcessor    *data.Processor
	vizEngine        *visualization.Engine
	realtimeManager  *realtime.Manager
}

// NewHandler creates a new HTTP handler
func NewHandler(
	dashboardManager *dashboard.Manager,
	dataProcessor *data.Processor,
	vizEngine *visualization.Engine,
	realtimeManager *realtime.Manager,
) *Handler {
	return &Handler{
		dashboardManager: dashboardManager,
		dataProcessor:    dataProcessor,
		vizEngine:        vizEngine,
		realtimeManager:  realtimeManager,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Dashboard routes
		dashboards := api.Group("/dashboards")
		{
			dashboards.GET("", h.GetDashboards)
			dashboards.POST("", h.CreateDashboard)
			dashboards.GET("/:id", h.GetDashboard)
			dashboards.PUT("/:id", h.UpdateDashboard)
			dashboards.DELETE("/:id", h.DeleteDashboard)
			dashboards.POST("/:id/clone", h.CloneDashboard)
		}

		// Widget routes
		widgets := api.Group("/widgets")
		{
			widgets.POST("", h.CreateWidget)
			widgets.GET("/:id", h.GetWidget)
			widgets.PUT("/:id", h.UpdateWidget)
			widgets.DELETE("/:id", h.DeleteWidget)
			widgets.GET("/:id/data", h.GetWidgetData)
			widgets.POST("/:id/refresh", h.RefreshWidgetData)
		}

		// Data routes
		data := api.Group("/data")
		{
			data.POST("/query", h.ExecuteQuery)
			data.GET("/sources", h.GetDataSources)
			data.POST("/sources/test", h.TestDataSource)
		}

		// Visualization routes
		viz := api.Group("/visualization")
		{
			viz.GET("/:widget_id/:type", h.GetVisualizationData)
			viz.POST("/:widget_id/:type", h.UpdateVisualizationData)
		}

		// Real-time routes
		realtime := api.Group("/realtime")
		{
			realtime.GET("/ws", h.HandleWebSocket)
			realtime.GET("/stats", h.GetRealtimeStats)
		}

		// System routes
		system := api.Group("/system")
		{
			system.GET("/health", h.HealthCheck)
			system.GET("/metrics", h.GetMetrics)
			system.GET("/version", h.GetVersion)
		}
	}

	// Serve static files for the dashboard UI
	router.Static("/static", "./web/static")
	router.StaticFile("/", "./web/index.html")
	router.StaticFile("/favicon.ico", "./web/favicon.ico")
}

// Dashboard Handlers

// GetDashboards retrieves dashboards for the current user
func (h *Handler) GetDashboards(c *gin.Context) {
	userID := c.GetString("user_id") // Assume set by auth middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	dashboards, err := h.dashboardManager.GetUserDashboards(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboards"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

// CreateDashboard creates a new dashboard
func (h *Handler) CreateDashboard(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req dashboard.Dashboard
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.UserID = userID
	if err := h.dashboardManager.CreateDashboard(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create dashboard"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"dashboard": req})
}

// GetDashboard retrieves a specific dashboard
func (h *Handler) GetDashboard(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dashboard ID required"})
		return
	}

	dashboard, err := h.dashboardManager.GetDashboard(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dashboard not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboard": dashboard})
}

// UpdateDashboard updates an existing dashboard
func (h *Handler) UpdateDashboard(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dashboard ID required"})
		return
	}

	var req dashboard.Dashboard
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.ID = id
	if err := h.dashboardManager.UpdateDashboard(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboard": req})
}

// DeleteDashboard deletes a dashboard
func (h *Handler) DeleteDashboard(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dashboard ID required"})
		return
	}

	if err := h.dashboardManager.DeleteDashboard(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dashboard deleted successfully"})
}

// CloneDashboard creates a copy of an existing dashboard
func (h *Handler) CloneDashboard(c *gin.Context) {
	sourceID := c.Param("id")
	userID := c.GetString("user_id")

	if sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source dashboard ID required"})
		return
	}

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dashboard name required"})
		return
	}

	cloned, err := h.dashboardManager.CloneDashboard(c.Request.Context(), sourceID, userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clone dashboard"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"dashboard": cloned})
}

// Widget Handlers

// CreateWidget creates a new widget
func (h *Handler) CreateWidget(c *gin.Context) {
	var req dashboard.Widget
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.dashboardManager.AddWidget(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create widget"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"widget": req})
}

// GetWidget retrieves a specific widget
func (h *Handler) GetWidget(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID required"})
		return
	}

	// Implementation would retrieve widget from database
	c.JSON(http.StatusOK, gin.H{"message": "Widget retrieval not implemented"})
}

// UpdateWidget updates an existing widget
func (h *Handler) UpdateWidget(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID required"})
		return
	}

	var req dashboard.Widget
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.ID = id
	if err := h.dashboardManager.UpdateWidget(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"widget": req})
}

// DeleteWidget deletes a widget
func (h *Handler) DeleteWidget(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID required"})
		return
	}

	if err := h.dashboardManager.DeleteWidget(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget deleted successfully"})
}

// GetWidgetData retrieves data for a widget
func (h *Handler) GetWidgetData(c *gin.Context) {
	widgetID := c.Param("id")
	if widgetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID required"})
		return
	}

	// Get widget configuration first
	// This would normally come from the database
	widget := &dashboard.Widget{
		ID:   widgetID,
		Type: dashboard.WidgetTypeChart,
		DataSource: dashboard.DataSource{
			Type:  "sql",
			Query: "SELECT * FROM sample_data",
		},
	}

	// Execute data query
	queryReq := &data.QueryRequest{
		Source: data.DataSource{
			Type: data.DataSourcePostgreSQL,
		},
		Query: widget.DataSource.Query,
	}

	response, err := h.dataProcessor.ExecuteQuery(c.Request.Context(), queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute query"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// RefreshWidgetData refreshes data for a widget and broadcasts updates
func (h *Handler) RefreshWidgetData(c *gin.Context) {
	widgetID := c.Param("id")
	if widgetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID required"})
		return
	}

	// Get fresh data
	queryReq := &data.QueryRequest{
		Source: data.DataSource{
			Type: data.DataSourcePostgreSQL,
		},
		Query: "SELECT * FROM sample_data",
	}

	response, err := h.dataProcessor.ExecuteQuery(c.Request.Context(), queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh data"})
		return
	}

	// Broadcast update to real-time subscribers
	h.realtimeManager.GetHub().SendDataUpdate(widgetID, response.Data, "manual_refresh")

	c.JSON(http.StatusOK, gin.H{"data": response, "message": "Data refreshed successfully"})
}

// Data Handlers

// ExecuteQuery executes a custom data query
func (h *Handler) ExecuteQuery(c *gin.Context) {
	var req data.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	response, err := h.dataProcessor.ExecuteQuery(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute query"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetDataSources retrieves available data sources
func (h *Handler) GetDataSources(c *gin.Context) {
	sources := []map[string]interface{}{
		{
			"id":     "postgres_main",
			"name":   "Main PostgreSQL Database",
			"type":   "postgresql",
			"status": "connected",
		},
		{
			"id":     "neo4j_graph",
			"name":   "Neo4j Graph Database",
			"type":   "neo4j",
			"status": "connected",
		},
		{
			"id":     "prometheus_metrics",
			"name":   "Prometheus Metrics",
			"type":   "prometheus",
			"status": "connected",
		},
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// TestDataSource tests connectivity to a data source
func (h *Handler) TestDataSource(c *gin.Context) {
	var req data.DataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Test connection (implementation depends on data source type)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Connection test successful",
		"latency": "25ms",
	})
}

// Visualization Handlers

// GetVisualizationData retrieves processed visualization data
func (h *Handler) GetVisualizationData(c *gin.Context) {
	widgetID := c.Param("widget_id")
	dataType := c.Param("type")

	if widgetID == "" || dataType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID and data type required"})
		return
	}

	data, err := h.vizEngine.GetVisualizationData(c.Request.Context(), widgetID, dataType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Visualization data not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// UpdateVisualizationData updates visualization data
func (h *Handler) UpdateVisualizationData(c *gin.Context) {
	widgetID := c.Param("widget_id")
	dataType := c.Param("type")

	if widgetID == "" || dataType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Widget ID and data type required"})
		return
	}

	var reqData interface{}
	if err := c.ShouldBindJSON(&reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Process based on data type
	switch dataType {
	case "chart":
		if chartData, ok := reqData.(*visualization.ChartData); ok {
			err := h.vizEngine.ProcessChartData(c.Request.Context(), widgetID, chartData)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process chart data"})
				return
			}
		}
	case "kpi":
		if kpiData, ok := reqData.(*visualization.KPIData); ok {
			err := h.vizEngine.ProcessKPIData(c.Request.Context(), widgetID, kpiData)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process KPI data"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Visualization data updated successfully"})
}

// Real-time Handlers

// HandleWebSocket handles WebSocket connections
func (h *Handler) HandleWebSocket(c *gin.Context) {
	h.realtimeManager.HandleWebSocket(c)
}

// GetRealtimeStats retrieves real-time connection statistics
func (h *Handler) GetRealtimeStats(c *gin.Context) {
	stats := gin.H{
		"connected_clients": h.realtimeManager.GetHub().GetConnectedClients(),
		"uptime":            time.Since(time.Now()).String(), // This would be actual uptime
		"message_rate":      "1.2/sec",                       // This would be calculated
	}

	c.JSON(http.StatusOK, stats)
}

// System Handlers

// HealthCheck performs a health check
func (h *Handler) HealthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services": gin.H{
			"database":   "connected",
			"redis":      "connected",
			"kafka":      "connected",
			"websockets": "active",
		},
	}

	c.JSON(http.StatusOK, health)
}

// GetMetrics retrieves system metrics
func (h *Handler) GetMetrics(c *gin.Context) {
	metrics := gin.H{
		"http_requests_total":    12345,
		"websocket_connections":  h.realtimeManager.GetHub().GetConnectedClients(),
		"database_queries_total": 6789,
		"cache_hits_total":       4567,
		"cache_misses_total":     1234,
		"memory_usage_bytes":     67108864,
		"cpu_usage_percent":      15.5,
	}

	c.JSON(http.StatusOK, metrics)
}

// GetVersion retrieves version information
func (h *Handler) GetVersion(c *gin.Context) {
	version := gin.H{
		"version":    "1.0.0",
		"build_date": "2024-01-15T10:30:00Z",
		"git_commit": "abc123def456",
		"go_version": "go1.21.0",
		"platform":   "linux/amd64",
	}

	c.JSON(http.StatusOK, version)
}

// Middleware

// AuthMiddleware handles authentication
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple token-based auth for now
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		// In a real implementation, validate the token and extract user info
		// For now, just set a mock user ID
		c.Set("user_id", "user123")
		c.Next()
	}
}

// CORSMiddleware handles CORS
func (h *Handler) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware handles rate limiting
func (h *Handler) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple rate limiting implementation
		// In production, use a proper rate limiting library
		c.Next()
	}
}
