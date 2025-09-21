package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
)

// SlackClient handles Slack notifications
type SlackClient struct {
	config config.SlackConfig
	logger *slog.Logger
	client *http.Client
}

// NewSlackClient creates a new Slack client
func NewSlackClient(config config.SlackConfig, logger *slog.Logger) *SlackClient {
	return &SlackClient{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		},
	}
}

// SendMessage sends a message to Slack
func (s *SlackClient) SendMessage(ctx context.Context, notification *database.Notification) error {
	// Create Slack message payload
	payload := SlackMessage{
		Channel: notification.Recipient,
		Text:    notification.Message,
		Blocks: []SlackBlock{
			{
				Type: "section",
				Text: &SlackText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*%s*\n%s", notification.Subject, notification.Message),
				},
			},
			{
				Type: "divider",
			},
			{
				Type: "section",
				Fields: []SlackField{
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Priority:*\n%s", notification.Priority),
					},
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Created:*\n%s", notification.CreatedAt.Format("2006-01-02 15:04:05 UTC")),
					},
				},
			},
		},
	}

	// Add template data if available
	if notification.TemplateData != nil {
		var templateData map[string]interface{}
		if err := json.Unmarshal(notification.TemplateData, &templateData); err == nil {
			if alertData, ok := templateData["alert"].(map[string]interface{}); ok {
				if severity, ok := alertData["severity"].(string); ok {
					payload.Blocks = append(payload.Blocks, SlackBlock{
						Type: "section",
						Text: &SlackText{
							Type: "mrkdwn",
							Text: fmt.Sprintf("*Severity:* %s", severity),
						},
					})
				}
			}
		}
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	s.logger.Debug("Slack message sent successfully",
		"notification_id", notification.ID,
		"channel", notification.Recipient)

	return nil
}

// TeamsClient handles Microsoft Teams notifications
type TeamsClient struct {
	config config.TeamsConfig
	logger *slog.Logger
	client *http.Client
}

// NewTeamsClient creates a new Teams client
func NewTeamsClient(config config.TeamsConfig, logger *slog.Logger) *TeamsClient {
	return &TeamsClient{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		},
	}
}

// SendMessage sends a message to Microsoft Teams
func (t *TeamsClient) SendMessage(ctx context.Context, notification *database.Notification) error {
	// Create Teams message payload
	payload := TeamsMessage{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: t.getThemeColor(notification.Priority),
		Summary:    notification.Subject,
		Sections: []TeamsSection{
			{
				ActivityTitle:    notification.Subject,
				ActivitySubtitle: fmt.Sprintf("Priority: %s", notification.Priority),
				Text:            notification.Message,
				Facts: []TeamsFact{
					{
						Name:  "Priority",
						Value: notification.Priority,
					},
					{
						Name:  "Channel",
						Value: notification.Channel,
					},
					{
						Name:  "Created",
						Value: notification.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
					},
				},
			},
		},
	}

	// Add template data if available
	if notification.TemplateData != nil {
		var templateData map[string]interface{}
		if err := json.Unmarshal(notification.TemplateData, &templateData); err == nil {
			if alertData, ok := templateData["alert"].(map[string]interface{}); ok {
				if severity, ok := alertData["severity"].(string); ok {
					payload.Sections[0].Facts = append(payload.Sections[0].Facts, TeamsFact{
						Name:  "Severity",
						Value: severity,
					})
				}
				if alertID, ok := alertData["id"].(string); ok {
					payload.Sections[0].Facts = append(payload.Sections[0].Facts, TeamsFact{
						Name:  "Alert ID",
						Value: alertID,
					})
				}
			}
		}
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Teams payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", t.config.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create Teams request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Teams message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Teams API returned status %d", resp.StatusCode)
	}

	t.logger.Debug("Teams message sent successfully",
		"notification_id", notification.ID,
		"webhook_url", t.config.WebhookURL)

	return nil
}

func (t *TeamsClient) getThemeColor(priority string) string {
	switch priority {
	case "critical":
		return "FF0000" // Red
	case "high":
		return "FF9900" // Orange
	case "medium":
		return "FFCC00" // Yellow
	case "low":
		return "00CC00" // Green
	default:
		return "0078D4" // Blue (default Teams color)
	}
}

// WebhookClient handles generic webhook notifications
type WebhookClient struct {
	config config.WebhooksConfig
	logger *slog.Logger
	client *http.Client
}

// NewWebhookClient creates a new webhook client
func NewWebhookClient(config config.WebhooksConfig, logger *slog.Logger) *WebhookClient {
	return &WebhookClient{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		},
	}
}

// SendWebhook sends a webhook notification
func (w *WebhookClient) SendWebhook(ctx context.Context, notification *database.Notification) error {
	// Create webhook payload
	payload := WebhookPayload{
		NotificationID: notification.ID,
		AlertID:        notification.AlertID,
		RuleID:         notification.RuleID,
		Channel:        notification.Channel,
		Recipient:      notification.Recipient,
		Subject:        notification.Subject,
		Message:        notification.Message,
		Priority:       notification.Priority,
		CreatedAt:      notification.CreatedAt,
		Metadata:       make(map[string]interface{}),
	}

	// Add metadata
	if notification.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(notification.Metadata, &metadata); err == nil {
			payload.Metadata = metadata
		}
	}

	// Add template data
	if notification.TemplateData != nil {
		var templateData map[string]interface{}
		if err := json.Unmarshal(notification.TemplateData, &templateData); err == nil {
			payload.TemplateData = templateData
		}
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Determine webhook URL (use recipient as URL for webhook notifications)
	webhookURL := notification.Recipient

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AegisShield-AlertingEngine/1.0")

	// Add authentication headers if configured
	if w.config.AuthHeader != "" && w.config.AuthToken != "" {
		req.Header.Set(w.config.AuthHeader, w.config.AuthToken)
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	w.logger.Debug("Webhook sent successfully",
		"notification_id", notification.ID,
		"webhook_url", webhookURL,
		"status_code", resp.StatusCode)

	return nil
}

// PagerDutyClient handles PagerDuty notifications
type PagerDutyClient struct {
	config config.PagerDutyConfig
	logger *slog.Logger
	client *http.Client
}

// NewPagerDutyClient creates a new PagerDuty client
func NewPagerDutyClient(config config.PagerDutyConfig, logger *slog.Logger) *PagerDutyClient {
	return &PagerDutyClient{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		},
	}
}

// SendAlert sends an alert to PagerDuty
func (p *PagerDutyClient) SendAlert(ctx context.Context, notification *database.Notification) error {
	// Create PagerDuty event payload
	payload := PagerDutyEvent{
		RoutingKey:  p.config.IntegrationKey,
		EventAction: "trigger",
		Payload: PagerDutyPayload{
			Summary:   notification.Subject,
			Source:    "AegisShield Alerting Engine",
			Severity:  p.mapPriorityToSeverity(notification.Priority),
			Timestamp: notification.CreatedAt.Format(time.RFC3339),
			Component: "alerting-engine",
			Group:     "financial-crimes",
			Class:     "alert",
		},
	}

	// Add custom details from template data
	if notification.TemplateData != nil {
		var templateData map[string]interface{}
		if err := json.Unmarshal(notification.TemplateData, &templateData); err == nil {
			payload.Payload.CustomDetails = templateData
		}
	}

	// Add dedup key if available (use alert ID)
	if notification.AlertID != "" {
		payload.DedupKey = notification.AlertID
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal PagerDuty payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://events.pagerduty.com/v2/enqueue", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create PagerDuty request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PagerDuty alert: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response PagerDutyResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse PagerDuty response: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("PagerDuty API returned status %d: %s", resp.StatusCode, response.Message)
	}

	// Update notification with dedup key
	if response.DedupKey != "" {
		externalID := response.DedupKey
		notification.ExternalID = &externalID
	}

	p.logger.Debug("PagerDuty alert sent successfully",
		"notification_id", notification.ID,
		"dedup_key", response.DedupKey,
		"status", response.Status)

	return nil
}

func (p *PagerDutyClient) mapPriorityToSeverity(priority string) string {
	switch priority {
	case "critical":
		return "critical"
	case "high":
		return "error"
	case "medium":
		return "warning"
	case "low":
		return "info"
	default:
		return "info"
	}
}

// Message structure types

type SlackMessage struct {
	Channel string       `json:"channel,omitempty"`
	Text    string       `json:"text"`
	Blocks  []SlackBlock `json:"blocks,omitempty"`
}

type SlackBlock struct {
	Type   string       `json:"type"`
	Text   *SlackText   `json:"text,omitempty"`
	Fields []SlackField `json:"fields,omitempty"`
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackField struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type TeamsMessage struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor string         `json:"themeColor"`
	Summary    string         `json:"summary"`
	Sections   []TeamsSection `json:"sections"`
}

type TeamsSection struct {
	ActivityTitle    string      `json:"activityTitle"`
	ActivitySubtitle string      `json:"activitySubtitle,omitempty"`
	Text            string      `json:"text,omitempty"`
	Facts           []TeamsFact `json:"facts,omitempty"`
}

type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type WebhookPayload struct {
	NotificationID string                 `json:"notification_id"`
	AlertID        string                 `json:"alert_id"`
	RuleID         string                 `json:"rule_id"`
	Channel        string                 `json:"channel"`
	Recipient      string                 `json:"recipient"`
	Subject        string                 `json:"subject"`
	Message        string                 `json:"message"`
	Priority       string                 `json:"priority"`
	CreatedAt      time.Time              `json:"created_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	TemplateData   map[string]interface{} `json:"template_data,omitempty"`
}

type PagerDutyEvent struct {
	RoutingKey   string            `json:"routing_key"`
	EventAction  string            `json:"event_action"`
	DedupKey     string            `json:"dedup_key,omitempty"`
	Payload      PagerDutyPayload  `json:"payload"`
}

type PagerDutyPayload struct {
	Summary       string                 `json:"summary"`
	Source        string                 `json:"source"`
	Severity      string                 `json:"severity"`
	Timestamp     string                 `json:"timestamp"`
	Component     string                 `json:"component,omitempty"`
	Group         string                 `json:"group,omitempty"`
	Class         string                 `json:"class,omitempty"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

type PagerDutyResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
}