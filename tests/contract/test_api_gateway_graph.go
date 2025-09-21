//go:build integration
// +build integration

package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIGateway_GraphExploration_Endpoints tests the graph exploration contract
func TestAPIGateway_GraphExploration_Endpoints(t *testing.T) {
	// These tests MUST FAIL initially (TDD principle)
	// They define the expected contract for graph exploration endpoints

	baseURL := "http://localhost:8080/api/v1"
	client := &http.Client{Timeout: 10 * time.Second}
	authToken := "Bearer test-jwt-token"

	t.Run("POST_Graph_Explore_ReturnsSubgraph", func(t *testing.T) {
		exploreRequest := GraphExploreRequest{
			CenterEntityID:    "person_12345",
			MaxDepth:          2,
			MaxNodes:          100,
			RelationshipTypes: []string{"FAMILY", "BUSINESS", "TRANSACTION"},
			MinConfidence:     0.7,
		}

		jsonData, err := json.Marshal(exploreRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", baseURL+"/graph/explore", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var graph GraphResponse
		err = json.NewDecoder(resp.Body).Decode(&graph)
		assert.NoError(t, err)
		assert.NotEmpty(t, graph.Nodes)
		assert.NotEmpty(t, graph.Edges)
		assert.Equal(t, exploreRequest.CenterEntityID, graph.CenterNode.ID)
	})

	t.Run("GET_Graph_ShortestPath_ReturnsPath", func(t *testing.T) {
		sourceID := "person_12345"
		targetID := "org_67890"

		req, err := http.NewRequest("GET", baseURL+"/graph/path?source="+sourceID+"&target="+targetID+"&max_depth=5", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var pathResponse PathResponse
		err = json.NewDecoder(resp.Body).Decode(&pathResponse)
		assert.NoError(t, err)
		assert.True(t, len(pathResponse.Path) >= 2) // At least source and target
		assert.Equal(t, sourceID, pathResponse.Path[0].ID)
		assert.Equal(t, targetID, pathResponse.Path[len(pathResponse.Path)-1].ID)
	})

	t.Run("POST_Graph_PatternMatch_ReturnsMatches", func(t *testing.T) {
		patternRequest := PatternMatchRequest{
			Pattern: GraphPattern{
				Nodes: []PatternNode{
					{
						ID:   "p1",
						Type: "PERSON",
						Properties: map[string]interface{}{
							"risk_score": map[string]interface{}{
								"$gte": 0.8,
							},
						},
					},
					{
						ID:   "o1",
						Type: "ORGANIZATION",
					},
				},
				Edges: []PatternEdge{
					{
						Source: "p1",
						Target: "o1",
						Type:   "CONTROLS",
					},
				},
			},
			MaxResults: 50,
		}

		jsonData, err := json.Marshal(patternRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", baseURL+"/graph/pattern-match", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var matches PatternMatchResponse
		err = json.NewDecoder(resp.Body).Decode(&matches)
		assert.NoError(t, err)
		assert.NotNil(t, matches.Matches)
		assert.True(t, len(matches.Matches) <= 50)
	})

	t.Run("GET_Graph_Centrality_ReturnsInfluentialNodes", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseURL+"/graph/centrality?algorithm=betweenness&limit=20", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var centrality CentralityResponse
		err = json.NewDecoder(resp.Body).Decode(&centrality)
		assert.NoError(t, err)
		assert.True(t, len(centrality.Nodes) <= 20)

		// Check nodes are sorted by centrality score (descending)
		for i := 1; i < len(centrality.Nodes); i++ {
			assert.GreaterOrEqual(t, centrality.Nodes[i-1].Score, centrality.Nodes[i].Score)
		}
	})

	t.Run("POST_Graph_Community_ReturnsCommunityClusters", func(t *testing.T) {
		communityRequest := CommunityDetectionRequest{
			Algorithm:   "louvain",
			MinSize:     3,
			MaxClusters: 10,
			EntityTypes: []string{"PERSON", "ORGANIZATION"},
		}

		jsonData, err := json.Marshal(communityRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", baseURL+"/graph/community", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Authorization", authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var communities CommunityResponse
		err = json.NewDecoder(resp.Body).Decode(&communities)
		assert.NoError(t, err)
		assert.NotNil(t, communities.Communities)
		assert.True(t, len(communities.Communities) <= 10)

		// Check each community has minimum size
		for _, community := range communities.Communities {
			assert.GreaterOrEqual(t, len(community.Members), 3)
		}
	})
}

// Test data structures for graph exploration
type GraphExploreRequest struct {
	CenterEntityID    string   `json:"center_entity_id"`
	MaxDepth          int      `json:"max_depth"`
	MaxNodes          int      `json:"max_nodes"`
	RelationshipTypes []string `json:"relationship_types"`
	MinConfidence     float64  `json:"min_confidence"`
}

type GraphResponse struct {
	CenterNode GraphNode   `json:"center_node"`
	Nodes      []GraphNode `json:"nodes"`
	Edges      []GraphEdge `json:"edges"`
	Metadata   GraphMeta   `json:"metadata"`
}

type GraphNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
	RiskScore  float64                `json:"risk_score"`
}

type GraphEdge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Confidence float64                `json:"confidence"`
	Properties map[string]interface{} `json:"properties"`
}

type GraphMeta struct {
	NodeCount int     `json:"node_count"`
	EdgeCount int     `json:"edge_count"`
	QueryTime float64 `json:"query_time_ms"`
}

type PathResponse struct {
	Path      []GraphNode `json:"path"`
	Distance  int         `json:"distance"`
	TotalCost float64     `json:"total_cost"`
}

type PatternMatchRequest struct {
	Pattern     GraphPattern `json:"pattern"`
	MaxResults  int          `json:"max_results"`
	Constraints []Constraint `json:"constraints,omitempty"`
}

type GraphPattern struct {
	Nodes []PatternNode `json:"nodes"`
	Edges []PatternEdge `json:"edges"`
}

type PatternNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type PatternEdge struct {
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type Constraint struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type PatternMatchResponse struct {
	Matches   []PatternMatch `json:"matches"`
	Count     int            `json:"count"`
	QueryTime float64        `json:"query_time_ms"`
}

type PatternMatch struct {
	Score    float64              `json:"score"`
	Bindings map[string]GraphNode `json:"bindings"`
	SubGraph GraphResponse        `json:"subgraph"`
}

type CentralityResponse struct {
	Algorithm string           `json:"algorithm"`
	Nodes     []CentralityNode `json:"nodes"`
}

type CentralityNode struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
	Rank  int     `json:"rank"`
}

type CommunityDetectionRequest struct {
	Algorithm   string   `json:"algorithm"`
	MinSize     int      `json:"min_size"`
	MaxClusters int      `json:"max_clusters"`
	EntityTypes []string `json:"entity_types"`
}

type CommunityResponse struct {
	Algorithm   string      `json:"algorithm"`
	Communities []Community `json:"communities"`
	Modularity  float64     `json:"modularity"`
}

type Community struct {
	ID      string      `json:"id"`
	Members []GraphNode `json:"members"`
	Score   float64     `json:"score"`
}
