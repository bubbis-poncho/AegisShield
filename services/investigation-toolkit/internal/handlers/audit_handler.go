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

type AuditHandler struct {
	auditRepo repository.AuditRepository
}

func NewAuditHandler(auditRepo repository.AuditRepository) *AuditHandler {
	return &AuditHandler{
		auditRepo: auditRepo,
	}
}

// Audit Logs
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	idParam := c.Param("id")
	logID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid audit log ID format"})
		return
	}

	log, err := h.auditRepo.GetAuditLog(c.Request.Context(), logID)
	if err != nil {
		if err.Error() == "audit log not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Audit log not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit log", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, log)
}

func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var filter models.AuditLogFilter

	// Parse query parameters
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if action := c.Query("action"); action != "" {
		filter.Action = action
	}

	if entityType := c.Query("entity_type"); entityType != "" {
		filter.EntityType = entityType
	}

	if entityIDStr := c.Query("entity_id"); entityIDStr != "" {
		if entityID, err := uuid.Parse(entityIDStr); err == nil {
			filter.EntityID = &entityID
		}
	}

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

	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filter.IPAddress = ipAddress
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	logs, total, err := h.auditRepo.ListAuditLogs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list audit logs", "details": err.Error()})
		return
	}

	response := models.ListAuditLogsResponse{
		Logs:   logs,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuditHandler) GetAuditLogsByEntity(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityIDParam := c.Param("entity_id")
	entityID, err := uuid.Parse(entityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity ID format"})
		return
	}

	logs, err := h.auditRepo.GetAuditLogsByEntity(c.Request.Context(), entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs by entity", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_logs": logs})
}

func (h *AuditHandler) GetAuditLogsByUser(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	limitStr := c.Query("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := h.auditRepo.GetAuditLogsByUser(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs by user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_logs": logs})
}

// Chain of Custody
func (h *AuditHandler) GetChainOfCustody(c *gin.Context) {
	evidenceIDParam := c.Param("evidence_id")
	evidenceID, err := uuid.Parse(evidenceIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID format"})
		return
	}

	entries, err := h.auditRepo.GetChainOfCustody(c.Request.Context(), evidenceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chain of custody", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chain_of_custody": entries})
}

func (h *AuditHandler) VerifyChainOfCustody(c *gin.Context) {
	evidenceIDParam := c.Param("evidence_id")
	evidenceID, err := uuid.Parse(evidenceIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evidence ID format"})
		return
	}

	verification, err := h.auditRepo.VerifyChainOfCustody(c.Request.Context(), evidenceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify chain of custody", "details": err.Error()})
		return
	}

	// Log the verification attempt
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			auditLog := &models.AuditLog{
				UserID:      &userID,
				Action:      "verify_chain_of_custody",
				EntityType:  "evidence",
				EntityID:    &evidenceID,
				Description: "Verified chain of custody",
				NewValues: map[string]interface{}{
					"verification_result": verification,
				},
			}
			h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)
		}
	}

	c.JSON(http.StatusOK, verification)
}

// Data Integrity
func (h *AuditHandler) GetDataIntegrityChecks(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityIDParam := c.Param("entity_id")
	entityID, err := uuid.Parse(entityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity ID format"})
		return
	}

	checks, err := h.auditRepo.GetDataIntegrityChecks(c.Request.Context(), entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get data integrity checks", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"integrity_checks": checks})
}

func (h *AuditHandler) VerifyDataIntegrity(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityIDParam := c.Param("entity_id")
	entityID, err := uuid.Parse(entityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity ID format"})
		return
	}

	result, err := h.auditRepo.VerifyDataIntegrity(c.Request.Context(), entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify data integrity", "details": err.Error()})
		return
	}

	// Log the verification attempt
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			auditLog := &models.AuditLog{
				UserID:      &userID,
				Action:      "verify_data_integrity",
				EntityType:  entityType,
				EntityID:    &entityID,
				Description: "Verified data integrity",
				NewValues: map[string]interface{}{
					"verification_result": result,
				},
			}
			h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)
		}
	}

	c.JSON(http.StatusOK, result)
}

// User Access Logs
func (h *AuditHandler) GetUserAccessLogs(c *gin.Context) {
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

	logs, err := h.auditRepo.GetUserAccessLogs(c.Request.Context(), userID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user access logs", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_logs": logs})
}

func (h *AuditHandler) GetResourceAccessLogs(c *gin.Context) {
	resource := c.Param("resource")
	if resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resource parameter is required"})
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

	logs, err := h.auditRepo.GetResourceAccessLogs(c.Request.Context(), resource, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get resource access logs", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_logs": logs})
}

// Compliance Reports
func (h *AuditHandler) GenerateComplianceReport(c *gin.Context) {
	var filter models.ComplianceReportFilter

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

	// Default to last 30 days if no dates provided
	if filter.DateFrom.IsZero() {
		filter.DateFrom = time.Now().AddDate(0, 0, -30)
	}
	if filter.DateTo.IsZero() {
		filter.DateTo = time.Now()
	}

	report, err := h.auditRepo.GenerateComplianceReport(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate compliance report", "details": err.Error()})
		return
	}

	// Log the report generation
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			auditLog := &models.AuditLog{
				UserID:      &userID,
				Action:      "generate_compliance_report",
				EntityType:  "compliance_report",
				Description: "Generated compliance report",
				NewValues: map[string]interface{}{
					"date_from": filter.DateFrom,
					"date_to":   filter.DateTo,
				},
			}
			h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)
		}
	}

	c.JSON(http.StatusOK, report)
}

func (h *AuditHandler) GetAuditSummary(c *gin.Context) {
	var filter models.AuditSummaryFilter

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

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if entityType := c.Query("entity_type"); entityType != "" {
		filter.EntityType = entityType
	}

	// Default to last 7 days if no dates provided
	if filter.DateFrom.IsZero() {
		filter.DateFrom = time.Now().AddDate(0, 0, -7)
	}
	if filter.DateTo.IsZero() {
		filter.DateTo = time.Now()
	}

	summary, err := h.auditRepo.GetAuditSummary(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit summary", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *AuditHandler) GetUserActivitySummary(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")

	// Default to last 30 days if no dates provided
	dateFrom := time.Now().AddDate(0, 0, -30)
	dateTo := time.Now()

	if dateFromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			dateFrom = parsed
		}
	}

	if dateToStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			dateTo = parsed
		}
	}

	summary, err := h.auditRepo.GetUserActivitySummary(c.Request.Context(), userID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user activity summary", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// Retention and Archival
func (h *AuditHandler) GetAuditLogRetentionStats(c *gin.Context) {
	stats, err := h.auditRepo.GetAuditLogRetentionStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit log retention stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AuditHandler) ArchiveOldAuditLogs(c *gin.Context) {
	var req models.ArchiveAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	retentionPeriod := time.Duration(req.RetentionDays) * 24 * time.Hour
	archivedCount, err := h.auditRepo.ArchiveOldAuditLogs(c.Request.Context(), retentionPeriod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive audit logs", "details": err.Error()})
		return
	}

	// Log the archival operation
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			auditLog := &models.AuditLog{
				UserID:      &userID,
				Action:      "archive_audit_logs",
				EntityType:  "audit_system",
				Description: "Archived old audit logs",
				NewValues: map[string]interface{}{
					"retention_days":  req.RetentionDays,
					"archived_count":  archivedCount,
					"initiated_by":    req.InitiatedBy,
				},
			}
			h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)
		}
	}

	response := models.ArchiveAuditLogsResponse{
		ArchivedCount: archivedCount,
		RetentionDays: req.RetentionDays,
		InitiatedBy:   req.InitiatedBy,
		InitiatedAt:   time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuditHandler) PurgeArchivedLogs(c *gin.Context) {
	var req models.PurgeArchivedLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	purgedCount, err := h.auditRepo.PurgeArchivedLogs(c.Request.Context(), req.ArchivalDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge archived logs", "details": err.Error()})
		return
	}

	// Log the purge operation
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			auditLog := &models.AuditLog{
				UserID:      &userID,
				Action:      "purge_archived_audit_logs",
				EntityType:  "audit_system",
				Description: "Purged archived audit logs",
				NewValues: map[string]interface{}{
					"archival_date": req.ArchivalDate,
					"purged_count":  purgedCount,
					"initiated_by":  req.InitiatedBy,
				},
			}
			h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)
		}
	}

	response := models.PurgeArchivedLogsResponse{
		PurgedCount:   purgedCount,
		ArchivalDate:  req.ArchivalDate,
		InitiatedBy:   req.InitiatedBy,
		InitiatedAt:   time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// Monitoring and Alerting
func (h *AuditHandler) GetSuspiciousActivities(c *gin.Context) {
	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")

	// Default to last 24 hours
	dateFrom := time.Now().AddDate(0, 0, -1)
	dateTo := time.Now()

	if dateFromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			dateFrom = parsed
		}
	}

	if dateToStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			dateTo = parsed
		}
	}

	// Query for suspicious patterns
	filter := models.AuditLogFilter{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Limit:    1000,
	}

	logs, _, err := h.auditRepo.ListAuditLogs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs for analysis", "details": err.Error()})
		return
	}

	// Analyze for suspicious patterns
	suspiciousActivities := h.analyzeSuspiciousPatterns(logs)

	c.JSON(http.StatusOK, gin.H{
		"suspicious_activities": suspiciousActivities,
		"analysis_period": gin.H{
			"from": dateFrom,
			"to":   dateTo,
		},
		"total_logs_analyzed": len(logs),
	})
}

// Helper function to analyze suspicious patterns
func (h *AuditHandler) analyzeSuspiciousPatterns(logs []*models.AuditLog) []models.SuspiciousActivity {
	var activities []models.SuspiciousActivity

	// Group logs by user and analyze patterns
	userActivityCount := make(map[uuid.UUID]int)
	userFailedLogins := make(map[uuid.UUID]int)
	userIPChanges := make(map[uuid.UUID]map[string]bool)

	for _, log := range logs {
		if log.UserID != nil {
			userActivityCount[*log.UserID]++

			// Track failed login attempts
			if log.Action == "access_failed_login" {
				userFailedLogins[*log.UserID]++
			}

			// Track IP address changes
			if log.IPAddress != nil {
				if userIPChanges[*log.UserID] == nil {
					userIPChanges[*log.UserID] = make(map[string]bool)
				}
				userIPChanges[*log.UserID][*log.IPAddress] = true
			}
		}
	}

	// Detect suspicious patterns
	for userID, count := range userActivityCount {
		// High activity volume
		if count > 100 {
			activities = append(activities, models.SuspiciousActivity{
				Type:        "high_activity_volume",
				UserID:      &userID,
				Description: "User has unusually high activity volume",
				Severity:    "medium",
				Count:       count,
				DetectedAt:  time.Now(),
			})
		}

		// Multiple failed logins
		if failedCount := userFailedLogins[userID]; failedCount > 5 {
			activities = append(activities, models.SuspiciousActivity{
				Type:        "multiple_failed_logins",
				UserID:      &userID,
				Description: "User has multiple failed login attempts",
				Severity:    "high",
				Count:       failedCount,
				DetectedAt:  time.Now(),
			})
		}

		// Multiple IP addresses
		if ipCount := len(userIPChanges[userID]); ipCount > 3 {
			activities = append(activities, models.SuspiciousActivity{
				Type:        "multiple_ip_addresses",
				UserID:      &userID,
				Description: "User accessed from multiple IP addresses",
				Severity:    "medium",
				Count:       ipCount,
				DetectedAt:  time.Now(),
			})
		}
	}

	return activities
}