package server

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"aegisshield/services/data-ingestion/internal/config"
	"aegisshield/services/data-ingestion/internal/database"
	"aegisshield/services/data-ingestion/internal/kafka"
	"aegisshield/services/data-ingestion/internal/metrics"
	"aegisshield/services/data-ingestion/internal/processor"
	"aegisshield/services/data-ingestion/internal/storage"
	"aegisshield/services/data-ingestion/internal/validator"
	pb "aegisshield/shared/proto/data-ingestion"
	shared "aegisshield/shared/proto/shared"
	"aegisshield/shared/utils"
)

// Repositories contains all data access objects
type Repositories struct {
	FileUpload  *database.FileUploadRepository
	DataJob     *database.DataJobRepository
	Transaction *database.TransactionRepository
	Validation  *database.ValidationRepository
}

// Services contains all external service dependencies
type Services struct {
	Storage storage.Service
	Kafka   kafka.Producer
	Metrics *metrics.Collector
	Logger  *logrus.Logger
}

// DataIngestionServer implements the DataIngestionService gRPC service
type DataIngestionServer struct {
	pb.UnimplementedDataIngestionServiceServer
	repos    *Repositories
	services *Services
	config   *config.Config
}

// NewDataIngestionServer creates a new DataIngestionServer
func NewDataIngestionServer(repos *Repositories, services *Services, cfg *config.Config) *DataIngestionServer {
	return &DataIngestionServer{
		repos:    repos,
		services: services,
		config:   cfg,
	}
}

// UploadFile handles single file upload
func (s *DataIngestionServer) UploadFile(ctx context.Context, req *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	start := time.Now()
	s.services.Metrics.IncrementCounter("upload_file_requests_total")

	// Validate request
	if err := s.validateUploadRequest(req); err != nil {
		s.services.Metrics.IncrementCounter("upload_file_errors_total")
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Generate file ID
	fileID := uuid.New().String()

	// Create file upload record
	upload := &database.FileUpload{
		ID:         fileID,
		FileName:   req.FileName,
		FileType:   req.FileType,
		FileSize:   int64(len(req.FileData)),
		Status:     "uploading",
		UploadedBy: req.UploadedBy,
		UploadedAt: time.Now(),
		Metadata:   req.Metadata,
	}

	// Store file
	storagePath, err := s.services.Storage.Store(ctx, fileID, req.FileName, req.FileData)
	if err != nil {
		s.services.Logger.WithError(err).Error("Failed to store file")
		s.services.Metrics.IncrementCounter("upload_file_errors_total")
		return nil, status.Errorf(codes.Internal, "failed to store file: %v", err)
	}

	upload.StoragePath = storagePath

	// Save upload record to database
	if err := s.repos.FileUpload.Create(upload); err != nil {
		s.services.Logger.WithError(err).Error("Failed to create file upload record")
		s.services.Metrics.IncrementCounter("upload_file_errors_total")
		return nil, status.Errorf(codes.Internal, "failed to save upload record: %v", err)
	}

	// Update status to uploaded
	if err := s.repos.FileUpload.UpdateStatus(fileID, "uploaded", nil); err != nil {
		s.services.Logger.WithError(err).Error("Failed to update upload status")
	}

	// Publish file upload event
	if err := s.publishFileUploadEvent(fileID, req); err != nil {
		s.services.Logger.WithError(err).Error("Failed to publish file upload event")
	}

	// Record metrics
	s.services.Metrics.RecordHistogram("upload_file_duration_seconds", time.Since(start).Seconds())
	s.services.Metrics.RecordHistogram("uploaded_file_size_bytes", float64(len(req.FileData)))

	return &pb.UploadFileResponse{
		FileId:    fileID,
		Success:   true,
		Message:   "File uploaded successfully",
		UploadUrl: storagePath,
	}, nil
}

// UploadFileStream handles streaming file upload
func (s *DataIngestionServer) UploadFileStream(stream pb.DataIngestionService_UploadFileStreamServer) error {
	start := time.Now()
	s.services.Metrics.IncrementCounter("upload_file_stream_requests_total")

	var fileID string
	var fileName string
	var fileType string
	var uploadedBy string
	var metadata map[string]string
	var fileData []byte

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.services.Logger.WithError(err).Error("Failed to receive stream chunk")
			s.services.Metrics.IncrementCounter("upload_file_stream_errors_total")
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		if chunk.GetMetadata() != nil {
			// First chunk with metadata
			meta := chunk.GetMetadata()
			fileID = uuid.New().String()
			fileName = meta.FileName
			fileType = meta.FileType
			uploadedBy = meta.UploadedBy
			metadata = meta.Metadata
		}

		if chunk.GetChunk() != nil {
			// Data chunk
			fileData = append(fileData, chunk.GetChunk().Data...)
		}
	}

	// Validate complete upload
	if fileID == "" || fileName == "" {
		s.services.Metrics.IncrementCounter("upload_file_stream_errors_total")
		return status.Errorf(codes.InvalidArgument, "missing file metadata")
	}

	// Create file upload record
	upload := &database.FileUpload{
		ID:         fileID,
		FileName:   fileName,
		FileType:   fileType,
		FileSize:   int64(len(fileData)),
		Status:     "uploading",
		UploadedBy: uploadedBy,
		UploadedAt: time.Now(),
		Metadata:   metadata,
	}

	// Store file
	ctx := stream.Context()
	storagePath, err := s.services.Storage.Store(ctx, fileID, fileName, fileData)
	if err != nil {
		s.services.Logger.WithError(err).Error("Failed to store streamed file")
		s.services.Metrics.IncrementCounter("upload_file_stream_errors_total")
		return status.Errorf(codes.Internal, "failed to store file: %v", err)
	}

	upload.StoragePath = storagePath

	// Save upload record
	if err := s.repos.FileUpload.Create(upload); err != nil {
		s.services.Logger.WithError(err).Error("Failed to create file upload record")
		s.services.Metrics.IncrementCounter("upload_file_stream_errors_total")
		return status.Errorf(codes.Internal, "failed to save upload record: %v", err)
	}

	// Update status
	if err := s.repos.FileUpload.UpdateStatus(fileID, "uploaded", nil); err != nil {
		s.services.Logger.WithError(err).Error("Failed to update upload status")
	}

	// Publish event
	uploadReq := &pb.UploadFileRequest{
		FileName:   fileName,
		FileType:   fileType,
		FileData:   fileData,
		UploadedBy: uploadedBy,
		Metadata:   metadata,
	}
	if err := s.publishFileUploadEvent(fileID, uploadReq); err != nil {
		s.services.Logger.WithError(err).Error("Failed to publish file upload event")
	}

	// Record metrics
	s.services.Metrics.RecordHistogram("upload_file_stream_duration_seconds", time.Since(start).Seconds())
	s.services.Metrics.RecordHistogram("uploaded_file_stream_size_bytes", float64(len(fileData)))

	// Send response
	return stream.SendAndClose(&pb.UploadFileResponse{
		FileId:    fileID,
		Success:   true,
		Message:   "File uploaded successfully via stream",
		UploadUrl: storagePath,
	})
}

// ProcessTransactionStream handles streaming transaction processing
func (s *DataIngestionServer) ProcessTransactionStream(stream pb.DataIngestionService_ProcessTransactionStreamServer) error {
	start := time.Now()
	s.services.Metrics.IncrementCounter("process_transaction_stream_requests_total")

	batchID := uuid.New().String()
	var transactions []*shared.Transaction
	processedCount := 0
	errorCount := 0

	// Create processing job
	job := &database.DataJob{
		ID:               uuid.New().String(),
		JobType:          "transaction_stream",
		Status:           "processing",
		Progress:         0.0,
		StartedAt:        time.Now(),
		CreatedBy:        "system",
		Metadata:         map[string]string{"batch_id": batchID},
	}

	if err := s.repos.DataJob.Create(job); err != nil {
		s.services.Logger.WithError(err).Error("Failed to create processing job")
		return status.Errorf(codes.Internal, "failed to create job: %v", err)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.services.Logger.WithError(err).Error("Failed to receive transaction stream")
			s.services.Metrics.IncrementCounter("process_transaction_stream_errors_total")
			return status.Errorf(codes.Internal, "failed to receive transaction: %v", err)
		}

		// Validate transaction
		if err := s.validateTransaction(req.Transaction); err != nil {
			errorCount++
			s.services.Logger.WithError(err).WithField("transaction_id", req.Transaction.Id).Error("Invalid transaction")
			
			// Send error response
			if err := stream.Send(&pb.ProcessTransactionResponse{
				TransactionId: req.Transaction.Id,
				Success:       false,
				Message:       fmt.Sprintf("validation failed: %v", err),
				ValidationErrors: []*shared.Error{{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				}},
			}); err != nil {
				s.services.Logger.WithError(err).Error("Failed to send error response")
			}
			continue
		}

		// Process transaction
		processedTxn, err := s.processTransaction(stream.Context(), req.Transaction, batchID)
		if err != nil {
			errorCount++
			s.services.Logger.WithError(err).WithField("transaction_id", req.Transaction.Id).Error("Failed to process transaction")
			
			// Send error response
			if err := stream.Send(&pb.ProcessTransactionResponse{
				TransactionId: req.Transaction.Id,
				Success:       false,
				Message:       fmt.Sprintf("processing failed: %v", err),
			}); err != nil {
				s.services.Logger.WithError(err).Error("Failed to send error response")
			}
			continue
		}

		transactions = append(transactions, processedTxn)
		processedCount++

		// Send success response
		if err := stream.Send(&pb.ProcessTransactionResponse{
			TransactionId: req.Transaction.Id,
			Success:       true,
			Message:       "Transaction processed successfully",
		}); err != nil {
			s.services.Logger.WithError(err).Error("Failed to send success response")
		}

		// Update job progress
		totalCount := processedCount + errorCount
		progress := float64(processedCount) / float64(totalCount) * 100
		s.repos.DataJob.UpdateProgress(job.ID, progress, processedCount, errorCount)
	}

	// Store transactions in batch
	if len(transactions) > 0 {
		dbTransactions := make([]*database.Transaction, len(transactions))
		for i, txn := range transactions {
			dbTransactions[i] = s.convertToDBTransaction(txn, batchID)
		}

		if err := s.repos.Transaction.CreateBatch(dbTransactions); err != nil {
			s.services.Logger.WithError(err).Error("Failed to store transaction batch")
			s.services.Metrics.IncrementCounter("process_transaction_stream_errors_total")
			return status.Errorf(codes.Internal, "failed to store transactions: %v", err)
		}
	}

	// Complete job
	var errorMessage *string
	status := "completed"
	if errorCount > 0 {
		status = "completed_with_errors"
		msg := fmt.Sprintf("Processed %d transactions with %d errors", processedCount, errorCount)
		errorMessage = &msg
	}

	if err := s.repos.DataJob.Complete(job.ID, status, errorMessage); err != nil {
		s.services.Logger.WithError(err).Error("Failed to complete job")
	}

	// Record metrics
	s.services.Metrics.RecordHistogram("process_transaction_stream_duration_seconds", time.Since(start).Seconds())
	s.services.Metrics.RecordGauge("processed_transactions_total", float64(processedCount))
	s.services.Metrics.RecordGauge("failed_transactions_total", float64(errorCount))

	return nil
}

// GetJobStatus returns the status of a processing job
func (s *DataIngestionServer) GetJobStatus(ctx context.Context, req *pb.GetJobStatusRequest) (*pb.GetJobStatusResponse, error) {
	job, err := s.repos.DataJob.GetByID(req.JobId)
	if err != nil {
		s.services.Logger.WithError(err).Error("Failed to get job status")
		return nil, status.Errorf(codes.Internal, "failed to get job: %v", err)
	}

	if job == nil {
		return nil, status.Errorf(codes.NotFound, "job not found")
	}

	response := &pb.GetJobStatusResponse{
		JobId:            job.ID,
		Status:           convertJobStatus(job.Status),
		Progress:         job.Progress,
		TotalRecords:     int32(job.TotalRecords),
		ProcessedRecords: int32(job.ProcessedRecords),
		FailedRecords:    int32(job.FailedRecords),
		StartedAt:        timestamppb.New(job.StartedAt),
		Metadata:         job.Metadata,
	}

	if job.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	if job.ErrorMessage != nil {
		response.ErrorMessage = *job.ErrorMessage
	}

	return response, nil
}

// ValidateData validates data without processing
func (s *DataIngestionServer) ValidateData(ctx context.Context, req *pb.ValidateDataRequest) (*pb.ValidateDataResponse, error) {
	start := time.Now()
	s.services.Metrics.IncrementCounter("validate_data_requests_total")

	validator := validator.NewValidator()
	var allErrors []*shared.Error

	for _, transaction := range req.Transactions {
		if err := s.validateTransaction(transaction); err != nil {
			allErrors = append(allErrors, &shared.Error{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
				Context: map[string]string{
					"transaction_id": transaction.Id,
					"field":         "transaction",
				},
			})
		}
	}

	// Additional data quality checks
	qualityErrors := validator.ValidateDataQuality(req.Transactions)
	allErrors = append(allErrors, qualityErrors...)

	isValid := len(allErrors) == 0
	s.services.Metrics.RecordHistogram("validate_data_duration_seconds", time.Since(start).Seconds())

	return &pb.ValidateDataResponse{
		IsValid:          isValid,
		ValidationErrors: allErrors,
		Summary: &pb.ValidationSummary{
			TotalRecords:   int32(len(req.Transactions)),
			ValidRecords:   int32(len(req.Transactions) - len(allErrors)),
			InvalidRecords: int32(len(allErrors)),
		},
	}, nil
}

// HealthCheck performs health check
func (s *DataIngestionServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:    pb.HealthStatus_HEALTHY,
		Message:   "Data Ingestion Service is healthy",
		Timestamp: timestamppb.Now(),
	}, nil
}

// Helper methods

func (s *DataIngestionServer) validateUploadRequest(req *pb.UploadFileRequest) error {
	if req.FileName == "" {
		return fmt.Errorf("file name is required")
	}
	if req.FileType == "" {
		return fmt.Errorf("file type is required")
	}
	if len(req.FileData) == 0 {
		return fmt.Errorf("file data is required")
	}
	if int64(len(req.FileData)) > s.config.Server.MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size")
	}
	if req.UploadedBy == "" {
		return fmt.Errorf("uploaded by is required")
	}
	return nil
}

func (s *DataIngestionServer) validateTransaction(txn *shared.Transaction) error {
	if txn.Id == "" {
		return fmt.Errorf("transaction ID is required")
	}
	if txn.Amount <= 0 {
		return fmt.Errorf("transaction amount must be positive")
	}
	if !utils.IsValidCurrencyCode(txn.Currency) {
		return fmt.Errorf("invalid currency code: %s", txn.Currency)
	}
	if txn.FromEntity == "" {
		return fmt.Errorf("from entity is required")
	}
	if txn.ToEntity == "" {
		return fmt.Errorf("to entity is required")
	}
	return nil
}

func (s *DataIngestionServer) processTransaction(ctx context.Context, txn *shared.Transaction, batchID string) (*shared.Transaction, error) {
	// Initialize transaction processor
	proc := processor.NewTransactionProcessor(s.services.Logger)
	
	// Process and enrich transaction
	processedTxn, err := proc.Process(ctx, txn, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to process transaction: %w", err)
	}

	// Publish transaction event
	if err := s.publishTransactionEvent(processedTxn); err != nil {
		s.services.Logger.WithError(err).Error("Failed to publish transaction event")
	}

	return processedTxn, nil
}

func (s *DataIngestionServer) convertToDBTransaction(txn *shared.Transaction, batchID string) *database.Transaction {
	return &database.Transaction{
		ID:            txn.Id,
		ExternalID:    txn.ExternalId,
		Type:          txn.Type.String(),
		Status:        txn.Status.String(),
		Amount:        txn.Amount,
		Currency:      txn.Currency,
		Description:   txn.Description,
		FromEntity:    txn.FromEntity,
		ToEntity:      txn.ToEntity,
		FromAccount:   txn.FromAccount,
		ToAccount:     txn.ToAccount,
		PaymentMethod: txn.PaymentMethod.String(),
		RiskLevel:     txn.RiskLevel.String(),
		RiskScore:     txn.RiskScore,
		SourceSystem:  txn.SourceSystem,
		BatchID:       batchID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Metadata:      txn.Metadata,
	}
}

func (s *DataIngestionServer) publishFileUploadEvent(fileID string, req *pb.UploadFileRequest) error {
	event := map[string]interface{}{
		"event_type":  "file_upload",
		"file_id":     fileID,
		"file_name":   req.FileName,
		"file_type":   req.FileType,
		"file_size":   len(req.FileData),
		"uploaded_by": req.UploadedBy,
		"timestamp":   time.Now().UTC(),
		"metadata":    req.Metadata,
	}

	return s.services.Kafka.Publish(s.config.Kafka.Topics.FileUpload, fileID, event)
}

func (s *DataIngestionServer) publishTransactionEvent(txn *shared.Transaction) error {
	event := map[string]interface{}{
		"event_type":     "transaction_ingested",
		"transaction_id": txn.Id,
		"amount":         txn.Amount,
		"currency":       txn.Currency,
		"from_entity":    txn.FromEntity,
		"to_entity":      txn.ToEntity,
		"risk_level":     txn.RiskLevel.String(),
		"risk_score":     txn.RiskScore,
		"timestamp":      time.Now().UTC(),
	}

	return s.services.Kafka.Publish(s.config.Kafka.Topics.TransactionFlow, txn.Id, event)
}

func convertJobStatus(status string) pb.JobStatus {
	switch status {
	case "pending":
		return pb.JobStatus_PENDING
	case "processing":
		return pb.JobStatus_PROCESSING
	case "completed":
		return pb.JobStatus_COMPLETED
	case "failed":
		return pb.JobStatus_FAILED
	case "cancelled":
		return pb.JobStatus_CANCELLED
	default:
		return pb.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}