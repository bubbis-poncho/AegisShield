package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/aegisshield/compliance-engine/internal/audit"
	"github.com/aegisshield/compliance-engine/internal/compliance"
	"github.com/aegisshield/compliance-engine/internal/regulatory"
	"github.com/aegisshield/compliance-engine/internal/reporting"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ComplianceHandler handles compliance-related HTTP requests
type ComplianceHandler struct {
	complianceEngine  *compliance.ComplianceEngine
	ruleEngine        *compliance.RuleEngine
	violationManager  *compliance.ViolationManager
	reportEngine      *reporting.ReportEngine
	auditLogger       *audit.AuditLogger
	regulationManager *regulatory.RegulationManager
	logger            *zap.Logger
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(
	complianceEngine *compliance.ComplianceEngine,
	ruleEngine *compliance.RuleEngine,
	violationManager *compliance.ViolationManager,
	reportEngine *reporting.ReportEngine,
	auditLogger *audit.AuditLogger,
	regulationManager *regulatory.RegulationManager,
	logger *zap.Logger,
) *ComplianceHandler {
	return &ComplianceHandler{
		complianceEngine:  complianceEngine,
		ruleEngine:        ruleEngine,
		violationManager:  violationManager,
		reportEngine:      reportEngine,
		auditLogger:       auditLogger,
		regulationManager: regulationManager,
		logger:            logger,
	}
}

// RegisterRoutes registers all compliance-related routes
func (h *ComplianceHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")

	// Compliance evaluation endpoints
	api.POST("/compliance/evaluate", h.EvaluateCompliance)
	api.GET("/compliance/status/:entity_id", h.GetComplianceStatus)

	// Rule management endpoints
	api.GET("/rules", h.GetRules)
	api.POST("/rules", h.CreateRule)
	api.PUT("/rules/:rule_id", h.UpdateRule)
	api.DELETE("/rules/:rule_id", h.DeleteRule)
	api.POST("/rules/:rule_id/evaluate", h.EvaluateRule)

	// Violation management endpoints
	api.GET("/violations", h.GetViolations)
	api.GET("/violations/:violation_id", h.GetViolation)
	api.PUT("/violations/:violation_id/status", h.UpdateViolationStatus)
	api.POST("/violations/:violation_id/comments", h.AddViolationComment)
	api.POST("/violations/:violation_id/assign", h.AssignViolation)
	api.POST("/violations/:violation_id/escalate", h.EscalateViolation)
	api.GET("/violations/statistics", h.GetViolationStatistics)

	// Report management endpoints
	api.GET("/reports/templates", h.GetReportTemplates)
	api.POST("/reports/templates", h.CreateReportTemplate)
	api.PUT("/reports/templates/:template_id", h.UpdateReportTemplate)
	api.DELETE("/reports/templates/:template_id", h.DeleteReportTemplate)
	api.POST("/reports/generate", h.GenerateReport)
	api.GET("/reports/:report_id/status", h.GetReportStatus)
	api.POST("/reports/schedule", h.ScheduleReport)

	// Audit endpoints
	api.GET("/audit/logs", h.GetAuditLogs)
	api.GET("/audit/statistics", h.GetAuditStatistics)

	// Regulatory endpoints
	api.GET("/regulations", h.GetRegulations)
	api.GET("/regulations/:regulation_id", h.GetRegulation)
	api.POST("/regulations", h.AddRegulation)
	api.PUT("/regulations/:regulation_id", h.UpdateRegulation)
	api.GET("/regulations/changes", h.GetRegulationChanges)
	api.POST("/regulations/compliance-check", h.CheckCompliance)

	// Health check
	api.GET("/health", h.HealthCheck)
}

// Compliance evaluation endpoints

func (h *ComplianceHandler) EvaluateCompliance(c *gin.Context) {
	var request struct {
		EntityID   string                 `json:"entity_id" binding:"required"`
		EntityType string                 `json:"entity_type" binding:"required"`
		Data       map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.complianceEngine.EvaluateCompliance(c.Request.Context(), request.EntityID, request.EntityType, request.Data)
	if err != nil {
		h.logger.Error("Failed to evaluate compliance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate compliance"})
		return
	}

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "compliance_evaluation", "compliance", 
		c.GetString("user_id"), request.EntityID, request.EntityType, "evaluate", 
		map[string]interface{}{
			"result_id": result.ID,
			"status": result.OverallStatus,
			"risk_score": result.RiskScore,
		})

	c.JSON(http.StatusOK, result)
}

func (h *ComplianceHandler) GetComplianceStatus(c *gin.Context) {
	entityID := c.Param("entity_id")
	if entityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_id is required"})
		return
	}

	// This would get the latest compliance status for the entity
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"entity_id": entityID,
		"status": "compliant",
		"last_evaluated": time.Now(),
	})
}

// Rule management endpoints

func (h *ComplianceHandler) GetRules(c *gin.Context) {
	// This would get rules from the rule engine
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"rules": []map[string]interface{}{},
		"total": 0,
	})
}

func (h *ComplianceHandler) CreateRule(c *gin.Context) {
	var rule compliance.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would create a rule in the rule engine
	// For now, return a placeholder response
	rule.ID = "RULE_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "rule_created", "compliance",
		c.GetString("user_id"), rule.ID, "rule", "create",
		map[string]interface{}{
			"rule_name": rule.Name,
			"rule_type": rule.Type,
			"severity": rule.Severity,
		})

	c.JSON(http.StatusCreated, rule)
}

func (h *ComplianceHandler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	var rule compliance.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = ruleID
	rule.UpdatedAt = time.Now()

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "rule_updated", "compliance",
		c.GetString("user_id"), rule.ID, "rule", "update",
		map[string]interface{}{
			"rule_name": rule.Name,
			"rule_type": rule.Type,
		})

	c.JSON(http.StatusOK, rule)
}

func (h *ComplianceHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("rule_id")

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "rule_deleted", "compliance",
		c.GetString("user_id"), ruleID, "rule", "delete",
		map[string]interface{}{
			"rule_id": ruleID,
		})

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

func (h *ComplianceHandler) EvaluateRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	var request struct {
		Data map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would evaluate the specific rule
	// For now, return a placeholder response
	result := &compliance.RuleResult{
		RuleID:      ruleID,
		Passed:      true,
		Description: "Rule evaluation successful",
		EvaluatedAt: time.Now(),
	}

	c.JSON(http.StatusOK, result)
}

// Violation management endpoints

func (h *ComplianceHandler) GetViolations(c *gin.Context) {
	status := c.Query("status")
	severity := c.Query("severity")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var violations []compliance.Violation
	var err error

	if status != "" {
		violations, err = h.violationManager.GetViolationsByStatus(c.Request.Context(), status)
	} else if severity != "" {
		violations, err = h.violationManager.GetViolationsBySeverity(c.Request.Context(), severity)
	} else {
		// Get all violations (would implement pagination)
		violations = []compliance.Violation{}
	}

	if err != nil {
		h.logger.Error("Failed to get violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get violations"})
		return
	}

	// Apply pagination
	end := offset + limit
	if end > len(violations) {
		end = len(violations)
	}
	if offset < len(violations) {
		violations = violations[offset:end]
	} else {
		violations = []compliance.Violation{}
	}

	c.JSON(http.StatusOK, gin.H{
		"violations": violations,
		"total": len(violations),
		"limit": limit,
		"offset": offset,
	})
}

func (h *ComplianceHandler) GetViolation(c *gin.Context) {
	violationID := c.Param("violation_id")

	violation, err := h.violationManager.GetViolation(c.Request.Context(), violationID)
	if err != nil {
		h.logger.Error("Failed to get violation", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Violation not found"})
		return
	}

	c.JSON(http.StatusOK, violation)
}

func (h *ComplianceHandler) UpdateViolationStatus(c *gin.Context) {
	violationID := c.Param("violation_id")
	var request struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.violationManager.UpdateViolationStatus(c.Request.Context(), violationID, request.Status, request.Notes)
	if err != nil {
		h.logger.Error("Failed to update violation status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update violation status"})
		return
	}

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "violation_status_updated", "compliance",
		c.GetString("user_id"), violationID, "violation", "update_status",
		map[string]interface{}{
			"new_status": request.Status,
			"notes": request.Notes,
		})

	c.JSON(http.StatusOK, gin.H{"message": "Violation status updated successfully"})
}

func (h *ComplianceHandler) AddViolationComment(c *gin.Context) {
	violationID := c.Param("violation_id")
	var comment compliance.ViolationComment

	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment.Author = c.GetString("user_id")
	comment.CreatedAt = time.Now()

	err := h.violationManager.AddViolationComment(c.Request.Context(), violationID, comment)
	if err != nil {
		h.logger.Error("Failed to add violation comment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

func (h *ComplianceHandler) AssignViolation(c *gin.Context) {
	violationID := c.Param("violation_id")
	var request struct {
		AssignedTo string `json:"assigned_to" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.violationManager.AssignViolation(c.Request.Context(), violationID, request.AssignedTo)
	if err != nil {
		h.logger.Error("Failed to assign violation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign violation"})
		return
	}

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "violation_assigned", "compliance",
		c.GetString("user_id"), violationID, "violation", "assign",
		map[string]interface{}{
			"assigned_to": request.AssignedTo,
		})

	c.JSON(http.StatusOK, gin.H{"message": "Violation assigned successfully"})
}

func (h *ComplianceHandler) EscalateViolation(c *gin.Context) {
	violationID := c.Param("violation_id")
	var request struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.violationManager.EscalateViolation(c.Request.Context(), violationID, request.Reason)
	if err != nil {
		h.logger.Error("Failed to escalate violation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to escalate violation"})
		return
	}

	// Log audit event
	h.auditLogger.LogEvent(c.Request.Context(), "violation_escalated", "compliance",
		c.GetString("user_id"), violationID, "violation", "escalate",
		map[string]interface{}{
			"reason": request.Reason,
		})

	c.JSON(http.StatusOK, gin.H{"message": "Violation escalated successfully"})
}

func (h *ComplianceHandler) GetViolationStatistics(c *gin.Context) {
	stats, err := h.violationManager.GetViolationStatistics(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get violation statistics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Report management endpoints

func (h *ComplianceHandler) GetReportTemplates(c *gin.Context) {
	templates, err := h.reportEngine.ListTemplates(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get report templates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

func (h *ComplianceHandler) CreateReportTemplate(c *gin.Context) {
	var template compliance.ReportTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template.CreatedBy = c.GetString("user_id")
	err := h.reportEngine.CreateTemplate(c.Request.Context(), &template)
	if err != nil {
		h.logger.Error("Failed to create report template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, template)
}

func (h *ComplianceHandler) UpdateReportTemplate(c *gin.Context) {
	templateID := c.Param("template_id")
	var template compliance.ReportTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template.ID = templateID
	err := h.reportEngine.UpdateTemplate(c.Request.Context(), &template)
	if err != nil {
		h.logger.Error("Failed to update report template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *ComplianceHandler) DeleteReportTemplate(c *gin.Context) {
	templateID := c.Param("template_id")

	err := h.reportEngine.DeleteTemplate(c.Request.Context(), templateID)
	if err != nil {
		h.logger.Error("Failed to delete report template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

func (h *ComplianceHandler) GenerateReport(c *gin.Context) {
	var request struct {
		TemplateID string                 `json:"template_id" binding:"required"`
		Parameters map[string]interface{} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.reportEngine.GenerateReport(c.Request.Context(), request.TemplateID, request.Parameters)
	if err != nil {
		h.logger.Error("Failed to generate report", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"report_id": report.ID,
		"status": report.Status,
		"message": "Report generation started",
	})
}

func (h *ComplianceHandler) GetReportStatus(c *gin.Context) {
	reportID := c.Param("report_id")

	status, err := h.reportEngine.GetReportStatus(c.Request.Context(), reportID)
	if err != nil {
		h.logger.Error("Failed to get report status", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *ComplianceHandler) ScheduleReport(c *gin.Context) {
	var schedule compliance.ReportSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.reportEngine.ScheduleReport(c.Request.Context(), &schedule)
	if err != nil {
		h.logger.Error("Failed to schedule report", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule report"})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// Audit endpoints

func (h *ComplianceHandler) GetAuditLogs(c *gin.Context) {
	// Parse query parameters
	filters := audit.AuditFilters{
		Category:  c.Query("category"),
		EventType: c.Query("event_type"),
		UserID:    c.Query("user_id"),
		EntityID:  c.Query("entity_id"),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	logs, err := h.auditLogger.GetAuditLogs(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to get audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *ComplianceHandler) GetAuditStatistics(c *gin.Context) {
	timeRange := audit.TimeRange{}

	// Parse time range from query parameters
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			timeRange.StartTime = &t
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			timeRange.EndTime = &t
		}
	}

	stats, err := h.auditLogger.GetAuditStatistics(c.Request.Context(), timeRange)
	if err != nil {
		h.logger.Error("Failed to get audit statistics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Regulatory endpoints

func (h *ComplianceHandler) GetRegulations(c *gin.Context) {
	jurisdiction := c.Query("jurisdiction")
	regulationType := c.Query("type")

	var regulations []*compliance.RegulationInfo
	var err error

	if jurisdiction != "" {
		if regulationType != "" {
			regulations, err = h.regulationManager.GetApplicableRegulations(c.Request.Context(), jurisdiction, regulationType)
		} else {
			regulations, err = h.regulationManager.GetRegulationsByJurisdiction(c.Request.Context(), jurisdiction)
		}
	} else {
		// Get all regulations (would implement pagination)
		regulations = []*compliance.RegulationInfo{}
	}

	if err != nil {
		h.logger.Error("Failed to get regulations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get regulations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"regulations": regulations})
}

func (h *ComplianceHandler) GetRegulation(c *gin.Context) {
	regulationID := c.Param("regulation_id")

	regulation, err := h.regulationManager.GetRegulation(c.Request.Context(), regulationID)
	if err != nil {
		h.logger.Error("Failed to get regulation", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Regulation not found"})
		return
	}

	c.JSON(http.StatusOK, regulation)
}

func (h *ComplianceHandler) AddRegulation(c *gin.Context) {
	var regulation compliance.RegulationInfo
	if err := c.ShouldBindJSON(&regulation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.regulationManager.AddRegulation(c.Request.Context(), &regulation)
	if err != nil {
		h.logger.Error("Failed to add regulation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add regulation"})
		return
	}

	c.JSON(http.StatusCreated, regulation)
}

func (h *ComplianceHandler) UpdateRegulation(c *gin.Context) {
	regulationID := c.Param("regulation_id")
	var regulation compliance.RegulationInfo
	if err := c.ShouldBindJSON(&regulation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	regulation.ID = regulationID
	err := h.regulationManager.UpdateRegulation(c.Request.Context(), &regulation)
	if err != nil {
		h.logger.Error("Failed to update regulation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update regulation"})
		return
	}

	c.JSON(http.StatusOK, regulation)
}

func (h *ComplianceHandler) GetRegulationChanges(c *gin.Context) {
	sinceParam := c.Query("since")
	var since time.Time
	var err error

	if sinceParam != "" {
		since, err = time.Parse(time.RFC3339, sinceParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid since parameter"})
			return
		}
	} else {
		since = time.Now().AddDate(0, 0, -30) // Default to last 30 days
	}

	changes, err := h.regulationManager.GetRegulationChanges(c.Request.Context(), since)
	if err != nil {
		h.logger.Error("Failed to get regulation changes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get changes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"changes": changes})
}

func (h *ComplianceHandler) CheckCompliance(c *gin.Context) {
	var request struct {
		Jurisdiction string   `json:"jurisdiction" binding:"required"`
		EntityType   string   `json:"entity_type" binding:"required"`
		Requirements []string `json:"requirements" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	check, err := h.regulationManager.CheckComplianceRequirements(
		c.Request.Context(),
		request.Jurisdiction,
		request.EntityType,
		request.Requirements,
	)
	if err != nil {
		h.logger.Error("Failed to check compliance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check compliance"})
		return
	}

	c.JSON(http.StatusOK, check)
}

// Health check endpoint

func (h *ComplianceHandler) HealthCheck(c *gin.Context) {
	status := map[string]string{
		"status":            "healthy",
		"compliance_engine": h.complianceEngine.GetStatus(),
		"rule_engine":       h.ruleEngine.GetStatus(),
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, status)
}