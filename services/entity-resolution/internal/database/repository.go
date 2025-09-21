package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/shared/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Repository handles database operations for entity resolution
type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

// Entity represents an entity in the system
type Entity struct {
	ID                uuid.UUID              `json:"id"`
	EntityType        string                 `json:"entity_type"`
	Name              string                 `json:"name"`
	StandardizedName  string                 `json:"standardized_name"`
	Identifiers       json.RawMessage        `json:"identifiers"`
	Attributes        json.RawMessage        `json:"attributes"`
	ContactInfo       json.RawMessage        `json:"contact_info"`
	ConfidenceScore   float64                `json:"confidence_score"`
	Status            string                 `json:"status"`
	Sources           json.RawMessage        `json:"sources"`
	Metadata          json.RawMessage        `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// EntityLink represents a link between two entities
type EntityLink struct {
	ID              uuid.UUID       `json:"id"`
	SourceEntityID  uuid.UUID       `json:"source_entity_id"`
	TargetEntityID  uuid.UUID       `json:"target_entity_id"`
	LinkType        string          `json:"link_type"`
	ConfidenceScore float64         `json:"confidence_score"`
	Evidence        json.RawMessage `json:"evidence"`
	Status          string          `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ResolutionJob represents an entity resolution job
type ResolutionJob struct {
	ID              uuid.UUID       `json:"id"`
	JobType         string          `json:"job_type"`
	Status          string          `json:"status"`
	InputEntityID   *uuid.UUID      `json:"input_entity_id,omitempty"`
	ProcessedCount  int             `json:"processed_count"`
	MatchedCount    int             `json:"matched_count"`
	ErrorCount      int             `json:"error_count"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	ErrorMessage    string          `json:"error_message"`
	Configuration   json.RawMessage `json:"configuration"`
	Results         json.RawMessage `json:"results"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewRepository creates a new database repository
func NewRepository(cfg config.DatabaseConfig, logger *slog.Logger) (*Repository, error) {
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxConnections / 2)
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetConnMaxIdleTime(cfg.MaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// Migrate runs database migrations
func (r *Repository) Migrate() error {
	driver, err := postgres.WithInstance(r.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	r.logger.Info("Database migrations completed successfully")
	return nil
}

// Entity operations

// CreateEntity creates a new entity
func (r *Repository) CreateEntity(ctx context.Context, entity *Entity) error {
	query := `
		INSERT INTO entities (
			id, entity_type, name, standardized_name, identifiers, 
			attributes, contact_info, confidence_score, status, 
			sources, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err := r.db.ExecContext(ctx, query,
		entity.ID,
		entity.EntityType,
		entity.Name,
		entity.StandardizedName,
		entity.Identifiers,
		entity.Attributes,
		entity.ContactInfo,
		entity.ConfidenceScore,
		entity.Status,
		entity.Sources,
		entity.Metadata,
		entity.CreatedAt,
		entity.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	return nil
}

// GetEntity retrieves an entity by ID
func (r *Repository) GetEntity(ctx context.Context, id uuid.UUID) (*Entity, error) {
	entity := &Entity{}
	query := `
		SELECT id, entity_type, name, standardized_name, identifiers,
			   attributes, contact_info, confidence_score, status,
			   sources, metadata, created_at, updated_at
		FROM entities
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.EntityType,
		&entity.Name,
		&entity.StandardizedName,
		&entity.Identifiers,
		&entity.Attributes,
		&entity.ContactInfo,
		&entity.ConfidenceScore,
		&entity.Status,
		&entity.Sources,
		&entity.Metadata,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity not found")
		}
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	return entity, nil
}

// UpdateEntity updates an existing entity
func (r *Repository) UpdateEntity(ctx context.Context, entity *Entity) error {
	query := `
		UPDATE entities SET
			entity_type = $2, name = $3, standardized_name = $4,
			identifiers = $5, attributes = $6, contact_info = $7,
			confidence_score = $8, status = $9, sources = $10,
			metadata = $11, updated_at = $12
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		entity.ID,
		entity.EntityType,
		entity.Name,
		entity.StandardizedName,
		entity.Identifiers,
		entity.Attributes,
		entity.ContactInfo,
		entity.ConfidenceScore,
		entity.Status,
		entity.Sources,
		entity.Metadata,
		entity.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// FindEntitiesByName finds entities by name (exact and fuzzy matching)
func (r *Repository) FindEntitiesByName(ctx context.Context, name string, limit int) ([]*Entity, error) {
	query := `
		SELECT id, entity_type, name, standardized_name, identifiers,
			   attributes, contact_info, confidence_score, status,
			   sources, metadata, created_at, updated_at,
			   similarity(name, $1) as name_sim,
			   similarity(standardized_name, $2) as std_name_sim
		FROM entities
		WHERE similarity(name, $1) > 0.3 OR similarity(standardized_name, $2) > 0.3
		ORDER BY GREATEST(similarity(name, $1), similarity(standardized_name, $2)) DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, name, name, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find entities by name: %w", err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity := &Entity{}
		var nameSim, stdNameSim float64

		err := rows.Scan(
			&entity.ID,
			&entity.EntityType,
			&entity.Name,
			&entity.StandardizedName,
			&entity.Identifiers,
			&entity.Attributes,
			&entity.ContactInfo,
			&entity.ConfidenceScore,
			&entity.Status,
			&entity.Sources,
			&entity.Metadata,
			&entity.CreatedAt,
			&entity.UpdatedAt,
			&nameSim,
			&stdNameSim,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// FindEntitiesByIdentifier finds entities by identifier
func (r *Repository) FindEntitiesByIdentifier(ctx context.Context, identifierType, identifierValue string) ([]*Entity, error) {
	query := `
		SELECT id, entity_type, name, standardized_name, identifiers,
			   attributes, contact_info, confidence_score, status,
			   sources, metadata, created_at, updated_at
		FROM entities
		WHERE identifiers->>$1 = $2
		ORDER BY confidence_score DESC`

	rows, err := r.db.QueryContext(ctx, query, identifierType, identifierValue)
	if err != nil {
		return nil, fmt.Errorf("failed to find entities by identifier: %w", err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity := &Entity{}

		err := rows.Scan(
			&entity.ID,
			&entity.EntityType,
			&entity.Name,
			&entity.StandardizedName,
			&entity.Identifiers,
			&entity.Attributes,
			&entity.ContactInfo,
			&entity.ConfidenceScore,
			&entity.Status,
			&entity.Sources,
			&entity.Metadata,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// ListEntities lists entities with pagination
func (r *Repository) ListEntities(ctx context.Context, limit, offset int, entityType string) ([]*Entity, error) {
	var query string
	var args []interface{}

	if entityType != "" {
		query = `
			SELECT id, entity_type, name, standardized_name, identifiers,
				   attributes, contact_info, confidence_score, status,
				   sources, metadata, created_at, updated_at
			FROM entities
			WHERE entity_type = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{entityType, limit, offset}
	} else {
		query = `
			SELECT id, entity_type, name, standardized_name, identifiers,
				   attributes, contact_info, confidence_score, status,
				   sources, metadata, created_at, updated_at
			FROM entities
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity := &Entity{}

		err := rows.Scan(
			&entity.ID,
			&entity.EntityType,
			&entity.Name,
			&entity.StandardizedName,
			&entity.Identifiers,
			&entity.Attributes,
			&entity.ContactInfo,
			&entity.ConfidenceScore,
			&entity.Status,
			&entity.Sources,
			&entity.Metadata,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// Entity link operations

// CreateEntityLink creates a new entity link
func (r *Repository) CreateEntityLink(ctx context.Context, link *EntityLink) error {
	query := `
		INSERT INTO entity_links (
			id, source_entity_id, target_entity_id, link_type,
			confidence_score, evidence, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.ExecContext(ctx, query,
		link.ID,
		link.SourceEntityID,
		link.TargetEntityID,
		link.LinkType,
		link.ConfidenceScore,
		link.Evidence,
		link.Status,
		link.CreatedAt,
		link.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create entity link: %w", err)
	}

	return nil
}

// GetEntityLinks retrieves links for an entity
func (r *Repository) GetEntityLinks(ctx context.Context, entityID uuid.UUID) ([]*EntityLink, error) {
	query := `
		SELECT id, source_entity_id, target_entity_id, link_type,
			   confidence_score, evidence, status, created_at, updated_at
		FROM entity_links
		WHERE source_entity_id = $1 OR target_entity_id = $1
		ORDER BY confidence_score DESC`

	rows, err := r.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity links: %w", err)
	}
	defer rows.Close()

	var links []*EntityLink
	for rows.Next() {
		link := &EntityLink{}

		err := rows.Scan(
			&link.ID,
			&link.SourceEntityID,
			&link.TargetEntityID,
			&link.LinkType,
			&link.ConfidenceScore,
			&link.Evidence,
			&link.Status,
			&link.CreatedAt,
			&link.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan entity link: %w", err)
		}

		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity links: %w", err)
	}

	return links, nil
}

// Resolution job operations

// CreateResolutionJob creates a new resolution job
func (r *Repository) CreateResolutionJob(ctx context.Context, job *ResolutionJob) error {
	query := `
		INSERT INTO resolution_jobs (
			id, job_type, status, input_entity_id, processed_count,
			matched_count, error_count, started_at, completed_at,
			error_message, configuration, results, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.JobType,
		job.Status,
		job.InputEntityID,
		job.ProcessedCount,
		job.MatchedCount,
		job.ErrorCount,
		job.StartedAt,
		job.CompletedAt,
		job.ErrorMessage,
		job.Configuration,
		job.Results,
		job.CreatedAt,
		job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create resolution job: %w", err)
	}

	return nil
}

// UpdateResolutionJob updates an existing resolution job
func (r *Repository) UpdateResolutionJob(ctx context.Context, job *ResolutionJob) error {
	query := `
		UPDATE resolution_jobs SET
			status = $2, processed_count = $3, matched_count = $4,
			error_count = $5, started_at = $6, completed_at = $7,
			error_message = $8, results = $9, updated_at = $10
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.Status,
		job.ProcessedCount,
		job.MatchedCount,
		job.ErrorCount,
		job.StartedAt,
		job.CompletedAt,
		job.ErrorMessage,
		job.Results,
		job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update resolution job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("resolution job not found")
	}

	return nil
}

// GetResolutionJob retrieves a resolution job by ID
func (r *Repository) GetResolutionJob(ctx context.Context, id uuid.UUID) (*ResolutionJob, error) {
	job := &ResolutionJob{}
	query := `
		SELECT id, job_type, status, input_entity_id, processed_count,
			   matched_count, error_count, started_at, completed_at,
			   error_message, configuration, results, created_at, updated_at
		FROM resolution_jobs
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&job.InputEntityID,
		&job.ProcessedCount,
		&job.MatchedCount,
		&job.ErrorCount,
		&job.StartedAt,
		&job.CompletedAt,
		&job.ErrorMessage,
		&job.Configuration,
		&job.Results,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("resolution job not found")
		}
		return nil, fmt.Errorf("failed to get resolution job: %w", err)
	}

	return job, nil
}