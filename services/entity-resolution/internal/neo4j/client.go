package neo4j

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps Neo4j driver for entity resolution operations
type Client struct {
	driver neo4j.DriverWithContext
	logger *slog.Logger
	config config.Neo4jConfig
}

// EntityNode represents an entity in the Neo4j graph
type EntityNode struct {
	ID               string                 `json:"id"`
	EntityType       string                 `json:"entity_type"`
	Name             string                 `json:"name"`
	StandardizedName string                 `json:"standardized_name"`
	Identifiers      map[string]interface{} `json:"identifiers"`
	Attributes       map[string]interface{} `json:"attributes"`
	ConfidenceScore  float64                `json:"confidence_score"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// RelationshipEdge represents a relationship between entities
type RelationshipEdge struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	SourceEntityID  string                 `json:"source_entity_id"`
	TargetEntityID  string                 `json:"target_entity_id"`
	Properties      map[string]interface{} `json:"properties"`
	ConfidenceScore float64                `json:"confidence_score"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// PathResult represents a path between entities
type PathResult struct {
	StartEntity   *EntityNode        `json:"start_entity"`
	EndEntity     *EntityNode        `json:"end_entity"`
	Relationships []*RelationshipEdge `json:"relationships"`
	Length        int                `json:"length"`
	Score         float64            `json:"score"`
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

	// Create indexes and constraints
	if err := client.createIndexes(ctx); err != nil {
		logger.Warn("Failed to create Neo4j indexes", "error", err)
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

// CreateEntity creates an entity node in the graph
func (c *Client) CreateEntity(ctx context.Context, entity *EntityNode) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		CREATE (e:Entity {
			id: $id,
			entity_type: $entity_type,
			name: $name,
			standardized_name: $standardized_name,
			identifiers: $identifiers,
			attributes: $attributes,
			confidence_score: $confidence_score,
			created_at: $created_at,
			updated_at: $updated_at
		})
		RETURN e.id
	`

	parameters := map[string]interface{}{
		"id":                entity.ID,
		"entity_type":       entity.EntityType,
		"name":              entity.Name,
		"standardized_name": entity.StandardizedName,
		"identifiers":       entity.Identifiers,
		"attributes":        entity.Attributes,
		"confidence_score":  entity.ConfidenceScore,
		"created_at":        entity.CreatedAt,
		"updated_at":        entity.UpdatedAt,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			return result.Record().Values[0], nil
		}

		return nil, fmt.Errorf("failed to create entity")
	})

	if err != nil {
		return fmt.Errorf("failed to create entity in Neo4j: %w", err)
	}

	c.logger.Info("Entity created in Neo4j", "entity_id", entity.ID)
	return nil
}

// UpdateEntity updates an entity node in the graph
func (c *Client) UpdateEntity(ctx context.Context, entity *EntityNode) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (e:Entity {id: $id})
		SET e.name = $name,
			e.standardized_name = $standardized_name,
			e.identifiers = $identifiers,
			e.attributes = $attributes,
			e.confidence_score = $confidence_score,
			e.updated_at = $updated_at
		RETURN e.id
	`

	parameters := map[string]interface{}{
		"id":                entity.ID,
		"name":              entity.Name,
		"standardized_name": entity.StandardizedName,
		"identifiers":       entity.Identifiers,
		"attributes":        entity.Attributes,
		"confidence_score":  entity.ConfidenceScore,
		"updated_at":        entity.UpdatedAt,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			return result.Record().Values[0], nil
		}

		return nil, fmt.Errorf("entity not found")
	})

	if err != nil {
		return fmt.Errorf("failed to update entity in Neo4j: %w", err)
	}

	c.logger.Info("Entity updated in Neo4j", "entity_id", entity.ID)
	return nil
}

// GetEntity retrieves an entity by ID
func (c *Client) GetEntity(ctx context.Context, entityID string) (*EntityNode, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (e:Entity {id: $id})
		RETURN e.id, e.entity_type, e.name, e.standardized_name,
			   e.identifiers, e.attributes, e.confidence_score,
			   e.created_at, e.updated_at
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{"id": entityID})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			entity := &EntityNode{
				ID:               record.Values[0].(string),
				EntityType:       record.Values[1].(string),
				Name:             record.Values[2].(string),
				StandardizedName: record.Values[3].(string),
				ConfidenceScore:  record.Values[6].(float64),
				CreatedAt:        record.Values[7].(time.Time),
				UpdatedAt:        record.Values[8].(time.Time),
			}

			// Handle optional map fields
			if record.Values[4] != nil {
				entity.Identifiers = record.Values[4].(map[string]interface{})
			}
			if record.Values[5] != nil {
				entity.Attributes = record.Values[5].(map[string]interface{})
			}

			return entity, nil
		}

		return nil, fmt.Errorf("entity not found")
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get entity from Neo4j: %w", err)
	}

	return result.(*EntityNode), nil
}

// CreateRelationship creates a relationship between two entities
func (c *Client) CreateRelationship(ctx context.Context, relationship *RelationshipEdge) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Entity {id: $source_id})
		MATCH (target:Entity {id: $target_id})
		CREATE (source)-[r:` + relationship.Type + ` {
			id: $id,
			properties: $properties,
			confidence_score: $confidence_score,
			created_at: $created_at,
			updated_at: $updated_at
		}]->(target)
		RETURN r.id
	`

	parameters := map[string]interface{}{
		"source_id":        relationship.SourceEntityID,
		"target_id":        relationship.TargetEntityID,
		"id":               relationship.ID,
		"properties":       relationship.Properties,
		"confidence_score": relationship.ConfidenceScore,
		"created_at":       relationship.CreatedAt,
		"updated_at":       relationship.UpdatedAt,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			return result.Record().Values[0], nil
		}

		return nil, fmt.Errorf("failed to create relationship")
	})

	if err != nil {
		return fmt.Errorf("failed to create relationship in Neo4j: %w", err)
	}

	c.logger.Info("Relationship created in Neo4j",
		"relationship_id", relationship.ID,
		"type", relationship.Type,
		"source", relationship.SourceEntityID,
		"target", relationship.TargetEntityID)

	return nil
}

// FindConnectedEntities finds entities connected to the given entity
func (c *Client) FindConnectedEntities(ctx context.Context, entityID string, maxDepth int) ([]*EntityNode, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (start:Entity {id: $entity_id})
		MATCH (start)-[*1..` + fmt.Sprintf("%d", maxDepth) + `]-(connected:Entity)
		WHERE connected.id <> $entity_id
		RETURN DISTINCT connected.id, connected.entity_type, connected.name,
			   connected.standardized_name, connected.identifiers,
			   connected.attributes, connected.confidence_score,
			   connected.created_at, connected.updated_at
		ORDER BY connected.confidence_score DESC
		LIMIT 100
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{"entity_id": entityID})
		if err != nil {
			return nil, err
		}

		var entities []*EntityNode
		for result.Next(ctx) {
			record := result.Record()
			entity := &EntityNode{
				ID:               record.Values[0].(string),
				EntityType:       record.Values[1].(string),
				Name:             record.Values[2].(string),
				StandardizedName: record.Values[3].(string),
				ConfidenceScore:  record.Values[6].(float64),
				CreatedAt:        record.Values[7].(time.Time),
				UpdatedAt:        record.Values[8].(time.Time),
			}

			// Handle optional map fields
			if record.Values[4] != nil {
				entity.Identifiers = record.Values[4].(map[string]interface{})
			}
			if record.Values[5] != nil {
				entity.Attributes = record.Values[5].(map[string]interface{})
			}

			entities = append(entities, entity)
		}

		return entities, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find connected entities: %w", err)
	}

	return result.([]*EntityNode), nil
}

// FindShortestPath finds the shortest path between two entities
func (c *Client) FindShortestPath(ctx context.Context, sourceID, targetID string, maxLength int) (*PathResult, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Entity {id: $source_id})
		MATCH (target:Entity {id: $target_id})
		MATCH path = shortestPath((source)-[*1..` + fmt.Sprintf("%d", maxLength) + `]-(target))
		RETURN path, length(path) as pathLength
		ORDER BY pathLength
		LIMIT 1
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"source_id": sourceID,
			"target_id": targetID,
		})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			path := record.Values[0].(neo4j.Path)
			length := record.Values[1].(int64)

			pathResult := &PathResult{
				Length:        int(length),
				Relationships: []*RelationshipEdge{},
			}

			// Extract start and end entities
			if len(path.Nodes) > 0 {
				startNode := path.Nodes[0]
				pathResult.StartEntity = c.nodeToEntity(startNode)

				endNode := path.Nodes[len(path.Nodes)-1]
				pathResult.EndEntity = c.nodeToEntity(endNode)
			}

			// Extract relationships
			for _, rel := range path.Relationships {
				relationship := c.relationshipToEdge(rel)
				pathResult.Relationships = append(pathResult.Relationships, relationship)
			}

			return pathResult, nil
		}

		return nil, fmt.Errorf("no path found")
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find shortest path: %w", err)
	}

	return result.(*PathResult), nil
}

// FindSimilarEntities finds entities similar to the given entity based on properties
func (c *Client) FindSimilarEntities(ctx context.Context, entityID string, similarity float64) ([]*EntityNode, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (target:Entity {id: $entity_id})
		MATCH (similar:Entity)
		WHERE similar.id <> $entity_id
		  AND similar.entity_type = target.entity_type
		  AND apoc.text.sorensenDiceSimilarity(similar.standardized_name, target.standardized_name) >= $similarity
		RETURN similar.id, similar.entity_type, similar.name,
			   similar.standardized_name, similar.identifiers,
			   similar.attributes, similar.confidence_score,
			   similar.created_at, similar.updated_at,
			   apoc.text.sorensenDiceSimilarity(similar.standardized_name, target.standardized_name) as similarity_score
		ORDER BY similarity_score DESC
		LIMIT 50
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"entity_id":  entityID,
			"similarity": similarity,
		})
		if err != nil {
			return nil, err
		}

		var entities []*EntityNode
		for result.Next(ctx) {
			record := result.Record()
			entity := &EntityNode{
				ID:               record.Values[0].(string),
				EntityType:       record.Values[1].(string),
				Name:             record.Values[2].(string),
				StandardizedName: record.Values[3].(string),
				ConfidenceScore:  record.Values[6].(float64),
				CreatedAt:        record.Values[7].(time.Time),
				UpdatedAt:        record.Values[8].(time.Time),
			}

			// Handle optional map fields
			if record.Values[4] != nil {
				entity.Identifiers = record.Values[4].(map[string]interface{})
			}
			if record.Values[5] != nil {
				entity.Attributes = record.Values[5].(map[string]interface{})
			}

			entities = append(entities, entity)
		}

		return entities, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find similar entities: %w", err)
	}

	return result.([]*EntityNode), nil
}

// GetEntityMetrics calculates metrics for an entity (degree centrality, etc.)
func (c *Client) GetEntityMetrics(ctx context.Context, entityID string) (map[string]interface{}, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (e:Entity {id: $entity_id})
		OPTIONAL MATCH (e)-[outgoing]->(other)
		OPTIONAL MATCH (e)<-[incoming]-(other2)
		RETURN 
			count(DISTINCT outgoing) as outgoing_count,
			count(DISTINCT incoming) as incoming_count,
			count(DISTINCT outgoing) + count(DISTINCT incoming) as total_degree
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{"entity_id": entityID})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			metrics := map[string]interface{}{
				"outgoing_count": record.Values[0],
				"incoming_count": record.Values[1],
				"total_degree":   record.Values[2],
			}
			return metrics, nil
		}

		return map[string]interface{}{}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get entity metrics: %w", err)
	}

	return result.(map[string]interface{}), nil
}

// Helper functions

func (c *Client) nodeToEntity(node neo4j.Node) *EntityNode {
	entity := &EntityNode{
		ID:               node.Props["id"].(string),
		EntityType:       node.Props["entity_type"].(string),
		Name:             node.Props["name"].(string),
		StandardizedName: node.Props["standardized_name"].(string),
		ConfidenceScore:  node.Props["confidence_score"].(float64),
		CreatedAt:        node.Props["created_at"].(time.Time),
		UpdatedAt:        node.Props["updated_at"].(time.Time),
	}

	if identifiers, ok := node.Props["identifiers"].(map[string]interface{}); ok {
		entity.Identifiers = identifiers
	}

	if attributes, ok := node.Props["attributes"].(map[string]interface{}); ok {
		entity.Attributes = attributes
	}

	return entity
}

func (c *Client) relationshipToEdge(rel neo4j.Relationship) *RelationshipEdge {
	edge := &RelationshipEdge{
		ID:              rel.Props["id"].(string),
		Type:            rel.Type,
		SourceEntityID:  fmt.Sprintf("%d", rel.StartId),
		TargetEntityID:  fmt.Sprintf("%d", rel.EndId),
		ConfidenceScore: rel.Props["confidence_score"].(float64),
		CreatedAt:       rel.Props["created_at"].(time.Time),
		UpdatedAt:       rel.Props["updated_at"].(time.Time),
	}

	if properties, ok := rel.Props["properties"].(map[string]interface{}); ok {
		edge.Properties = properties
	}

	return edge
}

// createIndexes creates necessary indexes and constraints
func (c *Client) createIndexes(ctx context.Context) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close(ctx)

	queries := []string{
		"CREATE CONSTRAINT entity_id_unique IF NOT EXISTS FOR (e:Entity) REQUIRE e.id IS UNIQUE",
		"CREATE INDEX entity_type_index IF NOT EXISTS FOR (e:Entity) ON (e.entity_type)",
		"CREATE INDEX entity_name_index IF NOT EXISTS FOR (e:Entity) ON (e.name)",
		"CREATE INDEX entity_standardized_name_index IF NOT EXISTS FOR (e:Entity) ON (e.standardized_name)",
		"CREATE INDEX entity_confidence_index IF NOT EXISTS FOR (e:Entity) ON (e.confidence_score)",
	}

	for _, query := range queries {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			_, err := tx.Run(ctx, query, nil)
			return nil, err
		})

		if err != nil {
			c.logger.Warn("Failed to execute index creation query", "query", query, "error", err)
		}
	}

	return nil
}