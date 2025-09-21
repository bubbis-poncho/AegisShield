package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"../../internal/config"
)

// NewAlertManager creates a new alert manager
func NewAlertManager(cfg *config.Config, logger *zap.Logger) *AlertManager {
	manager := &AlertManager{
		config:   cfg,
		logger:   logger,
		channels: make(map[string]AlertChannel),
	}

	// Register alert channels
	if cfg.ML.ModelMonitoring.Alerting.Email.Enabled {
		manager.channels["email"] = NewEmailChannel(cfg, logger)
	}
	if cfg.ML.ModelMonitoring.Alerting.Slack.Enabled {
		manager.channels["slack"] = NewSlackChannel(cfg, logger)
	}
	if cfg.ML.ModelMonitoring.Alerting.Webhook.Enabled {
		manager.channels["webhook"] = NewWebhookChannel(cfg, logger)
	}

	return manager
}

// SendAlert sends an alert through configured channels
func (am *AlertManager) SendAlert(ctx context.Context, alert *Alert) error {
	am.logger.Info("Sending alert",
		zap.String("type", alert.Type),
		zap.String("severity", alert.Severity),
		zap.String("model_id", alert.ModelID))

	var errors []string
	for channelName, channel := range am.channels {
		if err := channel.SendAlert(ctx, alert); err != nil {
			am.logger.Error("Failed to send alert through channel",
				zap.String("channel", channelName),
				zap.Error(err))
			errors = append(errors, fmt.Sprintf("%s: %v", channelName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send alerts: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetActiveChannels returns list of active alert channels
func (am *AlertManager) GetActiveChannels() []string {
	channels := make([]string, 0, len(am.channels))
	for name := range am.channels {
		channels = append(channels, name)
	}
	return channels
}

// EmailChannel implements email alert channel
type EmailChannel struct {
	config *config.Config
	logger *zap.Logger
}

// NewEmailChannel creates a new email channel
func NewEmailChannel(cfg *config.Config, logger *zap.Logger) *EmailChannel {
	return &EmailChannel{
		config: cfg,
		logger: logger,
	}
}

// SendAlert sends alert via email
func (ec *EmailChannel) SendAlert(ctx context.Context, alert *Alert) error {
	ec.logger.Info("Sending email alert",
		zap.String("recipient", ec.config.ML.ModelMonitoring.Alerting.Email.Recipients[0]),
		zap.String("alert_type", alert.Type))

	// In a real implementation, you would use an email service like SendGrid, SES, etc.
	// For now, we'll just log the alert
	emailBody := ec.formatEmailBody(alert)
	
	ec.logger.Info("Email alert content",
		zap.String("subject", fmt.Sprintf("[AegisShield] %s Alert - %s", alert.Severity, alert.Type)),
		zap.String("body", emailBody))

	return nil
}

// formatEmailBody formats the alert for email
func (ec *EmailChannel) formatEmailBody(alert *Alert) string {
	return fmt.Sprintf(`
AegisShield ML Pipeline Alert

Alert Type: %s
Severity: %s
Model ID: %s
Feature: %s
Timestamp: %s

Message: %s

Threshold: %.4f
Current Value: %.4f

Metadata:
%s

Please investigate this alert and take appropriate action.

Best regards,
AegisShield Monitoring System
`, alert.Type, alert.Severity, alert.ModelID, alert.Feature,
		alert.Timestamp.Format(time.RFC3339),
		alert.Message,
		alert.Threshold, alert.Value,
		ec.formatMetadata(alert.Metadata))
}

// formatMetadata formats metadata for display
func (ec *EmailChannel) formatMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return "None"
	}

	var parts []string
	for key, value := range metadata {
		parts = append(parts, fmt.Sprintf("  %s: %v", key, value))
	}
	return strings.Join(parts, "\n")
}

func (ec *EmailChannel) GetChannelName() string {
	return "email"
}

// SlackChannel implements Slack alert channel
type SlackChannel struct {
	config *config.Config
	logger *zap.Logger
}

// NewSlackChannel creates a new Slack channel
func NewSlackChannel(cfg *config.Config, logger *zap.Logger) *SlackChannel {
	return &SlackChannel{
		config: cfg,
		logger: logger,
	}
}

// SendAlert sends alert via Slack
func (sc *SlackChannel) SendAlert(ctx context.Context, alert *Alert) error {
	sc.logger.Info("Sending Slack alert",
		zap.String("webhook_url", sc.maskWebhookURL(sc.config.ML.ModelMonitoring.Alerting.Slack.WebhookURL)),
		zap.String("alert_type", alert.Type))

	payload := sc.createSlackPayload(alert)
	
	// In a real implementation, you would send HTTP POST to Slack webhook
	// For now, we'll just log the payload
	sc.logger.Info("Slack alert payload", zap.Any("payload", payload))

	return nil
}

// createSlackPayload creates Slack message payload
func (sc *SlackChannel) createSlackPayload(alert *Alert) map[string]interface{} {
	color := sc.getSeverityColor(alert.Severity)
	
	attachment := map[string]interface{}{
		"color":     color,
		"title":     fmt.Sprintf("%s Alert - %s", alert.Severity, alert.Type),
		"timestamp": alert.Timestamp.Unix(),
		"fields": []map[string]interface{}{
			{
				"title": "Model ID",
				"value": alert.ModelID,
				"short": true,
			},
			{
				"title": "Feature",
				"value": alert.Feature,
				"short": true,
			},
			{
				"title": "Threshold",
				"value": fmt.Sprintf("%.4f", alert.Threshold),
				"short": true,
			},
			{
				"title": "Current Value",
				"value": fmt.Sprintf("%.4f", alert.Value),
				"short": true,
			},
			{
				"title": "Message",
				"value": alert.Message,
				"short": false,
			},
		},
	}

	payload := map[string]interface{}{
		"username":    "AegisShield Monitoring",
		"icon_emoji":  ":warning:",
		"text":        fmt.Sprintf("ML Pipeline Alert: %s", alert.Type),
		"attachments": []map[string]interface{}{attachment},
	}

	return payload
}

// getSeverityColor returns color for alert severity
func (sc *SlackChannel) getSeverityColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "danger"
	case "high":
		return "warning"
	case "medium":
		return "warning"
	case "low":
		return "good"
	default:
		return "#36a64f"
	}
}

// maskWebhookURL masks webhook URL for logging
func (sc *SlackChannel) maskWebhookURL(url string) string {
	if len(url) < 20 {
		return "***masked***"
	}
	return url[:10] + "***masked***" + url[len(url)-10:]
}

func (sc *SlackChannel) GetChannelName() string {
	return "slack"
}

// WebhookChannel implements generic webhook alert channel
type WebhookChannel struct {
	config *config.Config
	logger *zap.Logger
	client *http.Client
}

// NewWebhookChannel creates a new webhook channel
func NewWebhookChannel(cfg *config.Config, logger *zap.Logger) *WebhookChannel {
	return &WebhookChannel{
		config: cfg,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendAlert sends alert via webhook
func (wc *WebhookChannel) SendAlert(ctx context.Context, alert *Alert) error {
	wc.logger.Info("Sending webhook alert",
		zap.String("url", wc.maskURL(wc.config.ML.ModelMonitoring.Alerting.Webhook.URL)),
		zap.String("alert_type", alert.Type))

	payload := wc.createWebhookPayload(alert)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// In a real implementation, you would send HTTP POST to the webhook URL
	// For now, we'll just log the payload
	wc.logger.Info("Webhook alert payload", zap.String("payload", string(payloadBytes)))

	return nil
}

// createWebhookPayload creates webhook payload
func (wc *WebhookChannel) createWebhookPayload(alert *Alert) map[string]interface{} {
	return map[string]interface{}{
		"alert_type": alert.Type,
		"severity":   alert.Severity,
		"model_id":   alert.ModelID,
		"feature":    alert.Feature,
		"message":    alert.Message,
		"threshold":  alert.Threshold,
		"value":      alert.Value,
		"timestamp":  alert.Timestamp.Format(time.RFC3339),
		"metadata":   alert.Metadata,
		"source":     "aegis-shield-ml-pipeline",
	}
}

// maskURL masks URL for logging
func (wc *WebhookChannel) maskURL(url string) string {
	if len(url) < 20 {
		return "***masked***"
	}
	return url[:15] + "***masked***"
}

func (wc *WebhookChannel) GetChannelName() string {
	return "webhook"
}

// TestChannel sends a test alert to verify channel configuration
func (am *AlertManager) TestChannel(ctx context.Context, channelName string) error {
	channel, exists := am.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	testAlert := &Alert{
		Type:      "test",
		Severity:  "low",
		ModelID:   "test-model",
		Feature:   "test-feature",
		Message:   "This is a test alert to verify channel configuration",
		Threshold: 0.5,
		Value:     0.6,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"test": true,
		},
	}

	return channel.SendAlert(ctx, testAlert)
}

// GetChannelStatus returns status of all channels
func (am *AlertManager) GetChannelStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	for name, channel := range am.channels {
		status[name] = map[string]interface{}{
			"name":   channel.GetChannelName(),
			"active": true,
		}
	}

	return status
}