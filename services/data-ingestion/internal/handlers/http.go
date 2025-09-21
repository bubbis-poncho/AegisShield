package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/aegisshield/data-ingestion/internal/database"
	"github.com/aegisshield/data-ingestion/internal/metrics"
	"github.com/aegisshield/data-ingestion/internal/storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// HTTPHandlers holds HTTP route handlers
type HTTPHandlers struct {
	repository    *database.Repository
	storage       storage.Storage
	metrics       *metrics.Collector
	logger        *slog.Logger
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	FileName    string            `json:"file_name"`
	FileType    string            `json:"file_type"`
	FileSize    int64             `json:"file_size"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// FileUploadResponse represents a file upload response
type FileUploadResponse struct {
	FileID      string    `json:"file_id"`
	UploadURL   string    `json:"upload_url,omitempty"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// JobStatusResponse represents a job status response
type JobStatusResponse struct {
	JobID           string                 `json:"job_id"`
	Status          string                 `json:"status"`
	FileID          string                 `json:"file_id"`
	ProcessedCount  int                    `json:"processed_count"`
	ErrorCount      int                    `json:"error_count"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	Services    map[string]string `json:"services"`
	Uptime      string            `json:"uptime"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Code      string    `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

var startTime = time.Now()

// NewHTTPHandlers creates new HTTP handlers
func NewHTTPHandlers(
	repository *database.Repository,
	storage storage.Storage,
	metrics *metrics.Collector,
	logger *slog.Logger,
) *HTTPHandlers {
	return &HTTPHandlers{
		repository: repository,
		storage:    storage,
		metrics:    metrics,
		logger:     logger,
	}
}

// RegisterRoutes registers HTTP routes
func (h *HTTPHandlers) RegisterRoutes(router *mux.Router) {
	// File upload routes
	router.HandleFunc("/api/v1/files/upload", h.UploadFile).Methods("POST")
	router.HandleFunc("/api/v1/files/{file_id}", h.GetFileStatus).Methods("GET")
	router.HandleFunc("/api/v1/files/{file_id}/download", h.DownloadFile).Methods("GET")
	router.HandleFunc("/api/v1/files", h.ListFiles).Methods("GET")

	// Job management routes
	router.HandleFunc("/api/v1/jobs", h.ListJobs).Methods("GET")
	router.HandleFunc("/api/v1/jobs/{job_id}", h.GetJobStatus).Methods("GET")
	router.HandleFunc("/api/v1/jobs/{job_id}/cancel", h.CancelJob).Methods("POST")

	// Health and monitoring routes
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/health/ready", h.ReadinessCheck).Methods("GET")
	router.HandleFunc("/health/live", h.LivenessCheck).Methods("GET")

	// Metrics route (if not using separate metrics server)
	router.HandleFunc("/metrics", h.MetricsHandler).Methods("GET")
}

// UploadFile handles file upload
func (h *HTTPHandlers) UploadFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		h.metrics.RecordHistogram("upload_file_duration_seconds", time.Since(start).Seconds())
	}()

	h.metrics.IncrementCounter("upload_file_requests_total")

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		h.metrics.IncrementCounter("upload_file_errors_total")
		h.sendError(w, http.StatusBadRequest, "INVALID_FORM", "Failed to parse multipart form", err)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		h.metrics.IncrementCounter("upload_file_errors_total")
		h.sendError(w, http.StatusBadRequest, "MISSING_FILE", "File is required", err)
		return
	}
	defer file.Close()

	// Generate file ID
	fileID := uuid.New()

	// Get metadata from form
	metadata := make(map[string]string)
	if metadataStr := r.FormValue("metadata"); metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			h.logger.Warn("failed to parse metadata", "error", err)
		}
	}

	// Create file upload record
	fileUpload := &database.FileUpload{
		ID:          fileID,
		FileName:    header.Filename,
		FileSize:    header.Size,
		ContentType: header.Header.Get("Content-Type"),
		Status:      "uploading",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add metadata
	if len(metadata) > 0 {
		metadataJSON, _ := json.Marshal(metadata)
		fileUpload.Metadata = metadataJSON
	}

	// Store file upload record
	if err := h.repository.CreateFileUpload(r.Context(), fileUpload); err != nil {
		h.metrics.IncrementCounter("upload_file_errors_total")
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create file record", err)
		return
	}

	// Store file
	storagePath := fmt.Sprintf("uploads/%s/%s", time.Now().Format("2006/01/02"), fileID.String())
	if err := h.storage.Store(r.Context(), storagePath, file); err != nil {
		h.metrics.IncrementCounter("upload_file_errors_total")
		
		// Update file status to failed
		fileUpload.Status = "failed"
		fileUpload.ErrorMessage = err.Error()
		fileUpload.UpdatedAt = time.Now()
		h.repository.UpdateFileUpload(r.Context(), fileUpload)

		h.sendError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to store file", err)
		return
	}

	// Update file upload record
	fileUpload.Status = "uploaded"
	fileUpload.StoragePath = storagePath
	fileUpload.UploadedAt = &time.Time{}
	*fileUpload.UploadedAt = time.Now()
	fileUpload.UpdatedAt = time.Now()

	if err := h.repository.UpdateFileUpload(r.Context(), fileUpload); err != nil {
		h.logger.Error("failed to update file upload record", "file_id", fileID, "error", err)
		// Don't fail the request as file is already uploaded
	}

	h.metrics.RecordHistogram("uploaded_file_size_bytes", float64(header.Size))

	response := FileUploadResponse{
		FileID:     fileID.String(),
		Status:     "uploaded",
		Message:    "File uploaded successfully",
		UploadedAt: time.Now(),
	}

	h.sendJSON(w, http.StatusCreated, response)

	h.logger.Info("file uploaded successfully",
		"file_id", fileID,
		"file_name", header.Filename,
		"file_size", header.Size)
}

// GetFileStatus gets file upload status
func (h *HTTPHandlers) GetFileStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileIDStr := vars["file_id"]

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_FILE_ID", "Invalid file ID format", err)
		return
	}

	fileUpload, err := h.repository.GetFileUpload(r.Context(), fileID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "FILE_NOT_FOUND", "File not found", err)
		return
	}

	// Parse metadata
	var metadata map[string]interface{}
	if fileUpload.Metadata != nil {
		json.Unmarshal(fileUpload.Metadata, &metadata)
	}

	response := map[string]interface{}{
		"file_id":      fileUpload.ID.String(),
		"file_name":    fileUpload.FileName,
		"file_size":    fileUpload.FileSize,
		"content_type": fileUpload.ContentType,
		"status":       fileUpload.Status,
		"created_at":   fileUpload.CreatedAt,
		"updated_at":   fileUpload.UpdatedAt,
		"uploaded_at":  fileUpload.UploadedAt,
		"metadata":     metadata,
	}

	if fileUpload.ErrorMessage != "" {
		response["error_message"] = fileUpload.ErrorMessage
	}

	h.sendJSON(w, http.StatusOK, response)
}

// DownloadFile handles file download
func (h *HTTPHandlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileIDStr := vars["file_id"]

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_FILE_ID", "Invalid file ID format", err)
		return
	}

	fileUpload, err := h.repository.GetFileUpload(r.Context(), fileID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "FILE_NOT_FOUND", "File not found", err)
		return
	}

	if fileUpload.Status != "uploaded" {
		h.sendError(w, http.StatusBadRequest, "FILE_NOT_AVAILABLE", "File is not available for download", nil)
		return
	}

	// Get file from storage
	reader, err := h.storage.Get(r.Context(), fileUpload.StoragePath)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to retrieve file", err)
		return
	}
	defer reader.Close()

	// Set headers
	w.Header().Set("Content-Type", fileUpload.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileUpload.FileName))
	if fileUpload.FileSize > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(fileUpload.FileSize, 10))
	}

	// Copy file content to response
	_, err = io.Copy(w, reader)
	if err != nil {
		h.logger.Error("failed to stream file", "file_id", fileID, "error", err)
		return
	}

	h.logger.Info("file downloaded", "file_id", fileID, "file_name", fileUpload.FileName)
}

// ListFiles lists uploaded files
func (h *HTTPHandlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	status := query.Get("status")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	files, err := h.repository.ListFileUploads(r.Context(), limit, offset, status)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list files", err)
		return
	}

	response := make([]map[string]interface{}, len(files))
	for i, file := range files {
		var metadata map[string]interface{}
		if file.Metadata != nil {
			json.Unmarshal(file.Metadata, &metadata)
		}

		response[i] = map[string]interface{}{
			"file_id":      file.ID.String(),
			"file_name":    file.FileName,
			"file_size":    file.FileSize,
			"content_type": file.ContentType,
			"status":       file.Status,
			"created_at":   file.CreatedAt,
			"updated_at":   file.UpdatedAt,
			"uploaded_at":  file.UploadedAt,
			"metadata":     metadata,
		}

		if file.ErrorMessage != "" {
			response[i]["error_message"] = file.ErrorMessage
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"files":  response,
		"limit":  limit,
		"offset": offset,
		"count":  len(files),
	})
}

// ListJobs lists processing jobs
func (h *HTTPHandlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	status := query.Get("status")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	jobs, err := h.repository.ListDataJobs(r.Context(), limit, offset, status)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list jobs", err)
		return
	}

	response := make([]JobStatusResponse, len(jobs))
	for i, job := range jobs {
		var metadata map[string]interface{}
		if job.Metadata != nil {
			json.Unmarshal(job.Metadata, &metadata)
		}

		response[i] = JobStatusResponse{
			JobID:          job.ID.String(),
			Status:         job.Status,
			FileID:         job.FileID.String(),
			ProcessedCount: job.ProcessedCount,
			ErrorCount:     job.ErrorCount,
			CreatedAt:      job.CreatedAt,
			UpdatedAt:      job.UpdatedAt,
			CompletedAt:    job.CompletedAt,
			ErrorMessage:   job.ErrorMessage,
			Metadata:       metadata,
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":   response,
		"limit":  limit,
		"offset": offset,
		"count":  len(jobs),
	})
}

// GetJobStatus gets job status
func (h *HTTPHandlers) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JOB_ID", "Invalid job ID format", err)
		return
	}

	job, err := h.repository.GetDataJob(r.Context(), jobID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found", err)
		return
	}

	var metadata map[string]interface{}
	if job.Metadata != nil {
		json.Unmarshal(job.Metadata, &metadata)
	}

	response := JobStatusResponse{
		JobID:          job.ID.String(),
		Status:         job.Status,
		FileID:         job.FileID.String(),
		ProcessedCount: job.ProcessedCount,
		ErrorCount:     job.ErrorCount,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		CompletedAt:    job.CompletedAt,
		ErrorMessage:   job.ErrorMessage,
		Metadata:       metadata,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// CancelJob cancels a processing job
func (h *HTTPHandlers) CancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JOB_ID", "Invalid job ID format", err)
		return
	}

	job, err := h.repository.GetDataJob(r.Context(), jobID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found", err)
		return
	}

	// Check if job can be cancelled
	if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
		h.sendError(w, http.StatusBadRequest, "JOB_NOT_CANCELLABLE", "Job cannot be cancelled in current status", nil)
		return
	}

	// Update job status
	job.Status = "cancelled"
	job.UpdatedAt = time.Now()
	completedAt := time.Now()
	job.CompletedAt = &completedAt

	if err := h.repository.UpdateDataJob(r.Context(), job); err != nil {
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to cancel job", err)
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{
		"job_id": job.ID.String(),
		"status": "cancelled",
		"message": "Job cancelled successfully",
	})

	h.logger.Info("job cancelled", "job_id", jobID)
}

// HealthCheck handles health check requests
func (h *HTTPHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Services: map[string]string{
			"database": "healthy",
			"storage":  "healthy",
			"kafka":    "healthy",
		},
		Uptime: time.Since(startTime).String(),
	}

	// TODO: Add actual health checks for dependencies
	// For now, assume all services are healthy

	h.sendJSON(w, http.StatusOK, response)
}

// ReadinessCheck handles readiness check requests
func (h *HTTPHandlers) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual readiness checks
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// LivenessCheck handles liveness check requests
func (h *HTTPHandlers) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// MetricsHandler handles metrics requests (if not using separate metrics server)
func (h *HTTPHandlers) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// This would typically be handled by the Prometheus HTTP handler
	// This is a placeholder for custom metrics endpoint
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# Custom metrics endpoint\n# Use /metrics with Prometheus handler for full metrics\n"))
}

// sendJSON sends a JSON response
func (h *HTTPHandlers) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// sendError sends an error response
func (h *HTTPHandlers) sendError(w http.ResponseWriter, statusCode int, code, message string, err error) {
	h.logger.Error("HTTP error",
		"status_code", statusCode,
		"code", code,
		"message", message,
		"error", err)

	errorResponse := ErrorResponse{
		Error:     message,
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
	}

	if err != nil {
		errorResponse.Error = err.Error()
	}

	h.sendJSON(w, statusCode, errorResponse)
}