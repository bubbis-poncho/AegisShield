package audit

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/aegisshield/compliance-engine/internal/compliance"
	"github.com/aegisshield/compliance-engine/internal/config"
	"go.uber.org/zap"
)

// AuditLogger manages audit trail logging and retention
type AuditLogger struct {
	config      config.AuditConfig
	logger      *zap.Logger
	auditLogs   map[string]*compliance.AuditLog
	categories  map[string]*AuditCategory
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
	logChannel  chan *compliance.AuditLog
	batchBuffer []*compliance.AuditLog
	lastFlush   time.Time
}

// AuditCategory represents an audit category configuration
type AuditCategory struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	RetentionPeriod time.Duration `json:"retention_period"`
	EncryptionLevel string        `json:"encryption_level"` // none, standard, high
	ForwardExternal bool          `json:"forward_external"`
	RequireApproval bool          `json:"require_approval"`
}

// NewAuditLogger creates a new audit logger instance
func NewAuditLogger(cfg config.AuditConfig, logger *zap.Logger) *AuditLogger {
	return &AuditLogger{
		config:      cfg,
		logger:      logger,
		auditLogs:   make(map[string]*compliance.AuditLog),
		categories:  make(map[string]*AuditCategory),
		stopChan:    make(chan struct{}),
		logChannel:  make(chan *compliance.AuditLog, cfg.BufferSize),
		batchBuffer: make([]*compliance.AuditLog, 0, cfg.BatchSize),
		lastFlush:   time.Now(),
	}
}

// Start starts the audit logger
func (al *AuditLogger) Start(ctx context.Context) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.running {
		return fmt.Errorf("audit logger is already running")
	}

	al.logger.Info("Starting audit logger")

	// Load audit categories
	if err := al.loadAuditCategories(); err != nil {
		return fmt.Errorf("failed to load audit categories: %w", err)
	}

	// Start background processes
	go al.logProcessingLoop(ctx)
	go al.retentionLoop(ctx)
	if al.config.EnableExternalForwarding {
		go al.externalForwardingLoop(ctx)
	}

	al.running = true
	al.logger.Info("Audit logger started successfully")

	return nil
}

// Stop stops the audit logger
func (al *AuditLogger) Stop(ctx context.Context) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.running {
		return nil
	}

	al.logger.Info("Stopping audit logger")

	close(al.stopChan)
	close(al.logChannel)

	// Flush remaining logs
	al.flushBatch()

	al.running = false
	al.logger.Info("Audit logger stopped")

	return nil
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, eventType, category, userID, entityID, entityType, action string, details map[string]interface{}) error {
	if !al.running {
		return fmt.Errorf("audit logger is not running")
	}

	auditLog := &compliance.AuditLog{
		ID:         al.generateLogID(),
		EventType:  eventType,
		Category:   category,
		UserID:     userID,
		EntityID:   entityID,
		EntityType: entityType,
		Action:     action,
		Details:    details,
		Timestamp:  time.Now(),
		Result:     "success", // Default to success, can be updated
	}

	// Add context information
	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		if ua, ok := userAgent.(string); ok {
			auditLog.UserAgent = ua
		}
	}

	if ipAddress := ctx.Value("ip_address"); ipAddress != nil {
		if ip, ok := ipAddress.(string); ok {
			auditLog.IPAddress = ip
		}
	}

	// Encrypt sensitive data if required
	if err := al.encryptSensitiveData(auditLog); err != nil {
		al.logger.Error("Failed to encrypt audit log data",
			zap.String("log_id", auditLog.ID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to encrypt audit log: %w", err)
	}

	// Send to processing channel
	select {
	case al.logChannel <- auditLog:
		return nil
	default:
		al.logger.Warn("Audit log channel full, dropping log",
			zap.String("log_id", auditLog.ID),
		)
		return fmt.Errorf("audit log channel full")
	}
}

// LogComplianceEvent logs a compliance-specific event
func (al *AuditLogger) LogComplianceEvent(ctx context.Context, eventType string, violation *compliance.Violation, userID string, action string, result string) error {
	details := map[string]interface{}{
		"violation_id": violation.ID,
		"rule_id":      violation.RuleID,
		"severity":     violation.Severity,
		"status":       violation.Status,
		"risk_score":   violation.RiskScore,
	}

	if len(violation.Details) > 0 {
		details["violation_details"] = violation.Details
	}

	auditLog := &compliance.AuditLog{
		ID:         al.generateLogID(),
		EventType:  eventType,
		Category:   "compliance",
		UserID:     userID,
		EntityID:   violation.EntityID,
		EntityType: violation.EntityType,
		Action:     action,
		Details:    details,
		Timestamp:  time.Now(),
		Result:     result,
	}

	// Add context information
	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		if ua, ok := userAgent.(string); ok {
			auditLog.UserAgent = ua
		}
	}

	if ipAddress := ctx.Value("ip_address"); ipAddress != nil {
		if ip, ok := ipAddress.(string); ok {
			auditLog.IPAddress = ip
		}
	}

	// Encrypt sensitive data
	if err := al.encryptSensitiveData(auditLog); err != nil {
		return fmt.Errorf("failed to encrypt compliance audit log: %w", err)
	}

	// Send to processing channel
	select {
	case al.logChannel <- auditLog:
		al.logger.Info("Compliance event logged",
			zap.String("log_id", auditLog.ID),
			zap.String("event_type", eventType),
			zap.String("violation_id", violation.ID),
		)
		return nil
	default:
		return fmt.Errorf("audit log channel full")
	}
}

// GetAuditLogs retrieves audit logs with optional filtering
func (al *AuditLogger) GetAuditLogs(ctx context.Context, filters AuditFilters) ([]*compliance.AuditLog, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if !al.running {
		return nil, fmt.Errorf("audit logger is not running")
	}

	var filteredLogs []*compliance.AuditLog

	for _, log := range al.auditLogs {
		if al.matchesFilters(log, filters) {
			// Decrypt sensitive data if user has permission
			decryptedLog := al.decryptSensitiveData(log)
			filteredLogs = append(filteredLogs, decryptedLog)
		}
	}

	// Sort by timestamp (newest first)
	al.sortLogsByTimestamp(filteredLogs)

	// Apply pagination if specified
	if filters.Limit > 0 {
		end := filters.Offset + filters.Limit
		if end > len(filteredLogs) {
			end = len(filteredLogs)
		}
		if filters.Offset < len(filteredLogs) {
			filteredLogs = filteredLogs[filters.Offset:end]
		} else {
			filteredLogs = []*compliance.AuditLog{}
		}
	}

	return filteredLogs, nil
}

// GetAuditStatistics returns audit trail statistics
func (al *AuditLogger) GetAuditStatistics(ctx context.Context, timeRange TimeRange) (*AuditStatistics, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if !al.running {
		return nil, fmt.Errorf("audit logger is not running")
	}

	stats := &AuditStatistics{
		TotalLogs:       0,
		CategoryCounts:  make(map[string]int),
		EventTypeCounts: make(map[string]int),
		ResultCounts:    make(map[string]int),
		UserCounts:      make(map[string]int),
		HourlyTrends:    make(map[int]int),
		GeneratedAt:     time.Now(),
		TimeRange:       timeRange,
	}

	for _, log := range al.auditLogs {
		// Check if log is within time range
		if timeRange.StartTime != nil && log.Timestamp.Before(*timeRange.StartTime) {
			continue
		}
		if timeRange.EndTime != nil && log.Timestamp.After(*timeRange.EndTime) {
			continue
		}

		stats.TotalLogs++
		stats.CategoryCounts[log.Category]++
		stats.EventTypeCounts[log.EventType]++
		stats.ResultCounts[log.Result]++
		stats.UserCounts[log.UserID]++

		// Hourly trends
		hour := log.Timestamp.Hour()
		stats.HourlyTrends[hour]++
	}

	return stats, nil
}

// ArchiveOldLogs archives old audit logs
func (al *AuditLogger) ArchiveOldLogs(ctx context.Context, archiveBefore time.Time) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.running {
		return fmt.Errorf("audit logger is not running")
	}

	archivedCount := 0
	for id, log := range al.auditLogs {
		if log.Timestamp.Before(archiveBefore) {
			// Archive log (simplified implementation - would typically export to external storage)
			if err := al.archiveLog(log); err != nil {
				al.logger.Error("Failed to archive audit log",
					zap.String("log_id", id),
					zap.Error(err),
				)
				continue
			}

			delete(al.auditLogs, id)
			archivedCount++
		}
	}

	al.logger.Info("Archived old audit logs",
		zap.Int("count", archivedCount),
		zap.Time("before", archiveBefore),
	)

	return nil
}

// Private methods

func (al *AuditLogger) loadAuditCategories() error {
	// Load default audit categories
	defaultCategories := []*AuditCategory{
		{
			Name:            "compliance",
			Description:     "Compliance-related events",
			RetentionPeriod: al.config.RetentionPeriod,
			EncryptionLevel: "high",
			ForwardExternal: true,
			RequireApproval: false,
		},
		{
			Name:            "authentication",
			Description:     "Authentication and authorization events",
			RetentionPeriod: 90 * 24 * time.Hour, // 90 days
			EncryptionLevel: "standard",
			ForwardExternal: false,
			RequireApproval: false,
		},
		{
			Name:            "data_access",
			Description:     "Data access and modification events",
			RetentionPeriod: al.config.RetentionPeriod,
			EncryptionLevel: "high",
			ForwardExternal: true,
			RequireApproval: true,
		},
		{
			Name:            "system",
			Description:     "System configuration and maintenance events",
			RetentionPeriod: 180 * 24 * time.Hour, // 180 days
			EncryptionLevel: "standard",
			ForwardExternal: false,
			RequireApproval: false,
		},
		{
			Name:            "violation",
			Description:     "Compliance violation events",
			RetentionPeriod: al.config.RetentionPeriod,
			EncryptionLevel: "high",
			ForwardExternal: true,
			RequireApproval: false,
		},
	}

	for _, category := range defaultCategories {
		al.categories[category.Name] = category
	}

	al.logger.Info("Audit categories loaded", zap.Int("count", len(defaultCategories)))
	return nil
}

func (al *AuditLogger) logProcessingLoop(ctx context.Context) {
	batchTicker := time.NewTicker(al.config.FlushInterval)
	defer batchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-al.stopChan:
			return
		case log := <-al.logChannel:
			if log != nil {
				al.addToBatch(log)
			}
		case <-batchTicker.C:
			al.flushBatchIfNeeded()
		}
	}
}

func (al *AuditLogger) retentionLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Daily retention check
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-al.stopChan:
			return
		case <-ticker.C:
			al.enforceRetentionPolicies()
		}
	}
}

func (al *AuditLogger) externalForwardingLoop(ctx context.Context) {
	ticker := time.NewTicker(al.config.ForwardingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-al.stopChan:
			return
		case <-ticker.C:
			al.forwardLogsToExternal()
		}
	}
}

func (al *AuditLogger) addToBatch(log *compliance.AuditLog) {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.batchBuffer = append(al.batchBuffer, log)

	if len(al.batchBuffer) >= al.config.BatchSize {
		al.flushBatch()
	}
}

func (al *AuditLogger) flushBatchIfNeeded() {
	al.mu.Lock()
	defer al.mu.Unlock()

	if len(al.batchBuffer) > 0 && time.Since(al.lastFlush) >= al.config.FlushInterval {
		al.flushBatch()
	}
}

func (al *AuditLogger) flushBatch() {
	if len(al.batchBuffer) == 0 {
		return
	}

	// Store logs in memory (in production, would write to persistent storage)
	for _, log := range al.batchBuffer {
		al.auditLogs[log.ID] = log
	}

	al.logger.Debug("Flushed audit log batch",
		zap.Int("count", len(al.batchBuffer)),
	)

	// Clear batch
	al.batchBuffer = al.batchBuffer[:0]
	al.lastFlush = time.Now()
}

func (al *AuditLogger) enforceRetentionPolicies() {
	al.mu.Lock()
	defer al.mu.Unlock()

	deletedCount := 0
	now := time.Now()

	for id, log := range al.auditLogs {
		category, exists := al.categories[log.Category]
		if !exists {
			// Use default retention if category not found
			category = &AuditCategory{
				RetentionPeriod: al.config.RetentionPeriod,
			}
		}

		if now.Sub(log.Timestamp) > category.RetentionPeriod {
			// Archive before deletion if required
			if al.config.ArchiveBeforeDelete {
				if err := al.archiveLog(log); err != nil {
					al.logger.Error("Failed to archive log before deletion",
						zap.String("log_id", id),
						zap.Error(err),
					)
					continue
				}
			}

			delete(al.auditLogs, id)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		al.logger.Info("Enforced retention policies",
			zap.Int("deleted_count", deletedCount),
		)
	}
}

func (al *AuditLogger) forwardLogsToExternal() {
	al.mu.RLock()
	logsToForward := make([]*compliance.AuditLog, 0)
	for _, log := range al.auditLogs {
		if category, exists := al.categories[log.Category]; exists && category.ForwardExternal {
			logsToForward = append(logsToForward, log)
		}
	}
	al.mu.RUnlock()

	if len(logsToForward) > 0 {
		// Forward to external systems (simplified implementation)
		al.logger.Info("Forwarding logs to external systems",
			zap.Int("count", len(logsToForward)),
		)
		// Implementation would send logs to SIEM, compliance systems, etc.
	}
}

func (al *AuditLogger) encryptSensitiveData(log *compliance.AuditLog) error {
	category, exists := al.categories[log.Category]
	if !exists || category.EncryptionLevel == "none" {
		return nil
	}

	// Simplified encryption (in production, use proper encryption)
	if category.EncryptionLevel == "high" || category.EncryptionLevel == "standard" {
		// Encrypt sensitive fields
		if log.Details != nil {
			// This would implement actual encryption
			log.Details["_encrypted"] = true
		}
	}

	return nil
}

func (al *AuditLogger) decryptSensitiveData(log *compliance.AuditLog) *compliance.AuditLog {
	// Create a copy to avoid modifying the original
	decrypted := *log
	if log.Details != nil {
		decrypted.Details = make(map[string]interface{})
		for k, v := range log.Details {
			decrypted.Details[k] = v
		}

		// Simplified decryption (in production, use proper decryption)
		if _, encrypted := log.Details["_encrypted"]; encrypted {
			delete(decrypted.Details, "_encrypted")
		}
	}

	return &decrypted
}

func (al *AuditLogger) matchesFilters(log *compliance.AuditLog, filters AuditFilters) bool {
	if filters.Category != "" && log.Category != filters.Category {
		return false
	}

	if filters.EventType != "" && log.EventType != filters.EventType {
		return false
	}

	if filters.UserID != "" && log.UserID != filters.UserID {
		return false
	}

	if filters.EntityID != "" && log.EntityID != filters.EntityID {
		return false
	}

	if filters.StartTime != nil && log.Timestamp.Before(*filters.StartTime) {
		return false
	}

	if filters.EndTime != nil && log.Timestamp.After(*filters.EndTime) {
		return false
	}

	return true
}

func (al *AuditLogger) sortLogsByTimestamp(logs []*compliance.AuditLog) {
	// Simple bubble sort (in production, use more efficient sorting)
	n := len(logs)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if logs[j].Timestamp.Before(logs[j+1].Timestamp) {
				logs[j], logs[j+1] = logs[j+1], logs[j]
			}
		}
	}
}

func (al *AuditLogger) archiveLog(log *compliance.AuditLog) error {
	// Simplified archival (in production, would export to external storage)
	al.logger.Debug("Archiving audit log",
		zap.String("log_id", log.ID),
		zap.Time("timestamp", log.Timestamp),
	)
	return nil
}

func (al *AuditLogger) generateLogID() string {
	// Generate random bytes
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Create hash
	hash := sha256.Sum256(bytes)
	return fmt.Sprintf("AUDIT_%s", hex.EncodeToString(hash[:8]))
}

// Supporting types

type AuditFilters struct {
	Category   string     `json:"category,omitempty"`
	EventType  string     `json:"event_type,omitempty"`
	UserID     string     `json:"user_id,omitempty"`
	EntityID   string     `json:"entity_id,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

type TimeRange struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

type AuditStatistics struct {
	TotalLogs       int            `json:"total_logs"`
	CategoryCounts  map[string]int `json:"category_counts"`
	EventTypeCounts map[string]int `json:"event_type_counts"`
	ResultCounts    map[string]int `json:"result_counts"`
	UserCounts      map[string]int `json:"user_counts"`
	HourlyTrends    map[int]int    `json:"hourly_trends"`
	GeneratedAt     time.Time      `json:"generated_at"`
	TimeRange       TimeRange      `json:"time_range"`
}