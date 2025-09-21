package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/rest/api/v2010"
	"golang.org/x/time/rate"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
)

// Manager handles multi-channel notification delivery
type Manager struct {
	config                *config.Config
	logger                *slog.Logger
	notificationRepo      *database.NotificationRepository
	emailTemplates        *template.Template
	smsTemplates         *template.Template
	slackClient          *SlackClient
	teamsClient          *TeamsClient
	webhookClient        *WebhookClient
	pagerDutyClient      *PagerDutyClient
	rateLimiters         map[string]*rate.Limiter
	rateLimiterMutex     sync.RWMutex
	retryQueue           chan *database.Notification
	workerCount          int
	shutdownChan         chan struct{}
	wg                   sync.WaitGroup
}

// NewManager creates a new notification manager
func NewManager(
	cfg *config.Config,
	logger *slog.Logger,
	notificationRepo *database.NotificationRepository,
) (*Manager, error) {
	manager := &Manager{
		config:           cfg,
		logger:           logger,
		notificationRepo: notificationRepo,
		rateLimiters:     make(map[string]*rate.Limiter),
		retryQueue:       make(chan *database.Notification, cfg.Notifications.QueueSize),
		workerCount:      cfg.Notifications.WorkerCount,
		shutdownChan:     make(chan struct{}),
	}

	// Initialize email templates
	if err := manager.initializeEmailTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize email templates: %w", err)
	}

	// Initialize SMS templates
	if err := manager.initializeSMSTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize SMS templates: %w", err)
	}

	// Initialize notification clients
	if err := manager.initializeClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize notification clients: %w", err)
	}

	// Initialize rate limiters
	manager.initializeRateLimiters()

	return manager, nil
}

// Start starts the notification manager workers
func (m *Manager) Start(ctx context.Context) {
	m.logger.Info("Starting notification manager", "workers", m.workerCount)

	// Start worker goroutines
	for i := 0; i < m.workerCount; i++ {
		m.wg.Add(1)
		go m.worker(ctx, i)
	}

	// Start retry processor
	m.wg.Add(1)
	go m.retryProcessor(ctx)
}

// Stop stops the notification manager
func (m *Manager) Stop() {
	m.logger.Info("Stopping notification manager")
	close(m.shutdownChan)
	close(m.retryQueue)
	m.wg.Wait()
	m.logger.Info("Notification manager stopped")
}

// SendNotification sends a notification through the appropriate channel
func (m *Manager) SendNotification(ctx context.Context, notification *database.Notification) error {
	// Check rate limiting
	if !m.checkRateLimit(notification.Channel, notification.Recipient) {
		return fmt.Errorf("rate limit exceeded for channel %s, recipient %s", 
			notification.Channel, notification.Recipient)
	}

	// Update status to sending
	if err := m.notificationRepo.UpdateStatus(ctx, notification.ID, "sending"); err != nil {
		m.logger.Error("Failed to update notification status to sending",
			"notification_id", notification.ID,
			"error", err)
	}

	var err error
	switch notification.Channel {
	case "email":
		err = m.sendEmail(ctx, notification)
	case "sms":
		err = m.sendSMS(ctx, notification)
	case "slack":
		err = m.sendSlack(ctx, notification)
	case "teams":
		err = m.sendTeams(ctx, notification)
	case "webhook":
		err = m.sendWebhook(ctx, notification)
	case "pagerduty":
		err = m.sendPagerDuty(ctx, notification)
	default:
		err = fmt.Errorf("unsupported notification channel: %s", notification.Channel)
	}

	if err != nil {
		m.logger.Error("Failed to send notification",
			"notification_id", notification.ID,
			"channel", notification.Channel,
			"error", err)

		// Increment retry count and queue for retry if applicable
		if notification.RetryCount < notification.MaxRetries {
			if retryErr := m.notificationRepo.IncrementRetryCount(ctx, notification.ID, err.Error()); retryErr != nil {
				m.logger.Error("Failed to increment retry count", "error", retryErr)
			}
			
			// Add to retry queue with delay
			go func() {
				time.Sleep(m.calculateRetryDelay(notification.RetryCount))
				select {
				case m.retryQueue <- notification:
				case <-m.shutdownChan:
				}
			}()
		} else {
			// Mark as failed
			if updateErr := m.notificationRepo.UpdateStatus(ctx, notification.ID, "failed"); updateErr != nil {
				m.logger.Error("Failed to update notification status to failed", "error", updateErr)
			}
		}
		return err
	}

	// Mark as sent
	if err := m.notificationRepo.UpdateStatus(ctx, notification.ID, "sent"); err != nil {
		m.logger.Error("Failed to update notification status to sent", "error", err)
	}

	m.logger.Info("Notification sent successfully",
		"notification_id", notification.ID,
		"channel", notification.Channel,
		"recipient", notification.Recipient)

	return nil
}

// ProcessPendingNotifications processes pending notifications
func (m *Manager) ProcessPendingNotifications(ctx context.Context) error {
	notifications, err := m.notificationRepo.GetPendingNotifications(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to get pending notifications: %w", err)
	}

	for _, notification := range notifications {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := m.SendNotification(ctx, notification); err != nil {
				m.logger.Error("Failed to send pending notification",
					"notification_id", notification.ID,
					"error", err)
			}
		}
	}

	return nil
}

// Worker processes notifications
func (m *Manager) worker(ctx context.Context, workerID int) {
	defer m.wg.Done()
	
	m.logger.Debug("Starting notification worker", "worker_id", workerID)
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			if err := m.ProcessPendingNotifications(ctx); err != nil {
				m.logger.Error("Worker failed to process pending notifications",
					"worker_id", workerID,
					"error", err)
			}
		}
	}
}

// RetryProcessor handles notification retries
func (m *Manager) retryProcessor(ctx context.Context) {
	defer m.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdownChan:
			return
		case notification := <-m.retryQueue:
			if notification != nil {
				if err := m.SendNotification(ctx, notification); err != nil {
					m.logger.Error("Failed to retry notification",
						"notification_id", notification.ID,
						"retry_count", notification.RetryCount,
						"error", err)
				}
			}
		}
	}
}

// Email sending methods

func (m *Manager) sendEmail(ctx context.Context, notification *database.Notification) error {
	switch m.config.Notifications.Email.Provider {
	case "sendgrid":
		return m.sendEmailViaSendGrid(ctx, notification)
	case "smtp":
		return m.sendEmailViaSMTP(ctx, notification)
	default:
		return fmt.Errorf("unsupported email provider: %s", m.config.Notifications.Email.Provider)
	}
}

func (m *Manager) sendEmailViaSendGrid(ctx context.Context, notification *database.Notification) error {
	from := mail.NewEmail(m.config.Notifications.Email.FromName, m.config.Notifications.Email.FromAddress)
	to := mail.NewEmail("", notification.Recipient)
	
	// Render email content
	content, err := m.renderEmailContent(notification)
	if err != nil {
		return fmt.Errorf("failed to render email content: %w", err)
	}

	message := mail.NewSingleEmail(from, notification.Subject, to, content.Text, content.HTML)
	
	client := sendgrid.NewSendClient(m.config.Notifications.Email.SendGrid.APIKey)
	response, err := client.SendWithContext(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	// Update notification with external ID
	notification.ExternalID = &response.Headers["X-Message-Id"][0]
	if err := m.notificationRepo.Update(ctx, notification); err != nil {
		m.logger.Error("Failed to update notification with external ID", "error", err)
	}

	return nil
}

func (m *Manager) sendEmailViaSMTP(ctx context.Context, notification *database.Notification) error {
	// Render email content
	content, err := m.renderEmailContent(notification)
	if err != nil {
		return fmt.Errorf("failed to render email content: %w", err)
	}

	// Create email message
	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		notification.Recipient, notification.Subject, content.HTML)

	// SMTP authentication
	auth := smtp.PlainAuth("",
		m.config.Notifications.Email.SMTP.Username,
		m.config.Notifications.Email.SMTP.Password,
		m.config.Notifications.Email.SMTP.Host)

	// Send email
	addr := fmt.Sprintf("%s:%d", m.config.Notifications.Email.SMTP.Host, m.config.Notifications.Email.SMTP.Port)
	err = smtp.SendMail(addr, auth, m.config.Notifications.Email.FromAddress, []string{notification.Recipient}, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// SMS sending methods

func (m *Manager) sendSMS(ctx context.Context, notification *database.Notification) error {
	if !m.config.Notifications.SMS.Enabled {
		return fmt.Errorf("SMS notifications are disabled")
	}

	// Render SMS content
	content, err := m.renderSMSContent(notification)
	if err != nil {
		return fmt.Errorf("failed to render SMS content: %w", err)
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: m.config.Notifications.SMS.Twilio.AccountSID,
		Password: m.config.Notifications.SMS.Twilio.AuthToken,
	})

	params := &v2010.CreateMessageParams{}
	params.SetTo(notification.Recipient)
	params.SetFrom(m.config.Notifications.SMS.FromNumber)
	params.SetBody(content)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS via Twilio: %w", err)
	}

	// Update notification with external ID
	notification.ExternalID = resp.Sid
	if err := m.notificationRepo.Update(ctx, notification); err != nil {
		m.logger.Error("Failed to update notification with external ID", "error", err)
	}

	return nil
}

// Slack sending methods

func (m *Manager) sendSlack(ctx context.Context, notification *database.Notification) error {
	if m.slackClient == nil {
		return fmt.Errorf("Slack client not initialized")
	}
	return m.slackClient.SendMessage(ctx, notification)
}

// Teams sending methods

func (m *Manager) sendTeams(ctx context.Context, notification *database.Notification) error {
	if m.teamsClient == nil {
		return fmt.Errorf("Teams client not initialized")
	}
	return m.teamsClient.SendMessage(ctx, notification)
}

// Webhook sending methods

func (m *Manager) sendWebhook(ctx context.Context, notification *database.Notification) error {
	if m.webhookClient == nil {
		return fmt.Errorf("Webhook client not initialized")
	}
	return m.webhookClient.SendWebhook(ctx, notification)
}

// PagerDuty sending methods

func (m *Manager) sendPagerDuty(ctx context.Context, notification *database.Notification) error {
	if m.pagerDutyClient == nil {
		return fmt.Errorf("PagerDuty client not initialized")
	}
	return m.pagerDutyClient.SendAlert(ctx, notification)
}

// Template rendering

func (m *Manager) renderEmailContent(notification *database.Notification) (*EmailContent, error) {
	var textBuf, htmlBuf bytes.Buffer
	
	templateData := m.createTemplateData(notification)
	
	// Render text template
	textTemplate := "email-text"
	if notification.TemplateID != nil && *notification.TemplateID != "" {
		textTemplate = *notification.TemplateID + "-text"
	}
	
	if err := m.emailTemplates.ExecuteTemplate(&textBuf, textTemplate, templateData); err != nil {
		return nil, fmt.Errorf("failed to render email text template: %w", err)
	}
	
	// Render HTML template
	htmlTemplate := "email-html"
	if notification.TemplateID != nil && *notification.TemplateID != "" {
		htmlTemplate = *notification.TemplateID + "-html"
	}
	
	if err := m.emailTemplates.ExecuteTemplate(&htmlBuf, htmlTemplate, templateData); err != nil {
		return nil, fmt.Errorf("failed to render email HTML template: %w", err)
	}
	
	return &EmailContent{
		Text: textBuf.String(),
		HTML: htmlBuf.String(),
	}, nil
}

func (m *Manager) renderSMSContent(notification *database.Notification) (string, error) {
	var buf bytes.Buffer
	
	templateData := m.createTemplateData(notification)
	
	templateName := "sms-default"
	if notification.TemplateID != nil && *notification.TemplateID != "" {
		templateName = *notification.TemplateID + "-sms"
	}
	
	if err := m.smsTemplates.ExecuteTemplate(&buf, templateName, templateData); err != nil {
		return "", fmt.Errorf("failed to render SMS template: %w", err)
	}
	
	return buf.String(), nil
}

func (m *Manager) createTemplateData(notification *database.Notification) map[string]interface{} {
	data := map[string]interface{}{
		"Subject":     notification.Subject,
		"Message":     notification.Message,
		"Recipient":   notification.Recipient,
		"Channel":     notification.Channel,
		"Priority":    notification.Priority,
		"CreatedAt":   notification.CreatedAt,
	}
	
	// Add template data if available
	if notification.TemplateData != nil {
		var templateData map[string]interface{}
		if err := json.Unmarshal(notification.TemplateData, &templateData); err == nil {
			for k, v := range templateData {
				data[k] = v
			}
		}
	}
	
	return data
}

// Rate limiting

func (m *Manager) checkRateLimit(channel, recipient string) bool {
	m.rateLimiterMutex.RLock()
	limiter, exists := m.rateLimiters[channel]
	m.rateLimiterMutex.RUnlock()
	
	if !exists {
		return true // No rate limit configured
	}
	
	return limiter.Allow()
}

func (m *Manager) initializeRateLimiters() {
	// Email rate limiter
	if m.config.Notifications.Email.RateLimit.Enabled {
		m.rateLimiters["email"] = rate.NewLimiter(
			rate.Limit(m.config.Notifications.Email.RateLimit.RequestsPerMinute)/60,
			m.config.Notifications.Email.RateLimit.Burst,
		)
	}
	
	// SMS rate limiter
	if m.config.Notifications.SMS.RateLimit.Enabled {
		m.rateLimiters["sms"] = rate.NewLimiter(
			rate.Limit(m.config.Notifications.SMS.RateLimit.RequestsPerMinute)/60,
			m.config.Notifications.SMS.RateLimit.Burst,
		)
	}
	
	// Slack rate limiter
	if m.config.Notifications.Slack.RateLimit.Enabled {
		m.rateLimiters["slack"] = rate.NewLimiter(
			rate.Limit(m.config.Notifications.Slack.RateLimit.RequestsPerMinute)/60,
			m.config.Notifications.Slack.RateLimit.Burst,
		)
	}
	
	// Teams rate limiter
	if m.config.Notifications.Teams.RateLimit.Enabled {
		m.rateLimiters["teams"] = rate.NewLimiter(
			rate.Limit(m.config.Notifications.Teams.RateLimit.RequestsPerMinute)/60,
			m.config.Notifications.Teams.RateLimit.Burst,
		)
	}
}

func (m *Manager) calculateRetryDelay(retryCount int) time.Duration {
	// Exponential backoff with jitter
	baseDelay := time.Duration(m.config.Notifications.RetryBaseDelayMs) * time.Millisecond
	delay := baseDelay * time.Duration(1<<retryCount)
	
	// Add jitter (up to 20% of delay)
	jitter := time.Duration(float64(delay) * 0.2 * (0.5 - 0.5))
	return delay + jitter
}

// Initialization methods

func (m *Manager) initializeEmailTemplates() error {
	templates := template.New("email")
	
	// Default email templates
	defaultTextTemplate := `
Subject: {{.Subject}}

{{.Message}}

Alert Details:
- Priority: {{.Priority}}
- Channel: {{.Channel}}
- Created: {{.CreatedAt.Format "2006-01-02 15:04:05 UTC"}}
`
	
	defaultHTMLTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
</head>
<body>
    <h2>{{.Subject}}</h2>
    <p>{{.Message}}</p>
    <hr>
    <table>
        <tr><td><strong>Priority:</strong></td><td>{{.Priority}}</td></tr>
        <tr><td><strong>Channel:</strong></td><td>{{.Channel}}</td></tr>
        <tr><td><strong>Created:</strong></td><td>{{.CreatedAt.Format "2006-01-02 15:04:05 UTC"}}</td></tr>
    </table>
</body>
</html>
`
	
	_, err := templates.New("email-text").Parse(defaultTextTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email text template: %w", err)
	}
	
	_, err = templates.New("email-html").Parse(defaultHTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email HTML template: %w", err)
	}
	
	m.emailTemplates = templates
	return nil
}

func (m *Manager) initializeSMSTemplates() error {
	templates := template.New("sms")
	
	// Default SMS template
	defaultSMSTemplate := `ALERT: {{.Subject}} - {{.Message}} (Priority: {{.Priority}})`
	
	_, err := templates.New("sms-default").Parse(defaultSMSTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse SMS template: %w", err)
	}
	
	m.smsTemplates = templates
	return nil
}

func (m *Manager) initializeClients() error {
	// Initialize Slack client
	if m.config.Notifications.Slack.Enabled {
		m.slackClient = NewSlackClient(m.config.Notifications.Slack, m.logger)
	}
	
	// Initialize Teams client
	if m.config.Notifications.Teams.Enabled {
		m.teamsClient = NewTeamsClient(m.config.Notifications.Teams, m.logger)
	}
	
	// Initialize Webhook client
	if m.config.Notifications.Webhooks.Enabled {
		m.webhookClient = NewWebhookClient(m.config.Notifications.Webhooks, m.logger)
	}
	
	// Initialize PagerDuty client
	if m.config.Notifications.PagerDuty.Enabled {
		m.pagerDutyClient = NewPagerDutyClient(m.config.Notifications.PagerDuty, m.logger)
	}
	
	return nil
}

// Supporting types

type EmailContent struct {
	Text string
	HTML string
}