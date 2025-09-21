package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/aegis-shield/services/alerting-engine/internal/database"
)

// EvaluationPool manages concurrent rule evaluations
type EvaluationPool struct {
	workers    int
	taskQueue  chan func()
	workerWg   sync.WaitGroup
	shutdownCh chan struct{}
}

// NewEvaluationPool creates a new evaluation pool
func NewEvaluationPool(maxWorkers int) *EvaluationPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	pool := &EvaluationPool{
		workers:    maxWorkers,
		taskQueue:  make(chan func(), maxWorkers*2),
		shutdownCh: make(chan struct{}),
	}

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		pool.workerWg.Add(1)
		go pool.worker()
	}

	return pool
}

// Submit submits a task to the evaluation pool
func (p *EvaluationPool) Submit(task func()) {
	select {
	case p.taskQueue <- task:
	case <-p.shutdownCh:
	}
}

// Close closes the evaluation pool
func (p *EvaluationPool) Close() {
	close(p.shutdownCh)
	close(p.taskQueue)
	p.workerWg.Wait()
}

func (p *EvaluationPool) worker() {
	defer p.workerWg.Done()

	for {
		select {
		case task := <-p.taskQueue:
			if task != nil {
				task()
			}
		case <-p.shutdownCh:
			return
		}
	}
}

// CreateAlertHandler handles alert creation actions
type CreateAlertHandler struct {
	config    map[string]interface{}
	alertRepo *database.AlertRepository
	logger    *slog.Logger
}

// NewCreateAlertHandler creates a new alert creation handler
func NewCreateAlertHandler(config map[string]interface{}, alertRepo *database.AlertRepository, logger *slog.Logger) *CreateAlertHandler {
	return &CreateAlertHandler{
		config:    config,
		alertRepo: alertRepo,
		logger:    logger,
	}
}

// Execute creates a new alert
func (h *CreateAlertHandler) Execute(ctx context.Context, result *EvaluationResult) error {
	// Extract alert parameters from config
	title, _ := h.config["title"].(string)
	if title == "" {
		title = "Alert triggered by rule: " + result.RuleName
	}

	description, _ := h.config["description"].(string)
	if description == "" {
		description = "Alert created by rule evaluation"
	}

	severity, _ := h.config["severity"].(string)
	if severity == "" {
		severity = "medium"
	}

	alertType, _ := h.config["type"].(string)
	if alertType == "" {
		alertType = "rule-based"
	}

	priority, _ := h.config["priority"].(string)
	if priority == "" {
		priority = "medium"
	}

	// Create alert
	alert := &database.Alert{
		ID:          generateID("alert"),
		RuleID:      result.RuleID,
		Title:       title,
		Description: description,
		Severity:    severity,
		Type:        alertType,
		Priority:    priority,
		Status:      "active",
		Source:      "rule-engine",
		CreatedBy:   "system",
		UpdatedBy:   "system",
	}

	// Add event data as metadata
	if eventData, err := json.Marshal(result.Context.Event); err == nil {
		alert.EventData = eventData
	}

	// Add metadata
	metadata := map[string]interface{}{
		"rule_name":       result.RuleName,
		"evaluation_time": result.ExecutionTime.String(),
		"matched_actions": result.Actions,
	}
	if metadataBytes, err := json.Marshal(metadata); err == nil {
		alert.Metadata = metadataBytes
	}

	// Save alert
	if err := h.alertRepo.Create(ctx, alert); err != nil {
		h.logger.Error("Failed to create alert from rule",
			"rule_id", result.RuleID,
			"rule_name", result.RuleName,
			"error", err)
		return err
	}

	h.logger.Info("Alert created from rule",
		"alert_id", alert.ID,
		"rule_id", result.RuleID,
		"rule_name", result.RuleName,
		"severity", severity)

	return nil
}

// GetType returns the handler type
func (h *CreateAlertHandler) GetType() string {
	return "create_alert"
}

// SendNotificationHandler handles notification sending actions
type SendNotificationHandler struct {
	config map[string]interface{}
	logger *slog.Logger
}

// NewSendNotificationHandler creates a new notification handler
func NewSendNotificationHandler(config map[string]interface{}, logger *slog.Logger) *SendNotificationHandler {
	return &SendNotificationHandler{
		config: config,
		logger: logger,
	}
}

// Execute sends a notification
func (h *SendNotificationHandler) Execute(ctx context.Context, result *EvaluationResult) error {
	// Extract notification parameters
	channel, _ := h.config["channel"].(string)
	if channel == "" {
		channel = "email"
	}

	recipient, _ := h.config["recipient"].(string)
	if recipient == "" {
		h.logger.Error("No recipient specified for notification action")
		return nil
	}

	subject, _ := h.config["subject"].(string)
	if subject == "" {
		subject = "Alert: " + result.RuleName
	}

	message, _ := h.config["message"].(string)
	if message == "" {
		message = "Rule " + result.RuleName + " has been triggered"
	}

	priority, _ := h.config["priority"].(string)
	if priority == "" {
		priority = "medium"
	}

	// Create notification (this would typically go through a notification service)
	notification := &database.Notification{
		ID:             generateID("notification"),
		RuleID:         result.RuleID,
		Channel:        channel,
		Recipient:      recipient,
		Subject:        subject,
		Message:        message,
		Priority:       priority,
		Status:         "pending",
		DeliveryMethod: "async",
		MaxRetries:     3,
		CreatedBy:      "system",
		UpdatedBy:      "system",
	}

	// Add template data
	templateData := map[string]interface{}{
		"rule_name":       result.RuleName,
		"rule_id":         result.RuleID,
		"evaluation_time": result.ExecutionTime.String(),
		"event":           result.Context.Event,
		"timestamp":       result.Context.Timestamp,
	}
	if templateBytes, err := json.Marshal(templateData); err == nil {
		notification.TemplateData = templateBytes
	}

	h.logger.Info("Notification action triggered",
		"rule_id", result.RuleID,
		"rule_name", result.RuleName,
		"channel", channel,
		"recipient", recipient)

	// Note: In a real implementation, this would be sent to a notification service
	// For now, we just log the notification creation
	return nil
}

// GetType returns the handler type
func (h *SendNotificationHandler) GetType() string {
	return "send_notification"
}

// WebhookActionHandler handles webhook actions
type WebhookActionHandler struct {
	config map[string]interface{}
	logger *slog.Logger
}

// NewWebhookActionHandler creates a new webhook action handler
func NewWebhookActionHandler(config map[string]interface{}, logger *slog.Logger) *WebhookActionHandler {
	return &WebhookActionHandler{
		config: config,
		logger: logger,
	}
}

// Execute sends a webhook
func (h *WebhookActionHandler) Execute(ctx context.Context, result *EvaluationResult) error {
	url, _ := h.config["url"].(string)
	if url == "" {
		h.logger.Error("No URL specified for webhook action")
		return nil
	}

	method, _ := h.config["method"].(string)
	if method == "" {
		method = "POST"
	}

	// Create webhook payload
	payload := map[string]interface{}{
		"rule_id":         result.RuleID,
		"rule_name":       result.RuleName,
		"matched":         result.Matched,
		"actions":         result.Actions,
		"evaluation_time": result.ExecutionTime.String(),
		"event":           result.Context.Event,
		"timestamp":       result.Context.Timestamp,
	}

	// Add custom payload data if specified
	if customPayload, ok := h.config["payload"].(map[string]interface{}); ok {
		for key, value := range customPayload {
			payload[key] = value
		}
	}

	h.logger.Info("Webhook action triggered",
		"rule_id", result.RuleID,
		"rule_name", result.RuleName,
		"url", url,
		"method", method)

	// Note: In a real implementation, this would make an HTTP request to the webhook URL
	// For now, we just log the webhook action
	return nil
}

// GetType returns the handler type
func (h *WebhookActionHandler) GetType() string {
	return "webhook"
}

// EscalationHandler handles alert escalation actions
type EscalationHandler struct {
	config        map[string]interface{}
	alertRepo     *database.AlertRepository
	escalationRepo *database.EscalationRepository
	logger        *slog.Logger
}

// NewEscalationHandler creates a new escalation handler
func NewEscalationHandler(
	config map[string]interface{}, 
	alertRepo *database.AlertRepository,
	escalationRepo *database.EscalationRepository,
	logger *slog.Logger,
) *EscalationHandler {
	return &EscalationHandler{
		config:         config,
		alertRepo:      alertRepo,
		escalationRepo: escalationRepo,
		logger:         logger,
	}
}

// Execute escalates an alert
func (h *EscalationHandler) Execute(ctx context.Context, result *EvaluationResult) error {
	// Get escalation policy
	policyID, _ := h.config["escalation_policy_id"].(string)
	if policyID == "" {
		h.logger.Error("No escalation policy specified")
		return nil
	}

	policy, err := h.escalationRepo.GetByID(ctx, policyID)
	if err != nil {
		h.logger.Error("Failed to get escalation policy",
			"policy_id", policyID,
			"error", err)
		return err
	}

	if !policy.Enabled {
		h.logger.Warn("Escalation policy is disabled",
			"policy_id", policyID)
		return nil
	}

	// Find or create alert
	var alertID string
	if alertIDFromEvent, ok := result.Context.Event["alert_id"].(string); ok {
		alertID = alertIDFromEvent
	} else {
		// Create new alert if none exists
		alert := &database.Alert{
			ID:                 generateID("alert"),
			RuleID:             result.RuleID,
			Title:              "Escalated Alert: " + result.RuleName,
			Description:        "Alert escalated by rule evaluation",
			Severity:           "high",
			Type:               "escalated",
			Priority:           "high",
			Status:             "escalated",
			Source:             "rule-engine",
			EscalationPolicyID: &policyID,
			EscalationLevel:    1,
			CreatedBy:          "system",
			UpdatedBy:          "system",
		}

		if err := h.alertRepo.Create(ctx, alert); err != nil {
			return err
		}
		alertID = alert.ID
	}

	h.logger.Info("Alert escalation triggered",
		"alert_id", alertID,
		"rule_id", result.RuleID,
		"rule_name", result.RuleName,
		"escalation_policy", policy.Name)

	return nil
}

// GetType returns the handler type
func (h *EscalationHandler) GetType() string {
	return "escalation"
}

// ThrottleHandler handles action throttling
type ThrottleHandler struct {
	config     map[string]interface{}
	throttleMap map[string]time.Time
	mutex      sync.RWMutex
	logger     *slog.Logger
}

// NewThrottleHandler creates a new throttle handler
func NewThrottleHandler(config map[string]interface{}, logger *slog.Logger) *ThrottleHandler {
	return &ThrottleHandler{
		config:      config,
		throttleMap: make(map[string]time.Time),
		logger:      logger,
	}
}

// Execute checks throttling for an action
func (h *ThrottleHandler) Execute(ctx context.Context, result *EvaluationResult) error {
	throttleKey := h.generateThrottleKey(result)
	throttleWindow, _ := h.config["throttle_window_minutes"].(float64)
	if throttleWindow <= 0 {
		throttleWindow = 5 // Default 5 minutes
	}

	h.mutex.RLock()
	lastExecution, exists := h.throttleMap[throttleKey]
	h.mutex.RUnlock()

	if exists {
		elapsed := time.Since(lastExecution)
		if elapsed < time.Duration(throttleWindow)*time.Minute {
			h.logger.Debug("Action throttled",
				"rule_id", result.RuleID,
				"throttle_key", throttleKey,
				"elapsed", elapsed,
				"window", throttleWindow)
			return nil // Throttled
		}
	}

	h.mutex.Lock()
	h.throttleMap[throttleKey] = time.Now()
	h.mutex.Unlock()

	return nil
}

// GetType returns the handler type
func (h *ThrottleHandler) GetType() string {
	return "throttle"
}

func (h *ThrottleHandler) generateThrottleKey(result *EvaluationResult) string {
	// Generate throttle key based on rule and event characteristics
	return result.RuleID + ":" + result.RuleName
}

// Utility function to generate IDs
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().Unix(), time.Now().Nanosecond())
}

// RuleMetrics tracks rule evaluation metrics
type RuleMetrics struct {
	RuleID           string
	RuleName         string
	EvaluationCount  int64
	MatchCount       int64
	ErrorCount       int64
	TotalExecutionTime time.Duration
	AverageExecutionTime time.Duration
	LastEvaluation   time.Time
	LastMatch        time.Time
	LastError        time.Time
}

// MetricsCollector collects rule evaluation metrics
type MetricsCollector struct {
	metrics map[string]*RuleMetrics
	mutex   sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*RuleMetrics),
	}
}

// RecordEvaluation records a rule evaluation
func (m *MetricsCollector) RecordEvaluation(result *EvaluationResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metrics, exists := m.metrics[result.RuleID]
	if !exists {
		metrics = &RuleMetrics{
			RuleID:   result.RuleID,
			RuleName: result.RuleName,
		}
		m.metrics[result.RuleID] = metrics
	}

	metrics.EvaluationCount++
	metrics.TotalExecutionTime += result.ExecutionTime
	metrics.AverageExecutionTime = metrics.TotalExecutionTime / time.Duration(metrics.EvaluationCount)
	metrics.LastEvaluation = time.Now()

	if result.Matched {
		metrics.MatchCount++
		metrics.LastMatch = time.Now()
	}

	if result.Error != nil {
		metrics.ErrorCount++
		metrics.LastError = time.Now()
	}
}

// GetMetrics returns all collected metrics
func (m *MetricsCollector) GetMetrics() map[string]*RuleMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*RuleMetrics)
	for k, v := range m.metrics {
		result[k] = &RuleMetrics{
			RuleID:               v.RuleID,
			RuleName:             v.RuleName,
			EvaluationCount:      v.EvaluationCount,
			MatchCount:           v.MatchCount,
			ErrorCount:           v.ErrorCount,
			TotalExecutionTime:   v.TotalExecutionTime,
			AverageExecutionTime: v.AverageExecutionTime,
			LastEvaluation:       v.LastEvaluation,
			LastMatch:            v.LastMatch,
			LastError:            v.LastError,
		}
	}

	return result
}

// GetRuleMetrics returns metrics for a specific rule
func (m *MetricsCollector) GetRuleMetrics(ruleID string) *RuleMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if metrics, exists := m.metrics[ruleID]; exists {
		// Return a copy
		return &RuleMetrics{
			RuleID:               metrics.RuleID,
			RuleName:             metrics.RuleName,
			EvaluationCount:      metrics.EvaluationCount,
			MatchCount:           metrics.MatchCount,
			ErrorCount:           metrics.ErrorCount,
			TotalExecutionTime:   metrics.TotalExecutionTime,
			AverageExecutionTime: metrics.AverageExecutionTime,
			LastEvaluation:       metrics.LastEvaluation,
			LastMatch:            metrics.LastMatch,
			LastError:            metrics.LastError,
		}
	}

	return nil
}