package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"investigation-toolkit/internal/config"
)

// Database represents the database connection and operations
type Database struct {
	db     *sqlx.DB
	logger *zap.Logger
	config *config.DatabaseConfig
}

// New creates a new database instance
func New(cfg *config.DatabaseConfig, logger *zap.Logger) (*Database, error) {
	if cfg == nil {
		return nil, errors.New("database config is required")
	}

	if logger == nil {
		return nil, errors.New("logger is required")
	}

	db := &Database{
		logger: logger.Named("database"),
		config: cfg,
	}

	if err := db.connect(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	return db, nil
}

// connect establishes database connection with proper configuration
func (d *Database) connect() error {
	d.logger.Info("Connecting to database", 
		zap.String("connection_string", d.maskConnectionString(d.config.ConnectionString)))

	// Open database connection
	db, err := sqlx.Connect("postgres", d.config.ConnectionString)
	if err != nil {
		return errors.Wrap(err, "failed to connect to postgres")
	}

	// Configure connection pool
	db.SetMaxOpenConns(d.config.MaxOpenConnections)
	db.SetMaxIdleConns(d.config.MaxIdleConnections)
	db.SetConnMaxLifetime(d.config.ConnectionLifetime)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), d.config.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return errors.Wrap(err, "failed to ping database")
	}

	d.db = db
	d.logger.Info("Successfully connected to database")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		d.logger.Info("Closing database connection")
		return d.db.Close()
	}
	return nil
}

// DB returns the underlying sqlx.DB instance
func (d *Database) DB() *sqlx.DB {
	return d.db
}

// Health checks the database health
func (d *Database) Health(ctx context.Context) error {
	if d.db == nil {
		return errors.New("database connection not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return d.db.PingContext(ctx)
}

// RunMigrations executes database migrations
func (d *Database) RunMigrations() error {
	d.logger.Info("Running database migrations", zap.String("path", d.config.MigrationPath))

	driver, err := postgres.WithInstance(d.db.DB, &postgres.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to create migration driver")
	}

	m, err := migrate.NewWithDatabaseInstance(d.config.MigrationPath, "postgres", driver)
	if err != nil {
		return errors.Wrap(err, "failed to create migration instance")
	}
	defer m.Close()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "failed to run migrations")
	}

	if err == migrate.ErrNoChange {
		d.logger.Info("No new migrations to apply")
	} else {
		d.logger.Info("Successfully applied database migrations")
	}

	return nil
}

// BeginTx starts a new transaction
func (d *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return d.db.BeginTxx(ctx, opts)
}

// ExecContext executes a query with context
func (d *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	result, err := d.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("EXEC", query, args, duration, err)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return result, err
}

// QueryContext executes a query that returns rows with context
func (d *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	rows, err := d.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("QUERY", query, args, duration, err)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return rows, err
}

// QueryRowContext executes a query that returns a single row with context
func (d *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	row := d.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("QUERY_ROW", query, args, duration, nil)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return row
}

// SelectContext executes a query and scans the result into dest
func (d *Database) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	err := d.db.SelectContext(ctx, dest, query, args...)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("SELECT", query, args, duration, err)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return err
}

// GetContext executes a query and scans the first row into dest
func (d *Database) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	err := d.db.GetContext(ctx, dest, query, args...)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("GET", query, args, duration, err)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return err
}

// NamedExecContext executes a named query with context
func (d *Database) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	result, err := d.db.NamedExecContext(ctx, query, arg)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("NAMED_EXEC", query, []interface{}{arg}, duration, err)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return result, err
}

// NamedQueryContext executes a named query with context
func (d *Database) NamedQueryContext(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	start := time.Now()
	rows, err := d.db.NamedQueryContext(ctx, query, arg)
	duration := time.Since(start)

	if d.config.EnableQueryLogging {
		d.logQuery("NAMED_QUERY", query, []interface{}{arg}, duration, err)
	}

	if duration > d.config.SlowQueryThreshold {
		d.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", d.config.SlowQueryThreshold))
	}

	return rows, err
}

// GetStats returns database connection statistics
func (d *Database) GetStats() sql.DBStats {
	if d.db == nil {
		return sql.DBStats{}
	}
	return d.db.Stats()
}

// logQuery logs database queries if logging is enabled
func (d *Database) logQuery(operation, query string, args []interface{}, duration time.Duration, err error) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("query", query),
		zap.Duration("duration", duration),
	}

	if len(args) > 0 {
		fields = append(fields, zap.Any("args", args))
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		d.logger.Error("Database query failed", fields...)
	} else {
		d.logger.Debug("Database query executed", fields...)
	}
}

// maskConnectionString masks sensitive information in connection string for logging
func (d *Database) maskConnectionString(connStr string) string {
	// Simple masking - replace password with asterisks
	// This is a basic implementation and could be improved
	masked := connStr
	
	// Look for password= pattern and replace the value
	if start := fmt.Sprintf("password="); start != "" {
		// This is a simplified implementation
		// In production, you might want to use a more robust regex-based approach
		return "postgresql://user:***@host:port/database?sslmode=disable"
	}
	
	return masked
}

// Repository represents a base repository with common database operations
type Repository struct {
	db     *Database
	logger *zap.Logger
}

// NewRepository creates a new repository instance
func NewRepository(db *Database, logger *zap.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger.Named("repository"),
	}
}

// DB returns the database instance
func (r *Repository) DB() *Database {
	return r.db
}

// WithTx executes a function within a database transaction
func (r *Repository) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.logger.Error("Failed to rollback transaction after panic",
					zap.Error(rollbackErr))
			}
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.logger.Error("Failed to rollback transaction",
				zap.Error(rollbackErr))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// Paginate provides pagination parameters for queries
type Paginate struct {
	Limit  int `json:"limit" validate:"min=1,max=1000"`
	Offset int `json:"offset" validate:"min=0"`
}

// NewPaginate creates a new pagination instance with defaults
func NewPaginate(limit, offset int) *Paginate {
	if limit <= 0 || limit > 1000 {
		limit = 50 // default limit
	}
	if offset < 0 {
		offset = 0
	}
	return &Paginate{
		Limit:  limit,
		Offset: offset,
	}
}

// PaginatedResult represents a paginated query result
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// NewPaginatedResult creates a new paginated result
func NewPaginatedResult(data interface{}, total int64, paginate *Paginate) *PaginatedResult {
	totalPages := int((total + int64(paginate.Limit) - 1) / int64(paginate.Limit))
	hasNext := paginate.Offset+paginate.Limit < int(total)
	hasPrev := paginate.Offset > 0

	return &PaginatedResult{
		Data:       data,
		Total:      total,
		Limit:      paginate.Limit,
		Offset:     paginate.Offset,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}
}