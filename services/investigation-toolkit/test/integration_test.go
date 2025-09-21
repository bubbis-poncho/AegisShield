package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"investigation-toolkit/internal/config"
	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/models"
	"investigation-toolkit/internal/server"
)

// IntegrationTestSuite provides comprehensive testing for the Investigation Toolkit service
type IntegrationTestSuite struct {
	suite.Suite
	cfg        *config.Config
	logger     *zap.Logger
	db         *database.Database
	server     *server.Server
	router     *gin.Engine
	container  testcontainers.Container
	testCtx    context.Context
	cancel     context.CancelFunc
	baseURL    string
}

// SetupSuite initializes the test environment
func (suite *IntegrationTestSuite) SetupSuite() {
	suite.testCtx, suite.cancel = context.WithCancel(context.Background())

	// Initialize logger
	var err error
	suite.logger, err = zap.NewDevelopment()
	require.NoError(suite.T(), err)

	// Setup test database
	suite.setupTestDatabase()

	// Initialize configuration
	suite.cfg = &config.Config{
		Environment: "test",
		Debug:       true,
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "investigation_toolkit_test",
			User:     "postgres",
			Password: "testpass",
		},
		Server: config.ServerConfig{
			HTTPPort:       8080,
			GRPCPort:       9090,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}

	// Initialize database
	suite.db, err = database.New(suite.cfg, suite.logger)
	require.NoError(suite.T(), err)

	// Run migrations
	err = suite.db.Migrate()
	require.NoError(suite.T(), err)

	// Initialize server
	suite.server = server.New(suite.cfg, suite.logger, suite.db)
	err = suite.server.Initialize()
	require.NoError(suite.T(), err)

	// Set base URL for API calls
	suite.baseURL = "/api/v1"

	suite.logger.Info("Integration test suite setup completed")
}

// TearDownSuite cleans up the test environment
func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.container != nil {
		suite.container.Terminate(suite.testCtx)
	}
	suite.cancel()
	suite.logger.Info("Integration test suite teardown completed")
}

// SetupTest prepares each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Clean database tables
	suite.cleanDatabase()
}

// setupTestDatabase creates a test PostgreSQL container
func (suite *IntegrationTestSuite) setupTestDatabase() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "investigation_toolkit_test",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	var err error
	suite.container, err = testcontainers.GenericContainer(suite.testCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(suite.T(), err)

	// Get the actual port
	port, err := suite.container.MappedPort(suite.testCtx, "5432")
	require.NoError(suite.T(), err)

	// Update configuration with actual port
	os.Setenv("DB_PORT", port.Port())
}

// cleanDatabase removes all test data
func (suite *IntegrationTestSuite) cleanDatabase() {
	tables := []string{
		"audit_logs",
		"collaboration_notifications",
		"collaboration_activities", 
		"collaboration_team_members",
		"collaboration_teams",
		"collaboration_assignments",
		"collaboration_comments",
		"workflow_instance_steps",
		"workflow_instances",
		"workflow_templates",
		"timeline_events",
		"evidence",
		"investigations",
	}

	for _, table := range tables {
		_, err := suite.db.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(suite.T(), err)
	}
}

// makeRequest creates and executes HTTP requests
func (suite *IntegrationTestSuite) makeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(suite.T(), err)
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, suite.baseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "test-user-123")

	recorder := httptest.NewRecorder()
	suite.server.(*server.Server).ServeHTTP(recorder, req)
	return recorder
}

// TestInvestigationWorkflow tests the complete investigation lifecycle
func (suite *IntegrationTestSuite) TestInvestigationWorkflow() {
	suite.T().Log("Testing complete investigation workflow")

	// 1. Create investigation
	investigation := models.CreateInvestigationRequest{
		Title:       "Money Laundering Investigation",
		Description: "Suspicious financial transactions",
		Type:        "AML",
		Priority:    "high",
		AssigneeID:  "analyst-001",
		Tags:        []string{"money-laundering", "suspicious-activity"},
	}

	resp := suite.makeRequest("POST", "/investigations", investigation)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInvestigation models.Investigation
	err := json.Unmarshal(resp.Body.Bytes(), &createdInvestigation)
	require.NoError(suite.T(), err)

	investigationID := createdInvestigation.ID
	suite.T().Logf("Created investigation: %s", investigationID)

	// 2. Add evidence
	evidence := models.CreateEvidenceRequest{
		InvestigationID: investigationID,
		Type:            "financial_record",
		Description:     "Bank transaction records",
		Source:          "Bank XYZ",
		CollectedBy:     "analyst-001",
		Hash:            "abc123def456",
		Metadata:        map[string]interface{}{"amount": 50000, "currency": "USD"},
	}

	resp = suite.makeRequest("POST", "/evidence", evidence)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdEvidence models.Evidence
	err = json.Unmarshal(resp.Body.Bytes(), &createdEvidence)
	require.NoError(suite.T(), err)

	evidenceID := createdEvidence.ID
	suite.T().Logf("Created evidence: %s", evidenceID)

	// 3. Create timeline events
	timelineEvent := models.CreateTimelineEventRequest{
		InvestigationID: investigationID,
		EventType:       "transaction",
		Title:           "Large Cash Deposit",
		Description:     "Unusual cash deposit of $50,000",
		EventTime:       time.Now().Add(-24 * time.Hour),
		Location:        "Branch Office Downtown",
		Entities:        []string{"John Doe", "Bank XYZ"},
		Metadata:        map[string]interface{}{"amount": 50000, "account": "12345"},
	}

	resp = suite.makeRequest("POST", "/timeline", timelineEvent)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	// 4. Create workflow template
	workflowTemplate := models.CreateWorkflowTemplateRequest{
		Name:        "AML Investigation Workflow",
		Description: "Standard workflow for AML investigations",
		Category:    "AML",
		Version:     "1.0",
		Steps: []models.WorkflowTemplateStep{
			{
				Name:        "Initial Assessment",
				Description: "Assess the suspicious activity",
				Type:        "manual",
				Order:       1,
				Required:    true,
				EstimatedDuration: 2 * time.Hour,
			},
			{
				Name:        "Evidence Collection",
				Description: "Collect supporting evidence",
				Type:        "manual", 
				Order:       2,
				Required:    true,
				EstimatedDuration: 4 * time.Hour,
			},
		},
	}

	resp = suite.makeRequest("POST", "/workflows/templates", workflowTemplate)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdTemplate models.WorkflowTemplate
	err = json.Unmarshal(resp.Body.Bytes(), &createdTemplate)
	require.NoError(suite.T(), err)

	templateID := createdTemplate.ID
	suite.T().Logf("Created workflow template: %s", templateID)

	// 5. Create workflow instance
	workflowInstance := models.CreateWorkflowInstanceRequest{
		TemplateID:      templateID,
		InvestigationID: investigationID,
		AssigneeID:      "analyst-001",
		Priority:        "high",
		DueDate:         time.Now().Add(7 * 24 * time.Hour),
	}

	resp = suite.makeRequest("POST", "/workflows/instances", workflowInstance)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInstance models.WorkflowInstance
	err = json.Unmarshal(resp.Body.Bytes(), &createdInstance)
	require.NoError(suite.T(), err)

	instanceID := createdInstance.ID
	suite.T().Logf("Created workflow instance: %s", instanceID)

	// 6. Add comments and collaboration
	comment := models.CreateCommentRequest{
		EntityType: "investigation",
		EntityID:   investigationID,
		Content:    "Initial review shows suspicious patterns",
		UserID:     "analyst-001",
		IsInternal: true,
	}

	resp = suite.makeRequest("POST", "/collaboration/comments", comment)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	// 7. Update investigation status
	statusUpdate := models.UpdateStatusRequest{
		Status: "in_progress",
		Notes:  "Investigation actively being pursued",
	}

	resp = suite.makeRequest("PUT", fmt.Sprintf("/investigations/%s/status", investigationID), statusUpdate)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// 8. Verify audit trail
	resp = suite.makeRequest("GET", fmt.Sprintf("/audit/logs/investigation/%s", investigationID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var auditLogs []models.AuditLog
	err = json.Unmarshal(resp.Body.Bytes(), &auditLogs)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), auditLogs)

	suite.T().Log("Investigation workflow test completed successfully")
}

// TestWorkflowAutomation tests workflow step execution
func (suite *IntegrationTestSuite) TestWorkflowAutomation() {
	suite.T().Log("Testing workflow automation")

	// Create template with multiple steps
	template := models.CreateWorkflowTemplateRequest{
		Name:        "Evidence Processing Workflow",
		Description: "Automated evidence processing",
		Category:    "Evidence",
		Version:     "1.0",
		Steps: []models.WorkflowTemplateStep{
			{
				Name:        "Intake",
				Description: "Evidence intake and cataloging",
				Type:        "manual",
				Order:       1,
				Required:    true,
			},
			{
				Name:        "Analysis",
				Description: "Technical analysis of evidence",
				Type:        "automated",
				Order:       2,
				Required:    true,
			},
			{
				Name:        "Review",
				Description: "Expert review of findings",
				Type:        "manual",
				Order:       3,
				Required:    true,
			},
		},
	}

	resp := suite.makeRequest("POST", "/workflows/templates", template)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdTemplate models.WorkflowTemplate
	err := json.Unmarshal(resp.Body.Bytes(), &createdTemplate)
	require.NoError(suite.T(), err)

	// Create investigation for workflow
	investigation := models.CreateInvestigationRequest{
		Title:       "Digital Evidence Case",
		Description: "Analysis of digital evidence",
		Type:        "Digital Forensics",
		Priority:    "medium",
		AssigneeID:  "forensics-expert-001",
	}

	resp = suite.makeRequest("POST", "/investigations", investigation)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInvestigation models.Investigation
	err = json.Unmarshal(resp.Body.Bytes(), &createdInvestigation)
	require.NoError(suite.T(), err)

	// Create workflow instance
	instance := models.CreateWorkflowInstanceRequest{
		TemplateID:      createdTemplate.ID,
		InvestigationID: createdInvestigation.ID,
		AssigneeID:      "forensics-expert-001",
		Priority:        "medium",
	}

	resp = suite.makeRequest("POST", "/workflows/instances", instance)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInstance models.WorkflowInstance
	err = json.Unmarshal(resp.Body.Bytes(), &createdInstance)
	require.NoError(suite.T(), err)

	// Get workflow steps
	resp = suite.makeRequest("GET", fmt.Sprintf("/workflows/instances/%s/steps", createdInstance.ID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var steps []models.WorkflowInstanceStep
	err = json.Unmarshal(resp.Body.Bytes(), &steps)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), steps, 3)

	// Start first step
	firstStepID := steps[0].ID
	resp = suite.makeRequest("PUT", fmt.Sprintf("/workflows/steps/%s/start", firstStepID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Complete first step
	completion := models.CompleteStepRequest{
		Notes:  "Evidence intake completed successfully",
		Output: map[string]interface{}{"items_collected": 5},
	}

	resp = suite.makeRequest("PUT", fmt.Sprintf("/workflows/steps/%s/complete", firstStepID), completion)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Verify next step is automatically started (if configured)
	resp = suite.makeRequest("GET", fmt.Sprintf("/workflows/instances/%s/steps", createdInstance.ID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	err = json.Unmarshal(resp.Body.Bytes(), &steps)
	require.NoError(suite.T(), err)

	// Check step statuses
	assert.Equal(suite.T(), "completed", steps[0].Status)

	suite.T().Log("Workflow automation test completed successfully")
}

// TestCollaborationFeatures tests team collaboration functionality
func (suite *IntegrationTestSuite) TestCollaborationFeatures() {
	suite.T().Log("Testing collaboration features")

	// Create team
	team := models.CreateTeamRequest{
		Name:        "Financial Crimes Unit",
		Description: "Specialized team for financial crime investigations",
		LeaderID:    "supervisor-001",
	}

	resp := suite.makeRequest("POST", "/collaboration/teams", team)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdTeam models.Team
	err := json.Unmarshal(resp.Body.Bytes(), &createdTeam)
	require.NoError(suite.T(), err)

	teamID := createdTeam.ID

	// Add team members
	member := models.AddTeamMemberRequest{
		UserID: "analyst-001",
		Role:   "analyst",
	}

	resp = suite.makeRequest("POST", fmt.Sprintf("/collaboration/teams/%s/members", teamID), member)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	// Create investigation for team collaboration
	investigation := models.CreateInvestigationRequest{
		Title:       "Team Investigation",
		Description: "Multi-analyst investigation",
		Type:        "Complex Financial Crime",
		Priority:    "high",
		AssigneeID:  "supervisor-001",
	}

	resp = suite.makeRequest("POST", "/investigations", investigation)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInvestigation models.Investigation
	err = json.Unmarshal(resp.Body.Bytes(), &createdInvestigation)
	require.NoError(suite.T(), err)

	investigationID := createdInvestigation.ID

	// Create assignment
	assignment := models.CreateAssignmentRequest{
		EntityType:  "investigation",
		EntityID:    investigationID,
		AssigneeID:  "analyst-001",
		AssignedBy:  "supervisor-001",
		Role:        "lead_analyst",
		Description: "Lead analyst for investigation",
		DueDate:     time.Now().Add(7 * 24 * time.Hour),
	}

	resp = suite.makeRequest("POST", "/collaboration/assignments", assignment)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	// Add comments with mentions
	comment := models.CreateCommentRequest{
		EntityType: "investigation",
		EntityID:   investigationID,
		Content:    "Found interesting patterns @analyst-001 please review",
		UserID:     "supervisor-001",
		IsInternal: true,
	}

	resp = suite.makeRequest("POST", "/collaboration/comments", comment)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	// Create notification
	notification := models.CreateNotificationRequest{
		UserID:      "analyst-001",
		Type:        "mention",
		Title:       "You were mentioned in a comment",
		Message:     "Supervisor mentioned you in investigation comment",
		EntityType:  "investigation",
		EntityID:    investigationID,
		Priority:    "medium",
	}

	resp = suite.makeRequest("POST", "/collaboration/notifications", notification)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	// Get user notifications
	resp = suite.makeRequest("GET", "/collaboration/notifications/user/analyst-001", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var notifications []models.Notification
	err = json.Unmarshal(resp.Body.Bytes(), &notifications)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), notifications)

	// Get collaboration statistics
	resp = suite.makeRequest("GET", "/collaboration/stats", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	suite.T().Log("Collaboration features test completed successfully")
}

// TestAuditAndCompliance tests audit trail and compliance features
func (suite *IntegrationTestSuite) TestAuditAndCompliance() {
	suite.T().Log("Testing audit and compliance features")

	// Create investigation to generate audit trail
	investigation := models.CreateInvestigationRequest{
		Title:       "Audit Test Investigation",
		Description: "Testing audit functionality",
		Type:        "Compliance Test",
		Priority:    "medium",
		AssigneeID:  "compliance-officer-001",
	}

	resp := suite.makeRequest("POST", "/investigations", investigation)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInvestigation models.Investigation
	err := json.Unmarshal(resp.Body.Bytes(), &createdInvestigation)
	require.NoError(suite.T(), err)

	investigationID := createdInvestigation.ID

	// Add evidence to create chain of custody
	evidence := models.CreateEvidenceRequest{
		InvestigationID: investigationID,
		Type:            "document",
		Description:     "Compliance document",
		Source:          "Internal Audit",
		CollectedBy:     "compliance-officer-001",
		Hash:            "compliance123",
	}

	resp = suite.makeRequest("POST", "/evidence", evidence)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdEvidence models.Evidence
	err = json.Unmarshal(resp.Body.Bytes(), &createdEvidence)
	require.NoError(suite.T(), err)

	evidenceID := createdEvidence.ID

	// Verify chain of custody
	resp = suite.makeRequest("GET", fmt.Sprintf("/audit/custody/evidence/%s", evidenceID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Get audit logs for investigation
	resp = suite.makeRequest("GET", fmt.Sprintf("/audit/logs/investigation/%s", investigationID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var auditLogs []models.AuditLog
	err = json.Unmarshal(resp.Body.Bytes(), &auditLogs)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), auditLogs)

	// Generate compliance report
	resp = suite.makeRequest("GET", "/audit/reports/compliance", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Get audit summary
	resp = suite.makeRequest("GET", "/audit/reports/summary", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var auditSummary models.AuditSummary
	err = json.Unmarshal(resp.Body.Bytes(), &auditSummary)
	require.NoError(suite.T(), err)
	assert.NotZero(suite.T(), auditSummary.TotalLogs)

	// Check data integrity
	resp = suite.makeRequest("POST", fmt.Sprintf("/audit/integrity/evidence/%s/verify", evidenceID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	suite.T().Log("Audit and compliance test completed successfully")
}

// TestHealthAndMonitoring tests health check endpoints
func (suite *IntegrationTestSuite) TestHealthAndMonitoring() {
	suite.T().Log("Testing health and monitoring endpoints")

	// Test health endpoint
	resp := suite.makeRequest("GET", "/health", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Test readiness endpoint
	resp = suite.makeRequest("GET", "/health/ready", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Test liveness endpoint
	resp = suite.makeRequest("GET", "/health/live", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	suite.T().Log("Health and monitoring test completed successfully")
}

// TestErrorHandling tests error scenarios and validation
func (suite *IntegrationTestSuite) TestErrorHandling() {
	suite.T().Log("Testing error handling and validation")

	// Test invalid investigation creation
	invalidInvestigation := models.CreateInvestigationRequest{
		Title: "", // Missing required field
	}

	resp := suite.makeRequest("POST", "/investigations", invalidInvestigation)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

	// Test non-existent resource
	resp = suite.makeRequest("GET", "/investigations/nonexistent-id", nil)
	assert.Equal(suite.T(), http.StatusNotFound, resp.Code)

	// Test invalid evidence creation
	invalidEvidence := models.CreateEvidenceRequest{
		InvestigationID: "nonexistent-investigation",
		Type:            "document",
	}

	resp = suite.makeRequest("POST", "/evidence", invalidEvidence)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

	suite.T().Log("Error handling test completed successfully")
}

// TestPerformance tests basic performance scenarios
func (suite *IntegrationTestSuite) TestPerformance() {
	suite.T().Log("Testing basic performance scenarios")

	// Create multiple investigations to test pagination
	for i := 0; i < 25; i++ {
		investigation := models.CreateInvestigationRequest{
			Title:       fmt.Sprintf("Performance Test Investigation %d", i),
			Description: fmt.Sprintf("Performance testing investigation number %d", i),
			Type:        "Performance Test",
			Priority:    "low",
			AssigneeID:  "performance-tester-001",
		}

		resp := suite.makeRequest("POST", "/investigations", investigation)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
	}

	// Test pagination
	resp := suite.makeRequest("GET", "/investigations?page=1&limit=10", nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var investigations []models.Investigation
	err := json.Unmarshal(resp.Body.Bytes(), &investigations)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), investigations, 10)

	// Test large batch operations
	events := make([]models.CreateTimelineEventRequest, 10)
	for i := 0; i < 10; i++ {
		events[i] = models.CreateTimelineEventRequest{
			EventType:   "batch_test",
			Title:       fmt.Sprintf("Batch Event %d", i),
			Description: fmt.Sprintf("Batch testing event %d", i),
			EventTime:   time.Now().Add(time.Duration(i) * time.Minute),
		}
	}

	resp = suite.makeRequest("POST", "/timeline/bulk", events)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	suite.T().Log("Performance test completed successfully")
}

// TestConcurrency tests concurrent operations
func (suite *IntegrationTestSuite) TestConcurrency() {
	suite.T().Log("Testing concurrent operations")

	// Create investigation for concurrency testing
	investigation := models.CreateInvestigationRequest{
		Title:       "Concurrency Test Investigation",
		Description: "Testing concurrent operations",
		Type:        "Concurrency Test",
		Priority:    "medium",
		AssigneeID:  "concurrency-tester-001",
	}

	resp := suite.makeRequest("POST", "/investigations", investigation)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var createdInvestigation models.Investigation
	err := json.Unmarshal(resp.Body.Bytes(), &createdInvestigation)
	require.NoError(suite.T(), err)

	investigationID := createdInvestigation.ID

	// Simulate concurrent comments
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(commentNum int) {
			comment := models.CreateCommentRequest{
				EntityType: "investigation",
				EntityID:   investigationID,
				Content:    fmt.Sprintf("Concurrent comment %d", commentNum),
				UserID:     fmt.Sprintf("user-%d", commentNum),
				IsInternal: true,
			}

			resp := suite.makeRequest("POST", "/collaboration/comments", comment)
			assert.Equal(suite.T(), http.StatusCreated, resp.Code)
			done <- true
		}(i)
	}

	// Wait for all concurrent operations to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all comments were created
	resp = suite.makeRequest("GET", fmt.Sprintf("/collaboration/comments/investigation/%s", investigationID), nil)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var comments []models.Comment
	err = json.Unmarshal(resp.Body.Bytes(), &comments)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), comments, 5)

	suite.T().Log("Concurrency test completed successfully")
}

// TestInvestigationToolkitIntegrationSuite runs the complete test suite
func TestInvestigationToolkitIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(IntegrationTestSuite))
}

// BenchmarkInvestigationCreation benchmarks investigation creation
func BenchmarkInvestigationCreation(b *testing.B) {
	// Setup test environment (simplified)
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		Environment: "test",
		Database: config.DatabaseConfig{
			Host: "localhost",
			Port: 5432,
			Name: "investigation_toolkit_bench",
		},
	}

	db, err := database.New(cfg, logger)
	if err != nil {
		b.Skip("Database not available for benchmarking")
	}
	defer db.Close()

	server := server.New(cfg, logger, db)
	server.Initialize()

	investigation := models.CreateInvestigationRequest{
		Title:       "Benchmark Investigation",
		Description: "Performance testing",
		Type:        "Benchmark",
		Priority:    "low",
		AssigneeID:  "benchmark-user",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bodyBytes, _ := json.Marshal(investigation)
			req := httptest.NewRequest("POST", "/api/v1/investigations", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", "benchmark-user")

			recorder := httptest.NewRecorder()
			server.(*server.Server).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusCreated {
				b.Errorf("Expected status 201, got %d", recorder.Code)
			}
		}
	})
}