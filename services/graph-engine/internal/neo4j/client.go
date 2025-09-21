package neo4j

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps Neo4j driver for graph analysis operations
type Client struct {
	driver neo4j.DriverWithContext
	logger *slog.Logger
	config config.Neo4jConfig
}

// Entity represents an entity node in the graph
type Entity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// Relationship represents a relationship edge in the graph
type Relationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	Properties map[string]interface{} `json:"properties"`
}

// Path represents a path between entities
type Path struct {
	StartEntity   *Entity         `json:"start_entity"`
	EndEntity     *Entity         `json:"end_entity"`
	Relationships []*Relationship `json:"relationships"`
	Entities      []*Entity       `json:"entities"`
	Length        int             `json:"length"`
	Cost          float64         `json:"cost"`
}

// SubGraph represents a subgraph containing entities and relationships
type SubGraph struct {
	Entities      []*Entity       `json:"entities"`
	Relationships []*Relationship `json:"relationships"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// CentralityMetrics represents centrality calculations for an entity
type CentralityMetrics struct {
	EntityID              string  `json:"entity_id"`
	DegreeCentrality      float64 `json:"degree_centrality"`
	BetweennessCentrality float64 `json:"betweenness_centrality"`
	ClosenessCentrality   float64 `json:"closeness_centrality"`
	EigenvectorCentrality float64 `json:"eigenvector_centrality"`
	PageRank              float64 `json:"page_rank"`
}

// Community represents a detected community/cluster
type Community struct {
	ID        string   `json:"id"`
	Entities  []string `json:"entities"`
	Size      int      `json:"size"`
	Density   float64  `json:"density"`
	Modularity float64 `json:"modularity"`
}

// PatternMatch represents a detected pattern in the graph
type PatternMatch struct {
	PatternType string                 `json:"pattern_type"`
	Entities    []*Entity              `json:"entities"`
	Relationships []*Relationship      `json:"relationships"`
	Confidence  float64                `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewClient creates a new Neo4j client
func NewClient(cfg config.Neo4jConfig, logger *slog.Logger) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
		func(config *neo4j.Config) {
			config.MaxConnectionPoolSize = cfg.MaxConnections
			config.ConnectionAcquisitionTimeout = cfg.ConnectionTimeout
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	client := &Client{
		driver: driver,
		logger: logger,
		config: cfg,
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := client.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return client, nil
}

// Close closes the Neo4j driver
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.driver.Close(ctx)
}

// VerifyConnectivity verifies the connection to Neo4j
func (c *Client) VerifyConnectivity(ctx context.Context) error {
	return c.driver.VerifyConnectivity(ctx)
}

// GetSubGraph retrieves a subgraph around specified entities
func (c *Client) GetSubGraph(ctx context.Context, entityIDs []string, depth int) (*SubGraph, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (start:Entity)
		WHERE start.id IN $entity_ids
		CALL apoc.path.subgraphAll(start, {
			relationshipFilter: "",
			minLevel: 0,
			maxLevel: $depth
		}) YIELD nodes, relationships
		RETURN nodes, relationships
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"entity_ids": entityIDs,
			"depth":      depth,
		})
		if err != nil {
			return nil, err
		}

		var entities []*Entity
		var relationships []*Relationship

		for result.Next(ctx) {
			record := result.Record()
			
			// Process nodes
			if nodes, ok := record.Get("nodes"); ok {
				nodeList := nodes.([]interface{})
				for _, nodeInterface := range nodeList {
					node := nodeInterface.(neo4j.Node)
					entity := c.nodeToEntity(node)
					entities = append(entities, entity)
				}
			}

			// Process relationships
			if rels, ok := record.Get("relationships"); ok {
				relList := rels.([]interface{})
				for _, relInterface := range relList {
					rel := relInterface.(neo4j.Relationship)
					relationship := c.relationshipToEdge(rel)
					relationships = append(relationships, relationship)
				}
			}
		}

		return &SubGraph{
			Entities:      entities,
			Relationships: relationships,
			Metadata: map[string]interface{}{
				"depth":        depth,
				"center_nodes": entityIDs,
				"retrieved_at": time.Now(),
			},
		}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get subgraph: %w", err)
	}

	return result.(*SubGraph), nil
}

// FindShortestPaths finds shortest paths between two sets of entities
func (c *Client) FindShortestPaths(ctx context.Context, sourceIDs, targetIDs []string, maxLength int) ([]*Path, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Entity), (target:Entity)
		WHERE source.id IN $source_ids AND target.id IN $target_ids
		MATCH path = shortestPath((source)-[*1..` + fmt.Sprintf("%d", maxLength) + `]-(target))
		RETURN path, length(path) as pathLength
		ORDER BY pathLength
		LIMIT 10
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"source_ids": sourceIDs,
			"target_ids": targetIDs,
		})
		if err != nil {
			return nil, err
		}

		var paths []*Path
		for result.Next(ctx) {
			record := result.Record()
			path := record.Values[0].(neo4j.Path)
			length := record.Values[1].(int64)

			pathResult := c.pathToResult(path, int(length))
			paths = append(paths, pathResult)
		}

		return paths, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find shortest paths: %w", err)
	}

	return result.([]*Path), nil
}

// CalculateCentralityMetrics calculates centrality metrics for entities
func (c *Client) CalculateCentralityMetrics(ctx context.Context, entityIDs []string) ([]*CentralityMetrics, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	// Calculate degree centrality
	degreeQuery := `
		MATCH (e:Entity)
		WHERE e.id IN $entity_ids
		RETURN e.id as entity_id, 
			   size((e)--()) as degree_centrality
	`

	// Calculate betweenness centrality (simplified)
	betweennessQuery := `
		CALL gds.betweenness.stream('graph-projection', {
			nodeQuery: 'MATCH (e:Entity) WHERE e.id IN $entity_ids RETURN id(e) as id',
			relationshipQuery: 'MATCH (e1:Entity)-[r]-(e2:Entity) RETURN id(e1) as source, id(e2) as target'
		})
		YIELD nodeId, score
		MATCH (e:Entity) WHERE id(e) = nodeId
		RETURN e.id as entity_id, score as betweenness_centrality
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Get degree centrality
		result, err := tx.Run(ctx, degreeQuery, map[string]interface{}{
			"entity_ids": entityIDs,
		})
		if err != nil {
			return nil, err
		}

		metrics := make(map[string]*CentralityMetrics)
		for result.Next(ctx) {
			record := result.Record()
			entityID := record.Values[0].(string)
			degree := record.Values[1].(int64)

			metrics[entityID] = &CentralityMetrics{
				EntityID:         entityID,
				DegreeCentrality: float64(degree),
			}
		}

		// Note: In a real implementation, you would calculate other centrality measures
		// using Graph Data Science library or similar algorithms
		
		var resultList []*CentralityMetrics
		for _, metric := range metrics {
			resultList = append(resultList, metric)
		}

		return resultList, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate centrality metrics: %w", err)
	}

	return result.([]*CentralityMetrics), nil
}

// DetectCommunities detects communities/clusters in the graph
func (c *Client) DetectCommunities(ctx context.Context, entityIDs []string) ([]*Community, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	// Simplified community detection using connected components
	query := `
		MATCH (e:Entity)
		WHERE e.id IN $entity_ids
		CALL apoc.algo.unionFind.stream(e, {
			relationshipTypes: ['']
		})
		YIELD nodeId, setId
		MATCH (n:Entity) WHERE id(n) = nodeId
		RETURN setId as community_id, 
			   collect(n.id) as entity_ids,
			   count(n) as size
		ORDER BY size DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"entity_ids": entityIDs,
		})
		if err != nil {
			return nil, err
		}

		var communities []*Community
		for result.Next(ctx) {
			record := result.Record()
			communityID := fmt.Sprintf("community_%d", record.Values[0].(int64))
			entities := record.Values[1].([]interface{})
			size := record.Values[2].(int64)

			var entityList []string
			for _, entity := range entities {
				entityList = append(entityList, entity.(string))
			}

			community := &Community{
				ID:       communityID,
				Entities: entityList,
				Size:     int(size),
				Density:  0.0, // Would calculate actual density
				Modularity: 0.0, // Would calculate actual modularity
			}

			communities = append(communities, community)
		}

		return communities, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to detect communities: %w", err)
	}

	return result.([]*Community), nil
}

// FindPatterns finds specific patterns in the graph
func (c *Client) FindPatterns(ctx context.Context, patternType string, entityIDs []string) ([]*PatternMatch, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	var query string
	switch patternType {
	case "triangle":
		query = `
			MATCH (a:Entity)-[r1]-(b:Entity)-[r2]-(c:Entity)-[r3]-(a)
			WHERE a.id IN $entity_ids OR b.id IN $entity_ids OR c.id IN $entity_ids
			RETURN a, b, c, r1, r2, r3
			LIMIT 50
		`
	case "star":
		query = `
			MATCH (center:Entity)-[r]-(leaf:Entity)
			WHERE center.id IN $entity_ids
			WITH center, collect(leaf) as leaves, collect(r) as relationships
			WHERE size(leaves) >= 3
			RETURN center, leaves, relationships
			LIMIT 50
		`
	case "chain":
		query = `
			MATCH path = (a:Entity)-[*3..5]-(b:Entity)
			WHERE a.id IN $entity_ids AND b.id IN $entity_ids
			RETURN path
			LIMIT 50
		`
	default:
		return nil, fmt.Errorf("unsupported pattern type: %s", patternType)
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"entity_ids": entityIDs,
		})
		if err != nil {
			return nil, err
		}

		var patterns []*PatternMatch
		for result.Next(ctx) {
			record := result.Record()
			
			pattern := &PatternMatch{
				PatternType: patternType,
				Entities:    []*Entity{},
				Relationships: []*Relationship{},
				Confidence:  0.8, // Would calculate actual confidence
				Metadata: map[string]interface{}{
					"detected_at": time.Now(),
				},
			}

			// Process pattern based on type
			switch patternType {
			case "triangle":
				// Extract triangle entities and relationships
				for i := 0; i < 3; i++ {
					if node, ok := record.Values[i].(neo4j.Node); ok {
						entity := c.nodeToEntity(node)
						pattern.Entities = append(pattern.Entities, entity)
					}
				}
				for i := 3; i < 6; i++ {
					if rel, ok := record.Values[i].(neo4j.Relationship); ok {
						relationship := c.relationshipToEdge(rel)
						pattern.Relationships = append(pattern.Relationships, relationship)
					}
				}
			}

			patterns = append(patterns, pattern)
		}

		return patterns, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find patterns: %w", err)
	}

	return result.([]*PatternMatch), nil
}

// GetEntityNeighborhood gets immediate neighbors of an entity
func (c *Client) GetEntityNeighborhood(ctx context.Context, entityID string, relationshipTypes []string) (*SubGraph, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	var typeFilter string
	if len(relationshipTypes) > 0 {
		typeFilter = ":" + fmt.Sprintf("[%s]", relationshipTypes[0])
		for i := 1; i < len(relationshipTypes); i++ {
			typeFilter += "|" + relationshipTypes[i]
		}
	}

	query := `
		MATCH (center:Entity {id: $entity_id})-[r` + typeFilter + `]-(neighbor:Entity)
		RETURN center, collect(DISTINCT neighbor) as neighbors, collect(DISTINCT r) as relationships
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"entity_id": entityID,
		})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			
			var entities []*Entity
			var relationships []*Relationship

			// Add center entity
			if centerNode, ok := record.Values[0].(neo4j.Node); ok {
				entities = append(entities, c.nodeToEntity(centerNode))
			}

			// Add neighbor entities
			if neighbors, ok := record.Values[1].([]interface{}); ok {
				for _, neighborInterface := range neighbors {
					neighbor := neighborInterface.(neo4j.Node)
					entities = append(entities, c.nodeToEntity(neighbor))
				}
			}

			// Add relationships
			if rels, ok := record.Values[2].([]interface{}); ok {
				for _, relInterface := range rels {
					rel := relInterface.(neo4j.Relationship)
					relationships = append(relationships, c.relationshipToEdge(rel))
				}
			}

			return &SubGraph{
				Entities:      entities,
				Relationships: relationships,
				Metadata: map[string]interface{}{
					"center_entity": entityID,
					"depth":         1,
					"retrieved_at":  time.Now(),
				},
			}, nil
		}

		return &SubGraph{
			Entities:      []*Entity{},
			Relationships: []*Relationship{},
			Metadata:      map[string]interface{}{},
		}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get entity neighborhood: %w", err)
	}

	return result.(*SubGraph), nil
}

// Helper functions

func (c *Client) nodeToEntity(node neo4j.Node) *Entity {
	entity := &Entity{
		Properties: make(map[string]interface{}),
	}

	// Get entity ID and type
	if id, exists := node.Props["id"]; exists {
		entity.ID = id.(string)
	}

	if len(node.Labels) > 0 {
		entity.Type = node.Labels[0]
	}

	// Copy all properties
	for key, value := range node.Props {
		entity.Properties[key] = value
	}

	return entity
}

func (c *Client) relationshipToEdge(rel neo4j.Relationship) *Relationship {
	relationship := &Relationship{
		Type:       rel.Type,
		SourceID:   fmt.Sprintf("%d", rel.StartId),
		TargetID:   fmt.Sprintf("%d", rel.EndId),
		Properties: make(map[string]interface{}),
	}

	// Get relationship ID
	if id, exists := rel.Props["id"]; exists {
		relationship.ID = id.(string)
	}

	// Copy all properties
	for key, value := range rel.Props {
		relationship.Properties[key] = value
	}

	return relationship
}

func (c *Client) pathToResult(path neo4j.Path, length int) *Path {
	pathResult := &Path{
		Length:        length,
		Cost:          float64(length), // Simple cost calculation
		Entities:      []*Entity{},
		Relationships: []*Relationship{},
	}

	// Convert nodes to entities
	for _, node := range path.Nodes {
		entity := c.nodeToEntity(node)
		pathResult.Entities = append(pathResult.Entities, entity)
	}

	// Set start and end entities
	if len(pathResult.Entities) > 0 {
		pathResult.StartEntity = pathResult.Entities[0]
		pathResult.EndEntity = pathResult.Entities[len(pathResult.Entities)-1]
	}

	// Convert relationships
	for _, rel := range path.Relationships {
		relationship := c.relationshipToEdge(rel)
		pathResult.Relationships = append(pathResult.Relationships, relationship)
	}

	return pathResult
}