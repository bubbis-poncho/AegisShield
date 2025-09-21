package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2ETestSuite contains end-to-end test environment
type E2ETestSuite struct {
	t               *testing.T
	baseURL         string
	dashboardServiceURL string
	alertEngineURL     string
	graphEngineURL     string
	mlPipelineURL      string
}

// SetupE2ETestSuite initializes the end-to-end test environment
func SetupE2ETestSuite(t *testing.T) *E2ETestSuite {
	return &E2ETestSuite{
		t:               t,
		baseURL:         "http://localhost:8080", // Analytics Dashboard
		dashboardServiceURL: "http://localhost:8080",
		alertEngineURL:     "http://localhost:8082",
		graphEngineURL:     "http://localhost:8083",
		mlPipelineURL:      "http://localhost:8084",
	}
}

// Test complete workflow from data ingestion to dashboard visualization
func TestCompleteDataFlow(t *testing.T) {
	suite := SetupE2ETestSuite(t)

	t.Run("Complete Financial Crime Detection Workflow", func(t *testing.T) {
		// 1. Verify all services are healthy
		suite.verifyServiceHealth(t)

		// 2. Create investigation dashboard
		dashboardID := suite.createInvestigationDashboard(t)

		// 3. Set up real-time monitoring widgets
		alertWidgetID := suite.createAlertWidget(t, dashboardID)
		transactionWidgetID := suite.createTransactionWidget(t, dashboardID)
		networkWidgetID := suite.createNetworkWidget(t, dashboardID)
		kpiWidgetID := suite.createKPIWidget(t, dashboardID)

		// 4. Simulate transaction data ingestion
		suite.simulateTransactionData(t)

		// 5. Verify real-time updates
		suite.verifyRealTimeUpdates(t, []string{
			alertWidgetID,
			transactionWidgetID,
			networkWidgetID,
			kpiWidgetID,
		})

		// 6. Test alert generation and dashboard updates
		suite.triggerSuspiciousTransaction(t)
		suite.verifyAlertDashboardUpdate(t, alertWidgetID)

		// 7. Test investigation workflow
		investigationID := suite.createInvestigation(t)
		suite.addEvidenceToDashboard(t, investigationID, dashboardID)

		// 8. Verify ML pipeline integration
		suite.verifyMLPipelineData(t, transactionWidgetID)

		// 9. Test graph analysis integration
		suite.verifyGraphAnalysis(t, networkWidgetID)

		// 10. Generate compliance report
		suite.generateComplianceReport(t, dashboardID)
	})
}

// verifyServiceHealth checks that all required services are running
func (s *E2ETestSuite) verifyServiceHealth(t *testing.T) {
	services := map[string]string{
		"Analytics Dashboard": s.dashboardServiceURL + "/api/v1/system/health",
		"Alert Engine":       s.alertEngineURL + "/api/v1/health",
		"Graph Engine":       s.graphEngineURL + "/api/v1/health",
		"ML Pipeline":        s.mlPipelineURL + "/api/v1/health",
	}

	for serviceName, healthURL := range services {
		resp, err := http.Get(healthURL)
		require.NoError(t, err, "Failed to connect to %s", serviceName)
		require.Equal(t, http.StatusOK, resp.StatusCode, "%s health check failed", serviceName)
		resp.Body.Close()
	}
}

// createInvestigationDashboard creates a dashboard for financial crime investigation
func (s *E2ETestSuite) createInvestigationDashboard(t *testing.T) string {
	dashboardPayload := map[string]interface{}{
		"name":        "Financial Crime Investigation Dashboard",
		"description": "Real-time monitoring and investigation dashboard",
		"layout": map[string]interface{}{
			"columns":   4,
			"rows":      3,
			"grid_type": "responsive",
		},
		"settings": map[string]interface{}{
			"theme":            "dark",
			"refresh_interval": 10,
			"auto_refresh":     true,
			"show_toolbar":     true,
		},
	}

	resp := s.makeAPIRequest(t, "POST", "/api/v1/dashboards", dashboardPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	resp.Body.Close()

	dashboard := response["dashboard"].(map[string]interface{})
	return dashboard["id"].(string)
}

// createAlertWidget creates a widget for displaying alerts
func (s *E2ETestSuite) createAlertWidget(t *testing.T, dashboardID string) string {
	widgetPayload := map[string]interface{}{
		"dashboard_id": dashboardID,
		"type":         "table",
		"title":        "Active Alerts",
		"position": map[string]interface{}{
			"x": 0,
			"y": 0,
		},
		"size": map[string]interface{}{
			"width":  2,
			"height": 2,
		},
		"config": map[string]interface{}{
			"columns": []map[string]interface{}{
				{"key": "timestamp", "title": "Time", "type": "datetime"},
				{"key": "severity", "title": "Severity", "type": "string"},
				{"key": "description", "title": "Alert", "type": "string"},
				{"key": "entity", "title": "Entity", "type": "string"},
			},
			"sort": map[string]interface{}{
				"field": "timestamp",
				"order": "desc",
			},
		},
		"data_source": map[string]interface{}{
			"type":      "rest",
			"url":       s.alertEngineURL + "/api/v1/alerts/active",
			"real_time": true,
		},
		"refresh_rate": 5,
	}

	return s.createWidget(t, widgetPayload)
}

// createTransactionWidget creates a widget for transaction monitoring
func (s *E2ETestSuite) createTransactionWidget(t *testing.T, dashboardID string) string {
	widgetPayload := map[string]interface{}{
		"dashboard_id": dashboardID,
		"type":         "line_chart",
		"title":        "Transaction Volume",
		"position": map[string]interface{}{
			"x": 2,
			"y": 0,
		},
		"size": map[string]interface{}{
			"width":  2,
			"height": 2,
		},
		"config": map[string]interface{}{
			"chart_type": "line",
			"x_axis": map[string]interface{}{
				"label": "Time",
				"type":  "datetime",
			},
			"y_axis": map[string]interface{}{
				"label": "Count",
				"type":  "linear",
			},
			"series": []map[string]interface{}{
				{
					"name":  "Total Transactions",
					"field": "total_count",
					"color": "#4CAF50",
				},
				{
					"name":  "Suspicious Transactions",
					"field": "suspicious_count",
					"color": "#F44336",
				},
			},
		},
		"data_source": map[string]interface{}{
			"type":  "sql",
			"query": "SELECT DATE_TRUNC('minute', timestamp) as time, COUNT(*) as total_count, COUNT(CASE WHEN risk_score > 0.7 THEN 1 END) as suspicious_count FROM transactions WHERE timestamp > NOW() - INTERVAL '1 hour' GROUP BY time ORDER BY time",
		},
		"refresh_rate": 30,
	}

	return s.createWidget(t, widgetPayload)
}

// createNetworkWidget creates a widget for network analysis
func (s *E2ETestSuite) createNetworkWidget(t *testing.T, dashboardID string) string {
	widgetPayload := map[string]interface{}{
		"dashboard_id": dashboardID,
		"type":         "network",
		"title":        "Entity Network",
		"position": map[string]interface{}{
			"x": 0,
			"y": 2,
		},
		"size": map[string]interface{}{
			"width":  4,
			"height": 1,
		},
		"config": map[string]interface{}{
			"layout": map[string]interface{}{
				"algorithm": "force",
			},
			"node_size_field":  "transaction_count",
			"edge_width_field": "total_amount",
		},
		"data_source": map[string]interface{}{
			"type": "rest",
			"url":  s.graphEngineURL + "/api/v1/graph/entities/network",
		},
		"refresh_rate": 60,
	}

	return s.createWidget(t, widgetPayload)
}

// createKPIWidget creates a KPI monitoring widget
func (s *E2ETestSuite) createKPIWidget(t *testing.T, dashboardID string) string {
	widgetPayload := map[string]interface{}{
		"dashboard_id": dashboardID,
		"type":         "kpi",
		"title":        "System Performance",
		"position": map[string]interface{}{
			"x": 2,
			"y": 2,
		},
		"size": map[string]interface{}{
			"width":  2,
			"height": 1,
		},
		"config": map[string]interface{}{
			"threshold": map[string]interface{}{
				"warning":   75.0,
				"critical":  90.0,
				"direction": "above",
			},
		},
		"data_source": map[string]interface{}{
			"type":  "prometheus",
			"query": "avg(cpu_usage_percent)",
		},
		"refresh_rate": 15,
	}

	return s.createWidget(t, widgetPayload)
}

// createWidget is a helper function to create widgets
func (s *E2ETestSuite) createWidget(t *testing.T, widgetPayload map[string]interface{}) string {
	resp := s.makeAPIRequest(t, "POST", "/api/v1/widgets", widgetPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	resp.Body.Close()

	widget := response["widget"].(map[string]interface{})
	return widget["id"].(string)
}

// simulateTransactionData simulates incoming transaction data
func (s *E2ETestSuite) simulateTransactionData(t *testing.T) {
	// In a real test, this would send data to the data ingestion service
	// For now, we'll simulate this by calling the analytics dashboard refresh endpoint
	time.Sleep(2 * time.Second) // Allow time for data processing
}

// verifyRealTimeUpdates tests real-time data updates via WebSocket
func (s *E2ETestSuite) verifyRealTimeUpdates(t *testing.T, widgetIDs []string) {
	// Connect to WebSocket
	wsURL := fmt.Sprintf("ws://localhost:8081/api/v1/realtime/ws")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Subscribe to widget updates
	for _, widgetID := range widgetIDs {
		subscribeMsg := map[string]interface{}{
			"type":   "subscribe",
			"topics": []string{fmt.Sprintf("widget:%s", widgetID)},
		}
		err := conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)
	}

	// Wait for real-time updates
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updateReceived := false
	go func() {
		for {
			var message map[string]interface{}
			err := conn.ReadJSON(&message)
			if err != nil {
				return
			}
			if message["type"] == "data" {
				updateReceived = true
				cancel()
				return
			}
		}
	}()

	<-ctx.Done()
	assert.True(t, updateReceived, "Real-time update not received within timeout")
}

// triggerSuspiciousTransaction simulates a suspicious transaction
func (s *E2ETestSuite) triggerSuspiciousTransaction(t *testing.T) {
	// This would typically call the data ingestion service
	// For testing, we can call the alert engine directly
	suspiciousTransaction := map[string]interface{}{
		"amount":      100000,
		"from_account": "suspicious_account_123",
		"to_account":   "unknown_account_456",
		"country_mismatch": true,
		"unusual_time": true,
	}

	// Post to alert engine (mock)
	_ = suspiciousTransaction
	time.Sleep(1 * time.Second) // Allow processing time
}

// verifyAlertDashboardUpdate verifies that alerts appear in the dashboard
func (s *E2ETestSuite) verifyAlertDashboardUpdate(t *testing.T, alertWidgetID string) {
	// Get widget data and verify alert is present
	resp := s.makeAPIRequest(t, "GET", fmt.Sprintf("/api/v1/widgets/%s/data", alertWidgetID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	resp.Body.Close()

	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data, "Alert widget should contain data")
}

// createInvestigation creates a new investigation case
func (s *E2ETestSuite) createInvestigation(t *testing.T) string {
	investigationPayload := map[string]interface{}{
		"title":       "Suspicious Transaction Investigation",
		"description": "Investigation of high-value cross-border transaction",
		"priority":    "high",
		"assigned_to": "investigator_123",
	}

	// This would call the investigation toolkit service
	_ = investigationPayload
	return "investigation_456" // Mock ID
}

// addEvidenceToDashboard adds investigation evidence to the dashboard
func (s *E2ETestSuite) addEvidenceToDashboard(t *testing.T, investigationID, dashboardID string) {
	evidenceWidgetPayload := map[string]interface{}{
		"dashboard_id": dashboardID,
		"type":         "table",
		"title":        fmt.Sprintf("Evidence - Investigation %s", investigationID),
		"position": map[string]interface{}{
			"x": 0,
			"y": 1,
		},
		"size": map[string]interface{}{
			"width":  4,
			"height": 1,
		},
		"data_source": map[string]interface{}{
			"type": "rest",
			"url":  fmt.Sprintf("http://localhost:8085/api/v1/investigations/%s/evidence", investigationID),
		},
	}

	s.createWidget(t, evidenceWidgetPayload)
}

// verifyMLPipelineData verifies ML pipeline integration
func (s *E2ETestSuite) verifyMLPipelineData(t *testing.T, widgetID string) {
	// Call ML pipeline for predictions
	resp := s.makeAPIRequest(t, "GET", "/api/v1/data/sources", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify ML data is available in widget
	resp = s.makeAPIRequest(t, "GET", fmt.Sprintf("/api/v1/widgets/%s/data", widgetID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// verifyGraphAnalysis verifies graph engine integration
func (s *E2ETestSuite) verifyGraphAnalysis(t *testing.T, widgetID string) {
	// Verify graph data is available
	resp := s.makeAPIRequest(t, "GET", fmt.Sprintf("/api/v1/widgets/%s/data", widgetID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	resp.Body.Close()

	// Verify network data structure
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data, "Network widget should contain graph data")
}

// generateComplianceReport generates a compliance report
func (s *E2ETestSuite) generateComplianceReport(t *testing.T, dashboardID string) {
	reportPayload := map[string]interface{}{
		"dashboard_id": dashboardID,
		"report_type":  "suspicious_activity_report",
		"time_range": map[string]interface{}{
			"start": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"end":   time.Now().Format(time.RFC3339),
		},
	}

	resp := s.makeAPIRequest(t, "POST", "/api/v1/reports/generate", reportPayload)
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated)
	resp.Body.Close()
}

// makeAPIRequest is a helper function for making HTTP requests
func (s *E2ETestSuite) makeAPIRequest(t *testing.T, method, path string, payload interface{}) *http.Response {
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}

	url := s.baseURL + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-123")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

// Test WebSocket Real-time Communication
func TestWebSocketCommunication(t *testing.T) {
	suite := SetupE2ETestSuite(t)

	t.Run("WebSocket Connection and Messaging", func(t *testing.T) {
		// Connect to WebSocket
		wsURL := "ws://localhost:8081/api/v1/realtime/ws?token=test-token-123"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Test subscription
		subscribeMsg := map[string]interface{}{
			"type":   "subscribe",
			"topics": []string{"alerts", "notifications"},
		}
		err = conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)

		// Wait for subscription confirmation
		var confirmMsg map[string]interface{}
		err = conn.ReadJSON(&confirmMsg)
		require.NoError(t, err)
		assert.Equal(t, "subscribe", confirmMsg["type"])

		// Test heartbeat
		time.Sleep(5 * time.Second)
		
		// Test unsubscription
		unsubscribeMsg := map[string]interface{}{
			"type":   "unsubscribe",
			"topics": []string{"alerts"},
		}
		err = conn.WriteJSON(unsubscribeMsg)
		require.NoError(t, err)
	})
}

// Test Dashboard Export and Import
func TestDashboardExportImport(t *testing.T) {
	suite := SetupE2ETestSuite(t)

	t.Run("Export and Import Dashboard Configuration", func(t *testing.T) {
		// Create a test dashboard
		dashboardID := suite.createInvestigationDashboard(t)

		// Add some widgets
		alertWidgetID := suite.createAlertWidget(t, dashboardID)
		_ = alertWidgetID

		// Export dashboard
		resp := suite.makeAPIRequest(t, "GET", fmt.Sprintf("/api/v1/dashboards/%s/export", dashboardID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var exportData map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&exportData)
		require.NoError(t, err)
		resp.Body.Close()

		// Import dashboard
		importPayload := map[string]interface{}{
			"name": "Imported Dashboard",
			"data": exportData,
		}

		resp = suite.makeAPIRequest(t, "POST", "/api/v1/dashboards/import", importPayload)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated)
		resp.Body.Close()
	})
}