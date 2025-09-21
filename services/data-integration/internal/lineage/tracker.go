package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Tracker handles data lineage tracking
type Tracker struct {
	logger *zap.Logger
	store  LineageStore
}

// LineageStore defines the interface for lineage storage
type LineageStore interface {
	Store(ctx context.Context, lineage *LineageRecord) error
	Get(ctx context.Context, id string) (*LineageRecord, error)
	Query(ctx context.Context, query *LineageQuery) ([]*LineageRecord, error)
	GetUpstream(ctx context.Context, entityID string) ([]*LineageRecord, error)
	GetDownstream(ctx context.Context, entityID string) ([]*LineageRecord, error)
}

// LineageInfo represents lineage information for tracking
type LineageInfo struct {
	JobID       string                 `json:"job_id"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	RecordCount int                    `json:"record_count"`
	ProcessedAt time.Time              `json:"processed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LineageRecord represents a complete lineage record
type LineageRecord struct {
	ID          string                 `json:"id"`
	JobID       string                 `json:"job_id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Source      *DataSource            `json:"source"`
	Target      *DataTarget            `json:"target"`
	Operation   string                 `json:"operation"`
	ProcessedAt time.Time              `json:"processed_at"`
	RecordCount int                    `json:"record_count"`
	Schema      *SchemaInfo            `json:"schema,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Children    []string               `json:"children,omitempty"`
}

// DataSource represents the source of data
type DataSource struct {
	Type        string                 `json:"type"`
	Location    string                 `json:"location"`
	Schema      string                 `json:"schema,omitempty"`
	Table       string                 `json:"table,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DataTarget represents the target of data
type DataTarget struct {
	Type        string                 `json:"type"`
	Location    string                 `json:"location"`
	Schema      string                 `json:"schema,omitempty"`
	Table       string                 `json:"table,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SchemaInfo represents schema information
type SchemaInfo struct {
	Version string                 `json:"version"`
	Fields  []FieldInfo            `json:"fields"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FieldInfo represents field information
type FieldInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// LineageQuery represents a query for lineage records
type LineageQuery struct {
	EntityID    string    `json:"entity_id,omitempty"`
	EntityType  string    `json:"entity_type,omitempty"`
	JobID       string    `json:"job_id,omitempty"`
	Operation   string    `json:"operation,omitempty"`
	Source      string    `json:"source,omitempty"`
	Target      string    `json:"target,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Offset      int       `json:"offset,omitempty"`
}

// LineageGraph represents a data lineage graph
type LineageGraph struct {
	Nodes []*LineageNode `json:"nodes"`
	Edges []*LineageEdge `json:"edges"`
}

// LineageNode represents a node in the lineage graph
type LineageNode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Label       string                 `json:"label"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// LineageEdge represents an edge in the lineage graph
type LineageEdge struct {
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// NewTracker creates a new lineage tracker
func NewTracker(store LineageStore, logger *zap.Logger) *Tracker {
	return &Tracker{
		logger: logger,
		store:  store,
	}
}

// Track tracks data lineage
func (t *Tracker) Track(ctx context.Context, info *LineageInfo) error {
	record := &LineageRecord{
		ID:          t.generateID(info),
		JobID:       info.JobID,
		EntityType:  "data_flow",
		EntityID:    fmt.Sprintf("%s_to_%s", info.Source, info.Target),
		Source:      t.parseDataSource(info.Source),
		Target:      t.parseDataTarget(info.Target),
		Operation:   "transform",
		ProcessedAt: info.ProcessedAt,
		RecordCount: info.RecordCount,
		Metadata:    info.Metadata,
	}

	t.logger.Info("Tracking data lineage",
		zap.String("job_id", info.JobID),
		zap.String("source", info.Source),
		zap.String("target", info.Target),
		zap.Int("record_count", info.RecordCount))

	return t.store.Store(ctx, record)
}

// GetLineage retrieves lineage information for an entity
func (t *Tracker) GetLineage(ctx context.Context, entityID string) (*LineageRecord, error) {
	return t.store.Get(ctx, entityID)
}

// QueryLineage queries lineage records
func (t *Tracker) QueryLineage(ctx context.Context, query *LineageQuery) ([]*LineageRecord, error) {
	return t.store.Query(ctx, query)
}

// GetUpstreamLineage gets upstream lineage for an entity
func (t *Tracker) GetUpstreamLineage(ctx context.Context, entityID string) ([]*LineageRecord, error) {
	return t.store.GetUpstream(ctx, entityID)
}

// GetDownstreamLineage gets downstream lineage for an entity
func (t *Tracker) GetDownstreamLineage(ctx context.Context, entityID string) ([]*LineageRecord, error) {
	return t.store.GetDownstream(ctx, entityID)
}

// BuildLineageGraph builds a lineage graph for visualization
func (t *Tracker) BuildLineageGraph(ctx context.Context, entityID string, depth int) (*LineageGraph, error) {
	graph := &LineageGraph{
		Nodes: []*LineageNode{},
		Edges: []*LineageEdge{},
	}

	visited := make(map[string]bool)
	
	// Build graph recursively
	if err := t.buildGraphRecursive(ctx, entityID, depth, visited, graph); err != nil {
		return nil, err
	}

	t.logger.Info("Built lineage graph",
		zap.String("entity_id", entityID),
		zap.Int("depth", depth),
		zap.Int("nodes", len(graph.Nodes)),
		zap.Int("edges", len(graph.Edges)))

	return graph, nil
}

// TrackSchemaEvolution tracks schema changes
func (t *Tracker) TrackSchemaEvolution(ctx context.Context, source string, oldSchema, newSchema *SchemaInfo) error {
	record := &LineageRecord{
		ID:          t.generateSchemaChangeID(source, newSchema.Version),
		EntityType:  "schema_evolution",
		EntityID:    source,
		Operation:   "schema_change",
		ProcessedAt: time.Now(),
		Schema:      newSchema,
		Metadata: map[string]interface{}{
			"old_schema_version": oldSchema.Version,
			"new_schema_version": newSchema.Version,
			"changes":           t.compareSchemas(oldSchema, newSchema),
		},
	}

	t.logger.Info("Tracking schema evolution",
		zap.String("source", source),
		zap.String("old_version", oldSchema.Version),
		zap.String("new_version", newSchema.Version))

	return t.store.Store(ctx, record)
}

// GetSchemaHistory gets schema evolution history for a source
func (t *Tracker) GetSchemaHistory(ctx context.Context, source string) ([]*LineageRecord, error) {
	query := &LineageQuery{
		EntityID:   source,
		EntityType: "schema_evolution",
		Operation:  "schema_change",
		Limit:      100,
	}

	return t.store.Query(ctx, query)
}

// Helper methods

func (t *Tracker) generateID(info *LineageInfo) string {
	return fmt.Sprintf("lineage_%s_%d", info.JobID, info.ProcessedAt.Unix())
}

func (t *Tracker) generateSchemaChangeID(source, version string) string {
	return fmt.Sprintf("schema_%s_%s_%d", source, version, time.Now().Unix())
}

func (t *Tracker) parseDataSource(source string) *DataSource {
	// Parse source string to extract type, location, etc.
	// This is a simplified implementation
	return &DataSource{
		Type:     "database",
		Location: source,
		Format:   "json",
	}
}

func (t *Tracker) parseDataTarget(target string) *DataTarget {
	// Parse target string to extract type, location, etc.
	// This is a simplified implementation
	return &DataTarget{
		Type:     "database",
		Location: target,
		Format:   "json",
	}
}

func (t *Tracker) buildGraphRecursive(ctx context.Context, entityID string, depth int, visited map[string]bool, graph *LineageGraph) error {
	if depth <= 0 || visited[entityID] {
		return nil
	}

	visited[entityID] = true

	// Add current node
	node := &LineageNode{
		ID:    entityID,
		Type:  "entity",
		Label: entityID,
	}
	graph.Nodes = append(graph.Nodes, node)

	// Get upstream lineage
	upstream, err := t.store.GetUpstream(ctx, entityID)
	if err != nil {
		return err
	}

	for _, record := range upstream {
		sourceID := record.Source.Location
		if !visited[sourceID] {
			// Add upstream node
			sourceNode := &LineageNode{
				ID:    sourceID,
				Type:  "source",
				Label: sourceID,
			}
			graph.Nodes = append(graph.Nodes, sourceNode)

			// Add edge
			edge := &LineageEdge{
				Source: sourceID,
				Target: entityID,
				Type:   "flows_to",
				Label:  record.Operation,
			}
			graph.Edges = append(graph.Edges, edge)

			// Recurse
			if err := t.buildGraphRecursive(ctx, sourceID, depth-1, visited, graph); err != nil {
				return err
			}
		}
	}

	// Get downstream lineage
	downstream, err := t.store.GetDownstream(ctx, entityID)
	if err != nil {
		return err
	}

	for _, record := range downstream {
		targetID := record.Target.Location
		if !visited[targetID] {
			// Add downstream node
			targetNode := &LineageNode{
				ID:    targetID,
				Type:  "target",
				Label: targetID,
			}
			graph.Nodes = append(graph.Nodes, targetNode)

			// Add edge
			edge := &LineageEdge{
				Source: entityID,
				Target: targetID,
				Type:   "flows_to",
				Label:  record.Operation,
			}
			graph.Edges = append(graph.Edges, edge)

			// Recurse
			if err := t.buildGraphRecursive(ctx, targetID, depth-1, visited, graph); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Tracker) compareSchemas(oldSchema, newSchema *SchemaInfo) map[string]interface{} {
	changes := map[string]interface{}{
		"added_fields":   []string{},
		"removed_fields": []string{},
		"modified_fields": []string{},
	}

	oldFields := make(map[string]FieldInfo)
	for _, field := range oldSchema.Fields {
		oldFields[field.Name] = field
	}

	newFields := make(map[string]FieldInfo)
	for _, field := range newSchema.Fields {
		newFields[field.Name] = field
	}

	// Find added and modified fields
	for name, newField := range newFields {
		if oldField, exists := oldFields[name]; exists {
			// Check if field was modified
			if oldField.Type != newField.Type || oldField.Required != newField.Required {
				changes["modified_fields"] = append(changes["modified_fields"].([]string), name)
			}
		} else {
			// Field was added
			changes["added_fields"] = append(changes["added_fields"].([]string), name)
		}
	}

	// Find removed fields
	for name := range oldFields {
		if _, exists := newFields[name]; !exists {
			changes["removed_fields"] = append(changes["removed_fields"].([]string), name)
		}
	}

	return changes
}

// In-memory lineage store implementation for testing/development
type InMemoryLineageStore struct {
	records map[string]*LineageRecord
	logger  *zap.Logger
}

// NewInMemoryLineageStore creates a new in-memory lineage store
func NewInMemoryLineageStore(logger *zap.Logger) *InMemoryLineageStore {
	return &InMemoryLineageStore{
		records: make(map[string]*LineageRecord),
		logger:  logger,
	}
}

func (s *InMemoryLineageStore) Store(ctx context.Context, lineage *LineageRecord) error {
	s.records[lineage.ID] = lineage
	s.logger.Debug("Stored lineage record", zap.String("id", lineage.ID))
	return nil
}

func (s *InMemoryLineageStore) Get(ctx context.Context, id string) (*LineageRecord, error) {
	if record, exists := s.records[id]; exists {
		return record, nil
	}
	return nil, fmt.Errorf("lineage record not found: %s", id)
}

func (s *InMemoryLineageStore) Query(ctx context.Context, query *LineageQuery) ([]*LineageRecord, error) {
	var results []*LineageRecord

	for _, record := range s.records {
		if s.matchesQuery(record, query) {
			results = append(results, record)
		}
	}

	// Apply limit and offset
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}

	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

func (s *InMemoryLineageStore) GetUpstream(ctx context.Context, entityID string) ([]*LineageRecord, error) {
	var results []*LineageRecord

	for _, record := range s.records {
		if record.Target != nil && record.Target.Location == entityID {
			results = append(results, record)
		}
	}

	return results, nil
}

func (s *InMemoryLineageStore) GetDownstream(ctx context.Context, entityID string) ([]*LineageRecord, error) {
	var results []*LineageRecord

	for _, record := range s.records {
		if record.Source != nil && record.Source.Location == entityID {
			results = append(results, record)
		}
	}

	return results, nil
}

func (s *InMemoryLineageStore) matchesQuery(record *LineageRecord, query *LineageQuery) bool {
	if query.EntityID != "" && record.EntityID != query.EntityID {
		return false
	}

	if query.EntityType != "" && record.EntityType != query.EntityType {
		return false
	}

	if query.JobID != "" && record.JobID != query.JobID {
		return false
	}

	if query.Operation != "" && record.Operation != query.Operation {
		return false
	}

	if !query.StartTime.IsZero() && record.ProcessedAt.Before(query.StartTime) {
		return false
	}

	if !query.EndTime.IsZero() && record.ProcessedAt.After(query.EndTime) {
		return false
	}

	return true
}