package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
)

func TestAlertRepository_Integration(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("alerting_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(postgres.DefaultWaitStrategy),
	)
	require.NoError(t, err)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := database.Connect(connStr)
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	logger := setupTestLogger()
	err = database.RunMigrations(db, logger)
	require.NoError(t, err)

	// Create repository
	alertRepo := database.NewAlertRepository(db, logger)

	t.Run("Create and Get Alert", func(t *testing.T) {
		alert := &database.Alert{
			ID:          "test-alert-1",
			RuleID:      "test-rule-1",
			Title:       "Test Alert",
			Description: "This is a test alert",
			Severity:    "high",
			Type:        "manual",
			Priority:    "urgent",
			Status:      "active",
			Source:      "test",
			CreatedBy:   "test-user",
			UpdatedBy:   "test-user",
		}

		// Create alert
		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		// Get alert by ID
		retrieved, err := alertRepo.GetByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, alert.ID, retrieved.ID)
		assert.Equal(t, alert.Title, retrieved.Title)
		assert.Equal(t, alert.Severity, retrieved.Severity)
	})

	t.Run("List Alerts with Filter", func(t *testing.T) {
		// Create multiple alerts
		alerts := []*database.Alert{
			{
				ID:        "test-alert-2",
				Title:     "High Severity Alert",
				Severity:  "high",
				Status:    "active",
				CreatedBy: "test-user",
				UpdatedBy: "test-user",
			},
			{
				ID:        "test-alert-3",
				Title:     "Medium Severity Alert",
				Severity:  "medium",
				Status:    "resolved",
				CreatedBy: "test-user",
				UpdatedBy: "test-user",
			},
		}

		for _, alert := range alerts {
			err := alertRepo.Create(ctx, alert)
			require.NoError(t, err)
		}

		// Filter by severity
		filter := database.Filter{
			Filters: map[string]interface{}{
				"severity": "high",
			},
			Limit: 10,
		}

		results, total, err := alertRepo.List(ctx, filter)
		require.NoError(t, err)
		assert.Greater(t, total, int64(0))
		
		// Verify all results have high severity
		for _, alert := range results {
			assert.Equal(t, "high", alert.Severity)
		}
	})

	t.Run("Acknowledge Alert", func(t *testing.T) {
		alert := &database.Alert{
			ID:        "test-alert-4",
			Title:     "Alert to Acknowledge",
			Status:    "active",
			CreatedBy: "test-user",
			UpdatedBy: "test-user",
		}

		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		// Acknowledge the alert
		err = alertRepo.Acknowledge(ctx, alert.ID, "test-analyst")
		require.NoError(t, err)

		// Verify status changed
		retrieved, err := alertRepo.GetByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, "acknowledged", retrieved.Status)
		assert.Equal(t, "test-analyst", retrieved.AcknowledgedBy)
		assert.NotNil(t, retrieved.AcknowledgedAt)
	})

	t.Run("Resolve Alert", func(t *testing.T) {
		alert := &database.Alert{
			ID:        "test-alert-5",
			Title:     "Alert to Resolve",
			Status:    "active",
			CreatedBy: "test-user",
			UpdatedBy: "test-user",
		}

		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		// Resolve the alert
		resolution := "False positive - normal business activity"
		err = alertRepo.Resolve(ctx, alert.ID, "test-analyst", resolution)
		require.NoError(t, err)

		// Verify status changed
		retrieved, err := alertRepo.GetByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, "resolved", retrieved.Status)
		assert.Equal(t, "test-analyst", retrieved.ResolvedBy)
		assert.Equal(t, resolution, retrieved.Resolution)
		assert.NotNil(t, retrieved.ResolvedAt)
	})

	t.Run("Get Alert Statistics", func(t *testing.T) {
		since := time.Now().Add(-1 * time.Hour)
		stats, err := alertRepo.GetStatsByTimeRange(ctx, since, time.Now())
		require.NoError(t, err)
		assert.NotNil(t, stats)
		
		// Verify we have some statistics
		if totalStats, ok := stats["total"].(map[string]interface{}); ok {
			assert.Greater(t, totalStats["count"], int64(0))
		}
	})
}

func TestRuleRepository_Integration(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("alerting_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(postgres.DefaultWaitStrategy),
	)
	require.NoError(t, err)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := database.Connect(connStr)
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	logger := setupTestLogger()
	err = database.RunMigrations(db, logger)
	require.NoError(t, err)

	// Create repository
	ruleRepo := database.NewRuleRepository(db, logger)

	t.Run("Create and Get Rule", func(t *testing.T) {
		rule := &database.Rule{
			ID:          "test-rule-1",
			Name:        "High Amount Transaction",
			Description: "Detects transactions over $10,000",
			Expression:  "amount > 10000",
			Severity:    "high",
			Priority:    "medium",
			Type:        "threshold",
			Category:    "financial",
			Enabled:     true,
			CreatedBy:   "test-user",
			UpdatedBy:   "test-user",
		}

		// Create rule
		err := ruleRepo.Create(ctx, rule)
		require.NoError(t, err)

		// Get rule by ID
		retrieved, err := ruleRepo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, rule.ID, retrieved.ID)
		assert.Equal(t, rule.Name, retrieved.Name)
		assert.Equal(t, rule.Expression, retrieved.Expression)
		assert.True(t, retrieved.Enabled)
	})

	t.Run("List Active Rules", func(t *testing.T) {
		filter := database.Filter{
			Filters: map[string]interface{}{
				"enabled": true,
			},
			Limit: 10,
		}

		rules, total, err := ruleRepo.List(ctx, filter)
		require.NoError(t, err)
		assert.Greater(t, total, int64(0))
		
		// Verify all rules are enabled
		for _, rule := range rules {
			assert.True(t, rule.Enabled)
		}
	})

	t.Run("Enable and Disable Rule", func(t *testing.T) {
		rule := &database.Rule{
			ID:        "test-rule-2",
			Name:      "Test Rule for Enable/Disable",
			Enabled:   true,
			CreatedBy: "test-user",
			UpdatedBy: "test-user",
		}

		err := ruleRepo.Create(ctx, rule)
		require.NoError(t, err)

		// Disable rule
		err = ruleRepo.Disable(ctx, rule.ID, "test-admin")
		require.NoError(t, err)

		// Verify disabled
		retrieved, err := ruleRepo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.False(t, retrieved.Enabled)

		// Enable rule
		err = ruleRepo.Enable(ctx, rule.ID, "test-admin")
		require.NoError(t, err)

		// Verify enabled
		retrieved, err = ruleRepo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.Enabled)
	})
}

func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}