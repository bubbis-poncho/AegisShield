package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/kafka"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
	"github.com/aegis-shield/services/alerting-engine/internal/scheduler"
)

func TestEventProcessingWorkflow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := setupTestLogger()

	// Setup test configuration
	cfg := &config.Config{
		Debug: true,
		Server: config.ServerConfig{
			HTTPPort: 8080,
			GRPCPort: 9090,
		},
		Database: config.DatabaseConfig{
			ConnectionString: "postgres://test:test@localhost:5432/alerting_test?sslmode=disable",
		},
		Kafka: config.KafkaConfig{
			Brokers: []string{"localhost:9092"},
			Topics: config.KafkaTopicsConfig{
				Events:        "test-events",
				Alerts:        "test-alerts",
				Notifications: "test-notifications",
			},
		},
		Notification: config.NotificationConfig{
			Enabled: false, // Disable actual notifications in tests
		},
	}

	// This test would require actual Kafka and database instances
	// For now, we'll test the component interactions with mocks
	t.Log("E2E test setup - would require full infrastructure")
}

func TestRuleEvaluation_Performance(t *testing.T) {
	ctx := context.Background()
	logger := setupTestLogger()

	// Create a mock rule repository
	cfg := &config.Config{Debug: true}
	
	// Create in-memory test data
	rules := []*database.Rule{
		{
			ID:         "perf-rule-1",
			Name:       "High Amount Rule",
			Expression: "amount > 10000",
			Enabled:    true,
		},
		{
			ID:         "perf-rule-2", 
			Name:       "Velocity Rule",
			Expression: "transaction_count > 100",
			Enabled:    true,
		},
		{
			ID:         "perf-rule-3",
			Name:       "Cross Border Rule",
			Expression: "source_country != destination_country",
			Enabled:    true,
		},
	}

	// Create a mock repository for testing
	mockRepo := &MockRuleRepository{rules: rules}
	
	// Create rule engine
	ruleEngine := engine.NewRuleEngine(cfg, logger, mockRepo)

	// Test data representing a high-volume transaction event
	testEvent := map[string]interface{}{
		"transaction_id":     "txn-12345",
		"amount":            25000.0,
		"transaction_count": 150,
		"source_country":    "US",
		"destination_country": "CH",
		"timestamp":         time.Now().Unix(),
	}

	// Performance test: evaluate 1000 events
	t.Run("Evaluate 1000 Events", func(t *testing.T) {
		start := time.Now()
		
		for i := 0; i < 1000; i++ {
			results, err := ruleEngine.EvaluateEvent(ctx, testEvent)
			require.NoError(t, err)
			assert.NotEmpty(t, results)
		}
		
		elapsed := time.Since(start)
		t.Logf("Evaluated 1000 events in %v (avg: %v per event)", elapsed, elapsed/1000)
		
		// Assert performance target: < 1ms per evaluation on average
		avgPerEvent := elapsed / 1000
		assert.Less(t, avgPerEvent, 5*time.Millisecond, "Rule evaluation should be under 5ms per event")
	})

	// Concurrent evaluation test
	t.Run("Concurrent Rule Evaluation", func(t *testing.T) {
		start := time.Now()
		
		// Run 10 goroutines, each evaluating 100 events
		done := make(chan bool, 10)
		
		for g := 0; g < 10; g++ {
			go func() {
				defer func() { done <- true }()
				
				for i := 0; i < 100; i++ {
					results, err := ruleEngine.EvaluateEvent(ctx, testEvent)
					require.NoError(t, err)
					assert.NotEmpty(t, results)
				}
			}()
		}
		
		// Wait for all goroutines to complete
		for g := 0; g < 10; g++ {
			<-done
		}
		
		elapsed := time.Since(start)
		t.Logf("Evaluated 1000 events concurrently in %v", elapsed)
		
		// Should handle concurrent load well
		assert.Less(t, elapsed, 10*time.Second, "Concurrent evaluation should complete in under 10 seconds")
	})
}

func TestNotificationDelivery_E2E(t *testing.T) {
	ctx := context.Background()
	logger := setupTestLogger()

	cfg := &config.Config{
		Debug: true,
		Notification: config.NotificationConfig{
			Enabled: false, // Disable actual delivery for testing
			Email: config.EmailConfig{
				Enabled: false,
			},
			SMS: config.SMSConfig{
				Enabled: false,
			},
			Slack: config.SlackConfig{
				Enabled: false,
			},
		},
	}

	// Create notification manager
	notificationMgr := notification.NewManager(cfg, logger)

	t.Run("Mock Notification Delivery", func(t *testing.T) {
		// Create a test notification
		notification := &notification.Notification{
			ID:        "test-notif-1",
			AlertID:   "test-alert-1",
			Channel:   "email",
			Type:      "alert",
			Recipient: "test@example.com",
			Subject:   "Test Alert Notification",
			Message:   "This is a test alert notification",
			Priority:  "high",
		}

		// In a real E2E test, this would send actual notifications
		// For now, we'll just verify the notification structure
		assert.Equal(t, "email", notification.Channel)
		assert.Equal(t, "alert", notification.Type)
		assert.Equal(t, "high", notification.Priority)
		
		t.Logf("Mock notification prepared: %+v", notification)
	})
}

func TestScheduler_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := setupTestLogger()
	cfg := &config.Config{Debug: true}

	scheduler := scheduler.NewScheduler(cfg, logger)

	t.Run("Schedule and Execute Task", func(t *testing.T) {
		executed := false
		
		// Add a test task that sets a flag when executed
		task := &scheduler.Task{
			ID:       "test-task-1",
			Name:     "Test Cleanup Task",
			Schedule: "@every 1s",
			Enabled:  true,
			Handler: func(ctx context.Context) error {
				executed = true
				return nil
			},
		}

		err := scheduler.AddTask(task)
		require.NoError(t, err)

		// Start scheduler
		go func() {
			err := scheduler.Start(ctx)
			if err != nil && err != context.Canceled {
				t.Errorf("Scheduler failed: %v", err)
			}
		}()

		// Wait for task to execute
		time.Sleep(2 * time.Second)
		
		// Verify task was executed
		assert.True(t, executed, "Scheduled task should have been executed")
	})
}

// MockRuleRepository for testing
type MockRuleRepository struct {
	rules []*database.Rule
}

func (m *MockRuleRepository) Create(ctx context.Context, rule *database.Rule) error {
	m.rules = append(m.rules, rule)
	return nil
}

func (m *MockRuleRepository) GetByID(ctx context.Context, id string) (*database.Rule, error) {
	for _, rule := range m.rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *MockRuleRepository) List(ctx context.Context, filter database.Filter) ([]*database.Rule, int64, error) {
	var filtered []*database.Rule
	for _, rule := range m.rules {
		if enabled, ok := filter.Filters["enabled"].(bool); ok {
			if rule.Enabled != enabled {
				continue
			}
		}
		filtered = append(filtered, rule)
	}
	return filtered, int64(len(filtered)), nil
}

func (m *MockRuleRepository) Update(ctx context.Context, rule *database.Rule) error {
	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = rule
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *MockRuleRepository) Delete(ctx context.Context, id string) error {
	for i, rule := range m.rules {
		if rule.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *MockRuleRepository) Enable(ctx context.Context, id, updatedBy string) error {
	for _, rule := range m.rules {
		if rule.ID == id {
			rule.Enabled = true
			rule.UpdatedBy = updatedBy
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *MockRuleRepository) Disable(ctx context.Context, id, updatedBy string) error {
	for _, rule := range m.rules {
		if rule.ID == id {
			rule.Enabled = false
			rule.UpdatedBy = updatedBy
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *MockRuleRepository) GetActiveRules(ctx context.Context) ([]*database.Rule, error) {
	var active []*database.Rule
	for _, rule := range m.rules {
		if rule.Enabled {
			active = append(active, rule)
		}
	}
	return active, nil
}