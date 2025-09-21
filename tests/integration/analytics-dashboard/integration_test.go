//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aegisshield/analytics-dashboard/internal/config"
	"github.com/aegisshield/analytics-dashboard/internal/dashboard"
	"github.com/aegisshield/analytics-dashboard/internal/data"
	"github.com/aegisshield/analytics-dashboard/internal/handlers"
	"github.com/aegisshield/analytics-dashboard/internal/realtime"
	"github.com/aegisshield/analytics-dashboard/internal/visualization"
)

// TestSuite contains the test environment
type TestSuite struct {
	t       *testing.T
	router  *gin.Engine
	db      *gorm.DB
	redis   *redis.Client
	config  *config.Config
	handler *handlers.Handler
}

// SetupTestSuite initializes the test environment
func SetupTestSuite(t *testing.T) *TestSuite {
	// Load test configuration
	cfg, err := config.Load("../../../services/analytics-dashboard/config/config.yaml")
	require.NoError(t, err)

	// Override with test settings
	cfg.Database.Name = "aegis_analytics_test"
	cfg.Redis.Database = 1

	// Initialize test database and Redis connections
	// (In a real test environment, you would set up actual test databases)

	gin.SetMode(gin.TestMode)

	suite := &TestSuite{
		t:      t,
		config: cfg,
	}

	suite.setupServices()
	suite.setupRouter()

	return suite
}

// setupServices initializes all services for testing
func (s *TestSuite) setupServices() {
	// Mock database and Redis for testing
	// In production tests, use actual test instances

	dashboardManager := dashboard.NewManager(s.db, s.redis)
	dataProcessor := data.NewProcessor(nil) // Mock cache
	vizEngine := visualization.NewEngine(s.redis)
	realtimeManager := realtime.NewManager(s.redis)

	s.handler = handlers.NewHandler(
		dashboardManager,
		dataProcessor,
		vizEngine,
		realtimeManager,
	)
}

// setupRouter initializes the test router
func (s *TestSuite) setupRouter() {
	s.router = gin.New()
	s.router.Use(gin.Recovery())

	// Add test middleware for authentication
	s.router.Use(func(c *gin.Context) {
		c.Set("user_id", "test_user_123")
		c.Next()
	})

	s.handler.RegisterRoutes(s.router)
}

// TearDown cleans up the test environment
func (s *TestSuite) TearDown() {
	// Clean up test data and close connections
	if s.redis != nil {
		s.redis.FlushDB(context.Background())
		s.redis.Close()
	}
}

// Test Dashboard CRUD Operations
func TestDashboardCRUD(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Create Dashboard", func(t *testing.T) {
		dashboard := map[string]interface{}{
			"name":        "Test Dashboard",
			"description": "A test dashboard for integration testing",
			"layout": map[string]interface{}{
				"columns":   3,
				"rows":      4,
				"grid_type": "responsive",
			},
			"settings": map[string]interface{}{
				"theme":            "dark",
				"refresh_interval": 30,
				"auto_refresh":     true,
			},
		}

		body, _ := json.Marshal(dashboard)
		req := httptest.NewRequest("POST", "/api/v1/dashboards", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "dashboard")
	})

	t.Run("Get Dashboards", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dashboards", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "dashboards")
	})

	t.Run("Update Dashboard", func(t *testing.T) {
		dashboardID := "test-dashboard-id"
		updates := map[string]interface{}{
			"name":        "Updated Dashboard",
			"description": "Updated description",
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/dashboards/%s", dashboardID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Clone Dashboard", func(t *testing.T) {
		sourceID := "source-dashboard-id"
		cloneRequest := map[string]interface{}{
			"name": "Cloned Dashboard",
		}

		body, _ := json.Marshal(cloneRequest)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/dashboards/%s/clone", sourceID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

// Test Widget Operations
func TestWidgetOperations(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Create Widget", func(t *testing.T) {
		widget := map[string]interface{}{
			"dashboard_id": "test-dashboard-id",
			"type":         "chart",
			"title":        "Test Chart Widget",
			"position": map[string]interface{}{
				"x": 0,
				"y": 0,
			},
			"size": map[string]interface{}{
				"width":  4,
				"height": 3,
			},
			"config": map[string]interface{}{
				"chart_type": "line",
				"x_axis": map[string]interface{}{
					"label": "Time",
					"type":  "datetime",
				},
				"y_axis": map[string]interface{}{
					"label": "Value",
					"type":  "linear",
				},
			},
			"data_source": map[string]interface{}{
				"type":  "sql",
				"query": "SELECT timestamp, value FROM metrics WHERE timestamp > NOW() - INTERVAL '1 hour'",
			},
		}

		body, _ := json.Marshal(widget)
		req := httptest.NewRequest("POST", "/api/v1/widgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Get Widget Data", func(t *testing.T) {
		widgetID := "test-widget-id"
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/widgets/%s/data", widgetID), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "data")
	})

	t.Run("Refresh Widget Data", func(t *testing.T) {
		widgetID := "test-widget-id"
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/widgets/%s/refresh", widgetID), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "message")
	})
}

// Test Data Query Operations
func TestDataQueries(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Execute SQL Query", func(t *testing.T) {
		queryRequest := map[string]interface{}{
			"source": map[string]interface{}{
				"type": "postgresql",
				"connection": map[string]interface{}{
					"host":     "localhost",
					"port":     5432,
					"database": "aegis_test",
				},
			},
			"query": "SELECT * FROM transactions WHERE amount > 10000 LIMIT 10",
			"time_range": map[string]interface{}{
				"start": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"end":   time.Now().Format(time.RFC3339),
				"field": "created_at",
			},
			"filters": []map[string]interface{}{
				{
					"field":    "status",
					"operator": "eq",
					"value":    "completed",
				},
			},
		}

		body, _ := json.Marshal(queryRequest)
		req := httptest.NewRequest("POST", "/api/v1/data/query", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "metadata")
	})

	t.Run("Get Data Sources", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/data/sources", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "sources")

		sources := response["sources"].([]interface{})
		assert.Greater(t, len(sources), 0)
	})

	t.Run("Test Data Source Connection", func(t *testing.T) {
		dataSource := map[string]interface{}{
			"type": "postgresql",
			"connection": map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "aegis_test",
				"username": "test_user",
				"password": "test_password",
			},
		}

		body, _ := json.Marshal(dataSource)
		req := httptest.NewRequest("POST", "/api/v1/data/sources/test", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "status")
		assert.Equal(t, "success", response["status"])
	})
}

// Test Visualization Operations
func TestVisualizationOperations(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Get Chart Visualization Data", func(t *testing.T) {
		widgetID := "test-widget-id"
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/visualization/%s/chart", widgetID), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		// This might return 404 if no cached data exists, which is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
	})

	t.Run("Update KPI Visualization Data", func(t *testing.T) {
		widgetID := "test-widget-id"
		kpiData := map[string]interface{}{
			"value":          75.5,
			"previous_value": 68.2,
			"target":         80.0,
			"unit":           "%",
			"metadata": map[string]interface{}{
				"title":       "System Uptime",
				"description": "System availability percentage",
				"category":    "performance",
			},
		}

		body, _ := json.Marshal(kpiData)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/visualization/%s/kpi", widgetID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test Real-time Operations
func TestRealtimeOperations(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Get Real-time Stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/realtime/stats", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "connected_clients")
		assert.Contains(t, response, "uptime")
	})
}

// Test System Health and Metrics
func TestSystemHealth(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Health Check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "status")
		assert.Equal(t, "healthy", response["status"])
		assert.Contains(t, response, "services")
	})

	t.Run("Get Metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/system/metrics", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "http_requests_total")
		assert.Contains(t, response, "memory_usage_bytes")
		assert.Contains(t, response, "cpu_usage_percent")
	})

	t.Run("Get Version", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/system/version", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "version")
		assert.Contains(t, response, "build_date")
		assert.Contains(t, response, "go_version")
	})
}

// Test Error Handling
func TestErrorHandling(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Invalid Dashboard Creation", func(t *testing.T) {
		invalidDashboard := map[string]interface{}{
			"invalid_field": "invalid_value",
		}

		body, _ := json.Marshal(invalidDashboard)
		req := httptest.NewRequest("POST", "/api/v1/dashboards", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Non-existent Dashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dashboards/non-existent-id", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid Query Request", func(t *testing.T) {
		invalidQuery := map[string]interface{}{
			"invalid_query": "this should fail",
		}

		body, _ := json.Marshal(invalidQuery)
		req := httptest.NewRequest("POST", "/api/v1/data/query", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test Performance and Load
func TestPerformance(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("Multiple Concurrent Requests", func(t *testing.T) {
		concurrency := 10
		results := make(chan int, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		// Collect results
		for i := 0; i < concurrency; i++ {
			statusCode := <-results
			assert.Equal(t, http.StatusOK, statusCode)
		}
	})

	t.Run("Response Time Check", func(t *testing.T) {
		start := time.Now()

		req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Less(t, duration, 100*time.Millisecond, "Health check should respond quickly")
	})
}

// Test Service Integration
func TestServiceIntegration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("End-to-End Dashboard Creation and Data Flow", func(t *testing.T) {
		// 1. Create a dashboard
		dashboard := map[string]interface{}{
			"name":        "Integration Test Dashboard",
			"description": "End-to-end test dashboard",
		}

		body, _ := json.Marshal(dashboard)
		req := httptest.NewRequest("POST", "/api/v1/dashboards", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var dashboardResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &dashboardResponse)
		require.NoError(t, err)

		// 2. Create a widget for the dashboard
		widget := map[string]interface{}{
			"dashboard_id": "test-dashboard-id", // In real test, get from step 1
			"type":         "kpi",
			"title":        "Test KPI",
			"config": map[string]interface{}{
				"threshold": map[string]interface{}{
					"warning":   75.0,
					"critical":  90.0,
					"direction": "above",
				},
			},
		}

		body, _ = json.Marshal(widget)
		req = httptest.NewRequest("POST", "/api/v1/widgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 3. Query data for the widget
		req = httptest.NewRequest("GET", "/api/v1/widgets/test-widget-id/data", nil)
		w = httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 4. Check system health
		req = httptest.NewRequest("GET", "/api/v1/system/health", nil)
		w = httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
