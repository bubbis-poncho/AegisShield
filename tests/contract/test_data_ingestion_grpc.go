//go:build integration
// +build integration

package contract

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "aegisshield/shared/proto"
)

// TestDataIngestionService_gRPC_Contract validates the data ingestion service gRPC contract
func TestDataIngestionService_gRPC_Contract(t *testing.T) {
	// This test MUST FAIL initially (TDD principle)
	// It defines the expected contract for the data ingestion service

	// Setup gRPC client connection
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewDataIngestionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("IngestTransaction_ValidPayload_Success", func(t *testing.T) {
		// Test successful transaction ingestion
		req := &pb.IngestTransactionRequest{
			TransactionId: "txn_123456789",
			Amount:        1000.50,
			Currency:      "USD",
			FromAccount:   "acc_sender_001",
			ToAccount:     "acc_receiver_001",
			Timestamp:     time.Now().Unix(),
			Description:   "Wire transfer payment",
			Metadata: map[string]string{
				"channel":    "online_banking",
				"ip_address": "192.168.1.100",
				"user_agent": "Mozilla/5.0",
			},
		}

		resp, err := client.IngestTransaction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, pb.IngestionStatus_SUCCESS, resp.Status)
		assert.NotEmpty(t, resp.InternalId)
		assert.True(t, resp.ProcessedAt > 0)
	})

	t.Run("IngestTransaction_InvalidAmount_ValidationError", func(t *testing.T) {
		// Test validation for invalid transaction amount
		req := &pb.IngestTransactionRequest{
			TransactionId: "txn_invalid_001",
			Amount:        -100.00, // Invalid negative amount
			Currency:      "USD",
			FromAccount:   "acc_sender_001",
			ToAccount:     "acc_receiver_001",
			Timestamp:     time.Now().Unix(),
		}

		resp, err := client.IngestTransaction(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("IngestTransaction_MissingRequiredFields_ValidationError", func(t *testing.T) {
		// Test validation for missing required fields
		req := &pb.IngestTransactionRequest{
			TransactionId: "", // Missing required field
			Amount:        1000.50,
			Currency:      "USD",
		}

		resp, err := client.IngestTransaction(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "transaction_id is required")
	})

	t.Run("BatchIngest_MultipleTransactions_Success", func(t *testing.T) {
		// Test batch ingestion functionality
		transactions := []*pb.IngestTransactionRequest{
			{
				TransactionId: "batch_txn_001",
				Amount:        500.00,
				Currency:      "USD",
				FromAccount:   "acc_001",
				ToAccount:     "acc_002",
				Timestamp:     time.Now().Unix(),
			},
			{
				TransactionId: "batch_txn_002",
				Amount:        750.00,
				Currency:      "EUR",
				FromAccount:   "acc_003",
				ToAccount:     "acc_004",
				Timestamp:     time.Now().Unix(),
			},
		}

		req := &pb.BatchIngestRequest{
			Transactions: transactions,
		}

		resp, err := client.BatchIngest(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 2)
		assert.Equal(t, int32(2), resp.SuccessCount)
		assert.Equal(t, int32(0), resp.ErrorCount)
	})

	t.Run("GetIngestionStatus_ValidId_ReturnsStatus", func(t *testing.T) {
		// Test status retrieval for ingested transaction
		req := &pb.GetStatusRequest{
			TransactionId: "txn_123456789",
		}

		resp, err := client.GetIngestionStatus(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "txn_123456789", resp.TransactionId)
		assert.NotEmpty(t, resp.Status)
		assert.True(t, resp.IngestedAt > 0)
	})

	t.Run("HealthCheck_ServiceAvailable_ReturnsHealthy", func(t *testing.T) {
		// Test service health check
		req := &pb.HealthCheckRequest{
			Service: "data-ingestion",
		}

		resp, err := client.HealthCheck(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, pb.HealthStatus_SERVING, resp.Status)
		assert.NotEmpty(t, resp.Message)
	})
}
