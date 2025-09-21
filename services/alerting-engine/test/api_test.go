package test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/handlers"
	"github.com/aegis-shield/services/alerting-engine/internal/server"
	alertingpb "github.com/aegis-shield/shared/proto"
)

func TestHTTPHandlers_API(t *testing.T) {
	logger := setupTestLogger()
	cfg := &config.Config{Debug: true}

	// Create mock repositories
	alertRepo := &MockAlertRepository{}
	ruleRepo := &MockRuleRepository{}
	notificationRepo := &MockNotificationRepository{}
	escalationRepo := &MockEscalationRepository{}

	// Create HTTP handler
	httpHandler := handlers.NewHTTPHandler(
		cfg,
		logger,
		alertRepo,
		ruleRepo,
		notificationRepo,
		escalationRepo,
		nil, // rule engine
		nil, // notification manager
		nil, // event processor
		nil, // scheduler
	)

	// Setup router
	router := mux.NewRouter()
	httpHandler.RegisterRoutes(router)

	t.Run("Health Check Endpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "alerting-engine", response["service"])
	})

	t.Run("Create Alert Endpoint", func(t *testing.T) {
		alertData := map[string]interface{}{
			"title":       "Test Alert",
			"description": "This is a test alert",
			"severity":    "high",
			"type":        "manual",
			"priority":    "urgent",
			"source":      "test",
			"created_by":  "test-user",
		}

		jsonData, err := json.Marshal(alertData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/alerts", strings.NewReader(string(jsonData)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response database.Alert
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Test Alert", response.Title)
		assert.Equal(t, "high", response.Severity)
		assert.Equal(t, "urgent", response.Priority)
	})

	t.Run("List Alerts Endpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/alerts?severity=high&limit=10", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "alerts")
		assert.Contains(t, response, "total_count")
		assert.Equal(t, float64(10), response["page_size"])
	})

	t.Run("Acknowledge Alert Endpoint", func(t *testing.T) {
		ackData := map[string]interface{}{
			"acknowledged_by": "test-analyst",
		}

		jsonData, err := json.Marshal(ackData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/alerts/test-alert-1/acknowledge", strings.NewReader(string(jsonData)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
	})

	t.Run("Invalid Request Handling", func(t *testing.T) {
		// Test invalid JSON
		req, err := http.NewRequest("POST", "/alerts", strings.NewReader("invalid json"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
	})
}

func TestGRPCServer_API(t *testing.T) {
	logger := setupTestLogger()
	cfg := &config.Config{Debug: true}

	// Create mock repositories
	alertRepo := &MockAlertRepository{}
	ruleRepo := &MockRuleRepository{}
	notificationRepo := &MockNotificationRepository{}
	escalationRepo := &MockEscalationRepository{}

	// Create gRPC server
	grpcServer := server.NewGRPCServer(
		cfg,
		logger,
		alertRepo,
		ruleRepo,
		notificationRepo,
		escalationRepo,
		nil, // rule engine
		nil, // notification manager
		nil, // event processor
		nil, // scheduler
	)

	// Setup test gRPC connection
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	alertingpb.RegisterAlertingEngineServer(s, grpcServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()
	defer s.Stop()

	// Create client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := alertingpb.NewAlertingEngineClient(conn)

	t.Run("CreateAlert gRPC", func(t *testing.T) {
		req := &alertingpb.CreateAlertRequest{
			Alert: &alertingpb.Alert{
				RuleId:      "test-rule-1",
				Title:       "gRPC Test Alert",
				Description: "Alert created via gRPC",
				Severity:    "high",
				Type:        "manual",
				Priority:    "medium",
				Source:      "grpc-test",
				CreatedBy:   "grpc-user",
			},
		}

		resp, err := client.CreateAlert(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.Alert)
		assert.Equal(t, "gRPC Test Alert", resp.Alert.Title)
		assert.Equal(t, "high", resp.Alert.Severity)
	})

	t.Run("GetAlert gRPC", func(t *testing.T) {
		req := &alertingpb.GetAlertRequest{
			AlertId: "test-alert-1",
		}

		resp, err := client.GetAlert(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.Alert)
		assert.Equal(t, "test-alert-1", resp.Alert.Id)
	})

	t.Run("ListAlerts gRPC", func(t *testing.T) {
		req := &alertingpb.ListAlertsRequest{
			PageSize: 10,
			Filter: map[string]string{
				"severity": "high",
			},
		}

		resp, err := client.ListAlerts(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.Alerts)
		assert.LessOrEqual(t, len(resp.Alerts), 10)
	})

	t.Run("AcknowledgeAlert gRPC", func(t *testing.T) {
		req := &alertingpb.AcknowledgeAlertRequest{
			AlertId:         "test-alert-1",
			AcknowledgedBy: "grpc-analyst",
		}

		resp, err := client.AcknowledgeAlert(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("CreateRule gRPC", func(t *testing.T) {
		req := &alertingpb.CreateRuleRequest{
			Rule: &alertingpb.Rule{
				Name:        "gRPC Test Rule",
				Description: "Rule created via gRPC",
				Expression:  "amount > 5000",
				Severity:    "medium",
				Priority:    "medium",
				Type:        "threshold",
				Category:    "financial",
				Enabled:     true,
				CreatedBy:   "grpc-user",
			},
		}

		resp, err := client.CreateRule(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.Rule)
		assert.Equal(t, "gRPC Test Rule", resp.Rule.Name)
		assert.True(t, resp.Rule.Enabled)
	})

	t.Run("SystemHealth gRPC", func(t *testing.T) {
		req := &alertingpb.SystemHealthRequest{}

		resp, err := client.SystemHealth(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, "alerting-engine", resp.Service)
	})
}

// Mock implementations for testing

type MockAlertRepository struct {
	alerts []*database.Alert
}

func (m *MockAlertRepository) Create(ctx context.Context, alert *database.Alert) error {
	if alert.ID == "" {
		alert.ID = "generated-alert-id"
	}
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *MockAlertRepository) GetByID(ctx context.Context, id string) (*database.Alert, error) {
	for _, alert := range m.alerts {
		if alert.ID == id {
			return alert, nil
		}
	}
	return &database.Alert{ID: id, Title: "Mock Alert"}, nil
}

func (m *MockAlertRepository) List(ctx context.Context, filter database.Filter) ([]*database.Alert, int64, error) {
	return m.alerts, int64(len(m.alerts)), nil
}

func (m *MockAlertRepository) Update(ctx context.Context, alert *database.Alert) error {
	return nil
}

func (m *MockAlertRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockAlertRepository) Acknowledge(ctx context.Context, id, acknowledgedBy string) error {
	return nil
}

func (m *MockAlertRepository) Resolve(ctx context.Context, id, resolvedBy, resolution string) error {
	return nil
}

func (m *MockAlertRepository) Escalate(ctx context.Context, id, escalatedBy string) error {
	return nil
}

func (m *MockAlertRepository) GetStatsByTimeRange(ctx context.Context, from, to time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total": map[string]interface{}{
			"count": int64(10),
		},
	}, nil
}

type MockNotificationRepository struct{}

func (m *MockNotificationRepository) Create(ctx context.Context, notif *database.Notification) error {
	return nil
}

func (m *MockNotificationRepository) GetByID(ctx context.Context, id string) (*database.Notification, error) {
	return &database.Notification{ID: id}, nil
}

func (m *MockNotificationRepository) List(ctx context.Context, filter database.Filter) ([]*database.Notification, int64, error) {
	return []*database.Notification{}, 0, nil
}

func (m *MockNotificationRepository) Update(ctx context.Context, notif *database.Notification) error {
	return nil
}

func (m *MockNotificationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return nil
}

func (m *MockNotificationRepository) GetPendingRetries(ctx context.Context) ([]*database.Notification, error) {
	return []*database.Notification{}, nil
}

func (m *MockNotificationRepository) GetStatsByTimeRange(ctx context.Context, from, to time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

type MockEscalationRepository struct{}

func (m *MockEscalationRepository) CreatePolicy(ctx context.Context, policy *database.EscalationPolicy) error {
	return nil
}

func (m *MockEscalationRepository) GetPolicyByID(ctx context.Context, id string) (*database.EscalationPolicy, error) {
	return &database.EscalationPolicy{ID: id}, nil
}

func (m *MockEscalationRepository) ListPolicies(ctx context.Context, filter database.Filter) ([]*database.EscalationPolicy, int64, error) {
	return []*database.EscalationPolicy{}, 0, nil
}

func (m *MockEscalationRepository) UpdatePolicy(ctx context.Context, policy *database.EscalationPolicy) error {
	return nil
}

func (m *MockEscalationRepository) DeletePolicy(ctx context.Context, id string) error {
	return nil
}

func (m *MockEscalationRepository) CreateEvent(ctx context.Context, event *database.EscalationEvent) error {
	return nil
}

func (m *MockEscalationRepository) GetEventsByAlert(ctx context.Context, alertID string) ([]*database.EscalationEvent, error) {
	return []*database.EscalationEvent{}, nil
}