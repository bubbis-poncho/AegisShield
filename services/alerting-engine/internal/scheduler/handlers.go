package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
)

// AlertCleanupHandler handles cleanup of old alerts
type AlertCleanupHandler struct {
	alertRepo *database.AlertRepository
	config    *config.Config
	logger    *slog.Logger
}

// NewAlertCleanupHandler creates a new alert cleanup handler
func NewAlertCleanupHandler(alertRepo *database.AlertRepository, cfg *config.Config, logger *slog.Logger) *AlertCleanupHandler {
	return &AlertCleanupHandler{
		alertRepo: alertRepo,
		config:    cfg,
		logger:    logger,
	}
}

// Execute performs alert cleanup
func (h *AlertCleanupHandler) Execute(ctx context.Context) error {
	h.logger.Info("Starting alert cleanup")

	// Clean up resolved alerts
	resolvedCount, err := h.alertRepo.CleanupOldAlerts(ctx, "resolved", h.config.Scheduler.AlertRetentionDays)
	if err != nil {
		h.logger.Error("Failed to cleanup resolved alerts", "error", err)
		return fmt.Errorf("failed to cleanup resolved alerts: %w", err)
	}

	// Clean up closed alerts
	closedCount, err := h.alertRepo.CleanupOldAlerts(ctx, "closed", h.config.Scheduler.AlertRetentionDays)
	if err != nil {
		h.logger.Error("Failed to cleanup closed alerts", "error", err)
		return fmt.Errorf("failed to cleanup closed alerts: %w", err)
	}

	totalCleaned := resolvedCount + closedCount
	h.logger.Info("Alert cleanup completed",
		"resolved_cleaned", resolvedCount,
		"closed_cleaned", closedCount,
		"total_cleaned", totalCleaned,
		"retention_days", h.config.Scheduler.AlertRetentionDays)

	return nil
}

// GetName returns the handler name
func (h *AlertCleanupHandler) GetName() string {
	return "Alert Cleanup"
}

// GetDescription returns the handler description
func (h *AlertCleanupHandler) GetDescription() string {
	return "Cleans up old resolved and closed alerts based on retention policy"
}

// NotificationCleanupHandler handles cleanup of old notifications
type NotificationCleanupHandler struct {
	notificationRepo *database.NotificationRepository
	config           *config.Config
	logger           *slog.Logger
}

// NewNotificationCleanupHandler creates a new notification cleanup handler
func NewNotificationCleanupHandler(notificationRepo *database.NotificationRepository, cfg *config.Config, logger *slog.Logger) *NotificationCleanupHandler {
	return &NotificationCleanupHandler{
		notificationRepo: notificationRepo,
		config:           cfg,
		logger:           logger,
	}
}

// Execute performs notification cleanup
func (h *NotificationCleanupHandler) Execute(ctx context.Context) error {
	h.logger.Info("Starting notification cleanup")

	cleanedCount, err := h.notificationRepo.CleanupOldNotifications(ctx, h.config.Scheduler.NotificationRetentionDays)
	if err != nil {
		h.logger.Error("Failed to cleanup notifications", "error", err)
		return fmt.Errorf("failed to cleanup notifications: %w", err)
	}

	h.logger.Info("Notification cleanup completed",
		"cleaned_count", cleanedCount,
		"retention_days", h.config.Scheduler.NotificationRetentionDays)

	return nil
}

// GetName returns the handler name
func (h *NotificationCleanupHandler) GetName() string {
	return "Notification Cleanup"
}

// GetDescription returns the handler description
func (h *NotificationCleanupHandler) GetDescription() string {
	return "Cleans up old delivered and failed notifications based on retention policy"
}

// HealthCheckHandler performs system health checks
type HealthCheckHandler struct {
	alertRepo  *database.AlertRepository
	ruleEngine *engine.RuleEngine
	config     *config.Config
	logger     *slog.Logger
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(alertRepo *database.AlertRepository, ruleEngine *engine.RuleEngine, cfg *config.Config, logger *slog.Logger) *HealthCheckHandler {
	return &HealthCheckHandler{
		alertRepo:  alertRepo,
		ruleEngine: ruleEngine,
		config:     cfg,
		logger:     logger,
	}
}

// Execute performs health checks
func (h *HealthCheckHandler) Execute(ctx context.Context) error {
	h.logger.Debug("Starting health check")

	var healthIssues []string

	// Check alert repository health
	if err := h.checkAlertRepositoryHealth(ctx); err != nil {
		healthIssues = append(healthIssues, fmt.Sprintf("Alert repository: %v", err))
		h.logger.Error("Alert repository health check failed", "error", err)
	}

	// Check rule engine health
	if err := h.checkRuleEngineHealth(ctx); err != nil {
		healthIssues = append(healthIssues, fmt.Sprintf("Rule engine: %v", err))
		h.logger.Error("Rule engine health check failed", "error", err)
	}

	// Check alert processing performance
	if err := h.checkAlertProcessingPerformance(ctx); err != nil {
		healthIssues = append(healthIssues, fmt.Sprintf("Alert processing: %v", err))
		h.logger.Warn("Alert processing performance issue", "error", err)
	}

	if len(healthIssues) > 0 {
		h.logger.Warn("Health check completed with issues", "issues", healthIssues)
		
		// Create a health alert if configured
		if h.config.Scheduler.CreateHealthAlerts {
			if err := h.createHealthAlert(ctx, healthIssues); err != nil {
				h.logger.Error("Failed to create health alert", "error", err)
			}
		}
	} else {
		h.logger.Debug("Health check completed successfully")
	}

	return nil
}

func (h *HealthCheckHandler) checkAlertRepositoryHealth(ctx context.Context) error {
	// Test database connectivity by getting alert count
	_, _, err := h.alertRepo.List(ctx, database.Filter{Limit: 1})
	if err != nil {
		return fmt.Errorf("database connectivity failed: %w", err)
	}
	return nil
}

func (h *HealthCheckHandler) checkRuleEngineHealth(ctx context.Context) error {
	// Check rule engine statistics
	stats := h.ruleEngine.GetRuleStats()
	totalRules, ok := stats["total_rules"].(int)
	if !ok || totalRules == 0 {
		return fmt.Errorf("no rules loaded in engine")
	}
	return nil
}

func (h *HealthCheckHandler) checkAlertProcessingPerformance(ctx context.Context) error {
	// Check for alerts stuck in processing state
	since := time.Now().Add(-1 * time.Hour)
	filter := database.Filter{
		DateFrom: &since,
		Filters: map[string]interface{}{
			"status": "processing",
		},
		Limit: 100,
	}

	stuckAlerts, _, err := h.alertRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check stuck alerts: %w", err)
	}

	if len(stuckAlerts) > 10 {
		return fmt.Errorf("found %d alerts stuck in processing state", len(stuckAlerts))
	}

	return nil
}

func (h *HealthCheckHandler) createHealthAlert(ctx context.Context, issues []string) error {
	alert := &database.Alert{
		ID:          generateHealthAlertID(),
		Title:       "System Health Check Alert",
		Description: fmt.Sprintf("Health check detected issues: %v", issues),
		Severity:    "medium",
		Type:        "health-check",
		Priority:    "medium",
		Status:      "active",
		Source:      "health-check",
		CreatedBy:   "system",
		UpdatedBy:   "system",
	}

	return h.alertRepo.Create(ctx, alert)
}

// GetName returns the handler name
func (h *HealthCheckHandler) GetName() string {
	return "Health Check"
}

// GetDescription returns the handler description
func (h *HealthCheckHandler) GetDescription() string {
	return "Performs system health checks and creates alerts for detected issues"
}

// EscalationProcessorHandler processes alert escalations
type EscalationProcessorHandler struct {
	alertRepo        *database.AlertRepository
	escalationRepo   *database.EscalationRepository
	notificationRepo *database.NotificationRepository
	config           *config.Config
	logger           *slog.Logger
}

// NewEscalationProcessorHandler creates a new escalation processor handler
func NewEscalationProcessorHandler(
	alertRepo *database.AlertRepository,
	escalationRepo *database.EscalationRepository,
	notificationRepo *database.NotificationRepository,
	cfg *config.Config,
	logger *slog.Logger,
) *EscalationProcessorHandler {
	return &EscalationProcessorHandler{
		alertRepo:        alertRepo,
		escalationRepo:   escalationRepo,
		notificationRepo: notificationRepo,
		config:           cfg,
		logger:           logger,
	}
}

// Execute processes escalations
func (h *EscalationProcessorHandler) Execute(ctx context.Context) error {
	h.logger.Debug("Starting escalation processing")

	// Get alerts that need escalation
	alertsToEscalate, err := h.getAlertsForEscalation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get alerts for escalation: %w", err)
	}

	escalatedCount := 0
	for _, alert := range alertsToEscalate {
		if err := h.processAlertEscalation(ctx, alert); err != nil {
			h.logger.Error("Failed to process escalation for alert",
				"alert_id", alert.ID,
				"error", err)
			continue
		}
		escalatedCount++
	}

	h.logger.Debug("Escalation processing completed",
		"processed_alerts", len(alertsToEscalate),
		"escalated_count", escalatedCount)

	return nil
}

func (h *EscalationProcessorHandler) getAlertsForEscalation(ctx context.Context) ([]*database.Alert, error) {
	// Get active alerts that have escalation policies and haven't been escalated recently
	cutoffTime := time.Now().Add(-time.Duration(h.config.Scheduler.EscalationWindowMinutes) * time.Minute)
	
	filter := database.Filter{
		Filters: map[string]interface{}{
			"status": "active",
			"has_escalation_policy": true,
		},
		DateTo: &cutoffTime,
		Limit:  100,
	}

	alerts, _, err := h.alertRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter alerts that need escalation
	var alertsToEscalate []*database.Alert
	for _, alert := range alerts {
		if h.shouldEscalateAlert(alert) {
			alertsToEscalate = append(alertsToEscalate, alert)
		}
	}

	return alertsToEscalate, nil
}

func (h *EscalationProcessorHandler) shouldEscalateAlert(alert *database.Alert) bool {
	// Check if alert has been unacknowledged for the escalation window
	if alert.AcknowledgedAt != nil {
		return false
	}

	// Check if alert is within escalation window
	escalationWindow := time.Duration(h.config.Scheduler.EscalationWindowMinutes) * time.Minute
	return time.Since(alert.CreatedAt) > escalationWindow
}

func (h *EscalationProcessorHandler) processAlertEscalation(ctx context.Context, alert *database.Alert) error {
	if alert.EscalationPolicyID == nil {
		return fmt.Errorf("alert has no escalation policy")
	}

	// Get escalation policy
	policy, err := h.escalationRepo.GetByID(ctx, *alert.EscalationPolicyID)
	if err != nil {
		return fmt.Errorf("failed to get escalation policy: %w", err)
	}

	// Escalate alert
	escalatedBy := "escalation-processor"
	if err := h.alertRepo.Escalate(ctx, alert.ID, escalatedBy); err != nil {
		return fmt.Errorf("failed to escalate alert: %w", err)
	}

	h.logger.Info("Alert escalated",
		"alert_id", alert.ID,
		"escalation_policy", policy.Name,
		"escalation_level", alert.EscalationLevel+1)

	return nil
}

// GetName returns the handler name
func (h *EscalationProcessorHandler) GetName() string {
	return "Escalation Processor"
}

// GetDescription returns the handler description
func (h *EscalationProcessorHandler) GetDescription() string {
	return "Processes alert escalations based on escalation policies and timeouts"
}

// MetricsCollectionHandler collects system metrics
type MetricsCollectionHandler struct {
	alertRepo        *database.AlertRepository
	notificationRepo *database.NotificationRepository
	ruleEngine       *engine.RuleEngine
	config           *config.Config
	logger           *slog.Logger
}

// NewMetricsCollectionHandler creates a new metrics collection handler
func NewMetricsCollectionHandler(
	alertRepo *database.AlertRepository,
	notificationRepo *database.NotificationRepository,
	ruleEngine *engine.RuleEngine,
	cfg *config.Config,
	logger *slog.Logger,
) *MetricsCollectionHandler {
	return &MetricsCollectionHandler{
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
		ruleEngine:       ruleEngine,
		config:           cfg,
		logger:           logger,
	}
}

// Execute collects metrics
func (h *MetricsCollectionHandler) Execute(ctx context.Context) error {
	h.logger.Debug("Starting metrics collection")

	// Collect alert metrics
	alertMetrics, err := h.collectAlertMetrics(ctx)
	if err != nil {
		h.logger.Error("Failed to collect alert metrics", "error", err)
	} else {
		h.logger.Debug("Alert metrics collected", "metrics", alertMetrics)
	}

	// Collect notification metrics
	notificationMetrics, err := h.collectNotificationMetrics(ctx)
	if err != nil {
		h.logger.Error("Failed to collect notification metrics", "error", err)
	} else {
		h.logger.Debug("Notification metrics collected", "metrics", notificationMetrics)
	}

	// Collect rule engine metrics
	ruleMetrics := h.ruleEngine.GetRuleStats()
	h.logger.Debug("Rule engine metrics collected", "metrics", ruleMetrics)

	h.logger.Debug("Metrics collection completed")
	return nil
}

func (h *MetricsCollectionHandler) collectAlertMetrics(ctx context.Context) (map[string]interface{}, error) {
	since := time.Now().Add(-24 * time.Hour)

	// Get alert statistics
	stats, err := h.alertRepo.GetStatsByTimeRange(ctx, since, time.Now())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_alerts":    stats.TotalCount,
		"active_alerts":   stats.ActiveCount,
		"resolved_alerts": stats.ResolvedCount,
		"closed_alerts":   stats.ClosedCount,
	}, nil
}

func (h *MetricsCollectionHandler) collectNotificationMetrics(ctx context.Context) (map[string]interface{}, error) {
	since := time.Now().Add(-24 * time.Hour)

	// Get notification statistics by channel
	stats, err := h.notificationRepo.GetStatsByChannel(ctx, since)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})
	for _, stat := range stats {
		metrics[stat.Channel] = map[string]interface{}{
			"total":     stat.TotalCount,
			"delivered": stat.DeliveredCount,
			"failed":    stat.FailedCount,
			"pending":   stat.PendingCount,
		}
	}

	return metrics, nil
}

// GetName returns the handler name
func (h *MetricsCollectionHandler) GetName() string {
	return "Metrics Collection"
}

// GetDescription returns the handler description
func (h *MetricsCollectionHandler) GetDescription() string {
	return "Collects system metrics for monitoring and analysis"
}

// PendingNotificationsHandler processes pending notifications
type PendingNotificationsHandler struct {
	notificationMgr *notification.Manager
	config          *config.Config
	logger          *slog.Logger
}

// NewPendingNotificationsHandler creates a new pending notifications handler
func NewPendingNotificationsHandler(notificationMgr *notification.Manager, cfg *config.Config, logger *slog.Logger) *PendingNotificationsHandler {
	return &PendingNotificationsHandler{
		notificationMgr: notificationMgr,
		config:          cfg,
		logger:          logger,
	}
}

// Execute processes pending notifications
func (h *PendingNotificationsHandler) Execute(ctx context.Context) error {
	h.logger.Debug("Starting pending notifications processing")

	if err := h.notificationMgr.ProcessPendingNotifications(ctx); err != nil {
		h.logger.Error("Failed to process pending notifications", "error", err)
		return fmt.Errorf("failed to process pending notifications: %w", err)
	}

	h.logger.Debug("Pending notifications processing completed")
	return nil
}

// GetName returns the handler name
func (h *PendingNotificationsHandler) GetName() string {
	return "Pending Notifications Processor"
}

// GetDescription returns the handler description
func (h *PendingNotificationsHandler) GetDescription() string {
	return "Processes pending notifications that need to be sent"
}

// Utility functions

func generateHealthAlertID() string {
	return fmt.Sprintf("health_%d", time.Now().Unix())
}