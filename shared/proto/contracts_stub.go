package proto

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

type IngestionStatus int32

const (
	IngestionStatus_UNKNOWN IngestionStatus = 0
	IngestionStatus_SUCCESS IngestionStatus = 1
	IngestionStatus_FAILED  IngestionStatus = 2
)

type HealthStatus int32

const (
	HealthStatus_UNKNOWN     HealthStatus = 0
	HealthStatus_SERVING     HealthStatus = 1
	HealthStatus_NOT_SERVING HealthStatus = 2
)

type IngestTransactionRequest struct {
	TransactionId string
	Amount        float64
	Currency      string
	FromAccount   string
	ToAccount     string
	Timestamp     int64
	Description   string
	Metadata      map[string]string
}

type IngestTransactionResponse struct {
	Status      IngestionStatus
	InternalId  string
	ProcessedAt int64
}

type BatchIngestRequest struct {
	Transactions []*IngestTransactionRequest
}

type BatchIngestResponse struct {
	Results      []*IngestTransactionResponse
	SuccessCount int32
	ErrorCount   int32
}

type GetStatusRequest struct {
	TransactionId string
}

type GetStatusResponse struct {
	TransactionId string
	Status        string
	IngestedAt    int64
}

type HealthCheckRequest struct {
	Service string
}

type HealthCheckResponse struct {
	Status  HealthStatus
	Message string
	Details map[string]string
}

type DataIngestionServiceClient interface {
	IngestTransaction(ctx context.Context, in *IngestTransactionRequest, opts ...grpc.CallOption) (*IngestTransactionResponse, error)
	BatchIngest(ctx context.Context, in *BatchIngestRequest, opts ...grpc.CallOption) (*BatchIngestResponse, error)
	GetIngestionStatus(ctx context.Context, in *GetStatusRequest, opts ...grpc.CallOption) (*GetStatusResponse, error)
	HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error)
}

type stubDataIngestionServiceClient struct{}

func NewDataIngestionServiceClient(grpc.ClientConnInterface) DataIngestionServiceClient {
	return &stubDataIngestionServiceClient{}
}

func (c *stubDataIngestionServiceClient) IngestTransaction(ctx context.Context, in *IngestTransactionRequest, opts ...grpc.CallOption) (*IngestTransactionResponse, error) {
	_ = ctx
	_ = opts

	return &IngestTransactionResponse{
		Status:      IngestionStatus_SUCCESS,
		InternalId:  "stub-" + in.TransactionId,
		ProcessedAt: time.Now().Unix(),
	}, nil
}

func (c *stubDataIngestionServiceClient) BatchIngest(ctx context.Context, in *BatchIngestRequest, opts ...grpc.CallOption) (*BatchIngestResponse, error) {
	_ = ctx
	_ = opts

	results := make([]*IngestTransactionResponse, 0, len(in.Transactions))
	for _, txn := range in.Transactions {
		results = append(results, &IngestTransactionResponse{
			Status:      IngestionStatus_SUCCESS,
			InternalId:  "stub-" + txn.TransactionId,
			ProcessedAt: time.Now().Unix(),
		})
	}

	return &BatchIngestResponse{
		Results:      results,
		SuccessCount: int32(len(results)),
		ErrorCount:   0,
	}, nil
}

func (c *stubDataIngestionServiceClient) GetIngestionStatus(ctx context.Context, in *GetStatusRequest, opts ...grpc.CallOption) (*GetStatusResponse, error) {
	_ = ctx
	_ = opts

	return &GetStatusResponse{
		TransactionId: in.TransactionId,
		Status:        "COMPLETED",
		IngestedAt:    time.Now().Unix(),
	}, nil
}

func (c *stubDataIngestionServiceClient) HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error) {
	_ = ctx
	_ = in
	_ = opts

	return &HealthCheckResponse{
		Status:  HealthStatus_SERVING,
		Message: "stubbed service is healthy",
		Details: map[string]string{"source": "stub"},
	}, nil
}
