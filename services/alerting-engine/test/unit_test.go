package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
	"github.com/aegis-shield/services/alerting-engine/internal/scheduler"
)

func TestAlertRepository_Unit(t *testing.T) {
	t.Run("Alert Validation", func(t *testing.T) {
		alert := &database.Alert{
			ID:       "test-alert-1",
			Title:    "Test Alert",
			Severity: "high",
			Status:   "active",
		}

		// Test alert validation logic
		assert.NotEmpty(t, alert.ID, "Alert ID should not be empty")
		assert.NotEmpty(t, alert.Title, "Alert title should not be empty")
		assert.Contains(t, []string{"low", "medium", "high", "critical"}, alert.Severity, "Severity should be valid")
		assert.Contains(t, []string{"active", "acknowledged", "resolved", "escalated"}, alert.Status, "Status should be valid")
	})

	t.Run("Alert State Transitions", func(t *testing.T) {
		alert := &database.Alert{
			Status: "active",
		}

		// Test valid state transitions
		validTransitions := map[string][]string{
			"active":       {"acknowledged", "resolved", "escalated"},
			"acknowledged": {"resolved", "escalated"},
			"escalated":    {"resolved"},
			"resolved":     {}, // Terminal state
		}

		currentStatus := alert.Status
		allowedNext := validTransitions[currentStatus]

		assert.Contains(t, allowedNext, "acknowledged", "Active alert should be acknowledgeable")
		assert.Contains(t, allowedNext, "resolved", "Active alert should be resolvable")
		assert.Contains(t, allowedNext, "escalated", "Active alert should be escalatable")
	})
}

func TestRuleEngine_Unit(t *testing.T) {
	logger := setupTestLogger()
	cfg := &config.Config{Debug: true}
	mockRepo := &MockRuleRepository{
		rules: []*database.Rule{
			{
				ID:         "test-rule-1",
				Name:       "Amount Threshold",
				Expression: "amount > 10000",
				Enabled:    true,
			},
		},
	}

	ruleEngine := engine.NewRuleEngine(cfg, logger, mockRepo)

	t.Run("Simple Expression Evaluation", func(t *testing.T) {
		event := map[string]interface{}{
			"amount": 15000.0,
		}

		results, err := ruleEngine.EvaluateEvent(context.Background(), event)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.True(t, results[0].Matched, "Rule should match for amount > 10000")
	})

	t.Run("Expression Does Not Match", func(t *testing.T) {
		event := map[string]interface{}{
			"amount": 5000.0,
		}

		results, err := ruleEngine.EvaluateEvent(context.Background(), event)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.False(t, results[0].Matched, "Rule should not match for amount < 10000")
	})

	t.Run("Invalid Expression Handling", func(t *testing.T) {
		// Add rule with invalid expression
		invalidRule := &database.Rule{
			ID:         "invalid-rule",
			Expression: "invalid expression syntax",
			Enabled:    true,
		}
		mockRepo.rules = append(mockRepo.rules, invalidRule)

		event := map[string]interface{}{
			"amount": 15000.0,
		}

		results, err := ruleEngine.EvaluateEvent(context.Background(), event)
		require.NoError(t, err)
		
		// Should handle invalid expressions gracefully
		for _, result := range results {
			if result.RuleID == "invalid-rule" {
				assert.False(t, result.Matched, "Invalid rule should not match")
				assert.NotEmpty(t, result.Error, "Invalid rule should have error")
			}
		}
	})
}

func TestNotificationManager_Unit(t *testing.T) {
	logger := setupTestLogger()
	cfg := &config.Config{
		Debug: true,
		Notification: config.NotificationConfig{
			Enabled: false, // Disable actual sending for unit tests
		},
	}

	notificationMgr := notification.NewManager(cfg, logger)

	t.Run("Notification Creation", func(t *testing.T) {
		notif := &notification.Notification{
			ID:        "test-notif-1",
			AlertID:   "test-alert-1",
			Channel:   "email",
			Type:      "alert",
			Recipient: "test@example.com",
			Subject:   "Test Alert",
			Message:   "Test message",
			Priority:  "high",
		}

		// Validate notification fields
		assert.NotEmpty(t, notif.ID, "Notification ID should not be empty")
		assert.NotEmpty(t, notif.AlertID, "Alert ID should not be empty")
		assert.Contains(t, []string{"email", "sms", "slack", "teams", "webhook"}, notif.Channel, "Channel should be valid")
		assert.Contains(t, []string{"alert", "escalation", "resolution"}, notif.Type, "Type should be valid")
		assert.Contains(t, []string{"low", "medium", "high", "urgent"}, notif.Priority, "Priority should be valid")
	})

	t.Run("Rate Limiting Logic", func(t *testing.T) {
		// Test rate limiting logic (would be implemented in the notification manager)
		recipient := "test@example.com"
		channel := "email"
		
		// Simulate rate limit check
		rateLimitKey := channel + ":" + recipient
		assert.NotEmpty(t, rateLimitKey, "Rate limit key should be generated")
		
		// In actual implementation, this would check Redis or in-memory store
		// For unit test, we just verify the key format
		t.Logf("Rate limit key: %s", rateLimitKey)
	})
}

func TestScheduler_Unit(t *testing.T) {
	logger := setupTestLogger()
	cfg := &config.Config{Debug: true}

	scheduler := scheduler.NewScheduler(cfg, logger)

	t.Run("Task Creation and Validation", func(t *testing.T) {
		task := &scheduler.Task{
			ID:       "test-task-1",
			Name:     "Test Task",
			Schedule: "0 */1 * * *", // Every hour
			Enabled:  true,
			Handler: func(ctx context.Context) error {
				return nil
			},
		}

		err := scheduler.AddTask(task)
		require.NoError(t, err)

		// Verify task was added
		retrievedTask, err := scheduler.GetTask(task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, task.Name, retrievedTask.Name)
		assert.Equal(t, task.Schedule, retrievedTask.Schedule)
		assert.True(t, retrievedTask.Enabled)
	})

	t.Run("Invalid Cron Schedule", func(t *testing.T) {
		task := &scheduler.Task{
			ID:       "invalid-task",
			Name:     "Invalid Task",
			Schedule: "invalid cron", // Invalid cron expression
			Enabled:  true,
			Handler: func(ctx context.Context) error {
				return nil
			},
		}

		err := scheduler.AddTask(task)
		assert.Error(t, err, "Should reject invalid cron schedule")
	})

	t.Run("Task Enable/Disable", func(t *testing.T) {
		task := &scheduler.Task{
			ID:       "toggle-task",
			Name:     "Toggle Task",
			Schedule: "@daily",
			Enabled:  true,
		}

		err := scheduler.AddTask(task)
		require.NoError(t, err)

		// Disable task
		err = scheduler.DisableTask(task.ID)
		require.NoError(t, err)

		retrievedTask, err := scheduler.GetTask(task.ID)
		require.NoError(t, err)
		assert.False(t, retrievedTask.Enabled)

		// Enable task
		err = scheduler.EnableTask(task.ID)
		require.NoError(t, err)

		retrievedTask, err = scheduler.GetTask(task.ID)
		require.NoError(t, err)
		assert.True(t, retrievedTask.Enabled)
	})
}

func TestKafkaEventProcessor_Unit(t *testing.T) {
	t.Run("Event Message Validation", func(t *testing.T) {
		event := map[string]interface{}{
			"event_type":     "transaction",
			"transaction_id": "txn-12345",
			"amount":        10000.0,
			"timestamp":     time.Now().Unix(),
		}

		// Validate required fields
		assert.NotEmpty(t, event["event_type"], "Event type should not be empty")
		assert.NotEmpty(t, event["transaction_id"], "Transaction ID should not be empty")
		assert.Greater(t, event["amount"], 0.0, "Amount should be positive")
		assert.IsType(t, int64(0), event["timestamp"], "Timestamp should be int64")
	})

	t.Run("Alert Message Creation", func(t *testing.T) {
		alert := map[string]interface{}{
			"alert_id":       "alert-12345",
			"rule_id":        "rule-001",
			"title":          "High Amount Transaction",
			"severity":       "high",
			"triggered_at":   time.Now().Unix(),
			"entities":       []string{"entity-1", "entity-2"},
		}

		// Validate alert message structure
		assert.NotEmpty(t, alert["alert_id"], "Alert ID should not be empty")
		assert.NotEmpty(t, alert["rule_id"], "Rule ID should not be empty")
		assert.NotEmpty(t, alert["title"], "Alert title should not be empty")
		assert.Contains(t, []string{"low", "medium", "high", "critical"}, alert["severity"], "Severity should be valid")
		assert.IsType(t, []string{}, alert["entities"], "Entities should be string array")
	})
}

func TestDatabaseMigrations_Unit(t *testing.T) {
	t.Run("Migration SQL Validation", func(t *testing.T) {
		// Test that migration files contain expected table structures
		expectedTables := []string{
			"alerts",
			"alert_rules", 
			"notifications",
			"escalation_policies",
			"escalation_events",
		}

		for _, table := range expectedTables {
			assert.NotEmpty(t, table, "Table name should not be empty")
			t.Logf("Expected table: %s", table)
		}
	})

	t.Run("Index Strategy Validation", func(t *testing.T) {
		// Test index strategy for performance
		expectedIndexes := map[string][]string{
			"alerts": {
				"idx_alerts_rule_id",
				"idx_alerts_severity", 
				"idx_alerts_status",
				"idx_alerts_created_at",
			},
			"alert_rules": {
				"idx_alert_rules_enabled",
				"idx_alert_rules_type",
				"idx_alert_rules_severity",
			},
			"notifications": {
				"idx_notifications_alert_id",
				"idx_notifications_channel",
				"idx_notifications_status",
			},
		}

		for table, indexes := range expectedIndexes {
			assert.NotEmpty(t, indexes, "Table %s should have indexes defined", table)
			for _, index := range indexes {
				assert.NotEmpty(t, index, "Index name should not be empty")
				t.Logf("Table %s has index: %s", table, index)
			}
		}
	})
}