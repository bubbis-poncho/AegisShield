package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"aegisshield/services/data-ingestion/internal/config"
)

// NewConnection creates a new database connection
func NewConnection(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
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
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// FileUploadRepository handles file upload data persistence
type FileUploadRepository struct {
	db *sql.DB
}

func NewFileUploadRepository(db *sql.DB) *FileUploadRepository {
	return &FileUploadRepository{db: db}
}

type FileUpload struct {
	ID           string    `db:"id"`
	FileName     string    `db:"file_name"`
	FileType     string    `db:"file_type"`
	FileSize     int64     `db:"file_size"`
	StoragePath  string    `db:"storage_path"`
	Status       string    `db:"status"`
	UploadedBy   string    `db:"uploaded_by"`
	UploadedAt   time.Time `db:"uploaded_at"`
	ProcessedAt  *time.Time `db:"processed_at"`
	ErrorMessage *string   `db:"error_message"`
	Metadata     map[string]string `db:"metadata"`
}

func (r *FileUploadRepository) Create(upload *FileUpload) error {
	query := `
		INSERT INTO file_uploads (
			id, file_name, file_type, file_size, storage_path, 
			status, uploaded_by, uploaded_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	metadataJSON, _ := json.Marshal(upload.Metadata)

	_, err := r.db.Exec(query,
		upload.ID, upload.FileName, upload.FileType, upload.FileSize,
		upload.StoragePath, upload.Status, upload.UploadedBy,
		upload.UploadedAt, metadataJSON,
	)

	return err
}

func (r *FileUploadRepository) GetByID(id string) (*FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, file_size, storage_path,
			   status, uploaded_by, uploaded_at, processed_at, 
			   error_message, metadata
		FROM file_uploads WHERE id = $1`

	upload := &FileUpload{}
	var metadataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&upload.ID, &upload.FileName, &upload.FileType, &upload.FileSize,
		&upload.StoragePath, &upload.Status, &upload.UploadedBy,
		&upload.UploadedAt, &upload.ProcessedAt, &upload.ErrorMessage,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &upload.Metadata)
	}

	return upload, nil
}

func (r *FileUploadRepository) UpdateStatus(id, status string, errorMessage *string) error {
	query := `
		UPDATE file_uploads 
		SET status = $2, error_message = $3, processed_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := r.db.Exec(query, id, status, errorMessage)
	return err
}

// DataJobRepository handles data processing job persistence
type DataJobRepository struct {
	db *sql.DB
}

func NewDataJobRepository(db *sql.DB) *DataJobRepository {
	return &DataJobRepository{db: db}
}

type DataJob struct {
	ID               string    `db:"id"`
	FileUploadID     string    `db:"file_upload_id"`
	JobType          string    `db:"job_type"`
	Status           string    `db:"status"`
	Progress         float64   `db:"progress"`
	TotalRecords     int       `db:"total_records"`
	ProcessedRecords int       `db:"processed_records"`
	FailedRecords    int       `db:"failed_records"`
	StartedAt        time.Time `db:"started_at"`
	CompletedAt      *time.Time `db:"completed_at"`
	ErrorMessage     *string   `db:"error_message"`
	CreatedBy        string    `db:"created_by"`
	Metadata         map[string]string `db:"metadata"`
}

func (r *DataJobRepository) Create(job *DataJob) error {
	query := `
		INSERT INTO data_jobs (
			id, file_upload_id, job_type, status, progress,
			total_records, processed_records, failed_records,
			started_at, created_by, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	metadataJSON, _ := json.Marshal(job.Metadata)

	_, err := r.db.Exec(query,
		job.ID, job.FileUploadID, job.JobType, job.Status, job.Progress,
		job.TotalRecords, job.ProcessedRecords, job.FailedRecords,
		job.StartedAt, job.CreatedBy, metadataJSON,
	)

	return err
}

func (r *DataJobRepository) GetByID(id string) (*DataJob, error) {
	query := `
		SELECT id, file_upload_id, job_type, status, progress,
			   total_records, processed_records, failed_records,
			   started_at, completed_at, error_message, created_by, metadata
		FROM data_jobs WHERE id = $1`

	job := &DataJob{}
	var metadataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&job.ID, &job.FileUploadID, &job.JobType, &job.Status, &job.Progress,
		&job.TotalRecords, &job.ProcessedRecords, &job.FailedRecords,
		&job.StartedAt, &job.CompletedAt, &job.ErrorMessage,
		&job.CreatedBy, &metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &job.Metadata)
	}

	return job, nil
}

func (r *DataJobRepository) UpdateProgress(id string, progress float64, processedRecords, failedRecords int) error {
	query := `
		UPDATE data_jobs 
		SET progress = $2, processed_records = $3, failed_records = $4
		WHERE id = $1`

	_, err := r.db.Exec(query, id, progress, processedRecords, failedRecords)
	return err
}

func (r *DataJobRepository) Complete(id string, status string, errorMessage *string) error {
	query := `
		UPDATE data_jobs 
		SET status = $2, completed_at = CURRENT_TIMESTAMP, error_message = $3
		WHERE id = $1`

	_, err := r.db.Exec(query, id, status, errorMessage)
	return err
}

// TransactionRepository handles transaction data persistence
type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

type Transaction struct {
	ID              string     `db:"id"`
	ExternalID      string     `db:"external_id"`
	Type            string     `db:"type"`
	Status          string     `db:"status"`
	Amount          float64    `db:"amount"`
	Currency        string     `db:"currency"`
	Description     string     `db:"description"`
	FromEntity      string     `db:"from_entity"`
	ToEntity        string     `db:"to_entity"`
	FromAccount     string     `db:"from_account"`
	ToAccount       string     `db:"to_account"`
	PaymentMethod   string     `db:"payment_method"`
	ProcessedAt     *time.Time `db:"processed_at"`
	RiskLevel       string     `db:"risk_level"`
	RiskScore       float64    `db:"risk_score"`
	SourceSystem    string     `db:"source_system"`
	BatchID         string     `db:"batch_id"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	Metadata        map[string]string `db:"metadata"`
}

func (r *TransactionRepository) CreateBatch(transactions []*Transaction) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO transactions (
			id, external_id, type, status, amount, currency, description,
			from_entity, to_entity, from_account, to_account, payment_method,
			processed_at, risk_level, risk_score, source_system, batch_id,
			created_at, updated_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`)
	
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, transaction := range transactions {
		metadataJSON, _ := json.Marshal(transaction.Metadata)

		_, err = stmt.Exec(
			transaction.ID, transaction.ExternalID, transaction.Type, transaction.Status,
			transaction.Amount, transaction.Currency, transaction.Description,
			transaction.FromEntity, transaction.ToEntity, transaction.FromAccount,
			transaction.ToAccount, transaction.PaymentMethod, transaction.ProcessedAt,
			transaction.RiskLevel, transaction.RiskScore, transaction.SourceSystem,
			transaction.BatchID, transaction.CreatedAt, transaction.UpdatedAt,
			metadataJSON,
		)
		
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TransactionRepository) GetByBatchID(batchID string) ([]*Transaction, error) {
	query := `
		SELECT id, external_id, type, status, amount, currency, description,
			   from_entity, to_entity, from_account, to_account, payment_method,
			   processed_at, risk_level, risk_score, source_system, batch_id,
			   created_at, updated_at, metadata
		FROM transactions WHERE batch_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(query, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*Transaction
	for rows.Next() {
		transaction := &Transaction{}
		var metadataJSON []byte

		err := rows.Scan(
			&transaction.ID, &transaction.ExternalID, &transaction.Type, &transaction.Status,
			&transaction.Amount, &transaction.Currency, &transaction.Description,
			&transaction.FromEntity, &transaction.ToEntity, &transaction.FromAccount,
			&transaction.ToAccount, &transaction.PaymentMethod, &transaction.ProcessedAt,
			&transaction.RiskLevel, &transaction.RiskScore, &transaction.SourceSystem,
			&transaction.BatchID, &transaction.CreatedAt, &transaction.UpdatedAt,
			&metadataJSON,
		)

		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &transaction.Metadata)
		}

		transactions = append(transactions, transaction)
	}

	return transactions, rows.Err()
}

// ValidationRepository handles validation error persistence
type ValidationRepository struct {
	db *sql.DB
}

func NewValidationRepository(db *sql.DB) *ValidationRepository {
	return &ValidationRepository{db: db}
}

type ValidationError struct {
	ID        string    `db:"id"`
	JobID     string    `db:"job_id"`
	RecordID  string    `db:"record_id"`
	Field     string    `db:"field"`
	ErrorCode string    `db:"error_code"`
	Message   string    `db:"message"`
	Value     *string   `db:"value"`
	Severity  string    `db:"severity"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *ValidationRepository) CreateBatch(errors []*ValidationError) error {
	if len(errors) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO validation_errors (
			id, job_id, record_id, field, error_code, message, value, severity, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, valErr := range errors {
		_, err = stmt.Exec(
			valErr.ID, valErr.JobID, valErr.RecordID, valErr.Field,
			valErr.ErrorCode, valErr.Message, valErr.Value, valErr.Severity,
			valErr.CreatedAt,
		)
		
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *ValidationRepository) GetByJobID(jobID string) ([]*ValidationError, error) {
	query := `
		SELECT id, job_id, record_id, field, error_code, message, value, severity, created_at
		FROM validation_errors 
		WHERE job_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var validationErrors []*ValidationError
	for rows.Next() {
		valErr := &ValidationError{}

		err := rows.Scan(
			&valErr.ID, &valErr.JobID, &valErr.RecordID, &valErr.Field,
			&valErr.ErrorCode, &valErr.Message, &valErr.Value, &valErr.Severity,
			&valErr.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		validationErrors = append(validationErrors, valErr)
	}

	return validationErrors, rows.Err()
}