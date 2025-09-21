package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aegisshield/graph-engine/internal/analytics"
	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/engine"
	"github.com/aegisshield/graph-engine/internal/handlers"
	"github.com/aegisshield/graph-engine/internal/patterns"
	"github.com/aegisshield/graph-engine/internal/resolution"
)

type IntegrationTestSuite struct {
	router           *mux.Router
	patternDetector  *patterns.PatternDetector
	analytics        *analytics.GraphAnalytics
	entityResolver   *resolution.EntityResolver
	httpHandlers     *handlers.HTTPHandlers
	enhancedHandlers *handlers.EnhancedHTTPHandlers
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	// Mock configuration for testing
	cfg := config.Config{
		Environment: "test",
		Server: config.ServerConfig{
			HTTPPort:     8080,
			GRPCPort:     9090,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  30,
		},
		Database: config.DatabaseConfig{
			URL:             "postgres://test:test@localhost/test_db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
		Neo4j: config.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			Username: "neo4j",
			Password: "test",
		},
	}

	// For integration tests, we would need actual Neo4j connection
	// For now, we'll mock the dependencies
	mockNeo4jClient := &mockNeo4jClient{}
	mockGraphEngine := &mockGraphEngine{}

	// Create logger for testing
	logger := createTestLogger()

	// Initialize components
	patternDetector := patterns.NewPatternDetector(mockNeo4jClient, logger)
	analytics := analytics.NewGraphAnalytics(mockNeo4jClient, logger)
	entityResolver := resolution.NewEntityResolver(mockNeo4jClient, logger)

	// Initialize handlers
	httpHandlers := handlers.NewHTTPHandlers(mockGraphEngine, cfg, logger)
	enhancedHandlers := handlers.NewEnhancedHTTPHandlers(
		mockGraphEngine,
		patternDetector,
		analytics,
		entityResolver,
		cfg,
		logger,
	)

	// Setup router
	router := mux.NewRouter()
	httpHandlers.RegisterRoutes(router)
	enhancedHandlers.RegisterEnhancedRoutes(router)

	return &IntegrationTestSuite{
		router:           router,
		patternDetector:  patternDetector,
		analytics:        analytics,
		entityResolver:   entityResolver,
		httpHandlers:     httpHandlers,
		enhancedHandlers: enhancedHandlers,
	}
}

func TestGraphEngineIntegration(t *testing.T) {
	suite := setupIntegrationTest(t)

	t.Run("Pattern Detection Endpoints", func(t *testing.T) {
		testPatternDetectionEndpoints(t, suite)
	})

	t.Run("Graph Analytics Endpoints", func(t *testing.T) {
		testGraphAnalyticsEndpoints(t, suite)
	})

	t.Run("Entity Resolution Endpoints", func(t *testing.T) {
		testEntityResolutionEndpoints(t, suite)
	})

	t.Run("Advanced Analysis Endpoints", func(t *testing.T) {
		testAdvancedAnalysisEndpoints(t, suite)
	})

	t.Run("Health and Monitoring Endpoints", func(t *testing.T) {
		testHealthAndMonitoringEndpoints(t, suite)
	})
}

func testPatternDetectionEndpoints(t *testing.T, suite *IntegrationTestSuite) {
	// Test pattern detection
	t.Run("Detect Patterns", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"types":          []string{"smurfing", "layering"},
			"entity_ids":     []string{"entity_1", "entity_2"},
			"min_confidence": 0.7,
			"max_depth":      5,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/patterns/detect", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
	})

	// Test pattern statistics
	t.Run("Get Pattern Statistics", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/patterns/statistics?time_window=24h", nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
	})

	// Test get pattern by ID
	t.Run("Get Pattern by ID", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/patterns/test_pattern_123", nil)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test list patterns
	t.Run("List Patterns", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/patterns?page=1&limit=10", nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "patterns")
		assert.Contains(t, result, "page")
		assert.Contains(t, result, "limit")
	})
}

func testGraphAnalyticsEndpoints(t *testing.T, suite *IntegrationTestSuite) {
	// Test network metrics calculation
	t.Run("Calculate Network Metrics", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entity_types": []string{"Account", "Transaction"},
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analytics/network-metrics", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test community detection
	t.Run("Detect Communities", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entity_ids":         []string{"entity_1", "entity_2"},
			"algorithm":          "louvain",
			"min_community_size": 3,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analytics/communities", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test path analysis
	t.Run("Analyze Paths", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"source_id":  "entity_1",
			"target_id":  "entity_2",
			"max_depth":  5,
			"max_paths":  100,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analytics/paths", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test influence analysis
	t.Run("Analyze Influence", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entity_ids":      []string{"entity_1", "entity_2"},
			"influence_type":  "both",
			"max_depth":       3,
			"decay_factor":    0.85,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analytics/influence", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test centrality metrics for specific entity
	t.Run("Get Centrality Metrics", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/analytics/centrality/entity_123", nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "entity_id")
		assert.Contains(t, result, "degree_centrality")
		assert.Contains(t, result, "betweenness_centrality")
	})
}

func testEntityResolutionEndpoints(t *testing.T, suite *IntegrationTestSuite) {
	// Test entity resolution
	t.Run("Resolve Entities", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entities": []map[string]interface{}{
				{
					"id":   "entity_1",
					"name": "John Doe",
					"type": "Person",
				},
				{
					"id":   "entity_2",
					"name": "J. Doe",
					"type": "Person",
				},
			},
			"resolution_strategy":   "hybrid",
			"similarity_threshold":  0.8,
			"max_candidates":        10,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/resolution/entities", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test relationship inference
	t.Run("Infer Relationships", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entity_ids":         []string{"entity_1", "entity_2"},
			"inference_strategy": "hybrid",
			"min_confidence":     0.7,
			"max_depth":          3,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/resolution/relationships", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	// Test get entity matches
	t.Run("Get Entity Matches", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/resolution/matches/entity_123?threshold=0.8&max_results=10", nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "entity_id")
		assert.Contains(t, result, "matches")
		assert.Contains(t, result, "threshold")
	})
}

func testAdvancedAnalysisEndpoints(t *testing.T, suite *IntegrationTestSuite) {
	// Test risk assessment
	t.Run("Perform Risk Assessment", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entity_ids":   []string{"entity_1", "entity_2"},
			"risk_factors": []string{"high_volume", "unusual_patterns"},
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analysis/risk-assessment", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "request_id")
		assert.Contains(t, result, "overall_risk")
		assert.Contains(t, result, "risk_score")
	})

	// Test anomaly detection
	t.Run("Detect Anomalies", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"entity_ids":     []string{"entity_1", "entity_2"},
			"time_window":    "7d",
			"anomaly_types":  []string{"volume", "frequency"},
			"sensitivity":    0.8,
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analysis/anomaly-detection", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "request_id")
		assert.Contains(t, result, "anomalies_found")
	})

	// Test investigation support
	t.Run("Generate Investigation Support", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"investigation_id": "inv_123",
			"entity_ids":       []string{"entity_1", "entity_2"},
			"focus":            []string{"patterns", "relationships"},
		}

		response := makeRequest(t, suite.router, "POST", "/api/v1/analysis/investigation-support", reqBody)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "investigation_id")
		assert.Contains(t, result, "entity_analysis")
		assert.Contains(t, result, "recommendations")
	})
}

func testHealthAndMonitoringEndpoints(t *testing.T, suite *IntegrationTestSuite) {
	// Test detailed health check
	t.Run("Detailed Health Check", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/health/detailed", nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "status")
		assert.Contains(t, result, "components")
		assert.Equal(t, "healthy", result["status"])
	})

	// Test system metrics
	t.Run("Get System Metrics", func(t *testing.T) {
		response := makeRequest(t, suite.router, "GET", "/api/v1/metrics", nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "performance")
		assert.Contains(t, result, "usage")
		assert.Contains(t, result, "patterns")
		assert.Contains(t, result, "resolution")
	})
}

// Helper function to make HTTP requests in tests
func makeRequest(t *testing.T, router *mux.Router, method, url string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer([]byte{})
	}

	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

// Mock implementations for testing

type mockNeo4jClient struct{}

func (m *mockNeo4jClient) Query(ctx context.Context, cypher string, params map[string]interface{}) (interface{}, error) {
	// Return mock data based on query type
	return map[string]interface{}{
		"nodes": []interface{}{},
		"relationships": []interface{}{},
	}, nil
}

func (m *mockNeo4jClient) Close() error {
	return nil
}

type mockGraphEngine struct{}

func (m *mockGraphEngine) GetDatabase() interface{} {
	return &mockNeo4jClient{}
}

func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// Benchmark tests for performance validation
func BenchmarkPatternDetection(b *testing.B) {
	suite := setupIntegrationTest(&testing.T{})

	reqBody := map[string]interface{}{
		"types":          []string{"smurfing", "layering"},
		"entity_ids":     []string{"entity_1", "entity_2"},
		"min_confidence": 0.7,
		"max_depth":      5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := makeRequest(&testing.T{}, suite.router, "POST", "/api/v1/patterns/detect", reqBody)
		if response.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", response.Code)
		}
	}
}

func BenchmarkNetworkMetrics(b *testing.B) {
	suite := setupIntegrationTest(&testing.T{})

	reqBody := map[string]interface{}{
		"entity_types": []string{"Account", "Transaction"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := makeRequest(&testing.T{}, suite.router, "POST", "/api/v1/analytics/network-metrics", reqBody)
		if response.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", response.Code)
		}
	}
}

func BenchmarkEntityResolution(b *testing.B) {
	suite := setupIntegrationTest(&testing.T{})

	reqBody := map[string]interface{}{
		"entities": []map[string]interface{}{
			{
				"id":   "entity_1",
				"name": "John Doe",
				"type": "Person",
			},
		},
		"resolution_strategy":  "hybrid",
		"similarity_threshold": 0.8,
		"max_candidates":       10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := makeRequest(&testing.T{}, suite.router, "POST", "/api/v1/resolution/entities", reqBody)
		if response.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", response.Code)
		}
	}
}

// Performance test to ensure endpoints respond within acceptable time limits
func TestEndpointPerformance(t *testing.T) {
	suite := setupIntegrationTest(t)

	endpoints := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		maxTime  time.Duration
	}{
		{
			name:    "Pattern Detection",
			method:  "POST",
			path:    "/api/v1/patterns/detect",
			body:    map[string]interface{}{"types": []string{"smurfing"}, "entity_ids": []string{"entity_1"}},
			maxTime: 500 * time.Millisecond,
		},
		{
			name:    "Network Metrics",
			method:  "POST",
			path:    "/api/v1/analytics/network-metrics",
			body:    map[string]interface{}{"entity_types": []string{"Account"}},
			maxTime: 300 * time.Millisecond,
		},
		{
			name:    "Entity Resolution",
			method:  "POST",
			path:    "/api/v1/resolution/entities",
			body:    map[string]interface{}{"entities": []map[string]interface{}{{"id": "1", "name": "Test"}}},
			maxTime: 400 * time.Millisecond,
		},
		{
			name:    "Health Check",
			method:  "GET",
			path:    "/api/v1/health/detailed",
			body:    nil,
			maxTime: 100 * time.Millisecond,
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			start := time.Now()
			response := makeRequest(t, suite.router, endpoint.method, endpoint.path, endpoint.body)
			duration := time.Since(start)

			assert.Equal(t, http.StatusOK, response.Code)
			assert.True(t, duration < endpoint.maxTime,
				"Endpoint %s took %v, expected less than %v", endpoint.name, duration, endpoint.maxTime)
		})
	}
}