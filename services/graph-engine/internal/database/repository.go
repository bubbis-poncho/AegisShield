package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/aegisshield/graph-engine/internal/config"
)

// Connection wraps the database connection
type Connection struct {
	db     *sql.DB
	logger *slog.Logger
}

// Repository provides database operations for graph engine
type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

// AnalysisJob represents a graph analysis job
type AnalysisJob struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Status           string                 `json:"status"`
	Parameters       map[string]interface{} `json:"parameters"`
	Results          map[string]interface{} `json:"results,omitempty"`
	Progress         int                    `json:"progress"`
	Total            int                    `json:"total"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	Error            string                 `json:"error,omitempty"`
	CreatedBy        string                 `json:"created_by,omitempty"`
}

// Investigation represents an investigation case
type Investigation struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Status      string                 `json:"status"`
	Priority    string                 `json:"priority"`
	Entities    []string               `json:"entities"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
}

// Pattern represents a detected pattern in the graph
type Pattern struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Entities    []string               `json:"entities"`
	Confidence  float64                `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	DetectedAt  time.Time              `json:"detected_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NetworkMetrics represents network analysis metrics
type NetworkMetrics struct {
	ID                string                 `json:"id"`
	EntityID          string                 `json:"entity_id"`
	DegreeCentrality  float64                `json:"degree_centrality"`
	BetweennessCentrality float64            `json:"betweenness_centrality"`
	ClosenessCentrality   float64            `json:"closeness_centrality"`
	EigenvectorCentrality float64            `json:"eigenvector_centrality"`
	PageRank          float64                `json:"page_rank"`
	ClusteringCoeff   float64                `json:"clustering_coefficient"`
	CommunityID       string                 `json:"community_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CalculatedAt      time.Time              `json:"calculated_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// NewConnection creates a new database connection
func NewConnection(cfg config.DatabaseConfig, logger *slog.Logger) (*Connection, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
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

	logger.Info("Connected to database successfully")

	return &Connection{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	return c.db.Close()
}

// RunMigrations runs database migrations
func RunMigrations(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// NewRepository creates a new repository
func NewRepository(conn *Connection, logger *slog.Logger) *Repository {
	return &Repository{
		db:     conn.db,
		logger: logger,
	}
}

// Analysis Job Operations

// CreateAnalysisJob creates a new analysis job
func (r *Repository) CreateAnalysisJob(ctx context.Context, job *AnalysisJob) error {
	query := `
		INSERT INTO analysis_jobs (id, type, status, parameters, progress, total, started_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Type, job.Status, job.Parameters,
		job.Progress, job.Total, job.StartedAt, job.CreatedBy)

	if err != nil {
		return fmt.Errorf("failed to create analysis job: %w", err)
	}

	r.logger.Info("Analysis job created", "job_id", job.ID, "type", job.Type)
	return nil
}

// UpdateAnalysisJob updates an analysis job
func (r *Repository) UpdateAnalysisJob(ctx context.Context, job *AnalysisJob) error {
	query := `
		UPDATE analysis_jobs 
		SET status = $2, results = $3, progress = $4, completed_at = $5, error = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Status, job.Results, job.Progress, job.CompletedAt, job.Error)

	if err != nil {
		return fmt.Errorf("failed to update analysis job: %w", err)
	}

	return nil
}

// GetAnalysisJob retrieves an analysis job by ID
func (r *Repository) GetAnalysisJob(ctx context.Context, jobID string) (*AnalysisJob, error) {
	query := `
		SELECT id, type, status, parameters, results, progress, total, 
			   started_at, completed_at, error, created_by, created_at, updated_at
		FROM analysis_jobs 
		WHERE id = $1
	`

	var job AnalysisJob
	var results, parameters interface{}
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.Type, &job.Status, &parameters, &results,
		&job.Progress, &job.Total, &job.StartedAt, &completedAt,
		&job.Error, &job.CreatedBy, &job.StartedAt, &job.StartedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("analysis job not found")
		}
		return nil, fmt.Errorf("failed to get analysis job: %w", err)
	}

	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	// Handle JSON fields
	if parameters != nil {
		if params, ok := parameters.(map[string]interface{}); ok {
			job.Parameters = params
		}
	}

	if results != nil {
		if res, ok := results.(map[string]interface{}); ok {
			job.Results = res
		}
	}

	return &job, nil
}

// ListAnalysisJobs lists analysis jobs with optional filtering
func (r *Repository) ListAnalysisJobs(ctx context.Context, status string, limit, offset int) ([]*AnalysisJob, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, type, status, parameters, results, progress, total,
				   started_at, completed_at, error, created_by
			FROM analysis_jobs 
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{status, limit, offset}
	} else {
		query = `
			SELECT id, type, status, parameters, results, progress, total,
				   started_at, completed_at, error, created_by
			FROM analysis_jobs 
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list analysis jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*AnalysisJob
	for rows.Next() {
		var job AnalysisJob
		var results, parameters interface{}
		var completedAt sql.NullTime

		err := rows.Scan(
			&job.ID, &job.Type, &job.Status, &parameters, &results,
			&job.Progress, &job.Total, &job.StartedAt, &completedAt,
			&job.Error, &job.CreatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan analysis job: %w", err)
		}

		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		// Handle JSON fields
		if parameters != nil {
			if params, ok := parameters.(map[string]interface{}); ok {
				job.Parameters = params
			}
		}

		if results != nil {
			if res, ok := results.(map[string]interface{}); ok {
				job.Results = res
			}
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// Investigation Operations

// CreateInvestigation creates a new investigation
func (r *Repository) CreateInvestigation(ctx context.Context, investigation *Investigation) error {
	query := `
		INSERT INTO investigations (id, name, description, status, priority, entities, metadata, created_by, assigned_to)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		investigation.ID, investigation.Name, investigation.Description,
		investigation.Status, investigation.Priority, investigation.Entities,
		investigation.Metadata, investigation.CreatedBy, investigation.AssignedTo)

	if err != nil {
		return fmt.Errorf("failed to create investigation: %w", err)
	}

	r.logger.Info("Investigation created", "investigation_id", investigation.ID, "name", investigation.Name)
	return nil
}

// UpdateInvestigation updates an investigation
func (r *Repository) UpdateInvestigation(ctx context.Context, investigation *Investigation) error {
	query := `
		UPDATE investigations 
		SET name = $2, description = $3, status = $4, priority = $5, 
			entities = $6, metadata = $7, assigned_to = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		investigation.ID, investigation.Name, investigation.Description,
		investigation.Status, investigation.Priority, investigation.Entities,
		investigation.Metadata, investigation.AssignedTo)

	if err != nil {
		return fmt.Errorf("failed to update investigation: %w", err)
	}

	return nil
}

// GetInvestigation retrieves an investigation by ID
func (r *Repository) GetInvestigation(ctx context.Context, investigationID string) (*Investigation, error) {
	query := `
		SELECT id, name, description, status, priority, entities, metadata,
			   created_at, updated_at, created_by, assigned_to
		FROM investigations 
		WHERE id = $1
	`

	var investigation Investigation
	var metadata interface{}
	var assignedTo sql.NullString

	err := r.db.QueryRowContext(ctx, query, investigationID).Scan(
		&investigation.ID, &investigation.Name, &investigation.Description,
		&investigation.Status, &investigation.Priority, &investigation.Entities,
		&metadata, &investigation.CreatedAt, &investigation.UpdatedAt,
		&investigation.CreatedBy, &assignedTo,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("investigation not found")
		}
		return nil, fmt.Errorf("failed to get investigation: %w", err)
	}

	if assignedTo.Valid {
		investigation.AssignedTo = assignedTo.String
	}

	// Handle JSON metadata
	if metadata != nil {
		if meta, ok := metadata.(map[string]interface{}); ok {
			investigation.Metadata = meta
		}
	}

	return &investigation, nil
}

// Pattern Operations

// CreatePattern creates a detected pattern
func (r *Repository) CreatePattern(ctx context.Context, pattern *Pattern) error {
	query := `
		INSERT INTO patterns (id, type, name, description, entities, confidence, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		pattern.ID, pattern.Type, pattern.Name, pattern.Description,
		pattern.Entities, pattern.Confidence, pattern.Metadata)

	if err != nil {
		return fmt.Errorf("failed to create pattern: %w", err)
	}

	r.logger.Info("Pattern created", "pattern_id", pattern.ID, "type", pattern.Type)
	return nil
}

// GetPattern retrieves a pattern by ID
func (r *Repository) GetPattern(ctx context.Context, patternID string) (*Pattern, error) {
	query := `
		SELECT id, type, name, description, entities, confidence, metadata,
			   detected_at, updated_at
		FROM patterns 
		WHERE id = $1
	`

	var pattern Pattern
	var metadata interface{}

	err := r.db.QueryRowContext(ctx, query, patternID).Scan(
		&pattern.ID, &pattern.Type, &pattern.Name, &pattern.Description,
		&pattern.Entities, &pattern.Confidence, &metadata,
		&pattern.DetectedAt, &pattern.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pattern not found")
		}
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}

	// Handle JSON metadata
	if metadata != nil {
		if meta, ok := metadata.(map[string]interface{}); ok {
			pattern.Metadata = meta
		}
	}

	return &pattern, nil
}

// Network Metrics Operations

// CreateNetworkMetrics creates network metrics for an entity
func (r *Repository) CreateNetworkMetrics(ctx context.Context, metrics *NetworkMetrics) error {
	query := `
		INSERT INTO network_metrics (id, entity_id, degree_centrality, betweenness_centrality,
			closeness_centrality, eigenvector_centrality, page_rank, clustering_coefficient,
			community_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (entity_id) DO UPDATE SET
			degree_centrality = EXCLUDED.degree_centrality,
			betweenness_centrality = EXCLUDED.betweenness_centrality,
			closeness_centrality = EXCLUDED.closeness_centrality,
			eigenvector_centrality = EXCLUDED.eigenvector_centrality,
			page_rank = EXCLUDED.page_rank,
			clustering_coefficient = EXCLUDED.clustering_coefficient,
			community_id = EXCLUDED.community_id,
			metadata = EXCLUDED.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query,
		metrics.ID, metrics.EntityID, metrics.DegreeCentrality,
		metrics.BetweennessCentrality, metrics.ClosenessCentrality,
		metrics.EigenvectorCentrality, metrics.PageRank,
		metrics.ClusteringCoeff, metrics.CommunityID, metrics.Metadata)

	if err != nil {
		return fmt.Errorf("failed to create network metrics: %w", err)
	}

	return nil
}

// GetNetworkMetrics retrieves network metrics for an entity
func (r *Repository) GetNetworkMetrics(ctx context.Context, entityID string) (*NetworkMetrics, error) {
	query := `
		SELECT id, entity_id, degree_centrality, betweenness_centrality,
			   closeness_centrality, eigenvector_centrality, page_rank,
			   clustering_coefficient, community_id, metadata,
			   calculated_at, updated_at
		FROM network_metrics 
		WHERE entity_id = $1
	`

	var metrics NetworkMetrics
	var metadata interface{}
	var communityID sql.NullString

	err := r.db.QueryRowContext(ctx, query, entityID).Scan(
		&metrics.ID, &metrics.EntityID, &metrics.DegreeCentrality,
		&metrics.BetweennessCentrality, &metrics.ClosenessCentrality,
		&metrics.EigenvectorCentrality, &metrics.PageRank,
		&metrics.ClusteringCoeff, &communityID, &metadata,
		&metrics.CalculatedAt, &metrics.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("network metrics not found")
		}
		return nil, fmt.Errorf("failed to get network metrics: %w", err)
	}

	if communityID.Valid {
		metrics.CommunityID = communityID.String
	}

	// Handle JSON metadata
	if metadata != nil {
		if meta, ok := metadata.(map[string]interface{}); ok {
			metrics.Metadata = meta
		}
	}

	return &metrics, nil
}