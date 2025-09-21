//go:build integration
// +build integration

package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIGateway_Investigation_Endpoints tests the API Gateway investigation contract
func TestAPIGateway_Investigation_Endpoints(t *testing.T) {
	// These tests MUST FAIL initially (TDD principle)
	// They define the expected contract for investigation endpoints

	baseURL := "http://localhost:8080/api/v1"
	client := &http.Client{Timeout: 10 * time.Second}

	// Setup authentication token for tests
	authToken := "Bearer test-jwt-token"

	t.Run("GET_Investigations_ReturnsInvestigationList", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseURL+"/investigations", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var investigations []Investigation
		err = json.NewDecoder(resp.Body).Decode(&investigations)
		assert.NoError(t, err)
		assert.NotNil(t, investigations)
	})

	t.Run("GET_Investigation_ById_ReturnsInvestigation", func(t *testing.T) {
		investigationID := "550e8400-e29b-41d4-a716-446655440001"
		req, err := http.NewRequest("GET", baseURL+"/investigations/"+investigationID, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var investigation Investigation
		err = json.NewDecoder(resp.Body).Decode(&investigation)
		assert.NoError(t, err)
		assert.Equal(t, investigationID, investigation.ID)
		assert.NotEmpty(t, investigation.Title)
		assert.NotEmpty(t, investigation.Status)
	})

	t.Run("POST_Investigation_CreatesNewInvestigation", func(t *testing.T) {
		newInvestigation := CreateInvestigationRequest{
			Title:       "Suspicious Wire Transfer Investigation",
			Description: "Large wire transfers detected between related entities",
			Priority:    "HIGH",
			AssignedTo:  "analyst@aegisshield.com",
			AlertIDs:    []string{"alert_001", "alert_002"},
		}

		jsonData, err := json.Marshal(newInvestigation)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", baseURL+"/investigations", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created Investigation
		err = json.NewDecoder(resp.Body).Decode(&created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, newInvestigation.Title, created.Title)
		assert.Equal(t, "OPEN", created.Status)
	})

	t.Run("PUT_Investigation_UpdatesExistingInvestigation", func(t *testing.T) {
		investigationID := "550e8400-e29b-41d4-a716-446655440001"
		update := UpdateInvestigationRequest{
			Title:       "Updated Investigation Title",
			Description: "Updated description with new findings",
			Status:      "IN_PROGRESS",
			Priority:    "CRITICAL",
		}

		jsonData, err := json.Marshal(update)
		require.NoError(t, err)

		req, err := http.NewRequest("PUT", baseURL+"/investigations/"+investigationID, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updated Investigation
		err = json.NewDecoder(resp.Body).Decode(&updated)
		assert.NoError(t, err)
		assert.Equal(t, update.Title, updated.Title)
		assert.Equal(t, update.Status, updated.Status)
	})

	t.Run("GET_Investigation_Entities_ReturnsRelatedEntities", func(t *testing.T) {
		investigationID := "550e8400-e29b-41d4-a716-446655440001"
		req, err := http.NewRequest("GET", baseURL+"/investigations/"+investigationID+"/entities", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var entities []Entity
		err = json.NewDecoder(resp.Body).Decode(&entities)
		assert.NoError(t, err)
		assert.NotNil(t, entities)
	})

	t.Run("GET_Investigation_Timeline_ReturnsEventTimeline", func(t *testing.T) {
		investigationID := "550e8400-e29b-41d4-a716-446655440001"
		req, err := http.NewRequest("GET", baseURL+"/investigations/"+investigationID+"/timeline", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var timeline []TimelineEvent
		err = json.NewDecoder(resp.Body).Decode(&timeline)
		assert.NoError(t, err)
		assert.NotNil(t, timeline)
	})

	t.Run("POST_Investigation_AddNote_AddsInvestigationNote", func(t *testing.T) {
		investigationID := "550e8400-e29b-41d4-a716-446655440001"
		note := AddNoteRequest{
			Content:  "Found additional suspicious transactions",
			Category: "FINDING",
			AuthorID: "analyst@aegisshield.com",
		}

		jsonData, err := json.Marshal(note)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", baseURL+"/investigations/"+investigationID+"/notes", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createdNote Note
		err = json.NewDecoder(resp.Body).Decode(&createdNote)
		assert.NoError(t, err)
		assert.NotEmpty(t, createdNote.ID)
		assert.Equal(t, note.Content, createdNote.Content)
	})

	t.Run("DELETE_Investigation_ArchivesInvestigation", func(t *testing.T) {
		investigationID := "550e8400-e29b-41d4-a716-446655440002"
		req, err := http.NewRequest("DELETE", baseURL+"/investigations/"+investigationID, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result ArchiveResult
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.True(t, result.Archived)
		assert.NotEmpty(t, result.ArchivedAt)
	})
}

// Test data structures - these define the expected API contracts
type Investigation struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	AssignedTo  string     `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

type CreateInvestigationRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	AssignedTo  string   `json:"assigned_to"`
	AlertIDs    []string `json:"alert_ids"`
}

type UpdateInvestigationRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssignedTo  string `json:"assigned_to,omitempty"`
}

type Entity struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	RiskScore float64           `json:"risk_score"`
	Metadata  map[string]string `json:"metadata"`
}

type TimelineEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	EntityID    string                 `json:"entity_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type AddNoteRequest struct {
	Content  string `json:"content"`
	Category string `json:"category"`
	AuthorID string `json:"author_id"`
}

type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ArchiveResult struct {
	Archived   bool   `json:"archived"`
	ArchivedAt string `json:"archived_at"`
}
